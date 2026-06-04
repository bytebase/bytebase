import { type Page, type Locator } from "@playwright/test";

// Page object for the React-migrated SQL Editor.
// Replaces the helpers that previously lived alongside the Vue components.
export class SqlEditorPage {
  readonly page: Page;
  private baseURL: string;

  readonly codeEditor: Locator;
  readonly runButton: Locator;
  readonly saveButton: Locator;
  readonly adminModeButton: Locator;
  readonly gutterWorksheetTab: Locator;
  readonly gutterSchemaTab: Locator;
  readonly gutterHistoryTab: Locator;
  readonly gutterAccessTab: Locator;

  constructor(page: Page, baseURL = "") {
    this.page = page;
    this.baseURL = baseURL;
    // Monaco-editor exposes role="code" on the editor surface.
    // Clicking this is the only reliable way to give the editor focus —
    // .focus() on the ime-text-area is insufficient (Monaco's focus chain
    // does not fire) and the role="textbox" wrapper is not the input.
    this.codeEditor = page.getByRole("code");
    // The Run button is rendered as an icon-only <button> with a
    // lucide-play SVG. The visible label is the row-limit dropdown
    // ("(limit 1000)"), not "Run" — so we locate by play icon.
    this.runButton = page.locator("button").filter({ has: page.locator("svg.lucide-play") }).first();
    this.saveButton = page.getByRole("button", { name: "Save", exact: true });
    // AdminModeButton has a lucide-wrench icon and renders only when
    // editorStore.allowAdmin && currentTab.mode === "WORKSHEET".
    //
    // Scoping note: the workspace-level "Bytebase has not configured
    // --external-url" banner (frontend/src/react/components/BannersWrapper.tsx)
    // ALSO renders a button with a lucide-wrench icon ("Configure now").
    // The framework's seed-test-data fixture silences that banner by
    // setting an external URL during setup, but we still anchor by the
    // editor's own warning-color class (`border-warning`, applied in
    // frontend/src/react/components/sql-editor/AdminModeButton.tsx:53)
    // so the locator stays correct even if a future build re-introduces
    // the banner.
    this.adminModeButton = page
      .locator("button.border-warning")
      .filter({ has: page.locator("svg.lucide-wrench") })
      .first();
    this.gutterWorksheetTab = page.getByRole("button", { name: "Worksheet", exact: true });
    this.gutterSchemaTab = page.getByRole("button", { name: "Schema", exact: true });
    this.gutterHistoryTab = page.getByRole("button", { name: "History", exact: true });
    // ACCESS gutter tab only renders when project.allowJustInTimeAccess=true.
    // The button's screen-reader name comes from i18n key
    // `sql-editor.jit` → "Just-In-Time Access" (not the shorter "Access"
    // — that's the label of an entirely different "Data Access" sidebar
    // entry in the workspace nav).
    this.gutterAccessTab = page.getByRole("button", {
      name: "Just-In-Time Access",
      exact: true,
    });
  }

  async gotoHome() {
    await this.page.goto(`${this.baseURL}/sql-editor`);
    await this.page.keyboard.press("Escape").catch(() => {});
    await this.page.waitForLoadState("networkidle").catch(() => {});
  }

  async gotoWithDb(projectId: string, instanceId: string, dbName: string) {
    // Direct DB URL avoids stale editor state from prior navigations.
    await this.page.goto(
      `${this.baseURL}/sql-editor/projects/${projectId}/instances/${instanceId}/databases/${dbName}`
    );
    await this.page.keyboard.press("Escape").catch(() => {});
    await this.page.waitForTimeout(2000);
  }

  async gotoSheet(projectId: string, sheetUuid: string) {
    await this.page.goto(
      `${this.baseURL}/sql-editor/projects/${projectId}/sheets/${sheetUuid}`
    );
    await this.page.keyboard.press("Escape").catch(() => {});
    await this.page.waitForTimeout(1000);
  }

  // Type a query into Monaco. Each keystroke costs a network turn on
  // CDP; keep queries short. Uses delay: 10 to avoid dropped keys.
  async typeQuery(sql: string) {
    await this.codeEditor.click();
    await this.page.waitForTimeout(150);
    await this.page.keyboard.press("ControlOrMeta+a");
    await this.page.waitForTimeout(50);
    await this.page.keyboard.press("Delete");
    await this.page.waitForTimeout(50);
    await this.page.keyboard.type(sql, { delay: 10 });
    await this.page.waitForTimeout(200);
  }

