/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";

export const protobufPackage = "bytebase.v1";

export interface GetInstanceRoleRequest {
  /**
   * The name of the role to retrieve.
   * Format: environments/{environment}/instances/{instance}/roles/{role name}
   * The role name is the unique name for the role.
   */
  name: string;
}

export interface ListInstanceRolesRequest {
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

export interface ListInstanceRolesResponse {
  /** The roles from the specified request. */
  roles: InstanceRole[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface CreateInstanceRoleRequest {
  /**
   * The parent resource where this role will be created.
   * Format: environments/{environment}/instances/{instance}
   */
  parent: string;
  /** The role to create. */
  role?: InstanceRole;
}

export interface UpdateInstanceRoleRequest {
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

export interface DeleteInstanceRoleRequest {
  /**
   * The name of the role to delete.
   * Format: environments/{environment}/instances/{instance}/roles/{role name}
   */
  name: string;
}

export interface UndeleteInstanceRoleRequest {
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

function createBaseGetInstanceRoleRequest(): GetInstanceRoleRequest {
  return { name: "" };
}

export const GetInstanceRoleRequest = {
  encode(message: GetInstanceRoleRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetInstanceRoleRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetInstanceRoleRequest();
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

  fromJSON(object: any): GetInstanceRoleRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetInstanceRoleRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<GetInstanceRoleRequest>): GetInstanceRoleRequest {
    return GetInstanceRoleRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<GetInstanceRoleRequest>): GetInstanceRoleRequest {
    const message = createBaseGetInstanceRoleRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListInstanceRolesRequest(): ListInstanceRolesRequest {
  return { parent: "", pageSize: 0, pageToken: "" };
}

export const ListInstanceRolesRequest = {
  encode(message: ListInstanceRolesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): ListInstanceRolesRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListInstanceRolesRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.parent = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.pageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListInstanceRolesRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListInstanceRolesRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    return obj;
  },

  create(base?: DeepPartial<ListInstanceRolesRequest>): ListInstanceRolesRequest {
    return ListInstanceRolesRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListInstanceRolesRequest>): ListInstanceRolesRequest {
    const message = createBaseListInstanceRolesRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListInstanceRolesResponse(): ListInstanceRolesResponse {
  return { roles: [], nextPageToken: "" };
}

export const ListInstanceRolesResponse = {
  encode(message: ListInstanceRolesResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.roles) {
      InstanceRole.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListInstanceRolesResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListInstanceRolesResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.roles.push(InstanceRole.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListInstanceRolesResponse {
    return {
      roles: Array.isArray(object?.roles) ? object.roles.map((e: any) => InstanceRole.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListInstanceRolesResponse): unknown {
    const obj: any = {};
    if (message.roles) {
      obj.roles = message.roles.map((e) => e ? InstanceRole.toJSON(e) : undefined);
    } else {
      obj.roles = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  create(base?: DeepPartial<ListInstanceRolesResponse>): ListInstanceRolesResponse {
    return ListInstanceRolesResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListInstanceRolesResponse>): ListInstanceRolesResponse {
    const message = createBaseListInstanceRolesResponse();
    message.roles = object.roles?.map((e) => InstanceRole.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseCreateInstanceRoleRequest(): CreateInstanceRoleRequest {
  return { parent: "", role: undefined };
}

export const CreateInstanceRoleRequest = {
  encode(message: CreateInstanceRoleRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.role !== undefined) {
      InstanceRole.encode(message.role, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateInstanceRoleRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateInstanceRoleRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.parent = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.role = InstanceRole.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateInstanceRoleRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      role: isSet(object.role) ? InstanceRole.fromJSON(object.role) : undefined,
    };
  },

  toJSON(message: CreateInstanceRoleRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.role !== undefined && (obj.role = message.role ? InstanceRole.toJSON(message.role) : undefined);
    return obj;
  },

  create(base?: DeepPartial<CreateInstanceRoleRequest>): CreateInstanceRoleRequest {
    return CreateInstanceRoleRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<CreateInstanceRoleRequest>): CreateInstanceRoleRequest {
    const message = createBaseCreateInstanceRoleRequest();
    message.parent = object.parent ?? "";
    message.role = (object.role !== undefined && object.role !== null)
      ? InstanceRole.fromPartial(object.role)
      : undefined;
    return message;
  },
};

function createBaseUpdateInstanceRoleRequest(): UpdateInstanceRoleRequest {
  return { role: undefined, updateMask: undefined };
}

export const UpdateInstanceRoleRequest = {
  encode(message: UpdateInstanceRoleRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.role !== undefined) {
      InstanceRole.encode(message.role, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateInstanceRoleRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateInstanceRoleRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.role = InstanceRole.decode(reader, reader.uint32());
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

  fromJSON(object: any): UpdateInstanceRoleRequest {
    return {
      role: isSet(object.role) ? InstanceRole.fromJSON(object.role) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateInstanceRoleRequest): unknown {
    const obj: any = {};
    message.role !== undefined && (obj.role = message.role ? InstanceRole.toJSON(message.role) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  create(base?: DeepPartial<UpdateInstanceRoleRequest>): UpdateInstanceRoleRequest {
    return UpdateInstanceRoleRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<UpdateInstanceRoleRequest>): UpdateInstanceRoleRequest {
    const message = createBaseUpdateInstanceRoleRequest();
    message.role = (object.role !== undefined && object.role !== null)
      ? InstanceRole.fromPartial(object.role)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseDeleteInstanceRoleRequest(): DeleteInstanceRoleRequest {
  return { name: "" };
}

export const DeleteInstanceRoleRequest = {
  encode(message: DeleteInstanceRoleRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteInstanceRoleRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteInstanceRoleRequest();
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

  fromJSON(object: any): DeleteInstanceRoleRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: DeleteInstanceRoleRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<DeleteInstanceRoleRequest>): DeleteInstanceRoleRequest {
    return DeleteInstanceRoleRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<DeleteInstanceRoleRequest>): DeleteInstanceRoleRequest {
    const message = createBaseDeleteInstanceRoleRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseUndeleteInstanceRoleRequest(): UndeleteInstanceRoleRequest {
  return { name: "" };
}

export const UndeleteInstanceRoleRequest = {
  encode(message: UndeleteInstanceRoleRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UndeleteInstanceRoleRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUndeleteInstanceRoleRequest();
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

  fromJSON(object: any): UndeleteInstanceRoleRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: UndeleteInstanceRoleRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<UndeleteInstanceRoleRequest>): UndeleteInstanceRoleRequest {
    return UndeleteInstanceRoleRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<UndeleteInstanceRoleRequest>): UndeleteInstanceRoleRequest {
    const message = createBaseUndeleteInstanceRoleRequest();
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseInstanceRole();
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

          message.roleName = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.password = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.connectionLimit = reader.int32();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.validUntil = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.attribute = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<InstanceRole>): InstanceRole {
    return InstanceRole.fromPartial(base ?? {});
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
    getInstanceRole: {
      name: "GetInstanceRole",
      requestType: GetInstanceRoleRequest,
      requestStream: false,
      responseType: InstanceRole,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              47,
              18,
              45,
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
              47,
              114,
              111,
              108,
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
    listInstanceRoles: {
      name: "ListInstanceRoles",
      requestType: ListInstanceRolesRequest,
      requestStream: false,
      responseType: ListInstanceRolesResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([6, 112, 97, 114, 101, 110, 116])],
          578365826: [
            new Uint8Array([
              47,
              18,
              45,
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
              47,
              114,
              111,
              108,
              101,
              115,
            ]),
          ],
        },
      },
    },
    createInstanceRole: {
      name: "CreateInstanceRole",
      requestType: CreateInstanceRoleRequest,
      requestStream: false,
      responseType: InstanceRole,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([11, 112, 97, 114, 101, 110, 116, 44, 114, 111, 108, 101])],
          578365826: [
            new Uint8Array([
              53,
              58,
              4,
              114,
              111,
              108,
              101,
              34,
              45,
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
              47,
              114,
              111,
              108,
              101,
              115,
            ]),
          ],
        },
      },
    },
    updateInstanceRole: {
      name: "UpdateInstanceRole",
      requestType: UpdateInstanceRoleRequest,
      requestStream: false,
      responseType: InstanceRole,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([16, 114, 111, 108, 101, 44, 117, 112, 100, 97, 116, 101, 95, 109, 97, 115, 107])],
          578365826: [
            new Uint8Array([
              58,
              58,
              4,
              114,
              111,
              108,
              101,
              50,
              50,
              47,
              118,
              49,
              47,
              123,
              114,
              111,
              108,
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
              47,
              114,
              111,
              108,
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
    deleteInstanceRole: {
      name: "DeleteInstanceRole",
      requestType: DeleteInstanceRoleRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              47,
              42,
              45,
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
              47,
              114,
              111,
              108,
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
    undeleteInstanceRole: {
      name: "UndeleteInstanceRole",
      requestType: UndeleteInstanceRoleRequest,
      requestStream: false,
      responseType: InstanceRole,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              59,
              58,
              1,
              42,
              34,
              54,
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
              47,
              114,
              111,
              108,
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
  },
} as const;

export interface InstanceRoleServiceImplementation<CallContextExt = {}> {
  getInstanceRole(
    request: GetInstanceRoleRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<InstanceRole>>;
  listInstanceRoles(
    request: ListInstanceRolesRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ListInstanceRolesResponse>>;
  createInstanceRole(
    request: CreateInstanceRoleRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<InstanceRole>>;
  updateInstanceRole(
    request: UpdateInstanceRoleRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<InstanceRole>>;
  deleteInstanceRole(
    request: DeleteInstanceRoleRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Empty>>;
  undeleteInstanceRole(
    request: UndeleteInstanceRoleRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<InstanceRole>>;
}

export interface InstanceRoleServiceClient<CallOptionsExt = {}> {
  getInstanceRole(
    request: DeepPartial<GetInstanceRoleRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<InstanceRole>;
  listInstanceRoles(
    request: DeepPartial<ListInstanceRolesRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ListInstanceRolesResponse>;
  createInstanceRole(
    request: DeepPartial<CreateInstanceRoleRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<InstanceRole>;
  updateInstanceRole(
    request: DeepPartial<UpdateInstanceRoleRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<InstanceRole>;
  deleteInstanceRole(
    request: DeepPartial<DeleteInstanceRoleRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Empty>;
  undeleteInstanceRole(
    request: DeepPartial<UndeleteInstanceRoleRequest>,
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
