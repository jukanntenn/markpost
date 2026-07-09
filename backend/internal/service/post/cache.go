package post

import (
	"fmt"
	"time"

	"markpost/internal/web"

	"github.com/dgraph-io/ristretto"
)

// renderResult is the cached payload for a rendered post variant: the title,
// the rendered body (minified HTML, or the raw markdown response), the response
// ETag, and the post's creation time (for Last-Modified). Storing them together
// means a cache hit returns everything with no hashing and no DB read on the
// hot path.
type renderResult struct {
	title     string
	body      string
	etag      string
	createdAt time.Time
}

// renderCache abstracts the in-process render cache so it can be disabled or
// swapped (e.g. a no-op or a fake) without touching the service. ristretto's
// Cache matches this surface, so the production implementation is a thin
// wrapper around it.
type renderCache interface {
	Get(key string) (renderResult, bool)
	Set(key string, value renderResult, cost int64) bool
	// Delete removes the key and blocks until the deletion (and any prior
	// buffered Set of the same key) is fully applied, so a subsequent Get
	// cannot observe a stale value re-admitted by a pending Set. This makes
	// origin-cache invalidation synchronous, which the delete path requires.
	Delete(key string)
	Close()
}

// cacheKey builds the namespaced render-cache key for a QID and variant.
// buildID rotates the whole namespace on release; the variant suffix keeps the
// HTML and raw entries from colliding.
func cacheKey(qid, variant string) string {
	return qid + ":" + web.BuildID() + ":" + variant
}

// ristrettoCache wraps *ristretto.Cache as a renderCache.
type ristrettoCache struct {
	c *ristretto.Cache
}

func newRistrettoCache(maxCost, numCounters, bufferItems int64) (*ristrettoCache, error) {
	if numCounters <= 0 {
		numCounters = 100000
	}
	if bufferItems <= 0 {
		bufferItems = 64
	}
	c, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: numCounters,
		MaxCost:     maxCost,
		BufferItems: bufferItems,
	})
	if err != nil {
		return nil, fmt.Errorf("new render cache: %w", err)
	}
	return &ristrettoCache{c: c}, nil
}

func (r *ristrettoCache) Get(key string) (renderResult, bool) {
	v, ok := r.c.Get(key)
	if !ok {
		return renderResult{}, false
	}
	rr, ok := v.(renderResult)
	if !ok {
		return renderResult{}, false
	}
	return rr, true
}

func (r *ristrettoCache) Set(key string, value renderResult, cost int64) bool {
	return r.c.Set(key, value, cost)
}

func (r *ristrettoCache) Delete(key string) {
	r.c.Del(key)
	// Ristretto applies Set/Del asynchronously through a buffered channel: a
	// pending Set that was buffered before this Del could otherwise re-admit
	// the entry after Del's synchronous store removal. Wait drains the buffer
	// so the deletion is durable before the caller proceeds.
	r.c.Wait()
}

func (r *ristrettoCache) Close() {
	r.c.Close()
}

// noopCache is used when the render cache is disabled via config. It always
// misses and stores nothing, so every request renders fresh — the behaviour of
// the pre-cache serving path.
type noopCache struct{}

func (noopCache) Get(_ string) (renderResult, bool)          { return renderResult{}, false }
func (noopCache) Set(_ string, _ renderResult, _ int64) bool { return true }
func (noopCache) Delete(_ string)                            {}
func (noopCache) Close()                                     {}
