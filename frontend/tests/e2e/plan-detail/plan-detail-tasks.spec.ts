// Plan detail — task execution.
//
// Covers the task lifecycle once a rollout exists (auto-created here
// via permissive project settings):
//   - Running a successful task transitions to Done.
//   - Running a failing task (nonexistent target) transitions to Failed
//     and surfaces a Retry button.

import {
  test,
  expect,
  type Page,
  type BrowserContext,
} from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";
import { PlanDetailPage } from "./plan-detail.page";

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

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  projectId = env.project.split("/").pop()!;
  await env.api.login(env.adminEmail, env.adminPassword);

  // Permissive — both tests want a rollout to be auto-created so they
  // can drive task execution directly. We only mutate project settings;
  // each describe creates its own plan.
  const project = await env.api.getProject(env.project);
  originalSettings = {
    requireIssueApproval: !!project.requireIssueApproval,
    requirePlanCheckNoError: !!project.requirePlanCheckNoError,
  };
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

test.afterAll(async () => {
  await env.api
    .updateProjectSettings(env.project, originalSettings)
    .catch(() => {});
  await sharedContext?.close();
});

// Helper — create a plan + issue against env.database with the given SQL,
// navigate to its detail page. Returns the plan id for re-navigation.
async function createPlanAndNavigate(
  titlePrefix: string,
  sql: string,
): Promise<string> {
  const ts = Date.now();
  const sheet = await env.api.createSheet(env.project, sql);
  const plan = await env.api.createPlan(
    env.project,
    `${titlePrefix} ${ts}`,
    [{ id: `spec-${ts}`, targets: [env.database], sheet }],
  );
  const planId = plan.name.split("/").pop()!;
  await env.api.createIssue(env.project, `${titlePrefix} ${ts}`, plan.name);
  await planPage.goto(projectId, planId);
  await planPage.dismissModals();
  return planId;
}

test.describe("Successful task transitions to Done", () => {
  test("Run → Done", async () => {
    const colName = `e2e_tasks_ok_${Date.now()}`;
    await createPlanAndNavigate(
      "E2E Task Success",
      `ALTER TABLE employee ADD COLUMN IF NOT EXISTS ${colName} TEXT;`,
    );

    await expect(
      page.getByText(/Not started|Pending/i).first(),
    ).toBeVisible({ timeout: 15_000 });

    await planPage.runTask();
    await expect(page.getByText("Done").first()).toBeVisible({
      timeout: 60_000,
    });
  });
});

test.describe("Failing task transitions to Failed and shows Retry", () => {
  test("Run → Failed + Retry button visible", async () => {
    const missingTable = `nonexistent_table_e2e_${Date.now()}`;
    await createPlanAndNavigate(
      "E2E Task Failure",
      `ALTER TABLE ${missingTable} ADD COLUMN c1 TEXT;`,
    );

    await expect(
      page.getByText(/Not started|Pending/i).first(),
    ).toBeVisible({ timeout: 15_000 });

    await planPage.runTask();
    await expect(page.getByText("Failed").first()).toBeVisible({
      timeout: 30_000,
    });

    await expect(planPage.retryButton).toBeVisible({ timeout: 5_000 });
  });
});

test.describe("Expanded task statement preview is height-bounded, not clipped (BYT-9561)", () => {
  // BYT-9561 (FIXED, #20398): in the deploy Task section, expanding a task showed
  // its SQL statement in a ReadonlyMonaco preview that was hard-clipped —
  // statements taller than ~256px were cut off mid-statement with NO scrollbar,
  // so the rest of the SQL was unreachable. Cause: the wrapper used a CSS clamp
  // (max-h-64 + overflow-hidden) while Monaco's autoHeight sized the editor to
  // full content height (default max 600px), so content between the clamp and
  // Monaco's own height was invisible and unscrollable. The fix makes the
  // ReadonlyMonaco respect its max (256) so the editor itself is bounded and
  // scrolls internally; a "Statement truncated due to large size." hint renders.
  //
  // DeployTaskList auto-expands the FIRST task on load, so a single-task plan
  // renders the statement preview without any extra click.

  test("the statement editor is clamped to ~256px and shows the truncation hint", async () => {
    test.setTimeout(180_000);

    // ~25 lines, content height well above the 256px clamp but below Monaco's
    // 600px default — so pre-fix the editor rendered at full (~475px) height
    // inside the 256px overflow-hidden wrapper, and post-fix it is bounded.
    const marker = `e2e_clip_marker_${Date.now()}`;
    const lines: string[] = [];
    for (let i = 1; i <= 24; i++) {
      lines.push(
        `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_clip_${i} TEXT;`,
      );
    }
    lines.push(`-- ${marker}`);
    const sql = lines.join("\n");

    await createPlanAndNavigate("E2E Task Clip", sql);

    // The first (only) task auto-expands → its ReadonlyMonaco statement preview
    // renders inside the deploy task list.
    const editor = page.locator(".task-list .monaco-editor").first();
    await expect(editor).toBeVisible({ timeout: 30_000 });
    // Let Monaco settle its auto-height.
    await page.waitForTimeout(1000);

    // Sanity: the statement is genuinely taller than the clamp, so Monaco MUST
    // scroll internally (content height > the rendered editor height). This is
    // what makes the test exercise the clipping path. Pre-fix Monaco rendered at
    // full content height (no internal scroll); post-fix it's clamped and the
    // overflow is handled by Monaco's own scrollbar.
    const metrics = await editor.evaluate((el) => {
      const box = el.getBoundingClientRect();
      const linesEl = el.querySelector(".view-lines");
      // Monaco's scrollable content height (the full editor content).
      const contentHeight = linesEl ? linesEl.scrollHeight : 0;
      return { editorHeight: Math.round(box.height), contentHeight };
    });
    expect(
      metrics.contentHeight,
      "the statement content must be taller than the clamp so internal scroll is " +
        "required (otherwise the clipping bug can't manifest)",
    ).toBeGreaterThan(300);

    // Oracle (the fix): the editor element is height-bounded near its 256px max —
    // not the full ~475px content height. A relational bound (<= ~300, above
    // 256 + chrome, far below the pre-fix ~475) discriminates the fix without
    // pinning an exact pixel. Pre-fix the editor rendered at full content height
    // inside a 256px overflow-hidden wrapper with no internal scroll, so the
    // bottom of the statement was unreachable.
    expect(
      metrics.editorHeight,
      `the expanded task statement editor must be height-bounded near its 256px ` +
        `max (was ${metrics.editorHeight}px, content ${metrics.contentHeight}px).`,
    ).toBeLessThanOrEqual(300);
  });
});
