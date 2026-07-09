package v1

import (
	"net/http"

	"markpost/internal/web"

	"github.com/gin-gonic/gin"
)

const cssCacheControl = "public, max-age=31536000, immutable"

// StaticCSS serves the content-hashed, minified, embedded post stylesheet.
// The URL (/static/post.<hash>.css) changes whenever the CSS changes, so the
// response is cacheable for one year as immutable.
func StaticCSS() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Param("filename") != "post."+web.CSSHash+".css" {
			c.Status(http.StatusNotFound)
			return
		}
		c.Header("Cache-Control", cssCacheControl)
		c.Data(http.StatusOK, "text/css; charset=utf-8", web.CSSBytes())
	}
}
