/* eslint-disable */
import _m0 from "protobufjs/minimal";
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
  roles: DatabaseRole[];
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
  role?: DatabaseRole;
}

export interface UpdateRoleRequest {
  /**
   * The role to update.
   *
   * The role's `name`, `environment` and `instance` field is used to identify the role to update.
   * Format: environments/{environment}/instances/{instance}/roles/{role name}
   */
  role?: DatabaseRole;
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

/** RoleAttribute is the attribute for role. Docs: https://www.postgresql.org/docs/current/role-attributes.html */
export interface DatabaseRoleAttribute {
  /** A database superuser bypasses all permission checks, except the right to log in. */
  superUser: boolean;
  /** A role is given permission to inherit the privileges of roles it is a member of. To create a role without the permission, use "noInherit = true" */
  noInherit: boolean;
  /** A role must be explicitly given permission to create more roles (except for superusers, since those bypass all permission checks). */
  createRole: boolean;
  /** A role must be explicitly given permission to create databases (except for superusers, since those bypass all permission checks). */
  createDb: boolean;
  /** Only roles that have the LOGIN attribute can be used as the initial role name for a database connection. */
  canLogin: boolean;
  /** A role must explicitly be given permission to initiate streaming replication (except for superusers, since those bypass all permission checks). */
  replication: boolean;
  /** A role must be explicitly given permission to bypass every row-level security (RLS) policy (except for superusers, since those bypass all permission checks). */
  bypassRls: boolean;
}

/** DatabaseRole is the API message for database role. */
export interface DatabaseRole {
  /**
   * The name of the role.
   * Format: environments/{environment}/instances/{instance}/roles/{role name}
   * The role name is the unique name for the role.
   */
  name: string;
  /** The role unique name. */
  title: string;
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
  /** The role attribute. */
  attribute?: DatabaseRoleAttribute;
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

