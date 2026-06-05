# E2E Testing Plan: Dagger + Playwright

## Decisions

| # | Decision | Choice | Rationale |
|---|---|---|---|
| 1 | SDK Language | TypeScript | Same language as Playwright tests; pnpm already in project |
| 2 | Orchestration | Pure Dagger containers | Self-contained, portable, no Docker Compose dependency |
| 3 | Frontend serving | Next.js standalone (`output: "standalone"`) | Already configured; matches production |
| 4 | Data isolation | Per-test-file dedicated Dagger sandbox | Each spec file gets its own postgres + backend + frontend containers |
| 5 | Auth strategy | API login in setup | Real JWT via `POST /api/v1/auth/login`; fast, exercises real auth |
| 6 | Tests to rewrite | login, dashboard, dashboard-create-post, posts, settings | Core user flows |
| 7 | Tests to drop | token-refresh.spec.ts | Requires short-lived token config; better as backend unit test |
| 8 | Browser matrix | Chromium only | Fastest; add Firefox/WebKit later as needed |
| 9 | Timeouts | 60s health wait / 30s per-test / 10min overall | Generous for cold starts, fast-fail on hangs |
| 10 | Directory structure | Official Dagger TS module layout | `dagger.json` + `src/index.ts` + `sdk/`; tests and lib alongside |

## Directory Structure

```
e2e/
├── dagger.json                  # name: "markpost-e2e", source: "src"
├── package.json                 # @playwright/test
├── src/
│   ├── src/
│   │   └── index.ts             # MarkpostE2E class: test(), all()
│   └── sdk/                     # Bundled Dagger SDK (auto-generated)
├── tests/                       # Playwright specs (1 file = 1 scenario)
│   ├── login.spec.ts            # 8 tests
│   ├── dashboard.spec.ts        # 4 tests
│   ├── dashboard-create-post.spec.ts  # 2 tests
│   ├── posts.spec.ts            # 2 tests
│   ├── settings.spec.ts         # 5 tests
│   └── settings-change-password.spec.ts  # 1 test (isolated: mutates password)
├── lib/                         # Playwright support code
│   ├── fixtures.ts              # Auth fixtures, test helpers
│   ├── helpers.ts               # HTTP health-check, API login
│   └── pages/                   # Page objects
│       ├── LoginPage.ts
│       ├── DashboardPage.ts
│       ├── PostsPage.ts
│       └── SettingsPage.ts
└── playwright.config.ts
```

## Dagger Pipeline Design

**`test(testFile: string, source: Directory): string`**
- Builds Go backend binary from `source.directory("backend")`
- Builds Next.js standalone from `source.directory("frontend")` with `BACKEND_URL=http://backend:7330`
- Starts 4 containers per sandbox:
  1. `postgres` — PostgreSQL 17 with markpost db
  2. `backend` — Go binary, bound to postgres via alias `postgres`
  3. `frontend` — Next.js standalone, bound to backend via alias `backend`
  4. `runner` — Playwright (Chromium), bound to frontend via alias `frontend`
- Runs `pnpm exec playwright test <file>` in runner
- Returns stdout

**`all(source: Directory): string`**
- Globs `tests/*.spec.ts`
- Runs `test()` for each sequentially (avoiding resource contention and state leakage)
- Aggregates results

## TODO / Next Steps

- [x] Check environment (dagger CLI, Go, Node, pnpm)
- [x] Initialize Dagger module (`dagger init` + `dagger develop`)
- [x] Implement `src/index.ts` pipeline
- [x] Create Playwright config, fixtures, helpers, page objects
- [x] Write test files
- [x] Delete old `frontend/tests/`
- [x] Write Dagger wiki doc
- [x] Verify all test files pass via `dagger call all --source=..`

## Test Results

| File | Tests | Status |
|------|-------|--------|
| login.spec.ts | 8 | ✅ All pass |
| dashboard.spec.ts | 4 | ✅ All pass |
| dashboard-create-post.spec.ts | 2 | ✅ All pass |
| posts.spec.ts | 2 | ✅ All pass |
| settings.spec.ts | 6 | ✅ All pass |
| settings-change-password.spec.ts | 1 | ✅ All pass |

**Total: 23/23 tests passing**

## Performance Analysis

