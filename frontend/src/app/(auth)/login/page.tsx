import { Metadata } from "next";
import LoginPage from "@/components/login/LoginPage";

export const metadata: Metadata = {
  title: "Login - Markpost",
};

export default function Login() {
  return <LoginPage />;
}
