import { expect, type Page, type Locator } from "@playwright/test";

export class PlanDetailPage {
  readonly page: Page;
  readonly baseURL: string;

  readonly changesSection: Locator;
  readonly deploySection: Locator;
  readonly manualCreateRolloutButton: Locator;
  readonly taskRerunButton: Locator;
  // The plan/issue title input rendered in PlanDetailHeader. It is an
  // <input> bound to the plan or issue title — first textbox on the page.
  readonly headerTitle: Locator;

  // --- Review section (AIO Plan Detail Review) ---
  // The header Review action button (PlanReviewSectionHeader). Distinct from the
  // "Review" phase label, which is a <span>, not a button.
  readonly reviewButton: Locator;
  // The markdown editor inside the Review popover (ReviewActionPopover).
  readonly reviewPopoverEditor: Locator;
  readonly reviewSubmitButton: Locator;
  // The rejection banner pinned above the timeline (ReviewRejectionBanner).
  readonly rejectionBanner: Locator;
  readonly reRequestButton: Locator;
  // The collapsed composer trigger + its expanded editor (ReviewCommentComposer).
  readonly composerTrigger: Locator;
  readonly composerEditor: Locator;
  readonly composerSubmitButton: Locator;
  // The readiness footer's single action, in either weight (link or button).
  readonly bypassAndDeployAction: Locator;

  // --- Inline comment threads (StatementThreadsLayer / CommentThreadCard) ---
  // The read-only statement editor of the selected change.
  readonly statementEditor: Locator;
  // Thread markers in the statement editor's glyph margin.
  readonly threadMarkers: Locator;
  // The hover affordance that starts a thread on the hovered line.
  readonly addThreadGlyph: Locator;
  // The inline composer that opens below the selected lines.
  readonly inlineComposer: Locator;
  readonly inlineComposerEditor: Locator;
  readonly inlineComposerPublishButton: Locator;
  // The unresolved-thread walker pinned to the editor's top-right corner.
  readonly threadWalker: Locator;
  readonly threadWalkerNext: Locator;
  readonly threadWalkerPrevious: Locator;
  // The walker's polite live region ("Thread 2 of 3").
  readonly threadWalkerAnnouncement: Locator;
  // Per-change unresolved counts on the Changes tab strip.
  readonly specUnresolvedCounts: Locator;
  // The Review phase block, whose summary line carries the plan-wide count.
  readonly reviewPhase: Locator;

  // --- Header lifecycle slot (PlanDetailHeader, BYT-9722) ---
  // The sticky title/action row. Scope every header-slot assertion to this so a
  // pill/stamp/label in a phase section can't be mistaken for the header slot.
  readonly headerRow: Locator;
  // The frontier-stage advance buttons in the header slot: "Run · <stage>" /
  // "Rerun · <stage>" (distinct from the deploy-section per-task exact "Run").
  readonly headerRunStage: Locator;
  readonly headerRerunStage: Locator;
  readonly headerCreateButton: Locator;
  readonly headerReadyForReviewButton: Locator;
  // The your-turn Review action in the header slot (opens the approve/reject
  // composer). Scoped so it never resolves to a phase-section control.
  readonly headerReviewButton: Locator;
  // The "⋯" overflow trigger and the promoted secondary action (e.g. Reopen).
  readonly headerOverflowButton: Locator;
  readonly headerReopenButton: Locator;

