// SQL Editor — result export workflow (PRs #20487 / #20491 / #20501).
//
// The export path was rewritten: the client-side `sql-download` formatter
// library was deleted and export now goes through the backend Export RPC. A
// workspace `DATA_QUERY` policy (`disableExport`) gates the toolbar's Export
// button; when export is disabled but the project allows just-in-time access,
// a new "Request export" affordance replaces it, and an active export grant
// for the EXACT statement re-enables the real Export button.
//
// One file per sub-area (the repo convention): all export CUJs live here as
// describe blocks with describe-scoped setup, and the bug-lock is inlined
// alongside them (matching how sql-editor-result / -admin-mode keep their locks
// with the CUJs). The BYT-9656 lock deliberately does NOT use test.fail() — see
// its block for why (it asserts the current buggy state so it fails only when
// the bug's state changes, not on an unrelated setup error).
//
// Groups (CUJ ids from the QA session):
//   - Direct export (export allowed): C1 backend Export RPC, C9 password
//   - Gated + JIT on: C2 Request-export, C3/C4 drawer pre-fill (unmask+export)
//   - Gated + JIT off: C8 no affordance
//   - Multi-statement: C10 one affordance above the tabs
//   - Active export grant: C7 exact-match re-enable, C5 badge, C6 filter
//   - Masking ↔ export security: no-leak without grant, reveal with grant
//   - BUG (BYT-9656): re-request of a revoked grant drops the Export pre-fill
//
// Owns its fixtures (API/psql for setup, browser for verification) and
// restores the workspace policy + project JIT in afterAll. Requires `psql` +
// `unzip` on PATH and an enterprise license (installed at bootstrap).

import { test, expect, type BrowserContext, type Page } from "@playwright/test";
import { execFileSync } from "child_process";
import { loadTestEnv, type TestEnv } from "../framework/env";
import type { BytebaseApiClient } from "../framework/api-client";
import { execSql, getInstancePgPort } from "../framework/psql";
import { SqlEditorPage } from "./sql-editor.page";

test.setTimeout(180_000);

let env: TestEnv & { api: BytebaseApiClient };
let projectId: string;
let workspace = "";
let pgPort = "";
let sharedContext: BrowserContext;
let page: Page;
let sqlEditor: SqlEditorPage;

// Snapshot of the original workspace QueryDataPolicy + project JIT flag so the
// file-level afterAll restores the baseline other specs depend on.
let originalDisableExport = false;
let originalDisableCopyData = false;
let originalMaxRows = 0;
let originalAllowAdminDataSource = false;
let originalJIT = false;
// createActiveGrant + the masking reveal test flip allowSelfApproval; snapshot it
// so afterAll restores it (otherwise it leaks into later specs).
let originalSelfApproval = false;

async function setWorkspaceExportPolicy(disableExport: boolean): Promise<void> {
  await env.api.upsertPolicy(workspace, "data_query", {
    name: `${workspace}/policies/data_query`,
    type: "DATA_QUERY",
    resourceType: "WORKSPACE",
    queryDataPolicy: {
      disableExport,
      disableCopyData: originalDisableCopyData,
      allowAdminDataSource: originalAllowAdminDataSource,
      maximumResultRows: originalMaxRows,
    },
  });
}

// Apply the export-gate state, then hard-reload the editor so the SQL editor
// store picks up the new workspace policy + project JIT flag (both cached
// per-tab; an in-flight API change doesn't invalidate that cache).
async function applyExportState(opts: {
  disableExport: boolean;
  jit: boolean;
}): Promise<void> {
  await setWorkspaceExportPolicy(opts.disableExport);
  await env.api.updateProjectSettings(env.project, {
    allowJustInTimeAccess: opts.jit,
  });
  await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
  await page.waitForTimeout(1200);
}

