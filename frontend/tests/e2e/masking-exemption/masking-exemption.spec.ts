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
// Extracted IDs for convenience — assigned once in file-level beforeAll
let projectId: string;

// Shared browser context and page — reused across ALL tests.
// Avoids browser open/close overhead and NVirtualList first-render issues.
let sharedContext: BrowserContext;
let page: Page;

// ── Feature-specific test data setup ──

// Fixed values we control: the test creates its own schema/table/rows,
// applies masking, and drops everything in teardown.
// All identifiers below must be validated by isSafeIdentifier before being
// interpolated into SQL strings passed to psql. The constants are static
// literals verified at load time; any dynamic values MUST go through the
// validator.
const TEST_SCHEMA = "e2e_masking";
const TEST_TABLE = "t";
const CLASSIFICATION_COLUMN = "col_classification";
const SEMANTIC_TYPE_COLUMN = "col_semantic";
const ROW_PK = 1;
const CLASSIFICATION_VALUE = "ClassValueABCDE";
const SEMANTIC_TYPE_VALUE = "SemValueABCDE";
const CLASSIFICATION_LEVEL = "1-2"; // matches demo classification rules

// Validate SQL identifiers/literal values. Only permits characters safe to
// inline into a SQL string without escaping: letters, digits, underscores.
// Any caller that interpolates into SQL MUST call this first.
function assertSafeSqlIdentifier(value: string, kind: string): void {
  if (!/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(value)) {
    throw new Error(`Unsafe SQL ${kind}: ${JSON.stringify(value)}`);
  }
}
// Validate-on-load so a typo in a constant fails fast, not at test runtime
for (const [name, value] of Object.entries({
  TEST_SCHEMA, TEST_TABLE, CLASSIFICATION_COLUMN, SEMANTIC_TYPE_COLUMN,
  CLASSIFICATION_VALUE, SEMANTIC_TYPE_VALUE,
})) {
  assertSafeSqlIdentifier(value, name);
}

// Get the Postgres port for the database's instance by reading the data source
// from the API. Avoids hardcoding offsets (PORT+3 vs PORT+4) which would break
// if discovery picks the "test" sample instance instead of "prod".
async function getInstancePgPort(env: TestEnv & { api: BytebaseApiClient }): Promise<string> {
  const instance = await env.api.getInstance(env.instance);
  const port = instance.dataSources?.[0]?.port;
  if (!port) {
    throw new Error(`Instance ${env.instance} has no data source port`);
  }
  return port;
}

// Execute SQL via psql over Unix socket on the sample Postgres instance.
// Used for DDL/DML setup and teardown — Bytebase's query API is read-only.
function execSql(dbName: string, port: string, sql: string): void {
  execFileSync("psql", [
    "-h", "/tmp",
    "-p", port,
    "-U", "bbsample",
    "-d", dbName,
    "-v", "ON_ERROR_STOP=1",
    "-c", sql,
  ], { stdio: "pipe" });
}

