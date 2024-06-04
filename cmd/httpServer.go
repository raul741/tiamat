package cmd

import (
	"embed"
	"net/http"

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
	r.Run(":" + WebPort)
}

func getRoot(c *gin.Context) {
	c.HTML(http.StatusOK, "templates/index.html", gin.H{
		"UserAgent": c.Request.UserAgent(),
		"Clients":   clientList,
	})
}
