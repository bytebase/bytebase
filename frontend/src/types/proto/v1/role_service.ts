/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";

export const protobufPackage = "bytebase.v1";

export interface ListRolesRequest {
  /**
   * The maximum number of roles to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 reviews will be returned.
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
  roles: Role[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface CreateRoleRequest {
  role:
    | Role
    | undefined;
  /**
   * The ID to use for the role, which will become the final component
   * of the role's resource name.
   *
   * This value should be 4-63 characters, and valid characters
   * are /[a-z][A-Z][0-9]/.
   */
  roleId: string;
}

export interface UpdateRoleRequest {
  role: Role | undefined;
  updateMask: string[] | undefined;
}

export interface DeleteRoleRequest {
  /** Format: roles/{role} */
  name: string;
}

export interface ListPermissionsRequest {
}

export interface ListPermissionsResponse {
  permissions: Permission[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface Role {
  /** Format: roles/{role} */
  name: string;
  title: string;
  description: string;
  /** Permissions that are granted to this role. */
  permissions: string[];
}

export interface Permission {
  name: string;
  /** The resource under which this permission applies. */
  resource: Permission_Resource;
}

export enum Permission_Resource {
  RESOURCE_UNSPECIFIED = 0,
  WORKSPACE = 1,
  PROJECT = 2,
  UNRECOGNIZED = -1,
}

export function permission_ResourceFromJSON(object: any): Permission_Resource {
  switch (object) {
    case 0:
    case "RESOURCE_UNSPECIFIED":
      return Permission_Resource.RESOURCE_UNSPECIFIED;
    case 1:
    case "WORKSPACE":
      return Permission_Resource.WORKSPACE;
    case 2:
    case "PROJECT":
      return Permission_Resource.PROJECT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Permission_Resource.UNRECOGNIZED;
  }
}

export function permission_ResourceToJSON(object: Permission_Resource): string {
  switch (object) {
    case Permission_Resource.RESOURCE_UNSPECIFIED:
      return "RESOURCE_UNSPECIFIED";
    case Permission_Resource.WORKSPACE:
      return "WORKSPACE";
    case Permission_Resource.PROJECT:
      return "PROJECT";
    case Permission_Resource.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

function createBaseListRolesRequest(): ListRolesRequest {
  return { pageSize: 0, pageToken: "" };
}

export const ListRolesRequest = {
  encode(message: ListRolesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pageSize !== 0) {
      writer.uint32(8).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(18).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListRolesRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListRolesRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
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
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListRolesRequest {
    return {
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
    };
  },

  toJSON(message: ListRolesRequest): unknown {
    const obj: any = {};
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListRolesRequest>): ListRolesRequest {
    return ListRolesRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListRolesRequest>): ListRolesRequest {
    const message = createBaseListRolesRequest();
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
      Role.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListRolesResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListRolesResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.roles.push(Role.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListRolesResponse {
    return {
      roles: globalThis.Array.isArray(object?.roles) ? object.roles.map((e: any) => Role.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListRolesResponse): unknown {
    const obj: any = {};
    if (message.roles?.length) {
      obj.roles = message.roles.map((e) => Role.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListRolesResponse>): ListRolesResponse {
    return ListRolesResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListRolesResponse>): ListRolesResponse {
    const message = createBaseListRolesResponse();
    message.roles = object.roles?.map((e) => Role.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseCreateRoleRequest(): CreateRoleRequest {
  return { role: undefined, roleId: "" };
}

export const CreateRoleRequest = {
  encode(message: CreateRoleRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.role !== undefined) {
      Role.encode(message.role, writer.uint32(10).fork()).ldelim();
    }
    if (message.roleId !== "") {
      writer.uint32(18).string(message.roleId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateRoleRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateRoleRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.role = Role.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.roleId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateRoleRequest {
    return {
      role: isSet(object.role) ? Role.fromJSON(object.role) : undefined,
      roleId: isSet(object.roleId) ? globalThis.String(object.roleId) : "",
    };
  },

  toJSON(message: CreateRoleRequest): unknown {
    const obj: any = {};
    if (message.role !== undefined) {
      obj.role = Role.toJSON(message.role);
    }
    if (message.roleId !== "") {
      obj.roleId = message.roleId;
    }
    return obj;
  },

  create(base?: DeepPartial<CreateRoleRequest>): CreateRoleRequest {
    return CreateRoleRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<CreateRoleRequest>): CreateRoleRequest {
    const message = createBaseCreateRoleRequest();
    message.role = (object.role !== undefined && object.role !== null) ? Role.fromPartial(object.role) : undefined;
    message.roleId = object.roleId ?? "";
    return message;
  },
};

function createBaseUpdateRoleRequest(): UpdateRoleRequest {
  return { role: undefined, updateMask: undefined };
}

export const UpdateRoleRequest = {
  encode(message: UpdateRoleRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.role !== undefined) {
      Role.encode(message.role, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateRoleRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateRoleRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.role = Role.decode(reader, reader.uint32());
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

  fromJSON(object: any): UpdateRoleRequest {
    return {
      role: isSet(object.role) ? Role.fromJSON(object.role) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateRoleRequest): unknown {
    const obj: any = {};
    if (message.role !== undefined) {
      obj.role = Role.toJSON(message.role);
    }
    if (message.updateMask !== undefined) {
      obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask));
    }
    return obj;
  },

  create(base?: DeepPartial<UpdateRoleRequest>): UpdateRoleRequest {
    return UpdateRoleRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UpdateRoleRequest>): UpdateRoleRequest {
    const message = createBaseUpdateRoleRequest();
    message.role = (object.role !== undefined && object.role !== null) ? Role.fromPartial(object.role) : undefined;
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteRoleRequest();
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

  fromJSON(object: any): DeleteRoleRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: DeleteRoleRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<DeleteRoleRequest>): DeleteRoleRequest {
    return DeleteRoleRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DeleteRoleRequest>): DeleteRoleRequest {
    const message = createBaseDeleteRoleRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListPermissionsRequest(): ListPermissionsRequest {
  return {};
}

export const ListPermissionsRequest = {
  encode(_: ListPermissionsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListPermissionsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListPermissionsRequest();
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

  fromJSON(_: any): ListPermissionsRequest {
    return {};
  },

  toJSON(_: ListPermissionsRequest): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<ListPermissionsRequest>): ListPermissionsRequest {
    return ListPermissionsRequest.fromPartial(base ?? {});
  },
  fromPartial(_: DeepPartial<ListPermissionsRequest>): ListPermissionsRequest {
    const message = createBaseListPermissionsRequest();
    return message;
  },
};

function createBaseListPermissionsResponse(): ListPermissionsResponse {
  return { permissions: [], nextPageToken: "" };
}

export const ListPermissionsResponse = {
  encode(message: ListPermissionsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.permissions) {
      Permission.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListPermissionsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListPermissionsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.permissions.push(Permission.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListPermissionsResponse {
    return {
      permissions: globalThis.Array.isArray(object?.permissions)
        ? object.permissions.map((e: any) => Permission.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListPermissionsResponse): unknown {
    const obj: any = {};
    if (message.permissions?.length) {
      obj.permissions = message.permissions.map((e) => Permission.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListPermissionsResponse>): ListPermissionsResponse {
    return ListPermissionsResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListPermissionsResponse>): ListPermissionsResponse {
    const message = createBaseListPermissionsResponse();
    message.permissions = object.permissions?.map((e) => Permission.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseRole(): Role {
  return { name: "", title: "", description: "", permissions: [] };
}

export const Role = {
  encode(message: Role, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    for (const v of message.permissions) {
      writer.uint32(34).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Role {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRole();
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

          message.title = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.description = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.permissions.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Role {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      description: isSet(object.description) ? globalThis.String(object.description) : "",
      permissions: globalThis.Array.isArray(object?.permissions)
        ? object.permissions.map((e: any) => globalThis.String(e))
        : [],
    };
  },

  toJSON(message: Role): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.description !== "") {
      obj.description = message.description;
    }
    if (message.permissions?.length) {
      obj.permissions = message.permissions;
    }
    return obj;
  },

  create(base?: DeepPartial<Role>): Role {
    return Role.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Role>): Role {
    const message = createBaseRole();
    message.name = object.name ?? "";
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.permissions = object.permissions?.map((e) => e) || [];
    return message;
  },
};

function createBasePermission(): Permission {
  return { name: "", resource: 0 };
}

export const Permission = {
  encode(message: Permission, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.resource !== 0) {
      writer.uint32(16).int32(message.resource);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Permission {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePermission();
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

          message.resource = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Permission {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      resource: isSet(object.resource) ? permission_ResourceFromJSON(object.resource) : 0,
    };
  },

  toJSON(message: Permission): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.resource !== 0) {
      obj.resource = permission_ResourceToJSON(message.resource);
    }
    return obj;
  },

  create(base?: DeepPartial<Permission>): Permission {
    return Permission.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Permission>): Permission {
    const message = createBasePermission();
    message.name = object.name ?? "";
    message.resource = object.resource ?? 0;
    return message;
  },
};

export type RoleServiceDefinition = typeof RoleServiceDefinition;
export const RoleServiceDefinition = {
  name: "RoleService",
  fullName: "bytebase.v1.RoleService",
  methods: {
    listRoles: {
      name: "ListRoles",
      requestType: ListRolesRequest,
      requestStream: false,
      responseType: ListRolesResponse,
      responseStream: false,
      options: {
        _unknownFields: { 578365826: [new Uint8Array([11, 18, 9, 47, 118, 49, 47, 114, 111, 108, 101, 115])] },
      },
    },
    createRole: {
      name: "CreateRole",
      requestType: CreateRoleRequest,
      requestStream: false,
      responseType: Role,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [new Uint8Array([17, 58, 4, 114, 111, 108, 101, 34, 9, 47, 118, 49, 47, 114, 111, 108, 101, 115])],
        },
      },
    },
    updateRole: {
      name: "UpdateRole",
      requestType: UpdateRoleRequest,
      requestStream: false,
      responseType: Role,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([16, 114, 111, 108, 101, 44, 117, 112, 100, 97, 116, 101, 95, 109, 97, 115, 107])],
          578365826: [
            new Uint8Array([
              31,
              58,
              4,
              114,
              111,
              108,
              101,
              50,
              23,
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
    deleteRole: {
      name: "DeleteRole",
      requestType: DeleteRoleRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              20,
              42,
              18,
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
    listPermissions: {
      name: "ListPermissions",
      requestType: ListPermissionsRequest,
      requestStream: false,
      responseType: ListPermissionsResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([17, 18, 15, 47, 118, 49, 47, 112, 101, 114, 109, 105, 115, 115, 105, 111, 110, 115]),
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
