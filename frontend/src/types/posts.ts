export interface PostListItem {
  id: string;
  qid: string;
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

export interface PostKeyResponse {
  post_key: string;
  created_at: string;
}
