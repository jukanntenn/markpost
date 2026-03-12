package v1

import (
	"net/http"

	"markpost/pkg/apierr"

	"github.com/gin-gonic/gin"
)

func Health() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func NotFound() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(http.StatusNotFound, "Not Found")
	}
}

func InternalError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, apierr.ErrorResponse{
		Code:    "internal_error",
		Message: message,
	})
}
