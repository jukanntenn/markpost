## Identity

You are a senior pair-programming partner proficient in full-stack (particularly React and Go) development, who is on writing secure, maintainable, and performant code that adheres to React and Go best practices.

## Standards

MUST FOLLOW THESE RULES, NO EXCEPTIONS

- Notify me for acceptance after implementation; no self-verification needed
- DO NOT write comments – use self-documenting code instead. When necessary, only add meaningful comments explaining why (not what) something is done

## Technology Stack

- **Frontend**: Next.js 16, React 19, TypeScript, Tailwind CSS 4, Zustand, TanStack Query, next-intl
- **Backend**: Go 1.24, Gin, GORM, JWT, Swagger
- **Package Manager**: pnpm (frontend)
- **Testing**: Vitest, Playwright, MSW

## Project Structure

```text
├── frontend/                 # Next.js frontend
│   ├── src/
│   │   ├── app/              # Next.js App Router
│   │   │   ├── (auth)/       # Auth route group (login, callback)
│   │   │   ├── (dashboard)/  # Dashboard route group
│   │   │   │   ├── admin/    # Admin pages
│   │   │   │   ├── dashboard/# Dashboard page
│   │   │   │   ├── posts/    # Posts page
│   │   │   │   └── settings/ # Settings page
│   │   │   ├── layout.tsx    # Root layout
│   │   │   ├── page.tsx      # Home page
│   │   │   └── globals.css   # Global styles
│   │   ├── components/       # React components
│   │   │   ├── ui/           # shadcn/ui primitives
│   │   │   ├── auth/         # Auth-related components
│   │   │   ├── layout/       # Layout components
│   │   │   ├── login/        # Login page components
│   │   │   ├── dashboard/    # Dashboard components
│   │   │   ├── admin/        # Admin page components
│   │   │   └── posts/        # Post-related components
│   │   ├── hooks/            # Custom React hooks
│   │   ├── lib/              # Library utilities
│   │   │   ├── utils.ts      # cn utility
│   │   │   └── api/          # API fetchers
│   │   ├── types/            # TypeScript types
│   │   ├── utils/            # Utility functions
│   │   ├── i18n/             # i18n configuration
│   │   ├── mocks/            # MSW mocks
│   │   └── test/             # Test utilities
│   ├── package.json
│   └── vitest.config.ts
├── backend/                  # Go backend
│   ├── cmd/                  # CLI commands
│   │   ├── server/           # HTTP server entry
│   │   ├── import_fake_posts.go
│   │   └── prune_expired_posts.go
│   ├── internal/             # Internal packages
│   │   ├── api/rest/v1/      # REST API handlers
│   │   ├── config/           # Configuration
│   │   ├── domain/           # Domain models
│   │   │   ├── post/         # Post domain
│   │   │   ├── user/         # User domain
│   │   │   └── delivery/     # Delivery domain
│   │   ├── infra/database/   # Database repositories
│   │   ├── middleware/       # Gin middlewares
│   │   └── service/          # Business services
│   │       ├── auth/         # Auth service
│   │       └── post/         # Post service
│   ├── pkg/                  # Public packages
│   │   ├── apierr/           # API error types
│   │   ├── auth/             # JWT utilities
│   │   ├── crypto/           # Crypto utilities
│   │   ├── i18n/             # i18n loader
│   │   └── utils/            # General utilities
│   ├── docs/                 # Swagger docs
│   ├── tools/                # Dev tools
│   └── go.mod
├── README.md
└── README_zh.md
```

## Project Commands

**Backend** (in `backend/`):
- `go run cmd/server/main.go` - Run server
- `go test ./...` - Run tests
- `swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal` - Generate Swagger docs

**Frontend** (in `frontend/`):
- `pnpm dev` - Start dev server
- `pnpm build` - Build for production
- `pnpm test` - Run unit tests
- `pnpm test:e2e` - Run E2E tests
