import { sheetNameForSpec } from "@/components/Plan";
import { planCheckRunSummaryForCheckRunList } from "@/components/PlanCheckRun/common";
import { databaseForTask } from "@/components/Rollout/RolloutDetail";
import type { ComposedIssue } from "@/types";
import {
  PlanCheckRun_Result_Status,
  type Plan_Spec,
} from "@/types/proto/v1/plan_service";
import type { Task } from "@/types/proto/v1/rollout_service";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import { specForTask, useIssueContext, projectOfIssue } from ".";

export const planSpecHasPlanChecks = (spec: Plan_Spec) => {
  if (spec.createDatabaseConfig) {
    return false;
  }
  if (spec.changeDatabaseConfig !== undefined) {
    return true;
  }
  if (spec.exportDataConfig !== undefined) {
    return true;
  }
  return false;
};

export const planCheckRunListForTask = (issue: ComposedIssue, task: Task) => {
  const target = databaseForTask(projectOfIssue(issue), task).name;
  const spec = specForTask(issue.planEntity, task);
  const sheet = spec ? sheetNameForSpec(spec) : "";
  return issue.planCheckRunList.filter((check) => {
    return check.sheet === sheet && check.target === target;
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
    if (summary.errorCount > 0) return PlanCheckRun_Result_Status.ERROR;
    if (summary.warnCount > 0) return PlanCheckRun_Result_Status.WARNING;
  }
  return undefined;
};

export const planCheckRunSummaryForIssue = (issue: ComposedIssue) => {
  const sheets = issue.planEntity?.specs.reduce((acc, spec) => {
    if (spec.changeDatabaseConfig?.sheet) {
      acc.add(spec.changeDatabaseConfig.sheet);
    }
    return acc;
  }, new Set<string>());
  const planCheckRunList = issue.planCheckRunList.filter((check) => {
    return sheets?.has(check.sheet);
  });

  return planCheckRunSummaryForCheckRunList(planCheckRunList);
};
