import type { Pagination } from "./pagination";

export interface PostListItem {
  id: number;
  qid: string;
  title: string;
  created_at: string;
}

export interface PostsPaginatedResponse {
  posts: PostListItem[];
  pagination: Pagination;
}

export interface CreateTestPostRequest {
  title: string;
  body: string;
}

export interface CreateTestPostResponse {
  id: string;
}

export interface PostKeyResponse {
  post_key: string;
  created_at: string;
}

export interface AdminPost {
  id: string;
  qid: string;
  title: string;
  user_id: number;
  username: string;
  created_at: string;
}

export interface AdminPostsResponse {
  posts: AdminPost[];
  pagination: Pagination;
}
