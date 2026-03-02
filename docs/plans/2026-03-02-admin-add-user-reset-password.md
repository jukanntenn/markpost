# Admin Add User and Reset Password Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add backend API and frontend UI for admin to create new users with username/password and reset existing user passwords (manual set or random generate).

**Architecture:**
- Backend: Add `CreateUser` and `ResetUserPassword` endpoints to admin handlers/services
- Frontend: Add `AddUserDialog` and `ResetPasswordDialog` components with two tabs (Set/Generate)
- Follow existing patterns in `admin.go`, `AdminUsers.tsx`, and shadcn/ui dialog components

**Tech Stack:** Go/Gin backend, React/TypeScript frontend, shadcn/ui components, SWR for data fetching

---

## Backend Tasks

### Task 1: Add Service Methods for CreateUser and ResetUserPassword

**Files:**
- Modify: `backend/services/admin.go`
- Test: `backend/services/admin_test.go`

**Step 1: Add interface methods to AdminServiceInterface**

Add to the interface in `handlers/admin.go`:
```go
type AdminServiceInterface interface {
    // ... existing methods ...
    CreateUser(username, password string) (*models.User, error)
    ResetUserPassword(id int, password string, generateRandom bool) (string, error)
}
```

**Step 2: Implement CreateUser service method**

In `backend/services/admin.go`, add:
```go
func (s *AdminService) CreateUser(username, password string) (*models.User, error) {
    user, err := s.users.CreateUser(username, password)
    if err != nil {
        return nil, NewServiceErrorWrap(ErrInternal, "create user failed", err)
    }
    return user, nil
}
```

**Step 3: Add GenerateRandomPassword utility function**

In `backend/utils/post_key.go`, add:
```go
func GenerateRandomPassword(length int) (string, error) {
    if length <= 0 {
        length = 12
    }
    // Include uppercase, lowercase, numbers for stronger passwords
    alphabet := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
    n := len(alphabet)
    out := make([]byte, length)
    buf := make([]byte, length)
    if _, err := rand.Read(buf); err != nil {
        return "", err
    }
    for i := 0; i < length; i++ {
        out[i] = alphabet[int(buf[i])%n]
    }
    return string(out), nil
}
```

**Step 4: Implement ResetUserPassword service method**

In `backend/services/admin.go`, add:
```go
func (s *AdminService) ResetUserPassword(id int, password string, generateRandom bool) (string, error) {
    _, err := s.users.GetUserByID(id)
    if err != nil {
        if errors.Is(err, models.ErrNotFound) {
            return "", NewServiceErrorWrap(ErrNotFound, fmt.Sprintf("user with ID %d not found", id), err)
        }
        return "", NewServiceErrorWrap(ErrInternal, fmt.Sprintf("get user with ID %d failed", id), err)
    }

    finalPassword := password
    if generateRandom {
        finalPassword, err = utils.GenerateRandomPassword(12)
        if err != nil {
            return "", NewServiceErrorWrap(ErrInternal, "generate random password failed", err)
        }
    }

    if err := s.users.SetUserPassword(id, finalPassword); err != nil {
        return "", NewServiceErrorWrap(ErrInternal, fmt.Sprintf("set password for user with ID %d failed", id), err)
    }

    return finalPassword, nil
}
```

**Step 4: Write unit tests**

In `backend/services/admin_test.go`, add tests for:
- `TestAdminService_CreateUser_Success`
- `TestAdminService_CreateUser_DuplicateUsername`
- `TestAdminService_ResetUserPassword_Manual`
- `TestAdminService_ResetUserPassword_Random`
- `TestAdminService_ResetUserPassword_NotFound`

**Step 5: Run tests**

```bash
cd backend
go test ./services/... -v -run TestAdminService
```

Expected: All tests pass

**Step 7: Commit**

```bash
git add backend/services/admin.go backend/services/admin_test.go
git commit -m "feat(admin): add CreateUser and ResetUserPassword service methods"
```

---

### Task 2: Add Handler Methods for CreateUser and ResetUserPassword

