package cmd

import (
	"embed"
	"fmt"
	"net/http"
	"strconv"

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
	r.Run(":" + WebPort)
}

func getRoot(c *gin.Context) {
	c.HTML(http.StatusOK, "templates/index.html", gin.H{
		"UserAgent": c.Request.UserAgent(),
		"Clients":   clientList,
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
		"UserAgent": c.Request.UserAgent(),
		"client":    client,
	})
}

func execCMD(c *gin.Context) {
	fmt.Println("HELLO")
}
