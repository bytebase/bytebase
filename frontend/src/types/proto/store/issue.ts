/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { Expr } from "../google/type/expr";
import { IssuePayloadApproval } from "./approval";

export const protobufPackage = "bytebase.store";

export interface IssuePayload {
  approval?: IssuePayloadApproval;
  grantRequest?: GrantRequest;
}

export interface GrantRequest {
  /** The requested role, e.g. roles/EXPORTER. */
  role: string;
  /** The requested user, e.g. users/hello@bytebase.com. */
  user: string;
  condition?: Expr;
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
    message.approval !== undefined &&
      (obj.approval = message.approval ? IssuePayloadApproval.toJSON(message.approval) : undefined);
    message.grantRequest !== undefined &&
      (obj.grantRequest = message.grantRequest ? GrantRequest.toJSON(message.grantRequest) : undefined);
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
  return { role: "", user: "", condition: undefined };
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
      role: isSet(object.role) ? String(object.role) : "",
      user: isSet(object.user) ? String(object.user) : "",
      condition: isSet(object.condition) ? Expr.fromJSON(object.condition) : undefined,
    };
  },

  toJSON(message: GrantRequest): unknown {
    const obj: any = {};
    message.role !== undefined && (obj.role = message.role);
    message.user !== undefined && (obj.user = message.user);
    message.condition !== undefined && (obj.condition = message.condition ? Expr.toJSON(message.condition) : undefined);
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
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
