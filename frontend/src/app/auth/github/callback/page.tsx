import { Suspense } from "react";
import LoginCallbackPage from "@/components/login/LoginCallbackPage";

export default function GitHubCallback() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <LoginCallbackPage />
    </Suspense>
  );
}
