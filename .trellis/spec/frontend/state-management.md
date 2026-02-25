# State Management

> How state is managed in this project.

---

## Overview

This project uses a hybrid state management approach:
- **Server state**: SWR for API data caching and synchronization
- **Global client state**: React Context for authentication and theme
- **Local state**: useState for component-specific state

---

## State Categories

| Category | Solution | Examples |
|----------|----------|----------|
| Server state | SWR | Posts, users, delivery channels |
| Auth state | React Context | User info, tokens, isAuthenticated |
| Theme state | next-themes | Dark/light mode |
| Form state | useState | Input values, form errors |
| UI state | useState | Modal visibility, loading states |

---

## Server State with SWR

### When to Use SWR

Use SWR for all data that comes from an API:
- Lists (posts, users, channels)
- Single resources
- User-specific data

### Data Fetching Pattern

```tsx
// hooks/swr/usePosts.ts
export function usePosts(page: number, limit: number = 20) {
  return useSWR<PostsPaginatedResponse>(
    page ? `/api/posts?page=${page}&limit=${limit}` : null,
    authFetcher
  );
}
```

### Cache Invalidation

Use `mutate` to refresh data after mutations:

```tsx
const { mutate } = usePosts(1, 20);
const { trigger } = useCreatePost();

await trigger(data);
mutate(); // Revalidate cache
```

---

## Global Client State

### Auth Context

User authentication state is managed via React Context:

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

### Provider Implementation

```tsx
// components/UserInfoProvider.tsx:6-48
export const UserInfoProvider = ({ children }: { children: JSX.Element }) => {
  const [userInfo, setUserInfo] = useState<UserInfo | null>(() => {
    try {
      const loginData = get<UserInfo>("login");
      return loginData || null;
    } catch (error) {
      console.error("Error reading login data from storage:", error);
      return null;
    }
  });

  const logout = () => {
    setUserInfo(null);
    remove("login");
  };

  const isAuthenticated = !!(
    userInfo?.user?.id &&
    userInfo?.access_token &&
    userInfo?.refresh_token
  );

  const isAdmin = () => {
    return userInfo?.user?.role === "admin";
  };

  return (
    <UserInfoContext.Provider value={{ isAuthenticated, userInfo, setUserInfo, logout, isAdmin }}>
      {children}
    </UserInfoContext.Provider>
  );
};
```

### Using Context

```tsx
// components/Layout.tsx:24
const { logout, userInfo, isAuthenticated, isAdmin } = useContext(UserInfoContext);
```

---

## Theme State

Theme is managed by `next-themes`:

```tsx
// contexts/ThemeProvider.tsx
export function ThemeProvider({ children }: { children: React.ReactNode }) {
  return (
    <NextThemesProvider
      attribute="class"
      defaultTheme="system"
      enableSystem
      disableTransitionOnChange
    >
      {children}
    </NextThemesProvider>
  );
}
```

---

## Local State

### Form State

Use `useState` for form inputs:

```tsx
// components/CreateTestPostModal.tsx:31-33
const [title, setTitle] = useState("");
const [body, setBody] = useState("");
const [error, setError] = useState<string>("");
```

### UI State

Use `useState` for UI-specific state:

```tsx
const [isOpen, setIsOpen] = useState(false);
const [isLoading, setIsLoading] = useState(false);
```

---

## When to Use Global State

Promote state to global (Context) when:

1. **Multiple components need access** - User info used across app
2. **State affects routing** - isAuthenticated determines protected routes
3. **State persists across navigation** - Theme preference

Keep state local when:

1. **Only one component uses it** - Form input values
2. **State is temporary** - Modal visibility
3. **State is derived** - Computed values from props

---

## Common Mistakes

### 1. Using Context for Server State

```tsx
// Bad - fetching in context, no caching
const [posts, setPosts] = useState([]);
useEffect(() => { fetchPosts().then(setPosts); }, []);

// Good - use SWR for server state
const { data: posts } = usePosts(1);
```

### 2. Prop Drilling Instead of Context

```tsx
// Bad
<App user={user} onLogout={logout}>
  <Layout user={user} onLogout={logout}>
    <Header user={user} onLogout={logout}>

// Good - use context
<UserInfoProvider>
  <App>
    <Layout>
      <Header>  {/* uses useContext */}
```

### 3. Not Persisting Auth State

```tsx
// Bad - state lost on refresh
const [user, setUser] = useState(null);

// Good - persist to localStorage
const [userInfo, setUserInfo] = useState<UserInfo | null>(() => {
  const loginData = get<UserInfo>("login");
  return loginData || null;
});
```

### 4. Storing Server State in Context

```tsx
// Bad - no caching, no revalidation
const [posts, setPosts] = useContext(PostsContext);

// Good - SWR handles caching
const { data: posts } = usePosts(1);
```
