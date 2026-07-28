'use client'

import { useTranslations } from 'next-intl'

import { Badge } from '@/components/ui/badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { truncate, cn } from '@/lib/utils'
import { buildPostUrl } from '@/utils/url'
import { formatToLocalTime } from '@/utils/time'
import type { DeliveryHistoryItem, DeliveryStatus } from '@/types/delivery'

const statusVariant: Record<
  DeliveryStatus,
  'secondary' | 'destructive' | 'outline'
> = {
  delivered: 'secondary',
  failed: 'destructive',
  expired: 'outline',
}

interface DeliveryHistoryTableProps {
  items: DeliveryHistoryItem[]
}

export function DeliveryHistoryTable({ items }: DeliveryHistoryTableProps) {
  const t = useTranslations('delivery')

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('history.colPost')}</TableHead>
          <TableHead>{t('history.colChannel')}</TableHead>
          <TableHead>{t('history.colStatus')}</TableHead>
          <TableHead>{t('history.colTime')}</TableHead>
          <TableHead>{t('history.colError')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.length === 0 ? (
          <TableRow>
            <TableCell
              colSpan={5}
              className="h-24 text-center text-sm text-muted-foreground"
            >
              {t('history.empty')}
            </TableCell>
          </TableRow>
        ) : (
          items.map((item) => {
            const showError =
              (item.status === 'failed' || item.status === 'expired') &&
              item.last_error
            return (
              <TableRow key={item.id}>
                <TableCell className="max-w-[200px]">
                  {item.post_qid ? (
                    <a
                      href={buildPostUrl(item.post_qid)}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="truncate font-medium hover:underline"
                    >
                      {item.post_title ?? item.post_qid}
                    </a>
                  ) : (
                    <span className="text-muted-foreground italic">
                      {t('history.postDeleted')}
                    </span>
                  )}
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  {item.channel_name ?? t('history.channelDeleted')}
                </TableCell>
                <TableCell>
                  <Badge variant={statusVariant[item.status]}>
                    {t(`history.status_${item.status}`)}
                  </Badge>
                </TableCell>
                <TableCell className="whitespace-nowrap text-sm text-muted-foreground">
                  {formatToLocalTime(item.created_at, {
                    includeSeconds: false,
                  })}
                </TableCell>
                <TableCell className="max-w-[220px]">
                  {showError ? (
                    <span
                      className={cn('block truncate text-xs text-destructive')}
                      title={item.last_error}
                    >
                      {truncate(item.last_error, 60)}
                    </span>
                  ) : (
                    <span className="text-muted-foreground">—</span>
                  )}
                </TableCell>
              </TableRow>
            )
          })
        )}
      </TableBody>
    </Table>
  )
}
