import '@testing-library/jest-dom'
import { describe, expect, it, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import {
  renderWithProviders,
  mockMatchMedia,
  mockNavigation,
} from '@/test/utils'
import AdminUsersPage from './AdminUsersPage'
import { server } from '@/mocks/server'
import { http, HttpResponse } from 'msw'

beforeEach(() => {
  mockMatchMedia()
  mockNavigation()
})

// D3.1 用户列表：搜索 + 状态 badge + 详情入口 + ⋮ 快捷操作。
describe('AdminUsersPage', () => {
  it('renders the page heading', async () => {
    renderWithProviders(<AdminUsersPage />)
    expect(
      await screen.findByRole('heading', { name: /users/i }),
    ).toBeInTheDocument()
  })

  it('renders usernames with active status', async () => {
    renderWithProviders(<AdminUsersPage />)
    expect((await screen.findAllByText('admin')).length).toBeGreaterThan(0)
    expect((await screen.findAllByText('user1')).length).toBeGreaterThan(0)
    expect(await screen.findAllByText('Active')).not.toHaveLength(0)
  })

  it('links to detail page via ?id=', async () => {
    renderWithProviders(<AdminUsersPage />)
    const links = await screen.findAllByRole('link', { name: 'admin' })
    for (const link of links) {
      expect(link).toHaveAttribute('href', '/admin/users?id=1')
    }
  })

  it('shows empty state when no users', async () => {
    server.use(
      http.get('/api/v1/admin/users', () =>
        HttpResponse.json({
          items: [],
          total: 0,
          page: 1,
          limit: 20,
          total_pages: 0,
        }),
      ),
    )
    renderWithProviders(<AdminUsersPage />)
    expect(await screen.findByText('No data')).toBeInTheDocument()
  })

  it('opens the create-user dialog', async () => {
    renderWithProviders(<AdminUsersPage />)
    const add = await screen.findByRole('button', { name: 'Add user' })
    add.click()
    expect(await screen.findByRole('dialog')).toBeInTheDocument()
  })
})

// D3.3 治理操作：删除需输入用户名确认（防误删）。
describe('UserGovernanceDialogs delete flow', () => {
  it('delete confirm button is disabled until username matches', async () => {
    renderWithProviders(<AdminUsersPage />)
    // 打开 ⋮ 菜单 → 删除
    const menuButtons = await screen.findAllByRole('button', {
      name: 'More actions',
    })
    menuButtons[0].click()
    const deleteItem = await screen.findByRole('menuitem', {
      name: 'Delete user',
    })
    deleteItem.click()

    const confirm = await screen.findByRole('button', {
      name: 'Delete permanently',
    })
    expect(confirm).toBeDisabled()

    const input = await screen.findByPlaceholderText('Type username to confirm')
    const { userEvent } = await import('@testing-library/user-event')
    const user = userEvent.setup()
    await user.type(input, 'admin')
    await waitFor(() => expect(confirm).toBeEnabled())
  })
})
