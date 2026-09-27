// SQL Editor — access-grant flow (Batch 8, L-series + H1 CUJs).
//
// Covers:
//   - L1 / L2 RequestQueryButton label switches to "Request query"
//     when project.allowJustInTimeAccess is on AND the missing
//     permissions consist only of bb.sql.select / bb.sql.info (the
//     access-grant path).
//   - L4 ACCESS pane shows the "Request access grant" CTA when JIT is enabled.
//   - L5 AccessGrantRequestDrawer renders Databases / Statement / Unmask /
//     Export / Expiration / Reason fields when opened from the pane
//     (Unmask + Export capability sections added by PRs #20491/#20487).
//   - L6 ACCESS gutter tab appears only while JIT is on.
//   - L7 masked-cell popover explains WHY a cell is masked and, only
//     while JIT is on, offers "Request unmask" → grant drawer with the
//     Unmask capability pre-checked (Export not).
//   - H1 Gutter tab count flips between 3 (no JIT) and 4 (with JIT).
//
// Deferred (out of this batch):
//   - L3 full JIT Unmask flow — request → admin approve → re-run.
//     Requires a multi-actor handoff (requestor user creates grant,
//     admin user approves, requestor re-runs and sees unmasked data).
//     The masked-column fixture it needs now lives in this file's L7
//     block — reuse it when picking L3 up.

import {
  test,
  expect,
  type BrowserContext,
  type Page,
} from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";
import { signInBrowserAs } from "../framework/sign-in";
import { execSql, getInstancePgPort, querySql } from "../framework/psql";
import { SqlEditorPage } from "./sql-editor.page";

test.setTimeout(180_000);

let env: TestEnv & { api: BytebaseApiClient };
let projectId: string;

// Single test user — projectViewer on env.project, so they can mount
// the editor but lack bb.sql.select. The JIT switch on the project
// toggles whether their Request-button reads as JIT or non-JIT.
//
// We use a fresh, file-scoped user so we don't have to coordinate with
// sql-editor-permissions.spec.ts (which manages its own e2e users).
const TEST_PASSWORD = "e2e-jit-pw-1!"; // NOSONAR — e2e fixture only
const VIEWER_USER = {
  email: "e2e-jit-viewer@example.com",
  title: "E2E JIT Viewer",
  authFile: ".auth/jit-viewer.json",
};

let viewerCtx: BrowserContext;
let viewerPage: Page;
let viewerEditor: SqlEditorPage;

// Admin-side context used to flip allowJustInTimeAccess between
// describe blocks. We keep one shared admin page for screenshots in
// case of debugging, but the JIT toggle itself goes through the API
// client — flipping via UI would require a separate `/projects/<id>`
// navigation that we can avoid.
let adminCtx: BrowserContext;
let adminPage: Page;

async function setJIT(enabled: boolean): Promise<void> {
  // Also set allowRequestRole=true: the JIT label tests below need a
  // RequestQueryButton to render at all (the component returns null
  // when allowRequestRole is false — see RequestQueryButton.tsx:131,
  // 159-161). The new sample project is born with allowRequestRole=false;
  // the JIT tests are about the LABEL flipping, not about whether the
  // button shows up.
  await env.api.updateProjectSettings(env.project, {
    allowRequestRole: true,
    allowJustInTimeAccess: enabled,
  });
  // Force-reload the viewer's page so the store picks up the new
  // project flag. The SQL Editor caches project state per-tab; flipping
  // the flag via API does not in-flight invalidate that cache.
  await viewerEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
  await viewerPage.waitForTimeout(1500);
}

test.beforeAll(async ({ browser }) => {
  test.setTimeout(180_000);
  env = loadTestEnv();
  projectId = env.project.split("/").pop()!;
  await env.api.login(env.adminEmail, env.adminPassword);

  // Provision the test user — idempotent on 409 for Playwright retries.
  try {
    await env.api.createUser(VIEWER_USER.email, TEST_PASSWORD, VIEWER_USER.title);
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    if (!msg.includes("(409)")) throw err;
  }
  await env.api.appendProjectBinding(
    env.project,
    "roles/projectViewer",
    [`user:${VIEWER_USER.email}`],
  );

  await signInBrowserAs(
    browser,
    env.baseURL,
    VIEWER_USER.email,
    TEST_PASSWORD,
    VIEWER_USER.authFile,
  );
  viewerCtx = await browser.newContext({ storageState: VIEWER_USER.authFile });
  viewerPage = await viewerCtx.newPage();
  viewerEditor = new SqlEditorPage(viewerPage, env.baseURL);

  adminCtx = await browser.newContext({ storageState: ".auth/state.json" });
  adminPage = await adminCtx.newPage();

  // Start from a known-off JIT state so the test ordering is
  // deterministic. Each describe block flips the flag explicitly.
  await env.api.updateProjectSettings(env.project, {
    allowJustInTimeAccess: false,
  });
});

