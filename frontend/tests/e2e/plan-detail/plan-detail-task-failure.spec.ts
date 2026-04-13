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

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  projectId = env.project.split("/").pop()!;
  await env.api.login(env.adminEmail, env.adminPassword);

  // Ensure permissive project settings — no approval required, no plan-check gate
  await env.api.updateProjectSettings(env.project, {
    requireIssueApproval: false,
    requirePlanCheckNoError: false,
  });

  // Create a sheet targeting a nonexistent table — will fail at execution time
  const tableName = `nonexistent_table_e2e_${Date.now()}`;
  const sheetName = await env.api.createSheet(
    env.project,
    `ALTER TABLE ${tableName} ADD COLUMN c1 TEXT;`
  );

  // Create the plan
  const specId = `spec-${Date.now()}`;
  const plan = await env.api.createPlan(env.project, `E2E Task Failure Test ${Date.now()}`, [
    { id: specId, targets: [env.database], sheet: sheetName },
  ]);
  planId = plan.name.split("/").pop()!;

  // Create an issue backed by the plan (required to drive the lifecycle UI)
  await env.api.createIssue(env.project, "E2E Task Failure", plan.name);

  // Open the shared browser context
  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
  planPage = new PlanDetailPage(page, env.baseURL);
});

test.afterAll(async () => {
  await sharedContext?.close();
});

test.describe("Plan Detail: Task Failure (CUJ 4)", () => {
  test.describe.configure({ mode: "serial" });

  test("navigate and create rollout", async () => {
    await planPage.goto(projectId, planId);
    await planPage.dismissModals();
    await planPage.createRolloutWithBypass();
    await expect(page.getByText(/Not started|Deploying/i).first()).toBeVisible({ timeout: 15_000 });
  });

  test("run task that will fail", async () => {
    await planPage.runTask();
    await expect(page.getByText("Failed").first()).toBeVisible({ timeout: 30_000 });
  });

  test("task shows Failed status", async () => {
    await expect(page.getByText("Failed").first()).toBeVisible();
  });

  test("Retry button is visible", async () => {
    await expect(planPage.retryButton).toBeVisible();
  });

  test("sidebar shows Failed", async () => {
    await expect(page.getByRole("complementary").getByText("Failed").first()).toBeVisible();
  });

  test("sections remain expanded after failure (BYT-9161)", async () => {
    expect(await planPage.isSectionExpanded("Changes")).toBe(true);
    expect(await planPage.isSectionExpanded("Review")).toBe(true);
  });
});
