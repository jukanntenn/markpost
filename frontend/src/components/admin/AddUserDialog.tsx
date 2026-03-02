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
  onSuccess: () => void | Promise<void>;
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
      await onSuccess();
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
