'use client'

import { useMemo, useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useTranslations } from 'next-intl'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { deliveryKeys, deliveryApi, postKeyKeys } from '@/lib/api'
import { useAuthStore } from '@/stores/auth'
import { usePostKey } from '@/hooks/usePostKey'
import { useLocaleContext } from '@/components/providers/LocaleProvider'
import { ListState } from '@/components/ui/list-state'
import { EmptyState } from '@/components/ui/empty-state'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button, buttonClass } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { PipelineStatus, type PipelineState } from './PipelineStatus'
import { EventUnit, type ActivityEvent } from './EventUnit'
import { ChannelHealth } from './ChannelHealth'
import { DeliveryTrendChart } from './DeliveryTrendChart'
import CreateTestPostModal from '@/components/CreateTestPostModal'
import { KeyRoundIcon, SendIcon, FileTextIcon } from 'lucide-react'
import { cn } from '@/lib/utils'

// B2 Dashboard 主副栏版式（B2.3）+ 管道状态机（B2.4）+ 活动流（B2.5/K.2）
// + 渠道健康（B2.6）+ 趋势图（B2.7）+ 空态/onboarding（B2.8/I.9）。
export function DashboardPage() {
  const t = useTranslations('dashboard')
  const tCommon = useTranslations('common')
  const tPostKey = useTranslations('postKey')
  const router = useRouter()
  const queryClient = useQueryClient()
  const user = useAuthStore((state) => state.user)
  const { locale } = useLocaleContext()

  const [showTestModal, setShowTestModal] = useState(false)
  const [onboardingStep2Seen] = useState(
    () =>
      typeof window !== 'undefined' &&
      window.localStorage.getItem('markpost_onboarding_postkey') === '1',
  )

  // H.3：活动流第 1 页 3s 轮询（现状保留）；stats/trend 60s 不轮询。
  const statsQuery = useQuery({
    queryKey: deliveryKeys.trend(7),
    queryFn: () => deliveryApi.stats(7),
    staleTime: 60_000,
  })
  const channelsQuery = useQuery({
    queryKey: deliveryKeys.channels(),
    queryFn: () => deliveryApi.list(),
    staleTime: 60_000,
  })
  const latestQuery = useQuery({
    queryKey: deliveryKeys.latest(),
    queryFn: () => deliveryApi.latestPerChannel(),
    staleTime: 60_000,
  })
  const historyQuery = useQuery({
    queryKey: deliveryKeys.history(1, 20, 0, 'all'),
    queryFn: () => deliveryApi.listHistory(1, 20),
    refetchInterval: 3_000,
  })
  const pendingQuery = useQuery({
    queryKey: deliveryKeys.pending(),
    queryFn: () => deliveryApi.pending(),
    refetchInterval: 3_000,
  })
  const postKeyQuery = usePostKey()

  const channels = useMemo(
    () => channelsQuery.data?.items ?? [],
    [channelsQuery.data],
  )
  const latest = useMemo(
    () => latestQuery.data?.items ?? [],
    [latestQuery.data],
  )
  const pending = useMemo(
    () => pendingQuery.data?.items ?? [],
    [pendingQuery.data],
  )
  const history = useMemo(
    () => historyQuery.data?.items ?? [],
    [historyQuery.data],
  )
  const today = statsQuery.data?.today

  // B2.4/K.7 B2-2 管道状态机。
  const pipeline: PipelineState = useMemo(() => {
    if (channels.length === 0) return 'unconfigured'
    const failedToday = (today?.failed ?? 0) > 0
    const deliveredToday = (today?.delivered ?? 0) > 0
    const hasAbnormalChannel = latest.some(
      (l) => l.status === 'failed' || l.status === 'expired',
    )
    if (failedToday && !deliveredToday && (today?.pending ?? 0) === 0)
      return 'allFailed'
    if (failedToday || hasAbnormalChannel) return 'partialFailure'
    if (deliveredToday || (today?.pending ?? 0) > 0) return 'running'
    return 'idle'
  }, [channels, latest, today])

  // B2.5 活动流聚合：history 分组 ∪ pending 分组（K.2）。
  const events: ActivityEvent[] = useMemo(() => {
    const byQid = new Map<string, ActivityEvent>()
    for (const h of history) {
      const qid = h.post_qid ?? `deleted-${h.id}`
      const existing = byQid.get(qid)
      if (existing) {
        existing.deliveries.push(h)
        if (existing.createdAt < h.created_at) existing.createdAt = h.created_at
      } else {
        byQid.set(qid, {
          postQid: qid,
          postTitle: h.post_title ?? tCommon('empty'),
          createdAt: h.created_at,
          deliveries: [h],
          pendingCount: 0,
        })
      }
    }
    for (const p of pending) {
      const existing = byQid.get(p.post_qid)
      if (existing) {
        existing.pendingCount += 1
        if (existing.createdAt < p.created_at) existing.createdAt = p.created_at
      } else {
        byQid.set(p.post_qid, {
          postQid: p.post_qid,
          postTitle: p.post_title,
          createdAt: p.created_at,
          deliveries: [],
          pendingCount: 1,
        })
      }
    }
    return [...byQid.values()].sort(
      (a, b) =>
        new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
    )
  }, [history, pending, tCommon])

  // I.9 onboarding 判定：无渠道 且 无帖子/投递。
  const hasAnyActivity = events.length > 0
  const showOnboarding = channels.length === 0 && !hasAnyActivity

  // B2.8 空态判定：无渠道但有内容。
  const showNoChannelEmpty = channels.length === 0 && hasAnyActivity

  const stepDone: [boolean, boolean, boolean] = [
    channels.length > 0,
    onboardingStep2Seen,
    hasAnyActivity,
  ]

  const loading = statsQuery.isLoading && channelsQuery.isLoading

  return (
    <div className="mx-auto max-w-5xl">
      <header className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <h1 className="font-display text-headline font-bold tracking-tight">
          {t('welcome', { name: user?.username ?? '' })}
        </h1>
        <span className="text-sm text-muted-foreground">
          {new Date().toLocaleDateString(locale, {
            month: 'long',
            day: 'numeric',
            weekday: 'short',
          })}
        </span>
      </header>

      {loading ? (
        <DashboardSkeleton />
      ) : showOnboarding ? (
        <OnboardingCard
          stepDone={stepDone}
          onCreateChannel={() => router.push('/delivery/channels')}
          onViewPostKey={() => {
            localStorage.setItem('markpost_onboarding_postkey', '1')
            router.push('/post-key')
          }}
          onSendTest={() => setShowTestModal(true)}
        />
      ) : (
        <>
          {/* 摘要带（B2.4） */}
          <div className="mb-6">
            <PipelineStatus
              state={pipeline}
              today={today}
              onTodayClick={() => router.push('/delivery/history')}
            />
          </div>

          {showNoChannelEmpty && (
            <Card className="mb-6">
              <CardContent className="pt-6">
                <EmptyState
                  icon={SendIcon}
                  title={t('empty.noChannel.title')}
                  description={t('empty.noChannel.desc')}
                  action={
                    <Link
                      href="/delivery/channels"
                      className={buttonClass('default')}
                    >
                      {t('empty.noChannel.cta')}
                    </Link>
                  }
                />
              </CardContent>
            </Card>
          )}

          <div className="grid gap-6 xl:grid-cols-[1fr_320px]">
            {/* 主区：活动叙事 */}
            <section aria-label={t('activity.title')}>
              <div className="mb-3 flex items-center justify-between">
                <h2 className="font-display text-section font-bold">
                  {t('activity.title')}
                </h2>
                <Link
                  href="/delivery/history"
                  className={buttonClass('ghost', 'sm')}
                >
                  {t('activity.viewAll')}
                </Link>
              </div>
              <ListState
                isLoading={historyQuery.isLoading && events.length === 0}
                error={historyQuery.error}
                loadingSkeleton={<EventListSkeleton />}
                emptyWhen={events.length === 0}
                empty={
                  <EmptyState
                    icon={FileTextIcon}
                    title={t('recentPosts.empty')}
                    description={t('empty.noChannel.desc')}
                  />
                }
                onRetry={() => historyQuery.refetch()}
              >
                <div className="space-y-3">
                  {events.map((e) => (
                    <EventUnit key={e.postQid} event={e} />
                  ))}
                </div>
              </ListState>
            </section>

            {/* 副区：度量（xl 起 sticky 常驻，B2.3） */}
            <aside className="space-y-6 xl:sticky xl:top-[calc(var(--header-height)+1rem)] xl:self-start">
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">
                    {t('channelHealth.title')}
                  </CardTitle>
                </CardHeader>
                <CardContent className="pt-0">
                  <ChannelHealth
                    channels={channels}
                    latest={latest}
                    isLoading={channelsQuery.isLoading}
                  />
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="text-base">
                    {t('trend.title')}
                  </CardTitle>
                </CardHeader>
                <CardContent className="pt-0">
                  <ListState
                    isLoading={statsQuery.isLoading}
                    error={statsQuery.error}
                    loadingSkeleton={<Skeleton className="h-56 w-full" />}
                    onRetry={() => statsQuery.refetch()}
                  >
                    <DeliveryTrendChart data={statsQuery.data?.trend ?? []} />
                  </ListState>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="text-base">
                    <Link
                      href="/post-key"
                      className="flex items-center gap-2 hover:underline"
                    >
                      <KeyRoundIcon className="size-4 text-primary" />
                      {tPostKey('title')}
                    </Link>
                  </CardTitle>
                </CardHeader>
                <CardContent className="pt-0">
                  <p className="text-sm text-muted-foreground">
                    {tPostKey('explanation')}
                  </p>
                  <div className="mt-3 flex flex-wrap gap-2">
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => setShowTestModal(true)}
                    >
                      {tPostKey('testPost')}
                    </Button>
                    <Link
                      href="/post-key"
                      className={buttonClass('outline', 'sm')}
                    >
                      {tPostKey('rotate')}
                    </Link>
                  </div>
                </CardContent>
              </Card>
            </aside>
          </div>
        </>
      )}

      {postKeyQuery.data?.post_key && (
        <CreateTestPostModal
          show={showTestModal}
          postKey={postKeyQuery.data.post_key}
          onHide={() => setShowTestModal(false)}
          onSuccess={() => {
            queryClient.invalidateQueries({ queryKey: postKeyKeys.all() })
            queryClient.invalidateQueries({ queryKey: ['delivery'] })
          }}
        />
      )}
    </div>
  )
}

