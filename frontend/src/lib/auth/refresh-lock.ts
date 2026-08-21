const REFRESH_LOCK = 'markpost:auth:refresh'

// The refresh token is one-time rotating: the backend revokes the user's
// entire session family when a consumed token is replayed (theft detection).
// With N tabs each holding an in-memory copy of the pair, uncoordinated
// refreshes make the loser replay a dead token and log everyone out. An
// origin-scoped Web Lock serializes the rotation across tabs; on browsers
// without the API this falls through to the unlocked call, which is exactly
// the pre-lock behavior — racy but no worse.
export async function withRefreshLock<T>(fn: () => Promise<T>): Promise<T> {
  if (typeof navigator !== 'undefined' && navigator.locks) {
    return await navigator.locks.request(
      REFRESH_LOCK,
      { mode: 'exclusive' },
      async () => fn(),
    )
  }
  return fn()
}
