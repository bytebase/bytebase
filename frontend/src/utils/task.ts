import { useDatabaseStore } from "@/store";
import {
  Task,
  TaskCreate,
  TaskDatabaseCreatePayload,
  TaskDatabasePITRRestorePayload,
  unknown,
} from "@/types";

export const extractDatabaseNameFromTask = (
  task: Task | TaskCreate
): string => {
  const taskEntity = task as Task;
  const taskCreate = task as TaskCreate;
  if (typeof taskEntity.id !== "number") {
    // The task is not created yet
    // Find the database by databaseId if possible
    if (taskCreate.databaseId) {
      return useDatabaseStore().getDatabaseById(taskCreate.databaseId!).name;
    }
    return unknown("DATABASE").name;
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
  return unknown("DATABASE").name;
};
