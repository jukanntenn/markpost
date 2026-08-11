'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { useQuery } from '@tanstack/react-query'
import { FileTextIcon } from 'lucide-react'
import { adminApi, adminKeys } from '@/lib/api'
import { useUrlQueryState } from '@/hooks/useUrlQueryState'
import { useDebouncedValue } from '@/hooks/useDebouncedValue'
import { relativeTime } from '@/utils/relative-time'
import { useLocaleContext } from '@/components/providers/LocaleProvider'
import { PageHeading } from '@/components/ui/page-heading'
import { Button } from '@/components/ui/button'
import { SearchInput } from '@/components/ui/search-input'
import { ListState } from '@/components/ui/list-state'
import { EmptyState } from '@/components/ui/empty-state'
import { Skeleton } from '@/components/ui/skeleton'
import { PaginationControls } from '@/components/ui/pagination-controls'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toastManager } from '@/stores/toast'
import type { AdminPost } from '@/types/posts'

// F.9 Admin 帖子管理页：标题搜索 + 用户名筛选 + 删除（AlertDialog）+ 移动卡片。
// 修复：帖子外链用相对路径 /{qid}（删除硬编码端口替换）。
export function AdminPostsPage() {
  const t = useTranslations('admin.posts')
  const tCommon = useTranslations('common')
  const { locale } = useLocaleContext()
  const queryClient = useQueryClient()

  const { state, setState, setPage } = useUrlQueryState<{
    page: string
    search: string
    username: string
  }>({ page: '1', search: '', username: '' })

  const page = Math.max(1, Number.parseInt(state.page, 10) || 1)
  const debouncedSearch = useDebouncedValue(state.search, 300)
  const debouncedUsername = useDebouncedValue(state.username, 300)

  const [deleteTarget, setDeleteTarget] = useState<AdminPost | null>(null)

  const query = useQuery({
    queryKey: adminKeys.posts.list(page, debouncedSearch, debouncedUsername),
    queryFn: () => adminApi.listPosts(page, debouncedSearch, debouncedUsername),
    staleTime: 30_000,
  })

  const deleteMutation = useMutation({
    mutationFn: (qid: string) => adminApi.deletePost(qid),
    onSuccess: () => {
      setDeleteTarget(null)
      queryClient.invalidateQueries({ queryKey: adminKeys.posts.all() })
      queryClient.invalidateQueries({ queryKey: adminKeys.stats() })
      toastManager.add({ type: 'success', title: t('deleted') })
    },
    onError: (err: Error) =>
      toastManager.add({ type: 'error', title: err.message }),
  })

  const posts = query.data?.items ?? []
  const total = query.data?.total ?? 0
  const totalPages = query.data?.total_pages ?? 0

  return (
    <div className="space-y-6">
      <PageHeading>{t('title')}</PageHeading>

      <div className="flex flex-wrap gap-3">
        <SearchInput
          placeholder={t('searchPlaceholder')}
          value={state.search}
          onChange={(v) => setState({ search: v })}
        />
        <SearchInput
          placeholder={t('filterUser')}
          value={state.username}
          onChange={(v) => setState({ username: v })}
        />
      </div>

      <ListState
        isLoading={query.isLoading}
        error={query.error}
        loadingSkeleton={<Skeleton className="h-64 w-full" />}
        emptyWhen={posts.length === 0}
        empty={<EmptyState icon={FileTextIcon} title={t('empty')} />}
        onRetry={() => query.refetch()}
      >
        <div className="hidden overflow-hidden rounded-lg border lg:block">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>QID</TableHead>
                <TableHead>{t('title')}</TableHead>
                <TableHead>{t('username')}</TableHead>
                <TableHead>{t('time')}</TableHead>
                <TableHead className="w-24 text-right"> </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {posts.map((p) => (
                <TableRow key={p.qid}>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {p.qid}
                  </TableCell>
                  <TableCell className="max-w-64">
                    <a
                      href={`/${p.qid}`}
                      target="_blank"
                      rel="noreferrer"
                      className="block truncate font-medium hover:underline"
                    >
                      {p.title}
                    </a>
                  </TableCell>
                  <TableCell>{p.username}</TableCell>
                  <TableCell className="text-muted-foreground">
                    {relativeTime(p.created_at, locale)}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="text-danger hover:text-danger"
                      onClick={() => setDeleteTarget(p)}
                    >
                      {t('delete')}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>

        <ul className="space-y-3 lg:hidden">
          {posts.map((p) => (
            <li key={p.qid} className="rounded-lg border bg-card p-4">
              <a
                href={`/${p.qid}`}
                target="_blank"
                rel="noreferrer"
                className="block truncate font-semibold hover:underline"
              >
                {p.title}
              </a>
              <p className="mt-0.5 text-xs text-muted-foreground">
                {p.username} · {relativeTime(p.created_at, locale)}
              </p>
              <div className="mt-2 flex justify-end">
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="text-danger hover:text-danger"
                  onClick={() => setDeleteTarget(p)}
                >
                  {t('delete')}
                </Button>
              </div>
            </li>
          ))}
        </ul>

        <PaginationControls
          page={page}
          totalPages={totalPages}
          total={total}
          onPageChange={setPage}
          prevLabel={tCommon('previous')}
          nextLabel={tCommon('next')}
          totalLabel={(n) => tCommon('total', { n })}
        />
      </ListState>

      <AlertDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('deleteTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {deleteTarget?.title
                ? t('deleteConfirm', { title: deleteTarget.title })
                : t('deleteConfirmEmpty')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteMutation.isPending}>
              {tCommon('cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              variant="danger"
              disabled={deleteMutation.isPending}
              onClick={() =>
                deleteTarget && deleteMutation.mutate(deleteTarget.qid)
              }
            >
              {t('delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

export default AdminPostsPage
