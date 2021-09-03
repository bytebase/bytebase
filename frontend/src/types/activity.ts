import { FieldId } from "../plugins";
import { ActivityId, ContainerId, IssueId, PrincipalId, TaskId } from "./id";
import { IssueStatus } from "./issue";
import { MemberStatus, RoleType } from "./member";
import { TaskStatus } from "./pipeline";
import { Principal } from "./principal";

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

export type ActivityLevel = "INFO" | "WARNING" | "ERROR";

export type ActionIssueCreatePayload = {
  issueName: string;
  rollbackIssueId?: IssueId;
};

export type ActionIssueCommentCreatePayload = {
  issueName: string;
};

export type ActionIssueFieldUpdatePayload = {
  fieldId: FieldId;
  oldValue?: string;
  newValue?: string;
  issueName: string;
};

export type ActionIssueStatusUpdatePayload = {
  oldStatus: IssueStatus;
  newStatus: IssueStatus;
  issueName: string;
};

export type ActionTaskStatusUpdatePayload = {
  taskId: TaskId;
  oldStatus: TaskStatus;
  newStatus: TaskStatus;
  issueName: string;
  taskName: string;
};

export type ActionMemberCreatePayload = {
  principalId: PrincipalId;
  principalName: string;
  principalEmail: string;
  memberStatus: MemberStatus;
  role: RoleType;
};

export type ActionMemberRoleUpdatePayload = {
  principalId: PrincipalId;
  principalName: string;
  principalEmail: string;
  oldRole: RoleType;
  newRole: RoleType;
};

export type ActionMemberActivateDeactivatePayload = {
  principalId: PrincipalId;
  principalName: string;
  principalEmail: string;
  role: RoleType;
};

export type ActionPayloadType =
  | ActionIssueCreatePayload
  | ActionIssueCommentCreatePayload
  | ActionIssueFieldUpdatePayload
  | ActionIssueStatusUpdatePayload
  | ActionTaskStatusUpdatePayload
  | ActionMemberCreatePayload
  | ActionMemberRoleUpdatePayload
  | ActionMemberActivateDeactivatePayload;

export type Activity = {
  id: ActivityId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  // The object where this activity belongs
  // e.g if actionType is "bb.issue.xxx", then this field refers to the corresponding issue's id.
  containerId: ContainerId;
  actionType: ActivityType;
  level: ActivityLevel;
  comment: string;
  payload?: ActionPayloadType;
};

export type ActivityCreate = {
  // Domain specific fields
  containerId: ContainerId;
  actionType: ActivityType;
  comment: string;
  payload?: ActionPayloadType;
};

export type ActivityPatch = {
  // Domain specific fields
  comment: string;
};
