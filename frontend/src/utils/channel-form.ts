import { z } from "zod";
import type {
  DeliveryChannel,
  CreateChannelPayload,
  UpdateChannelPayload,
  FeishuConfiguration,
} from "@/types/delivery";

export type UpdateChannelMutationVars = { id: number; data: UpdateChannelPayload };

export const feishuConfigurationSchema = z.object({
  webhook_url: z
    .string()
    .min(1, "Webhook URL is required")
    .url("Must be a valid URL"),
  card_link_url: z.string().default(""),
});

export const channelConfigurationSchemas: Record<
  string,
  z.ZodType<FeishuConfiguration>
> = {
  feishu: feishuConfigurationSchema,
};

export interface FormState {
  kind: string;
  name: string;
  configuration: FeishuConfiguration;
  keywords: string;
}

export const EMPTY_FORM: FormState = {
  kind: "feishu",
  name: "",
  configuration: { webhook_url: "", card_link_url: "" },
  keywords: "",
};

export function channelToForm(channel: DeliveryChannel): FormState {
  return {
    kind: channel.kind,
    name: channel.name,
    configuration: {
      webhook_url: channel.configuration?.webhook_url ?? "",
      card_link_url: channel.configuration?.card_link_url ?? "",
    },
    keywords: channel.keywords,
  };
}

export function formToCreatePayload(form: FormState): CreateChannelPayload {
  return {
    kind: form.kind,
    name: form.name,
    configuration: form.configuration,
    keywords: form.keywords,
  };
}

export function formToUpdatePayload(
  editingId: number,
  form: FormState,
): UpdateChannelMutationVars {
  return {
    id: editingId,
    data: {
      name: form.name,
      configuration: form.configuration,
      keywords: form.keywords,
    },
  };
}

export function validateConfiguration(
  kind: string,
  configuration: FeishuConfiguration,
): { valid: boolean; errors: Record<string, string> } {
  const schema = channelConfigurationSchemas[kind];
  if (!schema) {
    return { valid: false, errors: { kind: "Unsupported channel type" } };
  }

  const result = schema.safeParse(configuration);
  if (result.success) {
    return { valid: true, errors: {} };
  }

  const errors: Record<string, string> = {};
  for (const issue of result.error.issues) {
    const field = issue.path.join(".");
    if (!errors[field]) {
      errors[field] = issue.message;
    }
  }
  return { valid: false, errors };
}
