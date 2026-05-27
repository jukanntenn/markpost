# Three-Tier Testing Strategy

## Tier 1: Unit Tests

### Backend

- Real SQLite in-memory database for all tests
- Interface-level mocks only for hard-to-control scenarios (e.g., injecting DB errors)
- `net/http/httptest` for external HTTP services (email, cloud APIs)
- Rewrite all existing mock-repo service tests to real DB, batched by domain
- Repository tests for edge cases (constraints, pagination)
- Handler tests using existing `testutil` engine

### Frontend

- Strengthen existing unit/component tests
- Ensure MSW handlers cover full API contract
- Add missing component tests

## Tier 2: Integration Tests

- Go test binary orchestrates: real backend (SQLite) + `httptest.Server` (external services) + Playwright
- Frontend dev server started as subprocess
- Migrate existing E2E tests from mocked API routes to real backend
- Run on PR, merge gate

## Tier 3: Agent-Driven Tests

### SCN Format

Hybrid — API steps for setup/teardown, browser steps for user-visible behavior.

```
SCN-001: Admin publishes a post
- Backend healthy (GET /health -> 200)
- Admin authenticated (API token, role=ADMIN)

1. Create test post via API.
   POST /api/v1/posts {title: "Test", body: "Content"} -> 201
2. Navigate to posts page.
   BROWSER: goto /posts
3. Verify post appears in list.
   BROWSER: see "Test" in post list
4. Click publish.
   BROWSER: click "Publish" on post "Test"
5. Verify status changed.
   BROWSER: see "Published" badge
6. Verify via API.
   GET /api/v1/posts/1 -> 200, status=published
```

### Execution

- Main Claude agent executes SCNs via `/playwright-cli` against `devops/dev.py` environment
- No subagent pipeline — human-in-the-loop orchestration
- Human reviews SCNs, invokes agent to execute

### Authentication

- Pre-seeded test users and tokens in Docker setup
- API steps and most browser flows use injected tokens
- UI login only for SCNs that specifically test authentication behavior

### Mocking

- Almost no mocks — all services are real
- Only mock: OAuth2 and truly hard-to-run third-party services

### CI

- On-demand + nightly, not a merge gate
- Flakiness: accepted, human triages failures

## Artifact Locations

| Artifact | Location |
|---|---|
| Domain aggregates | `specs/aggregates/` |
| Structured Scenarios (SCNs) | `tests/e2e/scenarios/` |
| Agent tooling | `.claude/skills/`, `.claude/commands/` |

## Phasing

1. **Phase 1a** — Tier 1 backend: rewrite existing mock-based tests to real DB per domain, build shared test helpers
2. **Phase 1b** — Tier 1 frontend: strengthen existing unit/component tests
3. **Phase 2** — Tier 2: Go integration harness, migrate E2E tests to real backend
4. **Phase 3** — Tier 3: SCN format, first scenarios, agent execution against dev environment
