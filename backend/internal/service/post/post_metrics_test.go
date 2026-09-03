package post

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"markpost/internal/domain/post"
)

// recordingMetrics captures the render-cache counters; the remaining
// interface methods are inert for these tests.
type recordingMetrics struct {
	cacheHits   atomic.Int64
	cacheMisses atomic.Int64
}

func (r *recordingMetrics) IncPostsCreated(context.Context)    {}
func (r *recordingMetrics) IncRenderCacheHit(context.Context)  { r.cacheHits.Add(1) }
func (r *recordingMetrics) IncRenderCacheMiss(context.Context) { r.cacheMisses.Add(1) }
func (r *recordingMetrics) IncCDNPurgeSuccess(context.Context) {}
func (r *recordingMetrics) IncCDNPurgeFailure(context.Context) {}
func (r *recordingMetrics) IncCDNPurgeSkipped(context.Context) {}

// stubRepo serves one post from memory; only the read path is exercised.
type stubRepo struct {
	post.Repository
	p *post.Post
}

func (s *stubRepo) GetByQID(_ context.Context, _ string) (*post.Post, error) {
	return s.p, nil
}

func newMetricsTestService(cache renderCache, m Metrics) *Service {
	return &Service{
		postRepo: &stubRepo{p: &post.Post{
			ID:        1,
			QID:       "p-abc123",
			Title:     "Title",
			Body:      "Body",
			CreatedAt: time.Unix(1700000000, 0).UTC(),
		}},
		md:        newGoldmark(),
		sanitizer: newPostHTMLSanitizer(),
		minifier:  newHTMLMinifier(),
		cache:     cache,
		metrics:   m,
	}
}

// waitFlushed drains ristretto's buffered Set so a subsequent Get observes the
// entry: a cache fill is applied asynchronously and the second render must see
// it for the hit count to be deterministic.
func waitFlushed(svc *Service) {
	if rc, ok := svc.cache.(*ristrettoCache); ok {
		rc.c.Wait()
	}
}

func TestRenderMetrics_ColdThenWarmCountsOneMissOneHit(t *testing.T) {
	metrics := &recordingMetrics{}
	c, err := newRistrettoCache(1<<20, 10000, 64)
	if err != nil {
		t.Fatalf("ristretto cache: %v", err)
	}
	svc := newMetricsTestService(c, metrics)
	ctx := context.Background()

	if _, _, _, _, err := svc.RenderPostHTML(ctx, "p-abc123"); err != nil {
		t.Fatalf("cold render: %v", err)
	}
	if got := metrics.cacheMisses.Load(); got != 1 {
		t.Fatalf("after cold render: misses = %d, want 1", got)
	}
	if got := metrics.cacheHits.Load(); got != 0 {
		t.Fatalf("after cold render: hits = %d, want 0", got)
	}

	waitFlushed(svc)
	if _, _, _, _, err := svc.RenderPostHTML(ctx, "p-abc123"); err != nil {
		t.Fatalf("warm render: %v", err)
	}
	if got := metrics.cacheHits.Load(); got != 1 {
		t.Errorf("after warm render: hits = %d, want 1", got)
	}
	if got := metrics.cacheMisses.Load(); got != 1 {
		t.Errorf("warm render must not add a miss, misses = %d", got)
	}

	// The raw variant shares the counters on its own key: a miss, then a hit.
	if _, _, _, _, err := svc.GetPostMarkdown(ctx, "p-abc123"); err != nil {
		t.Fatalf("cold raw render: %v", err)
	}
	if got := metrics.cacheMisses.Load(); got != 2 {
		t.Errorf("after cold raw render: misses = %d, want 2", got)
	}
	waitFlushed(svc)
	if _, _, _, _, err := svc.GetPostMarkdown(ctx, "p-abc123"); err != nil {
		t.Fatalf("warm raw render: %v", err)
	}
	if got := metrics.cacheHits.Load(); got != 2 {
		t.Errorf("after warm raw render: hits = %d, want 2", got)
	}
}

func TestRenderMetrics_DisabledCacheCountsMissesOnly(t *testing.T) {
	metrics := &recordingMetrics{}
	svc := newMetricsTestService(noopCache{}, metrics)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if _, _, _, _, err := svc.RenderPostHTML(ctx, "p-abc123"); err != nil {
			t.Fatalf("render %d: %v", i, err)
		}
	}
	if got := metrics.cacheMisses.Load(); got != 2 {
		t.Errorf("disabled cache: misses = %d, want 2 (hit rate must read 0%%)", got)
	}
	if got := metrics.cacheHits.Load(); got != 0 {
		t.Errorf("disabled cache: hits = %d, want 0", got)
	}
}
