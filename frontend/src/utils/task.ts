import { groupBy, maxBy } from "lodash-es";
import { useCurrentUserV1, useDatabaseV1Store } from "@/store";
import {
  Issue,
  Task,
  TaskCheckStatus,
  TaskCheckType,
  TaskCreate,
  TaskDatabaseCreatePayload,
  TaskDatabaseDataUpdatePayload,
  TaskDatabasePITRRestorePayload,
  TaskDatabaseSchemaBaselinePayload,
  TaskDatabaseSchemaUpdateGhostSyncPayload,
  TaskDatabaseSchemaUpdatePayload,
  TaskDatabaseSchemaUpdateSDLPayload,
  TaskStatus,
  TaskType,
  unknownDatabase,
} from "@/types";
import { activeTask } from "./pipeline";
import { hasWorkspacePermissionV1 } from "./role";
import { issueSlug, stageSlug, taskSlug } from "./slug";
import { extractUserUID } from "./v1";

export const extractDatabaseNameFromTask = (
  task: Task | TaskCreate
): string => {
  const taskEntity = task as Task;
  const taskCreate = task as TaskCreate;
  if (typeof taskEntity.id !== "number") {
    // The task is not created yet
    // Find the database by databaseId if possible
    if (taskCreate.databaseId) {
      return useDatabaseV1Store().getDatabaseByUID(
        String(taskCreate.databaseId!)
      ).databaseName;
    }
    return unknownDatabase().databaseName;
  }

  const taskType = taskEntity.type;
  if (taskType === "bb.task.database.restore.pitr.restore") {
    // When PITR to new DB, the database might not be created yet.
    // So we need to extract the name from the task's payload。
    const payload = taskEntity.payload as TaskDatabasePITRRestorePayload;
    if (payload.databaseName) {
      return payload.databaseName;
    }

    // When PITR in-place, taskEntity.database will be the database entity itself.
  }

  // The task entity is related to a database entity
  if (taskEntity.database) {
    return taskEntity.database.name;
  }

  // The task entity is irrelative to any databases or the related
  // database is not created yet.
  if (
    taskType === "bb.task.database.create" ||
    taskType === "bb.task.database.restore"
  ) {
    // The database is not created yet.
    // extract database info from the task's and payload's properties.
    const payload = taskEntity.payload as TaskDatabaseCreatePayload;
    return payload.databaseName;
  }

  // Fallback to <<Unknown database>>. Won't be happy to see it.
  return unknownDatabase().databaseName;
};

export const buildIssueLinkWithTask = (
  issue: Issue,
  task: Task,
  simple = false
) => {
  const stage = task.stage;
  const stageIndex = issue.pipeline!.stageList.findIndex(
    (s) => s.id === stage.id
  );

  const issuePart = simple ? String(issue.id) : issueSlug(issue.name, issue.id);
  const stagePart = simple
    ? String(stageIndex + 1)
    : stageSlug(stage.name, stageIndex + 1);
  const taskPart = simple ? String(task.id) : taskSlug(task.name, task.id);

  const url = `/issue/${issuePart}?stage=${stagePart}&task=${taskPart}`;

  return url;
};

export const HiddenCheckTypes = new Set<TaskCheckType>([
  "bb.task-check.database.statement.type.report",
  "bb.task-check.database.statement.affected-rows.report",
]);

export type TaskCheckRunSummary = {
  runningCount: number;
  successCount: number;
  warnCount: number;
  errorCount: number;
};

export function taskCheckRunSummary(task?: Task): TaskCheckRunSummary {
  const summary: TaskCheckRunSummary = {
    runningCount: 0,
    successCount: 0,
    warnCount: 0,
    errorCount: 0,
  };

  if (!task) return summary;

  const taskCheckRunList = task.taskCheckRunList.filter(
    (check) => !HiddenCheckTypes.has(check.type)
  );

  const listGroupByType = groupBy(
    taskCheckRunList,
    (checkRun) => checkRun.type
  );

  const latestCheckRunOfEachType = Object.keys(listGroupByType).map((type) => {
    const listOfType = listGroupByType[type];
    const latest = maxBy(listOfType, (checkRun) => checkRun.updatedTs)!;
    return latest;
  });

  for (const checkRun of latestCheckRunOfEachType) {
    switch (checkRun.status) {
      case "CANCELED":
        // nothing todo
        break;
      case "FAILED":
        summary.errorCount++;
        break;
      case "RUNNING":
        summary.runningCount++;
        break;
      case "DONE":
        for (const result of checkRun.result.resultList ?? []) {
          switch (result.status) {
            case "SUCCESS":
              summary.successCount++;
              break;
            case "WARN":
              summary.warnCount++;
              break;
            case "ERROR":
              summary.errorCount++;
          }
        }
    }
  }

  return summary;
}

export const isTaskSkipped = (task: Task): boolean => {
  if (task.status === "DONE") {
    const payload = task.payload as any;
    return payload?.skipped === true;
  }
  return false;
};

export const isTaskTriggeredByVCS = (task: Task): boolean => {
  const taskTypesWithPushEvent: TaskType[] = [
    "bb.task.database.data.update",
    "bb.task.database.schema.update",
    "bb.task.database.schema.update-sdl",
    "bb.task.database.schema.update.ghost.sync",
    "bb.task.database.schema.baseline",
  ];

  type PayloadTypesWithPushEvent =
    | TaskDatabaseDataUpdatePayload
    | TaskDatabaseSchemaUpdatePayload
    | TaskDatabaseSchemaUpdateSDLPayload
    | TaskDatabaseSchemaUpdateGhostSyncPayload
    | TaskDatabaseSchemaBaselinePayload;

  if (taskTypesWithPushEvent.includes(task.type)) {
    const payload = task.payload as PayloadTypesWithPushEvent | undefined;

    if (payload && payload.pushEvent) {
      return true;
    }
  }

  return false;
};

export const isTaskEntity = (task: Task | TaskCreate): task is Task => {
  const obj = task as any;
  return typeof obj["id"] === "number";
};

export const isTaskCreate = (task: Task | TaskCreate): task is TaskCreate => {
  return !isTaskEntity(task);
};

/**
 *
 * @param task
 * @param issue
 * @param activeOnly if true, only "Active Task" can be skipped
 * @returns
 */
export const canSkipTask = (
  task: Task,
  issue: Issue,
  activeOnly = false,
  failedOnly = false
) => {
  const pipeline = issue.pipeline;
  const isActiveTask = task.id === activeTask(pipeline!).id;
  if (activeOnly && !isActiveTask) {
    return false;
  }

  const applicableStatusList: TaskStatus[] = failedOnly
    ? ["FAILED"]
    : ["PENDING_APPROVAL", "FAILED"];

  if (!applicableStatusList.includes(task.status)) {
    return false;
  }

  const currentUserV1 = useCurrentUserV1();

  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-issue",
      currentUserV1.value.userRole
    )
  ) {
    return true;
  }

  if (extractUserUID(currentUserV1.value.name) === String(issue.assignee.id)) {
    return true;
  }

  return false;
};

export const checkStatusOfTask = (task: Task): TaskCheckStatus | undefined => {
  if (task.status === "PENDING" || task.status === "PENDING_APPROVAL") {
    const summary = taskCheckRunSummary(task);
    if (summary.errorCount > 0) return "ERROR";
    if (summary.warnCount > 0) return "WARN";
  }
  return undefined;
};
