# Config Review — Issues & Improvements

## Issues

### 1. JWT signing keys: empty default with no validation

**Location:** `internal/config/config.go:208-209`

`access_signing_key` and `refresh_signing_key` default to `""`. No validation tag on either field. The application will start and sign tokens with an empty HMAC key, which is a silent security vulnerability.

**Decision:** Empty string should fail validation at startup, forcing users to set both keys. ([ref](https://github.com/jukanntenn/markpost/issues TBD))

### 2. Admin credentials: hardcoded defaults pass `required` validation

**Location:** `internal/config/config.go:189-190`

`initial_username` and `initial_password` default to `"markpost"` and are tagged `required`. Since the defaults are non-empty, `required` never triggers. Users deploying to production can easily miss this.

**Decision:** Keep defaults for dev convenience. Document that production deployments must override. No code change needed.

### 3. Rate limit: defaults to unlimited

**Location:** `internal/config/config.go:212-213`

`per_second` and `burst` default to `math.MaxInt`. An unconfigured instance has zero rate limiting.

**Decision:** Acceptable. Example TOML suggests `5`/`50`. Document clearly.

### 4. `server.frontend_url` port mismatch with dev server

**Location:** `internal/config/config.go:186` vs `frontend/package.json`

Backend defaults to `http://127.0.0.1:3000` (Next.js standard), but `FRONTEND_PORT` defaults to `3034`. Out-of-box dev setup requires either changing one or the other.

**Decision:** Intentional — `3000` is the canonical Next.js port. Users running the dev environment already have `deploy/dev/.env.example` that sets both ports correctly. No code change needed.

### 5. `server.public_url` fallback generates invalid URL with `0.0.0.0`

**Location:** `internal/config/config.go:183-185`

When `public_url` is empty and `host` is `0.0.0.0`, the fallback `http://0.0.0.0:{port}` is not a valid public URL. Deployments behind a reverse proxy or container will produce broken links.

**Decision:** Document that `public_url` must be set when binding to `0.0.0.0` or deploying behind a proxy. No code change.

### 6. OAuth GitHub: partial configuration not validated

**Location:** `internal/config/config.go:205-207`

All three OAuth fields default to empty with no cross-field validation. Setting `client_id` without `client_secret` will cause runtime errors from GitHub's API rather than a clear startup failure.

**Decision:** Acceptable. Let GitHub API errors surface the problem. No code change.

### 7. Frontend: dual API routing path ambiguity

**Location:** `frontend/next.config.ts` (rewrites) vs `frontend/src/lib/api/base.ts` (`NEXT_PUBLIC_API_URL`)

Two mechanisms exist for routing API requests: Next.js rewrites (server-side proxy) and `NEXT_PUBLIC_API_URL` (client-side direct). When both are present or both absent, the behavior is unclear.

**Decision:** Skipped during review. Needs a separate frontend config audit.

## Improvements (not blocking)

- **Config file discovery:** Currently searches `.` for `markpost.toml`. Could also search `/etc/markpost/`, `~/.config/markpost/` for better deployment ergonomics.
- **Config validation errors:** `go-playground/validator` errors are verbose. Could wrap them into human-readable messages pointing to the specific field and expected format.
- **Hot reload:** Config is loaded once via `sync.Once`. Adding SIGHUP reload would allow runtime changes without restart.
- **Schema validation for TOML:** Consider adding a `config.schema.toml` or JSON Schema for editor autocompletion.
