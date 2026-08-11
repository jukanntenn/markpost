import type { AuditLogItem } from '@/types/audit'

// DEV-1：叙事里的 {target} 对 user 类动作优先用 JOIN 来的用户名（"deleted
// user bob"），仅在用户已被删除/解析不到时回退到 target_id。post/channel
// 的 target_id 是 qid/渠道 id，始终用原值。
function userTarget(row: AuditLogItem): string {
  return row.target_username ?? row.target_id
}

// D2.3/D4.2 审计动作 i18n 映射（过去式叙事）。返回待翻译的 key 与参数；
// 调用方用 useTranslations('admin.audit.action') 渲染。
export function auditActionText(row: AuditLogItem): {
  key: string
  values: Record<string, string>
} {
  switch (row.action) {
    case 'user.create':
      return { key: 'user.create', values: { target: userTarget(row) } }
    case 'user.set_role':
      return {
        key: 'user.set_role',
        values: {
          target: userTarget(row),
          role: String((row.metadata as Record<string, unknown>)?.role ?? ''),
        },
      }
    case 'user.reset_password':
      return { key: 'user.reset_password', values: { target: userTarget(row) } }
    case 'user.set_active': {
      const active = (row.metadata as Record<string, unknown>)?.active
      return {
        key: active === true ? 'user.enable' : 'user.disable',
        values: { target: userTarget(row) },
      }
    }
    case 'user.delete':
      return { key: 'user.delete', values: { target: userTarget(row) } }
    case 'user.revoke_sessions':
      return {
        key: 'user.revoke_sessions',
        values: { target: userTarget(row) },
      }
    case 'post.delete':
      return { key: 'post.delete', values: { target: row.target_id } }
    case 'channel.create':
      return { key: 'channel.create', values: { target: row.target_id } }
    case 'channel.set_enabled': {
      const enabled = (row.metadata as Record<string, unknown>)?.enabled
      return {
        key: enabled === true ? 'channel.enable' : 'channel.disable',
        values: { target: row.target_id },
      }
    }
    case 'channel.delete':
      return { key: 'channel.delete', values: { target: row.target_id } }
    default:
      return { key: 'user.create', values: { target: row.action } }
  }
}
