// Plan detail — AIO Review section: rollout-readiness footer + bypass.
//
// Covers the footer state machine + bypass gating (spec: "Rollout readiness
// footer"; readinessFooterState.ts):
//   - Approved/skipped + failed checks → primary "Bypass and deploy" button →
//     confirm sheet → rollout created (CUJ F)
//   - Waiting-review bypass link is gated by projectRequireIssueApproval:
//     mandatory → no link (G1); optional → muted link (G2)
//   - A mandatory project gate (requirePlanCheckNoError) hard-blocks the
//     confirm-sheet Deploy (cannot be acknowledged away)
//   - BYT-9745 guard: confirm-sheet REVIEW box shows the skip note for
//     skipped-approval issues (was an empty box; fixed by #20662)
//
// The "approved + checks failed" and hard-gate states need a skipped approval
// (no rule) plus an ERROR-level review rule on the change, so those describes
// attach/detach a review config around their own setup.

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
const createdReviewConfigs: string[] = [];

const ONE_STEP_RULE = {
  source: "CHANGE_DATABASE",
  condition: { expression: "true" },
  template: {
    flow: { roles: ["roles/workspaceAdmin"] },
    title: "E2E Footer One-Step",
    description: "Single-step workspaceAdmin approval",
  },
};

async function goReview(planId: string): Promise<void> {
  await planPage.goto(projectId, planId);
  await planPage.dismissModals();
  await planPage.expandSection("Review");
}

// Attach a single ERROR-level COLUMN_NO_NULL rule to the project so a nullable
// column trips it. Returns nothing — the tag is detached in each describe that
// needs a clean slate and in afterAll.
async function attachErrorConfig(): Promise<void> {
  const id = `e2e-footer-err-${Date.now()}`;
  const cfg = await env.api.upsertReviewConfig(id, "E2E Footer ERROR", [
    { type: "COLUMN_NO_NULL", level: "ERROR", engine: "POSTGRES" },
  ]);
  createdReviewConfigs.push(cfg.name);
  await env.api.upsertReviewConfigTag(env.project, cfg.name);
}

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

  sharedContext = await browser.newContext({
    storageState: ".auth/state.json",
  });
  page = await sharedContext.newPage();
  planPage = new PlanDetailPage(page, env.baseURL);
});

