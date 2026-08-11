import { z } from 'zod'
import type {
  DeliveryChannel,
  CreateChannelPayload,
  UpdateChannelPayload,
  FeishuConfiguration,
} from '@/types/delivery'

export type UpdateChannelMutationVars = {
  id: number
  data: UpdateChannelPayload
}

// D5.2 字段规格：webhook_url required + url；card_link_url 可选但填了须 url
// （原 default('') 不校验 → optional().or(url())）。校验文案用 i18n key 标识，
// 由表单组件通过 next-intl 解析（错误消息不硬编码英文）。
export const feishuConfigurationSchema = z.object({
  webhook_url: z
    .string({ error: 'required' })
    .min(1, { error: 'required' })
    .url({ error: 'invalid_url' }),
  card_link_url: z
    .string({ error: 'invalid_url' })
    .url({ error: 'invalid_url' })
    .or(z.literal(''))
    .optional()
    .transform((v) => v ?? ''),
})

export const channelConfigurationSchemas: Record<
  string,
  z.ZodType<FeishuConfiguration>
> = {
  feishu: feishuConfigurationSchema,
}

export interface FormState {
  kind: string
  name: string
  configuration: FeishuConfiguration
  keywords: string
}

export const EMPTY_FORM: FormState = {
  kind: 'feishu',
  name: '',
  configuration: { webhook_url: '', card_link_url: '' },
  keywords: '',
}

export function channelToForm(channel: DeliveryChannel): FormState {
  return {
    kind: channel.kind,
    name: channel.name,
    configuration: {
      webhook_url: channel.configuration?.webhook_url ?? '',
      card_link_url: channel.configuration?.card_link_url ?? '',
    },
    keywords: channel.keywords,
  }
}

export function formToCreatePayload(form: FormState): CreateChannelPayload {
  return {
    kind: form.kind,
    name: form.name,
    configuration: form.configuration,
    keywords: form.keywords,
  }
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
  }
}

export function validateConfiguration(
  kind: string,
  configuration: FeishuConfiguration,
): { valid: boolean; errors: Record<string, string> } {
  const schema = channelConfigurationSchemas[kind]
  if (!schema) {
    return { valid: false, errors: { kind: 'Unsupported channel type' } }
  }

  const result = schema.safeParse(configuration)
  if (result.success) {
    return { valid: true, errors: {} }
  }

  const errors: Record<string, string> = {}
  for (const issue of result.error.issues) {
    const field = issue.path.join('.')
    if (!errors[field]) {
      errors[field] = issue.message
    }
  }
  return { valid: false, errors }
}
