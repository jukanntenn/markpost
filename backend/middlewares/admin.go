package middlewares

import (
	"github.com/gin-gonic/gin"
)

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(401, gin.H{"code": "unauthorized", "message": "unauthorized"})
			c.Abort()
			return
		}

		userInterface, ok := user.(interface{ IsAdmin() bool })
		if !ok || !userInterface.IsAdmin() {
			c.JSON(403, gin.H{"code": "forbidden", "message": "admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}
