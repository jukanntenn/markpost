"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Trash2Icon } from "lucide-react";

import { deliveryApi, deliveryKeys, invalidateKey } from "@/lib/api";
import { mutationOptions } from "@/lib/mutation-helpers";
import { toast } from "@/stores/toast";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  EMPTY_FORM,
  channelToForm,
  formToCreatePayload,
  formToUpdatePayload,
  type FormState,
  type UpdateChannelMutationVars,
} from "@/utils/channel-form";
import {
  compileKeywordFilter,
  describeFilter,
} from "@/lib/keyword-filter";
import type { DeliveryChannel } from "@/types/delivery";

interface DeliveryChannelDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  editingChannel: DeliveryChannel | null;
}

export function DeliveryChannelDialog({
  open,
  onOpenChange,
  editingChannel,
}: DeliveryChannelDialogProps) {
  const t = useTranslations("delivery");
  const queryClient = useQueryClient();

  const isEditing = editingChannel !== null;

  const [form, setForm] = useState<FormState>(
    editingChannel ? channelToForm(editingChannel) : EMPTY_FORM,
  );
  const [confirmDelete, setConfirmDelete] = useState(false);

  function invalidateChannels() {
    invalidateKey(queryClient, deliveryKeys.channels());
    invalidateKey(queryClient, deliveryKeys.latest());
  }

  const createMutation = useMutation(
    mutationOptions({
      mutationFn: deliveryApi.create,
      onSuccess: () => {
        invalidateChannels();
        toast.success(t("dialog.created"));
        onOpenChange(false);
      },
    }),
  );

  const updateMutation = useMutation(
    mutationOptions({
      mutationFn: ({ id, data }: UpdateChannelMutationVars) =>
        deliveryApi.update(id, data),
      onSuccess: () => {
        invalidateChannels();
        toast.success(t("dialog.updated"));
        onOpenChange(false);
      },
    }),
  );

  const deleteMutation = useMutation(
    mutationOptions({
      mutationFn: deliveryApi.delete,
      onSuccess: () => {
        invalidateChannels();
        toast.success(t("dialog.deleted"));
        onOpenChange(false);
      },
    }),
  );

  const testMutation = useMutation({
    mutationFn: deliveryApi.test,
    onSuccess: () => {
      invalidateKey(queryClient, deliveryKeys.latest());
      toast.success(t("dialog.testSuccess"));
    },
    onError: (err: Error) => {
      toast.error(t("dialog.testFailed"), { description: err.message });
    },
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (editingChannel !== null) {
      updateMutation.mutate(formToUpdatePayload(editingChannel.id, form));
    } else {
      createMutation.mutate(formToCreatePayload(form));
    }
  }

  const isSubmitting = createMutation.isPending || updateMutation.isPending;
  const disableForm = isSubmitting || confirmDelete;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <form onSubmit={handleSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>
              {isEditing ? t("dialog.editTitle") : t("dialog.createTitle")}
            </DialogTitle>
            <DialogDescription className="sr-only">
              {isEditing ? t("dialog.editTitle") : t("dialog.createTitle")}
            </DialogDescription>
          </DialogHeader>

          <ChannelFormFields
            form={form}
            onFormChange={setForm}
            disabled={disableForm}
          />

          <DialogFooter className="sm:justify-between">
            {isEditing && !confirmDelete ? (
              <Button
                type="button"
                variant="ghost"
                className="text-destructive hover:text-destructive"
                onClick={() => setConfirmDelete(true)}
                disabled={isSubmitting}
              >
                <Trash2Icon className="mr-1 size-4" />
                {t("dialog.delete")}
              </Button>
            ) : null}

            {isEditing && confirmDelete ? (
              <div className="flex gap-2">
                <LoadingButton
                  type="button"
                  variant="destructive"
                  loading={deleteMutation.isPending}
                  loadingText={t("dialog.deleting")}
                  onClick={() => editingChannel && deleteMutation.mutate(editingChannel.id)}
                >
                  {t("dialog.confirmDelete")}
                </LoadingButton>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setConfirmDelete(false)}
                  disabled={deleteMutation.isPending}
                >
                  {t("dialog.cancel")}
                </Button>
              </div>
            ) : (
              <div className="flex gap-2">
                {isEditing && (
                  <LoadingButton
                    type="button"
                    variant="outline"
                    loading={testMutation.isPending}
                    loadingText={t("dialog.testing")}
                    onClick={() => editingChannel && testMutation.mutate(editingChannel.id)}
                    disabled={isSubmitting}
                  >
                    {t("dialog.test")}
                  </LoadingButton>
                )}
                <DialogClose
                  render={
                    <Button type="button" variant="outline" disabled={disableForm} />
                  }
                >
                  {t("dialog.cancel")}
                </DialogClose>
                <LoadingButton
                  type="submit"
                  loading={isSubmitting}
                  loadingText={t("dialog.saving")}
                  disabled={disableForm}
                >
                  {isEditing ? t("dialog.save") : t("dialog.create")}
                </LoadingButton>
              </div>
            )}
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

interface ChannelFormFieldsProps {
  form: FormState;
  onFormChange: (form: FormState) => void;
  disabled: boolean;
}

function ChannelFormFields({ form, onFormChange, disabled }: ChannelFormFieldsProps) {
  const t = useTranslations("delivery");

  function updateConfiguration(field: string, value: string) {
    onFormChange({
      ...form,
      configuration: { ...form.configuration, [field]: value },
    });
  }

  return (
    <div className="space-y-3">
      <div className="space-y-2">
        <Label htmlFor="channel-name">{t("dialog.name")}</Label>
        <Input
          id="channel-name"
          value={form.name}
          onChange={(e) => onFormChange({ ...form, name: e.target.value })}
          placeholder={t("dialog.namePlaceholder")}
          required
          disabled={disabled}
          autoComplete="off"
          autoFocus
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="channel-webhook">{t("dialog.webhookURL")}</Label>
        <Input
          id="channel-webhook"
          value={form.configuration.webhook_url}
          onChange={(e) => updateConfiguration("webhook_url", e.target.value)}
          placeholder={t("dialog.webhookPlaceholder")}
          required
          disabled={disabled}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="channel-card-link-url">
          {t("dialog.cardLinkURL")}
        </Label>
        <Input
          id="channel-card-link-url"
          value={form.configuration.card_link_url}
          onChange={(e) => updateConfiguration("card_link_url", e.target.value)}
          placeholder={t("dialog.cardLinkURLPlaceholder")}
          disabled={disabled}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="channel-keywords">{t("dialog.keywords")}</Label>
        <Input
          id="channel-keywords"
          value={form.keywords}
          onChange={(e) => onFormChange({ ...form, keywords: e.target.value })}
          placeholder={t("dialog.keywordsPlaceholder")}
          disabled={disabled}
        />
        <KeywordFilterFeedback value={form.keywords} />
      </div>
    </div>
  );
}

function KeywordFilterFeedback({ value }: { value: string }) {
  const t = useTranslations("delivery");

  const trimmed = value.trim();
  if (trimmed === "") return null;

  const { node, error } = compileKeywordFilter(value);
  if (error !== null) {
    return (
      <p className="text-sm text-destructive">
        {t("dialog.keywordsInvalid", { error })}
      </p>
    );
  }

  const description = describeFilter(node);
  if (description === null) return null;

  return (
    <p className="text-sm text-muted-foreground">
      {t("dialog.keywordsPreview", { preview: description })}
    </p>
  );
}
