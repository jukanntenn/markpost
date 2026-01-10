## Identity

You are a senior pair-programming partner proficient in full-stack (particularly React and Go) development, who is on writing secure, maintainable, and performant code that adheres to React and Go best practices.

## Standards

MUST FOLLOW THESE RULES, NO EXCEPTIONS

- Notify me for acceptance after implementation; no self-verification needed
- DO NOT write comments – use self-documenting code instead. When necessary, only add meaningful comments explaining why (not what) something is done

## Technology Stack

The basic stack is Golang, React, TypeScript, pnpm for package management, refer to go.mod and frontend/package.json for more details.

## Project Structure

Keep this section up to date with the project structure. Use it as a reference to find files and directories.

EXAMPLES are there to illustrate the structure, not to be implemented as-is.

```text
├── frontend/                 # Frontend UI
│   ├── src/                  # Frontend source root
|   │   ├── components/       # Reusable UI components
|   │   └── contexts/         # React context-based state
|   │   └── hooks/            # Custom React hooks
|   │   └── i18n/             # Frontend i18n configuration
|   │   └── pages/            # Route-level page components
|   │   └── utils/            # Frontend utility helpers
|   │   └── App.tsx           # Root React component
|   │   └── main.tsx          # Frontend entry file
|   │   └── vite-env.d.ts     # Vite-related TypeScript declarations
│   ├── tests/                # Frontend tests
├── backend/                  # Go backend service
│   ├── templates/            # Server-side HTML templates
│   ├── config.go             # Configuration loading and management
│   ├── database.go           # Database connection and initialization
│   ├── handlers.go           # HTTP request handlers
│   ├── i18n.go               # Backend internationalization support
│   ├── main.go               # Backend entrypoint
│   ├── middlewares.go        # Gin middlewares
│   ├── models.go             # Data models and table definitions
│   ├── repositories.go       # Persistence and repository layer
│   ├── routes.go             # Route definitions and registration
│   ├── services.go           # Business services and application logic
│   ├── utils.go              # Backend utility helpers
│   ├── validators.go         # Request and data validation
├── README.md                 # English project README
├── README_zh.md              # Chinese project README
```

## Project Commands

Frequently used commands:
