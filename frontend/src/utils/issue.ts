import {
  Issue,
  TaskId,
  Task,
  EnvironmentId,
  DatabaseId,
  Step,
  FINAL_TASK,
  ZERO_ID,
} from "../types";

export function taskName(issue: Issue, taskId: TaskId): string {
  for (const task of issue.taskList) {
    if (task.id == taskId) {
      return task.name;
    }
  }
  return "<<Unknown task>>";
}

// Returns true if the active step is the last step in the entire issue
export function pendingResolve(issue: Issue): boolean {
  return activeTask(issue).type == "bytebase.task.final";
}

export function activeTask(issue: Issue): Task {
  for (const task of issue.taskList) {
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

export function activeTaskIsRunning(issue: Issue): boolean {
  return activeTask(issue).status === "RUNNING";
}

export function activeEnvironmentId(issue: Issue): EnvironmentId {
  const task: Task = activeTask(issue);
  if (task.id == ZERO_ID) {
    return ZERO_ID;
  }
  return task.database.instance.environment.id;
}

export function activeDatabaseId(issue: Issue): DatabaseId {
  const task = activeTask(issue);
  if (task.id == ZERO_ID) {
    return ZERO_ID;
  }
  return task.database.id;
}
