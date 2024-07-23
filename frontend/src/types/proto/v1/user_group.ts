/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { Timestamp } from "../google/protobuf/timestamp";

export const protobufPackage = "bytebase.v1";

export interface GetUserGroupRequest {
  /**
   * The name of the group to retrieve.
   * Format: groups/{email}
   */
  name: string;
}

export interface ListUserGroupsRequest {
  /**
   * The maximum number of groups to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 groups will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListUsers` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListUsers` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface ListUserGroupsResponse {
  /** The groups from the specified request. */
  groups: UserGroup[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface CreateUserGroupRequest {
  /** The group to create. */
  group: UserGroup | undefined;
}

export interface UpdateUserGroupRequest {
  /**
   * The group to update.
   *
   * The group's `name` field is used to identify the group to update.
   * Format: groups/{email}
   */
  group:
    | UserGroup
    | undefined;
  /** The list of fields to update. */
  updateMask: string[] | undefined;
}

export interface DeleteUserGroupRequest {
  /**
   * The name of the group to delete.
   * Format: groups/{email}
   */
  name: string;
}

export interface UserGroupMember {
  /**
   * Member is the principal who belong to this user group.
   *
   * Format: users/hello@world.com
   */
  member: string;
  role: UserGroupMember_Role;
}

export enum UserGroupMember_Role {
  ROLE_UNSPECIFIED = "ROLE_UNSPECIFIED",
  OWNER = "OWNER",
  MEMBER = "MEMBER",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function userGroupMember_RoleFromJSON(object: any): UserGroupMember_Role {
  switch (object) {
    case 0:
    case "ROLE_UNSPECIFIED":
      return UserGroupMember_Role.ROLE_UNSPECIFIED;
    case 1:
    case "OWNER":
      return UserGroupMember_Role.OWNER;
    case 2:
    case "MEMBER":
      return UserGroupMember_Role.MEMBER;
    case -1:
    case "UNRECOGNIZED":
    default:
      return UserGroupMember_Role.UNRECOGNIZED;
  }
}

export function userGroupMember_RoleToJSON(object: UserGroupMember_Role): string {
  switch (object) {
    case UserGroupMember_Role.ROLE_UNSPECIFIED:
      return "ROLE_UNSPECIFIED";
    case UserGroupMember_Role.OWNER:
      return "OWNER";
    case UserGroupMember_Role.MEMBER:
      return "MEMBER";
    case UserGroupMember_Role.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function userGroupMember_RoleToNumber(object: UserGroupMember_Role): number {
  switch (object) {
    case UserGroupMember_Role.ROLE_UNSPECIFIED:
      return 0;
    case UserGroupMember_Role.OWNER:
      return 1;
    case UserGroupMember_Role.MEMBER:
      return 2;
    case UserGroupMember_Role.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface UserGroup {
  /**
   * The name of the group to retrieve.
   * Format: groups/{email}
   */
  name: string;
  title: string;
  description: string;
  /**
   * The name for the creator.
   * Format: users/hello@world.com
   */
  creator: string;
  members: UserGroupMember[];
  /** The timestamp when the group was created. */
  createTime: Date | undefined;
}

function createBaseGetUserGroupRequest(): GetUserGroupRequest {
  return { name: "" };
}

export const GetUserGroupRequest = {
  encode(message: GetUserGroupRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetUserGroupRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetUserGroupRequest();
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

  fromJSON(object: any): GetUserGroupRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: GetUserGroupRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<GetUserGroupRequest>): GetUserGroupRequest {
    return GetUserGroupRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<GetUserGroupRequest>): GetUserGroupRequest {
    const message = createBaseGetUserGroupRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListUserGroupsRequest(): ListUserGroupsRequest {
  return { pageSize: 0, pageToken: "" };
}

export const ListUserGroupsRequest = {
  encode(message: ListUserGroupsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pageSize !== 0) {
      writer.uint32(8).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(18).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListUserGroupsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListUserGroupsRequest();
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

  fromJSON(object: any): ListUserGroupsRequest {
    return {
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
    };
  },

  toJSON(message: ListUserGroupsRequest): unknown {
    const obj: any = {};
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListUserGroupsRequest>): ListUserGroupsRequest {
    return ListUserGroupsRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListUserGroupsRequest>): ListUserGroupsRequest {
    const message = createBaseListUserGroupsRequest();
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListUserGroupsResponse(): ListUserGroupsResponse {
  return { groups: [], nextPageToken: "" };
}

export const ListUserGroupsResponse = {
  encode(message: ListUserGroupsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.groups) {
      UserGroup.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListUserGroupsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListUserGroupsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.groups.push(UserGroup.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListUserGroupsResponse {
    return {
      groups: globalThis.Array.isArray(object?.groups) ? object.groups.map((e: any) => UserGroup.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListUserGroupsResponse): unknown {
    const obj: any = {};
    if (message.groups?.length) {
      obj.groups = message.groups.map((e) => UserGroup.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListUserGroupsResponse>): ListUserGroupsResponse {
    return ListUserGroupsResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListUserGroupsResponse>): ListUserGroupsResponse {
    const message = createBaseListUserGroupsResponse();
    message.groups = object.groups?.map((e) => UserGroup.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseCreateUserGroupRequest(): CreateUserGroupRequest {
  return { group: undefined };
}

export const CreateUserGroupRequest = {
  encode(message: CreateUserGroupRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.group !== undefined) {
      UserGroup.encode(message.group, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateUserGroupRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateUserGroupRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.group = UserGroup.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateUserGroupRequest {
    return { group: isSet(object.group) ? UserGroup.fromJSON(object.group) : undefined };
  },

  toJSON(message: CreateUserGroupRequest): unknown {
    const obj: any = {};
    if (message.group !== undefined) {
      obj.group = UserGroup.toJSON(message.group);
    }
    return obj;
  },

  create(base?: DeepPartial<CreateUserGroupRequest>): CreateUserGroupRequest {
    return CreateUserGroupRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<CreateUserGroupRequest>): CreateUserGroupRequest {
    const message = createBaseCreateUserGroupRequest();
    message.group = (object.group !== undefined && object.group !== null)
      ? UserGroup.fromPartial(object.group)
      : undefined;
    return message;
  },
};

function createBaseUpdateUserGroupRequest(): UpdateUserGroupRequest {
  return { group: undefined, updateMask: undefined };
}

export const UpdateUserGroupRequest = {
  encode(message: UpdateUserGroupRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.group !== undefined) {
      UserGroup.encode(message.group, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateUserGroupRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateUserGroupRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.group = UserGroup.decode(reader, reader.uint32());
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

  fromJSON(object: any): UpdateUserGroupRequest {
    return {
      group: isSet(object.group) ? UserGroup.fromJSON(object.group) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateUserGroupRequest): unknown {
    const obj: any = {};
    if (message.group !== undefined) {
      obj.group = UserGroup.toJSON(message.group);
    }
    if (message.updateMask !== undefined) {
      obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask));
    }
    return obj;
  },

  create(base?: DeepPartial<UpdateUserGroupRequest>): UpdateUserGroupRequest {
    return UpdateUserGroupRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UpdateUserGroupRequest>): UpdateUserGroupRequest {
    const message = createBaseUpdateUserGroupRequest();
    message.group = (object.group !== undefined && object.group !== null)
      ? UserGroup.fromPartial(object.group)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseDeleteUserGroupRequest(): DeleteUserGroupRequest {
  return { name: "" };
}

export const DeleteUserGroupRequest = {
  encode(message: DeleteUserGroupRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteUserGroupRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteUserGroupRequest();
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

  fromJSON(object: any): DeleteUserGroupRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: DeleteUserGroupRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<DeleteUserGroupRequest>): DeleteUserGroupRequest {
    return DeleteUserGroupRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DeleteUserGroupRequest>): DeleteUserGroupRequest {
    const message = createBaseDeleteUserGroupRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseUserGroupMember(): UserGroupMember {
  return { member: "", role: UserGroupMember_Role.ROLE_UNSPECIFIED };
}

export const UserGroupMember = {
  encode(message: UserGroupMember, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.member !== "") {
      writer.uint32(10).string(message.member);
    }
    if (message.role !== UserGroupMember_Role.ROLE_UNSPECIFIED) {
      writer.uint32(16).int32(userGroupMember_RoleToNumber(message.role));
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UserGroupMember {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUserGroupMember();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.member = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.role = userGroupMember_RoleFromJSON(reader.int32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UserGroupMember {
    return {
      member: isSet(object.member) ? globalThis.String(object.member) : "",
      role: isSet(object.role) ? userGroupMember_RoleFromJSON(object.role) : UserGroupMember_Role.ROLE_UNSPECIFIED,
    };
  },

  toJSON(message: UserGroupMember): unknown {
    const obj: any = {};
    if (message.member !== "") {
      obj.member = message.member;
    }
    if (message.role !== UserGroupMember_Role.ROLE_UNSPECIFIED) {
      obj.role = userGroupMember_RoleToJSON(message.role);
    }
    return obj;
  },

  create(base?: DeepPartial<UserGroupMember>): UserGroupMember {
    return UserGroupMember.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UserGroupMember>): UserGroupMember {
    const message = createBaseUserGroupMember();
    message.member = object.member ?? "";
    message.role = object.role ?? UserGroupMember_Role.ROLE_UNSPECIFIED;
    return message;
  },
};

function createBaseUserGroup(): UserGroup {
  return { name: "", title: "", description: "", creator: "", members: [], createTime: undefined };
}

export const UserGroup = {
  encode(message: UserGroup, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    if (message.creator !== "") {
      writer.uint32(34).string(message.creator);
    }
    for (const v of message.members) {
      UserGroupMember.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    if (message.createTime !== undefined) {
      Timestamp.encode(toTimestamp(message.createTime), writer.uint32(50).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UserGroup {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUserGroup();
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

          message.creator = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.members.push(UserGroupMember.decode(reader, reader.uint32()));
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.createTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UserGroup {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      description: isSet(object.description) ? globalThis.String(object.description) : "",
      creator: isSet(object.creator) ? globalThis.String(object.creator) : "",
      members: globalThis.Array.isArray(object?.members)
        ? object.members.map((e: any) => UserGroupMember.fromJSON(e))
        : [],
      createTime: isSet(object.createTime) ? fromJsonTimestamp(object.createTime) : undefined,
    };
  },

  toJSON(message: UserGroup): unknown {
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
    if (message.creator !== "") {
      obj.creator = message.creator;
    }
    if (message.members?.length) {
      obj.members = message.members.map((e) => UserGroupMember.toJSON(e));
    }
    if (message.createTime !== undefined) {
      obj.createTime = message.createTime.toISOString();
    }
    return obj;
  },

  create(base?: DeepPartial<UserGroup>): UserGroup {
    return UserGroup.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UserGroup>): UserGroup {
    const message = createBaseUserGroup();
    message.name = object.name ?? "";
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.creator = object.creator ?? "";
    message.members = object.members?.map((e) => UserGroupMember.fromPartial(e)) || [];
    message.createTime = object.createTime ?? undefined;
    return message;
  },
};

export type UserGroupServiceDefinition = typeof UserGroupServiceDefinition;
export const UserGroupServiceDefinition = {
  name: "UserGroupService",
  fullName: "bytebase.v1.UserGroupService",
  methods: {
    getUserGroup: {
      name: "GetUserGroup",
      requestType: GetUserGroupRequest,
      requestStream: false,
      responseType: UserGroup,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          800010: [
            new Uint8Array([17, 98, 98, 46, 117, 115, 101, 114, 71, 114, 111, 117, 112, 115, 46, 103, 101, 116]),
          ],
          578365826: [
            new Uint8Array([
              21,
              18,
              19,
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
              103,
              114,
              111,
              117,
              112,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    listUserGroups: {
      name: "ListUserGroups",
      requestType: ListUserGroupsRequest,
      requestStream: false,
      responseType: ListUserGroupsResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          800010: [
            new Uint8Array([18, 98, 98, 46, 117, 115, 101, 114, 71, 114, 111, 117, 112, 115, 46, 108, 105, 115, 116]),
          ],
          578365826: [new Uint8Array([12, 18, 10, 47, 118, 49, 47, 103, 114, 111, 117, 112, 115])],
        },
      },
    },
    createUserGroup: {
      name: "CreateUserGroup",
      requestType: CreateUserGroupRequest,
      requestStream: false,
      responseType: UserGroup,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([5, 103, 114, 111, 117, 112])],
          800010: [
            new Uint8Array([
              20,
              98,
              98,
              46,
              117,
              115,
              101,
              114,
              71,
              114,
              111,
              117,
              112,
              115,
              46,
              99,
              114,
              101,
              97,
              116,
              101,
            ]),
          ],
          578365826: [
            new Uint8Array([19, 58, 5, 103, 114, 111, 117, 112, 34, 10, 47, 118, 49, 47, 103, 114, 111, 117, 112, 115]),
          ],
        },
      },
    },
    updateUserGroup: {
      name: "UpdateUserGroup",
      requestType: UpdateUserGroupRequest,
      requestStream: false,
      responseType: UserGroup,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([17, 103, 114, 111, 117, 112, 44, 117, 112, 100, 97, 116, 101, 95, 109, 97, 115, 107])],
          800010: [
            new Uint8Array([
              20,
              98,
              98,
              46,
              117,
              115,
              101,
              114,
              71,
              114,
              111,
              117,
              112,
              115,
              46,
              117,
              112,
              100,
              97,
              116,
              101,
            ]),
          ],
          800016: [new Uint8Array([2])],
          578365826: [
            new Uint8Array([
              34,
              58,
              5,
              103,
              114,
              111,
              117,
              112,
              50,
              25,
              47,
              118,
              49,
              47,
              123,
              103,
              114,
              111,
              117,
              112,
              46,
              110,
              97,
              109,
              101,
              61,
              103,
              114,
              111,
              117,
              112,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    deleteUserGroup: {
      name: "DeleteUserGroup",
      requestType: DeleteUserGroupRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          800010: [
            new Uint8Array([
              20,
              98,
              98,
              46,
              117,
              115,
              101,
              114,
              71,
              114,
              111,
              117,
              112,
              115,
              46,
              100,
              101,
              108,
              101,
              116,
              101,
            ]),
          ],
          800016: [new Uint8Array([2])],
          578365826: [
            new Uint8Array([
              21,
              42,
              19,
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
              103,
              114,
              111,
              117,
              112,
              115,
              47,
              42,
              125,
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

function toTimestamp(date: Date): Timestamp {
  const seconds = numberToLong(date.getTime() / 1_000);
  const nanos = (date.getTime() % 1_000) * 1_000_000;
  return { seconds, nanos };
}

function fromTimestamp(t: Timestamp): Date {
  let millis = (t.seconds.toNumber() || 0) * 1_000;
  millis += (t.nanos || 0) / 1_000_000;
  return new globalThis.Date(millis);
}

function fromJsonTimestamp(o: any): Date {
  if (o instanceof globalThis.Date) {
    return o;
  } else if (typeof o === "string") {
    return new globalThis.Date(o);
  } else {
    return fromTimestamp(Timestamp.fromJSON(o));
  }
}

function numberToLong(number: number) {
  return Long.fromNumber(number);
}

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
