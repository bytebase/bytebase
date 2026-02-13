import { useI18n } from "vue-i18n";
import { TASK_STATUS_FILTERS } from "@/components/RolloutV1/constants/task";
import { useDatabaseV1Store } from "@/store";
import { isValidDatabaseName, UNKNOWN_ID, unknownDatabase } from "@/types";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  type Rollout,
  type Stage,
  type Task,
  Task_Status,
  Task_Type,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractCoreDatabaseInfoFromDatabaseCreateTask,
  mockDatabase,
} from "./issue";

export const extractPlanUIDFromRolloutName = (name: string) => {
  const pattern = /(?:^|\/)plans\/([^/]+)\/rollout(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const extractPlanNameFromRolloutName = (name: string) => {
  // Rollout name format: projects/{project}/plans/{plan}/rollout
  // Returns: projects/{project}/plans/{plan}
  const pattern = /^(.+\/plans\/[^/]+)\/rollout(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const extractStageUID = (name: string) => {
  const pattern = /(?:^|\/)stages\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
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
  return taskUID && taskUID !== String(UNKNOWN_ID);
};

export const extractTaskRunUID = (name: string) => {
  const pattern = /(?:^|\/)taskRuns\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

/**
 * Extracts the rollout resource name from a task run resource name.
 * Task run name format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun}
 * Returns: projects/{project}/plans/{plan}/rollout
 */
export const extractRolloutNameFromTaskRunName = (
  taskRunName: string
): string => {
  const pattern = /^(projects\/[^/]+\/plans\/[^/]+\/rollout)\/stages\//;
  const matches = taskRunName.match(pattern);
  return matches?.[1] ?? "";
};

/**
 * Extracts the task resource name from a task run resource name.
 * Task run name format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun}
 * Returns: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}
 */
export const extractTaskNameFromTaskRunName = (taskRunName: string): string => {
  const idx = taskRunName.lastIndexOf("/taskRuns/");
  if (idx === -1) return "";
  return taskRunName.substring(0, idx);
};

/**
 * Extracts the stage resource name from a task resource name.
 * Task name format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}
 * Returns: projects/{project}/plans/{plan}/rollout/stages/{stage}
 */
export const extractStageNameFromTaskName = (taskName: string): string => {
  const pattern = /^(.+\/stages\/[^/]+)\/tasks\/[^/]+$/;
  const matches = taskName.match(pattern);
  return matches?.[1] ?? "";
};

export const sheetNameOfTaskV1 = (task: Task): string => {
  if (task.payload?.case === "databaseCreate") {
    return task.payload.value.sheet ?? "";
  }
  if (task.payload?.case === "databaseUpdate") {
    // Task.DatabaseUpdate now uses oneof source { sheet | release }
    if (task.payload.value.source.case === "sheet") {
      return task.payload.value.source.value ?? "";
    }
    return "";
  }
  if (task.payload?.case === "databaseDataExport") {
    return task.payload.value.sheet ?? "";
  }
  return "";
};

export const releaseNameOfTaskV1 = (task: Task): string => {
  if (task.payload?.case === "databaseUpdate") {
    // Task.DatabaseUpdate now uses oneof source { sheet | release }
    if (task.payload.value.source.case === "release") {
      return task.payload.value.source.value ?? "";
    }
  }
  return "";
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
    case TaskRun_Status.AVAILABLE:
      return t("task.status.available");
    default:
      return Task_Status[status] || String(status);
  }
};

export const getStageStatus = (stage: Stage): Task_Status => {
  const tasks = stage.tasks;
  if (tasks.length === 0) return Task_Status.NOT_STARTED;

  // Priority order follows TASK_STATUS_FILTERS
  for (const status of TASK_STATUS_FILTERS) {
    if (tasks.some((task) => task.status === status)) {
      return status;
    }
  }

  return Task_Status.NOT_STARTED;
};

export const getRolloutStatus = (rollout: Rollout): Task_Status => {
  const stages = rollout.stages;
  if (stages.length === 0) return Task_Status.NOT_STARTED;

  for (const status of TASK_STATUS_FILTERS) {
    if (stages.some((stage) => getStageStatus(stage) === status)) {
      return status;
    }
  }

  return Task_Status.NOT_STARTED;
};

export const databaseForTask = (project: Project, task: Task, plan?: Plan) => {
  switch (task.type) {
    case Task_Type.DATABASE_CREATE:
      // The database is not created yet.
      // extract database info from the task's and payload's properties.
      return extractCoreDatabaseInfoFromDatabaseCreateTask(project, task, plan);
    case Task_Type.DATABASE_MIGRATE:
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

export const isReleaseBasedTask = (task: Task): boolean => {
  if (task.payload?.case === "databaseUpdate") {
    return task.payload.value.source?.case === "release";
  }

  return false;
};
