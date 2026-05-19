export interface Pagination {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
}

export type Paginated<T, TKey extends string> = Record<TKey, T[]> & {
  pagination: Pagination;
};
