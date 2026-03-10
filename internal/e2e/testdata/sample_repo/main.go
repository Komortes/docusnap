package main

import (
	"example.com/sample/internal/server"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	api := r.Group("/api")
	api.GET("/health", server.Health)
}
