/* eslint-disable */
import * as Long from "long";
import * as _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export interface PlanCheckRunConfig {
  sheetId: number;
  databaseId: number;
}

export interface PlanCheckRunResult {
  results: PlanCheckRunResult_Result[];
}

export interface PlanCheckRunResult_Result {
  namespace: PlanCheckRunResult_Result_Namespace;
  code: number;
  status: PlanCheckRunResult_Result_Status;
  title: string;
  content: string;
  line: number;
  detail: string;
}

export enum PlanCheckRunResult_Result_Namespace {
  NAMESPACE_UNSPECIFIED = 0,
  BYTEBASE = 1,
  ADVISOR = 2,
  UNRECOGNIZED = -1,
}

export function planCheckRunResult_Result_NamespaceFromJSON(object: any): PlanCheckRunResult_Result_Namespace {
  switch (object) {
    case 0:
    case "NAMESPACE_UNSPECIFIED":
      return PlanCheckRunResult_Result_Namespace.NAMESPACE_UNSPECIFIED;
    case 1:
    case "BYTEBASE":
      return PlanCheckRunResult_Result_Namespace.BYTEBASE;
    case 2:
    case "ADVISOR":
      return PlanCheckRunResult_Result_Namespace.ADVISOR;
    case -1:
    case "UNRECOGNIZED":
    default:
      return PlanCheckRunResult_Result_Namespace.UNRECOGNIZED;
  }
}

