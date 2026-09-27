import { test, expect } from "@playwright/test";
import { loadTestEnv } from "../framework/env";
import { PlanDetailPage } from "./plan-detail.page";
import { createDatabaseChangePlanViaUI } from "../framework/ui-create-plan";

test.setTimeout(60_000);

test("plan-detail page loads with title, changes section, and a spec tab", async ({
  browser,
}) => {
  const env = loadTestEnv();
  const projectId = env.project.split("/").pop()!;

  const context = await browser.newContext({ storageState: ".auth/state.json" });
  const page = await context.newPage();
  const planPage = new PlanDetailPage(page, env.baseURL);

  // Create the plan the way the UI does (plan + draft review issue together),
  // not a bare api.createPlan.
  const planTitle = `smoke-${Date.now()}`;
  const { planId } = await createDatabaseChangePlanViaUI(page, {
    baseURL: env.baseURL,
    projectId,
    database: env.database,
    title: planTitle,
    sql: "SELECT 1;",
  });

  await planPage.goto(projectId, planId);
  await planPage.dismissModals();

  await expect(planPage.headerTitle).toHaveValue(planTitle);
  await expect(planPage.changesSection).toBeVisible();
  await expect(planPage.specTab(1)).toBeVisible();

  await context.close();
});
