// SQL Editor — query result rendering and interaction.
//
// Covers what happens after the user clicks Run: how the result panel
// renders rows, multi-statement tabs, errors, NULL placeholders,
// sort, the per-cell Detail panel, and the row-selection visual
// affordance. Mix of CUJ coverage (happy paths that should pass) and
// a bug-lock (BYT-9478 row-selection highlight) marked inline.

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

// Each test re-enters the connected DB URL via gotoWithDb. The SQL
// editor opens a fresh worksheet tab on that route, isolating each
// test from the previous test's editor content + result state.
// Reusing the same browser context (no new context per test) keeps the
// run fast; the per-test navigation is what gives state isolation.
async function openFreshConnectedTab(): Promise<void> {
  const projectId = env.project.split("/").pop()!;
  await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
  await page.waitForTimeout(800);
}

test.describe("Single-row result renders one data row", () => {
  // The smallest possible happy path: a literal SELECT 1 returns a
  // single row and the result panel renders it. If this breaks,
  // almost every other result test below is also broken.

  test("SELECT 1 produces a 1-row result", async () => {
    test.setTimeout(120_000);
    await openFreshConnectedTab();

    await sqlEditor.runPreparedQuery("SELECT 1 AS n;");

    await expect(page.getByText(/^1\s+rows?$/i).first()).toBeVisible({
      timeout: 10_000,
    });
    await expect(page.locator('[data-row-index="0"]').first()).toContainText(
      "1",
      { timeout: 5000 },
    );
  });
});

