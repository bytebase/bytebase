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
import { execSql, getInstancePgPort, querySql } from "../framework/psql";
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

// --- Automatic data source routes DML to the admin data source (BYT-9557) ----
//
// BYT-9557 (FIXED, #20390): on an instance with BOTH an admin and a read-only
// data source, the SQL editor in "Automatic" mode routed EVERY statement —
// including DDL/DML — to the read-only data source. With "Allow querying the
// admin data source" enabled, a DML statement still failed with a Postgres
// "permission denied" error because it executed as the read-only DB user;
// manually switching to the admin data source made the same statement succeed.
// The fix classifies the statement: read-only statements go to the read-only
// data source, but DDL/DML are routed to the admin data source.
//
// This is the M5 prerequisite the file header deferred (needs an instance with
// both an admin + a read-only data source). It mutates the shared env instance,
// so it adds the RO data source AFTER the admin source (index 0 stays admin, so
// getInstancePgPort is unaffected) and removes it in afterAll.

test.describe("Automatic mode routes DML to the admin data source (BYT-9557)", () => {
  const RO_DS_ID = "e2e-ro-9557";
  const RO_USER = "bb_e2e_ro_9557";
  const RO_PROBE_TABLE = "e2e_9557_probe";
  const RO_PASSWORD = "bytebase"; // NOSONAR — e2e fixture only
  let pgPort = "";
  let adminHost = "";
  let roAdded = false;

  test.beforeAll(async () => {
    test.setTimeout(120_000);
    pgPort = await getInstancePgPort(env);
    const instance = await env.api.getInstance(env.instance);
    adminHost = instance.dataSources?.[0]?.host ?? "/tmp";

    // Create a SELECT-only Postgres user on the sample instance (idempotent).
    // execSql throws synchronously; swallow the pre-create cleanup (the user may
    // not exist yet on a first run).
    try {
      execSql(env.databaseId, pgPort, `DROP OWNED BY ${RO_USER}`);
    } catch {
      /* user may not exist yet */
    }
    try {
      execSql(env.databaseId, pgPort, `DROP USER IF EXISTS ${RO_USER}`);
    } catch {
      /* may fail if it owns objects — DROP OWNED above handles the common case */
    }
    execSql(
      env.databaseId,
      pgPort,
      `CREATE USER ${RO_USER} PASSWORD '${RO_PASSWORD}'`,
    );
    execSql(
      env.databaseId,
      pgPort,
      `GRANT CONNECT ON DATABASE ${env.databaseId} TO ${RO_USER}`,
    );
    execSql(
      env.databaseId,
      pgPort,
      `GRANT USAGE ON SCHEMA public TO ${RO_USER}`,
    );
    execSql(
      env.databaseId,
      pgPort,
      `GRANT SELECT ON ALL TABLES IN SCHEMA public TO ${RO_USER}`,
    );
    // A scratch table OWNED BY the admin (bbsample) that the RO user can only
    // SELECT — never UPDATE. A plain UPDATE of this table thus succeeds only if
    // automatic mode routed the DML to the admin data source (the fix); if it
    // routed to the read-only data source it fails with "permission denied".
    execSql(env.databaseId, pgPort, `DROP TABLE IF EXISTS ${RO_PROBE_TABLE}`);
    execSql(
      env.databaseId,
      pgPort,
      `CREATE TABLE ${RO_PROBE_TABLE} (id int PRIMARY KEY, v text)`,
    );
    execSql(env.databaseId, pgPort, `INSERT INTO ${RO_PROBE_TABLE} VALUES (1, 'before')`);
    execSql(env.databaseId, pgPort, `GRANT SELECT ON ${RO_PROBE_TABLE} TO ${RO_USER}`);

    // Add the READ_ONLY data source pointing at the same Postgres.
    await env.api.addDataSource(env.instance, {
      id: RO_DS_ID,
      type: "READ_ONLY",
      username: RO_USER,
      password: RO_PASSWORD,
      host: adminHost,
      port: pgPort,
    });
    roAdded = true;
  });

  test.afterAll(async () => {
    if (roAdded) await env.api.removeDataSource(env.instance, RO_DS_ID);
    try {
      execSql(env.databaseId, pgPort, `DROP OWNED BY ${RO_USER}`);
      execSql(env.databaseId, pgPort, `DROP USER IF EXISTS ${RO_USER}`);
    } catch {
      /* best-effort teardown */
    }
    try {
      execSql(env.databaseId, pgPort, `DROP TABLE IF EXISTS ${RO_PROBE_TABLE}`);
    } catch {
      /* best-effort teardown */
    }
  });

  test("SELECT routes to the read-only data source; DDL succeeds (routed to admin)", async () => {
    test.setTimeout(120_000);
    // Allow the admin data source for automatic DDL/DML routing.
    await applyQueryDataPolicy({ allowAdminDataSource: true });

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(1200);

    // Sanity oracle: in Automatic mode (the default — empty dataSourceId), a
    // read-only statement routes to the read-only data source, so current_user
    // is the RO user. This also proves the RO data source is wired and selected
    // automatically.
    await sqlEditor.runPreparedQuery("SELECT current_user;");
    const selectCell = page.locator(
      '[data-row-index="0"] [data-col-index="1"]',
    );
    await expect(selectCell).toBeVisible({ timeout: 10_000 });
    await expect(
      selectCell,
      "Automatic mode must route a read-only SELECT to the read-only data source",
    ).toHaveText(RO_USER, { timeout: 5000 });

    // Regression oracle: a plain UPDATE (no RETURNING — so it can't be treated
    // as a read) of the admin-owned scratch table, which the RO user can only
    // SELECT. If automatic mode routed the DML to the read-only data source it
    // fails with "permission denied for table"; the fix routes DML to the ADMIN
    // data source (the table's owner), so it succeeds. Pre-fix this rendered the
    // permission-denied error.
    // Re-open the editor in a fresh tab before the UPDATE. runPreparedQuery's
    // setEditorContent does NOT reliably clear a SECOND statement once a result
    // is on screen: the prior SELECT's text is retained, the editor ends up with
    // "SELECT current_user;UPDATE …", Run executes only the SELECT, and the
    // UPDATE never runs — so the old "no permission-denied error" assertion
    // passed vacuously (no UPDATE → no denial). A fresh editor runs the UPDATE
    // in isolation. (Verified: with the fresh editor the UPDATE executes, the UI
    // shows "rows affected", and the row becomes 'after'.)
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(1200);
    await sqlEditor.setEditorContent(
      `UPDATE ${RO_PROBE_TABLE} SET v = 'after' WHERE id = 1;`,
    );
    await sqlEditor.runButton.click();

    // Positive oracle anchored to THIS statement's effect: poll the row as the
    // owner until v flips 'before' -> 'after'. This both waits for the UPDATE to
    // resolve and proves it actually applied. If automatic mode (wrongly) routed
    // the DML to the read-only data source it would be permission-denied and v
    // would stay 'before', so the poll would time out and fail — unlike a bare
    // "no error visible yet" assertion, which can pass before the statement even
    // resolves.
    await expect
      .poll(
        () =>
          querySql(
            env.databaseId,
            pgPort,
            `SELECT v FROM ${RO_PROBE_TABLE} WHERE id = 1`,
          ),
        {
          timeout: 15_000,
          message:
            "the admin-routed UPDATE must apply (v='after'); if automatic mode " +
            "routed the DML to the read-only data source it would be denied and " +
            "v would stay 'before'",
        },
      )
      .toBe("after");

    // The UPDATE having landed, the user must also not see a permission-denied
    // error. Safe now — the poll proved the statement resolved, so this is not
    // racing it. Pre-fix this rendered "ERROR: permission denied for table".
    await expect(
      page.getByText(/permission denied/i),
      "the DML must not be denied — it should route to the admin data source, " +
        "not the read-only one (which can only SELECT the admin-owned table)",
    ).toHaveCount(0, { timeout: 5000 });
  });
});
