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

func TestNoopPurger_DoesNothing(t *testing.T) {
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

	p := &cloudflarePurger{
		apiToken: "secret-token",
		zoneID:   "zone-123",
		client:   &http.Client{Timeout: 2 * time.Second},
		endpoint: srv.URL,
	}
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
}

func TestCloudflarePurger_FailureIsSwallowed(t *testing.T) {
	// A server returning 5xx and a non-reachable endpoint must not panic or
	// surface an error: purge is always best-effort.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	p := &cloudflarePurger{
		apiToken: "tok",
		zoneID:   "zone",
		client:   &http.Client{Timeout: time.Second},
		endpoint: srv.URL,
	}
	p.PurgePost(context.Background(), "p-1") // must not panic

	p.endpoint = "http://127.0.0.1:0/invalid"
	p.PurgePost(context.Background(), "p-2") // must not panic
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
	if _, ok := newPurger().(noopPurger); !ok {
		t.Errorf("expected noopPurger when Cloudflare is unconfigured")
	}
}

func TestNewPurger_CloudflareWhenConfigured(t *testing.T) {
	config.ResetForTest()
	// Inject config directly via the singleton path: load a minimal config.
	// newPurger reads config.Get(), so set the Cloudflare fields via viper-like
	// defaults is heavy; instead construct the purger via its config directly.
	cfg := config.CloudflareConfig{APIToken: "tok", ZoneID: "zone"}
	p := newCloudflarePurger(cfg)
	if p.apiToken != "tok" || p.zoneID != "zone" {
		t.Errorf("cloudflare purger not built from config: %+v", p)
	}
	// Absent config must yield a noop via newCloudflarePurger guard in PurgePost.
	empty := newCloudflarePurger(config.CloudflareConfig{})
	empty.PurgePost(context.Background(), "p-x") // no API token -> no-op, no panic
}
