# Compression and Page Weight

Transmission minimization for the read path: HTTP compression at Caddy, CSS externalization + minification + content-hash fingerprinting, and render-time HTML minification. Under the SaaS reference instance's 3 Mbps / 1 TB budget, every byte saved is simultaneously link headroom and quota headroom (see [`caching.md`](./caching.md) _Hardware envelope_), which is why byte reduction dominates CPU optimization in this design. The decision record (zstd over brotli, no precompression, no Node toolchain) lives in [the performance-pass MRFC](../../.agents/mrfcs/implemented/2026-07-09-read-path-performance-pass.md).

## Why page weight dominates

The measured byte-level breakdown of the post page (real `post.html` + a typical 1.8 KB body):

| Component                    | Uncompressed |    gzip -9 |
| ---------------------------- | -----------: | ---------: |
| `post.html` template total   |       8789 B |          — |
| Inline `<style>` block       |       8073 B | **1798 B** |
| HTML skeleton (no CSS)       |        247 B |     ~120 B |
| Typical 1.8 KB body          |       1850 B |       90 B |
| Full page (inline CSS)       |      10639 B |     2183 B |
| Full page (CSS externalized) |       2099 B |  **283 B** |

Externalizing + compressing + caching the CSS takes a repeat visit from ~10 KB (compressed, inline) down to a few hundred bytes of body, because the compressed CSS is fetched once and then served from the browser cache across the whole site. The page body is capped by `[post] body_max_bytes` (262144 in the production config template, 32768 default); live measurements of compressibility are in `scripts/loadtest/CAPACITY_REPORT.md`.

## HTTP compression: zstd + gzip via Caddy

Every Caddyfile in the repo (`docker/Caddyfile*`, `devops/ansible/templates/Caddyfile*`) carries `encode zstd gzip`; Caddy selects per request from `Accept-Encoding`.

- **zstd (Zstandard)** matches or beats gzip's ratio at ~3× the speed and typically produces 5–10% smaller output than gzip for text/HTML at the same CPU cost.
- **gzip** is retained as the universal fallback for clients that do not advertise zstd.
- **brotli** and **server-side precompression** (Caddy's `precompressed` directive) are rejected: brotli needs a non-default Caddy build for marginal gain, and precompression only helps static assets — the one static asset here (the fingerprinted CSS) is already ~1.8 KB compressed. Dynamic HTML/raw responses cannot be precompressed regardless.

`Vary: Accept-Encoding` (set by Caddy on compression, explicitly by the handler otherwise) keeps the CDN's gzip and zstd variants in separate cache entries. `304` responses have no body and are not compressed.

## CSS externalization, minification, and fingerprinting

The CSS is the largest single byte-cost on the page, so it is extracted from the page entirely:

1. **Extract and minify at build time.** The CSS source lives in `backend/templates/post.css`; `cmd/buildcss` (invoked by `go generate ./...`) minifies it with `github.com/tdewolff/minify/v2` (the de-facto Go minifier — pure Go, no Node toolchain; the CSS is a single self-contained file with no `@import` or `url()` references, so no bundler is needed), computes `xxhash64` of the minified output, and writes `backend/internal/web/post.<hash>.css` plus the generated `backend/internal/web/csshash.go` (`var CSSHash = "<hash>"`). The CSS file is embedded into the binary via `go:embed`.
2. **Content-address the filename.** The template references `<link rel="stylesheet" href="/static/post.{{.CSSHash}}.css">`. On a CSS upgrade the new minified bytes hash differently → a different filename → a different URL; every browser fetches the new CSS because it is at a URL it has never seen, and `Cache-Control: public, max-age=31536000, immutable` is strictly correct because the URL changes whenever the content does (MDN's "cache busting"). The HTML's one-hour CDN TTL rotates to the new shell naturally — the `<link>` href change alters the rendered HTML, which the output-hashing ETag captures automatically — and no purge API is needed for static assets: the URL _is_ the version.
3. **Serve from memory.** A gin route (`v1.StaticCSS`) serves the embedded bytes at `GET /static/:filename` with the immutable cache headers — no filesystem dependency at runtime, identical behavior in every deployment context.

## HTML minification at render time

The rendered HTML response is minified with `tdewolff/minify`'s HTML minifier inside the render pipeline (`internal/service/post/post.go`): a `minify.Minifier` is constructed once in `NewService` alongside the goldmark instance and bluemonday policy and reused across goroutines (it is concurrency-safe). The minified HTML is what gets stored in the render cache and hashed for the ETag, so the ETag and the served bytes are always consistent. Minification strips whitespace, comments, and redundant tags from the shell; the already-sanitized body is safe to minify. See [`caching.md`](./caching.md) for the surrounding render-cache mechanics.