test.afterAll(async () => {
  // Best-effort: turn JIT off (so other specs don't inherit the
  // flipped state) and detach contexts. We deliberately DO NOT delete
  // the test user — Playwright's worker semantics caused the
  // file-level beforeAll to fire again between describe blocks in an
  // earlier iteration, and after the first run deleting the user left the
  // row in a "deactivated" state, which then blocked
  // /v1/auth/login on the rerun ("user has been deactivated"). The
  // disposable bytebase server is torn down by globalTeardown at the
  // end of the run, so leaving the row in place is safe.
  try {
    await env.api.updateProjectSettings(env.project, {
      allowRequestRole: false,
      allowJustInTimeAccess: false,
    });
  } catch {
    /* ignore */
  }
  await viewerCtx?.close();
  await adminCtx?.close();
});

// L6 + H1 — the ACCESS gutter tab only renders when the project has
// `allowJustInTimeAccess` enabled, so the gutter count flips between
// 3 (SavedQuery/Schema/History) and 4 (+ Access).
test.describe("ACCESS gutter tab follows the project's allowJustInTimeAccess flag", () => {
  test("with JIT off, the gutter has 3 tabs and no Access tab", async () => {
    await setJIT(false);

    await expect(viewerEditor.gutterSavedQueryTab).toBeVisible({ timeout: 10_000 });
    await expect(viewerEditor.gutterSchemaTab).toBeVisible();
    await expect(viewerEditor.gutterHistoryTab).toBeVisible();
    // ACCESS tab is conditionally rendered — must be absent here.
    await expect(viewerEditor.gutterAccessTab).toHaveCount(0);
  });

  test("with JIT on, the gutter exposes an ACCESS tab in addition to the 3 defaults", async () => {
    await setJIT(true);

    await expect(viewerEditor.gutterSavedQueryTab).toBeVisible({ timeout: 10_000 });
    await expect(viewerEditor.gutterSchemaTab).toBeVisible();
    await expect(viewerEditor.gutterHistoryTab).toBeVisible();
    await expect(viewerEditor.gutterAccessTab).toBeVisible();
  });
});

// L1 / L2 — RequestQueryButton's `useJIT` is true when
// project.allowJustInTimeAccess is on AND the missing permissions
// consist only of bb.sql.select / bb.sql.info. The visible label
// switches from "Request role" to "Request query" accordingly.
test.describe("RequestQueryButton label tracks the project's JIT flag", () => {
  test("with access grants off, running a forbidden SELECT shows the 'Request role' CTA", async () => {
    await setJIT(false);
    await viewerEditor.runPreparedQuery("SELECT 1;");
    await viewerPage.waitForTimeout(800);

    await expect(viewerEditor.requestRoleButton.first()).toBeVisible({
      timeout: 10_000,
    });
    await expect(viewerEditor.requestQueryButton).toHaveCount(0);
  });

  test("with access grants on, running a forbidden SELECT shows the 'Request query' CTA", async () => {
    await setJIT(true);
    await viewerEditor.runPreparedQuery("SELECT 1;");
    await viewerPage.waitForTimeout(800);

    await expect(viewerEditor.requestQueryButton.first()).toBeVisible({
      timeout: 10_000,
    });
    await expect(viewerEditor.requestRoleButton).toHaveCount(0);
  });
});

