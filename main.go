// main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client

func main() {
	// Initialize Redis client
	rdb = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	// Routes
	r.GET("/", handleIndex)
	r.POST("/paste", handleCreatePaste)
	r.GET("/paste/:id", handleGetPaste)

	r.Run(":8080")
}

func handleIndex(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{})
}

func handleCreatePaste(c *gin.Context) {
	content := c.PostForm("content")
	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content cannot be empty"})
		return
	}

	// Generate unique ID for the paste
	id := uuid.New().String()

	// Store in Redis with 24 hour expiration
	ctx := context.Background()
	err := rdb.Set(ctx, id, content, 24*time.Hour).Err()
	if err != nil {
		log.Printf("Error storing paste: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store paste"})
		return
	}

	c.Redirect(http.StatusFound, "/paste/"+id)
}

func handleGetPaste(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()

	content, err := rdb.Get(ctx, id).Result()
	if err == redis.Nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "Paste not found",
		})
		return
	} else if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to retrieve paste",
		})
		return
	}

	c.HTML(http.StatusOK, "paste.html", gin.H{
		"content": content,
		"id":      id,
	})
}
