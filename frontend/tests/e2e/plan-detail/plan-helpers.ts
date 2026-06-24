import type { BytebaseApiClient } from "../framework/api-client";

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

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

// Seed a single-spec change plan + its issue against `database`, optionally
// running plan checks and waiting for them to finish. Returns the ids the
// review specs need. Creation only — the caller decides which approval state to
// wait for (PENDING vs SKIPPED) since that depends on the project/workspace
// settings the caller configured.
export async function seedReviewPlan(
  api: BytebaseApiClient,
  project: string,
  database: string,
  opts: { prefix: string; sql: string; runChecks?: boolean },
): Promise<{ planId: string; planName: string; issueName: string }> {
  const ts = `${Date.now()}-${Math.floor(Math.random() * 1e6)}`;
  const sheet = await api.createSheet(project, opts.sql);
  const plan = await api.createPlan(project, `${opts.prefix} ${ts}`, [
    { id: `spec-${ts}`, targets: [database], sheet },
  ]);
  const issue = await api.createIssue(project, `${opts.prefix} ${ts}`, plan.name);
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
      const run = await api.getPlanCheckRun(planName);
      lastStatus = run.status;
      if (run.status === "DONE") return;
    } catch {
      /* check run not created yet — keep polling */
    }
    await new Promise((r) => setTimeout(r, 1500));
  }
  // Fail closed: returning silently here would let the caller assert against
  // an unfinished plan and fail far from the cause. Throw so the failure
  // points at the stuck plan check, not a downstream UI timeout.
  throw new Error(
    `plan checks for ${planName} did not reach DONE within ${timeoutMs}ms (last status: ${lastStatus})`,
  );
}
