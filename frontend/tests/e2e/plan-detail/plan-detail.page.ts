import { expect, type Page, type Locator } from "@playwright/test";
import { BytebaseApiClient } from "../framework/api-client";

export class PlanDetailPage {
  readonly page: Page;
  readonly baseURL: string;

  readonly changesSection: Locator;
  readonly reviewSection: Locator;
  readonly deploySection: Locator;
  readonly sidebarStatus: Locator;
  readonly manualCreateRolloutButton: Locator;
  readonly bypassWarningsCheckbox: Locator;
  readonly confirmRolloutButton: Locator;
  readonly runTasksButton: Locator;
  readonly refreshButton: Locator;
  readonly retryButton: Locator;
  readonly deployBadge: Locator;
  readonly sidebarStatusLabel: Locator;

  constructor(page: Page, baseURL: string) {
    this.page = page;
    this.baseURL = baseURL;

    this.changesSection = page.getByText("Changes").first();
    this.reviewSection = page.getByText("Review").first();
    this.deploySection = page.getByText("Deploy").first();
    this.sidebarStatus = page.locator("complementary").getByText(/Status/i).locator("..");
    this.manualCreateRolloutButton = page.getByRole("button", { name: "Manually create rollout" });
    this.bypassWarningsCheckbox = page.getByRole("checkbox", { name: "Bypass warnings" });
    this.confirmRolloutButton = page.getByRole("dialog").getByRole("button", { name: "Confirm" });
    this.runTasksButton = page.getByRole("button", { name: "Run Tasks" });
    this.refreshButton = page.getByRole("button", { name: "Refresh" });
    this.retryButton = page.getByRole("button", { name: "Retry" });
    this.deployBadge = page.getByText(/Not started|In Progress|Done|Failed/i).first();
    this.sidebarStatusLabel = page
      .getByRole("complementary")
      .getByText(/Status/i)
      .locator("..")
      .getByText(/Open|Approved|Rejected|Canceled|Done/i);
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
    await this.confirmRolloutButton.click();
    await this.page.waitForResponse((r) => r.url().includes("Rollout") && r.status() < 400);
  }

  getTaskRunButton(nth = 0): Locator {
    return this.page.getByRole("button", { name: "Run", exact: true }).nth(nth);
  }

  getRunDialogConfirm(): Locator {
    return this.page.getByRole("dialog").getByRole("button", { name: "Run" });
  }

  async runTask() {
    // Use CSS :not([disabled]) on the element itself (not descendants) to find
    // an enabled Run button. Playwright's filter({ has/hasNot }) checks children,
    // which doesn't work for the button's own disabled attribute.
    const enabledRun = this.page.locator("button:not([disabled])", { hasText: "Run" }).last();
    await expect(enabledRun).toBeVisible({ timeout: 15_000 });
    await enabledRun.click();
    const confirmDialog = this.page.getByRole("dialog").filter({ hasText: "Run task" });
    await confirmDialog.getByRole("button", { name: "Run" }).click();
  }

  getSectionToggle(sectionName: string): Locator {
    // Find the section name text, then look for the toggle in the same parent container.
    // The toggle text includes an arrow: "Hide details ↑" or "Show details ↓".
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

  stageProgress(stageName: string): Locator {
    return this.page.getByText(stageName).locator("..").getByText(/\(\d+\/\d+\)/);
  }

  // Returns the spec tab locator for the nth spec (1-based) inside the Changes section.
  specTab(n: number): Locator {
    return this.page.getByText("Changes").locator("..").getByText(`#${n}`).first();
  }

  // Returns the count number sibling of the given status label (Warning/Success/Error)
  // in the inline Checks area within the Changes section (h3 level, not sidebar h4).
  inlineCheckCount(status: string): Locator {
    return this.page
      .getByRole("heading", { name: "Checks", level: 3 })
      .locator("../..")
      .getByText(status)
      .locator("..")
      .getByText(/^\d+$/);
  }

  // Returns the count number for the given status (Warning/Success/Error)
  // in the sidebar complementary Checks area (h4 level).
  sidebarCheckCount(status: string): Locator {
    return this.page
      .getByRole("complementary")
      .getByRole("heading", { name: "Checks", level: 4 })
      .locator("..")
      .getByText(/^\d+$/)
      .last();
  }

  // Polls api.getIssue() every 1 s until approvalStatus matches targetStatus or timeout elapses.
  async waitForApprovalStatus(
    api: BytebaseApiClient,
    issueName: string,
    targetStatus: string,
    timeoutMs = 30000
  ): Promise<void> {
    const deadline = Date.now() + timeoutMs;
    while (Date.now() < deadline) {
      const issue = await api.getIssue(issueName);
      if (issue.approvalStatus === targetStatus) return;
      await this.page.waitForTimeout(1000);
    }
    throw new Error(
      `Timed out waiting for approvalStatus "${targetStatus}" on issue "${issueName}" after ${timeoutMs}ms`
    );
  }
}
