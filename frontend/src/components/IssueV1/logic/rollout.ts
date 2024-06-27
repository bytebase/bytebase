import { t } from "@/plugins/i18n";
import type { ComposedIssue } from "@/types";
import type { User } from "@/types/proto/v1/auth_service";
import { IssueStatus, Issue_Type } from "@/types/proto/v1/issue_service";
import { TenantMode } from "@/types/proto/v1/project_service";
import type { Task } from "@/types/proto/v1/rollout_service";
import {
  Task_Status,
  task_StatusToJSON,
  Task_Type,
} from "@/types/proto/v1/rollout_service";
import {
  extractDatabaseGroupName,
  extractDeploymentConfigName,
  extractUserResourceName,
  flattenTaskV1List,
  hasProjectPermissionV2,
} from "@/utils";
import { planCheckRunSummaryForTask, specForTask } from ".";

export const isGroupingChangeTaskV1 = (issue: ComposedIssue, task: Task) => {
  const spec = specForTask(issue.planEntity, task);
  if (!spec) {
    return false; // Not sure actually, but doesn't matter.
  }
  const databaseGroup = extractDatabaseGroupName(
    spec.changeDatabaseConfig?.target ?? ""
  );
  return databaseGroup !== "";
};

export const isDeploymentConfigChangeTaskV1 = (
  issue: ComposedIssue,
  task: Task
) => {
  const spec = specForTask(issue.planEntity, task);
  if (!spec) {
    return false;
  }
  const deploymentConfig = extractDeploymentConfigName(
    spec.changeDatabaseConfig?.target ?? ""
  );
  return deploymentConfig !== "";
};

export const allowUserToEditStatementForTask = (
  issue: ComposedIssue,
  task: Task,
  user: User
): string[] => {
  const denyReasons: string[] = [];

  if (isTaskV1TriggeredByVCS(issue, task)) {
    // If an issue is triggered by VCS, its creator will be 1 (SYSTEM_BOT_ID)
    // We should "Allow" current user to edit the statement (via VCS).
    return [];
  }

  if (
    issue.type !== Issue_Type.DATABASE_CHANGE &&
    issue.type !== Issue_Type.DATABASE_DATA_EXPORT
  ) {
    denyReasons.push("Only database related issue type can be changed");
  }
  if (issue.status !== IssueStatus.OPEN) {
    denyReasons.push("The issue is not open");
  }
  if (!issue.projectEntity.allowModifyStatement) {
    denyReasons.push("Cannot edit statement after issue created");
  }

  if (task.type === Task_Type.DATABASE_CREATE) {
    // For standard mode projects, we are not allowed to edit the database
    // creation SQL statement.
    if (issue.projectEntity.tenantMode !== TenantMode.TENANT_MODE_ENABLED) {
      denyReasons.push("Cannot edit database creation statement");
    }

    // We allow to edit create database statement for tenant project to give users a
    // chance to edit the dumped schema from its peer databases, because the dumped schema
    // may not be perfectly correct.
    // So we fallthrough to the common checkpoints.
  }

  if (issue.projectEntity.tenantMode === TenantMode.TENANT_MODE_ENABLED) {
    const tasks = flattenTaskV1List(issue.rolloutEntity);
    if (!tasks.every((task) => isTaskEditable(issue, task).length === 0)) {
      denyReasons.push("Some of the tasks are not editable in batch mode");
    }
  }

  // if not creating, we are allowed to edit sql statement only when:
  // - user is the creator
  // - OR user has plans.update permission in the project

  denyReasons.push(...isTaskEditable(issue, task));

  if (extractUserResourceName(issue.creator) !== user.email) {
    if (!hasProjectPermissionV2(issue.projectEntity, user, "bb.plans.update")) {
      denyReasons.push(
        t("issue.error.you-don-have-privilege-to-edit-this-issue")
      );
    }
  }
  return denyReasons;
};

export const isTaskEditable = (issue: ComposedIssue, task: Task): string[] => {
  if (
    task.status === Task_Status.NOT_STARTED ||
    task.status === Task_Status.FAILED ||
    task.status === Task_Status.CANCELED
  ) {
    return [];
  }
  if (task.status === Task_Status.PENDING) {
    // If a task's status is "PENDING", its statement is editable if there
    // are at least ONE ERROR task checks.
    // Since once all its task checks are fulfilled, it might be queued by
    // the scheduler.
    // Editing a queued task's SQL statement is dangerous with kinds of race
    // condition risks.
    const summary = planCheckRunSummaryForTask(issue, task);
    if (summary.errorCount > 0) {
      return [];
    }
    return [`Task is pending to run`];
  }

  return [`${task_StatusToJSON(task.status)} task is not editable`];
};

export const isTaskFinished = (task: Task): boolean => {
  return [Task_Status.DONE, Task_Status.SKIPPED].includes(task.status);
};

export const isTaskV1TriggeredByVCS = (
  issue: ComposedIssue,
  task: Task
): boolean => {
  return false; // TODO
};

export const semanticTaskType = (type: Task_Type) => {
  switch (type) {
    case Task_Type.DATABASE_CREATE:
      return t("db.create");
    case Task_Type.DATABASE_DATA_UPDATE:
      return "DML";
    case Task_Type.DATABASE_SCHEMA_BASELINE:
      return t("common.baseline");
    case Task_Type.DATABASE_SCHEMA_UPDATE:
    case Task_Type.DATABASE_SCHEMA_UPDATE_SDL:
      return "DDL";
    case Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_SYNC:
      return `gh-ost ${t(
        "task.type.bb-task-database-schema-update-ghost-sync"
      )}`;
    case Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER:
      return `gh-ost ${t(
        "task.type.bb-task-database-schema-update-ghost-cutover"
      )}`;
  }
  return "";
};
