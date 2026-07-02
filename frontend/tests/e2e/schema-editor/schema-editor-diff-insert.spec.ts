// Schema editor — DDL generation + insertion (SchemaEditorSheet +
// generateDiffDDL + PreviewPane) and the sheet chrome.
//
// Covers:
//   - The full create-table CUJ: build a table + column in the editor, click
//     "Insert SQL", and confirm the generated DDL lands in the plan statement.
//   - The no-diff path: inserting with no edits shows "No changes to apply"
//     and leaves the statement untouched.
//   - Maximize / restore toggles the sheet size.

import {
  test,
  expect,
  type Page,
  type BrowserContext,
} from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";
import { SchemaEditorPage } from "./schema-editor.page";
import { createSchemaEditorPlan } from "./schema-editor-helpers";

test.setTimeout(120_000);

let env: TestEnv & { api: BytebaseApiClient };
let ctx: BrowserContext;
let page: Page;
let se: SchemaEditorPage;
let projectId: string;
let planId: string;

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  ({ projectId, planId } = await createSchemaEditorPlan(env, "E2E SchemaEditor DiffInsert"));
  ctx = await browser.newContext({ storageState: ".auth/state.json" });
  page = await ctx.newPage();
  se = new SchemaEditorPage(page, env.baseURL);
});

test.afterAll(async () => {
  await ctx?.close();
});

test.describe("generate and insert DDL", () => {
  test.describe.configure({ mode: "serial" });

  test.beforeEach(async () => {
    await se.gotoPlan(projectId, planId);
    await se.open();
  });

  test("inserts the generated CREATE TABLE into the plan statement", async () => {
    await se.selectSchema("public");
    const name = `e2e_ins_${Date.now()}`;
    await se.createTable(name);
    await se.addColumn();
    await se.columnNameInputs().last().fill("label");
    await se.selectColumnType(1, "text"); // reads metadata on insert, not preview

    await se.insertSql();

    // The generated DDL is written into the plan's statement editor. Insert
    // reads the edited metadata (not the preview), so the type is present.
    await expect
      .poll(async () => await se.planStatementText())
      .toContain("CREATE TABLE");
    const stmt = await se.planStatementText();
    expect(stmt).toContain(name);
    expect(stmt).toContain("label");
    expect(stmt).toContain("text");
  });

  test("shows 'No changes to apply' when inserting with no edits", async () => {
    // No edits made — the diff is empty.
    await se.insertSqlButton.click();
    await expect(page.getByText("No changes to apply")).toBeVisible({
      timeout: 10_000,
    });
    // The sheet stays open so the user can keep editing.
    await expect(se.sheet).toBeVisible();
  });

  test("maximize and restore toggles the sheet width", async () => {
    const normal = await se.sheet.boundingBox();
    await se.maximizeButton.click();
    await expect(se.restoreButton).toBeVisible();
    const maximized = await se.sheet.boundingBox();
    expect(maximized!.width).toBeGreaterThan(normal!.width);

    await se.restoreButton.click();
    await expect(se.maximizeButton).toBeVisible();
    const restored = await se.sheet.boundingBox();
    expect(restored!.width).toBeLessThan(maximized!.width);
  });
});
