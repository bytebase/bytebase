/* eslint-disable */
import * as _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export interface ApprovalFlow {
  steps: ApprovalStep[];
}

export interface ApprovalStep {
  type: ApprovalStep_Type;
  nodes: ApprovalNode[];
}

export enum ApprovalStep_Type {
  TYPE_UNSPECIFIED = 0,
  TYPE_AND = 1,
  TYPE_OR = 2,
  UNRECOGNIZED = -1,
}

export function approvalStep_TypeFromJSON(object: any): ApprovalStep_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return ApprovalStep_Type.TYPE_UNSPECIFIED;
    case 1:
    case "TYPE_AND":
      return ApprovalStep_Type.TYPE_AND;
    case 2:
    case "TYPE_OR":
      return ApprovalStep_Type.TYPE_OR;
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
    case ApprovalStep_Type.TYPE_AND:
      return "TYPE_AND";
    case ApprovalStep_Type.TYPE_OR:
      return "TYPE_OR";
    case ApprovalStep_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface ApprovalNode {
  /** id is a unique identifier of a node in a flow. */
  id: string;
  /** stauts of the node. */
  status: ApprovalNode_Status;
  /** type determines who should approve this node. */
  type: ApprovalNode_Type;
  rolePayload?: ApprovalNode_RolePayload | undefined;
}

export enum ApprovalNode_Status {
  STATUS_UNSPECIFIED = 0,
  STATUS_PENDING = 1,
  STATUS_APPROVED = 2,
  UNRECOGNIZED = -1,
}

