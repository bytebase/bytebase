import { planCheckRunSummaryForCheckRunList } from "@/components/PlanCheckRun/common";
import type { Plan_Spec } from "@/types/proto/v1/plan_service";
import type { CheckReleaseResponse_CheckResult } from "@/types/proto/v1/release_service";
import { Advice_Status } from "@/types/proto/v1/sql_service";
import { sheetNameOfSpec } from "@/utils";
import { type PlanContext, targetsForSpec } from "../../logic";

export const filterSpec = (
  planContext: PlanContext,
  sqlCheckResultMap: Record<string, CheckReleaseResponse_CheckResult>,
  spec: Plan_Spec,
  {
    adviceStatus,
  }: {
    adviceStatus?: Advice_Status;
  }
): boolean => {
  const { isCreating } = planContext;
  const targets = targetsForSpec(spec);

  if (adviceStatus) {
    if (isCreating.value) {
      // Check if any target matches the filter criteria.
      return targets.some((target) => {
        const result = sqlCheckResultMap[target];
        if (adviceStatus === Advice_Status.UNRECOGNIZED) {
          return !Boolean(result);
        }
        if (adviceStatus === Advice_Status.SUCCESS) {
          return result && result.advices.length === 0;
        }
        return (
          result &&
          result.advices.some((advice) => advice.status === adviceStatus)
        );
      });
    } else {
      // Check if any target has the specified advice status.
      return targets.some((target) => {
        const planCheckRuns = planContext.plan.value.planCheckRunList.filter(
          (run) => run.sheet === sheetNameOfSpec(spec) && run.target === target
        );
        const summary = planCheckRunSummaryForCheckRunList(planCheckRuns);
        if (summary.errorCount > 0) {
          return adviceStatus === Advice_Status.ERROR;
        } else if (summary.warnCount > 0) {
          return adviceStatus === Advice_Status.WARNING;
        } else {
          return adviceStatus === Advice_Status.SUCCESS;
        }
      });
    }
  }

  return false;
};
