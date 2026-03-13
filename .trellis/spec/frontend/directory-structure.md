# Directory Structure

> How frontend code is organized in this project.

---

## Overview

The frontend uses **Next.js 16 with App Router**, following a feature-based organization:

- **App Router** - File-based routing with route groups
- **UI primitives** - shadcn/ui components in `components/ui/`
- **Feature components** - Domain-specific components in `components/`
- **Hooks** - Custom hooks for data fetching and logic

---

## Directory Layout

```
frontend/src/
├── app/                    # Next.js App Router
│   ├── (auth)/             # Auth route group
│   │   ├── login/page.tsx
│   │   ├── auth/page.tsx
│   │   └── layout.tsx
│   ├── (dashboard)/        # Dashboard route group
│   │   ├── admin/          # Admin pages
│   │   ├── dashboard/page.tsx
│   │   ├── posts/page.tsx
│   │   ├── settings/page.tsx
│   │   └── layout.tsx
│   ├── layout.tsx          # Root layout
│   ├── page.tsx            # Home page
│   └── globals.css         # Global styles
├── components/             # React components
│   ├── ui/                 # shadcn/ui primitives
│   ├── auth/               # Auth components
│   ├── layout/             # Layout components
│   ├── login/              # Login components
│   ├── dashboard/          # Dashboard components
│   ├── admin/              # Admin components
│   └── posts/              # Post components
├── hooks/                  # Custom React hooks
├── lib/                    # Library utilities
│   ├── utils.ts            # cn utility
│   └── api/                # API fetchers
├── types/                  # TypeScript types
├── utils/                  # Utility functions
├── i18n/                   # next-intl configuration
├── mocks/                  # MSW handlers
└── test/                   # Test utilities
```

---

## Directory Responsibilities

### app/

Next.js App Router with route groups for layout organization:

```
app/
├── (auth)/                 # Auth routes (no sidebar)
│   ├── login/page.tsx
│   └── auth/page.tsx
├── (dashboard)/            # Dashboard routes (with sidebar)
│   ├── admin/
│   │   ├── channels/page.tsx
│   │   ├── posts/page.tsx
│   │   ├── users/page.tsx
│   │   ├── page.tsx
│   │   └── layout.tsx
│   ├── dashboard/page.tsx
│   ├── posts/page.tsx
│   ├── settings/page.tsx
│   └── layout.tsx
├── layout.tsx              # Root layout with providers
├── page.tsx                # Home/landing page
└── globals.css             # Tailwind + theme tokens
```

### components/

Feature components organized by domain:

```
components/
├── ui/                     # shadcn/ui primitives
│   ├── button.tsx
│   ├── dialog.tsx
│   ├── input.tsx
│   └── ...
├── auth/                   # Auth components
│   ├── AdminRoute.tsx
│   ├── ProtectedRoute.tsx
│   └── PublicRoute.tsx
├── layout/                 # Layout components
│   ├── AdminLayout.tsx
│   └── DashboardLayout.tsx
├── login/                  # Login components
│   ├── LoginPage.tsx
│   ├── LoginCallbackPage.tsx
│   ├── LoginTitle.tsx
│   └── LoginDivider.tsx
├── dashboard/
│   └── DashboardPage.tsx
├── admin/
│   ├── AdminChannelsPage.tsx
│   ├── AdminPostsPage.tsx
│   └── AdminUsersPage.tsx
└── posts/
    └── PostsPage.tsx
```

### hooks/

Custom React hooks:

```
hooks/
├── usePosts.ts
├── usePostKey.ts
└── useUserInfo.ts
```

### lib/

Library utilities:

```
lib/
├── utils.ts                # cn() utility
└── api/
    └── fetcher.ts          # API fetchers
```

### types/

TypeScript type definitions:

```
types/
├── auth.ts
└── posts.ts
```

---

## Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Component files | PascalCase | `DashboardPage.tsx` |
| Hook files | camelCase with `use` prefix | `usePosts.ts` |
| Type files | camelCase | `posts.ts` |
| Page files | `page.tsx` (Next.js convention) | `app/dashboard/page.tsx` |
| Layout files | `layout.tsx` (Next.js convention) | `app/(dashboard)/layout.tsx` |
| Test files | `.test.tsx` suffix | `ThemeToggle.test.tsx` |

---

## Adding a New Feature

1. Create types in `types/` if new data structures
2. Create hooks in `hooks/` for data fetching
3. Create page component in `app/(dashboard)/` or `app/(auth)/`
4. Create feature components in `components/`
5. Add tests alongside source files
