import { describe, expect, it } from 'vitest'
import { createTranslator } from 'next-intl'
import { auditActionText } from '@/lib/audit-action-text'
import type { AuditLogItem } from '@/types/audit'
import en from '@/i18n/locales/en.json'
import zhHans from '@/i18n/locales/zh-Hans.json'
import zhHant from '@/i18n/locales/zh-Hant.json'
import ja from '@/i18n/locales/ja.json'

// D2.3/D4.2 回归：审计动作 key 是嵌套路径（user.create 等），next-intl 的 t()
// 按点路径解析。带点字面 key（"user.create" 作为属性名）无法命中——这是
// 之前"动作列显示原始 key"的根因，嵌套结构修复后这里锁定行为。
describe('audit action i18n resolution', () => {
  const locales = [
    { name: 'en', messages: en },
    { name: 'zh-Hans', messages: zhHans },
    { name: 'zh-Hant', messages: zhHant },
    { name: 'ja', messages: ja },
  ] as const

  const actions: Array<[string, Record<string, string>]> = [
    ['user.create', { target: 'bob' }],
    ['user.set_role', { target: 'bob', role: 'admin' }],
    ['user.reset_password', { target: 'bob' }],
    ['user.disable', { target: 'bob' }],
    ['user.enable', { target: 'bob' }],
    ['user.delete', { target: 'bob' }],
    ['user.revoke_sessions', { target: 'bob' }],
    ['post.delete', { target: 'abc3' }],
    ['channel.create', { target: 'work' }],
    ['channel.enable', { target: 'work' }],
    ['channel.disable', { target: 'work' }],
    ['channel.delete', { target: 'work' }],
  ]

  for (const { name, messages } of locales) {
    const t = createTranslator({
      locale: name,
      messages,
      namespace: 'admin.audit.action',
    })

    it(`${name}: resolves every action key without falling back to the key path`, () => {
      for (const [key, values] of actions) {
        const resolved = t(key as never, values as never)
        expect(resolved, `${name} ${key}`).not.toContain('admin.audit.action')
        expect(resolved, `${name} ${key}`).not.toBe(key)
      }
    })
  }

  it('auditActionText maps every action to a resolvable key', () => {
    const base = {
      id: 1,
      actor_id: 1,
      actor_username: 'alice',
      target_type: 'user',
      target_id: '12',
      target_username: null,
      metadata: {},
      ip: '',
      created_at: '',
    } as AuditLogItem

    const cases: Array<[AuditLogItem, string]> = [
      [{ ...base, action: 'user.create' }, 'user.create'],
      [
        { ...base, action: 'user.set_role', metadata: { role: 'admin' } },
        'user.set_role',
      ],
      [{ ...base, action: 'user.reset_password' }, 'user.reset_password'],
      [
        { ...base, action: 'user.set_active', metadata: { active: true } },
        'user.enable',
      ],
      [
        { ...base, action: 'user.set_active', metadata: { active: false } },
        'user.disable',
      ],
      [{ ...base, action: 'user.delete' }, 'user.delete'],
      [{ ...base, action: 'user.revoke_sessions' }, 'user.revoke_sessions'],
      [
        {
          ...base,
          action: 'post.delete',
          target_type: 'post',
          target_id: 'abc3',
        },
        'post.delete',
      ],
      [
        {
          ...base,
          action: 'channel.create',
          target_type: 'channel',
          target_id: '9',
        },
        'channel.create',
      ],
      [
        {
          ...base,
          action: 'channel.set_enabled',
          target_type: 'channel',
          metadata: { enabled: true },
        },
        'channel.enable',
      ],
      [
        {
          ...base,
          action: 'channel.set_enabled',
          target_type: 'channel',
          metadata: { enabled: false },
        },
        'channel.disable',
      ],
      [
        { ...base, action: 'channel.delete', target_type: 'channel' },
        'channel.delete',
      ],
    ]

    for (const [row, expected] of cases) {
      expect(auditActionText(row).key, row.action).toBe(expected)
    }
  })

  // DEV-1：user 类动作的 {target} 优先用 target_username（"deleted user bob"），
  // 仅在 JOIN 不到（用户已删除）时回退 target_id。post/channel 不受影响。
  it('uses target_username over target_id for user-targeted actions', () => {
    const withUsername = {
      id: 2,
      actor_id: 1,
      actor_username: 'alice',
      action: 'user.delete',
      target_type: 'user',
      target_id: '12',
      target_username: 'bob',
      metadata: {},
      ip: '',
      created_at: '',
    } as AuditLogItem

    expect(auditActionText(withUsername).values.target).toBe('bob')

    const deleted = { ...withUsername, target_username: null } as AuditLogItem
    expect(auditActionText(deleted).values.target).toBe('12')
  })

  it('never uses target_username for post/channel actions', () => {
    const row = {
      id: 3,
      actor_id: 1,
      actor_username: 'alice',
      action: 'post.delete',
      target_type: 'post',
      target_id: 'mpk-abc3',
      target_username: null,
      metadata: {},
      ip: '',
      created_at: '',
    } as AuditLogItem
    expect(auditActionText(row).values.target).toBe('mpk-abc3')
  })
})