  constructor(page: Page, baseURL: string) {
    this.page = page;
    this.baseURL = baseURL;

    this.changesSection = page.getByText("Changes").first();
    this.deploySection = page.getByText("Deploy").first();
    this.manualCreateRolloutButton = page.getByRole("button", { name: "Manually create rollout" });
    this.taskRerunButton = page.getByRole("button", {
      name: "Rerun",
      exact: true,
    });
    this.headerTitle = page.getByRole("textbox").first();

    this.reviewButton = page.getByRole("button", { name: "Review", exact: true });
    this.reviewPopoverEditor = page.locator(
      "textarea[placeholder='Leave a comment...']",
    );
    this.reviewSubmitButton = page.getByRole("button", { name: "Submit", exact: true });
    this.rejectionBanner = page.getByText(/^Rejected by /).first();
    this.reRequestButton = page.getByRole("button", { name: "re-request review" });
    this.composerTrigger = page.getByRole("button", { name: "Add a comment..." });
    this.composerEditor = page.locator("textarea[placeholder='Add a comment...']");
    this.composerSubmitButton = page.getByRole("button", { name: "Comment", exact: true });
    this.bypassAndDeployAction = page.getByRole("button", { name: "Bypass and deploy" });

    this.statementEditor = page.locator("#plan-phase-changes .monaco-editor").first();
    this.threadMarkers = this.statementEditor.locator(".bb-thread-glyph");
    this.addThreadGlyph = this.statementEditor.locator(".bb-thread-add-glyph");
    this.inlineComposer = page.getByTestId("inline-thread-composer");
    this.inlineComposerEditor = this.inlineComposer.locator(
      "textarea[placeholder='Write a comment...']",
    );
    this.inlineComposerPublishButton = this.inlineComposer.getByRole("button", {
      name: "Publish",
      exact: true,
    });
    this.threadWalker = this.statementEditor.getByTestId("thread-walker");
    this.threadWalkerNext = this.threadWalker.getByRole("button", {
      name: "Next unresolved thread",
    });
    this.threadWalkerPrevious = this.threadWalker.getByRole("button", {
      name: "Previous unresolved thread",
    });
    this.threadWalkerAnnouncement = this.threadWalker.locator("[aria-live]");
    this.specUnresolvedCounts = page
      .locator("#plan-phase-changes")
      .getByTestId("spec-unresolved-threads");
    this.reviewPhase = page.locator("#plan-phase-review");

    // The sticky header row, located structurally (no product-code testid): the
    // title <input> sits in the row's left group (beside the terminal stamp), so
    // its grandparent is the row that also holds the lifecycle slot + ⋯ overflow.
    // Uses the same `..`-parent pattern as getSectionToggle below.
    this.headerRow = page.getByRole("textbox").first().locator("..").locator("..");
    // The frontier advance button reads "Run <stage>" / "Rerun <stage>" (the
    // "·" separator is aria-hidden, so it's not in the accessible name). "Rerun"
    // starts with "Re", so /^Run/ and /^Rerun/ never cross-match.
    this.headerRunStage = this.headerRow.getByRole("button", { name: /^Run/ });
    this.headerRerunStage = this.headerRow.getByRole("button", { name: /^Rerun/ });
    this.headerCreateButton = this.headerRow.getByRole("button", {
      name: "Create",
      exact: true,
    });
    this.headerReadyForReviewButton = this.headerRow.getByRole("button", {
      name: "Ready for Review",
      exact: true,
    });
    this.headerReviewButton = this.headerRow.getByRole("button", {
      name: "Review",
      exact: true,
    });
    this.headerOverflowButton = this.headerRow.getByRole("button", { name: "More" });
    this.headerReopenButton = this.headerRow.getByRole("button", {
      name: "Reopen",
      exact: true,
    });
  }

  // The read-only status pill in the header slot ("Under review", "Rejected",
  // "N checks failing", "Checking…"). It is a <button> (opens the gate popover),
  // which distinguishes it from the identical-text review-phase status <span>.
  headerStatusPill(name: string | RegExp): Locator {
    return this.headerRow.getByRole("button", { name });
  }

  // A terminal stamp in the header slot ("Deployed" / "Closed"), rendered as a
  // non-interactive badge left of the title.
  headerStamp(text: string): Locator {
    return this.headerRow.getByText(text, { exact: true });
  }

  // Open the "⋯" overflow menu and return a menu item by name. The menu portals
  // outside the header row, so the item is located at page scope.
  async openOverflow(): Promise<void> {
    await this.headerOverflowButton.click();
    await this.page.getByRole("menu").waitFor({ state: "visible" });
  }

  overflowItem(name: string): Locator {
    return this.page.getByRole("menuitem", { name, exact: true });
  }

