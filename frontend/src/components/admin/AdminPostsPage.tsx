'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Trash2Icon } from 'lucide-react'
import { adminApi, adminKeys, invalidateKey } from '@/lib/api'
import { mutationOptions } from '@/lib/mutation-helpers'
import { toast } from '@/stores/toast'
import { useAdminSearchTablePage } from '@/hooks/useAdminTablePage'
import { formatToLocalTime } from '@/utils/time'
import { SearchInput } from '@/components/ui/search-input'
import { Button } from '@/components/ui/button'
import { TableHead, TableRow, TableCell } from '@/components/ui/table'
import { AdminTablePage } from '@/components/admin/AdminTablePage'
import { PaginationControls } from '@/components/ui/pagination-controls'
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

export function AdminPostsPage() {
  const t = useTranslations('admin')
  const queryClient = useQueryClient()
  const [deleteTarget, setDeleteTarget] = useState<{
    qid: string
    title: string
  } | null>(null)

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

  const deleteMutation = useMutation(
    mutationOptions({
      mutationFn: (qid: string) => adminApi.deletePost(qid),
      onSuccess: () => {
        invalidateKey(queryClient, adminKeys.posts.list(''))
        invalidateKey(queryClient, adminKeys.stats())
        toast.success(t('posts.deleted'))
        setDeleteTarget(null)
      },
    })
  )

  const backendUrl =
    typeof window !== 'undefined'
      ? window.location.origin.replace(':3034', ':7330')
      : 'http://localhost:7330'

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
                href={`${backendUrl}/${post.qid}`}
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
              <Button
                variant="ghost"
                size="icon"
                onClick={() =>
                  setDeleteTarget({ qid: post.qid, title: post.title })
                }
              >
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

      <AlertDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('posts.deleteTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {deleteTarget?.title
                ? t('posts.deleteConfirm', { title: deleteTarget.title })
                : t('posts.deleteConfirmEmpty')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('users.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() =>
                deleteTarget && deleteMutation.mutate(deleteTarget.qid)
              }
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {t('posts.delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}

export default AdminPostsPage
