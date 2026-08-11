'use client'

import { useTranslations } from 'next-intl'
import { useQuery } from '@tanstack/react-query'
import { FileTextIcon, RefreshCwIcon } from 'lucide-react'
import { postsApi, postKeys } from '@/lib/api'
import { DEFAULT_PAGE_SIZE } from '@/lib/constants'
import { useUrlQueryState } from '@/hooks/useUrlQueryState'
import { useDebouncedValue } from '@/hooks/useDebouncedValue'
import { PostListItemRow } from './PostListItemRow'
import { ListState } from '@/components/ui/list-state'
import { EmptyState } from '@/components/ui/empty-state'
import { Card, CardContent } from '@/components/ui/card'
import { PaginationControls } from '@/components/ui/pagination-controls'
import { PageHeading } from '@/components/ui/page-heading'
import { SearchInput } from '@/components/ui/search-input'
import { Skeleton } from '@/components/ui/skeleton'

// F.5 帖子列表页：标题搜索（debounce，B3.3）+ 骨架 + 分页增强（页码/总条数）。
// 筛选/分页进 URL query（H.6）。第 1 页 3s 轮询保留（H.3）。
export function PostsPage() {
  const t = useTranslations('posts')
  const tCommon = useTranslations('common')
  const { state, setState, setPage } = useUrlQueryState<{
    page: string
    search: string
  }>({ page: '1', search: '' })

  const page = Math.max(1, Number.parseInt(state.page, 10) || 1)
  const debouncedSearch = useDebouncedValue(state.search, 300)
  const limit = DEFAULT_PAGE_SIZE

  const query = useQuery({
    queryKey: postKeys.list(page, limit, debouncedSearch),
    queryFn: () => postsApi.list(page, limit, debouncedSearch),
    refetchInterval: page === 1 && !debouncedSearch ? 3000 : undefined,
    refetchOnWindowFocus: true,
    staleTime: 60_000,
  })

  const posts = query.data?.items ?? []
  const total = query.data?.total ?? 0
  const totalPages = query.data?.total_pages ?? 0

  return (
    <div>
      <PageHeading
        actions={
          <button
            type="button"
            onClick={() => query.refetch()}
            aria-label={tCommon('refresh')}
            className="flex size-11 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground focus-visible:outline-2 focus-visible:-outline-offset-1 focus-visible:outline-ring"
          >
            <RefreshCwIcon className="size-4" />
          </button>
        }
      >
        {t('title')}
      </PageHeading>

      <div className="mb-4">
        <SearchInput
          placeholder={t('searchPlaceholder')}
          value={state.search}
          onChange={(v) => setState({ search: v })}
        />
      </div>

      <Card>
        <CardContent>
          <ListState
            isLoading={query.isLoading}
            error={query.error}
            loadingSkeleton={<PostsSkeleton />}
            emptyWhen={posts.length === 0}
            empty={
              <EmptyState
                icon={FileTextIcon}
                title={t('empty')}
                description={t('emptyHint')}
              />
            }
            onRetry={() => query.refetch()}
          >
            <ul className="divide-y">
              {posts.map((p) => (
                <PostListItemRow key={p.id} post={p} showSeconds={false} />
              ))}
            </ul>

            <PaginationControls
              page={page}
              totalPages={totalPages}
              total={total}
              onPageChange={setPage}
              prevLabel={t('pagination.prev')}
              nextLabel={t('pagination.next')}
              totalLabel={(n) => t('pagination.total', { n })}
            />
          </ListState>
        </CardContent>
      </Card>
    </div>
  )
}

function PostsSkeleton() {
  return (
    <ul className="divide-y">
      {Array.from({ length: 5 }).map((_, i) => (
        <li key={i} className="py-3">
          <Skeleton className="h-4 w-2/3" />
        </li>
      ))}
    </ul>
  )
}

export default PostsPage
