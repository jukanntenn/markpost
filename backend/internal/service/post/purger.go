package post

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"markpost/internal/config"
)

// Purger invalidates a post's cached responses at the CDN edge. The production
// implementation issues a best-effort Cloudflare cache-tag purge; self-hosted
// deployments without Cloudflare use a no-op purger and rely on natural TTL
// expiry. Purge is always best-effort: a failure must not fail the delete that
// triggered it.
type Purger interface {
	// PurgePost asks the CDN to invalidate the post-<qid> cache tag for the
	// given post. Implementations must be safe to call from a background
	// goroutine and must never panic on error.
	PurgePost(ctx context.Context, qid string)
}

// noopPurger does nothing. Used when Cloudflare is not configured.
type noopPurger struct{}

func (noopPurger) PurgePost(_ context.Context, _ string) {}

// cloudflarePurger issues a cache-tag purge against the Cloudflare API. The
// tag post-<qid> is set on every HTML/raw response by the RenderPost handler,
// so one call invalidates both variants regardless of Accept-Encoding entries.
type cloudflarePurger struct {
	apiToken string
	zoneID   string
	client   *http.Client
	endpoint string
}

func newCloudflarePurger(cfg config.CloudflareConfig) *cloudflarePurger {
	return &cloudflarePurger{
		apiToken: cfg.APIToken,
		zoneID:   cfg.ZoneID,
		client:   &http.Client{Timeout: 10 * time.Second},
		endpoint: fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/purge_cache", cfg.ZoneID),
	}
}

func (p *cloudflarePurger) PurgePost(ctx context.Context, qid string) {
	if p.apiToken == "" || p.zoneID == "" {
		return
	}
	tag := "post-" + sanitizeCacheTag(qid)
	body, err := json.Marshal(map[string][]string{"tags": {tag}})
	if err != nil {
		log.Printf("cdn purge: marshal body for qid %q: %v", qid, err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		log.Printf("cdn purge: build request for qid %q: %v", qid, err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+p.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		log.Printf("cdn purge: request for qid %q failed: %v", qid, err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		log.Printf("cdn purge: qid %q returned HTTP %d", qid, resp.StatusCode)
	}
}

// sanitizeCacheTag strips characters that could break the JSON body or allow
// header injection. QIDs are already constrained (p-<nanoid>), but this guards
// the boundary so a malformed QID cannot construct a malicious tag.
func sanitizeCacheTag(qid string) string {
	qid = strings.ReplaceAll(qid, "\"", "")
	qid = strings.ReplaceAll(qid, "\\", "")
	qid = strings.ReplaceAll(qid, "\n", "")
	qid = strings.ReplaceAll(qid, "\r", "")
	return qid
}

// newPurger builds the CDN purger from config: a Cloudflare purger when both an
// API token and zone ID are configured, otherwise a no-op.
func newPurger() Purger {
	cfg := config.Get().Cloudflare
	if cfg.APIToken == "" || cfg.ZoneID == "" {
		return noopPurger{}
	}
	return newCloudflarePurger(cfg)
}
