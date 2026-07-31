import { Skeleton } from '@/components/ui/skeleton'

// AppShellSkeleton mirrors the DashboardLayout chrome (sticky topbar + max-w
// content region) so the loading frame is layout-stable: the real UI fills the
// same boxes once messages load, avoiding a visible jump. Shown by
// LocaleProvider while the locale messages chunk is still loading.
export function AppShellSkeleton() {
  return (
    <>
      <header className="sticky top-0 z-50 w-full border-b bg-background/80 backdrop-blur">
        <div className="mx-auto flex h-14 max-w-[1200px] items-center justify-between px-6">
          <Skeleton className="size-6 rounded-sm" />
          <nav className="flex items-center gap-1">
            <Skeleton className="h-9 w-20" />
            <Skeleton className="h-9 w-20" />
            <Skeleton className="h-9 w-20" />
          </nav>
          <div className="flex items-center gap-2">
            <Skeleton className="size-9 rounded-md" />
            <Skeleton className="h-9 w-28" />
          </div>
        </div>
      </header>
      <main
        aria-busy="true"
        aria-live="polite"
        className="mx-auto w-full max-w-[1200px] px-6 py-6 md:py-8 lg:py-12"
      >
        <Skeleton className="mb-6 h-8 w-40" />
        <div className="rounded-xl border bg-card">
          <div className="flex items-center gap-2 border-b p-6 pb-4">
            <Skeleton className="size-4" />
            <Skeleton className="h-4 w-24" />
          </div>
          <ul className="divide-y">
            {Array.from({ length: 4 }).map((_, i) => (
              <li key={i} className="flex items-center gap-4 p-4">
                <Skeleton className="size-9 rounded-full" />
                <div className="flex-1 space-y-2">
                  <Skeleton className="h-4 w-1/3" />
                  <Skeleton className="h-3 w-1/2" />
                </div>
                <Skeleton className="size-8 rounded-md" />
              </li>
            ))}
          </ul>
        </div>
      </main>
    </>
  )
}
