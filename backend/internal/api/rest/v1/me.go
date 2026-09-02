package v1

import (
	"net/http"

	"markpost/internal/domain/user"
	"markpost/internal/service/me"

	"github.com/gin-gonic/gin"
)

// MeService is the surface /me/* handlers consume.
type MeService interface {
	EffectiveRetention(u *user.User) me.RetentionResult
}

// MeRetention godoc
// @Summary Report the caller's effective retention policy
// @Tags me
// @Produce json
// @Security BearerAuth
// @Success 200 {object} me.RetentionResult
// @Failure 401 {object} apierr.ErrorResponse
// @Router /api/v1/me/retention [get]
func MeRetention(meSvc MeService) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, ok := requireUser(c)
		if !ok {
			return
		}
		c.JSON(http.StatusOK, meSvc.EffectiveRetention(u))
	}
}
