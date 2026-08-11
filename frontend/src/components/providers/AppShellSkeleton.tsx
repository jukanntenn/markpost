import { Skeleton } from '@/components/ui/skeleton'

// A2.9 统一骨架：镜像新 AppShell 结构（Topbar + 侧栏 + Content），用于
// locale 加载、auth hydrate、路由切换三阶段，零跳变。
// 响应式：移动端 = Topbar 占位 + Content shimmer（无侧栏占位）；
// 桌面端 = Topbar + Sidebar 占位 + Content shimmer（lg: 前缀切换）。
export function AppShellSkeleton() {
  return (
    <div className="min-h-svh">
      <header className="sticky top-0 z-50 h-(--header-height) w-full border-b bg-background/80 backdrop-blur">
        <div className="mx-auto flex h-full max-w-(--container-max) items-center gap-2 px-4 md:px-6 lg:px-8">
          <Skeleton className="size-11 rounded-md lg:hidden" />
          <Skeleton className="size-6 rounded-sm" />
          <div className="flex-1" />
          <Skeleton className="size-11 rounded-md lg:size-10" />
          <Skeleton className="h-11 w-28 rounded-md lg:h-10" />
        </div>
      </header>
      <div className="mx-auto flex w-full max-w-(--container-max)">
        <aside className="hidden w-64 shrink-0 border-r border-sidebar-border lg:block">
          <div className="space-y-3 px-3 py-6">
            <Skeleton className="h-9 w-full" />
            <Skeleton className="h-9 w-full" />
            <Skeleton className="h-9 w-4/5" />
            <Skeleton className="mt-6 h-9 w-full" />
            <Skeleton className="h-9 w-3/5" />
          </div>
        </aside>
        <main
          aria-busy="true"
          aria-live="polite"
          className="min-w-0 flex-1 px-4 py-6 md:px-6 lg:px-8 lg:py-12"
        >
          <Skeleton className="mb-6 h-8 w-40" />
          <div className="rounded-lg border bg-card">
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
      </div>
    </div>
  )
}
