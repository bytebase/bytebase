/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { ApprovalStep } from "./approval";
import Long = require("long");

export const protobufPackage = "bytebase.store";

/**
 * ActivityIssueCreatePayload is the payloads for creating issues.
 * These payload types are only used when marshalling to the json format for saving into the database.
 * So we annotate with json tag using camelCase naming which is consistent with normal
 * json naming convention. More importantly, frontend code can simply use JSON.parse to
 * convert to the expected struct there.
 */
export interface ActivityIssueCreatePayload {
  /** Used by inbox to display info without paying the join cost */
  issueName: string;
}

/** ActivityIssueCommentCreatePayload is the payloads for creating issue comments. */
export interface ActivityIssueCommentCreatePayload {
  externalApprovalEvent?: ActivityIssueCommentCreatePayload_ExternalApprovalEvent | undefined;
  taskRollbackBy?: ActivityIssueCommentCreatePayload_TaskRollbackBy | undefined;
  approvalEvent?:
    | ActivityIssueCommentCreatePayload_ApprovalEvent
    | undefined;
  /** Used by inbox to display info without paying the join cost */
  issueName: string;
}

/**
 * TaskRollbackBy records an issue rollback activity.
 * The task with taskID in IssueID is rollbacked by the task with RollbackByTaskID in RollbackByIssueID.
 */
export interface ActivityIssueCommentCreatePayload_TaskRollbackBy {
  issueId: number;
  taskId: number;
  rollbackByIssueId: number;
  rollbackByTaskId: number;
}

export interface ActivityIssueCommentCreatePayload_ExternalApprovalEvent {
  type: ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Type;
  action: ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Action;
  stageName: string;
}

export enum ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Type {
  TYPE_UNSPECIFIED = 0,
  TYPE_FEISHU = 1,
  UNRECOGNIZED = -1,
}

export function activityIssueCommentCreatePayload_ExternalApprovalEvent_TypeFromJSON(
  object: any,
): ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Type.TYPE_UNSPECIFIED;
    case 1:
    case "TYPE_FEISHU":
      return ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Type.TYPE_FEISHU;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Type.UNRECOGNIZED;
  }
}

