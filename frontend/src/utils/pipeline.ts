import {
  Database,
  empty,
  EMPTY_ID,
  Environment,
  Pipeline,
  Task,
} from "../types";

export function activeTask(pipeline: Pipeline): Task {
  for (const task of pipeline.taskList) {
    if (
      task.status === "PENDING" ||
      task.status === "RUNNING" ||
      // "FAILED" is also a transient task status, which requires user
      // to take further action (e.g. Cancel, Skip, Retry)
      task.status === "FAILED"
    ) {
      return task;
    }
  }
  return empty("TASK") as Task;
}

export function activeTaskIsRunning(pipeline: Pipeline): boolean {
  return activeTask(pipeline).status === "RUNNING";
}

export function activeEnvironment(pipeline: Pipeline): Environment {
  const task: Task = activeTask(pipeline);
  if (task.id == EMPTY_ID) {
    return empty("ENVIRONMENT") as Environment;
  }
  return task.environment;
}

export function activeDatabase(pipeline: Pipeline): Database {
  const task = activeTask(pipeline);
  if (task.id == EMPTY_ID) {
    return empty("DATABASE") as Database;
  }
  return task.database;
}