async function createMaskingTestData(env: TestEnv & { api: BytebaseApiClient }): Promise<MaskingTestData> {
  const dbName = env.databaseId;
  const port = await getInstancePgPort(env);

  // Drop anything leftover from a previous run, then create fresh schema/table/row.
  // All interpolated identifiers are validated constants (see assertSafeSqlIdentifier above).
  execSql(dbName, port, `DROP SCHEMA IF EXISTS ${TEST_SCHEMA} CASCADE`);
  execSql(dbName, port, `CREATE SCHEMA ${TEST_SCHEMA}`);
  execSql(dbName, port, `CREATE TABLE ${TEST_SCHEMA}.${TEST_TABLE} (
    id INTEGER PRIMARY KEY,
    ${CLASSIFICATION_COLUMN} TEXT NOT NULL,
    ${SEMANTIC_TYPE_COLUMN} TEXT NOT NULL
  )`);
  execSql(dbName, port, `INSERT INTO ${TEST_SCHEMA}.${TEST_TABLE} (id, ${CLASSIFICATION_COLUMN}, ${SEMANTIC_TYPE_COLUMN})
    VALUES (${ROW_PK}, '${CLASSIFICATION_VALUE}', '${SEMANTIC_TYPE_VALUE}')`);

  // Sync the database schema so Bytebase picks up the new table.
  // Errors here are real — without sync, the catalog update will target a
  // table Bytebase doesn't know about and silently drop the configuration.
  await env.api.syncDatabase(env.database);

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
    primaryKeyValue: String(ROW_PK),
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

async function dropMaskingTestData(env: TestEnv & { api: BytebaseApiClient }): Promise<void> {
  try {
    const port = await getInstancePgPort(env);
    execSql(env.databaseId, port, `DROP SCHEMA IF EXISTS ${TEST_SCHEMA} CASCADE`);
  } catch (err) {
    // Server may have been torn down by globalTeardown before afterAll ran.
    // Surface the error so genuine cleanup failures are visible.
    console.warn(`dropMaskingTestData: ${err instanceof Error ? err.message : err}`);
  }
}

async function grantExemption(description: string, member = `user:${env.adminEmail}`): Promise<void> {
  const existing = await env.api.getPolicy(`${env.project}/policies/masking_exemption`) as {
    maskingExemptionPolicy?: { exemptions: { members: string[]; condition?: { expression: string; description: string } }[] };
  } | null;
  const exemptions = existing?.maskingExemptionPolicy?.exemptions ?? [];
  exemptions.push({
    members: [member],
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
  projectId = env.project.split("/").pop()!;
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
  await revokeAllExemptions().catch((err) => {
    console.warn(`afterAll revokeAllExemptions: ${err instanceof Error ? err.message : err}`);
  });
  await dropMaskingTestData(env);
});

// ── Exemption List Page ──

test.describe("Exemption List Page", () => {
  test.beforeAll(async () => {
    await grantExemption("Test grant A");
    await grantExemption("Test grant B");
  });

  test("loads and displays member list with grant details", async () => {
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
    await exemptionPage.goto(projectId);

    await expect(exemptionPage.grantExemptionButton).toBeVisible();
    await expect(exemptionPage.activeTab).toBeVisible();
    await expect(exemptionPage.expiredTab).toBeVisible();
    await expect(exemptionPage.allTab).toBeVisible();
    await expect(page.getByText(env.adminEmail).first()).toBeVisible();
  });

  test("Active tab shows only active grants", async () => {
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
    await exemptionPage.goto(projectId);
    await exemptionPage.activeTab.click();
    await page.waitForTimeout(500);

    await expect(page.getByText("Never expires").first()).toBeVisible();
    await expect(page.getByText("(Expired)")).toHaveCount(0);
  });

  test("selecting a member shows their grants in detail panel", async () => {
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
    await exemptionPage.goto(projectId);
    await exemptionPage.selectMember(env.adminEmail);
    await page.waitForTimeout(500);

    await expect(page.getByText(/\d+ masking exemption/)).toBeVisible();
    await expect(page.getByText("Reason:").first()).toBeVisible();
  });

  test("grant card shows reason", async () => {
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
    await exemptionPage.goto(projectId);
    await exemptionPage.selectMember(env.adminEmail);
    await page.waitForTimeout(500);

    await expect(page.getByText("Reason:").first()).toBeVisible();
  });

  test("clicking All tab removes Active filter and shows all data", async () => {
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
    await exemptionPage.goto(projectId);

    await exemptionPage.allTab.click();
    await page.waitForTimeout(500);

    await expect(page.getByText("status: Active")).not.toBeVisible();
    await expect(page.getByText("status: Expired")).not.toBeVisible();
  });

  test("grant card has no excessive top padding", async () => {
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
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
    await grantExemption("SA test", `serviceAccount:${saEmail}`);
  });

  test.afterAll(async () => {
    await revokeAllExemptions();
    await env.api.deleteServiceAccount(saEmail);
  });

  test("service account badge renders on single line", async () => {
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
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
    const sqlEditor = new SqlEditorPage(page, env.baseURL);
    const grantPage = new GrantExemptionPage(page, env.baseURL);
    const listPage = new MaskingExemptionPage(page, env.baseURL);
    const sql = `SELECT "${target.sampleColumn}" FROM "${target.sampleSchema}"."${target.sampleTable}" WHERE "${target.primaryKeyColumn}" = '${target.primaryKeyValue}';`;
    const adminName = env.adminEmail === "demo@example.com" ? "Demo" : "Admin";

    // Step 1: No exemption → data is masked
    await revokeAllExemptions();
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await sqlEditor.runQuery(sql);
    expect(await sqlEditor.resultContainsText(target.knownUnmaskedValue)).toBe(false);

    // Step 2: Grant exemption via UI → data becomes unmasked
    await grantPage.goto(projectId);
    await grantPage.reasonInput.fill(reason);
    await grantPage.selectAccount(adminName);
    await grantPage.submit();
    await page.waitForTimeout(1000);

    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await sqlEditor.runQuery(sql);
    expect(await sqlEditor.resultContainsText(target.knownUnmaskedValue)).toBe(true);

    // Step 3: Revoke exemption via UI → data becomes masked again
    await listPage.goto(projectId);
    await listPage.selectMember(env.adminEmail);
    await page.waitForTimeout(500);
    await page.getByRole("button", { name: "Revoke" }).first().click();
    await page.getByRole("dialog").getByRole("button", { name: "Confirm" }).click();
    await page.waitForTimeout(500);

    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
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
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
    await exemptionPage.goto(projectId);

    await expect(page.getByText(/\d+ masking exemption/).first()).toBeVisible();
    await expect(page.getByText("Reason:").first()).toBeVisible();
  });

  test("narrow screen shows expandable list", async () => {
    await page.setViewportSize({ width: 768, height: 1024 });
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
    await exemptionPage.goto(projectId);

    await expect(page.getByText(env.adminEmail).first()).toBeVisible();
    await page.getByText(env.adminEmail).first().click();
    await page.waitForTimeout(300);
    await expect(page.getByText("Reason:").first()).toBeVisible();
  });
});
