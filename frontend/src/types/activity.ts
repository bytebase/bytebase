import { FieldId } from "../plugins";
import { ActivityId, ContainerId, PrincipalId, TaskId } from "./id";
import { TaskStatus } from "./pipeline";
import { Principal } from "./principal";

export type IssueActionType =
  | "bytebase.issue.create"
  | "bytebase.issue.comment.create"
  | "bytebase.issue.field.update"
  | "bytebase.issue.status.update"
  | "bytebase.pipeline.task.status.update";

export type ActionType = IssueActionType;

export type ActionFieldUpdatePayload = {
  changeList: {
    fieldId: FieldId;
    oldValue?: string;
    newValue?: string;
  }[];
};

export type ActionTaskStatusUpdatePayload = {
  taskId: TaskId;
  oldStatus: TaskStatus;
  newStatus: TaskStatus;
};

export type ActionPayloadType =
  | ActionFieldUpdatePayload
  | ActionTaskStatusUpdatePayload;

export type Activity = {
  id: ActivityId;

  // Related fields
  // The object where this activity belongs
  // e.g if actionType is "bytebase.issue.xxx", then this field refers to the corresponding issue's id.
  containerId: ContainerId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  actionType: ActionType;
  comment: string;
  payload?: ActionPayloadType;
};

export type ActivityNew = {
  // Related fields
  containerId: ContainerId;

  // Standard fields
  creatorId: PrincipalId;

  // Domain specific fields
  actionType: ActionType;
  comment: string;
  payload?: ActionPayloadType;
};

export type ActivityPatch = {
  // Standard fields
  updaterId: PrincipalId;

  // Domain specific fields
  comment: string;
};
