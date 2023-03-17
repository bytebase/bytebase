/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { ApprovalFlow } from "./approval_service";

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
