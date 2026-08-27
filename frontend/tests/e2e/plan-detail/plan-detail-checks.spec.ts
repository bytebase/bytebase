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
//     the per-spec sidebar was removed), and switching spec tabs shows only the
//     selected spec's statement (BYT-9794 — a duplicate-key regression fixed by
//     #20662; guarded here so it can't re-regress).
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
import { createSubmittedMultiSpecChangePlanViaUI } from "../framework/ui-create-plan";
import { PlanDetailPage } from "./plan-detail.page";
import { waitForPlanChecksDone } from "./plan-helpers";

test.setTimeout(180_000);

let env: TestEnv & { api: BytebaseApiClient };
let projectId: string;
// Two distinct sample databases for multi-spec plans. sqlMap in the UI create
// URL is keyed by database, so two specs must target two different DBs — both
// carry the same HR schema (employee), so the review SQL applies to either.
let testDb: string;
let prodDb: string;

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

  const hrTest = await env.api.findDatabaseByShortName("hr_test", env.project);
  const hrProd = await env.api.findDatabaseByShortName("hr_prod", env.project);
  if (!hrTest || !hrProd) {
    throw new Error("plan-check multi-spec setup needs hr_test + hr_prod sample dbs");
  }
  testDb = hrTest.database;
  prodDb = hrProd.database;

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

// Create an N-spec change plan through the UI (create + "Ready for Review"),
// wait for its plan checks to finish, then navigate to the plan detail page.
// Returns the planId. Submitting the review issue is what makes checks run and
// (under permissive settings) a rollout auto-create — the same states the old
// bare api.createPlan + api.createIssue fabricated, now produced via the real
// user workflow so drift breaks loudly. Each spec is one DB; multi-spec callers
// pass distinct DBs (sqlMap is keyed by database).
async function createPlanAndWaitForChecks(
  titlePrefix: string,
  specs: { database: string; sql: string }[],
): Promise<string> {
  const ts = Date.now();
  const { planId } = await createSubmittedMultiSpecChangePlanViaUI(page, {
    baseURL: env.baseURL,
    projectId,
    title: `${titlePrefix} ${ts}`,
    specs,
  });
  const planName = `${env.project}/plans/${planId}`;
  // Submitting the review issue already triggers the plan checks — no explicit
  // runPlanChecks (an API-only escape hatch outside the user workflow that also
  // contends with submit's rollout-creation transaction). Just wait for the
  // submit-triggered checks to settle before the assertions read the summary.
  await waitForPlanChecksDone(env.api, planName);

  await planPage.goto(projectId, planId);
  await planPage.dismissModals();
  return planId;
}

