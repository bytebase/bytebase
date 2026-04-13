import { test, expect, type Page, type BrowserContext } from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";
import { PlanDetailPage } from "./plan-detail.page";

test.setTimeout(120_000);

let env: TestEnv & { api: BytebaseApiClient };
let projectId: string;
let planId: string;
let issueName: string;

let sharedContext: BrowserContext;
let page: Page;
let planPage: PlanDetailPage;
let shouldSkip = false;

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  projectId = env.project.split("/").pop()!;
  await env.api.login(env.adminEmail, env.adminPassword);

  // Require approval so the approval flow is generated; disable plan-check gate
  // so the SQL check result doesn't block us independently.
  await env.api.updateProjectSettings(env.project, {
    requireIssueApproval: true,
    requirePlanCheckNoError: false,
  });

  // Unique column name avoids collisions across runs.
  const columnName = `e2e_approval_${Date.now()}`;
  const sheetName = await env.api.createSheet(
    env.project,
    `ALTER TABLE employee ADD COLUMN IF NOT EXISTS ${columnName} TEXT;`
  );

  const specId = `spec-${Date.now()}`;
  const plan = await env.api.createPlan(
    env.project,
    `E2E Approval Blocker Test ${Date.now()}`,
    [{ id: specId, targets: [env.database], sheet: sheetName }]
  );
  planId = plan.name.split("/").pop()!;

  const issue = await env.api.createIssue(
    env.project,
    "E2E Approval Blocker",
    plan.name
  );
  issueName = issue.name;

  // Run plan checks — this triggers SQL classification which drives the
  // approval flow generation.  Without this call the approvalStatus may
  // stay "SKIPPED" (no matching rule) rather than becoming "PENDING".
  await env.api.runPlanChecks(plan.name);

  // Wait for the approval flow to reach PENDING.  If it instead reaches
  // SKIPPED (no approval rule matched), the suite is skipped gracefully
  // so it doesn't produce false failures in environments without the
  // required policy.
  const deadline = Date.now() + 30_000;
  let lastStatus = "";
  while (Date.now() < deadline) {
    const freshIssue = await env.api.getIssue(issueName);
    lastStatus = freshIssue.approvalStatus;
    if (lastStatus === "PENDING" || lastStatus === "SKIPPED" || lastStatus === "APPROVED") break;
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }

  if (lastStatus === "SKIPPED" || lastStatus === "APPROVED") {
    // No approval rule matched, or auto-approved — skip this blocker suite.
    shouldSkip = true;
    return;
  }

  if (lastStatus !== "PENDING") {
    throw new Error(
      `Timed out waiting for approvalStatus "PENDING" on issue "${issueName}"; last status: "${lastStatus}"`
    );
  }

  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
  planPage = new PlanDetailPage(page, env.baseURL);
});

test.afterAll(async () => {
  await sharedContext?.close();
  // Restore permissive approval setting so other test suites are unaffected.
  await env.api
    .updateProjectSettings(env.project, { requireIssueApproval: false })
    .catch((err: unknown) => {
      console.warn(
        `afterAll updateProjectSettings: ${err instanceof Error ? err.message : err}`
      );
    });
});

test.describe("Plan Detail: Approval Blocker", () => {
  test.describe.configure({ mode: "serial" });

  test("deploy section shows approval pending when required", async () => {
    test.skip(shouldSkip, "No approval rule matched — environment lacks required policy");
    await planPage.goto(projectId, planId);
    await planPage.dismissModals();

    // The deploy section should surface the approval gate message.
    const deployArea = page.getByText("Deploy").locator("..");
    await expect(
      deployArea.getByText(/Approval flow must be done/i)
    ).toBeVisible({ timeout: 15_000 });

    // A "Pending" indicator should also be visible somewhere in the deploy area.
    await expect(deployArea.getByText(/Pending/i).first()).toBeVisible();
  });

  test("no manual create rollout button when approval is pending and required", async () => {
    test.skip(shouldSkip, "No approval rule matched");
    // The button must NOT exist (or must be hidden) when approval is blocking.
    // Use a short timeout — if it's present the test fails fast.
    await expect(planPage.manualCreateRolloutButton).not.toBeVisible({
      timeout: 3_000,
    });
  });

  test("sidebar shows Under review", async () => {
    test.skip(shouldSkip, "No approval rule matched");
    await expect(
      page.getByRole("complementary").getByText(/Under review/i)
    ).toBeVisible();
  });

  test("after approving both steps, deploy section updates", async () => {
    test.skip(shouldSkip, "No approval rule matched");
    // First approval: admin (Project Owner / first approver in the flow).
    await env.api.approveIssue(issueName);

    // Second approval: DBA role.
    const dbaApi = await BytebaseApiClient.asUser(
      env.baseURL,
      "dba1@example.com",
      "12345678"
    );
    await dbaApi.approveIssue(issueName);

    // Reload so the browser picks up the updated approval state.
    await page.reload();
    await page.waitForLoadState("networkidle");

    // After both approvals, the sidebar should reflect "Approved" or the page
    // may show a rollout (auto-created if checks also passed). Either way,
    // the approval is no longer "Pending".
    await expect(
      page.getByText(/Approved|Approval is complete/i).first()
    ).toBeVisible({ timeout: 15_000 });
  });
});
