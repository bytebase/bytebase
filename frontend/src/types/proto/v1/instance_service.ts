/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Duration } from "../google/protobuf/duration";
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
   * Format: instances/{instance}
   */
  name: string;
}

export interface ListInstancesRequest {
  /**
   * The parent parameter's value depends on the target resource for the request.
   * - instances.list(): An empty string. This method doesn't require a resource; it simply returns all instances the user has access to.
   * - projects.instances.list(): projects/{PROJECT_ID}. This method lists all instances that have databases in the project.
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
  /** The instance to create. */
  instance:
    | Instance
    | undefined;
  /**
   * The ID to use for the instance, which will become the final component of
   * the instance's resource name.
   *
   * This value should be 4-63 characters, and valid characters
   * are /[a-z][0-9]-/.
   */
  instanceId: string;
  /** Validate only also tests the data source connection. */
  validateOnly: boolean;
}

export interface UpdateInstanceRequest {
  /**
   * The instance to update.
   *
   * The instance's `name` field is used to identify the instance to update.
   * Format: instances/{instance}
   */
  instance:
    | Instance
    | undefined;
  /** The list of fields to update. */
  updateMask: string[] | undefined;
}

export interface DeleteInstanceRequest {
  /**
   * The name of the instance to delete.
   * Format: instances/{instance}
   */
  name: string;
  /** If set to true, any databases and sheets from this project will also be moved to default project, and all open issues will be closed. */
  force: boolean;
}

export interface UndeleteInstanceRequest {
  /**
   * The name of the deleted instance.
   * Format: instances/{instance}
   */
  name: string;
}

export interface SyncInstanceRequest {
  /**
   * The name of instance.
   * Format: instances/{instance}
   */
  name: string;
}

export interface SyncInstanceResponse {
}

export interface BatchSyncInstanceRequest {
  /**
   * The request message specifying the instances to sync.
   * A maximum of 1000 instances can be synced in a batch.
   */
  requests: SyncInstanceRequest[];
}

export interface BatchSyncInstanceResponse {
}

export interface AddDataSourceRequest {
  /**
   * The name of the instance to add a data source to.
   * Format: instances/{instance}
   */
  instance: string;
  /**
   * Identified by type.
   * Only READ_ONLY data source can be added.
   */
  dataSource:
    | DataSource
    | undefined;
  /** Validate only also tests the data source connection. */
  validateOnly: boolean;
}

export interface RemoveDataSourceRequest {
  /**
   * The name of the instance to remove a data source from.
   * Format: instances/{instance}
   */
  instance: string;
  /**
   * Identified by type.
   * Only READ_ONLY data source can be removed.
   */
  dataSource: DataSource | undefined;
}

export interface UpdateDataSourceRequest {
  /**
   * The name of the instance to update a data source.
   * Format: instances/{instance}
   */
  instance: string;
  /** Identified by type. */
  dataSource:
    | DataSource
    | undefined;
  /** The list of fields to update. */
  updateMask:
    | string[]
    | undefined;
  /** Validate only also tests the data source connection. */
  validateOnly: boolean;
}

export interface SyncSlowQueriesRequest {
  /**
   * The name of the instance to sync slow queries.
   * Format: instances/{instance} for one instance
   *      or projects/{project} for one project.
   */
  parent: string;
}

/** InstanceOptions is the option for instances. */
export interface InstanceOptions {
  /**
   * The schema tenant mode is used to determine whether the instance is in schema tenant mode.
   * For Oracle schema tenant mode, the instance a Oracle database and the database is the Oracle schema.
   */
  schemaTenantMode: boolean;
  /** How often the instance is synced. */
  syncInterval: Duration | undefined;
}

export interface Instance {
  /**
   * The name of the instance.
   * Format: instances/{instance}
   */
  name: string;
  /** The system-assigned, unique identifier for a resource. */
  uid: string;
  state: State;
  title: string;
  engine: Engine;
  engineVersion: string;
  externalLink: string;
  dataSources: DataSource[];
  /**
   * The environment resource.
   * Format: environments/prod where prod is the environment resource ID.
   */
  environment: string;
  activation: boolean;
  options: InstanceOptions | undefined;
}

