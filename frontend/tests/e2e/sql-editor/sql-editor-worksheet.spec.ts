// SQL Editor — worksheet operations.
//
// Covers worksheet-level actions taken from the sidebar tree:
// create, duplicate, delete, save, search. Tests in this file each
// set up their own worksheet via the API and drop it in afterAll,
// never reusing demo data.

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

test.describe("Share popover copy-link button (BYT-9667)", () => {
  // BYT-9667 (FIXED, #20541): the worksheet Share popover's URL copy icon looked
  // inert — same flat grey background as the addon segments, no hover/click
  // feedback (only a bottom toast). The comment thread added two contracts:
  //   (a) the copy button must stay CLICKABLE for PRIVATE worksheets (an interim
  //       change disabled it for private — Peter rejected that), and
  //   (b) it should be disabled ONLY while the tab has unsaved changes.
  // The fix converts it to the shared ghost Button (white bg, grey on hover via
  // `enabled:hover:bg-control-bg-hover`) and gates disabled-ness purely on
  // `tabStatus !== "CLEAN"` (SharePopoverBody.tsx) — never on share visibility.
  //
  // Note: the Share TRIGGER itself requires a CLEAN tab (EditorAction.tsx
  // `allowShare`), so the only legitimate "disabled" path is a dirty tab, which
  // disables the whole Share button (the copy button is then unreachable).

  // This block needs clipboard permission, which the file's shared context
  // lacks — use a dedicated context (same pattern as sql-editor-export.spec.ts).
  let ctx9667: BrowserContext;
  let page9667: Page;
  let editor9667: SqlEditorPage;
  const TITLE = `e2e-byt9667-${Date.now()}`;
  let worksheet = "";

  test.beforeAll(async ({ browser }) => {
    // Created PRIVATE by default — exactly the regression cell.
    worksheet = (
      await env.api.createWorksheet(env.project, TITLE, env.database, "SELECT 1;")
    ).name;
    ctx9667 = await browser.newContext({
      storageState: ".auth/state.json",
      permissions: ["clipboard-read", "clipboard-write"],
    });
    page9667 = await ctx9667.newPage();
    editor9667 = new SqlEditorPage(page9667, env.baseURL);
  });

  test.afterAll(async () => {
    if (worksheet) await env.api.deleteWorksheet(worksheet);
    await ctx9667?.close();
  });

  test("copy button is enabled for a private worksheet, responds to hover, and copies the deep link", async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    const sheetUuid = worksheet.split("/").pop()!;
    await editor9667.gotoSheet(projectId, sheetUuid);
    await page9667.waitForTimeout(1500);

    // Tab must be CLEAN so the Share trigger is enabled.
    await expect(editor9667.activeTab()).toContainText(TITLE, {
      timeout: 10_000,
    });

    const shareButton = page9667.getByRole("button", {
      name: "Share",
      exact: true,
    });
    await expect(shareButton).toBeEnabled({ timeout: 10_000 });
    await shareButton.click();

    const copyBtn = page9667.locator("[data-copy-btn]");
    await expect(copyBtn).toBeVisible({ timeout: 5000 });

    // (a) Regression: the copy button stays clickable for a PRIVATE worksheet.
    await expect(
      copyBtn,
      "the copy button must remain enabled for a private worksheet",
    ).toBeEnabled();

    // (b) Visual response: hovering changes the background (white → hover grey).
    // Relational assertion (M) — don't pin an exact rgb, just require a change.
    const bgBefore = await copyBtn.evaluate(
      (el) => getComputedStyle(el).backgroundColor,
    );
    await copyBtn.hover();
    await page9667.waitForTimeout(150);
    const bgHover = await copyBtn.evaluate(
      (el) => getComputedStyle(el).backgroundColor,
    );
    expect(
      bgHover,
      `the copy button must show a hover background change (was the headline ` +
        `bug: static grey, no hover feedback). before=${bgBefore} hover=${bgHover}`,
    ).not.toBe(bgBefore);

    // (c) Clicking copies the worksheet deep link + raises the success toast.
    await copyBtn.click();
    await expect(
      page9667.getByText("The URL is copied to your clipboard").first(),
    ).toBeVisible({ timeout: 5000 });
    const clipboard = await page9667.evaluate(() =>
      navigator.clipboard.readText(),
    );
    expect(clipboard).toMatch(
      new RegExp(`/sql-editor/projects/[^/]+/sheets/${sheetUuid}`),
    );
  });

  test("the Share trigger is disabled while the tab has unsaved changes", async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    const sheetUuid = worksheet.split("/").pop()!;
    await editor9667.gotoSheet(projectId, sheetUuid);
    await page9667.waitForTimeout(1500);
    await expect(editor9667.activeTab()).toContainText(TITLE, {
      timeout: 10_000,
    });

    // Clean → Share enabled.
    const shareButton = page9667.getByRole("button", {
      name: "Share",
      exact: true,
    });
    await expect(shareButton).toBeEnabled({ timeout: 10_000 });

    // Make the tab dirty by editing the statement.
    await editor9667.codeEditor.click();
    await page9667.waitForTimeout(150);
    await page9667.keyboard.type(" -- dirty", { delay: 10 });
    await page9667.waitForTimeout(300);
    await expect(editor9667.activeTabStatus()).resolves.toBe("DIRTY");

    // The only legitimate disable condition after the fix: a dirty tab disables
    // the Share button entirely.
    await expect(
      shareButton,
      "a dirty tab must disable the Share button (the only legitimate disable)",
    ).toBeDisabled({ timeout: 5000 });
  });
});

