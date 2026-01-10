import useSWRMutation from "swr/mutation";
import { anno } from "../../utils/api";
import type { CreateTestPostResponse } from "../../types/posts";

export interface CreateTestPostArgs {
  title: string;
  body: string;
}

const sendRequest = async (
  url: string,
  { arg }: { arg: CreateTestPostArgs }
): Promise<CreateTestPostResponse> => {
  const res = await anno.post<CreateTestPostResponse>(url, arg);
  return res.data;
};

export function useCreateTestPost(postKey: string) {
  return useSWRMutation(postKey ? `/${postKey}` : null, sendRequest);
}
