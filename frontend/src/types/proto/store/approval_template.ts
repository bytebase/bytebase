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
  type: ApprovalNode_Type;
  roleValue?: ApprovalNode_RoleValue | undefined;
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
  return { uid: "", type: 0, roleValue: undefined };
}

export const ApprovalNode = {
  encode(message: ApprovalNode, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.uid !== "") {
      writer.uint32(10).string(message.uid);
    }
    if (message.type !== 0) {
      writer.uint32(16).int32(message.type);
    }
    if (message.roleValue !== undefined) {
      writer.uint32(24).int32(message.roleValue);
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
          message.type = reader.int32() as any;
          break;
        case 3:
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
      type: isSet(object.type) ? approvalNode_TypeFromJSON(object.type) : 0,
      roleValue: isSet(object.roleValue) ? approvalNode_RoleValueFromJSON(object.roleValue) : undefined,
    };
  },

  toJSON(message: ApprovalNode): unknown {
    const obj: any = {};
    message.uid !== undefined && (obj.uid = message.uid);
    message.type !== undefined && (obj.type = approvalNode_TypeToJSON(message.type));
    message.roleValue !== undefined &&
      (obj.roleValue = message.roleValue !== undefined ? approvalNode_RoleValueToJSON(message.roleValue) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<ApprovalNode>): ApprovalNode {
    const message = createBaseApprovalNode();
    message.uid = object.uid ?? "";
    message.type = object.type ?? 0;
    message.roleValue = object.roleValue ?? undefined;
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
