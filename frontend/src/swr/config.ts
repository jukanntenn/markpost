import type { SWRConfiguration } from "swr";
import { authFetcher } from "./fetcher";
import { authMiddleware } from "./middleware";

export const swrConfig: SWRConfiguration = {
  fetcher: authFetcher,
  use: [authMiddleware],
  revalidateOnFocus: true,
  revalidateOnReconnect: true,
  dedupingInterval: 2000,
  errorRetryCount: 3,
  shouldRetryOnError: (err) => {
    return err?.status !== 401 && err?.status !== 404;
  },
};