  // Confirm a "Run task" run-confirmation dialog (shared by the header Run·stage
  // slot and the deploy-section Run — both open PlanDetailTaskRolloutActionPanel).
  async confirmRunTaskDialog(): Promise<void> {
    const dialog = this.page.getByRole("dialog").filter({ hasText: "Run task" });
    await dialog.getByRole("button", { name: "Run", exact: true }).click();
  }

  // The Review phase status badge text (e.g. "Under review", "Approved",
  // "Rejected", "Skipped"). Scoped to the review phase section (#plan-phase-review)
  // because the header lifecycle slot (BYT-9722, #20720) now renders the SAME
  // status text as a pill — an unscoped getByText would match both and throw a
  // strict-mode violation (the header pill is asserted separately, scoped to the
  // header row, in plan-detail-lifecycle.spec.ts).
  reviewBadge(text: string): Locator {
    return this.page
      .locator("#plan-phase-review")
      .getByText(text, { exact: true });
  }

  // Open the Review popover, pick an action, optionally type a comment, submit.
  async submitReview(
    action: "Comment" | "Approve" | "Reject",
    comment?: string,
  ): Promise<void> {
    await this.reviewButton.click();
    await this.page.getByText("Submit feedback and request changes").waitFor();
    await this.page.locator(`label:has-text("${action}")`).click();
    if (comment !== undefined) {
      await this.reviewPopoverEditor.fill(comment);
    }
    await this.reviewSubmitButton.click();
  }

  async goto(projectId: string, planId: string) {
    await this.page.goto(`${this.baseURL}/projects/${projectId}/plans/${planId}`);
    await this.page.waitForLoadState("networkidle");
  }

  async gotoCreate(
    projectId: string,
    query: Record<string, string> = {},
  ): Promise<void> {
    const search = new URLSearchParams(query).toString();
    await this.page.goto(
      `${this.baseURL}/projects/${projectId}/plans/create${search ? `?${search}` : ""}`,
    );
    await this.page.waitForLoadState("networkidle");
  }

  lifecycleAlert(heading: string): Locator {
    return this.page.getByRole("alert").filter({ hasText: heading });
  }

