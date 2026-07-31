import '@testing-library/jest-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, render, screen, waitFor } from '@testing-library/react'
import { LocaleProvider, useLocaleContext } from './LocaleProvider'

// Mock the dynamic message loader so we can control resolution timing and
// content. Each locale maps to a tiny message tree; we keep the en/zh shapes
// distinct so a downstream consumer would see different text.
vi.mock('@/utils/i18n', async () => {
  const actual =
    await vi.importActual<typeof import('@/utils/i18n')>('@/utils/i18n')
  const messages: Record<string, Record<string, unknown>> = {
    en: { Greeting: { hello: 'Hello' } },
    'zh-Hans': { Greeting: { hello: '你好' } },
  }
  return {
    ...actual,
    loadMessages: vi.fn((locale: string) =>
      Promise.resolve(messages[locale] ?? messages.en)
    ),
  }
})

function Child() {
  return <span data-testid="child">child-content</span>
}

// A probe that exposes the provider's context value for assertions / actions.
function ContextProbe({
  onCtx,
}: {
  onCtx: (ctx: ReturnType<typeof useLocaleContext>) => void
}) {
  onCtx(useLocaleContext())
  return <Child />
}

describe('LocaleProvider', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('gates children behind the skeleton until messages load', async () => {
    render(
      <LocaleProvider>
        <Child />
      </LocaleProvider>
    )

    // First frame: skeleton present, children not mounted.
    expect(screen.queryByTestId('child')).not.toBeInTheDocument()
    expect(document.querySelector('header.sticky')).toBeInTheDocument()

    // After the async chunk resolves, children mount and skeleton is gone.
    await waitFor(() => {
      expect(screen.getByTestId('child')).toBeInTheDocument()
    })
    expect(document.querySelector('header.sticky')).not.toBeInTheDocument()
  })

  it('emits no dotted translation keys during the gated frame', async () => {
    const { container } = render(
      <LocaleProvider>
        <Child />
      </LocaleProvider>
    )
    expect(container.textContent).not.toMatch(/[a-z]+\.[a-z]+\./)
    await waitFor(() => {
      expect(screen.getByTestId('child')).toBeInTheDocument()
    })
  })

  it('reloads messages and persists locale when setLocale is called', async () => {
    let ctx: ReturnType<typeof useLocaleContext> | null = null
    render(
      <LocaleProvider>
        <ContextProbe onCtx={(c) => (ctx = c)} />
      </LocaleProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId('child')).toBeInTheDocument()
    })

    const { loadMessages } = await import('@/utils/i18n')
    await act(async () => {
      await ctx!.setLocale('zh-Hans')
    })
    expect(loadMessages).toHaveBeenCalledWith('zh-Hans')
    expect(localStorage.getItem('locale')).toBe('zh-Hans')
  })
})
