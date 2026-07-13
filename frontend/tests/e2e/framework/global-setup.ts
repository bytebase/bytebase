import { execFileSync } from "node:child_process";
import { saveTestEnv } from "./env";
import { cleanupOrphans, startServer, stopServer } from "./mode-start-new-bytebase";

// Playwright browsers live in a global cache (e.g. ~/Library/Caches/ms-playwright)
// keyed to the @playwright/test version, NOT in node_modules — so `pnpm i` never
// fetches them, and bumping @playwright/test silently invalidates the cache
// (the matching Chromium build simply isn't there until re-downloaded). Ensure
// the version-matched Chromium is present before any spec launches it. This is
// idempotent: a no-op with no download when the correct build is already cached.
function ensureBrowser() {
  // When BYTEBASE_BROWSER_CHANNEL is set, playwright.config.ts drives a locally
  // installed browser (e.g. Chrome) instead of the downloaded Playwright
  // Chromium — a deliberate escape hatch for when the browser download is
  // unavailable (e.g. offline). Installing Chromium here would defeat that hatch
  // by forcing the very download it exists to avoid, so skip it and let the
  // channel browser handle the run. Mirrors the config's `channel` selection.
  if (process.env.BYTEBASE_BROWSER_CHANNEL) {
    return;
  }
  execFileSync("pnpm", ["exec", "playwright", "install", "chromium"], {
    stdio: "inherit",
  });
}

async function globalSetup() {
  ensureBrowser();
  cleanupOrphans();

  let server: Awaited<ReturnType<typeof startServer>>;
  try {
    server = await startServer();
  } catch (err) {
    // startServer spawns the server process and creates its temp data dir
    // before it can fail (e.g. a /healthz timeout when the boot is slow).
    // Playwright does NOT run globalTeardown when globalSetup throws, so tear
    // the half-started server down here — otherwise a failed boot orphans a
    // process group and temp dir that starve every subsequent run's boot.
    stopServer();
    throw err;
  }

  const { baseURL, adminEmail, adminPassword } = server;
  saveTestEnv({
    baseURL, adminEmail, adminPassword,
    project: "", instance: "", instanceId: "", database: "", databaseId: "",
  });
}

export default globalSetup;
