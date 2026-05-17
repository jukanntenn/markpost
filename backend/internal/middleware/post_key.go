package middleware

import (
	"markpost/internal/domain/user"
	"markpost/internal/service"

	"github.com/gin-gonic/gin"
)

// PostKey returns a middleware that validates post key.
func PostKey(users user.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		postKey := c.Param("post_key")
		if postKey == "" {
			abortWithError(c, service.NewServiceError(service.ErrInvalidPostKey, "post key is required"))
			return
		}

		u, err := users.GetByPostKey(c.Request.Context(), postKey)
		if err != nil {
			abortWithError(c, service.NewServiceErrorWrap(service.ErrInvalidPostKey, "invalid post key", err))
			return
		}

		setUserFields(c, u)
		c.Next()
	}
}
