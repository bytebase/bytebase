// Schema editor — schema-level objects (AsideTree "+" DropdownMenu +
// SchemaNameDialog).
//
// Scope notes:
// - Views / functions / procedures and the indexes / partitions tabs are
//   engine-gated to MySQL/TiDB (core/spec.ts) and are NOT shown on the Postgres
//   target used here — they need a MySQL target, covered separately.
// - Rename / drop table / drop schema live behind the tree's *hand-rolled*
//   right-click context menu (a portaled <div> of <button>s, not a Base UI
//   Menu). That menu opens under Playwright (a dispatched `contextmenu`), but
//   its portaled items don't interact reliably (stability / click no-op),
//   unlike the Base UI "+" DropdownMenu used below which drives cleanly. Those
//   operations are left for a follow-up — ideally after the context menu is
//   migrated to the shared Base UI DropdownMenu (or given data-testids) for
//   parity and testability.
// - Create-schema IS Postgres-valid (engineSupportsMultiSchema(POSTGRES)).

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
  ({ projectId, planId } = await createSchemaEditorPlan(env, "E2E SchemaEditor Objects"));
  ctx = await browser.newContext({ storageState: ".auth/state.json" });
  page = await ctx.newPage();
  se = new SchemaEditorPage(page, env.baseURL);
});

test.afterAll(async () => {
  await ctx?.close();
});

test.describe("schema operations", () => {
  test.describe.configure({ mode: "serial" });

  test.beforeEach(async () => {
    await se.gotoPlan(projectId, planId);
    await se.open();
  });

  test.afterEach(async () => {
    await se.close();
  });

  test("creates a new schema from the + menu", async () => {
    const name = `e2e_sch_${Date.now()}`;
    await se.createSchema(name);
    // The new schema appears as a tree node.
    await expect(se.treeItem(name)).toBeVisible({ timeout: 10_000 });
  });

  test("rejects a duplicate schema name", async () => {
    await se.createMenuTrigger.click();
    await se.createMenuItem("Create schema").click();
    await expect(se.schemaNameInput).toBeVisible();
    await se.schemaNameInput.fill("public"); // already exists
    await expect(page.getByText("Schema name already exists")).toBeVisible();
    await expect(
      se.schemaDialog.getByRole("button", { name: "Create", exact: true })
    ).toBeDisabled();
    await se.schemaDialog.getByRole("button", { name: "Cancel" }).click();
  });
});
