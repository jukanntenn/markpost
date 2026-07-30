package middleware

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"github.com/gin-gonic/gin"
)

// activeRequests is the in-flight HTTP request counter
// (observability.md §http.server.active_requests). Resolved once from the
// global MeterProvider; a no-op before Init.
var activeRequests = func() metric.Int64UpDownCounter {
	c, _ := otel.Meter("markpost").Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithDescription("Number of active HTTP server requests."),
	)
	return c
}()

// ActiveRequests instruments the in-flight request count, decrementing on
// response completion. Register it early in the chain so it wraps all handlers.
func ActiveRequests() gin.HandlerFunc {
	return func(c *gin.Context) {
		activeRequests.Add(c.Request.Context(), 1)
		defer func() { activeRequests.Add(c.Request.Context(), -1) }()
		c.Next()
	}
}
