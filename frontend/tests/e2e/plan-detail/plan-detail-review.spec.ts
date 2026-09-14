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
//   - Inline comment threads: seeded thread in editor + timeline, gutter
//     creation, reply / resolve / reopen, View in Statement (CUJ K)
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
    const seeded = await seedReviewPlan(env, page, {
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
    const seeded = await seedReviewPlan(env, page, {
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
    const seeded = await seedReviewPlan(env, page, {
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
    const seeded = await seedReviewPlan(env, page, {
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
    const seeded = await seedReviewPlan(env, page, {
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
    const seeded = await seedReviewPlan(env, page, {
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
    const seeded = await seedReviewPlan(env, page, {
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

// Inline comment threads (design doc: Plan Review Comment and Thread UI/UX).
// A thread anchors to whole lines of the spec's saved sheet; it renders as a
// gutter marker + expanded card in the statement editor and as a card with
// its recorded context in the Review Activity timeline.
// Gated off with the feature: inlineThreadsEnabled() is false in the embedded
// binary this suite runs against, so the thread UI is absent by design.
// Re-enable together with the gate in src/utils/featureGates.ts.
test.describe.skip("Inline comment threads (CUJ K)", () => {
  test.describe.configure({ mode: "serial" });
  let planId: string;
  let issueName: string;
  let specId: string;
  let sheetSha256: string;
  const seededRoot = `K seeded root ${Date.now()}`;
  const seededReply = `K seeded reply ${Date.now()}`;
  const createdRoot = `K gutter root ${Date.now()}`;
  const typedReply = `K typed reply ${Date.now()}`;

  test.beforeAll(async () => {
    await setupApproval(ONE_STEP_RULE);
    const suffix = Date.now();
    const seeded = await seedReviewPlan(env, page, {
      prefix: "E2E Review K",
      sql: [
        `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_rev_k1_${suffix} TEXT;`,
        `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_rev_k2_${suffix} TEXT;`,
        `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_rev_k3_${suffix} TEXT;`,
        `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_rev_k4_${suffix} TEXT;`,
      ].join("\n"),
    });
    await waitForApprovalStatus(env.api, seeded.issueName, ["PENDING"]);
    planId = seeded.planId;
    issueName = seeded.issueName;
    // The anchor names the spec and the content hash embedded in the sheet name.
    const plan = await env.api.getPlan(seeded.planName);
    const spec = plan.specs?.[0];
    specId = spec?.id ?? "";
    sheetSha256 = spec?.changeDatabaseConfig?.sheet?.split("/").pop() ?? "";
    expect(specId).not.toBe("");
    expect(sheetSha256).toMatch(/^[0-9a-f]{64}$/);
    const root = await env.api.createIssueComment(issueName, seededRoot, {
      statementAnchor: { spec: specId, sheetSha256, startLine: 2, endLine: 3 },
    });
    await env.api.createIssueComment(issueName, seededReply, { root: root.name });
    await goReview(planId);
    await planPage.expandSection("Changes");
  });

  test("a seeded thread shows as a marker + expanded card in the editor and as an anchored card in the timeline", async () => {
    // Editor: one marker on the last anchored line, the unresolved thread expanded.
    await expect(planPage.threadMarkers).toHaveCount(1, { timeout: 15_000 });
    const editorCard = planPage.threadCardIn("changes", seededRoot);
    await expect(editorCard).toBeVisible();
    await expect(editorCard).toContainText(seededReply);

    // Timeline: the same thread with its recorded context.
    const timelineCard = planPage.threadCardIn("review", seededRoot);
    await expect(timelineCard).toBeVisible();
    await expect(timelineCard).toContainText(seededReply);
    // The recorded statement downloads once the card nears the viewport.
    await timelineCard.scrollIntoViewIfNeeded();
    const anchor = timelineCard.getByTestId("statement-anchor");
    await expect(anchor).toHaveAttribute("data-anchor-state", "CURRENT", { timeout: 15_000 });
    await expect(anchor).toContainText("Lines 2–3");
    await expect(anchor).toContainText("e2e_rev_k2_");
    await expect(anchor.getByRole("button", { name: "View in Statement" })).toBeVisible();
    // The reply never renders as its own timeline row.
    await expect(
      page.locator("#plan-phase-review [data-testid='comment-thread']"),
    ).toHaveCount(1);
  });

  // One journey: create a thread from the gutter, then reply, resolve,
  // reopen, and jump back into the editor. The final assertion, that the
  // gutter-created thread collapses when View in Statement expands the
  // seeded one, depends on the created thread being open, so the steps stay
  // in one test.
  test("a gutter-created thread, then reply, resolve, reopen, and View in Statement round-trip between the surfaces", async () => {
    await planPage.hoverStatementLine(1);
    await planPage.clickAddThreadGlyph();
    await expect(planPage.inlineComposer).toBeVisible();
    await expect(planPage.inlineComposer).toContainText("Add a comment on line 1");
    await planPage.inlineComposerEditor.fill(createdRoot);
    await planPage.inlineComposerPublishButton.click();

    // The composer closes; the new thread expands in place of the seeded one
    // and both markers now sit in the gutter.
    await expect(planPage.inlineComposer).not.toBeVisible({ timeout: 15_000 });
    await expect(planPage.threadCardIn("changes", createdRoot)).toBeVisible();
    await expect(planPage.threadMarkers).toHaveCount(2);
    const createdAnchor = planPage
      .threadCardIn("review", createdRoot)
      .getByTestId("statement-anchor");
    await expect(createdAnchor).toBeVisible({ timeout: 15_000 });
    await expect(createdAnchor).toContainText("Line 1");
    // The recorded statement downloads once the card nears the viewport.
    await createdAnchor.scrollIntoViewIfNeeded();
    await expect(createdAnchor).toContainText("e2e_rev_k1_", { timeout: 15_000 });

    const timelineCard = planPage.threadCardIn("review", seededRoot);
    await timelineCard.scrollIntoViewIfNeeded();
    await timelineCard.getByRole("button", { name: "Reply..." }).click();
    await timelineCard.locator("textarea[placeholder='Reply...']").fill(typedReply);
    await timelineCard.getByRole("button", { name: "Reply", exact: true }).click();
    await expect(timelineCard).toContainText(typedReply, { timeout: 15_000 });

    // Resolving collapses the timeline card to its summary row.
    await timelineCard.getByRole("button", { name: "Resolve", exact: true }).click();
    const resolvedCard = planPage.threadCardIn("review", seededRoot);
    await expect(resolvedCard).toHaveAttribute("data-thread-state", "resolved", { timeout: 15_000 });
    await expect(resolvedCard).toContainText("Resolved");
    await expect(resolvedCard).toContainText("2 replies");
    await expect(resolvedCard).toContainText(seededRoot);
    await expect(resolvedCard.getByTestId("thread-comment")).toHaveCount(0);

    // Expanding keeps it resolved and offers Reopen.
    await resolvedCard.getByRole("button", { name: /Resolved/ }).click();
    await expect(resolvedCard).toContainText(seededRoot);
    await resolvedCard.getByRole("button", { name: "Reopen", exact: true }).click();
    const reopened = planPage.threadCardIn("review", seededRoot);
    await expect(reopened.getByRole("button", { name: "Resolve", exact: true })).toBeVisible({
      timeout: 15_000,
    });

    // View in Statement lands on the anchored lines with the seeded thread
    // expanded, collapsing the thread created above.
    await reopened.getByRole("button", { name: "View in Statement" }).click();
    const editorCard = planPage.threadCardIn("changes", seededRoot);
    await expect(editorCard).toBeVisible({ timeout: 15_000 });
    await expect(editorCard).toContainText(typedReply);
    await expect(planPage.threadCardIn("changes", createdRoot)).not.toBeVisible();
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
    const seeded = await seedReviewPlan(env, page, {
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
    const seeded = await seedReviewPlan(env, page, {
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
    const seeded = await seedReviewPlan(env, page, {
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
    const seeded = await seedReviewPlan(env, page, {
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
    const seeded = await seedReviewPlan(env, page, {
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
    const seeded = await seedReviewPlan(env, page, {
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

// Inline comment threads, second pass: several composers at once, the
// reply composer's thread-state checkbox, the editor walker, and the
// unresolved counts on the change tab and the Review summary.
// Gated off with the feature: inlineThreadsEnabled() is false in the embedded
// binary this suite runs against, so the thread UI is absent by design.
// Re-enable together with the gate in src/utils/featureGates.ts.
test.describe.skip("Inline comment threads: composers, checkbox, walker, counts (CUJ L)", () => {
  test.describe.configure({ mode: "serial" });
  let planId: string;
  let issueName: string;
  let specId: string;
  let sheetSha256: string;
  const stamp = Date.now();
  const rootA = `L root A line 2 ${stamp}`;
  const rootB = `L root B lines 4-5 ${stamp}`;
  const rootResolved = `L resolved root line 6 ${stamp}`;
  const rootStale = `L stale root ${stamp}`;
  const draftOne = `L draft one ${stamp}`;
  const draftThree = `L draft three ${stamp}`;
  const replyResolving = `L resolving reply ${stamp}`;
  const replyReopening = `L reopening reply ${stamp}`;

  // The phase summary line renders only while the section is collapsed.
  const expectReviewSummary = async (text: string | RegExp) => {
    await planPage.setPhaseExpanded("review", false);
    await expect(planPage.reviewPhase).toContainText(text, { timeout: 15_000 });
    await planPage.setPhaseExpanded("review", true);
  };

  test.beforeAll(async () => {
    await setupApproval(ONE_STEP_RULE);
    const seeded = await seedReviewPlan(env, page, {
      prefix: "E2E Review L",
      sql: [1, 2, 3, 4, 5, 6]
        .map((n) => `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_rev_l${n}_${stamp} TEXT;`)
        .join("\n"),
    });
    await waitForApprovalStatus(env.api, seeded.issueName, ["PENDING"]);
    planId = seeded.planId;
    issueName = seeded.issueName;
    const plan = await env.api.getPlan(seeded.planName);
    const spec = plan.specs?.[0];
    specId = spec?.id ?? "";
    sheetSha256 = spec?.changeDatabaseConfig?.sheet?.split("/").pop() ?? "";
    expect(specId).not.toBe("");
    expect(sheetSha256).toMatch(/^[0-9a-f]{64}$/);
    const anchor = (startLine: number, endLine: number, sha = sheetSha256) => ({
      statementAnchor: { spec: specId, sheetSha256: sha, startLine, endLine },
    });
    await env.api.createIssueComment(issueName, rootA, anchor(2, 2));
    await env.api.createIssueComment(issueName, rootB, anchor(4, 5));
    const resolved = await env.api.createIssueComment(issueName, rootResolved, anchor(6, 6));
    await env.api.setIssueCommentThreadState(resolved.name, "RESOLVED");
    // Anchored to an earlier statement of this change whose line has no
    // counterpart in the displayed one: unresolved, but not placed here.
    const staleSheet = await env.api.createSheet(env.project, "SELECT 1;\n");
    const staleSha256 = staleSheet.split("/").pop() ?? "";
    expect(staleSha256).toMatch(/^[0-9a-f]{64}$/);
    await env.api.createIssueComment(issueName, rootStale, anchor(1, 1, staleSha256));
    await goReview(planId);
    await planPage.expandSection("Changes");
  });

  test("the change tab counts placed unresolved threads; the Review summary counts all of them", async () => {
    // Placed and unresolved: A and B. The resolved thread and the stale one
    // stay out of the tab and the walker; the stale one still counts for
    // the plan-wide summary.
    await expect(planPage.specUnresolvedCounts).toHaveCount(1, { timeout: 15_000 });
    await expect(planPage.specUnresolvedCounts).toHaveText("2");
    await expectReviewSummary("3 unresolved threads");
    await expect(planPage.threadWalker).toContainText("2");
    await expect(planPage.threadWalker).toHaveAttribute(
      "title",
      "2 unresolved threads on this version · 1 more not shown on this version",
    );
    // Markers: A, B, and the resolved one; the stale thread has none.
    await expect(planPage.threadMarkers).toHaveCount(3);
  });

  test("the walker steps through unresolved threads in editor order and wraps", async () => {
    // The first unresolved thread opens by default.
    await expect(planPage.threadCardIn("changes", rootA)).toBeVisible();
    await planPage.threadWalkerNext.click();
    await expect(planPage.threadCardIn("changes", rootB)).toBeVisible({ timeout: 10_000 });
    await expect(planPage.threadCardIn("changes", rootA)).not.toBeVisible();
    await planPage.threadWalkerNext.click();
    await expect(planPage.threadCardIn("changes", rootA)).toBeVisible({ timeout: 10_000 });
    await planPage.threadWalkerPrevious.click();
    await expect(planPage.threadCardIn("changes", rootB)).toBeVisible({ timeout: 10_000 });
    // The resolved thread is never a stop.
    await expect(planPage.threadCardIn("changes", rootResolved)).not.toBeVisible();
  });

  test("several composers stay open with their own drafts, and each closes on its own", async () => {
    const first = await planPage.openInlineComposerOn(1);
    const firstEditor = first.locator("textarea");
    const firstPublish = first.getByRole("button", { name: "Publish", exact: true });
    await expect(firstPublish).toBeDisabled();
    await firstEditor.fill(draftOne);
    await expect(firstPublish).toBeEnabled();

    // The add tile is gone on a line that already carries a form.
    await planPage.hoverStatementLine(1);
    await expect(planPage.addThreadGlyph).toHaveCount(0);

    const third = await planPage.openInlineComposerOn(3);
    await expect(planPage.inlineComposer).toHaveCount(2);
    await third.locator("textarea").fill(draftThree);
    await expect(firstEditor).toHaveValue(draftOne);

    // Cancel closes only the form it belongs to.
    await third.getByRole("button", { name: "Cancel" }).click();
    await expect(planPage.inlineComposer).toHaveCount(1);
    await expect(firstEditor).toHaveValue(draftOne);

    // Publishing the remaining form creates its thread and opens it.
    await firstPublish.click();
    await expect(planPage.inlineComposer).toHaveCount(0, { timeout: 15_000 });
    await expect(planPage.threadCardIn("changes", draftOne)).toBeVisible();
    await expect(planPage.threadMarkers).toHaveCount(4);
    await expect(planPage.specUnresolvedCounts).toHaveText("3");
    await expect(planPage.threadWalker).toContainText("3");
    await expect(planPage.threadCardIn("review", draftOne)).toBeVisible({ timeout: 15_000 });
  });

  test("the reply composer's checkbox resolves with the reply and reopens again", async () => {
    const card = planPage.threadCardIn("review", rootA);
    await card.scrollIntoViewIfNeeded();
    await card.getByRole("button", { name: "Reply..." }).click();
    const reply = card.getByRole("button", { name: "Reply", exact: true });
    const editor = card.locator("textarea[placeholder='Reply...']");
    await expect(reply).toBeDisabled();

    // An open thread offers Resolve thread, unchecked.
    const resolveBox = planPage.threadStateCheckbox(card, "Resolve thread");
    await expect(resolveBox).toHaveAttribute("aria-checked", "false");
    await editor.fill(replyResolving);
    await resolveBox.click();
    await expect(resolveBox).toHaveAttribute("aria-checked", "true");
    await reply.click();

    const resolvedCard = planPage.threadCardIn("review", rootA);
    await expect(resolvedCard).toHaveAttribute("data-thread-state", "resolved", { timeout: 15_000 });
    await expect(resolvedCard).toContainText("1 reply");
    // The counts follow: A left the placed-unresolved set.
    await expect(planPage.specUnresolvedCounts).toHaveText("2");
    await expect(planPage.threadWalker).toContainText("2");
    await expectReviewSummary("3 unresolved threads");

    // Expand and reply again: a resolved thread offers Reopen thread,
    // checked by default, so an untouched reply reopens it.
    await resolvedCard.getByRole("button", { name: /Resolved/ }).click();
    await expect(resolvedCard).toContainText(replyResolving);
    await resolvedCard.getByRole("button", { name: "Reply..." }).click();
    const reopenBox = planPage.threadStateCheckbox(resolvedCard, "Reopen thread");
    await expect(reopenBox).toHaveAttribute("aria-checked", "true");
    await resolvedCard.locator("textarea[placeholder='Reply...']").fill(replyReopening);
    await resolvedCard.getByRole("button", { name: "Reply", exact: true }).click();
    const reopenedCard = planPage.threadCardIn("review", rootA);
    await expect(reopenedCard).toHaveAttribute("data-thread-state", "open", { timeout: 15_000 });
    await expect(reopenedCard).toContainText(replyReopening);
    await expect(planPage.specUnresolvedCounts).toHaveText("3");

    // Unticking it keeps the thread resolved.
    await reopenedCard.getByRole("button", { name: "Resolve", exact: true }).click();
    await expect(reopenedCard).toHaveAttribute("data-thread-state", "resolved", { timeout: 15_000 });
    await reopenedCard.getByRole("button", { name: /Resolved/ }).click();
    await reopenedCard.getByRole("button", { name: "Reply..." }).click();
    await planPage.threadStateCheckbox(reopenedCard, "Reopen thread").click();
    await reopenedCard.locator("textarea[placeholder='Reply...']").fill(`${replyReopening} still`);
    await reopenedCard.getByRole("button", { name: "Reply", exact: true }).click();
    await expect(reopenedCard).toContainText(`${replyReopening} still`, { timeout: 15_000 });
    await expect(reopenedCard).toHaveAttribute("data-thread-state", "resolved");
    await expect(planPage.specUnresolvedCounts).toHaveText("2");

    // Leave A open for the next journey.
    await reopenedCard.getByRole("button", { name: "Reopen", exact: true }).click();
    await expect(planPage.threadCardIn("review", rootA)).toHaveAttribute("data-thread-state", "open", {
      timeout: 15_000,
    });
    await expect(planPage.specUnresolvedCounts).toHaveText("3");
  });

  test("resolving every placed thread removes the walker and the tab count, and reopening restores them", async () => {
    for (const rootText of [rootA, rootB, draftOne]) {
      const card = planPage.threadCardIn("review", rootText);
      await card.scrollIntoViewIfNeeded();
      await card.getByRole("button", { name: "Resolve", exact: true }).click();
      await expect(card).toHaveAttribute("data-thread-state", "resolved", { timeout: 15_000 });
    }
    await expect(planPage.threadWalker).toHaveCount(0);
    await expect(planPage.specUnresolvedCounts).toHaveCount(0);
    // Only the stale thread remains open.
    await expectReviewSummary(/1 unresolved thread(?!s)/);

    const card = planPage.threadCardIn("review", rootB);
    await card.getByRole("button", { name: /Resolved/ }).click();
    await card.getByRole("button", { name: "Reopen", exact: true }).click();
    await expect(planPage.threadWalker).toContainText("1", { timeout: 15_000 });
    await expect(planPage.specUnresolvedCounts).toHaveText("1");
    await expectReviewSummary("2 unresolved threads");
  });
});

// A reader who may reply but not resolve sees neither the standalone
// Resolve action nor the reply composer's state checkbox. No predefined role
// separates the two comment permissions, so the test provisions its own.
// Gated off with the feature: inlineThreadsEnabled() is false in the embedded
// binary this suite runs against, so the thread UI is absent by design.
// Re-enable together with the gate in src/utils/featureGates.ts.
test.describe.skip("Inline comment threads as a reply-only reader (CUJ L, restricted)", () => {
  const stamp = Date.now();
  const readerEmail = `e2e-reply-only-${stamp}@example.com`;
  const readerPassword = "12345678";
  const readerAuthFile = ".auth/plan-reply-only.json";
  const roleId = `e2e-reply-only-${stamp}`;
  let roleName = "";
  let readerContext: BrowserContext | undefined;
  let readerPlanPage: PlanDetailPage;
  let planId: string;
  const rootText = `L restricted root ${stamp}`;

  test.beforeAll(async ({ browser }) => {
    await setupApproval(ONE_STEP_RULE);
    const seeded = await seedReviewPlan(env, page, {
      prefix: "E2E Review L restricted",
      sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_rev_lr_${stamp} TEXT;`,
    });
    await waitForApprovalStatus(env.api, seeded.issueName, ["PENDING"]);
    planId = seeded.planId;
    const plan = await env.api.getPlan(seeded.planName);
    const spec = plan.specs?.[0];
    const sheetSha256 = spec?.changeDatabaseConfig?.sheet?.split("/").pop() ?? "";
    await env.api.createIssueComment(seeded.issueName, rootText, {
      statementAnchor: { spec: spec?.id ?? "", sheetSha256, startLine: 1, endLine: 1 },
    });

    // The viewer role plus what the plan page needs, minus comment update.
    const role = await env.api.createRole(roleId, "E2E reply-only reader", [
      "bb.projects.get",
      "bb.projects.getIamPolicy",
      "bb.databases.get",
      "bb.databases.list",
      "bb.databases.getSchema",
      "bb.instances.get",
      "bb.plans.get",
      "bb.plans.list",
      "bb.planCheckRuns.get",
      "bb.taskRuns.list",
      "bb.rollouts.get",
      "bb.rollouts.list",
      "bb.sheets.get",
      "bb.issues.get",
      "bb.issues.list",
      "bb.issueComments.list",
      "bb.issueComments.create",
    ]);
    roleName = role.name;
    await env.api.createUser(readerEmail, readerPassword, "E2E reply-only reader");
    await env.api.appendProjectBinding(env.project, roleName, [`user:${readerEmail}`]);
    await signInBrowserAs(browser, env.baseURL, readerEmail, readerPassword, readerAuthFile);
    readerContext = await browser.newContext({ storageState: readerAuthFile });
    readerPlanPage = new PlanDetailPage(await readerContext.newPage(), env.baseURL);
  });

  // DeleteRole refuses a role a binding still references, so drop the
  // binding and the reader first; the throwaway workspace must not carry
  // this role or user into later suites.
  test.afterAll(async () => {
    await readerContext?.close();
    if (roleName) {
      const policy = await env.api.getProjectIamPolicy(env.project);
      policy.bindings = policy.bindings.filter((binding) => binding.role !== roleName);
      await env.api.setProjectIamPolicy(env.project, policy);
    }
    await env.api.deleteUser(readerEmail).catch(() => {});
    if (roleName) await env.api.deleteRole(roleName);
  });

  test("the reader can reply but sees no Resolve action and no state checkbox", async () => {
    await readerPlanPage.goto(projectId, planId);
    await readerPlanPage.dismissModals();
    await readerPlanPage.expandSection("Review");
    const card = readerPlanPage.threadCardIn("review", rootText);
    await expect(card).toBeVisible({ timeout: 15_000 });
    await expect(card.getByRole("button", { name: "Resolve", exact: true })).toHaveCount(0);
    await card.getByRole("button", { name: "Reply..." }).click();
    await expect(card.getByRole("checkbox")).toHaveCount(0);
    await expect(card.getByRole("button", { name: "Reply", exact: true })).toBeDisabled();
  });
});

// Inline comment threads, third pass: a range selected from the gutter, two
// threads sharing one marker, the walker's announcement and its retreat
// behind the find widget, and a narrow viewport.
// Gated off with the feature: inlineThreadsEnabled() is false in the embedded
// binary this suite runs against, so the thread UI is absent by design.
// Re-enable together with the gate in src/utils/featureGates.ts.
test.describe.skip("Inline comment threads: ranges, shared markers, walker details, narrow view (CUJ M)", () => {
  test.describe.configure({ mode: "serial" });
  let planId: string;
  let issueName: string;
  let specId: string;
  let sheetSha256: string;
  let laterRootName = "";
  const stamp = Date.now();
  const earlierRoot = `M earlier root lines 4-5 ${stamp}`;
  const laterRoot = `M later root line 5 ${stamp}`;
  const rangeRoot = `M range root lines 2-3 ${stamp}`;

  test.beforeAll(async () => {
    await setupApproval(ONE_STEP_RULE);
    const seeded = await seedReviewPlan(env, page, {
      prefix: "E2E Review M",
      sql: [1, 2, 3, 4, 5, 6]
        .map((n) => `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_rev_m${n}_${stamp} TEXT;`)
        .join("\n"),
    });
    await waitForApprovalStatus(env.api, seeded.issueName, ["PENDING"]);
    planId = seeded.planId;
    issueName = seeded.issueName;
    const plan = await env.api.getPlan(seeded.planName);
    const spec = plan.specs?.[0];
    specId = spec?.id ?? "";
    sheetSha256 = spec?.changeDatabaseConfig?.sheet?.split("/").pop() ?? "";
    expect(specId).not.toBe("");
    expect(sheetSha256).toMatch(/^[0-9a-f]{64}$/);
    const anchor = (startLine: number, endLine: number) => ({
      statementAnchor: { spec: specId, sheetSha256, startLine, endLine },
    });
    // Both end on line 5, so they share its marker.
    await env.api.createIssueComment(issueName, earlierRoot, anchor(4, 5));
    const later = await env.api.createIssueComment(issueName, laterRoot, anchor(5, 5));
    laterRootName = later.name;
    await goReview(planId);
    await planPage.expandSection("Changes");
  });

  test.afterAll(async () => {
    await page?.setViewportSize({ width: 1280, height: 720 }).catch(() => {});
  });

  test("dragging across line numbers starts a thread on that range", async () => {
    await planPage.selectStatementLines(2, 3);
    await planPage.clickAddThreadGlyph();
    const composer = planPage.inlineComposer;
    await expect(composer).toBeVisible();
    await expect(composer).toContainText("Add a comment on lines 2 to 3");
    await planPage.inlineComposerEditor.fill(rangeRoot);
    await planPage.inlineComposerPublishButton.click();
    await expect(composer).not.toBeVisible({ timeout: 15_000 });
    await expect(planPage.threadCardIn("changes", rangeRoot)).toBeVisible();
    // One marker on line 3 for the new thread, one shared marker on line 5.
    await expect(planPage.threadMarkers).toHaveCount(2);
    const anchorContext = planPage.threadCardIn("review", rangeRoot).getByTestId("statement-anchor");
    await anchorContext.scrollIntoViewIfNeeded();
    await expect(anchorContext).toContainText("Lines 2–3", { timeout: 15_000 });
    await expect(anchorContext).toContainText(`e2e_rev_m2_${stamp}`);
  });

  test("two threads ending on one line share a counted marker, and the stack expands either one", async () => {
    const shared = planPage.statementEditor.locator(".bb-thread-glyph--count-2");
    await expect(shared).toHaveCount(1);
    await shared.click();
    // The earliest unresolved thread expands; the other waits as a row.
    await expect(planPage.threadCardIn("changes", earlierRoot)).toBeVisible();
    const laterRow = planPage.collapsedThreadRow(laterRootName);
    await expect(laterRow).toBeVisible();
    await expect(laterRow).toContainText(laterRoot);
    await laterRow.click();
    await expect(planPage.threadCardIn("changes", laterRoot)).toBeVisible();
    await expect(planPage.threadCardIn("changes", earlierRoot)).not.toBeVisible();
  });

  test("the walker announces its position and yields the corner to the find widget", async () => {
    await expect(planPage.threadWalker).toContainText("3");
    await planPage.threadWalkerNext.click();
    await expect(planPage.threadWalkerAnnouncement).toHaveText(/^Thread [1-3] of 3$/);
    // Monaco's find widget takes the corner while it is open. Focus the
    // editor through its input, since open thread cards cover the lines.
    await planPage.statementEditor.locator("textarea.inputarea").focus();
    await page.keyboard.press("Control+f");
    await expect(planPage.statementEditor.locator(".find-widget.visible")).toBeVisible();
    await expect(planPage.threadWalker).toHaveCount(0);
    await page.keyboard.press("Escape");
    await expect(planPage.threadWalker).toBeVisible({ timeout: 5_000 });
  });

  test("a narrow viewport keeps the walker inside the editor's top-right corner and composers usable", async () => {
    await page.setViewportSize({ width: 480, height: 1200 });
    await expect(planPage.threadWalker).toBeVisible({ timeout: 10_000 });
    const editorBox = await planPage.statementEditor.boundingBox();
    const walkerBox = await planPage.threadWalker.boundingBox();
    expect(editorBox && walkerBox).toBeTruthy();
    if (editorBox && walkerBox) {
      expect(walkerBox.x + walkerBox.width).toBeLessThanOrEqual(editorBox.x + editorBox.width);
      expect(walkerBox.y).toBeGreaterThanOrEqual(editorBox.y);
      expect(walkerBox.y - editorBox.y).toBeLessThan(40);
    }
    const composer = await planPage.openInlineComposerOn(1);
    await expect(composer.getByRole("button", { name: "Publish", exact: true })).toBeDisabled();
    await composer.getByRole("button", { name: "Cancel" }).click();
    await expect(planPage.inlineComposer).toHaveCount(0);
  });
});
