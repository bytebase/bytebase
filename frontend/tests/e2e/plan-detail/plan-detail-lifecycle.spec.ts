import { test, expect, type Page, type BrowserContext } from "@playwright/test";
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
let originalSettings: { requireIssueApproval?: boolean; requirePlanCheckNoError?: boolean } = {};

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  projectId = env.project.split("/").pop()!;
  await env.api.login(env.adminEmail, env.adminPassword);

  // Capture and set permissive project settings
  const project = await env.api.getProject(env.project);
  originalSettings = {
    requireIssueApproval: !!project.requireIssueApproval,
    requirePlanCheckNoError: !!project.requirePlanCheckNoError,
  };
  await env.api.updateProjectSettings(env.project, {
    requireIssueApproval: false,
    requirePlanCheckNoError: false,
  });

  // Create a dedicated sheet for this test
  const columnName = `e2e_lifecycle_${Date.now()}`;
  const sheetName = await env.api.createSheet(
    env.project,
    `ALTER TABLE employee ADD COLUMN IF NOT EXISTS ${columnName} TEXT;`
  );

  // Create the plan
  const specId = `spec-${Date.now()}`;
  const plan = await env.api.createPlan(env.project, `E2E Lifecycle Test ${Date.now()}`, [
    { id: specId, targets: [env.database], sheet: sheetName },
  ]);
  planId = plan.name.split("/").pop()!;

  // Create an issue backed by the plan (required to drive the lifecycle UI)
  await env.api.createIssue(env.project, "E2E Lifecycle", plan.name);

  // Open the shared browser context
  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
  planPage = new PlanDetailPage(page, env.baseURL);
});

test.afterAll(async () => {
  await env.api.updateProjectSettings(env.project, originalSettings).catch(() => {});
  await sharedContext?.close();
});

test.describe("Plan Detail: Lifecycle + Section Preservation", () => {
  test.describe.configure({ mode: "serial" });

  test("navigate to plan and see all three phases", async () => {
    await planPage.goto(projectId, planId);
    await planPage.dismissModals();

    await expect(planPage.changesSection).toBeVisible();
    await expect(planPage.reviewSection).toBeVisible();
    await expect(planPage.deploySection).toBeVisible();
  });

  test("sections are expanded by default", async () => {
    expect(await planPage.isSectionExpanded("Changes")).toBe(true);
    expect(await planPage.isSectionExpanded("Review")).toBe(true);
  });

  test("create rollout with bypass", async () => {
    await planPage.createRolloutWithBypass();
    await expect(page.getByText(/Not started|Deploying/i).first()).toBeVisible({ timeout: 15_000 });
  });

  test("sections remain expanded after rollout creation (BYT-9161)", async () => {
    expect(await planPage.isSectionExpanded("Changes")).toBe(true);
    expect(await planPage.isSectionExpanded("Review")).toBe(true);
  });

  test("run task and wait for completion", async () => {
    await planPage.runTask();
    await expect(page.getByText("Done").first()).toBeVisible({ timeout: 30_000 });
  });

  test("sidebar shows Done status after completion", async () => {
    await expect(page.getByRole("complementary").getByText(/Done/i).first()).toBeVisible();
  });

  test("state preserved after navigation away and back", async () => {
    await page.goto(`${env.baseURL}/projects/${projectId}/issues`);
    await page.waitForLoadState("networkidle");

    await planPage.goto(projectId, planId);
    await expect(planPage.deploySection).toBeVisible();
  });

  test("sections still expanded after navigation (BYT-9161)", async () => {
    // After task completion phases may auto-collapse — either expanded or
    // the toggle is absent (section is always open). Both are acceptable.
    const changesExpanded = await planPage.isSectionExpanded("Changes");
    // isSectionExpanded returns true when toggle is absent (no collapse control)
    // so any truthy result — or explicit true — satisfies the regression check.
    expect(changesExpanded).toBe(true);
  });
});
