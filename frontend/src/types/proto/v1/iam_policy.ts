/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { ParsedExpr } from "../google/api/expr/v1alpha1/syntax";
import { Expr } from "../google/type/expr";

export const protobufPackage = "bytebase.v1";

export interface GetIamPolicyRequest {
  /**
   * The name of the resource to get the IAM policy.
   * Format: projects/{project}
   * Format: workspaces/{workspace}
   */
  resource: string;
}

export interface SetIamPolicyRequest {
  /**
   * The name of the resource to set the IAM policy.
   * Format: projects/{project}
   * Format: workspaces/{workspace}
   */
  resource: string;
  policy:
    | IamPolicy
    | undefined;
  /** The current etag of the policy. */
  etag: string;
}

export interface IamPolicy {
  /**
   * Collection of binding.
   * A binding binds one or more project members to a single project role.
   */
  bindings: Binding[];
  /**
   * The current etag of the policy.
   * If an etag is provided and does not match the current etag of the poliy,
   * the call will be blocked and an ABORTED error will be returned.
   */
  etag: string;
}

export interface Binding {
  /**
   * The role that is assigned to the members.
   * Format: roles/{role}
   */
  role: string;
  /**
   * Specifies the principals requesting access for a Bytebase resource.
   * For users, the member should be: user:{email}
   * For groups, the member should be: group:{email}
   */
  members: string[];
  /**
   * The condition that is associated with this binding.
   * If the condition evaluates to true, then this binding applies to the current request.
   * If the condition evaluates to false, then this binding does not apply to the current request. However, a different role binding might grant the same role to one or more of the principals in this binding.
   */
  condition:
    | Expr
    | undefined;
  /** The parsed expression of the condition. */
  parsedExpr: ParsedExpr | undefined;
}

function createBaseGetIamPolicyRequest(): GetIamPolicyRequest {
  return { resource: "" };
}

