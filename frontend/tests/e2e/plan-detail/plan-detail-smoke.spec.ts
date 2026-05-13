import { test, expect } from "@playwright/test";
import { loadTestEnv } from "../framework/env";
import { PlanDetailPage } from "./plan-detail.page";

test.setTimeout(60_000);

test("plan-detail page loads with title, changes section, and a spec tab", async ({
  browser,
}) => {
  const env = loadTestEnv();
  const projectId = env.project.split("/").pop()!;
  await env.api.login(env.adminEmail, env.adminPassword);

  const planTitle = `smoke-${Date.now()}`;
  const sheetName = await env.api.createSheet(env.project, "SELECT 1;");
  const specId = `spec-${Date.now()}`;
  const plan = await env.api.createPlan(env.project, planTitle, [
    { id: specId, targets: [env.database], sheet: sheetName },
  ]);
  const planId = plan.name.split("/").pop()!;

  const context = await browser.newContext({ storageState: ".auth/state.json" });
  const page = await context.newPage();
  const planPage = new PlanDetailPage(page, env.baseURL);

  await planPage.goto(projectId, planId);
  await planPage.dismissModals();

  await expect(planPage.headerTitle).toHaveValue(planTitle);
  await expect(planPage.changesSection).toBeVisible();
  await expect(planPage.specTab(1)).toBeVisible();

  await context.close();
});
