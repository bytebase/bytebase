// Plan detail — AIO Review section (the whole sub-area in one file).
//
// Covers the issue-backed review workflow that now lives inline on Plan Detail
// (spec: docs/superpowers/specs/2026-06-12-aio-plan-review-section-design.md).
// Per the e2e convention (AGENTS.md: one <feature>.spec.ts per sub-area, never
// split by fixture), every facet of the Review section lives here as its own
// describe block — review actions, the rollout-readiness footer, and the
// adaptive approval-flow renderer:
//
//   Review actions / recovery / composer / timeline
//   - Approval flow + Review action render while PENDING (CUJ A)
//   - Approve via the Review popover → node green, action gone, rollout
//     auto-creates (CUJ B)
//   - Reject requires a comment; rejection banner + in-stream decision (CUJ C)
//   - Creator re-requests review without changes (CUJ D)
//   - Comment composer: draft survives collapse, post appends + re-collapses (CUJ E)
//   - Long-history timeline fold (torn separator + Show all) (CUJ J)
//   - Non-candidate sees no Review action but can still comment (permission boundary)
//   - BYT-9746 guard: "(edited)" marker shows in place after an inline edit
//
//   Rollout-readiness footer + bypass (readinessFooterState.ts)
//   - Approved/skipped + failed checks → "Bypass and deploy" → confirm sheet →
//     rollout created (CUJ F)
//   - Waiting-review bypass link gated by requireIssueApproval (G1 mandatory
//     hides it / G2 optional shows the muted link)
//   - A mandatory project gate (requirePlanCheckNoError) hard-blocks the
//     confirm-sheet Deploy (cannot be acknowledged away)
//   - BYT-9745 guard: confirm-sheet REVIEW box shows the skip note for
//     skipped-approval issues
//
//   Approval-flow renderer (approvalFlowLayout.ts)
//   - Long 5-step flow: constrained width folds approved/pending into chips
//     while the current step stays named; narrow width renders the full
//     vertical stepper with no chips (CUJ I)
//
// Each describe configures the project/workspace settings IT needs in its own
// beforeAll (approval rule, plan-check gate, review config), so the blocks are
// order-independent despite sharing the workspace-level WORKSPACE_APPROVAL
// setting. The file-level beforeAll only snapshots the originals + opens the
// shared browser; the file-level afterAll restores them.

import {
  test,
  expect,
  type Page,
  type BrowserContext,
} from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";
import { signInBrowserAs } from "../framework/sign-in";
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
    title: "E2E Review One-Step",
    description: "Single-step workspaceAdmin approval",
  },
};

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

async function goReview(planId: string): Promise<void> {
  await planPage.goto(projectId, planId);
  await planPage.dismissModals();
  await planPage.expandSection("Review");
}

// Configure a mandatory single/multi-step approval flow (the common case for
// the review-action + flow describes). Clears any review-config tag first so a
// prior describe's ERROR rule can't leak in. allowSelfApproval=true lets demo@
// (the issue creator) approve their own issue for single-admin tests.
async function setupApproval(rule: object): Promise<void> {
  await env.api.deletePolicy(env.project, "tag").catch(() => {});
  await env.api.updateProjectSettings(env.project, {
    requireIssueApproval: true,
    requirePlanCheckNoError: false,
    allowSelfApproval: true,
  });
  await env.api.upsertSetting(
    "WORKSPACE_APPROVAL",
    { workspaceApproval: { rules: [rule] } },
    "value.workspace_approval",
  );
}

