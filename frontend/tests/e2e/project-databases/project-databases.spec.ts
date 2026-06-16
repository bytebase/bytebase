// Project Databases page — selection action bar permission gating.
//
// BYT-9609 (FIXED, #20450; cherry-picked to 3.18.1): a user whose only role on a
// project is Project Developer opened the project's Database page, checked a
// database row, and found EVERY button on the selection action bar (Change
// Database, Export Data, Sync Schema, …) disabled — even though projectDeveloper
// grants those permissions. Visiting the project Members page and returning made
// them clickable, because only the Members route loaded the project IAM policy
// into the React app store; every other project route evaluated permissions
// against an EMPTY policy. The fix loads the project IAM policy on route entry
// for all project routes.
//
// Per AGENTS.md "F. Test by role": create scoped users via the API, sign each
// into its own context, and COLD-load /databases (without visiting Members
// first — that's the condition the bug needs).

import {
  test,
  expect,
  type Browser,
  type BrowserContext,
  type Page,
} from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";
import { signInBrowserAs } from "../framework/sign-in";

test.setTimeout(180_000);

let env: TestEnv & { api: BytebaseApiClient };
let projectId: string;

const TEST_PASSWORD = "e2e-byt9609-pw-1!"; // NOSONAR — e2e fixture only

const DEV_USER = {
  email: "e2e-byt9609-dev@example.com",
  title: "E2E 9609 Dev",
  authFile: ".auth/byt9609-dev.json",
  role: "roles/projectDeveloper",
};
const VIEWER_USER = {
  email: "e2e-byt9609-viewer@example.com",
  title: "E2E 9609 Viewer",
  authFile: ".auth/byt9609-viewer.json",
  role: "roles/projectViewer",
};

const contexts: BrowserContext[] = [];

async function openColdDatabasesPage(
  browser: Browser,
  authFile: string,
): Promise<Page> {
  // Fresh context + first navigation straight to /databases = cold IAM store,
  // exactly the repro condition (never touch /members first).
  const context = await browser.newContext({ storageState: authFile });
  contexts.push(context);
  const page = await context.newPage();
  return page;
}

// Check the env database's row checkbox so the selection action bar appears.
async function selectDatabaseRow(page: Page): Promise<void> {
  await page.goto(`${env.baseURL}/projects/${projectId}/databases`);
  await page.keyboard.press("Escape").catch(() => {});
  await page.waitForLoadState("networkidle").catch(() => {});

  const row = page
    .getByRole("row")
    .filter({ hasText: env.databaseId })
    .first();
  await expect(row).toBeVisible({ timeout: 15_000 });
  await row.getByRole("checkbox").first().click();

  // The sticky SelectionActionBar shows "1 selected" once a row is checked.
  await expect(page.getByText("1 selected").first()).toBeVisible({
    timeout: 10_000,
  });
}

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  projectId = env.project.split("/").pop()!;
  await env.api.login(env.adminEmail, env.adminPassword);

  for (const u of [DEV_USER, VIEWER_USER]) {
    try {
      await env.api.createUser(u.email, TEST_PASSWORD, u.title);
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      if (!msg.includes("(409)")) throw err;
    }
    await env.api.appendProjectBinding(env.project, u.role, [`user:${u.email}`]);
    await signInBrowserAs(
      browser,
      env.baseURL,
      u.email,
      TEST_PASSWORD,
      u.authFile,
    );
  }
});

test.afterAll(async () => {
  for (const c of contexts) await c.close();
});

test.describe("Project developer action bar is enabled on a cold Database page (BYT-9609)", () => {
  test("a projectDeveloper sees Change Database / Export Data / Sync Schema ENABLED without visiting Members first", async ({
    browser,
  }) => {
    const page = await openColdDatabasesPage(browser, DEV_USER.authFile);
    await selectDatabaseRow(page);

    // The regression: pre-fix these rendered disabled because the project IAM
    // policy was never fetched on the /databases route. Playwright auto-retries
    // toBeEnabled, tolerating the async IAM fetch the fix performs on entry.
    for (const name of ["Change Database", "Export Data", "Sync Schema"]) {
      await expect(
        page.getByRole("button", { name, exact: true }),
        `"${name}" must be enabled for a projectDeveloper on a cold Database page`,
      ).toBeEnabled({ timeout: 15_000 });
    }
  });

  test("a projectViewer sees Change Database / Sync Schema DISABLED (the bar genuinely gates on IAM)", async ({
    browser,
  }) => {
    // Vacuity guard: proves the developer's enabled state above reflects real
    // permissions, not a permissions-off regression. projectViewer lacks the
    // change/sync permissions, so those buttons must be disabled.
    const page = await openColdDatabasesPage(browser, VIEWER_USER.authFile);
    await selectDatabaseRow(page);

    for (const name of ["Change Database", "Sync Schema"]) {
      await expect(
        page.getByRole("button", { name, exact: true }),
        `"${name}" must be disabled for a projectViewer`,
      ).toBeDisabled({ timeout: 15_000 });
    }
  });
});
