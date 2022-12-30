/* eslint-disable */
import _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { State, stateFromJSON, stateToJSON } from "./common";

export const protobufPackage = "bytebase.v1";

export enum UserRole {
  USER_ROLE_UNSPECIFIED = 0,
  USER_ROLE_OWNER = 1,
  USER_ROLE_DBA = 2,
  USER_ROLE_DEVELOPER = 3,
  UNRECOGNIZED = -1,
}

export function userRoleFromJSON(object: any): UserRole {
  switch (object) {
    case 0:
    case "USER_ROLE_UNSPECIFIED":
      return UserRole.USER_ROLE_UNSPECIFIED;
    case 1:
    case "USER_ROLE_OWNER":
      return UserRole.USER_ROLE_OWNER;
    case 2:
    case "USER_ROLE_DBA":
      return UserRole.USER_ROLE_DBA;
    case 3:
    case "USER_ROLE_DEVELOPER":
      return UserRole.USER_ROLE_DEVELOPER;
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
    case UserRole.USER_ROLE_OWNER:
      return "USER_ROLE_OWNER";
    case UserRole.USER_ROLE_DBA:
      return "USER_ROLE_DBA";
    case UserRole.USER_ROLE_DEVELOPER:
      return "USER_ROLE_DEVELOPER";
    case UserRole.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface CreateUserRequest {
  /** The user to create. */
  user?: User;
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
  /** The user role will not be respected in the create user request, because the role is controlled by workspace owner. */
  userRole: UserRole;
}

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
  return { name: "", state: 0, email: "", title: "", password: "", userRole: 0 };
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
    if (message.userRole !== 0) {
      writer.uint32(48).int32(message.userRole);
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
    message.userRole = object.userRole ?? 0;
    return message;
  },
};

export interface AuthService {
  CreateUser(request: CreateUserRequest): Promise<User>;
  Login(request: LoginRequest): Promise<LoginResponse>;
  Logout(request: LogoutRequest): Promise<Empty>;
}

export class AuthServiceClientImpl implements AuthService {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "bytebase.v1.AuthService";
    this.rpc = rpc;
    this.CreateUser = this.CreateUser.bind(this);
    this.Login = this.Login.bind(this);
    this.Logout = this.Logout.bind(this);
  }
  CreateUser(request: CreateUserRequest): Promise<User> {
    const data = CreateUserRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "CreateUser", data);
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
