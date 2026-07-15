import type { Paginated } from "./pagination";

export interface PostListItem {
  id: number;
  qid: string;
  title: string;
  created_at: string;
}

export type PostsPaginatedResponse = Paginated<PostListItem>;

export interface CreateTestPostRequest {
  title: string;
  body: string;
}

export interface CreateTestPostResponse {
  id: string;
}

export interface AdminPost {
  qid: string;
  title: string;
  user_id: number;
  username: string;
  created_at: string;
}

export type AdminPostsResponse = Paginated<AdminPost>;
