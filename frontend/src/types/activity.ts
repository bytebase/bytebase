import { FieldId } from "../plugins";
import { ActivityId, ContainerId, PrincipalId, TaskId } from "./id";
import { TaskStatus } from "./pipeline";
import { Principal } from "./principal";

export type IssueActionType =
  | "bb.issue.create"
  | "bb.issue.comment.create"
  | "bb.issue.field.update"
  | "bb.issue.status.update"
  | "bb.pipeline.task.status.update";

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
  // e.g if actionType is "bb.issue.xxx", then this field refers to the corresponding issue's id.
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

export type ActivityCreate = {
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
