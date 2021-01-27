package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/crypto/ssh/terminal"
)

type HarborConfigs struct {
	HarborBaseUrl  string `json:"harborBaseUrl"`
	HarborLogin    string `json:"harborLogin"`
	HarborPassword string `json:"harborPassword"`
	HarborRepo     string `json:"harborRepo"`
	HarborChart    string `json:"harborChart"`
}

func validThenGetURL(reader *bufio.Reader) (rawurl string) {
	fmt.Print("Enter service URL: ")
	rawurl, _ = reader.ReadString('\n')
	rawurl = strings.TrimSpace(rawurl)
	if _, err := url.ParseRequestURI(rawurl); err != nil {
		fmt.Println("Invalid URL!")
		rawurl = validThenGetURL(reader)
	}
	return
}

func NewfileUploadRequest(uri string, params map[string]string, paramName, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}
	file.Close()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, fi.Name())
	if err != nil {
		return nil, err
	}
	part.Write(fileContents)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("POST", uri, body)
	request.Header.Add("Content-Type", writer.FormDataContentType())
	return request, err
}

func Push(configs *HarborConfigs) {
	path, _ := os.Getwd()
	path += "/" + configs.HarborChart + ".gz"
	urlBase := configs.HarborBaseUrl + "/api/chartrepo/" + configs.HarborRepo + "/charts"
	extraParams := map[string]string{
		"repo": configs.HarborChart,
	}
	request, err := NewfileUploadRequest(
		urlBase,
		extraParams,
		"chart",
		path,
	)
	if err != nil {
		log.Println(err)
	}

	request.SetBasicAuth(configs.HarborLogin, configs.HarborPassword)
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	} else {
		var bodyContent []byte
		resp.Body.Read(bodyContent)
		resp.Body.Close()
		fmt.Println(bodyContent)
		if resp.StatusCode == 201 {
			fmt.Println("Saved")
		}
	}
}

func TarFolder(configs *HarborConfigs) {
	cmd := exec.Command("/bin/sh", "-c", "tar -zcvf "+configs.HarborChart+".gz "+configs.HarborChart)
	err := cmd.Run()
	fmt.Println("Error GZ: ", err)
}

func main() {
	args := os.Args[1:]

	if len(args) < 2 {
		fmt.Printf("No valid args found. Usage: %q\r\n", "harborpush CHART_NAME REPO_NAME")
		os.Exit(0)
	}

	defaultPath := os.Getenv("HOME") + "/.harbor_config.json"
	configs := HarborConfigs{}

	if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
		reader := bufio.NewReader(os.Stdin)
		configs.HarborBaseUrl = strings.TrimSpace(validThenGetURL(reader))
		fmt.Print("Enter your harbor login: ")
		harborLogin, _ := reader.ReadString('\n')
		configs.HarborLogin = strings.TrimSpace(harborLogin)
		fmt.Print("Enter your harbor password: ")
		harborPassword, _ := terminal.ReadPassword(0)
		configs.HarborPassword = strings.TrimSpace(string(harborPassword))
		file, _ := json.MarshalIndent(configs, "", " ")
		if err = ioutil.WriteFile(defaultPath, file, 0644); err != nil {
			fmt.Println("Error trying to write config file. Err: ", err.Error())
		}
	} else {
		file, _ := ioutil.ReadFile(defaultPath)
		_ = json.Unmarshal([]byte(file), &configs)
	}

	configs.HarborChart = strings.TrimSpace(args[0])
	configs.HarborRepo = strings.TrimSpace(args[1])

	TarFolder(&configs)
	Push(&configs)
	os.Remove(configs.HarborChart + ".gz")
}
