import {
  type APIRequestContext,
  type APIResponse,
  expect,
  test,
} from "@playwright/test";
import { loadTestEnv } from "../framework/env";
import { execSql, getInstancePgPort } from "../framework/psql";
import { SqlEditorPage } from "../sql-editor/sql-editor.page";

const GROUP_EMAIL = "e2e-group-masking@example.com";
const GROUP_TITLE = "E2E Masked Readers";
const GROUP_OWNER = { member: "users/dba1@example.com", role: "OWNER" };
const RULE_TITLE = "E2E Group-Targeted Masking";
const DATABASE_SCHEMA = "e2e_group_masking";
const QUERY =
  "SELECT email, display_name FROM e2e_group_masking.group_masking_contact WHERE id = 1;";

test.setTimeout(90_000);

let env: ReturnType<typeof loadTestEnv>;
let api: APIRequestContext | undefined;
let pgPort: string | undefined;
let groupPath: string | undefined;
let restorePolicy: (() => Promise<void>) | undefined;
let setMembership: (isMember: boolean) => Promise<void>;

async function expectOK(response: APIResponse): Promise<void> {
  if (!response.ok()) {
    throw new Error(
      `${response.url()} failed (${response.status()}): ${await response.text()}`,
    );
  }
}

test.beforeAll(async ({ playwright }) => {
  env = loadTestEnv();
  const token = await env.api.login(env.adminEmail, env.adminPassword);
  const request = await playwright.request.newContext({
    baseURL: env.baseURL,
    extraHTTPHeaders: { Authorization: `Bearer ${token}` },
  });
  api = request;
  const { workspace } = await env.api.getActuatorInfo();
  const policyPath = `/v1/${workspace}/policies/masking_rule`;
  const originalPolicy = await request.get(policyPath);
  if (originalPolicy.status() !== 404) {
    await expectOK(originalPolicy);
  }

  const group = await request.post(
    `/v1/groups?groupEmail=${encodeURIComponent(GROUP_EMAIL)}`,
    { data: { title: GROUP_TITLE, members: [GROUP_OWNER] } },
  );
  await expectOK(group);
  const { name: groupName, members: initialMembers } = (await group.json()) as {
    name: string;
    members: { member: string; role: string }[];
  };
  const path = `/v1/${groupName}`;
  groupPath = path;
  expect(initialMembers).toEqual([GROUP_OWNER]);
  setMembership = async (isMember) => {
    const members = [GROUP_OWNER];
    if (isMember) {
      members.push({ member: `users/${env.adminEmail}`, role: "MEMBER" });
    }
    const update = await request.patch(`${path}?updateMask=members`, {
      data: { members },
    });
    await expectOK(update);
    const updated = (await update.json()) as { members: typeof members };
    expect(updated.members).toHaveLength(members.length);
    expect(updated.members).toEqual(expect.arrayContaining(members));
  };

  pgPort = await getInstancePgPort(env);
  execSql(
    env.databaseId,
    pgPort,
    `CREATE SCHEMA ${DATABASE_SCHEMA};
     CREATE TABLE ${DATABASE_SCHEMA}.group_masking_contact (
       id integer PRIMARY KEY,
       email text NOT NULL,
       display_name text NOT NULL
     );
     INSERT INTO ${DATABASE_SCHEMA}.group_masking_contact
       VALUES (1, 'fixture-reader@example.com', 'Unmasked fixture name');`,
  );
  await env.api.syncDatabase(env.database);

  const updatePath = `${policyPath}?allowMissing=true&updateMask=masking_rule_policy,enforce`;
  const savedPolicy = originalPolicy.ok() ? await originalPolicy.json() : null;
  await expectOK(
    await request.patch(updatePath, {
      data: {
        type: "MASKING_RULE",
        enforce: true,
        maskingRulePolicy: {
          rules: [
            {
              id: "78d396e8-9a43-4310-a243-110f6c61278e",
              condition: {
                title: RULE_TITLE,
                expression: `resource.table_name == "group_masking_contact" && resource.column_name == "email" && "${groupName}" in principal.groups`,
              },
              semanticType: "bb.default",
            },
          ],
        },
      },
    }),
  );
  restorePolicy = async () => {
    await expectOK(
      savedPolicy
        ? await request.patch(updatePath, { data: savedPolicy })
        : await request.delete(policyPath),
    );
  };
});

test.afterAll(async () => {
  const request = api;
  const path = groupPath;
  const port = pgPort;
  const results = await Promise.allSettled([
    restorePolicy?.(),
    request && path && request.delete(path).then(expectOK),
    port &&
      (async () => {
        execSql(
          env.databaseId,
          port,
          `DROP SCHEMA IF EXISTS ${DATABASE_SCHEMA} CASCADE`,
        );
        await env.api.syncDatabase(env.database);
      })(),
  ]);
  await request?.dispose();
  const failures = results.flatMap((result) =>
    result.status === "rejected" ? [result.reason] : [],
  );
  if (failures.length > 0) {
    throw new AggregateError(failures, "Group masking fixture cleanup failed");
  }
});

test("global masking follows current user group membership", async ({ page }) => {
  await test.step("the rule editor resolves the stored group in its selector", async () => {
    await page.goto(`${env.baseURL}/global-masking`);
    await page
      .getByRole("button", { name: `Edit ${RULE_TITLE}`, exact: true })
      .click();
    const condition = page.getByRole("group", { name: "Rule condition" });
    await expect(condition.getByText("principal.groups", { exact: true })).toBeVisible();
    await expect(condition.getByText("contains", { exact: true })).toBeVisible();
    const groupLabel = `${GROUP_TITLE} (${GROUP_EMAIL})`;
    await condition
      .getByRole("button", { name: groupLabel, exact: true })
      .click();
    await page
      .getByRole("dialog", { name: "Select value" })
      .getByRole("button", { name: groupLabel, exact: true })
      .click();
    await page.getByRole("button", { name: "Cancel", exact: true }).click();
  });

  const editor = new SqlEditorPage(page);
  const row = page.getByRole("table").locator('[data-row-index="0"]');
  const email = row.getByRole("cell").nth(1);
  const displayName = row.getByRole("cell").nth(2);
  const runAndExpectEmail = async (value: string) => {
    // Reopen the SQL Editor to reset its local query throttle, not the server session.
    await page.goto(`${env.baseURL}/sql-editor/${env.database}`);
    await editor.setEditorContent(QUERY);
    const [response] = await Promise.all([
      page.waitForResponse(
        (response) =>
          new URL(response.url()).pathname === "/bytebase.v1.SQLService/Query",
        { timeout: 15_000 },
      ),
      editor.runButton.click(),
    ]);
    expect(response.ok()).toBe(true);
    await expect(email).toHaveText(value);
    await expect(displayName).toHaveText("Unmasked fixture name");
  };

  await test.step("a nonmember reads the original value", async () => {
    await runAndExpectEmail("fixture-reader@example.com");
  });
  await test.step("adding membership masks the next query", async () => {
    await setMembership(true);
    await runAndExpectEmail("******");
  });
  await test.step("removing membership unmasks the next query", async () => {
    await setMembership(false);
    await runAndExpectEmail("fixture-reader@example.com");
  });
});
