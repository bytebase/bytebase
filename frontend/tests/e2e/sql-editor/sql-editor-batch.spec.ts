// SQL Editor — multi-DB / batch query behaviors (subset of Batch 11).
//
// Covers (this iteration):
//   - C4 switch-DB no-remount — switching the active tab's connection
//     via the breadcrumb preserves Monaco's editor content; the tab is
//     re-pointed, not re-created.
//   - C8 multi-member DatabaseGroup renders in the connection panel
//     under the "Database Group" tab.
//   - C9 single-member DatabaseGroup also renders (groups with one
//     match are not collapsed into the regular Databases tab).
//   - D4 batch query result tabs — one tab per group member after Run.
//   - D10 batch export — "Batch export" button opens a multi-DB export
//     drawer.
//   - C15 FlatTableList — once a database exceeds the 1000-table
//     threshold, SchemaPane switches to a flat list of `.bb-flat-table-row`
//     rows (search-only, no tree).
//
// Deferred (need fixture/harness work):
//   - D16 NoSQL MongoDB JSON view — blocked on the engine matrix
//     (Batch 10) work; the disposable server only ships PG.

import { test, expect, type BrowserContext, type Page } from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import type { BytebaseApiClient } from "../framework/api-client";
import { getInstancePgPort, execSql } from "../framework/psql";
import { SqlEditorPage } from "./sql-editor.page";

test.setTimeout(120_000);

let env: TestEnv & { api: BytebaseApiClient };
let sharedContext: BrowserContext;
let page: Page;
let sqlEditor: SqlEditorPage;

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
  sqlEditor = new SqlEditorPage(page, env.baseURL);
});

test.afterAll(async () => {
  await sharedContext?.close();
});

test.describe("Switching DB connection preserves editor content (C4)", () => {
  // The disposable server provisions two HR sample databases (hr_prod
  // on prod-sample-instance, hr_test on test-sample-instance). We use
  // whichever DB env.database resolved to as the source and pick the
  // other as the target — the contract is engine-/instance-agnostic.

  test("typing into Monaco then switching DBs keeps the SQL body intact", async () => {
    const sourceShort = env.databaseId;
    const targetShort = sourceShort === "hr_prod" ? "hr_test" : "hr_prod";

    const target = await env.api.findDatabaseByShortName(targetShort);
    test.skip(
      !target,
      `target DB ${targetShort} not present — needs both hr_prod and hr_test`,
    );

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(800);

    // Type a unique marker statement. The unique suffix lets us anchor
    // a getByText locator without false positives from other tests'
    // residue in the same browser context.
    const STAMP = `e2e_switch_${Date.now()}`;
    const MARKER = `SELECT 1 AS ${STAMP};`;
    await sqlEditor.setEditorContent(MARKER);
    await page.waitForTimeout(300);

    // Sanity: the marker is in Monaco. readEditorContent walks the DOM
    // `.view-line`s ("longest" pane); `window.monaco` is not exposed on
    // the React bundle. Single-line content is fully in the DOM.
    expect(
      await sqlEditor.readEditorContent({ which: "longest" }),
    ).toBe(MARKER);

    // Open the breadcrumb connection panel, click the target DB row.
    const breadcrumb = page
      .locator("button")
      .filter({ hasText: sourceShort })
      .first();
    await expect(breadcrumb).toBeVisible({ timeout: 10_000 });
    await breadcrumb.click();

    const connectionPanel = page.getByRole("dialog");
    await expect(connectionPanel).toBeVisible({ timeout: 5_000 });
    const targetRow = connectionPanel
      .getByText(targetShort, { exact: true })
      .first();
    await expect(targetRow).toBeVisible({ timeout: 10_000 });
    await targetRow.click();

    // Wait for the breadcrumb to reflect the new connection — the
    // signal that the connection switch actually applied to the tab.
    await expect(
      page.locator("button").filter({ hasText: targetShort }).first(),
    ).toBeVisible({ timeout: 10_000 });
    await page.waitForTimeout(500);

    // C4: editor content survives the switch (tab was re-pointed,
    // not re-mounted).
    expect(
      await sqlEditor.readEditorContent({ which: "longest" }),
      "switching the active tab's DB connection must NOT remount Monaco — content stays",
    ).toBe(MARKER);
  });
});

