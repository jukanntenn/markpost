import { UserInfoContext } from "../components/UserInfoContext";
import { useContext } from "react";

export const useUserInfo = () => {
  return useContext(UserInfoContext);
};
