// SQL Editor — schema browsing.
//
// Covers the Schema gutter pane (tree of schemas / tables / columns,
// search input) and the Schema Diagram canvas opened from it. These
// surfaces are how the user finds out what's in a database without
// running SQL — and how they jump from a known table to an editor
// with a SELECT skeleton.

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

test.describe("Schema Diagram fits all table cards into the viewport on mount", () => {
  // BYT-9481: the schema diagram is mounted with no auto-fit applied,
  // so most table cards land off-screen on a wide virtual canvas. The
  // user sees one or two cards on the left and assumes the diagram is
  // empty or broken — they must zoom out manually to discover the rest.
  //
  // Bug evidence: .playwright-cli/qa-session-2026-05-12/r42-byt9481-schema-diagram/
  //   - 02-card-positions.txt (DOM-measured card positions, only 1 of N visible)
  //   - 01-schema-diagram-full-view.png

  test("every table card lands inside the viewport at default zoom", async () => {
    // BYT-9481 is queued as not-high-priority. The fix is flaky on
    // main — auto-fit sometimes runs early enough to land all cards
    // in view, often it doesn't. Mark expected-fail so the suite
    // stays green; a real fix will surface as "Expected to fail, but
    // passed" → flip the marker off.
    test.fail();
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(800);

    // Open the Schema gutter pane → right-click the "public" schema row
    // → click the "Schema Diagram" menu item. This is the canonical user
    // path; opening the diagram via URL or programmatic store mutation
    // would skip the same mount lifecycle that triggers (or skips)
    // fit-to-view.
    await sqlEditor.gutterSchemaTab.click();
    await page.waitForTimeout(800);

    // Schema rows in the gutter are rendered as `.bb-schema-tree-row`
    // with `data-node-meta-type="schema"`; the public schema's row is
    // the one whose text matches "public" exactly.
    const schemaRow = page
      .locator('.bb-schema-tree-row[data-node-meta-type="schema"]')
      .filter({ hasText: "public" })
      .first();
    await expect(schemaRow).toBeVisible({ timeout: 10_000 });
    await schemaRow.click({ button: "right" });

    const diagramMenuItem = page.getByRole("menuitem", {
      name: "Schema Diagram",
      exact: true,
    });
    await expect(diagramMenuItem).toBeVisible({ timeout: 5000 });
    await diagramMenuItem.click();

    // Wait for the diagram canvas + at least one table card to mount.
    const diagramRoot = page.locator(".bb-react-schema-diagram");
    await expect(diagramRoot).toBeVisible({ timeout: 10_000 });
    const tableCards = page.locator('[data-bb-node-type="table"]');
    await expect(tableCards.first()).toBeVisible({ timeout: 15_000 });
    // Give the canvas a beat to settle (any mount-time fitView would
    // run synchronously after the first render commit).
    await page.waitForTimeout(800);

    // Bug-defining assertion: every card's bounding rect must fit
    // within the viewport. With the bug present, only the leftmost
    // card sits inside the canvas; every other card has right > innerWidth.
    const result = await page.evaluate(() => {
      const cards = Array.from(
        document.querySelectorAll('[data-bb-node-type="table"]'),
      );
      const innerWidth = window.innerWidth;
      const innerHeight = window.innerHeight;
      const offscreen: Array<{
        id: string;
        left: number;
        right: number;
        top: number;
        bottom: number;
      }> = [];
      for (const c of cards) {
        const r = c.getBoundingClientRect();
        // A card is "off-screen" if any of its edges lies outside the
        // viewport by more than 1px (tolerance for sub-pixel rounding).
        const offRight = r.right > innerWidth + 1;
        const offLeft = r.left < -1;
        const offBottom = r.bottom > innerHeight + 1;
        const offTop = r.top < -1;
        if (offRight || offLeft || offBottom || offTop) {
          offscreen.push({
            id: c.getAttribute("data-bb-node-id") ?? "?",
            left: Math.round(r.left),
            right: Math.round(r.right),
            top: Math.round(r.top),
            bottom: Math.round(r.bottom),
          });
        }
      }
      return { totalCards: cards.length, innerWidth, innerHeight, offscreen };
    });

    expect(
      result.totalCards,
      "schema diagram must render at least one table card",
    ).toBeGreaterThan(0);

    expect(
      result.offscreen,
      `every table card must fit within the ${result.innerWidth}x${result.innerHeight} ` +
        `viewport on mount — auto-fit must run. ` +
        `${result.offscreen.length} of ${result.totalCards} cards are off-screen: ` +
        JSON.stringify(result.offscreen.slice(0, 5)),
    ).toEqual([]);
  });
});

