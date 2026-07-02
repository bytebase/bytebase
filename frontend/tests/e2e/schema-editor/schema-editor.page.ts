import { type Page, type Locator, expect } from "@playwright/test";

/**
 * Page object for the React Schema Editor (`SchemaEditorLite`), reached from a
 * plan's Statement section via the "Schema editor" button.
 *
 * Locator strategy (see the sheet/popover component structure):
 * - The editor is a Base UI Dialog whose accessible name is its title
 *   "Schema editor" — scope everything rendered *inside* the sheet (tree, tabs,
 *   column grid, toolbar, preview, footer) to `this.sheet`.
 * - The New-table popover, the column-type suggestion dropdown, the tree "+"
 *   menu, and the name dialogs all portal to the shared overlay root as
 *   *siblings* of the sheet, so they are located at page level, not under
 *   `this.sheet`.
 * - The tree's create-object "+" button carries aria-label="Create" (an icon
 *   button with no text). That collides by accessible name with the popover's
 *   "Create" button, so the popover Create is matched by its visible text.
 */
export class SchemaEditorPage {
  readonly page: Page;
  readonly baseURL: string;

  // Plan statement section (outside the sheet).
  readonly openButton: Locator;
  readonly planStatementEditor: Locator;

  // Sheet container + footer.
  readonly sheet: Locator;
  readonly insertSqlButton: Locator;
  readonly sheetCancelButton: Locator;
  readonly sheetCloseButton: Locator;
  readonly maximizeButton: Locator;
  readonly restoreButton: Locator;

  // Toolbar (inside the sheet).
  readonly newTableToolbarButton: Locator;
  readonly addColumnButton: Locator;

  // Portaled overlays (page level).
  readonly tableNameInput: Locator;
  readonly popoverCreateButton: Locator;
  readonly popoverSaveButton: Locator;
  readonly popoverCancelButton: Locator;
  readonly duplicateNameError: Locator;

  constructor(page: Page, baseURL: string) {
    this.page = page;
    this.baseURL = baseURL;

    this.openButton = page.getByRole("button", { name: "Schema editor" });
    // The plan's own statement editor is the only Monaco on the page once the
    // sheet is closed.
    this.planStatementEditor = page.locator(".monaco-editor").first();

    this.sheet = page.getByRole("dialog", { name: "Schema editor" });
    this.insertSqlButton = this.sheet.getByRole("button", {
      name: "Insert SQL",
    });
    this.sheetCancelButton = this.sheet.getByRole("button", { name: "Cancel" });
    this.sheetCloseButton = this.sheet.getByRole("button", { name: "Close" });
    this.maximizeButton = this.sheet.getByRole("button", { name: "Maximize" });
    this.restoreButton = this.sheet.getByRole("button", { name: "Restore" });

    this.newTableToolbarButton = this.sheet.getByRole("button", {
      name: "New table",
    });
    this.addColumnButton = this.sheet.getByRole("button", {
      name: "Add column",
    });

    this.tableNameInput = page.getByRole("textbox", { name: "Table name" });
    // Popover Create/Save carry visible text; the tree "+" icon button shares
    // the "Create" accessible name but has no text, so filter by text.
    this.popoverCreateButton = page
      .getByRole("button", { name: "Create", exact: true })
      .filter({ hasText: "Create" });
    this.popoverSaveButton = page
      .getByRole("button", { name: "Save", exact: true })
      .filter({ hasText: "Save" });
    // The sheet footer also has a "Cancel"; scope the popover's Cancel to the
    // Create button's row (Cancel + Create are siblings in the popover).
    this.popoverCancelButton = this.popoverCreateButton
      .locator("..")
      .getByRole("button", { name: "Cancel" });
    this.duplicateNameError = page.getByText("Table name already exists");
  }

  async gotoPlan(projectId: string, planId: string): Promise<void> {
    await this.page.goto(`${this.baseURL}/projects/${projectId}/plans/${planId}`);
    await this.page.waitForLoadState("networkidle");
  }

  async open(): Promise<void> {
    await this.openButton.click();
    await expect(this.sheet).toBeVisible({ timeout: 15_000 });
    // Tree is ready once the schema nodes render.
    await expect(this.treeItem("public")).toBeVisible({ timeout: 15_000 });
  }

  async close(): Promise<void> {
    if (await this.sheet.isVisible().catch(() => false)) {
      await this.sheetCancelButton.click();
      await expect(this.sheet).toBeHidden({ timeout: 10_000 });
    }
  }

  // ---- Tree ----
  treeItem(name: string): Locator {
    return this.sheet.getByRole("treeitem", { name, exact: false });
  }

  async selectSchema(name: string): Promise<void> {
    await this.treeItem(name).click();
  }

  // ---- Tree "+" create menu (Base UI DropdownMenu → role=menuitem) ----
  get createMenuTrigger(): Locator {
    // The tree toolbar "+" button carries aria-label "Create" and has no text,
    // so it's the only "Create"-named button in the sheet when no popover is up.
    return this.sheet.getByRole("button", { name: "Create", exact: true });
  }

  createMenuItem(label: string): Locator {
    return this.page.getByRole("menuitem", { name: label });
  }

