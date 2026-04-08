import { test, expect } from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";
import { createSnapshot, restoreSnapshot, type Snapshot } from "../framework/snapshot";
import { MaskingExemptionPage, GrantExemptionPage, SqlEditorPage } from "./masking-exemption.page";

interface MaskingTestData {
  sampleTable: string;
  sampleSchema: string;
  sampleColumn: string;
  knownUnmaskedValue: string;
}

let env: TestEnv & { api: BytebaseApiClient };
let maskingData: MaskingTestData;
let snapshot: Snapshot | undefined;

// ── Feature-specific discovery ──

async function discoverMaskingData(env: TestEnv & { api: BytebaseApiClient }): Promise<MaskingTestData> {
  // Use SQL to discover tables and text columns (catalog API only returns configured metadata)
  const colResult = await env.api.query(
    env.database,
    `SELECT table_schema, table_name, column_name FROM information_schema.columns
     WHERE table_schema = 'public'
       AND data_type IN ('text', 'character varying')
       AND column_name NOT LIKE '%id%'
       AND column_name NOT LIKE '%date%'
       AND column_name NOT LIKE '%time%'
     LIMIT 10`
  );
  const colRows = (colResult.results?.[0] as {
    rows?: { values?: { stringValue?: string }[] }[];
  })?.rows ?? [];

  if (colRows.length === 0) {
    throw new Error(`Could not find a suitable text column to mask in ${env.database}`);
  }

  const sampleSchema = colRows[0].values?.[0]?.stringValue ?? "public";
  const sampleTable = colRows[0].values?.[1]?.stringValue ?? "";
  const sampleColumn = colRows[0].values?.[2]?.stringValue ?? "";

  if (!sampleTable || !sampleColumn) {
    throw new Error(`Could not find a suitable column to mask in ${env.database}`);
  }

  // Query for a known value
  const valueResult = await env.api.query(
    env.database,
    `SELECT "${sampleColumn}" FROM "${sampleSchema}"."${sampleTable}" WHERE "${sampleColumn}" IS NOT NULL LIMIT 1`
  );
  const firstResult = valueResult.results?.[0] as {
    rows?: { values?: { stringValue?: string }[] }[];
  };
  const knownUnmaskedValue = firstResult?.rows?.[0]?.values?.[0]?.stringValue ?? "";
  if (!knownUnmaskedValue) {
    throw new Error(`Could not get a known value from ${sampleSchema}.${sampleTable}.${sampleColumn}`);
  }

  return { sampleTable, sampleSchema, sampleColumn, knownUnmaskedValue };
}

async function configureMasking(env: TestEnv & { api: BytebaseApiClient }, data: MaskingTestData): Promise<void> {
  // Construct catalog payload to set masking on the discovered column
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

test.beforeAll(async () => {
  env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword!);
  maskingData = await discoverMaskingData(env);
  await configureMasking(env, maskingData);

  if (env.mode === "local") {
    snapshot = await createSnapshot(env.api, {
      policies: [`${env.project}/policies/masking_exemption`],
      catalogs: [env.database],
    });
  }
});

test.afterAll(async () => {
  if (snapshot) await restoreSnapshot(env.api, snapshot);
});

// ── Tests from masking-exemption-list.spec.ts ──

test.describe("Exemption List Page", () => {
  test.beforeAll(async () => {
    await grantExemption("Test grant A");
    await grantExemption("Test grant B");
  });

  test("loads and displays member list with grant details", async ({ page }) => {
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
    const projectId = env.project.split("/").pop()!;
    await exemptionPage.goto(projectId);

    await expect(exemptionPage.grantExemptionButton).toBeVisible();
    await expect(exemptionPage.activeTab).toBeVisible();
    await expect(exemptionPage.expiredTab).toBeVisible();
    await expect(exemptionPage.allTab).toBeVisible();
    await expect(page.getByText(env.adminEmail).first()).toBeVisible();
  });

  test("Active tab shows only active grants", async ({ page }) => {
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
    const projectId = env.project.split("/").pop()!;
    await exemptionPage.goto(projectId);
    await exemptionPage.activeTab.click();
    await page.waitForTimeout(500);

    await expect(page.getByText("Never expires").first()).toBeVisible();
    await expect(page.getByText("(Expired)")).toHaveCount(0);
  });

  test("selecting a member shows their grants in detail panel", async ({ page }) => {
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
    const projectId = env.project.split("/").pop()!;
    await exemptionPage.goto(projectId);
    await exemptionPage.selectMember(env.adminEmail);
    await page.waitForTimeout(500);

    await expect(page.getByText(/\d+ masking exemption/)).toBeVisible();
    await expect(page.getByText("Test grant A")).toBeVisible();
  });

  test("grant card shows reason", async ({ page }) => {
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
    const projectId = env.project.split("/").pop()!;
    await exemptionPage.goto(projectId);
    await exemptionPage.selectMember(env.adminEmail);
    await page.waitForTimeout(500);

    await expect(page.getByText("Reason:").first()).toBeVisible();
  });
});

// ── Tests from masking-exemption-grant-revoke.spec.ts ──

