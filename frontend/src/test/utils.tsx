import { render } from "@testing-library/react";
import { I18nextProvider } from "react-i18next";
import i18n from "../i18n";
import type { LoginResponse } from "../types/auth";
import { SWRConfig } from "swr";

export function renderWithI18n(ui: React.ReactElement) {
  return render(<I18nextProvider i18n={i18n}>{ui}</I18nextProvider>);
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
    access_token: "test_token",
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
    <SWRConfig
      value={{
        dedupingInterval: 0,
        revalidateOnFocus: false,
        revalidateOnReconnect: false,
        refreshWhenHidden: false,
        refreshInterval: 0,
      }}
    >
      <I18nextProvider i18n={i18n}>{children}</I18nextProvider>
    </SWRConfig>
  );
}
