'use client'

import { useTranslations } from 'next-intl'
import { cn } from '@/lib/utils'
import { passwordStrength } from '@/utils/password-strength'

// B1.11/K.5 密码强度指示器：Field.Description 信息色，仅提示不阻断提交。
export function PasswordStrengthMeter({ password }: { password: string }) {
  const t = useTranslations('settings.password.strength')

  if (!password) return null
  const strength = passwordStrength(password)
  const pct = strength === 'strong' ? 100 : strength === 'fair' ? 60 : 25
  const color =
    strength === 'strong'
      ? 'bg-success'
      : strength === 'fair'
        ? 'bg-warning'
        : 'bg-danger'

  return (
    <div className="flex items-center gap-2">
      <div
        className="h-1.5 w-24 overflow-hidden rounded-full bg-muted"
        role="presentation"
      >
        <div
          className={cn(
            'h-full rounded-full transition-all duration-300',
            color,
          )}
          style={{ width: `${pct}%` }}
        />
      </div>
      <span
        className={cn(
          'text-xs',
          strength === 'strong'
            ? 'text-success'
            : strength === 'fair'
              ? 'text-warning'
              : 'text-danger',
        )}
      >
        {t(strength)}
      </span>
    </div>
  )
}
