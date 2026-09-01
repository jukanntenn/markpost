'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { UsersIcon } from 'lucide-react'
import { adminApi, adminKeys } from '@/lib/api'
import { useAuthStore } from '@/stores/auth'
import { toastManager } from '@/stores/toast'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { VipStrategyToggle } from './VipStrategyToggle'
import { RetentionDialog } from './RetentionDialog'

// The VIP policy bar on the admin users page header (MRFC
// 2026-08-31-per-user-history-retention-policy): the existing grant-strategy
// toggle plus the VIP-class retention default — materialized onto a user at
// grant time — and the one-shot "apply to every VIP user" realignment.
export function VipPolicyBar() {
  const t = useTranslations('admin.retention.vipPolicy')
  const me = useAuthStore((state) => state.user)
  const queryClient = useQueryClient()
  const [alignOpen, setAlignOpen] = useState(false)
  const [days, setDays] = useState('')

  const settingsQuery = useQuery({
    queryKey: adminKeys.settings(),
    queryFn: () => adminApi.listSettings(),
    staleTime: 30_000,
    enabled: me?.role === 'admin',
  })

  // VIP population count for the apply button label; forever candidate keeps
  // the preview free of deletion bookkeeping.
  const vipCountQuery = useQuery({
    queryKey: ['admin', 'retention', 'vip-count'],
    queryFn: () => adminApi.retentionImpact({ scope: 'vip' }, 0),
    staleTime: 30_000,
    enabled: me?.role === 'admin',
  })

  const classSetting = settingsQuery.data?.items.find(
    (s) => s.key === 'vip_retention_days',
  )
  const classDays = classSetting?.value.days ?? null

  const save = useMutation({
    mutationFn: (value: number | null) =>
      adminApi.updateSettingDays('vip_retention_days', value),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: adminKeys.settings() })
      queryClient.invalidateQueries({ queryKey: adminKeys.audit.all() })
      toastManager.add({ type: 'success', title: t('toasts.saved') })
    },
    onError: (err: Error) => {
      toastManager.add({ type: 'error', title: err.message })
    },
  })

  const parsedDays = Number.parseInt(days, 10)
  const daysValid =
    Number.isFinite(parsedDays) && parsedDays >= 1 && parsedDays <= 3650

  function commit(next: number | null) {
    if (next !== null && (next < 0 || next > 3650)) return
    if (next !== null && next > 0 && !daysValid && String(next) !== days) return
    save.mutate(next)
  }

  const vipCount = vipCountQuery.data?.users_affected

  return (
    <div
      className="flex flex-wrap items-center gap-x-4 gap-y-2 rounded-lg border bg-card px-3 py-2"
      data-testid="vip-policy-bar"
    >
      <VipStrategyToggle />

      <div className="flex items-center gap-2 text-sm">
        <span className="text-muted-foreground">{t('classDefault')}</span>
        <div
          className="flex overflow-hidden rounded-md border"
          role="radiogroup"
          aria-label={t('classDefault')}
        >
          {(
            [
              [classDays === null, '', t('followGlobal')],
              [classDays === 0, '0', t('forever')],
              [classDays !== null && classDays > 0, 'days', t('nDays')],
            ] as const
          ).map(([active, value, label]) => (
            <button
              key={value || 'global'}
              type="button"
              role="radio"
              aria-checked={active}
              className={
                'px-2 py-1 text-xs transition-colors ' +
                (active
                  ? 'bg-primary/10 font-medium text-primary'
                  : 'text-muted-foreground hover:text-foreground')
              }
              onClick={() => {
                if (value === '') commit(null)
                else if (value === '0') commit(0)
                else if (daysValid) commit(parsedDays)
                else if (classDays && classDays > 0) commit(classDays)
              }}
            >
              {label}
            </button>
          ))}
        </div>
        {classDays !== null && classDays > 0 && (
          <span className="flex items-center gap-1">
            <Input
              type="number"
              min={1}
              max={3650}
              value={days || String(classDays)}
              onChange={(e) => setDays(e.target.value)}
              onBlur={() => daysValid && save.mutate(parsedDays)}
              className="h-7 w-20"
              aria-label={t('nDays')}
              data-testid="vip-class-days-input"
            />
            <span className="text-xs text-muted-foreground">
              {t('daysUnit')}
            </span>
          </span>
        )}
      </div>

      <Button
        variant="outline"
        size="sm"
        className="ml-auto"
        onClick={() => setAlignOpen(true)}
        data-testid="vip-apply-all-button"
      >
        <UsersIcon className="mr-1 size-4" />
        {vipCount !== undefined
          ? t('applyAll', { n: vipCount })
          : t('applyAllLoading')}
      </Button>

      <RetentionDialog
        target={{ kind: 'vip' }}
        open={alignOpen}
        onOpenChange={setAlignOpen}
        preselect={classDays}
      />
    </div>
  )
}

export default VipPolicyBar
