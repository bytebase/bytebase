/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { ApprovalFlow } from "./approval_template_service";

export const protobufPackage = "bytebase.v1";

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

export interface ListApprovalsRequest {
  /**
   * The parent, which owns this collection of instances.
   * Format: pipelines/{pipeline}
   * Use "pipelines/-" to list all instances from all pipelines.
   */
  parent: string;
  /**
   * The maximum number of instances to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 instances will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListInstances` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListInstances` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface ListApprovalsResponse {
  approvals: Approval[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface ApproveApprovalRequest {
  /** Format: pipelines/{pipeline}/approvals/{approval} */
  name: string;
  /** The `uid` of the approval node. */
  node: string;
  /** The new status of the approval node. */
  status: ApprovalNodeStatus;
}

export interface Approval {
  /** Format: pipelines/{pipeline}/approvals/{approval} */
  name: string;
  /** system-generated unique identifier */
  uid: string;
  stageId: string;
  taskId?: string | undefined;
  flow?: ApprovalFlow;
  history: ApprovalHistory[];
}

export interface ApprovalHistory {
  /** The `uid` of the approval node. */
  nodeUid: string;
  /** The new status. */
  status: ApprovalNodeStatus;
}

function createBaseListApprovalsRequest(): ListApprovalsRequest {
  return { parent: "", pageSize: 0, pageToken: "" };
}

