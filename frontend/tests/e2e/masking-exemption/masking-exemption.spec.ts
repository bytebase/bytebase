import { test, expect, type Page, type BrowserContext } from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";
import { getInstancePgPort, execSql } from "../framework/psql";
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
// Built-in semantic-type ID used in the previous demo dump. The new
// bootstrap doesn't seed it, so we register it ourselves via the
// SEMANTIC_TYPES workspace setting before applying it via catalog.
const SEMANTIC_TYPE_ID = "bb.default-partial";
// Classification-masking fixtures (the new bootstrap doesn't pre-seed them).
const CLASSIFICATION_CONFIG_ID = "e2e-classification-config";
const CLASSIFICATION_LEVEL_NUM = 2; // numeric level CLASSIFICATION_LEVEL ("1-2") maps to
const CLASSIFICATION_RULE_ID = "e2e00000-0000-4000-8000-000000000001"; // UUID for the workspace masking rule

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

// Register a semantic type via the SEMANTIC_TYPES workspace setting so
// the catalog update below can reference it. Without this, applying a
// column with semanticType="bb.default-partial" fails with "semantic
// type id not found" — the previous demo dump pre-seeded the type but
// the new bootstrap doesn't.
async function ensureSemanticType(env: TestEnv & { api: BytebaseApiClient }): Promise<void> {
  await env.api.upsertSetting(
    "SEMANTIC_TYPES",
    {
      semanticType: {
        types: [
          {
            id: SEMANTIC_TYPE_ID,
            title: "E2E Partial Mask",
            description: "Used by masking-exemption e2e tests",
            algorithm: {
              rangeMask: {
                slices: [{ start: 0, end: 3, substitution: "***" }],
              },
            },
          },
        ],
      },
    },
    "value.semantic_type",
  );
}

// Seed the workspace fixtures the classification -> masking path needs (the
// post-#20393 bootstrap no longer pre-seeds them, unlike the old demo dump):
//   1. a DATA_CLASSIFICATION config mapping classification id CLASSIFICATION_LEVEL
//      ("1-2") to a numeric level,
//   2. the test project pointing at that config (projects carry a
//      dataClassificationConfigId; the masking evaluator resolves the config
//      from it, with no fallback), and
//   3. a workspace MASKING_RULE whose CEL `classification_level == N` applies
//      the semantic type to columns at that level.
async function ensureClassificationFixtures(
  env: TestEnv & { api: BytebaseApiClient },
): Promise<void> {
  await env.api.upsertSetting(
    "DATA_CLASSIFICATION",
    {
      dataClassification: {
        configs: [
          {
            id: CLASSIFICATION_CONFIG_ID,
            title: "E2E Classification",
            levels: [
              { title: "Level 1", level: 1 },
              { title: "Level 2", level: 2 },
            ],
            classification: {
              [CLASSIFICATION_LEVEL]: {
                id: CLASSIFICATION_LEVEL,
                title: "E2E Internal",
                level: CLASSIFICATION_LEVEL_NUM,
              },
            },
          },
        ],
      },
    },
    "value.data_classification",
  );
  // Link the project to the config (it must exist first — UpdateProject validates it).
  await env.api.updateProjectSettings(env.project, {
    dataClassificationConfigId: CLASSIFICATION_CONFIG_ID,
  });
  // Workspace masking rule: any column at the level gets the semantic type.
  const { workspace } = await env.api.getActuatorInfo();
  await env.api.upsertPolicy(workspace, "masking_rule", {
    type: "MASKING_RULE",
    maskingRulePolicy: {
      rules: [
        {
          id: CLASSIFICATION_RULE_ID,
          condition: {
            expression: `resource.classification_level == ${CLASSIFICATION_LEVEL_NUM}`,
            title: "e2e classification masking",
          },
          semanticType: SEMANTIC_TYPE_ID,
        },
      ],
    },
  });
}

// Remove the workspace masking rule so it can't affect later specs. (The
// project's config link and the DATA_CLASSIFICATION setting are harmless once
// the rule is gone — no column outside this spec carries the classification —
// and the disposable server is torn down anyway.)
async function clearClassificationFixtures(
  env: TestEnv & { api: BytebaseApiClient },
): Promise<void> {
  try {
    const { workspace } = await env.api.getActuatorInfo();
    await env.api.upsertPolicy(workspace, "masking_rule", {
      type: "MASKING_RULE",
      maskingRulePolicy: { rules: [] },
    });
  } catch (err) {
    console.warn(
      `clearClassificationFixtures: ${err instanceof Error ? err.message : err}`,
    );
  }
}

async function createMaskingTestData(env: TestEnv & { api: BytebaseApiClient }): Promise<MaskingTestData> {
  const dbName = env.databaseId;
  const port = await getInstancePgPort(env);

  await ensureSemanticType(env);
  await ensureClassificationFixtures(env);

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
            { name: SEMANTIC_TYPE_COLUMN, semanticType: SEMANTIC_TYPE_ID },
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
  // Masking and classification are enterprise-gated; the license is installed at bootstrap (required to run this suite).
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
  await clearClassificationFixtures(env);
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

    // Use a stable data-testid locator (not CSS class substrings, which break
    // on theme refactors). Fail loudly if the card isn't found rather than
    // silently passing — the previous version of this test was a false
    // positive because its class selector matched nothing.
    const grantCard = page.getByTestId("exemption-grant-card").first();
    await expect(grantCard).toBeVisible();

    const cardBox = await grantCard.boundingBox();
    const header = grantCard.getByTestId("exemption-grant-header").first();
    await expect(header).toBeVisible();
    const headerBox = await header.boundingBox();
    expect(cardBox, "grant card boundingBox should be measurable").toBeTruthy();
    expect(headerBox, "grant card header boundingBox should be measurable").toBeTruthy();

    // Expected layout: 1px top border + header's own py-2 (8px) = ~9px gap.
    // The original bug added pt-4 (16px) on the wrapper, pushing the gap to ~25px.
    // Allow 0-14px range: tolerates border + minor padding, rejects pt-4 regression.
    const topGap = headerBox!.y - cardBox!.y;
    expect(topGap, `header should sit near top of card (got ${topGap}px)`).toBeGreaterThanOrEqual(0);
    expect(topGap, `header should sit near top of card (got ${topGap}px)`).toBeLessThanOrEqual(14);
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

  // Classification-based masking: col_classification carries classification id
  // "1-2" (set in the catalog above); ensureClassificationFixtures seeds the
  // DATA_CLASSIFICATION config that maps "1-2" -> level 2, links the project to
  // that config, and installs a workspace MASKING_RULE
  // (`classification_level == 2` -> the semantic type). So this column masks
  // via the classification -> rule path, distinct from col_semantic's direct
  // semantic_type.
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
