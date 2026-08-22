# Read-Path Caching

English | [中文](caching.zh.md)

This page specifies markpost's read-path caching design: the three cache layers (browser / CDN / origin render cache), the ETag/304 scheme, the CDN purge contract, and deletion-driven invalidation. Compression and page-weight work live in [`compression.md`](./compression.md); request throttling in [`rate-limiting.md`](./rate-limiting.md). The decision record — why Cloudflare, why these TTLs, what was rejected — is [the performance-pass MRFC](../../.agents/mrfcs/implemented/2026-07-09-read-path-performance-pass.md). The operational Cloudflare layer (onboarding, SSL mode, free-tier boundaries) is [`cloudflare.md`](./cloudflare.md).

## Scope and workload

markpost's core business is **storage and distribution of Markdown content** — a notification, a temporary share, a paste — not social posts. Four facts shape the caching design:

- **Posts are write-once and immutable.** There is no edit-after-publish and no `UpdatePost` path. This collapses cache invalidation to deletion events and permits aggressive edge caching of post _bodies_.
- **Posts are short-lived.** A retention floor of 7 days is the user-experience requirement.
- **The read path is the hot path.** The product is consumed by readers clicking shared links; the write path is a low-frequency authoring operation (~0.12 writes/second mean, hard-capped at 10 posts/minute and 1000 posts/day per user).
- **Two deployment contexts.** The project ships as self-hostable software _and_ runs as an official SaaS instance. Nothing SaaS-specific is baked into application code or configuration defaults.

## Hardware envelope (SaaS reference instance)

| Resource        | Limit   | Notes                                                                                                |
| --------------- | ------- | ---------------------------------------------------------------------------------------------------- |
| CPU             | 2 cores | Shared by Caddy + Go + (Next.js) inside the markpost container; Postgres runs in a sibling container |
| Memory          | 2 GB    | Shared across all processes                                                                          |
| Disk            | 40 GB   | Postgres data + WAL + container layers                                                               |
| Bandwidth       | 3 Mbps  | **375 KB/s peak** egress                                                                             |
| Monthly traffic | 1 TB    | 375 KB/s sustained for ~30.86 days ≈ 1 TB                                                            |

**Link equals quota.** Saturating the 3 Mbps link for a month consumes the entire 1 TB allowance. Every byte saved is simultaneously link headroom and quota headroom — one budget, not two. This is why transmission-minimization (see [`compression.md`](./compression.md)) dominates the design rather than CPU optimization. Self-hosted instances with more bandwidth feel this constraint less keenly but benefit from the same optimizations.

A 3 Mbps origin cannot serve the target "a few hundred reads/second" directly: 375 KB/s ÷ ~10 KB per compressed page ≈ 25 origin responses/second. An uncached origin is physically incapable of carrying the read load, so **a CDN is a precondition for the SaaS reference instance** — not an optional enhancement. Self-hosted instances on fatter pipes run without one and accept higher origin load; nothing breaks without a CDN, because all cache logic lives at the origin (see _Self-hosting compatibility_ below).

## Three cache layers, three invalidation stories

```
Browser ──[1]──> Cloudflare edge ──[2]──> Origin VPS (Caddy → Go)
 (private)        (shared)                  (render cache + DB)
```

| Layer               | TTL                                       | Invalidated by                                               |
| ------------------- | ----------------------------------------- | ------------------------------------------------------------ |
| Browser             | `max-age=300`                             | expiry only — cannot be purged by the server                 |
| CDN                 | `s-maxage=3600`                           | expiry + synchronous origin revalidation + cache-tag purge   |
| Origin render cache | unbounded (key = QID + buildID + variant) | process restart; `DeletePost` / `PruneExpired`; release bump |

**The decisive subtlety: only the post _body_ is immutable — the HTML _response_ is not.** A Go-rendered HTML response bundles the immutable body together with a mutable shell: the `<link>` tag pointing at the CSS file, the footer brand string, the page skeleton. The shell changes whenever the CSS or template is upgraded. This is why the three TTLs differ, and why neither `immutable` nor a one-year CDN TTL applies to the HTML response.