export const GetIamPolicyRequest = {
  encode(message: GetIamPolicyRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.resource !== "") {
      writer.uint32(10).string(message.resource);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetIamPolicyRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetIamPolicyRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.resource = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GetIamPolicyRequest {
    return { resource: isSet(object.resource) ? globalThis.String(object.resource) : "" };
  },

  toJSON(message: GetIamPolicyRequest): unknown {
    const obj: any = {};
    if (message.resource !== "") {
      obj.resource = message.resource;
    }
    return obj;
  },

  create(base?: DeepPartial<GetIamPolicyRequest>): GetIamPolicyRequest {
    return GetIamPolicyRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<GetIamPolicyRequest>): GetIamPolicyRequest {
    const message = createBaseGetIamPolicyRequest();
    message.resource = object.resource ?? "";
    return message;
  },
};

function createBaseSetIamPolicyRequest(): SetIamPolicyRequest {
  return { resource: "", policy: undefined, etag: "" };
}

export const SetIamPolicyRequest = {
  encode(message: SetIamPolicyRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.resource !== "") {
      writer.uint32(10).string(message.resource);
    }
    if (message.policy !== undefined) {
      IamPolicy.encode(message.policy, writer.uint32(18).fork()).ldelim();
    }
    if (message.etag !== "") {
      writer.uint32(26).string(message.etag);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SetIamPolicyRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSetIamPolicyRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.resource = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.policy = IamPolicy.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.etag = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SetIamPolicyRequest {
    return {
      resource: isSet(object.resource) ? globalThis.String(object.resource) : "",
      policy: isSet(object.policy) ? IamPolicy.fromJSON(object.policy) : undefined,
      etag: isSet(object.etag) ? globalThis.String(object.etag) : "",
    };
  },

  toJSON(message: SetIamPolicyRequest): unknown {
    const obj: any = {};
    if (message.resource !== "") {
      obj.resource = message.resource;
    }
    if (message.policy !== undefined) {
      obj.policy = IamPolicy.toJSON(message.policy);
    }
    if (message.etag !== "") {
      obj.etag = message.etag;
    }
    return obj;
  },

  create(base?: DeepPartial<SetIamPolicyRequest>): SetIamPolicyRequest {
    return SetIamPolicyRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SetIamPolicyRequest>): SetIamPolicyRequest {
    const message = createBaseSetIamPolicyRequest();
    message.resource = object.resource ?? "";
    message.policy = (object.policy !== undefined && object.policy !== null)
      ? IamPolicy.fromPartial(object.policy)
      : undefined;
    message.etag = object.etag ?? "";
    return message;
  },
};

function createBaseIamPolicy(): IamPolicy {
  return { bindings: [], etag: "" };
}

export const IamPolicy = {
  encode(message: IamPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.bindings) {
      Binding.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.etag !== "") {
      writer.uint32(18).string(message.etag);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IamPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIamPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.bindings.push(Binding.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.etag = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): IamPolicy {
    return {
      bindings: globalThis.Array.isArray(object?.bindings) ? object.bindings.map((e: any) => Binding.fromJSON(e)) : [],
      etag: isSet(object.etag) ? globalThis.String(object.etag) : "",
    };
  },

  toJSON(message: IamPolicy): unknown {
    const obj: any = {};
    if (message.bindings?.length) {
      obj.bindings = message.bindings.map((e) => Binding.toJSON(e));
    }
    if (message.etag !== "") {
      obj.etag = message.etag;
    }
    return obj;
  },

  create(base?: DeepPartial<IamPolicy>): IamPolicy {
    return IamPolicy.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<IamPolicy>): IamPolicy {
    const message = createBaseIamPolicy();
    message.bindings = object.bindings?.map((e) => Binding.fromPartial(e)) || [];
    message.etag = object.etag ?? "";
    return message;
  },
};

function createBaseBinding(): Binding {
  return { role: "", members: [], condition: undefined, parsedExpr: undefined };
}

export const Binding = {
  encode(message: Binding, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.role !== "") {
      writer.uint32(10).string(message.role);
    }
    for (const v of message.members) {
      writer.uint32(18).string(v!);
    }
    if (message.condition !== undefined) {
      Expr.encode(message.condition, writer.uint32(26).fork()).ldelim();
    }
    if (message.parsedExpr !== undefined) {
      ParsedExpr.encode(message.parsedExpr, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Binding {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBinding();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.role = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.members.push(reader.string());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.condition = Expr.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.parsedExpr = ParsedExpr.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Binding {
    return {
      role: isSet(object.role) ? globalThis.String(object.role) : "",
      members: globalThis.Array.isArray(object?.members) ? object.members.map((e: any) => globalThis.String(e)) : [],
      condition: isSet(object.condition) ? Expr.fromJSON(object.condition) : undefined,
      parsedExpr: isSet(object.parsedExpr) ? ParsedExpr.fromJSON(object.parsedExpr) : undefined,
    };
  },

  toJSON(message: Binding): unknown {
    const obj: any = {};
    if (message.role !== "") {
      obj.role = message.role;
    }
    if (message.members?.length) {
      obj.members = message.members;
    }
    if (message.condition !== undefined) {
      obj.condition = Expr.toJSON(message.condition);
    }
    if (message.parsedExpr !== undefined) {
      obj.parsedExpr = ParsedExpr.toJSON(message.parsedExpr);
    }
    return obj;
  },

  create(base?: DeepPartial<Binding>): Binding {
    return Binding.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Binding>): Binding {
    const message = createBaseBinding();
    message.role = object.role ?? "";
    message.members = object.members?.map((e) => e) || [];
    message.condition = (object.condition !== undefined && object.condition !== null)
      ? Expr.fromPartial(object.condition)
      : undefined;
    message.parsedExpr = (object.parsedExpr !== undefined && object.parsedExpr !== null)
      ? ParsedExpr.fromPartial(object.parsedExpr)
      : undefined;
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
