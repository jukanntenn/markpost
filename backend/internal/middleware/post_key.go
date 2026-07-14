package middleware

import (
	"markpost/internal/domain/user"
	"markpost/internal/service"
	"markpost/internal/service/auth"

	"github.com/gin-gonic/gin"
)

// PostKey returns a middleware that validates post key.
func PostKey(users user.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		postKey := c.Param("post_key")
		if postKey == "" {
			abortWithError(c, service.New(auth.ErrInvalidPostKey, "post key is required"))
			return
		}

		u, err := users.GetByPostKey(c.Request.Context(), postKey)
		if err != nil {
			abortWithError(c, service.Wrap(auth.ErrInvalidPostKey, "invalid post key", err))
			return
		}

		setUserFields(c, u)
		c.Next()
	}
}
