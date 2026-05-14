import { expect, type Page, type Locator } from "@playwright/test";

export class PlanDetailPage {
  readonly page: Page;
  readonly baseURL: string;

  readonly changesSection: Locator;
  readonly reviewSection: Locator;
  readonly deploySection: Locator;
  readonly manualCreateRolloutButton: Locator;
  readonly bypassWarningsCheckbox: Locator;
  readonly confirmRolloutButton: Locator;
  readonly retryButton: Locator;
  // The plan/issue title input rendered in PlanDetailHeader. It is an
  // <input> bound to the plan or issue title — first textbox on the page.
  readonly headerTitle: Locator;

  constructor(page: Page, baseURL: string) {
    this.page = page;
    this.baseURL = baseURL;

    this.changesSection = page.getByText("Changes").first();
    this.reviewSection = page.getByText("Review").first();
    this.deploySection = page.getByText("Deploy").first();
    this.manualCreateRolloutButton = page.getByRole("button", { name: "Manually create rollout" });
    this.bypassWarningsCheckbox = page.getByRole("checkbox", { name: "Bypass warnings" });
    this.confirmRolloutButton = page.getByRole("dialog").getByRole("button", { name: "Confirm" });
    this.retryButton = page.getByRole("button", { name: "Retry" });
    this.headerTitle = page.getByRole("textbox").first();
  }

  async goto(projectId: string, planId: string) {
    await this.page.goto(`${this.baseURL}/projects/${projectId}/plans/${planId}`);
    await this.page.waitForLoadState("networkidle");
  }

  async dismissModals() {
    await this.page.evaluate(() => {
      localStorage.setItem(
        "bb.release",
        JSON.stringify({
          ignoreRemindModalTillNextRelease: true,
          nextCheckTs: Date.now() + 86400000,
        })
      );
    });
    const dismiss = this.page.getByRole("button", { name: "Dismiss" });
    if (await dismiss.isVisible({ timeout: 2000 }).catch(() => false)) {
      await dismiss.click();
    }
  }

  async createRolloutWithBypass() {
    await this.manualCreateRolloutButton.click();
    await this.bypassWarningsCheckbox.check();
    // Arm the response waiter BEFORE clicking so a fast response isn't missed.
    const responsePromise = this.page.waitForResponse(
      (r) => r.url().includes("Rollout") && r.status() < 400
    );
    await this.confirmRolloutButton.click();
    await responsePromise;
  }

  async runTask() {
    // Match only buttons with exact text "Run" (not "Run check", "Run Tasks").
    // Use getByRole with exact name to avoid substring matches.
    const enabledRun = this.page
      .getByRole("button", { name: "Run", exact: true })
      .and(this.page.locator("button:not([disabled])"))
      .last();
    await expect(enabledRun).toBeVisible({ timeout: 15_000 });
    await enabledRun.click();
    const confirmDialog = this.page.getByRole("dialog").filter({ hasText: "Run task" });
    await confirmDialog.getByRole("button", { name: "Run" }).click();
  }

  getSectionToggle(sectionName: string): Locator {
    return this.page
      .getByText(sectionName, { exact: true })
      .first()
      .locator("..")
      .getByText(/Hide details|Show details/)
      .first();
  }

  async isSectionExpanded(sectionName: string): Promise<boolean> {
    const toggle = this.getSectionToggle(sectionName);
    if (await toggle.isVisible({ timeout: 1000 }).catch(() => false)) {
      const text = await toggle.textContent();
      return text?.includes("Hide") ?? false;
    }
    return true;
  }

  specTab(n: number): Locator {
    return this.page.getByText("Changes").locator("..").getByText(`#${n}`).first();
  }

  // Returns the count number for the given status (Warning/Success/Error)
  // in the inline Checks area (h3 level, not sidebar h4).
  inlineCheckCount(status: string): Locator {
    return this.page
      .getByRole("heading", { name: "Checks", level: 3 })
      .locator("../..")
      .getByText(status)
      .locator("..")
      .getByText(/^\d+$/);
  }

  // Returns the count number for the given status (Warning/Success/Error)
  // in the sidebar Checks area (h4 level).
  sidebarCheckCount(status: string): Locator {
    return this.page
      .getByRole("complementary")
      .getByRole("heading", { name: "Checks", level: 4 })
      .locator("..")
      .getByText(status)
      .locator("..")
      .getByText(/^\d+$/);
  }
}
