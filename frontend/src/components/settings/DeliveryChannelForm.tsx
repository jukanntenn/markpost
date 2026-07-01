"use client";

import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import {
  compileKeywordFilter,
  describeFilter,
} from "@/lib/keyword-filter";
import type { FormState } from "@/hooks/useChannelForm";

interface DeliveryChannelFormProps {
  form: FormState;
  onFormChange: (form: FormState) => void;
  onSubmit: (e: React.FormEvent) => void;
  onCancel: () => void;
  isSubmitting: boolean;
  isEditing: boolean;
}

export function DeliveryChannelForm({
  form,
  onFormChange,
  onSubmit,
  onCancel,
  isSubmitting,
  isEditing,
}: DeliveryChannelFormProps) {
  const t = useTranslations("settings");

  function updateConfiguration(field: string, value: string) {
    onFormChange({
      ...form,
      configuration: { ...form.configuration, [field]: value },
    });
  }

  return (
    <>
      <Separator />
      <form onSubmit={onSubmit} className="space-y-3">
        <div className="space-y-2">
          <Label htmlFor="channel-name">{t("deliveryChannelName")}</Label>
          <Input
            id="channel-name"
            value={form.name}
            onChange={(e) => onFormChange({ ...form, name: e.target.value })}
            placeholder={t("deliveryChannelNamePlaceholder")}
            required
            disabled={isSubmitting}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="channel-webhook">{t("deliveryChannelWebhookURL")}</Label>
          <Input
            id="channel-webhook"
            value={form.configuration.webhook_url}
            onChange={(e) => updateConfiguration("webhook_url", e.target.value)}
            placeholder={t("deliveryChannelWebhookPlaceholder")}
            required
            disabled={isSubmitting}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="channel-card-link-url">
            {t("deliveryChannelCardLinkURL")}
          </Label>
          <Input
            id="channel-card-link-url"
            value={form.configuration.card_link_url}
            onChange={(e) =>
              updateConfiguration("card_link_url", e.target.value)
            }
            placeholder={t("deliveryChannelCardLinkURLPlaceholder")}
            disabled={isSubmitting}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="channel-keywords">{t("deliveryChannelKeywords")}</Label>
          <Input
            id="channel-keywords"
            value={form.keywords}
            onChange={(e) => onFormChange({ ...form, keywords: e.target.value })}
            placeholder={t("deliveryChannelKeywordsPlaceholder")}
            disabled={isSubmitting}
          />
          <KeywordFilterFeedback value={form.keywords} />
        </div>

        <div className="flex gap-2">
          <LoadingButton type="submit" loading={isSubmitting} loadingText={t("deliveryChannelSaving")}>
            {isEditing ? t("deliveryChannelSave") : t("deliveryChannelCreate")}
          </LoadingButton>
          <Button
            type="button"
            variant="outline"
            onClick={onCancel}
            disabled={isSubmitting}
          >
            {t("deliveryChannelCancel")}
          </Button>
        </div>
      </form>
    </>
  );
}

function KeywordFilterFeedback({ value }: { value: string }) {
  const t = useTranslations("settings");

  const trimmed = value.trim();
  if (trimmed === "") return null;

  const { node, error } = compileKeywordFilter(value);
  if (error !== null) {
    return (
      <p className="text-sm text-destructive">
        {t("deliveryChannelKeywordsInvalid", { error })}
      </p>
    );
  }

  const description = describeFilter(node);
  if (description === null) return null;

  return (
    <p className="text-sm text-muted-foreground">
      {t("deliveryChannelKeywordsPreview", { preview: description })}
    </p>
  );
}
