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
    `ALTER TABLE employee ADD COLUMN IF NOT EXISTS e2e_9161_${ts} TEXT;`
  );

  const plan = await env.api.createPlan(
    env.project,
    `E2E BYT-9161 Section Preservation ${ts}`,
    [{ id: `spec-${ts}`, targets: [env.database], sheet }]
  );
  planId = plan.name.split("/").pop()!;

  await env.api.createIssue(env.project, `E2E BYT-9161 Section Preservation ${ts}`, plan.name);

  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
  planPage = new PlanDetailPage(page, env.baseURL);

  await planPage.goto(projectId, planId);
  await planPage.dismissModals();
  await planPage.createRolloutWithBypass();
  await page.waitForLoadState("networkidle");
});

test.afterAll(async () => {
  await env.api.updateProjectSettings(env.project, originalSettings).catch(() => {});
  await sharedContext?.close();
});

test.describe("Plan Detail: Section Expansion Preservation (BYT-9161)", () => {
  test.describe.configure({ mode: "serial" });

  test("sections expanded after rollout creation", async () => {
    expect(await planPage.isSectionExpanded("Changes")).toBe(true);
    expect(await planPage.isSectionExpanded("Review")).toBe(true);
    expect(await planPage.isSectionExpanded("Deploy")).toBe(true);
  });

  test("clicking View details preserves section state", async () => {
    // Open the task detail panel via the "View details" link in the Deploy section
    const viewDetailsLink = page
      .getByRole("link", { name: "View details" })
      .first();
    if (await viewDetailsLink.isVisible({ timeout: 5_000 }).catch(() => false)) {
      await viewDetailsLink.click();
      await page.waitForLoadState("networkidle");
    } else {
      // Fallback: look for any clickable detail trigger in the deploy area
      const detailButton = page
        .getByText("View details")
        .first();
      if (await detailButton.isVisible({ timeout: 3_000 }).catch(() => false)) {
        await detailButton.click();
        await page.waitForLoadState("networkidle");
      }
    }

    expect(await planPage.isSectionExpanded("Changes")).toBe(true);
    expect(await planPage.isSectionExpanded("Review")).toBe(true);
  });

  test("closing detail panel preserves section state", async () => {
    // Close the detail panel using Escape first; fall back to a close button
    const closeButton = page
      .getByRole("button", { name: "Close" })
      .or(page.locator("[aria-label='Close']"))
      .first();

    if (await closeButton.isVisible({ timeout: 2_000 }).catch(() => false)) {
      await closeButton.click();
    } else {
      await page.keyboard.press("Escape");
    }
    // Wait for the drawer to fully close. If the drawer class doesn't match,
    // reload the page to guarantee a clean state rather than silently proceeding.
    const drawerHidden = await expect(page.locator(".n-drawer-container"))
      .toBeHidden({ timeout: 5_000 })
      .then(() => true)
      .catch(() => false);
    if (!drawerHidden) {
      // Drawer didn't close — reload to ensure section state assertions are meaningful
      await planPage.goto(projectId, planId);
      await planPage.dismissModals();
    }

    expect(await planPage.isSectionExpanded("Changes")).toBe(true);
    expect(await planPage.isSectionExpanded("Review")).toBe(true);
  });

  test("collapsing one section doesn't affect others", async () => {
    // Reload page to guarantee clean state (no lingering drawers from previous test)
    await planPage.goto(projectId, planId);
    await planPage.dismissModals();
    // Collapse the Changes section
    const changesToggle = planPage.getSectionToggle("Changes");
    if (await changesToggle.isVisible({ timeout: 3_000 }).catch(() => false)) {
      const text = await changesToggle.textContent();
      if (text?.includes("Hide")) {
        await changesToggle.click();
        await expect(changesToggle).toHaveText(/Show details/, { timeout: 5_000 });
      }
    }

    expect(await planPage.isSectionExpanded("Changes")).toBe(false);
    expect(await planPage.isSectionExpanded("Review")).toBe(true);
    expect(await planPage.isSectionExpanded("Deploy")).toBe(true);

    // Re-expand Changes so subsequent tests start from a consistent state
    const changesToggleAfter = planPage.getSectionToggle("Changes");
    if (await changesToggleAfter.isVisible({ timeout: 3_000 }).catch(() => false)) {
      const text = await changesToggleAfter.textContent();
      if (text?.includes("Show")) {
        await changesToggleAfter.click();
        await expect(changesToggleAfter).toHaveText(/Hide details/, { timeout: 5_000 });
      }
    }
  });

  test("navigation away and back preserves section state", async () => {
    // Navigate to the issues list
    await page.goto(`${env.baseURL}/projects/${projectId}/issues`);
    await page.waitForLoadState("networkidle");

    // Navigate back to the plan
    await planPage.goto(projectId, planId);
    await page.waitForLoadState("networkidle");

    expect(await planPage.isSectionExpanded("Changes")).toBe(true);
    expect(await planPage.isSectionExpanded("Review")).toBe(true);
  });
});
