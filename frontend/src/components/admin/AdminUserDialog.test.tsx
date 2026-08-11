import '@testing-library/jest-dom'
import { describe, expect, it, beforeEach } from 'vitest'
import { screen } from '@testing-library/react'
import {
  renderWithProviders,
  mockMatchMedia,
  mockNavigation,
} from '@/test/utils'
import AdminUserDialog from './AdminUserDialog'

beforeEach(() => {
  mockMatchMedia()
  mockNavigation()
})

// F.10 新建用户 Dialog：用户名必填、邮箱可选但填了须 email、密码 min8/max72。
describe('AdminUserDialog', () => {
  it('renders all three fields', () => {
    renderWithProviders(<AdminUserDialog open onOpenChange={() => {}} />)
    expect(screen.getByLabelText('Username')).toBeInTheDocument()
    expect(screen.getByLabelText('Email (optional)')).toBeInTheDocument()
    expect(screen.getByLabelText('Password')).toBeInTheDocument()
  })

  it('shows a password strength hint for short passwords (info, not error)', () => {
    renderWithProviders(<AdminUserDialog open onOpenChange={() => {}} />)
    const password = screen.getByLabelText('Password')
    // 直接设置值并触发 input
    const setter = Object.getOwnPropertyDescriptor(
      window.HTMLInputElement.prototype,
      'value',
    )!.set!
    setter.call(password, 'abc')
    password.dispatchEvent(new Event('input', { bubbles: true }))
    expect(screen.getByText('Weak')).toBeInTheDocument()
  })
})
