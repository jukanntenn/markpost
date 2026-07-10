package post

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"markpost/internal/domain/post"
)

// benchRepo is an in-memory post.Repository that only answers GetByQID, so the
// render pipeline (goldmark + neutralize + bluemonday + minify) is measured in
// isolation from any DB I/O. The embedded interface leaves the unused methods
// as zero-value nil; they are never called on the render path.
type benchRepo struct {
	post.Repository
	p *post.Post
}

func (r *benchRepo) GetByQID(_ context.Context, _ string) (*post.Post, error) {
	return r.p, nil
}

// benchCountingRepo wraps benchRepo to count GetByQID calls, used by the
// singleflight benchmark to assert concurrent same-QID misses collapse to one
// render.
type benchCountingRepo struct {
	*benchRepo
	gets atomic.Int64
}

func (r *benchCountingRepo) GetByQID(ctx context.Context, qid string) (*post.Post, error) {
	r.gets.Add(1)
	return r.benchRepo.GetByQID(ctx, qid)
}

// benchBody builds a deterministic markdown body of approximately n bytes,
// exercising the features the render pipeline must handle (headings, lists,
// fenced code, tables, links) so the cost reflects real posts rather than a
// flat paragraph.
func benchBody(n int) string {
	var b strings.Builder
	block := "## Section %d\n\n" +
		"- alpha item with some words\n" +
		"- beta item with some words\n" +
		"- gamma item with some words\n\n" +
		"| col1 | col2 |\n|------|------|\n| a%[1]d | b%[1]d |\n\n" +
		"```go\nfunc example%d() { return %d }\n```\n\n" +
		"A paragraph referencing [a link](https://example.com/%[1]d) and some\n" +
		"`inline code` with ~~strikethrough~~ text to round it out.\n\n"
	i := 0
	for b.Len() < n {
		fmt.Fprintf(&b, block, i, i, i, i)
		i++
	}
	if b.Len() > n {
		return b.String()[:n]
	}
	return b.String()
}

var (
	benchBodyShort  = benchBody(32)
	benchBodyMedium = benchBody(256)
	benchBodyLong   = benchBody(32 * 1024)
)

var benchBodies = []struct {
	name string
	body string
}{
	{"Short", benchBodyShort},
	{"Medium", benchBodyMedium},
	{"Long", benchBodyLong},
}

// benchServiceCold builds a Service whose cache is a noopCache, so every
// RenderPostHTML call exercises the full DB-read + render pipeline (cold miss).
func benchServiceCold(b *testing.B, body string) *Service {
	b.Helper()
	return &Service{
		postRepo:  &benchRepo{p: benchPost(body)},
		md:        newGoldmark(),
		sanitizer: newPostHTMLSanitizer(),
		minifier:  newHTMLMinifier(),
		cache:     noopCache{},
		purger:    noopPurger{},
	}
}

// benchServiceWarm builds a Service with a real ristretto cache, for measuring
// the cache-hit path. The cache is sized generously so entries are not evicted
// mid-benchmark.
func benchServiceWarm(b *testing.B, body string) (*Service, *ristrettoCache) {
	b.Helper()
	cache, err := newRistrettoCache(1<<24, 10000, 64)
	if err != nil {
		b.Fatalf("newRistrettoCache: %v", err)
	}
	b.Cleanup(cache.Close)
	svc := &Service{
		postRepo:  &benchRepo{p: benchPost(body)},
		md:        newGoldmark(),
		sanitizer: newPostHTMLSanitizer(),
		minifier:  newHTMLMinifier(),
		cache:     cache,
		purger:    noopPurger{},
	}
	return svc, cache
}

func benchPost(body string) *post.Post {
	return &post.Post{
		ID:        1,
		QID:       "p-bench",
		Title:     "Bench Title",
		Body:      body,
		CreatedAt: time.Now(),
		UserID:    1,
	}
}