test.describe("Clicking a schema row reveals its child folders", () => {
  // Single-click on a schema row toggles expand on/off (SchemaPane.tsx:
  // toggleNode). When the user clicks "public" on a Postgres database,
  // the row's children — Tables / Views / Functions / Sequences group
  // folders — should become visible. The contract is "schemas in the
  // tree are explorable in-place" — clicking once gets the user one
  // level deeper.

  test("clicking the public schema reveals its grouping folders", async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(800);

    await sqlEditor.gutterSchemaTab.click();
    await page.waitForTimeout(800);

    const publicRow = page
      .locator('.bb-schema-tree-row[data-node-meta-type="schema"]')
      .filter({ hasText: "public" })
      .first();
    await expect(publicRow).toBeVisible({ timeout: 10_000 });

    // The "Tables" group folder is a child of public — anchor on it.
    // It exists for any Postgres schema. Probe whether the schema is
    // already expanded: if so, re-clicking would collapse it.
    const tablesFolder = page
      .locator(".bb-schema-tree-row")
      .filter({ hasText: /^Tables$/ })
      .first();
    const wasExpanded = await tablesFolder.isVisible();
    if (!wasExpanded) {
      await publicRow.click();
    }
    await expect(tablesFolder).toBeVisible({ timeout: 5000 });
  });
});

test.describe("Schema search filters the schema tree", () => {
  // Typing into the schema panel's search box filters the visible
  // tree to nodes whose name matches the query. Asserts a known
  // table appears for an exact match and a known sibling disappears.

  test('searching for "employee" hides non-matching tables', async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(800);

    await sqlEditor.gutterSchemaTab.click();
    await page.waitForTimeout(800);

    // The schema tree input has placeholder t("common.search") = "Search".
    // It's the only Search-placeholder textbox visible inside the
    // schema gutter pane (the worksheet pane uses "Search Sheets").
    const searchBox = page.getByPlaceholder("Search").first();
    await expect(searchBox).toBeVisible({ timeout: 10_000 });
    await searchBox.fill("employee");
    await page.waitForTimeout(800); // debounced filter

    const employeeRow = page
      .locator('.bb-schema-tree-row[data-node-meta-type="table"]')
      .filter({ hasText: /^employee$/ })
      .first();
    await expect(employeeRow).toBeVisible({ timeout: 5000 });

    // A sibling table that doesn't match — hr_prod has "department"
    // alongside "employee". After the filter, "department" must NOT
    // be in the visible tree.
    await expect(
      page
        .locator('.bb-schema-tree-row[data-node-meta-type="table"]')
        .filter({ hasText: /^department$/ }),
    ).toHaveCount(0);

    // Reset for any sibling describe sharing the page.
    await searchBox.fill("");
    await page.waitForTimeout(400);
  });
});

test.describe("Right-click on a schema opens a sub-panel for each metadata view", () => {
  // Right-clicking a schema row in the Schema gutter exposes a menu
  // of metadata views. Each item opens a new editor-area tab whose
  // panel is dedicated to that view (Info / Tables / Functions /
  // Procedures / External tables / Schema Diagram per
  // SchemaPane/availableActions.tsx). Each opens a different React
  // component inside Panels.tsx — we verify a unique-per-panel
  // anchor is reachable after the click.

  // Each row's `verify` function returns a locator that exists ONLY
  // when that specific panel has finished mounting. The verify is
  // intentionally loose (a known table name / function name / etc.
  // from the hr_prod demo seed) — tightening it to internal class
  // names would make the test brittle to design tweaks.
  const cases: Array<{
    menuLabel: string;
    verify: (p: Page) => ReturnType<Page["locator"]>;
  }> = [
    {
      menuLabel: "Info",
      // Info panel surfaces schema metadata fields with stable labels
      // ("Encoding", "Sync status", etc.) regardless of seed data.
      verify: (p) => p.getByText("Encoding", { exact: true }).first(),
    },
    {
      menuLabel: "Tables",
      // Tables panel lists every table in the schema; "employee" is
      // a known seed entry. It renders as a plain text cell, not a
      // link, so anchor by text + table-row context.
      verify: (p) =>
        p
          .locator("table")
          .getByText("employee", { exact: true })
          .first(),
    },
    {
      menuLabel: "Functions",
      // hr_prod doesn't seed user functions, so the panel may render
      // an empty state. The new tab's title is the most stable signal:
      // SchemaPane/actions.tsx:420 sets it to `[<db>] <action>`.
      verify: (p) =>
        p.locator("[data-tab-id]").filter({ hasText: "Functions" }).first(),
    },
    {
      menuLabel: "Procedures",
      verify: (p) =>
        p.locator("[data-tab-id]").filter({ hasText: "Procedures" }).first(),
    },
  ];

  for (const c of cases) {
    test(`opens the ${c.menuLabel} sub-panel`, async () => {
      test.setTimeout(120_000);

      const projectId = env.project.split("/").pop()!;
      await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
      await page.waitForTimeout(800);

      await sqlEditor.gutterSchemaTab.click();
      await page.waitForTimeout(800);

      const publicRow = page
        .locator('.bb-schema-tree-row[data-node-meta-type="schema"]')
        .filter({ hasText: "public" })
        .first();
      await expect(publicRow).toBeVisible({ timeout: 10_000 });
      await publicRow.click({ button: "right" });

      const menuItem = page.getByRole("menuitem", {
        name: c.menuLabel,
        exact: true,
      });
      await expect(menuItem).toBeVisible({ timeout: 5000 });
      await menuItem.click();
      await page.waitForTimeout(800);

      await expect(c.verify(page)).toBeVisible({ timeout: 10_000 });
    });
  }
});

