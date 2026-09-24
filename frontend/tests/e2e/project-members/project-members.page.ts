import type { Locator, Page } from "@playwright/test";

// Project members page, its Grant Access sheet, and a member's edit drawer.
// The pickers are the shared Combobox / AccountMultiSelect: a trigger showing
// placeholder text, a popup with a search box and plain option rows.
export class ProjectMembersPage {
  readonly page: Page;
  readonly baseURL: string;
  readonly sheet: Locator;
  readonly grantAccessButton: Locator;
  readonly createButton: Locator;
  readonly directExecutionField: Locator;
  readonly directExecutionSwitch: Locator;
  readonly environmentPicker: Locator;

  constructor(page: Page, baseURL = "") {
    this.page = page;
    this.baseURL = baseURL;
    // Toasts also carry role="dialog" (with a data-type); the sheet does not.
    this.sheet = page.locator("[role='dialog']:not([data-type])");
    this.grantAccessButton = page.getByRole("button", { name: "Grant Access" });
    this.createButton = this.sheet.getByRole("button", { name: "Create", exact: true });
    this.directExecutionField = this.sheet.getByTestId("direct-execution-field");
    this.directExecutionSwitch = this.directExecutionField.getByRole("switch");
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

  async pickAccount(email: string): Promise<void> {
    await this.sheet.getByText("Select accounts", { exact: true }).click();
    await this.page.getByPlaceholder("Search for more").fill(email);
    // The member table behind the sheet shows the same email; the option
    // lives in the picker's listbox.
    await this.page
      .getByRole("listbox")
      .getByText(email, { exact: true })
      .first()
      .click();
    await this.dismissPopup();
  }

  async pickRole(title: string): Promise<void> {
    // The field title and the trigger placeholder both read "Assign role".
    await this.sheet.getByText("Assign role", { exact: true }).last().click();
    await this.page.getByText(title, { exact: true }).last().click();
  }

  async pickEnvironment(title: string): Promise<void> {
    await this.environmentPicker.click();
    await this.page.getByText(title, { exact: true }).last().click();
    await this.dismissPopup();
  }

  // Opens the member's edit drawer, which lists every binding with its
  // direct-execution callout.
  async openMember(email: string): Promise<void> {
    const row = this.page.getByRole("row").filter({ hasText: email });
    await row
      .locator("button")
      .filter({ has: this.page.locator("svg.lucide-pencil") })
      .first()
      .click();
    await this.sheet.waitFor();
  }

  async closeSheet(): Promise<void> {
    await this.page.keyboard.press("Escape");
    await this.sheet.waitFor({ state: "hidden" });
  }
}
