import { execFileSync } from "child_process";
import { test, expect, type Page, type BrowserContext } from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";
import { MaskingExemptionPage, GrantExemptionPage, SqlEditorPage } from "./masking-exemption.page";

// Give all tests generous timeouts (Mode B's disposable server can be slow)
test.setTimeout(120_000);

interface MaskingTestColumn {
  sampleTable: string;
  sampleSchema: string;
  sampleColumn: string;
  primaryKeyColumn: string;
  primaryKeyValue: string;
  knownUnmaskedValue: string;
}

interface MaskingTestData {
  // Column that is masked via classification level (set in demo data)
  classificationColumn: MaskingTestColumn;
  // Column that we explicitly mask via semanticType (we apply it in beforeAll)
  semanticTypeColumn: MaskingTestColumn;
}

let env: TestEnv & { api: BytebaseApiClient };
let maskingData: MaskingTestData;

// Shared browser context and page — reused across ALL tests.
// Avoids browser open/close overhead and NVirtualList first-render issues.
let sharedContext: BrowserContext;
let page: Page;

// ── Feature-specific test data setup ──

// Fixed values we control: the test creates its own schema/table/rows,
// applies masking, and drops everything in teardown.
const TEST_SCHEMA = "e2e_masking";
const TEST_TABLE = "t";
const CLASSIFICATION_COLUMN = "col_classification";
const SEMANTIC_TYPE_COLUMN = "col_semantic";
const ROW_PK = "1";
const CLASSIFICATION_VALUE = "ClassValueABCDE";
const SEMANTIC_TYPE_VALUE = "SemValueABCDE";
const CLASSIFICATION_LEVEL = "1-2"; // matches demo classification rules

// Execute SQL via psql over Unix socket on the sample Postgres instance.
// Used for DDL/DML setup and teardown — Bytebase's query API is read-only.
function execSql(env: TestEnv & { api: BytebaseApiClient }, dbName: string, sql: string): void {
  // Sample instance Postgres runs on PORT+4 via Unix socket at /tmp.
  // Parse port from baseURL (http://localhost:PORT).
  const match = env.baseURL.match(/localhost:(\d+)/);
  if (!match) throw new Error(`Cannot parse port from baseURL: ${env.baseURL}`);
  const sampleInstancePort = String(parseInt(match[1], 10) + 4);
  execFileSync("psql", [
    "-h", "/tmp",
    "-p", sampleInstancePort,
    "-U", "bbsample",
    "-d", dbName,
    "-v", "ON_ERROR_STOP=1",
    "-c", sql,
  ], { stdio: "pipe" });
}

async function createMaskingTestData(env: TestEnv & { api: BytebaseApiClient }): Promise<MaskingTestData> {
  const dbName = env.database.split("/").pop()!;

  // Drop anything leftover from a previous run, then create fresh schema/table/row
  execSql(env, dbName, `DROP SCHEMA IF EXISTS ${TEST_SCHEMA} CASCADE`);
  execSql(env, dbName, `CREATE SCHEMA ${TEST_SCHEMA}`);
  execSql(env, dbName, `CREATE TABLE ${TEST_SCHEMA}.${TEST_TABLE} (
    id INTEGER PRIMARY KEY,
    ${CLASSIFICATION_COLUMN} TEXT NOT NULL,
    ${SEMANTIC_TYPE_COLUMN} TEXT NOT NULL
  )`);
  execSql(env, dbName, `INSERT INTO ${TEST_SCHEMA}.${TEST_TABLE} (id, ${CLASSIFICATION_COLUMN}, ${SEMANTIC_TYPE_COLUMN})
    VALUES (${ROW_PK}, '${CLASSIFICATION_VALUE}', '${SEMANTIC_TYPE_VALUE}')`);

  // Sync the database schema so Bytebase picks up the new table
  try {
    await env.api.syncDatabase(env.database);
  } catch {
    // sync endpoint may not be available; catalog update will still work
  }

  // Configure catalog: classification on one column, semanticType on the other
  await env.api.updateCatalog(env.database, {
    name: `${env.database}/catalog`,
    schemas: [{
      name: TEST_SCHEMA,
      tables: [{
        name: TEST_TABLE,
        columns: {
          columns: [
            { name: CLASSIFICATION_COLUMN, classification: CLASSIFICATION_LEVEL },
            { name: SEMANTIC_TYPE_COLUMN, semanticType: "bb.default-partial" },
          ],
        },
      }],
    }],
  });

  const common = {
    sampleTable: TEST_TABLE,
    sampleSchema: TEST_SCHEMA,
    primaryKeyColumn: "id",
    primaryKeyValue: ROW_PK,
  };
  return {
    classificationColumn: {
      ...common,
      sampleColumn: CLASSIFICATION_COLUMN,
      knownUnmaskedValue: CLASSIFICATION_VALUE,
    },
    semanticTypeColumn: {
      ...common,
      sampleColumn: SEMANTIC_TYPE_COLUMN,
      knownUnmaskedValue: SEMANTIC_TYPE_VALUE,
    },
  };
}

