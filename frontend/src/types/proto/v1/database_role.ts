/* eslint-disable */
import _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.v1";

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
  /** The role unique name. */
  name: string;
  /** The Bytebase instance id for this role. */
  instanceId: number;
  /** The connection count limit for this role. */
  connectionLimit: number;
  /** The expiration for the role's password. */
  validUntil?:
    | string
    | undefined;
  /** The role attribute. */
  attribute?: DatabaseRoleAttribute;
}

/** ListDatabaseRoleResponse is the API message for role list. */
export interface ListDatabaseRoleResponse {
  roles: DatabaseRole[];
}

/** DatabaseRoleUpsert is the API message for upserting a database role. */
export interface DatabaseRoleUpsert {
  /** The role unique name. */
  name: string;
  /** A password is only significant if the client authentication method requires the user to supply a password when connecting to the database. */
  password?:
    | string
    | undefined;
  /** Connection limit can specify how many concurrent connections a role can make. -1 (the default) means no limit. */
  connectionLimit?:
    | number
    | undefined;
  /** The VALID UNTIL clause sets a date and time after which the role's password is no longer valid. If this clause is omitted the password will be valid for all time. */
  validUntil?:
    | string
    | undefined;
  /** The role attribute. */
  attribute?: DatabaseRoleAttribute | undefined;
}

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
  return { name: "", instanceId: 0, connectionLimit: 0, validUntil: undefined, attribute: undefined };
}

export const DatabaseRole = {
  encode(message: DatabaseRole, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.instanceId !== 0) {
      writer.uint32(16).int32(message.instanceId);
    }
    if (message.connectionLimit !== 0) {
      writer.uint32(24).int32(message.connectionLimit);
    }
    if (message.validUntil !== undefined) {
      writer.uint32(34).string(message.validUntil);
    }
    if (message.attribute !== undefined) {
      DatabaseRoleAttribute.encode(message.attribute, writer.uint32(42).fork()).ldelim();
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
          message.instanceId = reader.int32();
          break;
        case 3:
          message.connectionLimit = reader.int32();
          break;
        case 4:
          message.validUntil = reader.string();
          break;
        case 5:
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
      instanceId: isSet(object.instanceId) ? Number(object.instanceId) : 0,
      connectionLimit: isSet(object.connectionLimit) ? Number(object.connectionLimit) : 0,
      validUntil: isSet(object.validUntil) ? String(object.validUntil) : undefined,
      attribute: isSet(object.attribute) ? DatabaseRoleAttribute.fromJSON(object.attribute) : undefined,
    };
  },

  toJSON(message: DatabaseRole): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.instanceId !== undefined && (obj.instanceId = Math.round(message.instanceId));
    message.connectionLimit !== undefined && (obj.connectionLimit = Math.round(message.connectionLimit));
    message.validUntil !== undefined && (obj.validUntil = message.validUntil);
    message.attribute !== undefined &&
      (obj.attribute = message.attribute ? DatabaseRoleAttribute.toJSON(message.attribute) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DatabaseRole>, I>>(object: I): DatabaseRole {
    const message = createBaseDatabaseRole();
    message.name = object.name ?? "";
    message.instanceId = object.instanceId ?? 0;
    message.connectionLimit = object.connectionLimit ?? 0;
    message.validUntil = object.validUntil ?? undefined;
    message.attribute = (object.attribute !== undefined && object.attribute !== null)
      ? DatabaseRoleAttribute.fromPartial(object.attribute)
      : undefined;
    return message;
  },
};

function createBaseListDatabaseRoleResponse(): ListDatabaseRoleResponse {
  return { roles: [] };
}

export const ListDatabaseRoleResponse = {
  encode(message: ListDatabaseRoleResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.roles) {
      DatabaseRole.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListDatabaseRoleResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListDatabaseRoleResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.roles.push(DatabaseRole.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ListDatabaseRoleResponse {
    return { roles: Array.isArray(object?.roles) ? object.roles.map((e: any) => DatabaseRole.fromJSON(e)) : [] };
  },

  toJSON(message: ListDatabaseRoleResponse): unknown {
    const obj: any = {};
    if (message.roles) {
      obj.roles = message.roles.map((e) => e ? DatabaseRole.toJSON(e) : undefined);
    } else {
      obj.roles = [];
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ListDatabaseRoleResponse>, I>>(object: I): ListDatabaseRoleResponse {
    const message = createBaseListDatabaseRoleResponse();
    message.roles = object.roles?.map((e) => DatabaseRole.fromPartial(e)) || [];
    return message;
  },
};

function createBaseDatabaseRoleUpsert(): DatabaseRoleUpsert {
  return { name: "", password: undefined, connectionLimit: undefined, validUntil: undefined, attribute: undefined };
}

export const DatabaseRoleUpsert = {
  encode(message: DatabaseRoleUpsert, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.password !== undefined) {
      writer.uint32(18).string(message.password);
    }
    if (message.connectionLimit !== undefined) {
      writer.uint32(24).int32(message.connectionLimit);
    }
    if (message.validUntil !== undefined) {
      writer.uint32(34).string(message.validUntil);
    }
    if (message.attribute !== undefined) {
      DatabaseRoleAttribute.encode(message.attribute, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DatabaseRoleUpsert {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDatabaseRoleUpsert();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.password = reader.string();
          break;
        case 3:
          message.connectionLimit = reader.int32();
          break;
        case 4:
          message.validUntil = reader.string();
          break;
        case 5:
          message.attribute = DatabaseRoleAttribute.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DatabaseRoleUpsert {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      password: isSet(object.password) ? String(object.password) : undefined,
      connectionLimit: isSet(object.connectionLimit) ? Number(object.connectionLimit) : undefined,
      validUntil: isSet(object.validUntil) ? String(object.validUntil) : undefined,
      attribute: isSet(object.attribute) ? DatabaseRoleAttribute.fromJSON(object.attribute) : undefined,
    };
  },

  toJSON(message: DatabaseRoleUpsert): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.password !== undefined && (obj.password = message.password);
    message.connectionLimit !== undefined && (obj.connectionLimit = Math.round(message.connectionLimit));
    message.validUntil !== undefined && (obj.validUntil = message.validUntil);
    message.attribute !== undefined &&
      (obj.attribute = message.attribute ? DatabaseRoleAttribute.toJSON(message.attribute) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DatabaseRoleUpsert>, I>>(object: I): DatabaseRoleUpsert {
    const message = createBaseDatabaseRoleUpsert();
    message.name = object.name ?? "";
    message.password = object.password ?? undefined;
    message.connectionLimit = object.connectionLimit ?? undefined;
    message.validUntil = object.validUntil ?? undefined;
    message.attribute = (object.attribute !== undefined && object.attribute !== null)
      ? DatabaseRoleAttribute.fromPartial(object.attribute)
      : undefined;
    return message;
  },
};

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
