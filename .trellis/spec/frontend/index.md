# Frontend Development Guidelines

> Best practices for frontend development in this project.

---

## Overview

Guidelines for frontend development using Next.js 16, React 19, TypeScript, and Tailwind CSS v4.

---

## Guidelines Index

| Guide | Description |
|-------|-------------|
| [Directory Structure](./directory-structure.md) | Component/page/hook organization |
| [Component Guidelines](./component-guidelines.md) | React patterns, props, styling |
| [Hook Guidelines](./hook-guidelines.md) | TanStack Query patterns |
| [State Management](./state-management.md) | Zustand + Context patterns |
| [Type Safety](./type-safety.md) | TypeScript conventions |
| [Quality Guidelines](./quality-guidelines.md) | Testing, accessibility |

---

## Quick Reference

### Technology Stack
- **Framework**: Next.js 16 with App Router
- **React**: React 19
- **Styling**: Tailwind CSS v4 + shadcn/ui
- **State**: Zustand (client), TanStack Query (server)
- **i18n**: next-intl
- **Testing**: Vitest + Testing Library + MSW + Playwright

### Architecture Pattern
```
Page (app/)
    ├── TanStack Query Hook (server state)
    ├── Zustand Store (client state)
    └── useState (local UI state)
```

### Key Conventions
1. **App Router** - File-based routing with route groups
2. **shadcn/ui** - Use components from `components/ui/`
3. **TanStack Query** - No useEffect for data fetching
4. **Zustand** - Global client state with persist

### Common Commands
```bash
pnpm dev        # Start dev server
pnpm build      # Build for production
pnpm lint       # Run ESLint
pnpm test       # Run unit tests
pnpm test:e2e   # Run E2E tests
```
