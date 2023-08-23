/* eslint-disable */
import * as _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

/**
 * IssuePayloadApproval is a part of the payload of an issue.
 * IssuePayloadApproval records the approval template used and the approval history.
 */
export interface IssuePayloadApproval {
  approvalTemplates: ApprovalTemplate[];
  approvers: IssuePayloadApproval_Approver[];
  /**
   * If the value is `false`, it means that the backend is still finding matching approval templates.
   * If `true`, other fields are available.
   */
  approvalFindingDone: boolean;
  approvalFindingError: string;
}

export interface IssuePayloadApproval_Approver {
  /** The new status. */
  status: IssuePayloadApproval_Approver_Status;
  /** The principal id of the approver. */
  principalId: number;
}

export enum IssuePayloadApproval_Approver_Status {
  STATUS_UNSPECIFIED = 0,
  PENDING = 1,
  APPROVED = 2,
  REJECTED = 3,
  UNRECOGNIZED = -1,
}

export function issuePayloadApproval_Approver_StatusFromJSON(object: any): IssuePayloadApproval_Approver_Status {
  switch (object) {
    case 0:
    case "STATUS_UNSPECIFIED":
      return IssuePayloadApproval_Approver_Status.STATUS_UNSPECIFIED;
    case 1:
    case "PENDING":
      return IssuePayloadApproval_Approver_Status.PENDING;
    case 2:
    case "APPROVED":
      return IssuePayloadApproval_Approver_Status.APPROVED;
    case 3:
    case "REJECTED":
      return IssuePayloadApproval_Approver_Status.REJECTED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return IssuePayloadApproval_Approver_Status.UNRECOGNIZED;
  }
}

