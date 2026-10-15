import type { Locator, Page } from "@playwright/test";

// Project members page, its Grant Access sheet, and a member's edit drawer.
export class ProjectMembersPage {
  readonly page: Page;
  readonly baseURL: string;
  readonly sheet: Locator;
  readonly grantAccessButton: Locator;
  readonly createButton: Locator;
  readonly directExecutionSwitch: Locator;
  readonly environmentPicker: Locator;

  constructor(page: Page, baseURL = "") {
    this.page = page;
    this.baseURL = baseURL;
    // Toasts also carry role="dialog" (with a data-type); the sheet does not.
    this.sheet = page.locator("[role='dialog']:not([data-type])");
    this.grantAccessButton = page.getByRole("button", { name: "Grant Access" });
    this.createButton = this.sheet.getByRole("button", { name: "Create", exact: true });
    this.directExecutionSwitch = this.sheet.getByRole("switch");
    this.environmentPicker = this.sheet.getByText("Select environments", { exact: true });
  }

  async goto(projectId: string): Promise<void> {
    await this.page.goto(`${this.baseURL}/projects/${projectId}/members`);
    await this.grantAccessButton.waitFor({ timeout: 15_000 });
  }

  async openGrantAccess(): Promise<void> {
    await this.grantAccessButton.click();
    await this.sheet.waitFor();
  }

  // Escape would close the whole sheet, so a popup is dismissed by clicking
  // the sheet's own heading.
  private async dismissPopup(): Promise<void> {
    await this.sheet.getByRole("heading").first().click({ force: true });
  }

  // The same email and titles usually also sit in a table behind the sheet,
  // so every pick stays inside the open listbox.
  async pickAccount(email: string): Promise<void> {
    await this.sheet.getByText("Select accounts", { exact: true }).click();
    await this.page.getByPlaceholder("Search for more").fill(email);
    await this.page.getByRole("listbox").getByText(email, { exact: true }).first().click();
    await this.dismissPopup();
  }

  private async pickOption(trigger: Locator, title: string): Promise<void> {
    await trigger.click();
    await this.page
      .getByRole("listbox")
      .getByRole("option")
      .filter({ has: this.page.getByText(title, { exact: true }) })
      .first()
      .click();
  }

  async pickRole(title: string): Promise<void> {
    // The field title and the trigger placeholder both read "Assign role".
    await this.pickOption(this.sheet.getByText("Assign role", { exact: true }).last(), title);
  }

  async pickEnvironment(title: string): Promise<void> {
    await this.pickOption(this.environmentPicker, title);
    await this.dismissPopup();
  }

  // Opens the member's edit drawer, which lists every binding with its
  // direct-execution callout.
  async openMember(email: string): Promise<void> {
    const row = this.page.getByRole("row").filter({ hasText: email });
    await row.getByRole("button", { name: "Edit" }).first().click();
    await this.sheet.waitFor();
  }

  async closeSheet(): Promise<void> {
    await this.page.keyboard.press("Escape");
    await this.sheet.waitFor({ state: "hidden" });
  }
}