test.describe("Right-click on a table opens its detail in a new tab", () => {
  // Right-clicking a leaf table row in the schema tree exposes a
  // "View detail" action that opens a new editor-area tab whose
  // Tables panel is focused on that specific table (header /
  // breadcrumb names the table). This is the user's path from
  // "I see a table name" to "I'm reading its column metadata".

  test('right-click employee → "View detail" opens a tab named for the table', async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(800);

    await sqlEditor.gutterSchemaTab.click();
    await page.waitForTimeout(800);

    // Type the table name into the schema search to surface "employee"
    // without having to walk the public → Tables hierarchy.
    const search = page.getByPlaceholder("Search").first();
    await expect(search).toBeVisible({ timeout: 10_000 });
    await search.fill("employee");
    await page.waitForTimeout(800);

    const employeeRow = page
      .locator('.bb-schema-tree-row[data-node-meta-type="table"]')
      .filter({ hasText: /^employee$/ })
      .first();
    await expect(employeeRow).toBeVisible({ timeout: 10_000 });
    await employeeRow.click({ button: "right" });

    const viewDetail = page.getByRole("menuitem", {
      name: "View detail",
      exact: true,
    });
    await expect(viewDetail).toBeVisible({ timeout: 5000 });
    await viewDetail.click();

    // viewDetail() opens a new editor-area tab titled `Detail for <type>
    // <name>` (SchemaPane/actions.tsx). Assert on that NEW TAB — a bare
    // getByText("employee") is already satisfied by the still-filtered
    // sidebar row (search isn't cleared until below), so the old assertion
    // passed even if the tab never opened. Anchoring on the tab strip
    // distinguishes "wired" from "actually opened the detail".
    await expect(
      page
        .locator("[data-tab-id]")
        .filter({ hasText: /Detail for table\s+employee/i })
        .first(),
    ).toBeVisible({ timeout: 10_000 });

    await search.fill("");
    await page.waitForTimeout(400);
  });
});

test.describe("Schema Diagram menuitem opens the diagram canvas", () => {
  // The "Schema Diagram" menu item on a schema row routes to the
  // DiagramPanel, which mounts the SchemaDiagram canvas with one
  // node per table. We assert at least one table card renders — the
  // mount-time auto-fit contract is covered by the BYT-9481 lock test
  // in the first describe of this file.

  test("Schema Diagram action mounts the canvas with at least one table card", async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(800);

    await sqlEditor.gutterSchemaTab.click();
    await page.waitForTimeout(800);

    const publicRow = page
      .locator('.bb-schema-tree-row[data-node-meta-type="schema"]')
      .filter({ hasText: "public" })
      .first();
    await expect(publicRow).toBeVisible({ timeout: 10_000 });
    await publicRow.click({ button: "right" });

    await page
      .getByRole("menuitem", { name: "Schema Diagram", exact: true })
      .click();

    const diagramRoot = page.locator(".bb-react-schema-diagram");
    await expect(diagramRoot).toBeVisible({ timeout: 10_000 });
    const tableCards = page.locator('[data-bb-node-type="table"]');
    await expect(tableCards.first()).toBeVisible({ timeout: 15_000 });
  });
});

