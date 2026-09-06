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

// MeProfile godoc
// @Summary Report the caller's own profile
// @Tags me
// @Produce json
// @Security BearerAuth
// @Success 200 {object} v1.UserResponse
// @Failure 401 {object} apierr.ErrorResponse
// @Router /api/v1/me [get]
func MeProfile() gin.HandlerFunc {
	return func(c *gin.Context) {
		// No service call: AuthWithBlacklist re-reads the user row on every
		// request, so the context user is already fresh — admin-side changes
		// (vip, role, ban state) are visible here without re-login.
		u, ok := requireUser(c)
		if !ok {
			return
		}
		c.JSON(http.StatusOK, newUserResponse(*u))
	}
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
