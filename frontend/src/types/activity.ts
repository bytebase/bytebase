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
  | "bb.pipeline.task.status.update";

export type MemberActivityType =
  | "bb.member.create"
  | "bb.member.role.update"
  | "bb.member.activate"
  | "bb.member.deactivate";

export type ProjectActivityType = "bb.project.repository.push";

export type ActivityType =
  | IssueActivityType
  | MemberActivityType
  | ProjectActivityType;

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
};

export type ActionPayloadType =
  | ActivityIssueCreatePayload
  | ActivityIssueCommentCreatePayload
  | ActivityIssueFieldUpdatePayload
  | ActivityIssueStatusUpdatePayload
  | ActivityTaskStatusUpdatePayload
  | ActivityMemberCreatePayload
  | ActivityMemberRoleUpdatePayload
  | ActivityMemberActivateDeactivatePayload
  | ActivityProjectRepositoryPushPayload;

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