test.describe("Multi-select delete shows the dedicated confirm dialog (BYT-9631)", () => {
  // BYT-9631 (FIXED, #20541): selecting multiple worksheets via Multi-select and
  // clicking Delete showed the WRONG dialog — the single-folder "Non-empty
  // folder" prompt ("Do you want to move all worksheets into the root folder or
  // just delete them?") with a "Delete all files" / "Move to root folder"
  // choice. For an explicit multi-selection that was confusing and offered a
  // nonsensical "move to root folder" option. The fix routes multi-delete to a
  // dedicated AlertDialog: "Delete selected items?" / "The selected worksheets
  // and folders will be permanently deleted."

  const STAMP = Date.now();
  const TITLE_A = `e2e-multidel-a-${STAMP}`;
  const TITLE_B = `e2e-multidel-b-${STAMP}`;
  const TITLE_C = `e2e-multidel-c-${STAMP}`;
  let wsA = "";
  let wsB = "";
  let wsC = "";

  test.beforeAll(async () => {
    wsA = (await env.api.createWorksheet(env.project, TITLE_A, env.database, "SELECT 1;")).name;
    wsB = (await env.api.createWorksheet(env.project, TITLE_B, env.database, "SELECT 2;")).name;
    wsC = (await env.api.createWorksheet(env.project, TITLE_C, env.database, "SELECT 3;")).name;
  });

  test.afterAll(async () => {
    // A and B may already be deleted by the test; deleteWorksheet swallows 404.
    for (const ws of [wsA, wsB, wsC]) {
      if (ws) await env.api.deleteWorksheet(ws);
    }
  });

  test("deleting two checked worksheets uses the multi-select dialog, not the non-empty-folder prompt", async () => {
    test.setTimeout(120_000);

    await sqlEditor.gotoHome();
    await page.waitForTimeout(1000);

    const tree = page.locator(".worksheet-tree");
    await expect(tree.getByText(TITLE_A).first()).toBeVisible({
      timeout: 10_000,
    });

    // Enter multi-select via the worksheet row's context menu.
    await tree.getByText(TITLE_A).first().click({ button: "right" });
    const multiSelectItem = page.getByRole("menuitem", {
      name: "Multi-select",
      exact: true,
    });
    await expect(multiSelectItem).toBeVisible({ timeout: 5000 });
    await multiSelectItem.click();
    await page.waitForTimeout(500);

    // Check A and B (a subset — leave C unchecked) via their row checkboxes.
    // Anchor on the LEAF row (the nearest treeitem ancestor of the title text) —
    // filtering treeitems by hasText would also match the enclosing "Mine"
    // folder, whose checkbox is the wrong target.
    for (const title of [TITLE_A, TITLE_B]) {
      const titleText = tree.getByText(title).first();
      await expect(titleText).toBeVisible({ timeout: 5000 });
      const row = titleText.locator(
        'xpath=ancestor-or-self::*[@role="treeitem"][1]',
      );
      await row.getByRole("checkbox").first().click();
      await page.waitForTimeout(200);
    }

    // Click the multi-select toolbar's Delete button (TrashIcon + "Delete").
    const toolbarDelete = page
      .getByRole("button", { name: "Delete", exact: true })
      .first();
    await expect(toolbarDelete).toBeEnabled({ timeout: 5000 });
    await toolbarDelete.click();

    // The dedicated multi-delete dialog — NOT the non-empty-folder prompt.
    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toBeVisible({ timeout: 5000 });
    await expect(dialog.getByText("Delete selected items?")).toBeVisible();
    await expect(
      dialog.getByText(
        "The selected worksheets and folders will be permanently deleted.",
      ),
    ).toBeVisible();
    // The wrong (pre-fix) dialog must be absent.
    await expect(page.getByText("Non-empty folder")).toHaveCount(0);
    await expect(
      page.getByRole("button", { name: "Delete all files" }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("button", { name: "Move to root folder" }),
    ).toHaveCount(0);

    // Confirm the delete via the dedicated dialog. The dialog closing on confirm
    // proves the multi-delete flow ran through the CORRECT path (the BYT-9631
    // fix), rather than the old non-empty-folder "move to root / delete all"
    // branch which this selection never should have hit.
    await dialog.getByRole("button", { name: "Delete", exact: true }).click();
    await expect(dialog).not.toBeVisible({ timeout: 5000 });
  });
});

test.describe("Duplicate", () => {
  // The duplicate handler in SheetTree.tsx (around line 1152) calls
  //   editorWorksheetStore.createWorksheet({ title, folders, database })
  // and is missing the `content` (or `statement`) field. As a result the
  // duplicated worksheet inherits the title and connection but its body
  // is empty — a silent data-loss bug visible only after clicking into
  // the new tab.
  //
  // Bug evidence: .playwright-cli/qa-session-2026-05-12/r25-r26-duplicate/
  //   - 02-after-duplicate-empty.png
  //   - 03-notes.txt

  const SOURCE_CONTENT = "SELECT 42;";
  const SOURCE_TITLE = `e2e-duplicate-${Date.now()}`;

  let sourceWorksheet = "";
  let duplicateWorksheet = "";

  test.beforeAll(async () => {
    const created = await env.api.createWorksheet(
      env.project,
      SOURCE_TITLE,
      env.database,
      SOURCE_CONTENT,
    );
    sourceWorksheet = created.name;
  });

  test.afterAll(async () => {
    if (sourceWorksheet) await env.api.deleteWorksheet(sourceWorksheet);
    if (duplicateWorksheet) await env.api.deleteWorksheet(duplicateWorksheet);
  });

  test("duplicating a worksheet preserves the SQL body", async () => {
    // R26 — Duplicate drops the SQL body. Queued as not-high-priority;
    // mark expected-fail until the SheetTree.tsx duplicate handler is
    // patched to pass `content` through. Passing here will re-flag it
    // for marker removal.
    test.fail();
    test.setTimeout(120_000);
    const sourceUuid = sourceWorksheet.split("/").pop()!;
    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoSheet(projectId, sourceUuid);
    await page.waitForTimeout(1500);

    // Sanity: the source tab matches what we created. Without this, a
    // failure later would conflate setup error with the bug.
    await expect(sqlEditor.activeTab()).toContainText(SOURCE_TITLE, {
      timeout: 10_000,
    });

    // Right-click the source row in the sidebar tree to open the
    // worksheet context menu. Anchor by the unique title text.
    const sourceRow = page
      .locator(".worksheet-tree")
      .getByText(SOURCE_TITLE)
      .first();
    await expect(sourceRow).toBeVisible({ timeout: 10_000 });
    await sourceRow.click({ button: "right" });

    const duplicateItem = page.getByRole("menuitem", {
      name: "Duplicate",
      exact: true,
    });
    await expect(duplicateItem).toBeVisible({ timeout: 5000 });
    await duplicateItem.click();

    // The AlertDialog ("Confirm to duplicate this worksheet?") opens.
    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toBeVisible({ timeout: 5000 });

    // Capture the duplicated UUID from the URL change so afterAll can
    // clean up regardless of whether the assertion below passes.
    const projectPrefix = env.project;
    const urlPromise = page.waitForURL(
      (url) => {
        const m = url.pathname.match(/\/sheets\/([0-9a-f-]+)/i);
        return !!m && m[1] !== sourceUuid;
      },
      { timeout: 10_000 },
    );

    await dialog.getByRole("button", { name: "Confirm", exact: true }).click();
    await urlPromise;

    const newUuid = page.url().match(/\/sheets\/([0-9a-f-]+)/i)?.[1];
    if (newUuid) {
      duplicateWorksheet = `${projectPrefix}/worksheets/${newUuid}`;
    }

    // Wait for Monaco on the new tab to settle before reading its value.
    await expect(sqlEditor.codeEditor).toBeVisible({ timeout: 10_000 });
    await page.waitForTimeout(1000);

    // Read the duplicated tab's content from the DOM. `window.monaco` is
    // NOT exposed on the production React bundle, so a getEditors() read
    // always yields "" — which would make this expected-fail hold pass
    // forever, even once the SheetTree duplicate handler is fixed. The DOM
    // reader walks the rendered `.view-lines`; "longest" ignores empty side
    // panes so a fixed duplicate (carrying the SQL body) is actually seen.
    const editorValue = await sqlEditor.readEditorContent({ which: "longest" });

    expect(editorValue).toBe(SOURCE_CONTENT);
  });
});

test.describe("Right-click → Delete prompts for confirmation and removes the worksheet", () => {
  // Deleting a worksheet from the sidebar must surface a confirm
  // dialog before the destructive action. After confirmation, the
  // worksheet vanishes from the sidebar tree.

  const TARGET_TITLE = `e2e-delete-${Date.now()}`;

  let createdWorksheet = "";

  test.beforeAll(async () => {
    const created = await env.api.createWorksheet(
      env.project,
      TARGET_TITLE,
      env.database,
      "SELECT 'to-delete';",
    );
    createdWorksheet = created.name;
  });

  test.afterAll(async () => {
    // If the test deleted the worksheet, the API delete here is a
    // no-op. If the test failed before deletion, this cleans up.
    if (createdWorksheet) {
      await env.api.deleteWorksheet(createdWorksheet);
    }
  });

  test("confirming the alert removes the worksheet from the sidebar", async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoHome();
    await page.waitForTimeout(800);

    // The worksheet appears in the sidebar tree under "Mine" — find
    // by its unique title.
    const row = page
      .locator(".worksheet-tree")
      .getByText(TARGET_TITLE)
      .first();
    await expect(row).toBeVisible({ timeout: 10_000 });
    await row.click({ button: "right" });

    const deleteItem = page.getByRole("menuitem", {
      name: "Delete",
      exact: true,
    });
    await expect(deleteItem).toBeVisible({ timeout: 5000 });
    await deleteItem.click();

    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toBeVisible({ timeout: 5000 });
    // The destructive confirm reads "Delete" (matches the action) —
    // not "Confirm" like the duplicate-worksheet dialog uses.
    await dialog.getByRole("button", { name: "Delete", exact: true }).click();
    await expect(dialog).not.toBeVisible({ timeout: 5000 });

    // The row must disappear from the sidebar tree.
    await expect(
      page.locator(".worksheet-tree").getByText(TARGET_TITLE),
    ).toHaveCount(0, { timeout: 10_000 });

    // Mark the worksheet as already deleted so afterAll doesn't
    // double-delete (deleteWorksheet swallows errors but log noise
    // would suggest a real problem).
    void projectId;
    createdWorksheet = "";
  });
});

