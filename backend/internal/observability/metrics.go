package observability

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds the business instruments defined in docs/metrics.md. They are
// resolved once from the global MeterProvider (set by Init). When the global
// provider has not been initialized (unit tests), otel.Meter returns a no-op
// meter whose instruments are themselves no-ops, so callers never need to guard
// against nil.
type Metrics struct {
	PostsCreated       metric.Int64Counter
	AuthLoginSuccess   metric.Int64Counter
	AuthLoginFailure   metric.Int64Counter
	TokenRefresh       metric.Int64Counter
	DeliveryPending    metric.Int64UpDownCounter
	DeliveryDispatched metric.Int64Counter
	DeliveryFailed     metric.Int64Counter
	RenderCacheHit     metric.Int64Counter
	RenderCacheMiss    metric.Int64Counter
	CDNPurgeSuccess    metric.Int64Counter
	CDNPurgeFailure    metric.Int64Counter
	CDNPurgeSkipped    metric.Int64Counter
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
	deliveryPending, _ := m.Int64UpDownCounter("markpost.delivery.pending", metric.WithDescription("Pending delivery attempts"))
	deliveryDispatched, _ := m.Int64Counter("markpost.delivery.dispatched_total", metric.WithDescription("Deliveries dispatched"))
	deliveryFailed, _ := m.Int64Counter("markpost.delivery.failed_total", metric.WithDescription("Deliveries failed"))
	renderCacheHit, _ := m.Int64Counter("markpost.render_cache.hit_total", metric.WithDescription("Render requests served from the render cache"))
	renderCacheMiss, _ := m.Int64Counter("markpost.render_cache.miss_total", metric.WithDescription("Render requests that missed the render cache"))
	cdnPurgeSuccess, _ := m.Int64Counter("markpost.cdn.purge_success_total", metric.WithDescription("Successful CDN cache-tag purges"))
	cdnPurgeFailure, _ := m.Int64Counter("markpost.cdn.purge_failure_total", metric.WithDescription("Failed CDN cache-tag purges"))
	cdnPurgeSkipped, _ := m.Int64Counter("markpost.cdn.purge_skipped_total", metric.WithDescription("Skipped CDN cache-tag purges"))
	return &Metrics{
		PostsCreated:       postsCreated,
		AuthLoginSuccess:   authLoginSuccess,
		AuthLoginFailure:   authLoginFailure,
		TokenRefresh:       tokenRefresh,
		DeliveryPending:    deliveryPending,
		DeliveryDispatched: deliveryDispatched,
		DeliveryFailed:     deliveryFailed,
		RenderCacheHit:     renderCacheHit,
		RenderCacheMiss:    renderCacheMiss,
		CDNPurgeSuccess:    cdnPurgeSuccess,
		CDNPurgeFailure:    cdnPurgeFailure,
		CDNPurgeSkipped:    cdnPurgeSkipped,
	}
}
