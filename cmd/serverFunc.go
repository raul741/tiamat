package cmd

import (
	"log"
	"net"

	"github.com/spf13/viper"
)

var (
	C2Port string = "1337"
)

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

}
