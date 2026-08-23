# Cloudflare CDN Integration

English | [中文](cloudflare.zh.md)

This document specifies how markpost integrates with Cloudflare's free-tier CDN for the SaaS reference instance, and records the three deployment modes the project supports. It is the authoritative reference for the DNS, TLS, caching, and cache-purge decisions. All Cloudflare behavior claims cite the Cloudflare documentation at `~/Workspace/contexts/cloudflare/cloudflare-docs/` (repo version 2026-07-08); citations use paths relative to that root.

The caching and invalidation _design_ (three cache layers, ETag scheme, cache-tag purge, render cache) lives in [`caching.md`](./caching.md). This document covers the _operational_ layer: how to wire a VPS origin to Cloudflare, which SSL mode to choose, how the purge API is called, and what the free-tier limits are.

## Deployment Modes

markpost ships as self-hostable software _and_ runs as an official SaaS instance. Every design choice must work in both contexts: nothing SaaS-specific is baked into application code or configuration defaults ([`caching.md`](./caching.md#self-hosting-compatibility)). The three modes differ only in deployment topology, Caddyfile, and DNS — the Go binary is identical.

| Mode                          | Origin    | DNS                                          | TLS termination                                                                                 | CDN | Caddyfile                                                                                | Typical use                        |
| ----------------------------- | --------- | -------------------------------------------- | ----------------------------------------------------------------------------------------------- | --- | ---------------------------------------------------------------------------------------- | ---------------------------------- |
| **SaaS**                      | VPS       | domain → Cloudflare (Proxied / orange cloud) | Cloudflare edge (visitor side) + Origin CA on the host Caddy gateway (origin side, Full strict) | yes | host gateway `markpost.caddy.j2` + `Caddyfile.production.j2` (plain HTTP, loopback-only) | official instance                  |
| **Self-hosted (with domain)** | VPS / NAS | domain → origin IP (DNS-only / gray cloud)   | Caddy automatic Let's Encrypt + HTTPS redirect                                                  | no  | per-domain site block (not yet templated in repo)                                        | personal / small-team self-hosting |
| **Homelab**                   | NAS       | none (LAN IP:port)                           | none (plaintext HTTP)                                                                           | no  | `docker/Caddyfile` (`:2053`, TLS-less)                                                   | home network, trusted LAN          |

**The CDN is a precondition for the SaaS reference instance only.** A 3 Mbps origin cannot serve hundreds of reads/second directly ([`caching.md`](./caching.md#hardware-envelope-saas-reference-instance)). Self-hosted instances on fatter pipes run without one and accept higher origin load. Nothing breaks without a CDN: all cache logic lives at the origin, and the CDN only adds an edge tier.

## SaaS Mode: Cloudflare Onboarding

The topology is:

```
Visitor ──HTTPS:443──> Cloudflare edge (orange cloud, Full strict) ──HTTPS:443──> host Caddy gateway ──HTTP──> 127.0.0.1:8080 → container Caddy :2053 ──> Go :7330 / static export
         [TLS-A: Cloudflare edge certificate]                  [TLS-B: Origin CA on the host gateway]  [plain HTTP, published loopback-only]
```

The edge listens on the visitor-facing port 443 and connects to the origin on 443 as well — terminated by the host's systemd Caddy (the shared gateway, `import /etc/caddy/conf.d/*.caddy`), which reverse-proxies `markpost.cc` to the container's loopback-published port. The container's Caddy serves the Next.js static export from `/app/frontend` and reverse-proxies the API paths to the Go binary — there is no Node server.

The gateway uses a domain site block with the manually installed Origin CA certificate; the container Caddy listens on a bare port, plain HTTP. Neither runs automatic Let's Encrypt in SaaS mode — a proxied domain resolves to Cloudflare IPs, so an ACME HTTP-01 challenge would never reach the VPS. The sections below explain the pieces.

### DNS and the orange cloud

A full DNS setup is used (`fundamentals/manage-domains/add-site.mdx`):

1. Add the domain to Cloudflare (Free plan).
2. Create an **A record** whose content is the **VPS real IP**. This IP is the origin address Cloudflare uses for origin pulls — "Your origin server address (cannot be a Cloudflare IP)" (`dns/manage-dns-records/reference/dns-record-types.mdx`).
3. Set the record to **Proxied (orange cloud)**.
4. Change nameservers at the registrar to the Cloudflare-assigned NS pair.

The decisive point, often misunderstood: with the orange cloud on, **a public DNS query returns a Cloudflare anycast IP, not the VPS IP**. The VPS IP in the A record is only seen by Cloudflare (for origin pulls), never by visitors:

> "A DNS query to the proxied record `blog.example.com` will be answered with Cloudflare anycast IP addresses … instead of `192.0.2.1`. This ensures that HTTP/HTTPS requests for this name will be sent to Cloudflare's network and can be proxied…" — `dns/proxy-status/index.mdx`

This is what "hiding the origin behind the CDN" means. DNS-only (gray cloud) would instead "respond with your server's actual IP address … This exposes your origin IP address" (`dns/proxy-status/index.mdx`). Only A/AAAA/CNAME records can be proxied; MX/TXT are always DNS-only.

Proxied records have a fixed TTL of Auto (300 s, not editable). Cloudflare recommends proxying all A/AAAA/CNAME records that serve web traffic.

### TLS: two segments and the SSL mode choice

Cloudflare splits TLS into two independent segments:

- **TLS-A (visitor ↔ edge):** terminated at the edge with a Cloudflare-managed certificate, automatic and free.
- **TLS-B (edge ↔ origin):** governed by the **SSL/TLS encryption mode** selected in the dashboard.

The four modes (`ssl/origin-configuration/ssl-modes/*.mdx`):

| Mode              | edge → origin                 | Origin cert required                                  | Failure code | Security                         |
| ----------------- | ----------------------------- | ----------------------------------------------------- | ------------ | -------------------------------- |
| Off               | none                          | no                                                    | —            | worst (TLS-A also off)           |
| Flexible          | plaintext HTTP                | no                                                    | —            | medium (TLS-B plaintext)         |
| Full              | HTTPS, **not validated**      | yes (self-signed/CA)                                  | 525          | medium-high (MITM can swap cert) |
| **Full (strict)** | HTTPS, **strictly validated** | yes (unexpired, public CA or Origin CA, CN/SAN match) | 526          | highest                          |

markpost selects **Full (strict)**. Cloudflare strongly recommends it:

> "For the best security, choose Full (strict) mode whenever possible. … You can use a certificate from a publicly trusted certificate authority (CA), or generate a free Origin CA certificate from Cloudflare." — `ssl/get-started.mdx`

Flexible is explicitly discouraged for applications with login:

> "If your application contains sensitive information (user login), use Full or Full (Strict) modes instead." — `ssl-modes/flexible.mdx`

markpost has user login, admin write paths, and password change — it is exactly such an application. The earlier "TLS-less Caddy" topology was equivalent to Flexible and is superseded by this decision.

**Port.** Visitors reach the edge on the canonical HTTPS port 443, and the origin leg must land on 443 too: of Cloudflare's proxied HTTPS ports (443, 2053, 2083, 2087, 2096, 8443), every one except 443 is listed under "Ports supported by Cloudflare, but with caching disabled" (`fundamentals/reference/network-ports.mdx`), and re-enabling any of them via `additional_cacheable_ports` is Enterprise-only (`cache/how-to/cache-rules/settings.mdx`). The earlier 2053 + Origin Rule topology (an Origin Rule rewriting the destination port 443 → 2053, the rules themselves available on Free at 10 — `rules/origin-rules/index.mdx`) held the whole zone at `CF-Cache-Status: DYNAMIC` for exactly this reason: rule availability was checked, the cache-port matrix was not. Since 443 binds to exactly one process, the VPS's shared host Caddy owns it and services join as `conf.d` site blocks; markpost's container publishes its plain-HTTP port loopback-only (`127.0.0.1:8080` → container `:2053`). Services that need no edge cache can stay on the cache-disabled ports.

### Origin CA certificate

An Origin CA certificate is Cloudflare's free, long-lived (15-year) origin certificate, trusted only by Cloudflare — ideal for a self-hosted VPS origin behind the orange cloud, with no renewal overhead.

**Steps:**

1. Dashboard → **SSL/TLS → Origin Server → Create Certificate**.
2. Key type RSA (2048); hostnames covering the domain; validity **15 years**.
3. Download both the **Origin Certificate** and the **Private Key** (the private key is shown only once).
4. Store on the VPS, e.g. `/app/certs/origin.pem` and `/app/certs/origin.key`.

The host Caddy gateway presents this certificate for TLS-B — a domain site block whose `tls` directive points at the certificate files, deployed to `/etc/caddy/conf.d/markpost.caddy` from `devops/ansible/templates/markpost.caddy.j2` (the playbook copies the cert from `~/docker/markpost/certs/` to `/etc/caddy/certs/markpost/`). Schematic of the gateway block and the plain-HTTP container listener it feeds (`devops/ansible/templates/Caddyfile.production.j2`):

```caddyfile
# gateway: /etc/caddy/conf.d/markpost.caddy (host Caddy, binds 443)
markpost.cc {
    tls /etc/caddy/certs/markpost/origin.pem /etc/caddy/certs/markpost/origin.key
    reverse_proxy 127.0.0.1:8080
}

# container: plain HTTP, published loopback-only (127.0.0.1:8080 -> :2053)
:2053 {
    encode zstd gzip
    handle /api/v1/* {
        reverse_proxy 127.0.0.1:7330 {
            trusted_proxies private_ranges
            header_up X-Forwarded-For {http.request.header.CF-Connecting-IP}
        }
    }
    # /static/*, /swagger/*, /mpk-*, /p-* blocks are identical
    handle {
        root * /app/frontend
        file_server
    }
}
```

Automatic Let's Encrypt is **not** used in SaaS mode: the proxied domain resolves to a Cloudflare IP, so an ACME HTTP-01 challenge would not reach the VPS and issuance would fail. The Origin CA certificate replaces it.

### Origin protection: only Cloudflare may connect

The orange cloud does **not** by itself prevent someone who learns the VPS IP from connecting directly to it, bypassing Cloudflare entirely:

> "If someone discovers your origin server's IP address … they could send traffic directly to your server, bypassing Cloudflare's security protections entirely. To prevent this, block all traffic that does not come from Cloudflare IP addresses…" — `fundamentals/concepts/cloudflare-ip-addresses.mdx`

Two layers, applied to the **VPS host firewall** (iptables, not Caddy):

1. **Allowlist Cloudflare CIDRs, drop everything else** (recommended baseline). The authoritative CIDR list is `https://www.cloudflare.com/ips/` (the docs do not inline the ranges). This is "moderately secure" and "vulnerable to IP spoofing" (`partials/learning-paths/limit-external-connections-network.mdx`).
2. **Authenticated Origin Pulls (AOP, optional hardening).** mTLS where Cloudflare presents a client certificate to the origin, so "any HTTPS requests outside of Cloudflare will not receive a response from your origin" (`ssl/origin-configuration/authenticated-origin-pull/explanation.mdx`). AOP requires Full or Full (strict) — which is why the SSL mode decision above is a prerequisite. AOP is a follow-on hardening step, not part of the initial onboarding.

This host-firewall layer is distinct from Caddy's `trusted_proxies` (forwarded-header handling, not packet filtering) — the CIDR allowlist is the packet-layer enforcement that, together with Cloudflare's edge overwrite of `CF-Connecting-IP`, makes the client-IP relay below trustworthy.

### Client IP detection

Behind the orange cloud, the origin sees only Cloudflare IPs as direct peers. Cloudflare adds headers carrying the real visitor IP (`fundamentals/reference/http-headers.mdx`):

- **`CF-Connecting-IP`** — the real client IP, single value, set only on the edge→origin leg. Recommended by the docs.
- **`True-Client-IP`** — identical to `CF-Connecting-IP` in value, but Enterprise-only.
- **`X-Forwarded-For`** — the proxy chain (comma-separated).

markpost does **not** use `TrustedPlatform = gin.PlatformCloudflare` (which trusts `CF-Connecting-IP` unconditionally, with no CIDR check, at the application layer). Instead, `CF-Connecting-IP` passes through the host Caddy gateway untouched (it is not a hop-by-hop header), and every `reverse_proxy` block in the container's `Caddyfile.production.j2` carries `header_up X-Forwarded-For {http.request.header.CF-Connecting-IP}`. Caddy applies user header operations after its default forwarded-header handling, so the `X-Forwarded-For` that reaches Go is always the single `CF-Connecting-IP` value — the multi-hop chain (visitor-supplied XFF, Cloudflare's populated chain, the gateway hop, Caddy's own hop) never reaches the application.

The collapse is required for correctness, not only hygiene: gin trusts only the loopback peer (`SetTrustedProxies(["127.0.0.1", "::1"])`, matching Caddy's loopback proxy hop), and `ClientIP()` walks `X-Forwarded-For` right-to-left, returning the first IP it does not trust. If Caddy forwarded the appended chain `<real-client>, <cloudflare-hop>`, gin would return the Cloudflare edge IP as the client for every visitor, collapsing the IP-keyed L1/login limiters onto a handful of edge addresses.

Trust anchors for that single value: Cloudflare overwrites any visitor-supplied `CF-Connecting-IP` at the edge, so on traffic that has passed through Cloudflare the header is Cloudflare-asserted; the host firewall (see _Origin protection_ above) keeps every non-Cloudflare peer off port 443 at the packet layer; and the container's port is published loopback-only, so the only possible direct peer is the host gateway itself. Residual exposure: a peer that bypasses the firewall can forge `CF-Connecting-IP` and the rewrite propagates the forged value — the firewall is the enforcement point for header authenticity. `trusted_proxies private_ranges` on each block covers the gateway's NAT'd bridge address; the XFF value itself is fixed by the `header_up` rewrite. The detailed mechanism is documented in [`rate-limiting.md`](./rate-limiting.md#ip-resolution-gin-not-tollbooth).

Operational requirement: the `cloudflare_cidrs` Ansible var in `devops/ansible/group_vars/production/vars.yml` carries the current Cloudflare CIDR list for the host firewall — the only enforcement point; no template consumes the list. Cloudflare occasionally updates these ranges; operators must resync the firewall (the var is the list's documented home). This maintenance responsibility is documented here and in [`rate-limiting.md`](./rate-limiting.md#ip-resolution-gin-not-tollbooth).

## Caching

### What Cloudflare caches by default

Cloudflare caches **by file extension, not by MIME type**, and **does not cache HTML or JSON by default** (`cache/concepts/default-cache-behavior.mdx`). Default-cached extensions are a set of static asset types (CSS, JS, JPG, PNG, PDF, WOFF2, SVG, …); HTML/JSON/API responses return `DYNAMIC` and go straight to origin unless a Cache Rule makes them eligible.

Origin headers alone cannot extend eligibility to non-default content — "To cache additional content, refer to Cache Rules" (`cache/concepts/default-cache-behavior.mdx`) — so a **Cache Rule** makes post pages cacheable: `URI Path` `starts with` `/p-` → **Eligible for cache** (dashboard: Rules → Overview → Create rule → Cache Rule), with the rule's Edge TTL left unset. Once eligible, the TTL comes from the origin: `RenderPost` serves HTML with `Cache-Control: public, max-age=300, s-maxage=3600` (`backend/internal/api/rest/v1/post.go:72`).

Cloudflare respects origin cache headers under **Origin Cache Control (OCC)**, which is enabled by default and cannot be disabled on Free/Pro/Business (`cache/concepts/cache-control.mdx`). This is why the `s-maxage`/`max-age` directives in markpost's headers take real effect.

### Why write/auth/dynamic traffic is safe behind the CDN

Placing all traffic (including writes and authenticated requests) behind the orange cloud is safe because Cloudflare defaults to **not caching** these:

- **Non-GET methods are never cached.** "Cloudflare does not cache the resource when … The HTTP request method is anything other than a GET." (`cache/concepts/default-cache-behavior.mdx`). All markpost writes (create post, delete, change password, delivery writes) are POST/PUT/DELETE → transparently proxied to origin.
- **`Authorization` responses are not cached** under OCC unless `must-revalidate`, `public`, or `s-maxage` is also present (`cache/concepts/cache-control.mdx`). markpost's authenticated routes carry `Authorization: Bearer …` and their responses never carry those directives → BYPASS, each re-originates.
- **`Set-Cookie` responses are not cached** under OCC (`cache/concepts/cache-control.mdx`). Login/OAuth/refresh responses set cookies → BYPASS.

The only path that is intentionally edge-cached is the public read `GET /:qid`. Everything else flows to origin on every request.

<a id="cf-cache-status-reference"></a>

### `CF-Cache-Status` reference

The `CF-Cache-Status` response header diagnoses cache behavior (`cache/concepts/cache-responses.mdx`):

| Value        | Meaning                                                                         |
| ------------ | ------------------------------------------------------------------------------- |
| HIT          | served from edge cache                                                          |
| MISS         | not in cache, fetched from origin                                               |
| EXPIRED      | was cached but TTL elapsed; synchronously revalidated                           |
| REVALIDATED  | origin confirmed unchanged via conditional request (304); served from cache     |
| UPDATING     | expired, serving stale during background revalidation (async SWR path)          |
| STALE        | expired and origin unreachable; serving stale                                   |
| DYNAMIC      | not eligible for caching; straight to origin (HTML not covered by a Cache Rule) |
| BYPASS       | origin instructed bypass (`no-cache`/`private`/`max-age=0`, or Set-Cookie)      |
| NONE/UNKNOWN | Cloudflare responded before reaching cache (Worker, WAF block, redirect)        |

If post pages consistently show `DYNAMIC`, the Cache Rule is missing or not matching `/p-*`; `MISS` then `HIT` on repeat confirms the cache works.

## Cache Purge

### The purge API contract

markpost issues a best-effort **cache-tag purge** on post deletion. The correct invocation, verified against the Cloudflare API:

- **Endpoint:** `POST https://api.cloudflare.com/client/v4/zones/{zone_id}/purge_cache` (`cache/how-to/purge-cache/purge-zone-versions.mdx`).
- **Authentication:** `Authorization: Bearer <api_token>`, using an **API Token** (scoped per zone; not the account-wide Global API Key). The token needs the **Zone → Cache Purge** permission (`fundamentals/api/reference/template.mdx`: "Zone Cache Purge | Cache Purge | Zone").
- **Body (purge by cache-tag):** `{"tags":["post-<qid>"]}`. This is the "purge cached content by tag, host, or prefix" form of the `purge_cache` endpoint.
- **Best-effort, no retry.** A dropped or rate-limited request falls back to natural `s-maxage=3600` TTL expiry — never worse than the self-healing the design already relies on.

**Implementation verification.** `backend/internal/service/post/purger.go` matches this contract exactly:

| Contract element        | `purger.go` location | Code                                                                                         |
| ----------------------- | -------------------- | -------------------------------------------------------------------------------------------- |
| Endpoint                | `:48`                | `fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/purge_cache", cfg.ZoneID)`       |
| Bearer auth             | `:68`                | `req.Header.Set("Authorization", "Bearer "+p.apiToken)`                                      |
| Tag body                | `:56-57`             | `tag := "post-" + sanitizeCacheTag(qid)`; `json.Marshal(map[string][]string{"tags": {tag}})` |
| Best-effort             | `:77-79`             | `if resp.StatusCode >= 300 { log.Printf(...) }` (swallowed, no error returned)               |
| No-op when unconfigured | `:95-101`            | `newPurger` returns `noopPurger` when `APIToken` or `ZoneID` is empty                        |

The implementation is correct. There is deliberately **no** `cloudflare-go` SDK dependency: purge is a single best-effort POST, and a raw `net/http` call keeps the dependency surface at zero. The five purge types (everything / by URL / by tag / by prefix / by hostname) all hit the same endpoint with different bodies; markpost only needs by-tag.

### Cache-Tags: how purge-by-tag works

Cache-Tags are Cloudflare's surrogate keys. The origin tags responses with a special header; Cloudflare associates the tag with the cached object and lets you bulk-purge by tag without enumerating URLs (`cache/how-to/purge-cache/purge-by-tags.mdx`):

1. Origin sets `Cache-Tag: post-<qid>` on the response.
2. Traffic is proxied through Cloudflare (orange cloud).
3. Cloudflare associates the tag with the cached content, and **strips the header before delivering to visitors** (it never leaks to clients).
4. A purge by that tag forces a cache MISS on all content carrying it.

markpost sets `Cache-Tag: post-<qid>` in `RenderPost` (`backend/internal/api/rest/v1/post.go:73`). Both the HTML and raw variants of a post carry the same tag, so one purge invalidates both regardless of how many `Accept-Encoding` entries the edge holds. **This header is the hard precondition for the entire purge mechanism** — without it, purge-by-tag matches nothing.

### Purge limits (Free tier)

The Free-tier purge limits, applied **per account** via a token-bucket model (`cache/how-to/purge-cache/index.mdx`, data in `plans/index.json`):

| Dimension                          | Free                  | Pro        | Business    | Enterprise  |
| ---------------------------------- | --------------------- | ---------- | ----------- | ----------- |
| Request rate (tag/prefix/hostname) | **5 / minute**        | 5 / second | 10 / second | 50 / second |
| Token bucket capacity              | **25**                | 25         | 50          | 500         |
| Max operations per request         | **100**               | 100        | 100         | 100         |
| by-URL rate                        | **800 URLs / second** | 1500 / s   | 1500 / s    | 3000 / s    |

The token bucket is not a hard "5/minute ceiling": it holds up to 25 tokens, refilled at 5/minute, so short bursts (up to 25 at once) are absorbed; only when the bucket is empty does a request get rate-limited. markpost purges one tag per deletion — far below any limit. At a hypothetical 3000 deletions/day the average is ~2/minute, comfortably under the 5/minute rate.

Purge is triggered only by **active user/admin deletion** (`DeletePostByQID`). The housekeeping `PruneExpired` deliberately does **not** purge: it reaps already-expired ephemeral content, where stale-but-harmless delivery is acceptable and the prune volume could be large ([`caching.md`](./caching.md#deletion-and-invalidation)).

## Free-tier limits at a glance

Aggregated from `plans/index.json` (cache segment) and the cache docs:

| Capability                          | Free tier                         | Notes                                                                                                                                                     |
| ----------------------------------- | --------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| CDN storage / bandwidth / requests  | free, no metered quota            | only explicit statement: "users can continue to use Cloudflare's CDN (without Cache Reserve) for free" (`cache/advanced-configuration/cache-reserve.mdx`) |
| Max cacheable file size             | 512 MB                            | `cache/concepts/default-cache-behavior.mdx`                                                                                                               |
| Max upload (request body)           | **100 MB**                        | `plans/index.json` network.max_upload_size — bounds create-post body (avg 32 KB, ~3000× headroom)                                                         |
| Min Edge Cache TTL                  | 2 hours                           | why `stale-while-revalidate` is not used (see [`caching.md`](./caching.md#http-cache-headers-in-detail))                                                  |
| Min Browser Cache TTL               | 2 minutes                         |                                                                                                                                                           |
| Cacheable HTTPS port                | **443 only**                      | every other proxied HTTPS port is cache-disabled; re-enabling is Enterprise-only (`fundamentals/reference/network-ports.mdx`)                             |
| Cache Rules                         | 10                                |                                                                                                                                                           |
| Purge (all 5 types incl. cache-tag) | yes                               | Free supports URL/Hostname/Tag/Prefix/Everything (`plans/index.json:454`)                                                                                 |
| Purge rate                          | 5/min, bucket 25, 100 ops/req     | per account                                                                                                                                               |
| Proxy Read Timeout (→ 524)          | **120 s**                         | not configurable on Free; only Enterprise can raise it (`fundamentals/reference/connection-limits.mdx`)                                                   |
| Tiered Cache (incl. Smart Topology) | free                              | `plans/index.json` cache.tiered_cache                                                                                                                     |
| ETag / Vary                         | yes                               |                                                                                                                                                           |
| Cache Reserve                       | paid add-on                       | not used                                                                                                                                                  |
| Cache Analytics                     | no                                |                                                                                                                                                           |
| Cache by status code                | Enterprise only                   |                                                                                                                                                           |
| Origin Cache Control                | on by default, **cannot disable** | so `s-maxage` etc. are strictly honored                                                                                                                   |

**The 100,000 requests/day limit does not apply to markpost.** That is a **Workers** (edge compute) quota (`workers/platform/limits.mdx`); markpost uses no Workers, only the static header-driven CDN proxy path, which has no such request cap. Even when a Worker exceeds 100k/day in fail-open mode, "requests behave as if no Worker is configured" — i.e. normal proxying to origin continues, confirming the proxy path itself is uncapped.

## Caddyfile selection by mode

The Caddyfile differs by mode because TLS handling differs:

**Homelab** — `docker/Caddyfile` (current repo baseline), TLS-less `:2053`, no `trusted_proxies`. Plaintext HTTP over LAN; no reverse-proxy chain so `ClientIP()` works directly.

**Self-hosted (with domain)** — a per-domain site block enables Caddy's automatic HTTPS (Let's Encrypt) and the HTTP→HTTPS redirect. Not yet templated in the repo; target shape:

```caddyfile
markpost.example.com {
    encode zstd gzip
    # Caddy automatically provisions a Let's Encrypt cert and redirects HTTP→HTTPS.

    handle /api/v1/* { reverse_proxy 127.0.0.1:7330 }
    handle /static/* { reverse_proxy 127.0.0.1:7330 }
    handle /swagger/* { reverse_proxy 127.0.0.1:7330 }
    handle /mpk-*    { reverse_proxy 127.0.0.1:7330 }
    handle /p-*      { reverse_proxy 127.0.0.1:7330 }
    handle           { reverse_proxy 127.0.0.1:3000 }

    log { output stderr format console }
}
```

Because Caddy only serves `Host: markpost.example.com`, a direct `http://<IP>:<port>` request does not match and is not routed to the app — this is how "IP:port not accessible" is enforced. The compose `ports` should expose 80 and 443 for ACME. This mode does not use Cloudflare, so `[cloudflare]` config is absent and the purger is a no-op.

**SaaS** — two Caddies. The host Caddy gateway binds 443 with the Origin CA certificate (`markpost.caddy.j2` → `/etc/caddy/conf.d/markpost.caddy`, a domain site block shared with the VPS's other services) and proxies to `127.0.0.1:8080`. The container Caddy (`devops/ansible/templates/Caddyfile.production.j2`) serves plain HTTP on `:2053`, published loopback-only, each `reverse_proxy` carrying `trusted_proxies private_ranges` plus the `header_up X-Forwarded-For {http.request.header.CF-Connecting-IP}` rewrite (see _Client IP detection_). It serves the static export with immutable `Cache-Control` on `/_next/static/*`, a 5-minute `max-age` on exported HTML, and the branded 404 page.

In all three modes the Go binary and config schema are identical — only the Caddyfile, DNS, and the optional `[cloudflare]` section differ.