  // ---- Schema name dialog (create schema) ----
  get schemaDialog(): Locator {
    return this.page.getByRole("dialog", { name: "Create schema" });
  }

  get schemaNameInput(): Locator {
    return this.page.getByRole("textbox", { name: "Schema name" });
  }

  async createSchema(name: string): Promise<void> {
    await this.createMenuTrigger.click();
    await this.createMenuItem("Create schema").click();
    await expect(this.schemaNameInput).toBeVisible({ timeout: 10_000 });
    await this.schemaNameInput.fill(name);
    await this.schemaDialog
      .getByRole("button", { name: "Create", exact: true })
      .click();
    await expect(this.schemaDialog).toBeHidden({ timeout: 10_000 });
  }

  // ---- Create table (via toolbar) ----
  async openNewTablePopover(): Promise<void> {
    await this.newTableToolbarButton.click();
    await expect(this.tableNameInput).toBeVisible({ timeout: 10_000 });
  }

  async createTable(name: string): Promise<void> {
    await this.openNewTablePopover();
    await this.tableNameInput.fill(name);
    await expect(this.popoverCreateButton).toBeEnabled();
    await this.popoverCreateButton.click();
    await expect(this.tableNameInput).toBeHidden({ timeout: 10_000 });
  }

  // ---- Column grid ----
  columnNameInputs(): Locator {
    return this.sheet.getByRole("textbox", { name: "Column name" });
  }

  columnTypeInputs(): Locator {
    return this.sheet.getByRole("textbox", { name: "column type" });
  }

  // A grid row by its rendered column name (row accessible name starts with it).
  columnRow(nameFragment: string | RegExp): Locator {
    return this.sheet.getByRole("row", { name: nameFragment });
  }

  async addColumn(): Promise<void> {
    const before = await this.columnNameInputs().count();
    await this.addColumnButton.click();
    await expect(this.columnNameInputs()).toHaveCount(before + 1, {
      timeout: 10_000,
    });
  }

  // Open the type suggestion dropdown for the column row at `rowIndex` and pick
  // `type`. The dropdown options portal to the overlay root (page level).
  async selectColumnType(rowIndex: number, type: string): Promise<void> {
    await this.columnTypeInputs().nth(rowIndex).click();
    const option = this.page
      .getByRole("button", { name: type, exact: true })
      .filter({ hasText: type });
    await expect(option.first()).toBeVisible({ timeout: 10_000 });
    await option.first().click();
    await expect(this.columnTypeInputs().nth(rowIndex)).toHaveValue(type, {
      timeout: 10_000,
    });
  }

  typeDropdownOption(type: string): Locator {
    return this.page
      .getByRole("button", { name: type, exact: true })
      .filter({ hasText: type });
  }

  // Not Null / Primary checkboxes for a column row. Cells are positional
  // (name, type, default, comment, not-null, primary, operations) per
  // TableColumnEditor's columnDefs; scope by cell index to avoid ambiguity.
  notNullCheckbox(nameFragment: string | RegExp): Locator {
    return this.columnRow(nameFragment)
      .getByRole("cell")
      .nth(4)
      .getByRole("checkbox")
      .first();
  }

  primaryCheckbox(nameFragment: string | RegExp): Locator {
    return this.columnRow(nameFragment)
      .getByRole("cell")
      .nth(5)
      .getByRole("checkbox")
      .first();
  }

  dropColumnButton(nameFragment: string | RegExp): Locator {
    return this.columnRow(nameFragment)
      .getByRole("cell")
      .last()
      .getByRole("button")
      .first();
  }

  // Drop/restore button for a table row in the DatabaseEditor's TableList.
  // (The trash button stops propagation, so it drops without opening the table.)
  dropTableButton(name: string | RegExp): Locator {
    return this.sheet
      .getByRole("row", { name })
      .getByRole("button")
      .last();
  }

  // Open an existing table by clicking its name cell in the DatabaseEditor's
  // TableList (row onClick → onEditTable). Requires the `public` schema tab.
  // Tab labels are truncated, so confirm via the TableEditor toolbar instead.
  async openTableFromList(name: string): Promise<void> {
    await this.selectSchema("public");
    await this.sheet.getByRole("cell", { name, exact: true }).first().click();
    await expect(this.addColumnButton).toBeVisible({ timeout: 10_000 });
  }

  // ---- Preview ----
  // Reads the Preview pane's readonly Monaco. Content extraction (not a control
  // assertion), so reading the editor's text is acceptable. Line numbers are
  // included but harmless for substring assertions.
  async previewText(): Promise<string> {
    const preview = this.sheet.locator(".monaco-editor .view-lines").last();
    await expect(preview).toBeVisible({ timeout: 10_000 });
    return (await preview.innerText()).replace(/ /g, " ");
  }

  // ---- Insert / statement ----
  async insertSql(): Promise<void> {
    await this.insertSqlButton.click();
    await expect(this.sheet).toBeHidden({ timeout: 15_000 });
  }

  async planStatementText(): Promise<string> {
    const editor = this.page.locator(".monaco-editor .view-lines").first();
    await expect(editor).toBeVisible({ timeout: 10_000 });
    return (await editor.innerText()).replace(/ /g, " ");
  }
}
