// Schema editor — create-table flow (DatabaseEditor + TableNamePopover).
//
// Covers:
//   - Creating a table from the "New table" toolbar button seeds a default
//     `id` primary-key column and opens it in a tab.
//   - Backspace works in the table-name input (BYT-9802 regression lock — a
//     potential customer reported the backspace key "never worked").
//   - Duplicate table-name validation blocks Create.
//   - Cancel closes the popover without creating a table.

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

  ({ projectId, planId } = await createSchemaEditorPlan(env, "E2E SchemaEditor CreateTable"));

  ctx = await browser.newContext({ storageState: ".auth/state.json" });
  page = await ctx.newPage();
  se = new SchemaEditorPage(page, env.baseURL);
  await se.gotoPlan(projectId, planId);
});

test.afterAll(async () => {
  await ctx?.close();
});

test.describe("create table", () => {
  test.describe.configure({ mode: "serial" });

  test.beforeEach(async () => {
    await se.gotoPlan(projectId, planId);
    await se.open();
    await se.selectSchema("public");
  });

  test.afterEach(async () => {
    await se.close();
  });

  test("creates a table with a default id primary-key column", async () => {
    const name = `e2e_widget_${Date.now()}`;
    await se.createTable(name);

    // The new table opens with its default `id` column in the grid.
    await expect(se.columnRow("id")).toBeVisible();

    // The generated DDL reflects the new table with an integer id PK. (Tab
    // labels are visually truncated, so the DDL is the stable oracle.) Poll —
    // the preview DDL is produced by an async getSchemaString RPC.
    await expect.poll(async () => await se.previewText()).toContain("CREATE TABLE");
    const preview = await se.previewText();
    expect(preview).toContain(name);
    expect(preview).toContain("PRIMARY KEY");
    expect(preview).toContain("id");
  });

  test("backspace deletes characters in the table-name input (BYT-9802)", async () => {
    await se.openNewTablePopover();
    // Type via the keyboard (not fill) so the Backspace key is exercised.
    await se.tableNameInput.click();
    await page.keyboard.type("abcdef");
    await expect(se.tableNameInput).toHaveValue("abcdef");
    await page.keyboard.press("Backspace");
    await page.keyboard.press("Backspace");
    await page.keyboard.press("Backspace");
    await expect(se.tableNameInput).toHaveValue("abc");
    await se.popoverCancelButton.click();
  });

  test("rejects a duplicate table name", async () => {
    await se.openNewTablePopover();
    await se.tableNameInput.fill("audit"); // pre-existing table in hr_test.public
    await expect(se.duplicateNameError).toBeVisible();
    await expect(se.popoverCreateButton).toBeDisabled();
    await se.popoverCancelButton.click();
  });

  test("cancel closes the popover without creating a table", async () => {
    const name = `e2e_cancel_${Date.now()}`;
    await se.openNewTablePopover();
    await se.tableNameInput.fill(name);
    await se.popoverCancelButton.click();
    await expect(se.tableNameInput).toBeHidden();
    // No table was created: we stay in the DatabaseEditor (its "New table"
    // toolbar button is shown), rather than switching to a new table tab
    // (which would show "Add column" instead).
    await expect(se.newTableToolbarButton).toBeVisible();
    await expect(se.addColumnButton).toBeHidden();
  });
});
