/* eslint-disable */
import _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { State, stateFromJSON, stateToJSON } from "./common";

export const protobufPackage = "bytebase.v1";

export interface GetEnvironmentRequest {
  /**
   * The name of the environment to retrieve.
   * Format: environments/{environment}
   */
  name: string;
  /** Show deleted environment if specified. */
  showDeleted: boolean;
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
}

function createBaseGetEnvironmentRequest(): GetEnvironmentRequest {
  return { name: "", showDeleted: false };
}

export const GetEnvironmentRequest = {
  encode(message: GetEnvironmentRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.showDeleted === true) {
      writer.uint32(16).bool(message.showDeleted);
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
        case 2:
          message.showDeleted = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GetEnvironmentRequest {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      showDeleted: isSet(object.showDeleted) ? Boolean(object.showDeleted) : false,
    };
  },

  toJSON(message: GetEnvironmentRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.showDeleted !== undefined && (obj.showDeleted = message.showDeleted);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<GetEnvironmentRequest>, I>>(object: I): GetEnvironmentRequest {
    const message = createBaseGetEnvironmentRequest();
    message.name = object.name ?? "";
    message.showDeleted = object.showDeleted ?? false;
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

  fromPartial<I extends Exact<DeepPartial<ListEnvironmentsRequest>, I>>(object: I): ListEnvironmentsRequest {
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

  fromPartial<I extends Exact<DeepPartial<ListEnvironmentsResponse>, I>>(object: I): ListEnvironmentsResponse {
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

  fromPartial<I extends Exact<DeepPartial<CreateEnvironmentRequest>, I>>(object: I): CreateEnvironmentRequest {
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

  fromPartial<I extends Exact<DeepPartial<UpdateEnvironmentRequest>, I>>(object: I): UpdateEnvironmentRequest {
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

  fromPartial<I extends Exact<DeepPartial<DeleteEnvironmentRequest>, I>>(object: I): DeleteEnvironmentRequest {
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

  fromPartial<I extends Exact<DeepPartial<UndeleteEnvironmentRequest>, I>>(object: I): UndeleteEnvironmentRequest {
    const message = createBaseUndeleteEnvironmentRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseEnvironment(): Environment {
  return { name: "", uid: "", state: 0, title: "", order: 0 };
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
    };
  },

  toJSON(message: Environment): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.state !== undefined && (obj.state = stateToJSON(message.state));
    message.title !== undefined && (obj.title = message.title);
    message.order !== undefined && (obj.order = Math.round(message.order));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<Environment>, I>>(object: I): Environment {
    const message = createBaseEnvironment();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.state = object.state ?? 0;
    message.title = object.title ?? "";
    message.order = object.order ?? 0;
    return message;
  },
};

export interface EnvironmentService {
  GetEnvironment(request: GetEnvironmentRequest): Promise<Environment>;
  ListEnvironments(request: ListEnvironmentsRequest): Promise<ListEnvironmentsResponse>;
  CreateEnvironment(request: CreateEnvironmentRequest): Promise<Environment>;
  UpdateEnvironment(request: UpdateEnvironmentRequest): Promise<Environment>;
  DeleteEnvironment(request: DeleteEnvironmentRequest): Promise<Empty>;
  UndeleteEnvironment(request: UndeleteEnvironmentRequest): Promise<Environment>;
}

export class EnvironmentServiceClientImpl implements EnvironmentService {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "bytebase.v1.EnvironmentService";
    this.rpc = rpc;
    this.GetEnvironment = this.GetEnvironment.bind(this);
    this.ListEnvironments = this.ListEnvironments.bind(this);
    this.CreateEnvironment = this.CreateEnvironment.bind(this);
    this.UpdateEnvironment = this.UpdateEnvironment.bind(this);
    this.DeleteEnvironment = this.DeleteEnvironment.bind(this);
    this.UndeleteEnvironment = this.UndeleteEnvironment.bind(this);
  }
  GetEnvironment(request: GetEnvironmentRequest): Promise<Environment> {
    const data = GetEnvironmentRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "GetEnvironment", data);
    return promise.then((data) => Environment.decode(new _m0.Reader(data)));
  }

  ListEnvironments(request: ListEnvironmentsRequest): Promise<ListEnvironmentsResponse> {
    const data = ListEnvironmentsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "ListEnvironments", data);
    return promise.then((data) => ListEnvironmentsResponse.decode(new _m0.Reader(data)));
  }

  CreateEnvironment(request: CreateEnvironmentRequest): Promise<Environment> {
    const data = CreateEnvironmentRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "CreateEnvironment", data);
    return promise.then((data) => Environment.decode(new _m0.Reader(data)));
  }

  UpdateEnvironment(request: UpdateEnvironmentRequest): Promise<Environment> {
    const data = UpdateEnvironmentRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "UpdateEnvironment", data);
    return promise.then((data) => Environment.decode(new _m0.Reader(data)));
  }

  DeleteEnvironment(request: DeleteEnvironmentRequest): Promise<Empty> {
    const data = DeleteEnvironmentRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "DeleteEnvironment", data);
    return promise.then((data) => Empty.decode(new _m0.Reader(data)));
  }

  UndeleteEnvironment(request: UndeleteEnvironmentRequest): Promise<Environment> {
    const data = UndeleteEnvironmentRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "UndeleteEnvironment", data);
    return promise.then((data) => Environment.decode(new _m0.Reader(data)));
  }
}

interface Rpc {
  request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

type KeysOfUnion<T> = T extends T ? keyof T : never;
export type Exact<P, I extends P> = P extends Builtin ? P
  : P & { [K in keyof P]: Exact<P[K], I[K]> } & { [K in Exclude<keyof I, KeysOfUnion<P>>]: never };

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
