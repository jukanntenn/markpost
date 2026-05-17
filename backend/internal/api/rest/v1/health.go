// Package v1 provides REST API v1 handlers.
package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Health returns a health check handler.
func Health() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

// NotFound returns a 404 not found handler.
func NotFound() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(http.StatusNotFound, "Not Found")
	}
}
