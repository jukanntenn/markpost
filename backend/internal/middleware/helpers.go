package middleware

import (
	"markpost/internal/domain/user"
	"markpost/pkg/apierr"

	"github.com/gin-gonic/gin"
)

func abortWithError(c *gin.Context, err error) {
	apierr.RespondError(c, err)
	c.Abort()
}

// ExtractUser extracts the authenticated user from the gin context.
func ExtractUser(c *gin.Context) (*user.User, bool) {
	u, ok := c.Get("user")
	if !ok {
		return nil, false
	}
	typed, ok := u.(*user.User)
	if !ok {
		return nil, false
	}
	return typed, true
}
