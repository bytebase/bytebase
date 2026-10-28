// Issue detail — the route-level redirect to Plan Detail (BYT-9721, PR #20722).
//
// Plan Detail is the canonical review surface for schema/data CHANGE issues, so
// the issue-detail route loader (issueDetailRedirectLoader) redirects those to
// /plans/{planId}. The discriminator is the plan's specs, not the issue type:
// create-database shares the DATABASE_CHANGE proto type but is NOT a change plan
// (shouldStayOnPlanDetailPage === false), so it must STAY on Issue Detail. Both
// directions are locked here.
//
//   B1 — a change issue redirects to Plan Detail, query string preserved.
//   B2 — a create-database issue (same DATABASE_CHANGE type) stays on Issue Detail.
//   B3 — Rollout on a create-database issue stays put: no bounce through Plan
//        Detail, no extra history entry, no plan link in Activity (BYT-10356).

import {
  test,
  expect,
  type Page,
  type BrowserContext,
} from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";
import { execSql, getInstancePgPort } from "../framework/psql";

test.setTimeout(180_000);
test.describe.configure({ mode: "serial" });

let env: TestEnv & { api: BytebaseApiClient };
let projectId: string;
let sharedContext: BrowserContext;
let page: Page;

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  projectId = env.project.split("/").pop()!;
  await env.api.login(env.adminEmail, env.adminPassword);
  // The redirect is decided by the route loader from the issue type + plan
  // specs, independent of approval / plan-check gates and rollout state, so this
  // spec deliberately does NOT touch project settings — leaving the shared
  // sample project's gates unchanged for later specs (the suite runs workers:1).
  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
});

test.afterAll(async () => {
  await sharedContext?.close();
});

test.describe("A schema/data change issue redirects to Plan Detail (B1)", () => {
  let issueId = "";
  let planId = "";

  test.beforeAll(async () => {
    const ts = Date.now();
    const sheet = await env.api.createSheet(env.project, "SELECT 1; -- redirect");
    const plan = await env.api.createPlan(env.project, `E2E Redirect B1 ${ts}`, [
      { id: `spec-${ts}`, targets: [env.database], sheet },
    ]);
    planId = plan.name.split("/").pop()!;
    const issue = await env.api.createIssue(
      env.project,
      `E2E Redirect B1 ${ts}`,
      plan.name,
    );
    issueId = issue.name.split("/").pop()!;
  });

  test("opening the issue URL lands on Plan Detail with the query string preserved", async () => {
    await page.goto(
      `${env.baseURL}/projects/${projectId}/issues/${issueId}?anchor=xyz`,
    );
    // The loader redirects the change issue to its plan, keeping ?anchor=xyz.
    await page.waitForURL(
      new RegExp(`/projects/${projectId}/plans/${planId}(\\b|[?/])`),
      { timeout: 20_000 },
    );
    expect(new URL(page.url()).pathname).toBe(
      `/projects/${projectId}/plans/${planId}`,
    );
    expect(page.url()).toContain("anchor=xyz");
  });
});

test.describe("A create-database issue stays on Issue Detail (B2)", () => {
  let issueId = "";

  test.beforeAll(async () => {
    const ts = Date.now();
    // Same DATABASE_CHANGE issue type, but a createDatabaseConfig spec — so the
    // loader's shouldStayOnPlanDetailPage predicate keeps it on Issue Detail.
    const plan = await env.api.createCreateDatabasePlan(
      env.project,
      `E2E Redirect B2 ${ts}`,
      {
        id: `spec-${ts}`,
        target: env.instance,
        database: `e2e_redirect_stay_${ts}`,
      },
    );
    const issue = await env.api.createIssue(
      env.project,
      `E2E Redirect B2 ${ts}`,
      plan.name,
    );
    issueId = issue.name.split("/").pop()!;
  });

  test("opening the create-database issue URL stays on Issue Detail (no redirect)", async () => {
    const issuePath = `/projects/${projectId}/issues/${issueId}`;
    await page.goto(`${env.baseURL}${issuePath}`);
    await page.waitForLoadState("networkidle").catch(() => {});
    // Give any (incorrect) redirect a chance to fire, then assert we stayed.
    await expect(async () => {
      expect(new URL(page.url()).pathname).toBe(issuePath);
    }).toPass({ timeout: 10_000 });
    expect(page.url()).not.toContain("/plans/");
  });
});

test.describe("Rollout on a create-database issue stays on Issue Detail (B3)", () => {
  let issueId = "";
  let planId = "";
  let databaseName = "";

  test.beforeAll(async () => {
    const ts = Date.now();
    databaseName = `e2e_redirect_rollout_${ts}`;
    const plan = await env.api.createCreateDatabasePlan(
      env.project,
      `E2E Redirect B3 ${ts}`,
      {
        id: `spec-${ts}`,
        target: env.instance,
        database: databaseName,
        // The rollout runs for real on the sample Postgres, which needs an
        // owner, and a stage needs an environment.
        environment: "environments/prod",
        characterSet: "UTF8",
        collation: "en_US.UTF-8",
        owner: "bbsample",
      },
    );
    planId = plan.name.split("/").pop()!;
    const issue = await env.api.createIssue(
      env.project,
      `E2E Redirect B3 ${ts}`,
      plan.name,
    );
    issueId = issue.name.split("/").pop()!;
  });

  test.afterAll(async () => {
    // The rollout creates the database on the sample instance; drop it so the
    // instance does not accumulate one per run.
    try {
      const port = await getInstancePgPort(env);
      execSql("postgres", port, `DROP DATABASE IF EXISTS "${databaseName}"`);
    } catch {
      /* best effort */
    }
  });

  test("clicking Rollout keeps the user on the issue without a bounce through Plan Detail", async () => {
    const issuePath = `/projects/${projectId}/issues/${issueId}`;
    await page.goto(`${env.baseURL}${issuePath}`);
    const rolloutButton = page.getByRole("button", { name: "Rollout", exact: true });
    await expect(rolloutButton).toBeEnabled({ timeout: 20_000 });

    // The bug pushed /plans/{id}/rollout and Plan Detail replaced it with the
    // issue URL: a visible bounce plus one extra history entry. Record both.
    const historyBefore = await page.evaluate(() => window.history.length);
    const visited: string[] = [];
    page.on("framenavigated", (frame) => {
      if (frame === page.mainFrame()) visited.push(frame.url());
    });

    await rolloutButton.click();
    // The create-database task runs on rollout; wait for it to report done.
    await expect(page.getByText("(Created)")).toBeVisible({ timeout: 60_000 });

    expect(new URL(page.url()).pathname).toBe(issuePath);
    expect(visited.filter((url) => url.includes("/plans/"))).toEqual([]);
    expect(await page.evaluate(() => window.history.length)).toBe(historyBefore);

    // Activity records the rollout without a "for plan #id" link, which would
    // bounce the same way.
    await expect(page.getByText(/rollout created/)).toBeVisible({ timeout: 20_000 });
    await expect(page.locator(`a[href*="/plans/${planId}"]`)).toHaveCount(0);
  });
});
