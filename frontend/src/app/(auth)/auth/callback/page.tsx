import { Metadata } from "next";
import { Suspense } from "react";
import LoginCallbackPage from "@/components/login/LoginCallbackPage";

export const metadata: Metadata = {
  title: "OAuth Callback - Markpost",
};

export default function AuthCallback() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <LoginCallbackPage />
    </Suspense>
  );
}
