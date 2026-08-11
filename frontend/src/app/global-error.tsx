'use client'

// A2.8 层级0：根 layout（Provider 链）崩溃的最终兜底。渲染在 root layout
// 之外，自包含：不依赖任何 Provider 与语义 token，用 Tailwind 原生类。
export default function GlobalError({
  reset,
}: {
  error: Error & { digest?: string }
  reset: () => void
}) {
  return (
    <html lang="en">
      <body
        style={{
          fontFamily: 'system-ui, sans-serif',
        }}
        className="m-0 flex min-h-svh items-center justify-center bg-stone-50 p-6 text-stone-900"
      >
        <div className="flex flex-col items-center gap-4 text-center">
          <svg
            className="size-12 text-stone-500"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            aria-hidden="true"
          >
            <circle cx="12" cy="12" r="10" />
            <line x1="12" y1="8" x2="12" y2="12" />
            <line x1="12" y1="16" x2="12.01" y2="16" />
          </svg>
          <h1 className="text-2xl font-bold">Application error</h1>
          <p className="max-w-md text-sm text-stone-600">
            A critical system error occurred. Please reload.
          </p>
          <button
            type="button"
            onClick={() => reset()}
            className="rounded-md bg-stone-900 px-4 py-2 text-sm font-semibold text-stone-50 transition-colors duration-150 hover:bg-stone-700 focus-visible:outline-2 focus-visible:-outline-offset-1 focus-visible:outline-stone-900"
          >
            Reload
          </button>
        </div>
      </body>
    </html>
  )
}
