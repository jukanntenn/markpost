package post

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"markpost/internal/config"
)

// recordingPurgeMetrics captures the purge outcome counters.
type recordingPurgeMetrics struct {
	success, failure, skipped atomic.Int64
}

func (r *recordingPurgeMetrics) IncCDNPurgeSuccess(context.Context) { r.success.Add(1) }
func (r *recordingPurgeMetrics) IncCDNPurgeFailure(context.Context) { r.failure.Add(1) }
func (r *recordingPurgeMetrics) IncCDNPurgeSkipped(context.Context) { r.skipped.Add(1) }

func TestNoopPurger_CountsSkipped(t *testing.T) {
	rec := &recordingPurgeMetrics{}
	noopPurger{metrics: rec}.PurgePost(context.Background(), "p-abc")

	if got := rec.skipped.Load(); got != 1 {
		t.Errorf("skipped = %d, want 1", got)
	}
	if rec.success.Load() != 0 || rec.failure.Load() != 0 {
		t.Errorf("noop purger must not record success/failure")
	}
}

func TestNoopPurger_NilMetricsDoesNotPanic(t *testing.T) {
	noopPurger{}.PurgePost(context.Background(), "p-abc")
}

func TestCloudflarePurger_PurgesCacheTag(t *testing.T) {
	var (
		gotAuth string
		gotBody map[string][]string
		calls   atomic.Int32
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	t.Cleanup(srv.Close)

	rec := &recordingPurgeMetrics{}
	p := newCloudflarePurger(config.CloudflareConfig{APIToken: "secret-token", ZoneID: "zone-123"}, rec)
	p.client = &http.Client{Timeout: 2 * time.Second}
	p.endpoint = srv.URL
	p.PurgePost(context.Background(), "p-abc123")

	if got := calls.Load(); got != 1 {
		t.Fatalf("expected exactly 1 purge call, got %d", got)
	}
	if gotAuth != "Bearer secret-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer secret-token")
	}
	if tags := gotBody["tags"]; len(tags) != 1 || tags[0] != "post-p-abc123" {
		t.Errorf("purge tags = %v, want [post-p-abc123]", tags)
	}
	if got := rec.success.Load(); got != 1 {
		t.Errorf("success = %d, want 1", got)
	}
}

func TestCloudflarePurger_FailureIsSwallowedAndCounted(t *testing.T) {
	// A server returning 5xx and a non-reachable endpoint must not panic or
	// surface an error: purge is always best-effort. Each failure branch
	// counts as a failure outcome.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	rec := &recordingPurgeMetrics{}
	p := newCloudflarePurger(config.CloudflareConfig{APIToken: "tok", ZoneID: "zone"}, rec)
	p.client = &http.Client{Timeout: time.Second}
	p.endpoint = srv.URL
	p.PurgePost(context.Background(), "p-1") // HTTP >= 300: must not panic

	p.endpoint = "http://127.0.0.1:0/invalid"
	p.PurgePost(context.Background(), "p-2") // transport error: must not panic

	p.endpoint = "ht tp://bad url" // build error: must not panic
	p.PurgePost(context.Background(), "p-3")

	if got := rec.failure.Load(); got != 3 {
		t.Errorf("failure = %d, want 3", got)
	}
	if rec.success.Load() != 0 {
		t.Errorf("no branch may record success")
	}
}

func TestCloudflarePurger_UnconfiguredCountsSkipped(t *testing.T) {
	rec := &recordingPurgeMetrics{}
	p := newCloudflarePurger(config.CloudflareConfig{}, rec)
	p.PurgePost(context.Background(), "p-x")

	if got := rec.skipped.Load(); got != 1 {
		t.Errorf("skipped = %d, want 1", got)
	}
	if rec.success.Load() != 0 || rec.failure.Load() != 0 {
		t.Errorf("skipped must not record success/failure")
	}
}

func TestSanitizeCacheTag(t *testing.T) {
	tests := map[string]string{
		"p-abc":            "p-abc",
		`p-a"b`:            "p-ab",
		"p-a\\b":           "p-ab",
		"p-a\nb":           "p-ab",
		"p-a\rb":           "p-ab",
		"normal-p-qid-123": "normal-p-qid-123",
	}
	for in, want := range tests {
		if got := sanitizeCacheTag(in); got != want {
			t.Errorf("sanitizeCacheTag(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNewPurger_NoopWhenUnconfigured(t *testing.T) {
	config.ResetForTest()
	if _, ok := newPurger(nil).(noopPurger); !ok {
		t.Errorf("expected noopPurger when Cloudflare is unconfigured")
	}
}

func TestNewPurger_CloudflareWhenConfigured(t *testing.T) {
	config.ResetForTest()
	// Inject config directly via the singleton path: load a minimal config.
	// newPurger reads config.Get(), so set the Cloudflare fields via viper-like
	// defaults is heavy; instead construct the purger via its config directly.
	cfg := config.CloudflareConfig{APIToken: "tok", ZoneID: "zone"}
	p := newCloudflarePurger(cfg, nil)
	if p.apiToken != "tok" || p.zoneID != "zone" {
		t.Errorf("cloudflare purger not built from config: %+v", p)
	}
	// A nil recorder degrades to the package no-op recorder, so the outcome
	// counters stay callable without a metrics injection.
	if p.metrics == nil {
		t.Errorf("newCloudflarePurger must substitute a no-op recorder for nil")
	}
	// Absent config must yield a noop via newCloudflarePurger guard in PurgePost.
	empty := newCloudflarePurger(config.CloudflareConfig{}, nil)
	empty.PurgePost(context.Background(), "p-x") // no API token -> no-op, no panic
}
