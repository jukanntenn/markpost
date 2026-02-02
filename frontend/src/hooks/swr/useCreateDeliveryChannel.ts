import useSWRMutation from "swr/mutation";
import { auth } from "../../utils/api";
import { withAuthRefresh } from "../../swr/middleware";

export interface CreateDeliveryChannelArgs {
  kind: string;
  name?: string;
  enabled?: boolean;
  webhook_url: string;
  keywords?: string;
}

const sendRequest = async (
  url: string,
  { arg }: { arg: CreateDeliveryChannelArgs }
) => {
  const res = await withAuthRefresh(() => auth.post(url, arg));
  return res.data;
};

export function useCreateDeliveryChannel() {
  return useSWRMutation("/api/delivery/channels", sendRequest);
}
