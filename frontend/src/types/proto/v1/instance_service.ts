/* eslint-disable */
import _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { State, stateFromJSON, stateToJSON } from "./common";

export const protobufPackage = "bytebase.v1";

export enum Engine {
  ENGINE_UNSPECIFIED = 0,
  CLICKHOUSE = 1,
  MYSQL = 2,
  POSTGRES = 3,
  SNOWFLAKE = 4,
  SQLITE = 5,
  TIDB = 6,
  MONGODB = 7,
  UNRECOGNIZED = -1,
}

export function engineFromJSON(object: any): Engine {
  switch (object) {
    case 0:
    case "ENGINE_UNSPECIFIED":
      return Engine.ENGINE_UNSPECIFIED;
    case 1:
    case "CLICKHOUSE":
      return Engine.CLICKHOUSE;
    case 2:
    case "MYSQL":
      return Engine.MYSQL;
    case 3:
    case "POSTGRES":
      return Engine.POSTGRES;
    case 4:
    case "SNOWFLAKE":
      return Engine.SNOWFLAKE;
    case 5:
    case "SQLITE":
      return Engine.SQLITE;
    case 6:
    case "TIDB":
      return Engine.TIDB;
    case 7:
    case "MONGODB":
      return Engine.MONGODB;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Engine.UNRECOGNIZED;
  }
}

