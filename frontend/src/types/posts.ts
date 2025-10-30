export interface PostListItem {
  id: string;
  title: string;
  created_at: string;
}

export interface PostsPaginatedResponse {
  posts: PostListItem[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    total_pages: number;
  };
}

export interface CreateTestPostRequest {
  title: string;
  body: string;
}

export interface CreateTestPostResponse {
  id: string;
}