export interface DataSource {
  id: string;
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
  /**
   * Connection over SSH.
   * The hostname of the SSH server agent.
   * Required.
   */
  sshHost: string;
  /**
   * The port of the SSH server agent. It's 22 typically.
   * Required.
   */
  sshPort: string;
  /**
   * The user to login the server.
   * Required.
   */
  sshUser: string;
  /** The password to login the server. If it's empty string, no password is required. */
  sshPassword: string;
  /** The private key to login the server. If it's empty string, we will use the system default private key from os.Getenv("SSH_AUTH_SOCK"). */
  sshPrivateKey: string;
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
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GetInstanceRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: GetInstanceRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
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
      writer.uint32(34).string(message.parent);
    }
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

  decode(input: _m0.Reader | Uint8Array, length?: number): ListInstancesRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListInstancesRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 4:
          if (tag !== 34) {
            break;
          }

          message.parent = reader.string();
          continue;
        case 1:
          if (tag !== 8) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.pageToken = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.showDeleted = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListInstancesRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
      showDeleted: isSet(object.showDeleted) ? globalThis.Boolean(object.showDeleted) : false,
    };
  },

  toJSON(message: ListInstancesRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    if (message.showDeleted === true) {
      obj.showDeleted = message.showDeleted;
    }
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
          if (tag !== 10) {
            break;
          }

          message.instances.push(Instance.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.nextPageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListInstancesResponse {
    return {
      instances: globalThis.Array.isArray(object?.instances)
        ? object.instances.map((e: any) => Instance.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListInstancesResponse): unknown {
    const obj: any = {};
    if (message.instances?.length) {
      obj.instances = message.instances.map((e) => Instance.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
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
  return { instance: undefined, instanceId: "", validateOnly: false };
}

export const CreateInstanceRequest = {
  encode(message: CreateInstanceRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.instance !== undefined) {
      Instance.encode(message.instance, writer.uint32(10).fork()).ldelim();
    }
    if (message.instanceId !== "") {
      writer.uint32(18).string(message.instanceId);
    }
    if (message.validateOnly === true) {
      writer.uint32(24).bool(message.validateOnly);
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
          if (tag !== 10) {
            break;
          }

          message.instance = Instance.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.instanceId = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.validateOnly = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateInstanceRequest {
    return {
      instance: isSet(object.instance) ? Instance.fromJSON(object.instance) : undefined,
      instanceId: isSet(object.instanceId) ? globalThis.String(object.instanceId) : "",
      validateOnly: isSet(object.validateOnly) ? globalThis.Boolean(object.validateOnly) : false,
    };
  },

  toJSON(message: CreateInstanceRequest): unknown {
    const obj: any = {};
    if (message.instance !== undefined) {
      obj.instance = Instance.toJSON(message.instance);
    }
    if (message.instanceId !== "") {
      obj.instanceId = message.instanceId;
    }
    if (message.validateOnly === true) {
      obj.validateOnly = message.validateOnly;
    }
    return obj;
  },

  create(base?: DeepPartial<CreateInstanceRequest>): CreateInstanceRequest {
    return CreateInstanceRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<CreateInstanceRequest>): CreateInstanceRequest {
    const message = createBaseCreateInstanceRequest();
    message.instance = (object.instance !== undefined && object.instance !== null)
      ? Instance.fromPartial(object.instance)
      : undefined;
    message.instanceId = object.instanceId ?? "";
    message.validateOnly = object.validateOnly ?? false;
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
          if (tag !== 10) {
            break;
          }

          message.instance = Instance.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
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
    if (message.instance !== undefined) {
      obj.instance = Instance.toJSON(message.instance);
    }
    if (message.updateMask !== undefined) {
      obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask));
    }
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
  return { name: "", force: false };
}

export const DeleteInstanceRequest = {
  encode(message: DeleteInstanceRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.force === true) {
      writer.uint32(16).bool(message.force);
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
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.force = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DeleteInstanceRequest {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      force: isSet(object.force) ? globalThis.Boolean(object.force) : false,
    };
  },

  toJSON(message: DeleteInstanceRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.force === true) {
      obj.force = message.force;
    }
    return obj;
  },

  create(base?: DeepPartial<DeleteInstanceRequest>): DeleteInstanceRequest {
    return DeleteInstanceRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DeleteInstanceRequest>): DeleteInstanceRequest {
    const message = createBaseDeleteInstanceRequest();
    message.name = object.name ?? "";
    message.force = object.force ?? false;
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
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UndeleteInstanceRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: UndeleteInstanceRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
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

function createBaseSyncInstanceRequest(): SyncInstanceRequest {
  return { name: "" };
}

export const SyncInstanceRequest = {
  encode(message: SyncInstanceRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SyncInstanceRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSyncInstanceRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SyncInstanceRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: SyncInstanceRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<SyncInstanceRequest>): SyncInstanceRequest {
    return SyncInstanceRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SyncInstanceRequest>): SyncInstanceRequest {
    const message = createBaseSyncInstanceRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseSyncInstanceResponse(): SyncInstanceResponse {
  return {};
}

export const SyncInstanceResponse = {
  encode(_: SyncInstanceResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SyncInstanceResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSyncInstanceResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): SyncInstanceResponse {
    return {};
  },

  toJSON(_: SyncInstanceResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<SyncInstanceResponse>): SyncInstanceResponse {
    return SyncInstanceResponse.fromPartial(base ?? {});
  },
  fromPartial(_: DeepPartial<SyncInstanceResponse>): SyncInstanceResponse {
    const message = createBaseSyncInstanceResponse();
    return message;
  },
};

function createBaseBatchSyncInstanceRequest(): BatchSyncInstanceRequest {
  return { requests: [] };
}

export const BatchSyncInstanceRequest = {
  encode(message: BatchSyncInstanceRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.requests) {
      SyncInstanceRequest.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchSyncInstanceRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchSyncInstanceRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.requests.push(SyncInstanceRequest.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BatchSyncInstanceRequest {
    return {
      requests: globalThis.Array.isArray(object?.requests)
        ? object.requests.map((e: any) => SyncInstanceRequest.fromJSON(e))
        : [],
    };
  },

  toJSON(message: BatchSyncInstanceRequest): unknown {
    const obj: any = {};
    if (message.requests?.length) {
      obj.requests = message.requests.map((e) => SyncInstanceRequest.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<BatchSyncInstanceRequest>): BatchSyncInstanceRequest {
    return BatchSyncInstanceRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BatchSyncInstanceRequest>): BatchSyncInstanceRequest {
    const message = createBaseBatchSyncInstanceRequest();
    message.requests = object.requests?.map((e) => SyncInstanceRequest.fromPartial(e)) || [];
    return message;
  },
};

function createBaseBatchSyncInstanceResponse(): BatchSyncInstanceResponse {
  return {};
}

export const BatchSyncInstanceResponse = {
  encode(_: BatchSyncInstanceResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchSyncInstanceResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchSyncInstanceResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): BatchSyncInstanceResponse {
    return {};
  },

  toJSON(_: BatchSyncInstanceResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<BatchSyncInstanceResponse>): BatchSyncInstanceResponse {
    return BatchSyncInstanceResponse.fromPartial(base ?? {});
  },
  fromPartial(_: DeepPartial<BatchSyncInstanceResponse>): BatchSyncInstanceResponse {
    const message = createBaseBatchSyncInstanceResponse();
    return message;
  },
};

function createBaseAddDataSourceRequest(): AddDataSourceRequest {
  return { instance: "", dataSource: undefined, validateOnly: false };
}

export const AddDataSourceRequest = {
  encode(message: AddDataSourceRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.instance !== "") {
      writer.uint32(10).string(message.instance);
    }
    if (message.dataSource !== undefined) {
      DataSource.encode(message.dataSource, writer.uint32(18).fork()).ldelim();
    }
    if (message.validateOnly === true) {
      writer.uint32(24).bool(message.validateOnly);
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
          if (tag !== 10) {
            break;
          }

          message.instance = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.dataSource = DataSource.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.validateOnly = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): AddDataSourceRequest {
    return {
      instance: isSet(object.instance) ? globalThis.String(object.instance) : "",
      dataSource: isSet(object.dataSource) ? DataSource.fromJSON(object.dataSource) : undefined,
      validateOnly: isSet(object.validateOnly) ? globalThis.Boolean(object.validateOnly) : false,
    };
  },

  toJSON(message: AddDataSourceRequest): unknown {
    const obj: any = {};
    if (message.instance !== "") {
      obj.instance = message.instance;
    }
    if (message.dataSource !== undefined) {
      obj.dataSource = DataSource.toJSON(message.dataSource);
    }
    if (message.validateOnly === true) {
      obj.validateOnly = message.validateOnly;
    }
    return obj;
  },

  create(base?: DeepPartial<AddDataSourceRequest>): AddDataSourceRequest {
    return AddDataSourceRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<AddDataSourceRequest>): AddDataSourceRequest {
    const message = createBaseAddDataSourceRequest();
    message.instance = object.instance ?? "";
    message.dataSource = (object.dataSource !== undefined && object.dataSource !== null)
      ? DataSource.fromPartial(object.dataSource)
      : undefined;
    message.validateOnly = object.validateOnly ?? false;
    return message;
  },
};

function createBaseRemoveDataSourceRequest(): RemoveDataSourceRequest {
  return { instance: "", dataSource: undefined };
}

export const RemoveDataSourceRequest = {
  encode(message: RemoveDataSourceRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.instance !== "") {
      writer.uint32(10).string(message.instance);
    }
    if (message.dataSource !== undefined) {
      DataSource.encode(message.dataSource, writer.uint32(18).fork()).ldelim();
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
          if (tag !== 10) {
            break;
          }

          message.instance = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.dataSource = DataSource.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): RemoveDataSourceRequest {
    return {
      instance: isSet(object.instance) ? globalThis.String(object.instance) : "",
      dataSource: isSet(object.dataSource) ? DataSource.fromJSON(object.dataSource) : undefined,
    };
  },

  toJSON(message: RemoveDataSourceRequest): unknown {
    const obj: any = {};
    if (message.instance !== "") {
      obj.instance = message.instance;
    }
    if (message.dataSource !== undefined) {
      obj.dataSource = DataSource.toJSON(message.dataSource);
    }
    return obj;
  },

  create(base?: DeepPartial<RemoveDataSourceRequest>): RemoveDataSourceRequest {
    return RemoveDataSourceRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<RemoveDataSourceRequest>): RemoveDataSourceRequest {
    const message = createBaseRemoveDataSourceRequest();
    message.instance = object.instance ?? "";
    message.dataSource = (object.dataSource !== undefined && object.dataSource !== null)
      ? DataSource.fromPartial(object.dataSource)
      : undefined;
    return message;
  },
};

function createBaseUpdateDataSourceRequest(): UpdateDataSourceRequest {
  return { instance: "", dataSource: undefined, updateMask: undefined, validateOnly: false };
}

export const UpdateDataSourceRequest = {
  encode(message: UpdateDataSourceRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.instance !== "") {
      writer.uint32(10).string(message.instance);
    }
    if (message.dataSource !== undefined) {
      DataSource.encode(message.dataSource, writer.uint32(18).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(26).fork()).ldelim();
    }
    if (message.validateOnly === true) {
      writer.uint32(32).bool(message.validateOnly);
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
          if (tag !== 10) {
            break;
          }

          message.instance = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.dataSource = DataSource.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.validateOnly = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UpdateDataSourceRequest {
    return {
      instance: isSet(object.instance) ? globalThis.String(object.instance) : "",
      dataSource: isSet(object.dataSource) ? DataSource.fromJSON(object.dataSource) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
      validateOnly: isSet(object.validateOnly) ? globalThis.Boolean(object.validateOnly) : false,
    };
  },

  toJSON(message: UpdateDataSourceRequest): unknown {
    const obj: any = {};
    if (message.instance !== "") {
      obj.instance = message.instance;
    }
    if (message.dataSource !== undefined) {
      obj.dataSource = DataSource.toJSON(message.dataSource);
    }
    if (message.updateMask !== undefined) {
      obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask));
    }
    if (message.validateOnly === true) {
      obj.validateOnly = message.validateOnly;
    }
    return obj;
  },

  create(base?: DeepPartial<UpdateDataSourceRequest>): UpdateDataSourceRequest {
    return UpdateDataSourceRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UpdateDataSourceRequest>): UpdateDataSourceRequest {
    const message = createBaseUpdateDataSourceRequest();
    message.instance = object.instance ?? "";
    message.dataSource = (object.dataSource !== undefined && object.dataSource !== null)
      ? DataSource.fromPartial(object.dataSource)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    message.validateOnly = object.validateOnly ?? false;
    return message;
  },
};

function createBaseSyncSlowQueriesRequest(): SyncSlowQueriesRequest {
  return { parent: "" };
}

export const SyncSlowQueriesRequest = {
  encode(message: SyncSlowQueriesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
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
          if (tag !== 10) {
            break;
          }

          message.parent = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SyncSlowQueriesRequest {
    return { parent: isSet(object.parent) ? globalThis.String(object.parent) : "" };
  },

  toJSON(message: SyncSlowQueriesRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    return obj;
  },

  create(base?: DeepPartial<SyncSlowQueriesRequest>): SyncSlowQueriesRequest {
    return SyncSlowQueriesRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SyncSlowQueriesRequest>): SyncSlowQueriesRequest {
    const message = createBaseSyncSlowQueriesRequest();
    message.parent = object.parent ?? "";
    return message;
  },
};

function createBaseInstanceOptions(): InstanceOptions {
  return { schemaTenantMode: false, syncInterval: undefined };
}

export const InstanceOptions = {
  encode(message: InstanceOptions, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schemaTenantMode === true) {
      writer.uint32(8).bool(message.schemaTenantMode);
    }
    if (message.syncInterval !== undefined) {
      Duration.encode(message.syncInterval, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): InstanceOptions {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseInstanceOptions();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.schemaTenantMode = reader.bool();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.syncInterval = Duration.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): InstanceOptions {
    return {
      schemaTenantMode: isSet(object.schemaTenantMode) ? globalThis.Boolean(object.schemaTenantMode) : false,
      syncInterval: isSet(object.syncInterval) ? Duration.fromJSON(object.syncInterval) : undefined,
    };
  },

  toJSON(message: InstanceOptions): unknown {
    const obj: any = {};
    if (message.schemaTenantMode === true) {
      obj.schemaTenantMode = message.schemaTenantMode;
    }
    if (message.syncInterval !== undefined) {
      obj.syncInterval = Duration.toJSON(message.syncInterval);
    }
    return obj;
  },

  create(base?: DeepPartial<InstanceOptions>): InstanceOptions {
    return InstanceOptions.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<InstanceOptions>): InstanceOptions {
    const message = createBaseInstanceOptions();
    message.schemaTenantMode = object.schemaTenantMode ?? false;
    message.syncInterval = (object.syncInterval !== undefined && object.syncInterval !== null)
      ? Duration.fromPartial(object.syncInterval)
      : undefined;
    return message;
  },
};

function createBaseInstance(): Instance {
  return {
    name: "",
    uid: "",
    state: 0,
    title: "",
    engine: 0,
    engineVersion: "",
    externalLink: "",
    dataSources: [],
    environment: "",
    activation: false,
    options: undefined,
  };
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
    if (message.engineVersion !== "") {
      writer.uint32(50).string(message.engineVersion);
    }
    if (message.externalLink !== "") {
      writer.uint32(58).string(message.externalLink);
    }
    for (const v of message.dataSources) {
      DataSource.encode(v!, writer.uint32(66).fork()).ldelim();
    }
    if (message.environment !== "") {
      writer.uint32(74).string(message.environment);
    }
    if (message.activation === true) {
      writer.uint32(80).bool(message.activation);
    }
    if (message.options !== undefined) {
      InstanceOptions.encode(message.options, writer.uint32(90).fork()).ldelim();
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
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.uid = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.state = reader.int32() as any;
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.title = reader.string();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.engine = reader.int32() as any;
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.engineVersion = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.externalLink = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.dataSources.push(DataSource.decode(reader, reader.uint32()));
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.environment = reader.string();
          continue;
        case 10:
          if (tag !== 80) {
            break;
          }

          message.activation = reader.bool();
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.options = InstanceOptions.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Instance {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      uid: isSet(object.uid) ? globalThis.String(object.uid) : "",
      state: isSet(object.state) ? stateFromJSON(object.state) : 0,
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      engine: isSet(object.engine) ? engineFromJSON(object.engine) : 0,
      engineVersion: isSet(object.engineVersion) ? globalThis.String(object.engineVersion) : "",
      externalLink: isSet(object.externalLink) ? globalThis.String(object.externalLink) : "",
      dataSources: globalThis.Array.isArray(object?.dataSources)
        ? object.dataSources.map((e: any) => DataSource.fromJSON(e))
        : [],
      environment: isSet(object.environment) ? globalThis.String(object.environment) : "",
      activation: isSet(object.activation) ? globalThis.Boolean(object.activation) : false,
      options: isSet(object.options) ? InstanceOptions.fromJSON(object.options) : undefined,
    };
  },

  toJSON(message: Instance): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.uid !== "") {
      obj.uid = message.uid;
    }
    if (message.state !== 0) {
      obj.state = stateToJSON(message.state);
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.engine !== 0) {
      obj.engine = engineToJSON(message.engine);
    }
    if (message.engineVersion !== "") {
      obj.engineVersion = message.engineVersion;
    }
    if (message.externalLink !== "") {
      obj.externalLink = message.externalLink;
    }
    if (message.dataSources?.length) {
      obj.dataSources = message.dataSources.map((e) => DataSource.toJSON(e));
    }
    if (message.environment !== "") {
      obj.environment = message.environment;
    }
    if (message.activation === true) {
      obj.activation = message.activation;
    }
    if (message.options !== undefined) {
      obj.options = InstanceOptions.toJSON(message.options);
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
    message.engineVersion = object.engineVersion ?? "";
    message.externalLink = object.externalLink ?? "";
    message.dataSources = object.dataSources?.map((e) => DataSource.fromPartial(e)) || [];
    message.environment = object.environment ?? "";
    message.activation = object.activation ?? false;
    message.options = (object.options !== undefined && object.options !== null)
      ? InstanceOptions.fromPartial(object.options)
      : undefined;
    return message;
  },
};

function createBaseDataSource(): DataSource {
  return {
    id: "",
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
    sshHost: "",
    sshPort: "",
    sshUser: "",
    sshPassword: "",
    sshPrivateKey: "",
  };
}

export const DataSource = {
  encode(message: DataSource, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
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
    if (message.sshHost !== "") {
      writer.uint32(122).string(message.sshHost);
    }
    if (message.sshPort !== "") {
      writer.uint32(130).string(message.sshPort);
    }
    if (message.sshUser !== "") {
      writer.uint32(138).string(message.sshUser);
    }
    if (message.sshPassword !== "") {
      writer.uint32(146).string(message.sshPassword);
    }
    if (message.sshPrivateKey !== "") {
      writer.uint32(154).string(message.sshPrivateKey);
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
          if (tag !== 10) {
            break;
          }

          message.id = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.type = reader.int32() as any;
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.username = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.password = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.sslCa = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.sslCert = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.sslKey = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.host = reader.string();
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.port = reader.string();
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.database = reader.string();
          continue;
        case 11:
          if (tag !== 88) {
            break;
          }

          message.srv = reader.bool();
          continue;
        case 12:
          if (tag !== 98) {
            break;
          }

          message.authenticationDatabase = reader.string();
          continue;
        case 13:
          if (tag !== 106) {
            break;
          }

          message.sid = reader.string();
          continue;
        case 14:
          if (tag !== 114) {
            break;
          }

          message.serviceName = reader.string();
          continue;
        case 15:
          if (tag !== 122) {
            break;
          }

          message.sshHost = reader.string();
          continue;
        case 16:
          if (tag !== 130) {
            break;
          }

          message.sshPort = reader.string();
          continue;
        case 17:
          if (tag !== 138) {
            break;
          }

          message.sshUser = reader.string();
          continue;
        case 18:
          if (tag !== 146) {
            break;
          }

          message.sshPassword = reader.string();
          continue;
        case 19:
          if (tag !== 154) {
            break;
          }

          message.sshPrivateKey = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DataSource {
    return {
      id: isSet(object.id) ? globalThis.String(object.id) : "",
      type: isSet(object.type) ? dataSourceTypeFromJSON(object.type) : 0,
      username: isSet(object.username) ? globalThis.String(object.username) : "",
      password: isSet(object.password) ? globalThis.String(object.password) : "",
      sslCa: isSet(object.sslCa) ? globalThis.String(object.sslCa) : "",
      sslCert: isSet(object.sslCert) ? globalThis.String(object.sslCert) : "",
      sslKey: isSet(object.sslKey) ? globalThis.String(object.sslKey) : "",
      host: isSet(object.host) ? globalThis.String(object.host) : "",
      port: isSet(object.port) ? globalThis.String(object.port) : "",
      database: isSet(object.database) ? globalThis.String(object.database) : "",
      srv: isSet(object.srv) ? globalThis.Boolean(object.srv) : false,
      authenticationDatabase: isSet(object.authenticationDatabase)
        ? globalThis.String(object.authenticationDatabase)
        : "",
      sid: isSet(object.sid) ? globalThis.String(object.sid) : "",
      serviceName: isSet(object.serviceName) ? globalThis.String(object.serviceName) : "",
      sshHost: isSet(object.sshHost) ? globalThis.String(object.sshHost) : "",
      sshPort: isSet(object.sshPort) ? globalThis.String(object.sshPort) : "",
      sshUser: isSet(object.sshUser) ? globalThis.String(object.sshUser) : "",
      sshPassword: isSet(object.sshPassword) ? globalThis.String(object.sshPassword) : "",
      sshPrivateKey: isSet(object.sshPrivateKey) ? globalThis.String(object.sshPrivateKey) : "",
    };
  },

  toJSON(message: DataSource): unknown {
    const obj: any = {};
    if (message.id !== "") {
      obj.id = message.id;
    }
    if (message.type !== 0) {
      obj.type = dataSourceTypeToJSON(message.type);
    }
    if (message.username !== "") {
      obj.username = message.username;
    }
    if (message.password !== "") {
      obj.password = message.password;
    }
    if (message.sslCa !== "") {
      obj.sslCa = message.sslCa;
    }
    if (message.sslCert !== "") {
      obj.sslCert = message.sslCert;
    }
    if (message.sslKey !== "") {
      obj.sslKey = message.sslKey;
    }
    if (message.host !== "") {
      obj.host = message.host;
    }
    if (message.port !== "") {
      obj.port = message.port;
    }
    if (message.database !== "") {
      obj.database = message.database;
    }
    if (message.srv === true) {
      obj.srv = message.srv;
    }
    if (message.authenticationDatabase !== "") {
      obj.authenticationDatabase = message.authenticationDatabase;
    }
    if (message.sid !== "") {
      obj.sid = message.sid;
    }
    if (message.serviceName !== "") {
      obj.serviceName = message.serviceName;
    }
    if (message.sshHost !== "") {
      obj.sshHost = message.sshHost;
    }
    if (message.sshPort !== "") {
      obj.sshPort = message.sshPort;
    }
    if (message.sshUser !== "") {
      obj.sshUser = message.sshUser;
    }
    if (message.sshPassword !== "") {
      obj.sshPassword = message.sshPassword;
    }
    if (message.sshPrivateKey !== "") {
      obj.sshPrivateKey = message.sshPrivateKey;
    }
    return obj;
  },

  create(base?: DeepPartial<DataSource>): DataSource {
    return DataSource.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DataSource>): DataSource {
    const message = createBaseDataSource();
    message.id = object.id ?? "";
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
    message.sshHost = object.sshHost ?? "";
    message.sshPort = object.sshPort ?? "";
    message.sshUser = object.sshUser ?? "";
    message.sshPassword = object.sshPassword ?? "";
    message.sshPrivateKey = object.sshPrivateKey ?? "";
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
              24,
              18,
              22,
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
          8410: [new Uint8Array([0])],
          578365826: [
            new Uint8Array([
              52,
              90,
              35,
              18,
              33,
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
              112,
              114,
              111,
              106,
              101,
              99,
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
              18,
              13,
              47,
              118,
              49,
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
          8410: [new Uint8Array([8, 105, 110, 115, 116, 97, 110, 99, 101])],
          578365826: [
            new Uint8Array([
              25,
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
              13,
              47,
              118,
              49,
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
              43,
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
              31,
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
              24,
              42,
              22,
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
              36,
              58,
              1,
              42,
              34,
              31,
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
    syncInstance: {
      name: "SyncInstance",
      requestType: SyncInstanceRequest,
      requestStream: false,
      responseType: SyncInstanceResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              32,
              58,
              1,
              42,
              34,
              27,
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
            ]),
          ],
        },
      },
    },
    batchSyncInstance: {
      name: "BatchSyncInstance",
      requestType: BatchSyncInstanceRequest,
      requestStream: false,
      responseType: BatchSyncInstanceResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              28,
              58,
              1,
              42,
              34,
              23,
              47,
              118,
              49,
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
              58,
              98,
              97,
              116,
              99,
              104,
              83,
              121,
              110,
              99,
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
              45,
              58,
              1,
              42,
              34,
              40,
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
              48,
              58,
              1,
              42,
              34,
              43,
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
              48,
              58,
              1,
              42,
              50,
              43,
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
              88,
              58,
              1,
              42,
              90,
              41,
              34,
              39,
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
              112,
              114,
              111,
              106,
              101,
              99,
              116,
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
              34,
              40,
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

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Long ? string | number | Long : T extends globalThis.Array<infer U> ? globalThis.Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
