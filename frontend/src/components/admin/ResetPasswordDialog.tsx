import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { CheckIcon, CopyIcon, Loader2Icon } from "lucide-react";

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
  onSuccess: () => void | Promise<void>;
}

type TabKey = "manual" | "random";

export default function ResetPasswordDialog({
  open,
  onOpenChange,
  userId,
  username,
  onSuccess,
}: ResetPasswordDialogProps) {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<TabKey>("manual");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [generatedPassword, setGeneratedPassword] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!open) {
      setActiveTab("manual");
      setPassword("");
      setConfirmPassword("");
      setGeneratedPassword("");
      setCopied(false);
    }
  }, [open]);

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
      onOpenChange(false);
      await onSuccess();
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
      await onSuccess();
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t("admin.errors.updateFailed")));
    } finally {
      setIsSubmitting(false);
    }
  };

  const copyPassword = async () => {
    try {
      await navigator.clipboard.writeText(generatedPassword);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
      toast.success(t("admin.users.passwordCopied"));
    } catch {
      toast.error(t("admin.errors.updateFailed"));
    }
  };

  return (
    <Dialog open={open} onOpenChange={(next) => (!isSubmitting ? onOpenChange(next) : undefined)}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("admin.users.resetPassword")}</DialogTitle>
          <DialogDescription>{t("admin.users.resetPasswordFor", { username })}</DialogDescription>
        </DialogHeader>

        <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as TabKey)} className="w-full">
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
                <Button type="button" variant="outline" size="icon" onClick={copyPassword} disabled={!generatedPassword}>
                  {copied ? <CheckIcon className="size-4" /> : <CopyIcon className="size-4" />}
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
            <Button type="button" onClick={handleGenerateRandom} disabled={isSubmitting || generatedPassword !== ""}>
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
