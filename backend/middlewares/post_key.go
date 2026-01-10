package middlewares

import (
	"database/sql"

	apperrors "markpost/errors"
	"markpost/repositories"
	"markpost/services"

	"github.com/gin-gonic/gin"
)

func PostKey(repo repositories.UserRepoInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		postKey := c.Param("post_key")
		user, err := repo.GetUserByPostKey(postKey)
		if err != nil {
			switch err {
			case sql.ErrNoRows:
				apperrors.RespondError(c, services.NewServiceErrorWrap(services.ErrInvalidPostKey, "invalid post key", err))
			default:
				apperrors.RespondError(c, services.NewServiceErrorWrap(services.ErrInternal, "internal error when validating post key", err))
			}
			c.Abort()
			return
		}
		c.Set("user", user)
		c.Next()
	}
}
