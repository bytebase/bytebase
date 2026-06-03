// Plan detail — SQL review / plan checks.
//
// Covers:
//   - WARNING-level rule produces an inline warning count but does NOT
//     block rollout creation (auto-rollout still happens).
//   - ERROR-level rule with `requirePlanCheckNoError=true` blocks the
//     auto-rollout with NO "Manually create rollout" button; relaxing the
//     gate (requirePlanCheckNoError=false) reveals the manual-create button.
//   - Multi-spec plans: smoke test that each spec tab is selectable and
//     renders an inline check summary (BYT-9160 context; NOT a strict
//     regression lock — see the in-test comment).
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
  //     manual create button DOES appear so the user can bypass the
  //     failed checks intentionally.
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
    // DEPLOY explains the block.
    await expect(
      page.getByText(/Failed checks are blocking/i).first(),
    ).toBeVisible({ timeout: 10_000 });
    // No manual create path is offered in this state.
    await expect(planPage.manualCreateRolloutButton).not.toBeVisible({
      timeout: 3_000,
    });
  });

  test("relaxing the gate (requirePlanCheckNoError=false) reveals the manual-create button", async () => {
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
    await expect(planPage.manualCreateRolloutButton).toBeVisible({
      timeout: 10_000,
    });
  });
});

test.describe("Per-spec scoping (BYT-9160)", () => {
  // BYT-9160 was a per-spec rendering bug: the right sidebar always showed
  // the LAST spec's check counts regardless of which spec tab was selected.
  // The React migration REMOVED that right sidebar. Per-spec data now lives
  // in the CHANGES editor (the statement + inline advice markers, scoped to
  // the selected spec via planCheckRunListForSpec), while check COUNTS are
  // shown PLAN-WIDE (PlanDetailAggregateChecks: one Success/Warning/Error
  // summary that opens a results drawer). There is no per-spec count to
  // compare anymore, so the surviving contract is: selecting a spec shows
  // THAT spec's statement, not a stale sibling's. The two specs carry
  // uniquely-stamped columns, so the BYT-9160-class regression (stale
  // per-spec content on tab switch) fails loudly.
  test("each spec tab shows its own statement; checks render plan-wide", async () => {
    const ts = Date.now();
    const colA = `e2e_spec_a_${ts}`;
    const colB = `e2e_spec_b_${ts}`;
    await createPlanAndWaitForChecks("E2E Per-Spec", [
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

    // Expand CHANGES so the spec tabs + statement editor are reachable.
    await planPage.expandSection("Changes");

    // Joined text of every mounted Monaco surface. The CHANGES statement
    // editor renders only the SELECTED spec (PlanDetailStatementSection is
    // driven by selectedSpec), so this reflects the active spec's statement.
    const readStatement = (): Promise<string> =>
      page.evaluate(() =>
        Array.from(document.querySelectorAll('[role="code"]'))
          .flatMap((c) => Array.from(c.querySelectorAll(".view-line")))
          .map((l) => l.textContent ?? "")
          .join("\n"),
      );

    // Spec #1 selected: its statement shows; spec #2's does not.
    await planPage.specTab(1).click();
    await expect.poll(readStatement, { timeout: 15_000 }).toContain(colA);
    expect(await readStatement()).not.toContain(colB);

    // Switching to spec #2 swaps the CHANGES content to spec #2's statement.
    // The BYT-9160 regression would leave spec #1's content rendered here.
    await planPage.specTab(2).click();
    await expect.poll(readStatement, { timeout: 15_000 }).toContain(colB);
    expect(await readStatement()).not.toContain(colA);

    // Check counts are now a single plan-wide summary (the per-spec sidebar
    // is gone); confirm it renders.
    await expect(page.getByText("Success").first()).toBeVisible({
      timeout: 15_000,
    });
  });
});
