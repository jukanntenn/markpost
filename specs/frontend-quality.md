# Frontend Quality

> Code quality, unit testing, and accessibility standards.

## Forbidden Patterns

1. **Class components** — use function components only
2. **Inline styles** — use Tailwind classes (`p-4` not `style={{ padding: "16px" }}`)
3. **Hardcoded colors** — use design tokens (`text-muted-foreground` not `text-gray-500`)
4. **useEffect for data fetching** — use TanStack Query (`usePosts()`) instead
5. **Prop drilling** — use Context or Zustand for shared state

## Required Patterns

1. **TypeScript types for props** — define interface above component
2. **Accessible form controls** — `<Label htmlFor={id}>` + `<Input id={id}>`
3. **Button type attributes** — always specify `type="submit"` or `type="button"`
4. **Key prop in lists** — `{items.map(item => <Item key={item.id} />)}`
5. **Error/loading state handling** — check `isLoading` and `error` before rendering data

## Testing

### File Organization

Tests alongside source files:

```
frontend/src/
├── components/
│   ├── CreateTestPostModal.tsx
│   └── CreateTestPostModal.test.tsx
├── hooks/
│   ├── usePosts.ts
│   └── usePosts.test.ts
└── utils/
    ├── storage.ts
    └── storage.test.ts
```

### Test Structure

Use `describe`/`it` pattern with Vitest:

```tsx
describe("usePosts", () => {
  beforeEach(() => { /* setup mocks */ });

  it("fetches posts successfully", async () => {
    const { result } = renderHook(() => usePosts(1, 20), { wrapper: createWrapper() });
    await waitFor(() => { expect(result.current.data).toEqual({ posts: [...] }); });
  });
});
```

### Test Utilities

Shared wrapper with providers:

```tsx
export function createWrapper() {
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>{children}</I18nextProvider>
    </QueryClientProvider>
  );
}
```

### MSW for API Mocking

```tsx
// mocks/handlers.ts
export const handlers = [
  http.get("/api/posts", () => {
    return HttpResponse.json({ posts: [{ id: "p1", title: "Test Post" }] });
  }),
];
```

### Running Tests

```bash
pnpm test          # Watch mode
pnpm test:run      # Single run
pnpm test:e2e      # E2E tests (Playwright)
```

## Code Review Checklist

- [ ] Component is a function component, not a class
- [ ] Props interface is defined above component
- [ ] Uses design tokens instead of hardcoded colors
- [ ] Form inputs have associated labels
- [ ] Buttons have explicit `type` attribute
- [ ] Lists have `key` prop on items
- [ ] Uses TanStack Query for data fetching (not useEffect)
- [ ] Handles loading and error states
- [ ] No `any` types
- [ ] Tests cover main use cases
- [ ] No hardcoded text — uses i18n

## Accessibility Checklist

- [ ] All form inputs have labels
- [ ] Interactive elements are focusable
- [ ] Focus is managed on modals/dialogs
- [ ] Color is not the only way to convey information
- [ ] Loading states are announced to screen readers
- [ ] Error messages are associated with form fields

## Running Quality Checks

```bash
pnpm lint       # ESLint
pnpm build      # Type check + production build
pnpm test:run   # Unit tests
```
