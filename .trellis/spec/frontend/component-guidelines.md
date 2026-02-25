# Component Guidelines

> How components are built in this project.

---

## Overview

This project uses React with TypeScript and follows these conventions:
- **Function components only** - No class components
- **shadcn/ui primitives** - Use existing UI components from `components/ui/`
- **Tailwind CSS v4** - Styling with design tokens (`bg-background`, `text-foreground`)
- **Radix primitives** - Accessible component foundations

---

## Component Structure

### Standard Component Template

```tsx
// 1. Imports (grouped: React, external, internal, types)
import { useState, useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Loader2Icon } from "lucide-react";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

import { useCreateTestPost } from "../hooks/swr/useCreateTestPost";
import * as api from "../utils/api";

// 2. Props interface
interface CreateTestPostModalProps {
  show: boolean;
  postKey: string;
  onHide: () => void;
  onSuccess: () => void;
}

// 3. Component function
function CreateTestPostModal({ show, postKey, onHide, onSuccess }: CreateTestPostModalProps) {
  const { t } = useTranslation();
  const [title, setTitle] = useState("");

  // ... component logic

  return (
    // JSX
  );
}

// 4. Default export
export default CreateTestPostModal;
```

### Example: CreateTestPostModal.tsx:1-133

---

## Props Conventions

### Props Interface

Always define props interface above the component:

```tsx
interface CreateTestPostModalProps {
  show: boolean;           // Required props first
  postKey: string;
  onHide: () => void;      // Callback props
  onSuccess: () => void;
}
```

### Destructuring

Destructure props in function signature:

```tsx
// Good
function CreateTestPostModal({ show, postKey, onHide, onSuccess }: CreateTestPostModalProps) {

// Avoid
function CreateTestPostModal(props: CreateTestPostModalProps) {
  const { show, postKey } = props;
```

### Children Prop

Use `JSX.Element` for single child:

```tsx
// components/UserInfoProvider.tsx:6
export const UserInfoProvider = ({ children }: { children: JSX.Element }) => {
```

---

## Styling Patterns

### Design Tokens

Use semantic design tokens instead of raw colors:

```tsx
// Good
<header className="bg-background/80 backdrop-blur border-b">

// Avoid
<header className="bg-white/80 backdrop-blur border-b-gray-200">
```

### Common Token Classes

| Token | Usage |
|-------|-------|
| `bg-background` | Page background |
| `text-foreground` | Primary text |
| `text-muted-foreground` | Secondary text |
| `border-border` | Border color |
| `bg-card` | Card background |

### Responsive Design

Use Tailwind responsive prefixes:

```tsx
// components/Layout.tsx:52-55
<span className="hidden sm:inline">
  {userInfo?.user?.username || t("common.user")}
</span>
```

### Component Composition

Use shadcn/ui primitives for composition:

```tsx
// components/Layout.tsx:48-80
<DropdownMenu>
  <DropdownMenuTrigger asChild>
    <Button type="button" variant="ghost" className="gap-2">
      {/* ... */}
    </Button>
  </DropdownMenuTrigger>
  <DropdownMenuContent align="end">
    <DropdownMenuLabel>{/* ... */}</DropdownMenuLabel>
    <DropdownMenuSeparator />
    <DropdownMenuItem onClick={() => navigate("/settings")}>
      {/* ... */}
    </DropdownMenuItem>
  </DropdownMenuContent>
</DropdownMenu>
```

---

## Accessibility

### Form Labels

Always associate labels with form controls:

```tsx
// components/CreateTestPostModal.tsx:86-97
<Label htmlFor="test-post-title">{t("createTestPost.titleLabel")}</Label>
<Input
  id="test-post-title"
  ref={titleRef}
  value={title}
  onChange={(e) => setTitle(e.target.value)}
  // ...
/>
```

### Screen Reader Text

Use `sr-only` for screen-reader-only content:

```tsx
// components/CreateTestPostModal.tsx:74-76
<DialogDescription className="sr-only">
  {t("createTestPost.bodyLabel")}
</DialogDescription>
```

### Focus Management

Focus first interactive element on open:

```tsx
// components/CreateTestPostModal.tsx:34-47
const titleRef = useRef<HTMLInputElement | null>(null);

useEffect(() => {
  if (show) {
    setTimeout(() => titleRef.current?.focus(), 0);
  }
}, [show]);

// ...
<Input ref={titleRef} id="test-post-title" />
```

### Button Types

Always specify button type:

```tsx
<Button type="submit">Submit</Button>
<Button type="button" variant="outline">Cancel</Button>
```

---

## Common Mistakes

### 1. Using Class Components

```tsx
// Bad
class CreatePostModal extends React.Component { }

// Good
function CreatePostModal({ show, onHide }: Props) { }
```

### 2. Inline Styles

```tsx
// Bad
<div style={{ padding: "16px" }}>

// Good
<div className="p-4">
```

### 3. Missing Key in Lists

```tsx
// Bad
{items.map(item => <Item data={item} />)}

// Good
{items.map(item => <Item key={item.id} data={item} />)}
```

### 4. Not Using Design Tokens

```tsx
// Bad - hardcoded colors
<div className="text-gray-500">

// Good - semantic tokens
<div className="text-muted-foreground">
```

### 5. Missing Accessibility

```tsx
// Bad
<div onClick={handleClick}>Click me</div>

// Good
<Button type="button" onClick={handleClick}>Click me</Button>
```
