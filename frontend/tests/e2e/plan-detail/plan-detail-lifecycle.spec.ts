// Plan Detail lifecycle — the whole header lifecycle sub-area in one file.
//
// PR #20720 (BYT-9722) funnels the plan's lifecycle into a single header slot:
// one resolver (resolvePlanLifecycleHeaderState) reads the plan/issue/rollout
// state and decides what the header shows — an advance ACTION when the current
// user can move the plan forward, a read-only STATUS when they can't, or a
// terminal STAMP. Which of those appears is persona-dependent (candidate vs
// observer; deployer vs not), so this file is organized as a
// state × persona × path matrix, not a happy-path walk:
//
//   DRAFT
//   - Create stays actionable and states every blocker on press. (C1)
//   - A clean draft becomes ready for review in one press. (S1)
//   - Failed checks are either a blocker or one named override. (S2)
//   - Forced labels stay in metadata and block only submission. (S3)
//   - Unsaved edits and missing permissions are actionable blockers. (S4/S5)
//
//   REVIEW
//   - Persona split: candidate sees the "Review" action; a non-candidate
//     observer sees the read-only "Under review" pill (same plan). (R1/R2)
//   - Rejected → "Rejected" pill (error) + re-request in the popover. (R3, unhappy)
//   - Approved + failing ERROR check → "N checks failing" pill. (R4, unhappy)
//
//   DEPLOY (frontier stage)
//   - canRun → "Run · <stage>" → run → "Deployed" terminal stamp. (D1, happy)
//   - A failed task → "Rerun · <stage>" (not "Run"). (D2, unhappy)
//   - Multi-stage rollout → the header advance walks stage by stage to
//     Deployed. (D3, multi-stage)
//
//   TERMINAL / overflow (⋯)
//   - Close + reopen the review issue. (T1)
//   - Close + reopen a Plan with a linked draft Issue. (T2)
//
// Not covered here (with reason): `review-generating` / `preparing-rollout`
// (sub-second transient states — not reliably observable in the browser);
// queued/running checks (the sample check finishes too quickly for a stable
// browser assertion); GitOps `none`
// (needs a release-backed plan + VCS setup, out of this sub-area).
//
// Each describe configures the settings IT needs and navigates fresh in its own
// beforeAll, so blocks are order-independent (they share one browser for speed,
// per AGENTS.md §2). The file-level beforeAll snapshots the originals + opens
// the admin + persona browsers; the file-level afterAll restores + closes them.

import {
  test,
  expect,
  type Page,
  type BrowserContext,
} from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import {
  BytebaseApiClient,
  type IamBinding,
} from "../framework/api-client";
import { signInBrowserAs } from "../framework/sign-in";
import { PlanDetailPage } from "./plan-detail.page";
import {
  seedDraftPlan,
  seedReviewPlan,
  waitForApprovalStatus,
} from "./plan-helpers";

test.setTimeout(240_000);
test.describe.configure({ mode: "serial" });

let env: TestEnv & { api: BytebaseApiClient };
let projectId: string;

let sharedContext: BrowserContext;
let page: Page;
let planPage: PlanDetailPage;

// A second browser signed in as dba1 (workspaceDBA) — the non-candidate
// observer persona for the review persona-split.
let dbaContext: BrowserContext;
let dbaPage: Page;
let dbaPlanPage: PlanDetailPage;

// A custom read-only persona can load Plan Detail but cannot update its draft
// Issue. This proves lifecycle blockers are real permission explanations, not
// admin-only rendering artifacts.
let viewerContext: BrowserContext;
let viewerPage: Page;
let viewerPlanPage: PlanDetailPage;

interface IssueLabelSetting {
  value: string;
  color?: Record<string, unknown>;
  group?: string;
}

let originalProjectSettings: {
  requireIssueApproval?: boolean;
  requirePlanCheckNoError?: boolean;
  allowSelfApproval?: boolean;
  enforceSqlReview?: boolean;
  forceIssueLabels?: boolean;
  issueLabels?: IssueLabelSetting[];
} = {};
let originalApproval: unknown = null;
const createdReviewConfigs: string[] = [];
let originalProjectIamBindings: IamBinding[] = [];

