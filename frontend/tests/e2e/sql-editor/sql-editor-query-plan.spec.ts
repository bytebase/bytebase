// SQL Editor — query plans drawn inline in the result pane (3.23: #21443,
// #21448, #21492, #21493; reshaped by #21646, #21652, #21666).
//
// What the pane draws and what it costs follow
// docs/design/query-plan-visualizer.md §3:
//   - Explain (Ctrl/Cmd+E)          → asks for the viewer's format: ONE Query call
//   - typed EXPLAIN (FORMAT JSON)   → drawn from what came back: ONE Query call
//   - typed EXPLAIN ANALYZE         → executed the statement, so never replayed: text only
//   - a plain SELECT                → the grid, with Plan in the view switch; choosing
//                                     it explains the executed statement once and the
//                                     plan is cached on the result
// A drawn plan opens on the Diagram tab with its costliest operator selected.
// The request budget is asserted, not just the tabs: a picture must never
// re-execute a statement or plan it twice.

import { test, expect, type BrowserContext, type Page } from "@playwright/test";
import enUS from "../../../src/locales/en-US.json";
import { loadTestEnv, type TestEnv } from "../framework/env";
import type { BytebaseApiClient } from "../framework/api-client";
import { SqlEditorPage } from "./sql-editor.page";

// Labels are read from the locale so a copy edit fails here, not silently.
const SQL_EDITOR = enUS["sql-editor"];
const VIEWER = SQL_EDITOR["query-plan-viewer"];
const TAB = VIEWER.tab;
const TEXT_ONLY_TAB = enUS.common.plan;
const COPY_PLAN = VIEWER.copy.plan;
const QUERY_RPC = "/bytebase.v1.SQLService/Query";

test.setTimeout(90_000);

let env: TestEnv & { api: BytebaseApiClient };
let sharedContext: BrowserContext;
let page: Page;
let sqlEditor: SqlEditorPage;
// The body of every SQLService/Query request since the last reset — the cost
// oracle. Bodies are binary protobuf; the statement text survives as ASCII.
const queryRequests: string[] = [];

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
  page.on("request", (request) => {
    if (request.url().endsWith(QUERY_RPC)) queryRequests.push(request.postData() ?? "");
  });
  sqlEditor = new SqlEditorPage(page, env.baseURL);
});

test.afterAll(async () => {
  await sharedContext?.close();
});

// Each describe re-enters the connected DB URL, which opens a fresh saved
// query tab and isolates it from the previous describe's editor + results.
async function openFreshConnectedTab(): Promise<void> {
  const projectId = env.project.split("/").pop()!;
  await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
  await expect(sqlEditor.codeEditor.first()).toBeVisible({ timeout: 15_000 });
}

function tab(name: string) {
  return page.getByRole("tab", { name, exact: true });
}

// The active panel of the plan viewer's tab row.
function activePanel() {
  return page.getByRole("tabpanel").last();
}

function viewSwitch() {
  return page.getByRole("radiogroup", { name: SQL_EDITOR["result-view-mode"] });
}

// The Explain shortcut is a Monaco keybinding, so the modifier is Control:
// Monaco reads its CtrlCmd key from the user agent, and the suite's Desktop
// Chrome device reports Windows on every host (AGENTS.md §6).
async function explainCurrentStatement(): Promise<void> {
  await sqlEditor.codeEditor.first().click();
  await page.keyboard.press("Control+e");
}

const STATEMENT = "SELECT * FROM employee WHERE emp_no = 10001";
const NODE = /^Index Scan, employee/;

// The plan opened on the Diagram tab, its one operator selected and ranked
// first in the side panel's costliest list.
async function expectDiagramOpenedOnCostliest(): Promise<void> {
  await expect(tab(TAB.diagram)).toBeVisible({ timeout: 20_000 });
  await expect(tab(TAB.diagram)).toHaveAttribute("aria-selected", "true");
  await expect(page.getByRole("button", { name: NODE })).toHaveAttribute("aria-pressed", "true");
  await expect(page.getByText(VIEWER.summary.costliest, { exact: true })).toBeVisible();
}

test.describe("Explain draws the plan inline in one call (P1)", () => {
  test.beforeAll(async () => {
    await openFreshConnectedTab();
    await sqlEditor.setEditorContent(STATEMENT);
    queryRequests.length = 0;
    await explainCurrentStatement();
    await expect(tab(TAB.diagram)).toBeVisible({ timeout: 20_000 });
  });

  test("Diagram opens first on the costliest operator; Table and Text are offered", async () => {
    await expectDiagramOpenedOnCostliest();
    await expect(page.getByText(/^1 nodes · estimated cost/)).toBeVisible();
    for (const name of [TAB.grid, TAB.text]) {
      await expect(tab(name)).toBeVisible();
    }
    // Retired by #21652: the Query and Summary tabs.
    await expect(page.getByRole("tab", { name: /^(Query|Summary)$/ })).toHaveCount(0);
  });

  test("Table lists the node; Text shows the engine's JSON plan with Copy plan", async () => {
    await tab(TAB.grid).click();
    await expect(page.getByRole("columnheader", { name: "Node" })).toBeVisible();
    await expect(page.getByRole("cell", { name: /Index Scan employee/ })).toBeVisible();

    await tab(TAB.text).click();
    await expect(activePanel()).toContainText('"Node Type": "Index Scan"');
    await expect(activePanel().getByRole("button", { name: COPY_PLAN, exact: true })).toBeVisible();
  });

  test("the picture costs one Query call: the plan is asked for in the viewer's format", async () => {
    // #21666: before, Explain fetched a text plan and replayed it as FORMAT
    // JSON for the picture — two planner runs for one plan.
    expect(queryRequests).toHaveLength(1);
    expect(queryRequests[0]).toContain(STATEMENT);
  });
});

