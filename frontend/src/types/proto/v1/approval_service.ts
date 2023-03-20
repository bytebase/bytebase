/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.v1";

export interface GetApprovalRequest {
  /** Format: stages/{stage}/tasks/{task}/approvals/{approval} */
  name: string;
}

export interface ListApprovalsRequest {
  /**
   * The parent, which owns this collection of instances.
   * Format: stages/{stage}/tasks/{task}
   * Use "stages/-/tasks/-" to list all instances from all stages.
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
  /** Show deleted instances if specified. */
  showDeleted: boolean;
}

export interface ListApprovalsResponse {
  approvals: Approval[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface PatchApprovalNodeStatusRequest {
  /** Format: stages/{stage}/tasks/{task}/approvals/{approval} */
  parent: string;
  /** The `uid` of the approval node. */
  node: string;
  /** The new status of the approval node. */
  status: ApprovalNode_Status;
}

export interface Approval {
  /** Format: stages/{stage}/tasks/{task}/approvals/{approval} */
  name: string;
  /** system-generated unique identifier */
  uid: string;
  flow?: ApprovalFlow;
}

export interface ApprovalFlow {
  steps: ApprovalStep[];
}

export interface ApprovalStep {
  type: ApprovalStep_Type;
  nodes: ApprovalNode[];
}

/**
 * Type of the ApprovalStep
 * AND means every node must be approved to proceed.
 * OR means approving any node will proceed.
 */
export enum ApprovalStep_Type {
  TYPE_UNSPECIFIED = 0,
  AND = 1,
  OR = 2,
  UNRECOGNIZED = -1,
}

export function approvalStep_TypeFromJSON(object: any): ApprovalStep_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return ApprovalStep_Type.TYPE_UNSPECIFIED;
    case 1:
    case "AND":
      return ApprovalStep_Type.AND;
    case 2:
    case "OR":
      return ApprovalStep_Type.OR;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ApprovalStep_Type.UNRECOGNIZED;
  }
}

