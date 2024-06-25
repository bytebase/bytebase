/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export interface UserGroupMember {
  /**
   * Member is the principal who belong to this user group.
   *
   * Format: users/{userUID}.
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

export interface UserGroupPayload {
  members: UserGroupMember[];
}

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

function createBaseUserGroupPayload(): UserGroupPayload {
  return { members: [] };
}

export const UserGroupPayload = {
  encode(message: UserGroupPayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.members) {
      UserGroupMember.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UserGroupPayload {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUserGroupPayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.members.push(UserGroupMember.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UserGroupPayload {
    return {
      members: globalThis.Array.isArray(object?.members)
        ? object.members.map((e: any) => UserGroupMember.fromJSON(e))
        : [],
    };
  },

  toJSON(message: UserGroupPayload): unknown {
    const obj: any = {};
    if (message.members?.length) {
      obj.members = message.members.map((e) => UserGroupMember.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<UserGroupPayload>): UserGroupPayload {
    return UserGroupPayload.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UserGroupPayload>): UserGroupPayload {
    const message = createBaseUserGroupPayload();
    message.members = object.members?.map((e) => UserGroupMember.fromPartial(e)) || [];
    return message;
  },
};

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