// BenchmarkRenderPostHTML measures the full render pipeline on a cache miss:
// DB read (in-memory) + goldmark convert + raw-HTML neutralize + bluemonday
// sanitize + HTML minify. It isolates the per-request cost paid when the render
// cache is cold (process start, release deploy, first-touch after eviction).
func BenchmarkRenderPostHTML(b *testing.B) {
	for _, c := range benchBodies {
		b.Run(c.name, func(b *testing.B) {
			svc := benchServiceCold(b, c.body)
			ctx := context.Background()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, _, _, _, err := svc.RenderPostHTML(ctx, "p-bench"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkRenderCacheHit measures the ristretto Get fast path: a warm cache
// returns the stored render result with no goldmark/bluemonday/minify work.
// This is the latency readers see once a post has been rendered within the
// process lifetime.
func BenchmarkRenderCacheHit(b *testing.B) {
	for _, c := range benchBodies {
		b.Run(c.name, func(b *testing.B) {
			svc, _ := benchServiceWarm(b, c.body)
			ctx := context.Background()
			if _, _, _, _, err := svc.RenderPostHTML(ctx, "p-bench"); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, _, _, _, err := svc.RenderPostHTML(ctx, "p-bench"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkSingleflightCollapse measures the effective throughput when many
// goroutines request the same cold QID concurrently: singleflight must collapse
// them to far fewer renders than goroutines. Each iteration uses a fresh QID
// (the benchRepo returns the same post regardless) so every iteration is a
// genuine cold miss. The collapse is not asserted as exactly 1: ristretto's
// async Set means a follower arriving just after the leader finishes may miss
// the in-flight singleflight window AND the not-yet-visible cache fill, becoming
// a second leader. The benchmark instead reports the average DB reads per burst
// (a healthy collapse is ~1; a broken singleflight would approach goroutines)
// and fails only if collapsing is absent (reads > half the goroutines), which is
// the regression this benchmark guards against.
func BenchmarkSingleflightCollapse(b *testing.B) {
	const goroutines = 50
	repo := &benchCountingRepo{benchRepo: &benchRepo{p: benchPost(benchBodyMedium)}}
	cache, err := newRistrettoCache(1<<24, 10000, 64)
	if err != nil {
		b.Fatalf("newRistrettoCache: %v", err)
	}
	b.Cleanup(cache.Close)
	svc := &Service{
		postRepo:  repo,
		md:        newGoldmark(),
		sanitizer: newPostHTMLSanitizer(),
		minifier:  newHTMLMinifier(),
		cache:     cache,
		purger:    noopPurger{},
	}
	ctx := context.Background()
	start := make(chan struct{})
	var wg sync.WaitGroup
	var totalReads atomic.Int64
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qid := fmt.Sprintf("p-bench-%d", i)
		repo.gets.Store(0)
		wg.Add(goroutines)
		for g := 0; g < goroutines; g++ {
			go func() {
				defer wg.Done()
				<-start
				if _, _, _, _, err := svc.RenderPostHTML(ctx, qid); err != nil {
					b.Error(err)
				}
			}()
		}
		close(start)
		wg.Wait()
		start = make(chan struct{})
		reads := repo.gets.Load()
		totalReads.Add(reads)
		if reads > goroutines/2 {
			b.Fatalf("iteration %d: singleflight collapse absent (%d reads for %d goroutines)", i, reads, goroutines)
		}
	}
	b.ReportMetric(float64(totalReads.Load())/float64(b.N), "reads/burst")
}

// BenchmarkETag isolates the xxhash64 ETag computation over a rendered body, the
// cost paid once per cache miss (the hot path serves the stored ETag directly).
func BenchmarkETag(b *testing.B) {
	for _, c := range benchBodies {
		b.Run(c.name, func(b *testing.B) {
			svc := benchServiceCold(b, c.body)
			_, html, _, _, err := svc.RenderPostHTML(context.Background(), "p-bench")
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = etagHex(html)
			}
		})
	}
}

// BenchmarkMinify isolates the tdewolff HTML minification cost over a sanitized
// render, independent of goldmark and bluemonday.
func BenchmarkMinify(b *testing.B) {
	for _, c := range benchBodies {
		b.Run(c.name, func(b *testing.B) {
			svc := benchServiceCold(b, c.body)
			ctx := context.Background()
			p, err := svc.postRepo.GetByQID(ctx, "p-bench")
			if err != nil {
				b.Fatal(err)
			}
			var rendered strings.Builder
			if err := svc.md.Convert([]byte(p.Body), &rendered); err != nil {
				b.Fatal(err)
			}
			sanitized := svc.sanitizer.Sanitize(neutralizeRawHTMLElements(rendered.String()))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := svc.minifyHTML(sanitized); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
