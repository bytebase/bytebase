import { computed, type ComputedRef, type Ref } from "vue";
import {
  PlanCheckRun_Result_Status,
  PlanCheckRun_Status,
  type Plan,
} from "@/types/proto-es/v1/plan_service_pb";

export interface PlanCheckStatusSummary {
  running: number;
  success: number;
  warning: number;
  error: number;
  total: number;
}

export const usePlanCheckStatus = (
  plan: ComputedRef<Plan | undefined> | Ref<Plan | undefined>
) => {
  const getCheckCount = (status: PlanCheckRun_Result_Status): number => {
    if (!plan.value) return 0;
    const statusKey = PlanCheckRun_Result_Status[status];
    return plan.value.planCheckRunStatusCount?.[statusKey] || 0;
  };

  const getOverallStatus = computed(() => {
    if (!plan.value) return PlanCheckRun_Result_Status.STATUS_UNSPECIFIED;

    const statusCount = plan.value.planCheckRunStatusCount || {};

    if (statusCount["ERROR"] > 0 || statusCount["FAILED"] > 0) {
      return PlanCheckRun_Result_Status.ERROR;
    } else if (statusCount["WARNING"] > 0) {
      return PlanCheckRun_Result_Status.WARNING;
    } else if (statusCount["SUCCESS"] > 0) {
      return PlanCheckRun_Result_Status.SUCCESS;
    }

    // Default to SUCCESS if no errors or warnings.
    return PlanCheckRun_Result_Status.SUCCESS;
  });

  const statusSummary = computed<PlanCheckStatusSummary>(() => {
    if (!plan.value) {
      return { running: 0, success: 0, warning: 0, error: 0, total: 0 };
    }

    const statusCount = plan.value.planCheckRunStatusCount || {};
    const running =
      statusCount[PlanCheckRun_Status[PlanCheckRun_Status.RUNNING]] || 0;
    const success =
      statusCount[
        PlanCheckRun_Result_Status[PlanCheckRun_Result_Status.SUCCESS]
      ] || 0;
    const warning =
      statusCount[
        PlanCheckRun_Result_Status[PlanCheckRun_Result_Status.WARNING]
      ] || 0;
    const error =
      statusCount[
        PlanCheckRun_Result_Status[PlanCheckRun_Result_Status.ERROR]
      ] || 0;
    // Also count FAILED plan check runs as errors
    const failed =
      statusCount[PlanCheckRun_Status[PlanCheckRun_Status.FAILED]] || 0;
    const totalError = error + failed;
    const total = running + success + warning + totalError;

    return { running, success, warning, error: totalError, total };
  });

  const hasAnyStatus = computed(() => {
    const summary = statusSummary.value;
    return summary.total > 0;
  });

  const hasRunning = computed(() => {
    return statusSummary.value.running > 0;
  });

  const hasErrors = computed(() => {
    return statusSummary.value.error > 0;
  });

  const hasWarnings = computed(() => {
    return statusSummary.value.warning > 0;
  });

  const statusCountString = computed(() => {
    return JSON.stringify(plan.value?.planCheckRunStatusCount || {});
  });

  return {
    getCheckCount,
    getOverallStatus,
    statusSummary,
    hasAnyStatus,
    hasRunning,
    hasErrors,
    hasWarnings,
    statusCountString,
  };
};
