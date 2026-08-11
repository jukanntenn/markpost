import '@testing-library/jest-dom'
import { describe, expect, it, beforeEach } from 'vitest'
import { render, waitFor } from '@testing-library/react'
import { mockNavigation } from '@/test/utils'
import { useAuthStore } from '@/stores/auth'
import { useAuthGuard } from './useAuthGuard'

function GuardProbe({
  redirectPath,
  withNext,
}: {
  redirectPath: string
  withNext?: boolean
}) {
  useAuthGuard({
    shouldRedirect: (isAuth, isAdm) => !isAuth && !isAdm,
    redirectPath,
    withNext,
  })
  return null
}

beforeEach(() => {
  useAuthStore.setState({
    token: null,
    refreshToken: null,
    user: null,
    sessionExpired: false,
  })
  mockNavigation()
})

// B1.8 场景 D：refresh 失败 → markSessionExpired → 守卫跳 /login?reason=session_expired。
describe('useAuthGuard session-expired reason', () => {
  it('redirects to /login with reason=session_expired when flagged', async () => {
    const nav = mockNavigation()
    useAuthStore.setState({ _hasHydrated: true, sessionExpired: true })

    render(<GuardProbe redirectPath="/login" />)

    await waitFor(() => {
      expect(nav.replace).toHaveBeenCalled()
    })
    const target = nav.replace.mock.calls[0][0] as string
    expect(target).toContain('reason=session_expired')
    // 消费后清标志，避免下次跳转重复携带。
    expect(useAuthStore.getState().sessionExpired).toBe(false)
  })

  it('does not add reason for a plain (voluntary) logout redirect', async () => {
    const nav = mockNavigation()
    useAuthStore.setState({ _hasHydrated: true, sessionExpired: false })

    render(<GuardProbe redirectPath="/login" />)

    await waitFor(() => {
      expect(nav.replace).toHaveBeenCalled()
    })
    const target = nav.replace.mock.calls[0][0] as string
    expect(target).not.toContain('reason=session_expired')
  })

  it('keeps ?next= alongside the reason when withNext is set', async () => {
    const nav = mockNavigation()
    useAuthStore.setState({ _hasHydrated: true, sessionExpired: true })
    nav.searchParams.set('pathname-probe', 'ignored')

    render(<GuardProbe redirectPath="/login" withNext />)

    await waitFor(() => {
      expect(nav.replace).toHaveBeenCalled()
    })
    const target = nav.replace.mock.calls[0][0] as string
    expect(target).toContain('reason=session_expired')
    expect(target).toContain('next=')
  })

  // BUG-4 回归：clearSessionExpired() 翻转 sessionExpired（依赖项），且
  // AuthGate 传入的内联 shouldRedirect 每次渲染都是新引用，effect 会在首次
  // 跳转后重跑。修复前第二次 router.replace 用无 reason 的 URL 覆盖了第一次；
  // 重定向锁必须保证 replace 只触发一次且保留 reason。
  it('fires redirect exactly once with reason, even if the effect re-runs', async () => {
    const nav = mockNavigation()
    useAuthStore.setState({ _hasHydrated: true, sessionExpired: true })

    const { rerender } = render(<GuardProbe redirectPath="/login" withNext />)

    await waitFor(() => {
      expect(nav.replace).toHaveBeenCalledTimes(1)
    })

    // Simulate the dep change that previously caused the clobbering re-run:
    // the store flips sessionExpired back to false (clearSessionExpired ran),
    // and React re-renders the guarded component.
    useAuthStore.setState({ sessionExpired: false })
    rerender(<GuardProbe redirectPath="/login" withNext />)
    rerender(<GuardProbe redirectPath="/login" withNext />)

    expect(nav.replace).toHaveBeenCalledTimes(1)
    expect(nav.replace.mock.calls[0][0]).toContain('reason=session_expired')
  })
})
