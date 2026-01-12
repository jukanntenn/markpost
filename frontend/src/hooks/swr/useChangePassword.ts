import useSWRMutation from "swr/mutation";
import { auth } from "../../utils/api";
import { withAuthRefresh } from "../../swr/middleware";

export interface ChangePasswordArgs {
  current_password: string;
  new_password: string;
}

const sendRequest = async (url: string, { arg }: { arg: ChangePasswordArgs }) => {
  return withAuthRefresh(() => auth.post(url, arg));
};

export function useChangePassword() {
  return useSWRMutation("/api/auth/change-password", sendRequest);
}
