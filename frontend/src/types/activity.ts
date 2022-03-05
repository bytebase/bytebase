import { FieldId } from "../plugins";
import { ActivityId, ContainerId, IssueId, PrincipalId, TaskId } from "./id";
import { IssueStatus } from "./issue";
import { MemberStatus, RoleType } from "./member";
import { TaskStatus } from "./pipeline";
import { Principal } from "./principal";
import { VCSPushEvent } from "./vcs";
import { t } from "../plugins/i18n";

export type IssueActivityType =
  | "bb.issue.create"
  | "bb.issue.comment.create"
  | "bb.issue.field.update"
  | "bb.issue.status.update"
  | "bb.pipeline.task.status.update"
  | "bb.pipeline.task.file.commit"
  | "bb.pipeline.task.statement.update"
  | "bb.pipeline.task.general.earliest-allowed-time.update";

export type MemberActivityType =
  | "bb.member.create"
  | "bb.member.role.update"
  | "bb.member.activate"
  | "bb.member.deactivate";

export type ProjectActivityType =
  | "bb.project.repository.push"
  | "bb.project.database.transfer"
  | "bb.project.member.create"
  | "bb.project.member.delete"
  | "bb.project.member.role.update";

export type ActivityType =
  | IssueActivityType
  | MemberActivityType
  | ProjectActivityType;

export function activityName(type: ActivityType): string {
  switch (type) {
    case "bb.issue.create":
      return t("activity.type.issue-create");
    case "bb.issue.comment.create":
      return t("activity.type.comment-create");
    case "bb.issue.field.update":
      return t("activity.type.issue-field-update");
    case "bb.issue.status.update":
      return t("activity.type.issue-status-update");
    case "bb.pipeline.task.status.update":
      return t("activity.type.pipeline-task-status-update");
    case "bb.pipeline.task.file.commit":
      return t("activity.type.pipeline-task-file-commit");
    case "bb.pipeline.task.statement.update":
      return t("activity.type.pipeline-task-statement-update");
    case "bb.pipeline.task.general.earliest-allowed-time.update":
      return t("activity.type.pipeline-task-earliest-allowed-time-update");
    case "bb.member.create":
      return t("activity.type.member-create");
    case "bb.member.role.update":
      return t("activity.type.member-role-update");
    case "bb.member.activate":
      return t("activity.type.member-activate");
    case "bb.member.deactivate":
      return t("activity.type.member-deactivate");
    case "bb.project.repository.push":
      return t("activity.type.project-repository-push");
    case "bb.project.database.transfer":
      return t("activity.type.project-database-transfer");
    case "bb.project.member.create":
      return t("activity.type.project-member-create");
    case "bb.project.member.delete":
      return t("activity.type.project-member-delete");
    case "bb.project.member.role.update":
      return t("activity.type.project-member-role-update");
  }
}

export type ActivityLevel = "INFO" | "WARN" | "ERROR";

export type ActivityIssueCreatePayload = {
  issueName: string;
};

export type ActivityIssueCommentCreatePayload = {
  issueName: string;
};

export type ActivityIssueFieldUpdatePayload = {
  fieldId: FieldId;
  oldValue?: string;
  newValue?: string;
  issueName: string;
};

export type ActivityIssueStatusUpdatePayload = {
  oldStatus: IssueStatus;
  newStatus: IssueStatus;
  issueName: string;
};

export type ActivityTaskStatusUpdatePayload = {
  taskId: TaskId;
  oldStatus: TaskStatus;
  newStatus: TaskStatus;
  issueName: string;
  taskName: string;
};

export type ActivityTaskFileCommitPayload = {
  taskId: TaskId;
  vcsInstanceUrl: string;
  repositoryFullPath: string;
  branch: string;
  filePath: string;
  commitId: string;
};

export type ActivityTaskStatementUpdatePayload = {
  taskId: TaskId;
  oldStatement: string;
  newStatement: string;
  issueName: string;
  taskName: string;
};

export type ActivityTaskEarliestAllowedTimeUpdatePayload = {
  taskId: TaskId;
  oldEarliestAllowedTs: number;
  newEarliestAllowedTs: number;
  issueName: string;
  taskName: string;
};

export type ActivityMemberCreatePayload = {
  principalId: PrincipalId;
  principalName: string;
  principalEmail: string;
  memberStatus: MemberStatus;
  role: RoleType;
};

export type ActivityMemberRoleUpdatePayload = {
  principalId: PrincipalId;
  principalName: string;
  principalEmail: string;
  oldRole: RoleType;
  newRole: RoleType;
};

export type ActivityMemberActivateDeactivatePayload = {
  principalId: PrincipalId;
  principalName: string;
  principalEmail: string;
  role: RoleType;
};

export type ActivityProjectRepositoryPushPayload = {
  pushEvent: VCSPushEvent;
  issueId?: number;
  issueName?: string;
};

export type ActivityProjectDatabaseTransferPayload = {
  databaseId: number;
  databaseName: string;
};

export type ActionPayloadType =
  | ActivityIssueCreatePayload
  | ActivityIssueCommentCreatePayload
  | ActivityIssueFieldUpdatePayload
  | ActivityIssueStatusUpdatePayload
  | ActivityTaskStatusUpdatePayload
  | ActivityTaskFileCommitPayload
  | ActivityTaskStatementUpdatePayload
  | ActivityTaskEarliestAllowedTimeUpdatePayload
  | ActivityMemberCreatePayload
  | ActivityMemberRoleUpdatePayload
  | ActivityMemberActivateDeactivatePayload
  | ActivityProjectRepositoryPushPayload
  | ActivityProjectDatabaseTransferPayload;

export type Activity = {
  id: ActivityId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  // The object where this activity belongs
  // e.g if type is "bb.issue.xxx", then this field refers to the corresponding issue's id.
  containerId: ContainerId;
  type: ActivityType;
  level: ActivityLevel;
  comment: string;
  payload?: ActionPayloadType;
};

export type ActivityCreate = {
  // Domain specific fields
  containerId: ContainerId;
  type: ActivityType;
  comment: string;
  payload?: ActionPayloadType;
};

export type ActivityPatch = {
  // Domain specific fields
  comment: string;
};