const VIEWER_EMAIL = "e2e-plan-lifecycle-viewer@example.com";
const VIEWER_PASSWORD = "e2e-plan-lifecycle-pw-1!"; // NOSONAR — e2e fixture only
const VIEWER_ROLE_ID = "e2e-plan-lifecycle-reader";
const VIEWER_ROLE = `roles/${VIEWER_ROLE_ID}`;
const VIEWER_PERMISSIONS = [
  "bb.databases.get",
  "bb.databases.list",
  "bb.issueComments.list",
  "bb.issues.get",
  "bb.planCheckRuns.get",
  "bb.plans.get",
  "bb.projects.get",
  "bb.projects.getIamPolicy",
  "bb.sheets.get",
  "bb.taskRuns.list",
];
const REQUIRED_LABEL = "e2e-lifecycle";

const ADMIN_RULE = {
  source: "CHANGE_DATABASE",
  condition: { expression: "true" },
  template: {
    flow: { roles: ["roles/workspaceAdmin"] },
    title: "E2E Header Admin",
    description: "single-step workspaceAdmin approval",
  },
};

// Mandatory single-step admin approval. allowSelfApproval decides whether the
// admin creator is a candidate (true → "Review" action) or an observer
// (false → "Under review" pill).
async function setApproval(allowSelfApproval: boolean): Promise<void> {
  await env.api.deletePolicy(env.project, "tag").catch(() => {});
  await env.api.updateProjectSettings(env.project, {
    requireIssueApproval: true,
    requirePlanCheckNoError: false,
    allowSelfApproval,
    enforceSqlReview: false,
    forceIssueLabels: false,
  });
  await env.api.upsertSetting(
    "WORKSPACE_APPROVAL",
    { workspaceApproval: { rules: [ADMIN_RULE] } },
    "value.workspace_approval",
  );
}

async function setPermissive(): Promise<void> {
  await env.api.deletePolicy(env.project, "tag").catch(() => {});
  await env.api.updateProjectSettings(env.project, {
    requireIssueApproval: false,
    requirePlanCheckNoError: false,
    enforceSqlReview: false,
    forceIssueLabels: false,
  });
  // Clear any approval rule a prior describe left in WORKSPACE_APPROVAL —
  // otherwise a leftover rule still forces a pending review (blocking the
  // auto-rollout) even with requireIssueApproval=false. (Each describe must
  // reset its arrival state; the page is shared.)
  await env.api.upsertSetting(
    "WORKSPACE_APPROVAL",
    { workspaceApproval: { rules: [] } },
    "value.workspace_approval",
  );
}

// Wait (via API) for the backend to auto-create the rollout, so the deploy
// tests navigate to a page that already shows the frontier Run advance rather
// than racing the async rollout creation.
async function waitForRollout(planName: string, timeoutMs = 30_000): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    if ((await env.api.getPlan(planName)).hasRollout) return;
    await new Promise((r) => setTimeout(r, 1000));
  }
  throw new Error(`rollout was not auto-created for ${planName} in ${timeoutMs}ms`);
}

// Attach a single ERROR-level COLUMN_NO_NULL rule so a nullable column trips it.
async function attachErrorConfig(): Promise<void> {
  const id = `e2e-hdr-err-${Date.now()}`;
  const cfg = await env.api.upsertReviewConfig(id, "E2E Header ERROR", [
    { type: "COLUMN_NO_NULL", level: "ERROR", engine: "POSTGRES" },
  ]);
  createdReviewConfigs.push(cfg.name);
  await env.api.upsertReviewConfigTag(env.project, cfg.name);
}

async function goPlan(planId: string): Promise<void> {
  await planPage.goto(projectId, planId);
  await planPage.dismissModals();
}

