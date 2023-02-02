/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";

export const protobufPackage = "bytebase.v1";

export interface GetRoleRequest {
  /**
   * The name of the role to retrieve.
   * Format: environments/{environment}/instances/{instance}/roles/{role name}
   * The role name is the unique name for the role.
   */
  name: string;
}

export interface ListRolesRequest {
  /**
   * The parent, which owns this collection of roles.
   * Format: environments/{environment}/instances/{instance}
   */
  parent: string;
  /**
   * The maximum number of roles to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 roles will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListRoles` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListRoles` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface ListRolesResponse {
  /** The roles from the specified request. */
  roles: InstanceRole[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface CreateRoleRequest {
  /**
   * The parent resource where this role will be created.
   * Format: environments/{environment}/instances/{instance}
   */
  parent: string;
  /** The role to create. */
  role?: InstanceRole;
}

export interface UpdateRoleRequest {
  /**
   * The role to update.
   *
   * The role's `name`, `environment` and `instance` field is used to identify the role to update.
   * Format: environments/{environment}/instances/{instance}/roles/{role name}
   */
  role?: InstanceRole;
  /** The list of fields to update. */
  updateMask?: string[];
}

export interface DeleteRoleRequest {
  /**
   * The name of the role to delete.
   * Format: environments/{environment}/instances/{instance}/roles/{role name}
   */
  name: string;
}

export interface UndeleteRoleRequest {
  /**
   * The name of the deleted role.
   * Format: environments/{environment}/instances/{instance}/roles/{role name}
   */
  name: string;
}

/** InstanceRole is the API message for instance role. */
export interface InstanceRole {
  /**
   * The name of the role.
   * Format: environments/{environment}/instances/{instance}/roles/{role name}
   * The role name is the unique name for the role.
   */
  name: string;
  /** The role name. It's unique within the instance. */
  roleName: string;
  /** The role password. */
  password?:
    | string
    | undefined;
  /** The connection count limit for this role. */
  connectionLimit?:
    | number
    | undefined;
  /** The expiration for the role's password. */
  validUntil?:
    | string
    | undefined;
  /**
   * The role attribute.
   * For PostgreSQL, it containt super_user, no_inherit, create_role, create_db, can_login, replication and bypass_rls. Docs: https://www.postgresql.org/docs/current/role-attributes.html
   * For MySQL, it's the global privileges as GRANT statements, which means it only contains "GRANT ... ON *.* TO ...". Docs: https://dev.mysql.com/doc/refman/8.0/en/grant.html
   */
  attribute?: string | undefined;
}

function createBaseGetRoleRequest(): GetRoleRequest {
  return { name: "" };
}

export const GetRoleRequest = {
  encode(message: GetRoleRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetRoleRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetRoleRequest();
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

  fromJSON(object: any): GetRoleRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetRoleRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<GetRoleRequest>): GetRoleRequest {
    const message = createBaseGetRoleRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListRolesRequest(): ListRolesRequest {
  return { parent: "", pageSize: 0, pageToken: "" };
}

export const ListRolesRequest = {
  encode(message: ListRolesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.pageSize !== 0) {
      writer.uint32(16).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(26).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListRolesRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListRolesRequest();
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
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ListRolesRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListRolesRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    return obj;
  },

  fromPartial(object: DeepPartial<ListRolesRequest>): ListRolesRequest {
    const message = createBaseListRolesRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListRolesResponse(): ListRolesResponse {
  return { roles: [], nextPageToken: "" };
}

export const ListRolesResponse = {
  encode(message: ListRolesResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.roles) {
      InstanceRole.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListRolesResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListRolesResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.roles.push(InstanceRole.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListRolesResponse {
    return {
      roles: Array.isArray(object?.roles) ? object.roles.map((e: any) => InstanceRole.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListRolesResponse): unknown {
    const obj: any = {};
    if (message.roles) {
      obj.roles = message.roles.map((e) => e ? InstanceRole.toJSON(e) : undefined);
    } else {
      obj.roles = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  fromPartial(object: DeepPartial<ListRolesResponse>): ListRolesResponse {
    const message = createBaseListRolesResponse();
    message.roles = object.roles?.map((e) => InstanceRole.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseCreateRoleRequest(): CreateRoleRequest {
  return { parent: "", role: undefined };
}

export const CreateRoleRequest = {
  encode(message: CreateRoleRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.role !== undefined) {
      InstanceRole.encode(message.role, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateRoleRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateRoleRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.parent = reader.string();
          break;
        case 2:
          message.role = InstanceRole.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): CreateRoleRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      role: isSet(object.role) ? InstanceRole.fromJSON(object.role) : undefined,
    };
  },

  toJSON(message: CreateRoleRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.role !== undefined && (obj.role = message.role ? InstanceRole.toJSON(message.role) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<CreateRoleRequest>): CreateRoleRequest {
    const message = createBaseCreateRoleRequest();
    message.parent = object.parent ?? "";
    message.role = (object.role !== undefined && object.role !== null)
      ? InstanceRole.fromPartial(object.role)
      : undefined;
    return message;
  },
};

function createBaseUpdateRoleRequest(): UpdateRoleRequest {
  return { role: undefined, updateMask: undefined };
}

export const UpdateRoleRequest = {
  encode(message: UpdateRoleRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.role !== undefined) {
      InstanceRole.encode(message.role, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateRoleRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateRoleRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.role = InstanceRole.decode(reader, reader.uint32());
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

  fromJSON(object: any): UpdateRoleRequest {
    return {
      role: isSet(object.role) ? InstanceRole.fromJSON(object.role) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateRoleRequest): unknown {
    const obj: any = {};
    message.role !== undefined && (obj.role = message.role ? InstanceRole.toJSON(message.role) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  fromPartial(object: DeepPartial<UpdateRoleRequest>): UpdateRoleRequest {
    const message = createBaseUpdateRoleRequest();
    message.role = (object.role !== undefined && object.role !== null)
      ? InstanceRole.fromPartial(object.role)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseDeleteRoleRequest(): DeleteRoleRequest {
  return { name: "" };
}

export const DeleteRoleRequest = {
  encode(message: DeleteRoleRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteRoleRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteRoleRequest();
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

  fromJSON(object: any): DeleteRoleRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: DeleteRoleRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<DeleteRoleRequest>): DeleteRoleRequest {
    const message = createBaseDeleteRoleRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseUndeleteRoleRequest(): UndeleteRoleRequest {
  return { name: "" };
}

export const UndeleteRoleRequest = {
  encode(message: UndeleteRoleRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UndeleteRoleRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUndeleteRoleRequest();
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

  fromJSON(object: any): UndeleteRoleRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: UndeleteRoleRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<UndeleteRoleRequest>): UndeleteRoleRequest {
    const message = createBaseUndeleteRoleRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseInstanceRole(): InstanceRole {
  return {
    name: "",
    roleName: "",
    password: undefined,
    connectionLimit: undefined,
    validUntil: undefined,
    attribute: undefined,
  };
}

export const InstanceRole = {
  encode(message: InstanceRole, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.roleName !== "") {
      writer.uint32(18).string(message.roleName);
    }
    if (message.password !== undefined) {
      writer.uint32(26).string(message.password);
    }
    if (message.connectionLimit !== undefined) {
      writer.uint32(32).int32(message.connectionLimit);
    }
    if (message.validUntil !== undefined) {
      writer.uint32(42).string(message.validUntil);
    }
    if (message.attribute !== undefined) {
      writer.uint32(50).string(message.attribute);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): InstanceRole {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseInstanceRole();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.roleName = reader.string();
          break;
        case 3:
          message.password = reader.string();
          break;
        case 4:
          message.connectionLimit = reader.int32();
          break;
        case 5:
          message.validUntil = reader.string();
          break;
        case 6:
          message.attribute = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): InstanceRole {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      roleName: isSet(object.roleName) ? String(object.roleName) : "",
      password: isSet(object.password) ? String(object.password) : undefined,
      connectionLimit: isSet(object.connectionLimit) ? Number(object.connectionLimit) : undefined,
      validUntil: isSet(object.validUntil) ? String(object.validUntil) : undefined,
      attribute: isSet(object.attribute) ? String(object.attribute) : undefined,
    };
  },

  toJSON(message: InstanceRole): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.roleName !== undefined && (obj.roleName = message.roleName);
    message.password !== undefined && (obj.password = message.password);
    message.connectionLimit !== undefined && (obj.connectionLimit = Math.round(message.connectionLimit));
    message.validUntil !== undefined && (obj.validUntil = message.validUntil);
    message.attribute !== undefined && (obj.attribute = message.attribute);
    return obj;
  },

  fromPartial(object: DeepPartial<InstanceRole>): InstanceRole {
    const message = createBaseInstanceRole();
    message.name = object.name ?? "";
    message.roleName = object.roleName ?? "";
    message.password = object.password ?? undefined;
    message.connectionLimit = object.connectionLimit ?? undefined;
    message.validUntil = object.validUntil ?? undefined;
    message.attribute = object.attribute ?? undefined;
    return message;
  },
};

export type InstanceRoleServiceDefinition = typeof InstanceRoleServiceDefinition;
export const InstanceRoleServiceDefinition = {
  name: "InstanceRoleService",
  fullName: "bytebase.v1.InstanceRoleService",
  methods: {
    getRole: {
      name: "GetRole",
      requestType: GetRoleRequest,
      requestStream: false,
      responseType: InstanceRole,
      responseStream: false,
      options: {},
    },
    listRoles: {
      name: "ListRoles",
      requestType: ListRolesRequest,
      requestStream: false,
      responseType: ListRolesResponse,
      responseStream: false,
      options: {},
    },
    createRole: {
      name: "CreateRole",
      requestType: CreateRoleRequest,
      requestStream: false,
      responseType: InstanceRole,
      responseStream: false,
      options: {},
    },
    updateRole: {
      name: "UpdateRole",
      requestType: UpdateRoleRequest,
      requestStream: false,
      responseType: InstanceRole,
      responseStream: false,
      options: {},
    },
    deleteRole: {
      name: "DeleteRole",
      requestType: DeleteRoleRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {},
    },
    undeleteRole: {
      name: "UndeleteRole",
      requestType: UndeleteRoleRequest,
      requestStream: false,
      responseType: InstanceRole,
      responseStream: false,
      options: {},
    },
  },
} as const;

export interface InstanceRoleServiceImplementation<CallContextExt = {}> {
  getRole(request: GetRoleRequest, context: CallContext & CallContextExt): Promise<DeepPartial<InstanceRole>>;
  listRoles(request: ListRolesRequest, context: CallContext & CallContextExt): Promise<DeepPartial<ListRolesResponse>>;
  createRole(request: CreateRoleRequest, context: CallContext & CallContextExt): Promise<DeepPartial<InstanceRole>>;
  updateRole(request: UpdateRoleRequest, context: CallContext & CallContextExt): Promise<DeepPartial<InstanceRole>>;
  deleteRole(request: DeleteRoleRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Empty>>;
  undeleteRole(request: UndeleteRoleRequest, context: CallContext & CallContextExt): Promise<DeepPartial<InstanceRole>>;
}

export interface InstanceRoleServiceClient<CallOptionsExt = {}> {
  getRole(request: DeepPartial<GetRoleRequest>, options?: CallOptions & CallOptionsExt): Promise<InstanceRole>;
  listRoles(request: DeepPartial<ListRolesRequest>, options?: CallOptions & CallOptionsExt): Promise<ListRolesResponse>;
  createRole(request: DeepPartial<CreateRoleRequest>, options?: CallOptions & CallOptionsExt): Promise<InstanceRole>;
  updateRole(request: DeepPartial<UpdateRoleRequest>, options?: CallOptions & CallOptionsExt): Promise<InstanceRole>;
  deleteRole(request: DeepPartial<DeleteRoleRequest>, options?: CallOptions & CallOptionsExt): Promise<Empty>;
  undeleteRole(
    request: DeepPartial<UndeleteRoleRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<InstanceRole>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
