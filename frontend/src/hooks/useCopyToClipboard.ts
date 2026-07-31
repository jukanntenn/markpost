import { useState, useCallback, useRef, useEffect } from 'react'

// copyTextViaExecCommand is the legacy fallback for non-secure contexts (plain
// HTTP) where `navigator.clipboard` is undefined. It synthesizes a hidden
// textarea, selects it, and runs `document.execCommand('copy')`. Returns false
// if execCommand is unavailable or fails. This API is deprecated but remains
// the only way to write to the clipboard over HTTP.
function copyTextViaExecCommand(text: string): boolean {
  if (typeof document === 'undefined' || !document.execCommand) return false

  const textarea = document.createElement('textarea')
  textarea.value = text
  // `position: fixed` keeps it in the viewport (off-screen) even on mobile,
  // where `absolute` + negative offsets can be ignored by some keyboards.
  textarea.setAttribute('readonly', '')
  textarea.style.position = 'fixed'
  textarea.style.top = '0'
  textarea.style.left = '0'
  textarea.style.opacity = '0'
  document.body.appendChild(textarea)

  // iOS Safari ignores `select()` on textareas; it needs a Range selection.
  if (navigator.userAgent.match(/ipad|iphone/i)) {
    const range = document.createRange()
    range.selectNodeContents(textarea)
    const selection = window.getSelection()
    selection?.removeAllRanges()
    selection?.addRange(range)
    textarea.setSelectionRange(0, text.length)
  } else {
    textarea.select()
  }

  let ok = false
  try {
    ok = document.execCommand('copy')
  } catch {
    ok = false
  } finally {
    document.body.removeChild(textarea)
  }
  return ok
}

export function useCopyToClipboard(resetDelay = 2000) {
  const [copied, setCopied] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)

  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  }, [])

  const copy = useCallback(
    async (text: string): Promise<boolean> => {
      let ok = false
      try {
        if (navigator.clipboard && window.isSecureContext) {
          await navigator.clipboard.writeText(text)
          ok = true
        } else {
          ok = copyTextViaExecCommand(text)
        }
      } catch {
        // Fall through to execCommand if the async API throws (e.g. permission
        // denied) and we haven't succeeded yet.
        if (!ok) ok = copyTextViaExecCommand(text)
      }

      if (ok) {
        if (timerRef.current) clearTimeout(timerRef.current)
        setCopied(true)
        timerRef.current = setTimeout(() => setCopied(false), resetDelay)
      } else {
        setCopied(false)
      }
      return ok
    },
    [resetDelay]
  )

  return { copied, copy } as const
}
