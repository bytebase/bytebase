/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { ChangedResources } from "./instance_change_history";

export const protobufPackage = "bytebase.store";

export interface PlanCheckRunConfig {
  sheetUid: number;
  changeDatabaseType: PlanCheckRunConfig_ChangeDatabaseType;
  instanceUid: number;
  databaseName: string;
  /** database_group_uid is optional. If it's set, it means the database is part of a database group. */
  databaseGroupUid?: Long | undefined;
  ghostFlags: { [key: string]: string };
}

export enum PlanCheckRunConfig_ChangeDatabaseType {
  CHANGE_DATABASE_TYPE_UNSPECIFIED = 0,
  DDL = 1,
  DML = 2,
  SDL = 3,
  UNRECOGNIZED = -1,
}

export function planCheckRunConfig_ChangeDatabaseTypeFromJSON(object: any): PlanCheckRunConfig_ChangeDatabaseType {
  switch (object) {
    case 0:
    case "CHANGE_DATABASE_TYPE_UNSPECIFIED":
      return PlanCheckRunConfig_ChangeDatabaseType.CHANGE_DATABASE_TYPE_UNSPECIFIED;
    case 1:
    case "DDL":
      return PlanCheckRunConfig_ChangeDatabaseType.DDL;
    case 2:
    case "DML":
      return PlanCheckRunConfig_ChangeDatabaseType.DML;
    case 3:
    case "SDL":
      return PlanCheckRunConfig_ChangeDatabaseType.SDL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return PlanCheckRunConfig_ChangeDatabaseType.UNRECOGNIZED;
  }
}

