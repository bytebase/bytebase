import { test, expect, type Page, type BrowserContext } from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";
import { MaskingExemptionPage, GrantExemptionPage, SqlEditorPage } from "./masking-exemption.page";

// Give all tests generous timeouts (Mode B's disposable server can be slow)
test.setTimeout(120_000);

interface MaskingTestData {
  sampleTable: string;
  sampleSchema: string;
  sampleColumn: string;
  primaryKeyColumn: string;
  primaryKeyValue: string;
  knownUnmaskedValue: string;
}

let env: TestEnv & { api: BytebaseApiClient };
let maskingData: MaskingTestData;

// Shared browser context and page — reused across ALL tests.
// Avoids browser open/close overhead and NVirtualList first-render issues.
let sharedContext: BrowserContext;
let page: Page;

// ── Feature-specific discovery ──

async function discoverMaskingData(env: TestEnv & { api: BytebaseApiClient }): Promise<MaskingTestData> {
  type Row = { values?: { stringValue?: string; int64Value?: string }[] };
  const getStr = (row: Row, idx: number) => row.values?.[idx]?.stringValue ?? row.values?.[idx]?.int64Value ?? "";
  const getRows = (result: { results: unknown[] }) =>
    ((result.results?.[0] as { rows?: Row[] })?.rows ?? []);

  const colResult = await env.api.query(
    env.database,
    `SELECT c.table_schema, c.table_name, c.column_name, pk.pk_column
     FROM information_schema.columns c
     JOIN (
       SELECT kcu.table_schema, kcu.table_name, kcu.column_name AS pk_column
       FROM information_schema.table_constraints tc
       JOIN information_schema.key_column_usage kcu
         ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
       WHERE tc.constraint_type = 'PRIMARY KEY' AND tc.table_schema = 'public'
     ) pk ON c.table_schema = pk.table_schema AND c.table_name = pk.table_name
     WHERE c.table_schema = 'public'
       AND c.data_type IN ('text', 'character varying')
       AND c.column_name NOT LIKE '%id%'
       AND c.column_name NOT LIKE '%date%'
       AND c.column_name NOT LIKE '%time%'
       AND c.column_name != pk.pk_column
     LIMIT 20`
  );
  const colRows = getRows(colResult);
  if (colRows.length === 0) {
    throw new Error(`Could not find a suitable text column in ${env.database}`);
  }

  for (const row of colRows) {
    const schema = getStr(row, 0);
    const table = getStr(row, 1);
    const column = getStr(row, 2);
    const pkColumn = getStr(row, 3);
    if (!table || !column || !pkColumn) continue;

    const dataResult = await env.api.query(
      env.database,
      `SELECT "${pkColumn}", "${column}" FROM "${schema}"."${table}" WHERE "${column}" IS NOT NULL AND "${column}" != '' LIMIT 1`
    );
    const dataRows = getRows(dataResult);
    const pkValue = getStr(dataRows[0], 0);
    const textValue = getStr(dataRows[0], 1);
    if (pkValue && textValue) {
      return {
        sampleTable: table, sampleSchema: schema, sampleColumn: column,
        primaryKeyColumn: pkColumn, primaryKeyValue: pkValue,
        knownUnmaskedValue: textValue,
      };
    }
  }

  throw new Error(`Could not find a column with data to mask in ${env.database}`);
}

async function configureMasking(env: TestEnv & { api: BytebaseApiClient }, data: MaskingTestData): Promise<void> {
  const catalog = {
    schemas: [{
      name: data.sampleSchema,
      tables: [{
        name: data.sampleTable,
        columns: {
          columns: [{
            name: data.sampleColumn,
            semanticType: "bb.default-partial",
          }],
        },
      }],
    }],
  };
  await env.api.updateCatalog(env.database, catalog);
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

  // Grant a temporary exemption to read TRUE unmasked values during discovery.
  // Demo data has `first_name` with classification-based masking, so without
  // an exemption the API returns masked values.
  await grantExemption("discovery temporary");
  maskingData = await discoverMaskingData(env);
  await revokeAllExemptions();

  await configureMasking(env, maskingData);

  // Create a shared browser context/page for all tests
  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
});

test.afterAll(async () => {
  await sharedContext?.close();
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
  test("masked → grant exemption → unmasked → revoke → masked", async () => {
    const projectId = env.project.split("/").pop()!;
    const instanceId = env.instance.split("/").pop()!;
    const dbId = env.database.split("/").pop()!;
    const sqlEditor = new SqlEditorPage(page, env.baseURL);
    const grantPage = new GrantExemptionPage(page, env.baseURL);
    const listPage = new MaskingExemptionPage(page, env.baseURL);
    const sql = `SELECT "${maskingData.sampleColumn}" FROM "${maskingData.sampleSchema}"."${maskingData.sampleTable}" WHERE "${maskingData.primaryKeyColumn}" = '${maskingData.primaryKeyValue}';`;
    const adminName = env.adminEmail === "demo@example.com" ? "Demo" : "Admin";

    // Step 1: No exemption → data is masked
    await revokeAllExemptions();
    await sqlEditor.gotoWithDb(projectId, instanceId, dbId);
    await sqlEditor.runQuery(sql);
    expect(await sqlEditor.resultContainsText(maskingData.knownUnmaskedValue)).toBe(false);

    // Step 2: Grant exemption via UI → data becomes unmasked
    await grantPage.goto(projectId);
    await grantPage.reasonInput.fill("e2e masking cycle test");
    await grantPage.selectAccount(adminName);
    await grantPage.submit();
    await page.waitForTimeout(1000);

    await sqlEditor.gotoWithDb(projectId, instanceId, dbId);
    await sqlEditor.runQuery(sql);
    expect(await sqlEditor.resultContainsText(maskingData.knownUnmaskedValue)).toBe(true);

    // Step 3: Revoke exemption via UI → data becomes masked again
    await listPage.goto(projectId);
    await listPage.selectMember(env.adminEmail);
    await page.waitForTimeout(500);
    await page.getByRole("button", { name: "Revoke" }).first().click();
    await page.getByRole("dialog").getByRole("button", { name: "Confirm" }).click();
    await page.waitForTimeout(500);

    await sqlEditor.gotoWithDb(projectId, instanceId, dbId);
    await sqlEditor.runQuery(sql);
    expect(await sqlEditor.resultContainsText(maskingData.knownUnmaskedValue)).toBe(false);
  });
});

// ── Responsive Layout ──

test.describe("Responsive Layout", () => {
  test.beforeAll(async () => {
    await grantExemption("Layout test grant");
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
