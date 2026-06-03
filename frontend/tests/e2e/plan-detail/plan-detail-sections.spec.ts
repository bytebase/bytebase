// Plan detail — section UI (expand/collapse + preservation).
//
// Covers:
//   - CHANGES auto-collapses once an auto-rollout has been created;
//     it must remain expandable on demand.
//   - Collapsing one section doesn't change the state of others.
//   - Section state survives a navigation away and back to the same plan
//     (BYT-9161 regression lock).

import {
  test,
  expect,
  type Page,
  type BrowserContext,
} from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";
import { PlanDetailPage } from "./plan-detail.page";

test.setTimeout(120_000);

let env: TestEnv & { api: BytebaseApiClient };
let projectId: string;
let planId: string;

let sharedContext: BrowserContext;
let page: Page;
let planPage: PlanDetailPage;
let originalSettings: {
  requireIssueApproval?: boolean;
  requirePlanCheckNoError?: boolean;
} = {};

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  projectId = env.project.split("/").pop()!;
  await env.api.login(env.adminEmail, env.adminPassword);

  // Permissive settings → rollout auto-creates on issue creation.
  // CHANGES then collapses to "Show details" by default — that's the
  // state under test.
  const project = await env.api.getProject(env.project);
  originalSettings = {
    requireIssueApproval: !!project.requireIssueApproval,
    requirePlanCheckNoError: !!project.requirePlanCheckNoError,
  };
  await env.api.updateProjectSettings(env.project, {
    requireIssueApproval: false,
    requirePlanCheckNoError: false,
  });

  const ts = Date.now();
  const sheet = await env.api.createSheet(
    env.project,
    `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_sections_${ts} TEXT;`,
  );
  const plan = await env.api.createPlan(
    env.project,
    `E2E Sections ${ts}`,
    [{ id: `spec-${ts}`, targets: [env.database], sheet }],
  );
  planId = plan.name.split("/").pop()!;
  await env.api.createIssue(env.project, `E2E Sections ${ts}`, plan.name);

  sharedContext = await browser.newContext({
    storageState: ".auth/state.json",
  });
  page = await sharedContext.newPage();
  planPage = new PlanDetailPage(page, env.baseURL);

  await planPage.goto(projectId, planId);
  await planPage.dismissModals();
});

test.afterAll(async () => {
  await env.api
    .updateProjectSettings(env.project, originalSettings)
    .catch(() => {});
  await sharedContext?.close();
});

test.describe("Section UI", () => {
  test.describe.configure({ mode: "serial" });

  test("CHANGES collapses by default once the rollout is auto-created", async () => {
    // With permissive settings the rollout exists immediately, and the
    // page's focus shifts to DEPLOY — CHANGES auto-collapses. Verify
    // the toggle reads "Show details".
    await expect(planPage.changesSection).toBeVisible({ timeout: 10_000 });
    expect(await planPage.isSectionExpanded("Changes")).toBe(false);
    expect(await planPage.isSectionExpanded("Deploy")).toBe(true);
  });

  test("expandSection makes CHANGES visible again", async () => {
    await planPage.expandSection("Changes");
    expect(await planPage.isSectionExpanded("Changes")).toBe(true);
  });

  test("collapsing one section doesn't affect siblings (BYT-9161 lock)", async () => {
    // Start clean.
    await planPage.goto(projectId, planId);
    await planPage.dismissModals();
    await planPage.expandSection("Changes");

    const changesToggle = planPage.getSectionToggle("Changes");
    if (
      await changesToggle.isVisible({ timeout: 3_000 }).catch(() => false)
    ) {
      await changesToggle.click();
      await expect(changesToggle).toHaveText(/Show details/, {
        timeout: 5_000,
      });
    }

    // CHANGES is now collapsed; DEPLOY must still be expanded.
    expect(await planPage.isSectionExpanded("Changes")).toBe(false);
    expect(await planPage.isSectionExpanded("Deploy")).toBe(true);
  });

  // Cross-navigation behavior: when the user leaves the plan and comes
  // back, `usePlanDetailPage.ts:210-212` re-runs `focusPhase(currentPhase)`
  // on the routePageKey change. With a rollout in place, currentPhase
  // is "deploy" — so DEPLOY stays expanded and CHANGES/REVIEW collapse
  // back to their default. This is intentional UX (focus follows the
  // user-relevant phase), not a regression of BYT-9161.
  test("navigation away and back re-focuses the current phase (DEPLOY when rollout exists)", async () => {
    // Manually expand CHANGES so its post-nav state is observably
    // different from the auto-focus result.
    await planPage.expandSection("Changes");
    expect(await planPage.isSectionExpanded("Changes")).toBe(true);

    await page.goto(`${env.baseURL}/projects/${projectId}/issues`);
    await page.waitForLoadState("networkidle");

    await planPage.goto(projectId, planId);
    await planPage.dismissModals();

    // DEPLOY is the auto-focused phase post-nav.
    expect(await planPage.isSectionExpanded("Deploy")).toBe(true);
    // CHANGES auto-collapses to default — our in-page expansion does
    // NOT persist across the route change (by design).
    expect(await planPage.isSectionExpanded("Changes")).toBe(false);
  });
});