// L4 + L5 — the ACCESS pane (visible only when JIT is on) renders a
// "Request access grant" button at its top-right. Clicking it opens the
// AccessGrantRequestDrawer with the four expected sections.
//
// REQUIRES ENTERPRISE LICENSE: the button is gated by
// `disabled={!hasJITFeature || disabled}` in AccessPane.tsx — without
// BYTEBASE_E2E_LICENSE, hasJITFeature is false and the click is a no-op
// (Playwright's `force: true` bypasses actionability checks but does NOT
// override the HTML `disabled` attribute, so onClick never fires and the
// drawer never opens). The license is installed at bootstrap, so the feature is available here.
test.describe("ACCESS pane Request-access-grant button opens the grant drawer", () => {
  test("opening the ACCESS tab and clicking Request access grant opens a drawer with Databases / Statement / Expiration / Reason", async () => {
    await setJIT(true);

    // Click the ACCESS gutter tab to mount the pane.
    await viewerEditor.gutterAccessTab.click();
    await viewerPage.waitForTimeout(500);

    // The pane's primary action — label from
    // `sql-editor.request-access-grant` ("Request access grant" in en-US).
    const requestAccessBtn = viewerEditor.requestAccessGrantButton.first();
    await expect(requestAccessBtn).toBeVisible({ timeout: 10_000 });
    // PermissionGuard renders the button disabled when the JIT feature is
    // unlicensed; the bootstrap-installed license makes it enabled here.
    await requestAccessBtn.click({ force: true });
    await viewerPage.waitForTimeout(500);

    // Drawer open — matched by the dialog's accessible name; see
    // SqlEditorPage.accessGrantDrawer for the button/title collision note.
    await expect(viewerEditor.accessGrantDrawer).toBeVisible({
      timeout: 10_000,
    });

    // L5: the labelled regions inside the drawer. Each is a headline
    // above its input — we match the label as a substring because the
    // rendered text is `<label>Databases *</label>` and similar (the
    // trailing `*` is the required-field marker, not a separate element).
    // Anchoring to the start of the line via a regex keeps the match
    // unambiguous (matches "Databases *" or "Databases" alone, not
    // "Some Databases List").
    //
    // PRs #20491/#20487 added the Unmask + Export capability sections
    // (the drawer previously had only Databases/Statement/Expiration/Reason).
    for (const label of [
      "Databases",
      "Statement",
      "Unmask",
      "Export",
      "Expiration",
      "Reason",
    ]) {
      const re = new RegExp(`^${label}(\\s*\\*)?\\s*$`);
      await expect(
        viewerEditor.accessGrantDrawer.getByText(re).first(),
        `drawer must contain a "${label}" section`,
      ).toBeVisible({ timeout: 5000 });
    }

    // The two capability checkboxes are present AND default UNCHECKED when the
    // drawer is opened from the generic ACCESS-pane CTA (no pendingCreate).
    // Asserting unchecked — not just visible — catches a regression that
    // pre-checks either capability for a plain request. (The export-blocked
    // entry point (RequestExportButton) pre-checks ONLY Export, not Unmask,
    // after BYT-9654/#20516 — covered in sql-editor-export.spec.ts.)
    const unmaskCheckbox = viewerEditor.accessGrantDrawer.getByRole("checkbox", {
      name: "See unmasked sensitive data",
    });
    const exportCheckbox = viewerEditor.accessGrantDrawer.getByRole("checkbox", {
      name: "Export the query result",
    });
    await expect(
      unmaskCheckbox,
      "drawer must expose the Unmask capability checkbox",
    ).toBeVisible({ timeout: 5000 });
    await expect(exportCheckbox).toBeVisible();
    await expect(
      unmaskCheckbox,
      "Unmask must default unchecked from the generic Request-access-grant CTA",
    ).not.toBeChecked();
    await expect(
      exportCheckbox,
      "Export must default unchecked from the generic Request-access-grant CTA",
    ).not.toBeChecked();

    // Close the drawer so any follow-up test doesn't inherit it.
    // The drawer is dismissed by clicking outside (Base UI Sheet) or
    // by an explicit Cancel — we use Escape which the Sheet honors.
    await viewerPage.keyboard.press("Escape");
    await viewerPage.waitForTimeout(300);
  });
});

