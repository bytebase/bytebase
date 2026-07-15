import i18n from "@/react/i18n";
import { getDatabaseByName } from "@/react/stores/app/databaseAccess";
import {
  getTimeForPbTimestampProtoEs,
  isValidDatabaseName,
  UNKNOWN_ID,
  unknownDatabase,
} from "@/types";
import type {
  Plan,
  Plan_TaskStatusCount,
} from "@/types/proto-es/v1/plan_service_pb";
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

// Canonical task status priority for aggregation and display. Failure outranks
// active work (a failed task needs attention even while siblings run — see
// BYT-9822); a cancel is deliberate rather than an error, so it stays below
// active statuses but above NOT_STARTED, surfacing once nothing is in motion.
export const TASK_STATUS_PRIORITY: readonly Task_Status[] = [
  Task_Status.FAILED,
  Task_Status.RUNNING,
  Task_Status.PENDING,
  Task_Status.CANCELED,
  Task_Status.NOT_STARTED,
  Task_Status.DONE,
  Task_Status.SKIPPED,
];

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

// A task is "actively transitioning" — worth fast-polling — when it is RUNNING,
// or PENDING and expected to start imminently. A PENDING task scheduled for a
// future run_time (a maintenance window, possibly hours out) is NOT imminent, so
// it's excluded: the deploy view backs off instead of hammering the rollout RPCs
// while it waits. `nowMs` is injected for testability.
export const isTaskActivelyTransitioning = (
  task: Task,
  nowMs: number = Date.now()
): boolean => {
  if (task.status === Task_Status.RUNNING) {
    return true;
  }
  if (task.status !== Task_Status.PENDING) {
    return false;
  }
  return !task.runTime || getTimeForPbTimestampProtoEs(task.runTime) <= nowMs;
};

// Task_Status ONLY. Task_Status and TaskRun_Status share names but their
// numeric values are offset by one (Task_Status.PENDING === 2 ===
// TaskRun_Status.RUNNING), so one switch cannot serve both enums — a task
// run's status must go through stringifyTaskRunStatus below.
export const stringifyTaskStatus = (
  status: Task_Status,
  translate: (
    key: string,
    named?: Record<string, unknown>
  ) => string = i18n.t.bind(i18n)
): string => {
  switch (status) {
    case Task_Status.NOT_STARTED:
      return translate("task.status.not-started");
    case Task_Status.PENDING:
      return translate("task.status.pending");
    case Task_Status.RUNNING:
      return translate("task.status.running");
    case Task_Status.DONE:
      return translate("task.status.done");
    case Task_Status.FAILED:
      return translate("task.status.failed");
    case Task_Status.CANCELED:
      return translate("task.status.canceled");
    case Task_Status.SKIPPED:
      return translate("task.status.skipped");
    default:
      return Task_Status[status] || String(status);
  }
};

export const stringifyTaskRunStatus = (
  status: TaskRun_Status,
  translate: (
    key: string,
    named?: Record<string, unknown>
  ) => string = i18n.t.bind(i18n)
): string => {
  switch (status) {
    case TaskRun_Status.PENDING:
      return translate("task.status.pending");
    case TaskRun_Status.RUNNING:
      return translate("task.status.running");
    case TaskRun_Status.DONE:
      return translate("task.status.done");
    case TaskRun_Status.FAILED:
      return translate("task.status.failed");
    case TaskRun_Status.CANCELED:
      return translate("task.status.canceled");
    case TaskRun_Status.AVAILABLE:
      return translate("task.status.available");
    default:
      return TaskRun_Status[status] || String(status);
  }
};

// Return the highest-priority Task_Status (per TASK_STATUS_PRIORITY)
// for which `has` reports a member, or `fallback` when none match. Shared by the
// stage and rollout status reducers below.
const foldByStatusPriority = (
  has: (status: Task_Status) => boolean,
  fallback: Task_Status
): Task_Status => {
  for (const status of TASK_STATUS_PRIORITY) {
    if (has(status)) return status;
  }
  return fallback;
};

export const getStageStatus = (stage: Stage): Task_Status => {
  if (stage.tasks.length === 0) return Task_Status.NOT_STARTED;
  return foldByStatusPriority(
    (status) => stage.tasks.some((task) => task.status === status),
    Task_Status.NOT_STARTED
  );
};

// Derive a stage status from aggregated task status counts (used by the plan
// table, which only has counts rather than the full task list).
export const getStageStatusFromCounts = (
  counts: Plan_TaskStatusCount[]
): Task_Status =>
  foldByStatusPriority(
    (status) => counts.some((item) => item.status === status),
    Task_Status.STATUS_UNSPECIFIED
  );

export const getRolloutStatus = (rollout: Rollout): Task_Status => {
  if (rollout.stages.length === 0) return Task_Status.NOT_STARTED;
  return foldByStatusPriority(
    (status) =>
      rollout.stages.some((stage) => getStageStatus(stage) === status),
    Task_Status.NOT_STARTED
  );
};

export const databaseForTask = (project: Project, task: Task, plan?: Plan) => {
  switch (task.type) {
    case Task_Type.DATABASE_CREATE:
      // The database is not created yet.
      // extract database info from the task's and payload's properties.
      return extractCoreDatabaseInfoFromDatabaseCreateTask(project, task, plan);
    case Task_Type.DATABASE_MIGRATE:
    case Task_Type.DATABASE_EXPORT:
      const db = getDatabaseByName(task.target);
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
