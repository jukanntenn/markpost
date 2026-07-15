export interface Pagination {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
}

// Paginated is the flat wrapped list object returned by all list endpoints
// (api-design.md §4): { items, total, page, limit, total_pages }. The resource
// key is always "items" — the field does not vary per endpoint.
export interface Paginated<T> {
  items: T[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}