function dropMaskingTestData(env: TestEnv & { api: BytebaseApiClient }): void {
  const dbName = env.database.split("/").pop()!;
  try {
    execSql(env, dbName, `DROP SCHEMA IF EXISTS ${TEST_SCHEMA} CASCADE`);
  } catch {
    // best effort — server may already be torn down
  }
}

async function grantExemption(description: string): Promise<void> {
  const existing = await env.api.getPolicy(`${env.project}/policies/masking_exemption`) as {
    maskingExemptionPolicy?: { exemptions: { members: string[]; condition?: { expression: string; description: string } }[] };
  } | null;
  const exemptions = existing?.maskingExemptionPolicy?.exemptions ?? [];
  exemptions.push({
    members: [`user:${env.adminEmail}`],
    condition: { expression: "", description },
  });
  await env.api.upsertPolicy(env.project, "masking_exemption", {
    type: "MASKING_EXEMPTION",
    maskingExemptionPolicy: { exemptions },
  });
}

async function revokeAllExemptions(): Promise<void> {
  await env.api.upsertPolicy(env.project, "masking_exemption", {
    type: "MASKING_EXEMPTION",
    maskingExemptionPolicy: { exemptions: [] },
  });
}

// ── Setup/Teardown ──

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);

  // Create our own test schema/table/data — no dependency on demo content.
  // Applies classification to one column and semanticType to another.
  maskingData = await createMaskingTestData(env);

  // Create a shared browser context/page for all tests
  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
});

test.afterAll(async () => {
  await sharedContext?.close();
  await revokeAllExemptions().catch(() => { /* ignore */ });
  dropMaskingTestData(env);
});

// ── Exemption List Page ──

test.describe("Exemption List Page", () => {
  test.beforeAll(async () => {
    await grantExemption("Test grant A");
    await grantExemption("Test grant B");
  });

  test("loads and displays member list with grant details", async () => {
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
    const projectId = env.project.split("/").pop()!;
    await exemptionPage.goto(projectId);

    await expect(exemptionPage.grantExemptionButton).toBeVisible();
    await expect(exemptionPage.activeTab).toBeVisible();
    await expect(exemptionPage.expiredTab).toBeVisible();
    await expect(exemptionPage.allTab).toBeVisible();
    await expect(page.getByText(env.adminEmail).first()).toBeVisible();
  });

  test("Active tab shows only active grants", async () => {
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
    const projectId = env.project.split("/").pop()!;
    await exemptionPage.goto(projectId);
    await exemptionPage.activeTab.click();
    await page.waitForTimeout(500);

    await expect(page.getByText("Never expires").first()).toBeVisible();
    await expect(page.getByText("(Expired)")).toHaveCount(0);
  });

  test("selecting a member shows their grants in detail panel", async () => {
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
    const projectId = env.project.split("/").pop()!;
    await exemptionPage.goto(projectId);
    await exemptionPage.selectMember(env.adminEmail);
    await page.waitForTimeout(500);

    await expect(page.getByText(/\d+ masking exemption/)).toBeVisible();
    await expect(page.getByText("Reason:").first()).toBeVisible();
  });

  test("grant card shows reason", async () => {
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
    const projectId = env.project.split("/").pop()!;
    await exemptionPage.goto(projectId);
    await exemptionPage.selectMember(env.adminEmail);
    await page.waitForTimeout(500);

    await expect(page.getByText("Reason:").first()).toBeVisible();
  });

  test("clicking All tab removes Active filter and shows all data", async () => {
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
    const projectId = env.project.split("/").pop()!;
    await exemptionPage.goto(projectId);

    await exemptionPage.allTab.click();
    await page.waitForTimeout(500);

    await expect(page.getByText("status: Active")).not.toBeVisible();
    await expect(page.getByText("status: Expired")).not.toBeVisible();
  });

  test("grant card has no excessive top padding", async () => {
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
    const projectId = env.project.split("/").pop()!;
    await exemptionPage.goto(projectId);
    await exemptionPage.selectMember(env.adminEmail);
    await page.waitForTimeout(500);

    const grantCard = page.locator("[class*='border border-gray']").first();
    const cardBox = await grantCard.boundingBox();
    const header = grantCard.locator("> div").first();
    const headerBox = await header.boundingBox();
    if (cardBox && headerBox) {
      const topGap = headerBox.y - cardBox.y;
      expect(topGap).toBeLessThanOrEqual(12);
    }
  });
});

