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

test.describe("Sidebar resize works on first drag (BYT-9611)", () => {
  // BYT-9611 (FIXED, #20461): on the VERY FIRST load of the SQL editor, dragging
  // the vertical divider between the left aside panel and the editor did
  // nothing — the first drag was swallowed. Root cause: the sidebar Panel mixed
  // controlled + uncontrolled sizing (`defaultSize` derived from a `useState`
  // the `onResize` callback updated), so the first drag's resize re-rendered a
  // new defaultSize back into react-resizable-panels and ate the gesture. A
  // second drag then worked.
  //
  // This test asserts the FIRST drag after a fresh load resizes the panel. It
  // must run on a freshly navigated page and be the first separator drag — so
  // it uses its own goto and no earlier test in this file drags the handle.

  test("the first drag of the vertical divider resizes the aside panel", async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    // Fresh navigation: the bug only manifests on the first drag after load.
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.keyboard.press("Escape").catch(() => {});
    await page.waitForTimeout(1000);

    // The PanelGroup is horizontal with a single vertical PanelResizeHandle
    // (Separator → role="separator") between the aside panel and the editor
    // (SQLEditorHomePage.tsx:218).
    const handle = page.getByRole("separator").first();
    await expect(handle).toBeVisible({ timeout: 10_000 });

    const before = await handle.boundingBox();
    expect(before, "resize handle must have a bounding box").not.toBeNull();
    const startX = before!.x + before!.width / 2;
    const startY = before!.y + before!.height / 2;

    // FIRST drag — push the divider ~120px to the right.
    await page.mouse.move(startX, startY);
    await page.mouse.down();
    await page.mouse.move(startX + 120, startY, { steps: 10 });
    await page.mouse.up();
    await page.waitForTimeout(400);

    const after = await handle.boundingBox();
    const delta = after!.x - before!.x;
    // Aside default is 25% (~320px @1280) with a 40% max (~512px), so a 120px
    // drag applies fully. Allow clamping/sub-pixel tolerance: require the FIRST
    // drag to move the divider substantially right. Pre-fix this delta was ~0.
    expect(
      delta,
      `the first drag must resize the aside panel — the divider moved ${Math.round(
        delta,
      )}px (expected > 80). Pre-fix the first drag was swallowed (delta ~0).`,
    ).toBeGreaterThan(80);

    // Lock that resizing keeps working: a second drag back left also moves it.
    const mid = await handle.boundingBox();
    const mx = mid!.x + mid!.width / 2;
    const my = mid!.y + mid!.height / 2;
    await page.mouse.move(mx, my);
    await page.mouse.down();
    await page.mouse.move(mx - 100, my, { steps: 10 });
    await page.mouse.up();
    await page.waitForTimeout(400);
    const back = await handle.boundingBox();
    expect(
      back!.x,
      "a subsequent drag left must move the divider back",
    ).toBeLessThan(after!.x);
  });
});
