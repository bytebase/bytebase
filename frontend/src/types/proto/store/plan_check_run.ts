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
  success?:
    | PlanCheckRunResultSuccess
    | undefined;
  /** Failure if the PlanCheckRun itself failed to run. */
  failure?: PlanCheckRunResultFailure | undefined;
}

export interface PlanCheckRunResultSuccess {
  status: PlanCheckRunResultSuccess_Status;
  title: string;
  content: string;
  code?: PlanCheckRunResultSuccess_Code | undefined;
  statementTypeReport?: PlanCheckRunResultSuccess_StatementTypeReport | undefined;
  affectedRowsReport?: PlanCheckRunResultSuccess_AffectedRowsReport | undefined;
  sqlReviewReport?: PlanCheckRunResultSuccess_SqlReviewReport | undefined;
  generalReport?: PlanCheckRunResultSuccess_GeneralReport | undefined;
}

export enum PlanCheckRunResultSuccess_Status {
  STATUS_UNSPECIFIED = 0,
  ERROR = 1,
  WARNING = 2,
  SUCCESS = 3,
  UNRECOGNIZED = -1,
}

export function planCheckRunResultSuccess_StatusFromJSON(object: any): PlanCheckRunResultSuccess_Status {
  switch (object) {
    case 0:
    case "STATUS_UNSPECIFIED":
      return PlanCheckRunResultSuccess_Status.STATUS_UNSPECIFIED;
    case 1:
    case "ERROR":
      return PlanCheckRunResultSuccess_Status.ERROR;
    case 2:
    case "WARNING":
      return PlanCheckRunResultSuccess_Status.WARNING;
    case 3:
    case "SUCCESS":
      return PlanCheckRunResultSuccess_Status.SUCCESS;
    case -1:
    case "UNRECOGNIZED":
    default:
      return PlanCheckRunResultSuccess_Status.UNRECOGNIZED;
  }
}

