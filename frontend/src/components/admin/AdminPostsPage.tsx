'use client'

import { useTranslations } from 'next-intl'
import { Trash2Icon } from 'lucide-react'
import { adminApi, adminKeys } from '@/lib/api'
import { useAdminSearchTablePage } from '@/hooks/useAdminTablePage'
import { formatToLocalTime } from '@/utils/time'
import { buildPostUrl } from '@/utils/url'
import { SearchInput } from '@/components/ui/search-input'
import { Button } from '@/components/ui/button'
import { TableHead, TableRow, TableCell } from '@/components/ui/table'
import { AdminTablePage } from '@/components/admin/AdminTablePage'
import { PaginationControls } from '@/components/ui/pagination-controls'

export function AdminPostsPage() {
  const t = useTranslations('admin')
  const {
    items: posts,
    search,
    setSearch,
    pagination,
    onPageChange,
    ...queryState
  } = useAdminSearchTablePage({
    queryKeyBuilder: adminKeys.posts.list,
    queryFn: adminApi.listPosts,
    t,
  })

  return (
    <>
      <AdminTablePage
        title={t('posts.title')}
        toolbar={
          <SearchInput
            placeholder={t('posts.searchPlaceholder')}
            value={search}
            onChange={setSearch}
          />
        }
        {...queryState}
        emptyText={t('posts.empty')}
        headers={
          <>
            <TableHead>{t('posts.id')}</TableHead>
            <TableHead>{t('posts.titleCol')}</TableHead>
            <TableHead>{t('username')}</TableHead>
            <TableHead>{t('createdAt')}</TableHead>
            <TableHead className="w-10"></TableHead>
          </>
        }
        colSpan={5}
        items={posts}
        renderRow={(post) => (
          <TableRow key={post.qid}>
            <TableCell>{post.qid.slice(0, 8)}...</TableCell>
            <TableCell>
              <a
                href={buildPostUrl(post.qid)}
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary underline-offset-4 hover:underline"
              >
                {post.title}
              </a>
            </TableCell>
            <TableCell>{post.username}</TableCell>
            <TableCell>{formatToLocalTime(post.created_at)}</TableCell>
            <TableCell>
              <Button variant="ghost" size="icon">
                <Trash2Icon className="size-4 text-destructive" />
              </Button>
            </TableCell>
          </TableRow>
        )}
      />
      {pagination && pagination.total_pages > 1 && (
        <PaginationControls
          page={pagination.page}
          totalPages={pagination.total_pages}
          onPageChange={onPageChange}
          prevLabel={t('previous')}
          nextLabel={t('next')}
        />
      )}
    </>
  )
}

export default AdminPostsPage
