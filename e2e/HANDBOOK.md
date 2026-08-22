# E2E Testing Handbook

English | [中文](HANDBOOK.zh.md)

> A quick-start guide for testers, covering architecture, execution, debugging, and common problems.

---

## 1. Architecture overview

```
┌──────────────┐     ┌──────────────────────┐     ┌──────────────┐
│   Playwright │────▶│   App Container      │────▶│  PostgreSQL  │
│ test runner  │     │  Caddy (HTTPS:2053)  │     │   (5432)     │
│  (host)      │     │  + Go backend (7330) │     │              │
└──────────────┘     │  + frontend static   │     └──────────────┘
                     └──────────────────────┘
                            │
                            ├─────────────────┐
                            ▼                 ▼
                     ┌──────────────┐  ┌──────────────┐
                     │ Webhook Mock │  │  OAuth Mock  │
                     │  (3002)      │  │  (3001)      │
                     └──────────────┘  └──────────────┘
```

### Core components

| Component         | Role                                                               | Port         |
| ----------------- | ------------------------------------------------------------------ | ------------ |
| **App Container** | Caddy reverse proxy + Go backend + frontend static files           | 2053 (HTTPS) |
| **PostgreSQL**    | Test database; a separate database per spec file                   | 5432 (inner) |
| **Webhook Mock**  | Simulates the Feishu webhook receiver                              | 3002 (inner) |
| **OAuth Mock**    | Simulates the full GitHub OAuth flow (authorize, token, user info) | 3001 (inner) |
| **Playwright**    | Browser automation test runner (host)                              | -            |

### Key design decisions

1. **Single entry point, HTTPS 2053**: matches production; Caddy uses a `tls internal` self-signed certificate and never exposes HTTP 8080
2. **Data isolation**: every spec file uses its own database name `markpost_{runId}`
3. **Test data cleanup**: each test cleans up through the API before and after, keeping tests independent
4. **One Dockerfile/Caddyfile**: fast-feedback tests, production, and the full Dagger runs share the same `docker/Dockerfile` and `docker/Caddyfile`

---

## 2. Quick start

### Prerequisites

- Docker installed and running
- Run commands from the repo root

### Option 1: Docker Compose fast feedback (recommended while developing)

```bash
# 1. Start the services (first run builds the image)
docker compose -f e2e/docker-compose.yml up -d --build

# 2. Wait for readiness (~30s)
curl -k https://localhost:2053/api/v1/health
# expect: {"status":"ok"}

# 3. Run all tests (from the host)
cd e2e && pnpm test

# 4. Run a single spec file
cd e2e && npx playwright test tests/login.spec.ts --reporter=list

# 5. Stop the services and wipe data
docker compose -f e2e/docker-compose.yml down -v
```

### Option 2: Dagger (used in CI)

```bash
# Run all tests (builds the image, starts services, isolates databases)
cd e2e
dagger call all --source=..

# Run a single spec file
dagger call test --source=.. --test-file=login.spec.ts
```

---

## 3. Test coverage

### Spec file inventory

| Spec file                           | Covered behavior                                                                                                                                    |
| ----------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `login.spec.ts`                     | Login form, validation, error messages, keyboard submit, redirect                                                                                   |
| `landing.spec.ts`                   | Landing smoke: structure present, CTA switches with session state (/login ↔ /dashboard)                                                             |
| `dashboard.spec.ts`                 | Post Key show/hide/copy, user menu, logout                                                                                                          |
| `dashboard-create-post.spec.ts`     | Quick post creation, form validation                                                                                                                |
| `posts.spec.ts`                     | Post list page, unauthenticated redirect                                                                                                            |
| `admin.spec.ts`                     | Admin page permissions, navigation links                                                                                                            |
| `admin-users.spec.ts`               | User list, admin display                                                                                                                            |
| `admin-posts.spec.ts`               | Post management, search                                                                                                                             |
| `admin-channels.spec.ts`            | Channel list, channel creation                                                                                                                      |
| `admin-delivery-history.spec.ts`    | Delivery history empty state                                                                                                                        |
| `settings.spec.ts`                  | Settings page rendering, language switch, password validation                                                                                       |
| `settings-change-password.spec.ts`  | Change password and verify login                                                                                                                    |
| `settings-delivery-channel.spec.ts` | Channel CRUD, enable/disable toggle                                                                                                                 |
| `delivery-history.spec.ts`          | Delivery history region display                                                                                                                     |
| `oauth-callback.spec.ts`            | Full OAuth flow: successful login, missing params, bad params, invalid state, token-exchange failure, user-info failure, one-time state consumption |
| `feishu-webhook.spec.ts`            | Webhook trigger and payload verification                                                                                                            |

---

## 4. Project structure