// Create a grant for `query` and drive it to ACTIVE. JIT grants open an
// approval issue even with no workspace approval rules; as workspace admin
// (creator + approver) we self-approve. The issue's approval flow can 404 for
// a beat right after creation, so retry approve until the grant activates.
async function createActiveGrant(
  query: string,
  caps: { unmask: boolean; export: boolean },
): Promise<string> {
  const grant = await env.api.createAccessGrant(env.project, {
    targets: [env.database],
    query,
    reason: "e2e export grant",
    unmask: caps.unmask,
    export: caps.export,
    ttlSeconds: 3600,
  });
  if (grant.status === "ACTIVE") return grant.name;
  await env.api.updateProjectSettings(env.project, { allowSelfApproval: true });
  await expect
    .poll(
      async () => {
        try {
          if (grant.issue) await env.api.approveIssue(grant.issue);
        } catch {
          /* approval flow not ready yet */
        }
        // Filter server-side by THIS grant's name (it keeps its name through
        // PENDING→ACTIVE) so the result is exactly our grant — independent of how
        // many other grants exist (a client-side scan of a status==ACTIVE page
        // could miss ours past pageSize, and matching by query/caps could pick a
        // stale grant from a prior CI retry against the same server).
        const r = await env.api.searchMyAccessGrants(
          env.project,
          `name == "${grant.name}"`,
        );
        return r.accessGrants.some(
          (g) => g.name === grant.name && g.status === "ACTIVE",
        );
      },
      { timeout: 30_000, message: "grant should activate after approval" },
    )
    .toBe(true);
  return grant.name;
}

// Open the export drawer (after a query is on screen) and return it, ready at
// the Confirm step. Centralizes the click-Export + "Export Data" dialog + wait
// ritual that C1/C7/C9 and the masking export helper all need.
async function openExportDrawer() {
  await page.getByRole("button", { name: /^Export$/ }).first().click();
  const drawer = page.getByRole("dialog").filter({ hasText: "Export Data" });
  await expect(drawer.getByText("Export rows")).toBeVisible({ timeout: 10_000 });
  return drawer;
}

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  projectId = env.project.split("/").pop()!;
  await env.api.login(env.adminEmail, env.adminPassword);
  ({ workspace } = await env.api.getActuatorInfo());
  pgPort = await getInstancePgPort(env);

  const existing = await env.api.getPolicy(`${workspace}/policies/data_query`);
  const qdp =
    (existing as { queryDataPolicy?: Record<string, unknown> } | null)
      ?.queryDataPolicy ?? {};
  originalDisableExport = Boolean(qdp.disableExport);
  originalDisableCopyData = Boolean(qdp.disableCopyData);
  originalAllowAdminDataSource = Boolean(qdp.allowAdminDataSource);
  originalMaxRows = Number(qdp.maximumResultRows ?? 0);
  const project = (await env.api.getProject(env.project)) as {
    allowJustInTimeAccess?: boolean;
    allowSelfApproval?: boolean;
  };
  originalJIT = Boolean(project.allowJustInTimeAccess);
  originalSelfApproval = Boolean(project.allowSelfApproval);

  // Clipboard permission so the C9 generate-password copy-gate hits its
  // SUCCESS path; acceptDownloads so the masking tests can read the export zip.
  sharedContext = await browser.newContext({
    storageState: ".auth/state.json",
    permissions: ["clipboard-read", "clipboard-write"],
    acceptDownloads: true,
  });
  page = await sharedContext.newPage();
  sqlEditor = new SqlEditorPage(page, env.baseURL);
});

test.afterAll(async () => {
  // Restore each piece of mutated state independently so one failing restore
  // doesn't skip the others (and so a leak in one field is isolated).
  for (const restore of [
    () => setWorkspaceExportPolicy(originalDisableExport),
    () =>
      env.api.updateProjectSettings(env.project, {
        allowJustInTimeAccess: originalJIT,
        allowSelfApproval: originalSelfApproval,
      }),
  ]) {
    try {
      await restore();
    } catch {
      /* best-effort restore */
    }
  }
  await sharedContext?.close();
});

// --- Direct export when the policy allows it (C1, C9) ---------------------

