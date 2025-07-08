import { planCheckRunSummaryForCheckRunList } from "@/components/PlanCheckRun/common";
import type { CheckReleaseResponse_CheckResult } from "@/types/proto-es/v1/release_service_pb";
import type { Task, Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { Advice_Status } from "@/types/proto-es/v1/sql_service_pb";
import type { IssueContext } from "../../logic";

export const filterTask = (
  issueContext: IssueContext,
  sqlCheckResultMap: Record<string, CheckReleaseResponse_CheckResult>,
  task: Task,
  {
    status,
    adviceStatus,
  }: {
    status?: Task_Status;
    adviceStatus?: Advice_Status;
  }
): boolean => {
  const { isCreating, getPlanCheckRunsForTask } = issueContext;
  if (status) {
    return task.status === status;
  }
  if (adviceStatus !== undefined) {
    if (isCreating.value) {
      const result = sqlCheckResultMap[task.target];
      if (adviceStatus === Advice_Status.STATUS_UNSPECIFIED) {
        return !Boolean(result);
      }
      if (adviceStatus === Advice_Status.SUCCESS) {
        return result && result.advices.length === 0;
      }
      return (
        result &&
        result.advices.some((advice) => advice.status === adviceStatus)
      );
    } else {
      const checkRuns = getPlanCheckRunsForTask(task);
      const summary = planCheckRunSummaryForCheckRunList(checkRuns);
      if (summary.errorCount > 0) {
        return adviceStatus === Advice_Status.ERROR;
      } else if (summary.warnCount > 0) {
        return adviceStatus === Advice_Status.WARNING;
      } else {
        return adviceStatus === Advice_Status.SUCCESS;
      }
    }
  }

  return false;
};
