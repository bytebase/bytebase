// Plan Detail logs — the task run log viewer inside the deploy section's task
// card, for a run the driver retried inside itself.
//
// A lock held by another session makes the statement time out; the driver
// retries within the same task run (project Execution retry policy), and the
// run passes once the lock is released. The user contract (BYT-9993): the log
// shows one gray "Previous attempts" row above the final attempt's green
// sections — nothing red, nothing pre-opened — and the failed attempt stays in
// the record behind it, its error entry intact.

import type { ChildProcess } from "child_process";
import { expect, type Locator, type Page, test } from "@playwright/test";
import type { BytebaseApiClient } from "../framework/api-client";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { execSqlScript, getInstancePgPort, querySql, spawnPsql } from "../framework/psql";
import { PlanDetailPage } from "./plan-detail.page";
import { seedReviewPlan, setPermissiveGates, waitForRollout } from "./plan-helpers";

test.setTimeout(240_000);

// Static identifiers, inlined into SQL below (letters, digits, underscores only).
const SCHEMA = "e2e_retry_log";
const TABLE = "t";
// Counts a run's executions; sequence advances survive the rolled-back attempts.
const ATTEMPT_SEQUENCE = "attempt";
const LOCK_TIMEOUT_TEXT = "canceling statement due to lock timeout";
// Well past the failures a test schedules, so the run cannot fail first.
const RETRY_BUDGET = 10;

let env: TestEnv & { api: BytebaseApiClient };
let projectId: string;
let pgPort: string;
let lockHolder: ChildProcess | undefined;
let originalSettings: {
  requireIssueApproval: boolean;
  requirePlanCheckNoError: boolean;
  enforceSqlReview: boolean;
  forceIssueLabels: boolean;
  executionRetryPolicy: { maximumRetries?: number };
};

// The sessions holding the fixture table exclusively (`granted` keeps the
// waiting ALTER TABLE out of it).
const HELD_LOCK_ROWS = `FROM pg_locks l
   JOIN pg_class c ON c.oid = l.relation
   JOIN pg_namespace n ON n.oid = c.relnamespace
  WHERE n.nspname = '${SCHEMA}' AND c.relname = '${TABLE}'
    AND l.mode = 'AccessExclusiveLock' AND l.granted`;

async function waitForLockHeld(): Promise<void> {
  await expect
    .poll(() => querySql(env.databaseId, pgPort, `SELECT count(*) ${HELD_LOCK_ROWS}`), {
      timeout: 15_000,
      message: `lock on ${SCHEMA}.${TABLE} was not granted`,
    })
    .not.toBe("0");
}

function releaseLock(): void {
  lockHolder?.kill();
  lockHolder = undefined;
}

// The first `failures` executions time out on the held lock; the one after
// them waits for the release, so the attempt count does not depend on timing.
// The driver re-runs the whole sheet on each attempt, and it retries only on
// this exact PostgreSQL error.
const retriedSql = (failures: number) =>
  [
    `SELECT set_config('lock_timeout', CASE WHEN nextval('${SCHEMA}.${ATTEMPT_SEQUENCE}') <= ${failures} THEN '2s' ELSE '0' END, false);`,
    `ALTER TABLE ${SCHEMA}.${TABLE} ADD COLUMN IF NOT EXISTS c${failures} TEXT;`,
  ].join("\n");

async function runRetriedPlan(
  page: Page,
  failures: number,
): Promise<{ viewer: Locator; umbrella: Locator }> {
  const planPage = new PlanDetailPage(page, env.baseURL);
  execSqlScript(env.databaseId, pgPort, `ALTER SEQUENCE ${SCHEMA}.${ATTEMPT_SEQUENCE} RESTART WITH 1`);

  const seeded = await seedReviewPlan(env, page, {
    prefix: `E2E Log retry x${failures}`,
    sql: retriedSql(failures),
    runChecks: false,
  });
  await waitForRollout(env.api, seeded.planName);

  await planPage.goto(projectId, seeded.planId);
  await planPage.dismissModals();
  await expect(planPage.headerRunStage).toBeVisible({ timeout: 20_000 });

  // Held only from here, so seeding and plan checks never wait on it. hr_test
  // carries a baseline changelog from the sample setup; a database without one
  // would sync its schema before executing and block on this lock instead.
  lockHolder = spawnPsql(
    env.databaseId,
    pgPort,
    `BEGIN; LOCK TABLE ${SCHEMA}.${TABLE} IN ACCESS EXCLUSIVE MODE;`,
  );
  await waitForLockHeld();

  await planPage.headerRunStage.click();
  await planPage.confirmRunTaskDialog();

  // The umbrella says so once the attempt after the last scheduled failure is
  // waiting on the lock.
  const viewer = page.getByTestId("task-run-log-viewer");
  const umbrella = page.getByTestId("task-run-log-previous-attempts");
  await expect(umbrella).toHaveText(
    new RegExp(`Previous attempts\\s*${failures} attempts? · retrying`),
    { timeout: 60_000 },
  );

  releaseLock();
  await expect(planPage.headerStamp("Deployed")).toBeVisible({ timeout: 60_000 });
  await expect(umbrella).toBeVisible({ timeout: 15_000 });
  return { viewer, umbrella };
}

