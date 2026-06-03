// SQL Editor — tab strip behavior.
//
// Covers the top tab strip's contract:
// - Active tab text reflects the live worksheet state (title + connection)
// - Tab strip remains scrollable when more tabs open than fit the viewport
// - Right-click on a tab opens the close-actions context menu
// - Closing a tab with unsaved changes prompts for confirmation
// - Unsaved tabs carry a `status-dirty` class that the icon binds to

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

test.describe("Active tab text tracks the live connection", () => {
  // BYT-9392 (originally surfaced as SUP-197): when the user switches
  // the database connection on a worksheet via the breadcrumb picker,
  // the breadcrumb, schema tree, and schema dropdown all update to the
  // new database — but the TAB TITLE keeps showing the old DB name.
  //
  // This is a "trust the screen" / wrong-DB-write hazard. A user typing
  // a destructive statement (UPDATE/DELETE/TRUNCATE/DDL) while
  // orienting on the tab title believes they target the previously-
  // bound DB; the query actually executes against the new connection.
  //
  // Bug evidence: .playwright-cli/qa-session-2026-05-12/r07-tab-title-stale/
  //   - 02-notes.txt
  //   - 01-tab-says-hrprod-actually-familyprod.png
  //
  // Test environment note: the QA evidence repro was hr_prod → family_prod
  // (Postgres → MySQL across instances). The disposable test server's
  // --demo seed includes only Postgres sample instances (hr_prod,
  // hr_test). This test uses the available pair hr_prod → hr_test.
  // The bug is engine-agnostic: the title binding has no engine-
  // specific code path.

  const SOURCE_DB_SHORT = "hr_prod";
  const TARGET_DB_SHORT = "hr_test";
  // Title intentionally embeds the source DB name verbatim — that is
  // the entire premise of the bug (the title is set once at create
  // time and never re-derived when the connection changes).
  const WORKSHEET_TITLE = `${SOURCE_DB_SHORT} e2e-tab-title-${Date.now()}`;

  let sourceWorksheet = "";

  test.beforeAll(async () => {
    const source = await env.api.findDatabaseByShortName(SOURCE_DB_SHORT);
    const target = await env.api.findDatabaseByShortName(TARGET_DB_SHORT);
    if (!source) {
      throw new Error(
        `Setup failed: no database named ${SOURCE_DB_SHORT} found in any instance`,
      );
    }
    if (!target) {
      throw new Error(
        `Setup failed: no database named ${TARGET_DB_SHORT} found in any instance`,
      );
    }
    // Resolve project from the source DB record so a project rename in
    // the demo seed doesn't break the test.
    const dbRecord = await env.api
      .listDatabases(source.instance)
      .then((res) => res.databases.find((d) => d.name === source.database));
    const project =
      (dbRecord as { project?: string } | undefined)?.project ?? env.project;

    const created = await env.api.createWorksheet(
      project,
      WORKSHEET_TITLE,
      source.database,
      "SELECT 1;",
    );
    sourceWorksheet = created.name;
  });

  test.afterAll(async () => {
    if (sourceWorksheet) {
      await env.api.deleteWorksheet(sourceWorksheet);
    }
  });

  test("tab title is the worksheet name and does NOT pick up the new DB name after a connection switch", async () => {
    test.setTimeout(120_000);
    const sourceUuid = sourceWorksheet.split("/").pop()!;
    const projectId = sourceWorksheet.split("/")[1];
    await sqlEditor.gotoSheet(projectId, sourceUuid);
    await page.waitForTimeout(2000);

    // Sanity: tab title reflects the worksheet's name on first open.
    // (The name happens to contain SOURCE_DB_SHORT because we composed
    // it that way at create time — that's purely incidental.)
    await expect(sqlEditor.activeTab()).toContainText(WORKSHEET_TITLE, {
      timeout: 10_000,
    });

    // Open the breadcrumb connection picker and switch to the target.
    const breadcrumb = page
      .locator("button")
      .filter({ hasText: SOURCE_DB_SHORT })
      .first();
    await expect(breadcrumb).toBeVisible({ timeout: 5000 });
    await breadcrumb.click();

    const connectionPanel = page.getByRole("dialog");
    await expect(connectionPanel).toBeVisible({ timeout: 5000 });
    const targetRow = connectionPanel
      .getByText(TARGET_DB_SHORT, { exact: true })
      .first();
    await expect(targetRow).toBeVisible({ timeout: 10_000 });
    await targetRow.click();

    // Breadcrumb tracks the live connection — wait for it to flip to
    // the target before reading the tab text.
    await expect(
      page.locator("button").filter({ hasText: TARGET_DB_SHORT }).first(),
    ).toBeVisible({ timeout: 10_000 });
    await page.waitForTimeout(500);

    // Current product behavior (R7 fix): the tab title is the
    // worksheet name and is NOT augmented with the live DB. So after
    // switching to TARGET_DB_SHORT (hr_test), the tab still shows the
    // original worksheet title and does NOT contain TARGET_DB_SHORT —
    // the DB info lives in the breadcrumb only.
    await expect(sqlEditor.activeTab()).toContainText(WORKSHEET_TITLE, {
      timeout: 5000,
    });
    await expect(sqlEditor.activeTab()).not.toContainText(TARGET_DB_SHORT);
  });
});

