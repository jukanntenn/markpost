'use client'

import { useSearchParams } from 'next/navigation'
import AdminUsersPage from '@/components/admin/AdminUsersPage'
import AdminUserDetailPage from '@/components/admin/AdminUserDetailPage'

// D3.2 用户列表 ↔ 详情切换：/admin/users?id= 深链展示详情。
export default function AdminUsersRoute() {
  const searchParams = useSearchParams()
  const id = searchParams.get('id')
  return id ? <AdminUserDetailPage /> : <AdminUsersPage />
}
