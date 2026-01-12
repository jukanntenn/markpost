import type { SWRConfiguration } from "swr";
import { authFetcher } from "./fetcher";
import { authMiddleware } from "./middleware";

type ErrorWithStatus = { status?: number };
type ErrorWithResponseStatus = { response?: { status?: number } };

const getStatus = (err: unknown): number | undefined => {
  const status = (err as ErrorWithStatus | undefined)?.status;
  if (typeof status === "number") return status;
  const responseStatus = (err as ErrorWithResponseStatus | undefined)?.response?.status;
  if (typeof responseStatus === "number") return responseStatus;
  return undefined;
};

export const swrConfig: SWRConfiguration = {
  fetcher: authFetcher,
  use: [authMiddleware],
  revalidateOnFocus: true,
  revalidateOnReconnect: true,
  dedupingInterval: 2000,
  errorRetryCount: 3,
  shouldRetryOnError: (err) => {
    const status = getStatus(err);
    const message = (err as Error)?.message;
    if (message === "No access token available") {
      return false;
    }
    return status !== 401 && status !== 404;
  },
};