async function waitForIssueDraft(
  issueName: string,
  expected: boolean,
  timeoutMs = 30_000,
): Promise<void> {
  await expect
    .poll(async () => (await env.api.getIssue(issueName)).draft ?? false, {
      message: `${issueName} draft should become ${expected}`,
      timeout: timeoutMs,
    })
    .toBe(expected);
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
    enforceSqlReview: !!project.enforceSqlReview,
    forceIssueLabels: !!project.forceIssueLabels,
    issueLabels: Array.isArray(project.issueLabels)
      ? (structuredClone(project.issueLabels) as IssueLabelSetting[])
      : [],
  };
  originalApproval = (await env.api.getSetting("WORKSPACE_APPROVAL"))?.value ?? null;
  originalProjectIamBindings = structuredClone(
    (await env.api.getProjectIamPolicy(env.project)).bindings,
  );

  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
  planPage = new PlanDetailPage(page, env.baseURL);

  // dba1 = the non-candidate observer persona.
  await signInBrowserAs(
    browser,
    env.baseURL,
    "dba1@example.com",
    "12345678",
    ".auth/dba1-header.json",
  );
  dbaContext = await browser.newContext({
    storageState: ".auth/dba1-header.json",
  });
  dbaPage = await dbaContext.newPage();
  dbaPlanPage = new PlanDetailPage(dbaPage, env.baseURL);

  try {
    await env.api.createUser(
      VIEWER_EMAIL,
      VIEWER_PASSWORD,
      "E2E Plan Lifecycle Viewer",
    );
  } catch (error) {
    if (!String(error).includes("(409)")) throw error;
  }
  try {
    await env.api.createRole(
      VIEWER_ROLE_ID,
      "E2E Plan Lifecycle Reader",
      VIEWER_PERMISSIONS,
    );
  } catch (error) {
    if (!String(error).includes("(409)")) throw error;
  }
  await env.api.appendProjectBinding(env.project, VIEWER_ROLE, [
    `user:${VIEWER_EMAIL}`,
  ]);
  await signInBrowserAs(
    browser,
    env.baseURL,
    VIEWER_EMAIL,
    VIEWER_PASSWORD,
    ".auth/plan-lifecycle-viewer.json",
  );
  viewerContext = await browser.newContext({
    storageState: ".auth/plan-lifecycle-viewer.json",
  });
  viewerPage = await viewerContext.newPage();
  viewerPlanPage = new PlanDetailPage(viewerPage, env.baseURL);
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
  const currentProjectIam = await env.api
    .getProjectIamPolicy(env.project)
    .catch(() => undefined);
  if (currentProjectIam) {
    await env.api
      .setProjectIamPolicy(env.project, {
        ...currentProjectIam,
        bindings: originalProjectIamBindings,
      })
      .catch(() => {});
  }
  await env.api.deleteRole(VIEWER_ROLE);
  await viewerContext?.close();
  await dbaContext?.close();
  await sharedContext?.close();
});

/* ------------------------------ DRAFT ---------------------------------- */

test.describe("Create states every blocker and advances to a real draft (C1)", () => {
  test.beforeAll(async () => {
    await setApproval(false);
    await env.api.updateProjectSettings(env.project, {
      forceIssueLabels: true,
      issueLabels: [
        ...(originalProjectSettings.issueLabels ?? []).filter(
          (label) => label.value !== REQUIRED_LABEL,
        ),
        { value: REQUIRED_LABEL },
      ],
    });
  });

  test("Create defers required labels, states its blockers, and creates a linked draft Issue", async () => {
    await planPage.gotoCreate(projectId, { databaseList: env.database });
    await planPage.dismissModals();

    const labels = page.getByRole("button", { name: /Issue Labels/ }).first();
    await expect(labels).not.toContainText("*");
    await expect(planPage.headerTitle).toHaveValue("");
    await expect(planPage.headerCreateButton).toBeEnabled();
    await planPage.headerCreateButton.click();

    const notice = planPage.lifecycleAlert("Cannot create plan");
    await expect(notice).toBeVisible();
    await expect(notice).toContainText("Title is required");
    await expect(notice).toContainText("Statement is empty");
    await expect(notice).not.toContainText("Issue Labels");
    await expect(planPage.headerTitle).toBeFocused();
    await expect(page).toHaveURL(/\/plans\/create/);

    const title = `E2E Lifecycle Create ${Date.now()}`;
    await planPage.headerTitle.fill(title);
    await expect(notice).not.toContainText("Title is required");
    await expect(notice).toContainText("Statement is empty");

    await planPage.fillPlanStatement("SELECT 1;");
    await expect(notice).toHaveCount(0);

    await planPage.headerCreateButton.click();
    await expect(page).toHaveURL(/\/plans\/(?!create(?:\/|$))[^/?]+$/, {
      timeout: 30_000,
    });
    await expect(planPage.headerReadyForReviewButton).toBeVisible({
      timeout: 15_000,
    });
    await expect(labels).toContainText("*");

    const planId = new URL(page.url()).pathname.split("/").pop()!;
    const plan = await env.api.getPlan(
      `projects/${projectId}/plans/${planId}`,
    );
    expect(plan.issue).toBeTruthy();
    await waitForIssueDraft(plan.issue, true);
  });
});

