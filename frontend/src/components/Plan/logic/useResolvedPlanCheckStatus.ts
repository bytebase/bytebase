import { type ComputedRef, computed, type Ref } from "vue";
import type { Plan, PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import {
  computePlanCheckStatusSummary,
  overallAdviceLevelFromPlanCheckSummary,
} from "./plan-check-status";
import { usePlanCheckStatus } from "./usePlanCheckStatus";

/**
 * Resolves plan check counts from either:
 * - explicit `planCheckRuns` (e.g. filtered to the current spec), or
 * - `plan.planCheckRunStatusCount` (API aggregate for the whole plan).
 */
export const useResolvedPlanCheckStatus = (
  plan: ComputedRef<Plan | undefined> | Ref<Plan | undefined>,
  planCheckRunsOpt?:
    | ComputedRef<PlanCheckRun[] | undefined>
    | Ref<PlanCheckRun[] | undefined>
) => {
  const planLevel = usePlanCheckStatus(plan);

  const runBasedSummary = computed(() => {
    const runs = planCheckRunsOpt?.value;
    if (runs === undefined) {
      return undefined;
    }
    return computePlanCheckStatusSummary(runs);
  });

  const statusSummary = computed(() => {
    return runBasedSummary.value ?? planLevel.statusSummary.value;
  });

  const hasAnyStatus = computed(() => statusSummary.value.total > 0);

  const overallAdviceLevel = computed((): Advice_Level => {
    if (runBasedSummary.value !== undefined) {
      return overallAdviceLevelFromPlanCheckSummary(runBasedSummary.value);
    }
    return planLevel.getOverallStatus.value;
  });

  const hasRunning = computed(() => {
    if (runBasedSummary.value !== undefined) {
      return runBasedSummary.value.running > 0;
    }
    return planLevel.hasRunning.value;
  });

  return {
    statusSummary,
    hasAnyStatus,
    overallAdviceLevel,
    hasRunning,
    planLevel,
  };
};
