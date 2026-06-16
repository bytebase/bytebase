// Issue detail — Targets "View all" sheet (scrollability).
//
// BYT-9558 (FIXED, #20427): on a Data Change Issue with many targets, clicking
// "View all (N)" opened a popup that was frozen — the user could not scroll down
// to see all the targets. Root cause: the targets list rendered in a Dialog whose
// nested flex/overflow chain never produced a scrollable container, so content
// below the fold was unreachable. The fix replaced the Dialog with a Sheet whose
// SheetBody is `overflow-hidden` and whose inner list is
// `min-h-0 flex-1 overflow-y-auto` (IssueDetailDatabaseChangeView.tsx) — a real
// scroll container.
//
// The same broken pattern + fix also lived on the Plan detail Changes/Targets
// sheet (PlanDetailChangesBranch.tsx); that sibling surface is a candidate for a
// follow-up lock (CUJ analysis) but is not covered here.
//
// Owns its fixtures: creates >20 databases via psql, syncs + transfers them into
// the project, targets them in a single-spec plan/issue, and drops them in
// afterAll.

import { test, expect, type BrowserContext, type Page } from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import type { BytebaseApiClient } from "../framework/api-client";
import { execSql, getInstancePgPort } from "../framework/psql";

test.describe.configure({ mode: "serial" });

// 21 > DEFAULT_VISIBLE_TARGETS (20), so the "View all (21)" button renders.
const TARGET_COUNT = 21;
const STAMP = Date.now();
// Unique per run so a CI retry against the same disposable server doesn't collide.
const DB_PREFIX = `e2e_tgt_${STAMP}_`;
const dbShortNames = Array.from(
  { length: TARGET_COUNT },
  (_, i) => `${DB_PREFIX}${String(i + 1).padStart(2, "0")}`,
);
// The last target — must require scrolling to reach inside the sheet.
const LAST_DB = dbShortNames[dbShortNames.length - 1];

let env: TestEnv & { api: BytebaseApiClient };
let pgPort = "";
let sharedContext: BrowserContext;
let page: Page;
let issueUrl = "";

test.beforeAll(async ({ browser }) => {
  test.setTimeout(240_000);
  env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  pgPort = await getInstancePgPort(env);

  // 1. Create the databases via psql (CREATE DATABASE can't run in the same db
  //    being changed, but a single psql -c against the env database is fine).
  for (const name of dbShortNames) {
    execSql(env.databaseId, pgPort, `CREATE DATABASE ${name}`);
  }

  // 2. Sync the instance so Bytebase discovers them, then poll until present.
  await env.api.syncInstance(env.instance);
  await expect
    .poll(
      async () => {
        const { databases } = await env.api.listDatabases(env.instance);
        const names = new Set((databases ?? []).map((d) => d.name));
        return dbShortNames.every((short) =>
          names.has(`${env.instance}/databases/${short}`),
        );
      },
      { timeout: 60_000, message: "synced databases should appear" },
    )
    .toBe(true);

  // 3. Transfer each into the project (backend validateSpecs rejects targets not
  //    in the plan's project).
  const targetFullNames: string[] = [];
  for (const short of dbShortNames) {
    const full = `${env.instance}/databases/${short}`;
    await env.api.transferDatabaseToProject(full, env.project);
    targetFullNames.push(full);
  }

  // 4. One spec targeting all 21 new databases → "View all (21)".
  const sheet = await env.api.createSheet(env.project, "SELECT 1; -- byt9558");
  const plan = await env.api.createPlan(env.project, `BYT-9558 ${STAMP}`, [
    { id: `spec-${STAMP}`, targets: targetFullNames, sheet },
  ]);
  const issue = await env.api.createIssue(env.project, `BYT-9558 ${STAMP}`, plan.name);

  const projectId = env.project.split("/").pop()!;
  const issueId = issue.name.split("/").pop()!;
  issueUrl = `${env.baseURL}/projects/${projectId}/issues/${issueId}`;

  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
});

test.afterAll(async () => {
  await sharedContext?.close();
  // Best-effort: drop the databases and re-sync so Bytebase forgets them.
  for (const name of dbShortNames) {
    try {
      execSql(env.databaseId, pgPort, `DROP DATABASE IF EXISTS ${name}`);
    } catch {
      /* best-effort — a lingering connection can block DROP on a disposable server */
    }
  }
  await env.api.syncInstance(env.instance).catch(() => {});
});

test.describe("Targets 'View all' sheet scrolls to the last target (BYT-9558)", () => {
  test("the all-targets sheet is scrollable — the last target can be brought into view", async () => {
    test.setTimeout(120_000);

    await page.goto(issueUrl);
    await page.keyboard.press("Escape").catch(() => {});
    await page.waitForLoadState("networkidle").catch(() => {});

    // The Targets section shows "View all (21)".
    const viewAll = page.getByRole("button", {
      name: `View all (${TARGET_COUNT})`,
    });
    await expect(viewAll).toBeVisible({ timeout: 15_000 });
    await viewAll.click();

    // The Sheet opens with title "Targets (21)".
    const sheet = page
      .getByRole("dialog")
      .filter({ hasText: `Targets (${TARGET_COUNT})` });
    await expect(sheet).toBeVisible({ timeout: 5000 });

    // The scroll container (min-h-0 flex-1 overflow-y-auto) must actually
    // overflow — sanity that we have enough rows to exercise the bug.
    const scrollRegion = sheet.locator("div.overflow-y-auto").first();
    await expect(scrollRegion).toBeVisible({ timeout: 5000 });
    const overflow = await scrollRegion.evaluate(
      (el) => el.scrollHeight - el.clientHeight,
    );
    expect(
      overflow,
      "the targets list must overflow its container (otherwise scroll can't be tested)",
    ).toBeGreaterThan(10);

    // THE REGRESSION ORACLE: the last target row can be scrolled into the
    // viewport. Pre-fix the Dialog had no functioning scroll container, so rows
    // below the fold were unreachable → scrollIntoViewIfNeeded can't surface it
    // and toBeInViewport fails.
    const lastRow = sheet.getByText(LAST_DB, { exact: false }).first();
    await expect(
      lastRow,
      "the last target must be present in the (initially scrolled-off) list",
    ).toHaveCount(1);
    await lastRow.scrollIntoViewIfNeeded();
    await expect(
      lastRow,
      "the last target must be reachable by scrolling the sheet (pre-fix the " +
        "popup was frozen and the bottom targets were unreachable)",
    ).toBeInViewport({ timeout: 5000 });

    // Anti-freeze: the in-sheet search narrows the list. Each target renders as
    // a `rounded-lg border` row inside the scroll region; at baseline there are
    // all 21 rows.
    const targetRows = scrollRegion.locator("div.rounded-lg.border");
    await expect(targetRows).toHaveCount(TARGET_COUNT, { timeout: 5000 });

    const search = sheet.getByPlaceholder("Search").first();
    await expect(search).toBeVisible({ timeout: 5000 });
    await search.click();
    await search.pressSequentially(LAST_DB, { delay: 15 });
    // Filtering down to the single matching target proves the list is live
    // (not a frozen popup).
    await expect(targetRows).toHaveCount(1, { timeout: 5000 });
    await expect(scrollRegion.getByText(LAST_DB, { exact: false })).toBeVisible();

    // Escape closes the sheet.
    await page.keyboard.press("Escape");
    await expect(sheet).toHaveCount(0, { timeout: 5000 });
  });
});