export function planCheckRunConfig_ChangeDatabaseTypeToJSON(object: PlanCheckRunConfig_ChangeDatabaseType): string {
  switch (object) {
    case PlanCheckRunConfig_ChangeDatabaseType.CHANGE_DATABASE_TYPE_UNSPECIFIED:
      return "CHANGE_DATABASE_TYPE_UNSPECIFIED";
    case PlanCheckRunConfig_ChangeDatabaseType.DDL:
      return "DDL";
    case PlanCheckRunConfig_ChangeDatabaseType.DML:
      return "DML";
    case PlanCheckRunConfig_ChangeDatabaseType.SDL:
      return "SDL";
    case PlanCheckRunConfig_ChangeDatabaseType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface PlanCheckRunConfig_GhostFlagsEntry {
  key: string;
  value: string;
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
  sqlSummaryReport?: PlanCheckRunResult_Result_SqlSummaryReport | undefined;
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

export interface PlanCheckRunResult_Result_SqlSummaryReport {
  code: number;
  /** statement_types are the types of statements that are found in the sql. */
  statementTypes: string[];
  affectedRows: number;
  changedResources: ChangedResources | undefined;
}

export interface PlanCheckRunResult_Result_SqlReviewReport {
  line: number;
  column: number;
  detail: string;
  /** Code from sql review. */
  code: number;
}

function createBasePlanCheckRunConfig(): PlanCheckRunConfig {
  return {
    sheetUid: 0,
    changeDatabaseType: 0,
    instanceUid: 0,
    databaseName: "",
    databaseGroupUid: undefined,
    ghostFlags: {},
  };
}

export const PlanCheckRunConfig = {
  encode(message: PlanCheckRunConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sheetUid !== 0) {
      writer.uint32(8).int32(message.sheetUid);
    }
    if (message.changeDatabaseType !== 0) {
      writer.uint32(16).int32(message.changeDatabaseType);
    }
    if (message.instanceUid !== 0) {
      writer.uint32(24).int32(message.instanceUid);
    }
    if (message.databaseName !== "") {
      writer.uint32(34).string(message.databaseName);
    }
    if (message.databaseGroupUid !== undefined) {
      writer.uint32(40).int64(message.databaseGroupUid);
    }
    Object.entries(message.ghostFlags).forEach(([key, value]) => {
      PlanCheckRunConfig_GhostFlagsEntry.encode({ key: key as any, value }, writer.uint32(50).fork()).ldelim();
    });
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

          message.sheetUid = reader.int32();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.changeDatabaseType = reader.int32() as any;
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.instanceUid = reader.int32();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.databaseName = reader.string();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.databaseGroupUid = reader.int64() as Long;
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          const entry6 = PlanCheckRunConfig_GhostFlagsEntry.decode(reader, reader.uint32());
          if (entry6.value !== undefined) {
            message.ghostFlags[entry6.key] = entry6.value;
          }
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
      sheetUid: isSet(object.sheetUid) ? globalThis.Number(object.sheetUid) : 0,
      changeDatabaseType: isSet(object.changeDatabaseType)
        ? planCheckRunConfig_ChangeDatabaseTypeFromJSON(object.changeDatabaseType)
        : 0,
      instanceUid: isSet(object.instanceUid) ? globalThis.Number(object.instanceUid) : 0,
      databaseName: isSet(object.databaseName) ? globalThis.String(object.databaseName) : "",
      databaseGroupUid: isSet(object.databaseGroupUid) ? Long.fromValue(object.databaseGroupUid) : undefined,
      ghostFlags: isObject(object.ghostFlags)
        ? Object.entries(object.ghostFlags).reduce<{ [key: string]: string }>((acc, [key, value]) => {
          acc[key] = String(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: PlanCheckRunConfig): unknown {
    const obj: any = {};
    if (message.sheetUid !== 0) {
      obj.sheetUid = Math.round(message.sheetUid);
    }
    if (message.changeDatabaseType !== 0) {
      obj.changeDatabaseType = planCheckRunConfig_ChangeDatabaseTypeToJSON(message.changeDatabaseType);
    }
    if (message.instanceUid !== 0) {
      obj.instanceUid = Math.round(message.instanceUid);
    }
    if (message.databaseName !== "") {
      obj.databaseName = message.databaseName;
    }
    if (message.databaseGroupUid !== undefined) {
      obj.databaseGroupUid = (message.databaseGroupUid || Long.ZERO).toString();
    }
    if (message.ghostFlags) {
      const entries = Object.entries(message.ghostFlags);
      if (entries.length > 0) {
        obj.ghostFlags = {};
        entries.forEach(([k, v]) => {
          obj.ghostFlags[k] = v;
        });
      }
    }
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRunConfig>): PlanCheckRunConfig {
    return PlanCheckRunConfig.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<PlanCheckRunConfig>): PlanCheckRunConfig {
    const message = createBasePlanCheckRunConfig();
    message.sheetUid = object.sheetUid ?? 0;
    message.changeDatabaseType = object.changeDatabaseType ?? 0;
    message.instanceUid = object.instanceUid ?? 0;
    message.databaseName = object.databaseName ?? "";
    message.databaseGroupUid = (object.databaseGroupUid !== undefined && object.databaseGroupUid !== null)
      ? Long.fromValue(object.databaseGroupUid)
      : undefined;
    message.ghostFlags = Object.entries(object.ghostFlags ?? {}).reduce<{ [key: string]: string }>(
      (acc, [key, value]) => {
        if (value !== undefined) {
          acc[key] = globalThis.String(value);
        }
        return acc;
      },
      {},
    );
    return message;
  },
};

function createBasePlanCheckRunConfig_GhostFlagsEntry(): PlanCheckRunConfig_GhostFlagsEntry {
  return { key: "", value: "" };
}

export const PlanCheckRunConfig_GhostFlagsEntry = {
  encode(message: PlanCheckRunConfig_GhostFlagsEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== "") {
      writer.uint32(18).string(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRunConfig_GhostFlagsEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRunConfig_GhostFlagsEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanCheckRunConfig_GhostFlagsEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? globalThis.String(object.value) : "",
    };
  },

  toJSON(message: PlanCheckRunConfig_GhostFlagsEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== "") {
      obj.value = message.value;
    }
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRunConfig_GhostFlagsEntry>): PlanCheckRunConfig_GhostFlagsEntry {
    return PlanCheckRunConfig_GhostFlagsEntry.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<PlanCheckRunConfig_GhostFlagsEntry>): PlanCheckRunConfig_GhostFlagsEntry {
    const message = createBasePlanCheckRunConfig_GhostFlagsEntry();
    message.key = object.key ?? "";
    message.value = object.value ?? "";
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
      results: globalThis.Array.isArray(object?.results)
        ? object.results.map((e: any) => PlanCheckRunResult_Result.fromJSON(e))
        : [],
      error: isSet(object.error) ? globalThis.String(object.error) : "",
    };
  },

  toJSON(message: PlanCheckRunResult): unknown {
    const obj: any = {};
    if (message.results?.length) {
      obj.results = message.results.map((e) => PlanCheckRunResult_Result.toJSON(e));
    }
    if (message.error !== "") {
      obj.error = message.error;
    }
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
  return { status: 0, title: "", content: "", code: 0, sqlSummaryReport: undefined, sqlReviewReport: undefined };
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
      writer.uint32(32).int32(message.code);
    }
    if (message.sqlSummaryReport !== undefined) {
      PlanCheckRunResult_Result_SqlSummaryReport.encode(message.sqlSummaryReport, writer.uint32(42).fork()).ldelim();
    }
    if (message.sqlReviewReport !== undefined) {
      PlanCheckRunResult_Result_SqlReviewReport.encode(message.sqlReviewReport, writer.uint32(50).fork()).ldelim();
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

          message.code = reader.int32();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.sqlSummaryReport = PlanCheckRunResult_Result_SqlSummaryReport.decode(reader, reader.uint32());
          continue;
        case 6:
          if (tag !== 50) {
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
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      content: isSet(object.content) ? globalThis.String(object.content) : "",
      code: isSet(object.code) ? globalThis.Number(object.code) : 0,
      sqlSummaryReport: isSet(object.sqlSummaryReport)
        ? PlanCheckRunResult_Result_SqlSummaryReport.fromJSON(object.sqlSummaryReport)
        : undefined,
      sqlReviewReport: isSet(object.sqlReviewReport)
        ? PlanCheckRunResult_Result_SqlReviewReport.fromJSON(object.sqlReviewReport)
        : undefined,
    };
  },

  toJSON(message: PlanCheckRunResult_Result): unknown {
    const obj: any = {};
    if (message.status !== 0) {
      obj.status = planCheckRunResult_Result_StatusToJSON(message.status);
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.content !== "") {
      obj.content = message.content;
    }
    if (message.code !== 0) {
      obj.code = Math.round(message.code);
    }
    if (message.sqlSummaryReport !== undefined) {
      obj.sqlSummaryReport = PlanCheckRunResult_Result_SqlSummaryReport.toJSON(message.sqlSummaryReport);
    }
    if (message.sqlReviewReport !== undefined) {
      obj.sqlReviewReport = PlanCheckRunResult_Result_SqlReviewReport.toJSON(message.sqlReviewReport);
    }
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
    message.sqlSummaryReport = (object.sqlSummaryReport !== undefined && object.sqlSummaryReport !== null)
      ? PlanCheckRunResult_Result_SqlSummaryReport.fromPartial(object.sqlSummaryReport)
      : undefined;
    message.sqlReviewReport = (object.sqlReviewReport !== undefined && object.sqlReviewReport !== null)
      ? PlanCheckRunResult_Result_SqlReviewReport.fromPartial(object.sqlReviewReport)
      : undefined;
    return message;
  },
};

function createBasePlanCheckRunResult_Result_SqlSummaryReport(): PlanCheckRunResult_Result_SqlSummaryReport {
  return { code: 0, statementTypes: [], affectedRows: 0, changedResources: undefined };
}

export const PlanCheckRunResult_Result_SqlSummaryReport = {
  encode(message: PlanCheckRunResult_Result_SqlSummaryReport, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.code !== 0) {
      writer.uint32(8).int32(message.code);
    }
    for (const v of message.statementTypes) {
      writer.uint32(18).string(v!);
    }
    if (message.affectedRows !== 0) {
      writer.uint32(24).int32(message.affectedRows);
    }
    if (message.changedResources !== undefined) {
      ChangedResources.encode(message.changedResources, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRunResult_Result_SqlSummaryReport {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRunResult_Result_SqlSummaryReport();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.code = reader.int32();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.statementTypes.push(reader.string());
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.affectedRows = reader.int32();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.changedResources = ChangedResources.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanCheckRunResult_Result_SqlSummaryReport {
    return {
      code: isSet(object.code) ? globalThis.Number(object.code) : 0,
      statementTypes: globalThis.Array.isArray(object?.statementTypes)
        ? object.statementTypes.map((e: any) => globalThis.String(e))
        : [],
      affectedRows: isSet(object.affectedRows) ? globalThis.Number(object.affectedRows) : 0,
      changedResources: isSet(object.changedResources) ? ChangedResources.fromJSON(object.changedResources) : undefined,
    };
  },

  toJSON(message: PlanCheckRunResult_Result_SqlSummaryReport): unknown {
    const obj: any = {};
    if (message.code !== 0) {
      obj.code = Math.round(message.code);
    }
    if (message.statementTypes?.length) {
      obj.statementTypes = message.statementTypes;
    }
    if (message.affectedRows !== 0) {
      obj.affectedRows = Math.round(message.affectedRows);
    }
    if (message.changedResources !== undefined) {
      obj.changedResources = ChangedResources.toJSON(message.changedResources);
    }
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRunResult_Result_SqlSummaryReport>): PlanCheckRunResult_Result_SqlSummaryReport {
    return PlanCheckRunResult_Result_SqlSummaryReport.fromPartial(base ?? {});
  },
  fromPartial(
    object: DeepPartial<PlanCheckRunResult_Result_SqlSummaryReport>,
  ): PlanCheckRunResult_Result_SqlSummaryReport {
    const message = createBasePlanCheckRunResult_Result_SqlSummaryReport();
    message.code = object.code ?? 0;
    message.statementTypes = object.statementTypes?.map((e) => e) || [];
    message.affectedRows = object.affectedRows ?? 0;
    message.changedResources = (object.changedResources !== undefined && object.changedResources !== null)
      ? ChangedResources.fromPartial(object.changedResources)
      : undefined;
    return message;
  },
};

function createBasePlanCheckRunResult_Result_SqlReviewReport(): PlanCheckRunResult_Result_SqlReviewReport {
  return { line: 0, column: 0, detail: "", code: 0 };
}

export const PlanCheckRunResult_Result_SqlReviewReport = {
  encode(message: PlanCheckRunResult_Result_SqlReviewReport, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.line !== 0) {
      writer.uint32(8).int32(message.line);
    }
    if (message.column !== 0) {
      writer.uint32(16).int32(message.column);
    }
    if (message.detail !== "") {
      writer.uint32(26).string(message.detail);
    }
    if (message.code !== 0) {
      writer.uint32(32).int32(message.code);
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

          message.line = reader.int32();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.column = reader.int32();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.detail = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.code = reader.int32();
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
      line: isSet(object.line) ? globalThis.Number(object.line) : 0,
      column: isSet(object.column) ? globalThis.Number(object.column) : 0,
      detail: isSet(object.detail) ? globalThis.String(object.detail) : "",
      code: isSet(object.code) ? globalThis.Number(object.code) : 0,
    };
  },

  toJSON(message: PlanCheckRunResult_Result_SqlReviewReport): unknown {
    const obj: any = {};
    if (message.line !== 0) {
      obj.line = Math.round(message.line);
    }
    if (message.column !== 0) {
      obj.column = Math.round(message.column);
    }
    if (message.detail !== "") {
      obj.detail = message.detail;
    }
    if (message.code !== 0) {
      obj.code = Math.round(message.code);
    }
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
    message.column = object.column ?? 0;
    message.detail = object.detail ?? "";
    message.code = object.code ?? 0;
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

function isObject(value: any): boolean {
  return typeof value === "object" && value !== null;
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
