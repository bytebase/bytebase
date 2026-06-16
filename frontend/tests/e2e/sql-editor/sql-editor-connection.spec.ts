// SQL Editor — connection lifecycle (CUJ coverage).
//
// Verifies the user-facing contracts of opening, restoring, and
// switching SQL Editor connections. These are happy-path flows: each
// test should PASS today, and a regression in routing / project
// context / connection-panel state surfaces as a failure here.
//
// Scope: CUJs C1, C2, C3, C5, C13 from PHASE-4-PLAN.md (Batch 5,
// Arc 1 — "Open + connect"). C6 schema dropdown and C7 BatchQuerySelect
// are deferred to a follow-up extension of this file.

import { test, expect, type BrowserContext, type Page } from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import type { BytebaseApiClient } from "../framework/api-client";
import { SqlEditorPage } from "./sql-editor.page";
import {
  ensureSecondaryProject,
  SECONDARY_PROJECT_TITLE,
} from "../framework/seed-test-data";

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

test.describe("Cold-open lands on the SQL Editor home view", () => {
  // C1 — visiting /sql-editor with no project / sheet / database in
  // the URL must render the editor shell (project picker visible,
  // gutter rail visible) without a hard redirect or a blank page.
  // The user opens the editor from the nav and expects to land in a
  // usable state.

  test("opens with project picker and gutter rail rendered", async () => {
    test.setTimeout(120_000);

    await sqlEditor.gotoHome();
    await page.waitForTimeout(800);

    // Project picker is rendered as a Combobox with `.project-select`.
    // (AsidePanel.tsx mounts <ProjectSelect className="...project-select"/>.)
    await expect(page.locator(".project-select").first()).toBeVisible({
      timeout: 10_000,
    });

    // Gutter rail buttons are stable user-facing labels (Worksheet /
    // Schema / History gutter tabs).
    await expect(sqlEditor.gutterWorksheetTab).toBeVisible();
    await expect(sqlEditor.gutterSchemaTab).toBeVisible();
    await expect(sqlEditor.gutterHistoryTab).toBeVisible();

    // The URL is either the home route or auto-redirected to a default
    // project (whichever the SQL editor's bootstrap picked) — not a
    // last-visited deep link the user can't recognize. Both shapes are
    // user-recoverable; what matters is that we landed inside
    // /sql-editor and not somewhere else (404, /auth, /settings).
    expect(page.url()).toMatch(/\/sql-editor(\/projects\/[^/]+)?\/?$/);
  });
});

test.describe("Opening a worksheet by UUID restores its title in a tab", () => {
  // C2 — deep-linking into /projects/<p>/sheets/<uuid> must hydrate
  // the worksheet and render its title as the active tab. This is the
  // shareable-link contract: a URL pasted from chat / the Recent menu
  // must reproducibly land the user on that worksheet.

  const SOURCE_TITLE = `e2e-cuj-c2-${Date.now()}`;
  const SOURCE_CONTENT = "SELECT 1;";

  let sourceWorksheet = "";

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
  });

  test("active tab text matches the worksheet title after deep-link", async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    const sheetUuid = sourceWorksheet.split("/").pop()!;
    await sqlEditor.gotoSheet(projectId, sheetUuid);
    await page.waitForTimeout(1500);

    await expect(sqlEditor.activeTab()).toContainText(SOURCE_TITLE, {
      timeout: 10_000,
    });
  });
});

test.describe("Instance-only URL lands without a connected database", () => {
  // C3 — visiting /projects/<p>/instances/<i> (no /databases/<db>) is
  // a partial connection: the user picked an instance but hasn't
  // chosen a database yet. The breadcrumb should show "Select a
  // database to start" rather than acting as if the user is connected.

  test('breadcrumb shows the "select a database" prompt', async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    await page.goto(
      `${env.baseURL}/sql-editor/projects/${projectId}/instances/${env.instanceId}`,
    );
    await page.waitForTimeout(1500);

    // DatabaseChooser renders `t("sql-editor.select-a-database-to-start")`
    // when the connection has an instance but no database. We anchor by
    // the visible label to keep the test resilient to translation
    // refactors that don't change the user-facing copy.
    await expect(
      page.getByText("Select a database to start", { exact: true }).first(),
    ).toBeVisible({ timeout: 10_000 });
  });
});

