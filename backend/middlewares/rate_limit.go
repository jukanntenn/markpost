package middlewares

import (
	apperrors "markpost/errors"

	"github.com/didip/tollbooth/v8"
	"github.com/didip/tollbooth/v8/limiter"
	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func RateLimitByIP(lmt *limiter.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		httpError := tollbooth.LimitByKeys(lmt, []string{c.ClientIP()})
		if httpError != nil {
			message := ginI18n.MustGetMessage(c, &i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "error.rate_limited",
					Other: "You have reached maximum request limit.",
				},
			})

			c.JSON(httpError.StatusCode, apperrors.ErrorResponse{
				Code:    "rate_limited",
				Message: message,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