  fromPartial<I extends Exact<DeepPartial<GetRoleRequest>, I>>(object: I): GetRoleRequest {
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

  fromPartial<I extends Exact<DeepPartial<ListRolesRequest>, I>>(object: I): ListRolesRequest {
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
      DatabaseRole.encode(v!, writer.uint32(10).fork()).ldelim();
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
          message.roles.push(DatabaseRole.decode(reader, reader.uint32()));
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
      roles: Array.isArray(object?.roles) ? object.roles.map((e: any) => DatabaseRole.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListRolesResponse): unknown {
    const obj: any = {};
    if (message.roles) {
      obj.roles = message.roles.map((e) => e ? DatabaseRole.toJSON(e) : undefined);
    } else {
      obj.roles = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ListRolesResponse>, I>>(object: I): ListRolesResponse {
    const message = createBaseListRolesResponse();
    message.roles = object.roles?.map((e) => DatabaseRole.fromPartial(e)) || [];
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
      DatabaseRole.encode(message.role, writer.uint32(18).fork()).ldelim();
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
          message.role = DatabaseRole.decode(reader, reader.uint32());
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
      role: isSet(object.role) ? DatabaseRole.fromJSON(object.role) : undefined,
    };
  },

  toJSON(message: CreateRoleRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.role !== undefined && (obj.role = message.role ? DatabaseRole.toJSON(message.role) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<CreateRoleRequest>, I>>(object: I): CreateRoleRequest {
    const message = createBaseCreateRoleRequest();
    message.parent = object.parent ?? "";
    message.role = (object.role !== undefined && object.role !== null)
      ? DatabaseRole.fromPartial(object.role)
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
      DatabaseRole.encode(message.role, writer.uint32(10).fork()).ldelim();
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
          message.role = DatabaseRole.decode(reader, reader.uint32());
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
      role: isSet(object.role) ? DatabaseRole.fromJSON(object.role) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateRoleRequest): unknown {
    const obj: any = {};
    message.role !== undefined && (obj.role = message.role ? DatabaseRole.toJSON(message.role) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<UpdateRoleRequest>, I>>(object: I): UpdateRoleRequest {
    const message = createBaseUpdateRoleRequest();
    message.role = (object.role !== undefined && object.role !== null)
      ? DatabaseRole.fromPartial(object.role)
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

  fromPartial<I extends Exact<DeepPartial<DeleteRoleRequest>, I>>(object: I): DeleteRoleRequest {
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

  fromPartial<I extends Exact<DeepPartial<UndeleteRoleRequest>, I>>(object: I): UndeleteRoleRequest {
    const message = createBaseUndeleteRoleRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseDatabaseRoleAttribute(): DatabaseRoleAttribute {
  return {
    superUser: false,
    noInherit: false,
    createRole: false,
    createDb: false,
    canLogin: false,
    replication: false,
    bypassRls: false,
  };
}

export const DatabaseRoleAttribute = {
  encode(message: DatabaseRoleAttribute, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.superUser === true) {
      writer.uint32(8).bool(message.superUser);
    }
    if (message.noInherit === true) {
      writer.uint32(16).bool(message.noInherit);
    }
    if (message.createRole === true) {
      writer.uint32(24).bool(message.createRole);
    }
    if (message.createDb === true) {
      writer.uint32(32).bool(message.createDb);
    }
    if (message.canLogin === true) {
      writer.uint32(40).bool(message.canLogin);
    }
    if (message.replication === true) {
      writer.uint32(48).bool(message.replication);
    }
    if (message.bypassRls === true) {
      writer.uint32(56).bool(message.bypassRls);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DatabaseRoleAttribute {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDatabaseRoleAttribute();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.superUser = reader.bool();
          break;
        case 2:
          message.noInherit = reader.bool();
          break;
        case 3:
          message.createRole = reader.bool();
          break;
        case 4:
          message.createDb = reader.bool();
          break;
        case 5:
          message.canLogin = reader.bool();
          break;
        case 6:
          message.replication = reader.bool();
          break;
        case 7:
          message.bypassRls = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DatabaseRoleAttribute {
    return {
      superUser: isSet(object.superUser) ? Boolean(object.superUser) : false,
      noInherit: isSet(object.noInherit) ? Boolean(object.noInherit) : false,
      createRole: isSet(object.createRole) ? Boolean(object.createRole) : false,
      createDb: isSet(object.createDb) ? Boolean(object.createDb) : false,
      canLogin: isSet(object.canLogin) ? Boolean(object.canLogin) : false,
      replication: isSet(object.replication) ? Boolean(object.replication) : false,
      bypassRls: isSet(object.bypassRls) ? Boolean(object.bypassRls) : false,
    };
  },

  toJSON(message: DatabaseRoleAttribute): unknown {
    const obj: any = {};
    message.superUser !== undefined && (obj.superUser = message.superUser);
    message.noInherit !== undefined && (obj.noInherit = message.noInherit);
    message.createRole !== undefined && (obj.createRole = message.createRole);
    message.createDb !== undefined && (obj.createDb = message.createDb);
    message.canLogin !== undefined && (obj.canLogin = message.canLogin);
    message.replication !== undefined && (obj.replication = message.replication);
    message.bypassRls !== undefined && (obj.bypassRls = message.bypassRls);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DatabaseRoleAttribute>, I>>(object: I): DatabaseRoleAttribute {
    const message = createBaseDatabaseRoleAttribute();
    message.superUser = object.superUser ?? false;
    message.noInherit = object.noInherit ?? false;
    message.createRole = object.createRole ?? false;
    message.createDb = object.createDb ?? false;
    message.canLogin = object.canLogin ?? false;
    message.replication = object.replication ?? false;
    message.bypassRls = object.bypassRls ?? false;
    return message;
  },
};

function createBaseDatabaseRole(): DatabaseRole {
  return {
    name: "",
    title: "",
    password: undefined,
    connectionLimit: undefined,
    validUntil: undefined,
    attribute: undefined,
  };
}

export const DatabaseRole = {
  encode(message: DatabaseRole, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
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
      DatabaseRoleAttribute.encode(message.attribute, writer.uint32(50).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DatabaseRole {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDatabaseRole();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.title = reader.string();
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
          message.attribute = DatabaseRoleAttribute.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DatabaseRole {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      title: isSet(object.title) ? String(object.title) : "",
      password: isSet(object.password) ? String(object.password) : undefined,
      connectionLimit: isSet(object.connectionLimit) ? Number(object.connectionLimit) : undefined,
      validUntil: isSet(object.validUntil) ? String(object.validUntil) : undefined,
      attribute: isSet(object.attribute) ? DatabaseRoleAttribute.fromJSON(object.attribute) : undefined,
    };
  },

  toJSON(message: DatabaseRole): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.title !== undefined && (obj.title = message.title);
    message.password !== undefined && (obj.password = message.password);
    message.connectionLimit !== undefined && (obj.connectionLimit = Math.round(message.connectionLimit));
    message.validUntil !== undefined && (obj.validUntil = message.validUntil);
    message.attribute !== undefined &&
      (obj.attribute = message.attribute ? DatabaseRoleAttribute.toJSON(message.attribute) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DatabaseRole>, I>>(object: I): DatabaseRole {
    const message = createBaseDatabaseRole();
    message.name = object.name ?? "";
    message.title = object.title ?? "";
    message.password = object.password ?? undefined;
    message.connectionLimit = object.connectionLimit ?? undefined;
    message.validUntil = object.validUntil ?? undefined;
    message.attribute = (object.attribute !== undefined && object.attribute !== null)
      ? DatabaseRoleAttribute.fromPartial(object.attribute)
      : undefined;
    return message;
  },
};

export interface InstanceRoleService {
  GetRole(request: GetRoleRequest): Promise<DatabaseRole>;
  ListRoles(request: ListRolesRequest): Promise<ListRolesResponse>;
  CreateRole(request: CreateRoleRequest): Promise<DatabaseRole>;
  UpdateRole(request: UpdateRoleRequest): Promise<DatabaseRole>;
  DeleteRole(request: DeleteRoleRequest): Promise<Empty>;
  UndeleteRole(request: UndeleteRoleRequest): Promise<DatabaseRole>;
}

export class InstanceRoleServiceClientImpl implements InstanceRoleService {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "bytebase.v1.InstanceRoleService";
    this.rpc = rpc;
    this.GetRole = this.GetRole.bind(this);
    this.ListRoles = this.ListRoles.bind(this);
    this.CreateRole = this.CreateRole.bind(this);
    this.UpdateRole = this.UpdateRole.bind(this);
    this.DeleteRole = this.DeleteRole.bind(this);
    this.UndeleteRole = this.UndeleteRole.bind(this);
  }
  GetRole(request: GetRoleRequest): Promise<DatabaseRole> {
    const data = GetRoleRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "GetRole", data);
    return promise.then((data) => DatabaseRole.decode(new _m0.Reader(data)));
  }

  ListRoles(request: ListRolesRequest): Promise<ListRolesResponse> {
    const data = ListRolesRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "ListRoles", data);
    return promise.then((data) => ListRolesResponse.decode(new _m0.Reader(data)));
  }

  CreateRole(request: CreateRoleRequest): Promise<DatabaseRole> {
    const data = CreateRoleRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "CreateRole", data);
    return promise.then((data) => DatabaseRole.decode(new _m0.Reader(data)));
  }

  UpdateRole(request: UpdateRoleRequest): Promise<DatabaseRole> {
    const data = UpdateRoleRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "UpdateRole", data);
    return promise.then((data) => DatabaseRole.decode(new _m0.Reader(data)));
  }

  DeleteRole(request: DeleteRoleRequest): Promise<Empty> {
    const data = DeleteRoleRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "DeleteRole", data);
    return promise.then((data) => Empty.decode(new _m0.Reader(data)));
  }

  UndeleteRole(request: UndeleteRoleRequest): Promise<DatabaseRole> {
    const data = UndeleteRoleRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "UndeleteRole", data);
    return promise.then((data) => DatabaseRole.decode(new _m0.Reader(data)));
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
