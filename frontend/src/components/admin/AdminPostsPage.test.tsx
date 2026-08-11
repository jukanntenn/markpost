import '@testing-library/jest-dom'
import { describe, expect, it, beforeEach } from 'vitest'
import { screen } from '@testing-library/react'
import {
  renderWithProviders,
  mockMatchMedia,
  mockNavigation,
} from '@/test/utils'
import AdminPostsPage from './AdminPostsPage'
import { server } from '@/mocks/server'
import { http, HttpResponse } from 'msw'

beforeEach(() => {
  mockMatchMedia()
  mockNavigation()
})

// F.9 Admin 帖子管理：标题搜索 + 用户名筛选 + 相对路径外链（端口修复）。
describe('AdminPostsPage', () => {
  it('renders the page heading', async () => {
    renderWithProviders(<AdminPostsPage />)
    expect(
      await screen.findByRole('heading', { name: /posts/i }),
    ).toBeInTheDocument()
  })

  it('displays post rows with relative-path links (no hardcoded port)', async () => {
    renderWithProviders(<AdminPostsPage />)
    const links = await screen.findAllByRole('link', { name: 'First Post' })
    for (const link of links) {
      expect(link).toHaveAttribute('href', '/p-1')
    }
  })

  it('shows empty state when no posts exist', async () => {
    server.use(
      http.get('/api/v1/admin/posts', () =>
        HttpResponse.json({
          items: [],
          total: 0,
          page: 1,
          limit: 20,
          total_pages: 0,
        }),
      ),
    )
    renderWithProviders(<AdminPostsPage />)
    expect(await screen.findByText(/no posts found/i)).toBeInTheDocument()
  })

  it('shows delete confirmation dialog with post title when delete clicked', async () => {
    renderWithProviders(<AdminPostsPage />)
    const deleteButtons = await screen.findAllByRole('button', {
      name: 'Delete',
    })
    deleteButtons[0].click()
    expect(
      await screen.findByText(/delete "First Post"\?/i),
    ).toBeInTheDocument()
  })
})
