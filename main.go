// main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
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

	// Parse expiration time
	expireHours := c.PostForm("expire")
	var expiration time.Duration

	if expireHours == "infinite" {
		expiration = 0 // 0 means no expiration in Redis
	} else {
		hours, err := strconv.Atoi(expireHours)
		if err != nil || hours <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expiration time"})
			return
		}
		expiration = time.Duration(hours) * time.Hour
	}

	// Generate a shorter unique ID for the paste
	id := uuid.New().String()[:8]

	// Store in Redis with specified expiration
	ctx := context.Background()
	err := rdb.Set(ctx, id, content, expiration).Err()
	if err != nil {
		log.Printf("Error storing paste: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store paste"})
		return
	}

	// If expiration is set, also store the expiration time for display
	if expiration > 0 {
		expirationTime := time.Now().Add(expiration)
		rdb.Set(ctx, id+":expires", expirationTime.Unix(), expiration)
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

	// Get expiration time if it exists
	var expiresAt string
	expirationTimestamp, err := rdb.Get(ctx, id+":expires").Int64()
	if err == nil {
		expirationTime := time.Unix(expirationTimestamp, 0)
		expiresAt = expirationTime.Format("2006-01-02 15:04:05")
	}

	c.HTML(http.StatusOK, "paste.html", gin.H{
		"content":   content,
		"id":        id,
		"expiresAt": expiresAt,
	})
}