**Files:**
- Modify: `backend/handlers/admin.go`

**Step 1: Add request types**

```go
type createUserRequest struct {
    Username string `json:"username" binding:"required,min=1,max=50"`
    Password string `json:"password" binding:"required,min=6"`
}

type resetPasswordRequest struct {
    Password      string `json:"password"`
    GenerateRandom *bool `json:"generate_random"`
}
```

**Step 2: Add CreateUser handler**

```go
func (h *AdminHandler) CreateUser(c *gin.Context) {
    var req createUserRequest
    if !bindJSON(c, &req) {
        return
    }

    user, err := h.svc.CreateUser(req.Username, req.Password)
    if err != nil {
        apperrors.RespondError(c, err)
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "user": adminUserResponse{
            ID:        user.ID,
            Username:  user.Username,
            Role:      string(user.Role),
            GitHubID:  user.GitHubID,
            CreatedAt: user.CreatedAt.Format(timeFormatRFC3339),
            UpdatedAt: user.UpdatedAt.Format(timeFormatRFC3339),
        },
    })
}
```

**Step 3: Add ResetUserPassword handler**

```go
func (h *AdminHandler) ResetUserPassword(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil || id <= 0 {
        apperrors.RespondError(c, services.NewServiceErrorWithDetails(services.ErrValidation, "request validation failed", []services.ServiceError{
            {Code: services.ErrFieldViolation, Description: "id"},
        }))
        return
    }

    var req resetPasswordRequest
    if !bindJSON(c, &req) {
        return
    }

    generateRandom := req.GenerateRandom != nil && *req.GenerateRandom

    // If not generating random, password is required
    if !generateRandom && req.Password == "" {
        apperrors.RespondError(c, services.NewServiceErrorWithDetails(services.ErrValidation, "request validation failed", []services.ServiceError{
            {Code: services.ErrFieldViolation, Description: "password"},
        }))
        return
    }

    password, err := h.svc.ResetUserPassword(id, req.Password, generateRandom)
    if err != nil {
        apperrors.RespondError(c, err)
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "password": password,
    })
}
```

**Step 4: Update AdminServiceInterface**

Add the two new methods to the interface in `handlers/admin.go`.

**Step 5: Commit**

```bash
git add backend/handlers/admin.go
git commit -m "feat(admin): add CreateUser and ResetUserPassword handlers"
```

---

### Task 3: Register Admin Routes

**Files:**
- Modify: `backend/routes.go`

**Step 1: Add routes**

In the admin route group, add:
```go
admin.POST("/users", adminHandler.CreateUser)
admin.POST("/users/:id/reset-password", adminHandler.ResetUserPassword)
```

**Step 2: Run and verify routes compile**

```bash
cd backend
go build -o markpost .
```

Expected: Build succeeds

**Step 3: Regenerate Swagger docs**

```bash
cd backend
swag init -g main.go -o docs --parseDependency --parseInternal
```

**Step 4: Commit**

```bash
git add backend/routes.go backend/docs
git commit -m "feat(admin): register CreateUser and ResetUserPassword routes"
```

---

## Frontend Tasks

### Task 4: Create AddUserDialog Component

**Files:**
- Create: `frontend/src/components/admin/AddUserDialog.tsx`

**Step 1: Create the component**

