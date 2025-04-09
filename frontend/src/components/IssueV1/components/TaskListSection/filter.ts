import type { Task, Task_Status } from "@/types/proto/v1/rollout_service";
import { Advice_Status } from "@/types/proto/v1/sql_service";
import { type IssueContext } from "../../logic";
import { type SQLCheckContext } from "../SQLCheckSection/context";

export interface TaskFilter {
  // Only for created tasks.
  status: Task_Status[];
  adviceStatus: Advice_Status[];
}

export const filterTask = (
  issueContext: IssueContext,
  sqlCheckContext: SQLCheckContext,
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
  if (adviceStatus) {
    if (isCreating.value) {
      const { enabled, resultMap } = sqlCheckContext;
      if (enabled.value) {
        const result = resultMap.value[task.target];
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
      }
    } else {
      const planCheckRuns = getPlanCheckRunsForTask(task);
      return planCheckRuns.some((run) =>
        run.results.some(
          (result) => result.status.toString() === adviceStatus.toString()
        )
      );
    }
  }

  return false;
};
