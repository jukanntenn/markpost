# Frontend Architecture

> Project structure and state management patterns.

## Directory Layout

```
frontend/src/
├── app/                    # Next.js App Router
│   ├── (auth)/             # Auth route group (no sidebar)
│   │   ├── login/page.tsx
│   │   ├── auth/page.tsx
│   │   └── layout.tsx
│   ├── (dashboard)/        # Dashboard route group (with sidebar)
│   │   ├── admin/          # Admin pages
│   │   ├── dashboard/page.tsx
│   │   ├── posts/page.tsx
│   │   ├── settings/page.tsx
│   │   └── layout.tsx
│   ├── layout.tsx          # Root layout with providers
│   ├── page.tsx            # Home/landing page
│   └── globals.css         # Tailwind + theme tokens
├── components/             # React components
│   ├── ui/                 # shadcn/ui primitives
│   ├── auth/               # Auth components (AdminRoute, ProtectedRoute, PublicRoute)
│   ├── layout/             # Layout components (AdminLayout, DashboardLayout)
│   ├── login/              # Login components
│   ├── dashboard/          # Dashboard components
│   ├── admin/              # Admin components
│   └── posts/              # Post components
├── hooks/                  # Custom React hooks
├── lib/                    # Library utilities
│   ├── utils.ts            # cn() utility
│   └── api/                # API fetchers
├── types/                  # TypeScript type definitions
├── utils/                  # Utility functions
├── i18n/                   # next-intl configuration
├── mocks/                  # MSW handlers
└── test/                   # Test utilities
```

## Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Component files | PascalCase | `DashboardPage.tsx` |
| Hook files | camelCase with `use` prefix | `usePosts.ts` |
| Type files | camelCase | `posts.ts` |
| Page files | `page.tsx` (Next.js convention) | `app/dashboard/page.tsx` |
| Layout files | `layout.tsx` (Next.js convention) | `app/(dashboard)/layout.tsx` |
| Test files | `.test.tsx` suffix | `ThemeToggle.test.tsx` |

## State Management

### State Categories

| Category | Solution | Examples |
|----------|----------|----------|
| Server state | TanStack Query | Posts, users, settings |
| Auth state | Zustand + localStorage | User info, tokens |
| Theme state | next-themes | Dark/light mode |
| Form state | useState | Input values, form errors |
| UI state | useState | Modal visibility, loading |

### Server State (TanStack Query)

- Define data-fetching hooks in `hooks/` using `useQuery`
- Define mutation hooks using `useMutation`
- Always invalidate cache on mutation success (`queryClient.invalidateQueries`)
- Never use `useEffect` for data fetching
- Always type the generic parameter: `useQuery<PostsResponse>(...)`

### Global Client State (Zustand)

- Use `create()` with `persist` middleware for state that survives page refresh
- Use `partialize` to select which fields to persist
- Auth tokens and user info are persisted in localStorage

### When to Use Global State

**Promote to global when:**
- Multiple components need access
- State affects routing
- State persists across navigation

**Keep local when:**
- Only one component uses it
- State is temporary
- State is derived from props

## Adding a New Feature

1. Create types in `types/` if new data structures
2. Create hooks in `hooks/` for data fetching
3. Create page component in `app/(dashboard)/` or `app/(auth)/`
4. Create feature components in `components/`
5. Add tests alongside source files
