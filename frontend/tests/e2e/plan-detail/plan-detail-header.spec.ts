// Plan detail — HEADER lifecycle slot (the whole sub-area in one file).
//
// PR #20720 (BYT-9722) funnels the plan's lifecycle into a single header slot:
// one resolver (resolvePlanLifecycleHeaderState) reads the plan/issue/rollout
// state and decides what the header shows — an advance ACTION when the current
// user can move the plan forward, a read-only STATUS when they can't, or a
// terminal STAMP. Which of those appears is persona-dependent (candidate vs
// observer; deployer vs not), so this file is organized as a
// state × persona × path matrix, not a happy-path walk:
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
//   - Close + reopen a draft plan (no issue). (T2)
//
// Not covered here (with reason): `create` (draft-create flow, covered by the
// smoke/create paths); `review-generating` / `preparing-rollout` (sub-second
// transient states — not reliably observable in the browser); GitOps `none`
// (needs a release-backed plan + VCS setup, out of this sub-area).
//
// Each describe configures the settings IT needs and navigates fresh in its own
// beforeAll, so blocks are order-independent (they share one browser for speed,
// per AGENTS.md §2). The file-level beforeAll snapshots the originals + opens
// the admin + dba1 browsers; the file-level afterAll restores + closes them.

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

let originalProjectSettings: {
  requireIssueApproval?: boolean;
  requirePlanCheckNoError?: boolean;
  allowSelfApproval?: boolean;
} = {};
let originalApproval: unknown = null;
const createdReviewConfigs: string[] = [];

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
  await dbaContext?.close();
  await sharedContext?.close();
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
    await env.api.deletePolicy(env.project, "tag").catch(() => {});
    await attachErrorConfig();
    // requirePlanCheckNoError:true keeps the rollout blocked after approval, so
    // the resolver stays on the checks-failing status (not deploy).
    await env.api.updateProjectSettings(env.project, {
      requireIssueApproval: true,
      requirePlanCheckNoError: true,
      allowSelfApproval: true,
    });
    await env.api.upsertSetting(
      "WORKSPACE_APPROVAL",
      { workspaceApproval: { rules: [ADMIN_RULE] } },
      "value.workspace_approval",
    );
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
  let planId = "";
  const acceptDialogs = (d: import("@playwright/test").Dialog) => d.accept();

  test.beforeAll(async () => {
    // Open review, no rollout yet (self-approval off → observer, unapproved).
    await setApproval(false);
    const seeded = await seedReviewPlan(env.api, env.project, env.database, {
      prefix: "E2E Hdr T1",
      sql: "SELECT 1;",
      runChecks: true,
    });
    planId = seeded.planId;
    await waitForApprovalStatus(env.api, seeded.issueName, ["PENDING"]);
    // Close/reopen confirm via window.confirm — auto-accept for this block.
    page.on("dialog", acceptDialogs);
  });

  test.afterAll(async () => {
    page.off("dialog", acceptDialogs);
  });

  test("Close cancels the review (Closed stamp); Reopen restores it", async () => {
    await goPlan(planId);
    await expect(planPage.headerStatusPill("Under review")).toBeVisible({
      timeout: 15_000,
    });

    // Close lives in the ⋯ overflow (the slot's primary is the status pill).
    await planPage.openOverflow();
    await planPage.overflowItem("Close").click();
    await expect(planPage.headerStamp("Closed")).toBeVisible({ timeout: 15_000 });

    // With a terminal slot (no primary), Reopen is promoted to a direct button.
    await expect(planPage.headerReopenButton).toBeVisible({ timeout: 10_000 });
    await planPage.headerReopenButton.click();
    await expect(planPage.headerStatusPill("Under review")).toBeVisible({
      timeout: 15_000,
    });
    await expect(planPage.headerStamp("Closed")).toHaveCount(0);
  });
});

test.describe("Close and reopen a draft plan from the ⋯ overflow menu (T2)", () => {
  let planId = "";
  const acceptDialogs = (d: import("@playwright/test").Dialog) => d.accept();

  test.beforeAll(async () => {
    await setPermissive();
    // A plan with NO issue and NO rollout → ready-for-review + close-plan.
    const ts = Date.now();
    const sheet = await env.api.createSheet(env.project, "SELECT 1;");
    const plan = await env.api.createPlan(env.project, `E2E Hdr T2 ${ts}`, [
      { id: `spec-${ts}`, targets: [env.database], sheet },
    ]);
    planId = plan.name.split("/").pop()!;
    page.on("dialog", acceptDialogs);
  });

  test.afterAll(async () => {
    page.off("dialog", acceptDialogs);
  });

  test("Close deletes the draft plan (Closed stamp); Reopen restores it", async () => {
    await goPlan(planId);
    // Draft with no issue → the primary is "Ready for Review"; Close is in ⋯.
    await expect(
      planPage.headerRow.getByRole("button", { name: "Ready for Review" }),
    ).toBeVisible({ timeout: 15_000 });

    await planPage.openOverflow();
    await planPage.overflowItem("Close").click();
    await expect(planPage.headerStamp("Closed")).toBeVisible({ timeout: 15_000 });

    await expect(planPage.headerReopenButton).toBeVisible({ timeout: 10_000 });
    await planPage.headerReopenButton.click();
    await expect(
      planPage.headerRow.getByRole("button", { name: "Ready for Review" }),
    ).toBeVisible({ timeout: 15_000 });
    await expect(planPage.headerStamp("Closed")).toHaveCount(0);
  });
});
