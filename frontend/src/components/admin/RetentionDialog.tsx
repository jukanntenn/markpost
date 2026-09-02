'use client'

import { useEffect, useMemo, useState } from 'react'
import { useTranslations } from 'next-intl'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { adminApi, adminKeys } from '@/lib/api'
import { toastManager } from '@/stores/toast'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import type { RetentionImpact } from '@/types/users'

// One retention dialog serves every surface (MRFC
// 2026-08-31-per-user-history-retention-policy): a single user from the row
// menu or the detail page, an explicit bulk selection, or the vip-align action
// from the VIP policy bar. Targets carry ids so applying never re-fetches.
export type RetentionTarget =
  | { kind: 'user'; id: number; username: string; current: number | null }
  | { kind: 'bulk'; userIds: number[]; count: number }
  | { kind: 'vip' }

type Segment = 'inherit' | 'forever' | 'days'

const PRESETS = [7, 30, 90, 365]
// MRFC: type-to-confirm threshold for the destructive shorten flow.
const TYPE_CONFIRM_THRESHOLD = 1000

// RetentionPolicyText renders the effective-value label shared by the list
// column, the detail row, and the dialog: forever / N days / inherit.
export function RetentionPolicyText({ days }: { days: number | null }) {
  const t = useTranslations('admin.retention')
  if (days === null) return <>{t('value.inherit')}</>
  if (days === 0) return <>{t('value.forever')}</>
  return <>{t('value.days', { n: days })}</>
}

