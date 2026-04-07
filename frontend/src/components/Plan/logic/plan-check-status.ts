import {
  type PlanCheckRun,
  PlanCheckRun_Status,
} from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";

/**
 * Counts derived from `PlanCheckRun[]` (same shape as API `planCheckRunStatusCount` aggregates).
 *
 * Cost is O(total `results` across runs). The API returns one consolidated run per plan
 * (`refreshPlanCheckRuns` stores `[response]`), so the dominant size is `run.results.length`,
 * not the number of runs.
 */
export interface PlanCheckStatusSummary {
  running: number;
  success: number;
  warning: number;
  error: number;
  total: number;
}

export const EMPTY_PLAN_CHECK_STATUS_SUMMARY: PlanCheckStatusSummary = {
  running: 0,
  success: 0,
  warning: 0,
  error: 0,
  total: 0,
};

/**
 * Counts advice rows plus run-level RUNNING / FAILED, matching ChecksView grouping logic.
 */
export function computePlanCheckStatusSummary(
  runs: PlanCheckRun[]
): PlanCheckStatusSummary {
  let running = 0;
  let success = 0;
  let warning = 0;
  let error = 0;

  for (const checkRun of runs) {
    if (checkRun.status === PlanCheckRun_Status.RUNNING) {
      running++;
    }
    if (checkRun.status === PlanCheckRun_Status.FAILED) {
      error++;
    }
    for (const result of checkRun.results) {
      if (result.status === Advice_Level.ERROR) {
        error++;
      } else if (result.status === Advice_Level.WARNING) {
        warning++;
      } else if (result.status === Advice_Level.SUCCESS) {
        success++;
      }
    }
  }

  const total = running + success + warning + error;
  return { running, success, warning, error, total };
}

/** Worst level for icon / badge emphasis (mirrors `usePlanCheckStatus` plan-level rules). */
export function overallAdviceLevelFromPlanCheckSummary(
  summary: PlanCheckStatusSummary
): Advice_Level {
  if (summary.error > 0) {
    return Advice_Level.ERROR;
  }
  if (summary.warning > 0) {
    return Advice_Level.WARNING;
  }
  if (summary.success > 0) {
    return Advice_Level.SUCCESS;
  }
  return Advice_Level.SUCCESS;
}
