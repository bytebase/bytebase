import { ActivityId, ContainerId, MessageId, PrincipalId } from "./id";
import { IssueStatus } from "./issue";
import { RoleType } from "./member";
import { Principal } from "./principal";
import { ProjectRoleType } from "./project";

export type MemberMessageType =
  | "bb.msg.member.create"
  | "bb.msg.member.invite"
  | "bb.msg.member.join"
  | "bb.msg.member.revoke"
  | "bb.msg.member.updaterole";

export type ProjectMemberMessageType =
  | "bb.msg.project.member.create"
  | "bb.msg.project.member.revoke"
  | "bb.msg.project.member.updaterole";

export type EnvironmentMessageType =
  | "bb.msg.environment.create"
  | "bb.msg.environment.update"
  | "bb.msg.environment.delete"
  | "bb.msg.environment.archive"
  | "bb.msg.environment.restore"
  | "bb.msg.environment.reorder";

export type InstanceMessageType =
  | "bb.msg.instance.create"
  | "bb.msg.instance.update"
  | "bb.msg.instance.archive"
  | "bb.msg.instance.restore";

export type IssueMessageType =
  | "bb.msg.issue.assign"
  | "bb.msg.issue.status.update"
  | "bb.msg.issue.stage.status.update"
  | "bb.msg.issue.comment";

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
export type MessageNew = Omit<Message, "id" | "createdTs" | "updatedTs">;

export type MessagePatch = {
  updaterId: PrincipalId;
  status: MessageStatus;
};
