/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { Engine, engineFromJSON, engineToJSON, State, stateFromJSON, stateToJSON } from "./common";

export const protobufPackage = "bytebase.v1";

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

export interface SyncSlowQueriesRequest {
  /**
   * The name of the instance to sync slow queries.
   * Format: environments/{environment}/instances/{instance}
   */
  instance: string;
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
  /** srv and authentication_database are used for MongoDB. */
  srv: boolean;
  authenticationDatabase: string;
  /** sid and service_name are used for Oracle. */
  sid: string;
  serviceName: string;
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetInstanceRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<GetInstanceRequest>): GetInstanceRequest {
    return GetInstanceRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<GetInstanceRequest>): GetInstanceRequest {
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListInstancesRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.parent = reader.string();
          continue;
        case 2:
          if (tag != 16) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.pageToken = reader.string();
          continue;
        case 4:
          if (tag != 32) {
            break;
          }

          message.showDeleted = reader.bool();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<ListInstancesRequest>): ListInstancesRequest {
    return ListInstancesRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListInstancesRequest>): ListInstancesRequest {
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListInstancesResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.instances.push(Instance.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.nextPageToken = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<ListInstancesResponse>): ListInstancesResponse {
    return ListInstancesResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListInstancesResponse>): ListInstancesResponse {
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateInstanceRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.parent = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.instance = Instance.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.instanceId = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<CreateInstanceRequest>): CreateInstanceRequest {
    return CreateInstanceRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<CreateInstanceRequest>): CreateInstanceRequest {
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateInstanceRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.instance = Instance.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<UpdateInstanceRequest>): UpdateInstanceRequest {
    return UpdateInstanceRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<UpdateInstanceRequest>): UpdateInstanceRequest {
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteInstanceRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<DeleteInstanceRequest>): DeleteInstanceRequest {
    return DeleteInstanceRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<DeleteInstanceRequest>): DeleteInstanceRequest {
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUndeleteInstanceRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<UndeleteInstanceRequest>): UndeleteInstanceRequest {
    return UndeleteInstanceRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<UndeleteInstanceRequest>): UndeleteInstanceRequest {
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAddDataSourceRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.instance = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.dataSources = DataSource.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<AddDataSourceRequest>): AddDataSourceRequest {
    return AddDataSourceRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<AddDataSourceRequest>): AddDataSourceRequest {
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRemoveDataSourceRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.instance = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.dataSources = DataSource.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<RemoveDataSourceRequest>): RemoveDataSourceRequest {
    return RemoveDataSourceRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<RemoveDataSourceRequest>): RemoveDataSourceRequest {
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateDataSourceRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.instance = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.dataSources = DataSource.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<UpdateDataSourceRequest>): UpdateDataSourceRequest {
    return UpdateDataSourceRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<UpdateDataSourceRequest>): UpdateDataSourceRequest {
    const message = createBaseUpdateDataSourceRequest();
    message.instance = object.instance ?? "";
    message.dataSources = (object.dataSources !== undefined && object.dataSources !== null)
      ? DataSource.fromPartial(object.dataSources)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseSyncSlowQueriesRequest(): SyncSlowQueriesRequest {
  return { instance: "" };
}

export const SyncSlowQueriesRequest = {
  encode(message: SyncSlowQueriesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.instance !== "") {
      writer.uint32(10).string(message.instance);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SyncSlowQueriesRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSyncSlowQueriesRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.instance = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SyncSlowQueriesRequest {
    return { instance: isSet(object.instance) ? String(object.instance) : "" };
  },

  toJSON(message: SyncSlowQueriesRequest): unknown {
    const obj: any = {};
    message.instance !== undefined && (obj.instance = message.instance);
    return obj;
  },

  create(base?: DeepPartial<SyncSlowQueriesRequest>): SyncSlowQueriesRequest {
    return SyncSlowQueriesRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SyncSlowQueriesRequest>): SyncSlowQueriesRequest {
    const message = createBaseSyncSlowQueriesRequest();
    message.instance = object.instance ?? "";
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseInstance();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.uid = reader.string();
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.state = reader.int32() as any;
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.title = reader.string();
          continue;
        case 5:
          if (tag != 40) {
            break;
          }

          message.engine = reader.int32() as any;
          continue;
        case 6:
          if (tag != 50) {
            break;
          }

          message.externalLink = reader.string();
          continue;
        case 7:
          if (tag != 58) {
            break;
          }

          message.dataSources.push(DataSource.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<Instance>): Instance {
    return Instance.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Instance>): Instance {
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
    srv: false,
    authenticationDatabase: "",
    sid: "",
    serviceName: "",
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
    if (message.srv === true) {
      writer.uint32(88).bool(message.srv);
    }
    if (message.authenticationDatabase !== "") {
      writer.uint32(98).string(message.authenticationDatabase);
    }
    if (message.sid !== "") {
      writer.uint32(106).string(message.sid);
    }
    if (message.serviceName !== "") {
      writer.uint32(114).string(message.serviceName);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DataSource {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDataSource();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.title = reader.string();
          continue;
        case 2:
          if (tag != 16) {
            break;
          }

          message.type = reader.int32() as any;
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.username = reader.string();
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.password = reader.string();
          continue;
        case 5:
          if (tag != 42) {
            break;
          }

          message.sslCa = reader.string();
          continue;
        case 6:
          if (tag != 50) {
            break;
          }

          message.sslCert = reader.string();
          continue;
        case 7:
          if (tag != 58) {
            break;
          }

          message.sslKey = reader.string();
          continue;
        case 8:
          if (tag != 66) {
            break;
          }

          message.host = reader.string();
          continue;
        case 9:
          if (tag != 74) {
            break;
          }

          message.port = reader.string();
          continue;
        case 10:
          if (tag != 82) {
            break;
          }

          message.database = reader.string();
          continue;
        case 11:
          if (tag != 88) {
            break;
          }

          message.srv = reader.bool();
          continue;
        case 12:
          if (tag != 98) {
            break;
          }

          message.authenticationDatabase = reader.string();
          continue;
        case 13:
          if (tag != 106) {
            break;
          }

          message.sid = reader.string();
          continue;
        case 14:
          if (tag != 114) {
            break;
          }

          message.serviceName = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
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
      srv: isSet(object.srv) ? Boolean(object.srv) : false,
      authenticationDatabase: isSet(object.authenticationDatabase) ? String(object.authenticationDatabase) : "",
      sid: isSet(object.sid) ? String(object.sid) : "",
      serviceName: isSet(object.serviceName) ? String(object.serviceName) : "",
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
    message.srv !== undefined && (obj.srv = message.srv);
    message.authenticationDatabase !== undefined && (obj.authenticationDatabase = message.authenticationDatabase);
    message.sid !== undefined && (obj.sid = message.sid);
    message.serviceName !== undefined && (obj.serviceName = message.serviceName);
    return obj;
  },

  create(base?: DeepPartial<DataSource>): DataSource {
    return DataSource.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<DataSource>): DataSource {
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
    message.srv = object.srv ?? false;
    message.authenticationDatabase = object.authenticationDatabase ?? "";
    message.sid = object.sid ?? "";
    message.serviceName = object.serviceName ?? "";
    return message;
  },
};

export type InstanceServiceDefinition = typeof InstanceServiceDefinition;
export const InstanceServiceDefinition = {
  name: "InstanceService",
  fullName: "bytebase.v1.InstanceService",
  methods: {
    getInstance: {
      name: "GetInstance",
      requestType: GetInstanceRequest,
      requestStream: false,
      responseType: Instance,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              39,
              18,
              37,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              101,
              110,
              118,
              105,
              114,
              111,
              110,
              109,
              101,
              110,
              116,
              115,
              47,
              42,
              47,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    listInstances: {
      name: "ListInstances",
      requestType: ListInstancesRequest,
      requestStream: false,
      responseType: ListInstancesResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([6, 112, 97, 114, 101, 110, 116])],
          578365826: [
            new Uint8Array([
              39,
              18,
              37,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              101,
              110,
              118,
              105,
              114,
              111,
              110,
              109,
              101,
              110,
              116,
              115,
              47,
              42,
              125,
              47,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
            ]),
          ],
        },
      },
    },
    createInstance: {
      name: "CreateInstance",
      requestType: CreateInstanceRequest,
      requestStream: false,
      responseType: Instance,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([15, 112, 97, 114, 101, 110, 116, 44, 105, 110, 115, 116, 97, 110, 99, 101])],
          578365826: [
            new Uint8Array([
              49,
              58,
              8,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              34,
              37,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              101,
              110,
              118,
              105,
              114,
              111,
              110,
              109,
              101,
              110,
              116,
              115,
              47,
              42,
              125,
              47,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
            ]),
          ],
        },
      },
    },
    updateInstance: {
      name: "UpdateInstance",
      requestType: UpdateInstanceRequest,
      requestStream: false,
      responseType: Instance,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([
              20,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              44,
              117,
              112,
              100,
              97,
              116,
              101,
              95,
              109,
              97,
              115,
              107,
            ]),
          ],
          578365826: [
            new Uint8Array([
              58,
              58,
              8,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              50,
              46,
              47,
              118,
              49,
              47,
              123,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              46,
              110,
              97,
              109,
              101,
              61,
              101,
              110,
              118,
              105,
              114,
              111,
              110,
              109,
              101,
              110,
              116,
              115,
              47,
              42,
              47,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    deleteInstance: {
      name: "DeleteInstance",
      requestType: DeleteInstanceRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              39,
              42,
              37,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              101,
              110,
              118,
              105,
              114,
              111,
              110,
              109,
              101,
              110,
              116,
              115,
              47,
              42,
              47,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    undeleteInstance: {
      name: "UndeleteInstance",
      requestType: UndeleteInstanceRequest,
      requestStream: false,
      responseType: Instance,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              51,
              58,
              1,
              42,
              34,
              46,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              101,
              110,
              118,
              105,
              114,
              111,
              110,
              109,
              101,
              110,
              116,
              115,
              47,
              42,
              47,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
              47,
              42,
              125,
              58,
              117,
              110,
              100,
              101,
              108,
              101,
              116,
              101,
            ]),
          ],
        },
      },
    },
    addDataSource: {
      name: "AddDataSource",
      requestType: AddDataSourceRequest,
      requestStream: false,
      responseType: Instance,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              60,
              58,
              1,
              42,
              34,
              55,
              47,
              118,
              49,
              47,
              123,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              61,
              101,
              110,
              118,
              105,
              114,
              111,
              110,
              109,
              101,
              110,
              116,
              115,
              47,
              42,
              47,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
              47,
              42,
              125,
              58,
              97,
              100,
              100,
              68,
              97,
              116,
              97,
              83,
              111,
              117,
              114,
              99,
              101,
            ]),
          ],
        },
      },
    },
    removeDataSource: {
      name: "RemoveDataSource",
      requestType: RemoveDataSourceRequest,
      requestStream: false,
      responseType: Instance,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              63,
              58,
              1,
              42,
              34,
              58,
              47,
              118,
              49,
              47,
              123,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              61,
              101,
              110,
              118,
              105,
              114,
              111,
              110,
              109,
              101,
              110,
              116,
              115,
              47,
              42,
              47,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
              47,
              42,
              125,
              58,
              114,
              101,
              109,
              111,
              118,
              101,
              68,
              97,
              116,
              97,
              83,
              111,
              117,
              114,
              99,
              101,
            ]),
          ],
        },
      },
    },
    updateDataSource: {
      name: "UpdateDataSource",
      requestType: UpdateDataSourceRequest,
      requestStream: false,
      responseType: Instance,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              63,
              58,
              1,
              42,
              50,
              58,
              47,
              118,
              49,
              47,
              123,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              61,
              101,
              110,
              118,
              105,
              114,
              111,
              110,
              109,
              101,
              110,
              116,
              115,
              47,
              42,
              47,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
              47,
              42,
              125,
              58,
              117,
              112,
              100,
              97,
              116,
              101,
              68,
              97,
              116,
              97,
              83,
              111,
              117,
              114,
              99,
              101,
            ]),
          ],
        },
      },
    },
    syncSlowQueries: {
      name: "SyncSlowQueries",
      requestType: SyncSlowQueriesRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              62,
              58,
              1,
              42,
              34,
              57,
              47,
              118,
              49,
              47,
              123,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              61,
              101,
              110,
              118,
              105,
              114,
              111,
              110,
              109,
              101,
              110,
              116,
              115,
              47,
              42,
              47,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
              47,
              42,
              125,
              58,
              115,
              121,
              110,
              99,
              83,
              108,
              111,
              119,
              81,
              117,
              101,
              114,
              105,
              101,
              115,
            ]),
          ],
        },
      },
    },
  },
} as const;