test.describe("Direct export when the policy allows it (C1, C9)", () => {
  test("running a query then Export opens the backend-export drawer and Confirm fires the Export RPC", async () => {
    await applyExportState({ disableExport: false, jit: originalJIT });
    await sqlEditor.runPreparedQuery("SELECT emp_no, first_name FROM employee LIMIT 5;");

    await expect(
      page.getByRole("button", { name: /^Export$/ }).first(),
    ).toBeVisible({ timeout: 10_000 });
    const drawer = await openExportDrawer();
    await expect(drawer.getByText("Export format")).toBeVisible();
    for (const fmt of ["CSV", "JSON", "SQL", "XLSX"]) {
      await expect(
        drawer.getByRole("radio", { name: fmt, exact: true }),
        `format radio "${fmt}" must be offered`,
      ).toBeVisible();
    }
    await expect(
      drawer.getByText("Encrypt with password (Optional)"),
    ).toBeVisible();

    // "Wired vs works": Confirm must fire the backend Export RPC, not just
    // close the drawer. The Connect path is "/bytebase.v1.SQLService/Export"
    // — a "." precedes "SQLService", so match without a leading slash.
    const exportResponse = page.waitForResponse(
      (r) => r.url().includes("SQLService/Export"),
      { timeout: 20_000 },
    );
    await drawer.getByRole("button", { name: "Confirm", exact: true }).click();
    const resp = await exportResponse;
    expect(resp.ok(), "Export RPC should succeed").toBeTruthy();
  });

  test("Generate password fills an 8-character value (copy-gated) (C9)", async () => {
    await applyExportState({ disableExport: false, jit: originalJIT });
    await sqlEditor.runPreparedQuery("SELECT emp_no FROM employee LIMIT 2;");

    const drawer = await openExportDrawer();
    await drawer.getByRole("button", { name: "Generate password" }).click();

    // The password commits to the field ONLY after the clipboard copy
    // resolves (the copy-gate). With clipboard permission granted, the
    // success path fills an 8-char crypto-random value.
    const passwordInput = drawer.locator('input[type="password"]');
    await expect
      .poll(async () => (await passwordInput.inputValue()).length, {
        timeout: 5000,
        message: "generated password should fill the field after a successful copy",
      })
      .toBe(8);

    // Prove the copy-gate actually copied — not just that the field filled. The
    // contract is "commit to the field ONLY after the clipboard write succeeds",
    // so the clipboard must hold exactly the generated value.
    const fieldValue = await passwordInput.inputValue();
    const clipboard = await page.evaluate(() => navigator.clipboard.readText());
    expect(
      clipboard,
      "the generated password must have been copied to the clipboard",
    ).toBe(fieldValue);

    await drawer.getByRole("button", { name: "Cancel", exact: true }).click();
  });
});

// --- Gated export, JIT on → Request export (C2, C3, C4) -------------------