export function issuePayloadApproval_Approver_StatusToJSON(object: IssuePayloadApproval_Approver_Status): string {
  switch (object) {
    case IssuePayloadApproval_Approver_Status.STATUS_UNSPECIFIED:
      return "STATUS_UNSPECIFIED";
    case IssuePayloadApproval_Approver_Status.PENDING:
      return "PENDING";
    case IssuePayloadApproval_Approver_Status.APPROVED:
      return "APPROVED";
    case IssuePayloadApproval_Approver_Status.REJECTED:
      return "REJECTED";
    case IssuePayloadApproval_Approver_Status.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface ApprovalTemplate {
  flow?: ApprovalFlow | undefined;
  title: string;
  description: string;
  creatorId: number;
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
 * ALL means every node must be approved to proceed.
 * ANY means approving any node will proceed.
 */
export enum ApprovalStep_Type {
  TYPE_UNSPECIFIED = 0,
  ALL = 1,
  ANY = 2,
  UNRECOGNIZED = -1,
}

export function approvalStep_TypeFromJSON(object: any): ApprovalStep_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return ApprovalStep_Type.TYPE_UNSPECIFIED;
    case 1:
    case "ALL":
      return ApprovalStep_Type.ALL;
    case 2:
    case "ANY":
      return ApprovalStep_Type.ANY;
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
    case ApprovalStep_Type.ALL:
      return "ALL";
    case ApprovalStep_Type.ANY:
      return "ANY";
    case ApprovalStep_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface ApprovalNode {
  type: ApprovalNode_Type;
  groupValue?:
    | ApprovalNode_GroupValue
    | undefined;
  /** Format: roles/{role} */
  role?: string | undefined;
  externalNodeId?: string | undefined;
}

/**
 * Type of the ApprovalNode.
 * type determines who should approve this node.
 * ANY_IN_GROUP means the ApprovalNode can be approved by an user from our predefined user group.
 * See GroupValue below for the predefined user groups.
 */
export enum ApprovalNode_Type {
  TYPE_UNSPECIFIED = 0,
  ANY_IN_GROUP = 1,
  UNRECOGNIZED = -1,
}

export function approvalNode_TypeFromJSON(object: any): ApprovalNode_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return ApprovalNode_Type.TYPE_UNSPECIFIED;
    case 1:
    case "ANY_IN_GROUP":
      return ApprovalNode_Type.ANY_IN_GROUP;
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
    case ApprovalNode_Type.ANY_IN_GROUP:
      return "ANY_IN_GROUP";
    case ApprovalNode_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

/**
 * The predefined user groups are:
 * - WORKSPACE_OWNER
 * - WORKSPACE_DBA
 * - PROJECT_OWNER
 * - PROJECT_MEMBER
 */
export enum ApprovalNode_GroupValue {
  GROUP_VALUE_UNSPECIFILED = 0,
  WORKSPACE_OWNER = 1,
  WORKSPACE_DBA = 2,
  PROJECT_OWNER = 3,
  PROJECT_MEMBER = 4,
  UNRECOGNIZED = -1,
}

export function approvalNode_GroupValueFromJSON(object: any): ApprovalNode_GroupValue {
  switch (object) {
    case 0:
    case "GROUP_VALUE_UNSPECIFILED":
      return ApprovalNode_GroupValue.GROUP_VALUE_UNSPECIFILED;
    case 1:
    case "WORKSPACE_OWNER":
      return ApprovalNode_GroupValue.WORKSPACE_OWNER;
    case 2:
    case "WORKSPACE_DBA":
      return ApprovalNode_GroupValue.WORKSPACE_DBA;
    case 3:
    case "PROJECT_OWNER":
      return ApprovalNode_GroupValue.PROJECT_OWNER;
    case 4:
    case "PROJECT_MEMBER":
      return ApprovalNode_GroupValue.PROJECT_MEMBER;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ApprovalNode_GroupValue.UNRECOGNIZED;
  }
}

export function approvalNode_GroupValueToJSON(object: ApprovalNode_GroupValue): string {
  switch (object) {
    case ApprovalNode_GroupValue.GROUP_VALUE_UNSPECIFILED:
      return "GROUP_VALUE_UNSPECIFILED";
    case ApprovalNode_GroupValue.WORKSPACE_OWNER:
      return "WORKSPACE_OWNER";
    case ApprovalNode_GroupValue.WORKSPACE_DBA:
      return "WORKSPACE_DBA";
    case ApprovalNode_GroupValue.PROJECT_OWNER:
      return "PROJECT_OWNER";
    case ApprovalNode_GroupValue.PROJECT_MEMBER:
      return "PROJECT_MEMBER";
    case ApprovalNode_GroupValue.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

function createBaseIssuePayloadApproval(): IssuePayloadApproval {
  return { approvalTemplates: [], approvers: [], approvalFindingDone: false, approvalFindingError: "" };
}

export const IssuePayloadApproval = {
  encode(message: IssuePayloadApproval, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.approvalTemplates) {
      ApprovalTemplate.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.approvers) {
      IssuePayloadApproval_Approver.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.approvalFindingDone === true) {
      writer.uint32(24).bool(message.approvalFindingDone);
    }
    if (message.approvalFindingError !== "") {
      writer.uint32(34).string(message.approvalFindingError);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IssuePayloadApproval {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIssuePayloadApproval();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.approvalTemplates.push(ApprovalTemplate.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.approvers.push(IssuePayloadApproval_Approver.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.approvalFindingDone = reader.bool();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.approvalFindingError = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): IssuePayloadApproval {
    return {
      approvalTemplates: Array.isArray(object?.approvalTemplates)
        ? object.approvalTemplates.map((e: any) => ApprovalTemplate.fromJSON(e))
        : [],
      approvers: Array.isArray(object?.approvers)
        ? object.approvers.map((e: any) => IssuePayloadApproval_Approver.fromJSON(e))
        : [],
      approvalFindingDone: isSet(object.approvalFindingDone) ? Boolean(object.approvalFindingDone) : false,
      approvalFindingError: isSet(object.approvalFindingError) ? String(object.approvalFindingError) : "",
    };
  },

  toJSON(message: IssuePayloadApproval): unknown {
    const obj: any = {};
    if (message.approvalTemplates?.length) {
      obj.approvalTemplates = message.approvalTemplates.map((e) => ApprovalTemplate.toJSON(e));
    }
    if (message.approvers?.length) {
      obj.approvers = message.approvers.map((e) => IssuePayloadApproval_Approver.toJSON(e));
    }
    if (message.approvalFindingDone === true) {
      obj.approvalFindingDone = message.approvalFindingDone;
    }
    if (message.approvalFindingError !== "") {
      obj.approvalFindingError = message.approvalFindingError;
    }
    return obj;
  },

  create(base?: DeepPartial<IssuePayloadApproval>): IssuePayloadApproval {
    return IssuePayloadApproval.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<IssuePayloadApproval>): IssuePayloadApproval {
    const message = createBaseIssuePayloadApproval();
    message.approvalTemplates = object.approvalTemplates?.map((e) => ApprovalTemplate.fromPartial(e)) || [];
    message.approvers = object.approvers?.map((e) => IssuePayloadApproval_Approver.fromPartial(e)) || [];
    message.approvalFindingDone = object.approvalFindingDone ?? false;
    message.approvalFindingError = object.approvalFindingError ?? "";
    return message;
  },
};

function createBaseIssuePayloadApproval_Approver(): IssuePayloadApproval_Approver {
  return { status: 0, principalId: 0 };
}

export const IssuePayloadApproval_Approver = {
  encode(message: IssuePayloadApproval_Approver, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.status !== 0) {
      writer.uint32(8).int32(message.status);
    }
    if (message.principalId !== 0) {
      writer.uint32(16).int32(message.principalId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IssuePayloadApproval_Approver {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIssuePayloadApproval_Approver();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.status = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.principalId = reader.int32();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): IssuePayloadApproval_Approver {
    return {
      status: isSet(object.status) ? issuePayloadApproval_Approver_StatusFromJSON(object.status) : 0,
      principalId: isSet(object.principalId) ? Number(object.principalId) : 0,
    };
  },

  toJSON(message: IssuePayloadApproval_Approver): unknown {
    const obj: any = {};
    if (message.status !== 0) {
      obj.status = issuePayloadApproval_Approver_StatusToJSON(message.status);
    }
    if (message.principalId !== 0) {
      obj.principalId = Math.round(message.principalId);
    }
    return obj;
  },

  create(base?: DeepPartial<IssuePayloadApproval_Approver>): IssuePayloadApproval_Approver {
    return IssuePayloadApproval_Approver.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<IssuePayloadApproval_Approver>): IssuePayloadApproval_Approver {
    const message = createBaseIssuePayloadApproval_Approver();
    message.status = object.status ?? 0;
    message.principalId = object.principalId ?? 0;
    return message;
  },
};

function createBaseApprovalTemplate(): ApprovalTemplate {
  return { flow: undefined, title: "", description: "", creatorId: 0 };
}

export const ApprovalTemplate = {
  encode(message: ApprovalTemplate, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.flow !== undefined) {
      ApprovalFlow.encode(message.flow, writer.uint32(10).fork()).ldelim();
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    if (message.creatorId !== 0) {
      writer.uint32(32).int32(message.creatorId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ApprovalTemplate {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApprovalTemplate();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.flow = ApprovalFlow.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.title = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.description = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.creatorId = reader.int32();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ApprovalTemplate {
    return {
      flow: isSet(object.flow) ? ApprovalFlow.fromJSON(object.flow) : undefined,
      title: isSet(object.title) ? String(object.title) : "",
      description: isSet(object.description) ? String(object.description) : "",
      creatorId: isSet(object.creatorId) ? Number(object.creatorId) : 0,
    };
  },

  toJSON(message: ApprovalTemplate): unknown {
    const obj: any = {};
    if (message.flow !== undefined) {
      obj.flow = ApprovalFlow.toJSON(message.flow);
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.description !== "") {
      obj.description = message.description;
    }
    if (message.creatorId !== 0) {
      obj.creatorId = Math.round(message.creatorId);
    }
    return obj;
  },

  create(base?: DeepPartial<ApprovalTemplate>): ApprovalTemplate {
    return ApprovalTemplate.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ApprovalTemplate>): ApprovalTemplate {
    const message = createBaseApprovalTemplate();
    message.flow = (object.flow !== undefined && object.flow !== null)
      ? ApprovalFlow.fromPartial(object.flow)
      : undefined;
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.creatorId = object.creatorId ?? 0;
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApprovalFlow();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.steps.push(ApprovalStep.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ApprovalFlow {
    return { steps: Array.isArray(object?.steps) ? object.steps.map((e: any) => ApprovalStep.fromJSON(e)) : [] };
  },

  toJSON(message: ApprovalFlow): unknown {
    const obj: any = {};
    if (message.steps?.length) {
      obj.steps = message.steps.map((e) => ApprovalStep.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<ApprovalFlow>): ApprovalFlow {
    return ApprovalFlow.fromPartial(base ?? {});
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApprovalStep();
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
          if (tag !== 18) {
            break;
          }

          message.nodes.push(ApprovalNode.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
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
    if (message.type !== 0) {
      obj.type = approvalStep_TypeToJSON(message.type);
    }
    if (message.nodes?.length) {
      obj.nodes = message.nodes.map((e) => ApprovalNode.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<ApprovalStep>): ApprovalStep {
    return ApprovalStep.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ApprovalStep>): ApprovalStep {
    const message = createBaseApprovalStep();
    message.type = object.type ?? 0;
    message.nodes = object.nodes?.map((e) => ApprovalNode.fromPartial(e)) || [];
    return message;
  },
};

function createBaseApprovalNode(): ApprovalNode {
  return { type: 0, groupValue: undefined, role: undefined, externalNodeId: undefined };
}

export const ApprovalNode = {
  encode(message: ApprovalNode, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.type !== 0) {
      writer.uint32(8).int32(message.type);
    }
    if (message.groupValue !== undefined) {
      writer.uint32(16).int32(message.groupValue);
    }
    if (message.role !== undefined) {
      writer.uint32(26).string(message.role);
    }
    if (message.externalNodeId !== undefined) {
      writer.uint32(34).string(message.externalNodeId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ApprovalNode {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApprovalNode();
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

          message.groupValue = reader.int32() as any;
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.role = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.externalNodeId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ApprovalNode {
    return {
      type: isSet(object.type) ? approvalNode_TypeFromJSON(object.type) : 0,
      groupValue: isSet(object.groupValue) ? approvalNode_GroupValueFromJSON(object.groupValue) : undefined,
      role: isSet(object.role) ? String(object.role) : undefined,
      externalNodeId: isSet(object.externalNodeId) ? String(object.externalNodeId) : undefined,
    };
  },

  toJSON(message: ApprovalNode): unknown {
    const obj: any = {};
    if (message.type !== 0) {
      obj.type = approvalNode_TypeToJSON(message.type);
    }
    if (message.groupValue !== undefined) {
      obj.groupValue = approvalNode_GroupValueToJSON(message.groupValue);
    }
    if (message.role !== undefined) {
      obj.role = message.role;
    }
    if (message.externalNodeId !== undefined) {
      obj.externalNodeId = message.externalNodeId;
    }
    return obj;
  },

  create(base?: DeepPartial<ApprovalNode>): ApprovalNode {
    return ApprovalNode.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ApprovalNode>): ApprovalNode {
    const message = createBaseApprovalNode();
    message.type = object.type ?? 0;
    message.groupValue = object.groupValue ?? undefined;
    message.role = object.role ?? undefined;
    message.externalNodeId = object.externalNodeId ?? undefined;
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
