/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";

export const protobufPackage = "bytebase.v1";

export interface GetApprovalTemplateRequest {
  /**
   * The name of the instance to retrieve.
   * Format: approvalTemplates/{approvalTemplate}
   */
  name: string;
}

export interface ListApprovalTemplatesRequest {
  /**
   * The maximum number of approval templates to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 projects will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListApprovalTemplates` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListApprovalTemplates` must match
   * the call that provided the page token.
   */
  pageToken: string;
  /** Show deleted projects if specified. */
  showDeleted: boolean;
}

export interface ListApprovalTemplatesResponse {
  /** The approval templates from the specified request. */
  approvalTemplates: ApprovalTemplate[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface CreateApprovalTemplateRequest {
  /** the approval template to be created */
  approvalTemplate?: ApprovalTemplate;
}

export interface UpdateApprovalTemplateRequest {
  approvalTemplate?: ApprovalTemplate;
  /** The list of fields to update. */
  updateMask?: string[];
}

export interface DeleteApprovalTemplateRequest {
  /**
   * The name of the instance to delete.
   * Format: approvalTemplates/{approvalTemplate}
   */
  name: string;
}

export interface ApprovalTemplate {
  /** Format: approvalTemplates/{approvalTemplate} */
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

function createBaseGetApprovalTemplateRequest(): GetApprovalTemplateRequest {
  return { name: "" };
}

export const GetApprovalTemplateRequest = {
  encode(message: GetApprovalTemplateRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetApprovalTemplateRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetApprovalTemplateRequest();
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

  fromJSON(object: any): GetApprovalTemplateRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetApprovalTemplateRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<GetApprovalTemplateRequest>): GetApprovalTemplateRequest {
    const message = createBaseGetApprovalTemplateRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListApprovalTemplatesRequest(): ListApprovalTemplatesRequest {
  return { pageSize: 0, pageToken: "", showDeleted: false };
}

export const ListApprovalTemplatesRequest = {
  encode(message: ListApprovalTemplatesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pageSize !== 0) {
      writer.uint32(8).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(18).string(message.pageToken);
    }
    if (message.showDeleted === true) {
      writer.uint32(24).bool(message.showDeleted);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListApprovalTemplatesRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListApprovalTemplatesRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pageSize = reader.int32();
          break;
        case 2:
          message.pageToken = reader.string();
          break;
        case 3:
          message.showDeleted = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ListApprovalTemplatesRequest {
    return {
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
      showDeleted: isSet(object.showDeleted) ? Boolean(object.showDeleted) : false,
    };
  },

  toJSON(message: ListApprovalTemplatesRequest): unknown {
    const obj: any = {};
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    message.showDeleted !== undefined && (obj.showDeleted = message.showDeleted);
    return obj;
  },

  fromPartial(object: DeepPartial<ListApprovalTemplatesRequest>): ListApprovalTemplatesRequest {
    const message = createBaseListApprovalTemplatesRequest();
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    message.showDeleted = object.showDeleted ?? false;
    return message;
  },
};

function createBaseListApprovalTemplatesResponse(): ListApprovalTemplatesResponse {
  return { approvalTemplates: [], nextPageToken: "" };
}

export const ListApprovalTemplatesResponse = {
  encode(message: ListApprovalTemplatesResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.approvalTemplates) {
      ApprovalTemplate.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListApprovalTemplatesResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListApprovalTemplatesResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.approvalTemplates.push(ApprovalTemplate.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListApprovalTemplatesResponse {
    return {
      approvalTemplates: Array.isArray(object?.approvalTemplates)
        ? object.approvalTemplates.map((e: any) => ApprovalTemplate.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListApprovalTemplatesResponse): unknown {
    const obj: any = {};
    if (message.approvalTemplates) {
      obj.approvalTemplates = message.approvalTemplates.map((e) => e ? ApprovalTemplate.toJSON(e) : undefined);
    } else {
      obj.approvalTemplates = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  fromPartial(object: DeepPartial<ListApprovalTemplatesResponse>): ListApprovalTemplatesResponse {
    const message = createBaseListApprovalTemplatesResponse();
    message.approvalTemplates = object.approvalTemplates?.map((e) => ApprovalTemplate.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseCreateApprovalTemplateRequest(): CreateApprovalTemplateRequest {
  return { approvalTemplate: undefined };
}

export const CreateApprovalTemplateRequest = {
  encode(message: CreateApprovalTemplateRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.approvalTemplate !== undefined) {
      ApprovalTemplate.encode(message.approvalTemplate, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateApprovalTemplateRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateApprovalTemplateRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.approvalTemplate = ApprovalTemplate.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): CreateApprovalTemplateRequest {
    return {
      approvalTemplate: isSet(object.approvalTemplate) ? ApprovalTemplate.fromJSON(object.approvalTemplate) : undefined,
    };
  },

  toJSON(message: CreateApprovalTemplateRequest): unknown {
    const obj: any = {};
    message.approvalTemplate !== undefined &&
      (obj.approvalTemplate = message.approvalTemplate ? ApprovalTemplate.toJSON(message.approvalTemplate) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<CreateApprovalTemplateRequest>): CreateApprovalTemplateRequest {
    const message = createBaseCreateApprovalTemplateRequest();
    message.approvalTemplate = (object.approvalTemplate !== undefined && object.approvalTemplate !== null)
      ? ApprovalTemplate.fromPartial(object.approvalTemplate)
      : undefined;
    return message;
  },
};

function createBaseUpdateApprovalTemplateRequest(): UpdateApprovalTemplateRequest {
  return { approvalTemplate: undefined, updateMask: undefined };
}

export const UpdateApprovalTemplateRequest = {
  encode(message: UpdateApprovalTemplateRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.approvalTemplate !== undefined) {
      ApprovalTemplate.encode(message.approvalTemplate, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateApprovalTemplateRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateApprovalTemplateRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.approvalTemplate = ApprovalTemplate.decode(reader, reader.uint32());
          break;
        case 2:
          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): UpdateApprovalTemplateRequest {
    return {
      approvalTemplate: isSet(object.approvalTemplate) ? ApprovalTemplate.fromJSON(object.approvalTemplate) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateApprovalTemplateRequest): unknown {
    const obj: any = {};
    message.approvalTemplate !== undefined &&
      (obj.approvalTemplate = message.approvalTemplate ? ApprovalTemplate.toJSON(message.approvalTemplate) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  fromPartial(object: DeepPartial<UpdateApprovalTemplateRequest>): UpdateApprovalTemplateRequest {
    const message = createBaseUpdateApprovalTemplateRequest();
    message.approvalTemplate = (object.approvalTemplate !== undefined && object.approvalTemplate !== null)
      ? ApprovalTemplate.fromPartial(object.approvalTemplate)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseDeleteApprovalTemplateRequest(): DeleteApprovalTemplateRequest {
  return { name: "" };
}

export const DeleteApprovalTemplateRequest = {
  encode(message: DeleteApprovalTemplateRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteApprovalTemplateRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteApprovalTemplateRequest();
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

  fromJSON(object: any): DeleteApprovalTemplateRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: DeleteApprovalTemplateRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<DeleteApprovalTemplateRequest>): DeleteApprovalTemplateRequest {
    const message = createBaseDeleteApprovalTemplateRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseApprovalTemplate(): ApprovalTemplate {
  return { name: "", uid: "", flow: undefined };
}

export const ApprovalTemplate = {
  encode(message: ApprovalTemplate, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): ApprovalTemplate {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApprovalTemplate();
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

  fromJSON(object: any): ApprovalTemplate {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      uid: isSet(object.uid) ? String(object.uid) : "",
      flow: isSet(object.flow) ? ApprovalFlow.fromJSON(object.flow) : undefined,
    };
  },

  toJSON(message: ApprovalTemplate): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.flow !== undefined && (obj.flow = message.flow ? ApprovalFlow.toJSON(message.flow) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<ApprovalTemplate>): ApprovalTemplate {
    const message = createBaseApprovalTemplate();
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

export type ApprovalTemplateServiceDefinition = typeof ApprovalTemplateServiceDefinition;
export const ApprovalTemplateServiceDefinition = {
  name: "ApprovalTemplateService",
  fullName: "bytebase.v1.ApprovalTemplateService",
  methods: {
    getApprovalTemplate: {
      name: "GetApprovalTemplate",
      requestType: GetApprovalTemplateRequest,
      requestStream: false,
      responseType: ApprovalTemplate,
      responseStream: false,
      options: {},
    },
    listApprovalTemplates: {
      name: "ListApprovalTemplates",
      requestType: ListApprovalTemplatesRequest,
      requestStream: false,
      responseType: ListApprovalTemplatesResponse,
      responseStream: false,
      options: {},
    },
    createApprovalTemplate: {
      name: "CreateApprovalTemplate",
      requestType: CreateApprovalTemplateRequest,
      requestStream: false,
      responseType: ApprovalTemplate,
      responseStream: false,
      options: {},
    },
    updateApprovalTemplate: {
      name: "UpdateApprovalTemplate",
      requestType: UpdateApprovalTemplateRequest,
      requestStream: false,
      responseType: ApprovalTemplate,
      responseStream: false,
      options: {},
    },
    deleteApprovalTemplate: {
      name: "DeleteApprovalTemplate",
      requestType: DeleteApprovalTemplateRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {},
    },
  },
} as const;

export interface ApprovalTemplateServiceImplementation<CallContextExt = {}> {
  getApprovalTemplate(
    request: GetApprovalTemplateRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ApprovalTemplate>>;
  listApprovalTemplates(
    request: ListApprovalTemplatesRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ListApprovalTemplatesResponse>>;
  createApprovalTemplate(
    request: CreateApprovalTemplateRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ApprovalTemplate>>;
  updateApprovalTemplate(
    request: UpdateApprovalTemplateRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ApprovalTemplate>>;
  deleteApprovalTemplate(
    request: DeleteApprovalTemplateRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Empty>>;
}

export interface ApprovalTemplateServiceClient<CallOptionsExt = {}> {
  getApprovalTemplate(
    request: DeepPartial<GetApprovalTemplateRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ApprovalTemplate>;
  listApprovalTemplates(
    request: DeepPartial<ListApprovalTemplatesRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ListApprovalTemplatesResponse>;
  createApprovalTemplate(
    request: DeepPartial<CreateApprovalTemplateRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ApprovalTemplate>;
  updateApprovalTemplate(
    request: DeepPartial<UpdateApprovalTemplateRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ApprovalTemplate>;
  deleteApprovalTemplate(
    request: DeepPartial<DeleteApprovalTemplateRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Empty>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
