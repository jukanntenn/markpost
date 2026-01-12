package handlers

import (
	"net/http"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// Health godoc
// @Summary      Health check
// @Description  Check API health status
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func Health() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"message": ginI18n.MustGetMessage(c, &i18n.LocalizeConfig{
				DefaultMessage: &i18n.Message{
					ID:    "health.running",
					Other: "markpost is running",
				},
			}),
		})
	}
}
