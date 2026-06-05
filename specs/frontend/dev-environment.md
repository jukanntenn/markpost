# Frontend Development Environment

## Prerequisites

- **Node.js 24+** — Required for the project
- **pnpm** — Package manager (npm and yarn are not used)

## Setup

```bash
cd frontend
pnpm install
```

## Dev Server

```bash
pnpm dev
```

Starts the Next.js dev server on port **3034**. Hot module replacement is enabled by default.

## Production Build

```bash
pnpm build
```

Creates an optimized production build.

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `BACKEND_URL` | Backend address for server-side proxy | `http://127.0.0.1:7330` |

The frontend ships a committed `.env` with `BACKEND_URL=http://127.0.0.1:7330`. The proxy (`src/proxy.ts`) uses this to forward `/api/*` requests to the backend. To override, create `.env.local` (gitignored).

## Testing

### Unit Tests

```bash
pnpm test          # Run tests in watch mode
pnpm test:run      # Run tests once (CI mode)
```

Tests use Vitest with jsdom environment and V8 coverage.

### E2E Tests

```bash
pnpm test:e2e
```

Playwright tests run across Chromium, Firefox, and WebKit. Test files are in `tests/`.

## Linting

```bash
pnpm lint
```

Uses ESLint with Next.js and React rules.

## Key Files

- `src/test/setup.ts` — Vitest setup, starts MSW server
- `src/mocks/handlers.ts` — MSW request handlers for API mocking
- `src/mocks/server.ts` — MSW server setup (uses `setupServer` for Node environment)
- `vitest.config.ts` — Vitest configuration
- `playwright.config.ts` — Playwright configuration
