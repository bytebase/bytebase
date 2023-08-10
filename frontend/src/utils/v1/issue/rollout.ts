import { last } from "lodash-es";
import { stringify } from "qs";
import slug from "slug";
import {
  EMPTY_TASK_NAME,
  emptyStage,
  emptyTask,
  unknownTask,
  unknownStage,
  ComposedIssue,
} from "@/types";
import {
  Rollout,
  Stage,
  Task,
  Task_Status,
} from "@/types/proto/v1/rollout_service";
import { flattenTaskV1List, issueV1Slug } from "./issue";

export const extractRolloutUID = (name: string) => {
  const pattern = /(?:^|\/)rollouts\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const extractTaskUID = (name: string) => {
  const pattern = /(?:^|\/)tasks\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const stageV1Slug = (stage: Stage): string => {
  return [slug(stage.title), stage.uid].join("-");
};

export const taskV1Slug = (task: Task): string => {
  return [slug(task.title), task.uid].join("-");
};

export const activeTaskInTaskList = (tasks: Task[]): Task => {
  for (const task of tasks) {
    if (
      task.status === Task_Status.NOT_STARTED ||
      task.status === Task_Status.PENDING ||
      task.status === Task_Status.RUNNING ||
      // "FAILED" is also a transient task status, which requires user
      // to take further action (e.g. Skip, Retry)
      task.status === Task_Status.FAILED ||
      // "CANCELED" tasks can be "RESTART"ed
      // So it should be an "active" task
      task.status === Task_Status.CANCELED
    ) {
      return task;
    }
  }

  return last(tasks) ?? emptyTask();
};

export const activeTaskInStageV1 = (stage: Stage): Task => {
  return activeTaskInTaskList(stage.tasks);
};

export const activeTaskInRollout = (rollout: Rollout): Task => {
  return activeTaskInTaskList(flattenTaskV1List(rollout));
};

export const activeStageInRollout = (rollout: Rollout): Stage => {
  const activeTask = activeTaskInRollout(rollout);
  if (activeTask.name !== EMPTY_TASK_NAME) {
    const stage = rollout.stages.find((stage) =>
      stage.tasks.includes(activeTask)
    );
    if (stage) {
      return stage;
    }
  }
  return emptyStage();
};

export const findTaskByUID = (
  rollout: Rollout | undefined,
  uid: string
): Task => {
  for (const stage of rollout?.stages ?? []) {
    for (const task of stage.tasks) {
      if (task.uid == uid) {
        return task;
      }
    }
  }
  return unknownTask();
};

export const findStageByUID = (
  rollout: Rollout | undefined,
  uid: string
): Stage => {
  return (rollout?.stages ?? []).find((s) => s.uid === uid) ?? unknownStage();
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
  }
};

export const buildIssueV1LinkWithTask = (
  issue: ComposedIssue,
  task: Task,
  simple = false
) => {
  const stage = issue.rolloutEntity.stages.find(
    (s) => s.tasks.findIndex((t) => t.uid === task.uid) >= 0
  );

  const issueSlug = simple ? issue.uid : issueV1Slug(issue);
  const query: Record<string, string> = {};
  if (stage) {
    query.stage = simple ? stage.uid : stageV1Slug(stage);
  }
  query.task = simple ? task.uid : taskV1Slug(task);

  const querystring = stringify(query);
  const url = `/issue/${issueSlug}?${querystring}`;

  return url;
};
