import { ComposedIssue } from "@/types";
import { Task } from "@/types/proto/v1/rollout_service";
import { extractTaskUID } from "@/utils";

export const taskRunListForTask = (issue: ComposedIssue, task: Task) => {
  return issue.rolloutTaskRunList.filter(
    (taskRun) => extractTaskUID(taskRun.name) === task.uid
  );
};
