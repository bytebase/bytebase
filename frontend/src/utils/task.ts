import { groupBy, maxBy } from "lodash-es";
import { useDatabaseV1Store } from "@/store";
import type {
  Task,
  TaskCheckStatus,
  TaskCheckType,
  TaskCreate,
  TaskDatabaseCreatePayload,
} from "@/types";
import { unknownDatabase } from "@/types";

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
  // The task entity is related to a database entity
  if (taskEntity.database) {
    return taskEntity.database.name;
  }

  // The task entity is irrelative to any databases or the related
  // database is not created yet.
  if (taskType === "bb.task.database.create") {
    // The database is not created yet.
    // extract database info from the task's and payload's properties.
    const payload = taskEntity.payload as TaskDatabaseCreatePayload;
    return payload.databaseName;
  }

  // Fallback to <<Unknown database>>. Won't be happy to see it.
  return unknownDatabase().databaseName;
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

export const isTaskEntity = (task: Task | TaskCreate): task is Task => {
  const obj = task as any;
  return typeof obj["id"] === "number";
};

export const isTaskCreate = (task: Task | TaskCreate): task is TaskCreate => {
  return !isTaskEntity(task);
};

export const checkStatusOfTask = (task: Task): TaskCheckStatus | undefined => {
  if (task.status === "PENDING" || task.status === "PENDING_APPROVAL") {
    const summary = taskCheckRunSummary(task);
    if (summary.errorCount > 0) return "ERROR";
    if (summary.warnCount > 0) return "WARN";
  }
  return undefined;
};
