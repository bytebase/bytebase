import { type Page, type Locator, expect } from "@playwright/test";

export class MaskingExemptionPage {
  readonly page: Page;
  private baseURL: string;

  readonly grantExemptionButton: Locator;
  readonly activeTab: Locator;
  readonly expiredTab: Locator;
  readonly allTab: Locator;

  constructor(page: Page, baseURL = "") {
    this.page = page;
    this.baseURL = baseURL;
    this.grantExemptionButton = page.getByRole("button", {
      name: /grant exemption/i,
    });
    this.activeTab = page.getByRole("tab", { name: "Active" });
    this.expiredTab = page.getByRole("tab", { name: "Expired" });
    this.allTab = page.getByRole("tab", { name: "All" });
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

  // Select a member by email. Uses data-testid for a stable locator,
  // filtered by the visible email text.
  getMemberItem(email: string): Locator {
    return this.page
      .getByTestId("exemption-member-item")
      .filter({ hasText: email });
  }

  async selectMember(email: string) {
    // Fail fast (10s) with a descriptive error if the member item is missing,
    // instead of letting the surrounding test timeout drag out to 120s.
    const item = this.getMemberItem(email).first();
    await expect(
      item,
      `member item for "${email}" (data-testid="exemption-member-item") not found`
    ).toBeVisible({ timeout: 10000 });
    await item.click();
  }
}

export class GrantExemptionPage {
  readonly page: Page;
  private baseURL: string;

  readonly allRadio: Locator;
  readonly reasonInput: Locator;
  readonly accountSelect: Locator;
  readonly confirmButton: Locator;

  constructor(page: Page, baseURL = "") {
    this.page = page;
    this.baseURL = baseURL;
    this.allRadio = page.getByRole("radio", { name: "All", exact: true });
    this.reasonInput = page.getByPlaceholder(/description/i);
    this.accountSelect = page.getByText("Select accounts", { exact: true });
    this.confirmButton = page.getByRole("button", { name: "Confirm" });
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
    // Select all and delete to clear editor (ControlOrMeta for cross-platform)
    await this.page.keyboard.press("ControlOrMeta+a");
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