export function approvalStep_TypeToJSON(object: ApprovalStep_Type): string {
  switch (object) {
    case ApprovalStep_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case ApprovalStep_Type.AND:
      return "AND";
    case ApprovalStep_Type.OR:
      return "OR";
    case ApprovalStep_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface ApprovalNode {
  /** uid uniquely identifies a node in a flow. */
  uid: string;
  status: ApprovalNode_Status;
  type: ApprovalNode_Type;
  roleValue?: ApprovalNode_RoleValue | undefined;
}

/** Status of the ApprovalNode. */
export enum ApprovalNode_Status {
  STATUS_UNSPECIFIED = 0,
  PENDING = 1,
  APPROVED = 2,
  UNRECOGNIZED = -1,
}

export function approvalNode_StatusFromJSON(object: any): ApprovalNode_Status {
  switch (object) {
    case 0:
    case "STATUS_UNSPECIFIED":
      return ApprovalNode_Status.STATUS_UNSPECIFIED;
    case 1:
    case "PENDING":
      return ApprovalNode_Status.PENDING;
    case 2:
    case "APPROVED":
      return ApprovalNode_Status.APPROVED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ApprovalNode_Status.UNRECOGNIZED;
  }
}

export function approvalNode_StatusToJSON(object: ApprovalNode_Status): string {
  switch (object) {
    case ApprovalNode_Status.STATUS_UNSPECIFIED:
      return "STATUS_UNSPECIFIED";
    case ApprovalNode_Status.PENDING:
      return "PENDING";
    case ApprovalNode_Status.APPROVED:
      return "APPROVED";
    case ApprovalNode_Status.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

/**
 * Type of the ApprovalNode.
 * type determines who should approve this node.
 * ROLE means the ApprovalNode can be approved by an user from our predefined user group.
 * See RoleValue below for the predefined user groups.
 */
export enum ApprovalNode_Type {
  TYPE_UNSPECIFIED = 0,
  ROLE = 1,
  UNRECOGNIZED = -1,
}

export function approvalNode_TypeFromJSON(object: any): ApprovalNode_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return ApprovalNode_Type.TYPE_UNSPECIFIED;
    case 1:
    case "ROLE":
      return ApprovalNode_Type.ROLE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ApprovalNode_Type.UNRECOGNIZED;
  }
}

export function approvalNode_TypeToJSON(object: ApprovalNode_Type): string {
  switch (object) {
    case ApprovalNode_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case ApprovalNode_Type.ROLE:
      return "ROLE";
    case ApprovalNode_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

/**
 * RoleValue is used if ApprovalNode Type is ROLE
 * The predefined user groups are:
 * - WORKSPACE_OWNER
 * - DBA
 * - PROJECT_OWNER
 * - PROJECT_MEMBER
 */
export enum ApprovalNode_RoleValue {
  ROLE_VALUE_UNSPECIFILED = 0,
  WORKSPACE_OWNER = 1,
  DBA = 2,
  PROJECT_OWNER = 3,
  PROJECT_MEMBER = 4,
  UNRECOGNIZED = -1,
}

export function approvalNode_RoleValueFromJSON(object: any): ApprovalNode_RoleValue {
  switch (object) {
    case 0:
    case "ROLE_VALUE_UNSPECIFILED":
      return ApprovalNode_RoleValue.ROLE_VALUE_UNSPECIFILED;
    case 1:
    case "WORKSPACE_OWNER":
      return ApprovalNode_RoleValue.WORKSPACE_OWNER;
    case 2:
    case "DBA":
      return ApprovalNode_RoleValue.DBA;
    case 3:
    case "PROJECT_OWNER":
      return ApprovalNode_RoleValue.PROJECT_OWNER;
    case 4:
    case "PROJECT_MEMBER":
      return ApprovalNode_RoleValue.PROJECT_MEMBER;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ApprovalNode_RoleValue.UNRECOGNIZED;
  }
}

export function approvalNode_RoleValueToJSON(object: ApprovalNode_RoleValue): string {
  switch (object) {
    case ApprovalNode_RoleValue.ROLE_VALUE_UNSPECIFILED:
      return "ROLE_VALUE_UNSPECIFILED";
    case ApprovalNode_RoleValue.WORKSPACE_OWNER:
      return "WORKSPACE_OWNER";
    case ApprovalNode_RoleValue.DBA:
      return "DBA";
    case ApprovalNode_RoleValue.PROJECT_OWNER:
      return "PROJECT_OWNER";
    case ApprovalNode_RoleValue.PROJECT_MEMBER:
      return "PROJECT_MEMBER";
    case ApprovalNode_RoleValue.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

function createBaseGetApprovalRequest(): GetApprovalRequest {
  return { name: "" };
}

export const GetApprovalRequest = {
  encode(message: GetApprovalRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetApprovalRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetApprovalRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GetApprovalRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetApprovalRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<GetApprovalRequest>): GetApprovalRequest {
    const message = createBaseGetApprovalRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListApprovalsRequest(): ListApprovalsRequest {
  return { parent: "", pageSize: 0, pageToken: "", showDeleted: false };
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
    if (message.showDeleted === true) {
      writer.uint32(32).bool(message.showDeleted);
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
        case 4:
          message.showDeleted = reader.bool();
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
      showDeleted: isSet(object.showDeleted) ? Boolean(object.showDeleted) : false,
    };
  },

  toJSON(message: ListApprovalsRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    message.showDeleted !== undefined && (obj.showDeleted = message.showDeleted);
    return obj;
  },

  fromPartial(object: DeepPartial<ListApprovalsRequest>): ListApprovalsRequest {
    const message = createBaseListApprovalsRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    message.showDeleted = object.showDeleted ?? false;
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

function createBasePatchApprovalNodeStatusRequest(): PatchApprovalNodeStatusRequest {
  return { parent: "", node: "", status: 0 };
}

export const PatchApprovalNodeStatusRequest = {
  encode(message: PatchApprovalNodeStatusRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.node !== "") {
      writer.uint32(18).string(message.node);
    }
    if (message.status !== 0) {
      writer.uint32(24).int32(message.status);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PatchApprovalNodeStatusRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePatchApprovalNodeStatusRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.parent = reader.string();
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

  fromJSON(object: any): PatchApprovalNodeStatusRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      node: isSet(object.node) ? String(object.node) : "",
      status: isSet(object.status) ? approvalNode_StatusFromJSON(object.status) : 0,
    };
  },

  toJSON(message: PatchApprovalNodeStatusRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.node !== undefined && (obj.node = message.node);
    message.status !== undefined && (obj.status = approvalNode_StatusToJSON(message.status));
    return obj;
  },

  fromPartial(object: DeepPartial<PatchApprovalNodeStatusRequest>): PatchApprovalNodeStatusRequest {
    const message = createBasePatchApprovalNodeStatusRequest();
    message.parent = object.parent ?? "";
    message.node = object.node ?? "";
    message.status = object.status ?? 0;
    return message;
  },
};

function createBaseApproval(): Approval {
  return { name: "", uid: "", flow: undefined };
}

export const Approval = {
  encode(message: Approval, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.uid !== "") {
      writer.uint32(18).string(message.uid);
    }
    if (message.flow !== undefined) {
      ApprovalFlow.encode(message.flow, writer.uint32(26).fork()).ldelim();
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
          message.flow = ApprovalFlow.decode(reader, reader.uint32());
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
      flow: isSet(object.flow) ? ApprovalFlow.fromJSON(object.flow) : undefined,
    };
  },

  toJSON(message: Approval): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.flow !== undefined && (obj.flow = message.flow ? ApprovalFlow.toJSON(message.flow) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<Approval>): Approval {
    const message = createBaseApproval();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.flow = (object.flow !== undefined && object.flow !== null)
      ? ApprovalFlow.fromPartial(object.flow)
      : undefined;
    return message;
  },
};

function createBaseApprovalFlow(): ApprovalFlow {
  return { steps: [] };
}

export const ApprovalFlow = {
  encode(message: ApprovalFlow, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.steps) {
      ApprovalStep.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ApprovalFlow {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApprovalFlow();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.steps.push(ApprovalStep.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ApprovalFlow {
    return { steps: Array.isArray(object?.steps) ? object.steps.map((e: any) => ApprovalStep.fromJSON(e)) : [] };
  },

  toJSON(message: ApprovalFlow): unknown {
    const obj: any = {};
    if (message.steps) {
      obj.steps = message.steps.map((e) => e ? ApprovalStep.toJSON(e) : undefined);
    } else {
      obj.steps = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<ApprovalFlow>): ApprovalFlow {
    const message = createBaseApprovalFlow();
    message.steps = object.steps?.map((e) => ApprovalStep.fromPartial(e)) || [];
    return message;
  },
};

function createBaseApprovalStep(): ApprovalStep {
  return { type: 0, nodes: [] };
}

export const ApprovalStep = {
  encode(message: ApprovalStep, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.type !== 0) {
      writer.uint32(8).int32(message.type);
    }
    for (const v of message.nodes) {
      ApprovalNode.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ApprovalStep {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApprovalStep();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.type = reader.int32() as any;
          break;
        case 2:
          message.nodes.push(ApprovalNode.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ApprovalStep {
    return {
      type: isSet(object.type) ? approvalStep_TypeFromJSON(object.type) : 0,
      nodes: Array.isArray(object?.nodes) ? object.nodes.map((e: any) => ApprovalNode.fromJSON(e)) : [],
    };
  },

  toJSON(message: ApprovalStep): unknown {
    const obj: any = {};
    message.type !== undefined && (obj.type = approvalStep_TypeToJSON(message.type));
    if (message.nodes) {
      obj.nodes = message.nodes.map((e) => e ? ApprovalNode.toJSON(e) : undefined);
    } else {
      obj.nodes = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<ApprovalStep>): ApprovalStep {
    const message = createBaseApprovalStep();
    message.type = object.type ?? 0;
    message.nodes = object.nodes?.map((e) => ApprovalNode.fromPartial(e)) || [];
    return message;
  },
};

function createBaseApprovalNode(): ApprovalNode {
  return { uid: "", status: 0, type: 0, roleValue: undefined };
}

export const ApprovalNode = {
  encode(message: ApprovalNode, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.uid !== "") {
      writer.uint32(10).string(message.uid);
    }
    if (message.status !== 0) {
      writer.uint32(16).int32(message.status);
    }
    if (message.type !== 0) {
      writer.uint32(24).int32(message.type);
    }
    if (message.roleValue !== undefined) {
      writer.uint32(32).int32(message.roleValue);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ApprovalNode {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApprovalNode();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.uid = reader.string();
          break;
        case 2:
          message.status = reader.int32() as any;
          break;
        case 3:
          message.type = reader.int32() as any;
          break;
        case 4:
          message.roleValue = reader.int32() as any;
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ApprovalNode {
    return {
      uid: isSet(object.uid) ? String(object.uid) : "",
      status: isSet(object.status) ? approvalNode_StatusFromJSON(object.status) : 0,
      type: isSet(object.type) ? approvalNode_TypeFromJSON(object.type) : 0,
      roleValue: isSet(object.roleValue) ? approvalNode_RoleValueFromJSON(object.roleValue) : undefined,
    };
  },

  toJSON(message: ApprovalNode): unknown {
    const obj: any = {};
    message.uid !== undefined && (obj.uid = message.uid);
    message.status !== undefined && (obj.status = approvalNode_StatusToJSON(message.status));
    message.type !== undefined && (obj.type = approvalNode_TypeToJSON(message.type));
    message.roleValue !== undefined &&
      (obj.roleValue = message.roleValue !== undefined ? approvalNode_RoleValueToJSON(message.roleValue) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<ApprovalNode>): ApprovalNode {
    const message = createBaseApprovalNode();
    message.uid = object.uid ?? "";
    message.status = object.status ?? 0;
    message.type = object.type ?? 0;
    message.roleValue = object.roleValue ?? undefined;
    return message;
  },
};

export type ApprovalServiceDefinition = typeof ApprovalServiceDefinition;
export const ApprovalServiceDefinition = {
  name: "ApprovalService",
  fullName: "bytebase.v1.ApprovalService",
  methods: {
    getApproval: {
      name: "GetApproval",
      requestType: GetApprovalRequest,
      requestStream: false,
      responseType: Approval,
      responseStream: false,
      options: {},
    },
    listApprovals: {
      name: "ListApprovals",
      requestType: ListApprovalsRequest,
      requestStream: false,
      responseType: ListApprovalsResponse,
      responseStream: false,
      options: {},
    },
    patchApprovalNodeStatus: {
      name: "PatchApprovalNodeStatus",
      requestType: PatchApprovalNodeStatusRequest,
      requestStream: false,
      responseType: Approval,
      responseStream: false,
      options: {},
    },
  },
} as const;

export interface ApprovalServiceImplementation<CallContextExt = {}> {
  getApproval(request: GetApprovalRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Approval>>;
  listApprovals(
    request: ListApprovalsRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ListApprovalsResponse>>;
  patchApprovalNodeStatus(
    request: PatchApprovalNodeStatusRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Approval>>;
}

export interface ApprovalServiceClient<CallOptionsExt = {}> {
  getApproval(request: DeepPartial<GetApprovalRequest>, options?: CallOptions & CallOptionsExt): Promise<Approval>;
  listApprovals(
    request: DeepPartial<ListApprovalsRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ListApprovalsResponse>;
  patchApprovalNodeStatus(
    request: DeepPartial<PatchApprovalNodeStatusRequest>,
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
