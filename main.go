package main

import (
	"ProyectoGinBack/config"

	"github.com/gin-gonic/gin"
)

func main() {
	config.ConectarDB()

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	r.Run()
}
