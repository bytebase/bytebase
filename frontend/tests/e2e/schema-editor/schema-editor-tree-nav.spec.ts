// Schema editor — tree navigation + table-level operations (AsideTree +
// TabsContainer + TableList).
//
// Covers:
//   - Opening multiple tables from the schema's table list navigates via tabs.
//   - Dropping an existing table from the list produces a DROP TABLE in the
//     generated statement.

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
  ({ projectId, planId } = await createSchemaEditorPlan(env, "E2E SchemaEditor TreeNav"));
  ctx = await browser.newContext({ storageState: ".auth/state.json" });
  page = await ctx.newPage();
  se = new SchemaEditorPage(page, env.baseURL);
});

test.afterAll(async () => {
  await ctx?.close();
});

test.describe("tree navigation and table operations", () => {
  test.describe.configure({ mode: "serial" });

  test.beforeEach(async () => {
    await se.gotoPlan(projectId, planId);
    await se.open();
    await se.selectSchema("public");
  });

  test.afterEach(async () => {
    await se.close();
  });

  test("navigating between objects swaps the editor content", async () => {
    // Open an existing table; its columns render.
    await se.openTableFromList("department");
    await expect(se.columnRow("dept_no")).toBeVisible();

    // Navigate to a freshly-created table; the editor swaps to its columns and
    // the previous table's columns are no longer mounted.
    await se.selectSchema("public");
    const name = `e2e_nav_${Date.now()}`;
    await se.createTable(name);
    await expect(se.columnRow("id")).toBeVisible();
    await expect(se.columnRow("dept_no")).toHaveCount(0);
  });

  test("dropping a table from the list produces a DROP TABLE statement", async () => {
    await se.dropTableButton("salary").click();
    await se.insertSql();

    await expect
      .poll(async () => await se.planStatementText())
      .toContain("DROP TABLE");
    expect(await se.planStatementText()).toContain("salary");
  });
});