export function activityIssueCommentCreatePayload_ExternalApprovalEvent_TypeToJSON(
  object: ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Type,
): string {
  switch (object) {
    case ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Type.TYPE_FEISHU:
      return "TYPE_FEISHU";
    case ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Action {
  ACTION_UNSPECIFIED = 0,
  ACTION_APPROVE = 1,
  ACTION_REJECT = 2,
  UNRECOGNIZED = -1,
}

export function activityIssueCommentCreatePayload_ExternalApprovalEvent_ActionFromJSON(
  object: any,
): ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Action {
  switch (object) {
    case 0:
    case "ACTION_UNSPECIFIED":
      return ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Action.ACTION_UNSPECIFIED;
    case 1:
    case "ACTION_APPROVE":
      return ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Action.ACTION_APPROVE;
    case 2:
    case "ACTION_REJECT":
      return ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Action.ACTION_REJECT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Action.UNRECOGNIZED;
  }
}

export function activityIssueCommentCreatePayload_ExternalApprovalEvent_ActionToJSON(
  object: ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Action,
): string {
  switch (object) {
    case ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Action.ACTION_UNSPECIFIED:
      return "ACTION_UNSPECIFIED";
    case ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Action.ACTION_APPROVE:
      return "ACTION_APPROVE";
    case ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Action.ACTION_REJECT:
      return "ACTION_REJECT";
    case ActivityIssueCommentCreatePayload_ExternalApprovalEvent_Action.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface ActivityIssueCommentCreatePayload_ApprovalEvent {
  /** The new status. */
  status: ActivityIssueCommentCreatePayload_ApprovalEvent_Status;
}

export enum ActivityIssueCommentCreatePayload_ApprovalEvent_Status {
  STATUS_UNSPECIFIED = 0,
  PENDING = 1,
  APPROVED = 2,
  REJECTED = 3,
  UNRECOGNIZED = -1,
}

export function activityIssueCommentCreatePayload_ApprovalEvent_StatusFromJSON(
  object: any,
): ActivityIssueCommentCreatePayload_ApprovalEvent_Status {
  switch (object) {
    case 0:
    case "STATUS_UNSPECIFIED":
      return ActivityIssueCommentCreatePayload_ApprovalEvent_Status.STATUS_UNSPECIFIED;
    case 1:
    case "PENDING":
      return ActivityIssueCommentCreatePayload_ApprovalEvent_Status.PENDING;
    case 2:
    case "APPROVED":
      return ActivityIssueCommentCreatePayload_ApprovalEvent_Status.APPROVED;
    case 3:
    case "REJECTED":
      return ActivityIssueCommentCreatePayload_ApprovalEvent_Status.REJECTED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ActivityIssueCommentCreatePayload_ApprovalEvent_Status.UNRECOGNIZED;
  }
}

export function activityIssueCommentCreatePayload_ApprovalEvent_StatusToJSON(
  object: ActivityIssueCommentCreatePayload_ApprovalEvent_Status,
): string {
  switch (object) {
    case ActivityIssueCommentCreatePayload_ApprovalEvent_Status.STATUS_UNSPECIFIED:
      return "STATUS_UNSPECIFIED";
    case ActivityIssueCommentCreatePayload_ApprovalEvent_Status.PENDING:
      return "PENDING";
    case ActivityIssueCommentCreatePayload_ApprovalEvent_Status.APPROVED:
      return "APPROVED";
    case ActivityIssueCommentCreatePayload_ApprovalEvent_Status.REJECTED:
      return "REJECTED";
    case ActivityIssueCommentCreatePayload_ApprovalEvent_Status.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface ActivityIssueApprovalNotifyPayload {
  approvalStep?: ApprovalStep | undefined;
}

function createBaseActivityIssueCreatePayload(): ActivityIssueCreatePayload {
  return { issueName: "" };
}

export const ActivityIssueCreatePayload = {
  encode(message: ActivityIssueCreatePayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.issueName !== "") {
      writer.uint32(10).string(message.issueName);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ActivityIssueCreatePayload {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseActivityIssueCreatePayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.issueName = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ActivityIssueCreatePayload {
    return { issueName: isSet(object.issueName) ? String(object.issueName) : "" };
  },

  toJSON(message: ActivityIssueCreatePayload): unknown {
    const obj: any = {};
    if (message.issueName !== "") {
      obj.issueName = message.issueName;
    }
    return obj;
  },

  create(base?: DeepPartial<ActivityIssueCreatePayload>): ActivityIssueCreatePayload {
    return ActivityIssueCreatePayload.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ActivityIssueCreatePayload>): ActivityIssueCreatePayload {
    const message = createBaseActivityIssueCreatePayload();
    message.issueName = object.issueName ?? "";
    return message;
  },
};

function createBaseActivityIssueCommentCreatePayload(): ActivityIssueCommentCreatePayload {
  return { externalApprovalEvent: undefined, taskRollbackBy: undefined, approvalEvent: undefined, issueName: "" };
}

export const ActivityIssueCommentCreatePayload = {
  encode(message: ActivityIssueCommentCreatePayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.externalApprovalEvent !== undefined) {
      ActivityIssueCommentCreatePayload_ExternalApprovalEvent.encode(
        message.externalApprovalEvent,
        writer.uint32(10).fork(),
      ).ldelim();
    }
    if (message.taskRollbackBy !== undefined) {
      ActivityIssueCommentCreatePayload_TaskRollbackBy.encode(message.taskRollbackBy, writer.uint32(18).fork())
        .ldelim();
    }
    if (message.approvalEvent !== undefined) {
      ActivityIssueCommentCreatePayload_ApprovalEvent.encode(message.approvalEvent, writer.uint32(26).fork()).ldelim();
    }
    if (message.issueName !== "") {
      writer.uint32(34).string(message.issueName);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ActivityIssueCommentCreatePayload {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseActivityIssueCommentCreatePayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.externalApprovalEvent = ActivityIssueCommentCreatePayload_ExternalApprovalEvent.decode(
            reader,
            reader.uint32(),
          );
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.taskRollbackBy = ActivityIssueCommentCreatePayload_TaskRollbackBy.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.approvalEvent = ActivityIssueCommentCreatePayload_ApprovalEvent.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.issueName = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ActivityIssueCommentCreatePayload {
    return {
      externalApprovalEvent: isSet(object.externalApprovalEvent)
        ? ActivityIssueCommentCreatePayload_ExternalApprovalEvent.fromJSON(object.externalApprovalEvent)
        : undefined,
      taskRollbackBy: isSet(object.taskRollbackBy)
        ? ActivityIssueCommentCreatePayload_TaskRollbackBy.fromJSON(object.taskRollbackBy)
        : undefined,
      approvalEvent: isSet(object.approvalEvent)
        ? ActivityIssueCommentCreatePayload_ApprovalEvent.fromJSON(object.approvalEvent)
        : undefined,
      issueName: isSet(object.issueName) ? String(object.issueName) : "",
    };
  },

  toJSON(message: ActivityIssueCommentCreatePayload): unknown {
    const obj: any = {};
    if (message.externalApprovalEvent !== undefined) {
      obj.externalApprovalEvent = ActivityIssueCommentCreatePayload_ExternalApprovalEvent.toJSON(
        message.externalApprovalEvent,
      );
    }
    if (message.taskRollbackBy !== undefined) {
      obj.taskRollbackBy = ActivityIssueCommentCreatePayload_TaskRollbackBy.toJSON(message.taskRollbackBy);
    }
    if (message.approvalEvent !== undefined) {
      obj.approvalEvent = ActivityIssueCommentCreatePayload_ApprovalEvent.toJSON(message.approvalEvent);
    }
    if (message.issueName !== "") {
      obj.issueName = message.issueName;
    }
    return obj;
  },

  create(base?: DeepPartial<ActivityIssueCommentCreatePayload>): ActivityIssueCommentCreatePayload {
    return ActivityIssueCommentCreatePayload.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ActivityIssueCommentCreatePayload>): ActivityIssueCommentCreatePayload {
    const message = createBaseActivityIssueCommentCreatePayload();
    message.externalApprovalEvent =
      (object.externalApprovalEvent !== undefined && object.externalApprovalEvent !== null)
        ? ActivityIssueCommentCreatePayload_ExternalApprovalEvent.fromPartial(object.externalApprovalEvent)
        : undefined;
    message.taskRollbackBy = (object.taskRollbackBy !== undefined && object.taskRollbackBy !== null)
      ? ActivityIssueCommentCreatePayload_TaskRollbackBy.fromPartial(object.taskRollbackBy)
      : undefined;
    message.approvalEvent = (object.approvalEvent !== undefined && object.approvalEvent !== null)
      ? ActivityIssueCommentCreatePayload_ApprovalEvent.fromPartial(object.approvalEvent)
      : undefined;
    message.issueName = object.issueName ?? "";
    return message;
  },
};

function createBaseActivityIssueCommentCreatePayload_TaskRollbackBy(): ActivityIssueCommentCreatePayload_TaskRollbackBy {
  return { issueId: 0, taskId: 0, rollbackByIssueId: 0, rollbackByTaskId: 0 };
}

export const ActivityIssueCommentCreatePayload_TaskRollbackBy = {
  encode(
    message: ActivityIssueCommentCreatePayload_TaskRollbackBy,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.issueId !== 0) {
      writer.uint32(8).int64(message.issueId);
    }
    if (message.taskId !== 0) {
      writer.uint32(16).int64(message.taskId);
    }
    if (message.rollbackByIssueId !== 0) {
      writer.uint32(24).int64(message.rollbackByIssueId);
    }
    if (message.rollbackByTaskId !== 0) {
      writer.uint32(32).int64(message.rollbackByTaskId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ActivityIssueCommentCreatePayload_TaskRollbackBy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseActivityIssueCommentCreatePayload_TaskRollbackBy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.issueId = longToNumber(reader.int64() as Long);
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.taskId = longToNumber(reader.int64() as Long);
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.rollbackByIssueId = longToNumber(reader.int64() as Long);
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.rollbackByTaskId = longToNumber(reader.int64() as Long);
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ActivityIssueCommentCreatePayload_TaskRollbackBy {
    return {
      issueId: isSet(object.issueId) ? Number(object.issueId) : 0,
      taskId: isSet(object.taskId) ? Number(object.taskId) : 0,
      rollbackByIssueId: isSet(object.rollbackByIssueId) ? Number(object.rollbackByIssueId) : 0,
      rollbackByTaskId: isSet(object.rollbackByTaskId) ? Number(object.rollbackByTaskId) : 0,
    };
  },

  toJSON(message: ActivityIssueCommentCreatePayload_TaskRollbackBy): unknown {
    const obj: any = {};
    if (message.issueId !== 0) {
      obj.issueId = Math.round(message.issueId);
    }
    if (message.taskId !== 0) {
      obj.taskId = Math.round(message.taskId);
    }
    if (message.rollbackByIssueId !== 0) {
      obj.rollbackByIssueId = Math.round(message.rollbackByIssueId);
    }
    if (message.rollbackByTaskId !== 0) {
      obj.rollbackByTaskId = Math.round(message.rollbackByTaskId);
    }
    return obj;
  },

  create(
    base?: DeepPartial<ActivityIssueCommentCreatePayload_TaskRollbackBy>,
  ): ActivityIssueCommentCreatePayload_TaskRollbackBy {
    return ActivityIssueCommentCreatePayload_TaskRollbackBy.fromPartial(base ?? {});
  },
  fromPartial(
    object: DeepPartial<ActivityIssueCommentCreatePayload_TaskRollbackBy>,
  ): ActivityIssueCommentCreatePayload_TaskRollbackBy {
    const message = createBaseActivityIssueCommentCreatePayload_TaskRollbackBy();
    message.issueId = object.issueId ?? 0;
    message.taskId = object.taskId ?? 0;
    message.rollbackByIssueId = object.rollbackByIssueId ?? 0;
    message.rollbackByTaskId = object.rollbackByTaskId ?? 0;
    return message;
  },
};

function createBaseActivityIssueCommentCreatePayload_ExternalApprovalEvent(): ActivityIssueCommentCreatePayload_ExternalApprovalEvent {
  return { type: 0, action: 0, stageName: "" };
}

export const ActivityIssueCommentCreatePayload_ExternalApprovalEvent = {
  encode(
    message: ActivityIssueCommentCreatePayload_ExternalApprovalEvent,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.type !== 0) {
      writer.uint32(8).int32(message.type);
    }
    if (message.action !== 0) {
      writer.uint32(16).int32(message.action);
    }
    if (message.stageName !== "") {
      writer.uint32(26).string(message.stageName);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ActivityIssueCommentCreatePayload_ExternalApprovalEvent {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseActivityIssueCommentCreatePayload_ExternalApprovalEvent();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.type = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.action = reader.int32() as any;
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.stageName = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ActivityIssueCommentCreatePayload_ExternalApprovalEvent {
    return {
      type: isSet(object.type) ? activityIssueCommentCreatePayload_ExternalApprovalEvent_TypeFromJSON(object.type) : 0,
      action: isSet(object.action)
        ? activityIssueCommentCreatePayload_ExternalApprovalEvent_ActionFromJSON(object.action)
        : 0,
      stageName: isSet(object.stageName) ? String(object.stageName) : "",
    };
  },

  toJSON(message: ActivityIssueCommentCreatePayload_ExternalApprovalEvent): unknown {
    const obj: any = {};
    if (message.type !== 0) {
      obj.type = activityIssueCommentCreatePayload_ExternalApprovalEvent_TypeToJSON(message.type);
    }
    if (message.action !== 0) {
      obj.action = activityIssueCommentCreatePayload_ExternalApprovalEvent_ActionToJSON(message.action);
    }
    if (message.stageName !== "") {
      obj.stageName = message.stageName;
    }
    return obj;
  },

  create(
    base?: DeepPartial<ActivityIssueCommentCreatePayload_ExternalApprovalEvent>,
  ): ActivityIssueCommentCreatePayload_ExternalApprovalEvent {
    return ActivityIssueCommentCreatePayload_ExternalApprovalEvent.fromPartial(base ?? {});
  },
  fromPartial(
    object: DeepPartial<ActivityIssueCommentCreatePayload_ExternalApprovalEvent>,
  ): ActivityIssueCommentCreatePayload_ExternalApprovalEvent {
    const message = createBaseActivityIssueCommentCreatePayload_ExternalApprovalEvent();
    message.type = object.type ?? 0;
    message.action = object.action ?? 0;
    message.stageName = object.stageName ?? "";
    return message;
  },
};

function createBaseActivityIssueCommentCreatePayload_ApprovalEvent(): ActivityIssueCommentCreatePayload_ApprovalEvent {
  return { status: 0 };
}

export const ActivityIssueCommentCreatePayload_ApprovalEvent = {
  encode(
    message: ActivityIssueCommentCreatePayload_ApprovalEvent,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.status !== 0) {
      writer.uint32(8).int32(message.status);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ActivityIssueCommentCreatePayload_ApprovalEvent {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseActivityIssueCommentCreatePayload_ApprovalEvent();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.status = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ActivityIssueCommentCreatePayload_ApprovalEvent {
    return {
      status: isSet(object.status) ? activityIssueCommentCreatePayload_ApprovalEvent_StatusFromJSON(object.status) : 0,
    };
  },

  toJSON(message: ActivityIssueCommentCreatePayload_ApprovalEvent): unknown {
    const obj: any = {};
    if (message.status !== 0) {
      obj.status = activityIssueCommentCreatePayload_ApprovalEvent_StatusToJSON(message.status);
    }
    return obj;
  },

  create(
    base?: DeepPartial<ActivityIssueCommentCreatePayload_ApprovalEvent>,
  ): ActivityIssueCommentCreatePayload_ApprovalEvent {
    return ActivityIssueCommentCreatePayload_ApprovalEvent.fromPartial(base ?? {});
  },
  fromPartial(
    object: DeepPartial<ActivityIssueCommentCreatePayload_ApprovalEvent>,
  ): ActivityIssueCommentCreatePayload_ApprovalEvent {
    const message = createBaseActivityIssueCommentCreatePayload_ApprovalEvent();
    message.status = object.status ?? 0;
    return message;
  },
};

function createBaseActivityIssueApprovalNotifyPayload(): ActivityIssueApprovalNotifyPayload {
  return { approvalStep: undefined };
}

export const ActivityIssueApprovalNotifyPayload = {
  encode(message: ActivityIssueApprovalNotifyPayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.approvalStep !== undefined) {
      ApprovalStep.encode(message.approvalStep, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ActivityIssueApprovalNotifyPayload {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseActivityIssueApprovalNotifyPayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.approvalStep = ApprovalStep.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ActivityIssueApprovalNotifyPayload {
    return { approvalStep: isSet(object.approvalStep) ? ApprovalStep.fromJSON(object.approvalStep) : undefined };
  },

  toJSON(message: ActivityIssueApprovalNotifyPayload): unknown {
    const obj: any = {};
    if (message.approvalStep !== undefined) {
      obj.approvalStep = ApprovalStep.toJSON(message.approvalStep);
    }
    return obj;
  },

  create(base?: DeepPartial<ActivityIssueApprovalNotifyPayload>): ActivityIssueApprovalNotifyPayload {
    return ActivityIssueApprovalNotifyPayload.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ActivityIssueApprovalNotifyPayload>): ActivityIssueApprovalNotifyPayload {
    const message = createBaseActivityIssueApprovalNotifyPayload();
    message.approvalStep = (object.approvalStep !== undefined && object.approvalStep !== null)
      ? ApprovalStep.fromPartial(object.approvalStep)
      : undefined;
    return message;
  },
};

declare const self: any | undefined;
declare const window: any | undefined;
declare const global: any | undefined;
const tsProtoGlobalThis: any = (() => {
  if (typeof globalThis !== "undefined") {
    return globalThis;
  }
  if (typeof self !== "undefined") {
    return self;
  }
  if (typeof window !== "undefined") {
    return window;
  }
  if (typeof global !== "undefined") {
    return global;
  }
  throw "Unable to locate global object";
})();

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function longToNumber(long: Long): number {
  if (long.gt(Number.MAX_SAFE_INTEGER)) {
    throw new tsProtoGlobalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
  }
  return long.toNumber();
}

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