- The **browser** cannot be purged by the server, so it gets a short TTL (300 s). When the shell changes, the next revalidation after 300 s picks up the new version.
- The **CDN** can revalidate against the origin, so it holds the page for one hour; when the shell has changed, the origin returns `200` with a new ETag and a fresh body, and the CDN swaps its copy. A one-year TTL is deliberately not used — it would freeze a stale shell until manual purge, and the one-hour TTL lets renderer/CSS upgrades propagate within an hour without a zone-wide purge. (The Cloudflare free tier does support per-post cache-tag purge — see _CDN caching_ below — but the purge API remains an _active-deletion_ mechanism, not a release-deployment mechanism.)
- The **origin render cache** keys on QID _plus release dimensions_; a release bump rotates the whole key namespace automatically (a release ships a new binary, which restarts the process, which clears the in-memory cache anyway).

## ETag design — hash the rendered response, not its inputs

`ETag` is the fingerprint of the _response body_, and the response bundles the immutable body with a mutable shell plus the renderer itself (goldmark + bluemonday + the raw-HTML neutralizer, any of which may change between releases). The only way to guarantee the ETag reflects **everything that determines the rendered bytes** is to hash the rendered output:

```
ETag (HTML) = xxhash64( minified renderedHTML )          // the exact bytes served
ETag (raw)  = xxhash64( "# " + title + "\n\n" + body )   // the exact bytes of the raw response
```

Hashing the inputs (`body + title + cssHash + templateVersion`) cannot work: a goldmark or bluemonday upgrade changes the rendered HTML but leaves the inputs unchanged, so the CDN's revalidation would hit `If-None-Match` equality, return `304`, and keep renewing a stale shell rendered by the old code. Hashing what the client actually receives makes that class of bug impossible — any change to the renderer, the template, or the CSS (which changes the `<link>` href in the shell) automatically produces a different ETag; no dimension checklist is needed.

`xxhash64` (`github.com/cespare/xxhash/v2`) is used instead of SHA-256: ETag generation needs no cryptographic collision resistance, xxhash is ~20× faster, and it arrives as a transitive dependency via ristretto. The 64-bit value is hex-encoded to 16 characters; collision probability (2⁻⁶⁴) is negligible for cache validation. The ETag is computed **once per cache miss**, inside `singleflight.Do` — the cost of hashing the full rendered HTML is paid only by the leader of a cold-miss burst, never by the hot path.

## Render-cache key — QID + buildID + variant

```
cache key (HTML) = qid + ":" + buildID + ":html"
cache key (raw)  = qid + ":" + buildID + ":raw"
cache value      = { title, body, etag, createdAt }    // stored together
```

Within a process lifetime the renderer, template, and CSS are all constants (built once in `NewService`), so the QID alone determines the output; a release ships a new binary, restarts the process, and clears the in-memory cache. `buildID` (`internal/web/buildid.go`, a compile-time-injected short hash of the build) is retained only as defense against a future hot-reload of templates without restart. The `:html`/`:raw` suffix separates the two variants so they do not collide. The value stores `createdAt` alongside body and ETag so the handler can emit `Last-Modified` without a DB round-trip.

## The render pipeline behind a cache miss

On a miss, the leader runs the full pipeline (`RenderPostHTML`, `internal/service/post/post.go`): Postgres `GetByQID` read (sub-millisecond on the unique QID index) → goldmark render (one shared, concurrency-safe `goldmark.Markdown` instance) → raw-HTML neutralization (a regex pass escaping the opening `<` of raw-text/RCDATA elements so an unterminated tag cannot swallow the document) → bluemonday sanitize (a shared `UGCPolicy`-derived policy, the most expensive step — a full HTML5 tokenizer pass) → `addNoReferrerToImages` → HTML minification (see [`compression.md`](./compression.md)). Steps after the DB read are a pure function of `Body`; on an immutable post they produce byte-identical output until the process restarts. The `?format=raw` variant's "render" is plain string concatenation (`"# " + title + "\n\n" + body`) — no goldmark/bluemonday pass.

## `singleflight` + ristretto, composed

The fast path is a ristretto `Get`; only on a miss does the request enter `singleflight.Do`, and inside `Do` the code re-checks the cache to avoid racing a concurrent fill:

