package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

var mapDetails MapDetails
var songFolder string
var filePath string = "test.zip"
var client = &http.Client{}

func main() {
	if len(os.Args) == 1 {
		setupRegistry()
		return
	}

	var songID string
	songID = os.Args[1][12:(len(os.Args[1]) - 1)]
	fmt.Println(songID)

	InstallSong(songID)
}

func setupRegistry() {
	if !IsAdmin() {
		RunAsElevated()
		return
	}

	key, _, err := registry.CreateKey(registry.CLASSES_ROOT, "beatsaver", registry.ALL_ACCESS)
	if err != nil {
		fmt.Println(err)
		fmt.Scanln()
	}

	key.SetStringValue("URL Protocol", "")
	key.SetStringValue("OneClick-Provider", "WHK")

	commandKey, _, err := registry.CreateKey(key, "shell\\open\\command", registry.ALL_ACCESS)
	commandKey.SetStringValue("", fmt.Sprintf("\"%s\"", os.Args[0])+"\"%1\"")
	if err != nil {
		fmt.Println(err)
		fmt.Scanln()
	}
}

// InstallSong Installs a song
func InstallSong(songID string) {
	setExportFolder()
	fmt.Println("Export Folder Set")

	songName := songID + " " + getSongName(songID)
	filePath = songFolder + "\\" + songName + ".zip"
	fmt.Println("Song & File Name Set")
	fmt.Println(songName + " - " + filePath)

	downloadSong(songID)
	fmt.Println("Song Downloaded")

	fmt.Println(songFolder + "\\" + songName)
	unzip(filePath, songFolder+"\\"+songName)

	os.Remove(filePath)
}

func setExportFolder() {
	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	songFolder = user.HomeDir + "\\Documents\\Beat Saber Songs"
}

func getSongName(songID string) string {
	url := "https://beatsaver.com/api/maps/detail/" + songID

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", "Go One Click Installer")

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)

	}
	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	jsonErr := json.Unmarshal(body, &mapDetails)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	return fmt.Sprintf("(%s - %s)", mapDetails.Metadata.SongName, mapDetails.Metadata.LevelAuthorName)
}

func downloadSong(songID string) {
	url := "https://beatsaver.com" + mapDetails.DirectDownload
	fmt.Println(url)
	client := &http.Client{}

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Set("User-Agent", "Go One Click Installer")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	out, err := os.Create(filePath)
	if err != nil {
		fmt.Println(err)
	}

	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Println(err)
	}
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

// MapDetails Details for a map
type MapDetails struct {
	Metadata       Metadata `json:"metadata"`
	DirectDownload string   `json:"directDownload"`
}

// Metadata Song data
type Metadata struct {
	SongName        string `json:"songName"`
	SongSubName     string `json:"songSubName"`
	LevelAuthorName string `json:"levelAuthorName"`
}
