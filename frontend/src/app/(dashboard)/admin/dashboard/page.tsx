'use client'

import { useTranslations } from 'next-intl'
import { useQuery } from '@tanstack/react-query'
import { adminApi, adminKeys } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { UsersIcon, FileTextIcon, RadioIcon, HistoryIcon } from 'lucide-react'

export default function AdminDashboardPage() {
  const t = useTranslations('admin')

  const { data: stats, isLoading } = useQuery({
    queryKey: adminKeys.stats(),
    queryFn: () => adminApi.getStats(),
  })

  const counts = stats?.counts ?? {
    users: 0,
    posts: 0,
    channels: 0,
    history: 0,
  }

  const cards = [
    { title: t('dashboard.users'), value: counts.users, icon: UsersIcon },
    { title: t('dashboard.posts'), value: counts.posts, icon: FileTextIcon },
    { title: t('dashboard.channels'), value: counts.channels, icon: RadioIcon },
    { title: t('dashboard.history'), value: counts.history, icon: HistoryIcon },
  ]

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t('dashboard.title')}</h1>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {cards.map((card) => {
          const Icon = card.icon
          return (
            <Card key={card.title}>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">
                  {card.title}
                </CardTitle>
                <Icon className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {isLoading ? '-' : card.value}
                </div>
              </CardContent>
            </Card>
          )
        })}
      </div>
    </div>
  )
}