// L7 — a masked result column's header renders an EyeOff trigger; opening it
// shows a popover explaining WHY the column is masked (semantic type, method),
// and — only while the project allows access grants — a "Request unmask"
// button that opens the grant drawer with the Unmask capability pre-checked
// and Export unchecked (MaskingReasonPopover.tsx passes unmask={true} only).
//
// Runs as the ADMIN: the viewer user above lacks bb.sql.select, and L7 needs
// the query to SUCCEED so masked cells render. Masking applies to admins too
// (sql-editor-export.spec.ts asserts the mask marker on-screen as admin).
// The masked column is seeded the same way as that spec's "Masking ↔ export
// security" block: register a scoped full-mask semantic type, create a
// throwaway schema via psql, and bind the column through the catalog —
// all restored in afterAll. workers=1, so overwriting the shared
// SEMANTIC_TYPES workspace setting cannot race the export spec.
test.describe("L7 — masked-cell popover offers Request unmask", () => {
  const SEMANTIC_TYPE_ID = "bb.unmask-l7-qa";
  const SCHEMA = "qa_unmask_l7";
  const TABLE = "secret";
  const COLUMN = "email_addr";
  // Unique full-mask substitution (not the default "******") so the cell
  // provably carries OUR semantic type — see the export spec for why.
  const MASK_MARKER = "QAL7MASKED";
  const STATEMENT = `SELECT id, ${COLUMN} FROM ${SCHEMA}.${TABLE} ORDER BY id;`;

  let adminEditor: SqlEditorPage;
  let l7PgPort = "";
  let originalSemanticTypes: unknown = null;

  test.beforeAll(async () => {
    adminEditor = new SqlEditorPage(adminPage, env.baseURL);
    l7PgPort = await getInstancePgPort(env);

    originalSemanticTypes =
      (await env.api.getSetting("SEMANTIC_TYPES"))?.value ?? null;
    await env.api.upsertSetting(
      "SEMANTIC_TYPES",
      {
        semanticType: {
          types: [
            {
              id: SEMANTIC_TYPE_ID,
              title: "QA L7 Unmask",
              description: "L7 masked-cell popover e2e",
              algorithm: { fullMask: { substitution: MASK_MARKER } },
            },
          ],
        },
      },
      "value.semantic_type",
    );

    execSql(env.databaseId, l7PgPort, `DROP SCHEMA IF EXISTS ${SCHEMA} CASCADE`);
    execSql(env.databaseId, l7PgPort, `CREATE SCHEMA ${SCHEMA}`);
    execSql(
      env.databaseId,
      l7PgPort,
      `CREATE TABLE ${SCHEMA}.${TABLE} (id INT PRIMARY KEY, ${COLUMN} TEXT)`,
    );
    execSql(
      env.databaseId,
      l7PgPort,
      `INSERT INTO ${SCHEMA}.${TABLE} VALUES (1, 'l7-plaintext-value')`,
    );
    await env.api.syncDatabase(env.database);
    await env.api.updateCatalog(env.database, {
      name: `${env.database}/catalog`,
      schemas: [
        {
          name: SCHEMA,
          tables: [
            {
              name: TABLE,
              columns: {
                columns: [{ name: COLUMN, semanticType: SEMANTIC_TYPE_ID }],
              },
            },
          ],
        },
      ],
    });
  });

  test.afterAll(async () => {
    try {
      execSql(
        env.databaseId,
        l7PgPort,
        `DROP SCHEMA IF EXISTS ${SCHEMA} CASCADE`,
      );
      await env.api.syncDatabase(env.database);
    } catch {
      /* best-effort */
    }
    try {
      await env.api.upsertSetting(
        "SEMANTIC_TYPES",
        originalSemanticTypes ?? { semanticType: { types: [] } },
        "value.semantic_type",
      );
    } catch {
      /* best-effort */
    }
  });

  // Set the project's JIT flag, load the editor fresh (the per-tab project
  // cache does not pick up API-side flag flips), run the statement, and
  // open the masking-reason popover. The EyeOff trigger renders in the
  // masked COLUMN's header (VirtualDataTable.tsx renders MaskingReasonPopover
  // next to the column name; masking is column-scoped), and its own click
  // handler stops propagation, so clicking it cannot toggle the adjacent
  // sort control. Click (not hover) to open: click-open is sticky, so moving
  // the pointer to the popover's button can't hover-close it on the way.
  async function openMaskedCellPopover(jit: boolean): Promise<void> {
    await env.api.updateProjectSettings(env.project, {
      allowJustInTimeAccess: jit,
    });
    await adminEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await adminPage.waitForTimeout(1500);
    await adminEditor.runPreparedQuery(STATEMENT);

    // The full mask replaces the value with OUR marker — masking is live.
    await expect(
      adminPage.getByText(MASK_MARKER).first(),
      "the seeded semantic type must mask the cell",
    ).toBeVisible({ timeout: 10_000 });
    await expect(adminPage.getByText("l7-plaintext-value")).toHaveCount(0);

    // Only the masked column's header renders the EyeOff popover trigger —
    // the statement selects exactly one masked column, so .first() is it.
    const trigger = adminPage.locator("svg.lucide-eye-off").first();
    await expect(trigger).toBeVisible();
    await trigger.click();
    await expect(
      adminPage.getByText("Why is this data masked?"),
      "the masking-reason popover must open from the masked cell",
    ).toBeVisible({ timeout: 5000 });
  }

  test("with JIT off, the popover explains the mask but offers no Request unmask", async () => {
    await openMaskedCellPopover(false);

    // The reason rows surface the seeded semantic type by title.
    await expect(adminPage.getByText("QA L7 Unmask").first()).toBeVisible();
    await expect(adminEditor.requestUnmaskButton).toHaveCount(0);

    await adminPage.keyboard.press("Escape");
    await adminPage.waitForTimeout(300);
  });

  test("with JIT on, Request unmask opens the grant drawer with Unmask pre-checked and Export not", async () => {
    await openMaskedCellPopover(true);

    await expect(adminEditor.requestUnmaskButton.first()).toBeVisible({
      timeout: 5000,
    });
    await adminEditor.requestUnmaskButton.first().click();

    const drawer = adminEditor.accessGrantDrawer;
    await expect(drawer).toBeVisible({ timeout: 10_000 });
    await expect(drawer.getByText(STATEMENT)).toBeVisible();

    // The masked-cell entry point requests exactly the Unmask capability
    // (MaskingReasonPopover passes unmask={true}; export stays opt-in —
    // the mirror of RequestExportButton's export-only pre-fill, BYT-9654).
    await expect(
      drawer.getByRole("checkbox", { name: "See unmasked sensitive data" }),
      "Unmask must be pre-checked from the masked-cell popover",
    ).toBeChecked();
    await expect(
      drawer.getByRole("checkbox", { name: "Export the query result" }),
      "Export must NOT be pre-checked from the masked-cell popover",
    ).not.toBeChecked();

    await adminPage.keyboard.press("Escape");
    await adminPage.waitForTimeout(300);
  });
});

