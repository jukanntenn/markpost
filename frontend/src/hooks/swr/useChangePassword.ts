import useSWRMutation from "swr/mutation";
import { auth } from "../../utils/api";

export interface ChangePasswordArgs {
  current_password: string;
  new_password: string;
}

const sendRequest = async (url: string, { arg }: { arg: ChangePasswordArgs }) => {
  return auth.post(url, arg);
};

export function useChangePassword() {
  return useSWRMutation("/api/auth/change-password", sendRequest);
}
