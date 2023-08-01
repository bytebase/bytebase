import { ComposedIssue } from "@/types";
import { User } from "@/types/proto/v1/auth_service";
import { IssueStatus, Issue_Type } from "@/types/proto/v1/issue_service";
import { TenantMode } from "@/types/proto/v1/project_service";
import {
  Task,
  Task_Status,
  task_StatusToJSON,
  Task_Type,
} from "@/types/proto/v1/rollout_service";
import {
  extractUserResourceName,
  flattenTaskV1List,
  hasWorkspacePermissionV1,
} from "@/utils";
import { planCheckRunSummaryForTask } from ".";

export const isGroupingChangeTaskV1 = (issue: ComposedIssue, task: Task) => {
  return false; // TODO
};

export const allowUserToEditStatementForTask = (
  issue: ComposedIssue,
  task: Task,
  user: User
): string[] => {
  const denyReasons: string[] = [];

  if (isGroupingChangeTaskV1(issue, task)) {
    denyReasons.push("Cannot edit database group change issue");
  }

  if (isTaskV1TriggeredByVCS(issue, task)) {
    // If an issue is triggered by VCS, its creator will be 1 (SYSTEM_BOT_ID)
    // We should "Allow" current user to edit the statement (via VCS).
    return [];
  }

  if (issue.type !== Issue_Type.DATABASE_CHANGE) {
    denyReasons.push("Only database change issue type can be changed");
  }
  if (issue.status !== IssueStatus.OPEN) {
    denyReasons.push("The issue is not open");
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
    if (!tasks.every((task) => isTaskEditable(issue, task))) {
      denyReasons.push("Some of the tasks are not editable in batch mode");
    }
  }

  // if not creating, we are allowed to edit sql statement only when:
  // - user is the creator
  // - OR user is Workspace DBA or Owner

  denyReasons.push(...isTaskEditable(issue, task));

  if (extractUserResourceName(issue.creator) !== user.email) {
    if (
      !hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-issue",
        user.userRole
      )
    ) {
      denyReasons.push("You don't have the privilege to edit this issue");
    }
  }
  return denyReasons;
};

export const isTaskEditable = (issue: ComposedIssue, task: Task): string[] => {
  if (
    task.status === Task_Status.PENDING_APPROVAL ||
    task.status === Task_Status.FAILED
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
  return [Task_Status.DONE, Task_Status.CANCELED, Task_Status.SKIPPED].includes(
    task.status
  );
};

export const isTaskV1TriggeredByVCS = (
  issue: ComposedIssue,
  task: Task
): boolean => {
  return false; // TODO
};