test.describe("A clean draft submits for review without an intermediate form (S1)", () => {
  let seeded: Awaited<ReturnType<typeof seedDraftPlan>>;

  test.beforeAll(async () => {
    await setApproval(false);
    seeded = await seedDraftPlan(env.api, env.project, env.database, {
      prefix: "E2E Lifecycle Direct Submit",
      sql: "SELECT 1;",
      runChecks: true,
    });
  });

  test("Ready for Review advances in one press", async () => {
    await goPlan(seeded.planId);
    await expect(planPage.headerReadyForReviewButton).toBeEnabled();

    await planPage.headerReadyForReviewButton.click();

    await expect(
      page.getByRole("button", { name: "Confirm", exact: true }),
    ).toHaveCount(0);
    await waitForIssueDraft(seeded.issueName, false);
    await waitForApprovalStatus(env.api, seeded.issueName, ["PENDING"]);
    await expect(planPage.headerStatusPill("Under review")).toBeVisible({
      timeout: 15_000,
    });
    await expect(planPage.headerReadyForReviewButton).toHaveCount(0);
  });
});

test.describe("Failed checks resolve to a blocker or one explicit override (S2)", () => {
  let seeded: Awaited<ReturnType<typeof seedDraftPlan>>;

  test.beforeAll(async () => {
    await setApproval(false);
    await attachErrorConfig();
    seeded = await seedDraftPlan(env.api, env.project, env.database, {
      prefix: "E2E Lifecycle Failed Checks",
      sql: `ALTER TABLE employee ADD COLUMN e2e_lifecycle_${Date.now()} TEXT;`,
      runChecks: true,
    });
    await env.api.updateProjectSettings(env.project, {
      enforceSqlReview: true,
    });
  });

  test("enforcement blocks submission; relaxing it offers Submit anyway with working cancel", async () => {
    await goPlan(seeded.planId);
    await planPage.headerReadyForReviewButton.click();

    const blocker = planPage.lifecycleAlert("Not ready for review");
    await expect(blocker).toContainText("Some task checks didn't pass.");
    await expect(
      page.getByRole("button", { name: "Submit anyway" }),
    ).toHaveCount(0);
    await waitForIssueDraft(seeded.issueName, true);

    await env.api.updateProjectSettings(env.project, {
      enforceSqlReview: false,
    });
    await page.reload();
    await page.waitForLoadState("networkidle");
    await planPage.headerReadyForReviewButton.click();

    const submitAnyway = page.getByRole("button", {
      name: "Submit anyway",
      exact: true,
    });
    await expect(submitAnyway).toBeVisible();
    const decision = submitAnyway.locator("..").locator("..");
    await expect(
      decision.getByText("Some checks were not successful", { exact: true }),
    ).toBeVisible();
    await expect(
      decision.getByText(
        "Some checks did not pass. Continue anyway only if this is expected.",
        { exact: true },
      ),
    ).toBeVisible();
    await expect(
      decision.getByRole("button", { name: "Confirm", exact: true }),
    ).toHaveCount(0);
    await expect(decision.getByRole("checkbox")).toHaveCount(0);

    await page.getByRole("button", { name: "Cancel", exact: true }).click();
    await expect(submitAnyway).toHaveCount(0);
    await waitForIssueDraft(seeded.issueName, true);

    await planPage.headerReadyForReviewButton.click();
    await page
      .getByRole("button", { name: "Submit anyway", exact: true })
      .click();
    await waitForIssueDraft(seeded.issueName, false);
    await expect(planPage.headerStatusPill("Under review")).toBeVisible({
      timeout: 15_000,
    });
  });
});

