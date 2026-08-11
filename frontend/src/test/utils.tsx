import { render } from '@testing-library/react'
import { vi } from 'vitest'
import type { LoginResponse } from '@/types/auth'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { NextIntlClientProvider } from 'next-intl'
import en from '../i18n/locales/en.json'
import { setClientMessages } from '@/i18n/messages'
import { setCurrentLocale } from '@/i18n/current'
import { setDefaultLocale } from '@/utils/time'
import { LocaleContext } from '@/components/providers/LocaleProvider'

type WrapperComponent = React.ComponentType<{ children: React.ReactNode }>

setClientMessages(en)
setCurrentLocale('en')
setDefaultLocale('en')

function MockLocaleProvider({ children }: { children: React.ReactNode }) {
  return (
    <LocaleContext.Provider
      value={{
        locale: 'en',
        setLocale: vi.fn(),
        availableLocales: ['en', 'zh-Hans'],
      }}
    >
      {children}
    </LocaleContext.Provider>
  )
}

// next/navigation mock：内存 searchParams（供 useUrlQueryState/守卫测试）。
const navState = {
  pathname: '/',
  searchParams: new URLSearchParams(),
  push: vi.fn(),
  replace: vi.fn(),
}

vi.mock('next/navigation', () => ({
  useRouter: () => ({
    push: navState.push,
    replace: navState.replace,
    back: vi.fn(),
    prefetch: vi.fn(),
  }),
  usePathname: () => navState.pathname,
  useSearchParams: () => navState.searchParams,
}))

export function mockNavigation() {
  navState.searchParams = new URLSearchParams()
  navState.push.mockReset()
  navState.replace.mockReset()
  return navState
}

export function renderWithProviders(
  ui: React.ReactElement,
  options?: { wrapper?: WrapperComponent },
) {
  const Wrapper = options?.wrapper ?? (({ children }) => <>{children}</>)
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return render(
    <QueryClientProvider client={client}>
      <NextIntlClientProvider locale="en" messages={en}>
        <MockLocaleProvider>
          <Wrapper>{ui}</Wrapper>
        </MockLocaleProvider>
      </NextIntlClientProvider>
    </QueryClientProvider>,
  )
}

export function mockMatchMedia() {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })
}

export function createMockUser(overrides = {}) {
  return {
    id: 1,
    username: 'testuser',
    ...overrides,
  }
}

export function createMockAuth(overrides = {}) {
  return {
    token: 'test_token',
    refresh_token: 'test_refresh',
    user: createMockUser(),
    ...overrides,
  }
}

export function setMockAuth(auth: LoginResponse) {
  localStorage.setItem('markpost_dev_login', JSON.stringify(auth))
}

export function clearMockAuth() {
  localStorage.removeItem('markpost_dev_login')
}

export function createQueryWrapper() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  function QueryWrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
  QueryWrapper.displayName = 'QueryWrapper'
  return QueryWrapper
}
