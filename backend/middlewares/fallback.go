package middlewares

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func Fallback() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if path == "/" {
			c.File("../dist/index.html")
			c.Abort()
			return
		}

		if strings.HasPrefix(path, "/ui") {
			if strings.HasPrefix(path, "/ui/assets") || path == "/ui/markpost.svg" {
				c.Next()
				return
			}
			c.File("../dist/index.html")
			c.Abort()
			return
		}

		c.Next()
	}
}