test.describe("Export gated by policy, JIT enabled (C2, C3, C4)", () => {
  test('the toolbar shows "Request export" instead of "Export"', async () => {
    await applyExportState({ disableExport: true, jit: true });
    await sqlEditor.runPreparedQuery("SELECT emp_no, first_name FROM employee LIMIT 5;");

    await expect(
      page.getByRole("button", { name: "Request export", exact: true }).first(),
    ).toBeVisible({ timeout: 10_000 });
    await expect(page.getByRole("button", { name: /^Export$/ })).toHaveCount(0);
    await expect(
      page.getByRole("button", { name: "Copy", exact: true }).first(),
    ).toBeVisible();
  });

  test('"Request export" opens the grant drawer pre-filled with the statement, DB, and Export checked (unmask not)', async () => {
    await applyExportState({ disableExport: true, jit: true });
    const statement = "SELECT emp_no, last_name FROM employee LIMIT 4;";
    await sqlEditor.runPreparedQuery(statement);

    await page
      .getByRole("button", { name: "Request export", exact: true })
      .first()
      .click();

    const drawer = page
      .getByRole("dialog")
      .filter({ hasText: "Request Data Access" });
    await expect(drawer.getByText("Request Data Access")).toBeVisible({
      timeout: 10_000,
    });
    await expect(drawer.getByText(statement)).toBeVisible();
    await expect(drawer.getByText(env.databaseId, { exact: false })).toBeVisible();

    // C4: the RequestExportButton scopes its pre-fill to the EXPORT capability
    // (#20516 changed it from unmask+export to export-only — "Request export"
    // requests export, not unmasking). So Export is pre-checked and Unmask is
    // left for the user to opt into.
    await expect(
      drawer.getByRole("checkbox", { name: "Export the query result" }),
      "Export must be pre-checked",
    ).toBeChecked();
    await expect(
      drawer.getByRole("checkbox", { name: "See unmasked sensitive data" }),
      "Unmask must NOT be pre-checked by Request-export (#20516)",
    ).not.toBeChecked();

    await page.keyboard.press("Escape");
  });

  test("submitting a Request-export creates a grant with export=true and unmask=false (BYT-9654 end-to-end)", async () => {
    // BYT-9654 end-to-end: the C4 sibling above proves the drawer PRE-FILL
    // (Export checked, Unmask not). This proves the SUBMITTED grant carries the
    // same capabilities — an export request must not silently escalate into an
    // unmask request that approvers would unknowingly grant.
    await applyExportState({ disableExport: true, jit: true });
    // Unique statement so we can find exactly this grant via the API.
    const marker = Date.now();
    const statement = `SELECT emp_no FROM employee WHERE emp_no = ${marker % 100000} LIMIT 1;`;
    await sqlEditor.runPreparedQuery(statement);

    await page
      .getByRole("button", { name: "Request export", exact: true })
      .first()
      .click();

    const drawer = page
      .getByRole("dialog")
      .filter({ hasText: "Request Data Access" });
    await expect(drawer.getByText("Request Data Access")).toBeVisible({
      timeout: 10_000,
    });
    // Leave the pre-filled capabilities untouched (Export checked, Unmask not).
    await expect(
      drawer.getByRole("checkbox", { name: "Export the query result" }),
    ).toBeChecked();
    await expect(
      drawer.getByRole("checkbox", { name: "See unmasked sensitive data" }),
    ).not.toBeChecked();

    // Reason is the only field the user must fill (targets, query, and a 4h
    // duration are pre-filled by RequestExportButton). The drawer ALSO contains
    // a Monaco editor whose hidden textarea has class "inputarea" — exclude it so
    // we target the real Reason <textarea>.
    const reason = drawer.locator('textarea:not([class*="inputarea"])').first();
    await expect(reason).toBeVisible({ timeout: 10_000 });
    await reason.click();
    await reason.fill("e2e BYT-9654 export request");
    await expect(reason).toHaveValue("e2e BYT-9654 export request");
    const submit = drawer.locator("[data-submit-btn]");
    await expect(submit).toBeEnabled({ timeout: 8000 });
    await submit.click();

    // The created grant (PENDING or ACTIVE) must carry export=true, unmask=false.
    let createdName = "";
    await expect
      .poll(
        async () => {
          const r = await env.api.searchMyAccessGrants(env.project);
          const g = r.accessGrants.find((x) => x.query === statement);
          if (!g) return "not-found";
          createdName = g.name;
          // The API omits `unmask` when false (returns undefined), so coerce to
          // boolean: the contract is export=true, unmask NOT set.
          return `export=${!!g.export},unmask=${!!g.unmask}`;
        },
        {
          timeout: 20_000,
          message: "the submitted Request-export grant should be created",
        },
      )
      .toBe("export=true,unmask=false");

    // Clean up the grant we created so it doesn't leak into sibling describes.
    if (createdName) await env.api.revokeAccessGrant(createdName).catch(() => {});
  });
});

// --- Gated export, JIT off → no export affordance (C8) --------------------

test.describe("Export gated by policy, JIT disabled (C8)", () => {
  test("neither an Export button nor a Request-export button is shown", async () => {
    await applyExportState({ disableExport: true, jit: false });
    await sqlEditor.runPreparedQuery("SELECT emp_no FROM employee LIMIT 3;");

    await expect(
      page.getByRole("button", { name: "Copy", exact: true }).first(),
    ).toBeVisible({ timeout: 10_000 });
    await expect(page.getByRole("button", { name: /^Export$/ })).toHaveCount(0);
    await expect(
      page.getByRole("button", { name: "Request export", exact: true }),
    ).toHaveCount(0);
  });
});

// --- Multi-statement result → one affordance above the tabs (C10) ---------

