import type { Middleware } from "swr";
import { get, set, remove } from "../utils/storage";
import type { LoginResponse } from "../types/auth";
import { anno } from "../utils/api";

let isRefreshing = false;
type PendingItem = { resolve: (token: string) => void; reject: (err: unknown) => void };
let pendingQueue: PendingItem[] = [];

const processQueue = (error: unknown, token: string | null) => {
  pendingQueue.forEach(({ resolve, reject }) => {
    if (error) {
      reject(error);
    } else if (token) {
      resolve(token);
    }
  });
  pendingQueue = [];
};

export const authMiddleware: Middleware = (useSWRNext) => (key, fetcher, config) => {
  const swr = useSWRNext(key, fetcher, config);

  if (swr.error?.status === 401) {
    const loginData = get<LoginResponse | null>("login");
    const refreshToken = loginData?.refresh_token;

    if (refreshToken && !isRefreshing) {
      isRefreshing = true;

      anno
        .post("/api/auth/refresh", { refresh_token: refreshToken })
        .then((res) => {
          const data = res.data;
          if (!data?.access_token || !data?.refresh_token) {
            throw new Error("invalid refresh response");
          }
          set("login", data);
          processQueue(null, data.access_token);
        })
        .catch((err) => {
          processQueue(err, null);
          remove("login");
          window.location.href = "/ui/login";
        })
        .finally(() => {
          isRefreshing = false;
        });
    }
  }

  return swr;
};