test.describe("Tab strip scrolls when tabs overflow the viewport", () => {
  // BYT-9457 (3.18 regression — 3.17.1 had a working scrollbar): when
  // enough tabs open to exceed the strip's width, the horizontal
  // scrollbar disappears and the leftmost tabs are clipped with no way
  // for the user to scroll back to them.
  //
  // Root cause is structural in TabList.tsx: the inner row that wraps
  // the TabItem list has `overflow-hidden flex-nowrap`. The outer
  // scroller above it has `overflow-x-auto`, but its descendant's
  // `overflow-hidden` short-circuits scrollbar rendering and clips the
  // leftmost tabs visually even when the user attempts to scroll.
  //
  // Bug evidence: .playwright-cli/qa-session-2026-05-12/f03-byt9457-tab-scroller/
  //   - 03-notes.txt
  //   - 01-narrow-900px-tabs-need-scroll.png
  //   - 02-wide-1440px-tabs-fit.png
  //
  // Why this matters: the work in clipped tabs is effectively orphaned —
  // the user can't switch to those tabs without closing the visible ones
  // first. Combined with worksheet-level data-loss bugs (e.g. duplicate
  // dropping content), unsaved work in inaccessible tabs disappears.

  // Reset viewport in afterAll — sibling describes share the page
  // and would inherit the narrow viewport otherwise.
  const DEFAULT_VIEWPORT = { width: 1280, height: 720 };
  const NARROW_VIEWPORT = { width: 900, height: 700 };
  const TARGET_TAB_COUNT = 8;
  const WORKSHEET_TITLE_PREFIX = `e2e-tabs-${Date.now()}-`;

  let createdWorksheets: string[] = [];

  test.beforeAll(async () => {
    // Pre-create the worksheets via API with a real DB connection.
    // Clicking the in-page "+" Add button creates worksheets WITHOUT a
    // database (worksheetStore.createWorksheet({}) — TabList.tsx:163),
    // which makes the editor pop the connection panel after each click
    // and starves the loop. Pre-created worksheets bypass that path
    // entirely and become tabs the moment we navigate to them.
    for (let i = 0; i < TARGET_TAB_COUNT; i++) {
      const created = await env.api.createWorksheet(
        env.project,
        `${WORKSHEET_TITLE_PREFIX}${i}`,
        env.database,
        `SELECT ${i};`,
      );
      createdWorksheets.push(created.name);
    }
  });

  test.afterAll(async () => {
    if (page) {
      await page.setViewportSize(DEFAULT_VIEWPORT).catch(() => {});
    }
    for (const name of createdWorksheets) {
      await env.api.deleteWorksheet(name);
    }
    createdWorksheets = [];
  });

  test("horizontal scrollbar is reachable with enough tabs to overflow", async () => {
    test.setTimeout(120_000);

    await page.setViewportSize(NARROW_VIEWPORT);

    // Open all the pre-created worksheets as tabs by navigating to each
    // sheet URL in turn. The SQL editor remembers open tabs per project
    // (tabStore.openTabList), so subsequent navigations append to the
    // strip rather than replacing the active tab.
    const projectId = env.project.split("/").pop()!;
    for (const name of createdWorksheets) {
      const uuid = name.split("/").pop()!;
      await sqlEditor.gotoSheet(projectId, uuid);
    }
    await page.waitForTimeout(500);

    // Sanity: every worksheet became a tab.
    const tabCount = await sqlEditor.tabCount();
    expect(
      tabCount,
      `every pre-created worksheet should appear as a tab (got ${tabCount})`,
    ).toBeGreaterThanOrEqual(TARGET_TAB_COUNT);

    // The outer scrollable container is the only direct child of
    // .bb-sql-editor-tab-list with overflow-x-auto (TabList.tsx renders
    // it as a single scrollRef wrapping the DndContext + tabs row).
    const outerScroller = page
      .locator(".bb-sql-editor-tab-list .overflow-x-auto")
      .first();
    await expect(outerScroller).toBeVisible({ timeout: 5000 });

    const dims = await outerScroller.evaluate((el) => {
      // The inner row sits two levels down: scroller > DndContext div >
      // SortableContext div > inner row. Find the first descendant with
      // [data-tab-id] and walk up to its parent — that's the row whose
      // computed overflow-x is the bug locus.
      const tab = el.querySelector("[data-tab-id]");
      const innerRow = tab?.parentElement;
      return {
        scrollWidth: el.scrollWidth,
        clientWidth: el.clientWidth,
        outerOverflowX: getComputedStyle(el).overflowX,
        innerOverflowX: innerRow ? getComputedStyle(innerRow).overflowX : null,
      };
    });

    // Sanity: the outer scroller is configured to scroll horizontally.
    expect(
      dims.outerOverflowX,
      "outer scroller must allow horizontal scroll",
    ).toMatch(/auto|scroll/);

    // Bug-defining assertion 1: the inner row that holds the tabs must
    // not clip overflow. With `overflow-x: hidden` on the inner row,
    // the outer scrollbar never appears and the leftmost tabs are
    // permanently clipped — even when the user tries to drag-scroll.
    // This is the structural fix; PR #20326 (BYT-9495's fix) and the
    // BYT-9457 fix both touch this region.
    expect(
      dims.innerOverflowX,
      `inner tab row must not clip overflow ` +
        `(got overflow-x=${JSON.stringify(dims.innerOverflowX)})`,
    ).not.toBe("hidden");

    // Bug-defining assertion 2 (consequence of #1): with the inner
    // clipping, the outer scroller never sees more content than fits
    // its viewport — scrollWidth collapses to clientWidth even with N
    // overflowing tabs visibly clipped on screen. After the fix, the
    // outer's scrollWidth should reflect the sum of tab widths and
    // exceed clientWidth.
    expect(
      dims.scrollWidth,
      `outer scroller must see overflow (got ${tabCount} tabs but ` +
        `scrollWidth=${dims.scrollWidth} === clientWidth=${dims.clientWidth} ` +
        `— inner overflow-hidden is masking the content width)`,
    ).toBeGreaterThan(dims.clientWidth);
  });
});

