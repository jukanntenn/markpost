import type { UseMutationOptions } from "@tanstack/react-query";

import { toast } from "@/stores/toast";

export function mutationOptions<TData, TVariables>(
  opts: Omit<UseMutationOptions<TData, Error, TVariables>, "onError"> & {
    onError?: (err: Error) => void;
  },
): UseMutationOptions<TData, Error, TVariables> {
  const { onError, ...rest } = opts;
  return {
    ...rest,
    onError: onError ?? ((err) => toast.error(err.message)),
  };
}

export function mutationSuccess(
  message: string,
  ...invalidations: Array<() => Promise<void>>
) {
  return {
    onSuccess: async () => {
      toast.success(message);
      await Promise.all(invalidations.map((fn) => fn()));
    },
  };
}

export const setErrorOnError =
  (setError: (msg: string) => void) => (err: Error) => setError(err.message);
