package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"thorium-go/process"
	"thorium-go/requests"
	"thorium-go/usage"
	"time"
)

import _ "github.com/lib/pq"
import "github.com/go-martini/martini"

var registerData request.MachineRegisterResponse
var port int

func main() {
	fmt.Println("hello world")

	timeNow := time.Now()
	rand.Seed(int64(timeNow.Second()))
	port = rand.Intn(50000)
	port = port + 10000

	fmt.Println(strconv.Itoa(port), "\n")

	reqData := &request.RegisterMachine{Port: port}
	jsonBytes, err := json.Marshal(reqData)
	if err != nil {
		log.Fatal(err)
	}
	request, err := http.NewRequest("POST", "http://52.25.124.72:6960/machines/register", bytes.NewBuffer(jsonBytes))
	if err != nil {
		log.Fatal(err)
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if response.StatusCode != 200 {
		log.Print("Error registering with master")
		os.Exit(1)
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	json.Unmarshal([]byte(body), &registerData)
	fmt.Println("Registered As Machine#", registerData.MachineId)

	m := martini.Classic()

	m.Get("/", handlePingRequest)
	m.Get("/status", handlePingRequest)
	m.Post("/games/:id", handlePostNewGame)
	m.Post("/games/:id/register_server", handleRegisterLocalServer)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGKILL, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		fmt.Println(<-c)
		shutdown()
		os.Exit(1)
	}()
	defer shutdown()

	ticker := time.NewTicker(2 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				sendHeartbeat()
			}
		}
	}()

	thisIp := fmt.Sprintf(":%d", port)
	m.RunOnAddr(thisIp)
}

func sendHeartbeat() {
	var err error
	statusData := &request.MachineStatus{}
	statusData.MachineToken = registerData.MachineToken
	statusData.UsageCPU, _ = usage.GetCPU()
	statusData.UsageNetwork, _ = usage.GetNetworkUtilization()
	statusData.PlayerCapacity = 0.0

	jsonBytes, err := json.Marshal(statusData)
	if err != nil {
		log.Print(err)
		return
	}

	request, err := http.NewRequest("POST", "http://52.25.124.72:6960/machines/status", bytes.NewBuffer(jsonBytes))
	if err != nil {
		log.Print(err)
		return
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Connection", "close")

	client := &http.Client{}
	var resp *http.Response
	resp, err = client.Do(request)
	if err != nil {
		log.Print(err)
		return
	}

	resp.Body.Close()
}

func handlePingRequest() (int, string) {
	return 200, "OK"
}

func handlePostNewGame(http_req *http.Request, params martini.Params) (int, string) {

	defer http_req.Body.Close()
	decoder := json.NewDecoder(http_req.Body)
	var req_data request.PostNewGame
	err := decoder.Decode(&req_data)
	if err != nil {
		log.Print("bad json request:\n", http_req.Body)
		return 400, "Bad Request"
	}

	if registerData.MachineToken != req_data.MachineToken {
		log.Print("rejecting failed post new game - incorrect machine token in request")
		return 400, "Bad Request"
	}

	var game_id int
	game_id, err = strconv.Atoi(params["id"])
	if err != nil {
		log.Print(err)
		return 400, "Bad Request"
	}

	var server *process.ManagedProcess
	server, err = process.NewGameServer(game_id, rand.Intn(50000)+10000, port, req_data.GameMode, req_data.MapName)
	if err != nil {
		log.Print(err)
		return 500, "Internal Server Error"
	}
	log.Print("started new game server w/ pid = ", server.Process.Pid)

	return 200, "OK"
}

func handleRegisterLocalServer(httpReq *http.Request, params martini.Params) (int, string) {

	decoder := json.NewDecoder(httpReq.Body)
	var data request.RegisterGameServer
	err := decoder.Decode(&data)
	if err != nil {
		log.Print(err)
		return 400, "Bad Request"
	}

	data.MachineId = registerData.MachineId

	var jsonBytes []byte
	jsonBytes, err = json.Marshal(&data)
	if err != nil {
		log.Print(err)
		return 500, "Internal Server Error"
	}

	endpoint := fmt.Sprintf("http://52.25.124.72:6960/games/%s/register_server", params["id"])
	var req *http.Request
	req, err = http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		log.Print(err)
		return 500, "Internal Server Error"
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		log.Print(err)
		return 500, "Internal Server Error"
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Print("error: couldn't register game server with master")
		return 500, "Internal Server Error"
	}

	return 200, "OK"
}

func shutdown() {

	var reqData request.UnregisterMachine
	reqData.MachineToken = registerData.MachineToken
	jsonBytes, err := json.Marshal(&reqData)
	if err != nil {
		return
	}

	var req *http.Request
	req, err = http.NewRequest("POST", fmt.Sprintf("http://52.25.124.72:6960/machines/%d/disconnect", registerData.MachineId), bytes.NewBuffer(jsonBytes))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		log.Print("failed to disconnect properly")
		return
	}
	resp.Body.Close()
}