export function approvalNode_StatusFromJSON(object: any): ApprovalNode_Status {
  switch (object) {
    case 0:
    case "STATUS_UNSPECIFIED":
      return ApprovalNode_Status.STATUS_UNSPECIFIED;
    case 1:
    case "STATUS_PENDING":
      return ApprovalNode_Status.STATUS_PENDING;
    case 2:
    case "STATUS_APPROVED":
      return ApprovalNode_Status.STATUS_APPROVED;
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
    case ApprovalNode_Status.STATUS_PENDING:
      return "STATUS_PENDING";
    case ApprovalNode_Status.STATUS_APPROVED:
      return "STATUS_APPROVED";
    case ApprovalNode_Status.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum ApprovalNode_Type {
  TYPE_UNSPECIFIED = 0,
  TYPE_ROLE = 1,
  UNRECOGNIZED = -1,
}

export function approvalNode_TypeFromJSON(object: any): ApprovalNode_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return ApprovalNode_Type.TYPE_UNSPECIFIED;
    case 1:
    case "TYPE_ROLE":
      return ApprovalNode_Type.TYPE_ROLE;
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
    case ApprovalNode_Type.TYPE_ROLE:
      return "TYPE_ROLE";
    case ApprovalNode_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum ApprovalNode_RoleValue {
  ROLE_VALUE_UNSPECIFILED = 0,
  ROLE_VALUE_WORKSPACE_OWNER = 1,
  ROLE_VALUE_DBA = 2,
  ROLE_VALUE_PROJECT_OWNER = 3,
  ROLE_VALUE_PROJECT_MEMBER = 4,
  UNRECOGNIZED = -1,
}

export function approvalNode_RoleValueFromJSON(object: any): ApprovalNode_RoleValue {
  switch (object) {
    case 0:
    case "ROLE_VALUE_UNSPECIFILED":
      return ApprovalNode_RoleValue.ROLE_VALUE_UNSPECIFILED;
    case 1:
    case "ROLE_VALUE_WORKSPACE_OWNER":
      return ApprovalNode_RoleValue.ROLE_VALUE_WORKSPACE_OWNER;
    case 2:
    case "ROLE_VALUE_DBA":
      return ApprovalNode_RoleValue.ROLE_VALUE_DBA;
    case 3:
    case "ROLE_VALUE_PROJECT_OWNER":
      return ApprovalNode_RoleValue.ROLE_VALUE_PROJECT_OWNER;
    case 4:
    case "ROLE_VALUE_PROJECT_MEMBER":
      return ApprovalNode_RoleValue.ROLE_VALUE_PROJECT_MEMBER;
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
    case ApprovalNode_RoleValue.ROLE_VALUE_WORKSPACE_OWNER:
      return "ROLE_VALUE_WORKSPACE_OWNER";
    case ApprovalNode_RoleValue.ROLE_VALUE_DBA:
      return "ROLE_VALUE_DBA";
    case ApprovalNode_RoleValue.ROLE_VALUE_PROJECT_OWNER:
      return "ROLE_VALUE_PROJECT_OWNER";
    case ApprovalNode_RoleValue.ROLE_VALUE_PROJECT_MEMBER:
      return "ROLE_VALUE_PROJECT_MEMBER";
    case ApprovalNode_RoleValue.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface ApprovalNode_RolePayload {
  roleValue: ApprovalNode_RoleValue;
}

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
  return { id: "", status: 0, type: 0, rolePayload: undefined };
}

export const ApprovalNode = {
  encode(message: ApprovalNode, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.status !== 0) {
      writer.uint32(16).int32(message.status);
    }
    if (message.type !== 0) {
      writer.uint32(24).int32(message.type);
    }
    if (message.rolePayload !== undefined) {
      ApprovalNode_RolePayload.encode(message.rolePayload, writer.uint32(34).fork()).ldelim();
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
          message.id = reader.string();
          break;
        case 2:
          message.status = reader.int32() as any;
          break;
        case 3:
          message.type = reader.int32() as any;
          break;
        case 4:
          message.rolePayload = ApprovalNode_RolePayload.decode(reader, reader.uint32());
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
      id: isSet(object.id) ? String(object.id) : "",
      status: isSet(object.status) ? approvalNode_StatusFromJSON(object.status) : 0,
      type: isSet(object.type) ? approvalNode_TypeFromJSON(object.type) : 0,
      rolePayload: isSet(object.rolePayload) ? ApprovalNode_RolePayload.fromJSON(object.rolePayload) : undefined,
    };
  },

  toJSON(message: ApprovalNode): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.status !== undefined && (obj.status = approvalNode_StatusToJSON(message.status));
    message.type !== undefined && (obj.type = approvalNode_TypeToJSON(message.type));
    message.rolePayload !== undefined &&
      (obj.rolePayload = message.rolePayload ? ApprovalNode_RolePayload.toJSON(message.rolePayload) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<ApprovalNode>): ApprovalNode {
    const message = createBaseApprovalNode();
    message.id = object.id ?? "";
    message.status = object.status ?? 0;
    message.type = object.type ?? 0;
    message.rolePayload = (object.rolePayload !== undefined && object.rolePayload !== null)
      ? ApprovalNode_RolePayload.fromPartial(object.rolePayload)
      : undefined;
    return message;
  },
};

function createBaseApprovalNode_RolePayload(): ApprovalNode_RolePayload {
  return { roleValue: 0 };
}

export const ApprovalNode_RolePayload = {
  encode(message: ApprovalNode_RolePayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.roleValue !== 0) {
      writer.uint32(8).int32(message.roleValue);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ApprovalNode_RolePayload {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApprovalNode_RolePayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.roleValue = reader.int32() as any;
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ApprovalNode_RolePayload {
    return { roleValue: isSet(object.roleValue) ? approvalNode_RoleValueFromJSON(object.roleValue) : 0 };
  },

  toJSON(message: ApprovalNode_RolePayload): unknown {
    const obj: any = {};
    message.roleValue !== undefined && (obj.roleValue = approvalNode_RoleValueToJSON(message.roleValue));
    return obj;
  },

  fromPartial(object: DeepPartial<ApprovalNode_RolePayload>): ApprovalNode_RolePayload {
    const message = createBaseApprovalNode_RolePayload();
    message.roleValue = object.roleValue ?? 0;
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