// Attach a single ERROR-level COLUMN_NO_NULL rule to the project so a nullable
// column trips it. Tracked in createdReviewConfigs for afterAll cleanup; each
// footer describe that uses it clears the tag in its own beforeAll first.
async function attachErrorConfig(): Promise<void> {
  const id = `e2e-review-err-${Date.now()}`;
  const cfg = await env.api.upsertReviewConfig(id, "E2E Review ERROR", [
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

test.describe("Review action and approval flow (CUJ A)", () => {
  test.describe.configure({ mode: "serial" });
  let planId: string;

  test.beforeAll(async () => {
    await setupApproval(ONE_STEP_RULE);
    const seeded = await seedReviewPlan(env.api, env.project, env.database, {
      prefix: "E2E Review A",
      sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_rev_a_${Date.now()} TEXT;`,
    });
    await waitForApprovalStatus(env.api, seeded.issueName, ["PENDING"]);
    planId = seeded.planId;
    await goReview(planId);
  });

  test("renders the approval flow with the current step and Review action", async () => {
    await expect(planPage.reviewBadge("Under review")).toBeVisible({
      timeout: 15_000,
    });
    // The current step: role name + a "Current" badge (never folds).
    await expect(page.getByText("Workspace Admin").first()).toBeVisible();
    await expect(page.getByText("Current", { exact: true })).toBeVisible();
    // The header Review action is offered to the candidate (admin).
    await expect(planPage.reviewButton).toBeVisible();
  });
});

test.describe("Approve via the Review popover (CUJ B)", () => {
  test.describe.configure({ mode: "serial" });
  let planId: string;

  test.beforeAll(async () => {
    await setupApproval(ONE_STEP_RULE);
    const seeded = await seedReviewPlan(env.api, env.project, env.database, {
      prefix: "E2E Review B",
      sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_rev_b_${Date.now()} TEXT;`,
    });
    await waitForApprovalStatus(env.api, seeded.issueName, ["PENDING"]);
    planId = seeded.planId;
    await goReview(planId);
  });

  test("approving turns the badge to Approved and removes the Review action", async () => {
    await expect(planPage.reviewButton).toBeVisible({ timeout: 15_000 });
    await planPage.submitReview("Approve");
    // patchState updates the issue inline — badge flips, action disappears.
    await expect(planPage.reviewBadge("Approved")).toBeVisible({
      timeout: 15_000,
    });
    await expect(planPage.reviewButton).not.toBeVisible();
    // The reviewer's step turned green ("Approved by …").
    await planPage.expandSection("Review");
    await expect(page.getByText(/^Approved by /).first()).toBeVisible({
      timeout: 15_000,
    });
  });
});

test.describe("Reject requires a comment; banner + in-stream decision (CUJ C)", () => {
  test.describe.configure({ mode: "serial" });
  let planId: string;

  test.beforeAll(async () => {
    await setupApproval(ONE_STEP_RULE);
    const seeded = await seedReviewPlan(env.api, env.project, env.database, {
      prefix: "E2E Review C",
      sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_rev_c_${Date.now()} TEXT;`,
    });
    await waitForApprovalStatus(env.api, seeded.issueName, ["PENDING"]);
    planId = seeded.planId;
    await goReview(planId);
  });

  test("reject is disabled until a comment is entered, then pins the rejection banner", async () => {
    const reason = "Add a default value before this can ship";
    await planPage.reviewButton.click();
    await page.getByText("Submit feedback and request changes").waitFor();
    await page.locator('label:has-text("Reject")').click();
    // Reject requires a non-empty comment.
    await expect(planPage.reviewSubmitButton).toBeDisabled();
    await planPage.reviewPopoverEditor.fill(reason);
    await expect(planPage.reviewSubmitButton).toBeEnabled();
    await planPage.reviewSubmitButton.click();

    // The rejection banner pins above the timeline with the reason.
    await expect(planPage.rejectionBanner).toBeVisible({ timeout: 15_000 });
    await expect(page.getByText(reason).first()).toBeVisible();
    // The decision also lands in-stream as a permanent timeline row.
    await expect(page.getByText("rejected issue").first()).toBeVisible();
    // Badge flips, action disappears, footer announces the block.
    await expect(planPage.reviewBadge("Rejected")).toBeVisible();
    await expect(planPage.reviewButton).not.toBeVisible();
    await expect(
      page.getByText("Blocked by the rejected review", { exact: false }),
    ).toBeVisible();
  });
});

test.describe("Creator re-requests review without changes (CUJ D)", () => {
  test.describe.configure({ mode: "serial" });
  let planId: string;

  test.beforeAll(async () => {
    await setupApproval(ONE_STEP_RULE);
    const seeded = await seedReviewPlan(env.api, env.project, env.database, {
      prefix: "E2E Review D",
      sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_rev_d_${Date.now()} TEXT;`,
    });
    await waitForApprovalStatus(env.api, seeded.issueName, ["PENDING"]);
    // Reach the rejected state via API (setup), then verify recovery via UI.
    await env.api.rejectIssue(seeded.issueName, "Please address the feedback");
    planId = seeded.planId;
    await goReview(planId);
  });

  test("inline re-request restarts review and restores the Review action", async () => {
    await expect(planPage.rejectionBanner).toBeVisible({ timeout: 15_000 });
    await planPage.reRequestButton.click();
    // Banner clears, status returns to under-review, the action comes back.
    await expect(planPage.rejectionBanner).toBeHidden({ timeout: 15_000 });
    await expect(planPage.reviewBadge("Under review")).toBeVisible();
    await expect(planPage.reviewButton).toBeVisible();
  });
});

test.describe("Comment composer: draft persistence + post (CUJ E)", () => {
  test.describe.configure({ mode: "serial" });
  let planId: string;

  test.beforeAll(async () => {
    await setupApproval(ONE_STEP_RULE);
    const seeded = await seedReviewPlan(env.api, env.project, env.database, {
      prefix: "E2E Review E",
      sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_rev_e_${Date.now()} TEXT;`,
    });
    await waitForApprovalStatus(env.api, seeded.issueName, ["PENDING"]);
    planId = seeded.planId;
    await goReview(planId);
  });

  test("draft survives collapse; posting appends the comment and re-collapses", async () => {
    const draft = "Draft that should survive collapse";
    const final = "Looks good — ready to ship";

    await planPage.composerTrigger.click();
    await expect(planPage.composerEditor).toBeVisible();
    await planPage.composerEditor.fill(draft);

    // Collapse via the composer's Cancel, then re-expand → draft restored.
    await page.getByRole("button", { name: "Cancel", exact: true }).click();
    await expect(planPage.composerTrigger).toBeVisible();
    await planPage.composerTrigger.click();
    await expect(planPage.composerEditor).toHaveValue(draft);

    // Replace and post.
    await planPage.composerEditor.fill(final);
    await planPage.composerSubmitButton.click();

    await expect(page.getByText(final).first()).toBeVisible({ timeout: 15_000 });
    // Composer re-collapses to its trigger after posting.
    await expect(planPage.composerTrigger).toBeVisible();
  });
});

test.describe("Long-history timeline fold (CUJ J)", () => {
  test.describe.configure({ mode: "serial" });
  let planId: string;

  test.beforeAll(async () => {
    await setupApproval(ONE_STEP_RULE);
    const seeded = await seedReviewPlan(env.api, env.project, env.database, {
      prefix: "E2E Review J",
      sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_rev_j_${Date.now()} TEXT;`,
    });
    await waitForApprovalStatus(env.api, seeded.issueName, ["PENDING"]);
    // 12 comments + 2 synthetic head rows = 14 entries → folds (head 3 + tail 3,
    // 8 hidden). NOTE: the shipped fold hides ALL middle rows including user
    // comments (foldTimeline.test.ts) — the design doc's "comments stay visible
    // as islands" was not implemented; this test locks the shipped behavior.
    for (let i = 1; i <= 12; i++) {
      await env.api.createIssueComment(seeded.issueName, `Timeline comment ${i}`);
    }
    planId = seeded.planId;
    await goReview(planId);
  });

  test("middle entries collapse behind a torn separator until Show all", async () => {
    // Head shows comment 1; tail shows comment 12; middle (comment 5) hidden.
    await expect(page.getByText("Timeline comment 1", { exact: true })).toBeVisible({
      timeout: 15_000,
    });
    await expect(
      page.getByText("Timeline comment 12", { exact: true }),
    ).toBeVisible();
    await expect(page.getByText(/\d+ hidden events?/)).toBeVisible();
    await expect(
      page.getByText("Timeline comment 5", { exact: true }),
    ).toBeHidden();

    await page.getByText("Show all").click();
    await expect(
      page.getByText("Timeline comment 5", { exact: true }),
    ).toBeVisible();
  });
});

test.describe("Permission boundary: non-candidate cannot review but can comment", () => {
  test.describe.configure({ mode: "serial" });
  let dbaContext: BrowserContext;
  let dbaPage: Page;
  let dbaPlanPage: PlanDetailPage;
  let planId: string;

  test.beforeAll(async ({ browser }) => {
    await setupApproval(ONE_STEP_RULE);
    const seeded = await seedReviewPlan(env.api, env.project, env.database, {
      prefix: "E2E Review Perm",
      sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_rev_perm_${Date.now()} TEXT;`,
    });
    await waitForApprovalStatus(env.api, seeded.issueName, ["PENDING"]);
    planId = seeded.planId;

    // dba1 is workspaceDBA — NOT a candidate of the workspaceAdmin step.
    await signInBrowserAs(
      browser,
      env.baseURL,
      "dba1@example.com",
      "12345678",
      ".auth/dba-review.json",
    );
    dbaContext = await browser.newContext({
      storageState: ".auth/dba-review.json",
    });
    dbaPage = await dbaContext.newPage();
    dbaPlanPage = new PlanDetailPage(dbaPage, env.baseURL);
    await dbaPlanPage.goto(projectId, planId);
    await dbaPlanPage.dismissModals();
    await dbaPlanPage.expandSection("Review");
  });

  test.afterAll(async () => {
    await dbaContext?.close();
  });

  test("a non-candidate sees no Review action but the composer is available", async () => {
    await expect(dbaPlanPage.reviewBadge("Under review")).toBeVisible({
      timeout: 15_000,
    });
    await expect(dbaPlanPage.reviewButton).not.toBeVisible();
    await expect(dbaPlanPage.composerTrigger).toBeVisible();
  });
});

// Regression guard for BYT-9746 (was finding O7): editing your own comment
// inline must show the "(edited)" marker immediately, not only after a reload.
// The original bug: stores/app/issueComment.ts updateIssueComment() patched
// { ...comment, comment } and discarded the server response, so updateTime was
// never bumped and isEdited (createdTs !== updatedTs) stayed false until a
// refetch. Fixed on main by #20649 (updateIssueComment now stores the RPC
// response). This was a test.fail() lock until the fix landed; it now runs as
// a normal passing guard so a re-regression fails loudly.
test.describe("inline comment edit shows the edited marker in place (BYT-9746)", () => {
  test.describe.configure({ mode: "serial" });
  let planId: string;
  const original = `O7 edit me ${Date.now()}`;

  test.beforeAll(async () => {
    await setupApproval(ONE_STEP_RULE);
    const seeded = await seedReviewPlan(env.api, env.project, env.database, {
      prefix: "E2E Review O7",
      sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_rev_o7_${Date.now()} TEXT;`,
    });
    await waitForApprovalStatus(env.api, seeded.issueName, ["PENDING"]);
    await env.api.createIssueComment(seeded.issueName, original);
    planId = seeded.planId;
    await goReview(planId);
  });

  test("(edited) marker appears immediately after an inline edit", async () => {
    const edited = `${original} (edited inline)`;
    const row = page.locator("li", { hasText: original });
    await expect(row).toBeVisible({ timeout: 15_000 });
    // The pencil edit affordance (own comment, no text label → target the icon).
    await row.locator("button:has(svg.lucide-pencil)").click();
    const editor = page.locator("textarea[placeholder='Leave a comment...']");
    await editor.fill(edited);
    await page.getByRole("button", { name: "Save", exact: true }).click();

    // Save succeeded once the new text renders — proves the assertion below
    // discriminates the marker, not an unfinished save.
    await expect(page.getByText(edited).first()).toBeVisible({ timeout: 15_000 });
    // Post-fix (#20649): the "(edited)" marker appears immediately, in place.
    await expect(page.getByText("(edited)").first()).toBeVisible({
      timeout: 5_000,
    });
  });
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
      prefix: "E2E Review F",
      sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_rev_f_${Date.now()} TEXT;`,
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
      prefix: "E2E Review G",
      sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_rev_g_${Date.now()} TEXT;`,
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
      prefix: "E2E Review Gate",
      sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_rev_gate_${Date.now()} TEXT;`,
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
      prefix: "E2E Review O5",
      sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_rev_o5_${Date.now()} TEXT;`,
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

test.describe("Long approval flow adaptive rendering (CUJ I)", () => {
  test.describe.configure({ mode: "serial" });

  test.beforeAll(async () => {
    await setupApproval(FIVE_STEP_RULE);
    const seeded = await seedReviewPlan(env.api, env.project, env.database, {
      prefix: "E2E Review Flow",
      sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_rev_flow_${Date.now()} TEXT;`,
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

  test.afterAll(async () => {
    await page?.setViewportSize({ width: 1280, height: 720 }).catch(() => {});
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
