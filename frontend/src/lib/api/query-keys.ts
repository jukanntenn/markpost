import type { QueryClient } from '@tanstack/react-query'

// H.3 标准 query key 工厂：所有 key 是数组结构，层级 [domain, scope, ...params]。
// invalidate 清单见 H.3（mutation 成功后的失效表）。
export function invalidateKey(
  queryClient: QueryClient,
  queryKey: readonly unknown[],
) {
  return queryClient.invalidateQueries({ queryKey })
}

export const postKeys = {
  all: () => ['posts'] as const,
  list: (page: number, limit: number, search?: string) =>
    [...postKeys.all(), 'list', page, limit, search ?? ''] as const,
  detail: (id: string) => [...postKeys.all(), 'detail', id] as const,
}

export const deliveryKeys = {
  all: () => ['delivery'] as const,
  channels: () => [...deliveryKeys.all(), 'channels'] as const,
  history: (page: number, limit: number, channelId?: number, status?: string) =>
    [
      ...deliveryKeys.all(),
      'history',
      page,
      limit,
      channelId ?? 0,
      status ?? 'all',
    ] as const,
  latest: () => [...deliveryKeys.all(), 'latest'] as const,
  trend: (days: number) => [...deliveryKeys.all(), 'trend', days] as const,
  pending: () => [...deliveryKeys.all(), 'pending'] as const,
  channelDetail: (id: number) =>
    [...deliveryKeys.all(), 'channel', id] as const,
}

export const adminKeys = {
  all: () => ['admin'] as const,
  settings: () => [...adminKeys.all(), 'settings'] as const,
  retention: {
    defaults: () => [...adminKeys.all(), 'retention', 'defaults'] as const,
  },
  users: {
    all: () => [...adminKeys.all(), 'users'] as const,
    list: (page: number, search?: string) =>
      [...adminKeys.users.all(), 'list', page, search ?? ''] as const,
    detail: (id: number) => [...adminKeys.users.all(), 'detail', id] as const,
  },
  posts: {
    all: () => [...adminKeys.all(), 'posts'] as const,
    list: (page: number, search?: string, username?: string) =>
      [
        ...adminKeys.posts.all(),
        'list',
        page,
        search ?? '',
        username ?? '',
      ] as const,
  },
  channels: {
    all: () => [...adminKeys.all(), 'channels'] as const,
    list: (page: number) =>
      [...adminKeys.channels.all(), 'list', page] as const,
  },
  history: {
    all: () => [...adminKeys.all(), 'history'] as const,
    list: (page: number, filter?: Record<string, string | number>) =>
      [...adminKeys.history.all(), 'list', page, filter ?? {}] as const,
  },
  audit: {
    all: () => [...adminKeys.all(), 'audit'] as const,
    list: (page: number, filter?: Record<string, string | number>) =>
      [...adminKeys.audit.all(), 'list', page, filter ?? {}] as const,
  },
  stats: () => [...adminKeys.all(), 'stats'] as const,
  trend: (days: number) => [...adminKeys.all(), 'trend', days] as const,
  lockedChannels: () => [...adminKeys.all(), 'locked-channels'] as const,
  sessions: (userId: number) =>
    [...adminKeys.all(), 'sessions', userId] as const,
}

export const postKeyKeys = {
  all: () => ['post-key'] as const,
  current: () => [...postKeyKeys.all()] as const,
}

export const sessionsKeys = {
  all: () => ['auth', 'sessions'] as const,
  current: () => [...sessionsKeys.all()] as const,
}
