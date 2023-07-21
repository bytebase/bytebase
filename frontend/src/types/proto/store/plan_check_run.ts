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
  error: string;
}

export interface PlanCheckRunResult_Result {
  status: PlanCheckRunResult_Result_Status;
  title: string;
  content: string;
  code: number;
  statementTypeReport?: PlanCheckRunResult_Result_StatementTypeReport | undefined;
  affectedRowsReport?: PlanCheckRunResult_Result_AffectedRowsReport | undefined;
  sqlReviewReport?: PlanCheckRunResult_Result_SqlReviewReport | undefined;
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

export interface PlanCheckRunResult_Result_StatementTypeReport {
  statementType: string;
}

export interface PlanCheckRunResult_Result_AffectedRowsReport {
  affectedRows: number;
}

export interface PlanCheckRunResult_Result_SqlReviewReport {
  line: number;
  detail: string;
  /** Code from sql review. */
  code: number;
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
  return { results: [], error: "" };
}

export const PlanCheckRunResult = {
  encode(message: PlanCheckRunResult, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.results) {
      PlanCheckRunResult_Result.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.error !== "") {
      writer.uint32(18).string(message.error);
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
        case 2:
          if (tag !== 18) {
            break;
          }

          message.error = reader.string();
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
      error: isSet(object.error) ? String(object.error) : "",
    };
  },

  toJSON(message: PlanCheckRunResult): unknown {
    const obj: any = {};
    if (message.results) {
      obj.results = message.results.map((e) => e ? PlanCheckRunResult_Result.toJSON(e) : undefined);
    } else {
      obj.results = [];
    }
    message.error !== undefined && (obj.error = message.error);
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRunResult>): PlanCheckRunResult {
    return PlanCheckRunResult.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanCheckRunResult>): PlanCheckRunResult {
    const message = createBasePlanCheckRunResult();
    message.results = object.results?.map((e) => PlanCheckRunResult_Result.fromPartial(e)) || [];
    message.error = object.error ?? "";
    return message;
  },
};

function createBasePlanCheckRunResult_Result(): PlanCheckRunResult_Result {
  return {
    status: 0,
    title: "",
    content: "",
    code: 0,
    statementTypeReport: undefined,
    affectedRowsReport: undefined,
    sqlReviewReport: undefined,
  };
}

