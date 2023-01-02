/* eslint-disable */
import _m0 from "protobufjs/minimal";
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
}

export interface LoginResponse {
  token: string;
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
  password: string;
  userType: UserType;
  /** The user role will not be respected in the create user request, because the role is controlled by workspace owner. */
  userRole: UserRole;
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

  fromPartial<I extends Exact<DeepPartial<GetUserRequest>, I>>(object: I): GetUserRequest {
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

  fromPartial<I extends Exact<DeepPartial<ListUsersRequest>, I>>(object: I): ListUsersRequest {
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

  fromPartial<I extends Exact<DeepPartial<ListUsersResponse>, I>>(object: I): ListUsersResponse {
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

  fromPartial<I extends Exact<DeepPartial<CreateUserRequest>, I>>(object: I): CreateUserRequest {
    const message = createBaseCreateUserRequest();
    message.user = (object.user !== undefined && object.user !== null) ? User.fromPartial(object.user) : undefined;
    return message;
  },
};

function createBaseUpdateUserRequest(): UpdateUserRequest {
  return { user: undefined, updateMask: undefined };
}

export const UpdateUserRequest = {
  encode(message: UpdateUserRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.user !== undefined) {
      User.encode(message.user, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
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
    };
  },

  toJSON(message: UpdateUserRequest): unknown {
    const obj: any = {};
    message.user !== undefined && (obj.user = message.user ? User.toJSON(message.user) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<UpdateUserRequest>, I>>(object: I): UpdateUserRequest {
    const message = createBaseUpdateUserRequest();
    message.user = (object.user !== undefined && object.user !== null) ? User.fromPartial(object.user) : undefined;
    message.updateMask = object.updateMask ?? undefined;
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

  fromPartial<I extends Exact<DeepPartial<DeleteUserRequest>, I>>(object: I): DeleteUserRequest {
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

  fromPartial<I extends Exact<DeepPartial<UndeleteUserRequest>, I>>(object: I): UndeleteUserRequest {
    const message = createBaseUndeleteUserRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseLoginRequest(): LoginRequest {
  return { email: "", password: "", web: false };
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
    };
  },

  toJSON(message: LoginRequest): unknown {
    const obj: any = {};
    message.email !== undefined && (obj.email = message.email);
    message.password !== undefined && (obj.password = message.password);
    message.web !== undefined && (obj.web = message.web);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<LoginRequest>, I>>(object: I): LoginRequest {
    const message = createBaseLoginRequest();
    message.email = object.email ?? "";
    message.password = object.password ?? "";
    message.web = object.web ?? false;
    return message;
  },
};

function createBaseLoginResponse(): LoginResponse {
  return { token: "" };
}

export const LoginResponse = {
  encode(message: LoginResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.token !== "") {
      writer.uint32(10).string(message.token);
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
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): LoginResponse {
    return { token: isSet(object.token) ? String(object.token) : "" };
  },

  toJSON(message: LoginResponse): unknown {
    const obj: any = {};
    message.token !== undefined && (obj.token = message.token);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<LoginResponse>, I>>(object: I): LoginResponse {
    const message = createBaseLoginResponse();
    message.token = object.token ?? "";
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

  fromPartial<I extends Exact<DeepPartial<LogoutRequest>, I>>(_: I): LogoutRequest {
    const message = createBaseLogoutRequest();
    return message;
  },
};

function createBaseUser(): User {
  return { name: "", state: 0, email: "", title: "", password: "", userType: 0, userRole: 0 };
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
    if (message.password !== "") {
      writer.uint32(42).string(message.password);
    }
    if (message.userType !== 0) {
      writer.uint32(48).int32(message.userType);
    }
    if (message.userRole !== 0) {
      writer.uint32(56).int32(message.userRole);
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
          message.password = reader.string();
          break;
        case 6:
          message.userType = reader.int32() as any;
          break;
        case 7:
          message.userRole = reader.int32() as any;
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
      password: isSet(object.password) ? String(object.password) : "",
      userType: isSet(object.userType) ? userTypeFromJSON(object.userType) : 0,
      userRole: isSet(object.userRole) ? userRoleFromJSON(object.userRole) : 0,
    };
  },

  toJSON(message: User): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.state !== undefined && (obj.state = stateToJSON(message.state));
    message.email !== undefined && (obj.email = message.email);
    message.title !== undefined && (obj.title = message.title);
    message.password !== undefined && (obj.password = message.password);
    message.userType !== undefined && (obj.userType = userTypeToJSON(message.userType));
    message.userRole !== undefined && (obj.userRole = userRoleToJSON(message.userRole));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<User>, I>>(object: I): User {
    const message = createBaseUser();
    message.name = object.name ?? "";
    message.state = object.state ?? 0;
    message.email = object.email ?? "";
    message.title = object.title ?? "";
    message.password = object.password ?? "";
    message.userType = object.userType ?? 0;
    message.userRole = object.userRole ?? 0;
    return message;
  },
};

export interface AuthService {
  GetUser(request: GetUserRequest): Promise<User>;
  ListUsers(request: ListUsersRequest): Promise<ListUsersResponse>;
  CreateUser(request: CreateUserRequest): Promise<User>;
  UpdateUser(request: UpdateUserRequest): Promise<User>;
  DeleteUser(request: DeleteUserRequest): Promise<Empty>;
  UndeleteUser(request: UndeleteUserRequest): Promise<User>;
  Login(request: LoginRequest): Promise<LoginResponse>;
  Logout(request: LogoutRequest): Promise<Empty>;
}

export class AuthServiceClientImpl implements AuthService {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "bytebase.v1.AuthService";
    this.rpc = rpc;
    this.GetUser = this.GetUser.bind(this);
    this.ListUsers = this.ListUsers.bind(this);
    this.CreateUser = this.CreateUser.bind(this);
    this.UpdateUser = this.UpdateUser.bind(this);
    this.DeleteUser = this.DeleteUser.bind(this);
    this.UndeleteUser = this.UndeleteUser.bind(this);
    this.Login = this.Login.bind(this);
    this.Logout = this.Logout.bind(this);
  }
  GetUser(request: GetUserRequest): Promise<User> {
    const data = GetUserRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "GetUser", data);
    return promise.then((data) => User.decode(new _m0.Reader(data)));
  }

  ListUsers(request: ListUsersRequest): Promise<ListUsersResponse> {
    const data = ListUsersRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "ListUsers", data);
    return promise.then((data) => ListUsersResponse.decode(new _m0.Reader(data)));
  }

  CreateUser(request: CreateUserRequest): Promise<User> {
    const data = CreateUserRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "CreateUser", data);
    return promise.then((data) => User.decode(new _m0.Reader(data)));
  }

  UpdateUser(request: UpdateUserRequest): Promise<User> {
    const data = UpdateUserRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "UpdateUser", data);
    return promise.then((data) => User.decode(new _m0.Reader(data)));
  }

  DeleteUser(request: DeleteUserRequest): Promise<Empty> {
    const data = DeleteUserRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "DeleteUser", data);
    return promise.then((data) => Empty.decode(new _m0.Reader(data)));
  }

  UndeleteUser(request: UndeleteUserRequest): Promise<User> {
    const data = UndeleteUserRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "UndeleteUser", data);
    return promise.then((data) => User.decode(new _m0.Reader(data)));
  }

  Login(request: LoginRequest): Promise<LoginResponse> {
    const data = LoginRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "Login", data);
    return promise.then((data) => LoginResponse.decode(new _m0.Reader(data)));
  }

  Logout(request: LogoutRequest): Promise<Empty> {
    const data = LogoutRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "Logout", data);
    return promise.then((data) => Empty.decode(new _m0.Reader(data)));
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