  async fillPlanStatement(sql: string): Promise<void> {
    const editor = this.page.getByRole("code").first();
    await expect(editor).toBeVisible();
    await editor.click();
    await this.page.keyboard.press("ControlOrMeta+a");
    await this.page.keyboard.press("Delete");
    await this.page.keyboard.insertText(sql);
    await expect(editor.locator(".view-lines")).toContainText(sql);
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

  // Change tabs expose the complete target-derived identity as
  // "Change N: <target>" even when the visual number is omitted because the
  // target is already unique. Match that stable accessible prefix rather than
  // the removed generic "N. Database Change" text.
  //
  // Caller must expandSection("Changes") first if a rollout exists,
  // since the section auto-collapses in that state and the tab won't be
  // in the visible DOM.
  specTab(n: number): Locator {
    return this.page
      .getByRole("button", { name: new RegExp(`^Change ${n}:`) })
      .first();
  }

  // The plan-wide "Checks" summary button in the CHANGES section. Shows
  // Success/Warning/Error entries (each rendered only when its count > 0)
  // and opens the results drawer on click. Distinct from "Run checks".
  checksSummary(): Locator {
    return this.page.getByRole("button", { name: "Checks", exact: true });
  }

  // A thread card by its root comment text, in either surface. Scope with
  // `threadCardIn` when the same thread shows in the editor and the timeline.
  threadCard(rootText: string): Locator {
    return this.page.getByTestId("comment-thread").filter({ hasText: rootText });
  }

  threadCardIn(phase: "changes" | "review", rootText: string): Locator {
    return this.page
      .locator(`#plan-phase-${phase}`)
      .getByTestId("comment-thread")
      .filter({ hasText: rootText });
  }

  // The inline composer anchored to one line, by its heading.
  inlineComposerOn(lineNumber: number): Locator {
    return this.inlineComposer.filter({
      hasText: `Add a comment on line ${lineNumber}`,
    });
  }

  // Start a thread from the gutter of one line: hover it, click the add tile.
  async openInlineComposerOn(lineNumber: number): Promise<Locator> {
    await this.hoverStatementLine(lineNumber);
    await this.clickAddThreadGlyph();
    const composer = this.inlineComposerOn(lineNumber);
    await expect(composer).toBeVisible();
    return composer;
  }

  // The reply composer's thread-state checkbox inside a thread card. It names
  // the action for the current state: "Resolve thread" on an open thread,
  // "Reopen thread" (checked by default) on a resolved one.
  threadStateCheckbox(card: Locator, label: "Resolve thread" | "Reopen thread"): Locator {
    return card.getByRole("checkbox", { name: label });
  }

  // Collapse or expand a phase block by its anchor id, which is unambiguous
  // where the section's label text is not (the header has a "Review" action).
  async setPhaseExpanded(phase: "changes" | "review" | "deploy", expanded: boolean): Promise<void> {
    const block = this.page.locator(`#plan-phase-${phase}`);
    const toggle = block.getByText(expanded ? "Show details" : "Hide details", { exact: true });
    if (!(await toggle.isVisible({ timeout: 1000 }).catch(() => false))) return;
    await toggle.click();
    await expect(
      block.getByText(expanded ? "Hide details" : "Show details", { exact: true }),
    ).toBeVisible({ timeout: 5_000 });
  }

  // Click the add-thread tile by its coordinates. Monaco re-creates the tile
  // as the pointer moves and it animates in, so Playwright's actionability
  // retries never settle on it; a plain pointer click does.
  async clickAddThreadGlyph(): Promise<void> {
    await expect(this.addThreadGlyph).toBeVisible({ timeout: 5_000 });
    const box = await this.addThreadGlyph.boundingBox();
    if (!box) throw new Error("add-thread tile has no box");
    await this.page.mouse.click(box.x + box.width / 2, box.y + box.height / 2);
  }

  // A line's number cell in the statement editor's gutter.
  statementLineNumber(lineNumber: number): Locator {
    return this.statementEditor
      .locator(".margin-view-overlays .line-numbers", {
        hasText: new RegExp(`^${lineNumber}$`),
      })
      .first();
  }

  // A collapsed thread row in the editor's open line stack, by root name.
  collapsedThreadRow(rootName: string): Locator {
    return this.page.locator(
      `#plan-phase-changes [data-testid='thread-row'][data-thread-name='${rootName}']`,
    );
  }

  // Scroll the editor clear of the sticky page header, which otherwise
  // intercepts pointer actions on its top lines. The dashboard body, not the
  // window, owns the page scroll, so walk up to the nearest scrolling
  // ancestor.
  private async scrollEditorClearOfHeader(): Promise<void> {
    await this.statementEditor.evaluate((node) => {
      const top = node.getBoundingClientRect().top;
      if (top >= 96) return;
      let owner: HTMLElement | null = node.parentElement;
      while (owner) {
        const overflowY = getComputedStyle(owner).overflowY;
        if ((overflowY === "auto" || overflowY === "scroll") && owner.scrollHeight > owner.clientHeight) break;
        owner = owner.parentElement;
      }
      (owner ?? window).scrollBy({ top: top - 96 });
    });
  }

  // Hover a line's number in the statement editor so the add-thread glyph
  // appears on that line.
  async hoverStatementLine(lineNumber: number): Promise<void> {
    await this.scrollEditorClearOfHeader();
    await this.statementLineNumber(lineNumber).hover();
  }

  // Drag across line numbers to select a range; the add-thread glyph then
  // sits on the last selected line.
  async selectStatementLines(from: number, to: number): Promise<void> {
    await this.scrollEditorClearOfHeader();
    const start = await this.statementLineNumber(from).boundingBox();
    const end = await this.statementLineNumber(to).boundingBox();
    if (!start || !end) throw new Error(`line numbers ${from}-${to} not rendered`);
    const mouse = this.page.mouse;
    await mouse.move(start.x + start.width / 2, start.y + start.height / 2);
    await mouse.down();
    await mouse.move(end.x + end.width / 2, end.y + end.height / 2, { steps: 4 });
    await mouse.up();
  }
}
