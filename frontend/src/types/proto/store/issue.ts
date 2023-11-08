/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Duration } from "../google/protobuf/duration";
import { Expr } from "../google/type/expr";
import { IssuePayloadApproval } from "./approval";

export const protobufPackage = "bytebase.store";

export interface IssuePayload {
  approval: IssuePayloadApproval | undefined;
  grantRequest: GrantRequest | undefined;
}

export interface GrantRequest {
  /**
   * The requested role.
   * Format: roles/EXPORTER.
   */
  role: string;
  /**
   * The user to be granted.
   * Format: users/{userUID}.
   */
  user: string;
  condition: Expr | undefined;
  expiration: Duration | undefined;
}

function createBaseIssuePayload(): IssuePayload {
  return { approval: undefined, grantRequest: undefined };
}

export const IssuePayload = {
  encode(message: IssuePayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.approval !== undefined) {
      IssuePayloadApproval.encode(message.approval, writer.uint32(10).fork()).ldelim();
    }
    if (message.grantRequest !== undefined) {
      GrantRequest.encode(message.grantRequest, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IssuePayload {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIssuePayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.approval = IssuePayloadApproval.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.grantRequest = GrantRequest.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): IssuePayload {
    return {
      approval: isSet(object.approval) ? IssuePayloadApproval.fromJSON(object.approval) : undefined,
      grantRequest: isSet(object.grantRequest) ? GrantRequest.fromJSON(object.grantRequest) : undefined,
    };
  },

  toJSON(message: IssuePayload): unknown {
    const obj: any = {};
    if (message.approval !== undefined) {
      obj.approval = IssuePayloadApproval.toJSON(message.approval);
    }
    if (message.grantRequest !== undefined) {
      obj.grantRequest = GrantRequest.toJSON(message.grantRequest);
    }
    return obj;
  },

  create(base?: DeepPartial<IssuePayload>): IssuePayload {
    return IssuePayload.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<IssuePayload>): IssuePayload {
    const message = createBaseIssuePayload();
    message.approval = (object.approval !== undefined && object.approval !== null)
      ? IssuePayloadApproval.fromPartial(object.approval)
      : undefined;
    message.grantRequest = (object.grantRequest !== undefined && object.grantRequest !== null)
      ? GrantRequest.fromPartial(object.grantRequest)
      : undefined;
    return message;
  },
};

function createBaseGrantRequest(): GrantRequest {
  return { role: "", user: "", condition: undefined, expiration: undefined };
}

export const GrantRequest = {
  encode(message: GrantRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.role !== "") {
      writer.uint32(10).string(message.role);
    }
    if (message.user !== "") {
      writer.uint32(18).string(message.user);
    }
    if (message.condition !== undefined) {
      Expr.encode(message.condition, writer.uint32(26).fork()).ldelim();
    }
    if (message.expiration !== undefined) {
      Duration.encode(message.expiration, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GrantRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGrantRequest();
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

          message.user = reader.string();
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

          message.expiration = Duration.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GrantRequest {
    return {
      role: isSet(object.role) ? globalThis.String(object.role) : "",
      user: isSet(object.user) ? globalThis.String(object.user) : "",
      condition: isSet(object.condition) ? Expr.fromJSON(object.condition) : undefined,
      expiration: isSet(object.expiration) ? Duration.fromJSON(object.expiration) : undefined,
    };
  },

  toJSON(message: GrantRequest): unknown {
    const obj: any = {};
    if (message.role !== "") {
      obj.role = message.role;
    }
    if (message.user !== "") {
      obj.user = message.user;
    }
    if (message.condition !== undefined) {
      obj.condition = Expr.toJSON(message.condition);
    }
    if (message.expiration !== undefined) {
      obj.expiration = Duration.toJSON(message.expiration);
    }
    return obj;
  },

  create(base?: DeepPartial<GrantRequest>): GrantRequest {
    return GrantRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<GrantRequest>): GrantRequest {
    const message = createBaseGrantRequest();
    message.role = object.role ?? "";
    message.user = object.user ?? "";
    message.condition = (object.condition !== undefined && object.condition !== null)
      ? Expr.fromPartial(object.condition)
      : undefined;
    message.expiration = (object.expiration !== undefined && object.expiration !== null)
      ? Duration.fromPartial(object.expiration)
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