test.describe("Multi-statement query renders one result tab per statement", () => {
  // Running two SELECTs separated by `;` produces a Query #1 / Query
  // #2 tab list, and clicking a tab switches the visible result to
  // that statement's rows.

  test("Query #1 and Query #2 tabs both render and switch", async () => {
    test.setTimeout(120_000);
    await openFreshConnectedTab();

    await sqlEditor.runPreparedQuery(
      "SELECT 1 AS first; SELECT 2 AS second;",
    );

    const tab1 = page.getByRole("tab", { name: /Query\s*#1/i }).first();
    const tab2 = page.getByRole("tab", { name: /Query\s*#2/i }).first();
    await expect(tab1).toBeVisible({ timeout: 10_000 });
    await expect(tab2).toBeVisible();

    // Default tab is Query #1 — its row should show the literal "1".
    await expect(page.locator('[data-row-index="0"]').first()).toContainText(
      "1",
      { timeout: 5000 },
    );

    // Switch to Query #2 — the row should now show "2".
    await tab2.click();
    await expect(page.locator('[data-row-index="0"]').first()).toContainText(
      "2",
      { timeout: 5000 },
    );
  });
});

test.describe("Bad SQL renders an inline error pane", () => {
  // When the statement fails to parse or execute, the user must see
  // an error message inline (not silently get an empty result, not
  // crash the panel).

  test("syntax error surfaces an ERROR card, no rows", async () => {
    test.setTimeout(120_000);
    await openFreshConnectedTab();

    await sqlEditor.runPreparedQuery("SEELECT 1;");

    await expect(page.getByText(/ERROR[: ]/i).first()).toBeVisible({
      timeout: 10_000,
    });
    await expect(page.locator("[data-row-index]")).toHaveCount(0);
  });
});

test.describe("NULL values render with a visible NULL marker", () => {
  // When the result contains a SQL NULL, the cell must show "NULL"
  // (or some other unambiguous placeholder) — never an empty cell
  // that the user mistakes for an empty string.

  test('SELECT NULL renders a "NULL" cell, not blank', async () => {
    test.setTimeout(120_000);
    await openFreshConnectedTab();

    await sqlEditor.runPreparedQuery("SELECT NULL AS empty_col;");

    await expect(page.getByText(/^1\s+rows?$/i).first()).toBeVisible({
      timeout: 10_000,
    });
    // The NULL marker is rendered inside the data cell at column 1.
    // We anchor by row 0 / col-index 1 (col 0 is the row-number cell).
    const cell = page.locator(
      '[data-row-index="0"] [data-col-index="1"]',
    );
    await expect(cell).toBeVisible({ timeout: 5000 });
    await expect(cell).toContainText(/NULL/i);
  });
});

test.describe("Clicking a column header sorts the result", () => {
  // Header click toggles a sort indicator and reorders the rendered
  // rows.

  test("clicking the header reorders rows by that column", async () => {
    test.setTimeout(120_000);
    await openFreshConnectedTab();

    // Initial order is descending so the sort click visibly reorders
    // the rows. UNION'd literals would be returned in (engine-defined)
    // ascending order, which would mask the sort effect entirely.
    await sqlEditor.runPreparedQuery(
      "SELECT n FROM (VALUES (1), (2), (3)) AS t(n) ORDER BY n DESC;",
    );
    await expect(page.getByText(/^3\s+rows?$/i).first()).toBeVisible({
      timeout: 10_000,
    });

    const readFirstColumn = async () =>
      page.evaluate(() =>
        Array.from(
          document.querySelectorAll(
            '[data-row-index] [data-col-index="1"]',
          ),
        )
          .map((c) => c.textContent?.trim() ?? "")
          .filter(Boolean),
      );
    const before = await readFirstColumn();
    expect(
      before,
      "initial order should be desc per ORDER BY DESC",
    ).toEqual(["3", "2", "1"]);

    // VirtualDataTable.tsx renders the header sort toggle as a
    // <span role="button"> wrapping a Lucide ArrowDownWideNarrow /
    // ArrowUpWideNarrow icon. With one column ("n"), there's exactly
    // one such icon visible — anchor on the SVG class. Clicking the
    // SVG bubbles up to the role="button" parent's onClick.
    const sortIcon = page
      .locator(
        "svg.lucide-arrow-down-wide-narrow, svg.lucide-arrow-up-wide-narrow",
      )
      .first();
    await expect(sortIcon).toBeVisible({ timeout: 5000 });

    // SingleResultView.tsx:163 cycles sort state: undefined → desc →
    // asc → desc → … (no clear-back-to-undefined). Our baseRows are
    // already desc thanks to ORDER BY DESC, so the FIRST click sets
    // sortState=desc (no visible reorder); the SECOND click sets
    // asc and the rows flip. Two clicks is the user-visible path.
    await sortIcon.click();
    await page.waitForTimeout(200);
    await sortIcon.click();
    await page.waitForTimeout(400);

    const after = await readFirstColumn();

    expect(after, "sorted ascending after second click").toEqual([
      "1",
      "2",
      "3",
    ]);
    expect(after).not.toEqual(before);
  });
});

test.describe("Cell click opens the Detail panel for that cell", () => {
  // Clicking a result cell opens a Detail side panel showing that
  // cell's full value. Useful for long strings / JSON / binary that
  // would be truncated in the inline grid.

  test("clicking a cell opens the Detail panel and shows the value", async () => {
    test.setTimeout(120_000);
    await openFreshConnectedTab();

    const known = "e2e-detail-marker";
    await sqlEditor.runPreparedQuery(`SELECT '${known}' AS marker;`);
    await expect(page.getByText(/^1\s+rows?$/i).first()).toBeVisible({
      timeout: 10_000,
    });

    const cell = page.locator(
      '[data-row-index="0"] [data-col-index="1"]',
    );
    await expect(cell).toBeVisible({ timeout: 5000 });
    await cell.click();

    // The Detail panel is mounted into the result area when
    // setDetail(...) fires. The cell value is rendered inside the
    // panel — we look for the marker text appearing twice on screen
    // (once in the cell, once in the panel).
    await expect(page.getByText(known).nth(1)).toBeVisible({
      timeout: 5000,
    });
  });
});

test.describe("Selecting a row paints visible feedback across the row", () => {
  // BYT-9478: in the result grid, clicking the row-select button
  // (the 12px bg-accent/5 hairline at the left of every row) only
  // tints the index column. The rest of the row stays transparent,
  // so the user has no visual confirmation that the row is selected.
  // Combined with the row-number column's permanent decorative stripe,
  // the "selection" is indistinguishable from baseline rendering.
  //
  // Bug evidence: .playwright-cli/qa-session-2026-05-12/r43-byt9478-row-purple/
  //   - 03-notes.txt
  //   - 02-multirow-select-row3.png

  const TARGET_ROW_INDEX = 2; // 0-based; visible row 3

  test("clicking the row-select hairline tints every cell in that row", async () => {
    test.setTimeout(120_000);
    await openFreshConnectedTab();

    await sqlEditor.runPreparedQuery(
      "SELECT emp_no, first_name, last_name FROM employee LIMIT 5;",
    );

    // Wait for the virtual row we plan to select to mount.
    const targetRow = page.locator(
      `[data-row-index="${TARGET_ROW_INDEX}"]`,
    );
    await expect(targetRow).toBeVisible({ timeout: 10_000 });

    // The selection trigger is a 12px-wide button with aria-label
    // "Select row N" rendered absolutely inside the index cell. Its
    // position (left:0 of the index cell) is what produces the
    // permanent left-edge stripe — clicking it toggles the row's
    // selected state.
    const selectButton = targetRow.getByRole("button", {
      name: `Select row ${TARGET_ROW_INDEX + 1}`,
      exact: true,
    });
    await expect(selectButton).toBeVisible({ timeout: 5000 });

    // Bug-defining contract (BYT-9478): selecting a row must paint the DATA
    // cells, not just the index cell. The fix lives on the inner TableCell
    // (`selected && "bg-accent/20!"`), one level inside the [data-col-index]
    // wrapper.
    //
    // We assert a RELATIONAL delta, not an absolute "something is painted":
    // every EVEN data row already carries a permanent `group-even` zebra
    // stripe on the [data-col-index] wrapper, so "is the cell subtree
    // non-transparent?" is already true even with the bug present. Instead
    // we count non-transparent backgrounds in the cell's subtree before vs.
    // after selecting and require the count to GROW — selection added paint
    // that wasn't there before. This is parity- and color-format-agnostic
    // (the constant zebra stripe cancels out) and fails if the `bg-accent/20`
    // handler is ever deleted (AGENTS.md K + M).
    const countPaintedInCell = (rowIdx: number, colIdx: number) =>
      page.evaluate(
        ({ rowIdx, colIdx }) => {
          const row = document.querySelector(`[data-row-index="${rowIdx}"]`);
          if (!row) return null;
          const cell = row.querySelector(`[data-col-index="${colIdx}"]`);
          if (!cell) return null;
          const TRANSPARENT = new Set(["rgba(0, 0, 0, 0)", "transparent", ""]);
          const isPainted = (node: Element) =>
            !TRANSPARENT.has(getComputedStyle(node).backgroundColor);
          let painted = isPainted(cell) ? 1 : 0;
          for (const desc of Array.from(cell.querySelectorAll("*"))) {
            if (isPainted(desc)) painted++;
          }
          return painted;
        },
        { rowIdx, colIdx },
      );

    const paintedBefore = await countPaintedInCell(TARGET_ROW_INDEX, 2);
    expect(
      paintedBefore,
      "target row's data cell must be locatable before selecting",
    ).not.toBeNull();

    await selectButton.click();
    // Move the mouse off the row so a transient `group-hover` background
    // can't be mistaken for selection paint; selection persists regardless.
    await page.mouse.move(0, 0);
    await page.waitForTimeout(300);

    const paintedAfter = await countPaintedInCell(TARGET_ROW_INDEX, 2);
    expect(
      paintedAfter!,
      `selecting the row must add a visible background to its data cell ` +
        `(non-transparent elements in the cell subtree: before=` +
        `${paintedBefore}, after=${paintedAfter}). With the bug present ` +
        `only the index cell tints, so the data cell's subtree gains no ` +
        `paint and the count is unchanged.`,
    ).toBeGreaterThan(paintedBefore!);
  });
});

test.describe("Result panel search with zero matches raises a no-result toast (BYT-9603)", () => {
  // BYT-9603 (FIXED, #20478): typing a token that matches nothing in the
  // result set used to give NO feedback — the user couldn't tell whether the
  // search ran and found nothing or simply did nothing. The fix raises an INFO
  // toast "No matching result found" (sql-editor.search-no-result) on the
  // transition into the no-matches state (SingleResultView.tsx:392).
  //
  // The toast is gated on `searchActive && indexes.length === 0 &&
  // !wasInNoResultsRef.current`, so it fires once on entering the empty state
  // — exactly the user-visible signal that was missing before the fix.

  test("typing a non-matching token surfaces the no-result toast; a matching token does not", async () => {
    test.setTimeout(120_000);
    await openFreshConnectedTab();

    await sqlEditor.runPreparedQuery("SELECT 1 AS n;");
    await expect(page.getByText(/^1\s+rows?$/i).first()).toBeVisible({
      timeout: 10_000,
    });

    // The result toolbar's AdvancedSearch input has an empty placeholder, so
    // anchor it inside the `.result-toolbar` container (SingleResultView.tsx).
    const searchInput = page.locator(".result-toolbar input").first();
    await expect(searchInput).toBeVisible({ timeout: 5000 });
    await searchInput.click();
    // pressSequentially mirrors a real user typing; the search query emit is
    // debounced (~300ms) and the no-result effect runs off the resulting
    // searchParams change.
    await searchInput.pressSequentially("zzz_no_such_value", { delay: 20 });
    await page.keyboard.press("Enter");

    // Oracle: the INFO toast appears. It auto-dismisses (~6s), so assert
    // promptly. This is the exact feedback BYT-9603 added.
    await expect(
      page.getByText("No matching result found").first(),
    ).toBeVisible({ timeout: 5000 });

    // Negative cell: clearing and typing a token that DOES match (the literal
    // "1" is in the single result row) must NOT raise the toast. Wait for the
    // prior toast to clear first so we don't read a stale one.
    await expect(page.getByText("No matching result found")).toHaveCount(0, {
      timeout: 8000,
    });
    await searchInput.click();
    await page.keyboard.press("ControlOrMeta+a");
    await page.keyboard.press("Delete");
    await searchInput.pressSequentially("1", { delay: 20 });
    await page.keyboard.press("Enter");
    await page.waitForTimeout(800);
    await expect(page.getByText("No matching result found")).toHaveCount(0);
  });
});

test.describe("Detail panel scrolls long content to the last line (BYT-9610)", () => {
  // BYT-9610 (FIXED, #20461): the cell Detail drawer used `h-full` on its body
  // wrapper, which is 100% of the Sheet popup even though SheetHeader already
  // consumed ~57px at the top. The inner `flex-1 overflow-auto` scroll region
  // therefore extended ~57px PAST the viewport bottom, so the last few lines of
  // long content rendered off-screen and could never be scrolled into view.
  //
  // The fix (DetailPanel.tsx) switched the wrapper to `flex-1 min-h-0`, so the
  // scroll region fits within the sheet and every line is reachable.

  test("the Detail scroll region fits inside the viewport for long content", async () => {
    test.setTimeout(120_000);
    await openFreshConnectedTab();

    // One row, one cell whose value is 300 newline-separated lines, with a
    // unique marker on the FIRST and LAST lines — far taller than the panel's
    // clamp, so the bottom marker sits well below the fold.
    const TOP = "e2e9610top";
    const BOTTOM = "e2e9610bottom";
    await sqlEditor.runPreparedQuery(
      `SELECT '${TOP}' || E'\\n' || string_agg('line-' || lpad(i::text, 4, '0'), E'\\n') || E'\\n${BOTTOM}' AS payload FROM generate_series(1, 300) i;`,
    );
    await expect(page.getByText(/^1\s+rows?$/i).first()).toBeVisible({
      timeout: 10_000,
    });

    const cell = page.locator('[data-row-index="0"] [data-col-index="1"]');
    await expect(cell).toBeVisible({ timeout: 5000 });
    await cell.click();

    // Confirm the Detail panel opened: the value renders a SECOND time inside the
    // panel (the top marker now appears in both the cell and the panel).
    await expect(page.getByText(TOP).nth(1)).toBeVisible({ timeout: 10_000 });

    // Bug-defining oracle (mirrors the BYT-9558 sheet-scroll lock): the LAST line
    // of the value must be reachable by scrolling the panel. The panel's copy of
    // the bottom marker is the last occurrence in the DOM (the Sheet portals to
    // the end). Pre-fix the scroll region's bottom sat below the viewport, so the
    // last lines could never be scrolled into view; post-fix the region fits and
    // the bottom marker can be brought into the viewport.
    const bottomInPanel = page.getByText(BOTTOM).last();
    await bottomInPanel.scrollIntoViewIfNeeded();
    await expect(
      bottomInPanel,
      "the last line of the Detail value must be reachable by scrolling the " +
        "panel — pre-fix the scroll region overflowed the viewport bottom and " +
        "the bottom lines were off-screen and unreachable",
    ).toBeInViewport({ timeout: 5000 });

    // Close the sheet so a sibling describe sharing the page starts clean.
    await page.keyboard.press("Escape");
  });
});
