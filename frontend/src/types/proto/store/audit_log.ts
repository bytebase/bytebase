/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Status } from "../google/rpc/status";

export const protobufPackage = "bytebase.store";

export interface AuditLog {
  /**
   * The project or workspace the audit log belongs to.
   * Formats:
   * - projects/{project}
   * - workspaces/{workspace}
   */
  parent: string;
  /** e.g. /bytebase.v1.SQLService/Query */
  method: string;
  /**
   * resource name
   * projects/{project}
   */
  resource: string;
  /** Format: users/d@d.com */
  user: string;
  severity: AuditLog_Severity;
  /** Marshalled request. */
  request: string;
  /**
   * Marshalled response.
   * Some fields are omitted because they are too large or contain sensitive information.
   */
  response: string;
  status: Status | undefined;
}

export enum AuditLog_Severity {
  DEFAULT = "DEFAULT",
  DEBUG = "DEBUG",
  INFO = "INFO",
  NOTICE = "NOTICE",
  WARNING = "WARNING",
  ERROR = "ERROR",
  CRITICAL = "CRITICAL",
  ALERT = "ALERT",
  EMERGENCY = "EMERGENCY",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function auditLog_SeverityFromJSON(object: any): AuditLog_Severity {
  switch (object) {
    case 0:
    case "DEFAULT":
      return AuditLog_Severity.DEFAULT;
    case 1:
    case "DEBUG":
      return AuditLog_Severity.DEBUG;
    case 2:
    case "INFO":
      return AuditLog_Severity.INFO;
    case 3:
    case "NOTICE":
      return AuditLog_Severity.NOTICE;
    case 4:
    case "WARNING":
      return AuditLog_Severity.WARNING;
    case 5:
    case "ERROR":
      return AuditLog_Severity.ERROR;
    case 6:
    case "CRITICAL":
      return AuditLog_Severity.CRITICAL;
    case 7:
    case "ALERT":
      return AuditLog_Severity.ALERT;
    case 8:
    case "EMERGENCY":
      return AuditLog_Severity.EMERGENCY;
    case -1:
    case "UNRECOGNIZED":
    default:
      return AuditLog_Severity.UNRECOGNIZED;
  }
}

export function auditLog_SeverityToJSON(object: AuditLog_Severity): string {
  switch (object) {
    case AuditLog_Severity.DEFAULT:
      return "DEFAULT";
    case AuditLog_Severity.DEBUG:
      return "DEBUG";
    case AuditLog_Severity.INFO:
      return "INFO";
    case AuditLog_Severity.NOTICE:
      return "NOTICE";
    case AuditLog_Severity.WARNING:
      return "WARNING";
    case AuditLog_Severity.ERROR:
      return "ERROR";
    case AuditLog_Severity.CRITICAL:
      return "CRITICAL";
    case AuditLog_Severity.ALERT:
      return "ALERT";
    case AuditLog_Severity.EMERGENCY:
      return "EMERGENCY";
    case AuditLog_Severity.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function auditLog_SeverityToNumber(object: AuditLog_Severity): number {
  switch (object) {
    case AuditLog_Severity.DEFAULT:
      return 0;
    case AuditLog_Severity.DEBUG:
      return 1;
    case AuditLog_Severity.INFO:
      return 2;
    case AuditLog_Severity.NOTICE:
      return 3;
    case AuditLog_Severity.WARNING:
      return 4;
    case AuditLog_Severity.ERROR:
      return 5;
    case AuditLog_Severity.CRITICAL:
      return 6;
    case AuditLog_Severity.ALERT:
      return 7;
    case AuditLog_Severity.EMERGENCY:
      return 8;
    case AuditLog_Severity.UNRECOGNIZED:
    default:
      return -1;
  }
}

function createBaseAuditLog(): AuditLog {
  return {
    parent: "",
    method: "",
    resource: "",
    user: "",
    severity: AuditLog_Severity.DEFAULT,
    request: "",
    response: "",
    status: undefined,
  };
}

export const AuditLog = {
  encode(message: AuditLog, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.method !== "") {
      writer.uint32(18).string(message.method);
    }
    if (message.resource !== "") {
      writer.uint32(26).string(message.resource);
    }
    if (message.user !== "") {
      writer.uint32(34).string(message.user);
    }
    if (message.severity !== AuditLog_Severity.DEFAULT) {
      writer.uint32(40).int32(auditLog_SeverityToNumber(message.severity));
    }
    if (message.request !== "") {
      writer.uint32(50).string(message.request);
    }
    if (message.response !== "") {
      writer.uint32(58).string(message.response);
    }
    if (message.status !== undefined) {
      Status.encode(message.status, writer.uint32(66).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): AuditLog {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAuditLog();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.parent = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.method = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.resource = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.user = reader.string();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.severity = auditLog_SeverityFromJSON(reader.int32());
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.request = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.response = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.status = Status.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): AuditLog {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      method: isSet(object.method) ? globalThis.String(object.method) : "",
      resource: isSet(object.resource) ? globalThis.String(object.resource) : "",
      user: isSet(object.user) ? globalThis.String(object.user) : "",
      severity: isSet(object.severity) ? auditLog_SeverityFromJSON(object.severity) : AuditLog_Severity.DEFAULT,
      request: isSet(object.request) ? globalThis.String(object.request) : "",
      response: isSet(object.response) ? globalThis.String(object.response) : "",
      status: isSet(object.status) ? Status.fromJSON(object.status) : undefined,
    };
  },

  toJSON(message: AuditLog): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.method !== "") {
      obj.method = message.method;
    }
    if (message.resource !== "") {
      obj.resource = message.resource;
    }
    if (message.user !== "") {
      obj.user = message.user;
    }
    if (message.severity !== AuditLog_Severity.DEFAULT) {
      obj.severity = auditLog_SeverityToJSON(message.severity);
    }
    if (message.request !== "") {
      obj.request = message.request;
    }
    if (message.response !== "") {
      obj.response = message.response;
    }
    if (message.status !== undefined) {
      obj.status = Status.toJSON(message.status);
    }
    return obj;
  },

  create(base?: DeepPartial<AuditLog>): AuditLog {
    return AuditLog.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<AuditLog>): AuditLog {
    const message = createBaseAuditLog();
    message.parent = object.parent ?? "";
    message.method = object.method ?? "";
    message.resource = object.resource ?? "";
    message.user = object.user ?? "";
    message.severity = object.severity ?? AuditLog_Severity.DEFAULT;
    message.request = object.request ?? "";
    message.response = object.response ?? "";
    message.status = (object.status !== undefined && object.status !== null)
      ? Status.fromPartial(object.status)
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
