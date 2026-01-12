import { useState } from "react";
import { get, remove } from "../utils/storage";
import { UserInfoContext } from "./UserInfoContext";
import type { UserInfo } from "./UserInfoContext";

export const UserInfoProvider = ({ children }: { children: JSX.Element }) => {
  const [userInfo, setUserInfo] = useState<UserInfo | null>(() => {
    try {
      // 从 storage 中读取 "login" 键的数据，与 LoginCallback 保持一致
      const loginData = get<UserInfo>("login");
      return loginData || null;
    } catch (error) {
      console.error("Error reading login data from storage:", error);
      return null;
    }
  });

  // 登出函数
  const logout = () => {
    setUserInfo(null);
    remove("login");
  };

  // 认证状态：检查是否有完整的用户信息和 token
  const isAuthenticated = !!(
    userInfo?.user?.id &&
    userInfo?.access_token &&
    userInfo?.refresh_token
  );

  return (
    <UserInfoContext.Provider
      value={{
        isAuthenticated,
        userInfo,
        setUserInfo: (info: UserInfo | null) => setUserInfo(info),
        logout,
      }}
    >
      {children}
    </UserInfoContext.Provider>
  );
};
