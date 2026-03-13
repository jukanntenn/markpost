"use client";

import { useState, useEffect, type ReactNode } from "react";
import { get, remove } from "../utils/storage";
import { UserInfoContext } from "./UserInfoContext";
import type { UserInfo } from "./UserInfoContext";

interface UserInfoProviderProps {
  children: ReactNode;
}

export const UserInfoProvider = ({ children }: UserInfoProviderProps) => {
  const [userInfo, setUserInfo] = useState<UserInfo | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    try {
      const loginData = get<UserInfo>("login");
      setUserInfo(loginData);
    } catch (error) {
      console.error("Error reading login data from storage:", error);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const logout = () => {
    setUserInfo(null);
    remove("login");
  };

  const isAuthenticated = !!(
    userInfo?.user?.id &&
    userInfo?.access_token &&
    userInfo?.refresh_token
  );

  const isAdmin = () => {
    return userInfo?.user?.role === "admin";
  };

  if (isLoading) {
    return null;
  }

  return (
    <UserInfoContext.Provider
      value={{
        isAuthenticated,
        userInfo,
        setUserInfo: (info: UserInfo | null) => setUserInfo(info),
        logout,
        isAdmin,
      }}
    >
      {children}
    </UserInfoContext.Provider>
  );
};
