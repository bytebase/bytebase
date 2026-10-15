// Project members — the direct DDL/DML execution field on the grant form.
//
// A role that carries bb.sql.ddl / bb.sql.dml gets a switch instead of a bare
// environment picker. Off writes `resource.environment_id in []` (the binding
// adds no direct execution); on names the environments where a member runs
// DDL/DML straight from SQL Editor. These tests own their user and bindings
// and verify the form, the stored condition, what SQL Editor then does, what
// the member list says, and what the approver's card says.
import {
  type BrowserContext,
  expect,
  type Page,
  test,
} from "@playwright/test";
import type { BytebaseApiClient, IamPolicy } from "../framework/api-client";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { execSql, getInstancePgPort, querySql } from "../framework/psql";
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

// One signed-in grantee context for the two SQL Editor tests: the sign-in is
// the setup bottleneck, and each test sets its own binding and drops the
// probe table before it runs.
let grantee: { context: BrowserContext; page: Page; sqlEditor: SqlEditorPage };

// The grantee's stored SQL Editor User condition, "" when there is none.
async function storedExpression(): Promise<string> {
  const policy = await env.api.getProjectIamPolicy(env.project);
  const binding = policy.bindings.find(
    (b) => b.role === SQL_EDITOR_USER_ROLE && b.members.includes(GRANTEE_MEMBER),
  );
  return binding?.condition?.expression ?? "";
}

// Every policy change is one read and one write; a binding left without
// members goes with it.
async function editPolicy(edit: (policy: IamPolicy) => void): Promise<void> {
  const policy = await env.api.getProjectIamPolicy(env.project);
  edit(policy);
  policy.bindings = policy.bindings.filter((b) => b.members.length > 0);
  await env.api.setProjectIamPolicy(env.project, policy);
}

function dropGrantee(policy: IamPolicy, role?: string): void {
  for (const binding of policy.bindings) {
    if (!role || binding.role === role) {
      binding.members = binding.members.filter((m) => m !== GRANTEE_MEMBER);
    }
  }
}

// Replaces the grantee's SQL Editor User binding with one carrying the expression.
function setGranteeSqlEditorUser(expression: string): Promise<void> {
  return editPolicy((policy) => {
    dropGrantee(policy, SQL_EDITOR_USER_ROLE);
    policy.bindings.push({
      role: SQL_EDITOR_USER_ROLE,
      members: [GRANTEE_MEMBER],
      condition: { expression },
    });
  });
}

function dropProbeTable(): void {
  execSql(env.databaseId, pgPort, `DROP TABLE IF EXISTS ${PROBE_TABLE}`);
}

// A positive oracle: psql failing for any other reason is an error, not "absent".
function probeTableExists(): boolean {
  return (
    querySql(env.databaseId, pgPort, `SELECT to_regclass('public.${PROBE_TABLE}') IS NOT NULL`) ===
    "t"
  );
}

// Runs the CREATE TABLE and returns; each test waits for the outcome it
// asserts. (`runPreparedQuery` waits for a row count or an ERROR, and a
// refused DDL produces neither.)
async function runProbeCreateAsGrantee(): Promise<void> {
  await grantee.sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
  await expect(grantee.sqlEditor.codeEditor).toBeVisible({ timeout: 30_000 });
  await grantee.sqlEditor.setEditorContent(`CREATE TABLE ${PROBE_TABLE} (id INT PRIMARY KEY);`);
  await grantee.sqlEditor.runButton.click();
}

test.beforeAll(async ({ browser }) => {
  // The file-level timeout covers tests only; a hook keeps the 30 s default.
  test.setTimeout(180_000);
  env = loadTestEnv();
  projectId = env.project.split("/").pop()!;
  await env.api.login(env.adminEmail, env.adminPassword);
  [pgPort] = await Promise.all([
    getInstancePgPort(env),
    // Idempotent: a retried run on the same server already has the user.
    env.api.createUser(GRANTEE.email, TEST_PASSWORD, GRANTEE.title).catch((err: unknown) => {
      const msg = err instanceof Error ? err.message : String(err);
      if (!msg.includes("(409)") && !msg.includes("already exists")) throw err;
    }),
  ]);
  // Project Developer carries bb.plans.create, so a refused statement is
  // answered with the change-plan hint instead of a bare error. Dropping the
  // grantee first keeps a retried run from stacking bindings.
  await Promise.all([
    signInBrowserAs(browser, env.baseURL, GRANTEE.email, TEST_PASSWORD, GRANTEE.authFile),
    editPolicy((policy) => {
      dropGrantee(policy);
      policy.bindings.push({ role: "roles/projectDeveloper", members: [GRANTEE_MEMBER] });
    }),
  ]);
  const context = await browser.newContext({ storageState: GRANTEE.authFile });
  const page = await context.newPage();
  grantee = { context, page, sqlEditor: new SqlEditorPage(page, env.baseURL) };
});