test.describe("Sidebar search filters the worksheet list", () => {
  // Typing into the "Search Sheets" input filters the visible
  // worksheet rows to those whose title matches the keyword.
  // Non-matching rows must disappear; matching rows must stay.

  // Two worksheets with distinct unique tokens — the search keyword
  // matches one and excludes the other.
  const STAMP = Date.now();
  const NEEDLE_TITLE = `e2e-search-needle-${STAMP}`;
  const HAYSTACK_TITLE = `e2e-search-other-${STAMP}`;
  let needleWorksheet = "";
  let haystackWorksheet = "";

  test.beforeAll(async () => {
    needleWorksheet = (
      await env.api.createWorksheet(
        env.project,
        NEEDLE_TITLE,
        env.database,
        "SELECT 1;",
      )
    ).name;
    haystackWorksheet = (
      await env.api.createWorksheet(
        env.project,
        HAYSTACK_TITLE,
        env.database,
        "SELECT 2;",
      )
    ).name;
  });

  test.afterAll(async () => {
    if (needleWorksheet) await env.api.deleteWorksheet(needleWorksheet);
    if (haystackWorksheet) await env.api.deleteWorksheet(haystackWorksheet);
  });

  test('typing "needle" hides the non-matching worksheet', async () => {
    test.setTimeout(120_000);

    await sqlEditor.gotoHome();
    await page.waitForTimeout(800);

    // Both worksheets visible at baseline.
    await expect(
      page.locator(".worksheet-tree").getByText(NEEDLE_TITLE),
    ).toBeVisible({ timeout: 10_000 });
    await expect(
      page.locator(".worksheet-tree").getByText(HAYSTACK_TITLE),
    ).toBeVisible();

    // The Search Sheets input is a textbox in the worksheet pane.
    const search = page.getByPlaceholder("Search Sheets").first();
    await expect(search).toBeVisible({ timeout: 5000 });
    await search.fill("needle");
    // Search is debounced (DEBOUNCE_SEARCH_DELAY constant in the
    // worksheet store). Wait for the filter to settle.
    await page.waitForTimeout(800);

    // Needle stays, haystack disappears.
    await expect(
      page.locator(".worksheet-tree").getByText(NEEDLE_TITLE),
    ).toBeVisible();
    await expect(
      page.locator(".worksheet-tree").getByText(HAYSTACK_TITLE),
    ).toHaveCount(0);

    // Reset for any sibling describe sharing the page.
    await search.fill("");
    await page.waitForTimeout(400);
  });
});