export function planCheckRunResult_Result_NamespaceToJSON(object: PlanCheckRunResult_Result_Namespace): string {
  switch (object) {
    case PlanCheckRunResult_Result_Namespace.NAMESPACE_UNSPECIFIED:
      return "NAMESPACE_UNSPECIFIED";
    case PlanCheckRunResult_Result_Namespace.BYTEBASE:
      return "BYTEBASE";
    case PlanCheckRunResult_Result_Namespace.ADVISOR:
      return "ADVISOR";
    case PlanCheckRunResult_Result_Namespace.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum PlanCheckRunResult_Result_Status {
  STATUS_UNSPECIFIED = 0,
  ERROR = 1,
  WARNING = 2,
  SUCCESS = 3,
  UNRECOGNIZED = -1,
}

export function planCheckRunResult_Result_StatusFromJSON(object: any): PlanCheckRunResult_Result_Status {
  switch (object) {
    case 0:
    case "STATUS_UNSPECIFIED":
      return PlanCheckRunResult_Result_Status.STATUS_UNSPECIFIED;
    case 1:
    case "ERROR":
      return PlanCheckRunResult_Result_Status.ERROR;
    case 2:
    case "WARNING":
      return PlanCheckRunResult_Result_Status.WARNING;
    case 3:
    case "SUCCESS":
      return PlanCheckRunResult_Result_Status.SUCCESS;
    case -1:
    case "UNRECOGNIZED":
    default:
      return PlanCheckRunResult_Result_Status.UNRECOGNIZED;
  }
}

export function planCheckRunResult_Result_StatusToJSON(object: PlanCheckRunResult_Result_Status): string {
  switch (object) {
    case PlanCheckRunResult_Result_Status.STATUS_UNSPECIFIED:
      return "STATUS_UNSPECIFIED";
    case PlanCheckRunResult_Result_Status.ERROR:
      return "ERROR";
    case PlanCheckRunResult_Result_Status.WARNING:
      return "WARNING";
    case PlanCheckRunResult_Result_Status.SUCCESS:
      return "SUCCESS";
    case PlanCheckRunResult_Result_Status.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

function createBasePlanCheckRunConfig(): PlanCheckRunConfig {
  return { sheetId: 0, databaseId: 0 };
}

export const PlanCheckRunConfig = {
  encode(message: PlanCheckRunConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sheetId !== 0) {
      writer.uint32(8).int32(message.sheetId);
    }
    if (message.databaseId !== 0) {
      writer.uint32(16).int32(message.databaseId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRunConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRunConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.sheetId = reader.int32();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.databaseId = reader.int32();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanCheckRunConfig {
    return {
      sheetId: isSet(object.sheetId) ? Number(object.sheetId) : 0,
      databaseId: isSet(object.databaseId) ? Number(object.databaseId) : 0,
    };
  },

  toJSON(message: PlanCheckRunConfig): unknown {
    const obj: any = {};
    message.sheetId !== undefined && (obj.sheetId = Math.round(message.sheetId));
    message.databaseId !== undefined && (obj.databaseId = Math.round(message.databaseId));
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRunConfig>): PlanCheckRunConfig {
    return PlanCheckRunConfig.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanCheckRunConfig>): PlanCheckRunConfig {
    const message = createBasePlanCheckRunConfig();
    message.sheetId = object.sheetId ?? 0;
    message.databaseId = object.databaseId ?? 0;
    return message;
  },
};

function createBasePlanCheckRunResult(): PlanCheckRunResult {
  return { results: [] };
}

export const PlanCheckRunResult = {
  encode(message: PlanCheckRunResult, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.results) {
      PlanCheckRunResult_Result.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRunResult {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRunResult();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.results.push(PlanCheckRunResult_Result.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanCheckRunResult {
    return {
      results: Array.isArray(object?.results)
        ? object.results.map((e: any) => PlanCheckRunResult_Result.fromJSON(e))
        : [],
    };
  },

  toJSON(message: PlanCheckRunResult): unknown {
    const obj: any = {};
    if (message.results) {
      obj.results = message.results.map((e) => e ? PlanCheckRunResult_Result.toJSON(e) : undefined);
    } else {
      obj.results = [];
    }
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRunResult>): PlanCheckRunResult {
    return PlanCheckRunResult.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanCheckRunResult>): PlanCheckRunResult {
    const message = createBasePlanCheckRunResult();
    message.results = object.results?.map((e) => PlanCheckRunResult_Result.fromPartial(e)) || [];
    return message;
  },
};

function createBasePlanCheckRunResult_Result(): PlanCheckRunResult_Result {
  return { namespace: 0, code: 0, status: 0, title: "", content: "", line: 0, detail: "" };
}

export const PlanCheckRunResult_Result = {
  encode(message: PlanCheckRunResult_Result, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.namespace !== 0) {
      writer.uint32(8).int32(message.namespace);
    }
    if (message.code !== 0) {
      writer.uint32(16).int64(message.code);
    }
    if (message.status !== 0) {
      writer.uint32(24).int32(message.status);
    }
    if (message.title !== "") {
      writer.uint32(34).string(message.title);
    }
    if (message.content !== "") {
      writer.uint32(42).string(message.content);
    }
    if (message.line !== 0) {
      writer.uint32(48).int64(message.line);
    }
    if (message.detail !== "") {
      writer.uint32(58).string(message.detail);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRunResult_Result {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRunResult_Result();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.namespace = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.code = longToNumber(reader.int64() as Long);
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.status = reader.int32() as any;
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.title = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.content = reader.string();
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.line = longToNumber(reader.int64() as Long);
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.detail = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanCheckRunResult_Result {
    return {
      namespace: isSet(object.namespace) ? planCheckRunResult_Result_NamespaceFromJSON(object.namespace) : 0,
      code: isSet(object.code) ? Number(object.code) : 0,
      status: isSet(object.status) ? planCheckRunResult_Result_StatusFromJSON(object.status) : 0,
      title: isSet(object.title) ? String(object.title) : "",
      content: isSet(object.content) ? String(object.content) : "",
      line: isSet(object.line) ? Number(object.line) : 0,
      detail: isSet(object.detail) ? String(object.detail) : "",
    };
  },

  toJSON(message: PlanCheckRunResult_Result): unknown {
    const obj: any = {};
    message.namespace !== undefined && (obj.namespace = planCheckRunResult_Result_NamespaceToJSON(message.namespace));
    message.code !== undefined && (obj.code = Math.round(message.code));
    message.status !== undefined && (obj.status = planCheckRunResult_Result_StatusToJSON(message.status));
    message.title !== undefined && (obj.title = message.title);
    message.content !== undefined && (obj.content = message.content);
    message.line !== undefined && (obj.line = Math.round(message.line));
    message.detail !== undefined && (obj.detail = message.detail);
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRunResult_Result>): PlanCheckRunResult_Result {
    return PlanCheckRunResult_Result.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanCheckRunResult_Result>): PlanCheckRunResult_Result {
    const message = createBasePlanCheckRunResult_Result();
    message.namespace = object.namespace ?? 0;
    message.code = object.code ?? 0;
    message.status = object.status ?? 0;
    message.title = object.title ?? "";
    message.content = object.content ?? "";
    message.line = object.line ?? 0;
    message.detail = object.detail ?? "";
    return message;
  },
};

declare const self: any | undefined;
declare const window: any | undefined;
declare const global: any | undefined;
const tsProtoGlobalThis: any = (() => {
  if (typeof globalThis !== "undefined") {
    return globalThis;
  }
  if (typeof self !== "undefined") {
    return self;
  }
  if (typeof window !== "undefined") {
    return window;
  }
  if (typeof global !== "undefined") {
    return global;
  }
  throw "Unable to locate global object";
})();

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function longToNumber(long: Long): number {
  if (long.gt(Number.MAX_SAFE_INTEGER)) {
    throw new tsProtoGlobalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
  }
  return long.toNumber();
}

// If you get a compile-error about 'Constructor<Long> and ... have no overlap',
// add '--ts_proto_opt=esModuleInterop=true' as a flag when calling 'protoc'.
if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