test.describe("Grant and Revoke", () => {
  test.beforeEach(async () => {
    await revokeAllExemptions();
  });

  test("grant exemption via UI and verify it appears in list", async ({ page }) => {
    const projectId = env.project.split("/").pop()!;
    const grantPage = new GrantExemptionPage(page, env.baseURL);
    const listPage = new MaskingExemptionPage(page, env.baseURL);

    await grantPage.goto(projectId);
    await expect(grantPage.allRadio).toBeChecked();
    await expect(grantPage.confirmButton).toBeDisabled();

    await grantPage.reasonInput.fill("E2E test grant");
    await grantPage.selectAccount("Admin");
    await expect(grantPage.confirmButton).toBeEnabled();
    await grantPage.submit();
    await page.waitForTimeout(1000);

    await listPage.goto(projectId);
    await listPage.selectMember(env.adminEmail);
    await page.waitForTimeout(500);
    await expect(page.getByText("E2E test grant")).toBeVisible();
  });

  test("revoke exemption via UI and verify it disappears", async ({ page }) => {
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

  test("revoke confirmation can be cancelled", async ({ page }) => {
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

// ── Tests from masking-exemption-e2e-masking.spec.ts ──

test.describe("E2E Masking Verification", () => {
  test.setTimeout(120_000);

  test("grant → unmasked, revoke → masked, re-grant → unmasked", async ({ page }) => {
    const projectId = env.project.split("/").pop()!;
    const instanceId = env.instance.split("/").pop()!;
    const dbId = env.database.split("/").pop()!;
    const sqlEditor = new SqlEditorPage(page, env.baseURL);
    const sql = `SELECT "${maskingData.sampleColumn}" FROM "${maskingData.sampleSchema}"."${maskingData.sampleTable}" LIMIT 5;`;

    // Step 1: Grant → unmasked
    await grantExemption("e2e test exemption");
    await sqlEditor.gotoWithDb(projectId, instanceId, dbId);
    await sqlEditor.runQuery(sql);
    expect(await sqlEditor.resultContainsText(maskingData.knownUnmaskedValue)).toBe(true);

    // Step 2: Revoke → masked
    await revokeAllExemptions();
    await sqlEditor.gotoWithDb(projectId, instanceId, dbId);
    await sqlEditor.runQuery(sql);
    expect(await sqlEditor.resultContainsText(maskingData.knownUnmaskedValue)).toBe(false);

    // Step 3: Re-grant → unmasked
    await grantExemption("e2e test re-grant");
    await sqlEditor.gotoWithDb(projectId, instanceId, dbId);
    await sqlEditor.runQuery(sql);
    expect(await sqlEditor.resultContainsText(maskingData.knownUnmaskedValue)).toBe(true);
  });

  test("revoke via UI and verify data becomes masked", async ({ page }) => {
    const projectId = env.project.split("/").pop()!;
    const instanceId = env.instance.split("/").pop()!;
    const dbId = env.database.split("/").pop()!;
    const listPage = new MaskingExemptionPage(page, env.baseURL);
    const sqlEditor = new SqlEditorPage(page, env.baseURL);

    await grantExemption("e2e UI revoke test");
    const sql = `SELECT "${maskingData.sampleColumn}" FROM "${maskingData.sampleSchema}"."${maskingData.sampleTable}" LIMIT 5;`;

    await sqlEditor.gotoWithDb(projectId, instanceId, dbId);
    await sqlEditor.runQuery(sql);
    expect(await sqlEditor.resultContainsText(maskingData.knownUnmaskedValue)).toBe(true);

    await listPage.goto(projectId);
    await listPage.selectMember(env.adminEmail);
    await page.waitForTimeout(500);
    const revokeBtn = page.getByRole("button", { name: "Revoke" }).first();
    await revokeBtn.click();
    await page.getByRole("dialog").getByRole("button", { name: "Confirm" }).click();
    await page.waitForTimeout(500);

    await sqlEditor.gotoWithDb(projectId, instanceId, dbId);
    await sqlEditor.runQuery(sql);
    expect(await sqlEditor.resultContainsText(maskingData.knownUnmaskedValue)).toBe(false);

    await grantExemption("cleanup");
  });

  test("grant via UI and verify data becomes unmasked", async ({ page }) => {
    const projectId = env.project.split("/").pop()!;
    const instanceId = env.instance.split("/").pop()!;
    const dbId = env.database.split("/").pop()!;
    const grantPage = new GrantExemptionPage(page, env.baseURL);
    const sqlEditor = new SqlEditorPage(page, env.baseURL);

    await revokeAllExemptions();
    const sql = `SELECT "${maskingData.sampleColumn}" FROM "${maskingData.sampleSchema}"."${maskingData.sampleTable}" LIMIT 5;`;

    await sqlEditor.gotoWithDb(projectId, instanceId, dbId);
    await sqlEditor.runQuery(sql);
    expect(await sqlEditor.resultContainsText(maskingData.knownUnmaskedValue)).toBe(false);

    await grantPage.goto(projectId);
    await grantPage.reasonInput.fill("e2e UI grant test");
    await grantPage.selectAccount("Admin");
    await grantPage.submit();
    await page.waitForTimeout(1000);

    await sqlEditor.gotoWithDb(projectId, instanceId, dbId);
    await sqlEditor.runQuery(sql);
    expect(await sqlEditor.resultContainsText(maskingData.knownUnmaskedValue)).toBe(true);
  });
});

// ── Responsive Layout ──

test.describe("Responsive Layout", () => {
  test.beforeAll(async () => {
    await grantExemption("Layout test grant");
  });

  test("wide screen shows split-panel layout", async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    const projectId = env.project.split("/").pop()!;
    const exemptionPage = new MaskingExemptionPage(page, env.baseURL);
    await exemptionPage.goto(projectId);

    await expect(page.getByText(/\d+ masking exemption/).first()).toBeVisible();
    await expect(page.getByText("Reason:").first()).toBeVisible();
  });

  test("narrow screen shows expandable list", async ({ page }) => {
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
