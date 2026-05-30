"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { useMutation } from "@tanstack/react-query";
import { PlusIcon, Trash2Icon, PencilIcon } from "lucide-react";
import { Spinner } from "@/components/ui/spinner";

import { deliveryApi } from "@/lib/api";
import { useDeliveryChannels } from "@/hooks/useDeliveryChannels";
import { toast } from "@/stores/toast";
import { mutationOptions } from "@/lib/mutation-helpers";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import { QueryState } from "@/components/ui/query-state";
import { DeliveryChannelForm } from "./DeliveryChannelForm";
import {
  useChannelForm,
  type UpdateChannelMutationVars,
} from "@/hooks/useChannelForm";
import { truncate } from "@/lib/utils";

export function DeliveryChannelsCard() {
  const { channels, invalidate, isLoading, error } = useDeliveryChannels();
  const t = useTranslations("settings");
  const [deleteConfirmId, setDeleteConfirmId] = useState<number | null>(null);
  const tCommon = useTranslations("common");

  const createMutation = useMutation(
    mutationOptions({
      mutationFn: deliveryApi.create,
      onSuccess: () => {
        invalidate();
        resetForm();
        toast.success(t("deliveryChannelCreated"));
      },
    }),
  );

  const updateMutation = useMutation(
    mutationOptions({
      mutationFn: ({ id, data }: UpdateChannelMutationVars) =>
        deliveryApi.update(id, data),
      onSuccess: () => {
        invalidate();
        resetForm();
        toast.success(t("deliveryChannelUpdated"));
      },
    }),
  );

  const deleteMutation = useMutation(
    mutationOptions({
      mutationFn: deliveryApi.delete,
      onSuccess: () => {
        invalidate();
        setDeleteConfirmId(null);
        toast.success(t("deliveryChannelDeleted"));
      },
    }),
  );

  const {
    form,
    setForm,
    showForm,
    editingId,
    isSubmitting,
    resetForm,
    startEdit,
    handleSubmit,
    openNewForm,
  } = useChannelForm({ createMutation, updateMutation });

  return (
    <Card data-testid="delivery-channels-card">
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>{t("deliveryChannels")}</CardTitle>
            <CardDescription>
              {t("deliveryChannelsList")}
            </CardDescription>
          </div>
          {!showForm && (
            <Button size="sm" onClick={openNewForm}>
              <PlusIcon className="mr-1 size-4" />
              {t("deliveryChannelAdd")}
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <QueryState
          isLoading={isLoading}
          error={error}
          loadingText={t("deliveryChannelsLoading")}
          errorText={t("deliveryChannelsLoadFailed")}
          loadingClassName="flex items-center justify-center gap-2 py-4"
        >
          {channels.length === 0 && !showForm && (
            <p className="py-4 text-center text-sm text-muted-foreground">
              {t("deliveryChannelsEmpty")}
            </p>
          )}

          {channels.length > 0 && (
            <div className="space-y-3">
              {channels.map((channel) => (
                <div
                  key={channel.id}
                  className="flex items-center justify-between rounded-lg border p-3"
                >
                  <div className="flex items-center gap-3">
                    <Switch
                      size="sm"
                      checked={channel.enabled}
                      onCheckedChange={(checked) =>
                        updateMutation.mutate({
                          id: channel.id,
                          data: { enabled: checked },
                        })
                      }
                    />
                    <div>
                      <p className="text-sm font-medium">{channel.name || t("deliveryChannelUnnamed")}</p>
                      <p className="text-xs text-muted-foreground">
                        {channel.kind} · {truncate(channel.configuration?.webhook_url ?? "", 40)}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      aria-label={t("deliveryChannelEdit")}
                      onClick={() => startEdit(channel)}
                    >
                      <PencilIcon className="size-4" />
                    </Button>
                    {deleteConfirmId === channel.id ? (
                      <div className="flex items-center gap-1">
                        <Button
                          variant="destructive"
                          size="sm"
                          aria-label={tCommon("confirm")}
                          onClick={() => deleteMutation.mutate(channel.id)}
                          disabled={deleteMutation.isPending}
                        >
                          {deleteMutation.isPending ? (
                            <Spinner className="size-4" />
                          ) : (
                            tCommon("confirm")
                          )}
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          aria-label={tCommon("cancel")}
                          onClick={() => setDeleteConfirmId(null)}
                        >
                          {tCommon("cancel")}
                        </Button>
                      </div>
                    ) : (
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label={t("deliveryChannelDelete")}
                        onClick={() => setDeleteConfirmId(channel.id)}
                      >
                        <Trash2Icon className="size-4" />
                      </Button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}

          {showForm && (
            <DeliveryChannelForm
              form={form}
              onFormChange={(updated) => setForm(updated)}
              onSubmit={handleSubmit}
              onCancel={resetForm}
              isSubmitting={isSubmitting}
              isEditing={!!editingId}
            />
          )}
        </QueryState>
      </CardContent>
    </Card>
  );
}