export const ListApprovalsRequest = {
  encode(message: ListApprovalsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.pageSize !== 0) {
      writer.uint32(16).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(26).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListApprovalsRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListApprovalsRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.parent = reader.string();
          break;
        case 2:
          message.pageSize = reader.int32();
          break;
        case 3:
          message.pageToken = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ListApprovalsRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListApprovalsRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    return obj;
  },

  fromPartial(object: DeepPartial<ListApprovalsRequest>): ListApprovalsRequest {
    const message = createBaseListApprovalsRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListApprovalsResponse(): ListApprovalsResponse {
  return { approvals: [], nextPageToken: "" };
}

export const ListApprovalsResponse = {
  encode(message: ListApprovalsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.approvals) {
      Approval.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListApprovalsResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListApprovalsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.approvals.push(Approval.decode(reader, reader.uint32()));
          break;
        case 2:
          message.nextPageToken = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ListApprovalsResponse {
    return {
      approvals: Array.isArray(object?.approvals) ? object.approvals.map((e: any) => Approval.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListApprovalsResponse): unknown {
    const obj: any = {};
    if (message.approvals) {
      obj.approvals = message.approvals.map((e) => e ? Approval.toJSON(e) : undefined);
    } else {
      obj.approvals = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  fromPartial(object: DeepPartial<ListApprovalsResponse>): ListApprovalsResponse {
    const message = createBaseListApprovalsResponse();
    message.approvals = object.approvals?.map((e) => Approval.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseApproveApprovalRequest(): ApproveApprovalRequest {
  return { name: "", node: "", status: 0 };
}

export const ApproveApprovalRequest = {
  encode(message: ApproveApprovalRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.node !== "") {
      writer.uint32(18).string(message.node);
    }
    if (message.status !== 0) {
      writer.uint32(24).int32(message.status);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ApproveApprovalRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApproveApprovalRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.node = reader.string();
          break;
        case 3:
          message.status = reader.int32() as any;
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ApproveApprovalRequest {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      node: isSet(object.node) ? String(object.node) : "",
      status: isSet(object.status) ? approvalNodeStatusFromJSON(object.status) : 0,
    };
  },

  toJSON(message: ApproveApprovalRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.node !== undefined && (obj.node = message.node);
    message.status !== undefined && (obj.status = approvalNodeStatusToJSON(message.status));
    return obj;
  },

  fromPartial(object: DeepPartial<ApproveApprovalRequest>): ApproveApprovalRequest {
    const message = createBaseApproveApprovalRequest();
    message.name = object.name ?? "";
    message.node = object.node ?? "";
    message.status = object.status ?? 0;
    return message;
  },
};

function createBaseApproval(): Approval {
  return { name: "", uid: "", stageId: "", taskId: undefined, flow: undefined, history: [] };
}

export const Approval = {
  encode(message: Approval, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.uid !== "") {
      writer.uint32(18).string(message.uid);
    }
    if (message.stageId !== "") {
      writer.uint32(26).string(message.stageId);
    }
    if (message.taskId !== undefined) {
      writer.uint32(34).string(message.taskId);
    }
    if (message.flow !== undefined) {
      ApprovalFlow.encode(message.flow, writer.uint32(42).fork()).ldelim();
    }
    for (const v of message.history) {
      ApprovalHistory.encode(v!, writer.uint32(50).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Approval {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApproval();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.uid = reader.string();
          break;
        case 3:
          message.stageId = reader.string();
          break;
        case 4:
          message.taskId = reader.string();
          break;
        case 5:
          message.flow = ApprovalFlow.decode(reader, reader.uint32());
          break;
        case 6:
          message.history.push(ApprovalHistory.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Approval {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      uid: isSet(object.uid) ? String(object.uid) : "",
      stageId: isSet(object.stageId) ? String(object.stageId) : "",
      taskId: isSet(object.taskId) ? String(object.taskId) : undefined,
      flow: isSet(object.flow) ? ApprovalFlow.fromJSON(object.flow) : undefined,
      history: Array.isArray(object?.history) ? object.history.map((e: any) => ApprovalHistory.fromJSON(e)) : [],
    };
  },

  toJSON(message: Approval): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.stageId !== undefined && (obj.stageId = message.stageId);
    message.taskId !== undefined && (obj.taskId = message.taskId);
    message.flow !== undefined && (obj.flow = message.flow ? ApprovalFlow.toJSON(message.flow) : undefined);
    if (message.history) {
      obj.history = message.history.map((e) => e ? ApprovalHistory.toJSON(e) : undefined);
    } else {
      obj.history = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<Approval>): Approval {
    const message = createBaseApproval();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.stageId = object.stageId ?? "";
    message.taskId = object.taskId ?? undefined;
    message.flow = (object.flow !== undefined && object.flow !== null)
      ? ApprovalFlow.fromPartial(object.flow)
      : undefined;
    message.history = object.history?.map((e) => ApprovalHistory.fromPartial(e)) || [];
    return message;
  },
};

function createBaseApprovalHistory(): ApprovalHistory {
  return { nodeUid: "", status: 0 };
}

export const ApprovalHistory = {
  encode(message: ApprovalHistory, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.nodeUid !== "") {
      writer.uint32(10).string(message.nodeUid);
    }
    if (message.status !== 0) {
      writer.uint32(16).int32(message.status);
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
    };
  },

  toJSON(message: ApprovalHistory): unknown {
    const obj: any = {};
    message.nodeUid !== undefined && (obj.nodeUid = message.nodeUid);
    message.status !== undefined && (obj.status = approvalNodeStatusToJSON(message.status));
    return obj;
  },

  fromPartial(object: DeepPartial<ApprovalHistory>): ApprovalHistory {
    const message = createBaseApprovalHistory();
    message.nodeUid = object.nodeUid ?? "";
    message.status = object.status ?? 0;
    return message;
  },
};

export type ApprovalServiceDefinition = typeof ApprovalServiceDefinition;
export const ApprovalServiceDefinition = {
  name: "ApprovalService",
  fullName: "bytebase.v1.ApprovalService",
  methods: {
    listApprovals: {
      name: "ListApprovals",
      requestType: ListApprovalsRequest,
      requestStream: false,
      responseType: ListApprovalsResponse,
      responseStream: false,
      options: {},
    },
    approveApproval: {
      name: "ApproveApproval",
      requestType: ApproveApprovalRequest,
      requestStream: false,
      responseType: Approval,
      responseStream: false,
      options: {},
    },
  },
} as const;

export interface ApprovalServiceImplementation<CallContextExt = {}> {
  listApprovals(
    request: ListApprovalsRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ListApprovalsResponse>>;
  approveApproval(
    request: ApproveApprovalRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Approval>>;
}

export interface ApprovalServiceClient<CallOptionsExt = {}> {
  listApprovals(
    request: DeepPartial<ListApprovalsRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ListApprovalsResponse>;
  approveApproval(
    request: DeepPartial<ApproveApprovalRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Approval>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
