import useSWR from "swr";
import { authFetcher } from "../../swr/fetcher";

export interface AdminChannel {
  id: number;
  user_id: number;
  username: string;
  kind: string;
  name: string;
  enabled: boolean;
  webhook_url: string;
  keywords: string;
  created_at: string;
  updated_at: string;
}

export interface AdminChannelsResponse {
  channels: AdminChannel[];
}

export function useAdminChannels() {
  return useSWR<AdminChannelsResponse>("/api/admin/channels", authFetcher, {
    refreshWhenHidden: false,
    revalidateOnFocus: false,
  });
}

