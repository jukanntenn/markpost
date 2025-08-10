# Markpost frontend

## Standards

MUST FOLLOW THESE RULES, NO EXCEPTIONS

- Stack: React, TypeScript, React Router v7, React Bootstrap, Axios
- Patterns: ALWAYS use Function components + Hooks, NEVER use Class components
- ALWAYS Keep types alongside your code, use TypeScript for type safety, prefer `interface` over `type` for defining types
- ONLY use the default styles of components, NEVER add any additional CSS styles.
- ONLY add meaningful comments that explain why something is done, not what it does
- Dev server is already running on `http://localhost:5173` with HMR enabled. NEVER launch it yourself
- ALWAYS use named functions when declaring methods, use arrow functions only for callbacks
- ALWAYS prefer named exports over default exports

## Project Structure

Keep this section up to date with the project structure. Use it as a reference to find files and directories.

EXAMPLES are there to illustrate the structure, not to be implemented as-is.

```text
public/ # Public static files (favicon, robots.txt, static images, etc.)
src/
├── api/ # MUST export individual functions that fetch data
│   ├── users.ts # EXAMPLE file for user-related API functions
│   └── posts.ts # EXAMPLE file for post-related API functions
├── components/ # Reusable Vue components
│   ├── Narvbar.tsx # EXAMPLE file for navigation bar component
│   └── PostList.tsx # EXAMPLE file for post list component
├── pages/ # Page components (Vue Router + Unplugin Vue Router)
│   ├── Login.tsx # EXAMPLE login page at /login
│   ├── Settings.tsx # EXAMPLE settings at /settings
├── utils/ # Global utility pure functions
├── hooks/ # React hooks
│   ├── auth.ts # EXAMPLE file for authentication-related hooks
│   └── user.ts # EXAMPLE file for user-related hooks
├── assets/ # Static assets that are processed by Vite (e.g CSS)
├── main.tsx # Entry point for the application, add and configure plugins, and mount the app
├── App.tsx # Root React component
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
