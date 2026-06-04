// SQL Editor — misc shell behaviors (subset of Batch 9 AI/misc CUJs).
//
// Covers:
//   - H3 ?panel=<tab> URL query restores the aside panel on first load
//     (SQLEditorRouteShell.tsx:402 reads the param and overrides the
//     localStorage-persisted tab).
//   - H4 Clicking the gutter tabs switches the visible aside panel
//     (Worksheet ↔ Schema ↔ History).
//   - H5 The SQL editor's gutter logo opens in a new tab
//     (target=_blank + rel=noopener noreferrer) — GutterBar.tsx:49.
//   - O1 The QueryContextSettingPopover trigger ("(limit N)") is hidden
//     in admin mode — EditorAction.tsx:171 wraps it in a `!isAdminMode`
//     block.
//
// Deferred from Batch 9 (need separate iteration):
//   - K1/K2 AI Assistant menu + chat mocking.
//   - O2 popover content + O4 MaxRowCountSelect preset switching.
//   - H2 last-visited-tab localStorage (harder to isolate from the
//     framework's pinned project).
//   - N1 unsaved-leave guard, N5 welcome screen, N6 language submenu,
//     N10 quickstart non-blocking.

import { test, expect, type BrowserContext, type Page } from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import type { BytebaseApiClient } from "../framework/api-client";
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

test.describe("Aside panel URL restoration (H3)", () => {
  // Locks `?panel=<tab>` deep-link restoration: landing on the SQL editor
  // with `?panel=schema` must activate the Schema gutter panel rather than
  // the persisted-default (WORKSHEET) tab. Restoration runs on the
  // `project-context-ready` event in SQLEditorRouteShell.tsx. Previously
  // skipped on a stale "fails under license" note; re-verified passing under
  // an enterprise license both in isolation and in the full suite, so it now
  // runs as a live regression lock. H4 below covers the gutter-click path.
  test("?panel=schema activates the Schema gutter tab on first load", async () => {
    const projectId = env.project.split("/").pop()!;
    await page.goto(
      `${env.baseURL}/sql-editor/projects/${projectId}/instances/${env.instanceId}/databases/${env.databaseId}?panel=schema`,
    );
    await page.keyboard.press("Escape").catch(() => {});
    await page.waitForTimeout(1500);

    // Schema gutter button is visible regardless; the assertion is that
    // its panel is the active one. SchemaPane mounts a `.bb-schema-tree`
    // (or a tree row with `data-node-meta-type="schema"`) so we anchor
    // on a known schema-pane DOM marker.
    await expect(
      page
        .locator('.bb-schema-tree-row[data-node-meta-type="schema"]')
        .first(),
    ).toBeVisible({ timeout: 10_000 });
  });
});

test.describe("Gutter tab switching (H4)", () => {
  test("clicking each gutter tab swaps the visible aside panel", async () => {
    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);

    // Start by clicking Worksheet — the worksheet tree shows the
    // "Search Sheets" textbox and Mine/Shared/Draft folders.
    await sqlEditor.gutterWorksheetTab.click();
    await expect(
      page.getByPlaceholder("Search Sheets").first(),
    ).toBeVisible({ timeout: 10_000 });

    // Switch to Schema — schema rows render.
    await sqlEditor.gutterSchemaTab.click();
    await expect(
      page
        .locator('.bb-schema-tree-row[data-node-meta-type="schema"]')
        .first(),
    ).toBeVisible({ timeout: 10_000 });
    // And the worksheet "Search Sheets" input must NOT be visible.
    await expect(page.getByPlaceholder("Search Sheets")).toHaveCount(0);

    // Switch to History — empty state placeholder OR an existing
    // [data-history-row]; both are valid (we run on a fresh server so
    // history may or may not be populated).
    await sqlEditor.gutterHistoryTab.click();
    await page.waitForTimeout(500);
    const hasRows = await page.locator("[data-history-row]").count();
    const hasEmpty = await page
      .getByText("No history found", { exact: true })
      .count();
    expect(
      hasRows + hasEmpty,
      "history pane must render either rows or its empty-state placeholder",
    ).toBeGreaterThan(0);
  });
});

test.describe("Gutter logo opens in a new tab (H5)", () => {
  test("logo anchor has target=_blank and rel=noopener noreferrer", async () => {
    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);

    // The gutter logo is the only <img alt="Bytebase"> on the page;
    // walk up to its anchor parent.
    const logoAnchor = page
      .locator("a")
      .filter({ has: page.locator('img[alt="Bytebase"]') })
      .first();
    await expect(logoAnchor).toBeVisible({ timeout: 10_000 });
    await expect(logoAnchor).toHaveAttribute("target", "_blank");
    await expect(logoAnchor).toHaveAttribute(
      "rel",
      /noopener.*noreferrer|noreferrer.*noopener/,
    );
  });
});

test.describe("QueryContextSettingPopover hidden in admin mode (O1)", () => {
  test("entering admin mode hides the '(limit N)' Run-button affordance", async () => {
    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);

    // In worksheet mode, "(limit N)" appears next to Run.
    await expect(page.getByText(/\(limit\s+\d+\)/i).first()).toBeVisible({
      timeout: 10_000,
    });

    // Enter admin mode — EditorAction.tsx replaces the Run + popover
    // group with the Exit-admin button.
    await sqlEditor.adminModeButton.click();
    await page.waitForTimeout(800);

    // No "(limit N)" text anywhere — the popover trigger isn't rendered.
    await expect(page.getByText(/\(limit\s+\d+\)/i)).toHaveCount(0);
    await expect(
      page.getByRole("button", { name: "Exit admin mode", exact: true }),
    ).toBeVisible({ timeout: 5_000 });

    // Exit so any sibling test landing on the same page starts in
    // worksheet mode.
    await page
      .getByRole("button", { name: "Exit admin mode", exact: true })
      .click();
    await page.waitForTimeout(400);
  });
});