function DashboardSkeleton() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-10 w-64" />
      <Skeleton className="h-20 w-full" />
      <div className="grid gap-6 xl:grid-cols-[1fr_320px]">
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-28 w-full" />
          ))}
        </div>
        <div className="space-y-6">
          <Skeleton className="h-40 w-full" />
          <Skeleton className="h-56 w-full" />
        </div>
      </div>
    </div>
  )
}

function EventListSkeleton() {
  return (
    <div className="space-y-3">
      {Array.from({ length: 3 }).map((_, i) => (
        <Skeleton key={i} className="h-28 w-full" />
      ))}
    </div>
  )
}

// I.9 三步引导态（渠道 → Post Key → 第一篇文章），进度 ●○○。
function OnboardingCard({
  stepDone,
  onCreateChannel,
  onViewPostKey,
  onSendTest,
}: {
  stepDone: [boolean, boolean, boolean]
  onCreateChannel: () => void
  onViewPostKey: () => void
  onSendTest: () => void
}) {
  const t = useTranslations('dashboard.onboarding')
  const doneCount = stepDone.filter(Boolean).length

  const steps = [
    {
      title: t('step1Title'),
      desc: t('step1Desc'),
      cta: t('step1Cta'),
      done: stepDone[0],
      action: onCreateChannel,
    },
    {
      title: t('step2Title'),
      desc: t('step2Desc'),
      cta: t('step2Cta'),
      done: stepDone[1],
      action: onViewPostKey,
    },
    {
      title: t('step3Title'),
      desc: t('step3Desc'),
      cta: t('step3Cta'),
      done: stepDone[2],
      action: onSendTest,
    },
  ]

  return (
    <Card>
      <CardContent className="space-y-4 pt-6">
        <div className="flex items-center justify-between">
          <h2 className="font-display text-section font-bold">{t('title')}</h2>
          <span className="text-sm text-muted-foreground" aria-live="polite">
            {t('progress', { done: doneCount, total: 3 })}
          </span>
        </div>
        <ol className="space-y-3">
          {steps.map((s, i) => (
            <li
              key={i}
              className={cn(
                'flex items-center gap-3 rounded-lg border p-4',
                s.done && 'border-success/40 bg-success/5',
              )}
            >
              <span
                className={cn(
                  'flex size-7 shrink-0 items-center justify-center rounded-full text-xs font-bold',
                  s.done
                    ? 'bg-success text-success-foreground'
                    : 'bg-muted text-muted-foreground',
                )}
                aria-hidden="true"
              >
                {i + 1}
              </span>
              <div className="min-w-0 flex-1">
                <p className="font-semibold">{s.title}</p>
                <p className="text-sm text-muted-foreground">{s.desc}</p>
              </div>
              {!s.done && (
                <Button type="button" size="sm" onClick={s.action}>
                  {s.cta}
                </Button>
              )}
            </li>
          ))}
        </ol>
      </CardContent>
    </Card>
  )
}

export default DashboardPage