test.describe("Forced labels stay in metadata and block only submission (S3)", () => {
  let seeded: Awaited<ReturnType<typeof seedDraftPlan>>;

  test.beforeAll(async () => {
    await setApproval(false);
    await env.api.updateProjectSettings(env.project, {
      forceIssueLabels: true,
      issueLabels: [
        ...(originalProjectSettings.issueLabels ?? []).filter(
          (label) => label.value !== REQUIRED_LABEL,
        ),
        { value: REQUIRED_LABEL },
      ],
    });
    seeded = await seedDraftPlan(env.api, env.project, env.database, {
      prefix: "E2E Lifecycle Required Label",
      sql: "SELECT 1;",
    });
  });

  test("the blocker points to metadata; selecting a label clears it before direct submission", async () => {
    await goPlan(seeded.planId);
    await planPage.headerReadyForReviewButton.click();

    const blocker = planPage.lifecycleAlert("Not ready for review");
    await expect(blocker).toContainText(
      "Issue Labels are required before review",
    );
    await expect(blocker.getByRole("button")).toHaveCount(0);

    const labels = page.getByRole("button", { name: /Issue Labels/ }).first();
    await expect(labels).toContainText("*");
    await labels.click();
    await page
      .getByRole("button", { name: REQUIRED_LABEL, exact: true })
      .click();

    await expect
      .poll(async () => (await env.api.getIssue(seeded.issueName)).labels ?? [])
      .toContain(REQUIRED_LABEL);
    await expect(blocker).toHaveCount(0);
    await page.keyboard.press("Escape");
    await expect(
      page.getByTestId("plan-detail-page").getByText(REQUIRED_LABEL, {
        exact: true,
      }),
    ).toBeVisible();

    await planPage.headerReadyForReviewButton.click();
    await waitForIssueDraft(seeded.issueName, false);
    await expect(planPage.headerStatusPill("Under review")).toBeVisible({
      timeout: 15_000,
    });
  });
});

test.describe("Unsaved Plan edits block lifecycle advance without disabling it (S4)", () => {
  let seeded: Awaited<ReturnType<typeof seedDraftPlan>>;

  test.beforeAll(async () => {
    await setApproval(false);
    seeded = await seedDraftPlan(env.api, env.project, env.database, {
      prefix: "E2E Lifecycle Unsaved Edit",
      sql: "SELECT 1;",
    });
  });

  test("Ready for Review explains the edit and advances after it is cancelled", async () => {
    await goPlan(seeded.planId);
    await page.getByRole("button", { name: "Edit", exact: true }).last().click();
    await planPage.fillPlanStatement("SELECT 2;");

    await expect(planPage.headerReadyForReviewButton).toBeEnabled();
    await planPage.headerReadyForReviewButton.click();
    await expect(
      planPage.lifecycleAlert("Not ready for review"),
    ).toContainText("Please save or cancel your changes before continuing");

    await page.getByRole("button", { name: "Cancel", exact: true }).click();
    await expect(planPage.lifecycleAlert("Not ready for review")).toHaveCount(0);
    await planPage.headerReadyForReviewButton.click();
    await waitForIssueDraft(seeded.issueName, false);
  });
});

test.describe("A read-only persona gets an actionable permission explanation (S5)", () => {
  let seeded: Awaited<ReturnType<typeof seedDraftPlan>>;

  test.beforeAll(async () => {
    await setApproval(false);
    seeded = await seedDraftPlan(env.api, env.project, env.database, {
      prefix: "E2E Lifecycle Viewer",
      sql: "SELECT 1;",
    });
  });

  test("a read-only user sees enabled Ready for Review and the missing permission blocker", async () => {
    await viewerPlanPage.goto(projectId, seeded.planId);
    await viewerPlanPage.dismissModals();
    await expect(viewerPlanPage.headerReadyForReviewButton).toBeEnabled();

    await viewerPlanPage.headerReadyForReviewButton.click();

    await expect(
      viewerPlanPage.lifecycleAlert("Not ready for review"),
    ).toContainText("you need bb.issues.update");
    await waitForIssueDraft(seeded.issueName, true);
  });
});

test.describe("A persisted Plan without its draft Issue is incomplete (S6)", () => {
  let planId = "";

  test.beforeAll(async () => {
    await setApproval(false);
    const ts = Date.now();
    const sheet = await env.api.createSheet(env.project, "SELECT 1;");
    const plan = await env.api.createPlan(
      env.project,
      `E2E Lifecycle Incomplete ${ts}`,
      [{ id: `spec-${ts}`, targets: [env.database], sheet }],
    );
    planId = plan.name.split("/").pop()!;
  });

  test("the header shows Incomplete instead of offering Ready for Review", async () => {
    await goPlan(planId);
    await expect(planPage.headerStamp("Incomplete")).toBeVisible({
      timeout: 15_000,
    });
    await expect(planPage.headerReadyForReviewButton).toHaveCount(0);
  });
});

/* ----------------------------- REVIEW ---------------------------------- */