async function expectFoldedHistory(
  viewer: Locator,
  umbrella: Locator,
  count: number,
): Promise<void> {
  await expect(umbrella).toHaveAttribute("aria-expanded", "false");
  await expect(umbrella).toHaveText(new RegExp(`Previous attempts\\s*${count} attempts?`));
  await expect(umbrella).not.toHaveText(/retrying/);
  await expect(viewer.getByText(LOCK_TIMEOUT_TEXT)).toHaveCount(0);
  await expect(viewer.locator("button[aria-expanded='true']")).toHaveCount(0);
  const finalCommand = viewer.getByRole("button", { name: /Command Execute/ }).last();
  await expect(finalCommand).toBeVisible();
  const [umbrellaBox, finalBox] = await Promise.all([
    umbrella.boundingBox(),
    finalCommand.boundingBox(),
  ]);
  expect(umbrellaBox!.y).toBeLessThan(finalBox!.y);
}

const failedEntries = (viewer: Locator) =>
  viewer.getByTestId("task-run-log-row").filter({ hasText: LOCK_TIMEOUT_TEXT });

test.beforeAll(async () => {
  env = loadTestEnv();
  projectId = env.project.split("/").pop()!;
  pgPort = await getInstancePgPort(env);

  const project = await env.api.getProject(env.project);
  originalSettings = {
    requireIssueApproval: !!project.requireIssueApproval,
    requirePlanCheckNoError: !!project.requirePlanCheckNoError,
    enforceSqlReview: !!project.enforceSqlReview,
    forceIssueLabels: !!project.forceIssueLabels,
    executionRetryPolicy:
      (project.executionRetryPolicy as { maximumRetries?: number } | undefined) ?? {
        maximumRetries: 0,
      },
  };

  await setPermissiveGates(env.api, env.project, {
    executionRetryPolicy: { maximumRetries: RETRY_BUDGET },
  });

  execSqlScript(
    env.databaseId,
    pgPort,
    [
      `DROP SCHEMA IF EXISTS ${SCHEMA} CASCADE;`,
      `CREATE SCHEMA ${SCHEMA};`,
      `CREATE TABLE ${SCHEMA}.${TABLE} (id INT PRIMARY KEY);`,
      `CREATE SEQUENCE ${SCHEMA}.${ATTEMPT_SEQUENCE};`,
    ].join("\n"),
  );
});

// A failed test must not leave the next one waiting on its lock.
test.afterEach(() => {
  releaseLock();
});

test.afterAll(async () => {
  // Settings first, so a psql failure below cannot leave them changed.
  await env.api.updateProjectSettings(env.project, originalSettings);
  execSqlScript(env.databaseId, pgPort, `DROP SCHEMA IF EXISTS ${SCHEMA} CASCADE`);
});

test("one previous attempt: the umbrella opens straight onto its sections, the failed one already open", async ({ page }) => {
  const { viewer, umbrella } = await runRetriedPlan(page, 1);
  await expectFoldedHistory(viewer, umbrella, 1);

  await umbrella.click();
  await expect(umbrella).toHaveAttribute("aria-expanded", "true");
  await expect(page.getByTestId("task-run-log-attempt")).toHaveCount(0);
  // The superseded Command Execute header comes first in the document.
  await expect(
    viewer.getByRole("button", { name: /Command Execute/ }).first(),
  ).toHaveAttribute("aria-expanded", "true");
  await expect(failedEntries(viewer).first()).toBeVisible();
});

test("several previous attempts: one nested row per attempt, each opening onto its sections", async ({ page }) => {
  const { viewer, umbrella } = await runRetriedPlan(page, 2);
  await expectFoldedHistory(viewer, umbrella, 2);

  await umbrella.click();
  const attemptRows = page.getByTestId("task-run-log-attempt");
  await expect(attemptRows).toHaveCount(2);
  for (const row of await attemptRows.all()) {
    await expect(row).toHaveAttribute("aria-expanded", "false");
  }
  await expect(failedEntries(viewer)).toHaveCount(0);

  await attemptRows.first().click();
  await expect(attemptRows.first()).toHaveAttribute("aria-expanded", "true");
  await expect(failedEntries(viewer).first()).toBeVisible();
});