export interface InstanceServiceImplementation<CallContextExt = {}> {
  getInstance(request: GetInstanceRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Instance>>;
  listInstances(
    request: ListInstancesRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ListInstancesResponse>>;
  createInstance(request: CreateInstanceRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Instance>>;
  updateInstance(request: UpdateInstanceRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Instance>>;
  deleteInstance(request: DeleteInstanceRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Empty>>;
  undeleteInstance(
    request: UndeleteInstanceRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Instance>>;
  addDataSource(request: AddDataSourceRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Instance>>;
  removeDataSource(
    request: RemoveDataSourceRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Instance>>;
  updateDataSource(
    request: UpdateDataSourceRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Instance>>;
  syncSlowQueries(request: SyncSlowQueriesRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Empty>>;
}

export interface InstanceServiceClient<CallOptionsExt = {}> {
  getInstance(request: DeepPartial<GetInstanceRequest>, options?: CallOptions & CallOptionsExt): Promise<Instance>;
  listInstances(
    request: DeepPartial<ListInstancesRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ListInstancesResponse>;
  createInstance(
    request: DeepPartial<CreateInstanceRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Instance>;
  updateInstance(
    request: DeepPartial<UpdateInstanceRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Instance>;
  deleteInstance(request: DeepPartial<DeleteInstanceRequest>, options?: CallOptions & CallOptionsExt): Promise<Empty>;
  undeleteInstance(
    request: DeepPartial<UndeleteInstanceRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Instance>;
  addDataSource(request: DeepPartial<AddDataSourceRequest>, options?: CallOptions & CallOptionsExt): Promise<Instance>;
  removeDataSource(
    request: DeepPartial<RemoveDataSourceRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Instance>;
  updateDataSource(
    request: DeepPartial<UpdateDataSourceRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Instance>;
  syncSlowQueries(request: DeepPartial<SyncSlowQueriesRequest>, options?: CallOptions & CallOptionsExt): Promise<Empty>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
