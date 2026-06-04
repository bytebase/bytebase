// Plan detail — rollout creation across (approval, plan-check) gates.
//
// The user contract: a rollout transitions through specific states based
// on project settings + plan-check outcomes:
//   - Permissive (no approval gate, no plan-check gate)
//       → rollout auto-creates on issue creation.
//   - Approval gate ON, no plan-check failures
//       → rollout blocked until each approval-flow step completes; on the
//         last approval the rollout auto-creates.
//   - Approval + plan-check gate with an ERROR-level SQL review rule
//       → after approval, the rollout STAYS fully blocked: DEPLOY shows
//         "Failed checks are blocking automatic rollout creation" and NO
//         "Manually create rollout" button is offered. The user must fix
//         the SQL or relax requirePlanCheckNoError (which then reveals the
//         manual-create button).
//
// Approval rules are workspace-level settings (`WORKSPACE_APPROVAL`); the
// new bootstrap leaves them empty, so each test that needs an approval
// flow configures one in `beforeAll` and restores in `afterAll`.

import {
  test,
  expect,
  type Page,
  type BrowserContext,
} from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";
import { PlanDetailPage } from "./plan-detail.page";
import { waitForPlanChecksDone } from "./plan-helpers";

test.setTimeout(180_000);

let env: TestEnv & { api: BytebaseApiClient };
let projectId: string;

let sharedContext: BrowserContext;
let page: Page;
let planPage: PlanDetailPage;
let originalProjectSettings: {
  requireIssueApproval?: boolean;
  requirePlanCheckNoError?: boolean;
  allowSelfApproval?: boolean;
} = {};
let originalApprovalSetting: unknown = null;
let createdReviewConfigs: string[] = [];

// Workspace approval rule used by the approval-gated tests. Single-step
// workspaceAdmin approval matched against `CHANGE_DATABASE` plans (the
// enum value covers DDL/DML — see proto `setting_service.proto`). The
// admin who created the issue (env.adminEmail) can self-approve, which
// is sufficient to verify the state transition without a second user.
const APPROVAL_RULE = {
  source: "CHANGE_DATABASE",
  condition: { expression: "true" },
  template: {
    flow: { roles: ["roles/workspaceAdmin"] },
    title: "E2E Approval",
    description: "Single-step workspaceAdmin approval",
  },
};

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  projectId = env.project.split("/").pop()!;
  await env.api.login(env.adminEmail, env.adminPassword);

  const project = await env.api.getProject(env.project);
  originalProjectSettings = {
    requireIssueApproval: !!project.requireIssueApproval,
    requirePlanCheckNoError: !!project.requirePlanCheckNoError,
    allowSelfApproval: !!project.allowSelfApproval,
  };

  // Snapshot WORKSPACE_APPROVAL so per-test mutations can be restored.
  const existing = await env.api.getSetting("WORKSPACE_APPROVAL");
  originalApprovalSetting = existing?.value ?? null;

  sharedContext = await browser.newContext({
    storageState: ".auth/state.json",
  });
  page = await sharedContext.newPage();
  planPage = new PlanDetailPage(page, env.baseURL);
});

test.afterAll(async () => {
  // Restore project settings.
  await env.api
    .updateProjectSettings(env.project, originalProjectSettings)
    .catch(() => {});

  // Restore workspace approval rules. If none existed, clear our rule.
  await env.api
    .upsertSetting(
      "WORKSPACE_APPROVAL",
      originalApprovalSetting ?? { workspaceApproval: { rules: [] } },
      "value.workspace_approval",
    )
    .catch(() => {});

  // Clean up any review configs / tag policies created mid-test.
  await env.api.deletePolicy(env.project, "tag").catch(() => {});
  for (const name of createdReviewConfigs) {
    await env.api.deleteReviewConfig(name).catch(() => {});
  }
  createdReviewConfigs = [];

  await sharedContext?.close();
});

// Helper — create a plan + issue against env.database; run plan checks;
// wait for them to complete; navigate to the detail page. Returns the
// plan's resource name + id so callers can poll approval state.
async function createIssueWithChecks(
  titlePrefix: string,
  sql: string,
): Promise<{ planName: string; planId: string; issueName: string }> {
  const ts = Date.now();
  const sheet = await env.api.createSheet(env.project, sql);
  const plan = await env.api.createPlan(
    env.project,
    `${titlePrefix} ${ts}`,
    [{ id: `spec-${ts}`, targets: [env.database], sheet }],
  );
  const planName = plan.name;
  const planId = planName.split("/").pop()!;
  const issue = await env.api.createIssue(
    env.project,
    `${titlePrefix} ${ts}`,
    planName,
  );
  await env.api.runPlanChecks(planName);

  await waitForPlanChecksDone(env.api, planName);

  await planPage.goto(projectId, planId);
  await planPage.dismissModals();
  return { planName, planId, issueName: issue.name };
}

test.describe("Permissive settings", () => {
  test.beforeAll(async () => {
    await env.api.updateProjectSettings(env.project, {
      requireIssueApproval: false,
      requirePlanCheckNoError: false,
    });
  });

  test("rollout auto-creates on issue creation; no manual button", async () => {
    const ts = Date.now();
    await createIssueWithChecks(
      "E2E Rollout Permissive",
      `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_perm_${ts} TEXT;`,
    );

    // DEPLOY section visible + task ready to run; no manual create btn.
    await expect(planPage.deploySection).toBeVisible({ timeout: 10_000 });
    await expect(
      page.getByText(/Not started|Pending/i).first(),
    ).toBeVisible({ timeout: 10_000 });
    await expect(planPage.manualCreateRolloutButton).not.toBeVisible({
      timeout: 3_000,
    });
  });
});