### Time Breakdown (cached build, sequential execution)

| Phase | Duration | Notes |
|-------|----------|-------|
| Dagger connect | ~1s | Connect to Dagger engine |
| Load workspace | ~1s | Load module definition |
| Compile pipeline | ~40s | Evaluate module, resolve container graph |
| Test execution (6 files) | ~106s | Includes container startup/shutdown per file |
| — Playwright tests only | ~39s | Actual test runtime |
| — Container overhead | ~67s | ~11s per file (postgres + backend + frontend + playwright startup) |
| **Total wall time** | **~2m28s** | |

### Per-File Breakdown

| Test File | Playwright Time | Tests |
|-----------|----------------|-------|
| dashboard-create-post | 11.8s | 2 |
| dashboard | 7.5s | 4 |
| login | 6.3s | 8 |
| posts | 2.5s | 2 |
| settings-change-password | 3.7s | 1 |
| settings | 7.5s | 6 |

### Resource Consumption

| Resource | Value |
|----------|-------|
| Container images pulled | ~5GB total (golang:1.26-alpine 721MB, node:24-alpine 230MB, postgres:17-alpine 399MB, playwright 3.61GB) |
| Dagger engine memory | ~4.7GB (resident) |
| Dagger engine cache | ~58GB (accumulated; includes all build layers) |
| Containers per test file | 4 (postgres, backend, frontend, playwright) |
| Max concurrent containers | 4 (sequential execution) |
| Build cache (Go modules) | Persistent across runs via `dag.cacheVolume` |
| Build cache (pnpm store) | Persistent across runs via `dag.cacheVolume` |

### Bottlenecks

1. **Pipeline compilation (~40s)**: Dagger evaluates the entire module code. Acceptable for CI, noticeable in dev loop.
2. **Per-file container startup (~11s)**: Each test file starts 4 containers from scratch. The `RUN_ID` env var prevents state leakage but also prevents reuse.
3. **First-run build time**: Backend (Go build) and frontend (Next.js build) take ~2-5min on cold start. Subsequent runs use Dagger's layer cache.

### Optimization Opportunities

- **Parallel execution**: `Promise.all` instead of sequential loop reduces wall time but requires more resources (24 containers simultaneously). Consider for CI with sufficient memory.
- **Shared backend/frontend builds**: Currently cached within a Dagger session. Could be further optimized by pre-building and reusing across sessions.
- **Warm health check**: `waitForBackend` polls at 2s intervals with 60s timeout. Fastest resolution is ~2-4s after backend starts.

1. **Next.js route announcer**: `role="alert"` matches Next.js's built-in `__next-route-announcer__` element. Page objects now use `[data-slot='alert']` (shadcn Alert) instead of `getByRole("alert")`.
2. **Bootstrap → shadcn/ui selectors**: Replaced `.font-monospace.fs-5` → `.font-mono`, `.alert.alert-danger` → `[data-slot='alert']`, `.dropdown-toggle` → `getByRole("button").filter({ hasText: /markpost/ })`.
3. **Input selectors**: Changed from `name` attributes (Bootstrap) to `id` attributes (shadcn: `#current-password`, `#new-password`, `#confirm-password`).
4. **Button names**: "Change Password" → "Save" (i18n: `common.save`), "Quickly create..." → "Create Test Post".
5. **Copy flow**: Copy button now opens a Menu popup; `clickCopyKey()` clicks trigger then "Copy Post Key" menu item.
6. **Language switch**: Native `<select>` dropdown instead of toggle button; uses `selectOption("zh")`.
7. **Validation errors**: Shown in `FormAlert` (`[data-slot='alert']`), not inline `.invalid-feedback`.
8. **Post list**: PostListItemRow shows only title; test provides explicit title instead of relying on body text.
9. **Password test isolation**: Moved "successfully changes password" to its own file (`settings-change-password.spec.ts`) since it mutates the admin password.
10. **Key visibility toggle**: After hiding, `.font-mono` div persists with masked text; test now checks actual key value is not visible.

## Deferred (Future Work)

- OAuth2 mock server for GitHub login E2E tests
- Feishu webhook mock server for delivery channel E2E tests
- Firefox / WebKit browser support
- CI integration (GitHub Actions)
