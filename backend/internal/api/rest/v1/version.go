package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Version godoc
// @Summary Server version
// @Description Reports the build version (VERSION build-arg: git tag for release images, git describe for local builds).
// @Tags health
// @Produce json
// @Success 200 {object} VersionResponse
// @Router /api/v1/version [get]
func Version(version string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, VersionResponse{Version: version})
	}
}
