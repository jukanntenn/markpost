package middleware

import (
	"markpost/internal/domain/user"
	"markpost/internal/service"
	"markpost/pkg/apierr"

	"github.com/gin-gonic/gin"
)

// PostKey returns a middleware that validates post key.
func PostKey(users user.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		postKey := c.Param("post_key")
		if postKey == "" {
			apierr.RespondError(c, service.NewServiceErrorWrap(service.ErrInvalidPostKey, "post key is required", nil))
			c.Abort()
			return
		}

		u, err := users.GetByPostKey(c.Request.Context(), postKey)
		if err != nil {
			apierr.RespondError(c, service.NewServiceErrorWrap(service.ErrInvalidPostKey, "invalid post key", err))
			c.Abort()
			return
		}

		c.Set("user", u)
		c.Next()
	}
}
