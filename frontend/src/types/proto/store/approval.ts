/* eslint-disable */
import * as _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export enum ApprovalNodeStatus {
  APPROVAL_NODE_STATUS_UNSPECIFIED = 0,
  PENDING = 1,
  APPROVED = 2,
  UNRECOGNIZED = -1,
}

export function approvalNodeStatusFromJSON(object: any): ApprovalNodeStatus {
  switch (object) {
    case 0:
    case "APPROVAL_NODE_STATUS_UNSPECIFIED":
      return ApprovalNodeStatus.APPROVAL_NODE_STATUS_UNSPECIFIED;
    case 1:
    case "PENDING":
      return ApprovalNodeStatus.PENDING;
    case 2:
    case "APPROVED":
      return ApprovalNodeStatus.APPROVED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ApprovalNodeStatus.UNRECOGNIZED;
  }
}

export function approvalNodeStatusToJSON(object: ApprovalNodeStatus): string {
  switch (object) {
    case ApprovalNodeStatus.APPROVAL_NODE_STATUS_UNSPECIFIED:
      return "APPROVAL_NODE_STATUS_UNSPECIFIED";
    case ApprovalNodeStatus.PENDING:
      return "PENDING";
    case ApprovalNodeStatus.APPROVED:
      return "APPROVED";
    case ApprovalNodeStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface ApprovalPayload {
  pipelineId: number;
  stageId: number;
  taskId?: number | undefined;
  history: ApprovalHistory[];
}

export interface ApprovalHistory {
  /** The `uid` of the approval node. */
  nodeUid: string;
  /** The new status. */
  status: ApprovalNodeStatus;
  /** The principal id of the approver. */
  principalId: number;
}

function createBaseApprovalPayload(): ApprovalPayload {
  return { pipelineId: 0, stageId: 0, taskId: undefined, history: [] };
}

export const ApprovalPayload = {
  encode(message: ApprovalPayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pipelineId !== 0) {
      writer.uint32(8).int32(message.pipelineId);
    }
    if (message.stageId !== 0) {
      writer.uint32(16).int32(message.stageId);
    }
    if (message.taskId !== undefined) {
      writer.uint32(24).int32(message.taskId);
    }
    for (const v of message.history) {
      ApprovalHistory.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ApprovalPayload {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApprovalPayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pipelineId = reader.int32();
          break;
        case 2:
          message.stageId = reader.int32();
          break;
        case 3:
          message.taskId = reader.int32();
          break;
        case 4:
          message.history.push(ApprovalHistory.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ApprovalPayload {
    return {
      pipelineId: isSet(object.pipelineId) ? Number(object.pipelineId) : 0,
      stageId: isSet(object.stageId) ? Number(object.stageId) : 0,
      taskId: isSet(object.taskId) ? Number(object.taskId) : undefined,
      history: Array.isArray(object?.history) ? object.history.map((e: any) => ApprovalHistory.fromJSON(e)) : [],
    };
  },

  toJSON(message: ApprovalPayload): unknown {
    const obj: any = {};
    message.pipelineId !== undefined && (obj.pipelineId = Math.round(message.pipelineId));
    message.stageId !== undefined && (obj.stageId = Math.round(message.stageId));
    message.taskId !== undefined && (obj.taskId = Math.round(message.taskId));
    if (message.history) {
      obj.history = message.history.map((e) => e ? ApprovalHistory.toJSON(e) : undefined);
    } else {
      obj.history = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<ApprovalPayload>): ApprovalPayload {
    const message = createBaseApprovalPayload();
    message.pipelineId = object.pipelineId ?? 0;
    message.stageId = object.stageId ?? 0;
    message.taskId = object.taskId ?? undefined;
    message.history = object.history?.map((e) => ApprovalHistory.fromPartial(e)) || [];
    return message;
  },
};

function createBaseApprovalHistory(): ApprovalHistory {
  return { nodeUid: "", status: 0, principalId: 0 };
}

export const ApprovalHistory = {
  encode(message: ApprovalHistory, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.nodeUid !== "") {
      writer.uint32(10).string(message.nodeUid);
    }
    if (message.status !== 0) {
      writer.uint32(16).int32(message.status);
    }
    if (message.principalId !== 0) {
      writer.uint32(24).int32(message.principalId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ApprovalHistory {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApprovalHistory();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.nodeUid = reader.string();
          break;
        case 2:
          message.status = reader.int32() as any;
          break;
        case 3:
          message.principalId = reader.int32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ApprovalHistory {
    return {
      nodeUid: isSet(object.nodeUid) ? String(object.nodeUid) : "",
      status: isSet(object.status) ? approvalNodeStatusFromJSON(object.status) : 0,
      principalId: isSet(object.principalId) ? Number(object.principalId) : 0,
    };
  },

  toJSON(message: ApprovalHistory): unknown {
    const obj: any = {};
    message.nodeUid !== undefined && (obj.nodeUid = message.nodeUid);
    message.status !== undefined && (obj.status = approvalNodeStatusToJSON(message.status));
    message.principalId !== undefined && (obj.principalId = Math.round(message.principalId));
    return obj;
  },

  fromPartial(object: DeepPartial<ApprovalHistory>): ApprovalHistory {
    const message = createBaseApprovalHistory();
    message.nodeUid = object.nodeUid ?? "";
    message.status = object.status ?? 0;
    message.principalId = object.principalId ?? 0;
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
