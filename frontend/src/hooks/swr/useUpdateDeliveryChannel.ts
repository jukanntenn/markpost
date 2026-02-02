import useSWRMutation from "swr/mutation";
import { auth } from "../../utils/api";
import { withAuthRefresh } from "../../swr/middleware";

export interface UpdateDeliveryChannelArgs {
  id: number;
  name?: string;
  enabled?: boolean;
  webhook_url?: string;
  keywords?: string;
}

const sendRequest = async (
  _url: string,
  { arg }: { arg: UpdateDeliveryChannelArgs }
) => {
  const { id, ...payload } = arg;
  const res = await withAuthRefresh(() =>
    auth.put(`/api/delivery/channels/${id}`, payload)
  );
  return res.data;
};

export function useUpdateDeliveryChannel() {
  return useSWRMutation("/api/delivery/channels", sendRequest);
}
