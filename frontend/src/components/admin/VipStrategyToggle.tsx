'use client'

import { useTranslations } from 'next-intl'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { adminApi, adminKeys } from '@/lib/api'
import { useAuthStore } from '@/stores/auth'
import { toastManager } from '@/stores/toast'
import { Switch } from '@/components/ui/switch'
import { VipBadge } from '@/components/ui/vip-badge'

// The GitHub-login VIP strategy switch on the admin users page header (MRFC
// 2026-08-23-vip-badge-and-admin-management): one toggle, immediate effect on
// the next login, audited server-side as setting.set.
export function VipStrategyToggle() {
  const t = useTranslations('admin.users.vipStrategy')
  const me = useAuthStore((state) => state.user)
  const queryClient = useQueryClient()

  const settingsQuery = useQuery({
    queryKey: adminKeys.settings(),
    queryFn: () => adminApi.listSettings(),
    staleTime: 30_000,
    enabled: me?.role === 'admin',
  })

  const toggle = useMutation({
    mutationFn: (enabled: boolean) => adminApi.updateSetting('vip', enabled),
    onSuccess: (_d, enabled) => {
      queryClient.invalidateQueries({ queryKey: adminKeys.settings() })
      queryClient.invalidateQueries({ queryKey: adminKeys.audit.all() })
      toastManager.add({
        type: 'success',
        title: enabled ? t('toasts.enabled') : t('toasts.disabled'),
      })
    },
    onError: (err: Error) => {
      toastManager.add({ type: 'error', title: err.message })
    },
  })

  const vipSetting = settingsQuery.data?.items.find((s) => s.key === 'vip')
  const enabled = vipSetting?.value.enabled ?? false

  return (
    <label className="text-muted-foreground flex cursor-pointer items-center gap-2 text-sm">
      <VipBadge />
      {t('label')}
      <Switch
        checked={enabled}
        onCheckedChange={(v) => toggle.mutate(v)}
        disabled={settingsQuery.isLoading || toggle.isPending}
        aria-label={t('label')}
      />
    </label>
  )
}

export default VipStrategyToggle
