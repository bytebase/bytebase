import { defineConfig, devices } from "@playwright/test";
import * as path from "path";

const headed = !!process.env.BYTEBASE_HEADED;

export default defineConfig({
  testDir: "./tests/e2e",
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: process.env.CI ? "github" : "html",
  globalSetup: path.resolve(__dirname, "tests/e2e/framework/global-setup.ts"),
  globalTeardown: path.resolve(__dirname, "tests/e2e/framework/global-teardown.ts"),
  use: {
    headless: !headed,
    // Set BYTEBASE_BROWSER_CHANNEL=chrome to drive the locally installed
    // Chrome instead of the downloaded Playwright Chromium (useful when
    // the browser download is unavailable).
    channel: process.env.BYTEBASE_BROWSER_CHANNEL || undefined,
    trace: "on-first-retry",
    screenshot: "only-on-failure",
  },
  projects: [
    {
      name: "setup",
      testMatch: /framework\/setup-project\.ts/,
    },
    {
      name: "chromium",
      testIgnore: ["**/framework/**"],
      use: { ...devices["Desktop Chrome"], storageState: ".auth/state.json" },
      dependencies: ["setup"],
    },
  ],
});
