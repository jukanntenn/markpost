# Performance & Resource Optimization

This document specifies the performance and resource-optimization design for markpost's read-heavy serving path under fixed hardware and bandwidth constraints. It is the authoritative reference for the caching, compression, rate-limiting, disaster-recovery, deployment-topology, and load-testing decisions, and it records the rejected alternatives and the rationale for each technology choice so the reasoning survives review.

## Scope and Scenario

markpost's core business is **storage and distribution of Markdown content** — a notification, a temporary share, a paste — not social posts. This shapes the entire design:

- **Posts are write-once and immutable.** The `UpdatePost` capability is removed from the contract. There is no edit-after-publish. This eliminates the entire cache-invalidation problem and permits aggressive edge caching of post *bodies*.
- **Posts are short-lived.** A retention floor of 7 days is the user-experience requirement; nothing about the design assumes data lives forever.
- **The read path is the hot path.** The product is consumed by readers clicking shared links; the write path is a low-frequency authoring operation.
- **Two deployment contexts.** The project ships as self-hostable software *and* runs as an official SaaS instance. Every design choice must work in both: nothing SaaS-specific may be baked into the application code or configuration defaults.

Anything not on this hot path (admin, dashboard, delivery webhook fan-out) is out of scope for this optimization pass.

## Constraints and Workload

### Hardware envelope (SaaS reference instance)

| Resource | Limit | Notes |
|----------|-------|-------|
| CPU | 2 cores | Shared by Caddy + Go + (Next.js) inside the markpost container; Postgres runs in a sibling container |
| Memory | 2 GB | Shared across all processes |
| Disk | 40 GB | Postgres data + WAL + container layers |
| Bandwidth | 3 Mbps | **375 KB/s peak** egress |
| Monthly traffic | 1 TB | 375 KB/s sustained for ~30.86 days ≈ 1 TB |

**Link equals quota.** Saturating the 3 Mbps link for a month consumes the entire 1 TB allowance. Every byte saved is simultaneously link headroom and quota headroom — there is one budget, not two. This is why transmission-minimization dominates the design rather than CPU optimization. Self-hosted instances with more bandwidth feel this constraint less keenly but benefit from the same optimizations.

### Workload

| Dimension | Value | Source |
|-----------|-------|--------|
| Users | 10 000 | target scale |
| Post body average | 32 KB | the dominant per-page byte cost |
| Read rate | a few hundred reads/second | can spike on a shared hot post |
| Write rate | ~0.12 writes/second mean | 10 000 users × single-digit posts/day ÷ 86400 |
| Write burst cap | 10 posts/minute per user | hard application limit |
| Daily cap | 1000 posts/day per user | hard application limit |
| Post retention | ≥ 7 days | user-experience requirement |

### Derived pressure check

A 3 Mbps origin cannot serve hundreds of reads/second directly:

```
375 KB/s ÷ ~10 KB per gzip'd page ≈ 25 origin responses/second
```

Two consequences fall out of this:

1. **An uncached origin is physically incapable of carrying the read load.** A CDN is therefore a precondition for the SaaS reference instance to function at scale — not an optional enhancement. Self-hosted instances on fatter pipes can run without one but accept higher origin load.
2. **The write rate is negligible.** At ~0.12 writes/second, Postgres ingest is never the bottleneck.

## Current State of the Code

These facts were verified against the codebase and define the gap this spec closes.

### The serving path, as it stands today

Every `GET /:id` (registered at `cmd/server/main.go:261`, a public unauthenticated route) executes this pipeline, fresh, with no caching at any layer:

1. **Postgres read** — `PostRepository.GetByQID` (`internal/infra/post_repo.go:64`) issues `WHERE qid = ?`. Sub-millisecond on the QID unique index.
2. **goldmark render** — `s.md.Convert([]byte(p.Body), &buf)` (`internal/service/post/post.go:120`). One `goldmark.Markdown` instance is built once in `NewService` and shared across goroutines (goldmark's `Markdown` is documented concurrency-safe; per-call state lives in local buffers).
3. **Raw-HTML neutralization** — `neutralizeRawHTMLElements` (`post.go:81`), a single regex pass that escapes the opening `<` of raw-text/RCDATA elements (`script`, `style`, `iframe`, …) so an unterminated tag in prose cannot swallow the rest of the document.
4. **bluemonday sanitize** — `s.sanitizer.Sanitize(...)` (`post.go:124`). A `*bluemonday.Policy` built once from `UGCPolicy()` with tasklist-checkbox and link-hardening additions. The policy is concurrency-safe; this is the most expensive step because it runs a full HTML5 tokenizer over the output.
5. **Gin template execute** — `c.HTML(http.StatusOK, "post.html", gin.H{...})` (`internal/api/rest/v1/post.go:83`). The `template.HTML` cast marks the already-sanitized body as trusted so Gin does not re-escape it.

Steps 2–4 are a pure function of `Body`. On an immutable post they produce byte-identical output on every request until the process restarts — an ideal cache candidate that nothing currently exploits.

### Six verified gaps

- **No caching anywhere.** No `Cache-Control`, no `ETag`, no in-memory render cache.
- **No HTTP compression.** `docker/Caddyfile` has no `encode` directive; every response is sent uncompressed.
- **Inline CSS dominates page weight.** `backend/templates/post.html` is 8789 bytes, of which **8073 bytes are the `<style>` block**. This CSS is retransmitted in full on every single page view.
- **No Postgres connection-pool tuning.** `internal/infra/db.go` `New` opens Postgres with `gorm.Open(...)` and sets no `SetMaxOpenConns` / `SetMaxIdleConns` / `SetConnMaxLifetime`. Harmless under SQLite (which pins `MaxOpenConns(1)` at line 84), harmful under Postgres with concurrent reads.
- **Single global rate limiter.** `cmd/server/main.go` attaches one tollbooth limiter at `r.Use(...)` (line 214). The same limiter governs both the public read route `GET /:id` and all write routes — coupling read throttling to write throttling.
- **`UpdatePost` still exists in code** (`domain/post/repository.go` interface, `infra/post_repo.go` impl, service, API) and must be removed, since the business rule forbids post updates.

### Measured byte-level breakdown

Generated by compressing the real `post.html` and a synthetic body:

| Component | Uncompressed | gzip -9 |
|-----------|-------------:|--------:|
| `post.html` template total | 8789 B | — |
| Inline `<style>` block | 8073 B | **1798 B** |
| HTML skeleton (no CSS) | 247 B | ~120 B |
| Typical 1.8 KB body | 1850 B | 90 B |
| Full page (inline CSS) | 10639 B | 2183 B |
| Full page (CSS externalized) | 2099 B | **283 B** |

Externalizing + compressing + caching the CSS takes a repeat-visit page from ~10 KB (gzip'd, inline) down to a few hundred bytes of body, because the 1.8 KB CSS is fetched once and then served from the browser cache across the whole site.

## Architecture (Target)

### Three cache layers, three invalidation stories

```
Browser ──[1]──> Cloudflare edge ──[2]──> Origin VPS (Caddy → Go)
 (private)        (shared)                  (render cache + DB)
```

| Layer | TTL | Invalidated by |
|-------|-----|----------------|
| Browser | `max-age=300` | expiry only — cannot be purged by the server |
| CDN | `s-maxage=3600` | expiry + synchronous origin revalidation (no purge API needed) |
| Origin render cache | unbounded (key = QID + release dimensions) | process restart; `DeletePost` / `PruneExpired`; release bump |

**The decisive subtlety: only the post *body* is immutable — the HTML *response* is not.** A Go-rendered HTML response bundles the immutable body together with a mutable shell: the `<link>` tag pointing at the CSS file, the footer brand string, the page skeleton. The shell changes whenever the CSS or template is upgraded. This is why the three TTLs differ, and why neither `immutable` nor a one-year CDN TTL may be applied to the HTML response.

- The **browser** cannot be purged by the server, so it gets a short TTL (300 s). When the shell changes, the next revalidation after 300 s picks up the new version.
- The **CDN** can revalidate against the origin, so it holds the page for an hour; when the shell has changed, the origin returns `200` with a new ETag and a fresh body (see *ETag design* below), and the CDN swaps its copy. A one-year TTL is deliberately *not* used here — it would freeze a stale shell until manual purge. (As of April 2025 the Cloudflare free tier does support per-post cache-tag purge — see *CDN caching* below — but the one-hour TTL is still chosen so that renderer/CSS upgrades propagate within an hour without requiring a zone-wide purge, and so that the purge API remains an *active-deletion* mechanism rather than a release-deployment mechanism.)
- The **origin render cache** keys on QID *plus release dimensions* (see below); a release bump rotates the whole key namespace automatically.

### ETag design — hash the rendered response, not its inputs

`ETag` is the fingerprint of the *response body*. Because the response bundles the immutable body with a mutable shell (the `<link>` to the CSS file, the footer brand string, the page skeleton), and because the renderer itself (goldmark + bluemonday + the raw-HTML neutralizer) may be upgraded between releases, the ETag must reflect **everything that determines the rendered bytes** — and the only way to guarantee that is to hash the rendered output, not the inputs.

Hashing the inputs (`body + title + cssHash + templateVersion`) was the previous design. It is **incorrect**: a goldmark or bluemonday upgrade changes the rendered HTML but leaves `body+title+cssHash+templateVersion` unchanged, so the ETag does not change, the CDN's one-hour revalidation hits `If-None-Match` equality and returns `304`, *renewing a stale shell rendered by the old code*. The new rendering never propagates. This is the same class of bug as the body-only ETag that failed on CSS upgrade; the only fully robust fix is to hash what the client actually receives.

The two ETag variants therefore hash their respective response bodies directly:

```
ETag (HTML) = xxhash64( renderedHTML )            // goldmark + neutralize + sanitize output
ETag (raw)  = xxhash64( "# " + title + "\n\n" + body )   // the exact bytes of the raw response
```

- **`xxhash64`** (via `github.com/cespare/xxhash/v2`) is used instead of SHA-256. ETag generation does not need cryptographic collision resistance; xxhash is ~20× faster than SHA-256 and is the de-facto Go non-cryptographic hash (used by ristretto, prometheus, badger). It is already a transitive dependency through ristretto, so no new dependency is added. The 64-bit value is hex-encoded to 16 characters; collision probability (2⁻⁶⁴) is negligible for cache validation.
- **The HTML ETag hashes `renderedHTML`** — the exact bytes the client receives. Any change to the renderer, the template, or the CSS (which changes the `<link>` href in the shell) automatically produces a different ETag. No dimension checklist is needed; nothing can be forgotten.
- **The raw ETag hashes the raw response body** (`"# " + title + "\n\n" + body`), which is the exact string returned by the `?format=raw` handler. The raw variant is pure string concatenation, not a rendered product, so its ETag is cheap (no goldmark/bluemonday pass).
- The ETag is computed **once per cache miss**, inside `singleflight.Do`. On cache hit the stored ETag is returned directly; no hashing occurs. So the cost of hashing the full rendered HTML is paid only by the leader of a cold-miss burst, never by the hot path.

### Render-cache key — QID + buildID, with a variant suffix

```
cache key (HTML) = qid + ":" + buildID + ":html"
cache key (raw)  = qid + ":" + buildID + ":raw"
cache value      = { etag, body }    // stored together
```

The key is simplified from the previous `qid:cssHash:templateVersion` design. Rationale: a release ships a new binary, which restarts the process, which clears the in-memory cache. Within a process lifetime the renderer, template, and CSS are all constants (built once in `NewService`), so the QID alone uniquely determines the output. `buildID` is a compile-time-injected process constant (a short hash of the build) retained **only as defense against a future hot-reload of templates without restart** — currently impossible, but zero-cost insurance. The `:html`/`:raw` suffix separates the two variants so they do not collide.

### How a single request flows through the layers

1. **First visit, browser cache cold, CDN cache cold.** Browser → Cloudflare edge → origin. Go renders, Caddy compresses, response flows back with cache headers. Cloudflare stores it at the edge for one hour; the browser stores it for 300 s.
2. **Repeat visit within 300 s.** Browser uses its local copy. **Zero network traffic**, not even a request.
3. **Repeat visit after 300 s, CDN still fresh (within the hour).** Browser sends a conditional request (`If-None-Match`). Cloudflare's edge copy has not expired, so Cloudflare itself answers `304` from the edge. **The origin is never contacted.**
4. **CDN copy past one hour.** Cloudflare sends a conditional request to the origin. If the shell has not changed, the origin returns `304` (cheap, bodyless); if a release has shipped, the origin returns `200` with a new body, and Cloudflare swaps its copy. This revalidation is synchronous — the request waits for the origin's `304`/`200` before responding. (`stale-while-revalidate` is **not** used: `s-maxage` implies `proxy-revalidate` per RFC 9111, which disables `stale-while-revalidate` at shared caches, so the directive would be a no-op. Synchronous revalidation is cheap here — the `304` is bodyless and the origin render cache serves the ETag without re-rendering.)
5. **First visit from a different geographic region.** That region's edge node has a cold cache and performs one origin fetch; subsequent regional visitors hit that edge. Origin load scales with the number of *edges that have seen the URL*, not with total request count.
6. **Post deleted by retention prune.** Origin removes it from the render cache and the DB. The CDN edge copy lingers up to its one-hour TTL; readers in that window get a stale-but-harmless 200. This is acceptable because the content is already-expired ephemeral data being reaped by housekeeping, not active censorship. `PruneExpired` does **not** issue CDN purges — it is an external housekeeping call, and stale-but-harmless delivery of already-expired content is the accepted tradeoff.
7. **Post deleted by a user or admin (active deletion).** Origin removes it from the render cache and the DB, and issues a **best-effort cache-tag purge** to Cloudflare (see *CDN caching* below). The CDN edge copy is invalidated within ~150 ms on success; readers see `404` immediately after the purge lands. If the purge API call fails (network, rate limit), it is logged and the CDN copy falls back to natural `s-maxage=3600` expiry — at most one hour of stale 200s. The browser's local copy is governed by `max-age=300`, so within five minutes of the deletion the browser revalidates and receives the `404`.

### Who handles the 304

| Situation | Handler |
|-----------|---------|
| CDN edge hit, browser revalidates | **Cloudflare** — compares `If-None-Match` against its stored ETag and returns `304` itself; origin never sees the request |
| CDN copy past TTL, revalidates against origin | **Gin handler** — calls `RenderPostHTML`/`GetPostMarkdown`, which returns the ETag from the render cache (or computes it on cache miss inside `singleflight`); the handler compares `If-None-Match` against it; on match, `c.AbortWithStatus(304)` returns no body. On cache hit this skips goldmark/bluemonday entirely; on cache miss the render still happens (and fills the cache for subsequent requests). |
| No CDN | **Gin handler**, same path |

Caddy does **not** generate or compare ETags — it is a reverse proxy that passes headers through and compresses bodies. `304` responses have no body and are not compressed.

## Detailed Technical Explanations

### HTTP caching: the headers, in detail

The HTML response from `RenderPost` carries:

```http
ETag: "<xxhash64(renderedHTML)>"
Last-Modified: <Post.CreatedAt as HTTP date>
Cache-Control: public, max-age=300, s-maxage=3600
Cache-Tag: post-<qid>
Vary: Accept-Encoding
```

The `?format=raw` response carries:

```http
ETag: "<xxhash64(\"# \"+title+\"\n\n\"+body)>"
Last-Modified: <Post.CreatedAt as HTTP date>
Cache-Control: public, max-age=300, s-maxage=3600
Cache-Tag: post-<qid>
Vary: Accept-Encoding
```

The hashed CSS asset (served at `/static/post.<cssHash>.css`) carries:

```http
Cache-Control: public, max-age=31536000, immutable
```

- **`public`** allows shared caches (the CDN) to store the response, in addition to the browser. The opposite token `private` would forbid CDN caching — appropriate for authenticated responses, wrong for markpost's public post pages.
- **`max-age=300`** is the browser's freshness lifetime. Within 300 s the browser serves from disk with no network activity at all. Both HTML and raw use this; neither is marked `immutable` (see below).
- **`s-maxage=3600`** overrides `max-age` for shared caches only. It tells Cloudflare to consider the response fresh for one hour regardless of the browser's 300 s. This is the knob that lets the CDN absorb the overwhelming majority of reads.
- **No `stale-while-revalidate`.** It is deliberately omitted. Per RFC 9111 (§5.2.2.10), `s-maxage` incorporates the semantics of `proxy-revalidate`, which prohibits shared caches (like Cloudflare) from serving stale content. Cloudflare documents this explicitly: with `s-maxage` present, `stale-while-revalidate` is a no-op and revalidation returns `EXPIRED` (synchronous) instead of `UPDATING` (async background refresh). Keeping a directive that does nothing would be misleading. The design instead accepts synchronous revalidation: on `s-maxage` expiry the CDN waits for the origin's `304`/`200`. This is cheap because the `304` is bodyless and the origin render cache serves the ETag without re-rendering. (Making `stale-while-revalidate` actually take effect would require dropping `s-maxage` and setting the CDN TTL via an Edge Cache TTL Cache Rule — but the Free-tier minimum for Edge Cache TTL is 2 hours, which conflicts with the one-hour upgrade-propagation target. The trade-off is documented in *CDN caching* below.)
- **`immutable`** appears **only on the CSS asset**, never on the HTML or raw response. The CSS filename is content-hashed, so the URL is never reused and the bytes truly never change — `immutable` is strictly correct there. The HTML and raw responses are served at URLs that do **not** change on release (`/:qid`, `/:qid?format=raw`), and a post can be actively deleted, so marking either `immutable` would (a) be factually incorrect and (b) prevent the browser from ever learning of a deletion — the local copy would persist for the full `max-age` year with no revalidation. The raw response's body is immutable per-QID, but its URL is not content-addressed, so it gets the same TTL scheme as HTML rather than `immutable`.
- **`ETag`** is the response-body fingerprint described above (xxhash64 of the rendered HTML for the HTML variant; of the raw markdown string for the raw variant). On revalidation, equality yields `304` (bodyless); inequality yields `200` with a fresh body.
- **`Last-Modified`** is set to `Post.CreatedAt` (HTTP-date format). Because posts are write-once, `CreatedAt` is the true last-modified time of the content. It serves as the secondary validator per RFC 9110 (which recommends sending both `ETag` and `Last-Modified`); `If-None-Match` takes precedence over `If-Modified-Since`, so the strong ETag is authoritative and `Last-Modified` only fills in for clients that send `If-Modified-Since` alone. `Last-Modified` does **not** reflect shell/renderer upgrades — but the ETag does, and the ETag wins on revalidation, so a stale `Last-Modified` cannot cause a wrong `304`.
- **`Cache-Tag: post-<qid>`** is Cloudflare's surrogate key. Both HTML and raw variants of a post carry the same tag, so a single purge-by-tag call invalidates both. Cloudflare strips this header from responses delivered to visitors (it is for CDN internal use only), so it does not leak to clients. This tag is what makes per-post CDN purge feasible without enumerating variant URLs.
- **`Vary: Accept-Encoding`** ensures the CDN stores separate cached copies for `gzip` and `zstd` variants, so a browser that asked for zstd does not receive a gzip'd body from a mismatched cache entry. (Caddy's `encode` adds this header automatically when it compresses; setting it explicitly in the handler covers the no-compression fallback for self-hosted instances.)

### CDN caching: why Cloudflare, and what the free tier actually provides

Cloudflare's free tier is the load-bearing choice of this design, so its boundaries matter.

- **Unlimited bandwidth.** Unlike AWS CloudFront or other metered CDNs, Cloudflare does not bill by the byte on any plan. The projected ~7.8 TB/month of edge egress (300 reads/s × 10 KB × 2.6 Ms/month) costs $0. A metered CDN at $0.085/GB would cost ~$660/month and re-impose the 1 TB constraint by invoice.
- **Unmetered DDoS protection.** Volumetric attacks are absorbed at the edge. For a 2-core origin behind a 3 Mbps link this is not optional — an uncached flood of even modest size would exhaust the origin instantly.
- **Global anycast edge, ~330 POPs.** Readers are served from a nearby node. First-touch latency is low everywhere.
- **Page Rules and Cache Rules.** The free tier allows limited but sufficient rule counts to set cache behavior per route pattern.

**What the free tier does *not* include, and why it does not matter here.** The 100k/day limit applies to **Workers** (edge compute — serverless functions that run custom code at the edge). markpost uses none. Cloudflare's CDN cache path is a static, header-driven cache; it requires no edge code and has no request limit. The Workers quota is a non-constraint for this design.

**Why the design uses the purge API, and how.** As of April 2025, Cloudflare made **all purge methods available on every plan, including Free** — purge by URL, by cache-tag, by prefix, by hostname, and "purge everything." This overturned the prior constraint that made the original spec avoid purging. The relevant limits for the Free tier (applied per **account** via a token-bucket model) are: **5 purge requests/minute** with a **bucket capacity of 25 tokens** (short bursts are absorbed by accumulated tokens), and **100 operations per request** (the count of tags/URLs in one call). Purge latency is documented at under 150 ms globally. The limits are shared across all zones on the same plan within an account.

This is sufficient for markpost's deletion volume. The design uses **purge by cache-tag** rather than purge by URL: each post's HTML and raw responses carry `Cache-Tag: post-<qid>`, and deleting a post issues a single `POST /zones/{zone}/purge_cache` with body `{"tags":["post-<qid>"]}`. One API call invalidates both variants regardless of how many `Accept-Encoding` entries the CDN holds. Even at a hypothetical 3,000 deletions/day, the average purge rate is ~2/minute — well under the 5/minute Free-tier ceiling. The purge is best-effort and does not retry: if a request is dropped (network error) or rate-limited (empty token bucket → 429), the CDN copy falls back to natural `s-maxage=3600` expiry, which is never worse than the TTL self-healing the design already relies on.

**Purge is best-effort and asynchronous.** The delete handler removes the origin render-cache entry synchronously (mandatory), then enqueues the Cloudflare purge call on a background goroutine (non-blocking). The HTTP delete response returns immediately and does not depend on the purge succeeding. If the purge fails (network error, 429 rate limit, invalid credentials), it is logged and the CDN copy falls back to natural expiry within one hour. This means active deletion of a post is **immediate at the origin, near-immediate at the CDN (typically <150 ms), and at most 5 minutes stale at the browser** (governed by `max-age=300`). This is the accepted consistency window for an ephemeral-content service.

**What is still avoided: "Purge Everything."** Clearing the entire zone is explicitly rejected: it forces every cached post to re-origin simultaneously, producing a thundering herd that can flatten a 2-core origin. The cache-tag mechanism provides per-post granularity without that risk. `PruneExpired` (the housekeeping prune of already-expired content) intentionally does **not** purge — stale delivery of already-expired ephemeral content is harmless and the prune volume could be large; only active user/admin deletion triggers a purge.

**Lock-in risk.** Cloudflare is a reverse proxy reachable purely by DNS. The origin VPS, Caddy, Go binary, and Postgres are unchanged. Migrating away is a DNS record change; there is no proprietary API embedded in the application. The migration cost is deliberately kept near-zero.

### Compression: zstd over gzip, both over brotli

Caddy's `encode zstd gzip` directive enables two codecs; Caddy selects per request based on `Accept-Encoding`.

- **zstd (Zstandard)** matches or beats gzip's ratio at ~3× the speed. For text/HTML it typically produces 5–10% smaller output than gzip at the same CPU cost.
- **gzip** is retained as the universal fallback for the rare client that does not advertise zstd support.
- **brotli** was considered and rejected. It can outperform both on static text, but it requires Caddy to be built with a non-default plugin, its advantage is marginal at our payload sizes, and zstd already covers the "better than gzip" slot.

**Server-side precompression** (Caddy's `precompressed` directive) was also rejected. It only helps static assets, and the static asset here (the fingerprinted CSS) is already 1.8 KB gzipped — the marginal saving is sub-millisecond per request and not worth the build-pipeline complexity.

### Client IP detection behind the reverse proxy

The deployment has a single request path for all traffic:

- **All traffic** (reads and writes): `Client → Cloudflare → Caddy → Go`. Cloudflare rewrites `X-Forwarded-For` and sets `CF-Connecting-IP` on origin pull. The origin firewall is locked to Cloudflare's CIDRs (see [`cloudflare.md`](./cloudflare.md) *Origin protection*), so no legitimate traffic reaches Caddy except via Cloudflare.

The earlier design assumed a separate direct-connection write path (`Client → Caddy → Go`, bypassing the CDN) and kept the origin publicly reachable to support it. That assumption is **superseded**: in the final SaaS topology all traffic flows through Cloudflare, and the origin is not directly reachable. The XFF + Caddy `trusted_proxies` mechanism is retained not because direct connections are expected, but as **defense in depth** against the residual threat that the IP allowlist is bypassed via IP spoofing.

The design therefore uses the standard `X-Forwarded-For` chain with a two-layer trust configuration:

**Caddy layer.** Caddy's `reverse_proxy` is configured with `trusted_proxies` set to **Cloudflare's published CIDR ranges** (IPv4 and IPv6). This has a precise effect grounded in Caddy's `addForwardedHeaders` logic (`modules/caddyhttp/reverseproxy/reverseproxy.go`):
- When a request arrives **from Cloudflare** (read-path origin pull), the direct peer IP is in a Cloudflare CIDR, so `trusted=true`. Caddy takes the **existing** `X-Forwarded-For` value (which Cloudflare populated with the real client IP) and **appends** its own hop to the right end: `X-Forwarded-For: <real-client>, <cloudflare-hop>`. Caddy also sets `X-Real-IP: {remote_host}` (the Cloudflare edge IP) via an explicit `header_up` — this is for log friendliness and downstream compatibility, not for IP determination (gin prefers `X-Forwarded-For`).
- When a request arrives **from a non-Cloudflare peer** (only possible if the IP allowlist is bypassed), the peer is not in any Cloudflare CIDR, so `trusted=false`. Caddy **overwrites** any client-supplied `X-Forwarded-For` with just the real client IP via `Header.Set` (discarding any spoofed value the client may have sent), and sets `X-Real-IP` to the same real client IP.

This is the core anti-spoofing property: a client that sends a forged `X-Forwarded-For: 1.2.3.4` has that value **overwritten** by Caddy, not appended to. Only Cloudflare (a trusted peer) is allowed to prepend values to the chain. In normal operation every request arrives via Cloudflare; the overwrite path is the defense-in-depth that catches the IP-spoofing bypass case.

**gin layer.** gin's `SetTrustedProxies(["127.0.0.1", "::1"])` reflects that Caddy proxies to Go over loopback. gin's `ClientIP()` checks that the direct TCP peer (`127.0.0.1`) is trusted, then walks the `X-Forwarded-For` header **right-to-left**, skipping IPs that are themselves trusted proxies and returning the first untrusted IP. Since all legitimate traffic arrives via Cloudflare, the chain is `<real-client>, <cloudflare-hop>` (Caddy appends its loopback hop, which gin recognizes as trusted); gin skips both hops and yields the real client IP. (gin's `validateHeader` implements MDN's "trusted proxy list" algorithm — walk right-to-left, skip trusted, return first non-trusted.)

**Cloudflare CIDR maintenance.** The CIDR list is hardcoded in the Caddyfile (the `trusted_proxies` directive accepts CIDR literals only; it cannot fetch a URL). Cloudflare occasionally updates its published ranges (https://www.cloudflare.com/ips/); operators must sync the Caddyfile when they change. This is an explicit operational responsibility, documented in [`cloudflare.md`](./cloudflare.md).

**Why not `TrustedPlatform = gin.PlatformCloudflare`.** That mode makes gin trust `CF-Connecting-IP` unconditionally with no CIDR check (verified in gin source `context.go:1000-1005`). Although the origin firewall is locked to Cloudflare's CIDRs, IP spoofing remains a residual threat (the docs flag IP allowlists as "vulnerable to IP spoofing"). An attacker who spoofs a Cloudflare source IP to bypass the firewall could also forge a `CF-Connecting-IP` request header, and `PlatformCloudflare` would trust it — evading rate limiting. The XFF + Caddy-`trusted_proxies` approach is self-protecting against this: Caddy validates the peer at the TCP layer (where forgery is impossible) and overwrites any non-trusted-peer XFF, so the forged header never reaches gin. This is why the XFF design is retained as defense in depth even though all normal traffic flows through Cloudflare.

### CSS externalization, minification, and content-hash fingerprinting

The ~8 KB inline `<style>` block is the largest single byte-cost on the page. Externalizing it has three parts:

1. **Extract and minify the CSS.** A build-time step (a `go:generate`-invoked Go program in `cmd/buildcss/`) reads the inline `<style>` from `backend/templates/post.html`, minifies it with **`github.com/tdewolff/minify/v2`** (the de-facto Go minifier — 4.1k★, actively maintained, supports CSS3), and writes the result to `backend/static/post.<hash>.css`. No Node/Vite toolchain is needed; the minifier is a pure-Go import compiled into the build helper. (The current CSS is a single self-contained file with no `@import` or `url()` references, so no bundler is required. If future CSS splits into multiple files or references external assets, `esbuild`'s Go API can be added later — but it is unnecessary now.)
2. **Content-address the filename.** The file is named `post.<xxhash-of-minified-content>.css`. The `Cache-Control: public, max-age=31536000, immutable` header is correct *only because* the filename changes whenever the content does. On a CSS upgrade:
   - the new minified file has a different hash → a different filename → a different URL;
   - the build step also writes the hash into a generated Go file `backend/internal/web/csshash.go` (`var CSSHash = "<hash>"`) and embeds the CSS file into the binary via `go:embed`;
   - the template references the CSS via `<link rel="stylesheet" href="/static/post.{{.CSSHash}}.css">`, and the handler serves the embedded file from memory (no filesystem dependency at runtime);
   - the HTML's one-hour CDN TTL rotates to the new shell naturally (the `<link>` href change alters the rendered HTML, which the xxhash ETag captures automatically);
   - every browser fetches the new CSS because it is at a URL it has never seen;
   - the old CSS file is left in place (embedded in old binaries); browsers naturally age it out.
3. **Serve the CSS route.** Caddy serves `/static/*` directly (a `file_server` within a `handle /static/*` block) — or, since the CSS is `go:embed`ded, a gin route `GET /static/:filename` can serve it from memory. The latter avoids any disk dependency and works identically in all deployment contexts.

This is the same fingerprinting strategy MDN documents as "cache busting" (the spec previously called it "fingerprinting"; MDN does not use that term). It avoids any need for a purge API on static assets: the URL *is* the version.

### HTML minification

Beyond the CSS, the rendered HTML response itself is minified with `tdewolff/minify` (the HTML minifier, same library). This is applied **at render time inside the service**, not at build time, because the HTML is dynamic (the post body is rendered per-QID). The minifier is constructed once in `NewService` alongside the goldmark instance and bluemonday policy, and reused across goroutines (the `minify.Minifier` is concurrency-safe). The minified HTML is what gets stored in the render cache and hashed for the ETag, so the ETag and the served bytes are always consistent. Minification removes whitespace, comments, and redundant tags from the shell; the already-sanitized body is safe to minify.

### Postgres TOAST and lz4

**TOAST (The Oversized-Attribute Storage Technique)** is Postgres's built-in mechanism for storing values larger than ~2 KB. When a `text` column value exceeds the page-friendly size, Postgres automatically compresses it (with `pglz` by default) and stores it out-of-line in a hidden "TOAST table". This is **on by default for `text` columns**, requires no schema work, and is transparent to SQL.

For a 32 KB post body, TOAST stores ~10–12 KB after pglz compression. This is why the storage estimate below is an order of magnitude smaller than the naive 32 KB × count.

**lz4** is an alternative TOAST compressor available since Postgres 14 (the project runs Postgres 17). Switching a column:

```sql
ALTER TABLE posts ALTER COLUMN body SET COMPRESSION lz4;
```

Lz4 decompresses roughly 3× faster than pglz with comparable ratios. On a 32 KB body the decompression cost is tens of microseconds per read — negligible on its own, and further amortized by the render cache. Existing rows are not rewritten by `ALTER`; run `VACUUM FULL` once to retrofit them.

**How the switch is applied.** The `ALTER` runs automatically on Postgres startup, in `Database.migratePostBodyCompressionLZ4` (`internal/infra/db.go`). It is guarded by a check of `pg_attribute.attcompression`: that column is a `char` set by `SET COMPRESSION` — `'p'` = pglz, `'l'` = lz4, `'\0'` = default (the constants `TOAST_PGLZ_COMPRESSION` / `TOAST_LZ4_COMPRESSION` / `InvalidCompressionMethod` in `src/include/access/toast_compression.h`); it is *not* an OID into `pg_am` (that catalog is for index access methods). The `ALTER` is issued only on the first startup where `attcompression <> 'l'`, and every subsequent boot runs only the catalog read and skips the DDL. This matters because `SET COMPRESSION` is idempotent in effect but still acquires an `AccessExclusiveLock` for the catalog update on each execution — guarding it means that lock is taken exactly once across the lifetime of a database, not on every restart. `VACUUM FULL` (which retrofits existing rows) is a separate one-time operator action, intentionally not automated, because it holds `AccessExclusiveLock` for the duration of the rewrite.

**Application-level gzip-into-BYTEA** was considered as a more aggressive option (changing `body` from `text` to `bytea` and gzipping on write). It achieves ~4× compression versus ~3× for TOAST. It was **rejected as the default**: TOAST already gets most of the gain with zero application code, and gzip moves decompression into the Go hot path on every read. It remains a documented escalation step if disk pressure ever demands it.

### `singleflight`: collapsing the thundering herd

`golang.org/x/sync/singleflight` provides one operation:

```go
v, err, shared := group.Do(key, func() (any, error) { ... })
```

The contract: concurrent calls to `Do` with the same `key` execute the function exactly **once**; all other callers block until that one call returns, then receive the same result. The mechanism is a `map[string]*call` guarded by a mutex, where each `call` carries a `sync.WaitGroup`. The first goroutine to arrive for a key becomes the *leader*: it registers a `call`, releases the mutex, and runs the function. Subsequent arrivals (*followers*) find the `call` in the map, increment a duplicate counter, release the mutex, and `Wait()` on the group. When the leader finishes, it stores the result, calls `Done()`, and all followers wake and return the same value.

The crucial property: **the mutex is held only for map operations, never during the function execution.** The leader renders with no lock held; followers park at zero CPU cost (goroutine scheduling only). This is why `singleflight` collapses a thundering herd without serializing rendering.

**Where it pays off in this design.** `singleflight` defends against three distinct spikes:

1. **CDN edge cold-miss on a hot post.** A post goes viral; many edges miss simultaneously and revalidate against the origin together. `singleflight` collapses them to one render.
2. **Release-deploy window.** A new release ships a new binary, which restarts the process, which clears the in-memory render cache. Every revalidation becomes a cache miss and a full render. Without `singleflight`, concurrent revalidations of the same QID each render independently; with it, they share one render.
3. **Origin process restart.** The render cache is empty; the first burst of reads misses on everything. `singleflight` limits the damage to one render per QID.

### ristretto: scan-resistant in-process caching

`github.com/dgraph-io/ristretto` is Dgraph's battle-tested concurrent cache. It was chosen over alternatives for one specific property: **TinyLFU admission control**, which makes the cache resistant to scan pollution.

Read access to markpost is **Zipfian** — a small number of hot posts account for the majority of reads, while a long tail of cold posts is rarely touched. A plain LRU cache has a known weakness here: a burst of one-time accesses to cold posts (a batch share, a crawler sweep) evicts the hot set, and the cache's hit rate collapses right when it matters most. TinyLFU tracks frequency sketches so that a new entry is only admitted if it is "hotter" than what it would evict; one-off cold accesses are rejected, preserving the working set.

Other configuration points:

- **`MaxCost`** is set in *bytes* (e.g. 128 MiB); entries are admitted with a cost equal to their rendered HTML length. The cache self-limits to a real memory budget rather than an item count.
- **Batched, asynchronous eviction.** Writes are buffered and applied in batches, so the hot read path never blocks on eviction bookkeeping.
- **`NumCounters`** ~10× the expected key count keeps the frequency sketch accurate.

Because posts are immutable, **within a release the cache key is just the QID** (plus the release dimensions described above). Invalidation is limited to deletion events (`DeletePost`, `PruneExpired`) and release bumps; there is no `UpdatePost` path.

### `singleflight` + ristretto, composed

The two libraries cooperate. The fast path is a ristretto `Get`; only on miss does the request enter `singleflight.Do`, and inside `Do` the code re-checks the cache to avoid racing a concurrent fill:

```go
func (s *Service) RenderPostHTML(ctx context.Context, qid string) (etag, html string, err error) {
    key := qid + ":" + buildID + ":html"

    // Fast path — cache hit; no lock, no Do
    if v, ok := s.cache.Get(key); ok {
        return v.etag, v.html, nil
    }

    // Slow path — singleflight dedupes concurrent misses for the same key
    v, err, _ := s.group.Do(key, func() (any, error) {
        // Double-check inside Do: another burst may have filled the cache
        // while we were waiting to become leader.
        if v, ok := s.cache.Get(key); ok {
            return v, nil
        }
        html, err := s.render(ctx, qid)   // only the leader runs this; includes HTML minify
        if err != nil {
            return nil, err
        }
        etag := xxhashHex16(html)                    // hash the rendered output, not the inputs
        s.cache.Set(key, renderResult{etag: etag, html: html}, int64(len(html)))
        return renderResult{etag: etag, html: html}, nil
    })
    return v.(renderResult).etag, v.(renderResult).html, err
}
```

Three layers, three responsibilities:

| Layer | Defends against | Mechanism |
|-------|----------------|-----------|
| ristretto `Get` (outside `Do`) | repeats across time | map lookup, nanoseconds |
| `singleflight.Do` | concurrency within an instant | `WaitGroup` barrier |
| ristretto `Get` (inside `Do`) | race during leader execution | double-checked fill |

The `?format=raw` variant follows the same shape with a separate cache key suffix (`:raw`) and a cheaper "render" (string concatenation, no goldmark/bluemonday). Its ETag is `xxhashHex16("# "+title+"\n\n"+body)`.

### Rate limiting: three independent limiters

The single global IP limiter is replaced by three independent tollbooth limiters, each scoped to a route class and keyed on the dimension that actually identifies the actor:

| Limiter | Routes | Key | Rate | Rationale |
|---------|--------|-----|------|-----------|
| **L1 read** | `GET /:qid` (public read) | client IP | 100/s, burst 200 | Generous — the CDN absorbs the vast majority of reads; this only governs the small fraction that revalidates against the origin. IP is the only identifier available on the public read path. |
| **L2 public write** | `POST /:post_key` | `user_id` | 10/min (≈0.167/s), burst 20; **plus** a daily cap of 1000/day (≈0.01157/s, burst 1000) | `post_key` is a per-user credential validated by the `PostKey` middleware, which resolves the `user_id` into the gin context. Keying on `user_id` (not the raw `post_key`) unifies the dimension with L3 and means a user rotating their post_key cannot evade the limit. The 10/min and 1000/day caps correspond to the business hard limits in the spec. |
| **L3 authenticated write** | JWT group write routes (`POST /auth/logout`, `POST /auth/change-password`, delivery channel writes, admin writes) | `user_id` | 30/min (≈0.5/s), burst 60 | Authenticated users performing state changes; `user_id` comes from the JWT via `AuthWithBlacklist`. |

**IP resolution is done by gin, not tollbooth.** The middleware calls `c.ClientIP()` (which applies the trusted-proxy logic described in *Client IP detection behind the reverse proxy*) and passes the result to `tollbooth.LimitByKeys(lmt, []string{key})`. Tollbooth's own `SetIPLookup` is **not** used — it performs no trusted-proxy validation (verified in tollbooth v8 source `libstring/libstring.go`), so delegating IP resolution to it would reintroduce the spoofing risk that gin's `SetTrustedProxies` closes.

**Daily cap implementation (L2).** Tollbooth's token bucket has a fixed 1-second window (`internal/time/rate/rate.go`), so the daily cap is expressed as `rate.Limit(1000.0/86400)` with burst 1000. This is mathematically equivalent to "1000 per day, spendable in a burst." It is the simpler of two options (the alternative being a separate date-keyed counter with UTC midnight reset); for a single-instance deployment the per-second token approximation is sufficient and avoids a second data structure. The tradeoff is that a user who spends all 1000 tokens at midnight UTC must wait ~86 seconds per additional token — acceptable for a low-frequency authoring operation.

**Anonymous clients.** If `c.ClientIP()` returns empty (cannot be determined), the L1 and any IP-keyed limiter returns `429` immediately rather than collapsing all anonymous clients into a shared `"unknown"` bucket (which would let one anonymous attacker exhaust the limit for all others). `429` is chosen over `400` because the semantic is "you are being rate-limited" (no identity → cannot be granted a quota), not "your request was malformed."

**Health check exemption.** `GET /api/v1/health` is exempt from all limiters (registered outside the limiter-wrapped groups). Docker-compose healthchecks hit this endpoint on a loopback timer; subjecting it to L1 would cause false-positive health failures under load.

**Defaults.** All three limiters ship with the safe values above as config defaults (replacing the previous `math.MaxInt` defaults, which effectively disabled limiting). Operators may override via `[ratelimit]` config sections. `RateLimit-Limit` / `RateLimit-Reset` / `RateLimit-Remaining` response headers are **not** set in this phase (the CORS expose list retains them for future implementation); tollbooth's response-writing path is bypassed because the middleware produces a custom i18n JSON 429 body via `apierr.RespondError`.

**Dead code removed.** `lmt.SetMessageContentType(...)` (set in the current `main.go`) is removed — it has no effect because the middleware never writes tollbooth's default response body.

### Disaster recovery: WAL archival vs. live replica

The DR goal is: if the server dies or the cloud loses the data, recover with minimal loss at minimal cost. Two architectures were compared.

**Live streaming replica** (a second Postgres instance continuously fed by WAL streaming, with automatic failover):

| Property | Value |
|----------|-------|
| RPO (data loss) | ~0 (synchronous) or seconds (asynchronous) |
| RTO (downtime) | seconds to minutes (automatic failover) |
| Extra infrastructure | a second always-on VPS (~$5/month) |
| Operational cost | high — replication-lag monitoring, failover automation, split-brain handling |

**WAL archival to object storage** (continuous upload of WAL segments to B2, plus periodic full base backups; restore by replaying WAL forward from a base backup):

| Property | Value |
|----------|-------|
| RPO | seconds (the most recent un-archived WAL segment) |
| RTO | ~30 minutes (provision new VPS, pull base backup, replay WAL) |
| Extra infrastructure | none — object storage only, ~$0.20/month for 40 GB |
| Operational cost | low — `pgBackRest` is configured once and runs unattended |

For markpost the replica's advantages do not justify its cost. The write rate is ~0.12/s; the data is 7-day-retention ephemeral content; and during an outage the *read* path stays alive on the CDN edge cache while only the *write* path is down. Spending 25× the money and taking on replication-operator complexity to shave 30 minutes off an already-rare RTO is a poor trade. **WAL archival is the chosen design.**

**Tool choice.** **`pgBackRest`** is selected over **`wal-g`**. Both are mainstream, both speak the S3 API that B2 implements. `pgBackRest` is Postgres-specific (purpose-built, deeper feature set — incremental backups, parallel processing, integrity checks) and has stronger community documentation; `wal-g` is more polyglot but thinner on Postgres-specific guidance. Either works; `pgBackRest` is the spec's recommendation.

### Backblaze B2 vs. Cloudflare R2 for backup storage

| | Backblaze B2 | Cloudflare R2 |
|---|---|---|
| Storage price | $0.005/GB/month | $0.015/GB/month |
| Egress price | $0.01/GB | **free** |
| Best fit | write-heavy, read-rarely | read-heavy |

**Backup is write-heavy, read-rarely** — WAL segments flow in continuously, but a restore is rare. B2's 3× cheaper storage wins. The one-time restore egress (~$0.40 for 40 GB) is negligible because restore is rare. A secondary reason to prefer B2 is vendor isolation: keeping the backup outside the Cloudflare umbrella means a single compromised Cloudflare account cannot delete both the live path and the backups.

## Deployment Topology

### Reference topology (SaaS)

```
Cloudflare edge ──HTTPS──► Host :443 ─► markpost container (s6-overlay supervises 3 processes)
(Full strict,                            ├─ Caddy :7157      (presents Cloudflare Origin CA cert for the
 Origin CA cert                          │                     edge→origin TLS-B leg; see cloudflare.md)
 on Caddy)                               ├─ Go    :7330      (loopback only)
                                         └─ Next.js :3000    (loopback only; dashboard/admin only)
                                         volumes:
                                           - ./data:/app/data
                                           - ./config.toml:/app/config.toml:ro
                                           - ./certs:/app/certs:ro   (Origin CA cert + key)
                                           - /var/run/postgresql:/var/run/postgresql  (shared socket dir)

                                     ─► postgres container (separate)
                                         volume:
                                           - pgdata:/var/lib/postgresql/data
                                         exposes:
                                           - Unix socket at /var/run/postgresql  (bind-mounted into markpost container)
```

TLS is split into two segments: the visitor↔edge leg (TLS-A) is terminated by Cloudflare with an automatically managed certificate; the edge↔origin leg (TLS-B) is terminated by Caddy presenting a Cloudflare Origin CA certificate, under the **Full (strict)** SSL mode. The earlier "TLS-less Caddy" design (equivalent to Flexible mode) is superseded — Flexible is explicitly discouraged for applications with user login. The SaaS onboarding, Origin CA steps, port selection (443), origin protection, and client-IP detection are specified in detail in [`cloudflare.md`](./cloudflare.md).

**Caddy terminates inbound traffic and routes to three loopback services.** Only one NAT hop exists (host `:8089` → container `:7157`); everything inside the markpost container runs on `127.0.0.1`. The Go↔Postgres path uses a **Unix domain socket** via a shared `/var/run/postgresql` named volume, eliminating the TCP/NAT overhead that a cross-container TCP connection would add. The path matches the postgres image's default `unix_socket_directories`, so the entrypoint owns the directory's permissions. Postgres data lives on a named volume (`pgdata`) that bypasses the container's overlay2 writable layer, so writes hit the host filesystem directly.

### Docker vs. bare metal — quantified

The question is whether Docker's overhead compromises the ability to serve the workload. The answer is no, and the reasons are specific to this topology:

| Overhead category | Impact here | Why |
|-------------------|-------------|-----|
| CPU virtualization | **none** | Linux containers use cgroups+namespaces; there is no instruction translation. The Go binary in the container is the same binary on bare metal, scheduled by the same kernel. (This is fundamentally different from a VM's hypervisor.) |
| Storage I/O | **negligible** | `./data`, `pgdata`, and the Postgres socket dir are bind mounts / named volumes — they bypass overlay2 and hit the host filesystem directly. Only the read-only `/app` image content lives on overlay2, and it is not on the hot path. |
| Network | **one NAT hop, ~microseconds** | The host→container `:8089→:7157` hop is the only cross-namespace traversal. All three in-container services talk over loopback. Go↔Postgres uses a Unix socket, not the docker0 bridge. |

This topology was specifically chosen to land in Docker's sweet spot: it sidesteps the two well-known Docker performance traps (cross-container networking and overlay2 data writes). **Bare metal would not yield a measurable improvement.**

The real bottleneck — outbound bytes on a 3 Mbps link — is unaffected by Docker entirely. Optimizing away the container layer would free single-digit microseconds per request while leaving the binding constraint untouched. The effort is better spent on the byte-reduction work in P0.

**Operational arguments for keeping Docker.** The project ships as self-hostable software. The Dockerfile is a declarative, reproducible environment; `docker compose up` is the documented one-command install; the Ansible templates already target compose; CI builds consistently across amd64/arm64. Bare metal would abandon these for no measurable runtime gain. **Docker is not a constraint here; it is an asset.**

### Self-hosting compatibility

The design must work both for the SaaS reference instance and for arbitrary self-hosted deployments. Every component falls into one of two tiers:

| Tier | Components | Self-hosted behavior |
|------|------------|----------------------|
| **In-image, always on** | Caddy `encode`, CSS externalization + minify + `go:embed`, HTML minify, HTTP cache headers + ETag/304, Postgres pool/lz4/GUC tuning, singleflight+ristretto, three-limiter rate limiting, delete endpoints + origin cache invalidation | Self-hosted users get these automatically with zero configuration. They are pure code and ship in the image. |
| **External, optional** | Cloudflare CDN, B2/WAL backup | These are operational layers hung in front of / behind the image. Self-hosted users can use them, ignore them, or substitute equivalents (nginx reverse proxy, restic to local NAS, etc.). Nothing in application code references them. |

**CDN is recommended, not required.** For the 3 Mbps SaaS reference instance a CDN is a precondition for serving hundreds of reads/second. A self-hosted instance on a fatter pipe (gigabit LAN, 100 Mbps VPS, internal network) can run without one and simply accept higher origin CPU and bandwidth use. Nothing breaks without a CDN: all cache logic lives at the origin, and the CDN only adds an edge tier.

**Configuration is config-driven, not compile-driven.** The render cache has an `[render] enabled` flag and a tunable `cache_size_bytes`, so a 512 MB small VPS can disable or shrink it. No SaaS-specific values (Cloudflare zones, B2 keys, official domains) appear in `config.go` or `config.example.toml`; `public_url` is operator-supplied. A single image serves both contexts.

**Three deployment modes.** markpost supports three deployment topologies — **SaaS** (VPS behind Cloudflare, the reference instance), **self-hosted with a domain** (VPS/NAS with Caddy automatic Let's Encrypt + forced HTTPS, no CDN), and **homelab** (NAS on a LAN, plaintext IP:port, no CDN). The modes differ only in Caddyfile, DNS, and the optional `[cloudflare]` config section — the Go binary is identical across all three. The full topology diagrams, Caddyfile selection, Cloudflare onboarding (Full strict + Origin CA), and cache-purge contract are specified in [`cloudflare.md`](./cloudflare.md). Self-hosting operators choose between "out-of-the-box" (run the image without CDN/backup — works, slower under load) and "hardened" (Cloudflare in front for bandwidth and DDoS, B2/Restic behind for backups — matches the SaaS reference instance).

## Deployment-Window Analysis (Release-Induced Origin Load)

A release ships a new binary, which restarts the process, which clears the in-memory render cache. Every post whose CDN copy revalidates after the release therefore misses the render cache and renders fresh. The question is whether this spike can flatten a 2-core origin.

**Why it does not arrive as a spike.** CDN copies do not proactively invalidate when the origin changes — they revalidate lazily, each on its own schedule. Each edge copy's TTL clock started when that copy was last filled or verified, not at release time, so revalidations distribute across the hour following the release rather than landing simultaneously. With ~330 POPs, even the same post's per-region copies expire at staggered times.

**Worst-case arithmetic.** Suppose 1 000 000 posts are cached and all revalidate within one hour (an extreme upper bound):

```
1 000 000 revalidations ÷ 3600 s ≈ 278 req/s, sustained for one hour
278 req/s × ~3 ms/render ÷ 2000 ms (2 cores) ≈ 42% CPU
```

Real active-cache populations are far smaller, so actual load is correspondingly lower. The 42% figure is a ceiling, not an expectation.

**The thundering-herd counter-example.** The arithmetic above assumes revalidations are spread out. A genuinely simultaneous burst *can* happen: a very hot post cached at many POPs, all revalidating together on the first post-release request. Fifty concurrent revalidations of the same QID would otherwise render fifty times and saturate two cores instantly. **This is precisely what `singleflight` defeats** — all fifty collapse to one render, the result is filled into the render cache, and the other forty-nine waiters receive it.

**No CDN-side spike smoothing from `stale-while-revalidate`.** An earlier version of this spec claimed `stale-while-revalidate=600` would smooth the post-release spike by letting the CDN serve stale copies during background refresh. That claim was incorrect: because the response also carries `s-maxage=3600`, and `s-maxage` implies `proxy-revalidate` per RFC 9111, `stale-while-revalidate` is a no-op at Cloudflare's shared cache (revalidation returns `EXPIRED`, not `UPDATING`). The directive has been removed. The spike is instead contained by the two mechanisms above — lazy, naturally-staggered CDN revalidation (each edge copy revalidates on its own TTL schedule, and ~330 POPs spread the load) plus `singleflight` collapsing any concurrent same-QID revalidations at the origin into one render. Both are independent of any SWR behavior.

**Operational lever.** Deploy during a low-traffic window if the operator wants extra margin. This costs nothing and is purely a scheduling choice.

**Conclusion.** With `singleflight` + ristretto in place (and the CDN's naturally staggered TTL revalidation), a release cannot flatten the origin. Without them, the thundering-herd case genuinely could. This is why those components are P1, not optional.

## Design Decisions and Rejected Alternatives

Each decision is recorded with its rationale and what was considered and turned down.

1. **CDN is a precondition for the SaaS reference instance, not for every deployment.** A 3 Mbps origin physically cannot carry hundreds of reads/second. *Rejected:* serving directly from the origin at SaaS scale — mathematically impossible. *Accepted for self-hosting:* no CDN, with the understanding that origin load is higher.

2. **Posts are immutable; cache bodies aggressively, responses cautiously.** Removing `UpdatePost` collapses the invalidation surface to `DeletePost` and `PruneExpired`. The render-cache key is `qid:buildID` (with a `:html`/`:raw` variant suffix); a release ships a new binary, restarts the process, and clears the cache, so `buildID` is retained only as hot-reload defense. The CDN `s-maxage` is one hour, not one year. *Rejected:* body-only ETag and one-year HTML TTL — freezes stale shells on upgrade. *Rejected:* `qid:cssHash:templateVersion` cache keys — unnecessary; the process restart on release already clears the cache, so the extra dimensions are redundant within a process lifetime.

3. **ETag hashes the rendered output, not its inputs.** `ETag = xxhash64(renderedHTML)` for HTML, `xxhash64(raw markdown string)` for raw. Hashing the output captures every dimension that affects the bytes — body, title, CSS shell, template, and crucially the renderer (goldmark/bluemonday/neutralizer) itself. *Rejected:* `sha256(body+title+cssHash+templateVersion)` — hashes the inputs, so a renderer upgrade (goldmark version bump, bluemonday policy change) leaves the ETag unchanged while the rendered HTML changes, causing the CDN to return `304` and renew a stale shell rendered by old code. *Rejected:* `sha256` — cryptographic strength is unnecessary for ETag validation; xxhash is ~20× faster and arrives as a transitive dependency via ristretto. *Rejected:* body+title only — fails on CSS/template/renderer upgrade.

4. **CDN invalidation uses cache-tag purge (best-effort) plus TTL self-healing.** Active user/admin deletion issues a `POST /zones/{zone}/purge_cache {"tags":["post-<qid>"]}` call asynchronously; origin cache removal is synchronous. If the purge fails, the CDN falls back to natural `s-maxage=3600` expiry. `PruneExpired` does not purge (stale delivery of already-expired content is harmless). *Rejected:* "Purge Everything" — clears the entire zone, thundering herd. *Rejected (previously):* avoiding purge entirely — based on the pre-April-2025 premise that the Cloudflare free tier lacked usable purge; as of 2025-04 all purge methods (by URL, cache-tag, prefix, hostname) are available on Free, with a ~5 req/min rate that handles markpost's deletion volume.

5. **Three different TTLs by layer, because invalidation differs by layer.** Browser (cannot purge) gets 300 s; CDN (can revalidate synchronously) gets 3600 s; origin cache (release-scoped) is unbounded within a release. *Rejected:* one uniform TTL — either propagates upgrades too slowly (if long) or forfeits CDN benefit (if short). *Rejected:* `stale-while-revalidate` on the CDN — it is a no-op in the presence of `s-maxage` (RFC 9111: `s-maxage` implies `proxy-revalidate`), so it would mislead readers into believing background refresh is active when it is not.

6. **`immutable` only on hashed assets, never on HTML or raw.** The CSS filename is content-addressed (`post.<hash>.css`), so the URL changes on upgrade and `immutable` is strictly correct. The HTML and raw responses are served at URLs that do not change (`/:qid`, `/:qid?format=raw`), and a post can be actively deleted; marking either `immutable` would prevent the browser from ever learning of a deletion. The raw response's body is immutable per-QID, but its URL is not content-addressed, so it gets the same TTL scheme as HTML rather than `immutable`. *Rejected:* `immutable` on HTML or raw — suppresses legitimate revalidation, is factually incorrect, and blocks deletion visibility.

7. **Transmission-minimization dominates over render-caching for ROI.** Externalizing + compressing the CSS removes ~75% of every page's bytes; enabling `encode` removes ~80% of the rest. Render caching only matters for the small fraction of requests that survive the CDN. *Rejected:* treating this as a CPU-optimization problem first.

8. **`singleflight` before `ristretto`.** Of the two, `singleflight` is the more important: it defeats the thundering herd on cold-miss, on release, and on restart. `ristretto` then absorbs repeats. *Rejected:* `ristretto` alone — does not prevent the stampede, only the repeats.

9. **`ristretto` over a plain LRU / `golang-lru`.** Read access is Zipfian. A plain LRU is vulnerable to scan pollution; TinyLFU resists it. *Rejected:* `bigcache` / `freecache` — designed for many small uniform entries, a poor fit for few large variable-size HTML blobs.

10. **No `UpdatePost`.** The business scenario does not allow updates. *Rejected:* keeping it with content-hash keys and purge-on-edit — complexity with no user benefit.

11. **WAL archival over a live replica.** ~0.12 writes/second and 7-day-retention data do not justify a second VPS and replication-operator complexity. *Rejected:* live replica.

12. **B2 over R2 for backup storage.** Backup is write-heavy; B2 is 3× cheaper for that pattern and provides vendor isolation. *Rejected:* R2 — more expensive for the access pattern.

13. **Cloudflare, not a metered CDN.** Only Cloudflare's free tier carries unlimited bandwidth and unmetered DDoS protection at $0. *Rejected:* AWS CloudFront / Bunny CDN — would re-impose the 1 TB quota as a cost ceiling.

14. **zstd over brotli.** zstd matches brotli's ratio without a custom Caddy build. *Rejected:* brotli — marginal gain, build complexity.

15. **CSS content-hash fingerprinting over purge-based invalidation.** The URL is the version, so no purge API is needed for static assets. *Rejected:* a stable CSS URL with purge-on-deploy — works but couples the asset cache to an external API call.

16. **Postgres in a sibling container, connected by Unix socket.** Eliminates cross-container TCP/NAT overhead while preserving deployment uniformity. *Rejected:* Postgres bare-metal on the host — loses container consistency for no measurable gain. *Rejected:* Postgres in the markpost container — couples failures and complicates independent scaling/backup.

17. **Docker over bare metal.** Container overhead is negligible in this topology (no instruction translation, data on bind mounts, one NAT hop). Bare metal abandons the Dockerfile's reproducibility, the one-command self-hosted install, and the existing Ansible/CI pipeline for no measurable runtime gain. *Rejected:* bare-metal deployment — no measurable runtime benefit; loses reproducibility and self-hosted story.

18. **Client IP via X-Forwarded-For + Caddy `trusted_proxies` (Cloudflare CIDRs), not via `TrustedPlatform`.** In the SaaS topology all traffic flows through Cloudflare and the origin firewall is locked to Cloudflare's CIDRs, so no legitimate traffic reaches Caddy except via Cloudflare. The XFF + Caddy `trusted_proxies` design is retained as **defense in depth**: it self-validates the peer at the TCP layer (where forgery is impossible), so even if the IP allowlist is bypassed via IP spoofing, a forged `CF-Connecting-IP` or `X-Forwarded-For` cannot reach gin — Caddy overwrites non-trusted-peer XFF rather than appending to it. This rules out `gin.PlatformCloudflare` (which trusts `CF-Connecting-IP` unconditionally with no CIDR check). gin's `SetTrustedProxies(["127.0.0.1","::1"])` then walks the XFF chain right-to-left (MDN's "trusted proxy list" algorithm) to recover the real client IP. *Rejected:* `TrustedPlatform = gin.PlatformCloudflare` — trusts `CF-Connecting-IP` unconditionally; vulnerable to a spoofed source IP bypassing the firewall plus a forged header. *Rejected:* gin default `trustedProxies = ["0.0.0.0/0","::/0"]` — trusts all peers, XFF spoofable. *Rejected:* tollbooth's `SetIPLookup` — performs no trusted-proxy validation.

19. **Three independent rate limiters (read/public-write/authed-write), keyed on the actor dimension.** L1 read is per-IP (the only identifier on the public read path); L2 public write is per-`user_id` (resolved by `PostKey` middleware, unifying the dimension with L3 and defeating post_key rotation); L3 authed write is per-`user_id` (from JWT). The daily cap on L2 is expressed as a per-second token-bucket rate (1000/86400 ≈ 0.01157/s, burst 1000) for simplicity. *Rejected:* single global limiter — couples read and write throttling. *Rejected:* per-`post_key` L2 key — user_id is more stable and unified. *Rejected:* separate date-keyed counter for the daily cap — redundant given tollbooth's token bucket can express per-second rates.

20. **Active deletion (user + admin) with best-effort cache-tag purge; `PruneExpired` does not purge.** New endpoints `DELETE /api/v1/posts/:qid` (JWT owner) and `DELETE /api/v1/admin/posts/:qid` (admin). Deletion removes the origin render cache synchronously and enqueues a Cloudflare cache-tag purge asynchronously. *Rejected:* no deletion endpoint — the spec now requires user self-deletion and admin moderation. *Rejected:* purging on `PruneExpired` — stale delivery of already-expired content is harmless and prune volume could be large. *Rejected:* synchronous purge — would couple delete latency to an external API.

21. **CSS via `tdewolff/minify` + `go:embed`; HTML minified at render time.** The CSS is extracted, minified, hashed, and embedded into the binary at build time (a `go:generate`-invoked Go program using `tdewolff/minify`). The HTML response is minified at render time with the same library (the `minify.Minifier` is concurrency-safe, built once in `NewService`). *Rejected:* Vite/Node toolchain — unnecessary for a single self-contained CSS file with no `@import`/`url()`; adds a Node dependency to the build. *Rejected:* esbuild Go API — not needed until CSS bundling is required (no `@import` today). *Rejected:* no HTML minification — leaves shell whitespace/tags in the response unnecessarily.

22. **Postgres tuning: `shared_buffers=256MB`, `effective_cache_size=1GB`, `maintenance_work_mem=128MB`, `max_connections=50`, `synchronous_commit=off`, body column lz4.** These are applied via `postgresql.conf` (Ansible template) and a one-time `ALTER TABLE ... SET COMPRESSION lz4` + `VACUUM FULL`. The GUCs are not Go code and cannot be unit-tested; lz4/Unix-socket depend on a real Postgres and are verified manually/integration. *Rejected:* `shared_buffers=25%` of 2 GB (512 MB) — the box is not a dedicated DB server (shares RAM with Caddy+Go+Next.js); 256 MB leaves adequate OS cache. *Rejected:* `synchronous_commit=on` — write rate is ~0.12/s of 7-day-retention ephemeral content; the crash-window data loss is acceptable and the docs confirm `off` cannot cause inconsistency.

23. **No server-side precompression (Caddy `precompressed` directive).** It only benefits static assets, and the only static asset (the fingerprinted CSS) is already ~1.8 KB gzipped — the marginal saving is sub-millisecond per request and not worth the build-pipeline complexity of generating `.gz`/`.zst` sidecar files. *Rejected:* `precompressed` directive — complexity disproportionate to gain. (Dynamic HTML/raw responses cannot be precompressed regardless.)

24. **TOAST lz4 over application-level gzip-into-BYTEA.** lz4 (via `ALTER TABLE ... SET COMPRESSION lz4`) gives faster decompression than the default pglz with comparable ratios, transparently, with no application code. Application-level gzip-into-BYTEA (~4× compression vs ~3× for TOAST) was considered but moves decompression into the Go hot path on every read and changes the column type from `text` to `bytea`. *Rejected as default:* gzip-into-BYTEA — TOAST already captures most of the gain at zero code cost. *Retained as escalation:* see "Deferred / future expansion" below.

25. **`pgBackRest` over `wal-g` for WAL archival.** Both are mainstream and both speak the S3 API that B2 implements. `pgBackRest` is Postgres-specific (incremental backups, parallel processing, integrity checks) with stronger community documentation; `wal-g` is more polyglot but thinner on Postgres-specific guidance. Either works; `pgBackRest` is the recommendation. *Rejected:* `wal-g` — valid but thinner on Postgres-specific docs. This choice is only relevant when the DR tier upgrades from `pg_dump` (see Decision 28).

26. **No static-file materialization (writing rendered HTML to disk for Caddy `file_server`).** At 10 000 posts the in-process ristretto cache already drives origin render cost near zero; static files would add write-time render, cross-process file coordination, and a full re-render on template change, for negligible gain. *Rejected:* on-disk HTML materialization — complexity disproportionate to gain; the in-process cache is sufficient and simpler.

27. **No `ALTER TABLE ... ADD COLUMN html` DB materialization.** Postgres indexed QID lookup is sub-millisecond; DB reads are not the bottleneck. Materializing rendered HTML into the DB would push render cost onto the write path, complicate migrations (the column must be re-rendered on every renderer/template upgrade), and double the storage. *Rejected:* `html` column materialization — DB reads are not the bottleneck; the render cache solves the render-cost problem without schema coupling.

28. **No Redis, no live replica, no second VPS (single-instance resilience).** The workload (~0.12 writes/s, 7-day-retention ephemeral content) and the cost envelope do not justify a second always-on VPS, replication-lag monitoring, failover automation, or a shared cache. Single-instance + WAL-to-B2 is the chosen resilience model. *Rejected:* live streaming replica — RPO/RTO gains do not justify 25× the cost and operational complexity (see Decision 11). *Rejected:* Redis for rate-limiting/render-cache — single-instance in-memory suffices; Redis adds a dependency and a failure mode.

29. **Cache 404 responses at the CDN edge (short TTL ~60 s).** QID-enumeration attacks (probing random QIDs) are absorbed at the edge rather than origin. Without this, every probe re-originates. *Rejected:* no 404 caching — exposes the origin to enumeration attacks. *Rejected:* long 404 TTL — a deleted post's 404 would persist too long after the post is gone (a post could be re-created with a colliding QID, though QID collision is astronomically unlikely).

30. **No HTTP/2 push, no per-shard hash directories, no XFS tuning.** Measured byte savings are negligible versus `zstd` + CSS externalization + HTML/CSS minification; complexity is not warranted at this scale. *Rejected:* HTTP/2 push — deprecated by browsers, negligible gain. *Rejected:* per-shard hash directories — relevant only for filesystems with millions of files; markpost has thousands. *Rejected:* XFS over ext4 — no measurable difference at this I/O profile.

### Deferred / future expansion (not adopted now, not rejected)

These options are explicitly **not rejected** — they are documented as escalation steps for future growth or changed requirements, and the design does not preclude them:

- **Application-level gzip-into-BYTEA** for post bodies. If disk pressure exceeds the lz4-TOAST budget, changing `body` from `text` to `bytea` and gzipping on write achieves ~4× compression (vs ~3× for TOAST). The tradeoff is decompression cost on every read (mitigated by the render cache) and a schema migration. Documented in the Storage Estimation escalation ladder.
- **Cold-tier offload to B2** for posts older than a few days. ~$0.20/month for 40 GB; would require a fetch-on-miss path from the origin to B2. Documented in the Storage Estimation escalation ladder.
- **`pgBackRest` continuous WAL archival** (upgrade from hourly `pg_dump`). When the hourly RPO is no longer acceptable, switch to near-real-time WAL streaming to B2 plus a weekly full base backup. RPO drops to seconds with PITR. This is P4 in the Implementation Plan.
- **esbuild Go API for CSS bundling.** If the CSS grows to multiple files with `@import` or `url()` references to fonts/images, `tdewolff/minify` alone is insufficient (it does not do `@import` resolution or `url()` rewriting). `esbuild`'s Go API (`github.com/evanw/esbuild/pkg/api`) can be added then — it is a Go import, no Node required.
- **Redis-backed shared rate-limiting / render cache.** Only relevant if the deployment scales to multiple origin instances behind a load balancer. The single-instance design does not preclude adding Redis later; the limiter and cache interfaces are abstracted behind `post.Service`.
- **Cloudflare Workers for edge personalization.** The 100k/day free-tier Workers quota is currently a non-constraint because no edge code is used. If future features require edge compute (A/B testing, edge redirects, geo-personalization), Workers can be added without touching the origin.
- **Browser `max-age` adjustment.** The spec recommends 300 s; operators may tune longer (e.g. 3600 s) if slower brand-upgrade rollout is tolerable. This is config-driven, not a code change.
- **Purge by URL instead of cache-tag.** If Cloudflare changes cache-tag availability or rate limits unfavorably, the purge implementation can switch to by-URL (up to 100 URLs per request; the by-URL rate limit on Free is 800 URLs/second, far looser than the tag/prefix 5 req/min). Both HTML and raw variants would be enumerated per post. The `Cache-Tag` header would become optional. The design does not depend on cache-tag exclusively.
- **`stale-if-error` on CDN responses.** Adding `stale-if-error` would let the CDN serve a stale copy when the origin returns 5xx, improving availability during origin outages. Caveat: Cloudflare ignores `stale-if-error` when an applicable `s-maxage` (or `proxy-revalidate`) directive is present — which this design uses — so to actually take effect it would require the same Edge Cache TTL Cache Rule workaround as `stale-while-revalidate` (dropping `s-maxage` and accepting a ≥2-hour Free-tier minimum Edge TTL). Not added now; the header stays minimal and the workaround is non-trivial.

## Storage Estimation

"Up to 1000 posts/day/user" is a hard cap, not the expected mean. Real writing volume for a notification/temporary-share tool follows a distribution where most users write single-digit posts per day. Using a conservative mean of μ = 10 posts/user/day:

```
10 000 users × 10 posts/day × 32 KB × 7 days = 22.4 GB (raw)
× 1.3 (Postgres row/index overhead) = 29 GB
× TOAST compression (32 KB → ~12 KB) ≈ 11 GB on disk
```

**11 GB fits comfortably in 40 GB.** Even doubling the mean to μ = 20 yields ~22 GB, still within budget. Storage is not the binding constraint.

If actual growth exceeds this, the documented escalation ladder is: (a) `ALTER TABLE ... SET COMPRESSION lz4` — free, faster decompression; (b) application-level gzip-into-BYTEA — ~4× compression, more code; (c) cold-tier offload to B2 for posts older than a few days — $0.20/month for 40 GB. In practice the estimation suggests (b) and (c) will not be needed.

## Implementation Plan

Organized by priority. P0 is the highest-ROI, lowest-risk work; later phases depend on it. Each item lists the verification approach (unit / benchmark / fuzzy / manual) inline.

> **Implementation status (P0–P2).** Items 1–6 and 10–12 are delivered in application code and verified by unit tests, fuzz tests, and `-race`. Item 8 (lz4 TOAST) is delivered as an automated, guarded startup migration. Items 7 (Postgres GUC tuning) and 9 (Unix-socket connect) are delivered in the **production** Ansible templates only: the production `docker-compose.yml.j2` runs a sibling `postgres:17-alpine` container with `pgdata` on a named volume and a shared `postgres-socket` volume, mounts a `postgresql.conf` (GUCs: `shared_buffers=256MB`, `effective_cache_size=1GB`, `maintenance_work_mem=128MB`, `max_connections=50`, `synchronous_commit=off`) via `postgres -c config_file=...`, and the markpost container connects over the Unix socket (`host=/var/run/postgres`). The **staging** templates are intentionally left on their existing SQLite configuration (already in use) and are not touched. `devops/dev.py` (dev `docker-compose.yml`) keeps TCP for the dev environment but applies the same GUCs via `command: postgres -c ...` and the same three-limiter / render-cache configuration via `MARKPOST_*` env vars, so dev aligns with production's tuning. Two deliberate deviations from the literal spec text, both confirmed by the operator, are recorded in the relevant items: the CSS source lives in `backend/templates/post.css` (extracted from `post.html`, not read live from the template) so the served template contains only the `<link>`; and the SQLite pool keeps its existing `SetMaxOpenConns(1)`-during-migrate + reset behavior rather than being pinned to 1 permanently. The Caddy `trusted_proxies` is operator-supplied: the `Caddyfile.j2` defaults to `private_ranges` and the `cloudflare_cidrs` Ansible var must be set to Cloudflare's published CIDRs (https://www.cloudflare.com/ips/) — this list is not hardcoded because it cannot be fetched without network access and must be synced by the operator. The SaaS TLS strategy is decided but not yet implemented in the devops templates: Cloudflare SSL mode is set to **Full (strict)**, with Caddy presenting a manually installed Cloudflare Origin CA certificate (replacing the earlier TLS-less / Flexible-mode design). The Origin CA provisioning steps, port selection (443 for the HTTPS origin pull), and origin protection (CIDR allowlist + optional Authenticated Origin Pulls) are specified in [`cloudflare.md`](./cloudflare.md); the `Caddyfile.j2` and `docker-compose.yml.j2` updates to mount the certificate and expose 443 are tracked follow-ups.

### P0 — Transmission layer

1. **Enable compression in Caddy.** Add `encode zstd gzip` to `docker/Caddyfile`. Immediate 3–5× bandwidth reduction across every response. *Verification:* manual — `curl -H "Accept-Encoding: zstd" -I` confirms `Content-Encoding: zstd`; Caddy v2 (current) supports both codecs natively.

2. **Externalize, minify, and fingerprint the CSS.** A `go:generate`-invoked build helper `cmd/buildcss/main.go` reads the CSS source from `backend/templates/post.css` (the inline `<style>` extracted from `post.html`), minifies it with `github.com/tdewolff/minify/v2`, computes `xxhash64` of the minified output, writes `backend/internal/web/post.<hash>.css`, and generates `backend/internal/web/csshash.go` (`var CSSHash = "<hash>"`) with a `go:embed` of the CSS file. `post.html` references `<link rel="stylesheet" href="/static/post.{{.CSSHash}}.css">` and contains no inline `<style>`. A gin route `GET /static/:filename` (`v1.StaticCSS`) serves the embedded bytes from memory with `Cache-Control: public, max-age=31536000, immutable`. Caddy proxies `/static/*` to Go. *Verification:* unit test — `CSSHash` equals xxhash64 of the embedded minified bytes and is 16 hex chars.

3. **Minify rendered HTML at render time.** In `internal/service/post/post.go` `NewService`, construct a `*minify.Minifier` (HTML) once and store it on `Service`; apply it to the rendered HTML before returning from `render`. The minified HTML is what gets cached and ETag-hashed. *Verification:* unit test — minified output is smaller than unminified and re-minifying is idempotent.

4. **Add HTTP cache headers and ETag/304 to `RenderPost`.** In `internal/api/rest/v1/post.go`, change `PostService` interface signatures to return the ETag: `RenderPostHTML(ctx, qid) (title, html, etag, error)` and `GetPostMarkdown(ctx, qid) (title, body, etag, error)`. The handler:
   - Calls the service (which returns the precomputed ETag alongside the body).
   - Compares `If-None-Match` against the ETag; on match, sets `ETag`/`Cache-Control`/`Cache-Tag`/`Vary`/`Last-Modified` headers and `c.AbortWithStatus(304)`.
   - On mismatch, sets the same headers plus `Last-Modified: <Post.CreatedAt>` and serves the body.
   - HTML variant: `Cache-Control: public, max-age=300, s-maxage=3600`, `ETag: <xxhash64(renderedHTML)>`, `Cache-Tag: post-<qid>`, `Vary: Accept-Encoding`, `Last-Modified`.
   - Raw variant (`?format=raw`): same `Cache-Control`/`Cache-Tag`/`Vary`/`Last-Modified`, but `ETag: <xxhash64("# "+title+"\n\n"+body)>`.
   *Verification:* unit test — If-None-Match equality yields 304 with correct headers; inequality yields 200 with body; `*` wildcard matches; multiple comma-separated ETags match any. *Fuzzy:* fuzz `If-None-Match` header values (malformed, `W/` prefix, empty, multiple) to confirm no panic and RFC-conformant behavior.

5. **Remove `UpdatePost` end-to-end.** Delete `UpdateByID` from `domain/post/repository.go`, `infra/post_repo.go`, `internal/service/post/post.go`, and the `PostService` interface. The remaining mutators are `CreatePost`, `DeletePost`, `PruneExpired`. *Verification:* `go build ./...` and existing tests pass.

### P0+ — Origin hardening

6. **Tune the Postgres connection pool.** In `internal/infra/db.go` `New`, within the `postgresql` branch only (leave the SQLite `MaxOpenConns(1)` path untouched):
   ```go
   sqlDB, _ := db.DB()
   sqlDB.SetMaxOpenConns(25)
   sqlDB.SetMaxIdleConns(10)
   sqlDB.SetConnMaxLifetime(30 * time.Minute)
   ```
   *Verification:* unit test (SQLite path) confirms the SQLite `MaxOpenConns(1)` invariant is unchanged; the Postgres path is verified by `go build` and integration.

7. **Apply Postgres GUC tuning.** The 5 tuned GUCs (`shared_buffers = 256MB`, `effective_cache_size = 1GB`, `maintenance_work_mem = 128MB`, `max_connections = 50`, `synchronous_commit = off`) are applied as `-c` flags on the postgres service command in `devops/ansible/templates/production/docker-compose.yml.j2`, layering only the overrides on top of the image's default `postgresql.conf` (initdb-generated) rather than replacing it. These require a Postgres restart (`shared_buffers`, `max_connections` are `postmaster`-context; the rest are reloadable). *Verification:* manual — after restart, `SHOW shared_buffers;` etc. confirm the values; `pg_settings.source` reports `command line` for the overrides and `configuration file` for everything else. These are not Go code and cannot be unit-tested.

8. **Switch TOAST compression to lz4 (automated on startup).** The `ALTER TABLE posts ALTER COLUMN body SET COMPRESSION lz4` runs automatically on every Postgres startup via `Database.migratePostBodyCompressionLZ4` in `internal/infra/db.go` (the SQLite path is skipped — SQLite has no TOAST). The migration is **guarded**: it first checks `pg_attribute.attcompression` and only issues the `ALTER` on the first startup after the upgrade; once the column is lz4, every subsequent boot runs only a cheap catalog read and skips the DDL, so the `AccessExclusiveLock` that `SET COMPRESSION` takes is acquired exactly once, not on every restart. `SET COMPRESSION` is metadata-only (idempotent in effect; it does not rewrite rows), so even an unguarded re-run is safe — the guard exists only to avoid needlessly reacquiring the lock. Existing rows keep their old compression until rewritten, so a one-time `VACUUM FULL posts` is recommended in a maintenance window to retrofit them — that is intentionally **not** done automatically because it takes a long-lived `AccessExclusiveLock`. *Verification:* integration — `\d+ posts` shows `compression: lz4` after first startup; a second startup runs the catalog check and issues no `ALTER`. Cannot be unit-tested (SQLite has no TOAST); the guard query and the skip-on-idempotent path are verified against a real Postgres.

9. **Connect Go to Postgres via Unix socket.** Update `devops/ansible/templates/{production,staging}/docker-compose.yml.j2` to bind-mount the Postgres socket directory (`/var/run/postgresql`) into the markpost container. Update the DSN in the config template to `host=/var/run/postgresql user=... dbname=... sslmode=disable` (drop the TCP `port`). *Verification:* manual — after redeploy, `SELECT 1` from the Go process confirms socket connectivity. Integration-test only.

### P1 — Read path and cache

10. **Add `singleflight` + `ristretto` render cache.** New dependencies: `github.com/dgraph-io/ristretto` (brings `cespare/xxhash/v2` transitively); `golang.org/x/sync/singleflight`. The cache lives on `post.Service`. Keys: `qid:buildID:html` and `qid:buildID:raw` (where `buildID` is a compile-time-injected `var buildID = "..."` in `internal/web/buildid.go`). Value: `{etag, body}`. `RenderPostHTML`/`GetPostMarkdown` check the cache; on miss they wrap the DB-read + render pipeline in `singleflight.Do(key, fn)`, with a double-checked cache `Get` inside `Do`. Config:
    ```toml
    [render]
    enabled = true
    cache_size_bytes = 134217728   # 128 MiB
    ```
    *Verification:* unit test — cache hit returns stored value; cache miss renders once and fills; `qid:buildID:html` and `qid:buildID:raw` do not collide; different `buildID` keys do not collide. *Benchmark:* `BenchmarkRenderCacheHit` (nanoseconds, cache hit path) and `BenchmarkSingleflightCollapse` (N goroutines same qid → one render). *Race:* `go test -race` with N concurrent goroutines confirms single render. *Fuzzy:* fuzz QID strings (empty, unicode, special chars, very long) to confirm key construction does not panic or collide.

11. **Wire deletion to cache invalidation + CDN purge.** Added `DELETE /api/v1/posts/:id` (`v1.DeleteOwnPost`, JWT, owner-checked via `DeleteByQID`'s `ownerID` guard) and `DELETE /api/v1/admin/posts/:id` (`v1.DeleteAnyPost`, JWT + `RequireAdmin()`, `ownerID=0`). The service-layer `DeletePostByQID(ctx, qid, ownerID)`:
    - Removes the DB row via `DeleteByQID` (owner-scoped when `ownerID > 0`; unconstrained when `ownerID == 0` for the admin path). Returns `ErrNotFound` when no row matched.
    - Removes both cache entries synchronously (`invalidateCache` → `cache.Delete` for the `html` and `raw` keys). The ristretto wrapper's `Delete` calls `cache.Wait()` so a pending buffered `Set` from a concurrent render cannot re-admit the entry after the deletion — the invalidation is durable before the call returns.
    - Enqueues a best-effort Cloudflare cache-tag purge (`{"tags":["post-<qid>"]}`) on a background goroutine via a `Purger` interface (`cloudflarePurger` when `[cloudflare] api_token`+`zone_id` are set, `noopPurger` otherwise). The QID is sanitized (`sanitizeCacheTag`) before going into the JSON body. Failures are logged and swallowed.
    `PruneExpired` removes DB rows and origin cache entries (it returns the pruned QIDs from the repo so the service can invalidate them) but does **not** issue CDN purges. *Verification:* unit test — owner-scoped and admin deletes invalidate both cache variants and a recording purger is invoked exactly once; wrong-owner delete deletes nothing and does not purge; prune invalidates the cache without purging. *Fuzzy:* `FuzzSanitizeCacheTag` confirms the purge tag resists quote/backslash/newline injection.

### P2 — Defense (rate limiting)

12. **Replace the global limiter with three independent limiters.** In `cmd/server/main.go` `SetupRoutes`, remove the `r.Use(middleware.RateLimitByIP(lmt))` global wiring. Construct three tollbooth limiters from a new `[ratelimit]` config section (with the safe defaults: L1 100/s burst 200; L2 0.167/s burst 20 plus 0.01157/s burst 1000; L3 0.5/s burst 60). Register:
    - L1 (IP) on `GET /:qid` only.
    - L2 (user_id) on `POST /:post_key` (after `PostKey` middleware, so `user_id` is in context).
    - L3 (user_id) on the JWT group's write routes (after `AuthWithBlacklist`).
    - `GET /api/v1/health` exempt (registered outside all limiter groups).
    IP resolution via `c.ClientIP()`; anonymous (empty IP) returns 429. Remove the `SetMessageContentType` dead code. *Verification:* unit test — under-limit requests pass; over-limit returns 429; L1/L2/L3 keys are isolated (L1 doesn't count against L2); anonymous IP yields 429; `/health` is exempt. *Fuzzy:* fuzz limiter keys (user_id, ip) to confirm stable key construction.

### P3 — Operations (Cloudflare)

13. **Cloudflare free tier + cache-tag purge.** Move authoritative DNS to Cloudflare. Configure a Cloudflare API token (Zone.Cache Purge) and zone ID in the SaaS config (`[cloudflare]` section, optional — absent for self-hosted). The delete handler reads these to issue purge calls. For self-hosted instances without Cloudflare, the purge step is skipped (no-op) and the CDN falls back to TTL. *Verification:* manual — purge a test URL and confirm the `age` header resets.

14. **Cache 404s at the edge.** Configure Cloudflare to cache `404` responses with a short TTL (~60 s) so QID-enumeration attacks are absorbed at the edge. *Verification:* manual.

### P4 — Disaster recovery (unchanged)

15. **Start with `pg_dump` to B2 hourly.** Cron runs `pg_dump` and uploads to B2. RPO ≤ 1 hour, RTO ~10 minutes.

16. **Upgrade to `pgBackRest` continuous WAL archival** when the hourly RPO is no longer acceptable.

### P5 — Load testing

17. **Micro-benchmark single-render cost.** `internal/service/post/post_bench_test.go` with `BenchmarkRenderPostHTML` over a 32 KB fixture. Confirms whether goldmark, bluemonday, or the HTML minifier dominates.

18. **End-to-end HTTP load test with `vegeta`.** Three scenarios against `GET /:qid`: cold (empty cache), hot (fixed QID), all-cold (unique QID per request).

19. **Soak test with `pprof` heap diff.** Run `vegeta` for 1–2 hours; verify ristretto reaches steady state and Postgres connections do not leak.

20. **Release-window test.** With the cache warm, bump `buildID` (simulating a release) and measure the post-release render-load spike.

### Testing strategy summary

| Category | Items | Approach |
|----------|-------|----------|
| **Unit** | ETag/304 logic, render cache hit/miss/collision, singleflight collapse, rate limiter L1/L2/L3 isolation + anonymous 429 + health exemption, CSS hash determinism, HTML minify idempotence, delete→cache invalidation + purge invocation | `go test`; mock the Cloudflare client and the DB repo |
| **Benchmark** | `BenchmarkRenderCacheHit` (ns, hit path), `BenchmarkRenderPostHTML` (µs, full render), `BenchmarkSingleflightCollapse` (N goroutines → 1 render), `BenchmarkETag` (xxhash vs sha256 comparison), `BenchmarkCSSMinify` | `go test -bench -benchmem` |
| **Race** | concurrent same-QID render (singleflight), concurrent cache read/write (ristretto) | `go test -race` |
| **Fuzzy** | render cache key construction (QID fuzz), `If-None-Match` parsing (ETag fuzz: `*`, `W/`, multiple, malformed, empty), limiter key construction (user_id/ip fuzz), purge tag (QID fuzz for header injection) | `go test -fuzz=Fuzz...` (Go 1.18+ native fuzzing) |
| **Manual / Integration** | lz4 compression (`\d+ posts`), Unix socket connectivity, Postgres GUC values (`SHOW ...`), Cloudflare purge effectiveness (`age` header reset), Caddy `encode` (`Content-Encoding` header) | operator-run; documented in deployment guide |

### Tooling rationale

| Tool | Role | Why this one | Rejected |
|------|------|--------------|----------|
| `go test -bench` / `-fuzz` / `-race` | micro-bench / fuzz / race | Go built-in, zero dependency | — |
| `net/http/pprof` | profiling | Go built-in, official | — |
| `vegeta` (`tsenart/vegeta`) | HTTP load test | de-facto Go-community standard, single binary, clear reports | `wrk` (config-heavy), `ab` (limited), `JMeter` (heavyweight) |
| `pgBackRest` | Postgres backup | Postgres-community standard, purpose-built | `wal-g` (valid but thinner on PG-specific docs) |
| `ristretto` (`dgraph-io/ristretto`) | in-process cache | TinyLFU, scan-resistance, battle-tested | `golang-lru` (scan-vulnerable), `bigcache`/`freecache` (wrong shape) |
| `golang.org/x/sync/singleflight` | thundering-herd collapse | stdlib-adjacent, tiny | — |
| `tdewolff/minify/v2` | HTML + CSS minification | Go standard minifier, 4.1k★, actively maintained | Vite/Node (unnecessary dep), esbuild (no `@import` today) |
| `cespare/xxhash/v2` | ETag hash | ~20× faster than sha256, transitive via ristretto | sha256 (overkill for non-cryptographic ETag) |

## Cost

| Item | Cost |
|------|------|
| Cloudflare free tier | $0/month (unlimited bandwidth, free DDoS) |
| Backblaze B2 backup (~40 GB) | $0.20/month |
| `ristretto`, `singleflight`, `pgBackRest`, `vegeta`, `pprof` | $0 (open source) |
| **Total marginal** | **~$0.20/month** |

## Recovery Matrix

| Failure | Impact | Recovery |
|---------|--------|----------|
| VPS crash (restartable) | service down | restart; RPO = 0 |
| VPS destroyed / host data loss | all data gone | new VPS, restore from B2; RPO ~seconds (WAL) or ≤1 h (dump); RTO ~30 min |
| Postgres data-file corruption | partial data | PITR to the instant before corruption |
| Accidental `DELETE` | rows lost | PITR to before the deletion |
| Cloudflare outage | new posts cannot be created | existing posts still readable from CDN edge; write path returns when Cloudflare recovers |
| B2 outage (rare) | new backups stall | existing backups intact; B2 is itself multi-replica |

## What This Spec Explicitly Does Not Do

A consolidated list of explicit rejections. Each entry cross-references the Design Decision where the full rationale and rejected alternatives are recorded:

- **No static-file materialization** (writing rendered HTML to disk for Caddy `file_server`) — Decision 26.
- **No `UpdatePost` path** — Decision 10. Removed by business rule; this is what permits release-scoped cache keys and CDN self-healing.
- **No Redis, no live replica, no second VPS** — Decisions 11 and 28. Single-instance + WAL-to-B2 is the chosen resilience model.
- **No `ALTER TABLE ... ADD COLUMN html` materialization** — Decision 27.
- **No brotli, no HTTP/2 push, no per-shard hash directories, no XFS** — Decisions 14 and 30.
- **No server-side precompression** (Caddy `precompressed`) — Decision 23.
- **No "Purge Everything" invalidation** — Decision 4. The cache-tag purge mechanism provides per-post granularity; zone-wide purge is rejected as a thundering-herd risk.
- **No bare-metal deployment** — Decision 17.

## Open Items Requiring Confirmation

These were left for the operator to decide and do not block the P0 implementation. They are also listed in the "Deferred / future expansion" section above with full context:

1. **Browser `max-age`**: this spec recommends 300 s (five-minute template-upgrade propagation, negligible revalidation cost). A longer value (e.g. 3600 s) is acceptable if slower brand-upgrade rollout is tolerable. This is config-driven, not a code change.
2. **DR starting point**: begin with hourly `pg_dump` to B2 (simplest), or go straight to `pgBackRest` WAL archival (second-level RPO). The spec recommends starting with `pg_dump` and upgrading after the restore procedure is validated — see Decision 25 and the "pgBackRest" entry in Deferred.
