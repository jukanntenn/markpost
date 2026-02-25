# Type Safety

> Type safety patterns in this project.

---

## Overview

This project uses TypeScript with strict type checking. Key conventions:
- **Prefer `interface`** for object shapes
- **Type definitions** in `types/` directory
- **Explicit return types** for exported functions
- **No `any`** - use `unknown` when type is uncertain

---

## Type Organization

### types/ Directory

Organize types by domain:

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
  pagination: {
    page: number;
    limit: number;
    total: number;
    total_pages: number;
  };
}

export interface CreateTestPostRequest {
  title: string;
  body: string;
}

export interface CreateTestPostResponse {
  id: string;
}
```

```tsx
// types/auth.ts
export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  user: {
    id: number;
    username: string;
    role: string;
  };
}
```

### Colocated Types

Define types alongside code when only used locally:

```tsx
// hooks/swr/usePosts.ts
interface UsePostsOptions {
  refreshInterval?: number;
}
```

### Props Interfaces

Define props interface above component:

```tsx
// components/CreateTestPostModal.tsx:22-27
interface CreateTestPostModalProps {
  show: boolean;
  postKey: string;
  onHide: () => void;
  onSuccess: () => void;
}
```

---

## Common Patterns

### API Response Types

```tsx
// types/posts.ts:8-16
export interface PostsPaginatedResponse {
  posts: PostListItem[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    total_pages: number;
  };
}
```

### Hook Return Types

Let TypeScript infer when using SWR:

```tsx
// Return type is inferred from generic parameter
export function usePosts(page: number, limit: number = 20) {
  return useSWR<PostsPaginatedResponse>(/* ... */);
}

// Usage - type is inferred
const { data } = usePosts(1); // data: PostsPaginatedResponse | undefined
```

### Mutation Args Types

```tsx
// hooks/swr/useCreateTestPost.ts:5-8
export interface CreateTestPostArgs {
  title: string;
  body: string;
}
```

### Context Types

```tsx
// components/UserInfoContext.ts
export interface UserInfo {
  access_token: string;
  refresh_token: string;
  user: {
    id: number;
    username: string;
    role: string;
  };
}

export interface UserInfoContextType {
  isAuthenticated: boolean;
  userInfo: UserInfo | null;
  setUserInfo: (info: UserInfo | null) => void;
  logout: () => void;
  isAdmin: () => boolean;
}
```

---

## Validation

### API Error Handling

Type-safe error extraction:

```tsx
// utils/api.ts
export function getErrorMessage(err: unknown, fallback: string): string {
  if (axios.isAxiosError(err)) {
    return err.response?.data?.message || fallback;
  }
  if (err instanceof Error) {
    return err.message;
  }
  return fallback;
}
```

### Error Status Extraction

```tsx
// swr/config.ts:5-14
type ErrorWithStatus = { status?: number };
type ErrorWithResponseStatus = { response?: { status?: number } };

const getStatus = (err: unknown): number | undefined => {
  const status = (err as ErrorWithStatus | undefined)?.status;
  if (typeof status === "number") return status;
  const responseStatus = (err as ErrorWithResponseStatus | undefined)?.response?.status;
  if (typeof responseStatus === "number") return responseStatus;
  return undefined;
};
```

---

## Forbidden Patterns

### 1. Using `any`

```tsx
// Bad
function processData(data: any) {
  return data.value;
}

// Good - use unknown with type guard
function processData(data: unknown) {
  if (typeof data === "object" && data !== null && "value" in data) {
    return data.value;
  }
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

// Good
if (user) {
  const name = user.name;
}

// Or use optional chaining
const name = user?.name;
```

### 4. Implicit Any in Callbacks

```tsx
// Bad
array.map(item => item.id)

// Good
array.map((item: PostListItem) => item.id)
```

---

## Type Imports

Use type-only imports for types:

```tsx
// Good
import type { PostsPaginatedResponse } from "../../types/posts";
import type { LoginResponse } from "../types/auth";

// Also acceptable for mixed imports
import { usePosts } from "./usePosts";
import type { UsePostsOptions } from "./usePosts";
```
