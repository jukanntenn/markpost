package post

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"markpost/internal/domain/post"
	"markpost/internal/infra"
	"markpost/internal/service"
)

// newServiceWithCache builds a post.Service wired to a real ristretto cache so
// the hit/miss/singleflight paths are exercised against the production cache
// implementation rather than the noop fallback used when config is unloaded.
func newServiceWithCache(t *testing.T) (*Service, *infra.PostRepository, *ristrettoCache) {
	t.Helper()
	db := infra.SetupTestDB(t)
	repo := infra.NewPostRepository(db)
	cache, err := newRistrettoCache(1<<20, 10000, 64)
	if err != nil {
		t.Fatalf("newRistrettoCache: %v", err)
	}
	t.Cleanup(cache.Close)
	svc := &Service{
		postRepo:  repo,
		md:        newGoldmark(),
		sanitizer: newPostHTMLSanitizer(),
		minifier:  newHTMLMinifier(),
		cache:     cache,
		purger:    noopPurger{},
	}
	return svc, repo.(*infra.PostRepository), cache
}

func TestRenderCache_HitReturnsStoredValue(t *testing.T) {
	svc, repo, _ := newServiceWithCache(t)
	ctx := context.Background()
	created, _ := repo.Create(ctx, "Cached", "# Hello\n\nworld", 1)

	title1, html1, etag1, created1, err := svc.RenderPostHTML(ctx, created.QID)
	if err != nil {
		t.Fatalf("first render: %v", err)
	}
	if title1 != "Cached" || html1 == "" || etag1 == "" || created1.IsZero() {
		t.Fatalf("first render returned incomplete result")
	}

	title2, html2, etag2, created2, err := svc.RenderPostHTML(ctx, created.QID)
	if err != nil {
		t.Fatalf("second render: %v", err)
	}
	if html1 != html2 || etag1 != etag2 || title1 != title2 || !created1.Equal(created2) {
		t.Errorf("cache hit returned different values:\nhtml1=%q html2=%q\netag1=%q etag2=%q", html1, html2, etag1, etag2)
	}
}

func TestRenderCache_RawVariantSeparateFromHTML(t *testing.T) {
	svc, repo, _ := newServiceWithCache(t)
	ctx := context.Background()
	created, _ := repo.Create(ctx, "T", "# Heading\n\npara", 1)

	_, htmlContent, htmlEtag, _, err := svc.RenderPostHTML(ctx, created.QID)
	if err != nil {
		t.Fatalf("render html: %v", err)
	}
	_, body, rawEtag, _, err := svc.GetPostMarkdown(ctx, created.QID)
	if err != nil {
		t.Fatalf("get raw: %v", err)
	}

	if htmlEtag == rawEtag {
		t.Errorf("html and raw variants must have distinct ETags, both = %q", htmlEtag)
	}
	if body == htmlContent {
		t.Errorf("raw body must differ from rendered html")
	}

	// Both cached under separate keys: a second HTML hit is unaffected by the
	// raw miss having filled its own slot.
	_, html2, htmlEtag2, _, err := svc.RenderPostHTML(ctx, created.QID)
	if err != nil {
		t.Fatalf("second html render: %v", err)
	}
	if html2 != htmlContent || htmlEtag2 != htmlEtag {
		t.Errorf("html variant not cached distinctly from raw")
	}
}

func TestRenderCache_DeletionInvalidatesBothVariants(t *testing.T) {
	svc, repo, _ := newServiceWithCache(t)
	ctx := context.Background()
	created, _ := repo.Create(ctx, "Doomed", "# bye", 1)

	if _, _, _, _, err := svc.RenderPostHTML(ctx, created.QID); err != nil {
		t.Fatalf("render html: %v", err)
	}
	if _, _, _, _, err := svc.GetPostMarkdown(ctx, created.QID); err != nil {
		t.Fatalf("get raw: %v", err)
	}

	if err := svc.DeletePostByQID(ctx, created.QID, 0); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := repo.GetByQID(ctx, created.QID); err == nil {
		t.Fatal("post should be deleted from the DB")
	}
	if _, _, _, _, err := svc.RenderPostHTML(ctx, created.QID); err == nil {
		t.Error("render html after delete should miss cache and error")
	}
	if _, _, _, _, err := svc.GetPostMarkdown(ctx, created.QID); err == nil {
		t.Error("get raw after delete should miss cache and error")
	}
}

func TestRenderCache_SingleflightCollapsesBurst(t *testing.T) {
	// Rendering is wrapped in singleflight, so N concurrent calls for the same
	// cold QID must issue exactly one DB read. A counting wrapper asserts this.
	db := infra.SetupTestDB(t)
	realRepo := infra.NewPostRepository(db)
	created, _ := realRepo.Create(context.Background(), "SF", "# concurrency", 1)
	counting := &countingRepo{Repository: realRepo, qid: created.QID}

	cache, err := newRistrettoCache(1<<20, 10000, 64)
	if err != nil {
		t.Fatalf("ristretto: %v", err)
	}
	t.Cleanup(cache.Close)
	svc := &Service{
		postRepo:  counting,
		md:        newGoldmark(),
		sanitizer: newPostHTMLSanitizer(),
		minifier:  newHTMLMinifier(),
		cache:     cache,
		purger:    noopPurger{},
	}

	const n = 50
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan error, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			<-start
			if _, _, _, _, err := svc.RenderPostHTML(context.Background(), created.QID); err != nil {
				errs <- err
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("goroutine error: %v", err)
	}

	if got := counting.gets.Load(); got != 1 {
		t.Errorf("expected exactly 1 DB GetByQID under a same-QID burst, got %d", got)
	}
}

