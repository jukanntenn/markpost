import '@testing-library/jest-dom'
import { describe, expect, it, beforeEach } from 'vitest'
import { screen } from '@testing-library/react'
import {
  renderWithProviders,
  mockMatchMedia,
  mockNavigation,
} from '@/test/utils'
import CreateTestPostModal from './CreateTestPostModal'

beforeEach(() => {
  mockMatchMedia()
  mockNavigation()
})

// F.11 测试发帖 Dialog：title/body 必填、title ≤150、成功 toast。
describe('CreateTestPostModal', () => {
  const base = { postKey: 'mpk-test', onHide: () => {}, onSuccess: () => {} }

  it('renders title and body fields', () => {
    renderWithProviders(<CreateTestPostModal show {...base} />)
    expect(screen.getByLabelText('Title')).toBeInTheDocument()
    expect(screen.getByLabelText('Body')).toBeInTheDocument()
  })

  it('shows field errors when submitting empty', async () => {
    renderWithProviders(<CreateTestPostModal show {...base} />)
    const { userEvent } = await import('@testing-library/user-event')
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Create' }))
    expect(
      (await screen.findAllByText('Title is required')).length,
    ).toBeGreaterThan(0)
    expect(await screen.findByText('Body is required')).toBeInTheDocument()
  })

  it('rejects titles longer than 150 chars', async () => {
    renderWithProviders(<CreateTestPostModal show {...base} />)
    const { userEvent } = await import('@testing-library/user-event')
    const user = userEvent.setup()
    await user.type(screen.getByLabelText('Title'), 'x'.repeat(151))
    await user.type(screen.getByLabelText('Body'), 'hello')
    await user.click(screen.getByRole('button', { name: 'Create' }))
    expect(
      await screen.findByText('Title must be at most 150 characters'),
    ).toBeInTheDocument()
  })
})
