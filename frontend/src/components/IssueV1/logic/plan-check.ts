import { ComposedIssue } from "@/types";
import {
  PlanCheckRun,
  PlanCheckRun_Result_Status,
  PlanCheckRun_Status,
  PlanCheckRun_Type,
  Task,
  Task_Status,
} from "@/types/proto/v1/rollout_service";
import { groupBy, maxBy } from "lodash-es";
import { databaseForTask } from ".";

export const planCheckRunListForTask = (issue: ComposedIssue, task: Task) => {
  const database = databaseForTask(issue, task);
  return issue.planCheckRunList.filter(
    (check) => check.target === database.name
  );
};

export const allPlanChecksPassedForTask = (
  issue: ComposedIssue,
  task: Task
) => {
  return false; // TODO
};

export const HiddenPlanCheckTypes = new Set<PlanCheckRun_Type>([
  PlanCheckRun_Type.DATABASE_STATEMENT_SUMMARY_REPORT,
]);

export const planCheckRunResultStatus = (checkRun: PlanCheckRun) => {
  let status = PlanCheckRun_Result_Status.SUCCESS;

  for (const result of checkRun.results) {
    if (result.status === PlanCheckRun_Result_Status.ERROR) {
      return PlanCheckRun_Result_Status.ERROR;
    }
    if (result.status === PlanCheckRun_Result_Status.WARNING) {
      status = PlanCheckRun_Result_Status.WARNING;
    }
  }
  return status;
};

export const planCheckStatusForTask = (issue: ComposedIssue, task: Task) => {
  if (
    task.status === Task_Status.PENDING ||
    task.status === Task_Status.PENDING_APPROVAL
  ) {
    const summary = planCheckRunSummaryForTask(issue, task);
    if (summary.errorCount > 0) return PlanCheckRun_Result_Status.ERROR;
    if (summary.warnCount > 0) return PlanCheckRun_Result_Status.WARNING;
  }
  return undefined;
};

export type PlanCheckRunSummary = {
  runningCount: number;
  successCount: number;
  warnCount: number;
  errorCount: number;
};

export const planCheckRunSummaryForCheckRunList = (
  planCheckRunList: PlanCheckRun[]
) => {
  const summary: PlanCheckRunSummary = {
    runningCount: 0,
    successCount: 0,
    warnCount: 0,
    errorCount: 0,
  };

  planCheckRunList = planCheckRunList.filter(
    (check) => !HiddenPlanCheckTypes.has(check.type)
  );

  const listGroupByType = groupBy(
    planCheckRunList,
    (checkRun) => checkRun.type
  );

  const latestCheckRunOfEachType = Object.keys(listGroupByType).map((type) => {
    const listOfType = listGroupByType[type];
    const latest = maxBy(listOfType, (checkRun) => Number(checkRun.uid))!;
    return latest;
  });

  for (const checkRun of latestCheckRunOfEachType) {
    switch (checkRun.status) {
      case PlanCheckRun_Status.CANCELED:
        // nothing todo
        break;
      case PlanCheckRun_Status.FAILED:
        summary.errorCount++;
        break;
      case PlanCheckRun_Status.RUNNING:
        summary.runningCount++;
        break;
      case PlanCheckRun_Status.DONE:
        switch (planCheckRunResultStatus(checkRun)) {
          case PlanCheckRun_Result_Status.SUCCESS:
            summary.successCount++;
            break;
          case PlanCheckRun_Result_Status.WARNING:
            summary.warnCount++;
            break;
          case PlanCheckRun_Result_Status.ERROR:
            summary.errorCount++;
        }
    }
  }

  return summary;
};

export const planCheckRunSummaryForIssue = (issue: ComposedIssue) => {
  return planCheckRunSummaryForCheckRunList(issue.planCheckRunList);
};

export const planCheckRunSummaryForTask = (
  issue: ComposedIssue,
  task: Task
) => {
  const planCheckRunList = planCheckRunListForTask(issue, task);
  return planCheckRunSummaryForCheckRunList(planCheckRunList);
};

// export const isTaskSkipped = (task: Task): boolean => {
//   if (task.status === "DONE") {
//     const payload = task.payload as any;
//     return payload?.skipped === true;
//   }
//   return false;
// };

// export const isTaskTriggeredByVCS = (task: Task): boolean => {
//   const taskTypesWithPushEvent: TaskType[] = [
//     "bb.task.database.data.update",
//     "bb.task.database.schema.update",
//     "bb.task.database.schema.update-sdl",
//     "bb.task.database.schema.update.ghost.sync",
//     "bb.task.database.schema.baseline",
//   ];

//   type PayloadTypesWithPushEvent =
//     | TaskDatabaseDataUpdatePayload
//     | TaskDatabaseSchemaUpdatePayload
//     | TaskDatabaseSchemaUpdateSDLPayload
//     | TaskDatabaseSchemaUpdateGhostSyncPayload
//     | TaskDatabaseSchemaBaselinePayload;

//   if (taskTypesWithPushEvent.includes(task.type)) {
//     const payload = task.payload as PayloadTypesWithPushEvent | undefined;

//     if (payload && payload.pushEvent) {
//       return true;
//     }
//   }

//   return false;
// };

// export const isTaskEntity = (task: Task | TaskCreate): task is Task => {
//   const obj = task as any;
//   return typeof obj["id"] === "number";
// };

// export const isTaskCreate = (task: Task | TaskCreate): task is TaskCreate => {
//   return !isTaskEntity(task);
// };

// /**
//  *
//  * @param task
//  * @param issue
//  * @param activeOnly if true, only "Active Task" can be skipped
//  * @returns
//  */
// export const canSkipTask = (
//   task: Task,
//   issue: Issue,
//   activeOnly = false,
//   failedOnly = false
// ) => {
//   const pipeline = issue.pipeline;
//   const isActiveTask = task.id === activeTask(pipeline!).id;
//   if (activeOnly && !isActiveTask) {
//     return false;
//   }

//   const applicableStatusList: TaskStatus[] = failedOnly
//     ? ["FAILED"]
//     : ["PENDING_APPROVAL", "FAILED"];

//   if (!applicableStatusList.includes(task.status)) {
//     return false;
//   }

//   const currentUserV1 = useCurrentUserV1();

//   if (
//     hasWorkspacePermissionV1(
//       "bb.permission.workspace.manage-issue",
//       currentUserV1.value.userRole
//     )
//   ) {
//     return true;
//   }

//   if (extractUserUID(currentUserV1.value.name) === String(issue.assignee.id)) {
//     return true;
//   }

//   return false;
// };

// export const checkStatusOfTask = (task: Task): TaskCheckStatus | undefined => {
//   if (task.status === "PENDING" || task.status === "PENDING_APPROVAL") {
//     const summary = taskCheckRunSummary(task);
//     if (summary.errorCount > 0) return "ERROR";
//     if (summary.warnCount > 0) return "WARN";
//   }
//   return undefined;
// };
