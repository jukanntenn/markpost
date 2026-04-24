# Frontend Components & Hooks

> React component patterns and TanStack Query hook conventions.

## Component Rules

- **Function components only** — no class components
- **shadcn/ui primitives** — use existing components from `components/ui/`
- **Tailwind CSS v4** — style with design tokens, not raw colors
- **Radix primitives** — accessible component foundations

## Component Structure

1. Imports (grouped: React, external, internal, types)
2. Props interface (defined above the component)
3. Component function (destructure props in signature)
4. Default export

## Props Conventions

- Always define a `Props` interface above the component
- Destructure props in the function signature
- For children: `{ children }: { children: JSX.Element }`

## Styling

### Design Tokens

| Token | Usage |
|-------|-------|
| `bg-background` | Page background |
| `text-foreground` | Primary text |
| `text-muted-foreground` | Secondary text |
| `border-border` | Border color |
| `bg-card` | Card background |

- Use semantic tokens (`text-muted-foreground`) instead of raw colors (`text-gray-500`)
- Use Tailwind responsive prefixes (`hidden sm:inline`)
- Compose with shadcn/ui primitives (`DropdownMenu`, `Dialog`, etc.)

## Accessibility

- Always associate `<Label>` with form controls via `htmlFor`/`id`
- Use `sr-only` class for screen-reader-only content
- Manage focus on modal open (focus first interactive element)
- Always specify `type` attribute on `<Button>` (`"submit"` or `"button"`)
- Always add `key` prop to list items
- Use semantic HTML elements and ARIA attributes

## Hook Conventions

### Naming

| Type | Pattern | Example |
|------|---------|---------|
| Query hook | `use<Resource>` | `usePosts`, `useUser` |
| Mutation hook | `use<Create/Update/Delete><Resource>` | `useCreatePost` |
| Params hook | `use<Param>` | `usePostKey` |

### Query Hook

- Define in `hooks/` with explicit generic type
- Handle loading and error states in the consuming component
- Never fetch in `useEffect` — always use `useQuery`

```tsx
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

- Always invalidate relevant query keys on success
- Use `isPending` (not `isLoading`) for mutations

```tsx
export function useCreatePost() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: CreatePostInput) => { ... },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["posts"] }),
  });
}
```

### Return Values

| useQuery Property | Type | Description |
|-------------------|------|-------------|
| `data` | `T \| undefined` | Fetched data |
| `error` | `Error \| null` | Fetch error |
| `isLoading` | `boolean` | Initial load |
| `isError` | `boolean` | Error state |
| `refetch` | `function` | Manual refetch |

| useMutation Property | Type | Description |
|----------------------|------|-------------|
| `mutate` | `function` | Execute mutation |
| `isPending` | `boolean` | Mutation in progress |
| `isError` | `boolean` | Error state |
| `error` | `Error \| null` | Error object |
| `reset` | `function` | Reset state |

## Common Mistakes

### Components

1. **Class components** — use function components
2. **Inline styles** — use Tailwind classes (`p-4` not `style={{ padding: "16px" }}`)
3. **Hardcoded colors** — use design tokens (`text-muted-foreground` not `text-gray-500`)
4. **Missing key in lists** — always add `key={item.id}` to mapped elements
5. **Missing accessibility** — use `<Button>` instead of `<div onClick>`; add labels to inputs

### Hooks

6. **Not typing the response** — use `useQuery<PostsResponse>` not untyped `useQuery`
7. **Not handling loading state** — check `isLoading` and `error` before rendering data
8. **Fetching in useEffect** — use `useQuery` instead
9. **Not invalidating cache** — call `queryClient.invalidateQueries` on mutation success
