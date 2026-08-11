import { buildPageMetadata } from '@/lib/metadata'
import { Suspense } from 'react'
import AdminUsersRoute from '@/components/admin/AdminUsersRoute'

export const generateMetadata = buildPageMetadata('adminUsers')

// D3.2 用户详情深链：/admin/users?id= （静态导出约束，与渠道详情同范式）。
export default function AdminUsers() {
  return (
    <Suspense fallback={null}>
      <AdminUsersRoute />
    </Suspense>
  )
}
