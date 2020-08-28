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
	"os"
	"os/exec"
	"strings"
)

type HarborConfigs struct {
	HarborBaseUrl  string `json:"harborBaseUrl"`
	HarborLogin    string `json:"harborLogin"`
	HarborPassword string `json:"harborPassword"`
	HarborRepo     string `json:"harborRepo"`
	HarborChart    string `json:"harborChart"`
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

func Push(configs HarborConfigs) {
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

func TarFolder(configs HarborConfigs) {
	cmd := exec.Command("/bin/sh", "-c", "$(which tar) -zcvf "+configs.HarborChart+".gz "+configs.HarborChart)
	err := cmd.Run()
	fmt.Println("Error: ", err)
}

func main() {
	var defaultPath = "~/.harbor_config.json"
	args := os.Args[1:]
	var configs = HarborConfigs{
		HarborRepo:  args[1],
		HarborChart: args[0],
	}

	if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
		var harborConfigsData HarborConfigs
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter your harbor url: ")
		harborBaseUrl, _ := reader.ReadString('\n')
		harborConfigsData.HarborBaseUrl = harborBaseUrl
		fmt.Print("Enter your harbor login: ")
		harborLogin, _ := reader.ReadString('\n')
		harborConfigsData.HarborLogin = harborLogin
		fmt.Print("Enter your harbor password: ")
		harborPassword, _ := reader.ReadString('\n')
		harborConfigsData.HarborPassword = harborPassword
		file, _ := json.MarshalIndent(harborConfigsData, "", " ")
		err = ioutil.WriteFile(defaultPath, file, 0644)
		fmt.Println("err: ", err)
	} else {
		file, _ := ioutil.ReadFile(defaultPath)
		_ = json.Unmarshal([]byte(file), &configs)
	}
	configs.HarborLogin = strings.TrimSpace(configs.HarborLogin)
	configs.HarborPassword = strings.TrimSpace(configs.HarborPassword)
	configs.HarborBaseUrl = strings.TrimSpace(configs.HarborBaseUrl)
	TarFolder(configs)
	Push(configs)
	os.Remove(configs.HarborChart + ".gz")
}