export const PlanCheckRunResult_Result = {
  encode(message: PlanCheckRunResult_Result, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.status !== 0) {
      writer.uint32(8).int32(message.status);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.content !== "") {
      writer.uint32(26).string(message.content);
    }
    if (message.code !== 0) {
      writer.uint32(32).int64(message.code);
    }
    if (message.statementTypeReport !== undefined) {
      PlanCheckRunResult_Result_StatementTypeReport.encode(message.statementTypeReport, writer.uint32(42).fork())
        .ldelim();
    }
    if (message.affectedRowsReport !== undefined) {
      PlanCheckRunResult_Result_AffectedRowsReport.encode(message.affectedRowsReport, writer.uint32(50).fork())
        .ldelim();
    }
    if (message.sqlReviewReport !== undefined) {
      PlanCheckRunResult_Result_SqlReviewReport.encode(message.sqlReviewReport, writer.uint32(58).fork()).ldelim();
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

          message.status = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.title = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.content = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.code = longToNumber(reader.int64() as Long);
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.statementTypeReport = PlanCheckRunResult_Result_StatementTypeReport.decode(reader, reader.uint32());
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.affectedRowsReport = PlanCheckRunResult_Result_AffectedRowsReport.decode(reader, reader.uint32());
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.sqlReviewReport = PlanCheckRunResult_Result_SqlReviewReport.decode(reader, reader.uint32());
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
      status: isSet(object.status) ? planCheckRunResult_Result_StatusFromJSON(object.status) : 0,
      title: isSet(object.title) ? String(object.title) : "",
      content: isSet(object.content) ? String(object.content) : "",
      code: isSet(object.code) ? Number(object.code) : 0,
      statementTypeReport: isSet(object.statementTypeReport)
        ? PlanCheckRunResult_Result_StatementTypeReport.fromJSON(object.statementTypeReport)
        : undefined,
      affectedRowsReport: isSet(object.affectedRowsReport)
        ? PlanCheckRunResult_Result_AffectedRowsReport.fromJSON(object.affectedRowsReport)
        : undefined,
      sqlReviewReport: isSet(object.sqlReviewReport)
        ? PlanCheckRunResult_Result_SqlReviewReport.fromJSON(object.sqlReviewReport)
        : undefined,
    };
  },

  toJSON(message: PlanCheckRunResult_Result): unknown {
    const obj: any = {};
    message.status !== undefined && (obj.status = planCheckRunResult_Result_StatusToJSON(message.status));
    message.title !== undefined && (obj.title = message.title);
    message.content !== undefined && (obj.content = message.content);
    message.code !== undefined && (obj.code = Math.round(message.code));
    message.statementTypeReport !== undefined && (obj.statementTypeReport = message.statementTypeReport
      ? PlanCheckRunResult_Result_StatementTypeReport.toJSON(message.statementTypeReport)
      : undefined);
    message.affectedRowsReport !== undefined && (obj.affectedRowsReport = message.affectedRowsReport
      ? PlanCheckRunResult_Result_AffectedRowsReport.toJSON(message.affectedRowsReport)
      : undefined);
    message.sqlReviewReport !== undefined && (obj.sqlReviewReport = message.sqlReviewReport
      ? PlanCheckRunResult_Result_SqlReviewReport.toJSON(message.sqlReviewReport)
      : undefined);
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRunResult_Result>): PlanCheckRunResult_Result {
    return PlanCheckRunResult_Result.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanCheckRunResult_Result>): PlanCheckRunResult_Result {
    const message = createBasePlanCheckRunResult_Result();
    message.status = object.status ?? 0;
    message.title = object.title ?? "";
    message.content = object.content ?? "";
    message.code = object.code ?? 0;
    message.statementTypeReport = (object.statementTypeReport !== undefined && object.statementTypeReport !== null)
      ? PlanCheckRunResult_Result_StatementTypeReport.fromPartial(object.statementTypeReport)
      : undefined;
    message.affectedRowsReport = (object.affectedRowsReport !== undefined && object.affectedRowsReport !== null)
      ? PlanCheckRunResult_Result_AffectedRowsReport.fromPartial(object.affectedRowsReport)
      : undefined;
    message.sqlReviewReport = (object.sqlReviewReport !== undefined && object.sqlReviewReport !== null)
      ? PlanCheckRunResult_Result_SqlReviewReport.fromPartial(object.sqlReviewReport)
      : undefined;
    return message;
  },
};

function createBasePlanCheckRunResult_Result_StatementTypeReport(): PlanCheckRunResult_Result_StatementTypeReport {
  return { statementType: "" };
}