test.describe("Review advance is persona-scoped (R1/R2)", () => {
  let planId = "";

  test.beforeAll(async () => {
    // allowSelfApproval → admin (creator) IS the candidate.
    await setApproval(true);
    const seeded = await seedReviewPlan(env.api, env.project, env.database, {
      prefix: "E2E Hdr R1R2",
      sql: "SELECT 1;",
      runChecks: true,
    });
    planId = seeded.planId;
    await waitForApprovalStatus(env.api, seeded.issueName, ["PENDING"]);
  });

  test("candidate sees the Review action in the header slot", async () => {
    await goPlan(planId);
    await expect(planPage.headerReviewButton).toBeVisible({ timeout: 15_000 });
    // The candidate must NOT also see the read-only status pill.
    await expect(planPage.headerStatusPill("Under review")).toHaveCount(0);
  });

  test("a non-candidate observer sees the read-only 'Under review' pill instead", async () => {
    await dbaPlanPage.goto(projectId, planId);
    await dbaPlanPage.dismissModals();
    const pill = dbaPlanPage.headerStatusPill("Under review");
    await expect(pill).toBeVisible({ timeout: 15_000 });
    // The observer must NOT see the Review advance.
    await expect(dbaPlanPage.headerReviewButton).toHaveCount(0);

    // The pill opens the gate popover: Review gate + Checks gate.
    await pill.click();
    await expect(
      dbaPage.getByText("Review in progress", { exact: true }),
    ).toBeVisible({ timeout: 10_000 });
    await expect(
      dbaPage.getByText("All checks passed", { exact: true }),
    ).toBeVisible();
  });
});

test.describe("Rejected review shows the Rejected pill + re-request (R3)", () => {
  let planId = "";

  test.beforeAll(async () => {
    await setApproval(true);
    const seeded = await seedReviewPlan(env.api, env.project, env.database, {
      prefix: "E2E Hdr R3",
      sql: "SELECT 1;",
      runChecks: true,
    });
    planId = seeded.planId;
    await waitForApprovalStatus(env.api, seeded.issueName, ["PENDING"]);
    await env.api.rejectIssue(seeded.issueName, "e2e header reject");
    await waitForApprovalStatus(env.api, seeded.issueName, ["REJECTED"]);
  });

  test("header shows the error-toned Rejected pill and the popover offers re-request", async () => {
    await goPlan(planId);
    const pill = planPage.headerStatusPill("Rejected");
    await expect(pill).toBeVisible({ timeout: 15_000 });
    await pill.click();
    // Scope to the popover the pill controls — a "re-request review" button also
    // lives in the review-section rejection banner, so an unscoped locator is
    // ambiguous (strict-mode violation).
    const popupId = await pill.getAttribute("aria-controls");
    expect(popupId, "pill should control a popover").toBeTruthy();
    const popover = page.locator(`#${popupId}`);
    // The popover's Review gate reflects the rejection…
    await expect(
      popover.getByText("Changes requested", { exact: true }),
    ).toBeVisible({ timeout: 10_000 });
    // …and the creator (admin) can re-request review from its footer.
    await expect(
      popover.getByRole("button", { name: "re-request review" }),
    ).toBeVisible();
  });
});

test.describe("Approved with a failing ERROR check shows the checks-failing pill (R4)", () => {
  let planId = "";

  test.beforeAll(async () => {
    await setApproval(true);
    await attachErrorConfig();
    // requirePlanCheckNoError:true keeps the rollout blocked after approval, so
    // the resolver stays on the checks-failing status (not deploy).
    await env.api.updateProjectSettings(env.project, {
      requirePlanCheckNoError: true,
    });
    const ts = Date.now();
    const seeded = await seedReviewPlan(env.api, env.project, env.database, {
      // A nullable column trips COLUMN_NO_NULL at ERROR level.
      prefix: "E2E Hdr R4",
      sql: `ALTER TABLE employee ADD COLUMN e2e_r4_${ts} TEXT;`,
      runChecks: true,
    });
    planId = seeded.planId;
    await waitForApprovalStatus(env.api, seeded.issueName, ["PENDING"]);
    await env.api.approveIssue(seeded.issueName);
    await waitForApprovalStatus(env.api, seeded.issueName, ["APPROVED"]);
  });

  test("header shows the '… checks failing' pill after approval", async () => {
    await goPlan(planId);
    const pill = planPage.headerStatusPill(/check(s)? failing/i);
    await expect(pill).toBeVisible({ timeout: 15_000 });
    await pill.click();
    await expect(
      page.getByText("Some checks were not successful", { exact: true }),
    ).toBeVisible({ timeout: 10_000 });
  });
});