test.describe("Right-click on a tab opens the close-actions menu", () => {
  // Right-click on a tab in the strip opens a Base UI dropdown menu
  // with the standard set of close actions and (for worksheet tabs)
  // a Rename item. Six items total when Rename is shown.

  test("the menu lists Close / Close others / Close to the right / Close saved / Close all / Rename", async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(800);

    // Right-click the active tab (created by gotoWithDb).
    await sqlEditor.activeTab().click({ button: "right" });

    const expected = [
      "Close",
      "Close others",
      "Close to the right",
      "Close saved",
      "Close all",
      "Rename",
    ];
    for (const label of expected) {
      await expect(
        page.getByRole("menuitem", { name: label, exact: true }),
        `menu must include "${label}"`,
      ).toBeVisible({ timeout: 5000 });
    }
  });
});

test.describe("Closing a tab with unsaved changes prompts for confirmation", () => {
  // When the user clicks the X on a tab with `status: DIRTY`, the
  // editor must NOT close it silently — an AlertDialog appears and
  // the close only proceeds after the user confirms. Without the
  // confirmation, work would be lost on misclick.

  test("dirty tab close opens an alert dialog before closing", async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(800);

    // Type into Monaco to mark the tab dirty (status changes from
    // CLEAN → DIRTY on the first keystroke). setEditorContent uses
    // insertText to dispatch a real input event, so the dirty-state
    // tracker actually fires.
    await sqlEditor.setEditorContent("SELECT 1;");
    await page.waitForTimeout(300);
    expect(
      await sqlEditor.activeTabStatus(),
      "tab must be DIRTY before we attempt to close it",
    ).toBe("DIRTY");

    // The trailing icon swap: when the tab is DIRTY and not hovered,
    // the suffix renders a Circle (unsaved indicator). Hovering the
    // suffix div swaps it to the X close button (Suffix.tsx:
    // hovering ? "close" : icon). React's onMouseEnter only fires
    // when entering the suffix element itself — hovering the parent
    // tab doesn't bubble — so we hover the suffix container and then
    // click the X that appears.
    const suffix = sqlEditor.activeTab().locator(".suffix").first();
    await suffix.hover();
    const closeIcon = suffix.locator("svg.lucide-x").first();
    await expect(closeIcon).toBeVisible({ timeout: 5000 });
    await closeIcon.click();

    // The unsaved-changes alert appears; closing it via Cancel
    // leaves the tab in place (no work lost).
    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toBeVisible({ timeout: 5000 });
    // Cancel keeps the tab.
    await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
    await expect(dialog).not.toBeVisible({ timeout: 5000 });
    await expect(sqlEditor.activeTab()).toBeVisible();
  });
});

test.describe("Unsaved tabs carry the status-dirty class", () => {
  // The DIRTY status is what TabItem uses to render the dirty-state
  // affordance (the dot icon instead of the file icon). Asserting on
  // the className keeps the regression check stable even if the icon
  // swap changes shape.

  test("typing into Monaco flips the active tab to status-dirty", async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(800);

    expect(
      await sqlEditor.activeTabStatus(),
      "tab starts CLEAN before any edits",
    ).toBe("CLEAN");

    await sqlEditor.setEditorContent("SELECT 1;");
    await page.waitForTimeout(300);

    expect(
      await sqlEditor.activeTabStatus(),
      "tab must flip to DIRTY after typing into Monaco",
    ).toBe("DIRTY");
  });
});
