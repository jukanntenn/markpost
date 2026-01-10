package middlewares

import (
	"strings"

	apperrors "markpost/errors"
	"markpost/repositories"
	"markpost/services"

	"github.com/gin-gonic/gin"
)

func Auth(jwtSvc *services.JWTService, users repositories.UserRepoInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			apperrors.RespondError(c, services.NewServiceErrorWrap(services.ErrMissingAuthorizationHeader, "missing Authorization header", nil))
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := jwtSvc.ValidateAccess(tokenString)
		if err != nil {
			apperrors.RespondError(c, services.NewServiceErrorWrap(services.ErrInvalidToken, "invalid token", err))
			c.Abort()
			return
		}

		userID, err := claims.UserID()
		if err != nil {
			apperrors.RespondError(c, services.NewServiceErrorWrap(services.ErrInvalidToken, "invalid token", err))
			c.Abort()
			return
		}
		user, err := users.GetUserByID(userID)
		if err != nil {
			apperrors.RespondError(c, services.NewServiceErrorWrap(services.ErrInternal, "failed to get user information", err))
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Next()
	}
}
