import { expect, type Page } from "@playwright/test";

// UI-driven plan creation — drives the real create page so setup follows the
// SAME workflow a user does (plan + draft review issue created together). This
// stays correct across workflow changes: whatever the create page requires in
// future, this clicks through it, and drift surfaces as a loud selector failure
// rather than a silent wrong-state (which is how bare `api.createPlan` setups
// silently broke when issue-creation was coupled to plan creation).

export interface UiCreatePlanOptions {
  baseURL: string;
  projectId: string;
  /**
   * Full database resource name (instances/x/databases/y). Pass several to
   * build ONE spec targeting multiple databases (they share the single typed
   * statement) — e.g. a two-environment change that the rollout groups into two
   * stages. For N INDEPENDENT specs (distinct per-spec SQL) use
   * createMultiSpecChangePlanViaUI instead.
   */
  database: string | string[];
  title: string;
  /** Statement to enter; any non-empty SQL enables the Create button. */
  sql: string;
}

// Create a database-change plan through the UI create page. Returns the numeric
// plan id. On return the page sits on the created plan's detail (a draft review
// issue exists, so its header reads "Ready for Review").
export async function createDatabaseChangePlanViaUI(
  page: Page,
  opts: UiCreatePlanOptions,
): Promise<{ planId: string }> {
  const databaseList = Array.isArray(opts.database)
    ? opts.database.join(",")
    : opts.database;
  const q = new URLSearchParams({
    template: "bb.plan.change-database",
    databaseList,
    name: opts.title,
  });
  await page.goto(
    `${opts.baseURL}/projects/${opts.projectId}/plans/create/specs/placeholder?${q.toString()}`,
  );
  await page.locator(".monaco-editor").first().click();
  await page.keyboard.type(opts.sql);
  const createBtn = page.getByRole("button", { name: "Create", exact: true });
  await expect(createBtn).toBeEnabled({ timeout: 15_000 });
  await createBtn.click();
  await page.waitForURL(/\/plans\/\d+/, { timeout: 20_000 });
  const planId = page.url().match(/\/plans\/(\d+)/)?.[1] ?? "";
  return { planId };
}

export interface UiSpec {
  /** Full database resource name for this spec's single target (env.database). */
  database: string;
  /** Statement for this spec. */
  sql: string;
}

export interface UiMultiSpecOptions {
  baseURL: string;
  projectId: string;
  title: string;
  /**
   * One spec per entry. `createPlanSkeleton` emits one spec per target ONLY when
   * a sqlMap is present (otherwise a single spec carries all targets), so this
   * always passes a sqlMap. sqlMap is keyed by database name — two specs on the
   * SAME database would collapse to one key, so multi-spec callers must use
   * DISTINCT databases (e.g. hr_test + hr_prod). Single-entry works too (one
   * key → one spec).
   */
  specs: UiSpec[];
}

// Create an N-spec database-change plan through the UI create page. The SQL is
// pre-filled from the URL sqlMap (not typed), so no per-spec Monaco interaction
// is needed — the create page hydrates each spec's local sheet from the map.
// Works for one spec (single key) or many (distinct-database keys). Returns the
// numeric plan id; on return the page sits on the created plan's detail (a draft
// review issue exists).
export async function createMultiSpecChangePlanViaUI(
  page: Page,
  opts: UiMultiSpecOptions,
): Promise<{ planId: string }> {
  const sqlMap: Record<string, string> = {};
  for (const s of opts.specs) sqlMap[s.database] = s.sql;
  const q = new URLSearchParams({
    template: "bb.plan.change-database",
    databaseList: opts.specs.map((s) => s.database).join(","),
    name: opts.title,
    sqlMap: JSON.stringify(sqlMap),
  });
  await page.goto(
    `${opts.baseURL}/projects/${opts.projectId}/plans/create/specs/placeholder?${q.toString()}`,
  );
  const createBtn = page.getByRole("button", { name: "Create", exact: true });
  await expect(createBtn).toBeEnabled({ timeout: 15_000 });
  await createBtn.click();
  await page.waitForURL(/\/plans\/\d+/, { timeout: 20_000 });
  const planId = page.url().match(/\/plans\/(\d+)/)?.[1] ?? "";
  return { planId };
}

// Submit the current draft plan for review — the UI's "Ready for Review"
// advance. Call while on the created plan's detail page.
//
// The advance is a tiered press (LifecycleAdvance.tsx / advanceState.ts), not
// a confirm dialog: a clean press advances immediately with no UI; when checks
// failed and the project does not enforce SQL review, a warning popover offers
// "Submit anyway"; a hard blocker (e.g. enforceSqlReview + failed checks)
// shows a read-only list and the press is a no-op. Success is the header state
// machine leaving `ready-for-review` — the button disappears. ONE press, then
// one positive wait: no re-press/Escape loop, which fires stray events into a
// transitioning React tree on the shared page and destabilises later tests.
export async function submitDraftForReviewViaUI(page: Page): Promise<void> {
  const readyButton = page.getByRole("button", { name: "Ready for Review" });
  const submitAnyway = page.getByRole("button", { name: "Submit anyway" });
  await readyButton.click();
  // Tier 2: a "Submit anyway" decision popover appears only when checks failed.
  const decided = await submitAnyway
    .waitFor({ state: "visible", timeout: 3_000 })
    .then(() => true)
    .catch(() => false);
  if (decided) await submitAnyway.click();
  // The advance completes when the header leaves ready-for-review. Plan checks
  // may still be finishing (a "wait" blocker clears on its own), so allow time.
  await expect(readyButton).toHaveCount(0, { timeout: 60_000 });
}

// Create AND submit a database-change plan through the UI — the full "start a
// change and send it for review" workflow. The result is a plan with a
// submitted (non-draft) review issue, which is what makes a rollout auto-create
// under permissive project settings (the state bare api.createPlan +
// api.createIssue used to fabricate directly).
export async function createSubmittedDatabaseChangePlanViaUI(
  page: Page,
  opts: UiCreatePlanOptions,
): Promise<{ planId: string }> {
  const result = await createDatabaseChangePlanViaUI(page, opts);
  await submitDraftForReviewViaUI(page);
  return result;
}

// Multi-spec create + submit — the N-spec analogue of the single-spec submit
// helper. Used by the plan-check tests, which build one- and two-spec plans and
// need a submitted issue so checks run and a rollout auto-creates.
export async function createSubmittedMultiSpecChangePlanViaUI(
  page: Page,
  opts: UiMultiSpecOptions,
): Promise<{ planId: string }> {
  const result = await createMultiSpecChangePlanViaUI(page, opts);
  await submitDraftForReviewViaUI(page);
  return result;
}