// ── Service Account badge ──

test.describe("Member Type Badges", () => {
  const saId = "e2e-test-sa";
  let saEmail = "";

  test.beforeAll(async () => {
    const sa = await env.api.createServiceAccount(env.project, saId, "E2E Test SA");
    saEmail = sa.email;
    const existing = await env.api.getPolicy(`${env.project}/policies/masking_exemption`) as {
      maskingExemptionPolicy?: { exemptions: { members: string[]; condition?: { expression: string; description: string } }[] };
    } | null;
    const exemptions = existing?.maskingExemptionPolicy?.exemptions ?? [];
    exemptions.push({ members: [`serviceAccount:${saEmail}`], condition: { expression: "", description: "SA test" } });
    await env.api.upsertPolicy(env.project, "masking_exemption", {
      type: "MASKING_EXEMPTION",
      maskingExemptionPolicy: { exemptions },
    });
  });

  test.afterAll(async () => {
    await revokeAllExemptions();
    await env.api.deleteServiceAccount(saEmail);
  });

  test("service account badge renders on single line", async () => {
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
    const projectId = env.project.split("/").pop()!;
    await exemptionPage.goto(projectId);
    await exemptionPage.allTab.click();
    await page.waitForTimeout(500);

    const badge = page.getByText("Service Account", { exact: true }).first();
    await expect(badge).toBeVisible();
    const box = await badge.boundingBox();
    expect(box).toBeTruthy();
    expect(box!.height).toBeLessThan(45);
  });
});

// ── Grant and Revoke ──

test.describe("Grant and Revoke", () => {
  test.beforeEach(async () => {
    await revokeAllExemptions();
  });

  test("grant exemption via UI and verify it appears in list", async () => {
    const projectId = env.project.split("/").pop()!;
    const grantPage = new GrantExemptionPage(page, env.baseURL);
    const listPage = new MaskingExemptionPage(page, env.baseURL);

    await grantPage.goto(projectId);
    await expect(grantPage.allRadio).toBeChecked();
    await expect(grantPage.confirmButton).toBeDisabled();

    await grantPage.reasonInput.fill("E2E test grant");
    const adminName = env.adminEmail === "demo@example.com" ? "Demo" : "Admin";
    await grantPage.selectAccount(adminName);
    await expect(grantPage.confirmButton).toBeEnabled();
    await grantPage.submit();
    await page.waitForTimeout(1000);

    await listPage.goto(projectId);
    await listPage.selectMember(env.adminEmail);
    await page.waitForTimeout(500);
    await expect(page.getByText("E2E test grant")).toBeVisible();
  });

  test("revoke exemption via UI and verify it disappears", async () => {
    const projectId = env.project.split("/").pop()!;
    const listPage = new MaskingExemptionPage(page, env.baseURL);

    await grantExemption("To be revoked");

    await listPage.goto(projectId);
    await listPage.selectMember(env.adminEmail);
    await page.waitForTimeout(500);
    await expect(page.getByText("To be revoked")).toBeVisible();

    const revokeBtn = page.getByRole("button", { name: "Revoke" }).first();
    await revokeBtn.click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await page.getByRole("dialog").getByRole("button", { name: "Confirm" }).click();
    await page.waitForTimeout(500);

    await expect(page.getByText("To be revoked")).not.toBeVisible();
  });

  test("revoke confirmation can be cancelled", async () => {
    const projectId = env.project.split("/").pop()!;
    const listPage = new MaskingExemptionPage(page, env.baseURL);

    await grantExemption("Should survive cancel");

    await listPage.goto(projectId);
    await listPage.selectMember(env.adminEmail);
    await page.waitForTimeout(500);

    await page.getByRole("button", { name: "Revoke" }).first().click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await page.getByRole("dialog").getByRole("button", { name: "Cancel" }).click();
    await expect(page.getByRole("dialog")).not.toBeVisible();
    await expect(page.getByText("Should survive cancel")).toBeVisible();
  });
});

