package observability

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// The instruments must record under the names documented in observability.md
// §Metric inventory — the spec table is the operator's jq query contract, and
// the five cache/purge counters are deliberately attribute-free (one series
// per outcome, MRFC 2026-09-03-cache-purge-observability).
func TestNewMetrics_InstrumentNamesCountsAndAttributes(t *testing.T) {
	reader := metric.NewManualReader()
	prev := otel.GetMeterProvider()
	otel.SetMeterProvider(metric.NewMeterProvider(metric.WithReader(reader)))
	t.Cleanup(func() { otel.SetMeterProvider(prev) })

	m := NewMetrics()
	ctx := context.Background()
	m.PostsCreated.Add(ctx, 1)
	m.AuthLoginSuccess.Add(ctx, 1)
	m.AuthLoginFailure.Add(ctx, 1)
	m.TokenRefresh.Add(ctx, 1)
	m.DeliveryPending.Add(ctx, 2)
	m.DeliveryDispatched.Add(ctx, 1)
	m.DeliveryFailed.Add(ctx, 1)
	m.RenderCacheHit.Add(ctx, 3)
	m.RenderCacheMiss.Add(ctx, 4)
	m.CDNPurgeSuccess.Add(ctx, 5)
	m.CDNPurgeFailure.Add(ctx, 6)
	m.CDNPurgeSkipped.Add(ctx, 7)

	want := map[string]int64{
		"markpost.posts.created_total":       1,
		"markpost.auth.login_success_total":  1,
		"markpost.auth.login_failure_total":  1,
		"markpost.auth.token_refresh_total":  1,
		"markpost.delivery.pending":          2,
		"markpost.delivery.dispatched_total": 1,
		"markpost.delivery.failed_total":     1,
		"markpost.render_cache.hit_total":    3,
		"markpost.render_cache.miss_total":   4,
		"markpost.cdn.purge_success_total":   5,
		"markpost.cdn.purge_failure_total":   6,
		"markpost.cdn.purge_skipped_total":   7,
	}

	attributeFree := map[string]bool{
		"markpost.render_cache.hit_total":  true,
		"markpost.render_cache.miss_total": true,
		"markpost.cdn.purge_success_total": true,
		"markpost.cdn.purge_failure_total": true,
		"markpost.cdn.purge_skipped_total": true,
	}

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("collect: %v", err)
	}
	got := map[string]metricdata.Metrics{}
	for _, sm := range rm.ScopeMetrics {
		for _, met := range sm.Metrics {
			got[met.Name] = met
		}
	}

	for name, wantVal := range want {
		met, ok := got[name]
		if !ok {
			t.Errorf("metric %q not recorded", name)
			continue
		}
		sum, ok := met.Data.(metricdata.Sum[int64])
		if !ok {
			t.Errorf("metric %q is %T, want Sum[int64]", name, met.Data)
			continue
		}
		var total int64
		for _, dp := range sum.DataPoints {
			total += dp.Value
			if attributeFree[name] && dp.Attributes.Len() != 0 {
				t.Errorf("metric %q carries attributes %v, want none", name, dp.Attributes.ToSlice())
			}
		}
		if total != wantVal {
			t.Errorf("metric %q = %d, want %d", name, total, wantVal)
		}
	}
}
