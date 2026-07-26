import '@testing-library/jest-dom'
import { beforeEach, describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'

import { DeliveryHistoryTable } from './DeliveryHistoryTable'
import { renderWithProviders, mockMatchMedia } from '@/test/utils'
import type { DeliveryHistoryItem } from '@/types/delivery'

beforeEach(() => {
  mockMatchMedia()
})

describe('DeliveryHistoryTable', () => {
  it('shows empty state when there are no items', () => {
    renderWithProviders(<DeliveryHistoryTable items={[]} />)

    expect(screen.getByText(/no delivery history yet/i)).toBeInTheDocument()
  })

  it('renders a delivered row with post link and status badge', () => {
    const items: DeliveryHistoryItem[] = [
      {
        id: 1,
        status: 'delivered',
        last_error: '',
        created_at: '2024-01-01T12:00:00Z',
        channel_id: 1,
        post_title: 'Hello World',
        post_qid: 'p-abc',
        channel_name: 'My Channel',
        username: 'user',
      },
    ]

    renderWithProviders(<DeliveryHistoryTable items={items} />)

    const link = screen.getByRole('link', { name: /hello world/i })
    expect(link).toHaveAttribute('href', '/p-abc')
    expect(screen.getByText(/delivered/i)).toBeInTheDocument()
    expect(screen.getByText('My Channel')).toBeInTheDocument()
  })

  it('shows deleted-post placeholder when post_qid is null', () => {
    const items: DeliveryHistoryItem[] = [
      {
        id: 2,
        status: 'failed',
        last_error: 'boom',
        created_at: '2024-01-01T12:00:00Z',
        channel_id: 1,
        post_title: null,
        post_qid: null,
        channel_name: 'My Channel',
        username: 'user',
      },
    ]

    renderWithProviders(<DeliveryHistoryTable items={items} />)

    expect(screen.getByText(/post deleted/i)).toBeInTheDocument()
    expect(screen.getByText('boom')).toBeInTheDocument()
    expect(screen.getByText(/failed/i)).toBeInTheDocument()
  })
})