// DatabaseGroup tests (C8/C9). Groups are project-scoped CEL-condition
// matchers — a group with `database_name.startsWith("hr_")` covers both
// hr_test and hr_prod (multi-member); `database_name == "hr_prod"`
// covers exactly one (single-member). Both should appear in the
// connection panel's "Database Group" tab regardless of match count.
test.describe("Connection panel Database Group tab (C8/C9)", () => {
  let multiGroup = "";
  let singleGroup = "";

  test.beforeAll(async () => {
    // C8/C9 exercise DatabaseGroups (FEATURE_DATABASE_GROUPS), which is
    // enterprise-only. The license is installed at bootstrap, so the feature is available.
    const ts = Date.now();
    const multi = await env.api.createDatabaseGroup(
      env.project,
      `e2e-multi-${ts}`,
      `E2E Multi ${ts}`,
      // hr_test + hr_prod both start with "hr_"
      'resource.database_name.startsWith("hr_")',
    );
    multiGroup = multi.name;

    const single = await env.api.createDatabaseGroup(
      env.project,
      `e2e-single-${ts}`,
      `E2E Single ${ts}`,
      `resource.database_name == "${env.databaseId}"`,
    );
    singleGroup = single.name;
  });

  test.afterAll(async () => {
    if (multiGroup) await env.api.deleteDatabaseGroup(multiGroup);
    if (singleGroup) await env.api.deleteDatabaseGroup(singleGroup);
  });

  // Helper — open the SQL editor on env.database, then click the
  // breadcrumb to open the connection panel and switch to the Database
  // Group tab. Returns the panel dialog locator scoped to the tab
  // content.
  async function openDatabaseGroupTab() {
    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(500);

    const breadcrumb = page
      .locator("button")
      .filter({ hasText: env.databaseId })
      .first();
    await expect(breadcrumb).toBeVisible({ timeout: 10_000 });
    await breadcrumb.click();

    const panel = page.getByRole("dialog");
    await expect(panel).toBeVisible({ timeout: 5_000 });
    await panel
      .getByRole("tab", { name: "Database Group", exact: true })
      .click();
    await page.waitForTimeout(500);
    return panel;
  }

  test("multi-member group renders in the Database Group tab (C8)", async () => {
    const panel = await openDatabaseGroupTab();
    // Group title was set to `E2E Multi <ts>`. Match by the stable
    // prefix so timestamp differences across runs don't matter.
    await expect(
      panel.getByText(/E2E Multi/).first(),
    ).toBeVisible({ timeout: 10_000 });
  });

  test("single-member group renders in the Database Group tab (C9)", async () => {
    const panel = await openDatabaseGroupTab();
    await expect(
      panel.getByText(/E2E Single/).first(),
    ).toBeVisible({ timeout: 10_000 });
  });

  // D4 — Running a query against a multi-member DBGroup produces one
  // result tab per member database (BatchQuerySelect renders the tab
  // strip above the result panel). The "E2E Multi" group covers both
  // hr_test and hr_prod, so we expect ≥ 2 tabs after Run.
  test("multi-member group runs as batch query and renders one tab per database (D4)", async () => {
    const panel = await openDatabaseGroupTab();
    const groupRow = panel.getByText(/E2E Multi/).first();
    await expect(groupRow).toBeVisible({ timeout: 10_000 });
    await groupRow.click();
    // Panel auto-closes after selection; give the tab store a moment to
    // attach the multi-database context.
    await page.waitForTimeout(800);

    await sqlEditor.runPreparedQuery("SELECT 1 AS n;");

    // BatchQuerySelect rendering signal: the "Batch export" button is
    // present alongside one button per queried database. The DB tabs
    // are plain <button name="hr_prod"> elements (the breadcrumb's
    // button is "Prod Prod Sample Instance hr_prod", so exact match
    // disambiguates).
    await expect(
      page.getByRole("button", { name: /Batch export/i }).first(),
    ).toBeVisible({ timeout: 15_000 });
    await expect(
      page.getByRole("button", { name: "hr_prod", exact: true }).first(),
    ).toBeVisible({ timeout: 5_000 });
    await expect(
      page.getByRole("button", { name: "hr_test", exact: true }).first(),
    ).toBeVisible({ timeout: 5_000 });
  });

  // D10 — Clicking "Batch export" after a batch query opens an export
  // drawer that lets the user pick a format and trigger a multi-DB
  // export. We assert only the drawer opens with the expected sections
  // (Databases / Statement / Format); the actual download mechanics
  // live in a separate codepath best exercised by a unit test.
  test("Batch export button opens the multi-DB export drawer (D10)", async () => {
    const panel = await openDatabaseGroupTab();
    await panel.getByText(/E2E Multi/).first().click();
    await page.waitForTimeout(800);
    await sqlEditor.runPreparedQuery("SELECT 1 AS n;");

    const batchExportBtn = page
      .getByRole("button", { name: /Batch export/i })
      .first();
    await expect(batchExportBtn).toBeVisible({ timeout: 15_000 });
    await batchExportBtn.click();

    // The drawer header carries the export action's title; the
    // standard DataExportButton drawer uses "Export Data" (i18n key
    // `export-data.export-data`).
    const drawer = page.getByRole("dialog").last();
    await expect(
      drawer.getByText(/Export\s+Data/i).first(),
    ).toBeVisible({ timeout: 10_000 });

    // The drawer's "Select database" table lists both queried DBs as
    // selectable rows — scope to the drawer so we skip hidden debug
    // <li> elements elsewhere on the page that contain the DB name.
    await expect(
      drawer.getByRole("cell", { name: "hr_prod", exact: true }).first(),
    ).toBeVisible({ timeout: 5_000 });
    await expect(
      drawer.getByRole("cell", { name: "hr_test", exact: true }).first(),
    ).toBeVisible({ timeout: 5_000 });
  });
});

