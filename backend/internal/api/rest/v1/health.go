// Package v1 provides REST API v1 handlers.
package v1

import (
	"net/http"

	"markpost/pkg/apierr"

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

// InternalError writes an internal error response.
func InternalError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, apierr.ErrorResponse{
		Code:    "internal_error",
		Message: message,
	})
}
