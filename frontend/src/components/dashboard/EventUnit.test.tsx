import '@testing-library/jest-dom'
import { describe, expect, it, beforeEach } from 'vitest'
import { screen } from '@testing-library/react'
import {
  renderWithProviders,
  mockMatchMedia,
  mockNavigation,
} from '@/test/utils'
import { EventUnit, type ActivityEvent } from './EventUnit'
import type { DeliveryHistoryItem } from '@/types/delivery'

beforeEach(() => {
  mockMatchMedia()
  mockNavigation()
})

function delivery(
  id: number,
  status: DeliveryHistoryItem['status'],
  channelName = `ch${id}`,
): DeliveryHistoryItem {
  return {
    id,
    channel_id: id,
    channel_name: channelName,
    post_qid: 'p-qid',
    post_title: 'My post',
    status,
    last_error: '',
    created_at: '2026-08-10T09:00:00Z',
    username: null,
  }
}

const event = (
  deliveries: DeliveryHistoryItem[],
  pendingCount = 0,
): ActivityEvent => ({
  postQid: 'p-qid',
  postTitle: 'My post',
  createdAt: '2026-08-10T09:00:00Z',
  deliveries,
  pendingCount,
})

// B2.5 交互：全成功默认折叠（只显示摘要），点击展开；部分失败默认展开失败项。
describe('EventUnit', () => {
  it('collapses fully-successful deliveries by default', () => {
    renderWithProviders(<EventUnit event={event([delivery(1, 'delivered')])} />)
    expect(screen.getByText('1/1 delivered ✓')).toBeInTheDocument()
    expect(screen.queryByText('ch1')).not.toBeInTheDocument()
  })

  it('expands on summary click', async () => {
    const { userEvent } = await import('@testing-library/user-event')
    renderWithProviders(<EventUnit event={event([delivery(1, 'delivered')])} />)
    await userEvent.click(
      screen.getByRole('button', { name: 'Show delivery details' }),
    )
    expect(screen.getByText('ch1')).toBeInTheDocument()
  })

  it('expands failed deliveries by default with error reason', () => {
    renderWithProviders(
      <EventUnit
        event={event([delivery(1, 'delivered'), delivery(2, 'failed', 'mail')])}
      />,
    )
    expect(screen.getByText('ch1')).toBeInTheDocument()
    expect(screen.getByText('mail')).toBeInTheDocument()
    expect(
      screen.getByRole('link', { name: 'View detail' }),
    ).toBeInTheDocument()
  })

  it('renders pending deliveries with the delivering summary', () => {
    renderWithProviders(<EventUnit event={event([], 2)} />)
    expect(screen.getByText('Delivering...')).toBeInTheDocument()
  })
})