// ── E2E Masking Verification ──

test.describe("E2E Masking Verification", () => {
  // Shared cycle test: masked → grant via UI → unmasked → revoke via UI → masked.
  // Parameterized by which column (classification-masked vs semantic-type-masked).
  const runMaskingCycle = async (target: MaskingTestColumn, reason: string) => {
    const projectId = env.project.split("/").pop()!;
    const instanceId = env.instance.split("/").pop()!;
    const dbId = env.database.split("/").pop()!;
    const sqlEditor = new SqlEditorPage(page, env.baseURL);
    const grantPage = new GrantExemptionPage(page, env.baseURL);
    const listPage = new MaskingExemptionPage(page, env.baseURL);
    const sql = `SELECT "${target.sampleColumn}" FROM "${target.sampleSchema}"."${target.sampleTable}" WHERE "${target.primaryKeyColumn}" = '${target.primaryKeyValue}';`;
    const adminName = env.adminEmail === "demo@example.com" ? "Demo" : "Admin";

    // Step 1: No exemption → data is masked
    await revokeAllExemptions();
    await sqlEditor.gotoWithDb(projectId, instanceId, dbId);
    await sqlEditor.runQuery(sql);
    expect(await sqlEditor.resultContainsText(target.knownUnmaskedValue)).toBe(false);

    // Step 2: Grant exemption via UI → data becomes unmasked
    await grantPage.goto(projectId);
    await grantPage.reasonInput.fill(reason);
    await grantPage.selectAccount(adminName);
    await grantPage.submit();
    await page.waitForTimeout(1000);

    await sqlEditor.gotoWithDb(projectId, instanceId, dbId);
    await sqlEditor.runQuery(sql);
    expect(await sqlEditor.resultContainsText(target.knownUnmaskedValue)).toBe(true);

    // Step 3: Revoke exemption via UI → data becomes masked again
    await listPage.goto(projectId);
    await listPage.selectMember(env.adminEmail);
    await page.waitForTimeout(500);
    await page.getByRole("button", { name: "Revoke" }).first().click();
    await page.getByRole("dialog").getByRole("button", { name: "Confirm" }).click();
    await page.waitForTimeout(500);

    await sqlEditor.gotoWithDb(projectId, instanceId, dbId);
    await sqlEditor.runQuery(sql);
    expect(await sqlEditor.resultContainsText(target.knownUnmaskedValue)).toBe(false);
  };

  test("classification-based masking: cycle via UI", async () => {
    await runMaskingCycle(maskingData.classificationColumn, "e2e classification masking test");
  });

  test("semantic-type-based masking: cycle via UI", async () => {
    await runMaskingCycle(maskingData.semanticTypeColumn, "e2e semantic type masking test");
  });
});

// ── Responsive Layout ──

test.describe("Responsive Layout", () => {
  test.beforeAll(async () => {
    await grantExemption("Layout test grant");
  });

  test.afterAll(async () => {
    // Reset viewport so subsequent tests (if any) aren't stuck at mobile size
    await page.setViewportSize({ width: 1280, height: 720 });
  });

  test("wide screen shows split-panel layout", async () => {
    await page.setViewportSize({ width: 1440, height: 900 });
    const projectId = env.project.split("/").pop()!;
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
    await exemptionPage.goto(projectId);

    await expect(page.getByText(/\d+ masking exemption/).first()).toBeVisible();
    await expect(page.getByText("Reason:").first()).toBeVisible();
  });

  test("narrow screen shows expandable list", async () => {
    await page.setViewportSize({ width: 768, height: 1024 });
    const projectId = env.project.split("/").pop()!;
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
    await exemptionPage.goto(projectId);

    await expect(page.getByText(env.adminEmail).first()).toBeVisible();
    await page.getByText(env.adminEmail).first().click();
    await page.waitForTimeout(300);
    await expect(page.getByText("Reason:").first()).toBeVisible();
  });
});
