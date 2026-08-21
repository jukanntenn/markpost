package middleware

import (
	"github.com/gin-gonic/gin"
)

// NoStore stamps an explicit Cache-Control: no-store on every wrapped
// response. Dynamic API payloads — notably /oauth/url, whose body carries a
// one-time CSRF state — must never be cacheable by a shared cache. Today they
// survive on the CDN's implicit "no file extension, no cache" default; that is
// a silent dependency that a future Cache-Control: public in a handler or a
// more aggressive cache rule would break. Handlers may still override the
// header afterwards for responses that are deliberately cacheable.
func NoStore() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Next()
	}
}
