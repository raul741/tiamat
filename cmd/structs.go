package cmd

import (
	"net"
	"time"
)

type ClientJSON struct {
	Date time.Time
	List []ClientBasicInfo
}

type Client struct {
	ClientBasicInfo
	Conn     net.Conn
	IsOnline bool
	ClientID int
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
	IsHeartbeat  bool
	IsCommand    bool
	IsKillswitch bool
	Message      string
}

type Response struct {
	Successful bool
	Message    string
}
