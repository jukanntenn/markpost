import { defineConfig, devices } from "@playwright/test";

const frontendPort = process.env.FRONTEND_PORT || "3034";

export default defineConfig({
  testDir: "./tests",
  timeout: 30000,
  fullyParallel: true,
  workers: 2,
  reporter: [["list"]],
  use: {
    baseURL: `http://localhost:${frontendPort}/ui/`,
    trace: "on-first-retry",
  },
  webServer: {
    command: `FRONTEND_PORT=${frontendPort} pnpm dev`,
    url: `http://localhost:${frontendPort}/ui/`,
    reuseExistingServer: true,
    timeout: 120000,
  },
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "firefox", use: { ...devices["Desktop Firefox"] }, workers: 1 },
    { name: "webkit", use: { ...devices["Desktop Safari"] } },
  ],
});