test.describe("Project switcher updates the editor's project context", () => {
  // C5 — choosing a different project from the picker must update the
  // editor's project context (URL changes to /projects/<newProj>) so
  // the sidebar tree, schema panel, and worksheet list all refresh
  // against the new project's databases.
  //
  // The post-demo bootstrap (#20393) only creates one sample project
  // ("Sample Project" / `project-sample`); the switcher needs a SECOND
  // project to be a meaningful test. We provision it on-demand here
  // (NOT in the global seedTestData) so that adding a second project
  // doesn't shift the SQL editor's default landing project for other
  // specs (e.g. sql-editor-worksheet.spec.ts uses gotoHome() and
  // assumes `project-sample` is current).

  test.beforeAll(async () => {
    // Created on demand and intentionally NOT deleted afterward.
    // ensureSecondaryProject is idempotent (tolerates 409) and the disposable
    // server is fresh per run. A soft delete would reserve the resource ID,
    // and a Playwright CI retry reuses the SAME server — the swallowed-409
    // recreate would then leave the switcher with no visible second project.
    // Leaving it in place is retry-stable; no later spec enumerates projects.
    await ensureSecondaryProject(env.api);
  });

  test("picking a different project switches the editor's project context", async () => {
    test.setTimeout(120_000);

    // Navigate to the discovered project explicitly (not gotoHome) so
    // the test starts in a deterministic place even after the
    // secondary project briefly becomes the default-landing target.
    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(800);

    // The Combobox primitive (combobox.tsx) renders the trigger as a
    // bare <div> with onClick — no implicit role="combobox". We click
    // the wrapper directly. Options inside the dropdown are <button>
    // elements containing the project label text.
    const projectSelectTrigger = page.locator(".project-select").first();
    await expect(projectSelectTrigger).toBeVisible({ timeout: 10_000 });

    // Pick the secondary project — different from the env default.
    await projectSelectTrigger.click();
    // Each project option's accessible name combines label + slug
    // (e.g., "Sample Project project-sample"), so `exact: true` on
    // the label alone won't match. We match the button containing the
    // label as visible text — narrower than role-only and still
    // resilient to slug renames.
    const option = page
      .getByRole("button")
      .filter({ hasText: SECONDARY_PROJECT_TITLE })
      .first();
    await expect(option).toBeVisible({ timeout: 5000 });
    await option.click();

    // The editor's project context switches to the picked project — the
    // project-select trigger now displays it. (The SQL-editor route is
    // database-centric: the URL's project segment only changes once a
    // database in the new project is opened, so an empty project like the
    // secondary one keeps the prior DB URL while the context switches.)
    await expect(projectSelectTrigger).toContainText(SECONDARY_PROJECT_TITLE, {
      timeout: 10_000,
    });
  });
});

test.describe("Connection panel exposes Databases + Database Group tabs", () => {
  // C13 — clicking the breadcrumb opens a connection-panel Sheet that
  // lets the user pick from a database tree (Databases tab) OR a
  // saved database group (Database Group tab). Both tabs must be
  // present so the user can route the active tab to either kind of
  // target.

  test("both tabs render after opening the connection panel", async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(800);

    // Click the breadcrumb to open the connection panel. The
    // breadcrumb currently shows the env / instance / db chain; we
    // anchor by the database short name (env.databaseId) which the
    // chooser renders verbatim.
    const breadcrumb = page
      .locator("button")
      .filter({ hasText: env.databaseId })
      .first();
    await expect(breadcrumb).toBeVisible({ timeout: 10_000 });
    await breadcrumb.click();

    const connectionPanel = page.getByRole("dialog");
    await expect(connectionPanel).toBeVisible({ timeout: 5000 });

    await expect(
      connectionPanel.getByRole("tab", { name: "Databases", exact: true }),
    ).toBeVisible({ timeout: 5000 });
    await expect(
      connectionPanel.getByRole("tab", {
        name: "Database Group",
        exact: true,
      }),
    ).toBeVisible({ timeout: 5000 });
  });
});

