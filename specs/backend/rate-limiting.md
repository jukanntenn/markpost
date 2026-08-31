# Rate Limiting

English | [中文](rate-limiting.zh.md)

Request throttling for the public read path, the API-authenticated write path, and the login endpoints: four independent token-bucket limiters built on tollbooth v8, each scoped to a route class and keyed on the dimension that actually identifies the actor. The wiring lives in `cmd/server/main.go` (`SetupRoutes`) and `internal/middleware/rate_limit.go`; the configuration in the `[ratelimit]` TOML section. The decision record (why four limiters, why these key dimensions, what was rejected) lives in [the performance-pass MRFC](../../.agents/mrfcs/implemented/2026-07-09-read-path-performance-pass.md).

## The four limiters

| Limiter               | Routes                                                                                                                                                                               | Key       | Rate (default)                                                                        |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------- | ------------------------------------------------------------------------------------- |
| **read** (L1)         | `GET /:qid` (public post render)                                                                                                                                                     | client IP | 100/s, burst 200                                                                      |
| **public_write** (L2) | `POST /:post_key`                                                                                                                                                                    | `user_id` | 10/min (0.1667/s), burst 20; **plus** a daily cap of 1000/day (0.01157/s, burst 1000) |
| **authed_write** (L3) | JWT group write routes (`POST /auth/logout`, `POST /auth/change-password`, `POST /post-key/rotate`, delivery channel writes, `DELETE /posts/:id`, session revocations, admin writes) | `user_id` | 30/min (0.5/s), burst 60                                                              |
| **login**             | `POST /auth/login`, `POST /oauth/login`                                                                                                                                              | client IP | 5/min (0.0833/s), burst 5                                                             |

- **L1** is generous because the CDN absorbs the vast majority of reads; it only governs the small fraction that revalidates against the origin. IP is the only identifier available on the public read path.
- **L2** keys on `user_id`, resolved by the `PostKey` middleware that validates the per-user credential before the limiter runs. Keying on `user_id` rather than the raw `post_key` means rotating a post_key cannot evade the limit, and unifies the dimension with L3. The 10/min and 1000/day caps are the business hard limits.
- **L3** keys on `user_id` from the JWT (`AuthWithBlacklist`). Reads (GET) stay outside the limiter so listing does not consume the write budget.
- **login** is a dedicated per-IP limiter for credential endpoints — login attempts have no authenticated identity to key on, and the tight 5/min default exists to blunt credential-stuffing.

**Daily cap implementation (L2).** Tollbooth's token bucket has a fixed 1-second window, so the daily cap is expressed as `rate.Limit(1000.0/86400)` with burst 1000 — mathematically "1000 per day, spendable in a burst". The trade-off: a user who spends all 1000 tokens at midnight UTC waits ~86 seconds per additional token, acceptable for a low-frequency authoring operation, and it avoids a second date-keyed counter data structure.

**429 responses carry `Retry-After`** (computed from the bucket's refill time) and a custom i18n JSON body via `apierr.RespondError`; tollbooth's own response-writing path is bypassed. `RateLimit-Limit` / `RateLimit-Reset` / `RateLimit-Remaining` headers are not set — the CORS expose list retains them for possible future use.

**Anonymous clients.** If `c.ClientIP()` returns empty (unresolvable), IP-keyed limiters return `429` immediately rather than collapsing all anonymous clients into a shared `"unknown"` bucket — one anonymous attacker must not be able to exhaust the limit for everyone else. `429` over `400` because the semantic is "you are being rate-limited" (no identity → no quota), not "malformed request".

**Exemptions.** `GET /api/v1/health`, `GET /api/v1/ready`, and `GET /api/v1/version` are registered outside every limiter group; Docker healthchecks hit health on a loopback timer and external uptime monitors poll ready, and subjecting them to L1 would cause false-positive health failures under load.

## IP resolution: gin, not tollbooth

The middleware calls `c.ClientIP()` (which applies the trusted-proxy logic below) and passes the result to `tollbooth.LimitByKeys`. Tollbooth's own `SetIPLookup` performs no trusted-proxy validation, so delegating IP resolution to it would reintroduce the spoofing risk that the trusted-proxy configuration closes.

All traffic in the SaaS topology flows one path: `Client → Cloudflare → host Caddy gateway → container Caddy → Go`. Cloudflare sets `CF-Connecting-IP` (the real client IP) and its own `X-Forwarded-For` chain on origin pull; the origin firewall is locked to Cloudflare's CIDRs on 443, and the container port is published loopback-only (see [`cloudflare.md`](./cloudflare.md) _Origin protection_). Client-IP recovery is a single-value relay resting on aligned trust anchors — Cloudflare's edge assertion of the header, the host firewall's CIDR allowlist, and the loopback-only publish:

**Caddy layer.** `CF-Connecting-IP` passes through the host gateway untouched (not a hop-by-hop header), and every `reverse_proxy` in the container's `Caddyfile.production.j2` carries `header_up X-Forwarded-For {http.request.header.CF-Connecting-IP}`. Caddy applies user header operations after its default forwarded-header handling, so the `X-Forwarded-For` delivered to Go is always the single `CF-Connecting-IP` value: Cloudflare overwrites any visitor-supplied copy of that header at the edge, and only the host gateway can reach the container port, so on every legitimate request the value is Cloudflare-asserted. `trusted_proxies` is set to `private_ranges` on the same blocks (the gateway's NAT'd bridge address is the peer), keeping default forwarded-header handling consistent; the XFF value itself is fixed by the `header_up` rewrite.

**gin layer.** `SetTrustedProxies(["127.0.0.1", "::1"])` reflects that Caddy proxies to Go over loopback. `ClientIP()` trusts the loopback peer and returns the single-value `X-Forwarded-For`. A plain appended chain would break here: gin walks the chain right-to-left and returns the first untrusted IP, and with loopback-only trust the rightmost entry — Cloudflare's edge hop — would be returned for every visitor, collapsing the IP-keyed limiters onto a handful of edge addresses.

`gin.PlatformCloudflare` (trusting `CF-Connecting-IP` unconditionally) is deliberately not used at the application layer: it performs no CIDR check, so an attacker reaching the port directly could forge the header and evade rate limiting. The deployed design anchors the header's authenticity on Cloudflare's edge overwrite plus the host firewall — a firewall bypass is the residual threat, and the firewall is the enforcement point.

**Cloudflare CIDR maintenance.** The CIDR list is operator-supplied: `cloudflare_cidrs` in `devops/ansible/group_vars/production/vars.yml` is the list's documented home, consumed by the host firewall allowlist on 443 (the only enforcement point — no template consumes it since `trusted_proxies` moved to `private_ranges`). Cloudflare occasionally updates its published ranges (https://www.cloudflare.com/ips/); operators must resync the firewall — an explicit operational responsibility documented in [`cloudflare.md`](./cloudflare.md).

## Configuration

```toml
[ratelimit.read]          # per_second = 100, burst = 200
[ratelimit.public_write]  # per_second = 0.1666666667, burst = 20,
                          # daily_per_second = 0.0115740741, daily_burst = 1000
[ratelimit.authed_write]  # per_second = 0.5, burst = 60
[ratelimit.login]         # per_second = 0.0833333333, burst = 5
```

All values are operator-tunable with the defaults above (env-mapped via `MARKPOST_RATELIMIT__*`); `internal/middleware/rate_limit_fuzz_test.go` fuzzes the key construction, and `rate_limit_test.go` covers limiter isolation (L1 does not count against L2), the anonymous-429 path, and the health exemption.