```go
func (s *Service) RenderPostHTML(ctx context.Context, qid string) (title, html, etag string, createdAt time.Time, err error) {
    key := qid + ":" + buildID + ":html"

    if v, ok := s.cache.Get(key); ok {          // fast path — no lock, no Do
        return v.title, v.body, v.etag, v.createdAt, nil
    }

    v, err, _ := s.group.Do(key, func() (any, error) {
        if v, ok := s.cache.Get(key); ok {      // double-check inside Do
            return v, nil
        }
        ... render, minify, etag := etagHex(minified) ...
        s.cache.Set(key, r, int64(len(r.body)))
        return r, nil
    })
    ...
}
```

| Layer                          | Defends against               | Mechanism               |
| ------------------------------ | ----------------------------- | ----------------------- |
| ristretto `Get` (outside `Do`) | repeats across time           | map lookup, nanoseconds |
| `singleflight.Do`              | concurrency within an instant | `WaitGroup` barrier     |
| ristretto `Get` (inside `Do`)  | race during leader execution  | double-checked fill     |

**ristretto** (`internal/service/post/cache.go`) is chosen for TinyLFU admission control: read access is Zipfian, and a plain LRU loses its hot set to a burst of one-time cold accesses (a crawler sweep, a batch share). TinyLFU's frequency sketches admit a new entry only if it is "hotter" than what it would evict. `MaxCost` is set in _bytes_ (default 128 MiB via `[render] cache_size_bytes`; entries cost their body length), writes are batched asynchronously so the hot path never blocks on eviction bookkeeping, and `NumCounters` (~10× the expected key count) keeps the sketch accurate. The cache is config-driven: `[render] enabled` can disable it, and the size can shrink for small instances.

## HTTP cache headers, in detail

The HTML response from `RenderPost` carries:

```http
ETag: "<xxhash64(minified renderedHTML)>"
Last-Modified: <Post.CreatedAt as HTTP date>
Cache-Control: public, max-age=300, s-maxage=3600
Cache-Tag: post-<qid>
Vary: Accept-Encoding
```

The `?format=raw` response carries the same `Cache-Control`/`Cache-Tag`/`Vary`/`Last-Modified` with `ETag: <xxhash64("# "+title+"\n\n"+body)>`. The hashed CSS asset (served at `/static/post.<cssHash>.css`) carries `Cache-Control: public, max-age=31536000, immutable` (see [`compression.md`](./compression.md)). Post-page 404s carry `Cache-Control: public, max-age=60, s-maxage=60` (`setNotFoundCacheHeader`, `internal/api/rest/v1/post.go`) so QID-enumeration probes are absorbed at the CDN edge instead of re-originating on every request; only the not-found case is marked — other errors stay uncacheable.

- **`public`** allows shared caches (the CDN) to store the response in addition to the browser.
- **`max-age=300`** is the browser's freshness lifetime: within 300 s the browser serves from disk with no network activity at all.
- **`s-maxage=3600`** overrides `max-age` for shared caches only — the knob that lets the CDN absorb the overwhelming majority of reads.
- **No `stale-while-revalidate`.** Per RFC 9111, `s-maxage` incorporates the semantics of `proxy-revalidate`, which prohibits shared caches from serving stale content; at Cloudflare the directive is a no-op and revalidation is synchronous (`EXPIRED`) rather than a background refresh. Keeping a directive that does nothing would mislead readers. Synchronous revalidation is cheap here — the `304` is bodyless and the origin render cache serves the ETag without re-rendering. (Making SWR take effect would require dropping `s-maxage` and setting the CDN TTL via an Edge Cache TTL rule, whose free-tier minimum is 2 hours — conflicting with the one-hour upgrade-propagation target.)
- **`immutable` appears only on the CSS asset**, never on HTML or raw. The HTML and raw responses live at URLs that do not change on release (`/:qid`, `/:qid?format=raw`) and a post can be actively deleted; marking either `immutable` would be factually incorrect and would prevent the browser from ever learning of a deletion. The raw body is immutable per-QID, but its URL is not content-addressed, so it gets the same TTL scheme as HTML.
- **`Last-Modified`** is `Post.CreatedAt` — posts are write-once, so it is the true last-modified time. It serves as the secondary validator (RFC 9110 recommends sending both); `If-None-Match` takes precedence over `If-Modified-Since`, so a stale `Last-Modified` cannot cause a wrong `304` — the ETag wins, and the ETag tracks shell/renderer upgrades.
- **`Cache-Tag: post-<qid>`** is Cloudflare's surrogate key. Both HTML and raw variants of a post carry the same tag, so one purge-by-tag call invalidates both regardless of how many `Accept-Encoding` variants the CDN holds. Cloudflare strips the header from visitor-facing responses.
- **`Vary: Accept-Encoding`** keeps gzip and zstd cache entries separate so a zstd-capable browser never receives a mismatched gzip'd body. Caddy's `encode` adds the header when it compresses; setting it explicitly in the handler covers the no-compression fallback.
- **`Cache-Control: no-store` on API responses.** The `/api/v1` group wraps every response in `NoStore` middleware — dynamic payloads (notably `/oauth/url`, whose body carries a one-time CSRF state) must never be cacheable by a shared cache. Handlers may still override the header afterwards for deliberately cacheable responses.

