import { UserInfoContext } from "../components/UserInfoProvider";
import { useContext } from "react";

export const useUserInfo = () => {
  return useContext(UserInfoContext);
};
