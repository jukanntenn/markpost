import { useState, useEffect } from "react";
import { CircleCheckIcon, InfoIcon, Loader2Icon, LockIcon, SettingsIcon } from "lucide-react";
import * as api from "../utils/api";

import { useChangePassword } from "../hooks/swr/useChangePassword";
import { useDeliveryChannels } from "../hooks/swr/useDeliveryChannels";
import { useCreateDeliveryChannel } from "../hooks/swr/useCreateDeliveryChannel";
import { useUpdateDeliveryChannel } from "../hooks/swr/useUpdateDeliveryChannel";
import { useDeleteDeliveryChannel } from "../hooks/swr/useDeleteDeliveryChannel";
import { useTranslation } from "react-i18next";
import LanguageToggle from "../components/LanguageToggle";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

interface PasswordChangeRequest {
  current_password: string;
  new_password: string;
  confirm_password: string;
}

function Settings() {
  const { t } = useTranslation();
  const [formData, setFormData] = useState<PasswordChangeRequest>({
    current_password: "",
    new_password: "",
    confirm_password: "",
  });
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [showValidation, setShowValidation] = useState(false);
  const [deliveryError, setDeliveryError] = useState("");
  const [deliverySuccess, setDeliverySuccess] = useState("");
  const [newChannelName, setNewChannelName] = useState("");
  const [newChannelWebhookURL, setNewChannelWebhookURL] = useState("");
  const [newChannelKeywords, setNewChannelKeywords] = useState("");
  const [editingChannelID, setEditingChannelID] = useState<number | null>(null);
  const [editChannelName, setEditChannelName] = useState("");
  const [editChannelWebhookURL, setEditChannelWebhookURL] = useState("");
  const [editChannelKeywords, setEditChannelKeywords] = useState("");

  const { trigger, isMutating, reset } = useChangePassword();
  const {
    data: deliveryChannelsData,
    error: deliveryChannelsLoadError,
    isLoading: isDeliveryChannelsLoading,
    mutate: mutateDeliveryChannels,
  } = useDeliveryChannels();
  const { trigger: createDeliveryChannel, isMutating: isCreatingDeliveryChannel } =
    useCreateDeliveryChannel();
  const { trigger: updateDeliveryChannel, isMutating: isUpdatingDeliveryChannel } =
    useUpdateDeliveryChannel();
  const { trigger: deleteDeliveryChannel, isMutating: isDeletingDeliveryChannel } =
    useDeleteDeliveryChannel();

  useEffect(() => {
    document.title = t("common.pageTitle.settings");
  }, [t]);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
    setError("");
    setSuccess("");
    setShowValidation(false);
  };

  const maskWebhookURL = (raw: string) => {
    try {
      const u = new URL(raw);
      const last = u.pathname.split("/").filter(Boolean).pop() ?? "";
      const tail = last.slice(-6);
      return `${u.host}/…${tail}`;
    } catch {
      return raw;
    }
  };

  const beginEditChannel = (channel: {
    id: number;
    name: string;
    webhook_url: string;
    keywords: string;
  }) => {
    setEditingChannelID(channel.id);
    setEditChannelName(channel.name ?? "");
    setEditChannelWebhookURL(channel.webhook_url ?? "");
    setEditChannelKeywords(channel.keywords ?? "");
    setDeliveryError("");
    setDeliverySuccess("");
  };

  const cancelEditChannel = () => {
    setEditingChannelID(null);
    setEditChannelName("");
    setEditChannelWebhookURL("");
    setEditChannelKeywords("");
  };

  const handleCreateChannel = async (e: React.FormEvent) => {
    e.preventDefault();
    setDeliveryError("");
    setDeliverySuccess("");

    try {
      await createDeliveryChannel({
        kind: "feishu",
        name: newChannelName,
        webhook_url: newChannelWebhookURL,
        keywords: newChannelKeywords,
        enabled: true,
      });
      setNewChannelName("");
      setNewChannelWebhookURL("");
      setNewChannelKeywords("");
      setDeliverySuccess(t("settings.deliveryChannelCreated"));
      await mutateDeliveryChannels();
    } catch (err: unknown) {
      setDeliveryError(api.getErrorMessage(err, t("settings.deliveryChannelCreateFailed")));
    }
  };

  const handleToggleChannel = async (id: number, enabled: boolean) => {
    setDeliveryError("");
    setDeliverySuccess("");

    try {
      await updateDeliveryChannel({ id, enabled });
      await mutateDeliveryChannels();
    } catch (err: unknown) {
      setDeliveryError(api.getErrorMessage(err, t("settings.deliveryChannelUpdateFailed")));
    }
  };

  const handleSaveChannel = async () => {
    if (editingChannelID == null) return;
    setDeliveryError("");
    setDeliverySuccess("");

    try {
      await updateDeliveryChannel({
        id: editingChannelID,
        name: editChannelName,
        webhook_url: editChannelWebhookURL,
        keywords: editChannelKeywords,
      });
      setDeliverySuccess(t("settings.deliveryChannelUpdated"));
      cancelEditChannel();
      await mutateDeliveryChannels();
    } catch (err: unknown) {
      setDeliveryError(api.getErrorMessage(err, t("settings.deliveryChannelUpdateFailed")));
    }
  };

  const handleDeleteChannel = async (id: number) => {
    setDeliveryError("");
    setDeliverySuccess("");

    try {
      await deleteDeliveryChannel({ id });
      setDeliverySuccess(t("settings.deliveryChannelDeleted"));
      await mutateDeliveryChannels();
    } catch (err: unknown) {
      setDeliveryError(api.getErrorMessage(err, t("settings.deliveryChannelDeleteFailed")));
    }
  };

  const validatePassword = (password: string) => {
    return password.length >= 6;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    setShowValidation(true);

    if (!validatePassword(formData.new_password)) {
      setError(t("settings.passwordMinLength"));
      return;
    }

    if (formData.new_password !== formData.confirm_password) {
      setError(t("settings.passwordsNotMatch"));
      return;
    }

    if (formData.current_password === formData.new_password) {
      setError(t("settings.passwordSameAsCurrent"));
      return;
    }

    try {
      await trigger({
        current_password: formData.current_password,
        new_password: formData.new_password,
      });

      setSuccess(t("settings.passwordChangeSuccess"));

      setFormData({
        current_password: "",
        new_password: "",
        confirm_password: "",
      });
      setShowValidation(false);
      reset();
    } catch (err: unknown) {
      console.error("Password change failed:", err);
      setError(api.getErrorMessage(err, t("settings.passwordChangeFailed")));
    }
  };

  const passwordValid = validatePassword(formData.new_password);
  const passwordsMatch =
    formData.new_password === formData.confirm_password &&
    formData.new_password.length > 0;

  return (
    <div className="mx-auto max-w-xl space-y-6">
      <Card>
        <CardHeader className="flex-row items-center justify-between space-y-0">
          <div className="flex items-center gap-2">
            <SettingsIcon className="size-4" />
            <CardTitle className="text-base">{t("settings.applicationSettings")}</CardTitle>
          </div>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium text-muted-foreground">
                {t("settings.language")}
              </span>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    className="size-8"
                    aria-label={t("settings.languageDescription")}
                  >
                    <InfoIcon className="size-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{t("settings.languageDescription")}</TooltipContent>
              </Tooltip>
            </div>
            <LanguageToggle />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex-row items-center justify-between space-y-0">
          <CardTitle className="text-base">{t("settings.deliveryChannels")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {deliveryChannelsLoadError && (
            <Alert variant="destructive">
              <AlertDescription>
                {api.getErrorMessage(
                  deliveryChannelsLoadError,
                  t("settings.deliveryChannelsLoadFailed")
                )}
              </AlertDescription>
            </Alert>
          )}
          {deliveryError && (
            <Alert variant="destructive">
              <AlertDescription>{deliveryError}</AlertDescription>
            </Alert>
          )}
          {deliverySuccess && (
            <Alert>
              <CircleCheckIcon />
              <AlertDescription>{deliverySuccess}</AlertDescription>
            </Alert>
          )}

          <div className="space-y-2">
            <div className="text-sm font-medium text-muted-foreground">
              {t("settings.deliveryChannelsList")}
            </div>
            {isDeliveryChannelsLoading ? (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Loader2Icon className="size-4 animate-spin" />
                {t("settings.deliveryChannelsLoading")}
              </div>
            ) : (deliveryChannelsData?.channels?.length ?? 0) === 0 ? (
              <div className="text-sm text-muted-foreground">
                {t("settings.deliveryChannelsEmpty")}
              </div>
            ) : (
              <div className="space-y-3">
                {deliveryChannelsData?.channels?.map((ch) => (
                  <div key={ch.id} className="rounded-lg border p-4">
                    <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                      <div className="min-w-0 space-y-1">
                        <div className="truncate text-sm font-medium">
                          {ch.name?.trim()
                            ? ch.name
                            : t("settings.deliveryChannelUnnamed")}
                        </div>
                        <div className="text-xs text-muted-foreground">
                          {t("settings.deliveryChannelType")}: {ch.kind}
                        </div>
                        <div className="text-xs text-muted-foreground">
                          {t("settings.deliveryChannelWebhook")}:{" "}
                          {maskWebhookURL(ch.webhook_url)}
                        </div>
                        {ch.keywords?.trim() ? (
                          <div className="text-xs text-muted-foreground">
                            {t("settings.deliveryChannelKeywords")}: {ch.keywords}
                          </div>
                        ) : null}
                      </div>

                      <div className="flex shrink-0 flex-col items-start gap-3 sm:items-end">
                        <div className="flex items-center gap-2">
                          <Switch
                            checked={ch.enabled}
                            onCheckedChange={(checked) =>
                              handleToggleChannel(ch.id, checked)
                            }
                            disabled={
                              isUpdatingDeliveryChannel ||
                              isDeletingDeliveryChannel ||
                              isCreatingDeliveryChannel
                            }
                          />
                          <span className="text-sm text-muted-foreground">
                            {t("settings.deliveryChannelEnabled")}
                          </span>
                        </div>

                        <div className="flex gap-2">
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            onClick={() => beginEditChannel(ch)}
                            disabled={
                              editingChannelID !== null ||
                              isUpdatingDeliveryChannel ||
                              isDeletingDeliveryChannel ||
                              isCreatingDeliveryChannel
                            }
                          >
                            {t("settings.deliveryChannelEdit")}
                          </Button>
                          <Button
                            type="button"
                            variant="destructive"
                            size="sm"
                            onClick={() => handleDeleteChannel(ch.id)}
                            disabled={
                              editingChannelID !== null ||
                              isDeletingDeliveryChannel ||
                              isUpdatingDeliveryChannel ||
                              isCreatingDeliveryChannel
                            }
                          >
                            {t("settings.deliveryChannelDelete")}
                          </Button>
                        </div>
                      </div>
                    </div>

                    {editingChannelID === ch.id && (
                      <div className="mt-4 space-y-3 border-t pt-4">
                        <div className="space-y-2">
                          <Label htmlFor={`edit-channel-name-${ch.id}`}>
                            {t("settings.deliveryChannelName")}
                          </Label>
                          <Input
                            id={`edit-channel-name-${ch.id}`}
                            value={editChannelName}
                            onChange={(e) => setEditChannelName(e.target.value)}
                            placeholder={t("settings.deliveryChannelNamePlaceholder")}
                          />
                        </div>

                        <div className="space-y-2">
                          <Label htmlFor={`edit-channel-webhook-${ch.id}`}>
                            {t("settings.deliveryChannelWebhookURL")}
                          </Label>
                          <Input
                            id={`edit-channel-webhook-${ch.id}`}
                            value={editChannelWebhookURL}
                            onChange={(e) => setEditChannelWebhookURL(e.target.value)}
                            placeholder={t("settings.deliveryChannelWebhookPlaceholder")}
                          />
                        </div>

                        <div className="space-y-2">
                          <Label htmlFor={`edit-channel-keywords-${ch.id}`}>
                            {t("settings.deliveryChannelKeywords")}
                          </Label>
                          <Input
                            id={`edit-channel-keywords-${ch.id}`}
                            value={editChannelKeywords}
                            onChange={(e) => setEditChannelKeywords(e.target.value)}
                            placeholder={t("settings.deliveryChannelKeywordsPlaceholder")}
                          />
                          <p className="text-xs text-muted-foreground">
                            {t("settings.deliveryChannelKeywordsHelp")}
                          </p>
                        </div>

                        <div className="flex justify-end gap-2">
                          <Button
                            type="button"
                            variant="outline"
                            onClick={cancelEditChannel}
                            disabled={isUpdatingDeliveryChannel}
                          >
                            {t("settings.deliveryChannelCancel")}
                          </Button>
                          <Button
                            type="button"
                            onClick={handleSaveChannel}
                            disabled={
                              isUpdatingDeliveryChannel ||
                              editChannelWebhookURL.trim().length === 0
                            }
                          >
                            {isUpdatingDeliveryChannel ? (
                              <span className="inline-flex items-center gap-2">
                                <Loader2Icon className="size-4 animate-spin" />
                                {t("settings.deliveryChannelSaving")}
                              </span>
                            ) : (
                              t("settings.deliveryChannelSave")
                            )}
                          </Button>
                        </div>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="space-y-3 border-t pt-4">
            <div className="text-sm font-medium text-muted-foreground">
              {t("settings.deliveryChannelAdd")}
            </div>
            <form onSubmit={handleCreateChannel} className="space-y-3">
              <div className="space-y-2">
                <Label htmlFor="new-channel-name">
                  {t("settings.deliveryChannelName")}
                </Label>
                <Input
                  id="new-channel-name"
                  value={newChannelName}
                  onChange={(e) => setNewChannelName(e.target.value)}
                  placeholder={t("settings.deliveryChannelNamePlaceholder")}
                  disabled={isCreatingDeliveryChannel || editingChannelID !== null}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="new-channel-webhook">
                  {t("settings.deliveryChannelWebhookURL")}
                </Label>
                <Input
                  id="new-channel-webhook"
                  value={newChannelWebhookURL}
                  onChange={(e) => setNewChannelWebhookURL(e.target.value)}
                  placeholder={t("settings.deliveryChannelWebhookPlaceholder")}
                  required
                  disabled={isCreatingDeliveryChannel || editingChannelID !== null}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="new-channel-keywords">
                  {t("settings.deliveryChannelKeywords")}
                </Label>
                <Input
                  id="new-channel-keywords"
                  value={newChannelKeywords}
                  onChange={(e) => setNewChannelKeywords(e.target.value)}
                  placeholder={t("settings.deliveryChannelKeywordsPlaceholder")}
                  disabled={isCreatingDeliveryChannel || editingChannelID !== null}
                />
                <p className="text-xs text-muted-foreground">
                  {t("settings.deliveryChannelKeywordsHelp")}
                </p>
              </div>

              <Button
                type="submit"
                className="w-full"
                disabled={
                  isCreatingDeliveryChannel ||
                  editingChannelID !== null ||
                  newChannelWebhookURL.trim().length === 0
                }
              >
                {isCreatingDeliveryChannel ? (
                  <span className="inline-flex items-center gap-2">
                    <Loader2Icon className="size-4 animate-spin" />
                    {t("settings.deliveryChannelCreating")}
                  </span>
                ) : (
                  t("settings.deliveryChannelCreate")
                )}
              </Button>
            </form>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex-row items-center justify-between space-y-0">
          <div className="flex items-center gap-2">
            <LockIcon className="size-4" />
            <CardTitle className="text-base">{t("settings.changePassword")}</CardTitle>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
          {success && (
            <Alert>
              <CircleCheckIcon />
              <AlertDescription>{success}</AlertDescription>
            </Alert>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="current_password">{t("settings.currentPassword")}</Label>
              <Input
                id="current_password"
                type="password"
                name="current_password"
                value={formData.current_password}
                onChange={handleInputChange}
                placeholder={t("settings.currentPasswordPlaceholder")}
                disabled={isMutating}
                autoComplete="current-password"
              />
              <p className="text-xs text-muted-foreground">
                {t("settings.currentPasswordHelp")}
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="new_password">{t("settings.newPassword")}</Label>
              <Input
                id="new_password"
                type="password"
                name="new_password"
                value={formData.new_password}
                onChange={handleInputChange}
                placeholder={t("settings.newPasswordPlaceholder")}
                required
                disabled={isMutating}
                aria-invalid={
                  showValidation &&
                  !passwordValid &&
                  formData.new_password.length > 0 &&
                  !success
                }
                autoComplete="new-password"
              />
              {showValidation && formData.new_password.length > 0 && !success ? (
                <p
                  className={
                    passwordValid
                      ? "text-xs text-emerald-600"
                      : "text-xs text-destructive"
                  }
                >
                  {passwordValid
                    ? t("settings.passwordStrengthValid")
                    : t("settings.passwordMinLength")}
                </p>
              ) : showValidation && !formData.new_password && !success ? (
                <p className="text-xs text-destructive">
                  {t("settings.enterNewPassword")}
                </p>
              ) : null}
            </div>

            <div className="space-y-2">
              <Label htmlFor="confirm_password">
                {t("settings.confirmPassword")}
              </Label>
              <Input
                id="confirm_password"
                type="password"
                name="confirm_password"
                value={formData.confirm_password}
                onChange={handleInputChange}
                placeholder={t("settings.confirmPasswordPlaceholder")}
                required
                disabled={isMutating}
                aria-invalid={
                  showValidation &&
                  formData.confirm_password.length > 0 &&
                  !passwordsMatch &&
                  !success
                }
                autoComplete="new-password"
              />
              {showValidation &&
              formData.confirm_password.length > 0 &&
              !success ? (
                <p
                  className={
                    passwordsMatch
                      ? "text-xs text-emerald-600"
                      : "text-xs text-destructive"
                  }
                >
                  {passwordsMatch
                    ? t("settings.passwordsMatch")
                    : t("settings.passwordsNotMatch")}
                </p>
              ) : showValidation && !formData.confirm_password && !success ? (
                <p className="text-xs text-destructive">
                  {t("settings.confirmPasswordRequired")}
                </p>
              ) : null}
            </div>

            <Button
              type="submit"
              className="w-full"
              disabled={
                isMutating || !formData.new_password || !formData.confirm_password
              }
            >
              {isMutating ? (
                <span className="inline-flex items-center gap-2">
                  <Loader2Icon className="size-4 animate-spin" />
                  {t("settings.changingPassword")}
                </span>
              ) : (
                t("settings.changePassword")
              )}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

export default Settings;
