// Workspace-level external-URL banner.
//
// BannersWrapper.tsx (line 352) renders the "Bytebase has not configured
// --external-url" banner when `serverInfo.externalUrl` is empty AND the
// frontend is NOT in dev mode. The banner ships a wrench icon + a
// "Configure now" button that links to the workspace general settings.
//
// `seedTestData` silences the banner by setting a workspace external_url
// during setup-project. This spec is the defense-in-depth pair:
//   1. With external_url set (the default state) — banner is absent.
//   2. With external_url cleared — banner is visible with the wrench +
//      "Configure now" + dismiss controls.
//
// We capture the original external_url in beforeAll and restore it in
// afterAll so the rest of the suite continues with seeded state.

import {
  test,
  expect,
  type Page,
  type BrowserContext,
} from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";

test.setTimeout(120_000);

let env: TestEnv & { api: BytebaseApiClient };
let sharedContext: BrowserContext;
let page: Page;
let originalExternalUrl = "";

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);

  // Snapshot the workspace external_url so we can restore it after the
  // "banner-on" test clears it.
  const setting = await env.api.getSetting("WORKSPACE_PROFILE");
  originalExternalUrl =
    (
      (setting?.value as { workspaceProfile?: { externalUrl?: string } })
        ?.workspaceProfile?.externalUrl ?? ""
    );

  sharedContext = await browser.newContext({
    storageState: ".auth/state.json",
  });
  page = await sharedContext.newPage();
});

test.afterAll(async () => {
  // Always restore the seeded URL — other specs (and the framework's
  // own admin-mode wrench locator scoping) depend on the banner being
  // silenced.
  if (originalExternalUrl) {
    await env.api.setWorkspaceExternalUrl(originalExternalUrl).catch(() => {});
  }
  await sharedContext?.close();
});

test.describe("External-URL banner", () => {
  test.describe.configure({ mode: "serial" });

  test("with external_url set, the banner is not rendered", async () => {
    // Sanity: seedTestData should have run and set a URL already.
    expect(originalExternalUrl).not.toBe("");

    await page.goto(env.baseURL);
    await page.waitForLoadState("networkidle");

    // The configure CTA + wrench icon are the visible signal of the
    // banner. Both must be absent. The CTA is a RouterLink (role="link")
    // styled to look like a button (BannerExternalUrl in BannersWrapper.tsx),
    // so query it by the link role.
    await expect(
      page.getByRole("link", { name: /Configure now/i }),
    ).toHaveCount(0);
  });

  test("clearing external_url makes the banner appear with wrench + Configure now", async () => {
    // Empty string flips `needConfigureExternalUrl()` to true on the
    // server-info store.
    await env.api.setWorkspaceExternalUrl("");

    await page.goto(env.baseURL);
    await page.waitForLoadState("networkidle");

    // The CTA is a RouterLink styled like a button (buttonVariants), so it has
    // role="link", not role="button". (Pre-fix this looked for a button and
    // never matched, even though the product renders the banner correctly.)
    const configureNow = page
      .getByRole("link", { name: /Configure now/i })
      .first();
    await expect(configureNow).toBeVisible({ timeout: 10_000 });

    // The button hosts a Wrench svg from lucide-react. Scope to the
    // button so we don't match an unrelated wrench elsewhere on the
    // page (e.g., the SQL editor's admin-mode button on other routes,
    // though it shouldn't be rendered on the landing page).
    await expect(
      configureNow.locator("svg.lucide-wrench"),
    ).toBeVisible({ timeout: 5_000 });
  });
});