  async runQuery(sql: string) {
    await this.typeQuery(sql);
    await this.runButton.click();
    // Result is either an N-rows footer (success) or an inline ERROR card.
    await Promise.race([
      this.page.getByText(/\d+ rows?/i).first().waitFor({ timeout: 15000 }),
      this.page.getByText(/ERROR[: ]/).first().waitFor({ timeout: 15000 }),
    ]).catch(() => {});
    await this.page.waitForTimeout(300);
  }

  // Replace the active Monaco editor's content reliably. `keyboard.type`
  // with delay<30ms/key occasionally drops the first character (e.g.,
  // turning "SELECT" into "SEECT") and the resulting syntax error masks
  // any downstream assertion. `insertText` paints the whole string into
  // the focused control in a single CDP call AND fires the synthetic
  // input event, so Monaco's dirty-state tracking and Run-button enable
  // logic both update — unlike a raw `monaco.editor.getEditors()[i].setValue`
  // call which the editor treats as programmatic and ignores.
  async setEditorContent(
    sql: string,
    opts: { which?: "first" | "last" } = {},
  ): Promise<void> {
    // `which: "last"` targets the admin terminal's editable prompt (the
    // last role="code"); older terminal rows are read-only.
    const editor =
      opts.which === "last" ? this.codeEditor.last() : this.codeEditor;
    await editor.click();
    await this.page.waitForTimeout(150);
    await this.page.keyboard.press("ControlOrMeta+a");
    await this.page.waitForTimeout(50);
    await this.page.keyboard.press("Delete");
    await this.page.waitForTimeout(50);
    await this.page.keyboard.insertText(sql);
    await this.page.waitForTimeout(200);
  }

  // Read a Monaco editor's content by walking the DOM. `window.monaco` is
  // NOT exposed on the production React bundle, so any read MUST go through
  // the rendered `.view-lines` (a `monaco.editor.getEditors()` read returns
  // undefined and silently yields ""). `which`:
  //   - "first"   the first role="code" surface (the worksheet editor)
  //   - "last"    the last surface (the admin terminal's editable prompt)
  //   - "longest" the surface with the most content (ignores empty panes)
  async readEditorContent(
    opts: { which?: "first" | "last" | "longest" } = {},
  ): Promise<string> {
    const which = opts.which ?? "first";
    return await this.page.evaluate((which) => {
      const codes = Array.from(document.querySelectorAll('[role="code"]'));
      if (codes.length === 0) return "";
      const textOf = (code: Element): string =>
        Array.from(code.querySelectorAll(".view-lines .view-line"))
          .map((l) => (l.textContent ?? "").replace(/ /g, " "))
          .join("\n");
      const texts = codes.map(textOf);
      let value: string;
      if (which === "last") value = texts[texts.length - 1] ?? "";
      else if (which === "longest")
        value = texts.reduce((best, v) => (v.length >= best.length ? v : best), "");
      else value = texts[0] ?? "";
      return value.trim();
    }, which);
  }

  async runPreparedQuery(sql: string): Promise<void> {
    await this.setEditorContent(sql);
    await this.runButton.click();
    await Promise.race([
      this.page.getByText(/\d+ rows?/i).first().waitFor({ timeout: 15000 }),
      this.page.getByText(/ERROR[: ]/).first().waitFor({ timeout: 15000 }),
    ]).catch(() => {});
    await this.page.waitForTimeout(300);
  }

  // Returns number of [data-tab-id] elements currently in the DOM.
  async tabCount(): Promise<number> {
    return await this.page.locator("[data-tab-id]").count();
  }

  // Returns the active (current) tab element, or null if none.
  activeTab(): Locator {
    return this.page.locator("[data-tab-id].current").first();
  }

  // Tab status is encoded as a class on [data-tab-id]: "status-dirty"
  // when unsaved, "status-clean" otherwise (per TabList state model).
  async activeTabStatus(): Promise<"DIRTY" | "CLEAN" | "UNKNOWN"> {
    const cls = (await this.activeTab().getAttribute("class")) ?? "";
    if (cls.includes("status-dirty")) return "DIRTY";
    if (cls.includes("status-clean")) return "CLEAN";
    return "UNKNOWN";
  }

}
