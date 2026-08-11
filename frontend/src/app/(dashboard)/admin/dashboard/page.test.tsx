import '@testing-library/jest-dom'
import { describe, expect, it, beforeEach } from 'vitest'
import { screen } from '@testing-library/react'
import {
  renderWithProviders,
  mockMatchMedia,
  mockNavigation,
} from '@/test/utils'
import AdminDashboardPage from '@/components/admin/AdminDashboardPage'

beforeEach(() => {
  mockMatchMedia()
  mockNavigation()
})

// D2 态势感知仪表盘：需要关注（正常态确认感）+ 存量指标 + 趋势图。
describe('AdminDashboardPage', () => {
  it('renders the dashboard heading', async () => {
    renderWithProviders(<AdminDashboardPage />)
    expect(
      await screen.findByRole('heading', { name: /overview/i }),
    ).toBeInTheDocument()
  })

  it('shows loading skeleton initially', () => {
    renderWithProviders(<AdminDashboardPage />)
    expect(document.querySelector('.animate-shimmer')).not.toBeNull()
  })

  it('renders stats after data loads', async () => {
    renderWithProviders(<AdminDashboardPage />)
    expect(await screen.findByText('Users')).toBeInTheDocument()
    expect(await screen.findByText('Posts')).toBeInTheDocument()
  })

  it('shows "all good" attention state when no issues', async () => {
    renderWithProviders(<AdminDashboardPage />)
    expect(await screen.findByText('All good')).toBeInTheDocument()
  })

  it('shows recent admin actions feed', async () => {
    renderWithProviders(<AdminDashboardPage />)
    expect(await screen.findByText('Recent admin actions')).toBeInTheDocument()
  })
})
