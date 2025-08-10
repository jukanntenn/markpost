import { createContext, useEffect, useState } from "react";
import { get, remove, set } from "../utils/storage";

// 用户信息接口，与后端返回的数据结构匹配
interface UserInfo {
  user: {
    id: number;
    username: string;
  };
  access_token: string;
  refresh_token: string;
}

interface UserInfoContextType {
  isAuthenticated: boolean;
  userInfo: UserInfo | null;
  setUserInfo: (info: UserInfo | null) => void;
  logout: () => void;
}

export const UserInfoContext = createContext<UserInfoContextType>({
  isAuthenticated: false,
  userInfo: null,
  setUserInfo: () => {},
  logout: () => {},
});

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

  useEffect(() => {
    if (userInfo) {
      // 存储到 "login" 键，与 LoginCallback 保持一致
      set("login", userInfo);
    }
  }, [userInfo]);

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

export type { UserInfo };
