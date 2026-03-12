package middleware

import (
	"strings"

	"markpost/internal/domain/user"
	"markpost/internal/service"
	"markpost/internal/service/auth"
	"markpost/pkg/apierr"

	"github.com/gin-gonic/gin"
)

func Auth(jwtSvc *auth.JWTService, users user.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			apierr.RespondError(c, service.NewServiceErrorWrap(service.ErrMissingAuthorizationHeader, "missing Authorization header", nil))
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := jwtSvc.ValidateAccess(tokenString)
		if err != nil {
			apierr.RespondError(c, service.NewServiceErrorWrap(service.ErrInvalidToken, "invalid token", err))
			c.Abort()
			return
		}

		u, err := users.GetByID(c.Request.Context(), claims.UserID)
		if err != nil {
			apierr.RespondError(c, service.NewServiceErrorWrap(service.ErrInternal, "failed to get user information", err))
			c.Abort()
			return
		}

		c.Set("user", u)
		c.Next()
	}
}
