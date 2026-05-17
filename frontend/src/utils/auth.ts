import type { LoginResponse } from "@/lib/api/auth";

export const checkLoginResponse = (data: LoginResponse | null) => {
  return (
    !!data &&
    !!data?.token &&
    !!data?.refresh_token &&
    !!data?.user &&
    data.user.id != null &&
    !!data.user.username
  );
};
