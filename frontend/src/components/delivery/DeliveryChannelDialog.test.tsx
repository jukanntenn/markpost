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
  // base-ui 弹层在 jsdom 中需要 ResizeObserver（浏览器原生）。
  if (typeof globalThis.ResizeObserver === 'undefined') {
    globalThis.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    } as unknown as typeof ResizeObserver
  }
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
    const input = screen.getByPlaceholderText('error, warning & !debug')
    const { userEvent } = await import('@testing-library/user-event')
    const user = userEvent.setup()
    await user.type(input, '"unterminated')
    expect(
      await screen.findByText(
        'Syntax error: the quote at character 1 has no closing "',
      ),
    ).toBeInTheDocument()
  })

  it('previews the filter as a natural-language sentence while typing', async () => {
    renderWithProviders(<DeliveryChannelDialog {...base} />)
    const input = screen.getByPlaceholderText('error, warning & !debug')
    expect(
      screen.getByText('Delivers every post (empty — no filtering)'),
    ).toBeInTheDocument()

    const { userEvent } = await import('@testing-library/user-event')
    const user = userEvent.setup()
    await user.type(input, 'prod & (error, warning) & !debug')
    expect(
      await screen.findByText(
        'Delivers when the title contains “prod” and (contains “error” or contains “warning”) and does not contain “debug”',
      ),
    ).toBeInTheDocument()
  })

  it('opens the syntax cheat sheet and fills an example on click', async () => {
    renderWithProviders(<DeliveryChannelDialog {...base} />)
    const { userEvent } = await import('@testing-library/user-event')
    const user = userEvent.setup()

    await user.click(
      screen.getByRole('button', { name: 'Keyword syntax help' }),
    )
    expect(await screen.findByText('Syntax cheat sheet')).toBeInTheDocument()
    expect(
      screen.getByText(
        'Spaces are part of the keyword: key word 1 is one keyword — no quotes needed.',
      ),
    ).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'alert' }))
    const input = screen.getByPlaceholderText('error, warning & !debug')
    expect(input).toHaveValue('alert')
    expect(
      await screen.findByText('Delivers when the title contains “alert”'),
    ).toBeInTheDocument()
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