// BYT-9801 — the access-grant drawer's database picker used to fetch a single flat page of
// 100 databases with no server-side search, so any database past that page was
// invisible AND unsearchable in the drawer (even though the SQL Editor tree
// found it via per-environment pagination + server search). The fix routes the
// picker through the shared DatabaseSelect, which re-queries the server on every
// keystroke (onSearch). This test seeds > one picker page of databases (the
// picker fetches getDefaultPagination() = 50 for the @example.com viewer), then
// proves a database past that first page is absent from the initial list but
// resolvable by typing its name.
//
// Why this locks the fix: the pre-fix picker fetched a flat 100-row page, so all
// ~60 seeded databases loaded at once and the target was present initially — this
// test would fail at the "absent initially" assertion. Against a regression that
// drops onSearch while keeping the 50-row page, the target stays absent AND never
// appears on search — failing the "found via search" assertion. Either way the
// server-side-search fix is required for this test to pass.
test.describe("access-grant drawer picker finds a database beyond the first page (BYT-9801)", () => {
  const DB_COUNT = 60; // > the picker's first page (50) with comfortable margin
  const dbShortNames = Array.from(
    { length: DB_COUNT },
    (_, i) => `jitsrch_${String(i).padStart(2, "0")}`,
  );
  let pgPort = "";
  let targetShort = "";
  let controlShort = "";

  test.beforeAll(async () => {
    test.setTimeout(300_000);
    // env.api is already logged in as admin from the file-level beforeAll.
    pgPort = await getInstancePgPort(env);

    // 1. Create > one picker page of databases via psql — idempotently.
    //    The disposable e2e server (and its Postgres) is per-run, not
    //    per-attempt: globalSetup boots it once and Playwright retries reuse
    //    it. Teardown swallows DROP failures from lingering Bytebase
    //    connections, so a prior failed attempt can leave jitsrch_* databases
    //    behind. `CREATE DATABASE` (ON_ERROR_STOP=1) would then hard-fail the
    //    retry's setup before the BYT-9801 regression is ever exercised. Create
    //    only the missing ones (mirrors the 409-swallow on createUser above).
    const existing = new Set(
      querySql(
        env.databaseId,
        pgPort,
        "SELECT datname FROM pg_database WHERE datname LIKE 'jitsrch%'",
      )
        .split("\n")
        .map((s) => s.trim())
        .filter(Boolean),
    );
    for (const name of dbShortNames) {
      if (!existing.has(name)) {
        execSql(env.databaseId, pgPort, `CREATE DATABASE ${name}`);
      }
    }

    // 2. Widen the sample instance's sync allowlist to cover the new databases,
    //    then sync. The project-scoped sample instance carries an explicit
    //    `sync_databases` allowlist and the schema syncer skips any discovered
    //    database not on it — without this the poll below can never see them.
    const { databases: listed } = await env.api.listDatabases(env.instance);
    const listedShort = (listed ?? []).map((d) => d.name.split("/").pop()!);
    await env.api.updateInstanceSyncDatabases(env.instance, [
      ...new Set([...listedShort, ...dbShortNames]),
    ]);
    await env.api.syncInstance(env.instance);
    await expect
      .poll(
        async () => {
          const { databases } = await env.api.listDatabases(env.instance);
          const names = new Set((databases ?? []).map((d) => d.name));
          return dbShortNames.every((s) =>
            names.has(`${env.instance}/databases/${s}`),
          );
        },
        { timeout: 120_000, message: "created databases should sync" },
      )
      .toBe(true);

    // 3. Transfer each into the project the viewer will open.
    const createdFull = dbShortNames.map(
      (s) => `${env.instance}/databases/${s}`,
    );
    for (const full of createdFull) {
      await env.api.transferDatabaseToProject(full, env.project);
    }

    // 4. Deterministically pick a target the picker's first page omits. The
    //    picker and this API call hit the same ListDatabases default order, so
    //    the first 50 of the API list == the picker's first (only) fetch.
    const { databases } = await env.api.listDatabases(env.project);
    const order = (databases ?? []).map((d) => d.name);
    const firstPage = new Set(order.slice(0, 50));
    const targetFull = createdFull.find((f) => !firstPage.has(f));
    if (!targetFull) {
      throw new Error("no created database sorts past the picker's first page");
    }
    targetShort = targetFull.split("/").pop()!;
    // A database KNOWN to be in the first page — used to prove the initial
    // fetch loaded before asserting the target's absence.
    controlShort = (createdFull.find((f) => firstPage.has(f)) ?? order[0])
      .split("/")
      .pop()!;

    // 5. Enable JIT and remount the viewer's editor on the project.
    await setJIT(true);
  });

  test.afterAll(async () => {
    // Best-effort: drop the seeded databases and re-sync so Bytebase forgets
    // them (keeps env.project clean for any later spec).
    for (const name of dbShortNames) {
      try {
        execSql(env.databaseId, pgPort, `DROP DATABASE IF EXISTS ${name}`);
      } catch {
        /* a lingering connection can block DROP on a disposable server */
      }
    }
    await env.api.syncInstance(env.instance).catch(() => {});
  });

  test("a database past the first page is absent initially but found via search", async () => {
    test.setTimeout(120_000);

    // Open the ACCESS pane → Request access grant → drawer.
    await viewerEditor.gutterAccessTab.click();
    await viewerPage.waitForTimeout(500);
    await viewerEditor.requestAccessGrantButton.first().click({ force: true });
    await expect(viewerEditor.accessGrantDrawer).toBeVisible({
      timeout: 10_000,
    });

    // Locate the "Databases" section (label headline + picker) and open the
    // picker's dropdown.
    const dbSection = viewerEditor.accessGrantDrawer
      // FormField renders StyleX atomic classes since #20942 — target the
      // stable data-slot attribute, not utility class names.
      .locator('div[data-slot="form-field"]')
      .filter({ has: viewerPage.getByText(/^Databases\s*\*?$/) })
      .first();
    await dbSection.locator("div.cursor-pointer").first().click();

    // Prove the first page actually loaded (a known first-page database is
    // present) before asserting the target's absence — otherwise "absent"
    // could just mean "nothing has loaded yet".
    await expect(
      dbSection.getByText(controlShort, { exact: true }).first(),
    ).toBeVisible({ timeout: 10_000 });

    // The target sorts past the picker's first (only) fetch → not in the DOM.
    await expect(
      dbSection.getByText(targetShort, { exact: true }),
    ).toHaveCount(0);

    // Type the target's name — the fix re-queries the server (onSearch).
    await dbSection.locator("input").first().fill(targetShort);

    // Server-side search surfaces the database that the first page omitted.
    await expect(
      dbSection.getByText(targetShort, { exact: true }).first(),
    ).toBeVisible({ timeout: 15_000 });

    // Select it, then close the dropdown by clicking the drawer title (outside
    // the combobox) so the option row disappears. The target text remaining
    // proves a selected CHIP was rendered — not merely the still-open row.
    await dbSection.getByText(targetShort, { exact: true }).first().click();
    await viewerEditor.accessGrantDrawerTitle.click();
    await viewerPage.waitForTimeout(300);
    await expect(
      dbSection.getByText(targetShort, { exact: true }).first(),
    ).toBeVisible();

    await viewerPage.keyboard.press("Escape");
    await viewerPage.waitForTimeout(300);
  });
});
