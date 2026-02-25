# Quality Guidelines

> Code quality standards for frontend development.

---

## Overview

This project follows React and TypeScript best practices with emphasis on:
- Function components with hooks (no class components)
- Comprehensive testing with Vitest and Testing Library
- Accessibility-first component design
- Self-documenting code without excessive comments

---

## Forbidden Patterns

### 1. Class Components

```tsx
// Bad
class CreatePostModal extends React.Component {
  render() { return <div>...</div>; }
}

// Good
function CreatePostModal({ show, onHide }: Props) {
  return <div>...</div>;
}
```

### 2. Inline Styles

```tsx
// Bad
<div style={{ padding: "16px", backgroundColor: "red" }}>

// Good
<div className="p-4 bg-destructive">
```

### 3. Hardcoded Colors

```tsx
// Bad
<div className="text-gray-500 bg-white">

// Good
<div className="text-muted-foreground bg-background">
```

### 4. useEffect for Data Fetching

```tsx
// Bad - manual fetching, no caching
useEffect(() => {
  fetch("/api/posts")
    .then(res => res.json())
    .then(setData);
}, []);

// Good - use SWR
const { data } = usePosts(1);
```

### 5. Prop Drilling

```tsx
// Bad
<App user={user}>
  <Layout user={user}>
    <Header user={user}>

// Good - use context
const { userInfo } = useContext(UserInfoContext);
```

---

## Required Patterns

### 1. TypeScript Types for Props

```tsx
// Good
interface CreateTestPostModalProps {
  show: boolean;
  postKey: string;
  onHide: () => void;
  onSuccess: () => void;
}

function CreateTestPostModal({ show, postKey, onHide, onSuccess }: CreateTestPostModalProps) {
```

### 2. Accessible Form Controls

```tsx
// Good
<Label htmlFor="test-post-title">{t("createTestPost.titleLabel")}</Label>
<Input
  id="test-post-title"
  value={title}
  onChange={(e) => setTitle(e.target.value)}
/>
```

### 3. Button Type Attributes

```tsx
// Good
<Button type="submit">Submit</Button>
<Button type="button" variant="outline" onClick={onHide}>Cancel</Button>
```

### 4. Key Prop in Lists

```tsx
// Good
{posts.map((post) => (
  <PostCard key={post.id} post={post} />
))}
```

### 5. Error State Handling

```tsx
// Good
if (isLoading) return <div>Loading...</div>;
if (error) return <div>Error: {error.message}</div>;
return <div>{data?.items}</div>;
```

---

## Testing Requirements

### Test File Organization

Tests are placed alongside source files:

```
frontend/src/
├── components/
│   ├── CreateTestPostModal.tsx
│   └── CreateTestPostModal.test.tsx
├── hooks/swr/
│   ├── usePosts.ts
│   └── usePosts.test.ts
└── utils/
    ├── storage.ts
    └── storage.test.ts
```

### Test Structure

Use describe/it pattern with Vitest:

```tsx
// hooks/swr/usePosts.test.ts:8-31
describe("usePosts", () => {
  beforeEach(() => {
    setMockAuth({
      access_token: "test_token",
      refresh_token: "test_refresh",
      user: { id: 1, username: "testuser" },
    });
  });

  it("fetches posts successfully", async () => {
    const { result } = renderHook(() => usePosts(1, 20), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.data).toEqual({
        posts: [/* ... */],
        pagination: { /* ... */ },
      });
    });
  });
});
```

### Test Utilities

Use shared test utilities:

```tsx
// test/utils.tsx:36-50
export function createWrapper() {
  return ({ children }: { children: React.ReactNode }) => (
    <SWRConfig
      value={{
        dedupingInterval: 0,
        revalidateOnFocus: false,
      }}
    >
      <I18nextProvider i18n={i18n}>{children}</I18nextProvider>
    </SWRConfig>
  );
}
```

### MSW for API Mocking

```tsx
// mocks/handlers.ts
export const handlers = [
  http.get("/api/posts", () => {
    return HttpResponse.json({
      posts: [{ id: "p1", title: "Test Post" }],
      pagination: { page: 1, total: 1 },
    });
  }),
];
```

### Running Tests

```bash
pnpm test          # Run tests in watch mode
pnpm test:run      # Run tests once
pnpm test:ui       # Run tests with UI
```

---

## Code Review Checklist

- [ ] Component is a function component, not a class
- [ ] Props interface is defined above component
- [ ] Uses design tokens instead of hardcoded colors
- [ ] Form inputs have associated labels
- [ ] Buttons have explicit `type` attribute
- [ ] Lists have `key` prop on items
- [ ] Uses SWR for data fetching (not useEffect)
- [ ] Handles loading and error states
- [ ] No `any` types - use proper TypeScript
- [ ] Tests cover main use cases
- [ ] No hardcoded text - uses i18n

---

## Accessibility Checklist

- [ ] All form inputs have labels
- [ ] Interactive elements are focusable
- [ ] Focus is managed on modals/dialogs
- [ ] Color is not the only way to convey information
- [ ] Loading states are announced to screen readers
- [ ] Error messages are associated with form fields

---

## Running Quality Checks

```bash
# Lint
pnpm lint

# Type check
pnpm build

# Tests
pnpm test:run
```
