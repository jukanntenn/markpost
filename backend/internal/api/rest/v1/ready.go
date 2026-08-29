package v1

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Pinger is the minimal dependency Readiness needs: the database connection
// pool. Declared consumer-side so the API layer stays free of gorm imports.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// Readiness godoc
// @Summary Readiness check
// @Description Deep health probe: a driver-level database round trip. Unlike /health (liveness only), it reports 503 when the process is up but unfit to serve — e.g. Postgres unreachable or the pool exhausted.
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} HealthResponse
// @Router /api/v1/ready [get]
func Readiness(pinger Pinger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Bound the probe so a hung database cannot outlive the monitor's
		// own timeout and stack up dangling requests.
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		if err := pinger.PingContext(ctx); err != nil {
			// The 503 alone says "not ready"; the error is what makes the
			// cause diagnosable from app-*.jsonl, so it is logged here
			// rather than swallowed into the response body.
			slog.Warn("readiness probe failed", "error", err)
			c.JSON(http.StatusServiceUnavailable, HealthResponse{Status: "unavailable"})
			return
		}
		c.JSON(http.StatusOK, HealthResponse{Status: "ready"})
	}
}