// C15 — Once a database has more than FLAT_TABLE_THRESHOLD (=1000)
// tables, SchemaPane swaps the hierarchical tree for FlatTableList
// (search-only, virtualized rows with the `bb-flat-table-row` class).
// We create a dedicated schema with 1001 tables via a single psql
// DO-loop (fast — single round-trip), sync the database so Bytebase's
// catalog picks them up, then assert the flat layout renders.
test.describe("Schema pane switches to FlatTableList above 1000 tables (C15)", () => {
  const TEST_SCHEMA = "e2e_c15";
  const TABLE_COUNT = 1001;

  test.beforeAll(async () => {
    const port = await getInstancePgPort(env);
    // Drop any leftover schema from a previous run, then create the
    // schema + 1001 tables in a single round-trip via a DO block.
    execSql(env.databaseId, port, `DROP SCHEMA IF EXISTS ${TEST_SCHEMA} CASCADE`);
    execSql(env.databaseId, port, `CREATE SCHEMA ${TEST_SCHEMA}`);
    execSql(
      env.databaseId,
      port,
      `DO $$
       BEGIN
         FOR i IN 1..${TABLE_COUNT} LOOP
           EXECUTE format('CREATE TABLE ${TEST_SCHEMA}.t_%s (id INT)', i);
         END LOOP;
       END $$;`,
    );
    // Force Bytebase to rescan so its in-memory schema metadata
    // reflects the 1001 new tables.
    await env.api.syncDatabase(env.database);
  });

  test.afterAll(async () => {
    try {
      const port = await getInstancePgPort(env);
      execSql(env.databaseId, port, `DROP SCHEMA IF EXISTS ${TEST_SCHEMA} CASCADE`);
      // Re-sync so other specs running later don't see ghost tables in
      // Bytebase's cached metadata.
      await env.api.syncDatabase(env.database);
    } catch (err) {
      // Server may have been torn down by globalTeardown.
      console.warn(`C15 afterAll cleanup: ${err instanceof Error ? err.message : err}`);
    }
  });

  test("FlatTableList renders once table count exceeds 1000", async () => {
    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await sqlEditor.gutterSchemaTab.click();
    // The schema fetch is async; the flat list mounts only after
    // metadata loads with > 1000 tables.
    await expect(
      page.locator(".bb-flat-table-row").first(),
    ).toBeVisible({ timeout: 30_000 });

    // Searching by our test prefix narrows the list — verify one of
    // our generated rows is reachable (anchors that flat-list search
    // works, not just that any row exists).
    const search = page.getByPlaceholder("Search").first();
    await expect(search).toBeVisible({ timeout: 5_000 });
    await search.fill("t_500");
    await page.waitForTimeout(600); // debounced filter

    // Search results show `<schema>.<table>` per row; matching the
    // fully-qualified name avoids any false match on shorter substrings.
    await expect(
      page.locator(".bb-flat-table-row").filter({ hasText: `${TEST_SCHEMA}.t_500` }).first(),
    ).toBeVisible({ timeout: 5_000 });
  });
});
