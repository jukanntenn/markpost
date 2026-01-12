import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./tests",
  timeout: 30000,
  fullyParallel: true,
  workers: 2,
  reporter: [["list"]],
  use: {
    baseURL: "http://localhost:5174/ui/",
    trace: "on-first-retry",
  },
  webServer: {
    command: "VITE_PORT=5174 pnpm dev",
    url: "http://localhost:5174/ui/",
    reuseExistingServer: true,
    timeout: 120000,
  },
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "firefox", use: { ...devices["Desktop Firefox"] }, workers: 1 },
    { name: "webkit", use: { ...devices["Desktop Safari"] } },
  ],
});