```tsx
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Loader2Icon } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { auth, getErrorMessage } from "@/utils/api";

interface AddUserDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export default function AddUserDialog({ open, onOpenChange, onSuccess }: AddUserDialogProps) {
  const { t } = useTranslation();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async () => {
    if (!username.trim()) {
      toast.error(t("admin.users.usernameRequired"));
      return;
    }
    if (password.length < 6) {
      toast.error(t("admin.users.passwordMinLength"));
      return;
    }
    if (password !== confirmPassword) {
      toast.error(t("admin.users.passwordsNotMatch"));
      return;
    }

    try {
      setIsSubmitting(true);
      await auth.post("/api/admin/users", { username: username.trim(), password });
      toast.success(t("admin.users.userCreated"));
      setUsername("");
      setPassword("");
      setConfirmPassword("");
      onOpenChange(false);
      onSuccess();
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t("admin.errors.createFailed")));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(next) => (!isSubmitting ? onOpenChange(next) : undefined)}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("admin.users.addUser")}</DialogTitle>
          <DialogDescription>{t("admin.users.addUserDescription")}</DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="username">{t("admin.username")}</Label>
            <Input
              id="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder={t("admin.users.usernamePlaceholder")}
              disabled={isSubmitting}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="password">{t("login.password")}</Label>
            <Input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder={t("admin.users.passwordPlaceholder")}
              disabled={isSubmitting}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="confirmPassword">{t("settings.confirmPassword")}</Label>
            <Input
              id="confirmPassword"
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              placeholder={t("admin.users.confirmPasswordPlaceholder")}
              disabled={isSubmitting}
            />
          </div>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={isSubmitting}>
            {t("common.cancel")}
          </Button>
          <Button type="button" onClick={handleSubmit} disabled={isSubmitting}>
            {isSubmitting ? (
              <span className="inline-flex items-center gap-2">
                <Loader2Icon className="size-4 animate-spin" />
                {t("common.processing")}
              </span>
            ) : (
              t("common.create")
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
```

**Step 2: Commit**

```bash
git add frontend/src/components/admin/AddUserDialog.tsx
git commit -m "feat(admin): add AddUserDialog component"
```

---

### Task 5: Create ResetPasswordDialog Component with Tabs

**Files:**
- Create: `frontend/src/components/admin/ResetPasswordDialog.tsx`

**Step 1: Create the component with tabs**

```tsx
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Loader2Icon, Copy, Check } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { auth, getErrorMessage } from "@/utils/api";

interface ResetPasswordDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  userId: number;
  username: string;
  onSuccess: () => void;
}

export default function ResetPasswordDialog({ open, onOpenChange, userId, username, onSuccess }: ResetPasswordDialogProps) {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<"manual" | "random">("manual");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [generatedPassword, setGeneratedPassword] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [copied, setCopied] = useState(false);

  const handleSubmitManual = async () => {
    if (password.length < 6) {
      toast.error(t("admin.users.passwordMinLength"));
      return;
    }
    if (password !== confirmPassword) {
      toast.error(t("admin.users.passwordsNotMatch"));
      return;
    }

    try {
      setIsSubmitting(true);
      await auth.post(`/api/admin/users/${userId}/reset-password`, { password });
      toast.success(t("admin.users.passwordResetSuccess"));
      setPassword("");
      setConfirmPassword("");
      onOpenChange(false);
      onSuccess();
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t("admin.errors.updateFailed")));
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleGenerateRandom = async () => {
    try {
      setIsSubmitting(true);
      const { data } = await auth.post(`/api/admin/users/${userId}/reset-password`, { generate_random: true });
      setGeneratedPassword(data.password);
      toast.success(t("admin.users.passwordGenerated"));
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t("admin.errors.updateFailed")));
    } finally {
      setIsSubmitting(false);
    }
  };

  const copyPassword = () => {
    navigator.clipboard.writeText(generatedPassword);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
    toast.success(t("admin.users.passwordCopied"));
  };

  return (
    <Dialog open={open} onOpenChange={(next) => (!isSubmitting ? onOpenChange(next) : undefined)}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("admin.users.resetPassword")}</DialogTitle>
          <DialogDescription>
            {t("admin.users.resetPasswordFor", { username })}
          </DialogDescription>
        </DialogHeader>
        <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as "manual" | "random")} className="w-full">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="manual">{t("admin.users.setPassword")}</TabsTrigger>
            <TabsTrigger value="random">{t("admin.users.generateRandom")}</TabsTrigger>
          </TabsList>
          <TabsContent value="manual" className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="new-password">{t("settings.newPassword")}</Label>
              <Input
                id="new-password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder={t("admin.users.passwordPlaceholder")}
                disabled={isSubmitting}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="confirm-new-password">{t("settings.confirmPassword")}</Label>
              <Input
                id="confirm-new-password"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder={t("admin.users.confirmPasswordPlaceholder")}
                disabled={isSubmitting}
              />
            </div>
          </TabsContent>
          <TabsContent value="random" className="space-y-4 py-4">
            <p className="text-sm text-muted-foreground">{t("admin.users.generateRandomDescription")}</p>
            {generatedPassword ? (
              <div className="flex items-center gap-2">
                <Input value={generatedPassword} readOnly className="font-mono" />
                <Button type="button" variant="outline" size="icon" onClick={copyPassword}>
                  {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
                </Button>
              </div>
            ) : null}
          </TabsContent>
        </Tabs>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={isSubmitting}>
            {t("common.cancel")}
          </Button>
          {activeTab === "manual" ? (
            <Button type="button" onClick={handleSubmitManual} disabled={isSubmitting}>
              {isSubmitting ? (
                <span className="inline-flex items-center gap-2">
                  <Loader2Icon className="size-4 animate-spin" />
                  {t("common.processing")}
                </span>
              ) : (
                t("common.confirm")
              )}
            </Button>
          ) : (
            <Button
              type="button"
              onClick={handleGenerateRandom}
              disabled={isSubmitting || generatedPassword !== ""}
            >
              {isSubmitting ? (
                <span className="inline-flex items-center gap-2">
                  <Loader2Icon className="size-4 animate-spin" />
                  {t("common.processing")}
                </span>
              ) : generatedPassword ? (
                t("common.done")
              ) : (
                t("admin.users.generate")
              )}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
```