test.describe("Mine / Shared / Draft folders collapse and re-expand", () => {
  // Each top-level folder in the worksheet tree (Mine, Shared, Draft)
  // is independently collapsible. Clicking a folder header toggles
  // its expanded state — its children disappear, and clicking again
  // brings them back.

  const STAMP = Date.now();
  const TITLE = `e2e-collapse-${STAMP}`;
  let createdWorksheet = "";

  test.beforeAll(async () => {
    // Create one worksheet under Mine so we have a child whose
    // visibility we can probe.
    createdWorksheet = (
      await env.api.createWorksheet(
        env.project,
        TITLE,
        env.database,
        "SELECT 1;",
      )
    ).name;
  });

  test.afterAll(async () => {
    if (createdWorksheet) await env.api.deleteWorksheet(createdWorksheet);
  });

  test("clicking Mine hides its children, clicking again restores them", async () => {
    test.setTimeout(120_000);

    await sqlEditor.gotoHome();
    await page.waitForTimeout(800);

    const child = page.locator(".worksheet-tree").getByText(TITLE).first();
    await expect(child).toBeVisible({ timeout: 10_000 });

    const mineFolder = page
      .locator(".worksheet-tree")
      .getByRole("treeitem")
      .filter({ hasText: /^Mine$/ })
      .first();
    await expect(mineFolder).toBeVisible({ timeout: 5000 });

    // First click collapses Mine — child should disappear from the
    // rendered tree.
    await mineFolder.click();
    await expect(child).toHaveCount(0, { timeout: 5000 });

    // Second click re-expands.
    await mineFolder.click();
    await expect(child).toBeVisible({ timeout: 5000 });
  });
});