/* ----------------------------- DEPLOY ---------------------------------- */

test.describe("Running the frontier stage from the header reaches the Deployed stamp (D1)", () => {
  let planId = "";

  test.beforeAll(async () => {
    await setPermissive();
    const ts = Date.now();
    const seeded = await seedReviewPlan(env.api, env.project, env.database, {
      prefix: "E2E Hdr D1",
      sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_d1_${ts} TEXT;`,
      runChecks: true,
    });
    planId = seeded.planId;
    await waitForRollout(seeded.planName);
  });

  test("header Run·<stage> runs the frontier, then the slot becomes the Deployed stamp", async () => {
    await goPlan(planId);
    // The header advance is present and distinct from the deploy-section Run.
    await expect(planPage.headerRunStage).toBeVisible({ timeout: 20_000 });

    await planPage.headerRunStage.click();
    await planPage.confirmRunTaskDialog();

    // After the frontier stage completes, the header shows the terminal stamp
    // and no Run advance remains.
    await expect(planPage.headerStamp("Deployed")).toBeVisible({
      timeout: 30_000,
    });
    await expect(planPage.headerRunStage).toHaveCount(0);
  });
});

test.describe("A failed task surfaces Rerun in the header slot (D2)", () => {
  let planId = "";

  test.beforeAll(async () => {
    await setPermissive();
    const ts = Date.now();
    // A nonexistent target makes the task fail at execution.
    const seeded = await seedReviewPlan(env.api, env.project, env.database, {
      prefix: "E2E Hdr D2",
      sql: `ALTER TABLE nonexistent_e2e_hdr_${ts} ADD COLUMN c1 TEXT;`,
      runChecks: false,
    });
    planId = seeded.planId;
    await waitForRollout(seeded.planName);
  });

  test("after the task fails, the header advance reads Rerun·<stage>", async () => {
    await goPlan(planId);
    // First run (the stage has never executed) — labelled "Run".
    await expect(planPage.headerRunStage).toBeVisible({ timeout: 20_000 });
    await planPage.headerRunStage.click();
    await planPage.confirmRunTaskDialog();

    // The task fails; the frontier now has a failed task → the advance flips to
    // "Rerun · <stage>" (a re-run of a stage that already executed).
    await expect(planPage.headerRerunStage).toBeVisible({ timeout: 30_000 });
    await expect(planPage.headerRunStage).toHaveCount(0);
  });
});

test.describe("A multi-stage rollout advances the header stage by stage to Deployed (D3)", () => {
  let planId = "";

  test.beforeAll(async () => {
    await setPermissive();
    const ts = Date.now();
    const testDb = await env.api.findDatabaseByShortName("hr_test");
    const prodDb = await env.api.findDatabaseByShortName("hr_prod");
    if (!testDb || !prodDb) {
      throw new Error("multi-stage seed needs hr_test + hr_prod sample dbs");
    }
    const sheet = await env.api.createSheet(
      env.project,
      `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_d3_${ts} TEXT;`,
    );
    // One spec targeting two environments → the rollout groups tasks into two
    // stages (Test, Prod).
    const plan = await env.api.createPlan(env.project, `E2E Hdr D3 ${ts}`, [
      { id: `spec-${ts}`, targets: [testDb.database, prodDb.database], sheet },
    ]);
    await env.api.createIssue(env.project, `E2E Hdr D3 ${ts}`, plan.name);
    planId = plan.name.split("/").pop()!;
    await waitForRollout(plan.name);
  });

  test("the header Run advance walks each stage until the plan is Deployed", async () => {
    await goPlan(planId);

    // Frontier = first (earliest-environment) incomplete stage.
    await expect(planPage.headerRunStage).toBeVisible({ timeout: 20_000 });
    const firstLabel = (await planPage.headerRunStage.textContent())?.trim();

    await planPage.headerRunStage.click();
    await planPage.confirmRunTaskDialog();

    // Frontier advances to the next stage — still a Run advance, different stage.
    await expect(planPage.headerRunStage).toBeVisible({ timeout: 30_000 });
    await expect(async () => {
      const nextLabel = (await planPage.headerRunStage.textContent())?.trim();
      expect(nextLabel).not.toBe(firstLabel);
    }).toPass({ timeout: 15_000 });

    await planPage.headerRunStage.click();
    await planPage.confirmRunTaskDialog();

    // Both stages complete → Deployed stamp, no remaining Run advance.
    await expect(planPage.headerStamp("Deployed")).toBeVisible({
      timeout: 30_000,
    });
    await expect(planPage.headerRunStage).toHaveCount(0);
  });
});

/* -------------------------- TERMINAL / overflow ------------------------- */

test.describe("Close and reopen the review from the ⋯ overflow menu (T1)", () => {
  const titlePrefix = "E2E Hdr T1";
  const acceptDialogs = (d: import("@playwright/test").Dialog) => d.accept();

  test.beforeAll(async () => {
    // Open review, no rollout yet (self-approval off → observer, unapproved).
    await setApproval(false);
    const seeded = await seedReviewPlan(env.api, env.project, env.database, {
      prefix: titlePrefix,
      sql: "SELECT 1;",
      runChecks: true,
    });
    await waitForApprovalStatus(env.api, seeded.issueName, ["PENDING"]);
    // Close/reopen confirm via window.confirm — auto-accept for this block.
    page.on("dialog", acceptDialogs);
  });

  test.afterAll(async () => {
    page.off("dialog", acceptDialogs);
  });

  test("Close and Reopen update both Plan Detail and the restored Plan List row", async () => {
    const listURL = `${env.baseURL}/projects/${projectId}/plans`;
    const listRow = () =>
      page
        .locator('[data-testid="plan-list-item"]')
        .filter({ hasText: titlePrefix })
        .first();

    await page.goto(listURL);
    await page.waitForLoadState("networkidle");
    await expect(listRow()).toContainText("Under review", {
      timeout: 15_000,
    });
    await listRow().click();
    await expect(planPage.headerStatusPill("Under review")).toBeVisible({
      timeout: 15_000,
    });

    // Close lives in the ⋯ overflow (the slot's primary is the status pill).
    await planPage.openOverflow();
    await planPage.overflowItem("Close").click();
    await expect(planPage.headerStamp("Closed")).toBeVisible({ timeout: 15_000 });

    // Back must not restore the stale under-review cache (#21017).
    await page.goBack();
    await expect(page).toHaveURL(listURL);
    await expect(listRow()).toContainText("Closed", { timeout: 15_000 });

    await listRow().click();
    // With a terminal slot (no primary), Reopen is promoted to a direct button.
    await expect(planPage.headerReopenButton).toBeVisible({ timeout: 10_000 });
    await planPage.headerReopenButton.click();
    await expect(planPage.headerStatusPill("Under review")).toBeVisible({
      timeout: 15_000,
    });
    await expect(planPage.headerStamp("Closed")).toHaveCount(0);

    await page.goBack();
    await expect(page).toHaveURL(listURL);
    await expect(listRow()).toContainText("Under review", {
      timeout: 15_000,
    });
  });
});

test.describe("Close and reopen a draft plan from the ⋯ overflow menu (T2)", () => {
  let planId = "";
  const acceptDialogs = (d: import("@playwright/test").Dialog) => d.accept();

  test.beforeAll(async () => {
    await setApproval(false);
    // A UI-authored draft is a Plan plus its linked draft Issue. A persisted
    // Plan without that Issue is an incomplete partial-creation state and must
    // not be mistaken for ready-for-review.
    const seeded = await seedDraftPlan(env.api, env.project, env.database, {
      prefix: "E2E Hdr T2",
      sql: "SELECT 1;",
    });
    planId = seeded.planId;
    page.on("dialog", acceptDialogs);
  });

  test.afterAll(async () => {
    page.off("dialog", acceptDialogs);
  });

  test("Close deletes the draft plan (Closed stamp); Reopen restores it", async () => {
    await goPlan(planId);
    // A linked draft Issue owns the Ready for Review lifecycle state; Close is
    // the secondary action in ⋯.
    await expect(planPage.headerReadyForReviewButton).toBeVisible({
      timeout: 15_000,
    });

    await planPage.openOverflow();
    await planPage.overflowItem("Close").click();
    await expect(planPage.headerStamp("Closed")).toBeVisible({ timeout: 15_000 });

    await expect(planPage.headerReopenButton).toBeVisible({ timeout: 10_000 });
    await planPage.headerReopenButton.click();
    await expect(planPage.headerReadyForReviewButton).toBeVisible({
      timeout: 15_000,
    });
    await expect(planPage.headerStamp("Closed")).toHaveCount(0);
  });
});
