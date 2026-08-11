'use client'

import { useTranslations } from 'next-intl'
import {
  ResponsiveContainer,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
} from 'recharts'
import type { DailyStat } from '@/types/delivery'
import { EmptyState } from '@/components/ui/empty-state'
import { LineChartIcon } from 'lucide-react'

// B2.7 投递趋势图：双线（成功=chart-1 绿 / 失败=chart-2 红），
// 悬浮 tooltip 显示当天具体数，图例可切换显隐系列。
export function DeliveryTrendChart({ data }: { data: DailyStat[] }) {
  const t = useTranslations('dashboard.trend')

  const rows = data.map((d) => ({
    day: d.day,
    success: d.delivered,
    failed: d.failed + d.expired,
  }))

  if (rows.length === 0) {
    return (
      <EmptyState
        icon={LineChartIcon}
        title={t('empty')}
        className="border-0 py-10"
      />
    )
  }

  return (
    <div className="h-56 w-full" role="img" aria-label={t('title')}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart
          data={rows}
          margin={{ top: 8, right: 8, bottom: 0, left: -18 }}
        >
          <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
          <XAxis
            dataKey="day"
            tick={{ fontSize: 11, fill: 'var(--muted-foreground)' }}
            tickFormatter={(v: string) => v.slice(5)}
            stroke="var(--border)"
          />
          <YAxis
            allowDecimals={false}
            tick={{ fontSize: 11, fill: 'var(--muted-foreground)' }}
            stroke="var(--border)"
          />
          <Tooltip
            contentStyle={{
              background: 'var(--popover)',
              border: '1px solid var(--border)',
              borderRadius: 8,
              fontSize: 12,
              color: 'var(--popover-foreground)',
            }}
          />
          <Legend wrapperStyle={{ fontSize: 12 }} />
          <Line
            type="monotone"
            dataKey="success"
            name={t('success')}
            stroke="var(--chart-1)"
            strokeWidth={2}
            dot={{ r: 2 }}
            activeDot={{ r: 4 }}
          />
          <Line
            type="monotone"
            dataKey="failed"
            name={t('failed')}
            stroke="var(--chart-2)"
            strokeWidth={2}
            dot={{ r: 2 }}
            activeDot={{ r: 4 }}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}
