// Schema editor — column editing (TableColumnEditor + DataTypeCell +
// DefaultValueCell).
//
// Covers:
//   - Add column; edit column name (backspace works — BYT-9802 sibling lock).
//   - Select a column type from the dropdown (BYT-9802 lock — a potential
//     customer reported "couldn't select the column type from the editor").
//   - Type a custom free-text type.
//   - Primary is disabled on an existing table (PK only editable on created).
//   - Drop a created column (spliced out) vs an existing column (excluded from
//     the generated DDL) and restore it.
//   - Default-value editing reflected in the generated DDL.
//
// Bug locks (Layer 2) — one root cause, two symptoms: on a NEWLY-CREATED table,
// column edits routed through refreshStatus() never re-render the editor
// because refreshTableEditStatus short-circuits for `created` tables and never
// bumps editStatus.version (refreshEditStatus.ts:95). Symptoms:
//   1. The DDL Preview stays frozen at the table's creation state.
//   2. The Primary / Not-Null checkboxes don't reflect a toggle (and PK is only
//      editable on created tables, so the PK toggle is effectively unusable).

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
  ({ projectId, planId } = await createSchemaEditorPlan(env, "E2E SchemaEditor Columns"));
  ctx = await browser.newContext({ storageState: ".auth/state.json" });
  page = await ctx.newPage();
  se = new SchemaEditorPage(page, env.baseURL);
});

test.afterAll(async () => {
  await ctx?.close();
});

// Create a uniquely-named table (opens in a tab with a default `id` column).
async function newTable(): Promise<string> {
  const name = `e2e_col_${Date.now()}_${Math.floor(Math.random() * 1e4)}`;
  await se.createTable(name);
  return name;
}

test.describe("column editing", () => {
  test.describe.configure({ mode: "serial" });

  test.beforeEach(async () => {
    await se.gotoPlan(projectId, planId);
    await se.open();
    await se.selectSchema("public");
  });

  test.afterEach(async () => {
    await se.close();
  });

  test("adds a new empty column row", async () => {
    await newTable();
    const before = await se.columnNameInputs().count();
    await se.addColumn();
    expect(await se.columnNameInputs().count()).toBe(before + 1);
  });

  test("selects a column type from the dropdown (BYT-9802)", async () => {
    await newTable();
    await se.addColumn();
    // The dropdown must open on click AND the selection must land in the cell.
    await se.selectColumnType(1, "boolean");
    await expect(se.columnTypeInputs().nth(1)).toHaveValue("boolean");
  });

  test("backspace deletes characters in the column-name input (BYT-9802)", async () => {
    await newTable();
    await se.addColumn();
    const nameInput = se.columnNameInputs().last();
    await nameInput.click();
    await page.keyboard.type("widget_label");
    await expect(nameInput).toHaveValue("widget_label");
    for (let i = 0; i < 5; i++) await page.keyboard.press("Backspace");
    await expect(nameInput).toHaveValue("widget_");
  });

  test("accepts a custom free-text column type", async () => {
    await newTable();
    await se.addColumn();
    const typeInput = se.columnTypeInputs().last();
    await typeInput.fill("varchar(255)");
    await expect(typeInput).toHaveValue("varchar(255)");
  });

  test("checking Primary forces Not-Null on a created table", async () => {
    await newTable();
    await se.addColumn();
    await se.columnNameInputs().last().fill("zcode");
    await se.selectColumnType(1, "text");

    const primary = se.primaryCheckbox("zcode");
    const notNull = se.notNullCheckbox("zcode");
    await expect(notNull).not.toBeChecked();
    // Use click (not check) — Playwright's check() mis-handles Base UI's
    // span[role=checkbox] and reports "state did not change" spuriously.
    await primary.click();
    await expect(primary).toBeChecked();
    // PK implies NOT NULL, and the Not-Null box locks on.
    await expect(notNull).toBeChecked();
    await expect(notNull).toBeDisabled();
  });

  test("Primary is disabled for every column on an existing table", async () => {
    await se.openTableFromList("department");
    // PK can only be changed on newly-created tables.
    await expect(se.primaryCheckbox("dept_no")).toBeDisabled();
    await expect(se.primaryCheckbox("dept_name")).toBeDisabled();
  });

  test("dropping a created column removes its row", async () => {
    await newTable();
    await se.addColumn();
    const nameInput = se.columnNameInputs().last();
    await nameInput.fill("zdrop");
    const before = await se.columnNameInputs().count();
    await se.dropColumnButton("zdrop").click();
    // A created column is spliced out entirely.
    await expect(se.columnNameInputs()).toHaveCount(before - 1);
  });

  test("dropping then restoring an existing column round-trips in the DDL", async () => {
    await se.openTableFromList("department");
    // Existing-table edits DO refresh the preview, so the DDL is the oracle.
    // Match the column *definition* (`"dept_name" text`) — a leftover UNIQUE
    // constraint keeps the bare name in the DDL even after the column is gone.
    // Poll for the initial (async) preview load before asserting.
    await expect
      .poll(async () => await se.previewText())
      .toContain('"dept_name" text');

    await se.dropColumnButton("dept_name").click();
    await expect
      .poll(async () => await se.previewText())
      .not.toContain('"dept_name" text');

    // The operations cell now shows Restore; click it to bring the column back.
    await se.dropColumnButton("dept_name").click();
    await expect
      .poll(async () => await se.previewText())
      .toContain('"dept_name" text');
  });

  test("editing a default value is reflected in the generated DDL", async () => {
    // Use an existing table so the live preview is the oracle (created-table
    // previews are stale — see the bug locks below).
    await se.openTableFromList("department");
    const defaultInput = se
      .columnRow("dept_name")
      .getByRole("textbox", { name: /Enter default value/ });
    await defaultInput.fill("n/a");
    await expect
      .poll(async () => await se.previewText())
      .toContain("DEFAULT");
  });
});

// Root cause: refreshTableEditStatus early-returns for `created` tables
// (refreshEditStatus.ts:95), so a column-value edit routed through
// refreshStatus() never bumps editStatus.version — and PreviewPane's
// mockedMetadata memo (keyed on editStatus.version) never recomputes, so the
// DDL Preview stays frozen at the table's creation state. Existing-table edits
// DO refresh the preview (they mark the column "updated" → version bumps).
test.describe("created-table edit reactivity (BUG BYT-9802-preview)", () => {
  test.describe.configure({ mode: "serial" });

  test.beforeEach(async () => {
    await se.gotoPlan(projectId, planId);
    await se.open();
    await se.selectSchema("public");
  });

  test.afterEach(async () => {
    await se.close();
  });

  // Repro: create table (id integer) → change id's type → preview still shows
  // "integer" even though the grid cell (and the inserted SQL) reflect "bigint".
  test.fail("changing a column type updates the preview DDL", async () => {
    await newTable();
    // Wait for the initial (async) preview to load, so the failure below is on
    // the stale type, not a not-yet-rendered preview.
    await expect.poll(async () => await se.previewText()).toContain("integer");

    await se.selectColumnType(0, "bigint");
    await expect(se.columnTypeInputs().nth(0)).toHaveValue("bigint"); // control

    // Bug: the preview should reflect the new type but stays "integer".
    expect(await se.previewText()).toContain("bigint");
  });
});