export function planCheckRunResultSuccess_StatusToJSON(object: PlanCheckRunResultSuccess_Status): string {
  switch (object) {
    case PlanCheckRunResultSuccess_Status.STATUS_UNSPECIFIED:
      return "STATUS_UNSPECIFIED";
    case PlanCheckRunResultSuccess_Status.ERROR:
      return "ERROR";
    case PlanCheckRunResultSuccess_Status.WARNING:
      return "WARNING";
    case PlanCheckRunResultSuccess_Status.SUCCESS:
      return "SUCCESS";
    case PlanCheckRunResultSuccess_Status.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface PlanCheckRunResultSuccess_Code {
  code: number;
  namespace: PlanCheckRunResultSuccess_Code_Namespace;
}

export enum PlanCheckRunResultSuccess_Code_Namespace {
  NAMESPACE_UNSPECIFIED = 0,
  BYTEBASE = 1,
  ADVISOR = 2,
  UNRECOGNIZED = -1,
}

export function planCheckRunResultSuccess_Code_NamespaceFromJSON(
  object: any,
): PlanCheckRunResultSuccess_Code_Namespace {
  switch (object) {
    case 0:
    case "NAMESPACE_UNSPECIFIED":
      return PlanCheckRunResultSuccess_Code_Namespace.NAMESPACE_UNSPECIFIED;
    case 1:
    case "BYTEBASE":
      return PlanCheckRunResultSuccess_Code_Namespace.BYTEBASE;
    case 2:
    case "ADVISOR":
      return PlanCheckRunResultSuccess_Code_Namespace.ADVISOR;
    case -1:
    case "UNRECOGNIZED":
    default:
      return PlanCheckRunResultSuccess_Code_Namespace.UNRECOGNIZED;
  }
}

export function planCheckRunResultSuccess_Code_NamespaceToJSON(
  object: PlanCheckRunResultSuccess_Code_Namespace,
): string {
  switch (object) {
    case PlanCheckRunResultSuccess_Code_Namespace.NAMESPACE_UNSPECIFIED:
      return "NAMESPACE_UNSPECIFIED";
    case PlanCheckRunResultSuccess_Code_Namespace.BYTEBASE:
      return "BYTEBASE";
    case PlanCheckRunResultSuccess_Code_Namespace.ADVISOR:
      return "ADVISOR";
    case PlanCheckRunResultSuccess_Code_Namespace.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface PlanCheckRunResultSuccess_StatementTypeReport {
  statementType: string;
}

export interface PlanCheckRunResultSuccess_AffectedRowsReport {
  affectedRows: number;
}

export interface PlanCheckRunResultSuccess_SqlReviewReport {
  line: number;
  detail: string;
}

export interface PlanCheckRunResultSuccess_GeneralReport {
}

export interface PlanCheckRunResultFailure {
  title: string;
  content: string;
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
  return { success: undefined, failure: undefined };
}

export const PlanCheckRunResult_Result = {
  encode(message: PlanCheckRunResult_Result, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.success !== undefined) {
      PlanCheckRunResultSuccess.encode(message.success, writer.uint32(10).fork()).ldelim();
    }
    if (message.failure !== undefined) {
      PlanCheckRunResultFailure.encode(message.failure, writer.uint32(18).fork()).ldelim();
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
          if (tag !== 10) {
            break;
          }

          message.success = PlanCheckRunResultSuccess.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.failure = PlanCheckRunResultFailure.decode(reader, reader.uint32());
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
      success: isSet(object.success) ? PlanCheckRunResultSuccess.fromJSON(object.success) : undefined,
      failure: isSet(object.failure) ? PlanCheckRunResultFailure.fromJSON(object.failure) : undefined,
    };
  },

  toJSON(message: PlanCheckRunResult_Result): unknown {
    const obj: any = {};
    message.success !== undefined &&
      (obj.success = message.success ? PlanCheckRunResultSuccess.toJSON(message.success) : undefined);
    message.failure !== undefined &&
      (obj.failure = message.failure ? PlanCheckRunResultFailure.toJSON(message.failure) : undefined);
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRunResult_Result>): PlanCheckRunResult_Result {
    return PlanCheckRunResult_Result.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanCheckRunResult_Result>): PlanCheckRunResult_Result {
    const message = createBasePlanCheckRunResult_Result();
    message.success = (object.success !== undefined && object.success !== null)
      ? PlanCheckRunResultSuccess.fromPartial(object.success)
      : undefined;
    message.failure = (object.failure !== undefined && object.failure !== null)
      ? PlanCheckRunResultFailure.fromPartial(object.failure)
      : undefined;
    return message;
  },
};

function createBasePlanCheckRunResultSuccess(): PlanCheckRunResultSuccess {
  return {
    status: 0,
    title: "",
    content: "",
    code: undefined,
    statementTypeReport: undefined,
    affectedRowsReport: undefined,
    sqlReviewReport: undefined,
    generalReport: undefined,
  };
}

export const PlanCheckRunResultSuccess = {
  encode(message: PlanCheckRunResultSuccess, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.status !== 0) {
      writer.uint32(8).int32(message.status);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.content !== "") {
      writer.uint32(26).string(message.content);
    }
    if (message.code !== undefined) {
      PlanCheckRunResultSuccess_Code.encode(message.code, writer.uint32(34).fork()).ldelim();
    }
    if (message.statementTypeReport !== undefined) {
      PlanCheckRunResultSuccess_StatementTypeReport.encode(message.statementTypeReport, writer.uint32(42).fork())
        .ldelim();
    }
    if (message.affectedRowsReport !== undefined) {
      PlanCheckRunResultSuccess_AffectedRowsReport.encode(message.affectedRowsReport, writer.uint32(50).fork())
        .ldelim();
    }
    if (message.sqlReviewReport !== undefined) {
      PlanCheckRunResultSuccess_SqlReviewReport.encode(message.sqlReviewReport, writer.uint32(58).fork()).ldelim();
    }
    if (message.generalReport !== undefined) {
      PlanCheckRunResultSuccess_GeneralReport.encode(message.generalReport, writer.uint32(66).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRunResultSuccess {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRunResultSuccess();
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
          if (tag !== 34) {
            break;
          }

          message.code = PlanCheckRunResultSuccess_Code.decode(reader, reader.uint32());
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.statementTypeReport = PlanCheckRunResultSuccess_StatementTypeReport.decode(reader, reader.uint32());
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.affectedRowsReport = PlanCheckRunResultSuccess_AffectedRowsReport.decode(reader, reader.uint32());
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.sqlReviewReport = PlanCheckRunResultSuccess_SqlReviewReport.decode(reader, reader.uint32());
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.generalReport = PlanCheckRunResultSuccess_GeneralReport.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanCheckRunResultSuccess {
    return {
      status: isSet(object.status) ? planCheckRunResultSuccess_StatusFromJSON(object.status) : 0,
      title: isSet(object.title) ? String(object.title) : "",
      content: isSet(object.content) ? String(object.content) : "",
      code: isSet(object.code) ? PlanCheckRunResultSuccess_Code.fromJSON(object.code) : undefined,
      statementTypeReport: isSet(object.statementTypeReport)
        ? PlanCheckRunResultSuccess_StatementTypeReport.fromJSON(object.statementTypeReport)
        : undefined,
      affectedRowsReport: isSet(object.affectedRowsReport)
        ? PlanCheckRunResultSuccess_AffectedRowsReport.fromJSON(object.affectedRowsReport)
        : undefined,
      sqlReviewReport: isSet(object.sqlReviewReport)
        ? PlanCheckRunResultSuccess_SqlReviewReport.fromJSON(object.sqlReviewReport)
        : undefined,
      generalReport: isSet(object.generalReport)
        ? PlanCheckRunResultSuccess_GeneralReport.fromJSON(object.generalReport)
        : undefined,
    };
  },

  toJSON(message: PlanCheckRunResultSuccess): unknown {
    const obj: any = {};
    message.status !== undefined && (obj.status = planCheckRunResultSuccess_StatusToJSON(message.status));
    message.title !== undefined && (obj.title = message.title);
    message.content !== undefined && (obj.content = message.content);
    message.code !== undefined &&
      (obj.code = message.code ? PlanCheckRunResultSuccess_Code.toJSON(message.code) : undefined);
    message.statementTypeReport !== undefined && (obj.statementTypeReport = message.statementTypeReport
      ? PlanCheckRunResultSuccess_StatementTypeReport.toJSON(message.statementTypeReport)
      : undefined);
    message.affectedRowsReport !== undefined && (obj.affectedRowsReport = message.affectedRowsReport
      ? PlanCheckRunResultSuccess_AffectedRowsReport.toJSON(message.affectedRowsReport)
      : undefined);
    message.sqlReviewReport !== undefined && (obj.sqlReviewReport = message.sqlReviewReport
      ? PlanCheckRunResultSuccess_SqlReviewReport.toJSON(message.sqlReviewReport)
      : undefined);
    message.generalReport !== undefined && (obj.generalReport = message.generalReport
      ? PlanCheckRunResultSuccess_GeneralReport.toJSON(message.generalReport)
      : undefined);
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRunResultSuccess>): PlanCheckRunResultSuccess {
    return PlanCheckRunResultSuccess.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanCheckRunResultSuccess>): PlanCheckRunResultSuccess {
    const message = createBasePlanCheckRunResultSuccess();
    message.status = object.status ?? 0;
    message.title = object.title ?? "";
    message.content = object.content ?? "";
    message.code = (object.code !== undefined && object.code !== null)
      ? PlanCheckRunResultSuccess_Code.fromPartial(object.code)
      : undefined;
    message.statementTypeReport = (object.statementTypeReport !== undefined && object.statementTypeReport !== null)
      ? PlanCheckRunResultSuccess_StatementTypeReport.fromPartial(object.statementTypeReport)
      : undefined;
    message.affectedRowsReport = (object.affectedRowsReport !== undefined && object.affectedRowsReport !== null)
      ? PlanCheckRunResultSuccess_AffectedRowsReport.fromPartial(object.affectedRowsReport)
      : undefined;
    message.sqlReviewReport = (object.sqlReviewReport !== undefined && object.sqlReviewReport !== null)
      ? PlanCheckRunResultSuccess_SqlReviewReport.fromPartial(object.sqlReviewReport)
      : undefined;
    message.generalReport = (object.generalReport !== undefined && object.generalReport !== null)
      ? PlanCheckRunResultSuccess_GeneralReport.fromPartial(object.generalReport)
      : undefined;
    return message;
  },
};

function createBasePlanCheckRunResultSuccess_Code(): PlanCheckRunResultSuccess_Code {
  return { code: 0, namespace: 0 };
}

export const PlanCheckRunResultSuccess_Code = {
  encode(message: PlanCheckRunResultSuccess_Code, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.code !== 0) {
      writer.uint32(8).int64(message.code);
    }
    if (message.namespace !== 0) {
      writer.uint32(16).int32(message.namespace);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRunResultSuccess_Code {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRunResultSuccess_Code();
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
          if (tag !== 16) {
            break;
          }

          message.namespace = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanCheckRunResultSuccess_Code {
    return {
      code: isSet(object.code) ? Number(object.code) : 0,
      namespace: isSet(object.namespace) ? planCheckRunResultSuccess_Code_NamespaceFromJSON(object.namespace) : 0,
    };
  },

  toJSON(message: PlanCheckRunResultSuccess_Code): unknown {
    const obj: any = {};
    message.code !== undefined && (obj.code = Math.round(message.code));
    message.namespace !== undefined &&
      (obj.namespace = planCheckRunResultSuccess_Code_NamespaceToJSON(message.namespace));
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRunResultSuccess_Code>): PlanCheckRunResultSuccess_Code {
    return PlanCheckRunResultSuccess_Code.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanCheckRunResultSuccess_Code>): PlanCheckRunResultSuccess_Code {
    const message = createBasePlanCheckRunResultSuccess_Code();
    message.code = object.code ?? 0;
    message.namespace = object.namespace ?? 0;
    return message;
  },
};

function createBasePlanCheckRunResultSuccess_StatementTypeReport(): PlanCheckRunResultSuccess_StatementTypeReport {
  return { statementType: "" };
}

export const PlanCheckRunResultSuccess_StatementTypeReport = {
  encode(message: PlanCheckRunResultSuccess_StatementTypeReport, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.statementType !== "") {
      writer.uint32(10).string(message.statementType);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRunResultSuccess_StatementTypeReport {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRunResultSuccess_StatementTypeReport();
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

  fromJSON(object: any): PlanCheckRunResultSuccess_StatementTypeReport {
    return { statementType: isSet(object.statementType) ? String(object.statementType) : "" };
  },

  toJSON(message: PlanCheckRunResultSuccess_StatementTypeReport): unknown {
    const obj: any = {};
    message.statementType !== undefined && (obj.statementType = message.statementType);
    return obj;
  },

  create(
    base?: DeepPartial<PlanCheckRunResultSuccess_StatementTypeReport>,
  ): PlanCheckRunResultSuccess_StatementTypeReport {
    return PlanCheckRunResultSuccess_StatementTypeReport.fromPartial(base ?? {});
  },

  fromPartial(
    object: DeepPartial<PlanCheckRunResultSuccess_StatementTypeReport>,
  ): PlanCheckRunResultSuccess_StatementTypeReport {
    const message = createBasePlanCheckRunResultSuccess_StatementTypeReport();
    message.statementType = object.statementType ?? "";
    return message;
  },
};

function createBasePlanCheckRunResultSuccess_AffectedRowsReport(): PlanCheckRunResultSuccess_AffectedRowsReport {
  return { affectedRows: 0 };
}

export const PlanCheckRunResultSuccess_AffectedRowsReport = {
  encode(message: PlanCheckRunResultSuccess_AffectedRowsReport, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.affectedRows !== 0) {
      writer.uint32(8).int64(message.affectedRows);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRunResultSuccess_AffectedRowsReport {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRunResultSuccess_AffectedRowsReport();
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

  fromJSON(object: any): PlanCheckRunResultSuccess_AffectedRowsReport {
    return { affectedRows: isSet(object.affectedRows) ? Number(object.affectedRows) : 0 };
  },

  toJSON(message: PlanCheckRunResultSuccess_AffectedRowsReport): unknown {
    const obj: any = {};
    message.affectedRows !== undefined && (obj.affectedRows = Math.round(message.affectedRows));
    return obj;
  },

  create(
    base?: DeepPartial<PlanCheckRunResultSuccess_AffectedRowsReport>,
  ): PlanCheckRunResultSuccess_AffectedRowsReport {
    return PlanCheckRunResultSuccess_AffectedRowsReport.fromPartial(base ?? {});
  },

  fromPartial(
    object: DeepPartial<PlanCheckRunResultSuccess_AffectedRowsReport>,
  ): PlanCheckRunResultSuccess_AffectedRowsReport {
    const message = createBasePlanCheckRunResultSuccess_AffectedRowsReport();
    message.affectedRows = object.affectedRows ?? 0;
    return message;
  },
};

function createBasePlanCheckRunResultSuccess_SqlReviewReport(): PlanCheckRunResultSuccess_SqlReviewReport {
  return { line: 0, detail: "" };
}

export const PlanCheckRunResultSuccess_SqlReviewReport = {
  encode(message: PlanCheckRunResultSuccess_SqlReviewReport, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.line !== 0) {
      writer.uint32(8).int64(message.line);
    }
    if (message.detail !== "") {
      writer.uint32(18).string(message.detail);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRunResultSuccess_SqlReviewReport {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRunResultSuccess_SqlReviewReport();
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
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanCheckRunResultSuccess_SqlReviewReport {
    return {
      line: isSet(object.line) ? Number(object.line) : 0,
      detail: isSet(object.detail) ? String(object.detail) : "",
    };
  },

  toJSON(message: PlanCheckRunResultSuccess_SqlReviewReport): unknown {
    const obj: any = {};
    message.line !== undefined && (obj.line = Math.round(message.line));
    message.detail !== undefined && (obj.detail = message.detail);
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRunResultSuccess_SqlReviewReport>): PlanCheckRunResultSuccess_SqlReviewReport {
    return PlanCheckRunResultSuccess_SqlReviewReport.fromPartial(base ?? {});
  },

  fromPartial(
    object: DeepPartial<PlanCheckRunResultSuccess_SqlReviewReport>,
  ): PlanCheckRunResultSuccess_SqlReviewReport {
    const message = createBasePlanCheckRunResultSuccess_SqlReviewReport();
    message.line = object.line ?? 0;
    message.detail = object.detail ?? "";
    return message;
  },
};

function createBasePlanCheckRunResultSuccess_GeneralReport(): PlanCheckRunResultSuccess_GeneralReport {
  return {};
}

export const PlanCheckRunResultSuccess_GeneralReport = {
  encode(_: PlanCheckRunResultSuccess_GeneralReport, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRunResultSuccess_GeneralReport {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRunResultSuccess_GeneralReport();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): PlanCheckRunResultSuccess_GeneralReport {
    return {};
  },

  toJSON(_: PlanCheckRunResultSuccess_GeneralReport): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRunResultSuccess_GeneralReport>): PlanCheckRunResultSuccess_GeneralReport {
    return PlanCheckRunResultSuccess_GeneralReport.fromPartial(base ?? {});
  },

  fromPartial(_: DeepPartial<PlanCheckRunResultSuccess_GeneralReport>): PlanCheckRunResultSuccess_GeneralReport {
    const message = createBasePlanCheckRunResultSuccess_GeneralReport();
    return message;
  },
};

function createBasePlanCheckRunResultFailure(): PlanCheckRunResultFailure {
  return { title: "", content: "" };
}

export const PlanCheckRunResultFailure = {
  encode(message: PlanCheckRunResultFailure, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.title !== "") {
      writer.uint32(10).string(message.title);
    }
    if (message.content !== "") {
      writer.uint32(18).string(message.content);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanCheckRunResultFailure {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanCheckRunResultFailure();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.title = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.content = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanCheckRunResultFailure {
    return {
      title: isSet(object.title) ? String(object.title) : "",
      content: isSet(object.content) ? String(object.content) : "",
    };
  },

  toJSON(message: PlanCheckRunResultFailure): unknown {
    const obj: any = {};
    message.title !== undefined && (obj.title = message.title);
    message.content !== undefined && (obj.content = message.content);
    return obj;
  },

  create(base?: DeepPartial<PlanCheckRunResultFailure>): PlanCheckRunResultFailure {
    return PlanCheckRunResultFailure.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanCheckRunResultFailure>): PlanCheckRunResultFailure {
    const message = createBasePlanCheckRunResultFailure();
    message.title = object.title ?? "";
    message.content = object.content ?? "";
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