export function engineToJSON(object: Engine): string {
  switch (object) {
    case Engine.ENGINE_UNSPECIFIED:
      return "ENGINE_UNSPECIFIED";
    case Engine.CLICKHOUSE:
      return "CLICKHOUSE";
    case Engine.MYSQL:
      return "MYSQL";
    case Engine.POSTGRES:
      return "POSTGRES";
    case Engine.SNOWFLAKE:
      return "SNOWFLAKE";
    case Engine.SQLITE:
      return "SQLITE";
    case Engine.TIDB:
      return "TIDB";
    case Engine.MONGODB:
      return "MONGODB";
    case Engine.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum DataSourceType {
  DATA_SOURCE_UNSPECIFIED = 0,
  ADMIN = 1,
  READ_ONLY = 2,
  UNRECOGNIZED = -1,
}

export function dataSourceTypeFromJSON(object: any): DataSourceType {
  switch (object) {
    case 0:
    case "DATA_SOURCE_UNSPECIFIED":
      return DataSourceType.DATA_SOURCE_UNSPECIFIED;
    case 1:
    case "ADMIN":
      return DataSourceType.ADMIN;
    case 2:
    case "READ_ONLY":
      return DataSourceType.READ_ONLY;
    case -1:
    case "UNRECOGNIZED":
    default:
      return DataSourceType.UNRECOGNIZED;
  }
}

export function dataSourceTypeToJSON(object: DataSourceType): string {
  switch (object) {
    case DataSourceType.DATA_SOURCE_UNSPECIFIED:
      return "DATA_SOURCE_UNSPECIFIED";
    case DataSourceType.ADMIN:
      return "ADMIN";
    case DataSourceType.READ_ONLY:
      return "READ_ONLY";
    case DataSourceType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface GetInstanceRequest {
  /**
   * The name of the instance to retrieve.
   * Format: environments/{environment}/instances/{instance}
   */
  name: string;
}

export interface ListInstancesRequest {
  /**
   * The parent, which owns this collection of instances.
   * Format: environments/{environment}
   * Use "environments/-" to list all instances from all environments.
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

export interface ListInstancesResponse {
  /** The instances from the specified request. */
  instances: Instance[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface CreateInstanceRequest {
  /**
   * The parent resource where this instance will be created.
   * Format: environments/{environment}
   */
  parent: string;
  /** The instance to create. */
  instance?: Instance;
  /**
   * The ID to use for the instance, which will become the final component of
   * the instance's resource name.
   *
   * This value should be 4-63 characters, and valid characters
   * are /[a-z][0-9]-/.
   */
  instanceId: string;
}

export interface UpdateInstanceRequest {
  /**
   * The instance to update.
   *
   * The instance's `name` field is used to identify the instance to update.
   * Format: environments/{environment}/instances/{instance}
   */
  instance?: Instance;
  /** The list of fields to update. */
  updateMask?: string[];
}

export interface DeleteInstanceRequest {
  /**
   * The name of the instance to delete.
   * Format: environments/{environment}/instances/{instance}
   */
  name: string;
}

export interface UndeleteInstanceRequest {
  /**
   * The name of the deleted instance.
   * Format: environments/{environment}/instances/{instance}
   */
  name: string;
}

export interface AddDataSourceRequest {
  /**
   * The name of the instance to add a data source to.
   * Format: environments/{environment}/instances/{instance}
   */
  instance: string;
  /**
   * Identified by type.
   * Only READ_ONLY data source can be added.
   */
  dataSources?: DataSource;
}

export interface RemoveDataSourceRequest {
  /**
   * The name of the instance to remove a data source from.
   * Format: environments/{environment}/instances/{instance}
   */
  instance: string;
  /**
   * Identified by type.
   * Only READ_ONLY data source can be removed.
   */
  dataSources?: DataSource;
}

export interface UpdateDataSourceRequest {
  /**
   * The name of the instance to update a data source.
   * Format: environments/{environment}/instances/{instance}
   */
  instance: string;
  /** Identified by type. */
  dataSources?: DataSource;
  /** The list of fields to update. */
  updateMask?: string[];
}

export interface Instance {
  /**
   * The name of the instance.
   * Format: environments/{environment}/instances/{instance}
   */
  name: string;
  /** The system-assigned, unique identifier for a resource. */
  uid: string;
  state: State;
  title: string;
  engine: Engine;
  externalLink: string;
  dataSources: DataSource[];
}

export interface DataSource {
  title: string;
  type: DataSourceType;
  username: string;
  password: string;
  sslCa: string;
  sslCert: string;
  sslKey: string;
  host: string;
  port: string;
  database: string;
}

function createBaseGetInstanceRequest(): GetInstanceRequest {
  return { name: "" };
}

export const GetInstanceRequest = {
  encode(message: GetInstanceRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetInstanceRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetInstanceRequest();
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

  fromJSON(object: any): GetInstanceRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetInstanceRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<GetInstanceRequest>, I>>(object: I): GetInstanceRequest {
    const message = createBaseGetInstanceRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListInstancesRequest(): ListInstancesRequest {
  return { parent: "", pageSize: 0, pageToken: "", showDeleted: false };
}

export const ListInstancesRequest = {
  encode(message: ListInstancesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): ListInstancesRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListInstancesRequest();
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

  fromJSON(object: any): ListInstancesRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
      showDeleted: isSet(object.showDeleted) ? Boolean(object.showDeleted) : false,
    };
  },

  toJSON(message: ListInstancesRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    message.showDeleted !== undefined && (obj.showDeleted = message.showDeleted);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ListInstancesRequest>, I>>(object: I): ListInstancesRequest {
    const message = createBaseListInstancesRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    message.showDeleted = object.showDeleted ?? false;
    return message;
  },
};

function createBaseListInstancesResponse(): ListInstancesResponse {
  return { instances: [], nextPageToken: "" };
}

export const ListInstancesResponse = {
  encode(message: ListInstancesResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.instances) {
      Instance.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListInstancesResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListInstancesResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.instances.push(Instance.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListInstancesResponse {
    return {
      instances: Array.isArray(object?.instances) ? object.instances.map((e: any) => Instance.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListInstancesResponse): unknown {
    const obj: any = {};
    if (message.instances) {
      obj.instances = message.instances.map((e) => e ? Instance.toJSON(e) : undefined);
    } else {
      obj.instances = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ListInstancesResponse>, I>>(object: I): ListInstancesResponse {
    const message = createBaseListInstancesResponse();
    message.instances = object.instances?.map((e) => Instance.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseCreateInstanceRequest(): CreateInstanceRequest {
  return { parent: "", instance: undefined, instanceId: "" };
}

export const CreateInstanceRequest = {
  encode(message: CreateInstanceRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.instance !== undefined) {
      Instance.encode(message.instance, writer.uint32(18).fork()).ldelim();
    }
    if (message.instanceId !== "") {
      writer.uint32(26).string(message.instanceId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateInstanceRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateInstanceRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.parent = reader.string();
          break;
        case 2:
          message.instance = Instance.decode(reader, reader.uint32());
          break;
        case 3:
          message.instanceId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): CreateInstanceRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      instance: isSet(object.instance) ? Instance.fromJSON(object.instance) : undefined,
      instanceId: isSet(object.instanceId) ? String(object.instanceId) : "",
    };
  },

  toJSON(message: CreateInstanceRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.instance !== undefined && (obj.instance = message.instance ? Instance.toJSON(message.instance) : undefined);
    message.instanceId !== undefined && (obj.instanceId = message.instanceId);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<CreateInstanceRequest>, I>>(object: I): CreateInstanceRequest {
    const message = createBaseCreateInstanceRequest();
    message.parent = object.parent ?? "";
    message.instance = (object.instance !== undefined && object.instance !== null)
      ? Instance.fromPartial(object.instance)
      : undefined;
    message.instanceId = object.instanceId ?? "";
    return message;
  },
};

function createBaseUpdateInstanceRequest(): UpdateInstanceRequest {
  return { instance: undefined, updateMask: undefined };
}

export const UpdateInstanceRequest = {
  encode(message: UpdateInstanceRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.instance !== undefined) {
      Instance.encode(message.instance, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateInstanceRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateInstanceRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.instance = Instance.decode(reader, reader.uint32());
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

  fromJSON(object: any): UpdateInstanceRequest {
    return {
      instance: isSet(object.instance) ? Instance.fromJSON(object.instance) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateInstanceRequest): unknown {
    const obj: any = {};
    message.instance !== undefined && (obj.instance = message.instance ? Instance.toJSON(message.instance) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<UpdateInstanceRequest>, I>>(object: I): UpdateInstanceRequest {
    const message = createBaseUpdateInstanceRequest();
    message.instance = (object.instance !== undefined && object.instance !== null)
      ? Instance.fromPartial(object.instance)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseDeleteInstanceRequest(): DeleteInstanceRequest {
  return { name: "" };
}

export const DeleteInstanceRequest = {
  encode(message: DeleteInstanceRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteInstanceRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteInstanceRequest();
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

  fromJSON(object: any): DeleteInstanceRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: DeleteInstanceRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DeleteInstanceRequest>, I>>(object: I): DeleteInstanceRequest {
    const message = createBaseDeleteInstanceRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseUndeleteInstanceRequest(): UndeleteInstanceRequest {
  return { name: "" };
}

export const UndeleteInstanceRequest = {
  encode(message: UndeleteInstanceRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UndeleteInstanceRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUndeleteInstanceRequest();
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

  fromJSON(object: any): UndeleteInstanceRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: UndeleteInstanceRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<UndeleteInstanceRequest>, I>>(object: I): UndeleteInstanceRequest {
    const message = createBaseUndeleteInstanceRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseAddDataSourceRequest(): AddDataSourceRequest {
  return { instance: "", dataSources: undefined };
}

export const AddDataSourceRequest = {
  encode(message: AddDataSourceRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.instance !== "") {
      writer.uint32(10).string(message.instance);
    }
    if (message.dataSources !== undefined) {
      DataSource.encode(message.dataSources, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): AddDataSourceRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAddDataSourceRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.instance = reader.string();
          break;
        case 2:
          message.dataSources = DataSource.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): AddDataSourceRequest {
    return {
      instance: isSet(object.instance) ? String(object.instance) : "",
      dataSources: isSet(object.dataSources) ? DataSource.fromJSON(object.dataSources) : undefined,
    };
  },

  toJSON(message: AddDataSourceRequest): unknown {
    const obj: any = {};
    message.instance !== undefined && (obj.instance = message.instance);
    message.dataSources !== undefined &&
      (obj.dataSources = message.dataSources ? DataSource.toJSON(message.dataSources) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<AddDataSourceRequest>, I>>(object: I): AddDataSourceRequest {
    const message = createBaseAddDataSourceRequest();
    message.instance = object.instance ?? "";
    message.dataSources = (object.dataSources !== undefined && object.dataSources !== null)
      ? DataSource.fromPartial(object.dataSources)
      : undefined;
    return message;
  },
};

function createBaseRemoveDataSourceRequest(): RemoveDataSourceRequest {
  return { instance: "", dataSources: undefined };
}

export const RemoveDataSourceRequest = {
  encode(message: RemoveDataSourceRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.instance !== "") {
      writer.uint32(10).string(message.instance);
    }
    if (message.dataSources !== undefined) {
      DataSource.encode(message.dataSources, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RemoveDataSourceRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRemoveDataSourceRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.instance = reader.string();
          break;
        case 2:
          message.dataSources = DataSource.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): RemoveDataSourceRequest {
    return {
      instance: isSet(object.instance) ? String(object.instance) : "",
      dataSources: isSet(object.dataSources) ? DataSource.fromJSON(object.dataSources) : undefined,
    };
  },

  toJSON(message: RemoveDataSourceRequest): unknown {
    const obj: any = {};
    message.instance !== undefined && (obj.instance = message.instance);
    message.dataSources !== undefined &&
      (obj.dataSources = message.dataSources ? DataSource.toJSON(message.dataSources) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<RemoveDataSourceRequest>, I>>(object: I): RemoveDataSourceRequest {
    const message = createBaseRemoveDataSourceRequest();
    message.instance = object.instance ?? "";
    message.dataSources = (object.dataSources !== undefined && object.dataSources !== null)
      ? DataSource.fromPartial(object.dataSources)
      : undefined;
    return message;
  },
};

function createBaseUpdateDataSourceRequest(): UpdateDataSourceRequest {
  return { instance: "", dataSources: undefined, updateMask: undefined };
}

export const UpdateDataSourceRequest = {
  encode(message: UpdateDataSourceRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.instance !== "") {
      writer.uint32(10).string(message.instance);
    }
    if (message.dataSources !== undefined) {
      DataSource.encode(message.dataSources, writer.uint32(18).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateDataSourceRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateDataSourceRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.instance = reader.string();
          break;
        case 2:
          message.dataSources = DataSource.decode(reader, reader.uint32());
          break;
        case 3:
          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): UpdateDataSourceRequest {
    return {
      instance: isSet(object.instance) ? String(object.instance) : "",
      dataSources: isSet(object.dataSources) ? DataSource.fromJSON(object.dataSources) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateDataSourceRequest): unknown {
    const obj: any = {};
    message.instance !== undefined && (obj.instance = message.instance);
    message.dataSources !== undefined &&
      (obj.dataSources = message.dataSources ? DataSource.toJSON(message.dataSources) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<UpdateDataSourceRequest>, I>>(object: I): UpdateDataSourceRequest {
    const message = createBaseUpdateDataSourceRequest();
    message.instance = object.instance ?? "";
    message.dataSources = (object.dataSources !== undefined && object.dataSources !== null)
      ? DataSource.fromPartial(object.dataSources)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseInstance(): Instance {
  return { name: "", uid: "", state: 0, title: "", engine: 0, externalLink: "", dataSources: [] };
}

export const Instance = {
  encode(message: Instance, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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
    if (message.engine !== 0) {
      writer.uint32(40).int32(message.engine);
    }
    if (message.externalLink !== "") {
      writer.uint32(50).string(message.externalLink);
    }
    for (const v of message.dataSources) {
      DataSource.encode(v!, writer.uint32(58).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Instance {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseInstance();
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
          message.engine = reader.int32() as any;
          break;
        case 6:
          message.externalLink = reader.string();
          break;
        case 7:
          message.dataSources.push(DataSource.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Instance {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      uid: isSet(object.uid) ? String(object.uid) : "",
      state: isSet(object.state) ? stateFromJSON(object.state) : 0,
      title: isSet(object.title) ? String(object.title) : "",
      engine: isSet(object.engine) ? engineFromJSON(object.engine) : 0,
      externalLink: isSet(object.externalLink) ? String(object.externalLink) : "",
      dataSources: Array.isArray(object?.dataSources) ? object.dataSources.map((e: any) => DataSource.fromJSON(e)) : [],
    };
  },

  toJSON(message: Instance): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.state !== undefined && (obj.state = stateToJSON(message.state));
    message.title !== undefined && (obj.title = message.title);
    message.engine !== undefined && (obj.engine = engineToJSON(message.engine));
    message.externalLink !== undefined && (obj.externalLink = message.externalLink);
    if (message.dataSources) {
      obj.dataSources = message.dataSources.map((e) => e ? DataSource.toJSON(e) : undefined);
    } else {
      obj.dataSources = [];
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<Instance>, I>>(object: I): Instance {
    const message = createBaseInstance();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.state = object.state ?? 0;
    message.title = object.title ?? "";
    message.engine = object.engine ?? 0;
    message.externalLink = object.externalLink ?? "";
    message.dataSources = object.dataSources?.map((e) => DataSource.fromPartial(e)) || [];
    return message;
  },
};

function createBaseDataSource(): DataSource {
  return {
    title: "",
    type: 0,
    username: "",
    password: "",
    sslCa: "",
    sslCert: "",
    sslKey: "",
    host: "",
    port: "",
    database: "",
  };
}

export const DataSource = {
  encode(message: DataSource, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.title !== "") {
      writer.uint32(10).string(message.title);
    }
    if (message.type !== 0) {
      writer.uint32(16).int32(message.type);
    }
    if (message.username !== "") {
      writer.uint32(26).string(message.username);
    }
    if (message.password !== "") {
      writer.uint32(34).string(message.password);
    }
    if (message.sslCa !== "") {
      writer.uint32(42).string(message.sslCa);
    }
    if (message.sslCert !== "") {
      writer.uint32(50).string(message.sslCert);
    }
    if (message.sslKey !== "") {
      writer.uint32(58).string(message.sslKey);
    }
    if (message.host !== "") {
      writer.uint32(66).string(message.host);
    }
    if (message.port !== "") {
      writer.uint32(74).string(message.port);
    }
    if (message.database !== "") {
      writer.uint32(82).string(message.database);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DataSource {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDataSource();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.title = reader.string();
          break;
        case 2:
          message.type = reader.int32() as any;
          break;
        case 3:
          message.username = reader.string();
          break;
        case 4:
          message.password = reader.string();
          break;
        case 5:
          message.sslCa = reader.string();
          break;
        case 6:
          message.sslCert = reader.string();
          break;
        case 7:
          message.sslKey = reader.string();
          break;
        case 8:
          message.host = reader.string();
          break;
        case 9:
          message.port = reader.string();
          break;
        case 10:
          message.database = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DataSource {
    return {
      title: isSet(object.title) ? String(object.title) : "",
      type: isSet(object.type) ? dataSourceTypeFromJSON(object.type) : 0,
      username: isSet(object.username) ? String(object.username) : "",
      password: isSet(object.password) ? String(object.password) : "",
      sslCa: isSet(object.sslCa) ? String(object.sslCa) : "",
      sslCert: isSet(object.sslCert) ? String(object.sslCert) : "",
      sslKey: isSet(object.sslKey) ? String(object.sslKey) : "",
      host: isSet(object.host) ? String(object.host) : "",
      port: isSet(object.port) ? String(object.port) : "",
      database: isSet(object.database) ? String(object.database) : "",
    };
  },

  toJSON(message: DataSource): unknown {
    const obj: any = {};
    message.title !== undefined && (obj.title = message.title);
    message.type !== undefined && (obj.type = dataSourceTypeToJSON(message.type));
    message.username !== undefined && (obj.username = message.username);
    message.password !== undefined && (obj.password = message.password);
    message.sslCa !== undefined && (obj.sslCa = message.sslCa);
    message.sslCert !== undefined && (obj.sslCert = message.sslCert);
    message.sslKey !== undefined && (obj.sslKey = message.sslKey);
    message.host !== undefined && (obj.host = message.host);
    message.port !== undefined && (obj.port = message.port);
    message.database !== undefined && (obj.database = message.database);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DataSource>, I>>(object: I): DataSource {
    const message = createBaseDataSource();
    message.title = object.title ?? "";
    message.type = object.type ?? 0;
    message.username = object.username ?? "";
    message.password = object.password ?? "";
    message.sslCa = object.sslCa ?? "";
    message.sslCert = object.sslCert ?? "";
    message.sslKey = object.sslKey ?? "";
    message.host = object.host ?? "";
    message.port = object.port ?? "";
    message.database = object.database ?? "";
    return message;
  },
};

export interface InstanceService {
  GetInstance(request: GetInstanceRequest): Promise<Instance>;
  ListInstances(request: ListInstancesRequest): Promise<ListInstancesResponse>;
  CreateInstance(request: CreateInstanceRequest): Promise<Instance>;
  UpdateInstance(request: UpdateInstanceRequest): Promise<Instance>;
  DeleteInstance(request: DeleteInstanceRequest): Promise<Empty>;
  UndeleteInstance(request: UndeleteInstanceRequest): Promise<Instance>;
  AddDataSource(request: AddDataSourceRequest): Promise<Instance>;
  RemoveDataSource(request: RemoveDataSourceRequest): Promise<Instance>;
  UpdateDataSource(request: UpdateDataSourceRequest): Promise<Instance>;
}

export class InstanceServiceClientImpl implements InstanceService {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "bytebase.v1.InstanceService";
    this.rpc = rpc;
    this.GetInstance = this.GetInstance.bind(this);
    this.ListInstances = this.ListInstances.bind(this);
    this.CreateInstance = this.CreateInstance.bind(this);
    this.UpdateInstance = this.UpdateInstance.bind(this);
    this.DeleteInstance = this.DeleteInstance.bind(this);
    this.UndeleteInstance = this.UndeleteInstance.bind(this);
    this.AddDataSource = this.AddDataSource.bind(this);
    this.RemoveDataSource = this.RemoveDataSource.bind(this);
    this.UpdateDataSource = this.UpdateDataSource.bind(this);
  }
  GetInstance(request: GetInstanceRequest): Promise<Instance> {
    const data = GetInstanceRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "GetInstance", data);
    return promise.then((data) => Instance.decode(new _m0.Reader(data)));
  }

  ListInstances(request: ListInstancesRequest): Promise<ListInstancesResponse> {
    const data = ListInstancesRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "ListInstances", data);
    return promise.then((data) => ListInstancesResponse.decode(new _m0.Reader(data)));
  }

  CreateInstance(request: CreateInstanceRequest): Promise<Instance> {
    const data = CreateInstanceRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "CreateInstance", data);
    return promise.then((data) => Instance.decode(new _m0.Reader(data)));
  }

  UpdateInstance(request: UpdateInstanceRequest): Promise<Instance> {
    const data = UpdateInstanceRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "UpdateInstance", data);
    return promise.then((data) => Instance.decode(new _m0.Reader(data)));
  }

  DeleteInstance(request: DeleteInstanceRequest): Promise<Empty> {
    const data = DeleteInstanceRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "DeleteInstance", data);
    return promise.then((data) => Empty.decode(new _m0.Reader(data)));
  }

  UndeleteInstance(request: UndeleteInstanceRequest): Promise<Instance> {
    const data = UndeleteInstanceRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "UndeleteInstance", data);
    return promise.then((data) => Instance.decode(new _m0.Reader(data)));
  }

  AddDataSource(request: AddDataSourceRequest): Promise<Instance> {
    const data = AddDataSourceRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "AddDataSource", data);
    return promise.then((data) => Instance.decode(new _m0.Reader(data)));
  }

  RemoveDataSource(request: RemoveDataSourceRequest): Promise<Instance> {
    const data = RemoveDataSourceRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "RemoveDataSource", data);
    return promise.then((data) => Instance.decode(new _m0.Reader(data)));
  }

  UpdateDataSource(request: UpdateDataSourceRequest): Promise<Instance> {
    const data = UpdateDataSourceRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "UpdateDataSource", data);
    return promise.then((data) => Instance.decode(new _m0.Reader(data)));
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