test.describe("A typed EXPLAIN (FORMAT JSON) is drawn from what came back (P2)", () => {
  test("the diagram renders with no replay: one Query call", async () => {
    await openFreshConnectedTab();
    await sqlEditor.setEditorContent(`EXPLAIN (FORMAT JSON) ${STATEMENT}`);
    queryRequests.length = 0;
    await sqlEditor.runButton.click();
    await expectDiagramOpenedOnCostliest();
    await tab(TAB.text).click();
    await expect(activePanel()).toContainText('"Node Type": "Index Scan"');
    // Settled: the result rendered its picture without asking the server again.
    expect(queryRequests).toHaveLength(1);
  });
});

test.describe("A typed EXPLAIN ANALYZE executed the statement and is never replayed (P3)", () => {
  test("the plan stays text-only, with no Diagram tab, no Plan option and one Query call", async () => {
    await openFreshConnectedTab();
    await sqlEditor.setEditorContent(`EXPLAIN ANALYZE ${STATEMENT}`);
    queryRequests.length = 0;
    await sqlEditor.runButton.click();
    await expect(tab(TEXT_ONLY_TAB)).toBeVisible({ timeout: 20_000 });
    await expect(activePanel()).toContainText(/actual time=/);
    // The same labelled copy control as the visualizer's Text tab.
    await expect(activePanel().getByRole("button", { name: COPY_PLAN, exact: true })).toBeVisible();
    await expect(tab(TAB.diagram)).toHaveCount(0);
    // A result that already is a plan offers no Plan view of itself.
    await expect(viewSwitch().getByRole("radio", { name: SQL_EDITOR["plan-view"] })).toHaveCount(0);
    expect(queryRequests).toHaveLength(1);
  });
});

test.describe("A plain SELECT offers its plan in the view switch (P4)", () => {
  test("Plan explains the executed statement once, is cached on the result, and is remembered on re-run", async () => {
    test.setTimeout(150_000);
    await test.step("rows render in the Table view; no plan is fetched until asked", async () => {
      await openFreshConnectedTab();
      await sqlEditor.setEditorContent(STATEMENT);
      queryRequests.length = 0;
      await sqlEditor.runButton.click();
      await expect(page.getByText(/^1\s+rows?$/i).first()).toBeVisible({ timeout: 20_000 });
      await expect(page.locator('[data-row-index="0"]').first()).toContainText("10001");
      await expect(viewSwitch().getByRole("radio", { name: SQL_EDITOR["table-view"] })).toBeChecked();
      await expect(viewSwitch().getByRole("radio", { name: SQL_EDITOR["plan-view"] })).toBeEnabled();
      await expect(tab(TAB.diagram)).toHaveCount(0);
      expect(queryRequests).toHaveLength(1);
    });

    await test.step("Plan explains the executed statement once, limit included, and marks it Estimated", async () => {
      queryRequests.length = 0;
      // The radio is visually hidden inside its segment; the label takes the click.
      await viewSwitch().getByRole("radio", { name: SQL_EDITOR["plan-view"] }).click({ force: true });
      await expectDiagramOpenedOnCostliest();
      await expect(page.getByText(SQL_EDITOR["plan-estimated"], { exact: true })).toBeVisible();
      expect(queryRequests).toHaveLength(1);
      expect(queryRequests[0]).toContain(`${STATEMENT} LIMIT 1000`);
    });

    await test.step("Table and back to Plan sends nothing: the plan is cached on the result", async () => {
      queryRequests.length = 0;
      await viewSwitch().getByRole("radio", { name: SQL_EDITOR["table-view"] }).click({ force: true });
      await expect(page.locator('[data-row-index="0"]').first()).toContainText("10001");
      await viewSwitch().getByRole("radio", { name: SQL_EDITOR["plan-view"] }).click({ force: true });
      await expect(page.getByRole("button", { name: NODE })).toBeVisible();
      expect(queryRequests).toHaveLength(0);
    });

    await test.step("a re-run in the same tab reopens on the remembered Plan view", async () => {
      // The editor drops a run within a second of the previous one
      // (QUERY_INTERVAL_LIMIT in hooks/useExecuteSQL.ts).
      await page.waitForTimeout(1_100);
      queryRequests.length = 0;
      await sqlEditor.runButton.click();
      // The run, then the fresh result's own plan.
      await expect.poll(() => queryRequests.length, { timeout: 20_000 }).toBe(2);
      await expect(viewSwitch().getByRole("radio", { name: SQL_EDITOR["plan-view"] })).toBeChecked();
      await expectDiagramOpenedOnCostliest();
    });
  });
});
