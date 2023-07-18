/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { Duration } from "../google/protobuf/duration";
import { Expr } from "../google/type/expr";
import { IssuePayloadApproval } from "./approval";

export const protobufPackage = "bytebase.store";

export interface IssuePayload {
  approval?: IssuePayloadApproval | undefined;
  grantRequest?: GrantRequest | undefined;
  grouping?: Grouping | undefined;
}

export interface Grouping {
  /** The group name, format projects/{project}/database_groups/{database_group} */
  databaseGroupName: string;
}

export interface GrantRequest {
  /** The requested role, e.g. roles/EXPORTER. */
  role: string;
  /** The requested user, e.g. users/hello@bytebase.com. */
  user: string;
  condition?: Expr | undefined;
  expiration?: Duration | undefined;
}

function createBaseIssuePayload(): IssuePayload {
  return { approval: undefined, grantRequest: undefined, grouping: undefined };
}

export const IssuePayload = {
  encode(message: IssuePayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.approval !== undefined) {
      IssuePayloadApproval.encode(message.approval, writer.uint32(10).fork()).ldelim();
    }
    if (message.grantRequest !== undefined) {
      GrantRequest.encode(message.grantRequest, writer.uint32(18).fork()).ldelim();
    }
    if (message.grouping !== undefined) {
      Grouping.encode(message.grouping, writer.uint32(26).fork()).ldelim();
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
        case 3:
          if (tag !== 26) {
            break;
          }

          message.grouping = Grouping.decode(reader, reader.uint32());
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
      grouping: isSet(object.grouping) ? Grouping.fromJSON(object.grouping) : undefined,
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
    if (message.grouping !== undefined) {
      obj.grouping = Grouping.toJSON(message.grouping);
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
    message.grouping = (object.grouping !== undefined && object.grouping !== null)
      ? Grouping.fromPartial(object.grouping)
      : undefined;
    return message;
  },
};

function createBaseGrouping(): Grouping {
  return { databaseGroupName: "" };
}

export const Grouping = {
  encode(message: Grouping, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.databaseGroupName !== "") {
      writer.uint32(10).string(message.databaseGroupName);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Grouping {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGrouping();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.databaseGroupName = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Grouping {
    return { databaseGroupName: isSet(object.databaseGroupName) ? String(object.databaseGroupName) : "" };
  },

  toJSON(message: Grouping): unknown {
    const obj: any = {};
    if (message.databaseGroupName !== "") {
      obj.databaseGroupName = message.databaseGroupName;
    }
    return obj;
  },

  create(base?: DeepPartial<Grouping>): Grouping {
    return Grouping.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Grouping>): Grouping {
    const message = createBaseGrouping();
    message.databaseGroupName = object.databaseGroupName ?? "";
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
      role: isSet(object.role) ? String(object.role) : "",
      user: isSet(object.user) ? String(object.user) : "",
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
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
