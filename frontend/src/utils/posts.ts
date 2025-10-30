import { auth } from "./api";
import type { PostsPaginatedResponse } from "../types/posts";

export async function fetchPosts(page: number = 1, limit: number = 20): Promise<PostsPaginatedResponse> {
  const res = await auth.get<PostsPaginatedResponse>("/api/posts", { params: { page, limit } });
  return res.data;
}

