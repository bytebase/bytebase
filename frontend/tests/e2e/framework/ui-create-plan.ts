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

// Submit the current draft plan for review (draft issue → open/non-draft), the
// UI's "Ready for Review" flow. Call while on the created plan's detail page.
//
// The confirm dialog runs the plan checks and has TWO terminal states, both
// handled here so the helper is correct whether or not the plan has failing
// checks (review / rollout / plan-check callers deliberately attach ERROR-level
// rules):
//   - checks pass  → the Confirm button enables on its own.
//   - checks fail  → the product gates submission behind an explicit
//                    "Confirm anyway" acknowledgment checkbox; Confirm stays
//                    disabled until it is checked.
export async function submitDraftForReviewViaUI(page: Page): Promise<void> {
  await page.getByRole("button", { name: "Ready for Review" }).click();
  const dialog = page.getByRole("dialog");
  const confirm = dialog
    .getByRole("button", { name: /Ready for Review|Submit|Confirm/ })
    .first();
  const confirmAnyway = dialog
    .getByRole("checkbox", { name: "Confirm anyway" })
    .first();

  // Wait for the checks to settle into whichever terminal state applies: the
  // ack checkbox appears (failing) or Confirm becomes enabled (passing).
  await expect(async () => {
    const gated = await confirmAnyway.isVisible().catch(() => false);
    const enabled = await confirm.isEnabled().catch(() => false);
    expect(gated || enabled).toBe(true);
  }).toPass({ timeout: 45_000 });

  if (await confirmAnyway.isVisible().catch(() => false)) {
    await confirmAnyway.click();
  }
  await expect(confirm).toBeEnabled({ timeout: 10_000 });
  await confirm.click();
  // After submission the header advances out of the ready-for-review state.
  await expect(
    page.getByRole("button", { name: "Ready for Review" }),
  ).toHaveCount(0, { timeout: 15_000 });
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
