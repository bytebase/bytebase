// The project GitOps setup page: one page that turns a few choices into the
// files to add to a repository. The suite creates its own database group,
// workload identity and schema, and walks the state-based setup, which exports
// that schema as files.
import { expect, test } from "@playwright/test";
import type { BytebaseApiClient } from "../framework/api-client";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { execSql, getInstancePgPort } from "../framework/psql";

let env: TestEnv & { api: BytebaseApiClient };
let projectId: string;
const suffix = Date.now().toString(36);
const GROUP_TITLE = `E2E setup group ${suffix}`;
const WI_TITLE = `E2E setup CI ${suffix}`;
// The schema the export is asserted on. The suffix is base 36, so the name is
// a safe SQL identifier.
const SCHEMA = `e2e_setup_${suffix}`;
const SCHEMA_FOLDER = `schemas/${SCHEMA}/tables/`;
const created: { group?: string; workloadIdentity?: string } = {};
const SHOTS = process.env.BYTEBASE_E2E_SHOTS;

test.beforeAll(async () => {
  env = loadTestEnv();
  projectId = env.project.split("/").pop()!;
  const port = await getInstancePgPort(env);
  execSql(env.databaseId, port, `CREATE SCHEMA ${SCHEMA}`);
  execSql(env.databaseId, port, `CREATE TABLE ${SCHEMA}.setup_item (id INT PRIMARY KEY, label TEXT)`);
  execSql(env.databaseId, port, `CREATE TABLE ${SCHEMA}.setup_note (id INT PRIMARY KEY, body TEXT)`);
  // The export reads the synced metadata, not the live database.
  await env.api.syncDatabase(env.database);
  const group = await env.api.createDatabaseGroup(
    env.project,
    `e2e-setup-${suffix}`,
    GROUP_TITLE,
    `resource.database_name.startsWith("hr_")`,
  );
  created.group = group.name;
  const wi = await env.api.createWorkloadIdentity(env.project, `e2e-setup-ci-${suffix}`, WI_TITLE, {
    providerType: "GITHUB",
    issuerUrl: "https://token.actions.githubusercontent.com",
    allowedAudiences: ["bytebase"],
    subjectPattern: "repo:example-org/billing-api:ref:refs/heads/main",
  });
  created.workloadIdentity = wi.email;
  await env.api.appendProjectBinding(env.project, "roles/gitopsServiceAgent", [
    `workloadIdentity:${wi.email}`,
  ]);
});

test.afterAll(async () => {
  const policy = await env.api.getProjectIamPolicy(env.project);
  for (const binding of policy.bindings) {
    binding.members = binding.members.filter(
      (m) => m !== `workloadIdentity:${created.workloadIdentity}`,
    );
  }
  policy.bindings = policy.bindings.filter((b) => b.members.length > 0);
  await env.api.setProjectIamPolicy(env.project, policy);
  if (created.workloadIdentity) await env.api.deleteWorkloadIdentity(created.workloadIdentity);
  if (created.group) await env.api.deleteDatabaseGroup(created.group);
  execSql(env.databaseId, await getInstancePgPort(env), `DROP SCHEMA IF EXISTS ${SCHEMA} CASCADE`);
  await env.api.syncDatabase(env.database);
});

test("state-based setup: the choices unlock the steps, and step 2 exports the schema files", async ({ page }) => {
  await page.goto(`${env.baseURL}/projects/${projectId}/gitops`);

  // First view: only the workflow type is picked, and the steps are locked.
  await expect(page.getByRole("radio", { name: /Migration-based/ })).toBeChecked();
  const waitingFor = page.getByRole("group", { name: "Waiting for" });
  await expect(waitingFor.getByRole("button")).toHaveText(["Deploy to", "Sign in as"]);
  await expect(page.locator("pre")).toHaveCount(0);
  await expect(page.getByText("Before you choose state-based")).toBeHidden();
  if (SHOTS) await page.screenshot({ path: `${SHOTS}/gitops-setup-locked.png`, fullPage: true });

  // State-based says what it cannot do, with a link to the docs.
  await page.getByRole("radio", { name: /State-based/ }).click();
  await expect(page.getByText("Before you choose state-based")).toBeVisible();
  await expect(page.getByRole("link", { name: "See all limitations in the docs" })).toHaveAttribute(
    "href",
    /state-based-workflow\/limitations/,
  );
  await expect(page.getByRole("heading", { name: "Add the schema files" })).toBeVisible();

  // An open choice opens its field's picker.
  await waitingFor.getByRole("button", { name: "Deploy to" }).click();
  await page.getByRole("option", { name: GROUP_TITLE }).click();
  await expect(page.getByRole("button", { name: "Deploy order" })).toBeVisible();
  await expect(waitingFor.getByRole("button")).toHaveText(["Sign in as"]);

  await waitingFor.getByRole("button", { name: "Sign in as" }).click();
  await page.getByRole("option").filter({ hasText: WI_TITLE }).click();

  // The steps unlock without an export database, and the form has no field for it.
  await expect(waitingFor).toBeHidden();
  await expect(page.locator("ol").getByRole("heading")).toHaveText([
    "Add the workflow files",
    "Add the schema files",
    "Open a pull request",
  ]);
  await expect(page.locator("pre").first()).toContainText('paths: ["schema/**/*.sql"]');
  await expect(page.getByRole("button", { name: "Copy file path" }).first()).toBeVisible();
  await expect(page.getByRole("button", { name: "Download" })).toHaveCount(0);
  if (SHOTS) await page.screenshot({ path: `${SHOTS}/gitops-setup-steps.png`, fullPage: true });

  // Step 2: picking the export database lists its schema files.
  await page.getByText("Select a database", { exact: true }).click();
  await page.getByRole("option").filter({ hasText: `/databases/${env.databaseId}` }).click();
  const download = page.getByRole("button", { name: "Download" });
  await expect(download).toBeEnabled();
  await expect(page.getByText(/\d+ files? from /)).toBeVisible();
  const tables = page.getByRole("button", { name: SCHEMA_FOLDER });
  await expect(tables).toBeVisible();
  if ((await tables.getAttribute("aria-expanded")) !== "true") await tables.click();
  await expect(page.getByRole("button", { name: "setup_note.sql", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "setup_item.sql", exact: true }).click();
  await expect(
    page.locator("pre").filter({ hasText: `CREATE TABLE "${SCHEMA}"."setup_item"` }),
  ).toBeVisible();
  if (SHOTS) await page.screenshot({ path: `${SHOTS}/gitops-setup-schema.png`, fullPage: true });

  const saved = page.waitForEvent("download");
  await download.click();
  expect((await saved).suggestedFilename()).toBe(`${env.databaseId}_schema.zip`);

  // Migration-based drops the schema step and the limits, and keeps the choices.
  await page.getByRole("radio", { name: /Migration-based/ }).click();
  await expect(page.getByText("Before you choose state-based")).toBeHidden();
  await expect(page.locator("ol").getByRole("heading")).toHaveText([
    "Add the workflow files",
    "Open a pull request",
  ]);
  await expect(page.locator("pre").first()).toContainText('paths: ["migrations/*.sql"]');
});
