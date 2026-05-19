import type { DeliveryChannel, CreateChannelPayload, UpdateChannelPayload } from "@/types/delivery";

export type UpdateChannelMutationVars = { id: number; data: UpdateChannelPayload };

export interface FormState {
  kind: string;
  name: string;
  webhookUrl: string;
  keywords: string;
}

export const EMPTY_FORM: FormState = { kind: "feishu", name: "", webhookUrl: "", keywords: "" };

export function channelToForm(channel: DeliveryChannel): FormState {
  return {
    kind: channel.kind,
    name: channel.name,
    webhookUrl: channel.webhook_url,
    keywords: channel.keywords,
  };
}

export function formToCreatePayload(form: FormState): CreateChannelPayload {
  return {
    kind: form.kind,
    name: form.name,
    webhook_url: form.webhookUrl,
    keywords: form.keywords,
  };
}

export function formToUpdatePayload(editingId: number, form: FormState): UpdateChannelMutationVars {
  return {
    id: editingId,
    data: {
      name: form.name,
      webhook_url: form.webhookUrl,
      keywords: form.keywords,
    },
  };
}
