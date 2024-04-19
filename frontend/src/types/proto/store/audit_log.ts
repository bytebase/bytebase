/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export interface AuditLog {
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
}

export enum AuditLog_Severity {
  DEFAULT = 0,
  DEBUG = 1,
  INFO = 2,
  NOTICE = 3,
  WARNING = 4,
  ERROR = 5,
  CRITICAL = 6,
  ALERT = 7,
  EMERGENCY = 8,
  UNRECOGNIZED = -1,
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

function createBaseAuditLog(): AuditLog {
  return { method: "", resource: "", user: "", severity: 0, request: "", response: "" };
}

export const AuditLog = {
  encode(message: AuditLog, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.method !== "") {
      writer.uint32(10).string(message.method);
    }
    if (message.resource !== "") {
      writer.uint32(18).string(message.resource);
    }
    if (message.user !== "") {
      writer.uint32(26).string(message.user);
    }
    if (message.severity !== 0) {
      writer.uint32(32).int32(message.severity);
    }
    if (message.request !== "") {
      writer.uint32(42).string(message.request);
    }
    if (message.response !== "") {
      writer.uint32(50).string(message.response);
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

          message.method = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.resource = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.user = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.severity = reader.int32() as any;
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.request = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.response = reader.string();
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
      method: isSet(object.method) ? globalThis.String(object.method) : "",
      resource: isSet(object.resource) ? globalThis.String(object.resource) : "",
      user: isSet(object.user) ? globalThis.String(object.user) : "",
      severity: isSet(object.severity) ? auditLog_SeverityFromJSON(object.severity) : 0,
      request: isSet(object.request) ? globalThis.String(object.request) : "",
      response: isSet(object.response) ? globalThis.String(object.response) : "",
    };
  },

  toJSON(message: AuditLog): unknown {
    const obj: any = {};
    if (message.method !== "") {
      obj.method = message.method;
    }
    if (message.resource !== "") {
      obj.resource = message.resource;
    }
    if (message.user !== "") {
      obj.user = message.user;
    }
    if (message.severity !== 0) {
      obj.severity = auditLog_SeverityToJSON(message.severity);
    }
    if (message.request !== "") {
      obj.request = message.request;
    }
    if (message.response !== "") {
      obj.response = message.response;
    }
    return obj;
  },

  create(base?: DeepPartial<AuditLog>): AuditLog {
    return AuditLog.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<AuditLog>): AuditLog {
    const message = createBaseAuditLog();
    message.method = object.method ?? "";
    message.resource = object.resource ?? "";
    message.user = object.user ?? "";
    message.severity = object.severity ?? 0;
    message.request = object.request ?? "";
    message.response = object.response ?? "";
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
