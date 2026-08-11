import '@testing-library/jest-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { renderWithProviders, mockMatchMedia } from '@/test/utils'
import { ListState } from './list-state'
import { EmptyState } from './empty-state'

beforeEach(() => mockMatchMedia())

// B3.1/H.2 统一三态范式：loading→骨架、error→图标+说明+重试、empty→EmptyState。
describe('ListState', () => {
  it('renders the loading skeleton while loading', () => {
    renderWithProviders(
      <ListState
        isLoading
        error={null}
        loadingSkeleton={<div data-testid="skeleton">loading</div>}
        onRetry={() => {}}
      >
        content
      </ListState>,
    )
    expect(screen.getByTestId('skeleton')).toBeInTheDocument()
    expect(screen.queryByText('content')).not.toBeInTheDocument()
  })

  it('renders error state with retry for server errors', () => {
    const retry = vi.fn()
    renderWithProviders(
      <ListState
        isLoading={false}
        error={new Error('boom')}
        loadingSkeleton={<div />}
        onRetry={retry}
      >
        content
      </ListState>,
    )
    expect(screen.getByText('Failed to load')).toBeInTheDocument()
    screen.getByRole('button', { name: 'Retry' }).click()
    expect(retry).toHaveBeenCalled()
    expect(screen.queryByText('content')).not.toBeInTheDocument()
  })

  it('renders children when loaded without error', () => {
    renderWithProviders(
      <ListState
        isLoading={false}
        error={null}
        loadingSkeleton={<div />}
        onRetry={() => {}}
      >
        content
      </ListState>,
    )
    expect(screen.getByText('content')).toBeInTheDocument()
  })

  it('renders the empty state when emptyWhen is true', () => {
    renderWithProviders(
      <ListState
        isLoading={false}
        error={null}
        loadingSkeleton={<div />}
        emptyWhen
        empty={<EmptyState title="No data here" />}
        onRetry={() => {}}
      >
        content
      </ListState>,
    )
    expect(screen.getByText('No data here')).toBeInTheDocument()
    expect(screen.queryByText('content')).not.toBeInTheDocument()
  })
})

// EmptyState 渲染契约：图标 + 标题 + 说明 + CTA。
describe('EmptyState', () => {
  it('renders title, description and action', () => {
    render(
      <EmptyState
        title="Empty title"
        description="Empty description"
        action={<button type="button">CTA</button>}
      />,
    )
    expect(screen.getByText('Empty title')).toBeInTheDocument()
    expect(screen.getByText('Empty description')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'CTA' })).toBeInTheDocument()
  })
})