// countingRepo wraps a post.Repository to count GetByQID calls for the
// singleflight collapse test.
type countingRepo struct {
	post.Repository
	gets atomic.Int32
	qid  string
}

func (c *countingRepo) GetByQID(ctx context.Context, qid string) (*post.Post, error) {
	c.gets.Add(1)
	return c.Repository.GetByQID(ctx, qid)
}

// recordingPurger records each PurgePost call so the delete path can be
// asserted to purge exactly once per deletion (and not at all for prune).
type recordingPurger struct {
	mu    sync.Mutex
	calls []string
}

func (r *recordingPurger) PurgePost(_ context.Context, qid string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, qid)
}

func (r *recordingPurger) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

// waitFor polls cond until it returns true or the timeout elapses, failing the
// test on timeout. Used for best-effort asynchronous assertions (e.g. the CDN
// purge goroutine).
func waitFor(t *testing.T, cond func() bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func TestDeletePostByQID_OwnerScopedAndPurges(t *testing.T) {
	t.Run("owner deletes: removes row, invalidates cache, purges once", func(t *testing.T) {
		db := infra.SetupTestDB(t)
		repo := infra.NewPostRepository(db)
		cache, err := newRistrettoCache(1<<20, 10000, 64)
		if err != nil {
			t.Fatalf("ristretto: %v", err)
		}
		t.Cleanup(cache.Close)
		purger := &recordingPurger{}
		svc := &Service{
			postRepo:  repo,
			md:        newGoldmark(),
			sanitizer: newPostHTMLSanitizer(),
			minifier:  newHTMLMinifier(),
			cache:     cache,
			purger:    purger,
		}
		ctx := context.Background()
		created, _ := repo.Create(ctx, "T", "# bye", 7)

		if _, _, _, _, err := svc.RenderPostHTML(ctx, created.QID); err != nil {
			t.Fatalf("render: %v", err)
		}

		if err := svc.DeletePostByQID(ctx, created.QID, 7); err != nil {
			t.Fatalf("delete: %v", err)
		}
		if _, err := repo.GetByQID(ctx, created.QID); err == nil {
			t.Error("row should be deleted")
		}
		// Purge is best-effort and asynchronous: poll briefly for it to land.
		waitFor(t, func() bool { return purger.Count() == 1 }, time.Second)
		if got := purger.Count(); got != 1 {
			t.Errorf("expected exactly 1 purge, got %d", got)
		}
		// Cache miss after invalidation -> the re-render must error, not serve
		// a stale cached copy.
		if _, _, _, _, err := svc.RenderPostHTML(ctx, created.QID); err == nil {
			t.Error("re-render after delete should error")
		}
	})

	t.Run("wrong owner returns NotFound and does not delete or purge", func(t *testing.T) {
		db := infra.SetupTestDB(t)
		repo := infra.NewPostRepository(db)
		cache, err := newRistrettoCache(1<<20, 10000, 64)
		if err != nil {
			t.Fatalf("ristretto: %v", err)
		}
		t.Cleanup(cache.Close)
		purger := &recordingPurger{}
		svc := &Service{
			postRepo:  repo,
			md:        newGoldmark(),
			sanitizer: newPostHTMLSanitizer(),
			minifier:  newHTMLMinifier(),
			cache:     cache,
			purger:    purger,
		}
		ctx := context.Background()
		created, _ := repo.Create(ctx, "T", "# bye", 7)

		err = svc.DeletePostByQID(ctx, created.QID, 999) // wrong owner
		if err == nil {
			t.Fatal("expected error for wrong owner")
		}
		se, ok := service.AsError(err)
		if !ok || se.Code != service.ErrNotFound {
			t.Errorf("expected not_found error, got %v", err)
		}
		if _, err := repo.GetByQID(ctx, created.QID); err != nil {
			t.Errorf("row must still exist after wrong-owner delete: %v", err)
		}
		if got := purger.Count(); got != 0 {
			t.Errorf("wrong-owner delete must not purge, got %d", got)
		}
	})
}

func TestPruneExpired_InvalidatesCacheWithoutPurging(t *testing.T) {
	db := infra.SetupTestDB(t)
	repo := infra.NewPostRepository(db)
	cache, err := newRistrettoCache(1<<20, 10000, 64)
	if err != nil {
		t.Fatalf("ristretto: %v", err)
	}
	t.Cleanup(cache.Close)
	purger := &recordingPurger{}
	svc := &Service{
		postRepo:  repo,
		md:        newGoldmark(),
		sanitizer: newPostHTMLSanitizer(),
		minifier:  newHTMLMinifier(),
		cache:     cache,
		purger:    purger,
	}
	ctx := context.Background()
	old, _ := repo.Create(ctx, "Old", "# old", 1)
	// Backdate its created_at past the retention window.
	if err := db.Model(&post.Post{}).Where("id = ?", old.ID).
		Update("created_at", time.Now().AddDate(0, 0, -10)).Error; err != nil {
		t.Fatalf("backdate: %v", err)
	}
	// Warm the cache for the soon-to-be-pruned post.
	if _, _, _, _, err := svc.RenderPostHTML(ctx, old.QID); err != nil {
		t.Fatalf("render: %v", err)
	}

	if err := svc.PruneExpired(ctx, 7, 100); err != nil {
		t.Fatalf("prune: %v", err)
	}
	if got := purger.Count(); got != 0 {
		t.Errorf("prune must not issue CDN purges, got %d", got)
	}
	// Cache invalidated: re-render errors instead of serving stale HTML.
	if _, _, _, _, err := svc.RenderPostHTML(ctx, old.QID); err == nil {
		t.Error("re-render after prune should error (cache invalidated)")
	}
}
