import '@testing-library/jest-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { renderWithProviders, mockMatchMedia } from '@/test/utils'
import { ThemeProvider } from '@/components/theme-provider'
import { AdminUserDialog } from './AdminUserDialog'

vi.mock('@/stores/toast', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
  },
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockMatchMedia()
})

describe('AdminUserDialog', () => {
  it('submits successfully without email', async () => {
    const user = userEvent.setup()
    const onOpenChange = vi.fn()
    renderWithProviders(
      <AdminUserDialog open={true} onOpenChange={onOpenChange} />,
      { wrapper: ThemeProvider }
    )

    // 不填 email，只填 username + password
    await user.type(screen.getByPlaceholderText(/enter username/i), 'newuser')
    await user.type(
      screen.getByPlaceholderText(/min 6 characters/i),
      'password123'
    )

    await user.click(screen.getByRole('button', { name: /^create$/i }))

    // 提交成功后应显示 toast 并关闭弹框
    const { toast } = await import('@/stores/toast')
    await waitFor(() => {
      expect(toast.success).toHaveBeenCalledWith('User created successfully')
    })
  })

  it('submits successfully with valid email', async () => {
    const user = userEvent.setup()
    const onOpenChange = vi.fn()
    renderWithProviders(
      <AdminUserDialog open={true} onOpenChange={onOpenChange} />,
      { wrapper: ThemeProvider }
    )

    await user.type(
      screen.getByPlaceholderText(/user@example.com/i),
      'good@example.com'
    )
    await user.type(screen.getByPlaceholderText(/enter username/i), 'emailuser')
    await user.type(
      screen.getByPlaceholderText(/min 6 characters/i),
      'password123'
    )

    await user.click(screen.getByRole('button', { name: /^create$/i }))

    const { toast } = await import('@/stores/toast')
    await waitFor(() => {
      expect(toast.success).toHaveBeenCalledWith('User created successfully')
    })
  })

  it('shows error for invalid email format but allows empty email', async () => {
    const user = userEvent.setup()
    renderWithProviders(
      <AdminUserDialog open={true} onOpenChange={vi.fn()} />,
      { wrapper: ThemeProvider }
    )

    // 填了格式错误的 email（form noValidate 确保 React 校验逻辑执行）
    await user.type(
      screen.getByPlaceholderText(/user@example\.com/i),
      'not-an-email'
    )
    await user.type(screen.getByPlaceholderText(/enter username/i), 'xuser')
    await user.type(
      screen.getByPlaceholderText(/min 6 characters/i),
      'password123'
    )

    await user.click(screen.getByRole('button', { name: /^create$/i }))

    // React 校验应显示格式错误，且 mutation 不触发（无 success toast）
    await waitFor(() => {
      expect(screen.getByText(/invalid email format/i)).toBeInTheDocument()
    })
    const { toast } = await import('@/stores/toast')
    expect(toast.success).not.toHaveBeenCalled()
  })
})
