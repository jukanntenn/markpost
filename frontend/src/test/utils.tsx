import { render } from "@testing-library/react";
import type { LoginResponse } from "../types/auth";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
    },
  },
});

export function renderWithProviders(ui: React.ReactElement) {
  return render(
    <QueryClientProvider client={queryClient}>
      {ui}
    </QueryClientProvider>
  );
}

export function createMockUser(overrides = {}) {
  return {
    id: 1,
    username: "testuser",
    ...overrides,
  };
}

export function createMockAuth(overrides = {}) {
  return {
    token: "test_token",
    refresh_token: "test_refresh",
    user: createMockUser(),
    ...overrides,
  };
}

export function setMockAuth(auth: LoginResponse) {
  localStorage.setItem("markpost_dev_login", JSON.stringify(auth));
}

export function clearMockAuth() {
  localStorage.removeItem("markpost_dev_login");
}

export function createWrapper() {
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
}
