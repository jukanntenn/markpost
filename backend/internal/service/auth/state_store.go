package auth

import (
	"time"

	"github.com/dgraph-io/ristretto"
)

// ristrettoOAuthStateStore is the OAuthStateStore backed by a ristretto cache.
// State→verifier entries are short-lived (10 min) and one-time: Consume does a
// Get followed by a Del. The Get+Del is not atomic, but markpost is a
// single-instance deployment and a state is only ever consumed by the one
// /oauth/login callback that follows its /oauth/url generation, so concurrent
// consumption of the same state does not occur in practice.
type ristrettoOAuthStateStore struct {
	cache *ristretto.Cache
}

// NewRistrettoOAuthStateStore builds an OAuthStateStore over a small ristretto
// cache tuned for the low cardinality of in-flight OAuth flows.
func NewRistrettoOAuthStateStore() (OAuthStateStore, error) {
	c, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 10000,
		MaxCost:     10000,
		BufferItems: 64,
	})
	if err != nil {
		return nil, err
	}
	return &ristrettoOAuthStateStore{cache: c}, nil
}

func (s *ristrettoOAuthStateStore) Save(state string, entry oauthStateEntry, ttl time.Duration) {
	s.cache.SetWithTTL(state, entry, 1, ttl)
	s.cache.Wait()
}

func (s *ristrettoOAuthStateStore) Consume(state string) (oauthStateEntry, bool) {
	v, ok := s.cache.Get(state)
	if !ok {
		return oauthStateEntry{}, false
	}
	s.cache.Del(state)
	entry, ok := v.(oauthStateEntry)
	if !ok {
		return oauthStateEntry{}, false
	}
	return entry, true
}