test.describe("SQL Editor entry from the database page connects the tab (BYT-9616)", () => {
  // BYT-9616 (FIXED, #20473): opening the SQL Editor from a database detail page
  // (the "SQL Editor" button, which window.open's a new tab at
  // /sql-editor/projects/<p>/instances/<i>/databases/<db>) landed on a tab that
  // was NOT actually connected — Run was grayed out for every database and admin
  // mode couldn't be entered. It only reproduced in a COLD session: the editor
  // shell hydrated databases into the React app store, but isConnectedSQLEditorTab
  // still read the (empty-on-cold-deep-link) Pinia store, so the tab was treated
  // as disconnected. A warm session masked it.
  //
  // The lock uses a FRESH context (cold store, like the incognito repro) and does
  // NOT visit /sql-editor first, then drives the database-page entry button.

  let ctx9616: BrowserContext;
  let dbPage: Page;

  test.beforeAll(async ({ browser }) => {
    // Fresh context = cold SQL-editor store, matching the incognito-only repro.
    ctx9616 = await browser.newContext({ storageState: ".auth/state.json" });
    dbPage = await ctx9616.newPage();
  });

  test.afterAll(async () => {
    await ctx9616?.close();
  });

  test("clicking 'SQL Editor' on the database page opens a connected, runnable tab", async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    // Database detail page (cold load — do NOT visit /sql-editor first).
    await dbPage.goto(
      `${env.baseURL}/projects/${projectId}/instances/${env.instanceId}/databases/${env.databaseId}`,
    );
    await dbPage.keyboard.press("Escape").catch(() => {});
    await dbPage.waitForTimeout(1500);

    // The "SQL Editor" entry is a <dd> (text = sql-editor.self) with onClick that
    // window.open's a new tab (DatabaseSQLEditorButton.tsx). Scope to the <dd>:
    // DashboardHeader also renders a "SQL Editor" *button* that, on a database
    // route, opens the SAME deep link — so a global getByText("SQL Editor")
    // .first() resolves to the header (it precedes the page content in the DOM,
    // confirmed: 2 exact matches, .first() = a SPAN inside a <button>), and the
    // test would pass even if this database-page entry were disabled or its
    // handler regressed. The <dd> locator exercises the intended entry point.
    const entry = dbPage.locator("dd").filter({ hasText: "SQL Editor" });
    await expect(entry).toBeVisible({ timeout: 10_000 });

    const [popup] = await Promise.all([
      ctx9616.waitForEvent("page"),
      entry.click(),
    ]);
    await popup.waitForLoadState("domcontentloaded");
    await popup.keyboard.press("Escape").catch(() => {});
    await popup.waitForTimeout(2000);

    // The popup must be the SQL-editor DB deep link for this database.
    expect(popup.url()).toContain(
      `/sql-editor/projects/${projectId}/instances/${env.instanceId}/databases/${env.databaseId}`,
    );

    const popupEditor = new SqlEditorPage(popup, env.baseURL);
    await expect(popupEditor.runButton).toBeVisible({ timeout: 15_000 });

    // (1) Type a query, THEN assert Run is enabled. Run is correctly disabled on
    // an EMPTY editor, so the discriminating check is "enabled once a query is
    // present" — pre-fix Run stayed grayed even with a query because the tab was
    // treated as disconnected.
    await popupEditor.setEditorContent("SELECT 1 AS n;");
    await expect(
      popupEditor.runButton,
      "Run must become enabled once a query is typed on the connected tab " +
        "(pre-fix it stayed grayed because the tab was disconnected)",
    ).toBeEnabled({ timeout: 15_000 });

    // (2) Running it returns a result — proves the tab is truly connected.
    await popupEditor.runButton.click();
    await expect(popup.getByText(/^1\s+rows?$/i).first()).toBeVisible({
      timeout: 15_000,
    });

    // (3) Admin mode (the wrench) is reachable — it was unavailable pre-fix.
    await expect(
      popupEditor.adminModeButton,
      "admin-mode wrench must be available on the database-page-opened tab",
    ).toBeVisible({ timeout: 10_000 });

    await popup.close();
  });
});
