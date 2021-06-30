package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/tidwall/gjson"
)

var (
	learningLanguage string
	fromLanguage string
	wg sync.WaitGroup
)

func init() {
	flag.StringVar(&learningLanguage, "l", "", "Learning Language")
	flag.StringVar(&fromLanguage, "f", "en", "From Language")
	flag.Parse()
}

func DownloadFile(filepath string, url string) {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	out, err := os.Create(filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	wg.Done()
}

func main() {
	jwt, err := ioutil.ReadFile("jwt.txt")
	if err != nil {
		log.Fatal(err)
	}
	url := "https://stories.duolingo.com/api2/stories?masterVersions=false&learningLanguage=" + learningLanguage + "&fromLanguage=" + fromLanguage
	pair := fromLanguage + "_" + learningLanguage
	src := pair + string(os.PathSeparator) + "src"
	var bearer = "Bearer " + string(jwt)
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", bearer)
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	respString := string(respData)
	ids := gjson.Get(respString, "sets.#.#.id|@flatten")
	jsonStringX := ids.String()
	jsonX := gjson.Get(jsonStringX, "@valid")
	xlen := gjson.Get(jsonStringX, "#")
	zeroX := len(xlen.String())
	i := 1
	jsonX.ForEach(func(key, story gjson.Result) bool {
		url := "https://stories.duolingo.com/api2/stories/" + story.String() + "?masterVersion=false"
		var bearer = "Bearer " + string(jwt)
		req, err := http.NewRequest("GET", url, nil)
		req.Header.Add("Authorization", bearer)
		client := http.DefaultClient
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		respData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		respString := string(respData)
		text := gjson.Get(respString, "elements.0.line.content.text")
		r := strings.NewReplacer("¿", "", "?", "", "¡", "", "!", "")
		x := r.Replace(text.String())
		padX := fmt.Sprintf("%0" + strconv.Itoa(zeroX) + "d", i)
		xpath := strings.TrimSpace(filepath.Join(".", src, padX + " - " + x))
		os.MkdirAll(xpath, os.ModePerm)
		audioUrls := gjson.Get(respString, "elements.#.line.content.audio.url")
		jsonStringY := audioUrls.String()
		jsonY := gjson.Get(jsonStringY, "@valid")
		ylen := gjson.Get(jsonStringY, "#")
		zeroY := len(ylen.String())
		j := 1
		jsonY.ForEach(func(key, audio gjson.Result) bool {
			wg.Add(1)
			padY := fmt.Sprintf("%0" + strconv.Itoa(zeroY) + "d", j)
			go DownloadFile(xpath + string(os.PathSeparator) + padY + ".mp3", audio.String())
			j = j + 1
			return true
		})
		i = i + 1
		return true
	})
	wg.Wait()
	homeDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	folders, err := ioutil.ReadDir(src)
	if err != nil {
		log.Fatal(err)
	}
	regEx, err := regexp.Compile("^.+\\.(mp3)$")
	if err != nil {
		log.Fatal(err)
	}
	xpath := filepath.Join(homeDir, string(os.PathSeparator), pair, string(os.PathSeparator), "concat")
	os.MkdirAll(xpath, os.ModePerm)
	for _, f := range folders {
		os.Chdir(src + string(os.PathSeparator) + f.Name())
		files, err := ioutil.ReadDir(".")
		if err != nil {
			log.Fatal(err)
		}
		for _, g := range files {
			if regEx.MatchString(g.Name()) {
				h, err := os.OpenFile("ffmpeg.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					log.Fatal(err)
				}
				if _, err := h.Write([]byte("file './" + g.Name() + "'" + "\n")); err != nil {
					h.Close()
					log.Fatal(err)
				}
				if err := h.Close(); err != nil {
					log.Fatal(err)
				}
			}
		}
		output := xpath + string(os.PathSeparator) + f.Name() + ".mp3"
		cmd := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", "ffmpeg.txt", "-c", "copy", output)
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
		cleanUp := os.Remove("ffmpeg.txt")
		if cleanUp != nil {
			log.Fatal(cleanUp)
		}
		os.Chdir(homeDir)
	}
}