// Guarded: a beforeAll that failed before these were assigned must not have
// its error buried under a teardown error. Every binding the spec made goes.
test.afterAll(async () => {
  await grantee?.context.close();
  if (pgPort) dropProbeTable();
  if (env) await editPolicy((policy) => dropGrantee(policy));
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
    await expect(members.sheet).toContainText("DDL/DML cannot run in SQL Editor without approval.");
    await expect(members.environmentPicker).toHaveCount(0);

    await members.directExecutionSwitch.click();
    await expect(members.sheet).toContainText("At least one environment is required.");
    await expect(members.createButton).toBeDisabled();

    await members.pickEnvironment("Test");
    await expect(members.sheet).toContainText(
      "Members with this role can run DDL/DML in SQL Editor without approval in:",
    );
    await expect(members.createButton).toBeEnabled();
    await members.createButton.click();
    await expect(members.sheet).toBeHidden({ timeout: 10_000 });

    await expect
      .poll(storedExpression, { timeout: 10_000 })
      .toContain('resource.environment_id in ["test"]');
  });
});

test.describe("what the grant does in SQL Editor", () => {
  test("on for Test: a CREATE TABLE runs immediately, with no plan offered", async () => {
    await setGranteeSqlEditorUser('resource.environment_id in ["test"]');
    dropProbeTable();

    await runProbeCreateAsGrantee();

    await expect(grantee.page.getByText(/rows affected/i).first()).toBeVisible({
      timeout: 15_000,
    });
    expect(probeTableExists()).toBe(true);
    await expect(grantee.page.getByText(/Data Change Plan/i)).toHaveCount(0);
  });

  test("off: the same statement is refused and answered with the change-plan hint", async () => {
    await setGranteeSqlEditorUser("resource.environment_id in []");
    dropProbeTable();

    await runProbeCreateAsGrantee();

    await expect(grantee.page.getByText(/Data Change Plan/i).first()).toBeVisible({
      timeout: 15_000,
    });
    expect(probeTableExists()).toBe(false);
  });
});

test.describe("member list", () => {
  test("an off binding says it adds nothing; an on binding lists its environment; a clause under || reads as every environment", async ({ page }) => {
    await setGranteeSqlEditorUser("resource.environment_id in []");
    const members = new ProjectMembersPage(page, env.baseURL);
    await members.goto(projectId);
    await members.openMember(GRANTEE.email);
    await expect(members.sheet).toContainText("DDL/DML cannot run in SQL Editor without approval.");
    await members.closeSheet();

    await setGranteeSqlEditorUser('resource.environment_id in ["test"]');
    await members.goto(projectId);
    await members.openMember(GRANTEE.email);
    await expect(members.sheet).toContainText("Can run DDL/DML in SQL Editor without approval in:");
    await expect(members.sheet.getByText("Test", { exact: true }).first()).toBeVisible();
    // A binding is removed, never edited: the grant form only adds.
    await expect(members.sheet.getByRole("button", { name: "Delete", exact: true }).first()).toBeVisible();
    await expect(members.sheet.getByRole("button", { name: "Edit", exact: true })).toHaveCount(0);
    await members.closeSheet();

    // `|| true` makes the clause inert: only a clause at the root narrows.
    await setGranteeSqlEditorUser("resource.environment_id in [] || true");
    await members.goto(projectId);
    await members.openMember(GRANTEE.email);
    await expect(members.sheet).toContainText(
      "Can run DDL/DML in SQL Editor in every environment without approval.",
    );
  });
});

// The approver's card reads the request's condition through the same parser
// and reader. The API creates the requests on the grantee's behalf, so the
// card also names the requester. Role requests need the approval-workflow
// feature the suite's license provides. Last, because an approved request
// grants the role, and afterAll drops every binding.
test.describe("approver's card", () => {
  const states: [string, string | undefined, string, string?][] = [
    [
      "on for Test",
      'resource.environment_id in ["test"]',
      `Approving lets ${GRANTEE.title} run DDL/DML in SQL Editor without approval in:`,
      "Test",
    ],
    ["off", "resource.environment_id in []", "DDL/DML cannot run in SQL Editor without approval."],
    [
      "no clause",
      undefined,
      "Can run DDL/DML in SQL Editor in every environment without approval.",
    ],
    [
      "a clause under ||",
      "resource.environment_id in [] || true",
      "Can run DDL/DML in SQL Editor in every environment without approval.",
    ],
  ];
  for (const [state, expression, text, chip] of states) {
    test(`${state}: the execution row says what approving grants`, async ({ page }) => {
      const issue = await env.api.createRoleGrantIssue(
        env.project,
        { role: SQL_EDITOR_USER_ROLE, user: `users/${GRANTEE.email}`, expression },
        `e2e role grant, ${state}`,
      );
      await page.goto(`${env.baseURL}/projects/${projectId}/issues/${issue.name.split("/").pop()}`);
      const card = page.getByTestId("role-grant-direct-execution");
      await expect(card).toContainText(text, { timeout: 15_000 });
      if (chip) await expect(card.getByText(chip, { exact: true })).toBeVisible();
      await expect(page.getByText(/^Requested by /)).toBeVisible();
    });
  }
});