test.afterAll(async () => {
  await env.api.deletePolicy(env.project, "tag").catch(() => {});
  for (const name of createdReviewConfigs) {
    await env.api.deleteReviewConfig(name).catch(() => {});
  }
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

test.describe("Bypass when approved but checks failed (CUJ F)", () => {
  test.describe.configure({ mode: "serial" });
  let planId: string;

  test.beforeAll(async () => {
    await env.api.deletePolicy(env.project, "tag").catch(() => {});
    await env.api.updateProjectSettings(env.project, {
      requireIssueApproval: false,
      requirePlanCheckNoError: false,
      allowSelfApproval: true,
    });
    await env.api.upsertSetting(
      "WORKSPACE_APPROVAL",
      { workspaceApproval: { rules: [] } },
      "value.workspace_approval",
    );
    await attachErrorConfig();
    const seeded = await seedReviewPlan(env.api, env.project, env.database, {
      prefix: "E2E Footer F",
      sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_footer_f_${Date.now()} TEXT;`,
      runChecks: true,
    });
    await waitForApprovalStatus(env.api, seeded.issueName, ["SKIPPED"]);
    planId = seeded.planId;
    await goReview(planId);
  });

  test("footer is the primary action; confirm sheet → deploy creates the rollout", async () => {
    await expect(
      page.getByText("Review approved, but plan checks failed"),
    ).toBeVisible({ timeout: 15_000 });
    await expect(planPage.bypassAndDeployAction).toBeVisible();

    await planPage.bypassAndDeployAction.click();
    const sheet = page.getByRole("dialog");
    await expect(sheet).toBeVisible();
    // A soft (non-mandatory) failed-check warning must be acknowledged.
    await sheet.getByRole("checkbox").check();
    await sheet.getByRole("button", { name: "Deploy", exact: true }).click();

    // Rollout created → the footer (only shown while !hasRollout) disappears.
    await expect(
      page.getByText("Review approved, but plan checks failed"),
    ).toBeHidden({ timeout: 20_000 });
    await expect(planPage.deploySection).toBeVisible();
  });
});

test.describe("Waiting-review bypass link is gated by requireIssueApproval (CUJ G1/G2)", () => {
  test.describe.configure({ mode: "serial" });
  let planId: string;

  test.beforeAll(async () => {
    await env.api.deletePolicy(env.project, "tag").catch(() => {});
    await env.api.updateProjectSettings(env.project, {
      requireIssueApproval: true,
      requirePlanCheckNoError: false,
      allowSelfApproval: true,
    });
    await env.api.upsertSetting(
      "WORKSPACE_APPROVAL",
      { workspaceApproval: { rules: [ONE_STEP_RULE] } },
      "value.workspace_approval",
    );
    const seeded = await seedReviewPlan(env.api, env.project, env.database, {
      prefix: "E2E Footer G",
      sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_footer_g_${Date.now()} TEXT;`,
      runChecks: true,
    });
    await waitForApprovalStatus(env.api, seeded.issueName, ["PENDING"]);
    planId = seeded.planId;
    await goReview(planId);
  });

  test("mandatory approval hides the link; optional approval shows it (same plan)", async () => {
    // G1 — requireIssueApproval=true: footer waits, NO bypass link.
    await expect(page.getByText("Waiting on review")).toBeVisible({
      timeout: 15_000,
    });
    await expect(planPage.bypassAndDeployAction).not.toBeVisible();

    // G2 — flip the project to optional approval; the muted link appears.
    await env.api.updateProjectSettings(env.project, {
      requireIssueApproval: false,
    });
    await goReview(planId);
    await expect(page.getByText("Waiting on review")).toBeVisible({
      timeout: 15_000,
    });
    await expect(planPage.bypassAndDeployAction).toBeVisible();
  });
});

test.describe("A mandatory project gate hard-blocks the bypass confirm", () => {
  test.describe.configure({ mode: "serial" });
  let planId: string;

  test.beforeAll(async () => {
    await env.api.deletePolicy(env.project, "tag").catch(() => {});
    await env.api.updateProjectSettings(env.project, {
      requireIssueApproval: false,
      requirePlanCheckNoError: true,
      allowSelfApproval: true,
    });
    await env.api.upsertSetting(
      "WORKSPACE_APPROVAL",
      { workspaceApproval: { rules: [] } },
      "value.workspace_approval",
    );
    await attachErrorConfig();
    const seeded = await seedReviewPlan(env.api, env.project, env.database, {
      prefix: "E2E Footer Gate",
      sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_footer_gate_${Date.now()} TEXT;`,
      runChecks: true,
    });
    await waitForApprovalStatus(env.api, seeded.issueName, ["SKIPPED"]);
    planId = seeded.planId;
    await goReview(planId);
  });

  test("confirm sheet reports the unmet gate and Deploy stays disabled", async () => {
    await expect(planPage.bypassAndDeployAction).toBeVisible({ timeout: 15_000 });
    await planPage.bypassAndDeployAction.click();
    const sheet = page.getByRole("dialog");
    await expect(sheet).toBeVisible();
    await expect(
      sheet.getByText("Required project gates are not met", { exact: false }),
    ).toBeVisible();
    // The mandatory gate cannot be acknowledged away — Deploy is disabled.
    await expect(
      sheet.getByRole("button", { name: "Deploy", exact: true }),
    ).toBeDisabled();
  });
});

// Regression guard for BYT-9745 (finding O5): the bypass confirm sheet used to
// render an empty bordered box under REVIEW for skipped-approval issues, while
// the main Review section showed "No approval required". Root cause:
// ReviewReadinessFooter rendered <ReviewApprovalFlow> (zero nodes when there
// are no roles) without the skipped-guard PlanReviewSection had. Fixed by
// #20662 — ReviewApprovalFlow now renders the skip note itself. This was a
// test.fail() lock until the fix landed; it now runs as a normal passing guard.
test.describe("confirm sheet shows the skipped state in its review box (BYT-9745)", () => {
  test.describe.configure({ mode: "serial" });
  let planId: string;

  test.beforeAll(async () => {
    await env.api.deletePolicy(env.project, "tag").catch(() => {});
    await env.api.updateProjectSettings(env.project, {
      requireIssueApproval: false,
      requirePlanCheckNoError: false,
      allowSelfApproval: true,
    });
    await env.api.upsertSetting(
      "WORKSPACE_APPROVAL",
      { workspaceApproval: { rules: [] } },
      "value.workspace_approval",
    );
    await attachErrorConfig();
    const seeded = await seedReviewPlan(env.api, env.project, env.database, {
      prefix: "E2E Footer O5",
      sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_footer_o5_${Date.now()} TEXT;`,
      runChecks: true,
    });
    await waitForApprovalStatus(env.api, seeded.issueName, ["SKIPPED"]);
    planId = seeded.planId;
    await goReview(planId);
  });

  test("confirm-sheet review box shows the skip note for a skipped approval", async () => {
    await expect(planPage.bypassAndDeployAction).toBeVisible({ timeout: 15_000 });
    await planPage.bypassAndDeployAction.click();
    const sheet = page.getByRole("dialog");
    await expect(sheet).toBeVisible();
    // The main section shows "No approval required"; post-fix (#20662) the
    // confirm sheet's review box shows it too. Scoped to the sheet so the main
    // section behind the scrim doesn't satisfy it. (Pre-fix the box was empty.)
    await expect(
      sheet.getByText("No approval required", { exact: false }),
    ).toBeVisible({ timeout: 5_000 });
  });
});