test.describe("Schema tree folder labels render on a single line (BYT-9602)", () => {
  // BYT-9602 (FIXED, #20446): the SchemaPane tree had two text branches in
  // CommonNode.tsx — the search-highlight branch (HighlightLabelText) carried
  // `flex-1 truncate pl-[2px] min-w-16`, but the plain-text branch (TextNode
  // folder rows like Tables / Views / Indexes / Foreign keys) was a bare
  // `<span className="pl-[2px]">{text}</span>` with NO truncate. At the default
  // pane width and deep indent, two-word labels wrapped onto two lines inside
  // the fixed-height virtualized rows, so the list looked "crunched" with
  // overlapping text. The fix gave the plain-text branch the same
  // `flex-1 truncate ... min-w-16` contract.
  //
  // Oracle: every folder-row label is single-line — `white-space: nowrap`
  // (the `truncate` contract) AND a label box no taller than one line. The
  // CSS-contract half flips exactly with the fix (pre-fix the plain branch had
  // `white-space: normal`); the geometry half is the user-visible "doesn't
  // wrap" effect (M: a relational single-line bound, not an absolute pixel).

  test("expanded folder-row labels do not wrap to a second line", async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(800);

    await sqlEditor.gutterSchemaTab.click();
    await page.waitForTimeout(800);

    // Expand the public schema so its folder rows (Tables / Views / …) — the
    // plain-text TextNode labels the bug affected — become visible.
    const publicRow = page
      .locator('.bb-schema-tree-row[data-node-meta-type="schema"]')
      .filter({ hasText: "public" })
      .first();
    await expect(publicRow).toBeVisible({ timeout: 10_000 });
    const tablesFolder = page
      .locator(".bb-schema-tree-row")
      .filter({ hasText: /^Tables$/ })
      .first();
    if (!(await tablesFolder.isVisible().catch(() => false))) {
      await publicRow.click();
    }
    await expect(tablesFolder).toBeVisible({ timeout: 5000 });

    // Inspect the FOLDER-row labels the fix touches — the plain-text TextNode
    // rows whose label is a known grouping-folder name (Tables / Views /
    // Functions / Indexes / Foreign keys / …). We deliberately exclude
    // placeholder rows (e.g. an italic "<Empty>" state) which are a different
    // node type and legitimately not truncate-styled.
    const FOLDER_NAMES = new Set([
      "Tables",
      "Views",
      "Functions",
      "Procedures",
      "Sequences",
      "External Tables",
      "Indexes",
      "Foreign keys",
      "Triggers",
      "Partitions",
      "Columns",
      "Dependencies",
      "Packages",
    ]);
    const labels = await page.evaluate((folderNames: string[]) => {
      const names = new Set(folderNames);
      const rows = Array.from(document.querySelectorAll(".bb-schema-tree-row"));
      const out: Array<{
        text: string;
        whiteSpace: string;
        labelHeight: number;
      }> = [];
      for (const row of rows) {
        for (const span of Array.from(row.querySelectorAll("span"))) {
          const direct = Array.from(span.childNodes)
            .filter((n) => n.nodeType === Node.TEXT_NODE)
            .map((n) => n.textContent ?? "")
            .join("")
            .trim();
          if (!names.has(direct)) continue;
          const cs = getComputedStyle(span);
          out.push({
            text: direct,
            whiteSpace: cs.whiteSpace,
            labelHeight: Math.round(span.getBoundingClientRect().height),
          });
        }
      }
      return out;
    }, [...FOLDER_NAMES]);

    expect(
      labels.length,
      "expanding public must reveal at least one grouping-folder label (e.g. Tables)",
    ).toBeGreaterThan(0);

    // CSS-contract: the fix applies `truncate` (white-space: nowrap) to the
    // plain-text branch. Every folder label must be nowrap (pre-fix: normal → wrap).
    const wrapping = labels.filter((l) => l.whiteSpace === "normal");
    expect(
      wrapping,
      `every schema-tree folder label must be single-line (white-space: nowrap); ` +
        `these still wrap: ${JSON.stringify(wrapping)}`,
    ).toEqual([]);

    // User-visible: no folder label is taller than a single line (~20px;
    // 28px allows padding while still failing a 2-line ~40px wrap).
    const tooTall = labels.filter((l) => l.labelHeight > 28);
    expect(
      tooTall,
      `no schema-tree folder label may wrap to a second line; these are multi-line: ` +
        JSON.stringify(tooTall),
    ).toEqual([]);
  });
});
