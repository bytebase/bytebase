import { last } from "lodash-es";
import { stringify } from "qs";
import slug from "slug";
import type { ComposedIssue } from "@/types";
import {
  EMPTY_TASK_NAME,
  emptyStage,
  emptyTask,
  unknownTask,
  unknownStage,
  EMPTY_ID,
  UNKNOWN_ID,
} from "@/types";
import type { Rollout, Stage, Task } from "@/types/proto/v1/rollout_service";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import { extractProjectResourceName } from "../project";
import { extractIssueUID, flattenTaskV1List, issueV1Slug } from "./issue";

export const extractRolloutUID = (name: string) => {
  const pattern = /(?:^|\/)rollouts\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const extractStageUID = (name: string) => {
  const pattern = /(?:^|\/)stages\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const isValidStageName = (name: string | undefined) => {
  if (!name) {
    return false;
  }
  const stageUID = extractStageUID(name);
  return (
    stageUID && stageUID !== String(EMPTY_ID) && stageUID !== String(UNKNOWN_ID)
  );
};

export const extractTaskUID = (name: string) => {
  const pattern = /(?:^|\/)tasks\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const isValidTaskName = (name: string | undefined) => {
  if (!name) {
    return false;
  }
  const taskUID = extractTaskUID(name);
  return (
    taskUID && taskUID !== String(EMPTY_ID) && taskUID !== String(UNKNOWN_ID)
  );
};

export const extractTaskRunUID = (name: string) => {
  const pattern = /(?:^|\/)taskRuns\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const stageV1Slug = (stage: Stage): string => {
  return [slug(stage.title), extractStageUID(stage.name)].join("-");
};

export const taskV1Slug = (task: Task): string => {
  return [slug(task.title), extractTaskUID(task.name)].join("-");
};

export const activeTaskInTaskList = (tasks: Task[]): Task => {
  // Focus on the running task first.
  const runningTask = tasks.find((task) => task.status === Task_Status.RUNNING);
  if (runningTask) {
    return runningTask;
  }

  const maybeActiveTask = tasks.find(
    (task) =>
      task.status === Task_Status.NOT_STARTED ||
      task.status === Task_Status.PENDING ||
      // "FAILED" is also a transient task status, which requires user
      // to take further action (e.g. Skip, Retry)
      task.status === Task_Status.FAILED ||
      // "CANCELED" tasks can be "RESTART"ed
      // So it should be an "active" task
      task.status === Task_Status.CANCELED ||
      task.status === Task_Status.STATUS_UNSPECIFIED // compatibility for preview phase
  );
  if (maybeActiveTask) {
    return maybeActiveTask;
  }

  // fallback
  return last(tasks) ?? emptyTask();
};

export const activeTaskInStageV1 = (stage: Stage): Task => {
  return activeTaskInTaskList(stage.tasks);
};

export const activeTaskInRollout = (rollout: Rollout | undefined): Task => {
  if (!rollout) {
    return emptyTask();
  }
  return activeTaskInTaskList(flattenTaskV1List(rollout));
};

export const activeStageInRollout = (rollout: Rollout | undefined): Stage => {
  const activeTask = activeTaskInRollout(rollout);
  if (activeTask.name !== EMPTY_TASK_NAME) {
    const stage = rollout?.stages.find((stage) =>
      stage.tasks.includes(activeTask)
    );
    if (stage) {
      return stage;
    }
  }
  return emptyStage();
};

export const findTaskByName = (
  rollout: Rollout | undefined,
  name: string
): Task => {
  for (const stage of rollout?.stages ?? []) {
    for (const task of stage.tasks) {
      if (task.name == name) {
        return task;
      }
    }
  }
  return unknownTask();
};

export const findStageByName = (
  rollout: Rollout | undefined,
  name: string
): Stage => {
  return (rollout?.stages ?? []).find((s) => s.name === name) ?? unknownStage();
};

export const extractSchemaVersionFromTask = (task: Task): string => {
  // The schema version is specified in the filename
  // parsed and stored to the payload.schemaVersion
  // fallback to empty if we can't read the field.
  return (
    task.databaseDataUpdate?.schemaVersion ??
    task.databaseSchemaBaseline?.schemaVersion ??
    task.databaseSchemaUpdate?.schemaVersion ??
    ""
  );
};

export const sheetNameOfTaskV1 = (task: Task): string => {
  return (
    task.databaseCreate?.sheet ??
    task.databaseDataUpdate?.sheet ??
    task.databaseSchemaUpdate?.sheet ??
    task.databaseDataExport?.sheet ??
    ""
  );
};

export const setSheetNameForTask = (task: Task, sheetName: string) => {
  if (task.databaseCreate) {
    task.databaseCreate.sheet = sheetName;
  } else if (task.databaseDataUpdate) {
    task.databaseDataUpdate.sheet = sheetName;
  } else if (task.databaseSchemaUpdate) {
    task.databaseSchemaUpdate.sheet = sheetName;
  } else if (task.databaseDataExport) {
    task.databaseDataExport.sheet = sheetName;
  }
};

export const buildIssueV1LinkWithTask = (
  issue: ComposedIssue,
  task: Task,
  simple = false
) => {
  const stage = issue.rolloutEntity?.stages.find(
    (s) => s.tasks.findIndex((t) => t.name === task.name) >= 0
  );

  const projectId = extractProjectResourceName(issue.project);
  const issueSlug = simple ? extractIssueUID(issue.name) : issueV1Slug(issue);
  const query: Record<string, string> = {};
  if (stage) {
    query.stage = simple ? extractStageUID(stage.name) : stageV1Slug(stage);
  }
  query.task = simple ? extractTaskUID(task.name) : taskV1Slug(task);

  const querystring = stringify(query);
  const url = `/projects/${projectId}/issues/${issueSlug}?${querystring}`;

  return url;
};
