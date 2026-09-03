package post

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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

// PurgeMetrics is the subset of the observability instruments the purger
// records. One counter per outcome (no attributes), so a purge attempt is
// derivable as success + failure and "skipped" always means "not attempted"
// (MRFC 2026-09-03-cache-purge-observability).
type PurgeMetrics interface {
	IncCDNPurgeSuccess(ctx context.Context)
	IncCDNPurgeFailure(ctx context.Context)
	IncCDNPurgeSkipped(ctx context.Context)
}

// noopPurger does nothing but count the skip. Used when Cloudflare is not
// configured: the CDN copy falls back to natural TTL expiry, so a steadily
// climbing skip counter is the expected steady state there, not a fault.
type noopPurger struct{ metrics PurgeMetrics }

func (p noopPurger) PurgePost(ctx context.Context, _ string) {
	if p.metrics != nil {
		p.metrics.IncCDNPurgeSkipped(ctx)
	}
}

// cloudflarePurger issues a cache-tag purge against the Cloudflare API. The
// tag post-<qid> is set on every HTML/raw response by the RenderPost handler,
// so one call invalidates both variants regardless of Accept-Encoding entries.
type cloudflarePurger struct {
	apiToken string
	zoneID   string
	client   *http.Client
	endpoint string
	metrics  PurgeMetrics
}

func newCloudflarePurger(cfg config.CloudflareConfig, metrics PurgeMetrics) *cloudflarePurger {
	if metrics == nil {
		metrics = noopMetrics{}
	}
	return &cloudflarePurger{
		apiToken: cfg.APIToken,
		zoneID:   cfg.ZoneID,
		client:   &http.Client{Timeout: 10 * time.Second},
		endpoint: fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/purge_cache", cfg.ZoneID),
		metrics:  metrics,
	}
}

func (p *cloudflarePurger) PurgePost(ctx context.Context, qid string) {
	if p.apiToken == "" || p.zoneID == "" {
		if p.metrics != nil {
			p.metrics.IncCDNPurgeSkipped(ctx)
		}
		return
	}
	tag := "post-" + sanitizeCacheTag(qid)
	body, err := json.Marshal(map[string][]string{"tags": {tag}})
	if err != nil {
		slog.WarnContext(ctx, "cdn purge: marshal body failed", "qid", qid, "error", err)
		p.metrics.IncCDNPurgeFailure(ctx)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		slog.WarnContext(ctx, "cdn purge: build request failed", "qid", qid, "error", err)
		p.metrics.IncCDNPurgeFailure(ctx)
		return
	}
	req.Header.Set("Authorization", "Bearer "+p.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		slog.WarnContext(ctx, "cdn purge: request failed", "qid", qid, "error", err)
		p.metrics.IncCDNPurgeFailure(ctx)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		slog.WarnContext(ctx, "cdn purge: unexpected status", "qid", qid, "status", resp.StatusCode)
		p.metrics.IncCDNPurgeFailure(ctx)
		return
	}
	p.metrics.IncCDNPurgeSuccess(ctx)
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
// API token and zone ID are configured, otherwise a no-op. Both record their
// outcome through the given recorder (nil degrades to a no-op recorder).
func newPurger(metrics PurgeMetrics) Purger {
	cfg := config.Get().Cloudflare
	if cfg.APIToken == "" || cfg.ZoneID == "" {
		return noopPurger{metrics: metrics}
	}
	return newCloudflarePurger(cfg, metrics)
}
