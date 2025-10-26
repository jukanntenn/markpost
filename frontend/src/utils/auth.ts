import type { LoginResponse } from "../types/auth";

export const checkLoginResponse = (data: LoginResponse | null) => {
  return (
    !!data &&
    !!data?.access_token &&
    !!data?.refresh_token &&
    !!data?.user &&
    data.user.id != null &&
    !!data.user.username
  );
};
