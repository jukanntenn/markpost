import useSWR from "swr";
import { authFetcher } from "../../swr/fetcher";

export interface PostKeyResponse {
  post_key: string;
  created_at: string;
}

export function usePostKey() {
  return useSWR<PostKeyResponse>("/api/post_key", authFetcher);
}
