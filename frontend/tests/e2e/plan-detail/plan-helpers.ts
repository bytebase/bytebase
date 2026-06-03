import type { BytebaseApiClient } from "../framework/api-client";

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
