// Plan detail — AIO Review section: adaptive approval-flow renderer.
//
// Covers the long multi-step flow rendering (spec: "Approval flow renderer";
// approvalFlowLayout.ts) on a 5-step flow with 2 steps approved, 1 current,
// 2 pending:
//   - Wide-but-constrained container → horizontal row folds the approved steps
//     into an "N approved" chip and trailing pending steps into an "N pending"
//     chip, while the current step never folds (CUJ I)
//   - Narrow container (< 560px) → vertical stepper renders every node, no chips
//
// The layout is driven by a ResizeObserver on the section card, so the test
// drives it by resizing the viewport and asserting the rendered anatomy.

import {
  test,
  expect,
  type Page,
  type BrowserContext,
} from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";
import { PlanDetailPage } from "./plan-detail.page";
import { seedReviewPlan, waitForApprovalStatus } from "./plan-helpers";

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
let originalApproval: unknown = null;

const FIVE_STEP_RULE = {
  source: "CHANGE_DATABASE",
  condition: { expression: "true" },
  template: {
    flow: {
      roles: [
        "roles/workspaceAdmin",
        "roles/workspaceDBA",
        "roles/projectOwner",
        "roles/projectDeveloper",
        "roles/projectReleaser",
      ],
    },
    title: "E2E Five-Step",
    description: "Five-step approval flow",
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
  originalApproval = (await env.api.getSetting("WORKSPACE_APPROVAL"))?.value ?? null;

  await env.api.updateProjectSettings(env.project, {
    requireIssueApproval: true,
    requirePlanCheckNoError: false,
    allowSelfApproval: true,
  });
  await env.api.upsertSetting(
    "WORKSPACE_APPROVAL",
    { workspaceApproval: { rules: [FIVE_STEP_RULE] } },
    "value.workspace_approval",
  );

  sharedContext = await browser.newContext({
    storageState: ".auth/state.json",
  });
  page = await sharedContext.newPage();
  planPage = new PlanDetailPage(page, env.baseURL);
});

test.afterAll(async () => {
  await page?.setViewportSize({ width: 1280, height: 720 }).catch(() => {});
  await env.api
    .updateProjectSettings(env.project, originalProjectSettings)
    .catch(() => {});
  await env.api
    .upsertSetting(
      "WORKSPACE_APPROVAL",
      originalApproval ?? { workspaceApproval: { rules: [] } },
      "value.workspace_approval",
    )
    .catch(() => {});
  await sharedContext?.close();
});

test.describe("Long approval flow adaptive rendering (CUJ I)", () => {
  test.describe.configure({ mode: "serial" });

  test.beforeAll(async () => {
    const seeded = await seedReviewPlan(env.api, env.project, env.database, {
      prefix: "E2E Flow Long",
      sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_flow_long_${Date.now()} TEXT;`,
    });
    await waitForApprovalStatus(env.api, seeded.issueName, ["PENDING"]);
    // Approve step 1 (workspaceAdmin) as demo, step 2 (workspaceDBA) as dba1 →
    // 2 approved, step 3 (projectOwner) current, steps 4-5 pending.
    await env.api.approveIssue(seeded.issueName);
    const dba = await BytebaseApiClient.asUser(
      env.baseURL,
      "dba1@example.com",
      "12345678",
    );
    await dba.approveIssue(seeded.issueName);
    await waitForApprovalStatus(env.api, seeded.issueName, ["PENDING"]);
    await planPage.goto(projectId, seeded.planId);
    await planPage.dismissModals();
    await planPage.expandSection("Review");
  });

  test("constrained width folds approved + pending into chips; current stays named", async () => {
    await page.setViewportSize({ width: 900, height: 1100 });
    // Leading approved chip, trailing pending chip, current step named.
    await expect(page.getByText("2 approved")).toBeVisible({ timeout: 15_000 });
    await expect(page.getByText("2 pending")).toBeVisible();
    await expect(page.getByText("Project Owner").first()).toBeVisible();
    await expect(page.getByText("Current", { exact: true })).toBeVisible();
  });

  test("narrow width renders the vertical stepper with every node, no chips", async () => {
    await page.setViewportSize({ width: 480, height: 1500 });
    // Every role is named in the vertical stepper.
    for (const role of [
      "Workspace Admin",
      "Workspace DBA",
      "Project Owner",
      "Project Developer",
      "Project Releaser",
    ]) {
      await expect(page.getByText(role).first()).toBeVisible({ timeout: 15_000 });
    }
    // No fold chips in the vertical layout.
    await expect(page.getByText("2 approved")).toBeHidden();
    await expect(page.getByText("2 pending")).toBeHidden();
  });
});
