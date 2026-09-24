// Project members — the direct DDL/DML execution field on the grant form.
//
// A role that carries bb.sql.ddl / bb.sql.dml gets a switch instead of a bare
// environment picker. Off writes `resource.environment_id in []` (the binding
// adds no direct execution); on names the environments where a member runs
// DDL/DML straight from SQL Editor. These tests own their user and bindings
// and verify the form, the stored condition, what SQL Editor then does, and
// what the member list says.
import {
  type BrowserContext,
  expect,
  type Page,
  test,
} from "@playwright/test";
import type { BytebaseApiClient, IamBinding } from "../framework/api-client";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { execSql, getInstancePgPort } from "../framework/psql";
import { signInBrowserAs } from "../framework/sign-in";
import { SqlEditorPage } from "../sql-editor/sql-editor.page";
import { ProjectMembersPage } from "./project-members.page";

test.setTimeout(180_000);

let env: TestEnv & { api: BytebaseApiClient };
let projectId: string;
let pgPort: string;

const TEST_PASSWORD = "e2e-members-pw-1!"; // NOSONAR — e2e fixture only
const GRANTEE = {
  email: "e2e-members-grantee@example.com",
  title: "E2E Members Grantee",
  authFile: ".auth/members-grantee.json",
};
const GRANTEE_MEMBER = `user:${GRANTEE.email}`;
const SQL_EDITOR_USER_ROLE = "roles/sqlEditorUser";
// hr_test lives in the sample "Test" environment.
const PROBE_TABLE = "e2e_direct_exec_probe";

let grantee: { context: BrowserContext; page: Page; sqlEditor: SqlEditorPage };

async function granteeSqlEditorBinding(): Promise<IamBinding | undefined> {
  const policy = await env.api.getProjectIamPolicy(env.project);
  return policy.bindings.find(
    (b) => b.role === SQL_EDITOR_USER_ROLE && b.members.includes(GRANTEE_MEMBER),
  );
}

async function removeGranteeSqlEditorBindings(): Promise<void> {
  const policy = await env.api.getProjectIamPolicy(env.project);
  policy.bindings = policy.bindings
    .map((b) =>
      b.role === SQL_EDITOR_USER_ROLE
        ? { ...b, members: b.members.filter((m) => m !== GRANTEE_MEMBER) }
        : b,
    )
    .filter((b) => b.members.length > 0);
  await env.api.setProjectIamPolicy(env.project, policy);
}

async function grantSqlEditorUser(expression: string): Promise<void> {
  await removeGranteeSqlEditorBindings();
  await env.api.appendProjectBinding(env.project, SQL_EDITOR_USER_ROLE, [GRANTEE_MEMBER], {
    expression,
  });
}

function dropProbeTable(): void {
  execSql(env.databaseId, pgPort, `DROP TABLE IF EXISTS ${PROBE_TABLE}`);
}

function probeTableExists(): boolean {
  try {
    execSql(env.databaseId, pgPort, `SELECT 1 FROM ${PROBE_TABLE} LIMIT 1`);
    return true;
  } catch {
    return false;
  }
}

async function runProbeCreateAsGrantee(): Promise<void> {
  await grantee.sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
  await grantee.page.waitForTimeout(1500);
  await grantee.sqlEditor.runPreparedQuery(`CREATE TABLE ${PROBE_TABLE} (id INT PRIMARY KEY);`);
}

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  projectId = env.project.split("/").pop()!;
  await env.api.login(env.adminEmail, env.adminPassword);
  pgPort = await getInstancePgPort(env);

  // Idempotent: a retried run on the same server already has the user.
  try {
    await env.api.createUser(GRANTEE.email, TEST_PASSWORD, GRANTEE.title);
  } catch {
    /* already exists */
  }
  // Project Developer carries bb.plans.create, so a refused statement is
  // answered with the change-plan hint instead of a bare error.
  const policy = await env.api.getProjectIamPolicy(env.project);
  const isDeveloper = policy.bindings.some(
    (b) => b.role === "roles/projectDeveloper" && b.members.includes(GRANTEE_MEMBER),
  );
  if (!isDeveloper) {
    await env.api.appendProjectBinding(env.project, "roles/projectDeveloper", [GRANTEE_MEMBER]);
  }
  await removeGranteeSqlEditorBindings();
  dropProbeTable();

  await signInBrowserAs(browser, env.baseURL, GRANTEE.email, TEST_PASSWORD, GRANTEE.authFile);
  const context = await browser.newContext({ storageState: GRANTEE.authFile });
  const page = await context.newPage();
  grantee = { context, page, sqlEditor: new SqlEditorPage(page, env.baseURL) };
});

