import { http, HttpResponse } from 'msw'
import type { PostsPaginatedResponse } from '@/types/posts'
import type { PostKeyResponse } from '@/types/auth'
import type { DeliveryChannel, DeliveryHistoryItem } from '@/types/delivery'

export const mockPostKey: PostKeyResponse = {
  post_key: 'test_key_abc123',
  created_at: '2024-01-01T00:00:00Z',
}

export const mockEmptyPosts: PostsPaginatedResponse = {
  items: [],
  total: 0,
  page: 1,
  limit: 20,
  total_pages: 0,
}

export const mockPosts: PostsPaginatedResponse = {
  items: [
    {
      id: 1,
      qid: 'p-qid-1',
      title: 'Test Post 1',
      created_at: '2024-01-01T12:00:00Z',
    },
    {
      id: 2,
      qid: 'p-qid-2',
      title: 'Test Post 2',
      created_at: '2024-01-02T13:00:00Z',
    },
  ],
  total: 2,
  page: 1,
  limit: 20,
  total_pages: 1,
}

export const handlers = [
  http.get('/api/v1/post-key', () => {
    return HttpResponse.json<PostKeyResponse>(mockPostKey)
  }),

  http.get('/api/v1/posts', ({ request }) => {
    const url = new URL(request.url)
    const page = url.searchParams.get('page')

    if (page === '2') {
      return HttpResponse.json<PostsPaginatedResponse>({
        items: [],
        total: 2,
        page: 2,
        limit: 20,
        total_pages: 1,
      })
    }

    return HttpResponse.json<PostsPaginatedResponse>(mockPosts)
  }),

  http.post('/api/v1/auth/change-password', async () => {
    return HttpResponse.json({ message: 'Password changed successfully' })
  }),

  http.post('/:postKey', async () => {
    return HttpResponse.json<{ id: string }>({ id: 'new_post_123' })
  }),

  http.post('/api/v1/auth/login', async () => {
    return HttpResponse.json({
      token: 'test_access_token',
      access_token: 'test_access_token',
      refresh_token: 'test_refresh_token',
      expires_in: 86400,
      user: { id: 1, username: 'testuser', email: 'test@example.com' },
    })
  }),

  http.post('/api/v1/auth/refresh', async () => {
    return HttpResponse.json({
      token: 'refreshed_access_token',
      access_token: 'refreshed_access_token',
      refresh_token: 'refreshed_refresh_token',
      expires_in: 86400,
    })
  }),

  http.post('/api/v1/auth/logout', async () => {
    return new HttpResponse(null, { status: 204 })
  }),

  http.get('/api/v1/oauth/url', async () => {
    return HttpResponse.json({
      url: 'https://github.com/login/oauth/authorize?mock=true',
      state: 'mock-state',
    })
  }),

  http.get('/api/v1/delivery/channels', async () => {
    return HttpResponse.json({ items: mockDeliveryChannels })
  }),

  http.post('/api/v1/delivery/channels', async ({ request }) => {
    const body = (await request.json()) as Partial<DeliveryChannel>
    const created: DeliveryChannel = {
      id: nextDeliveryChannelId++,
      kind: body.kind ?? 'feishu',
      name: body.name ?? '',
      enabled: true,
      configuration: body.configuration ?? {
        webhook_url: '',
        card_link_url: '',
      },
      keywords: body.keywords ?? '',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    mockDeliveryChannels.push(created)
    return HttpResponse.json({ channel: created }, { status: 201 })
  }),

  http.patch('/api/v1/delivery/channels/:id', async ({ request, params }) => {
    const id = Number(params.id)
    const body = (await request.json()) as Partial<DeliveryChannel>
    const channel = mockDeliveryChannels.find((c) => c.id === id)
    if (!channel) {
      return HttpResponse.json({ message: 'not found' }, { status: 404 })
    }
    Object.assign(channel, body, { updated_at: new Date().toISOString() })
    return HttpResponse.json({ channel })
  }),

  http.delete('/api/v1/delivery/channels/:id', async ({ params }) => {
    const id = Number(params.id)
    const index = mockDeliveryChannels.findIndex((c) => c.id === id)
    if (index === -1) {
      return HttpResponse.json({ message: 'not found' }, { status: 404 })
    }
    mockDeliveryChannels.splice(index, 1)
    return HttpResponse.json({ message: 'deleted' })
  }),

  http.post('/api/v1/delivery/channels/:id/test', async () => {
    return HttpResponse.json({ message: 'test message sent' })
  }),

  http.get('/api/v1/delivery/latest', async () => {
    return HttpResponse.json({ items: mockDeliveryLatest })
  }),

  http.get('/api/v1/delivery/history', async () => {
    return HttpResponse.json({
      items: mockDeliveryHistory,
      total: mockDeliveryHistory.length,
      page: 1,
      limit: 10,
      total_pages: 1,
    })
  }),

  // Admin endpoints
  http.get('/api/v1/admin/users', () => {
    return HttpResponse.json({
      items: mockAdminUsers,
      total: mockAdminUsers.length,
      page: 1,
      limit: 20,
      total_pages: 1,
    })
  }),

  http.post('/api/v1/admin/users', async ({ request }) => {
    const body = (await request.json()) as {
      email?: string
      username: string
      password: string
    }
    const created = {
      id: mockAdminUsers.length + 1,
      email: body.email ?? '',
      username: body.username,
      role: 'user',
      is_active: true,
      created_at: new Date().toISOString(),
    }
    mockAdminUsers.push(created)
    return HttpResponse.json(created, { status: 201 })
  }),

  http.get('/api/v1/admin/posts', ({ request }) => {
    const url = new URL(request.url)
    const search = url.searchParams.get('search')
    const filtered = search
      ? mockAdminPosts.filter((p) => p.title.includes(search))
      : mockAdminPosts
    return HttpResponse.json({
      items: filtered,
      total: filtered.length,
      page: 1,
      limit: 20,
      total_pages: 1,
    })
  }),

  http.delete('/api/v1/admin/posts/:qid', ({ params }) => {
    const qid = params.qid as string
    const index = mockAdminPosts.findIndex((p) => p.qid === qid)
    if (index === -1) {
      return HttpResponse.json({ message: 'not found' }, { status: 404 })
    }
    mockAdminPosts.splice(index, 1)
    return new HttpResponse(null, { status: 204 })
  }),

  http.get('/api/v1/admin/delivery/channels', () => {
    return HttpResponse.json({
      items: mockAdminChannels,
      total: mockAdminChannels.length,
      page: 1,
      limit: 20,
      total_pages: 1,
    })
  }),

  http.get('/api/v1/admin/delivery/history', () => {
    return HttpResponse.json({
      items: [],
      total: 0,
      page: 1,
      limit: 20,
      total_pages: 0,
    })
  }),

  http.get('/api/v1/admin/stats', () => {
    return HttpResponse.json({
      counts: {
        users: mockAdminUsers.length,
        posts: mockAdminPosts.length,
        channels: mockAdminChannels.length,
        history: 0,
      },
    })
  }),
]

export const mockDeliveryChannels: DeliveryChannel[] = []
export const mockDeliveryHistory: DeliveryHistoryItem[] = []
export const mockDeliveryLatest: DeliveryHistoryItem[] = []
let nextDeliveryChannelId = 1

export function resetDeliveryMocks() {
  mockDeliveryChannels.length = 0
  mockDeliveryHistory.length = 0
  mockDeliveryLatest.length = 0
  nextDeliveryChannelId = 1
}

// Admin mock data
export const mockAdminUsers = [
  {
    id: 1,
    email: 'admin@example.com',
    username: 'admin',
    role: 'admin',
    is_active: true,
    created_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 2,
    email: 'user@example.com',
    username: 'user1',
    role: 'user',
    is_active: true,
    created_at: '2024-01-02T00:00:00Z',
  },
]

export const mockAdminPosts = [
  {
    id: 1,
    qid: 'p-1',
    title: 'First Post',
    created_at: '2024-01-01T00:00:00Z',
    username: 'admin',
  },
  {
    id: 2,
    qid: 'p-2',
    title: 'Second Post',
    created_at: '2024-01-02T00:00:00Z',
    username: 'user1',
  },
]

export const mockAdminChannels = [
  {
    id: 1,
    kind: 'feishu',
    name: 'Alert Channel',
    enabled: true,
    username: 'admin',
    created_at: '2024-01-01T00:00:00Z',
  },
]
