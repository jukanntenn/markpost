import { createContext } from "react";

export interface UserInfo {
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

