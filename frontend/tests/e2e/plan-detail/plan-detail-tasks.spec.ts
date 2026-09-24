// Plan detail — task execution.
//
// Covers the task lifecycle once a rollout exists (auto-created here
// via permissive project settings):
//   - Running a successful task transitions to Done.
//   - Running a failing task (nonexistent target) transitions to Failed
//     and surfaces a Rerun button.

import {
  test,
  expect,
  type Page,
  type BrowserContext,
} from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";
import { createSubmittedDatabaseChangePlanViaUI } from "../framework/ui-create-plan";
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

// Helper — create + submit a plan against env.database with the given SQL
// through the UI (the real user workflow: plan + draft review issue, then
// "Ready for Review"), so a rollout auto-creates under the permissive settings
// this file sets. Navigate to its detail page; return the plan id.
async function createPlanAndNavigate(
  titlePrefix: string,
  sql: string,
): Promise<string> {
  const { planId } = await createSubmittedDatabaseChangePlanViaUI(page, {
    baseURL: env.baseURL,
    projectId,
    database: env.database,
    title: `${titlePrefix} ${Date.now()}`,
    sql,
  });
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

test.describe("Failing task transitions to Failed and shows Rerun", () => {
  test("Run → Failed + Rerun button visible", async () => {
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

    await expect(planPage.taskRerunButton).toBeVisible({ timeout: 5_000 });
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

test.describe("Failed task's log keeps the statement that failed (BYT-10168)", () => {
  // The log used to cut every statement to 80 characters and, on a failed
  // command, show the driver's error alone — so the row a reader opened the log
  // for could not say which SQL produced it, and nothing could be copied.
  // See docs/design/task-run-log-statement-unfold.md.
  test("a full log keeps its height; the failure unfolds from its line, copies, and folds back into place", async () => {
    test.setTimeout(180_000);
    await sharedContext.grantPermissions(
      ["clipboard-read", "clipboard-write"],
      { origin: env.baseURL },
    );

    // Nine successes then the failure: ten rows fill the section's box, and
    // the failing statement (16 lines, ~300px) is taller than the box on its
    // own. Not longer: the create page's editor renders about 26 lines, and
    // the plan helper checks the first line is still in view after insertion.
    const stamp = Date.now();
    const okColumn = (index: number) => `e2e_log_ok_${stamp}_${index}`;
    const missingTable = `nonexistent_table_e2e_${stamp}`;
    const failedLines = Array.from(
      { length: 15 },
      (_, index) => `  ADD COLUMN c${index + 1} TEXT${index === 14 ? ";" : ","}`,
    );
    await createPlanAndNavigate(
      "E2E Task Log",
      [
        ...Array.from({ length: 9 }, (_, index) =>
          `ALTER TABLE employee ADD COLUMN IF NOT EXISTS ${okColumn(index)} TEXT;`,
        ),
        `ALTER TABLE ${missingTable}`,
        ...failedLines,
      ].join("\n"),
    );

    await planPage.runTask();
    await expect(page.getByText("Failed").first()).toBeVisible({
      timeout: 30_000,
    });

    const rows = page.getByTestId("task-run-log-row");
    const failedRow = rows.filter({ hasText: "does not exist" });
    const rowText = (row: typeof failedRow) =>
      row.evaluate((element) => element.textContent ?? "");
    const failedSql = new RegExp(
      `ALTER TABLE ${missingTable}\\s*\\n\\s*ADD COLUMN c1 TEXT,[\\s\\S]*ADD COLUMN c15 TEXT;`,
    );
    const box = failedRow.locator("xpath=..");
    const boxHeight = () =>
      box.evaluate((element) => element.getBoundingClientRect().height);
    // Where the box sits on screen: unchanged means nothing around it moved,
    // whichever container the page scrolls in.
    const boxY = () => box.evaluate((element) => element.getBoundingClientRect().y);
    const errorText = failedRow.getByText("does not exist");
    const lineY = async () => (await errorText.boundingBox())?.y ?? Number.NaN;

    // Nothing is unfolded for the reader: the failure shows its error, folded,
    // and the section is scrolled to it — it is the tenth row, at the box's
    // bottom.
    await expect(failedRow).toBeVisible({ timeout: 30_000 });
    await expect(failedRow).toContainText(`"${missingTable}" does not exist`);
    const showFailed = failedRow.getByRole("button", {
      name: "Show full statement",
    });
    await expect(showFailed).toHaveAttribute("aria-expanded", "false");
    expect(await rowText(failedRow)).not.toMatch(failedSql);
    await expect(rows).toHaveCount(10);
    // The reader has scrolled the page to the failure; from here on nothing
    // the toggles do may move it.
    await failedRow.scrollIntoViewIfNeeded();
    const fullBoxHeight = await boxHeight();
    const parkedY = await lineY();
    const boxYBefore = await boxY();

    // Every line with a fold control toggles on click, the error included. The
    // box does not grow: the statement that failed opens inside it, and since
    // it is taller than the box the line goes to the box's top so the SQL can
    // fill the rest.
    await expect(errorText).toHaveCSS("cursor", "pointer");
    await errorText.click();
    const hideFailed = failedRow.getByRole("button", {
      name: "Hide full statement",
    });
    await expect(hideFailed).toHaveAttribute("aria-expanded", "true");
    await expect.poll(() => rowText(failedRow)).toMatch(failedSql);
    expect(await boxHeight()).toBe(fullBoxHeight);
    expect(await boxY()).toBe(boxYBefore);
    expect(Math.abs((await lineY()) - boxYBefore)).toBeLessThanOrEqual(6);
    const revealScroll = await box.evaluate((element) => element.scrollTop);

    // Copy takes the SQL, never the error — and stays in reach: scrolled to the
    // end of a block taller than the box, the button is still inside the box.
    const copy = failedRow.getByRole("button", { name: "Copy" });
    await box.evaluate((element) => {
      element.scrollTop = element.scrollHeight;
    });
    const copyBox = await copy.boundingBox();
    const boxBox = await box.boundingBox();
    expect(copyBox).not.toBeNull();
    expect(copyBox!.y).toBeGreaterThanOrEqual(boxBox!.y);
    expect(copyBox!.y + copyBox!.height).toBeLessThanOrEqual(
      boxBox!.y + boxBox!.height,
    );
    await copy.click();
    const clipboard = () => page.evaluate(() => navigator.clipboard.readText());
    await expect.poll(clipboard).toMatch(failedSql);
    expect(await clipboard()).not.toContain("does not exist");

    // Folding puts the line back where it was clicked from: the box keeps its
    // height and its place on the page, and the row returns to the box's
    // bottom.
    await box.evaluate((element, top) => {
      element.scrollTop = top;
    }, revealScroll);
    await errorText.click();
    await expect(showFailed).toHaveAttribute("aria-expanded", "false");
    await expect(failedRow).not.toContainText("ADD COLUMN c1 TEXT");
    await expect(failedRow).toContainText("does not exist");
    expect(await boxHeight()).toBe(fullBoxHeight);
    expect(await boxY()).toBe(boxYBefore);
    expect(Math.abs((await lineY()) - parkedY)).toBeLessThanOrEqual(1);

    // A successful statement that fits its line has no fold control, and
    // copies from the row.
    const okRow = rows.filter({ hasText: okColumn(3) });
    await expect(
      okRow.getByRole("button", { name: "Show full statement" }),
    ).toHaveCount(0);
    await okRow.hover();
    await okRow.getByRole("button", { name: "Copy" }).click();
    await expect.poll(clipboard).toContain(okColumn(3));
  });
});
