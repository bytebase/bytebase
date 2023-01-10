/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { State, stateFromJSON, stateToJSON } from "./common";

export const protobufPackage = "bytebase.v1";

export enum EnvironmentTier {
  ENVIRONMENT_TIER_UNSPECIFIED = 0,
  PROTECTED = 1,
  UNPROTECTED = 2,
  UNRECOGNIZED = -1,
}

export function environmentTierFromJSON(object: any): EnvironmentTier {
  switch (object) {
    case 0:
    case "ENVIRONMENT_TIER_UNSPECIFIED":
      return EnvironmentTier.ENVIRONMENT_TIER_UNSPECIFIED;
    case 1:
    case "PROTECTED":
      return EnvironmentTier.PROTECTED;
    case 2:
    case "UNPROTECTED":
      return EnvironmentTier.UNPROTECTED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return EnvironmentTier.UNRECOGNIZED;
  }
}

export function environmentTierToJSON(object: EnvironmentTier): string {
  switch (object) {
    case EnvironmentTier.ENVIRONMENT_TIER_UNSPECIFIED:
      return "ENVIRONMENT_TIER_UNSPECIFIED";
    case EnvironmentTier.PROTECTED:
      return "PROTECTED";
    case EnvironmentTier.UNPROTECTED:
      return "UNPROTECTED";
    case EnvironmentTier.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface GetEnvironmentRequest {
  /**
   * The name of the environment to retrieve.
   * Format: environments/{environment}
   */
  name: string;
}

export interface ListEnvironmentsRequest {
  /**
   * The maximum number of environments to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 environments will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListEnvironments` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListEnvironments` must match
   * the call that provided the page token.
   */
  pageToken: string;
  /** Show deleted environments if specified. */
  showDeleted: boolean;
}

export interface ListEnvironmentsResponse {
  /** The environments from the specified request. */
  environments: Environment[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface CreateEnvironmentRequest {
  /** The environment to create. */
  environment?: Environment;
  /**
   * The ID to use for the environment, which will become the final component of
   * the environment's resource name.
   *
   * This value should be 4-63 characters, and valid characters
   * are /[a-z][0-9]-/.
   */
  environmentId: string;
}

export interface UpdateEnvironmentRequest {
  /**
   * The environment to update.
   *
   * The environment's `name` field is used to identify the environment to update.
   * Format: environments/{environment}
   */
  environment?: Environment;
  /** The list of fields to update. */
  updateMask?: string[];
}

export interface DeleteEnvironmentRequest {
  /**
   * The name of the environment to delete.
   * Format: environments/{environment}
   */
  name: string;
}

export interface UndeleteEnvironmentRequest {
  /**
   * The name of the deleted environment.
   * Format: environments/{environment}
   */
  name: string;
}

export interface Environment {
  /**
   * The name of the environment.
   * Format: environments/{environment}
   */
  name: string;
  /** The system-assigned, unique identifier for a resource. */
  uid: string;
  state: State;
  title: string;
  order: number;
  tier: EnvironmentTier;
}

function createBaseGetEnvironmentRequest(): GetEnvironmentRequest {
  return { name: "" };
}

export const GetEnvironmentRequest = {
  encode(message: GetEnvironmentRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetEnvironmentRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetEnvironmentRequest();
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

  fromJSON(object: any): GetEnvironmentRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetEnvironmentRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<GetEnvironmentRequest>): GetEnvironmentRequest {
    const message = createBaseGetEnvironmentRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListEnvironmentsRequest(): ListEnvironmentsRequest {
  return { pageSize: 0, pageToken: "", showDeleted: false };
}

export const ListEnvironmentsRequest = {
  encode(message: ListEnvironmentsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): ListEnvironmentsRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListEnvironmentsRequest();
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

  fromJSON(object: any): ListEnvironmentsRequest {
    return {
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
      showDeleted: isSet(object.showDeleted) ? Boolean(object.showDeleted) : false,
    };
  },

  toJSON(message: ListEnvironmentsRequest): unknown {
    const obj: any = {};
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    message.showDeleted !== undefined && (obj.showDeleted = message.showDeleted);
    return obj;
  },

  fromPartial(object: DeepPartial<ListEnvironmentsRequest>): ListEnvironmentsRequest {
    const message = createBaseListEnvironmentsRequest();
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    message.showDeleted = object.showDeleted ?? false;
    return message;
  },
};

function createBaseListEnvironmentsResponse(): ListEnvironmentsResponse {
  return { environments: [], nextPageToken: "" };
}

export const ListEnvironmentsResponse = {
  encode(message: ListEnvironmentsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.environments) {
      Environment.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListEnvironmentsResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListEnvironmentsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.environments.push(Environment.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListEnvironmentsResponse {
    return {
      environments: Array.isArray(object?.environments)
        ? object.environments.map((e: any) => Environment.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListEnvironmentsResponse): unknown {
    const obj: any = {};
    if (message.environments) {
      obj.environments = message.environments.map((e) => e ? Environment.toJSON(e) : undefined);
    } else {
      obj.environments = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  fromPartial(object: DeepPartial<ListEnvironmentsResponse>): ListEnvironmentsResponse {
    const message = createBaseListEnvironmentsResponse();
    message.environments = object.environments?.map((e) => Environment.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseCreateEnvironmentRequest(): CreateEnvironmentRequest {
  return { environment: undefined, environmentId: "" };
}

export const CreateEnvironmentRequest = {
  encode(message: CreateEnvironmentRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.environment !== undefined) {
      Environment.encode(message.environment, writer.uint32(10).fork()).ldelim();
    }
    if (message.environmentId !== "") {
      writer.uint32(18).string(message.environmentId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateEnvironmentRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateEnvironmentRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.environment = Environment.decode(reader, reader.uint32());
          break;
        case 2:
          message.environmentId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): CreateEnvironmentRequest {
    return {
      environment: isSet(object.environment) ? Environment.fromJSON(object.environment) : undefined,
      environmentId: isSet(object.environmentId) ? String(object.environmentId) : "",
    };
  },

  toJSON(message: CreateEnvironmentRequest): unknown {
    const obj: any = {};
    message.environment !== undefined &&
      (obj.environment = message.environment ? Environment.toJSON(message.environment) : undefined);
    message.environmentId !== undefined && (obj.environmentId = message.environmentId);
    return obj;
  },

  fromPartial(object: DeepPartial<CreateEnvironmentRequest>): CreateEnvironmentRequest {
    const message = createBaseCreateEnvironmentRequest();
    message.environment = (object.environment !== undefined && object.environment !== null)
      ? Environment.fromPartial(object.environment)
      : undefined;
    message.environmentId = object.environmentId ?? "";
    return message;
  },
};

function createBaseUpdateEnvironmentRequest(): UpdateEnvironmentRequest {
  return { environment: undefined, updateMask: undefined };
}

export const UpdateEnvironmentRequest = {
  encode(message: UpdateEnvironmentRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.environment !== undefined) {
      Environment.encode(message.environment, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateEnvironmentRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateEnvironmentRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.environment = Environment.decode(reader, reader.uint32());
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

  fromJSON(object: any): UpdateEnvironmentRequest {
    return {
      environment: isSet(object.environment) ? Environment.fromJSON(object.environment) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateEnvironmentRequest): unknown {
    const obj: any = {};
    message.environment !== undefined &&
      (obj.environment = message.environment ? Environment.toJSON(message.environment) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  fromPartial(object: DeepPartial<UpdateEnvironmentRequest>): UpdateEnvironmentRequest {
    const message = createBaseUpdateEnvironmentRequest();
    message.environment = (object.environment !== undefined && object.environment !== null)
      ? Environment.fromPartial(object.environment)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseDeleteEnvironmentRequest(): DeleteEnvironmentRequest {
  return { name: "" };
}

export const DeleteEnvironmentRequest = {
  encode(message: DeleteEnvironmentRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteEnvironmentRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteEnvironmentRequest();
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

  fromJSON(object: any): DeleteEnvironmentRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: DeleteEnvironmentRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<DeleteEnvironmentRequest>): DeleteEnvironmentRequest {
    const message = createBaseDeleteEnvironmentRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseUndeleteEnvironmentRequest(): UndeleteEnvironmentRequest {
  return { name: "" };
}

export const UndeleteEnvironmentRequest = {
  encode(message: UndeleteEnvironmentRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UndeleteEnvironmentRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUndeleteEnvironmentRequest();
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

  fromJSON(object: any): UndeleteEnvironmentRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: UndeleteEnvironmentRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<UndeleteEnvironmentRequest>): UndeleteEnvironmentRequest {
    const message = createBaseUndeleteEnvironmentRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseEnvironment(): Environment {
  return { name: "", uid: "", state: 0, title: "", order: 0, tier: 0 };
}

export const Environment = {
  encode(message: Environment, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.uid !== "") {
      writer.uint32(18).string(message.uid);
    }
    if (message.state !== 0) {
      writer.uint32(24).int32(message.state);
    }
    if (message.title !== "") {
      writer.uint32(34).string(message.title);
    }
    if (message.order !== 0) {
      writer.uint32(40).int32(message.order);
    }
    if (message.tier !== 0) {
      writer.uint32(48).int32(message.tier);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Environment {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEnvironment();
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
          message.state = reader.int32() as any;
          break;
        case 4:
          message.title = reader.string();
          break;
        case 5:
          message.order = reader.int32();
          break;
        case 6:
          message.tier = reader.int32() as any;
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Environment {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      uid: isSet(object.uid) ? String(object.uid) : "",
      state: isSet(object.state) ? stateFromJSON(object.state) : 0,
      title: isSet(object.title) ? String(object.title) : "",
      order: isSet(object.order) ? Number(object.order) : 0,
      tier: isSet(object.tier) ? environmentTierFromJSON(object.tier) : 0,
    };
  },

  toJSON(message: Environment): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.state !== undefined && (obj.state = stateToJSON(message.state));
    message.title !== undefined && (obj.title = message.title);
    message.order !== undefined && (obj.order = Math.round(message.order));
    message.tier !== undefined && (obj.tier = environmentTierToJSON(message.tier));
    return obj;
  },

  fromPartial(object: DeepPartial<Environment>): Environment {
    const message = createBaseEnvironment();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.state = object.state ?? 0;
    message.title = object.title ?? "";
    message.order = object.order ?? 0;
    message.tier = object.tier ?? 0;
    return message;
  },
};

export type EnvironmentServiceDefinition = typeof EnvironmentServiceDefinition;
export const EnvironmentServiceDefinition = {
  name: "EnvironmentService",
  fullName: "bytebase.v1.EnvironmentService",
  methods: {
    getEnvironment: {
      name: "GetEnvironment",
      requestType: GetEnvironmentRequest,
      requestStream: false,
      responseType: Environment,
      responseStream: false,
      options: {},
    },
    listEnvironments: {
      name: "ListEnvironments",
      requestType: ListEnvironmentsRequest,
      requestStream: false,
      responseType: ListEnvironmentsResponse,
      responseStream: false,
      options: {},
    },
    createEnvironment: {
      name: "CreateEnvironment",
      requestType: CreateEnvironmentRequest,
      requestStream: false,
      responseType: Environment,
      responseStream: false,
      options: {},
    },
    updateEnvironment: {
      name: "UpdateEnvironment",
      requestType: UpdateEnvironmentRequest,
      requestStream: false,
      responseType: Environment,
      responseStream: false,
      options: {},
    },
    deleteEnvironment: {
      name: "DeleteEnvironment",
      requestType: DeleteEnvironmentRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {},
    },
    undeleteEnvironment: {
      name: "UndeleteEnvironment",
      requestType: UndeleteEnvironmentRequest,
      requestStream: false,
      responseType: Environment,
      responseStream: false,
      options: {},
    },
  },
} as const;

export interface EnvironmentServiceImplementation<CallContextExt = {}> {
  getEnvironment(
    request: GetEnvironmentRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Environment>>;
  listEnvironments(
    request: ListEnvironmentsRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ListEnvironmentsResponse>>;
  createEnvironment(
    request: CreateEnvironmentRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Environment>>;
  updateEnvironment(
    request: UpdateEnvironmentRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Environment>>;
  deleteEnvironment(
    request: DeleteEnvironmentRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Empty>>;
  undeleteEnvironment(
    request: UndeleteEnvironmentRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Environment>>;
}

export interface EnvironmentServiceClient<CallOptionsExt = {}> {
  getEnvironment(
    request: DeepPartial<GetEnvironmentRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Environment>;
  listEnvironments(
    request: DeepPartial<ListEnvironmentsRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ListEnvironmentsResponse>;
  createEnvironment(
    request: DeepPartial<CreateEnvironmentRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Environment>;
  updateEnvironment(
    request: DeepPartial<UpdateEnvironmentRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Environment>;
  deleteEnvironment(
    request: DeepPartial<DeleteEnvironmentRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Empty>;
  undeleteEnvironment(
    request: DeepPartial<UndeleteEnvironmentRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Environment>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