**Step 2: Commit**

```bash
git add frontend/src/components/admin/ResetPasswordDialog.tsx
git commit -m "feat(admin): add ResetPasswordDialog component with tabs"
```

---

### Task 6: Add i18n Translations

**Files:**
- Modify: `frontend/src/i18n/locales/en.json`
- Modify: `frontend/src/i18n/locales/zh.json`

**Step 1: Add English translations**

In `admin.users` section, add:
```json
{
  "addUser": "Add User",
  "addUserDescription": "Create a new user account with username and password",
  "usernamePlaceholder": "Enter username",
  "passwordPlaceholder": "Enter password (min 6 characters)",
  "confirmPasswordPlaceholder": "Confirm password",
  "passwordsNotMatch": "Passwords do not match",
  "passwordMinLength": "Password must be at least 6 characters",
  "usernameRequired": "Username is required",
  "userCreated": "User created successfully",
  "resetPassword": "Reset Password",
  "resetPasswordFor": "Reset password for {{username}}",
  "setPassword": "Set Password",
  "generateRandom": "Generate Random",
  "generateRandomDescription": "Generate a secure random password",
  "generate": "Generate",
  "passwordResetSuccess": "Password reset successfully",
  "passwordGenerated": "Password generated",
  "passwordCopied": "Password copied to clipboard",
  "createFailed": "Failed to create user"
}
```

**Step 2: Add Chinese translations**

Same keys with Chinese values.

**Step 3: Commit**

```bash
git add frontend/src/i18n/locales/en.json frontend/src/i18n/locales/zh.json
git commit -m "feat(i18n): add admin user management translations"
```

---

### Task 7: Integrate Dialogs into AdminUsers Page

**Files:**
- Modify: `frontend/src/pages/AdminUsers.tsx`

**Step 1: Import new components**

```tsx
import AddUserDialog from "@/components/admin/AddUserDialog";
import ResetPasswordDialog from "@/components/admin/ResetPasswordDialog";
```

**Step 2: Add state for dialogs**

```tsx
const [addUserOpen, setAddUserOpen] = useState(false);
const [resetPasswordOpen, setResetPasswordOpen] = useState(false);
const [resetPasswordUser, setResetPasswordUser] = useState<User | null>(null);
```

**Step 3: Add Add User button**

In the CardHeader section, add:
```tsx
<CardHeader className="px-0">
  <div className="flex items-center justify-between">
    <CardTitle>{t("admin.users.title")}</CardTitle>
    <Button type="button" onClick={() => setAddUserOpen(true)}>
      {t("admin.users.addUser")}
    </Button>
  </div>
</CardHeader>
```