### Who handles the 304

| Situation                                     | Handler                                                                                        |
| --------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| CDN edge hit, browser revalidates             | **Cloudflare** — answers `304` from its stored ETag; the origin never sees the request         |
| CDN copy past TTL, revalidates against origin | **Gin handler** — compares `If-None-Match` against the render-cache ETag; match → bodyless 304 |
| No CDN                                        | **Gin handler**, same path                                                                     |

On a cache hit the handler skips goldmark/bluemonday entirely; on a miss the render happens and fills the cache for subsequent requests. Caddy neither generates nor compares ETags — it is a reverse proxy that passes headers through and compresses bodies. `304` responses have no body and are not compressed.

## CDN caching: Cloudflare free tier and the purge contract

Cloudflare's free tier is the load-bearing choice: unlimited bandwidth (the projected ~7.8 TB/month of edge egress costs $0; a metered CDN at $0.085/GB would cost ~$660/month and re-impose the 1 TB constraint by invoice), unmetered DDoS protection (a 2-core origin behind 3 Mbps cannot survive an uncached flood), and a global anycast edge (~330 POPs). The 100k/day free-tier limit applies to **Workers** (edge compute), which markpost does not use; the CDN cache path is static, header-driven, and has no request limit. Lock-in risk is near-zero: Cloudflare is a reverse proxy reachable purely by DNS, and no proprietary API is embedded in the application (the `[cloudflare]` config section is optional).

**Purge API.** All purge methods are available on every plan (purge by URL, cache-tag, prefix, hostname, "purge everything"). Free-tier limits, per account via a token bucket: 5 purge requests/minute, bucket capacity 25, 100 operations (tags/URLs) per request; purge latency is documented under 150 ms globally. The design uses **purge by cache-tag**: deleting a post issues a single `POST /zones/{zone}/purge_cache` with `{"tags":["post-<qid>"]}`. Even at a hypothetical 3 000 deletions/day the average purge rate (~2/minute) sits far under the ceiling. "Purge everything" is rejected: it forces every cached post to re-origin simultaneously — a thundering herd that can flatten the origin. The cache-tag mechanism provides per-post granularity without that risk, and a by-URL fallback (800 URLs/second on Free) exists if cache-tag availability ever changes.

**Purge is best-effort and asynchronous.** The delete handler removes the origin render-cache entry synchronously (mandatory), then enqueues the Cloudflare purge call on a background goroutine via the `Purger` interface (`internal/service/post/purger.go`): `cloudflarePurger` when `[cloudflare] api_token` + `zone_id` are configured, `noopPurger` otherwise (self-hosted without Cloudflare — the CDN copy falls back to natural TTL expiry). The QID is sanitized (`sanitizeCacheTag`) before entering the JSON body; failures are logged and swallowed, and no retry is attempted. Active deletion is therefore **immediate at the origin, near-immediate at the CDN (typically <150 ms), and at most 5 minutes stale at the browser** (`max-age=300`).

## Deletion and invalidation

