import { defineConfig, devices } from "@playwright/test";
import * as path from "path";

export default defineConfig({
  testDir: "./tests/e2e",
  testIgnore: ["**/framework/**"],
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: process.env.CI ? "github" : "html",
  globalSetup: path.resolve(__dirname, "tests/e2e/framework/global-setup.ts"),
  globalTeardown: path.resolve(__dirname, "tests/e2e/framework/global-teardown.ts"),
  use: {
    storageState: ".auth/state.json",
    trace: "on-first-retry",
    screenshot: "only-on-failure",
  },
  projects: [
    {
      name: "setup",
      testMatch: /framework\/setup-project\.ts/,
      use: { storageState: undefined },
    },
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
      dependencies: ["setup"],
    },
  ],
});