```
e2e/
├── docker-compose.yml             # fast-feedback Docker Compose config
├── tests/                         # spec files
├── lib/
│   ├── fixtures.ts                # Playwright fixtures (auth, page objects)
│   ├── helpers.ts                 # API helpers (login, seed data, cleanup)
│   └── pages/                     # page object models
├── mock-services/
│   ├── oauth-mock/                # GitHub OAuth mock (oauth2-mock-server)
│   │   ├── index.ts
│   │   ├── Dockerfile
│   │   └── package.json
│   └── webhook-mock/              # Feishu webhook mock service
│       ├── index.ts
│       ├── Dockerfile
│       └── package.json
├── src/                           # Dagger module
│   └── src/index.ts
├── playwright.config.ts           # Playwright config
├── package.json
├── HANDBOOK.md                    # this handbook (English)
└── HANDBOOK.zh.md                 # this handbook (Chinese)
```

---

## 5. Page objects

All page objects live in `e2e/lib/pages/`, wrapping page interaction logic.

---

## 6. Test data management

### Data cleanup

Every spec calls `cleanupTestData()` in `beforeEach` and `afterEach`:

```typescript
import { test, expect, cleanupTestData } from "../lib/fixtures";

test.beforeEach(async ({ request, authToken }) => {
  await cleanupTestData(request, authToken.token);
});

test.afterEach(async ({ request, authToken }) => {
  await cleanupTestData(request, authToken.token);
});
```

`cleanupTestData` will:

1. Delete all posts
2. Delete all delivery channels
3. Clear the Webhook Mock received records
4. Clear the OAuth Mock request records

---

## 7. Environment variables

| Variable                       | Default                  | Purpose                          |
| ------------------------------ | ------------------------ | -------------------------------- |
| `BASE_URL`                     | `https://localhost:2053` | Frontend address                 |
| `BACKEND_URL`                  | `https://localhost:2053` | Backend API address              |
| `ADMIN_USERNAME`               | `markpost`               | Admin username                   |
| `ADMIN_PASSWORD`               | `markpost`               | Admin password                   |
| `WEBHOOK_MOCK_URL`             | `http://localhost:3002`  | Webhook Mock address             |
| `OAUTH_MOCK_URL`               | `http://localhost:3001`  | OAuth Mock address               |
| `NODE_TLS_REJECT_UNAUTHORIZED` | -                        | Set `0` to skip TLS verification |

---

## 8. FAQ and solutions

### Q1: TLS handshake failure `SSL routines:ssl3_read_bytes:tlsv1 alert internal error`

**Cause**: the SNI of Caddy's self-signed certificate does not match the requested hostname.

**Fix**:

- Docker Compose: set `NODE_TLS_REJECT_UNAUTHORIZED=0`
- Dagger: the module already sets this variable

### Q2: CORS configuration panics the backend

**Cause**: Viper cannot parse a JSON-array CORS origins value from environment variables.

**Fix**: configure CORS in `config.toml`; never pass array values through environment variables.

### Q3: Data pollution between tests

**Cause**: data created by an earlier test affects later tests.

**Fix**:

- Call `cleanupTestData()` in each spec's `beforeEach`
- Password-change tests must clean up both before the change and after the reset
- The API response shape is `data.items` (not `data.channels`)

### Q4: Webhook tests never receive the callback

**Cause**: the Webhook Mock service is not bound to the app container.

**Fix**: in the Dagger module, `appService` needs `with_serviceBinding("webhook-mock", webhookMock)`.

### Q5: OAuth tests fail

**Cause**: the OAuth Mock is not running, or the backend is not pointed at it.

**Fix**:

- Make sure the `oauth-mock` service in `e2e/docker-compose.yml` is healthy
- Make sure the backend environment includes `MARKPOST_OAUTH__GITHUB__AUTH_URL`, `TOKEN_URL`, `USER_URL`

### Q6: The first Dagger run is slow

**Cause**: the first run downloads all dependencies and builds the image.

**Fix**: use Docker Compose for fast feedback while developing; move to Dagger once things pass.

---

## 9. Debugging tips

### View failure screenshots

Failed tests write screenshots to `e2e/test-results/` automatically.

### View backend logs

```bash
# Docker Compose
docker compose -f e2e/docker-compose.yml logs app --tail=50

# View GIN request logs
docker compose -f e2e/docker-compose.yml exec app cat /tmp/markpost.log
```

### Run a single case

```bash
cd e2e && npx playwright test tests/login.spec.ts -g "logs in with valid" --reporter=list
```

---

## 10. CI integration

### GitHub Actions configuration

```yaml
name: E2E Tests
on: [push, pull_request]
jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install Dagger
        run: curl -fsSL https://dl.dagger.io/dagger/install.sh | sh
      - name: Run E2E tests
        run: cd e2e && dagger call all --source=..
```

---

## 11. Notes

1. **Single entry point, HTTPS 2053**: HTTP 8080 is never exposed, matching production
2. **Docker Compose mounts no data volumes**: test data is ephemeral and cleaned up when containers stop
3. **Tests must run serially**: `workers: 1` guarantees data isolation
4. **Every test cleans its data**: avoids cross-test pollution
5. **One config**: fast-feedback tests, production, and full Dagger runs share `docker/Dockerfile` and `docker/Caddyfile`; no separate E2E config is maintained

---

_Handbook version: 2026-07-16_
