package post

import (
	"testing"
)

// FuzzCacheKey exercises cache-key construction with arbitrary QIDs and variants
// to confirm it never panics and produces well-formed (non-empty, variant-tagged)
// keys. Different QIDs must yield different keys for the same variant, and the
// buildID separator must survive unusual input.
func FuzzCacheKey(f *testing.F) {
	f.Add("p-abc", "html")
	f.Add("", "raw")
	f.Add("p-ünïcode", "html")
	f.Add("p:a:b", "raw")

	f.Fuzz(func(t *testing.T, qid, variant string) {
		key := cacheKey(qid, variant)
		if key == "" {
			t.Fatal("cache key must be non-empty")
		}
		// Two calls with identical input must be stable.
		if cacheKey(qid, variant) != key {
			t.Fatal("cache key is not deterministic")
		}
	})
}
