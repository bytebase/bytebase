/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { State, stateFromJSON, stateToJSON } from "./common";

export const protobufPackage = "bytebase.v1";

export enum UserType {
  USER_TYPE_UNSPECIFIED = 0,
  USER = 1,
  SYSTEM_BOT = 2,
  SERVICE_ACCOUNT = 3,
  UNRECOGNIZED = -1,
}

export function userTypeFromJSON(object: any): UserType {
  switch (object) {
    case 0:
    case "USER_TYPE_UNSPECIFIED":
      return UserType.USER_TYPE_UNSPECIFIED;
    case 1:
    case "USER":
      return UserType.USER;
    case 2:
    case "SYSTEM_BOT":
      return UserType.SYSTEM_BOT;
    case 3:
    case "SERVICE_ACCOUNT":
      return UserType.SERVICE_ACCOUNT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return UserType.UNRECOGNIZED;
  }
}

export function userTypeToJSON(object: UserType): string {
  switch (object) {
    case UserType.USER_TYPE_UNSPECIFIED:
      return "USER_TYPE_UNSPECIFIED";
    case UserType.USER:
      return "USER";
    case UserType.SYSTEM_BOT:
      return "SYSTEM_BOT";
    case UserType.SERVICE_ACCOUNT:
      return "SERVICE_ACCOUNT";
    case UserType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum UserRole {
  USER_ROLE_UNSPECIFIED = 0,
  OWNER = 1,
  DBA = 2,
  DEVELOPER = 3,
  UNRECOGNIZED = -1,
}

export function userRoleFromJSON(object: any): UserRole {
  switch (object) {
    case 0:
    case "USER_ROLE_UNSPECIFIED":
      return UserRole.USER_ROLE_UNSPECIFIED;
    case 1:
    case "OWNER":
      return UserRole.OWNER;
    case 2:
    case "DBA":
      return UserRole.DBA;
    case 3:
    case "DEVELOPER":
      return UserRole.DEVELOPER;
    case -1:
    case "UNRECOGNIZED":
    default:
      return UserRole.UNRECOGNIZED;
  }
}

export function userRoleToJSON(object: UserRole): string {
  switch (object) {
    case UserRole.USER_ROLE_UNSPECIFIED:
      return "USER_ROLE_UNSPECIFIED";
    case UserRole.OWNER:
      return "OWNER";
    case UserRole.DBA:
      return "DBA";
    case UserRole.DEVELOPER:
      return "DEVELOPER";
    case UserRole.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface GetUserRequest {
  /**
   * The name of the user to retrieve.
   * Format: users/{user}
   */
  name: string;
}

export interface ListUsersRequest {
  /**
   * The maximum number of users to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 users will be returned.
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
  /** Show deleted users if specified. */
  showDeleted: boolean;
}

export interface ListUsersResponse {
  /** The users from the specified request. */
  users: User[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface CreateUserRequest {
  /** The user to create. */
  user?: User;
}

export interface UpdateUserRequest {
  /**
   * The user to update.
   *
   * The user's `name` field is used to identify the user to update.
   * Format: users/{user}
   */
  user?: User;
  /** The list of fields to update. */
  updateMask?: string[];
  /** The otp_code is used to verify the user's identity by MFA. */
  otpCode?:
    | string
    | undefined;
  /**
   * The regenerate_temp_mfa_secret flag means to regenerate tempary MFA secret for user.
   * This is used for MFA setup. The tempary MFA secret and recovery codes will be returned in the response.
   */
  regenerateTempMfaSecret: boolean;
  /** The regenerate_recovery_codes flag means to regenerate recovery codes for user. */
  regenerateRecoveryCodes: boolean;
}

export interface DeleteUserRequest {
  /**
   * The name of the user to delete.
   * Format: users/{user}
   */
  name: string;
}

export interface UndeleteUserRequest {
  /**
   * The name of the deleted user.
   * Format: users/{user}
   */
  name: string;
}

export interface LoginRequest {
  email: string;
  password: string;
  /** If web is set, we will set access token, refresh token, and user to the cookie. */
  web: boolean;
  /**
   * The name of the identity provider.
   * Format: idps/{idp}
   */
  idpName: string;
  /** The idp_context is using to get the user information from identity provider. */
  idpContext?: IdentityProviderContext;
  /** The otp_code is used to verify the user's identity by MFA. */
  otpCode?:
    | string
    | undefined;
  /** The recovery_code is used to recovery the user's identity with MFA. */
  recoveryCode?:
    | string
    | undefined;
  /** The mfa_temp_token is used to verify the user's identity by MFA. */
  mfaTempToken?: string | undefined;
}

export interface IdentityProviderContext {
  oauth2Context?: OAuth2IdentityProviderContext | undefined;
  oidcContext?: OIDCIdentityProviderContext | undefined;
}

export interface OAuth2IdentityProviderContext {
  code: string;
}

export interface OIDCIdentityProviderContext {
}

export interface LoginResponse {
  token: string;
  mfaTempToken?: string | undefined;
}

export interface LogoutRequest {
}

export interface User {
  /**
   * The name of the user.
   * Format: users/{user}. {user} is a system-generated unique ID.
   */
  name: string;
  state: State;
  email: string;
  title: string;
  userType: UserType;
  /** The user role will not be respected in the create user request, because the role is controlled by workspace owner. */
  userRole: UserRole;
  password: string;
  serviceKey: string;
  /** The mfa_enabled flag means if the user has enabled MFA. */
  mfaEnabled: boolean;
  /** The mfa_secret is the tempary secret using in two phase verification. */
  mfaSecret: string;
  /** The recovery_codes is the tempary recovery codes using in two phase verification. */
  recoveryCodes: string[];
}

function createBaseGetUserRequest(): GetUserRequest {
  return { name: "" };
}

export const GetUserRequest = {
  encode(message: GetUserRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetUserRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetUserRequest();
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

  fromJSON(object: any): GetUserRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetUserRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<GetUserRequest>): GetUserRequest {
    const message = createBaseGetUserRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListUsersRequest(): ListUsersRequest {
  return { pageSize: 0, pageToken: "", showDeleted: false };
}

export const ListUsersRequest = {
  encode(message: ListUsersRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): ListUsersRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListUsersRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pageSize = reader.int32();
          break;
        case 2:
          message.pageToken = reader.string();
          break;
        case 3:
          message.showDeleted = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ListUsersRequest {
    return {
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
      showDeleted: isSet(object.showDeleted) ? Boolean(object.showDeleted) : false,
    };
  },

  toJSON(message: ListUsersRequest): unknown {
    const obj: any = {};
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    message.showDeleted !== undefined && (obj.showDeleted = message.showDeleted);
    return obj;
  },

  fromPartial(object: DeepPartial<ListUsersRequest>): ListUsersRequest {
    const message = createBaseListUsersRequest();
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    message.showDeleted = object.showDeleted ?? false;
    return message;
  },
};

function createBaseListUsersResponse(): ListUsersResponse {
  return { users: [], nextPageToken: "" };
}

export const ListUsersResponse = {
  encode(message: ListUsersResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.users) {
      User.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListUsersResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListUsersResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.users.push(User.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListUsersResponse {
    return {
      users: Array.isArray(object?.users) ? object.users.map((e: any) => User.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListUsersResponse): unknown {
    const obj: any = {};
    if (message.users) {
      obj.users = message.users.map((e) => e ? User.toJSON(e) : undefined);
    } else {
      obj.users = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  fromPartial(object: DeepPartial<ListUsersResponse>): ListUsersResponse {
    const message = createBaseListUsersResponse();
    message.users = object.users?.map((e) => User.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseCreateUserRequest(): CreateUserRequest {
  return { user: undefined };
}

export const CreateUserRequest = {
  encode(message: CreateUserRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.user !== undefined) {
      User.encode(message.user, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateUserRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateUserRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.user = User.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): CreateUserRequest {
    return { user: isSet(object.user) ? User.fromJSON(object.user) : undefined };
  },

  toJSON(message: CreateUserRequest): unknown {
    const obj: any = {};
    message.user !== undefined && (obj.user = message.user ? User.toJSON(message.user) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<CreateUserRequest>): CreateUserRequest {
    const message = createBaseCreateUserRequest();
    message.user = (object.user !== undefined && object.user !== null) ? User.fromPartial(object.user) : undefined;
    return message;
  },
};

function createBaseUpdateUserRequest(): UpdateUserRequest {
  return {
    user: undefined,
    updateMask: undefined,
    otpCode: undefined,
    regenerateTempMfaSecret: false,
    regenerateRecoveryCodes: false,
  };
}

export const UpdateUserRequest = {
  encode(message: UpdateUserRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.user !== undefined) {
      User.encode(message.user, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    if (message.otpCode !== undefined) {
      writer.uint32(26).string(message.otpCode);
    }
    if (message.regenerateTempMfaSecret === true) {
      writer.uint32(32).bool(message.regenerateTempMfaSecret);
    }
    if (message.regenerateRecoveryCodes === true) {
      writer.uint32(40).bool(message.regenerateRecoveryCodes);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateUserRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateUserRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.user = User.decode(reader, reader.uint32());
          break;
        case 2:
          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          break;
        case 3:
          message.otpCode = reader.string();
          break;
        case 4:
          message.regenerateTempMfaSecret = reader.bool();
          break;
        case 5:
          message.regenerateRecoveryCodes = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): UpdateUserRequest {
    return {
      user: isSet(object.user) ? User.fromJSON(object.user) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
      otpCode: isSet(object.otpCode) ? String(object.otpCode) : undefined,
      regenerateTempMfaSecret: isSet(object.regenerateTempMfaSecret) ? Boolean(object.regenerateTempMfaSecret) : false,
      regenerateRecoveryCodes: isSet(object.regenerateRecoveryCodes) ? Boolean(object.regenerateRecoveryCodes) : false,
    };
  },

  toJSON(message: UpdateUserRequest): unknown {
    const obj: any = {};
    message.user !== undefined && (obj.user = message.user ? User.toJSON(message.user) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    message.otpCode !== undefined && (obj.otpCode = message.otpCode);
    message.regenerateTempMfaSecret !== undefined && (obj.regenerateTempMfaSecret = message.regenerateTempMfaSecret);
    message.regenerateRecoveryCodes !== undefined && (obj.regenerateRecoveryCodes = message.regenerateRecoveryCodes);
    return obj;
  },

  fromPartial(object: DeepPartial<UpdateUserRequest>): UpdateUserRequest {
    const message = createBaseUpdateUserRequest();
    message.user = (object.user !== undefined && object.user !== null) ? User.fromPartial(object.user) : undefined;
    message.updateMask = object.updateMask ?? undefined;
    message.otpCode = object.otpCode ?? undefined;
    message.regenerateTempMfaSecret = object.regenerateTempMfaSecret ?? false;
    message.regenerateRecoveryCodes = object.regenerateRecoveryCodes ?? false;
    return message;
  },
};

function createBaseDeleteUserRequest(): DeleteUserRequest {
  return { name: "" };
}

export const DeleteUserRequest = {
  encode(message: DeleteUserRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteUserRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteUserRequest();
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

  fromJSON(object: any): DeleteUserRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: DeleteUserRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<DeleteUserRequest>): DeleteUserRequest {
    const message = createBaseDeleteUserRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseUndeleteUserRequest(): UndeleteUserRequest {
  return { name: "" };
}

export const UndeleteUserRequest = {
  encode(message: UndeleteUserRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UndeleteUserRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUndeleteUserRequest();
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

  fromJSON(object: any): UndeleteUserRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: UndeleteUserRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<UndeleteUserRequest>): UndeleteUserRequest {
    const message = createBaseUndeleteUserRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseLoginRequest(): LoginRequest {
  return {
    email: "",
    password: "",
    web: false,
    idpName: "",
    idpContext: undefined,
    otpCode: undefined,
    recoveryCode: undefined,
    mfaTempToken: undefined,
  };
}

export const LoginRequest = {
  encode(message: LoginRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.email !== "") {
      writer.uint32(10).string(message.email);
    }
    if (message.password !== "") {
      writer.uint32(18).string(message.password);
    }
    if (message.web === true) {
      writer.uint32(24).bool(message.web);
    }
    if (message.idpName !== "") {
      writer.uint32(34).string(message.idpName);
    }
    if (message.idpContext !== undefined) {
      IdentityProviderContext.encode(message.idpContext, writer.uint32(42).fork()).ldelim();
    }
    if (message.otpCode !== undefined) {
      writer.uint32(50).string(message.otpCode);
    }
    if (message.recoveryCode !== undefined) {
      writer.uint32(58).string(message.recoveryCode);
    }
    if (message.mfaTempToken !== undefined) {
      writer.uint32(66).string(message.mfaTempToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): LoginRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseLoginRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.email = reader.string();
          break;
        case 2:
          message.password = reader.string();
          break;
        case 3:
          message.web = reader.bool();
          break;
        case 4:
          message.idpName = reader.string();
          break;
        case 5:
          message.idpContext = IdentityProviderContext.decode(reader, reader.uint32());
          break;
        case 6:
          message.otpCode = reader.string();
          break;
        case 7:
          message.recoveryCode = reader.string();
          break;
        case 8:
          message.mfaTempToken = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): LoginRequest {
    return {
      email: isSet(object.email) ? String(object.email) : "",
      password: isSet(object.password) ? String(object.password) : "",
      web: isSet(object.web) ? Boolean(object.web) : false,
      idpName: isSet(object.idpName) ? String(object.idpName) : "",
      idpContext: isSet(object.idpContext) ? IdentityProviderContext.fromJSON(object.idpContext) : undefined,
      otpCode: isSet(object.otpCode) ? String(object.otpCode) : undefined,
      recoveryCode: isSet(object.recoveryCode) ? String(object.recoveryCode) : undefined,
      mfaTempToken: isSet(object.mfaTempToken) ? String(object.mfaTempToken) : undefined,
    };
  },

  toJSON(message: LoginRequest): unknown {
    const obj: any = {};
    message.email !== undefined && (obj.email = message.email);
    message.password !== undefined && (obj.password = message.password);
    message.web !== undefined && (obj.web = message.web);
    message.idpName !== undefined && (obj.idpName = message.idpName);
    message.idpContext !== undefined &&
      (obj.idpContext = message.idpContext ? IdentityProviderContext.toJSON(message.idpContext) : undefined);
    message.otpCode !== undefined && (obj.otpCode = message.otpCode);
    message.recoveryCode !== undefined && (obj.recoveryCode = message.recoveryCode);
    message.mfaTempToken !== undefined && (obj.mfaTempToken = message.mfaTempToken);
    return obj;
  },

  fromPartial(object: DeepPartial<LoginRequest>): LoginRequest {
    const message = createBaseLoginRequest();
    message.email = object.email ?? "";
    message.password = object.password ?? "";
    message.web = object.web ?? false;
    message.idpName = object.idpName ?? "";
    message.idpContext = (object.idpContext !== undefined && object.idpContext !== null)
      ? IdentityProviderContext.fromPartial(object.idpContext)
      : undefined;
    message.otpCode = object.otpCode ?? undefined;
    message.recoveryCode = object.recoveryCode ?? undefined;
    message.mfaTempToken = object.mfaTempToken ?? undefined;
    return message;
  },
};

function createBaseIdentityProviderContext(): IdentityProviderContext {
  return { oauth2Context: undefined, oidcContext: undefined };
}

export const IdentityProviderContext = {
  encode(message: IdentityProviderContext, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.oauth2Context !== undefined) {
      OAuth2IdentityProviderContext.encode(message.oauth2Context, writer.uint32(10).fork()).ldelim();
    }
    if (message.oidcContext !== undefined) {
      OIDCIdentityProviderContext.encode(message.oidcContext, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IdentityProviderContext {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIdentityProviderContext();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.oauth2Context = OAuth2IdentityProviderContext.decode(reader, reader.uint32());
          break;
        case 2:
          message.oidcContext = OIDCIdentityProviderContext.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): IdentityProviderContext {
    return {
      oauth2Context: isSet(object.oauth2Context)
        ? OAuth2IdentityProviderContext.fromJSON(object.oauth2Context)
        : undefined,
      oidcContext: isSet(object.oidcContext) ? OIDCIdentityProviderContext.fromJSON(object.oidcContext) : undefined,
    };
  },

  toJSON(message: IdentityProviderContext): unknown {
    const obj: any = {};
    message.oauth2Context !== undefined && (obj.oauth2Context = message.oauth2Context
      ? OAuth2IdentityProviderContext.toJSON(message.oauth2Context)
      : undefined);
    message.oidcContext !== undefined &&
      (obj.oidcContext = message.oidcContext ? OIDCIdentityProviderContext.toJSON(message.oidcContext) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<IdentityProviderContext>): IdentityProviderContext {
    const message = createBaseIdentityProviderContext();
    message.oauth2Context = (object.oauth2Context !== undefined && object.oauth2Context !== null)
      ? OAuth2IdentityProviderContext.fromPartial(object.oauth2Context)
      : undefined;
    message.oidcContext = (object.oidcContext !== undefined && object.oidcContext !== null)
      ? OIDCIdentityProviderContext.fromPartial(object.oidcContext)
      : undefined;
    return message;
  },
};

function createBaseOAuth2IdentityProviderContext(): OAuth2IdentityProviderContext {
  return { code: "" };
}

export const OAuth2IdentityProviderContext = {
  encode(message: OAuth2IdentityProviderContext, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.code !== "") {
      writer.uint32(10).string(message.code);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): OAuth2IdentityProviderContext {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseOAuth2IdentityProviderContext();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.code = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): OAuth2IdentityProviderContext {
    return { code: isSet(object.code) ? String(object.code) : "" };
  },

  toJSON(message: OAuth2IdentityProviderContext): unknown {
    const obj: any = {};
    message.code !== undefined && (obj.code = message.code);
    return obj;
  },

  fromPartial(object: DeepPartial<OAuth2IdentityProviderContext>): OAuth2IdentityProviderContext {
    const message = createBaseOAuth2IdentityProviderContext();
    message.code = object.code ?? "";
    return message;
  },
};

function createBaseOIDCIdentityProviderContext(): OIDCIdentityProviderContext {
  return {};
}

export const OIDCIdentityProviderContext = {
  encode(_: OIDCIdentityProviderContext, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): OIDCIdentityProviderContext {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseOIDCIdentityProviderContext();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): OIDCIdentityProviderContext {
    return {};
  },

  toJSON(_: OIDCIdentityProviderContext): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<OIDCIdentityProviderContext>): OIDCIdentityProviderContext {
    const message = createBaseOIDCIdentityProviderContext();
    return message;
  },
};

function createBaseLoginResponse(): LoginResponse {
  return { token: "", mfaTempToken: undefined };
}

export const LoginResponse = {
  encode(message: LoginResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.token !== "") {
      writer.uint32(10).string(message.token);
    }
    if (message.mfaTempToken !== undefined) {
      writer.uint32(18).string(message.mfaTempToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): LoginResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseLoginResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.token = reader.string();
          break;
        case 2:
          message.mfaTempToken = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): LoginResponse {
    return {
      token: isSet(object.token) ? String(object.token) : "",
      mfaTempToken: isSet(object.mfaTempToken) ? String(object.mfaTempToken) : undefined,
    };
  },

  toJSON(message: LoginResponse): unknown {
    const obj: any = {};
    message.token !== undefined && (obj.token = message.token);
    message.mfaTempToken !== undefined && (obj.mfaTempToken = message.mfaTempToken);
    return obj;
  },

  fromPartial(object: DeepPartial<LoginResponse>): LoginResponse {
    const message = createBaseLoginResponse();
    message.token = object.token ?? "";
    message.mfaTempToken = object.mfaTempToken ?? undefined;
    return message;
  },
};

function createBaseLogoutRequest(): LogoutRequest {
  return {};
}

export const LogoutRequest = {
  encode(_: LogoutRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): LogoutRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseLogoutRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): LogoutRequest {
    return {};
  },

  toJSON(_: LogoutRequest): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<LogoutRequest>): LogoutRequest {
    const message = createBaseLogoutRequest();
    return message;
  },
};

function createBaseUser(): User {
  return {
    name: "",
    state: 0,
    email: "",
    title: "",
    userType: 0,
    userRole: 0,
    password: "",
    serviceKey: "",
    mfaEnabled: false,
    mfaSecret: "",
    recoveryCodes: [],
  };
}

export const User = {
  encode(message: User, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.state !== 0) {
      writer.uint32(16).int32(message.state);
    }
    if (message.email !== "") {
      writer.uint32(26).string(message.email);
    }
    if (message.title !== "") {
      writer.uint32(34).string(message.title);
    }
    if (message.userType !== 0) {
      writer.uint32(40).int32(message.userType);
    }
    if (message.userRole !== 0) {
      writer.uint32(48).int32(message.userRole);
    }
    if (message.password !== "") {
      writer.uint32(58).string(message.password);
    }
    if (message.serviceKey !== "") {
      writer.uint32(66).string(message.serviceKey);
    }
    if (message.mfaEnabled === true) {
      writer.uint32(72).bool(message.mfaEnabled);
    }
    if (message.mfaSecret !== "") {
      writer.uint32(82).string(message.mfaSecret);
    }
    for (const v of message.recoveryCodes) {
      writer.uint32(90).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): User {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUser();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.state = reader.int32() as any;
          break;
        case 3:
          message.email = reader.string();
          break;
        case 4:
          message.title = reader.string();
          break;
        case 5:
          message.userType = reader.int32() as any;
          break;
        case 6:
          message.userRole = reader.int32() as any;
          break;
        case 7:
          message.password = reader.string();
          break;
        case 8:
          message.serviceKey = reader.string();
          break;
        case 9:
          message.mfaEnabled = reader.bool();
          break;
        case 10:
          message.mfaSecret = reader.string();
          break;
        case 11:
          message.recoveryCodes.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): User {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      state: isSet(object.state) ? stateFromJSON(object.state) : 0,
      email: isSet(object.email) ? String(object.email) : "",
      title: isSet(object.title) ? String(object.title) : "",
      userType: isSet(object.userType) ? userTypeFromJSON(object.userType) : 0,
      userRole: isSet(object.userRole) ? userRoleFromJSON(object.userRole) : 0,
      password: isSet(object.password) ? String(object.password) : "",
      serviceKey: isSet(object.serviceKey) ? String(object.serviceKey) : "",
      mfaEnabled: isSet(object.mfaEnabled) ? Boolean(object.mfaEnabled) : false,
      mfaSecret: isSet(object.mfaSecret) ? String(object.mfaSecret) : "",
      recoveryCodes: Array.isArray(object?.recoveryCodes) ? object.recoveryCodes.map((e: any) => String(e)) : [],
    };
  },

  toJSON(message: User): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.state !== undefined && (obj.state = stateToJSON(message.state));
    message.email !== undefined && (obj.email = message.email);
    message.title !== undefined && (obj.title = message.title);
    message.userType !== undefined && (obj.userType = userTypeToJSON(message.userType));
    message.userRole !== undefined && (obj.userRole = userRoleToJSON(message.userRole));
    message.password !== undefined && (obj.password = message.password);
    message.serviceKey !== undefined && (obj.serviceKey = message.serviceKey);
    message.mfaEnabled !== undefined && (obj.mfaEnabled = message.mfaEnabled);
    message.mfaSecret !== undefined && (obj.mfaSecret = message.mfaSecret);
    if (message.recoveryCodes) {
      obj.recoveryCodes = message.recoveryCodes.map((e) => e);
    } else {
      obj.recoveryCodes = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<User>): User {
    const message = createBaseUser();
    message.name = object.name ?? "";
    message.state = object.state ?? 0;
    message.email = object.email ?? "";
    message.title = object.title ?? "";
    message.userType = object.userType ?? 0;
    message.userRole = object.userRole ?? 0;
    message.password = object.password ?? "";
    message.serviceKey = object.serviceKey ?? "";
    message.mfaEnabled = object.mfaEnabled ?? false;
    message.mfaSecret = object.mfaSecret ?? "";
    message.recoveryCodes = object.recoveryCodes?.map((e) => e) || [];
    return message;
  },
};

export type AuthServiceDefinition = typeof AuthServiceDefinition;
export const AuthServiceDefinition = {
  name: "AuthService",
  fullName: "bytebase.v1.AuthService",
  methods: {
    getUser: {
      name: "GetUser",
      requestType: GetUserRequest,
      requestStream: false,
      responseType: User,
      responseStream: false,
      options: {},
    },
    listUsers: {
      name: "ListUsers",
      requestType: ListUsersRequest,
      requestStream: false,
      responseType: ListUsersResponse,
      responseStream: false,
      options: {},
    },
    createUser: {
      name: "CreateUser",
      requestType: CreateUserRequest,
      requestStream: false,
      responseType: User,
      responseStream: false,
      options: {},
    },
    updateUser: {
      name: "UpdateUser",
      requestType: UpdateUserRequest,
      requestStream: false,
      responseType: User,
      responseStream: false,
      options: {},
    },
    deleteUser: {
      name: "DeleteUser",
      requestType: DeleteUserRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {},
    },
    undeleteUser: {
      name: "UndeleteUser",
      requestType: UndeleteUserRequest,
      requestStream: false,
      responseType: User,
      responseStream: false,
      options: {},
    },
    login: {
      name: "Login",
      requestType: LoginRequest,
      requestStream: false,
      responseType: LoginResponse,
      responseStream: false,
      options: {},
    },
    logout: {
      name: "Logout",
      requestType: LogoutRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {},
    },
  },
} as const;

export interface AuthServiceImplementation<CallContextExt = {}> {
  getUser(request: GetUserRequest, context: CallContext & CallContextExt): Promise<DeepPartial<User>>;
  listUsers(request: ListUsersRequest, context: CallContext & CallContextExt): Promise<DeepPartial<ListUsersResponse>>;
  createUser(request: CreateUserRequest, context: CallContext & CallContextExt): Promise<DeepPartial<User>>;
  updateUser(request: UpdateUserRequest, context: CallContext & CallContextExt): Promise<DeepPartial<User>>;
  deleteUser(request: DeleteUserRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Empty>>;
  undeleteUser(request: UndeleteUserRequest, context: CallContext & CallContextExt): Promise<DeepPartial<User>>;
  login(request: LoginRequest, context: CallContext & CallContextExt): Promise<DeepPartial<LoginResponse>>;
  logout(request: LogoutRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Empty>>;
}

export interface AuthServiceClient<CallOptionsExt = {}> {
  getUser(request: DeepPartial<GetUserRequest>, options?: CallOptions & CallOptionsExt): Promise<User>;
  listUsers(request: DeepPartial<ListUsersRequest>, options?: CallOptions & CallOptionsExt): Promise<ListUsersResponse>;
  createUser(request: DeepPartial<CreateUserRequest>, options?: CallOptions & CallOptionsExt): Promise<User>;
  updateUser(request: DeepPartial<UpdateUserRequest>, options?: CallOptions & CallOptionsExt): Promise<User>;
  deleteUser(request: DeepPartial<DeleteUserRequest>, options?: CallOptions & CallOptionsExt): Promise<Empty>;
  undeleteUser(request: DeepPartial<UndeleteUserRequest>, options?: CallOptions & CallOptionsExt): Promise<User>;
  login(request: DeepPartial<LoginRequest>, options?: CallOptions & CallOptionsExt): Promise<LoginResponse>;
  logout(request: DeepPartial<LogoutRequest>, options?: CallOptions & CallOptionsExt): Promise<Empty>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
