import { useState } from "react";
import { authApi } from "@/lib/api";

const OAUTH_STATE_KEY = "oauth_state";

// useGitHubOAuth implements the same-page redirect OAuth flow (auth.md §3.1-3.3).
// startOAuth fetches {url, state} from the backend, stores the expected state
// in sessionStorage for the callback page's second-layer check, then navigates
// the whole page to GitHub (no popup). The popup model is deliberately rejected
// (auth.md §3.1) — same-page redirect avoids popup blockers, cross-window
// messaging, and mobile UX problems.
export function useGitHubOAuth() {
  const [loading, setLoading] = useState(false);

  const startOAuth = async () => {
    setLoading(true);
    try {
      const { url, state } = await authApi.getOAuthUrl();
      sessionStorage.setItem(OAUTH_STATE_KEY, state);
      window.location.href = url;
    } catch (err) {
      setLoading(false);
      throw err;
    }
  };

  return { startOAuth, loading };
}

// getExpectedOAuthState returns (and clears) the state stored before the
// redirect, used by the callback page for the front-end second-layer state
// check (the backend is the primary defense; auth.md §3.3).
export function consumeExpectedOAuthState(): string | null {
  const state = sessionStorage.getItem(OAUTH_STATE_KEY);
  sessionStorage.removeItem(OAUTH_STATE_KEY);
  return state;
}
