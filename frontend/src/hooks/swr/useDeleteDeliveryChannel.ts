import useSWRMutation from "swr/mutation";
import { auth } from "../../utils/api";
import { withAuthRefresh } from "../../swr/middleware";

export interface DeleteDeliveryChannelArgs {
  id: number;
}

const sendRequest = async (_url: string, { arg }: { arg: DeleteDeliveryChannelArgs }) => {
  const res = await withAuthRefresh(() =>
    auth.delete(`/api/delivery/channels/${arg.id}`)
  );
  return res.data;
};

export function useDeleteDeliveryChannel() {
  return useSWRMutation("/api/delivery/channels", sendRequest);
}
