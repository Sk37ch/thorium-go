package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"thorium-go/requests"
)

import "bytes"
import "io/ioutil"

var address string = "52.25.124.72"
var port int = 6960

func PingMaster() (bool, error) {

	endpoint := fmt.Sprintf("http://%s:%d/status", address, port)
	req, err := http.NewRequest("GET", endpoint, bytes.NewBuffer([]byte("")))
	if err != nil {
		return false, err
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Print("ping master - error:\n", err)
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return false, errors.New("http status error")
	} else {
		return true, nil
	}
}

func LoginRequest(username string, password string) (string, error) {
	var loginReq request.Authentication
	loginReq.Username = username
	loginReq.Password = password
	jsonBytes, err := json.Marshal(&loginReq)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%d/clients/login", address, port), bytes.NewBuffer(jsonBytes))
	if err != nil {
		log.Print("error with request: ", err)
		return "err", err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Print("error with sending request", err)
		return "err", err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	tokenString := bytes.NewBuffer(body).String()
	log.Print("account token:\n", tokenString)
	return tokenString, nil
}

func CharacterSelectRequest(token string, id int) (string, error) {

	var selectReq request.SelectCharacter
	selectReq.AccountToken = token
	selectReq.ID = id
	jsonBytes, err := json.Marshal(&selectReq)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%d/characters/%d/select", address, port, id), bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Print("Error with request 2: ", err)
		return "err", err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return string(body), nil
}

func CharacterCreateRequest(token string, name string) (string, error) {

	var charCreateReq request.CreateCharacter
	charCreateReq.AccountToken = token
	charCreateReq.Name = name
	jsonBytes, err := json.Marshal(&charCreateReq)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%d/characters/new", address, port), bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Print("Error with request 2: ", err)
		return "err", err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	log.Print("create character response: ", string(body))
	return string(body), nil
}

func DisconnectRequest(token string) (string, error) {

	buf := []byte(token)
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%d/clients/disconnect", address, port), bytes.NewBuffer(buf))
	if err != nil {
		log.Print("error with request: ", err)
		return "err", err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Print("error with sending request", err)
		return "err", err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	log.Print("disonnect response: ", string(body))
	return string(body), nil
}
