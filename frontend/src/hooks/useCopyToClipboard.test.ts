import { renderHook, act, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { useCopyToClipboard } from './useCopyToClipboard'

// Toggle the secure-context flag the hook reads. jsdom defaults it to false.
function setSecureContext(value: boolean) {
  Object.defineProperty(window, 'isSecureContext', {
    configurable: true,
    value,
  })
}

// jsdom has no `document.execCommand`; provide a spy we can assert against.
function mockExecCommand(returns: boolean) {
  const spy = vi.fn(() => returns)
  document.execCommand = spy
  return spy
}

describe('useCopyToClipboard', () => {
  let writeTextSpy: ReturnType<typeof vi.fn>

  beforeEach(() => {
    setSecureContext(true)
    Object.assign(navigator, {
      clipboard: { writeText: vi.fn().mockResolvedValue(undefined) },
    })
    writeTextSpy = vi.mocked(navigator.clipboard.writeText)
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('returns initial state with copied=false and a copy function', () => {
    const { result } = renderHook(() => useCopyToClipboard())
    expect(result.current.copied).toBe(false)
    expect(typeof result.current.copy).toBe('function')
  })

  it('sets copied to true after successful copy via Clipboard API', async () => {
    const { result } = renderHook(() => useCopyToClipboard())

    await act(async () => {
      await result.current.copy('hello')
    })

    await waitFor(() => {
      expect(result.current.copied).toBe(true)
    })
    expect(writeTextSpy).toHaveBeenCalledWith('hello')
  })

  it('resets copied to false after default delay', async () => {
    vi.useFakeTimers()
    const { result } = renderHook(() => useCopyToClipboard())

    await act(async () => {
      await result.current.copy('hello')
    })
    expect(result.current.copied).toBe(true)

    act(() => {
      vi.advanceTimersByTime(2000)
    })
    expect(result.current.copied).toBe(false)

    vi.useRealTimers()
  })

  it('respects custom resetDelay', async () => {
    vi.useFakeTimers()
    const { result } = renderHook(() => useCopyToClipboard(1000))

    await act(async () => {
      await result.current.copy('hello')
    })
    expect(result.current.copied).toBe(true)

    act(() => {
      vi.advanceTimersByTime(999)
    })
    expect(result.current.copied).toBe(true)

    act(() => {
      vi.advanceTimersByTime(1)
    })
    expect(result.current.copied).toBe(false)

    vi.useRealTimers()
  })

  it('falls back to execCommand when writeText throws', async () => {
    writeTextSpy.mockRejectedValue(new Error('denied'))
    const execSpy = mockExecCommand(true)

    const { result } = renderHook(() => useCopyToClipboard())

    let ok = false
    await act(async () => {
      ok = await result.current.copy('hello')
    })

    expect(execSpy).toHaveBeenCalledWith('copy')
    expect(ok).toBe(true)
    expect(result.current.copied).toBe(true)
  })

  describe('non-secure context (plain HTTP)', () => {
    beforeEach(() => {
      // Over plain HTTP, navigator.clipboard is unavailable and the context is
      // not secure — the pre-fix behavior would have thrown silently here.
      setSecureContext(false)
      Object.assign(navigator, { clipboard: undefined })
    })

    it('copies via execCommand fallback and reports success', async () => {
      const execSpy = mockExecCommand(true)
      const { result } = renderHook(() => useCopyToClipboard())

      let ok = false
      await act(async () => {
        ok = await result.current.copy('hello')
      })

      expect(execSpy).toHaveBeenCalledWith('copy')
      expect(ok).toBe(true)
      expect(result.current.copied).toBe(true)
    })

    it('reports failure (copied stays false) when execCommand returns false', async () => {
      const execSpy = mockExecCommand(false)
      const { result } = renderHook(() => useCopyToClipboard())

      let ok = true
      await act(async () => {
        ok = await result.current.copy('hello')
      })

      expect(execSpy).toHaveBeenCalled()
      expect(ok).toBe(false)
      expect(result.current.copied).toBe(false)
    })
  })
})
