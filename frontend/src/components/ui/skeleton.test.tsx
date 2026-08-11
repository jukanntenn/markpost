import '@testing-library/jest-dom'
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Skeleton } from './skeleton'

describe('Skeleton', () => {
  it('renders a div with shimmer animation and muted background classes', () => {
    const { container } = render(<Skeleton />)
    const el = container.firstChild as HTMLElement
    expect(el).toBeInTheDocument()
    expect(el).toHaveClass('animate-shimmer')
    expect(el).toHaveClass('bg-muted')
    expect(el).toHaveClass('skeleton-shimmer')
  })

  it('applies custom className alongside defaults', () => {
    render(<Skeleton className="h-4 w-24" data-testid="sk" />)
    const el = screen.getByTestId('sk')
    expect(el).toHaveClass('h-4', 'w-24')
    expect(el).toHaveClass('bg-muted')
  })
})
