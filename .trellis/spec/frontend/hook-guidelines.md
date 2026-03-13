# Hook Guidelines

> How hooks are used in this project.

---

## Overview

This project uses **TanStack Query** for data fetching. Custom hooks follow these conventions:
- Data fetching hooks use `useQuery` or `useMutation`
- Hooks are organized in `hooks/`
- Return types are explicitly typed

---

## Custom Hook Patterns

### Data Fetching Hook

```tsx
// hooks/usePosts.ts
import { useQuery } from "@tanstack/react-query";
import type { PostsResponse } from "@/types/posts";

export function usePosts() {
  return useQuery<PostsResponse>({
    queryKey: ["posts"],
    queryFn: async () => {
      const res = await fetch("/api/v1/posts");
      if (!res.ok) throw new Error("Failed to fetch posts");
      return res.json();
    },
  });
}
```

### Mutation Hook

```tsx
// hooks/useCreatePost.ts
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function useCreatePost() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (data: CreatePostInput) => {
      const res = await fetch("/api/v1/posts", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
      });
      if (!res.ok) throw new Error("Failed to create post");
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["posts"] });
    },
  });
}
```

### Using Hooks in Components

```tsx
// components/PostsPage.tsx
export function PostsPage() {
  const { data, isLoading, error } = usePosts();
  const { mutate, isPending } = useCreatePost();

  if (isLoading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <div>
      {data?.posts.map(post => <PostCard key={post.id} post={post} />)}
    </div>
  );
}
```

---

## Naming Conventions

| Type | Pattern | Example |
|------|---------|---------|
| Query hook | `use<Resource>` | `usePosts`, `useUser` |
| Mutation hook | `use<Create/Update/Delete><Resource>` | `useCreatePost` |
| Params hook | `use<Param>` | `usePostKey` |

---

## Return Values

### useQuery Return

```tsx
const { data, error, isLoading, isError, refetch } = usePosts();
```

| Property | Type | Description |
|----------|------|-------------|
| `data` | `T \| undefined` | Fetched data |
| `error` | `Error \| null` | Fetch error |
| `isLoading` | `boolean` | Initial load |
| `isError` | `boolean` | Error state |
| `refetch` | `function` | Manual refetch |

### useMutation Return

```tsx
const { mutate, isPending, isError, error, reset } = useCreatePost();
```

| Property | Type | Description |
|----------|------|-------------|
| `mutate` | `function` | Execute mutation |
| `isPending` | `boolean` | Mutation in progress |
| `isError` | `boolean` | Error state |
| `error` | `Error \| null` | Error object |
| `reset` | `function` | Reset state |

---

## Common Mistakes

### Not Typing the Response

```tsx
// Bad
const { data } = useQuery({ queryKey: ["posts"], queryFn: fetchPosts });

// Good
const { data } = useQuery<PostsResponse>({ queryKey: ["posts"], queryFn: fetchPosts });
```

### Not Handling Loading State

```tsx
// Bad
return <div>{data.items}</div>;

// Good
if (isLoading) return <div>Loading...</div>;
if (error) return <div>Error: {error.message}</div>;
return <div>{data?.items}</div>;
```

### Fetching in useEffect

```tsx
// Bad
useEffect(() => {
  fetch("/api/posts").then(res => res.json()).then(setData);
}, []);

// Good
const { data } = usePosts();
```

### Not Invalidating Cache

```tsx
// Bad - cache not updated after mutation
const { mutate } = useMutation({ mutationFn: createPost });

// Good - invalidate on success
const { mutate } = useMutation({
  mutationFn: createPost,
  onSuccess: () => queryClient.invalidateQueries({ queryKey: ["posts"] }),
});
```
