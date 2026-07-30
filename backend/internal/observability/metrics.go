package observability

import (
	"context"
	"runtime"

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
	DeliveryPending    metric.Int64UpDownCounter
	DeliveryDispatched metric.Int64Counter
	DeliveryFailed     metric.Int64Counter
}

// meterName is the Meter used for all markpost business metrics.
const meterName = "markpost"

// NewMetrics resolves the business instruments from the global MeterProvider
// and registers the runtime-metrics callback. otel.Meter returns a no-op meter
// before a provider is registered, so this never panics and instruments
// degrade to no-ops in tests.
func NewMetrics() *Metrics {
	m := otel.Meter(meterName)
	postsCreated, _ := m.Int64Counter("markpost.posts.created_total", metric.WithDescription("Posts created"))
	authLoginSuccess, _ := m.Int64Counter("markpost.auth.login_success_total", metric.WithDescription("Successful logins"))
	authLoginFailure, _ := m.Int64Counter("markpost.auth.login_failure_total", metric.WithDescription("Failed logins"))
	tokenRefresh, _ := m.Int64Counter("markpost.auth.token_refresh_total", metric.WithDescription("Token refreshes"))
	deliveryPending, _ := m.Int64UpDownCounter("markpost.delivery.pending", metric.WithDescription("Pending delivery attempts"))
	deliveryDispatched, _ := m.Int64Counter("markpost.delivery.dispatched_total", metric.WithDescription("Deliveries dispatched"))
	deliveryFailed, _ := m.Int64Counter("markpost.delivery.failed_total", metric.WithDescription("Deliveries failed"))
	registerRuntimeMetrics(m)
	return &Metrics{
		PostsCreated:       postsCreated,
		AuthLoginSuccess:   authLoginSuccess,
		AuthLoginFailure:   authLoginFailure,
		TokenRefresh:       tokenRefresh,
		DeliveryPending:    deliveryPending,
		DeliveryDispatched: deliveryDispatched,
		DeliveryFailed:     deliveryFailed,
	}
}

// registerRuntimeMetrics wires the Go runtime gauges (goroutine count, heap
// alloc, GC count) as observable instruments sampled by the PeriodicReader on
// each collection (observability.md §系统 runtime metrics). A failing
// registration is silent: the SDK returns no-op instruments, and tests that
// never call Init keep working.
func registerRuntimeMetrics(m metric.Meter) {
	goroutines, _ := m.Int64ObservableGauge(
		"process.runtime.go.goroutines",
		metric.WithDescription("Number of goroutines"),
	)
	heapAlloc, _ := m.Int64ObservableGauge(
		"process.runtime.go.mem.heap_alloc",
		metric.WithDescription("Bytes of allocated heap objects"),
		metric.WithUnit("By"),
	)
	heapSys, _ := m.Int64ObservableGauge(
		"process.runtime.go.mem.heap_sys",
		metric.WithDescription("Bytes of heap memory obtained from the OS"),
		metric.WithUnit("By"),
	)
	gcCount, _ := m.Int64ObservableGauge(
		"process.runtime.go.gc.count",
		metric.WithDescription("Number of completed GC cycles"),
	)
	_, _ = m.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		o.ObserveInt64(goroutines, int64(runtime.NumGoroutine()))
		o.ObserveInt64(heapAlloc, int64(ms.HeapAlloc))
		o.ObserveInt64(heapSys, int64(ms.HeapSys))
		o.ObserveInt64(gcCount, int64(ms.NumGC))
		return nil
	}, goroutines, heapAlloc, heapSys, gcCount)
}
