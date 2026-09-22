import type { Page } from "@playwright/test";
import type { BytebaseApiClient } from "../framework/api-client";
import type { TestEnv } from "../framework/env";
import { createSubmittedDatabaseChangePlanViaUI } from "../framework/ui-create-plan";

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

// Bound a single polling request so a server-side stall can't block the whole
// poll past its deadline. Without this a hung fetch (e.g. the server holding a
// lock in the plan-check/rollout path) never returns, so the surrounding
// `while (Date.now() < deadline)` loop never re-checks the deadline and the test
// hangs for many minutes instead of failing at its intended timeout. The
// underlying request is left to settle in the background (ignored); the caller
// simply retries on the next tick.
function withTimeout<T>(p: Promise<T>, ms: number, label: string): Promise<T> {
  return Promise.race([
    p,
    new Promise<T>((_, reject) =>
      setTimeout(() => reject(new Error(`${label} timed out after ${ms}ms`)), ms),
    ),
  ]);
}

// Recover a plan's linked review-issue name from the Plan proto's OUTPUT_ONLY
// `issue` field. A UI-created plan's draft review issue is created together with
// the plan, so this is usually populated immediately — but poll briefly to
// absorb any read-after-write lag before failing closed.
export async function resolveIssueName(
  api: BytebaseApiClient,
  planName: string,
  timeoutMs = 15_000,
): Promise<string> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      const issue = (await withTimeout(api.getPlan(planName), 10_000, "getPlan"))
        .issue;
      if (issue) return issue;
    } catch {
      /* transient read stall — keep polling until the deadline */
    }
    await sleep(500);
  }
  throw new Error(
    `plan ${planName} has no linked issue within ${timeoutMs}ms`,
  );
}

// Poll an issue's approvalStatus until it reaches one of `accept`. Approval-flow
// generation is async after issue creation, so the status briefly sits at
// CHECKING before settling on PENDING / APPROVED / SKIPPED / REJECTED. Fail
// closed on timeout so a regression in flow generation surfaces loudly.
export async function waitForApprovalStatus(
  api: BytebaseApiClient,
  issueName: string,
  accept: string[],
  timeoutMs = 40_000,
): Promise<string> {
  const deadline = Date.now() + timeoutMs;
  let last = "<none>";
  while (Date.now() < deadline) {
    last = (await api.getIssue(issueName)).approvalStatus;
    if (accept.includes(last)) return last;
    await sleep(1000);
  }
  throw new Error(
    `issue ${issueName} approvalStatus did not reach ${accept.join("|")} within ${timeoutMs}ms (last: ${last})`,
  );
}

// Seed a single-spec change plan + its submitted review issue against
// env.database, optionally waiting for plan checks to finish. Returns the ids
// the review specs need. Driven through the UI (create + "Ready for Review") so
// setup follows the real user workflow and breaks loudly on drift, not silently
// — the same reason the schema-editor / scroll / smoke setups were migrated off
// bare api.createPlan. The issue name comes from the plan's OUTPUT_ONLY `issue`
// field (resolveIssueName). Submission only — the caller decides which approval
// state to wait for (PENDING vs SKIPPED) via the project/workspace settings it
// configured before calling.
export async function seedReviewPlan(
  env: TestEnv & { api: BytebaseApiClient },
  page: Page,
  opts: { prefix: string; sql: string; runChecks?: boolean },
): Promise<{ planId: string; planName: string; issueName: string }> {
  const ts = `${Date.now()}-${Math.floor(Math.random() * 1e6)}`;
  const projectId = env.project.split("/").pop()!;
  const { planId } = await createSubmittedDatabaseChangePlanViaUI(page, {
    baseURL: env.baseURL,
    projectId,
    database: env.database,
    title: `${opts.prefix} ${ts}`,
    sql: opts.sql,
  });
  const planName = `${env.project}/plans/${planId}`;
  const issueName = await resolveIssueName(env.api, planName);
  if (opts.runChecks) {
    await waitForPlanChecksDone(env.api, planName);
  }
  return { planId, planName, issueName };
}

// Seed the same database-change shape as seedReviewPlan, but keep its linked
// Issue in the draft lifecycle state. Ready-for-review E2E tests use this as
// setup so the browser owns the actual submission transition.
export async function seedDraftPlan(
  api: BytebaseApiClient,
  project: string,
  database: string,
  opts: {
    prefix: string;
    sql: string;
    labels?: string[];
    runChecks?: boolean;
  },
): Promise<{ planId: string; planName: string; issueName: string }> {
  const ts = `${Date.now()}-${Math.floor(Math.random() * 1e6)}`;
  const title = `${opts.prefix} ${ts}`;
  const sheet = await api.createSheet(project, opts.sql);
  const plan = await api.createPlan(project, title, [
    { id: `spec-${ts}`, targets: [database], sheet },
  ]);
  const issue = await api.createDraftIssue(
    project,
    title,
    plan.name,
    opts.labels,
  );
  if (opts.runChecks) {
    await api.runPlanChecks(plan.name);
    await waitForPlanChecksDone(api, plan.name);
  }
  return {
    planId: plan.name.split("/").pop()!,
    planName: plan.name,
    issueName: issue.name,
  };
}

// Poll the latest planCheckRun on `planName` until status === "DONE" or the
// timeout elapses. The check run is created asynchronously after
// runPlanChecks(), so getPlanCheckRun may briefly 404 — we swallow and
// retry. Tuned in one place so a slow-CI timeout bump applies to every
// caller (plan-detail-checks + plan-detail-rollout).
export async function waitForPlanChecksDone(
  api: BytebaseApiClient,
  planName: string,
  timeoutMs = 60_000,
): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  let lastStatus = "<none>";
  while (Date.now() < deadline) {
    try {
      const run = await withTimeout(
        api.getPlanCheckRun(planName),
        10_000,
        "getPlanCheckRun",
      );
      lastStatus = run.status;
      if (run.status === "DONE") return;
    } catch {
      /* check run not created yet, or the request stalled — keep polling */
    }
    await sleep(1500);
  }
  // Fail closed: returning silently here would let the caller assert against
  // an unfinished plan and fail far from the cause. Throw so the failure
  // points at the stuck plan check, not a downstream UI timeout.
  throw new Error(
    `plan checks for ${planName} did not reach DONE within ${timeoutMs}ms (last status: ${lastStatus})`,
  );
}
