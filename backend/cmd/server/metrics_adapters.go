package main

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"markpost/internal/observability"
)

// dispatcherMetrics adapts *observability.Metrics to the delivery dispatcher's
// Metrics interface. It exists so the service layer depends on a small local
// interface rather than the observability package directly.
type dispatcherMetrics struct{ m *observability.Metrics }

func (d *dispatcherMetrics) AddDeliveryPending(ctx context.Context, delta int64) {
	d.m.DeliveryPending.Add(ctx, delta)
}
func (d *dispatcherMetrics) IncDeliveryDispatched(ctx context.Context) {
	d.m.DeliveryDispatched.Add(ctx, 1)
}
func (d *dispatcherMetrics) IncDeliveryFailed(ctx context.Context, category string) {
	d.m.DeliveryFailed.Add(ctx, 1, metric.WithAttributes(attribute.String("error_category", category)))
}

// postMetrics adapts *observability.Metrics to the post service's Metrics
// interface.
type postMetrics struct{ m *observability.Metrics }

func (p *postMetrics) IncPostsCreated(ctx context.Context) {
	p.m.PostsCreated.Add(ctx, 1)
}
func (p *postMetrics) IncRenderCacheHit(ctx context.Context) {
	p.m.RenderCacheHit.Add(ctx, 1)
}
func (p *postMetrics) IncRenderCacheMiss(ctx context.Context) {
	p.m.RenderCacheMiss.Add(ctx, 1)
}
func (p *postMetrics) IncCDNPurgeSuccess(ctx context.Context) {
	p.m.CDNPurgeSuccess.Add(ctx, 1)
}
func (p *postMetrics) IncCDNPurgeFailure(ctx context.Context) {
	p.m.CDNPurgeFailure.Add(ctx, 1)
}
func (p *postMetrics) IncCDNPurgeSkipped(ctx context.Context) {
	p.m.CDNPurgeSkipped.Add(ctx, 1)
}