**Step 4: Add Reset Password button to actions**

In `renderActions`:
```tsx
const renderActions = (user: User) => {
  const disabled = !canModifyUser(user) || roleSubmitting;
  return (
    <div className="flex items-center justify-end gap-2">
      <Button type="button" variant="outline" size="sm" onClick={() => openRoleDialog(user)} disabled={disabled}>
        {t("admin.users.changeRole")}
      </Button>
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => {
          setResetPasswordUser(user);
          setResetPasswordOpen(true);
        }}
        disabled={disabled}
      >
        {t("admin.users.resetPassword")}
      </Button>
      <Button
        type="button"
        variant="destructive"
        size="sm"
        onClick={() => openDeleteDialog(user)}
        disabled={disabled}
      >
        {t("admin.users.delete")}
      </Button>
    </div>
  );
};
```

**Step 5: Add dialog components at bottom of render**

```tsx
return (
  <>
    {/* ... existing content ... */}

    <AddUserDialog
      open={addUserOpen}
      onOpenChange={setAddUserOpen}
      onSuccess={() => mutate()}
    />

    <ResetPasswordDialog
      open={resetPasswordOpen}
      onOpenChange={setResetPasswordOpen}
      userId={resetPasswordUser?.id ?? 0}
      username={resetPasswordUser?.username ?? ""}
      onSuccess={() => {
        setResetPasswordUser(null);
        mutate();
      }}
    />
  </>
);
```

**Step 6: Run type check and lint**

```bash
cd frontend
pnpm typecheck
pnpm lint
```

Expected: No errors

**Step 7: Commit**

```bash
git add frontend/src/pages/AdminUsers.tsx
git commit -m "feat(admin): integrate AddUserDialog and ResetPasswordDialog"
```

---

## Testing Tasks

### Task 8: Backend Integration Testing

**Files:**
- Modify: `backend/services/admin_test.go` (if needed)

**Step 1: Run all backend tests**

```bash
cd backend
go test ./... -v
```

Expected: All tests pass

**Step 2: Manual API testing (optional)**

Test endpoints with curl or Postman:
```bash
# Create user
curl -X POST http://localhost:8080/api/admin/users \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"password123"}'

# Reset password (manual)
curl -X POST http://localhost:8080/api/admin/users/1/reset-password \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"password":"newpassword123"}'

# Reset password (random)
curl -X POST http://localhost:8080/api/admin/users/1/reset-password \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"generate_random":true}'
```

---

### Task 9: Frontend Manual Testing

**Step 1: Start development server**

```bash
cd frontend
pnpm dev
```

**Step 2: Manual testing checklist**

- [ ] Login as admin
- [ ] Navigate to Admin > Users
- [ ] Click "Add User" button
- [ ] Fill in username, password, confirm password
- [ ] Submit and verify user is created
- [ ] Try duplicate username (should fail)
- [ ] Try password mismatch (should fail)
- [ ] Try password < 6 chars (should fail)
- [ ] Click "Reset Password" on a user
- [ ] Test "Set Password" tab
- [ ] Test "Generate Random" tab
- [ ] Verify password is copied to clipboard
- [ ] Verify user list refreshes after operations

---

## Summary

**Total Tasks:** 9
**Estimated Time:** 2-3 hours

**Files Created:**
- `frontend/src/components/admin/AddUserDialog.tsx`
- `frontend/src/components/admin/ResetPasswordDialog.tsx`

**Files Modified:**
- `backend/services/admin.go`
- `backend/services/admin_test.go`
- `backend/handlers/admin.go`
- `backend/routes.go`
- `frontend/src/pages/AdminUsers.tsx`
- `frontend/src/i18n/locales/en.json`
- `frontend/src/i18n/locales/zh.json`

**API Endpoints Added:**
- `POST /api/admin/users` - Create new user
- `POST /api/admin/users/:id/reset-password` - Reset user password
