package cmd

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

var (
	WebPort string = "8080"
)

func WebServer() {
	p := viper.GetString("Server.WebPort")
	if p != "" {
		WebPort = p
	}
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.GET("/", getRoot)
	r.Run(":" + WebPort)
}

func getRoot(c *gin.Context) {
	c.String(http.StatusOK, "Hello world!\n")
}
