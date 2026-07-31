import '@testing-library/jest-dom'
import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'
import { AppShellSkeleton } from './AppShellSkeleton'

describe('AppShellSkeleton', () => {
  it('renders a sticky header (topbar) region', () => {
    const { container } = render(<AppShellSkeleton />)
    const header = container.querySelector('header')
    expect(header).toBeInTheDocument()
    expect(header).toHaveClass('sticky', 'top-0')
  })

  it('renders a content region marked busy while loading', () => {
    const { container } = render(<AppShellSkeleton />)
    const main = container.querySelector('main')
    expect(main).toBeInTheDocument()
    expect(main).toHaveAttribute('aria-busy', 'true')
    expect(main).toHaveAttribute('aria-live', 'polite')
  })

  it('renders skeleton placeholder rows for the content list', () => {
    const { container } = render(<AppShellSkeleton />)
    const rows = container.querySelectorAll('main ul > li')
    expect(rows.length).toBe(4)
  })

  it('emits no localized keys (no dotted translation paths in text)', () => {
    const { container } = render(<AppShellSkeleton />)
    // No text content should leak translation keys during the gated frame.
    expect(container.textContent).not.toMatch(/[a-z]+\.[a-z]+\./)
  })
})
