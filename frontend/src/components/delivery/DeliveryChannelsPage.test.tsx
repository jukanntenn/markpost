import '@testing-library/jest-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import { DeliveryChannelsPage } from './DeliveryChannelsPage'
import { ThemeProvider } from '@/components/theme-provider'
import { renderWithProviders, mockMatchMedia } from '@/test/utils'
import {
  mockDeliveryChannels,
  mockDeliveryLatest,
  resetDeliveryMocks,
} from '@/mocks/handlers'

vi.mock('@/stores/toast', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
  },
}))

beforeEach(() => {
  vi.clearAllMocks()
  resetDeliveryMocks()
  mockMatchMedia()
})

describe('DeliveryChannelsPage', () => {
  it('shows empty state when no channels exist', async () => {
    renderWithProviders(<DeliveryChannelsPage />, { wrapper: ThemeProvider })

    await waitFor(() => {
      expect(screen.getByText(/no delivery channels yet/i)).toBeInTheDocument()
    })
  })

  it('renders channel rows with name and kind', async () => {
    mockDeliveryChannels.push({
      id: 1,
      kind: 'feishu',
      name: 'My Feishu Channel',
      enabled: true,
      configuration: {
        webhook_url: 'https://example.com/hook',
        card_link_url: '',
      },
      keywords: 'alert',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    })

    renderWithProviders(<DeliveryChannelsPage />, { wrapper: ThemeProvider })

    await waitFor(() => {
      expect(screen.getByText('My Feishu Channel')).toBeInTheDocument()
      expect(screen.getByText('feishu')).toBeInTheDocument()
    })
  })

  it("shows 'Never' for latest delivery when channel has no history", async () => {
    mockDeliveryChannels.push({
      id: 1,
      kind: 'feishu',
      name: 'No History Channel',
      enabled: true,
      configuration: {
        webhook_url: 'https://example.com/hook',
        card_link_url: '',
      },
      keywords: '',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    })

    renderWithProviders(<DeliveryChannelsPage />, { wrapper: ThemeProvider })

    await waitFor(() => {
      expect(screen.getByText(/never/i)).toBeInTheDocument()
    })
  })

  it('shows latest delivery status badge when history exists', async () => {
    mockDeliveryChannels.push({
      id: 1,
      kind: 'feishu',
      name: 'With History',
      enabled: true,
      configuration: {
        webhook_url: 'https://example.com/hook',
        card_link_url: '',
      },
      keywords: '',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    })
    mockDeliveryLatest.push({
      id: 10,
      status: 'delivered',
      last_error: '',
      created_at: '2024-01-02T00:00:00Z',
      channel_id: 1,
      post_title: 'Some Post',
      post_qid: 'p-1',
      channel_name: 'With History',
      username: 'user',
    })

    renderWithProviders(<DeliveryChannelsPage />, { wrapper: ThemeProvider })

    await waitFor(() => {
      expect(screen.getByText(/delivered/i)).toBeInTheDocument()
    })
  })

  it('creates a new channel via the dialog', async () => {
    const user = userEvent.setup()
    renderWithProviders(<DeliveryChannelsPage />, { wrapper: ThemeProvider })

    await waitFor(() => {
      expect(screen.getByText(/no delivery channels yet/i)).toBeInTheDocument()
    })

    const addButtons = screen.getAllByRole('button', { name: /add channel/i })
    await user.click(addButtons[0])

    const nameInput = screen.getByLabelText(/name/i)
    const webhookInput = screen.getByLabelText(/webhook url/i)
    await user.type(nameInput, 'New Channel')
    await user.type(webhookInput, 'https://example.com/new-hook')

    await user.click(screen.getByRole('button', { name: /^create$/i }))

    await waitFor(() => {
      expect(screen.getByText('New Channel')).toBeInTheDocument()
    })
  })
})
