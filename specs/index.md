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

## DevOps

| File | When to Read |
|------|-------------|
| [devops-spec.md](./devops-spec.md) | Dev script changes, Docker/Compose config, Ansible playbooks, adding new deployment environments |
| [service-routing-spec.md](./service-routing-spec.md) | General frontend-backend integration mechanism, environment models, Docker topology, image builds — reusable across projects |

## Configuration

| File | When to Read |
|------|-------------|
| [dev-config-spec.md](./dev-config-spec.md) | Adding or changing configuration values, setting up dev environment defaults, reducing env file duplication, IDE task runner env vars |

## Design

| File | When to Read |
|------|-------------|
| [routing-design.md](./routing-design.md) | This project's route tables, value-prefix definitions, permission model, rewrite rules |

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
- **Adding or changing a config value** → `dev-config-spec.md`
- **Refactoring shared logic** → `thinking-guides.md`
- **Adding a deployment environment** → `devops-spec.md` + `service-routing-spec.md`
- **Modifying dev startup script** → `devops-spec.md` + `dev-config-spec.md`
- **Frontend-backend routing changes** → `routing-design.md` + `service-routing-spec.md`
- **Docker Compose topology changes** → `service-routing-spec.md`
- **Adding or changing API routes** → `routing-design.md`
- **Permission/auth changes** → `routing-design.md`
