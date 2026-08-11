import { AdminRoute } from '@/components/auth/AdminRoute'

// A2 统一 AppShell：admin 导航并入侧栏"管理"分组，不再有独立 AdminLayout。
// 非 admin 已登录用户访问 → AdminRoute 渲染"无权限"友好态。
export default function AdminLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return <AdminRoute>{children}</AdminRoute>
}
