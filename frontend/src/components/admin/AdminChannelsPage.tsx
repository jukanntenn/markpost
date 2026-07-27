'use client'

import { useTranslations } from 'next-intl'
import { adminApi, adminKeys } from '@/lib/api'
import { useAdminTablePage } from '@/hooks/useAdminTablePage'
import { formatToLocalTime } from '@/utils/time'
import { TableHead, TableRow, TableCell } from '@/components/ui/table'
import { AdminTablePage } from '@/components/admin/AdminTablePage'
import { PaginationControls } from '@/components/ui/pagination-controls'

export function AdminChannelsPage() {
  const t = useTranslations('admin')

  const {
    items: channels,
    pagination,
    onPageChange,
    ...queryState
  } = useAdminTablePage({
    queryKey: adminKeys.channels.all(),
    queryFn: (page, limit) => adminApi.listChannels(page, limit),
    t,
  })

  return (
    <>
      <AdminTablePage
        title={t('channels.title')}
        {...queryState}
        emptyText={t('channels.empty')}
        headers={
          <>
            <TableHead>{t('channels.id')}</TableHead>
            <TableHead>{t('channels.name')}</TableHead>
            <TableHead>{t('channels.kind')}</TableHead>
            <TableHead>{t('channels.enabled')}</TableHead>
            <TableHead>{t('createdAt')}</TableHead>
          </>
        }
        colSpan={5}
        items={channels}
        renderRow={(channel) => (
          <TableRow key={channel.id}>
            <TableCell>{channel.id}</TableCell>
            <TableCell>{channel.name}</TableCell>
            <TableCell>{channel.kind}</TableCell>
            <TableCell>
              {channel.enabled ? t('channels.enabled') : '-'}
            </TableCell>
            <TableCell>{formatToLocalTime(channel.created_at)}</TableCell>
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

export default AdminChannelsPage
