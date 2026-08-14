import '@testing-library/jest-dom'
import { describe, expect, it, beforeEach, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import {
  renderWithProviders,
  mockMatchMedia,
  mockNavigation,
} from '@/test/utils'
import PostKeyPage from './PostKeyPage'

vi.mock('@/hooks/usePostKey', () => ({
  usePostKey: () => ({
    data: {
      post_key: 'mpk-abcdef123456',
      created_at: '2026-01-01T00:00:00Z',
    },
    isLoading: false,
    error: null,
  }),
}))

const { toastAdd, clipboard } = vi.hoisted(() => ({
  toastAdd: vi.fn(),
  clipboard: {
    ok: true,
    copied: false,
    calls: [] as string[],
  },
}))

vi.mock('@/stores/toast', () => ({
  toastManager: { add: toastAdd },
}))

// Mock the copy hook so the test asserts WHAT the page asks to copy (the real
// clipboard API is covered by useCopyToClipboard.test). `copied` reflects the
// last result so the green-check swap is exercisable.
vi.mock('@/hooks/useCopyToClipboard', () => ({
  useCopyToClipboard: () => ({
    get copied() {
      return clipboard.copied
    },
    copy: async (text: string) => {
      clipboard.calls.push(text)
      clipboard.copied = clipboard.ok
      return clipboard.ok
    },
  }),
}))

describe('PostKeyPage copy UX', () => {
  beforeEach(() => {
    mockMatchMedia()
    mockNavigation()
    toastAdd.mockClear()
    clipboard.ok = true
    clipboard.copied = false
    clipboard.calls = []
  })

  it('shows the masked key in the public URL form, reveals on toggle', async () => {
    const { userEvent } = await import('@testing-library/user-event')
    const user = userEvent.setup()
    renderWithProviders(<PostKeyPage />)

    expect(screen.getByText(/\/•{4,}/)).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Show' }))
    expect(screen.getByText(/\/mpk-abcdef123456$/)).toBeInTheDocument()
  })

  it('copies the full URL by default with no success toast', async () => {
    const { userEvent } = await import('@testing-library/user-event')
    const user = userEvent.setup()
    renderWithProviders(<PostKeyPage />)

    await user.click(screen.getByRole('button', { name: 'Copy post URL' }))

    await waitFor(() => {
      expect(clipboard.calls).toEqual(
        expect.arrayContaining([expect.stringMatching(/\/mpk-abcdef123456$/)]),
      )
    })
    expect(toastAdd).not.toHaveBeenCalled()
  })

  it('can copy just the post key via the options menu', async () => {
    const { userEvent } = await import('@testing-library/user-event')
    const user = userEvent.setup()
    renderWithProviders(<PostKeyPage />)

    await user.click(screen.getByRole('button', { name: 'Copy options' }))
    const keyItem = await screen.findByRole('menuitem', { name: /Copy key/ })
    await user.click(keyItem)

    await waitFor(() => {
      expect(clipboard.calls).toContain('mpk-abcdef123456')
    })
    expect(toastAdd).not.toHaveBeenCalled()
  })

  it('toasts an error when the copy fails', async () => {
    clipboard.ok = false
    const { userEvent } = await import('@testing-library/user-event')
    const user = userEvent.setup()
    renderWithProviders(<PostKeyPage />)

    await user.click(screen.getByRole('button', { name: 'Copy post URL' }))

    await waitFor(() => {
      expect(toastAdd).toHaveBeenCalledWith(
        expect.objectContaining({ type: 'error' }),
      )
    })
  })
})
