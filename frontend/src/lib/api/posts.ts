import { request } from "./base";
import type { PostsPaginatedResponse } from "@/types/posts";

export const postsApi = {
  list: (page: number, limit: number) =>
    request<PostsPaginatedResponse>("/api/v1/posts", { params: { page, limit } }),
};
