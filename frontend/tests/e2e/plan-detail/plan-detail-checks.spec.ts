// Plan detail — SQL review / plan checks.
//
// Covers:
//   - WARNING-level rule produces an inline warning count but does NOT
//     block rollout creation (auto-rollout still happens).
//   - ERROR-level rule with `requirePlanCheckNoError=true` blocks the
//     auto-rollout; the Review readiness footer reports "Review approved, but
//     plan checks failed" and NO "Manually create rollout" button is offered.
//     Relaxing the gate (requirePlanCheckNoError=false) lets the user bypass
//     via the footer's "Bypass and deploy" action (the old DEPLOY manual
//     button is GitOps-only now — AIO review section, 3.19.1).
//   - Multi-spec plans: check counts render PLAN-WIDE (BYT-9160 resolution —
//     the per-spec sidebar was removed), and a test.fail lock for a NEW
//     regression where switching spec tabs leaves the prior spec's statement
//     editor stacked (BUG BYT-9794).
//
// Each test owns its review config + project TagPolicy via API, and
// cleans up in afterEach so a sibling test doesn't inherit state.

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
let originalSettings: {
  requireIssueApproval?: boolean;
  requirePlanCheckNoError?: boolean;
} = {};

// Track per-test review configs so afterEach can delete them.
let createdReviewConfigs: string[] = [];

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  projectId = env.project.split("/").pop()!;
  await env.api.login(env.adminEmail, env.adminPassword);

  const project = await env.api.getProject(env.project);
  originalSettings = {
    requireIssueApproval: !!project.requireIssueApproval,
    requirePlanCheckNoError: !!project.requirePlanCheckNoError,
  };
  // No approval gate by default; individual tests flip
  // requirePlanCheckNoError as needed.
  await env.api.updateProjectSettings(env.project, {
    requireIssueApproval: false,
    requirePlanCheckNoError: false,
  });

  sharedContext = await browser.newContext({
    storageState: ".auth/state.json",
  });
  page = await sharedContext.newPage();
  planPage = new PlanDetailPage(page, env.baseURL);
});

test.afterEach(async () => {
  // Remove the project's TagPolicy so the next test starts with no
  // review config attached. Then delete the configs we created.
  await env.api.deletePolicy(env.project, "tag").catch(() => {});
  for (const name of createdReviewConfigs) {
    await env.api.deleteReviewConfig(name).catch(() => {});
  }
  createdReviewConfigs = [];
  // Restore permissive plan-check setting; individual tests may have
  // flipped it.
  await env.api
    .updateProjectSettings(env.project, { requirePlanCheckNoError: false })
    .catch(() => {});
});

test.afterAll(async () => {
  await env.api
    .updateProjectSettings(env.project, originalSettings)
    .catch(() => {});
  await sharedContext?.close();
});

// Build a review config with a single COLUMN_NO_NULL rule on POSTGRES at
// the given level (WARNING or ERROR), bind it to the test project via a
// TagPolicy, and return its resource name so afterEach can delete it.
async function attachReviewConfig(
  level: "WARNING" | "ERROR",
): Promise<string> {
  const id = `e2e-${level.toLowerCase()}-${Date.now()}`;
  const cfg = await env.api.upsertReviewConfig(
    id,
    `E2E ${level} Review Config`,
    [{ type: "COLUMN_NO_NULL", level, engine: "POSTGRES" }],
    /* enabled */ true,
  );
  await env.api.upsertReviewConfigTag(env.project, cfg.name);
  createdReviewConfigs.push(cfg.name);
  return cfg.name;
}

// Poll the latest planCheckRun on `planName` until status === "DONE",
// then navigate to the plan detail page. Returns the planId.
async function createPlanAndWaitForChecks(
  titlePrefix: string,
  specs: { id: string; targets: string[]; sql: string }[],
): Promise<string> {
  const ts = Date.now();
  const planSpecs = await Promise.all(
    specs.map(async (s) => ({
      id: s.id,
      targets: s.targets,
      sheet: await env.api.createSheet(env.project, s.sql),
    })),
  );
  const plan = await env.api.createPlan(
    env.project,
    `${titlePrefix} ${ts}`,
    planSpecs,
  );
  const planName = plan.name;
  const planId = planName.split("/").pop()!;
  await env.api.createIssue(env.project, `${titlePrefix} ${ts}`, planName);
  await env.api.runPlanChecks(planName);

  await waitForPlanChecksDone(env.api, planName);

  await planPage.goto(projectId, planId);
  await planPage.dismissModals();
  return planId;
}