export function RetentionDialog({
  target,
  open,
  onOpenChange,
  preselect,
}: {
  target: RetentionTarget
  open: boolean
  onOpenChange: (open: boolean) => void
  // vip-align preselects the class default so the bar's action lands on the
  // configured promise in one confirm.
  preselect?: number | null
}) {
  const t = useTranslations('admin.retention')
  const tCommon = useTranslations('common')
  const queryClient = useQueryClient()

  const [segment, setSegment] = useState<Segment>('inherit')
  const [days, setDays] = useState('')
  const [impact, setImpact] = useState<RetentionImpact | null>(null)
  const [impactLoading, setImpactLoading] = useState(false)
  const [confirmText, setConfirmText] = useState('')

  useEffect(() => {
    if (!open) return
    setImpact(null)
    setConfirmText('')
    if (preselect === 0) setSegment('forever')
    else if (preselect && preselect > 0) {
      setSegment('days')
      setDays(String(preselect))
    } else {
      setSegment('inherit')
    }
    // Reset per open: target/preselect are fixed while the dialog is open.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  const daysValue = useMemo(() => {
    const n = Number.parseInt(days, 10)
    return Number.isFinite(n) && n >= 1 && n <= 3650 ? n : null
  }, [days])

  const candidate: number | null =
    segment === 'inherit' ? null : segment === 'forever' ? 0 : daysValue

  const impactRequest =
    target.kind === 'vip'
      ? () => adminApi.retentionImpact({ scope: 'vip' }, candidate)
      : target.kind === 'bulk'
        ? () =>
            adminApi.retentionImpact({ user_ids: target.userIds }, candidate)
        : () => adminApi.retentionImpact({ user_ids: [target.id] }, candidate)

  // vip-align opens with the population count already visible: the label
  // reads the count, which otherwise arrives only with the confirm-time
  // preview. It must not feed the confirm gate — the preview's deletion
  // counts belong to the candidate on the confirm click, not the open.
  const [vipCount, setVipCount] = useState<number | null>(null)
  useEffect(() => {
    if (!open || target.kind !== 'vip') return
    setVipCount(null)
    let cancelled = false
    void adminApi
      .retentionImpact({ scope: 'vip' }, 0)
      .then((r) => {
        if (!cancelled) setVipCount(r.users_affected)
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [open, target.kind])

  const apply = useMutation({
    mutationFn: async () => {
      if (target.kind === 'user') {
        return {
          updated: 1 as number,
          user: await adminApi.setUserRetention(target.id, candidate),
        }
      }
      if (target.kind === 'bulk') {
        return {
          updated: (
            await adminApi.bulkSetRetention(
              { user_ids: target.userIds },
              candidate,
            )
          ).updated,
        }
      }
      return {
        updated: (await adminApi.bulkSetRetention({ scope: 'vip' }, candidate))
          .updated,
      }
    },
    onSuccess: ({ updated }) => {
      queryClient.invalidateQueries({ queryKey: adminKeys.users.all() })
      queryClient.invalidateQueries({ queryKey: adminKeys.settings() })
      queryClient.invalidateQueries({ queryKey: adminKeys.audit.all() })
      toastManager.add({
        type: 'success',
        title: t('toasts.applied', { n: updated }),
      })
      onOpenChange(false)
    },
    onError: (err: Error) => {
      toastManager.add({ type: 'error', title: err.message })
    },
  })

  async function onConfirm() {
    if (segment === 'days' && !daysValue) return
    // Fetch the deletion preview first; only a zero-deletion candidate (or
    // forever, which matches nothing) goes straight through — the MRFC's
    // shorten-with-impact flow demands an explicit confirmation.
    if (candidate !== 0) {
      setImpactLoading(true)
      try {
        const result = await impactRequest()
        setImpact(result)
        if (result.posts_to_delete + result.history_to_delete > 0) return
      } catch (err) {
        toastManager.add({
          type: 'error',
          title: err instanceof Error ? err.message : 'impact failed',
        })
        return
      } finally {
        setImpactLoading(false)
      }
    }
    apply.mutate()
  }

  const deletions = impact
    ? impact.posts_to_delete + impact.history_to_delete
    : 0
  const needsConfirm = impact !== null && deletions > 0
  const needsTypeConfirm =
    needsConfirm && (impact?.posts_to_delete ?? 0) >= TYPE_CONFIRM_THRESHOLD

  const targetLabel =
    target.kind === 'user'
      ? target.username
      : target.kind === 'bulk'
        ? t('targets.bulk', { n: target.count })
        : t('targets.vip', { n: vipCount ?? 0 })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('title')}</DialogTitle>
          <DialogDescription>
            {target.kind === 'user' && target.current !== null ? (
              <>
                {t('subtitleSingle', { name: target.username })}{' '}
                <RetentionPolicyText days={target.current} />
              </>
            ) : (
              t('subtitle', { target: targetLabel })
            )}
          </DialogDescription>
        </DialogHeader>

        <div
          className="grid grid-cols-3 gap-1 rounded-lg bg-muted p-1"
          role="radiogroup"
          aria-label={t('title')}
        >
          {(
            [
              ['inherit', t('segments.inherit')],
              ['forever', t('segments.forever')],
              ['days', t('segments.days')],
            ] as const
          ).map(([value, label]) => (
            <button
              key={value}
              type="button"
              role="radio"
              aria-checked={segment === value}
              data-segment={value}
              className={
                'rounded-md px-2 py-1.5 text-sm transition-colors ' +
                (segment === value
                  ? 'bg-background font-medium shadow-sm'
                  : 'text-muted-foreground hover:text-foreground')
              }
              onClick={() => setSegment(value)}
            >
              {label}
            </button>
          ))}
        </div>

        {segment === 'days' && (
          <div className="space-y-2" data-testid="retention-days-section">
            <div className="flex flex-wrap gap-1.5">
              {PRESETS.map((p) => (
                <button
                  key={p}
                  type="button"
                  className={
                    'rounded-full border px-3 py-1 text-xs transition-colors ' +
                    (daysValue === p
                      ? 'border-primary bg-primary/10 text-primary'
                      : 'text-muted-foreground hover:border-primary/40')
                  }
                  onClick={() => setDays(String(p))}
                >
                  {t('presetDays', { n: p })}
                </button>
              ))}
            </div>
            <div className="flex items-center gap-2">
              <Input
                type="number"
                min={1}
                max={3650}
                value={days}
                onChange={(e) => setDays(e.target.value)}
                aria-label={t('daysInputLabel')}
                className="w-24"
                data-testid="retention-days-input"
              />
              <span className="text-sm text-muted-foreground">
                {t('daysUnit')}
              </span>
            </div>
          </div>
        )}

        {needsConfirm && (
          <div
            className="space-y-2 rounded-md border border-danger bg-danger/5 p-3 text-sm"
            data-testid="retention-impact-warning"
          >
            <p className="font-medium text-danger">{t('impact.title')}</p>
            <p>
              {t('impact.counts', {
                posts: impact?.posts_to_delete ?? 0,
                history: impact?.history_to_delete ?? 0,
              })}
            </p>
            {needsTypeConfirm && (
              <input
                className="w-full rounded-md border px-3 py-1.5 text-sm"
                placeholder={t('impact.typeConfirm')}
                value={confirmText}
                onChange={(e) => setConfirmText(e.target.value)}
                data-testid="retention-type-confirm"
              />
            )}
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {tCommon('cancel')}
          </Button>
          <Button
            variant={needsConfirm ? 'danger' : 'default'}
            disabled={
              apply.isPending ||
              impactLoading ||
              (segment === 'days' && !daysValue) ||
              (needsTypeConfirm && confirmText !== 'DELETE')
            }
            onClick={() => void onConfirm()}
            data-testid="retention-confirm"
          >
            {impactLoading
              ? t('checking')
              : needsConfirm
                ? t('confirmDelete')
                : t('confirmSet')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export default RetentionDialog