export const PlanCheckRunResult_Result_StatementTypeReport = {
  encode(message: PlanCheckRunResult_Result_StatementTypeReport, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.statementType !== "") {
      writer.uint32(10).string(message.statementType);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRunResult_Result_StatementTypeReport {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRunResult_Result_StatementTypeReport();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.statementType = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanCheckRunResult_Result_StatementTypeReport {
    return { statementType: isSet(object.statementType) ? String(object.statementType) : "" };
  },

  toJSON(message: PlanCheckRunResult_Result_StatementTypeReport): unknown {
    const obj: any = {};
    message.statementType !== undefined && (obj.statementType = message.statementType);
    return obj;
  },

  create(
    base?: DeepPartial<PlanCheckRunResult_Result_StatementTypeReport>,
  ): PlanCheckRunResult_Result_StatementTypeReport {
    return PlanCheckRunResult_Result_StatementTypeReport.fromPartial(base ?? {});
  },

  fromPartial(
    object: DeepPartial<PlanCheckRunResult_Result_StatementTypeReport>,
  ): PlanCheckRunResult_Result_StatementTypeReport {
    const message = createBasePlanCheckRunResult_Result_StatementTypeReport();
    message.statementType = object.statementType ?? "";
    return message;
  },
};

function createBasePlanCheckRunResult_Result_AffectedRowsReport(): PlanCheckRunResult_Result_AffectedRowsReport {
  return { affectedRows: 0 };
}

export const PlanCheckRunResult_Result_AffectedRowsReport = {
  encode(message: PlanCheckRunResult_Result_AffectedRowsReport, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.affectedRows !== 0) {
      writer.uint32(8).int64(message.affectedRows);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRunResult_Result_AffectedRowsReport {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRunResult_Result_AffectedRowsReport();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.affectedRows = longToNumber(reader.int64() as Long);
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanCheckRunResult_Result_AffectedRowsReport {
    return { affectedRows: isSet(object.affectedRows) ? Number(object.affectedRows) : 0 };
  },

  toJSON(message: PlanCheckRunResult_Result_AffectedRowsReport): unknown {
    const obj: any = {};
    message.affectedRows !== undefined && (obj.affectedRows = Math.round(message.affectedRows));
    return obj;
  },

  create(
    base?: DeepPartial<PlanCheckRunResult_Result_AffectedRowsReport>,
  ): PlanCheckRunResult_Result_AffectedRowsReport {
    return PlanCheckRunResult_Result_AffectedRowsReport.fromPartial(base ?? {});
  },

  fromPartial(
    object: DeepPartial<PlanCheckRunResult_Result_AffectedRowsReport>,
  ): PlanCheckRunResult_Result_AffectedRowsReport {
    const message = createBasePlanCheckRunResult_Result_AffectedRowsReport();
    message.affectedRows = object.affectedRows ?? 0;
    return message;
  },
};

function createBasePlanCheckRunResult_Result_SqlReviewReport(): PlanCheckRunResult_Result_SqlReviewReport {
  return { line: 0, detail: "", code: 0 };
}

export const PlanCheckRunResult_Result_SqlReviewReport = {
  encode(message: PlanCheckRunResult_Result_SqlReviewReport, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.line !== 0) {
      writer.uint32(8).int64(message.line);
    }
    if (message.detail !== "") {
      writer.uint32(18).string(message.detail);
    }
    if (message.code !== 0) {
      writer.uint32(24).int64(message.code);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRunResult_Result_SqlReviewReport {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRunResult_Result_SqlReviewReport();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.line = longToNumber(reader.int64() as Long);
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.detail = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.code = longToNumber(reader.int64() as Long);
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanCheckRunResult_Result_SqlReviewReport {
    return {
      line: isSet(object.line) ? Number(object.line) : 0,
      detail: isSet(object.detail) ? String(object.detail) : "",
      code: isSet(object.code) ? Number(object.code) : 0,
    };
  },

  toJSON(message: PlanCheckRunResult_Result_SqlReviewReport): unknown {
    const obj: any = {};
    message.line !== undefined && (obj.line = Math.round(message.line));
    message.detail !== undefined && (obj.detail = message.detail);
    message.code !== undefined && (obj.code = Math.round(message.code));
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRunResult_Result_SqlReviewReport>): PlanCheckRunResult_Result_SqlReviewReport {
    return PlanCheckRunResult_Result_SqlReviewReport.fromPartial(base ?? {});
  },

  fromPartial(
    object: DeepPartial<PlanCheckRunResult_Result_SqlReviewReport>,
  ): PlanCheckRunResult_Result_SqlReviewReport {
    const message = createBasePlanCheckRunResult_Result_SqlReviewReport();
    message.line = object.line ?? 0;
    message.detail = object.detail ?? "";
    message.code = object.code ?? 0;
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