// Read the CHANGES-section statement editors' text — the [role=code] Monaco
// surfaces between the "Changes" and "Deploy" phase labels. Deliberately
// excludes the DEPLOY task-statement preview (which shows the first task's SQL
// and is independent of the spec tab), so the read is purely about CHANGES.
function readChangesStatements(): Promise<string> {
  return page.evaluate(() => {
    const spans = Array.from(document.querySelectorAll("span"));
    const changesLabel = spans.find((e) => e.textContent?.trim() === "Changes");
    const deployLabel = spans.find((e) => e.textContent?.trim() === "Deploy");
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
}

test.describe("WARNING-level review rule", () => {
  test("violating SQL produces a warning but does not block auto-rollout", async () => {
    await attachReviewConfig("WARNING");

    const ts = Date.now();
    await createPlanAndWaitForChecks("E2E Checks Warning", [
      {
        database: env.database,
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
  // Product contract observed on the free-plan sample setup (2026-05):
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
        database: env.database,
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
        database: env.database,
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
  // aggregate summary (PlanDetailAggregateChecks, rendered once in
  // PlanDetailChangesBranch.tsx). That per-spec UI no longer exists, so the
  // original bug cannot recur — the aggregate is plan-wide by construction (one
  // component, not one-per-spec). This test locks the resolution: the aggregate
  // summary renders for a multi-spec plan.
  //
  // (Originally this clicked through spec tabs to prove the summary stayed put;
  // that dance is gone — clicking a spec tab currently collapses the CHANGES
  // section, locked separately below as the spec-switch BUG.)
  test("the plan-wide aggregate check summary renders for a multi-spec plan", async () => {
    const ts = Date.now();
    await createPlanAndWaitForChecks("E2E Plan-Wide Checks", [
      {
        database: testDb,
        sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_pw_a_${ts} TEXT;`,
      },
      {
        database: prodDb,
        sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_pw_b_${ts} TEXT;`,
      },
    ]);

    await planPage.expandSection("Changes");

    // The single plan-wide aggregate summary renders a Success entry covering
    // all specs (the removed per-spec sidebar would have shown per-spec counts).
    await expect(page.getByText(/Success/).first()).toBeVisible({
      timeout: 15_000,
    });
  });
});

// BYT-9794 (FIXED, #20662): switching spec tabs used to leave the
// PREVIOUSLY-selected spec's statement editor mounted, so both specs' SQL
// stacked in CHANGES. Root cause: the spec-detail sections (TargetsSection /
// StatementSection / OptionsSection) each carried the SAME `key={selectedSpec
// .id}` — duplicate React keys broke reconciliation. Fixed by consolidating
// them under a single keyed wrapper `<div key={selectedSpec.id}>`. Guarded here
// by switching between the two spec tabs and asserting only the selected
// spec's statement is mounted at each step (never both stacked), alongside the
// spec-scoped URL / history contract (BYT-9805 / BYT-9913).
//
// The plan is created through the UI with two DISTINCT target databases
// (hr_test / hr_prod): the UI create page keys per-spec SQL by database, so a
// two-spec plan needs two targets — which also makes each tab's label
// ("Change N: <db>") distinct.
test.describe(
  "Spec identity and resource routing stay synchronized (BYT-9794/BYT-9805/BYT-9913)",
  () => {
    test(
      "target-derived tabs, URLs, history, and the selected statement move together",
      async () => {
        const ts = Date.now();
        const colA = `e2e_stale_a_${ts}`;
        const colB = `e2e_stale_b_${ts}`;
        const planId = await createPlanAndWaitForChecks(
          "E2E Stale Spec Editor",
          [
            {
              database: testDb,
              sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS ${colA} TEXT;`,
            },
            {
              database: prodDb,
              sql: `ALTER TABLE employee ADD COLUMN IF NOT EXISTS ${colB} TEXT;`,
            },
          ],
        );
        // The plan was created through the UI, so the product assigned each
        // spec a generated id. Read the real ids (in spec order — spec 1 targets
        // testDb, spec 2 targets prodDb) for the spec-scoped URL assertions.
        const created = await env.api.getPlan(`${env.project}/plans/${planId}`);
        const specIds = (created.specs ?? []).map((sp) => sp.id);
        expect(specIds).toHaveLength(2);
        const [specA, specB] = specIds;
        const testDbId = testDb.split("/").pop()!;
        const prodDbId = prodDb.split("/").pop()!;

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

        const tab1 = planPage.specTab(1);
        const tab2 = planPage.specTab(2);
        await expect(tab1).toHaveAccessibleName(`Change 1: ${testDbId}`);
        await expect(tab2).toHaveAccessibleName(`Change 2: ${prodDbId}`);
        await expect(
          page.getByRole("button", { name: /Database Change/ }),
        ).toHaveCount(0);

        await expect
          .poll(readChangesStatements, { timeout: 15_000 })
          .toContain(colA);

        await tab2.click();
        await expect(page).toHaveURL(
          new RegExp(`/plans/${planId}/specs/${specB}$`),
        );
        await expect
          .poll(readChangesStatements, { timeout: 15_000 })
          .toContain(colB);

        // Post-fix (#20662): spec #1's editor is unmounted on switch, so only
        // spec #2's statement remains. (Pre-fix, duplicate keys left spec #1's
        // editor stacked and this assertion failed.)
        expect(await readChangesStatements()).not.toContain(colA);

        await tab1.click();
        await expect(page).toHaveURL(
          new RegExp(`/plans/${planId}/specs/${specA}$`),
        );
        await expect
          .poll(readChangesStatements, { timeout: 15_000 })
          .toContain(colA);

        // Same-plan Back restores resource selection without remounting the
        // shell or collapsing Changes (BYT-9913 resource-routing contract).
        await page.goBack();
        await expect(page).toHaveURL(
          new RegExp(`/plans/${planId}/specs/${specB}$`),
        );
        await expect(tab2).toBeVisible();
        await expect
          .poll(readChangesStatements, { timeout: 15_000 })
          .toContain(colB);
      },
    );
  },
);
