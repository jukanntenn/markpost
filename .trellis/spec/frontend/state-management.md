# State Management

> How state is managed in this project.

---

## Overview

Hybrid state management approach:
- **Server state**: TanStack Query for API data
- **Global client state**: Zustand for auth, React Context for user info
- **Local state**: useState for component-specific state

---

## State Categories

| Category | Solution | Examples |
|----------|----------|----------|
| Server state | TanStack Query | Posts, users, settings |
| Auth state | Zustand + localStorage | User info, tokens |
| Theme state | next-themes | Dark/light mode |
| Form state | useState | Input values, form errors |
| UI state | useState | Modal visibility, loading |

---

## Server State with TanStack Query

### Data Fetching Hook

```tsx
// hooks/usePosts.ts
import { useQuery } from "@tanstack/react-query";

export function usePosts() {
  return useQuery({
    queryKey: ["posts"],
    queryFn: async () => {
      const res = await fetch("/api/posts");
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
      const res = await fetch("/api/posts", {
        method: "POST",
        body: JSON.stringify(data),
      });
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["posts"] });
    },
  });
}
```

---

## Global Client State

### Auth State (Zustand)

```tsx
// stores/auth.ts
import { create } from "zustand";
import { persist } from "zustand/middleware";

interface AuthState {
  token: string | null;
  user: User | null;
  setAuth: (token: string, user: User) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      token: null,
      user: null,
      setAuth: (token, user) => set({ token, user }),
      logout: () => set({ token: null, user: null }),
    }),
    { name: "auth" }
  )
);
```

### User Info Context

```tsx
// components/UserInfoContext.ts
export interface UserInfoContextType {
  isAuthenticated: boolean;
  userInfo: UserInfo | null;
  setUserInfo: (info: UserInfo | null) => void;
  logout: () => void;
  isAdmin: () => boolean;
}
```

---

## When to Use Global State

Promote to global when:
1. Multiple components need access
2. State affects routing
3. State persists across navigation

Keep local when:
1. Only one component uses it
2. State is temporary
3. State is derived from props

---

## Common Mistakes

### Using Context for Server State

```tsx
// Bad
const [posts, setPosts] = useState([]);
useEffect(() => { fetchPosts().then(setPosts); }, []);

// Good
const { data: posts } = usePosts();
```

### Not Persisting Auth State

```tsx
// Bad - lost on refresh
const [user, setUser] = useState(null);

// Good - Zustand persist
useAuthStore with persist middleware
```
