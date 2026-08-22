# AGENTS.md — frontend

Next.js 16 App Router with `output: "export"` static export, React 19, TypeScript, Tailwind CSS 4. Repo-wide orders live in the [root AGENTS.md](../AGENTS.md); this file adds this tree's own.

## Commands (run in `frontend/`)

- `pnpm dev` — dev server (port 3034; `/api/v1` proxied to the backend via `rewrites` in `next.config.ts`)
- `pnpm build` — production build (static export to `out/`)
- `pnpm lint` — ESLint
- `pnpm format` / `pnpm format:check` — Prettier write / check
- `pnpm test` — Vitest watch; `pnpm test:run` — run once (jsdom + v8 coverage)

## Layout

```
src/app/           App Router ((auth), (dashboard): admin/dashboard/posts/settings)
src/components/    ui/ (shadcn-style), auth/, layout/, dashboard/, admin/, posts/
src/lib/           utils.ts, api/ fetchers
src/i18n/          next-intl + locales (en, zh)
src/stores/        Zustand
```

## Style and boundaries

- Prettier owns formatting (`.prettierrc.json`); eslint-config-next owns correctness.
- Function components + hooks only, never class components; PascalCase component files (`PostList.tsx`).
- The app is a static export: never add `middleware.ts`/`proxy.ts` — dev `/api` proxying is `rewrites` in `next.config.ts`, prod is Caddy.
- Never edit lock files (`pnpm-lock.yaml`).