Deletion endpoints: `DELETE /api/v1/posts/:id` (JWT owner) and `DELETE /api/v1/admin/posts/:id` (admin). `DeletePostByQID(ctx, qid, ownerID)` removes the DB row (owner-scoped when `ownerID > 0`, unconstrained for the admin path; `ErrNotFound` when no row matched), removes both render-cache entries synchronously — the ristretto wrapper's `Delete` calls `cache.Wait()`, so a pending buffered `Set` from a concurrent render cannot re-admit the entry after the deletion — and enqueues the best-effort cache-tag purge above.

`PruneExpired` (the housekeeping prune of already-expired content) removes DB rows and origin cache entries (the repo returns the pruned QIDs so the service can invalidate them) but does **not** purge: stale-but-harmless delivery of already-expired ephemeral content is the accepted tradeoff, and prune volume could be large. CDN edge copies of pruned posts linger up to their one-hour TTL; readers in that window get a stale-but-harmless 200.

## Request-flow walkthrough

1. **First visit, browser and CDN cold.** Browser → Cloudflare edge → origin. Go renders (or serves from the origin render cache), Caddy compresses, the response flows back with cache headers. The edge stores it for one hour; the browser for 300 s.
2. **Repeat visit within 300 s.** The browser uses its local copy — zero network traffic.
3. **Repeat visit after 300 s, CDN still fresh.** The browser sends a conditional request; the edge copy has not expired, so Cloudflare itself answers `304`. The origin is never contacted.
4. **CDN copy past one hour.** Cloudflare sends a conditional request to the origin. If nothing changed, the origin returns a bodyless `304`; after a release, a `200` with the new body, and the edge swaps its copy.
5. **First visit from a new geographic region.** That region's edge node performs one origin fetch; subsequent regional visitors hit that edge. Origin load scales with the number of _edges that have seen the URL_, not with total request count.
6. **Post deleted by retention prune.** Origin removes it from cache and DB; the edge copy lingers up to one hour (stale-but-harmless).
7. **Post deleted by user or admin.** Origin removal + best-effort cache-tag purge (see above).

## Deployment-window analysis (release-induced origin load)

A release ships a new binary, restarts the process, and clears the in-memory render cache — every post whose CDN copy revalidates afterwards misses and renders fresh. This does not arrive as a spike: CDN copies revalidate lazily, each on its own TTL schedule (set when the copy was last filled or verified, not at release time), so revalidations distribute across the hour following the release, and ~330 POPs stagger even a single post's per-region copies. Worst-case arithmetic — 1 000 000 cached posts all revalidating within one hour — yields ~278 req/s ≈ 42% of two cores; real active-cache populations are far smaller, so 42% is a ceiling, not an expectation. The genuine thundering-herd case (one very hot post revalidating simultaneously from many POPs) is what `singleflight` defeats: fifty concurrent revalidations of the same QID collapse to one render, and the forty-nine waiters receive its result. Deploying in a low-traffic window remains a free operational lever for extra margin.

## Self-hosting compatibility

Every component falls into one of two tiers:

| Tier                    | Components                                                                                                                                                                                                                                     | Self-hosted behavior                                                                                                                         |
| ----------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| **In-image, always on** | Caddy `encode`, CSS externalization + minify + `go:embed`, HTML minify, HTTP cache headers + ETag/304, Postgres pool/lz4/GUC tuning, singleflight+ristretto, three-limiter + login rate limiting, delete endpoints + origin cache invalidation | Pure code; ships in the image with zero configuration.                                                                                       |
| **External, optional**  | Cloudflare CDN, B2/WAL backup                                                                                                                                                                                                                  | Operational layers hung in front of / behind the image; use, ignore, or substitute equivalents. Nothing in application code references them. |

A CDN is recommended, not required: a self-hosted instance on a fatter pipe can run without one and accept higher origin CPU and bandwidth use. Configuration is config-driven, not compile-driven — the render cache has an `[render] enabled` flag and a tunable size, and no SaaS-specific value appears in `config.go` or `config.example.toml`. The three deployment modes (SaaS / self-hosted with a domain / homelab) differ only in Caddyfile, DNS, and the optional `[cloudflare]` section; the Go binary is identical. Full topology and onboarding: [`cloudflare.md`](./cloudflare.md); backup and restore: [`disaster-recovery.md`](./disaster-recovery.md).
