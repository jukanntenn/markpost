import type { Pagination } from "./pagination";

export interface AdminUser {
  id: number;
  username: string;
  email: string;
  role: string;
  is_active: boolean;
  created_at: string;
}

export interface AdminUsersResponse {
  users: AdminUser[];
  pagination: Pagination;
}
