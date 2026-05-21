import { render } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import type { LoginResponse } from "@/types/auth";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { NextIntlClientProvider } from "next-intl";
import en from "../i18n/locales/en.json";

type WrapperComponent = React.ComponentType<{ children: React.ReactNode }>;

export function renderWithProviders(
  ui: React.ReactElement,
  options?: { wrapper?: WrapperComponent }
) {
  const Wrapper = options?.wrapper ?? (({ children }) => <>{children}</>);
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });
  return render(
    <QueryClientProvider client={client}>
      <NextIntlClientProvider locale="en" messages={en}>
        <Wrapper>{ui}</Wrapper>
      </NextIntlClientProvider>
    </QueryClientProvider>
  );
}

export function mockMatchMedia() {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
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

export function createQueryWrapper() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}
