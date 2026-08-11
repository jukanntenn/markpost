import '@testing-library/jest-dom'
import { describe, expect, it, beforeEach } from 'vitest'
import { screen } from '@testing-library/react'
import {
  renderWithProviders,
  mockMatchMedia,
  mockNavigation,
} from '@/test/utils'
import { DeliveryChannelDialog } from './DeliveryChannelDialog'

beforeEach(() => {
  mockMatchMedia()
  mockNavigation()
})

// D5 渠道编辑 Dialog：字段规格 + 关键词实时语法校验 + 测试按钮状态机。
const base = { open: true, onOpenChange: () => {}, editingChannel: null }

describe('DeliveryChannelDialog', () => {
  it('renders all fields in create mode', () => {
    renderWithProviders(<DeliveryChannelDialog {...base} />)
    expect(screen.getByLabelText('Channel name')).toBeInTheDocument()
    expect(screen.getByLabelText('Webhook URL')).toBeInTheDocument()
    expect(screen.getByLabelText('Card link URL')).toBeInTheDocument()
    expect(screen.getByText('Keyword filter')).toBeInTheDocument()
  })

  it('has no delete/test buttons in create mode (D5.3)', () => {
    renderWithProviders(<DeliveryChannelDialog {...base} />)
    expect(screen.queryByText('Delete')).not.toBeInTheDocument()
    expect(screen.queryByText('Send test')).not.toBeInTheDocument()
  })

  it('shows a syntax error for unterminated quotes in keywords (D5.2)', async () => {
    renderWithProviders(<DeliveryChannelDialog {...base} />)
    const input = screen.getByPlaceholderText('mark, post')
    const { userEvent } = await import('@testing-library/user-event')
    const user = userEvent.setup()
    await user.type(input, '"unterminated')
    expect(await screen.findByText(/syntax error/i)).toBeInTheDocument()
  })

  it('shows required errors on submit', async () => {
    renderWithProviders(<DeliveryChannelDialog {...base} />)
    const { userEvent } = await import('@testing-library/user-event')
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Create' }))
    expect(
      await screen.findByText('Channel name is required'),
    ).toBeInTheDocument()
  })
})
