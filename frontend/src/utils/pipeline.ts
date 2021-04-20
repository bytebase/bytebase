import {
  DatabaseId,
  EMPTY_ID,
  EnvironmentId,
  FINAL_TASK,
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
  return FINAL_TASK;
}

export function activeTaskIsRunning(pipeline: Pipeline): boolean {
  return activeTask(pipeline).status === "RUNNING";
}

export function activeEnvironmentId(pipeline: Pipeline): EnvironmentId {
  const task: Task = activeTask(pipeline);
  if (task.id == EMPTY_ID) {
    return EMPTY_ID;
  }
  return task.database.instance.environment.id;
}

export function activeDatabaseId(pipeline: Pipeline): DatabaseId {
  const task = activeTask(pipeline);
  if (task.id == EMPTY_ID) {
    return EMPTY_ID;
  }
  return task.database.id;
}
