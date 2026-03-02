import useSWR from "swr";
import { authFetcher } from "../../swr/fetcher";

export interface AdminPostUser {
  id: number;
  username: string;
}

export interface AdminPost {
  id: number;
  qid: string;
  title: string;
  body: string;
  user_id: number;
  user?: AdminPostUser;
  created_at: string;
  updated_at: string;
}

export interface AdminPostsResponse {
  posts: AdminPost[];
  total: number;
  page: number;
  page_size: number;
}

export function useAdminPosts(page: number, limit: number, search: string) {
  const params = new URLSearchParams();
  params.set("page", String(page));
  params.set("limit", String(limit));
  if (search.trim()) params.set("search", search.trim());

  const key = page ? `/api/admin/posts?${params.toString()}` : null;

  return useSWR<AdminPostsResponse>(key, authFetcher, {
    refreshWhenHidden: false,
    revalidateOnFocus: false,
  });
}

