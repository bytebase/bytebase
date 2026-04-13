import { test, expect, type Page, type BrowserContext } from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";
import { PlanDetailPage } from "./plan-detail.page";

test.setTimeout(120_000);

let env: TestEnv & { api: BytebaseApiClient };
let projectId: string;
let planId: string;
let issueName: string;
let planURL: string;

let sharedContext: BrowserContext;
let page: Page;
let shouldSkip = false;
let planPage: PlanDetailPage;

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  projectId = env.project.split("/").pop()!;
  await env.api.login(env.adminEmail, env.adminPassword);

  // Enable approval + plan-check gate
  await env.api.updateProjectSettings(env.project, {
    requireIssueApproval: true,
    requirePlanCheckNoError: true,
  });

  // Upgrade COLUMN_NO_NULL from WARNING to ERROR so checks will fail
  await env.api.updateReviewConfigRuleLevel(
    "reviewConfigs/sql-review-sample-policy",
    "COLUMN_NO_NULL",
    "POSTGRES",
    "ERROR"
  );

  // Create sheet — nullable TEXT column triggers COLUMN_NO_NULL ERROR
  const columnName = `e2e_checks_${Date.now()}`;
  const sheetName = await env.api.createSheet(
    env.project,
    `ALTER TABLE employee ADD COLUMN IF NOT EXISTS ${columnName} TEXT;`
  );

  // Create the plan
  const specId = `spec-${Date.now()}`;
  const plan = await env.api.createPlan(env.project, `E2E Checks Blocker Test ${Date.now()}`, [
    { id: specId, targets: [env.database], sheet: sheetName },
  ]);
  planId = plan.name.split("/").pop()!;
  planURL = `${env.baseURL}/projects/${projectId}/plans/${planId}`;

  // Create an issue backed by the plan
  const issue = await env.api.createIssue(env.project, "E2E Checks Blocker", plan.name);
  issueName = issue.name;

  // Run plan checks to trigger the SQL review errors
  await env.api.runPlanChecks(plan.name);

  // Poll until approval is no longer CHECKING
  const deadline = Date.now() + 60_000;
  let approvalStatus = "CHECKING";
  while (Date.now() < deadline) {
    const issueData = await env.api.getIssue(issueName);
    approvalStatus = issueData.approvalStatus;
    if (approvalStatus !== "CHECKING") break;
    await new Promise((r) => setTimeout(r, 1000));
  }

  // If PENDING, approve both steps (Project Owner + DBA)
  if (approvalStatus === "PENDING") {
    await env.api.approveIssue(issueName);
    const dbaApi = await BytebaseApiClient.asUser(env.baseURL, "dba1@example.com", "12345678");
    await dbaApi.approveIssue(issueName);
  }

  // Verify approval is APPROVED. If the environment has no matching approval
  // rule, status may be SKIPPED — skip the suite rather than false-fail.
  const finalIssue = await env.api.getIssue(issueName);
  if (finalIssue.approvalStatus === "SKIPPED") {
    shouldSkip = true;
    return;
  }
  if (finalIssue.approvalStatus !== "APPROVED") {
    throw new Error(
      `Expected approvalStatus "APPROVED" but got "${finalIssue.approvalStatus}" on issue "${issueName}"`
    );
  }

  // Open the shared browser context
  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
  planPage = new PlanDetailPage(page, env.baseURL);
});

test.afterAll(async () => {
  // Restore permissive project settings
  await env.api.updateProjectSettings(env.project, {
    requireIssueApproval: false,
    requirePlanCheckNoError: false,
  });

  // Restore COLUMN_NO_NULL rule back to WARNING
  await env.api.updateReviewConfigRuleLevel(
    "reviewConfigs/sql-review-sample-policy",
    "COLUMN_NO_NULL",
    "POSTGRES",
    "WARNING"
  );

  await sharedContext?.close();
});

test.describe("Plan Detail: Checks Blocker (BYT-9159)", () => {
  test.describe.configure({ mode: "serial" });

  test("checks show ERROR result", async () => {
    test.skip(shouldSkip, "Approval resolved to SKIPPED — environment lacks matching rule");
    await planPage.goto(projectId, planId);
    await planPage.dismissModals();

    await expect(page.getByText(/Error|Failed/i).first()).toBeVisible({ timeout: 15_000 });
  });

  test("approval shows Done", async () => {
    test.skip(shouldSkip, "Approval resolved to SKIPPED");
    await expect(
      page.getByText(/Approved|Done/i).first()
    ).toBeVisible({ timeout: 10_000 });
  });

  test("deploy section says checks are blocking", async () => {
    test.skip(shouldSkip, "Approval resolved to SKIPPED");
    await expect(page.getByText("Failed checks are blocking")).toBeVisible({ timeout: 10_000 });
  });

  test("no manual create rollout button when checks required and failed (BYT-9159)", async () => {
    test.skip(shouldSkip, "Approval resolved to SKIPPED");
    await expect(planPage.manualCreateRolloutButton).not.toBeVisible({ timeout: 3_000 });
  });

  test("button appears after disabling check requirement", async () => {
    test.skip(shouldSkip, "Approval resolved to SKIPPED");
    await env.api.updateProjectSettings(env.project, { requirePlanCheckNoError: false });

    await page.goto(planURL);
    await page.waitForLoadState("networkidle");
    await planPage.dismissModals();

    await expect(planPage.manualCreateRolloutButton).toBeVisible({ timeout: 10_000 });
  });

  test("deploy text confirms checks are optional", async () => {
    test.skip(shouldSkip, "Approval resolved to SKIPPED");
    await expect(page.getByText(/Failed checks won't block/i)).toBeVisible({ timeout: 10_000 });
  });
});
