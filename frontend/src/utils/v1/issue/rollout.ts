import { last } from "lodash-es";
import { stringify } from "qs";
import { useI18n } from "vue-i18n";
import { extractCoreDatabaseInfoFromDatabaseCreateTask } from "@/components/IssueV1";
import { mockDatabase } from "@/components/IssueV1/logic/utils";
import { useDatabaseV1Store } from "@/store";
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
import {
  isValidDatabaseName,
  unknownDatabase,
  type ComposedProject,
} from "@/types";
import {
  Task_Type,
  TaskRun_Status,
  type Rollout,
  type Stage,
  type Task,
} from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
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
  // Stage UID is now the environment ID
  return extractStageUID(stage.name);
};

export const taskV1Slug = (task: Task): string => {
  return extractTaskUID(task.name);
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
  if (task.payload?.case === "databaseDataUpdate") {
    return task.payload.value.schemaVersion ?? "";
  }
  if (task.payload?.case === "databaseSchemaUpdate") {
    return task.payload.value.schemaVersion ?? "";
  }
  return "";
};

export const sheetNameOfTaskV1 = (task: Task): string => {
  if (task.payload?.case === "databaseCreate") {
    return task.payload.value.sheet ?? "";
  }
  if (task.payload?.case === "databaseDataUpdate") {
    return task.payload.value.sheet ?? "";
  }
  if (task.payload?.case === "databaseSchemaUpdate") {
    return task.payload.value.sheet ?? "";
  }
  if (task.payload?.case === "databaseDataExport") {
    return task.payload.value.sheet ?? "";
  }
  return "";
};

export const setSheetNameForTask = (task: Task, sheetName: string) => {
  if (task.payload?.case === "databaseCreate") {
    task.payload.value.sheet = sheetName;
  } else if (task.payload?.case === "databaseDataUpdate") {
    task.payload.value.sheet = sheetName;
  } else if (task.payload?.case === "databaseSchemaUpdate") {
    task.payload.value.sheet = sheetName;
  } else if (task.payload?.case === "databaseDataExport") {
    task.payload.value.sheet = sheetName;
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

  const projectId = extractProjectResourceName(issue.name);
  const issueSlug = simple
    ? extractIssueUID(issue.name)
    : issueV1Slug(issue.name, issue.title);
  const query: Record<string, string> = {};
  if (stage) {
    // Stage UID is now always the environment ID
    query.stage = extractStageUID(stage.name);
  }
  query.task = simple ? extractTaskUID(task.name) : taskV1Slug(task);

  const querystring = stringify(query);
  const url = `/projects/${projectId}/issues/${issueSlug}?${querystring}`;

  return url;
};

export const stringifyTaskStatus = (
  status: Task_Status | TaskRun_Status
): string => {
  const { t } = useI18n();
  switch (status) {
    case Task_Status.NOT_STARTED:
      return t("task.status.not-started");
    case Task_Status.PENDING:
      return t("task.status.pending");
    case Task_Status.RUNNING:
      return t("task.status.running");
    case Task_Status.DONE:
      return t("task.status.done");
    case Task_Status.FAILED:
      return t("task.status.failed");
    case Task_Status.CANCELED:
      return t("task.status.canceled");
    case Task_Status.SKIPPED:
      return t("task.status.skipped");
    default:
      return Task_Status[status] || String(status);
  }
};

export const getStageStatus = (stage: Stage): Task_Status => {
  const tasks = stage.tasks;
  if (tasks.length === 0) return Task_Status.NOT_STARTED;

  // Priority order for stage status:
  // 1. Failed - if any task failed
  // 2. Canceled - if any task canceled (and none failed)
  // 3. Running - if any task is running
  // 4. Pending - if any task is pending
  // 5. Not Started - if all tasks are not started
  // 6. Skipped - if all tasks are skipped
  // 7. Done - if all tasks are done

  // Check for any failed tasks
  if (tasks.some((task) => task.status === Task_Status.FAILED)) {
    return Task_Status.FAILED;
  }

  // Check for any canceled tasks
  if (tasks.some((task) => task.status === Task_Status.CANCELED)) {
    return Task_Status.CANCELED;
  }

  // Check for any running tasks
  if (tasks.some((task) => task.status === Task_Status.RUNNING)) {
    return Task_Status.RUNNING;
  }

  // Check for any pending tasks
  if (tasks.some((task) => task.status === Task_Status.PENDING)) {
    return Task_Status.PENDING;
  }

  // Check if all tasks are done
  if (tasks.every((task) => task.status === Task_Status.DONE)) {
    return Task_Status.DONE;
  }

  // Check if all tasks are skipped
  if (tasks.every((task) => task.status === Task_Status.SKIPPED)) {
    return Task_Status.SKIPPED;
  }

  // Default to not started
  return Task_Status.NOT_STARTED;
};

export const databaseForTask = (project: ComposedProject, task: Task) => {
  switch (task.type) {
    case Task_Type.DATABASE_CREATE:
      // The database is not created yet.
      // extract database info from the task's and payload's properties.
      return extractCoreDatabaseInfoFromDatabaseCreateTask(project, task);
    case Task_Type.DATABASE_SCHEMA_UPDATE:
    case Task_Type.DATABASE_SCHEMA_UPDATE_SDL:
    case Task_Type.DATABASE_SCHEMA_UPDATE_GHOST:
    case Task_Type.DATABASE_DATA_UPDATE:
    case Task_Type.DATABASE_EXPORT:
      const db = useDatabaseV1Store().getDatabaseByName(task.target);
      if (!isValidDatabaseName(db.name)) {
        return mockDatabase(project, task.target);
      }
      return db;
    default:
      return unknownDatabase();
  }
};
