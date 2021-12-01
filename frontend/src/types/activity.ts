import { FieldId } from "../plugins";
import { ActivityId, ContainerId, IssueId, PrincipalId, TaskId } from "./id";
import { IssueStatus } from "./issue";
import { MemberStatus, RoleType } from "./member";
import { TaskStatus } from "./pipeline";
import { Principal } from "./principal";
import { VCSPushEvent } from "./vcs";

export type IssueActivityType =
  | "bb.issue.create"
  | "bb.issue.comment.create"
  | "bb.issue.field.update"
  | "bb.issue.status.update"
  | "bb.pipeline.task.status.update"
  | "bb.pipeline.task.file.commit";

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
      return "Create issue";
    case "bb.issue.comment.create":
      return "Create comment";
    case "bb.issue.field.update":
      return "Update issue field";
    case "bb.issue.status.update":
      return "Update issue status";
    case "bb.pipeline.task.status.update":
      return "Update issue task status";
    case "bb.pipeline.task.file.commit":
      return "Commit file";
    case "bb.member.create":
      return "Create member";
    case "bb.member.role.update":
      return "Update role";
    case "bb.member.activate":
      return "Activate member";
    case "bb.member.deactivate":
      return "Deactivate member";
    case "bb.project.repository.push":
      return "Repository push event";
    case "bb.project.database.transfer":
      return "Database transfer";
    case "bb.project.member.create":
      return "Add project member";
    case "bb.project.member.delete":
      return "Delete project member";
    case "bb.project.member.role.update":
      return "Change project member role";
  }
}

export type ActivityLevel = "INFO" | "WARN" | "ERROR";

export type ActivityIssueCreatePayload = {
  issueName: string;
  rollbackIssueId?: IssueId;
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
