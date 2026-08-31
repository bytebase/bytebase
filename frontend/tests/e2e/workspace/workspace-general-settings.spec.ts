// Workspace Settings → General → SQL Editor section: the data-export policy
// control. #21081 unified all wording to "access grant" terminology; customers
// then reported they couldn't tell that DISABLING export is how you enable
// approval-gated (access-grant) export — the control was a bare "Enable data
// export" checkbox with no description. The rework retitles it "Allow data
// export without approval" and adds a description naming the "Request export"
// CTA users see in the SQL editor.
//
// Covers:
//   - W1 wording + policy→UI: the retitled label and its description render
//     in the SQL Editor settings section, and the checkbox reflects the LIVE
//     workspace DATA_QUERY policy in both states (checked = export allowed
//     without approval = disableExport:false).
//   - W2 UI→policy wiring: unchecking + Update persists disableExport:true
//     (verified via the API, not just the UI), and re-checking + Update
//     restores it — both directions of the `checked={!disableExport}`
//     inversion in SQLEditorSection.tsx.
//
// State safety: the DATA_QUERY policy is snapshotted in beforeAll and
// restored via API in afterAll. The sql-editor specs that flip the same
// policy run earlier (directory order) and restore their own state; this
// file runs before workspace-seat-limit's license drop.

import {
  test,
  expect,
  type BrowserContext,
  type Page,
} from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";

test.setTimeout(120_000);

let env: TestEnv & { api: BytebaseApiClient };
let workspace = "";
let sharedContext: BrowserContext;
let page: Page;
let originalQueryDataPolicy: Record<string, unknown> = {};

const EXPORT_LABEL = "Allow data export without approval";

async function readDisableExport(): Promise<boolean> {
  const policy = (await env.api.getPolicy(
    `${workspace}/policies/data_query`
  )) as { queryDataPolicy?: Record<string, unknown> } | null;
  return Boolean(policy?.queryDataPolicy?.disableExport);
}

async function setQueryDataPolicy(
  payload: Record<string, unknown>
): Promise<void> {
  await env.api.upsertPolicy(workspace, "data_query", {
    name: `${workspace}/policies/data_query`,
    type: "DATA_QUERY",
    queryDataPolicy: payload,
  });
}

// The checkbox lives inside the FormField title span next to the label text
// (SQLEditorSection.tsx renders `<Checkbox/>{label}<FeatureBadge/>` in one
// inline-flex span), so scope by that span rather than by accessible name —
// the label text is a sibling, not an aria label.
function exportCheckbox() {
  return page
    .locator("span", { hasText: EXPORT_LABEL })
    .first()
    .getByRole("checkbox");
}

async function gotoGeneralSettings(): Promise<void> {
  await page.goto(`${env.baseURL}/setting/general`);
  await page.waitForLoadState("networkidle").catch(() => {});
  await expect(page.getByText(EXPORT_LABEL, { exact: true })).toBeVisible({
    timeout: 10_000,
  });
}

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  ({ workspace } = await env.api.getActuatorInfo());

  // Snapshot the whole queryDataPolicy so afterAll restores every field we
  // may clobber via the UI's Update (it writes the full payload).
  const existing = (await env.api.getPolicy(
    `${workspace}/policies/data_query`
  )) as { queryDataPolicy?: Record<string, unknown> } | null;
  originalQueryDataPolicy = existing?.queryDataPolicy ?? {};

  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
});

test.afterAll(async () => {
  try {
    await setQueryDataPolicy(originalQueryDataPolicy);
  } catch {
    /* best-effort restore */
  }
  await sharedContext?.close();
});

test.describe("data-export policy control wording and wiring", () => {
  test("the retitled checkbox and its description render, reflecting the live policy in both states (W1)", async () => {
    // Known state first (own your fixtures): export allowed without approval.
    await setQueryDataPolicy({ ...originalQueryDataPolicy, disableExport: false });
    await gotoGeneralSettings();

    // The description teaches the access-grant path and quotes the exact
    // "Request export" button label (coupling locked by messages.test.ts).
    await expect(
      page.getByText(
        /In projects that allow access grants, users can still click "Request export"/
      )
    ).toBeVisible();
    await expect(exportCheckbox()).toBeChecked();

    // Flip the policy via API → a fresh load shows the checkbox unchecked.
    await setQueryDataPolicy({ ...originalQueryDataPolicy, disableExport: true });
    await gotoGeneralSettings();
    await expect(exportCheckbox()).not.toBeChecked();
  });

  test("unchecking + Update persists disableExport, re-checking restores it (W2)", async () => {
    // Start from export-allowed so the test owns its precondition.
    await setQueryDataPolicy({ ...originalQueryDataPolicy, disableExport: false });
    await gotoGeneralSettings();
    await expect(exportCheckbox()).toBeChecked();

    // Uncheck → the dirty bar's Update commits the policy.
    await exportCheckbox().click();
    await expect(exportCheckbox()).not.toBeChecked();
    await page.getByRole("button", { name: "Update", exact: true }).click();
    await expect(
      page.getByText("Configuration is updated.").first()
    ).toBeVisible({ timeout: 10_000 });
    await expect
      .poll(readDisableExport, { timeout: 10_000 })
      .toBe(true);

    // Re-check → Update → the policy returns to export-without-approval.
    await exportCheckbox().click();
    await expect(exportCheckbox()).toBeChecked();
    await page.getByRole("button", { name: "Update", exact: true }).click();
    await expect
      .poll(readDisableExport, { timeout: 10_000 })
      .toBe(false);
  });
});
