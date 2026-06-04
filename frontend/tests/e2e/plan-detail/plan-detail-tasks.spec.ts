// Plan detail — task execution.
//
// Covers the task lifecycle once a rollout exists (auto-created here
// via permissive project settings):
//   - Running a successful task transitions to Done.
//   - Running a failing task (nonexistent target) transitions to Failed
//     and surfaces a Retry button.

import {
  test,
  expect,
  type Page,
  type BrowserContext,
} from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";
import { PlanDetailPage } from "./plan-detail.page";

test.setTimeout(180_000);

let env: TestEnv & { api: BytebaseApiClient };
let projectId: string;

let sharedContext: BrowserContext;
let page: Page;
let planPage: PlanDetailPage;
let originalSettings: {
  requireIssueApproval?: boolean;
  requirePlanCheckNoError?: boolean;
} = {};

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  projectId = env.project.split("/").pop()!;
  await env.api.login(env.adminEmail, env.adminPassword);

  // Permissive — both tests want a rollout to be auto-created so they
  // can drive task execution directly. We only mutate project settings;
  // each describe creates its own plan.
  const project = await env.api.getProject(env.project);
  originalSettings = {
    requireIssueApproval: !!project.requireIssueApproval,
    requirePlanCheckNoError: !!project.requirePlanCheckNoError,
  };
  await env.api.updateProjectSettings(env.project, {
    requireIssueApproval: false,
    requirePlanCheckNoError: false,
  });

  sharedContext = await browser.newContext({
    storageState: ".auth/state.json",
  });
  page = await sharedContext.newPage();
  planPage = new PlanDetailPage(page, env.baseURL);
});

test.afterAll(async () => {
  await env.api
    .updateProjectSettings(env.project, originalSettings)
    .catch(() => {});
  await sharedContext?.close();
});

// Helper — create a plan + issue against env.database with the given SQL,
// navigate to its detail page. Returns the plan id for re-navigation.
async function createPlanAndNavigate(
  titlePrefix: string,
  sql: string,
): Promise<string> {
  const ts = Date.now();
  const sheet = await env.api.createSheet(env.project, sql);
  const plan = await env.api.createPlan(
    env.project,
    `${titlePrefix} ${ts}`,
    [{ id: `spec-${ts}`, targets: [env.database], sheet }],
  );
  const planId = plan.name.split("/").pop()!;
  await env.api.createIssue(env.project, `${titlePrefix} ${ts}`, plan.name);
  await planPage.goto(projectId, planId);
  await planPage.dismissModals();
  return planId;
}

test.describe("Successful task transitions to Done", () => {
  test("Run → Done", async () => {
    const colName = `e2e_tasks_ok_${Date.now()}`;
    await createPlanAndNavigate(
      "E2E Task Success",
      `ALTER TABLE employee ADD COLUMN IF NOT EXISTS ${colName} TEXT;`,
    );

    await expect(
      page.getByText(/Not started|Pending/i).first(),
    ).toBeVisible({ timeout: 15_000 });

    await planPage.runTask();
    await expect(page.getByText("Done").first()).toBeVisible({
      timeout: 60_000,
    });
  });
});

test.describe("Failing task transitions to Failed and shows Retry", () => {
  test("Run → Failed + Retry button visible", async () => {
    const missingTable = `nonexistent_table_e2e_${Date.now()}`;
    await createPlanAndNavigate(
      "E2E Task Failure",
      `ALTER TABLE ${missingTable} ADD COLUMN c1 TEXT;`,
    );

    await expect(
      page.getByText(/Not started|Pending/i).first(),
    ).toBeVisible({ timeout: 15_000 });

    await planPage.runTask();
    await expect(page.getByText("Failed").first()).toBeVisible({
      timeout: 30_000,
    });

    await expect(planPage.retryButton).toBeVisible({ timeout: 5_000 });
  });
});
