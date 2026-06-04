// SQL Editor — query history sidebar.
//
// Covers the History gutter pane's contract: the visible history list
// must reflect persisted history, and running a new query must NOT
// nuke the rendered list (the list either keeps the prior entries or
// updates to include the new one).

import { test, expect, type BrowserContext, type Page } from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import type { BytebaseApiClient } from "../framework/api-client";
import { SqlEditorPage } from "./sql-editor.page";

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

test.describe("Running a query preserves the History sidebar list", () => {
  // BYT-9495: when the user has the History gutter pane open and runs a
  // query, the history list immediately empties to the "No history
  // found" placeholder, even though the underlying data is intact (a
  // page reload restores the list). Caused by the post-run query
  // history fetch racing with the panel's own fetch and clobbering its
  // local list state.
  //
  // Bug evidence: .playwright-cli/qa-session-2026-05-12/r41-byt9495-history-nuked/
  //   - 01-history-tab-before-run.png  (list populated)
  //   - 02-history-tab-after-run.png   (list shows "No history found")
  //   - 03-history-tab-after-reload-restored.png (data is intact)

  test("history list keeps showing entries after a query runs", async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);

    // Run an initial query so there's at least one history entry to
    // observe. Disposable test servers start with an empty history list,
    // so we cannot rely on demo-seeded entries.
    await sqlEditor.runQuery("SELECT 1;");
    await page.waitForTimeout(500);

    // Open the History gutter pane.
    await sqlEditor.gutterHistoryTab.click();
    // Wait for the panel to fetch + render the entry we just created.
    // HistoryPane.tsx renders each entry as a [data-history-row] div.
    const historyRows = page.locator("[data-history-row]");
    await expect(historyRows.first()).toBeVisible({ timeout: 10_000 });
    const initialCount = await historyRows.count();
    expect(
      initialCount,
      "history pane should show at least one entry after the first run",
    ).toBeGreaterThanOrEqual(1);

    // The bug repro: with the History pane open, run another query.
    // The pane's local state should incorporate the new entry — not
    // collapse to the empty-state placeholder.
    await sqlEditor.runQuery("SELECT 2;");
    await page.waitForTimeout(1500);

    // Bug-defining assertion: the empty-state placeholder must NOT be
    // showing. With the bug present, the placeholder text appears even
    // though the second query's history entry was persisted server-side.
    await expect(
      page.getByText("No history found", { exact: true }),
      "history pane must not collapse to empty state after running a query",
    ).toHaveCount(0);

    // Stronger assertion: the rendered list must contain at least the
    // entries we have a right to expect (≥ 1). The fix may dedupe or
    // reorder entries; we don't pin the exact count.
    const finalCount = await historyRows.count();
    expect(
      finalCount,
      "history pane must keep rendering entries after running a query",
    ).toBeGreaterThanOrEqual(1);
  });
});
