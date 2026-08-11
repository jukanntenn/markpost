import { AppShellSkeleton } from '@/components/providers/AppShellSkeleton'

// A2.8 层级1：根路由切换的 AppShell 骨架过渡（与 locale/auth hydrate 同款，
// 零跳变）。
export default function Loading() {
  return <AppShellSkeleton />
}
