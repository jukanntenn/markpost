# AGENTS.md — e2e

Playwright end-to-end suite, chromium only, in a separate workspace with its own `package.json`. Repo-wide orders live in the [root AGENTS.md](../AGENTS.md); this file adds this tree's own.

- `pnpm test` (in `e2e/`) — run the suite locally
- `dagger call -m e2e all --source ..` — full e2e via dagger, from the repo root; the CI-fidelity path