test.describe("Approval required, no plan-check gate", () => {
  // WORKSPACE_APPROVAL is an ENTERPRISE feature (FEATURE_APPROVAL_WORKFLOW);
  // mutating it returns 403 on the free plan. The license is installed at bootstrap.
  test.beforeAll(async () => {
    await env.api.updateProjectSettings(env.project, {
      requireIssueApproval: true,
      requirePlanCheckNoError: false,
      // Default with license is false → demo@ (issue creator) gets a
      // 403 trying to approve their own issue. We're doing single-
      // admin approval for test simplicity; flip on.
      allowSelfApproval: true,
    });
    await env.api.upsertSetting(
      "WORKSPACE_APPROVAL",
      { workspaceApproval: { rules: [APPROVAL_RULE] } },
      "value.workspace_approval",
    );
  });

  test("issue blocks until admin approves; afterward rollout auto-creates", async () => {
    const { planName, issueName } = await createIssueWithChecks(
      "E2E Rollout Approval",
      `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_appr_${Date.now()} TEXT;`,
    );
    void planName;

    // Approval flow generation can take a moment after plan checks finish.
    const deadline = Date.now() + 30_000;
    let status = "";
    while (Date.now() < deadline) {
      status = (await env.api.getIssue(issueName)).approvalStatus;
      if (status === "PENDING" || status === "APPROVED" || status === "SKIPPED") {
        break;
      }
      await new Promise((r) => setTimeout(r, 1000));
    }
    // Fail closed: after configuring the approval rule (beforeAll), the issue
    // MUST be PENDING. A non-PENDING status means rule matching / approval-flow
    // generation regressed — assert it loudly rather than skipping, which would
    // hide exactly the approval-gate failure this test exists to catch.
    expect(
      status,
      `issue ${issueName} must be PENDING after configuring a workspace approval rule (last status=${status})`,
    ).toBe("PENDING");

    // While pending: no rollout, no manual create button.
    await expect(planPage.manualCreateRolloutButton).not.toBeVisible({
      timeout: 5_000,
    });

    // Approve. Reload so the UI picks up the new state.
    await env.api.approveIssue(issueName);
    await page.reload();
    await page.waitForLoadState("networkidle");

    // Auto-rollout proceeds (no plan-check failures to block).
    await expect(planPage.deploySection).toBeVisible({ timeout: 15_000 });
    await expect(
      page.getByText(/Not started|Pending/i).first(),
    ).toBeVisible({ timeout: 15_000 });
  });
});

test.describe("Approval + plan-check gate with ERROR-level SQL review", () => {
  let reviewConfigName = "";

  test.beforeAll(async () => {
    await env.api.updateProjectSettings(env.project, {
      requireIssueApproval: true,
      requirePlanCheckNoError: true,
      allowSelfApproval: true,
    });
    await env.api.upsertSetting(
      "WORKSPACE_APPROVAL",
      { workspaceApproval: { rules: [APPROVAL_RULE] } },
      "value.workspace_approval",
    );

    const id = `e2e-rollout-error-${Date.now()}`;
    const cfg = await env.api.upsertReviewConfig(
      id,
      "E2E Rollout ERROR Review Config",
      [{ type: "COLUMN_NO_NULL", level: "ERROR", engine: "POSTGRES" }],
    );
    reviewConfigName = cfg.name;
    createdReviewConfigs.push(cfg.name);
    await env.api.upsertReviewConfigTag(env.project, reviewConfigName);
  });

  test("after approval, plan-check failure keeps the rollout fully blocked (no manual-create)", async () => {
    const { issueName } = await createIssueWithChecks(
      "E2E Rollout Approval+Error",
      `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_appr_err_${Date.now()} TEXT;`,
    );

    // Wait for approval state to materialize.
    const deadline = Date.now() + 30_000;
    let status = "";
    while (Date.now() < deadline) {
      status = (await env.api.getIssue(issueName)).approvalStatus;
      if (status === "PENDING" || status === "APPROVED" || status === "SKIPPED") {
        break;
      }
      await new Promise((r) => setTimeout(r, 1000));
    }
    // Fail closed (same contract as the approval test above): a configured
    // approval rule must put the issue in PENDING; assert rather than skip so a
    // regression in approval-flow generation fails loudly.
    expect(
      status,
      `issue ${issueName} must be PENDING after configuring a workspace approval rule (last status=${status})`,
    ).toBe("PENDING");

    await env.api.approveIssue(issueName);
    await page.reload();
    await page.waitForLoadState("networkidle");

    // Product contract: with requirePlanCheckNoError=true and an ERROR
    // rule, even after the approval flow completes the rollout stays
    // fully blocked. DEPLOY shows "Checks must pass. Failed" with the
    // helper text "Failed checks are blocking automatic rollout
    // creation." NO "Manually create rollout" button is offered — the
    // user must either fix the SQL or relax the gate. (Symmetric to
    // the `with gate on` test in plan-detail-checks.spec.ts.)
    await expect(planPage.checksSummary()).toContainText("Error", {
      timeout: 15_000,
    });
    await expect(
      page.getByText(/Failed checks are blocking/i).first(),
    ).toBeVisible({ timeout: 10_000 });
    await expect(planPage.manualCreateRolloutButton).not.toBeVisible({
      timeout: 3_000,
    });
  });
});