test.describe("WARNING-level review rule", () => {
  test("violating SQL produces a warning but does not block auto-rollout", async () => {
    await attachReviewConfig("WARNING");

    const ts = Date.now();
    await createPlanAndWaitForChecks("E2E Checks Warning", [
      {
        id: `spec-${ts}`,
        targets: [env.database],
        // Nullable TEXT with no default → trips COLUMN_NO_NULL.
        sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_warn_${ts} TEXT;`,
      },
    ]);

    // CHANGES summary line carries the warning count ("1 passed, 1 warning").
    // Scope the search to the plan body and match a numeric warning count.
    await expect(
      page.getByText(/\d+\s+warning/i).first(),
    ).toBeVisible({ timeout: 15_000 });

    // Rollout still auto-created — DEPLOY is "Not started", no manual
    // create button needed.
    await expect(planPage.deploySection).toBeVisible();
    await expect(planPage.manualCreateRolloutButton).not.toBeVisible({
      timeout: 3_000,
    });
  });
});

test.describe("ERROR-level review rule with requirePlanCheckNoError=true", () => {
  // Product contract observed on free-plan setupSample (2026-05):
  //   - ERROR check + requirePlanCheckNoError=true: rollout is BLOCKED.
  //     DEPLOY surfaces "Checks must pass. Failed" with helper text
  //     "Failed checks are blocking automatic rollout creation." NO
  //     "Manually create rollout" button is offered — the user must
  //     either fix the SQL or relax the gate.
  //   - When the gate is relaxed (requirePlanCheckNoError=false), the
  //     manual deploy path appears so the user can bypass the failed
  //     checks intentionally.
  //
  // NOTE (AIO plan review section, 3.19.1): the manual "Manually create
  // rollout" button was REMOVED from DEPLOY for issue-backed plans and is
  // now GitOps-only (PlanDetailDeployFuture.tsx). For issue-backed plans the
  // single manual path is the Review section's readiness-footer "Bypass and
  // deploy" action (ReviewReadinessFooter.tsx). The gate-off test below was
  // updated to assert that new path; the gate-on test is unchanged (DEPLOY
  // still explains the block and offers no manual button).
  // Both halves are covered here so a regression on either side fails
  // loudly.

  test("with gate on, rollout is blocked and no manual-create option is shown", async () => {
    await attachReviewConfig("ERROR");
    await env.api.updateProjectSettings(env.project, {
      requirePlanCheckNoError: true,
    });

    const ts = Date.now();
    await createPlanAndWaitForChecks("E2E Checks Error Gate-On", [
      {
        id: `spec-${ts}`,
        targets: [env.database],
        sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_err_on_${ts} TEXT;`,
      },
    ]);

    // The ERROR check is surfaced in the CHANGES "Checks" summary as an Error entry.
    await expect(planPage.checksSummary()).toContainText("Error", {
      timeout: 15_000,
    });
    // The blocking status moved from DEPLOY to the Review readiness footer (AIO
    // review section): no approval rule → SKIPPED, plus failed checks → the
    // footer reads "Review approved, but plan checks failed". The old DEPLOY
    // "Failed checks are blocking automatic rollout creation" helper text was
    // removed with the DeployFuture dedup.
    await planPage.expandSection("Review");
    await expect(
      page.getByText("Review approved, but plan checks failed"),
    ).toBeVisible({ timeout: 10_000 });
    // No manual create path is offered in this state (the gate is mandatory, so
    // the footer's bypass confirm sheet would hard-block deploy anyway).
    await expect(planPage.manualCreateRolloutButton).not.toBeVisible({
      timeout: 3_000,
    });
  });

  test("relaxing the gate (requirePlanCheckNoError=false) reveals the readiness-footer bypass action", async () => {
    await attachReviewConfig("ERROR");
    // Gate OFF — failed checks no longer block; user can bypass.
    await env.api.updateProjectSettings(env.project, {
      requirePlanCheckNoError: false,
    });

    const ts = Date.now();
    await createPlanAndWaitForChecks("E2E Checks Error Gate-Off", [
      {
        id: `spec-${ts}`,
        targets: [env.database],
        sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_err_off_${ts} TEXT;`,
      },
    ]);

    await expect(planPage.checksSummary()).toContainText("Error", {
      timeout: 15_000,
    });
    // No approval rule (seedTestData clears WORKSPACE_APPROVAL) →
    // approvalStatus SKIPPED; ERROR checks + gate OFF → the Review readiness
    // footer is "Review approved, but plan checks failed" and offers the
    // single manual path: "Bypass and deploy". The old "Manually create
    // rollout" button no longer exists for issue-backed plans.
    await planPage.expandSection("Review");
    await expect(
      planPage.page.getByText("Review approved, but plan checks failed"),
    ).toBeVisible({ timeout: 10_000 });
    await expect(planPage.bypassAndDeployAction).toBeVisible({
      timeout: 10_000,
    });
    await expect(planPage.manualCreateRolloutButton).not.toBeVisible({
      timeout: 3_000,
    });
  });
});

test.describe("Per-spec check counts render plan-wide (BYT-9160)", () => {
  // BYT-9160 (original): the per-spec right SIDEBAR always showed the LAST
  // spec's check counts regardless of which spec tab was selected. The React
  // migration REMOVED that sidebar; check counts are now a single PLAN-WIDE
  // aggregate summary (PlanDetailAggregateChecks). That UI element no longer
  // exists, so the original bug cannot recur. This test locks the resolution:
  // the aggregate summary renders and stays present regardless of the selected
  // spec (it is plan-wide, not per-spec).
  //
  // NOTE: the separate contract "selecting a spec shows only THAT spec's
  // STATEMENT" is a *different* concern and is currently BROKEN by a new
  // regression — switching tabs leaves the prior spec's statement editor
  // stacked. That is locked separately below (BUG BYT-9794),
  // not here.
  test("the aggregate check summary stays plan-wide across spec switches", async () => {
    const ts = Date.now();
    await createPlanAndWaitForChecks("E2E Plan-Wide Checks", [
      {
        id: `spec-a-${ts}`,
        targets: [env.database],
        sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_pw_a_${ts} TEXT;`,
      },
      {
        id: `spec-b-${ts}`,
        targets: [env.database],
        sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_pw_b_${ts} TEXT;`,
      },
    ]);

    await planPage.expandSection("Changes");

    // The plan-wide aggregate summary renders (the removed per-spec sidebar
    // would have shown per-spec counts here instead).
    await expect(page.getByText("Success").first()).toBeVisible({
      timeout: 15_000,
    });

    // It is plan-wide: still present after switching specs (the BYT-9160
    // sidebar would have re-bound to / gone stale on the selected spec).
    await planPage.specTab(1).click();
    await expect(page.getByText("Success").first()).toBeVisible();
    await planPage.specTab(2).click();
    await expect(page.getByText("Success").first()).toBeVisible();
  });
});

// NEW regression — distinct from BYT-9160 (whose buggy sidebar was deleted in
// the React migration). Switching spec tabs leaves the PREVIOUSLY-selected
// spec's statement EDITOR mounted in the CHANGES section, so both specs' SQL
// stack. Visible to users on any multi-spec plan. Root-cause lead: the old
// PlanDetailStatementSection is not unmounted on spec switch (despite
// key={selectedSpec.id}) — MonacoEditor disposes correctly on unmount, so the
// stale editor surviving means the component itself stays mounted; most likely
// a side effect of #20652's statement cache-seeding / re-derive-on-render.
// test.fail() until the product bug is fixed; flips to a passing guard then.
test.describe(
  "Spec tab switch must not leave a stale statement editor (BUG BYT-9794)",
  () => {
    test.fail(
      "only the selected spec's statement is shown in CHANGES after switching tabs",
      async () => {
        const ts = Date.now();
        const colA = `e2e_stale_a_${ts}`;
        const colB = `e2e_stale_b_${ts}`;
        await createPlanAndWaitForChecks("E2E Stale Spec Editor", [
          {
            id: `spec-a-${ts}`,
            targets: [env.database],
            sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS ${colA} TEXT;`,
          },
          {
            id: `spec-b-${ts}`,
            targets: [env.database],
            sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS ${colB} TEXT;`,
          },
        ]);

        await planPage.expandSection("Changes");

        // Read ONLY the CHANGES section's statement editors — the [role=code]
        // Monaco surfaces between the "Changes" and "Deploy" phase labels. This
        // deliberately excludes the DEPLOY task-statement preview (which shows
        // the first task's SQL and is independent of the spec tab), so the
        // assertion is purely about the CHANGES section leaking a stale editor.
        const readChangesStatements = (): Promise<string> =>
          page.evaluate(() => {
            const spans = Array.from(document.querySelectorAll("span"));
            const changesLabel = spans.find(
              (e) => e.textContent?.trim() === "Changes",
            );
            const deployLabel = spans.find(
              (e) => e.textContent?.trim() === "Deploy",
            );
            if (!changesLabel) return "";
            const FOLLOWING = Node.DOCUMENT_POSITION_FOLLOWING;
            const PRECEDING = Node.DOCUMENT_POSITION_PRECEDING;
            return Array.from(document.querySelectorAll('[role="code"]'))
              .filter(
                (c) =>
                  !!(changesLabel.compareDocumentPosition(c) & FOLLOWING) &&
                  (!deployLabel ||
                    !!(deployLabel.compareDocumentPosition(c) & PRECEDING)),
              )
              .flatMap((c) => Array.from(c.querySelectorAll(".view-line")))
              .map((l) => l.textContent ?? "")
              .join("\n");
          });

        await planPage.specTab(1).click();
        await expect
          .poll(readChangesStatements, { timeout: 15_000 })
          .toContain(colA);

        await planPage.specTab(2).click();
        await expect
          .poll(readChangesStatements, { timeout: 15_000 })
          .toContain(colB);

        // BUG: spec #1's statement editor is left mounted (stacked) in CHANGES,
        // so its SQL is still present after switching to spec #2.
        expect(await readChangesStatements()).not.toContain(colA);
      },
    );
  },
);
