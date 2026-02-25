# Markpost frontend

## Standards

MUST FOLLOW THESE RULES, NO EXCEPTIONS

- Stack: React, TypeScript, Vite, React Router, Axios, i18next, next-themes, shadcn/ui, Tailwind CSS v4, Radix primitives
- Patterns: ALWAYS use Function components + Hooks, NEVER use Class components
- Types: Keep types alongside code; prefer `interface` for object shapes
- Styling: Use shadcn/ui primitives + Tailwind utilities; prefer design tokens (`bg-background`, `text-foreground`, `border-border`)
- CSS: Global styles and theme tokens live in `src/index.css`; avoid adding additional CSS files unless required
- Accessibility: Ensure focus-visible states, labels for form fields, and keyboard navigation for menus/dialogs
- Responsive: Implement responsive layouts for mobile/tablet/desktop

## Project Structure

Keep this section up to date with the project structure. Use it as a reference to find files and directories.

EXAMPLES are there to illustrate the structure, not to be implemented as-is.

```text
src/
├── components/             # Reusable UI + feature components
│   ├── ui/                 # shadcn/ui primitives
│   └── login/              # Login page-specific components
├── contexts/               # React context-based state
├── hooks/                  # Custom React hooks
├── i18n/                   # Frontend i18n configuration
├── lib/                    # Shared library helpers (e.g. cn)
├── mocks/                  # MSW handlers and test mocks
├── pages/                  # Route-level pages
├── utils/                  # Utility helpers (api, storage, etc.)
├── App.tsx                 # Root React component
├── index.css               # Tailwind v4 + shadcn theme tokens
└── main.tsx                # Frontend entry file
```

## Research & Documentation

- **NEVER hallucinate or guess URLs**
- ALWAYS try accessing the `llms.txt` file first to find relevant documentation. EXAMPLE: `https://pinia-colada.esm.dev/llms.txt`
  - If it exists, it will contain other links to the documentation for the LLMs used in this project
- ALWAYS follow existing links in table of contents or documentation indices
- Verify examples and patterns from documentation before using

## React Components Best Practices

- Name files consistently using PascalCase (`PostList.tsx`)
- ALWAYS use PascalCase for component names in source code
- Compose names from the most general to the most specific: `SearchButtonClear.tsx` not `ClearSearchButton.tsx`
- Call hooks (useState, useEffect, etc.) only at the top level
- Extract reusable logic into custom hooks (useAuth, useFormValidation)
- Prefer controlled components for forms
