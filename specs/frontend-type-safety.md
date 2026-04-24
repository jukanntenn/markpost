# Frontend Type Safety

> TypeScript conventions and type organization.

## Rules

- **Prefer `interface`** for object shapes
- **Type definitions** in `types/` directory
- **Explicit return types** for exported functions
- **No `any`** — use `unknown` when type is uncertain
- **Type-only imports** for types: `import type { PostsResponse } from "./posts"`

## Type Organization

### `types/` Directory

Define shared types by domain in `types/`:

```tsx
// types/posts.ts
export interface PostListItem {
  id: string;
  qid: string;
  title: string;
  created_at: string;
}

export interface PostsPaginatedResponse {
  posts: PostListItem[];
  pagination: { page: number; limit: number; total: number; total_pages: number };
}
```

### Colocated Types

Define types alongside code when only used locally:

```tsx
// hooks/usePosts.ts
interface UsePostsOptions {
  refreshInterval?: number;
}
```

### Props Interfaces

Define above the component:

```tsx
interface CreateTestPostModalProps {
  show: boolean;
  postKey: string;
  onHide: () => void;
  onSuccess: () => void;
}
```

## Validation

### Error Extraction

Type-safe error handling without `any`:

```tsx
export function getErrorMessage(err: unknown, fallback: string): string {
  if (err instanceof Error) return err.message;
  return fallback;
}
```

## Forbidden Patterns

### 1. Using `any`

```tsx
// Bad
function processData(data: any) { return data.value; }

// Good — use unknown with type guard
function processData(data: unknown) {
  if (typeof data === "object" && data !== null && "value" in data) return data.value;
  throw new Error("Invalid data");
}
```

### 2. Type Assertions Without Validation

```tsx
// Bad
const user = response as User;

// Good
const user = parseUser(response); // validates and transforms
```

### 3. Non-Null Assertions

```tsx
// Bad
const name = user!.name;

// Good — optional chaining or conditional
const name = user?.name;
```

### 4. Implicit Any in Callbacks

```tsx
// Bad
array.map(item => item.id)

// Good
array.map((item: PostListItem) => item.id)
```

## Type Imports

Use `type` keyword for type-only imports:

```tsx
import type { PostsPaginatedResponse } from "@/types/posts";
```

Mixed imports are also acceptable:

```tsx
import { usePosts } from "./usePosts";
import type { UsePostsOptions } from "./usePosts";
```
