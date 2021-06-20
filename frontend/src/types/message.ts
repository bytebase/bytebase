import { ActivityId, ContainerId, MessageId, PrincipalId } from "./id";
import { IssueStatus } from "./issue";
import { RoleType } from "./member";
import { Principal } from "./principal";
import { ProjectRoleType } from "./project";

export type MemberMessageType =
  | "bb.message.member.create"
  | "bb.message.member.invite"
  | "bb.message.member.join"
  | "bb.message.member.revoke"
  | "bb.message.member.updaterole";

export type ProjectMemberMessageType =
  | "bb.message.project.member.create"
  | "bb.message.project.member.revoke"
  | "bb.message.project.member.updaterole";

export type EnvironmentMessageType =
  | "bb.message.environment.create"
  | "bb.message.environment.update"
  | "bb.message.environment.delete"
  | "bb.message.environment.archive"
  | "bb.message.environment.restore"
  | "bb.message.environment.reorder";

export type InstanceMessageType =
  | "bb.message.instance.create"
  | "bb.message.instance.update"
  | "bb.message.instance.archive"
  | "bb.message.instance.restore";

export type IssueMessageType =
  | "bb.message.issue.assign"
  | "bb.message.issue.status.update"
  | "bb.message.issue.comment";

export type MessageType =
  | MemberMessageType
  | EnvironmentMessageType
  | InstanceMessageType
  | IssueMessageType;

export type MemberMessagePayload = {
  principalId: PrincipalId;
  oldRole?: RoleType;
  newRole?: RoleType;
};

export type ProjectMemberMessagePayload = {
  principalId: PrincipalId;
  oldRole?: ProjectRoleType;
  newRole?: ProjectRoleType;
};

export type EnvironmentUpdateMessagePayload = {
  environmentName: string;
};

export enum EnvironmentBuiltinFieldId {
  ROW_STATUS = "1",
  NAME = "2",
}

export type EnvironmentMessagePayload = {
  environmentName: string;
  changeList: {
    fieldId: EnvironmentBuiltinFieldId;
    oldValue?: any;
    newValue?: any;
  }[];
};

export enum InstanceBuiltinFieldId {
  ROW_STATUS = "1",
  NAME = "2",
  ENVIRONMENT = "3",
  EXTERNAL_LINK = "4",
  HOST = "5",
  PORT = "6",
  username = "7",
  password = "8",
}

export type InstanceMessagePaylaod = {
  instanceName: string;
};

export type IssueAssignMessagePayload = {
  issueName: string;
  oldAssigneeId: PrincipalId;
  newAssigneeId: PrincipalId;
};

export type IssueUpdateStatusMessagePayload = {
  issueName: string;
  oldStatus: IssueStatus;
  newStatus: IssueStatus;
};

export type IssueCommentMessagePayload = {
  issueName: string;
  commentId: ActivityId;
};

export type MessagePayload =
  | MemberMessagePayload
  | EnvironmentMessagePayload
  | EnvironmentUpdateMessagePayload
  | InstanceMessagePaylaod
  | IssueAssignMessagePayload
  | IssueUpdateStatusMessagePayload
  | IssueCommentMessagePayload;

export type MessageStatus = "DELIVERED" | "CONSUMED";

export type Message = {
  id: MessageId;

  // Related fields
  // The object where this message originates, simliar to containerId in Activity
  containerId: ContainerId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  type: MessageType;
  status: MessageStatus;
  description: string;
  receiver: Principal;
  payload?: MessagePayload;
};
export type MessageCreate = Omit<Message, "id" | "createdTs" | "updatedTs">;

export type MessagePatch = {
  status: MessageStatus;
};
