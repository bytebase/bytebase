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
