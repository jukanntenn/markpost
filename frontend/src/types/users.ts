import type { UserRole } from "./auth";
import type { Paginated } from "./pagination";

export interface AdminUser {
  id: number;
  username: string;
  email: string;
  role: UserRole;
  is_active: boolean;
  created_at: string;
}

export type AdminUsersResponse = Paginated<AdminUser>;