test.describe("Multi-statement result export affordance (C10)", () => {
  test("two SELECTs render Query #1 / #2 tabs with a single Request-export control above them", async () => {
    await applyExportState({ disableExport: true, jit: true });
    await sqlEditor.runPreparedQuery("SELECT 1 AS a; SELECT 2 AS b;");

    await expect(
      page.getByRole("tab", { name: /Query\s*#1/ }).first(),
    ).toBeVisible({ timeout: 10_000 });
    await expect(
      page.getByRole("tab", { name: /Query\s*#2/ }).first(),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Request export", exact: true }),
    ).toHaveCount(1);
  });
});

// --- Active export grant re-enables Export for its exact statement (C7),
//     surfaces an Export badge (C5), and is filterable (C6) ----------------

test.describe("Active export grant (C5, C6, C7)", () => {
  const GRANTED_STATEMENT =
    "SELECT emp_no, first_name, last_name FROM employee LIMIT 6;";
  const UNGRANTED_STATEMENT = "SELECT emp_no FROM employee LIMIT 6;";
  let grantedAccessName = "";

  test.beforeAll(async () => {
    await env.api.updateProjectSettings(env.project, {
      allowJustInTimeAccess: true,
    });
    await setWorkspaceExportPolicy(true);
    grantedAccessName = await createActiveGrant(GRANTED_STATEMENT, {
      unmask: true,
      export: true,
    });
  });

  test.afterAll(async () => {
    // Best-effort: revokeAccessGrant now throws on non-OK, so guard the teardown
    // so a transient cleanup failure can't fail the suite after the tests passed.
    try {
      if (grantedAccessName) await env.api.revokeAccessGrant(grantedAccessName);
    } catch {
      /* best-effort */
    }
  });

  test.beforeEach(async () => {
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(1000);
  });

  test("the granted statement re-enables Export AND exporting under the grant succeeds (C7)", async () => {
    await sqlEditor.runPreparedQuery(GRANTED_STATEMENT);

    const exportButton = page.getByRole("button", { name: /^Export$/ }).first();
    await expect(exportButton).toBeVisible({ timeout: 10_000 });
    await expect(
      page.getByRole("button", { name: "Request export", exact: true }),
    ).toHaveCount(0);

    await exportButton.hover();
    // The tooltip attributes the export to the grant. #20513 made the headline
    // "Export available via issue \"<title>\"" when the grant has an approval
    // issue (ours does), falling back to "Export enabled by access grant"
    // otherwise (or before the issue title resolves) — accept either.
    await expect(
      page.getByText(/Export (available via issue|enabled by access grant)/),
    ).toBeVisible({ timeout: 5000 });

    // Not just "the button shows" — actually export. Under disableExport=true the
    // backend must honor the export grant, so Confirm fires a successful Export
    // RPC. (We can't read ExportResponse.applied_access_grant here: the Export RPC
    // uses the Connect protobuf codec, so the response body is binary protobuf —
    // the zip-bearing ExportResponse — not JSON. The grant-vs-ACL authorization is
    // a backend concern covered by backend tests; here we prove the UI exports
    // end-to-end under the grant.)
    const drawer = await openExportDrawer();
    const exportResponse = page.waitForResponse(
      (r) => r.url().includes("SQLService/Export"),
      { timeout: 20_000 },
    );
    await drawer.getByRole("button", { name: "Confirm", exact: true }).click();
    const resp = await exportResponse;
    expect(
      resp.ok(),
      "exporting under an active export grant must succeed",
    ).toBeTruthy();
  });

  test("running a DIFFERENT statement keeps export gated (Request export, no Export) (C7)", async () => {
    await sqlEditor.runPreparedQuery(UNGRANTED_STATEMENT);

    await expect(
      page.getByRole("button", { name: "Request export", exact: true }).first(),
    ).toBeVisible({ timeout: 10_000 });
    await expect(page.getByRole("button", { name: /^Export$/ })).toHaveCount(0);
  });

  test("the granted access is listed in the ACCESS pane with an Export badge (C5)", async () => {
    await sqlEditor.gutterAccessTab.click();
    await page.waitForTimeout(1000);

    const grantRow = page
      .locator("div.border-b")
      .filter({ hasText: GRANTED_STATEMENT })
      .first();
    await expect(grantRow).toBeVisible({ timeout: 10_000 });
    await expect(
      grantRow.getByText("Export", { exact: true }),
      "the access grant must carry an Export badge",
    ).toBeVisible();
    await expect(grantRow.getByText("Unmask", { exact: true })).toBeVisible();
  });

  test('filtering the ACCESS pane by export "No" hides the export grant (C6)', async () => {
    await sqlEditor.gutterAccessTab.click();
    await page.waitForTimeout(1000);
    await expect(page.getByText(GRANTED_STATEMENT).first()).toBeVisible({
      timeout: 10_000,
    });

    const searchRow = page
      .locator("div")
      .filter({ has: page.getByRole("button", { name: "Request Access" }) })
      .last();
    await searchRow.locator("input").first().click();
    await page
      .getByText("Filter by whether the grant allows exporting the query result")
      .click();
    await page.getByText("No", { exact: true }).click();
    await page.waitForTimeout(800);

    await expect(page.getByText(GRANTED_STATEMENT)).toHaveCount(0);
    await expect(page.getByText("No access requests")).toBeVisible({
      timeout: 5000,
    });
  });
});

// --- Masking ↔ export security contract -----------------------------------
//
// Exporting a MASKED column must not leak the plaintext (no grant); an active
// unmask grant for the exact statement reveals it. We download the real export
// zip and inspect the CSV bytes — a leak only shows in the file, not on screen.

test.describe("Masking ↔ export security", () => {
  const SEMANTIC_TYPE_ID = "bb.export-mask-qa";
  const SCHEMA = "qa_export_mask";
  const TABLE = "secret";
  const COLUMN = "ssn";
  const SECRET_1 = "QASECRETVALUEAAA";
  const SECRET_2 = "QASECRETVALUEBBB";
  // The distinctive shared token in both secrets. With a FULL mask the whole
  // value becomes MASK_MARKER, so this token must be wholly absent from a masked
  // export — asserting only the full 16-char value is too weak, because a PARTIAL
  // mask intentionally reveals substrings and would let the recognizable token
  // leak while the test stayed green.
  const SECRET_TOKEN = "QASECRETVALUE";
  // A UNIQUE full-mask substitution — not Bytebase's default "******". Asserting
  // a unique marker proves OUR semantic type actually applied: if it silently
  // failed to apply, the column would render plaintext (no masking rules exist on
  // the fresh server) or a default "******", and this marker would be absent.
  const MASK_MARKER = "QAEXPORTMASKED";
  const STATEMENT = `SELECT id, ${COLUMN} FROM ${SCHEMA}.${TABLE} ORDER BY id;`;
  let unmaskGrantName = "";
  // Snapshot the SEMANTIC_TYPES setting so afterAll restores it (we overwrite it
  // here, which would otherwise clobber it for later specs).
  let originalSemanticTypes: unknown = null;

  function readExportedCsv(zipPath: string): string {
    return execFileSync("unzip", ["-p", zipPath, "*.result.csv"], {
      encoding: "utf-8",
    });
  }

  // Run STATEMENT, click Export → Confirm, return the exported CSV text.
  async function exportStatementToCsv(): Promise<string> {
    await sqlEditor.runPreparedQuery(STATEMENT);
    const drawer = await openExportDrawer();
    // Bind the download to the backend Export RPC: the masking we're verifying is
    // server-side, so the exported bytes MUST come from SQLService/Export — not a
    // client-side dump of the (already-masked) result grid. Waiting on both proves
    // the masked CSV is the backend's output.
    const exportRpc = page.waitForResponse(
      (r) => r.url().includes("SQLService/Export"),
      { timeout: 20_000 },
    );
    const downloadPromise = page.waitForEvent("download", { timeout: 20_000 });
    await drawer.getByRole("button", { name: "Confirm", exact: true }).click();
    const resp = await exportRpc;
    expect(resp.ok(), "the masked export must come from the backend Export RPC").toBeTruthy();
    const download = await downloadPromise;
    return readExportedCsv(await download.path());
  }

  test.beforeAll(async () => {
    // Export allowed (so the Export button shows), JIT on for the grant test.
    await setWorkspaceExportPolicy(false);
    await env.api.updateProjectSettings(env.project, {
      allowJustInTimeAccess: true,
    });

    // Snapshot then register a FULL-mask semantic type. The masker resolves a
    // column's algorithm from this registered setting by id (masking_evaluator.go),
    // and FullMasker replaces the ENTIRE value with the substitution — so the
    // masked cell is exactly MASK_MARKER and NO plaintext substring survives.
    // (A partial/range mask would expose substrings and make a "no leak" check
    // meaningless.)
    originalSemanticTypes =
      (await env.api.getSetting("SEMANTIC_TYPES"))?.value ?? null;
    await env.api.upsertSetting(
      "SEMANTIC_TYPES",
      {
        semanticType: {
          types: [
            {
              id: SEMANTIC_TYPE_ID,
              title: "QA Export Mask",
              description: "masking ↔ export e2e",
              algorithm: { fullMask: { substitution: MASK_MARKER } },
            },
          ],
        },
      },
      "value.semantic_type",
    );

    // Create the table with known plaintext via psql, sync, then mask the
    // column through the catalog (note the nested columns:{columns:[...]} shape).
    execSql(env.databaseId, pgPort, `DROP SCHEMA IF EXISTS ${SCHEMA} CASCADE`);
    execSql(env.databaseId, pgPort, `CREATE SCHEMA ${SCHEMA}`);
    execSql(
      env.databaseId,
      pgPort,
      `CREATE TABLE ${SCHEMA}.${TABLE} (id INT PRIMARY KEY, ${COLUMN} TEXT)`,
    );
    execSql(
      env.databaseId,
      pgPort,
      `INSERT INTO ${SCHEMA}.${TABLE} VALUES (1, '${SECRET_1}'), (2, '${SECRET_2}')`,
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
    // Restore each mutation independently and clean up the catalog: drop the
    // schema THEN re-sync so Bytebase forgets the masked table's metadata, and
    // restore the SEMANTIC_TYPES setting we overwrote.
    try {
      if (unmaskGrantName) await env.api.revokeAccessGrant(unmaskGrantName);
    } catch {
      /* best-effort */
    }
    try {
      execSql(env.databaseId, pgPort, `DROP SCHEMA IF EXISTS ${SCHEMA} CASCADE`);
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

  test("exporting a masked column does NOT leak the plaintext (no grant)", async () => {
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(1000);

    await sqlEditor.runPreparedQuery(STATEMENT);
    // No recognizable plaintext on screen (the full mask renders MASK_MARKER).
    await expect(
      page.locator(`text=${SECRET_TOKEN}`),
      "no recognizable plaintext substring may be visible on screen without a grant",
    ).toHaveCount(0);

    const csv = await exportStatementToCsv();
    // Airtight no-leak: the distinctive token must be wholly absent (full mask),
    // not just the full 16-char value.
    expect(
      csv.includes(SECRET_TOKEN) ||
        csv.includes(SECRET_1) ||
        csv.includes(SECRET_2),
      `export leaked recognizable plaintext — CSV:\n${csv}`,
    ).toBe(false);
    // And the masked rows are actually present (not an empty/zero-row export):
    // the masked marker proves both that rows exist and that masking produced
    // the no-leak result.
    expect(
      csv.includes(MASK_MARKER),
      `expected ${csv.split("\n").length - 1} masked rows with the mask marker, CSV:\n${csv}`,
    ).toBe(true);
    expect(csv).toContain(COLUMN);
  });

  test("an active unmask grant for the exact statement reveals the plaintext in the export", async () => {
    await env.api.updateProjectSettings(env.project, {
      allowSelfApproval: true,
    });
    unmaskGrantName = await createActiveGrant(STATEMENT, {
      unmask: true,
      export: true,
    });

    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(1000);

    await sqlEditor.runPreparedQuery(STATEMENT);
    await expect(page.locator(`text=${SECRET_1}`).first()).toBeVisible({
      timeout: 10_000,
    });

    const csv = await exportStatementToCsv();
    expect(
      csv,
      `with an unmask grant the export must reveal the plaintext, got:\n${csv}`,
    ).toContain(SECRET_1);
  });
});

// --- REGRESSION GUARD (BYT-9656, FIXED) -----------------------------------
//
// Re-requesting a revoked grant from the ACCESS pane must restore BOTH the
// Unmask AND Export capabilities the original grant held. BYT-9656 was the bug
// where Export was dropped (AccessPane forwarded `unmask` but not `export`).
//
// FIXED in #20516 ("chore: update access ui"): AccessPane now renders
// `<AccessGrantRequestDrawer ... export={pendingCreate?.export}>`, so the
// re-request drawer pre-checks Export again. This started life as a `test.fail`
// -free bug-lock asserting the buggy `Export not.toBeChecked()` state; verified
// against main that the fix flips it (Export now checked), so it is now a
// forward regression guard asserting both capabilities are preserved.

test.describe("Re-requesting a revoked grant restores Unmask AND Export (BYT-9656)", () => {
  const REVOKED_STATEMENT = "SELECT emp_no, first_name FROM employee LIMIT 9;";

  test.beforeAll(async () => {
    await env.api.updateProjectSettings(env.project, {
      allowJustInTimeAccess: true,
    });
    // Seed an unmask+export grant, then revoke it so the ACCESS pane offers
    // a "Re-request" affordance for it. (revokeAccessGrant throws on non-OK,
    // so a failed revoke surfaces here instead of silently leaving it ACTIVE.)
    const name = await createActiveGrant(REVOKED_STATEMENT, {
      unmask: true,
      export: true,
    });
    await env.api.revokeAccessGrant(name);
  });

  test("re-request restores both the Unmask and Export capabilities of the original grant", async () => {
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(1000);
    await sqlEditor.gutterAccessTab.click();
    await page.waitForTimeout(1000);

    // The default filter is status ACTIVE + PENDING, which hides REVOKED
    // grants. Clear all filters (the AdvancedSearch clear-all button — the
    // only `rounded-full mr-1` button on screen with no query run, so no
    // result scroll buttons). Class locator is acceptable for a bug-lock.
    await page.locator("button.rounded-full.mr-1").first().click();
    await page.waitForTimeout(800);

    // Hard preconditions — a failure here is a REAL failure, not the bug.
    // Scope to the grant ITEM (`div.border-b` is the AccessGrantItem root), not a
    // plain `div...first()` which resolves to the whole list container. Earlier
    // describes leave their own grants REVOKED, so several "Revoked" badges are
    // on screen — only this item carries REVOKED_STATEMENT.
    const grantRow = page
      .locator("div.border-b")
      .filter({ hasText: REVOKED_STATEMENT })
      .first();
    await expect(grantRow).toBeVisible({ timeout: 10_000 });
    await expect(grantRow.getByText("Revoked", { exact: true })).toBeVisible();
    // Precondition: the grant we re-request MUST be an export grant (carries the
    // Export badge). This pins the lock: if we ever re-requested a non-export
    // grant, Export-unchecked would be correct, not the bug — so prove it here.
    await expect(
      grantRow.getByText("Export", { exact: true }),
      "the revoked grant must be an export grant (Export badge) for this lock to be meaningful",
    ).toBeVisible();
    await grantRow.getByRole("button", { name: "Re-request" }).click();

    const drawer = page
      .getByRole("dialog")
      .filter({ hasText: "Request Data Access" });
    await expect(drawer.getByText("Request Data Access")).toBeVisible({
      timeout: 10_000,
    });

    // Both capabilities the original grant held must be pre-checked on
    // re-request. Unmask was always carried over; Export is the one BYT-9656
    // dropped and #20516 restored — this is the assertion that guards it.
    await expect(
      drawer.getByRole("checkbox", { name: "See unmasked sensitive data" }),
      "Unmask must be pre-checked on re-request",
    ).toBeChecked();
    await expect(
      drawer.getByRole("checkbox", { name: "Export the query result" }),
      "BYT-9656 regression guard: Export must be pre-checked on re-request of an " +
        "export grant (AccessPane must forward `export`)",
    ).toBeChecked();
  });
});
