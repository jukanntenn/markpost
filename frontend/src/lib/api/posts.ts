import { request, paginationParams } from './base'
import type { PostsPaginatedResponse } from '@/types/posts'

export const postsApi = {
  // B3.3/F.5: 用户帖子标题搜索 + 分页。
  list: (page: number, limit: number, search?: string) =>
    request<PostsPaginatedResponse>('/api/v1/posts', {
      params: {
        ...(search ? { search } : {}),
        ...paginationParams(page, limit),
      },
    }),
}
