import useSWR from "swr";
import { authFetcher } from "../../swr/fetcher";

export interface User {
  id: number;
  username: string;
  role: string;
  github_id: number | null;
  created_at: string;
  updated_at: string;
}

export interface UsersResponse {
  users: User[];
  total: number;
  page: number;
  page_size: number;
}

export function useUsers(page: number, limit: number = 10) {
  return useSWR<UsersResponse>(
    page ? `/api/admin/users?page=${page}&limit=${limit}` : null,
    authFetcher,
    {
      refreshWhenHidden: false,
      revalidateOnFocus: false,
    }
  );
}
