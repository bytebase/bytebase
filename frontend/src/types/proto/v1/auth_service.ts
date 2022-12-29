/* eslint-disable */
import _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";

export const protobufPackage = "bytebase.v1";

export interface SignupRequest {
  /** email is the account email. */
  email: string;
  /** name is the name of the user. */
  name: string;
  /** password is the login password. */
  password: string;
}

export interface SignupResponse {
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

function createBaseSignupRequest(): SignupRequest {
  return { email: "", name: "", password: "" };
}

export const SignupRequest = {
  encode(message: SignupRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.email !== "") {
      writer.uint32(10).string(message.email);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.password !== "") {
      writer.uint32(26).string(message.password);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SignupRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSignupRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.email = reader.string();
          break;
        case 2:
          message.name = reader.string();
          break;
        case 3:
          message.password = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): SignupRequest {
    return {
      email: isSet(object.email) ? String(object.email) : "",
      name: isSet(object.name) ? String(object.name) : "",
      password: isSet(object.password) ? String(object.password) : "",
    };
  },

  toJSON(message: SignupRequest): unknown {
    const obj: any = {};
    message.email !== undefined && (obj.email = message.email);
    message.name !== undefined && (obj.name = message.name);
    message.password !== undefined && (obj.password = message.password);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<SignupRequest>, I>>(object: I): SignupRequest {
    const message = createBaseSignupRequest();
    message.email = object.email ?? "";
    message.name = object.name ?? "";
    message.password = object.password ?? "";
    return message;
  },
};

function createBaseSignupResponse(): SignupResponse {
  return {};
}

export const SignupResponse = {
  encode(_: SignupResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SignupResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSignupResponse();
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

  fromJSON(_: any): SignupResponse {
    return {};
  },

  toJSON(_: SignupResponse): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<SignupResponse>, I>>(_: I): SignupResponse {
    const message = createBaseSignupResponse();
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

export interface AuthService {
  Signup(request: SignupRequest): Promise<SignupResponse>;
  Login(request: LoginRequest): Promise<LoginResponse>;
  Logout(request: LogoutRequest): Promise<Empty>;
}

export class AuthServiceClientImpl implements AuthService {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "bytebase.v1.AuthService";
    this.rpc = rpc;
    this.Signup = this.Signup.bind(this);
    this.Login = this.Login.bind(this);
    this.Logout = this.Logout.bind(this);
  }
  Signup(request: SignupRequest): Promise<SignupResponse> {
    const data = SignupRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "Signup", data);
    return promise.then((data) => SignupResponse.decode(new _m0.Reader(data)));
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
