# Development Configuration Specification

Rules and mechanisms for achieving **zero-config or minimal-config** local
development environments. Any project following this spec reproduces the same
layered configuration mechanism with project-specific values.

---

## 1. Principles

**Single source of truth.** Every configuration value has exactly one primary
source. All other occurrences are explicit references, never independent
definitions. Changing a value must require editing the fewest possible files.

**Code defaults first.** Non-sensitive defaults are defined in application code
via framework-native mechanisms (Viper `SetDefault`, Next.js config constants,
module-level constants). Every other layer derives from or overrides them.

**Layered override.** Higher layers override lower:

```
Environment variables  >  Config files  >  Code defaults
```

A project must start using **only** code defaults, except for
security-sensitive values (see §2).

**Zero-config goal.** A fresh clone must start the full dev stack with no
manually created config files for at least one launch mechanism (IDE task
runner, dev script, or direct command).

---

## 2. Security-Sensitive Values

Values that could cause harm if accidentally used in production must **not**
have code defaults. Missing values cause startup failure — this is intentional.

Applies to: signing keys, encryption keys, production database passwords.

These must be provided through exactly one of:

1. A local config file (gitignored, created from a template by the developer)
2. Environment variables injected by an IDE task runner (`options.env`)
3. Environment variables in a Docker Compose file

No other mechanism may supply these values. The example config file documents
them as required.

---

## 3. Configuration Files

- **Example/template** config files are git-tracked and document every option
  with its default and environment variable override.
- **Actual** config files are gitignored and created by developers from
  templates.
- **No runtime generation.** No script or tool may create config files inside
  the source tree. Developers create them manually.

---

## 4. Environment Files

**Frontend:** Local dev requires zero env files. All defaults live in framework
config code (e.g. proxy target in `next.config.ts`). `.env.local` may be used
for overrides but must never be required. `.env` is reserved for production —
do not introduce git-tracked `.env` files in the frontend.

**DevOps:** No `.env` file. Development Docker Compose and dev scripts contain
all values directly — inline in YAML, as constants in code.

---

## 5. Docker Compose (Development)

Hardcode all values inline. No `${VAR:-default}` substitution, no external
`.env` reference. Only set values that **differ** from code defaults:

```yaml
environment:
  - SERVER_HOST=0.0.0.0          # Yes — differs from code default (127.0.0.1)
  - DB_DRIVER=postgresql          # Yes — differs from code default (sqlite)
  - JWT_SIGNING_KEY=dev-secret    # Yes — no code default (security-sensitive)
  # SERVER_PORT — omitted, same as code default
  # ADMIN_USERNAME — omitted, same as code default
```

Resource naming (containers, volumes, networks) uses a fixed project
identifier with no variable substitution.

---

## 6. Development Script

Module-level constants for ports and URLs, matching code defaults. No `.env`
file loading. Frontend subprocess inherits parent environment — no injection
needed since the framework config provides all defaults.

```python
BACKEND_PORT = 7330   # matches backend code default
FRONTEND_PORT = 3000  # matches frontend package.json default
```

---

## 7. IDE / Tooling

IDE task runners inject environment variables via their native mechanism
(e.g. VS Code `options.env`). Only inject security-sensitive values that
cannot have code defaults. Do not inject values that already have defaults.

At least one IDE launch configuration must start the full stack with **no
prerequisites** — combining code defaults with IDE-injected secrets.

---

## 8. Frontend Proxy

The rewrite/proxy config must include a **code-level default** for the backend
address, matching the backend's default port:

```ts
async rewrites() {
  const target = process.env.API_PROXY_TARGET || "http://127.0.0.1:7330";
  return [
    { source: "/api/:path*", destination: `${target}/api/:path*` },
  ];
}
```

Production overrides `API_PROXY_TARGET` via environment variable (Docker
internal DNS, reverse proxy, etc.). The code default only applies when unset.

---

## 9. Anti-Patterns

- **Duplicating defaults** across `.env`, compose, code, and scripts — define
  once in code, reference everywhere else.
- **`${VAR:-default}` in development compose** — hardcode inline instead.
- **Generating config files at runtime** — developers create them manually.
- **Code defaults for signing keys** — require explicit configuration.
- **Git-tracked `.env` in frontend** — use framework config defaults for dev.
- **Dev script loading `.env`** — use constants matching code defaults.

---

## 10. Checklist

- [ ] Backend starts with only a config file for security-sensitive values
- [ ] Frontend starts with zero env files; proxy works via code default
- [ ] IDE "Start All" launches full stack with no prerequisites
- [ ] Dev script launches Docker stack with no external `.env`
- [ ] Changing backend port requires at most two file edits
- [ ] No security-sensitive value has a code default
- [ ] Development compose has no `${VAR:-default}` syntax
- [ ] Dev script has no `.env` loading logic
- [ ] Actual config files gitignored; templates git-tracked
