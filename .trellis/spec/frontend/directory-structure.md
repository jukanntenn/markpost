# Directory Structure

> How frontend code is organized in this project.

---

## Overview

The frontend follows a feature-based organization with clear separation between:
- **UI primitives** - shadcn/ui components in `components/ui/`
- **Feature components** - Domain-specific components in `components/`
- **Pages** - Route-level components in `pages/`
- **Hooks** - Custom hooks organized by type in `hooks/`

---

## Directory Layout

```
frontend/src/
├── components/             # Reusable UI + feature components
│   ├── ui/                 # shadcn/ui primitives (button, dialog, etc.)
│   └── login/              # Feature-specific components
├── contexts/               # React context-based state
├── hooks/                  # Custom React hooks
│   └── swr/                # SWR-based data fetching hooks
├── i18n/                   # i18next configuration
├── lib/                    # Shared library helpers (cn utility)
├── mocks/                  # MSW handlers and test mocks
├── pages/                  # Route-level page components
├── swr/                    # SWR configuration and utilities
├── test/                   # Test utilities and setup
├── types/                  # TypeScript type definitions
├── utils/                  # Utility helpers
├── App.tsx                 # Root React component
├── index.css               # Tailwind v4 + shadcn theme tokens
├── main.tsx                # Frontend entry file
└── vite-env.d.ts           # Vite-related TypeScript declarations
```

---

## Directory Responsibilities

### components/ui/

shadcn/ui primitives - do not modify unless customizing the design system:

```
components/ui/
├── button.tsx
├── dialog.tsx
├── input.tsx
├── textarea.tsx
└── ...
```

### components/

Feature components organized by domain or function:

```
components/
├── Layout.tsx              # Main layout with header
├── ThemeToggle.tsx         # Dark/light mode toggle
├── LanguageToggle.tsx      # Language selector
├── CreateTestPostModal.tsx # Modal component
├── UserInfoProvider.tsx    # Context provider
├── UserInfoContext.ts      # Context definition
└── login/                  # Login page-specific components
    ├── LoginTitle.tsx
    └── LoginDivider.tsx
```

### pages/

Route-level components mapped to URL paths:

```
pages/
├── Dashboard.tsx           # /dashboard
├── Posts.tsx               # /posts
├── Settings.tsx            # /settings
├── Admin.tsx               # /admin
├── Login.tsx               # /login
├── LoginCallback.tsx       # /login/callback
└── NotFound.tsx            # 404 page
```

### hooks/swr/

SWR-based data fetching hooks:

```
hooks/swr/
├── usePosts.ts             # Fetch posts list
├── useCreateTestPost.ts    # Create post mutation
├── usePostKey.ts           # Fetch user's post key
├── useUsers.ts             # Fetch users (admin)
└── useDeliveryChannels.ts  # Delivery channel operations
```

### types/

TypeScript type definitions by domain:

```
types/
├── posts.ts                # Post-related types
├── auth.ts                 # Auth-related types
```

### utils/

Utility functions and API configuration:

```
utils/
├── api.ts                  # Axios instances and error handling
├── storage.ts              # localStorage utilities
├── auth.ts                 # Auth helpers
├── i18n.ts                 # i18n helpers
└── url.ts                  # URL utilities
```

---

## Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Component files | PascalCase | `CreateTestPostModal.tsx` |
| Hook files | camelCase with `use` prefix | `usePosts.ts` |
| Type files | camelCase | `posts.ts`, `auth.ts` |
| Context files | PascalCase with `Context` suffix | `UserInfoContext.ts` |
| Provider files | PascalCase with `Provider` suffix | `UserInfoProvider.tsx` |
| Test files | Same as source with `.test.tsx` | `usePosts.test.ts` |

---

## Adding a New Feature

1. Create types in `types/` if new data structures
2. Create SWR hooks in `hooks/swr/` for data fetching
3. Create page component in `pages/`
4. Create feature components in `components/`
5. Add route in `App.tsx`
6. Add tests alongside source files

---

## Examples

- **Complete feature**: `components/CreateTestPostModal.tsx` + `hooks/swr/useCreateTestPost.ts` + `types/posts.ts`
- **Context pattern**: `components/UserInfoContext.ts` + `components/UserInfoProvider.tsx`
- **SWR hook with tests**: `hooks/swr/usePosts.ts` + `hooks/swr/usePosts.test.ts`
