import { type Page, type Locator, expect } from "@playwright/test";

export class MaskingExemptionPage {
  readonly page: Page;
  private baseURL: string;

  // Navigation
  readonly grantExemptionButton: Locator;

  // Tabs
  readonly activeTab: Locator;
  readonly expiredTab: Locator;
  readonly allTab: Locator;

  // Search
  readonly searchBar: Locator;

  // Member list
  readonly memberList: Locator;

  // Detail panel
  readonly detailPanel: Locator;

  // Revoke dialog
  readonly revokeConfirmButton: Locator;
  readonly revokeCancelButton: Locator;

  constructor(page: Page, baseURL = "") {
    this.page = page;
    this.baseURL = baseURL;
    this.grantExemptionButton = page.getByRole("button", {
      name: /grant exemption/i,
    });
    this.activeTab = page.getByRole("button", { name: "Active" });
    this.expiredTab = page.getByRole("button", { name: "Expired" });
    this.allTab = page.getByRole("button", { name: "All" });
    this.searchBar = page.getByPlaceholder(/filter/i);
    this.memberList = page.locator("[class*='divide-y']").first();
    this.detailPanel = page.locator("[class*='flex-1 min-w-0']").first();
    this.revokeConfirmButton = page
      .getByRole("dialog")
      .getByRole("button", { name: "Confirm" });
    this.revokeCancelButton = page
      .getByRole("dialog")
      .getByRole("button", { name: "Cancel" });
  }

  async goto(projectId: string) {
    await this.page.goto(
      `${this.baseURL}/projects/${projectId}/masking-exemption`
    );
    // Dismiss any modal overlays (e.g., "New version available")
    await this.page.keyboard.press("Escape").catch(() => {});
    await this.page.waitForTimeout(300);
    // Wait for the page to finish loading
    await this.page
      .getByText(/exemptions|no exemptions/i)
      .first()
      .waitFor({ timeout: 10000 });
  }

  getMemberItem(email: string): Locator {
    return this.page.locator(`[class*="cursor-pointer"]`).filter({
      hasText: email,
    });
  }

  async selectMember(email: string) {
    await this.getMemberItem(email).click();
  }

  getGrantCard(title: string): Locator {
    return this.page.locator("[class*='border border-gray']").filter({
      hasText: title,
    });
  }

  getRevokeButton(grantTitle: string): Locator {
    return this.getGrantCard(grantTitle).getByRole("button", {
      name: "Revoke",
    });
  }

  async revokeGrant(grantTitle: string) {
    await this.getRevokeButton(grantTitle).click();
    await this.revokeConfirmButton.click();
    // Wait for optimistic update
    await this.page.waitForTimeout(500);
  }

  async getMemberCount(): Promise<number> {
    return this.page.locator("[class*='cursor-pointer']").filter({
      has: this.page.locator("[class*='rounded-full']"),
    }).count();
  }

  getExpiryLabel(grantTitle: string): Locator {
    return this.getGrantCard(grantTitle).locator("[class*='text-xs']").first();
  }
}

export class GrantExemptionPage {
  readonly page: Page;
  private baseURL: string;

  readonly allRadio: Locator;
  readonly expressionRadio: Locator;
  readonly selectRadio: Locator;
  readonly reasonInput: Locator;
  readonly expirationInput: Locator;
  readonly accountSelect: Locator;
  readonly confirmButton: Locator;
  readonly cancelButton: Locator;

  constructor(page: Page, baseURL = "") {
    this.page = page;
    this.baseURL = baseURL;
    this.allRadio = page.getByRole("radio", { name: "All", exact: true });
    this.expressionRadio = page.getByRole("radio", {
      name: /use cel/i,
    });
    this.selectRadio = page.getByRole("radio", {
      name: /manually select/i,
    });
    this.reasonInput = page.getByPlaceholder(/description/i);
    this.expirationInput = page.getByRole("textbox").nth(1);
    this.accountSelect = page.getByText("Select accounts", { exact: true });
    this.confirmButton = page.getByRole("button", { name: "Confirm" });
    this.cancelButton = page.getByRole("button", { name: "Cancel" });
  }

  async goto(projectId: string) {
    await this.page.goto(
      `${this.baseURL}/projects/${projectId}/masking-exemption/create`
    );
    await this.page.keyboard.press("Escape").catch(() => {});
    await this.page.waitForTimeout(300);
    await this.page
      .getByText(/grant exemption/i)
      .first()
      .waitFor({ timeout: 10000 });
  }

  async selectAccount(name: string) {
    await this.accountSelect.click();
    await this.page.getByText(name, { exact: true }).click();
    // Close dropdown
    await this.page.keyboard.press("Escape");
  }

  async submit() {
    await expect(this.confirmButton).toBeEnabled();
    await this.confirmButton.click();
  }
}

export class SqlEditorPage {
  readonly page: Page;
  private baseURL: string;
  readonly codeEditor: Locator;
  readonly runButton: Locator;

  constructor(page: Page, baseURL = "") {
    this.page = page;
    this.baseURL = baseURL;
    this.codeEditor = page.getByRole("code");
    this.runButton = page.getByRole("button", { name: /limit \d+/i });
  }

  async gotoWithDb(projectId: string, instanceId: string, dbName: string) {
    // Navigate directly to a SQL editor sheet URL with the database pre-selected.
    // This avoids stale editor state from previous navigations.
    await this.page.goto(
      `${this.baseURL}/sql-editor/projects/${projectId}/instances/${instanceId}/databases/${dbName}`
    );
    await this.page.keyboard.press("Escape").catch(() => {});
    await this.page.waitForTimeout(2000);
  }

  async runQuery(sql: string) {
    await this.page.keyboard.press("Escape").catch(() => {});
    await this.page.waitForTimeout(300);
    // Focus the editor
    await this.codeEditor.click();
    await this.page.waitForTimeout(200);
    // Select all and delete to clear editor
    await this.page.keyboard.press("Meta+a");
    await this.page.waitForTimeout(100);
    await this.page.keyboard.press("Delete");
    await this.page.waitForTimeout(200);
    // Verify editor is empty, then type the query
    await this.page.keyboard.type(sql, { delay: 10 });
    await this.page.waitForTimeout(300);
    await this.runButton.click();
    // Wait for results to load (look for "N rows" or "N row" indicator)
    await this.page.getByText(/\d+ rows?/).first().waitFor({ timeout: 15000 }).catch(() => {});
    await this.page.waitForTimeout(500);
  }

  async hasResultsTable(): Promise<boolean> {
    // Check for the "rows" indicator that appears after query execution
    return (await this.page.getByText(/\d+ rows?/).count()) > 0;
  }

  async resultContainsText(text: string, timeout = 15000): Promise<boolean> {
    // Scope to result grid rows via [data-row-index] attribute set by
    // VirtualDataTable. This avoids matching text in the SQL editor,
    // query history, or overflow-hidden ancestor containers.
    try {
      await this.page
        .locator("[data-row-index]")
        .filter({ hasText: text })
        .first()
        .waitFor({ state: "visible", timeout });
      return true;
    } catch {
      return false;
    }
  }
}
