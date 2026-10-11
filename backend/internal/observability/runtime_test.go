package observability

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// startRuntimeMetrics must surface the contrib runtime family (go.*) on the
// global provider and must not resurrect the deleted hand-rolled
// process.runtime.go.* instruments (MRFC 2026-10-11-metrics-wiring-runtime-source).
// go.memory.limit is deliberately not asserted: the upstream callback only
// observes it when a GOMEMLIMIT exists.
func TestStartRuntimeMetrics_ContribFamily(t *testing.T) {
	reader := metric.NewManualReader()
	prev := otel.GetMeterProvider()
	mp := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(mp)
	t.Cleanup(func() {
		otel.SetMeterProvider(prev)
		_ = mp.Shutdown(context.Background())
	})

	// startRuntimeMetrics logs and swallows a registration failure, so the
	// assertion is purely on the collected instrument family.
	startRuntimeMetrics()

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("collect: %v", err)
	}
	got := map[string]bool{}
	for _, sm := range rm.ScopeMetrics {
		for _, met := range sm.Metrics {
			got[met.Name] = true
		}
	}

	for _, name := range []string{
		"go.memory.used",
		"go.memory.allocated",
		"go.memory.allocations",
		"go.memory.gc.goal",
		"go.goroutine.count",
		"go.processor.limit",
		"go.config.gogc",
	} {
		if !got[name] {
			t.Errorf("runtime metric %q not recorded", name)
		}
	}
	for _, name := range []string{
		"process.runtime.go.goroutines",
		"process.runtime.go.mem.heap_alloc",
		"process.runtime.go.mem.heap_sys",
		"process.runtime.go.gc.count",
	} {
		if got[name] {
			t.Errorf("hand-rolled runtime metric %q still recorded", name)
		}
	}
}
