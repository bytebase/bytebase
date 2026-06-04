import { expect, type Page, type Locator } from "@playwright/test";

export class PlanDetailPage {
  readonly page: Page;
  readonly baseURL: string;

  readonly changesSection: Locator;
  readonly deploySection: Locator;
  readonly manualCreateRolloutButton: Locator;
  readonly retryButton: Locator;
  // The plan/issue title input rendered in PlanDetailHeader. It is an
  // <input> bound to the plan or issue title — first textbox on the page.
  readonly headerTitle: Locator;

  constructor(page: Page, baseURL: string) {
    this.page = page;
    this.baseURL = baseURL;

    this.changesSection = page.getByText("Changes").first();
    this.deploySection = page.getByText("Deploy").first();
    this.manualCreateRolloutButton = page.getByRole("button", { name: "Manually create rollout" });
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

  // Click "Show details" if collapsed; no-op when already expanded or when no
  // toggle exists (always-open section). When the rollout already exists the
  // CHANGES section auto-collapses by default — call this before reading any
  // content inside CHANGES (check counts, spec tabs).
  async expandSection(sectionName: string): Promise<void> {
    const toggle = this.getSectionToggle(sectionName);
    if (!(await toggle.isVisible({ timeout: 1000 }).catch(() => false))) return;
    const text = (await toggle.textContent()) ?? "";
    if (text.includes("Show")) {
      await toggle.click();
      await expect(toggle).toHaveText(/Hide details/, { timeout: 5_000 });
    }
  }

  // Spec tabs render as `<button>N. <Kind></button>` (e.g.
  // "1. Database Change") inside the CHANGES section. The "1." and
  // "Database Change" pieces live in separate child spans — so plain
  // textContent has no separator. Matching the BUTTON's accessible name
  // (which inserts the space) via getByRole sidesteps that.
  //
  // Caller must expandSection("Changes") first if a rollout exists,
  // since the section auto-collapses in that state and the tab won't be
  // in the visible DOM.
  specTab(n: number): Locator {
    return this.page
      .getByRole("button", { name: new RegExp(`^${n}\\.\\s+\\w`) })
      .first();
  }

  // The plan-wide "Checks" summary button in the CHANGES section. Shows
  // Success/Warning/Error entries (each rendered only when its count > 0)
  // and opens the results drawer on click. Distinct from "Run checks".
  checksSummary(): Locator {
    return this.page.getByRole("button", { name: "Checks", exact: true });
  }
}
