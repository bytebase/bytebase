import { Issue, TaskId } from "../types";
import { activeTask } from "./pipeline";

export function taskName(issue: Issue, taskId: TaskId): string {
  for (const task of issue.pipeline.taskList) {
    if (task.id == taskId) {
      return task.name;
    }
  }
  return "<<Unknown task>>";
}

// Returns true if the active step is the last step in the entire issue
export function pendingResolve(issue: Issue): boolean {
  return activeTask(issue.pipeline).type == "bytebase.task.final";
}