test.afterAll(async () => {
  await grantee?.context.close();
  dropProbeTable();
  await removeGranteeSqlEditorBindings();
  // The user stays: DELETE deactivates rather than purges, and a retried
  // run would then fail to sign in. The disposable server is torn down anyway.
});

test.describe("grant form", () => {
  test("the switch is off by default, refuses a bare on, and writes the picked environment", async ({ page }) => {
    const members = new ProjectMembersPage(page, env.baseURL);
    await members.goto(projectId);
    await members.openGrantAccess();
    await members.pickAccount(GRANTEE.email);
    await members.pickRole("SQL Editor User");

    await expect(members.directExecutionSwitch).toHaveAttribute("aria-checked", "false");
    await expect(members.sheet).toContainText("this grant adds no direct DDL/DML access");
    await expect(members.environmentPicker).toHaveCount(0);

    await members.directExecutionSwitch.click();
    await expect(members.sheet).toContainText("Pick at least one environment, or turn this off.");
    await expect(members.createButton).toBeDisabled();

    await members.pickEnvironment("Test");
    await expect(members.sheet).toContainText(
      "Members with this role run DDL/DML straight from SQL Editor",
    );
    await expect(members.createButton).toBeEnabled();
    await members.createButton.click();
    await expect(members.sheet).toBeHidden({ timeout: 10_000 });

    await expect
      .poll(async () => (await granteeSqlEditorBinding())?.condition?.expression ?? "", {
        timeout: 10_000,
      })
      .toContain('resource.environment_id in ["test"]');
  });
});

test.describe("what the grant does in SQL Editor", () => {
  test("on for Test: a CREATE TABLE runs immediately, with no plan offered", async () => {
    await grantSqlEditorUser('resource.environment_id in ["test"]');
    dropProbeTable();

    await runProbeCreateAsGrantee();

    await expect.poll(() => probeTableExists(), { timeout: 15_000 }).toBe(true);
    await expect(grantee.page.getByText(/Data Change Plan/i)).toHaveCount(0);
  });

  test("off: the same statement is refused and answered with the change-plan hint", async () => {
    await grantSqlEditorUser("resource.environment_id in []");
    dropProbeTable();

    await runProbeCreateAsGrantee();

    await expect(grantee.page.getByText(/Data Change Plan/i).first()).toBeVisible({
      timeout: 15_000,
    });
    expect(probeTableExists()).toBe(false);
  });
});

test.describe("member list", () => {
  test("an off binding says it adds nothing; an on binding lists its environment", async ({ page }) => {
    await grantSqlEditorUser("resource.environment_id in []");
    const members = new ProjectMembersPage(page, env.baseURL);
    await members.goto(projectId);
    await members.openMember(GRANTEE.email);
    await expect(members.sheet).toContainText("This binding adds no direct DDL/DML access");
    await members.closeSheet();

    await grantSqlEditorUser('resource.environment_id in ["test"]');
    await page.reload();
    await members.grantAccessButton.waitFor({ timeout: 15_000 });
    await members.openMember(GRANTEE.email);
    await expect(members.sheet).toContainText("Runs DDL/DML in SQL Editor without approval in:");
    await expect(members.sheet.getByText("Test", { exact: true }).first()).toBeVisible();
  });
});
