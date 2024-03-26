import { first, orderBy } from "lodash-es";
import type { ComposedIssue } from "@/types";
import type { Task } from "@/types/proto/v1/rollout_service";
import { extractTaskUID } from "@/utils";

export const taskRunListForTask = (issue: ComposedIssue, task: Task) => {
  return issue.rolloutTaskRunList.filter(
    (taskRun) => extractTaskUID(taskRun.name) === task.uid
  );
};

export const latestTaskRunForTask = (issue: ComposedIssue, task: Task) => {
  const taskRunList = taskRunListForTask(issue, task);
  return first(orderBy(taskRunList, (taskRun) => Number(taskRun.uid), "desc"));
};
