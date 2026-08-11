import { z } from 'zod'

// B1.5/B1.7/B1.11 zod schema 共享（前后端规则对齐 specs/auth.md §4.2：
// min 8 / max 72，不强制复杂度；C2.3 一致性裁决：zod 侧字节级 max 同步）。
// 错误消息用 i18n key 标识，由表单组件解析。
const byteLength = (s: string) => new TextEncoder().encode(s).length

// B1.7：登录页只做 required 实时校验，不做长度校验
// （不在合法用户输短密码时干扰；不向攻击者泄露密码策略）。
export const loginSchema = z.object({
  username: z.string({ error: 'required' }).min(1, { error: 'required' }),
  password: z.string({ error: 'required' }).min(1, { error: 'required' }),
})

export const passwordChangeSchema = z
  .object({
    currentPassword: z.string(),
    newPassword: z
      .string({ error: 'required' })
      .min(1, { error: 'required' })
      .refine((v) => v.length >= 8, { error: 'min_length' })
      .refine((v) => v.length <= 72 && byteLength(v) <= 72, {
        error: 'max_length',
      }),
    confirmPassword: z
      .string({ error: 'required' })
      .min(1, { error: 'required' }),
  })
  .refine((v) => v.newPassword === v.confirmPassword, {
    error: 'not_match',
    path: ['confirmPassword'],
  })

export const adminCreateUserSchema = z.object({
  username: z.string({ error: 'required' }).min(1, { error: 'required' }),
  email: z
    .string()
    .optional()
    .or(z.literal(''))
    .refine((v) => !v || /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v), {
      error: 'invalid_email',
    }),
  password: z
    .string({ error: 'required' })
    .min(1, { error: 'required' })
    .refine((v) => v.length >= 8, { error: 'min_length' })
    .refine((v) => v.length <= 72 && byteLength(v) <= 72, {
      error: 'max_length',
    }),
})

export const createTestPostSchema = z.object({
  title: z
    .string({ error: 'required' })
    .min(1, { error: 'required' })
    .refine((v) => v.length <= 150, { error: 'title_too_long' }),
  body: z.string({ error: 'required' }).min(1, { error: 'required' }),
})
