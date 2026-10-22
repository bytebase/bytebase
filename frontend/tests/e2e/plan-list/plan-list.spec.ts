// Plans lists ClickOps Plans and Releases lists GitOps Plans (BYT-10019).
// Setup goes through the API; the browser checks which list shows each Plan,
// the status filter, the Releases row target, the setup empty state, and the
// redirect from the retired release detail URL.

import { expect, test } from "@playwright/test";
import { BytebaseApiClient } from "../framework/api-client";
import { loadTestEnv, type TestEnv } from "../framework/env";
import {
  seedDraftPlan,
  setPermissiveGates,
  waitForRollout,
} from "../plan-detail/plan-helpers";

test.setTimeout(180_000);

const SCREENSHOT_DIR = process.env.BYTEBASE_E2E_SCREENSHOT_DIR;
const COMMIT_URL = "https://github.com/acme/app/commit/0123456789abcdef";

let env: TestEnv & { api: BytebaseApiClient };
let projectId: string;
let emptyProjectId: string;
let failedPlan: { id: string; title: string };
let draftPlan: { id: string; title: string };
let gitOpsPlan: { id: string; title: string; releaseId: string };
let originalProjectSettings: Record<string, boolean> = {};
let originalApprovalSetting: unknown = null;

const planId = (name: string) => name.split("/").pop()!;

async function waitForFailedTask(planName: string): Promise<void> {
  const deadline = Date.now() + 60_000;
  while (Date.now() < deadline) {
    const plans = await env.api.listPlans(env.project, 'task_status in ["FAILED"]');
    if (plans.some((plan) => plan.name === planName)) return;
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  throw new Error(`no task of ${planName} failed within 60s`);
}

async function screenshot(page: import("@playwright/test").Page, name: string) {
  if (SCREENSHOT_DIR) {
    await page.screenshot({ path: `${SCREENSHOT_DIR}/${name}.png`, fullPage: true });
  }
}

test.beforeAll(async () => {
  env = loadTestEnv();
  projectId = planId(env.project);
  await env.api.login(env.adminEmail, env.adminPassword);
  const ts = `${Date.now()}`;

  const project = await env.api.getProject(env.project);
  originalProjectSettings = {
    requireIssueApproval: !!project.requireIssueApproval,
    requirePlanCheckNoError: !!project.requirePlanCheckNoError,
    enforceSqlReview: !!project.enforceSqlReview,
    forceIssueLabels: !!project.forceIssueLabels,
  };
  originalApprovalSetting =
    (await env.api.getSetting("WORKSPACE_APPROVAL"))?.value ?? null;
  await setPermissiveGates(env.api, env.project);

  // A ClickOps Plan whose task fails: the table does not exist.
  const failedTitle = `plan-list failed ${ts}`;
  const failedSheet = await env.api.createSheet(
    env.project,
    `ALTER TABLE plan_list_missing_${ts} ADD COLUMN c INT;`,
  );
  const failed = await env.api.createPlan(env.project, failedTitle, [
    { id: `failed-${ts}`, targets: [env.database], sheet: failedSheet },
  ]);
  await env.api.createIssue(env.project, failedTitle, failed.name);
  await waitForRollout(env.api, failed.name);
  const rollout = await env.api.getRollout(failed.name);
  for (const stage of rollout.stages ?? []) {
    const tasks = (stage.tasks ?? []).map((task) => task.name);
    await env.api.batchRunTasks(stage.name, tasks).catch(() => {});
  }
  await waitForFailedTask(failed.name);
  failedPlan = { id: planId(failed.name), title: failedTitle };

  // A ClickOps Plan still in draft review, so it has no tasks.
  const draft = await seedDraftPlan(env.api, env.project, env.database, {
    prefix: "plan-list draft",
    sql: "SELECT 1;",
  });
  draftPlan = {
    id: draft.planId,
    title: (await env.api.getPlan(draft.planName)).title,
  };

  // A GitOps Plan, as bytebase-action creates one.
  const release = await env.api.createRelease(env.project, {
    files: [
      {
        path: `migrations/0001_plan_list_${ts}.sql`,
        version: "0001",
        statement: `CREATE TABLE plan_list_gitops_${ts} (id INT);`,
      },
    ],
    vcsSource: { vcsType: "GITHUB", url: COMMIT_URL },
  });
  const gitOpsTitle = `plan-list gitops ${ts}`;
  const gitOps = await env.api.createReleasePlan(env.project, gitOpsTitle, {
    id: `gitops-${ts}`,
    targets: [env.database],
    release: release.name,
  });
  await env.api.createRollout(gitOps.name);
  gitOpsPlan = {
    id: planId(gitOps.name),
    title: gitOpsTitle,
    releaseId: release.name.split("/").pop()!,
  };

  emptyProjectId = `plan-list-empty-${ts}`;
  await env.api.createProject(emptyProjectId, `Plan list empty ${ts}`);
});

test.afterAll(async () => {
  await env.api.updateProjectSettings(env.project, originalProjectSettings);
  if (originalApprovalSetting) {
    await env.api.upsertSetting(
      "WORKSPACE_APPROVAL",
      originalApprovalSetting,
      "value.workspace_approval",
    );
  }
});

test("Plans lists only ClickOps Plans and filters them by status", async ({ page }) => {
  await page.goto(`${env.baseURL}/projects/${projectId}/plans`);
  const rows = page.getByTestId("plan-list-item");
  await expect(rows.filter({ hasText: failedPlan.title })).toBeVisible();
  await expect(rows.filter({ hasText: draftPlan.title })).toBeVisible();
  await expect(rows.filter({ hasText: gitOpsPlan.title })).toHaveCount(0);
  await screenshot(page, "plans");

  await page.goto(
    `${env.baseURL}/projects/${projectId}/plans?q=${encodeURIComponent("state:ACTIVE status:FAILED")}`,
  );
  await expect(rows.filter({ hasText: failedPlan.title })).toBeVisible();
  await expect(rows.filter({ hasText: draftPlan.title })).toHaveCount(0);
  await screenshot(page, "plans-status-failed");
});

test("Releases lists GitOps Plans and opens a row's rollout", async ({ page }) => {
  await page.goto(`${env.baseURL}/projects/${projectId}/releases`);
  const rows = page.getByTestId("plan-list-item");
  const row = rows.filter({ hasText: gitOpsPlan.releaseId });
  await expect(row).toBeVisible();
  await expect(row.locator(`a[href="${COMMIT_URL}"]`)).toHaveText("0123456");
  await expect(rows.filter({ hasText: failedPlan.title })).toHaveCount(0);
  await expect(page.getByTestId("release-gitops-setup")).toBeVisible();
  await screenshot(page, "releases");

  await row.click();
  await page.waitForURL(`**/projects/${projectId}/plans/${gitOpsPlan.id}/rollout`);
});

test("Releases shows the setup empty state for a project without GitOps Plans", async ({ page }) => {
  await page.goto(`${env.baseURL}/projects/${emptyProjectId}/releases`);
  await expect(page.getByTestId("release-setup-empty-state")).toBeVisible();
  await expect(page.getByTestId("release-gitops-setup")).toHaveCount(0);
  await screenshot(page, "releases-empty");
});

test("an old release detail URL opens the Plan that used the release", async ({ page }) => {
  await page.goto(
    `${env.baseURL}/projects/${projectId}/releases/${gitOpsPlan.releaseId}`,
  );
  await page.waitForURL(`**/projects/${projectId}/plans/${gitOpsPlan.id}/rollout`);
});
