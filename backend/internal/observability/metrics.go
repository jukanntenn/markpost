package observability

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds the business instruments defined in observability.md §指标清单.
// They are resolved once from the global MeterProvider (set by Init). When the
// global provider has not been initialized (unit tests), otel.Meter returns a
// no-op meter whose instruments are themselves no-ops, so callers never need to
// guard against nil.
type Metrics struct {
	PostsCreated       metric.Int64Counter
	AuthLoginSuccess   metric.Int64Counter
	AuthLoginFailure   metric.Int64Counter
	TokenRefresh       metric.Int64Counter
	DeliveryDispatched metric.Int64Counter
	DeliveryFailed     metric.Int64Counter
}

// meterName is the Meter used for all markpost business metrics.
const meterName = "markpost"

// NewMetrics resolves the business instruments from the global MeterProvider.
// otel.Meter returns a no-op meter before a provider is registered, so this
// never panics and instruments degrade to no-ops in tests.
func NewMetrics() *Metrics {
	m := otel.Meter(meterName)
	postsCreated, _ := m.Int64Counter("markpost.posts.created_total", metric.WithDescription("Posts created"))
	authLoginSuccess, _ := m.Int64Counter("markpost.auth.login_success_total", metric.WithDescription("Successful logins"))
	authLoginFailure, _ := m.Int64Counter("markpost.auth.login_failure_total", metric.WithDescription("Failed logins"))
	tokenRefresh, _ := m.Int64Counter("markpost.auth.token_refresh_total", metric.WithDescription("Token refreshes"))
	deliveryDispatched, _ := m.Int64Counter("markpost.delivery.dispatched_total", metric.WithDescription("Deliveries dispatched"))
	deliveryFailed, _ := m.Int64Counter("markpost.delivery.failed_total", metric.WithDescription("Deliveries failed"))
	return &Metrics{
		PostsCreated:       postsCreated,
		AuthLoginSuccess:   authLoginSuccess,
		AuthLoginFailure:   authLoginFailure,
		TokenRefresh:       tokenRefresh,
		DeliveryDispatched: deliveryDispatched,
		DeliveryFailed:     deliveryFailed,
	}
}
