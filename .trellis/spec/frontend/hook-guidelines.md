# Hook Guidelines

> How hooks are used in this project.

---

## Overview

This project uses **SWR** for data fetching and caching. Custom hooks follow these conventions:
- Data fetching hooks use `useSWR` or `useSWRMutation`
- Hooks are organized in `hooks/swr/`
- Return types are explicitly typed with generics

---

## Custom Hook Patterns

### Data Fetching Hook

Use `useSWR` for GET requests:

```tsx
// hooks/swr/usePosts.ts:1-19
import useSWR from "swr";
import { authFetcher } from "../../swr/fetcher";
import type { PostsPaginatedResponse } from "../../types/posts";

interface UsePostsOptions {
  refreshInterval?: number;
}

export function usePosts(page: number, limit: number = 20, options?: UsePostsOptions) {
  return useSWR<PostsPaginatedResponse>(
    page ? `/api/posts?page=${page}&limit=${limit}` : null,
    authFetcher,
    {
      refreshInterval: options?.refreshInterval,
      refreshWhenHidden: false,
      revalidateOnFocus: false,
    }
  );
}
```

### Mutation Hook

Use `useSWRMutation` for POST/PUT/DELETE:

```tsx
// hooks/swr/useCreateTestPost.ts:1-20
import useSWRMutation from "swr/mutation";
import { anno } from "../../utils/api";
import type { CreateTestPostResponse } from "../../types/posts";

export interface CreateTestPostArgs {
  title: string;
  body: string;
}

const sendRequest = async (
  url: string,
  { arg }: { arg: CreateTestPostArgs }
): Promise<CreateTestPostResponse> => {
  const res = await anno.post<CreateTestPostResponse>(url, arg);
  return res.data;
};

export function useCreateTestPost(postKey: string) {
  return useSWRMutation(postKey ? `/${postKey}` : null, sendRequest);
}
```

### Using Hooks in Components

```tsx
// components/CreateTestPostModal.tsx:36
const { trigger, isMutating, reset } = useCreateTestPost(postKey);

// Trigger mutation
await trigger({ title: title.trim(), body });
```

---

## Data Fetching

### SWR Configuration

Global SWR config in `swr/config.ts`:

```tsx
// swr/config.ts:16-31
export const swrConfig: SWRConfiguration = {
  fetcher: authFetcher,
  use: [authMiddleware],
  revalidateOnFocus: true,
  revalidateOnReconnect: true,
  dedupingInterval: 2000,
  errorRetryCount: 3,
  shouldRetryOnError: (err) => {
    const status = getStatus(err);
    const message = (err as Error)?.message;
    if (message === "No access token available") {
      return false;
    }
    return status !== 401 && status !== 404;
  },
};
```

### Fetchers

Two types of fetchers:

```tsx
// swr/fetcher.ts:1-16
// Authenticated fetcher (includes access token)
export const authFetcher = (url: string) => {
  const loginData = get<LoginResponse | null>("login");
  const accessToken = loginData?.access_token;
  if (!accessToken) {
    return Promise.reject(new Error("No access token available"));
  }
  return auth.get(url).then((res) => res.data);
};

// Anonymous fetcher (no auth required)
export const annoFetcher = (url: string) => anno.get(url).then((res) => res.data);
```

### Conditional Fetching

Pass `null` as key to disable fetching:

```tsx
// hooks/swr/usePosts.ts:11
page ? `/api/posts?page=${page}&limit=${limit}` : null

// hooks/swr/useCreateTestPost.ts:19
postKey ? `/${postKey}` : null
```

---

## Naming Conventions

| Type | Pattern | Example |
|------|---------|---------|
| Data fetching hook | `use<Resource>` | `usePosts`, `useUsers` |
| Mutation hook | `use<Create/Update/Delete><Resource>` | `useCreateTestPost` |
| Options interface | `Use<Resource>Options` | `UsePostsOptions` |
| Args interface | `<Action>Args` | `CreateTestPostArgs` |

---

## Return Values

### useSWR Return

```tsx
const { data, error, isLoading, mutate } = usePosts(1, 20);
```

| Property | Type | Description |
|----------|------|-------------|
| `data` | `T \| undefined` | Fetched data |
| `error` | `Error \| undefined` | Fetch error |
| `isLoading` | `boolean` | Initial load in progress |
| `mutate` | `function` | Revalidate or update cache |

### useSWRMutation Return

```tsx
const { trigger, isMutating, reset } = useCreateTestPost(postKey);
```

| Property | Type | Description |
|----------|------|-------------|
| `trigger` | `function` | Execute mutation |
| `isMutating` | `boolean` | Mutation in progress |
| `reset` | `function` | Reset mutation state |

---

## Common Mistakes

### 1. Not Typing the Response

```tsx
// Bad - no type safety
const { data } = useSWR("/api/posts", authFetcher);

// Good - typed response
const { data } = useSWR<PostsPaginatedResponse>("/api/posts", authFetcher);
```

### 2. Not Handling Loading State

```tsx
// Bad
return <div>{data.items}</div>;

// Good
if (isLoading) return <div>Loading...</div>;
if (error) return <div>Error: {error.message}</div>;
return <div>{data?.items}</div>;
```

### 3. Not Using Conditional Fetching

```tsx
// Bad - fetches even without required params
const { data } = useSWR(`/api/posts?page=${page}`, authFetcher);

// Good - only fetches when page is valid
const { data } = useSWR(page ? `/api/posts?page=${page}` : null, authFetcher);
```

### 4. Fetching in useEffect

```tsx
// Bad - manual fetching
useEffect(() => {
  fetch("/api/posts").then(res => res.json()).then(setData);
}, []);

// Good - use SWR
const { data } = usePosts(1);
```

### 5. Not Resetting Mutation State

```tsx
// Bad - mutation state persists
await trigger(args);
onSuccess();

// Good - reset after success
await trigger(args);
onSuccess();
reset();
```
