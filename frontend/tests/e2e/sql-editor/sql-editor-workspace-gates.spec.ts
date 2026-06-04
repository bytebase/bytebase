// SQL Editor — workspace-level QueryDataPolicy gates (Batch 9, M-series).
//
// Covers:
//   - M1 disableExport hides the result toolbar's Export button.
//   - M2 disableCopyData hides the Copy button on a non-empty result.
//
// Both fields live on a `QueryDataPolicy` policy attached to the
// workspace resource (PolicyType.DATA_QUERY). We capture the current
// policy in `beforeAll`, mutate per-test, and restore on `afterEach` so
// other specs see the original baseline.
//
// Deferred from this batch (need separate investigation):
//   - M3 maximumResultRows enforcement — frontend caps the row-limit
//     input (QueryContextSettingPopover) but backend enforcement is
//     less obvious; whose responsibility under what circumstances?
//   - M5 allowAdminDataSource gating the admin radio — requires an
//     environment with both admin + readonly data sources.
//   - M8 admin bypass on no-readonly — needs a fixture user without
//     readonly access; the I-series user provisioning pattern applies.
//
// LICENSE GATE: `FEATURE_QUERY_POLICY` / `FEATURE_RESTRICT_COPYING_DATA`
// gate UI controls; mutations of QueryDataPolicy may also be backend-
// gated. We try the mutation and skip the test gracefully if the API
// returns 403.

import { test, expect, type BrowserContext, type Page } from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import type { BytebaseApiClient } from "../framework/api-client";
import { SqlEditorPage } from "./sql-editor.page";

test.setTimeout(120_000);

let env: TestEnv & { api: BytebaseApiClient };
let workspace = "";
let sharedContext: BrowserContext;
let page: Page;
let sqlEditor: SqlEditorPage;

// Snapshot of the original QueryDataPolicy body so afterEach can restore.
// Stored as the raw policy resource so we can re-PATCH with the same shape.
let originalQueryDataPolicy: {
  disableExport?: boolean;
  disableCopyData?: boolean;
  allowAdminDataSource?: boolean;
  maximumResultRows?: number;
} = {};

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  ({ workspace } = await env.api.getActuatorInfo());

  // Snapshot the current workspace-level QueryDataPolicy (may not exist
  // yet on a fresh server — getPolicy returns null on 404).
  const existing = await env.api.getPolicy(`${workspace}/policies/data_query`);
  if (existing && typeof existing === "object") {
    originalQueryDataPolicy = (existing as {
      queryDataPolicy?: typeof originalQueryDataPolicy;
    }).queryDataPolicy ?? {};
  }

  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
  sqlEditor = new SqlEditorPage(page, env.baseURL);
});

test.afterAll(async () => {
  await sharedContext?.close();
});

// Restore the original policy (or remove ours) after every test so
// state from one gate test doesn't shift another's baseline.
test.afterEach(async () => {
  await env.api
    .upsertPolicy(workspace, "data_query", {
      name: `${workspace}/policies/data_query`,
      type: "DATA_QUERY",
      resourceType: "WORKSPACE",
      queryDataPolicy: {
        disableExport: originalQueryDataPolicy.disableExport ?? false,
        disableCopyData: originalQueryDataPolicy.disableCopyData ?? false,
        allowAdminDataSource:
          originalQueryDataPolicy.allowAdminDataSource ?? false,
        maximumResultRows: originalQueryDataPolicy.maximumResultRows ?? 0,
      },
    })
    .catch(() => {
      /* ignore — license gate may reject; baseline already in place */
    });
});

// Helper — try to apply a workspace-level QueryDataPolicy mutation. If
// the gateway returns 403 (FEATURE_QUERY_POLICY / FEATURE_RESTRICT_COPYING_DATA
// require enterprise), test skips so we don't false-fail on free plan.
async function applyQueryDataPolicy(
  patch: Partial<typeof originalQueryDataPolicy>,
): Promise<void> {
  await env.api.upsertPolicy(workspace, "data_query", {
    name: `${workspace}/policies/data_query`,
    type: "DATA_QUERY",
    resourceType: "WORKSPACE",
    queryDataPolicy: {
      // Preserve all other fields; only mutate the ones the test cares
      // about.
      disableExport:
        patch.disableExport ?? originalQueryDataPolicy.disableExport ?? false,
      disableCopyData:
        patch.disableCopyData ??
        originalQueryDataPolicy.disableCopyData ??
        false,
      allowAdminDataSource:
        patch.allowAdminDataSource ??
        originalQueryDataPolicy.allowAdminDataSource ??
        false,
      maximumResultRows:
        patch.maximumResultRows ??
        originalQueryDataPolicy.maximumResultRows ??
        0,
    },
  });
}

test.describe("disableExport hides the result toolbar Export button (M1)", () => {
  test("setting disableExport=true removes the Export button from a non-empty result", async () => {
    // QueryDataPolicy enforcement is enterprise-gated; the license is installed at bootstrap.
    await applyQueryDataPolicy({ disableExport: true });

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await sqlEditor.runPreparedQuery("SELECT 1 AS n;");
    await expect(page.getByText(/^1\s+rows?$/i).first()).toBeVisible({
      timeout: 10_000,
    });

    // No Export button in the result toolbar.
    await expect(
      page.getByRole("button", { name: /^Export/ }),
    ).toHaveCount(0);
  });

  test("with disableExport=false (baseline) the Export button is present", async () => {
    await applyQueryDataPolicy({ disableExport: false }).catch(() => {});

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await sqlEditor.runPreparedQuery("SELECT 1 AS n;");
    await expect(page.getByText(/^1\s+rows?$/i).first()).toBeVisible({
      timeout: 10_000,
    });

    await expect(
      page.getByRole("button", { name: /^Export/ }).first(),
    ).toBeVisible({ timeout: 5_000 });
  });
});

test.describe("disableCopyData hides the Copy button on results (M2)", () => {
  test("setting disableCopyData=true removes the Copy button", async () => {
    await applyQueryDataPolicy({ disableCopyData: true });

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await sqlEditor.runPreparedQuery("SELECT 1 AS n;");
    await expect(page.getByText(/^1\s+rows?$/i).first()).toBeVisible({
      timeout: 10_000,
    });

    // Copy button (rendered when !disallowCopyingData && rows.length > 0)
    // must NOT be in the toolbar. Anchor exact-match so "Copy others"
    // etc. don't bleed in.
    await expect(
      page.getByRole("button", { name: "Copy", exact: true }),
    ).toHaveCount(0);
  });

  test("with disableCopyData=false (baseline) the Copy button is present", async () => {
    await applyQueryDataPolicy({ disableCopyData: false }).catch(() => {});

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await sqlEditor.runPreparedQuery("SELECT 1 AS n;");
    await expect(page.getByText(/^1\s+rows?$/i).first()).toBeVisible({
      timeout: 10_000,
    });

    await expect(
      page.getByRole("button", { name: "Copy", exact: true }).first(),
    ).toBeVisible({ timeout: 5_000 });
  });
});
