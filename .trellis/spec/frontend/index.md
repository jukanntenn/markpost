# Frontend Development Guidelines

> Best practices for frontend development in this project.

---

## Overview

This directory contains guidelines for frontend development using React, TypeScript, and Tailwind CSS v4 with shadcn/ui components.

---

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | Component/page/hook organization | ✓ Filled |
| [Component Guidelines](./component-guidelines.md) | React patterns, props, styling | ✓ Filled |
| [Hook Guidelines](./hook-guidelines.md) | SWR hooks, data fetching patterns | ✓ Filled |
| [State Management](./state-management.md) | SWR + React Context patterns | ✓ Filled |
| [Type Safety](./type-safety.md) | TypeScript conventions | ✓ Filled |
| [Quality Guidelines](./quality-guidelines.md) | Testing, accessibility, linting | ✓ Filled |

---

## Quick Reference

### Technology Stack
- **Framework**: React 18 with TypeScript
- **Build**: Vite 7
- **Styling**: Tailwind CSS v4 + shadcn/ui
- **Data Fetching**: SWR
- **Routing**: React Router v6
- **i18n**: i18next
- **Testing**: Vitest + Testing Library + MSW

### Architecture Pattern
```
Page Component
    ├── SWR Hook (server state)
    │   └── fetcher → API
    ├── Context (global client state)
    └── useState (local UI state)
```

### Key Conventions
1. **Function components only** - No class components
2. **shadcn/ui primitives** - Use components from `components/ui/`
3. **Design tokens** - Use `bg-background`, `text-foreground`, etc.
4. **SWR for server state** - No useEffect for data fetching
5. **Context for auth** - UserInfoContext for authentication

### Common Commands
```bash
pnpm dev        # Start development server
pnpm build      # Build for production
pnpm lint       # Run ESLint
pnpm test       # Run tests
```

---

**Language**: All documentation is written in **English**.
