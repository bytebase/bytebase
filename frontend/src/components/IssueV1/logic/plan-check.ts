import { planCheckRunSummaryForCheckRunList } from "@/components/PlanCheckRun/common";
import type { ComposedIssue } from "@/types";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { databaseForTask } from "@/utils";
import { projectOfIssue, useIssueContext } from ".";

export const planSpecHasPlanChecks = (spec: Plan_Spec) => {
  if (spec.config?.case === "createDatabaseConfig") {
    return false;
  }
  if (spec.config?.case === "changeDatabaseConfig") {
    return true;
  }
  if (spec.config?.case === "exportDataConfig") {
    return true;
  }
  return false;
};

export const planCheckRunListForTask = (issue: ComposedIssue, task: Task) => {
  const target = databaseForTask(projectOfIssue(issue), task).name;

  // With consolidated model, return runs that have results for this target
  return issue.planCheckRunList.filter((run) => {
    return run.results.some((result) => result.target === target);
  });
};

export const planCheckStatusForTask = (task: Task) => {
  const { getPlanCheckRunsForTask } = useIssueContext();
  if (
    task.status === Task_Status.PENDING ||
    task.status === Task_Status.NOT_STARTED
  ) {
    const summary = planCheckRunSummaryForCheckRunList(
      getPlanCheckRunsForTask(task)
    );
    if (summary.errorCount > 0) return Advice_Level.ERROR;
    if (summary.warnCount > 0) return Advice_Level.WARNING;
  }
  return undefined;
};

export const planCheckRunSummaryForIssue = (issue: ComposedIssue) => {
  // With consolidated model, just use all plan check runs
  return planCheckRunSummaryForCheckRunList(issue.planCheckRunList);
};
