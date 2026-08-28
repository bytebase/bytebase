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
// a confirm dialog. One press resolves to exactly one of:
//   advanced  — nothing unresolved: the header leaves ready-for-review and the
//               button disappears.
//   decision  — checks failed and the project does not enforce SQL review: a
//               popover offers "Submit anyway".
//   blocked   — a read-only blockers popover (role="alert") and NO advance.
//               `wait` rows ("Clears on its own": checks queued/running) close
//               the popover by themselves once the checks finish — but closing
//               never submits, so the press must be repeated. `fix` rows
//               (title, statement, permission) never clear without the user.
// So: press, observe which tier answered, and re-press only after a wait-only
// popover has closed on its own. Never re-press blindly or send Escape — stray
// events into a transitioning React tree on the shared page destabilise later
// tests.
export async function submitDraftForReviewViaUI(page: Page): Promise<void> {
  const readyButton = page.getByRole("button", { name: "Ready for Review" });
  const submitAnyway = page.getByRole("button", { name: "Submit anyway" });
  const blockers = page.getByRole("alert").filter({ hasText: /./ });
  // Blocker rows without the "Clears on its own" tag need the user to act.
  const fixRows = blockers
    .locator(":scope > div")
    .filter({ hasNotText: "Clears on its own" });

  const observe = async () => {
    if ((await readyButton.count()) === 0) return "advanced";
    if (await submitAnyway.isVisible()) return "decision";
    if (await blockers.isVisible()) return "blocked";
    return "pending";
  };

  for (let press = 1; press <= 8; press++) {
    await readyButton.click();
    await expect
      .poll(observe, { timeout: 20_000 })
      .not.toBe("pending");
    const tier = await observe();
    if (tier === "advanced") return;
    if (tier === "decision") {
      await submitAnyway.click();
      await expect(readyButton).toHaveCount(0, { timeout: 60_000 });
      return;
    }
    // Blocked: a fix-kind row cannot self-clear — fail with what the user sees.
    if ((await fixRows.count()) > 0) {
      throw new Error(
        `Ready for Review is blocked by: ${(await fixRows.allInnerTexts()).join(" | ")}`,
      );
    }
    // Wait-only blockers (plan checks still running) close the popover by
    // themselves when the checks finish; then press again.
    await expect(blockers).toBeHidden({ timeout: 120_000 });
  }
  throw new Error("Ready for Review kept re-blocking on running checks");
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
