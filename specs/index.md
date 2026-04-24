# Specification Files

> Reference the most relevant spec file(s) for your current task.

## Tech Stack

- **Backend**: Go 1.24, Gin, GORM, PostgreSQL (production) / SQLite (dev/test), JWT, Swagger
- **Frontend**: Next.js 16, React 19, TypeScript, Tailwind CSS 4, Zustand, TanStack Query, next-intl
- **Testing**: Vitest, Playwright, MSW

---

## Backend Specs

| File | When to Read |
|------|-------------|
| [backend-architecture.md](./backend-architecture.md) | New backend feature, project structure, layer responsibilities, database patterns |
| [backend-error-handling.md](./backend-error-handling.md) | Adding/modifying error handling, new API endpoints, new error codes, i18n error messages |
| [backend-quality.md](./backend-quality.md) | Writing tests, code review, logging, Swagger documentation |

## Frontend Specs

| File | When to Read |
|------|-------------|
| [frontend-architecture.md](./frontend-architecture.md) | New frontend feature, project structure, state management decisions |
| [frontend-components-hooks.md](./frontend-components-hooks.md) | Building React components, creating hooks, styling, accessibility |
| [frontend-type-safety.md](./frontend-type-safety.md) | TypeScript types, API response types, type definitions |
| [frontend-quality.md](./frontend-quality.md) | Writing unit tests, code review, accessibility checks |
| [frontend-e2e-testing.md](./frontend-e2e-testing.md) | Writing/running E2E tests, Playwright setup, Page Objects |

## Cross-Cutting

| File | When to Read |
|------|-------------|
| [thinking-guides.md](./thinking-guides.md) | Before any coding task — catch duplication, cross-layer issues, and edge cases early |

---

## Task-Based Quick Reference

- **New backend API endpoint** → `backend-architecture.md` + `backend-error-handling.md`
- **New frontend page/component** → `frontend-architecture.md` + `frontend-components-hooks.md`
- **Adding a data-fetching hook** → `frontend-components-hooks.md` + `frontend-type-safety.md`
- **Writing unit tests** → `backend-quality.md` or `frontend-quality.md`
- **Writing E2E tests** → `frontend-e2e-testing.md`
- **Adding a new error code** → `backend-error-handling.md` (3 places to update)
- **Refactoring shared logic** → `thinking-guides.md`
