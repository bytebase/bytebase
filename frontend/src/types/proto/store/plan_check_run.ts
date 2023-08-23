/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { ChangedResources } from "./instance_change_history";
import Long = require("long");

export const protobufPackage = "bytebase.store";

export interface PlanCheckRunConfig {
  sheetUid: number;
  changeDatabaseType: PlanCheckRunConfig_ChangeDatabaseType;
  databaseTarget?: PlanCheckRunConfig_DatabaseTarget | undefined;
  databaseGroupTarget?: PlanCheckRunConfig_DatabaseGroupTarget | undefined;
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

export interface PlanCheckRunConfig_DatabaseTarget {
  instanceUid: number;
  databaseName: string;
}

export interface PlanCheckRunConfig_DatabaseGroupTarget {
  databaseGroupUid: number;
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
  changedResources?: ChangedResources | undefined;
}

export interface PlanCheckRunResult_Result_SqlReviewReport {
  line: number;
  column: number;
  detail: string;
  /** Code from sql review. */
  code: number;
}

function createBasePlanCheckRunConfig(): PlanCheckRunConfig {
  return { sheetUid: 0, changeDatabaseType: 0, databaseTarget: undefined, databaseGroupTarget: undefined };
}

export const PlanCheckRunConfig = {
  encode(message: PlanCheckRunConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sheetUid !== 0) {
      writer.uint32(8).int32(message.sheetUid);
    }
    if (message.changeDatabaseType !== 0) {
      writer.uint32(16).int32(message.changeDatabaseType);
    }
    if (message.databaseTarget !== undefined) {
      PlanCheckRunConfig_DatabaseTarget.encode(message.databaseTarget, writer.uint32(26).fork()).ldelim();
    }
    if (message.databaseGroupTarget !== undefined) {
      PlanCheckRunConfig_DatabaseGroupTarget.encode(message.databaseGroupTarget, writer.uint32(34).fork()).ldelim();
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

          message.sheetUid = reader.int32();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.changeDatabaseType = reader.int32() as any;
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.databaseTarget = PlanCheckRunConfig_DatabaseTarget.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.databaseGroupTarget = PlanCheckRunConfig_DatabaseGroupTarget.decode(reader, reader.uint32());
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
      sheetUid: isSet(object.sheetUid) ? Number(object.sheetUid) : 0,
      changeDatabaseType: isSet(object.changeDatabaseType)
        ? planCheckRunConfig_ChangeDatabaseTypeFromJSON(object.changeDatabaseType)
        : 0,
      databaseTarget: isSet(object.databaseTarget)
        ? PlanCheckRunConfig_DatabaseTarget.fromJSON(object.databaseTarget)
        : undefined,
      databaseGroupTarget: isSet(object.databaseGroupTarget)
        ? PlanCheckRunConfig_DatabaseGroupTarget.fromJSON(object.databaseGroupTarget)
        : undefined,
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
    if (message.databaseTarget !== undefined) {
      obj.databaseTarget = PlanCheckRunConfig_DatabaseTarget.toJSON(message.databaseTarget);
    }
    if (message.databaseGroupTarget !== undefined) {
      obj.databaseGroupTarget = PlanCheckRunConfig_DatabaseGroupTarget.toJSON(message.databaseGroupTarget);
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
    message.databaseTarget = (object.databaseTarget !== undefined && object.databaseTarget !== null)
      ? PlanCheckRunConfig_DatabaseTarget.fromPartial(object.databaseTarget)
      : undefined;
    message.databaseGroupTarget = (object.databaseGroupTarget !== undefined && object.databaseGroupTarget !== null)
      ? PlanCheckRunConfig_DatabaseGroupTarget.fromPartial(object.databaseGroupTarget)
      : undefined;
    return message;
  },
};

function createBasePlanCheckRunConfig_DatabaseTarget(): PlanCheckRunConfig_DatabaseTarget {
  return { instanceUid: 0, databaseName: "" };
}

export const PlanCheckRunConfig_DatabaseTarget = {
  encode(message: PlanCheckRunConfig_DatabaseTarget, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.instanceUid !== 0) {
      writer.uint32(8).int32(message.instanceUid);
    }
    if (message.databaseName !== "") {
      writer.uint32(18).string(message.databaseName);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRunConfig_DatabaseTarget {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRunConfig_DatabaseTarget();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.instanceUid = reader.int32();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.databaseName = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanCheckRunConfig_DatabaseTarget {
    return {
      instanceUid: isSet(object.instanceUid) ? Number(object.instanceUid) : 0,
      databaseName: isSet(object.databaseName) ? String(object.databaseName) : "",
    };
  },

  toJSON(message: PlanCheckRunConfig_DatabaseTarget): unknown {
    const obj: any = {};
    if (message.instanceUid !== 0) {
      obj.instanceUid = Math.round(message.instanceUid);
    }
    if (message.databaseName !== "") {
      obj.databaseName = message.databaseName;
    }
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRunConfig_DatabaseTarget>): PlanCheckRunConfig_DatabaseTarget {
    return PlanCheckRunConfig_DatabaseTarget.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<PlanCheckRunConfig_DatabaseTarget>): PlanCheckRunConfig_DatabaseTarget {
    const message = createBasePlanCheckRunConfig_DatabaseTarget();
    message.instanceUid = object.instanceUid ?? 0;
    message.databaseName = object.databaseName ?? "";
    return message;
  },
};

function createBasePlanCheckRunConfig_DatabaseGroupTarget(): PlanCheckRunConfig_DatabaseGroupTarget {
  return { databaseGroupUid: 0 };
}

export const PlanCheckRunConfig_DatabaseGroupTarget = {
  encode(message: PlanCheckRunConfig_DatabaseGroupTarget, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.databaseGroupUid !== 0) {
      writer.uint32(8).int64(message.databaseGroupUid);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRunConfig_DatabaseGroupTarget {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRunConfig_DatabaseGroupTarget();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.databaseGroupUid = longToNumber(reader.int64() as Long);
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanCheckRunConfig_DatabaseGroupTarget {
    return { databaseGroupUid: isSet(object.databaseGroupUid) ? Number(object.databaseGroupUid) : 0 };
  },

  toJSON(message: PlanCheckRunConfig_DatabaseGroupTarget): unknown {
    const obj: any = {};
    if (message.databaseGroupUid !== 0) {
      obj.databaseGroupUid = Math.round(message.databaseGroupUid);
    }
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRunConfig_DatabaseGroupTarget>): PlanCheckRunConfig_DatabaseGroupTarget {
    return PlanCheckRunConfig_DatabaseGroupTarget.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<PlanCheckRunConfig_DatabaseGroupTarget>): PlanCheckRunConfig_DatabaseGroupTarget {
    const message = createBasePlanCheckRunConfig_DatabaseGroupTarget();
    message.databaseGroupUid = object.databaseGroupUid ?? 0;
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
      writer.uint32(32).int64(message.code);
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

          message.code = longToNumber(reader.int64() as Long);
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
      title: isSet(object.title) ? String(object.title) : "",
      content: isSet(object.content) ? String(object.content) : "",
      code: isSet(object.code) ? Number(object.code) : 0,
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
      writer.uint32(8).int64(message.code);
    }
    for (const v of message.statementTypes) {
      writer.uint32(18).string(v!);
    }
    if (message.affectedRows !== 0) {
      writer.uint32(24).int64(message.affectedRows);
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

          message.code = longToNumber(reader.int64() as Long);
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

          message.affectedRows = longToNumber(reader.int64() as Long);
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
      code: isSet(object.code) ? Number(object.code) : 0,
      statementTypes: Array.isArray(object?.statementTypes) ? object.statementTypes.map((e: any) => String(e)) : [],
      affectedRows: isSet(object.affectedRows) ? Number(object.affectedRows) : 0,
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
      writer.uint32(8).int64(message.line);
    }
    if (message.column !== 0) {
      writer.uint32(16).int64(message.column);
    }
    if (message.detail !== "") {
      writer.uint32(26).string(message.detail);
    }
    if (message.code !== 0) {
      writer.uint32(32).int64(message.code);
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
          if (tag !== 16) {
            break;
          }

          message.column = longToNumber(reader.int64() as Long);
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
      column: isSet(object.column) ? Number(object.column) : 0,
      detail: isSet(object.detail) ? String(object.detail) : "",
      code: isSet(object.code) ? Number(object.code) : 0,
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

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
