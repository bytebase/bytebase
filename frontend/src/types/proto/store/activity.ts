/* eslint-disable */
import * as Long from "long";
import * as _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export interface ActivityPayload {
  issueCreatePayload?: ActivityIssueCreatePayload | undefined;
  issueCommentCreatePayload?: ActivityIssueCommentCreatePayload | undefined;
}

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
  taskRollbackBy?:
    | ActivityIssueCommentCreatePayload_TaskRollbackBy
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

function createBaseActivityPayload(): ActivityPayload {
  return { issueCreatePayload: undefined, issueCommentCreatePayload: undefined };
}

export const ActivityPayload = {
  encode(message: ActivityPayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.issueCreatePayload !== undefined) {
      ActivityIssueCreatePayload.encode(message.issueCreatePayload, writer.uint32(10).fork()).ldelim();
    }
    if (message.issueCommentCreatePayload !== undefined) {
      ActivityIssueCommentCreatePayload.encode(message.issueCommentCreatePayload, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ActivityPayload {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseActivityPayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.issueCreatePayload = ActivityIssueCreatePayload.decode(reader, reader.uint32());
          break;
        case 2:
          message.issueCommentCreatePayload = ActivityIssueCommentCreatePayload.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ActivityPayload {
    return {
      issueCreatePayload: isSet(object.issueCreatePayload)
        ? ActivityIssueCreatePayload.fromJSON(object.issueCreatePayload)
        : undefined,
      issueCommentCreatePayload: isSet(object.issueCommentCreatePayload)
        ? ActivityIssueCommentCreatePayload.fromJSON(object.issueCommentCreatePayload)
        : undefined,
    };
  },

  toJSON(message: ActivityPayload): unknown {
    const obj: any = {};
    message.issueCreatePayload !== undefined && (obj.issueCreatePayload = message.issueCreatePayload
      ? ActivityIssueCreatePayload.toJSON(message.issueCreatePayload)
      : undefined);
    message.issueCommentCreatePayload !== undefined &&
      (obj.issueCommentCreatePayload = message.issueCommentCreatePayload
        ? ActivityIssueCommentCreatePayload.toJSON(message.issueCommentCreatePayload)
        : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<ActivityPayload>): ActivityPayload {
    const message = createBaseActivityPayload();
    message.issueCreatePayload = (object.issueCreatePayload !== undefined && object.issueCreatePayload !== null)
      ? ActivityIssueCreatePayload.fromPartial(object.issueCreatePayload)
      : undefined;
    message.issueCommentCreatePayload =
      (object.issueCommentCreatePayload !== undefined && object.issueCommentCreatePayload !== null)
        ? ActivityIssueCommentCreatePayload.fromPartial(object.issueCommentCreatePayload)
        : undefined;
    return message;
  },
};

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
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseActivityIssueCreatePayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.issueName = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ActivityIssueCreatePayload {
    return { issueName: isSet(object.issueName) ? String(object.issueName) : "" };
  },

  toJSON(message: ActivityIssueCreatePayload): unknown {
    const obj: any = {};
    message.issueName !== undefined && (obj.issueName = message.issueName);
    return obj;
  },

  fromPartial(object: DeepPartial<ActivityIssueCreatePayload>): ActivityIssueCreatePayload {
    const message = createBaseActivityIssueCreatePayload();
    message.issueName = object.issueName ?? "";
    return message;
  },
};

function createBaseActivityIssueCommentCreatePayload(): ActivityIssueCommentCreatePayload {
  return { externalApprovalEvent: undefined, taskRollbackBy: undefined, issueName: "" };
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
    if (message.issueName !== "") {
      writer.uint32(26).string(message.issueName);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ActivityIssueCommentCreatePayload {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseActivityIssueCommentCreatePayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.externalApprovalEvent = ActivityIssueCommentCreatePayload_ExternalApprovalEvent.decode(
            reader,
            reader.uint32(),
          );
          break;
        case 2:
          message.taskRollbackBy = ActivityIssueCommentCreatePayload_TaskRollbackBy.decode(reader, reader.uint32());
          break;
        case 3:
          message.issueName = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
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
      issueName: isSet(object.issueName) ? String(object.issueName) : "",
    };
  },

  toJSON(message: ActivityIssueCommentCreatePayload): unknown {
    const obj: any = {};
    message.externalApprovalEvent !== undefined && (obj.externalApprovalEvent = message.externalApprovalEvent
      ? ActivityIssueCommentCreatePayload_ExternalApprovalEvent.toJSON(message.externalApprovalEvent)
      : undefined);
    message.taskRollbackBy !== undefined && (obj.taskRollbackBy = message.taskRollbackBy
      ? ActivityIssueCommentCreatePayload_TaskRollbackBy.toJSON(message.taskRollbackBy)
      : undefined);
    message.issueName !== undefined && (obj.issueName = message.issueName);
    return obj;
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
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseActivityIssueCommentCreatePayload_TaskRollbackBy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.issueId = longToNumber(reader.int64() as Long);
          break;
        case 2:
          message.taskId = longToNumber(reader.int64() as Long);
          break;
        case 3:
          message.rollbackByIssueId = longToNumber(reader.int64() as Long);
          break;
        case 4:
          message.rollbackByTaskId = longToNumber(reader.int64() as Long);
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
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
    message.issueId !== undefined && (obj.issueId = Math.round(message.issueId));
    message.taskId !== undefined && (obj.taskId = Math.round(message.taskId));
    message.rollbackByIssueId !== undefined && (obj.rollbackByIssueId = Math.round(message.rollbackByIssueId));
    message.rollbackByTaskId !== undefined && (obj.rollbackByTaskId = Math.round(message.rollbackByTaskId));
    return obj;
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
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseActivityIssueCommentCreatePayload_ExternalApprovalEvent();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.type = reader.int32() as any;
          break;
        case 2:
          message.action = reader.int32() as any;
          break;
        case 3:
          message.stageName = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
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
    message.type !== undefined &&
      (obj.type = activityIssueCommentCreatePayload_ExternalApprovalEvent_TypeToJSON(message.type));
    message.action !== undefined &&
      (obj.action = activityIssueCommentCreatePayload_ExternalApprovalEvent_ActionToJSON(message.action));
    message.stageName !== undefined && (obj.stageName = message.stageName);
    return obj;
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

declare var self: any | undefined;
declare var window: any | undefined;
declare var global: any | undefined;
var globalThis: any = (() => {
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
    throw new globalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
  }
  return long.toNumber();
}

// If you get a compile-error about 'Constructor<Long> and ... have no overlap',
// add '--ts_proto_opt=esModuleInterop=true' as a flag when calling 'protoc'.
if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
