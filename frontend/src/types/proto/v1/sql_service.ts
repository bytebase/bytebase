/* eslint-disable */
import * as Long from "long";
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Duration } from "../google/protobuf/duration";
import { NullValue, nullValueFromJSON, nullValueToJSON, Value } from "../google/protobuf/struct";
import { Engine, engineFromJSON, engineToJSON } from "./common";
import { DatabaseMetadata } from "./database_service";

export const protobufPackage = "bytebase.v1";

export interface DifferPreviewRequest {
  engine: Engine;
  oldSchema: string;
  newMetadata?: DatabaseMetadata | undefined;
}

export interface DifferPreviewResponse {
  schema: string;
}

export interface AdminExecuteRequest {
  /**
   * The name is the instance name to execute the query against.
   * Format: instances/{instance}
   */
  name: string;
  /**
   * The connection database name to execute the query against.
   * For PostgreSQL, it's required.
   * For other database engines, it's optional. Use empty string to execute against without specifying a database.
   */
  connectionDatabase: string;
  /** The SQL statement to execute. */
  statement: string;
  /** The maximum number of rows to return. */
  limit: number;
  /** The timeout for the request. */
  timeout?: Duration | undefined;
}

export interface AdminExecuteResponse {
  /** The query results. */
  results: QueryResult[];
}

export interface ExportRequest {
  /**
   * The name is the instance name to execute the query against.
   * Format: instances/{instance}
   */
  name: string;
  /**
   * The connection database name to execute the query against.
   * For PostgreSQL, it's required.
   * For other database engines, it's optional. Use empty string to execute against without specifying a database.
   */
  connectionDatabase: string;
  /** The SQL statement to execute. */
  statement: string;
  /** The maximum number of rows to return. */
  limit: number;
  /** The export format. */
  format: ExportRequest_Format;
  /**
   * The admin is used for workspace owner and DBA for exporting data from SQL Editor Admin mode.
   * The exported data is not masked.
   */
  admin: boolean;
}

export enum ExportRequest_Format {
  FORMAT_UNSPECIFIED = 0,
  CSV = 1,
  JSON = 2,
  SQL = 3,
  XLSX = 4,
  UNRECOGNIZED = -1,
}

export function exportRequest_FormatFromJSON(object: any): ExportRequest_Format {
  switch (object) {
    case 0:
    case "FORMAT_UNSPECIFIED":
      return ExportRequest_Format.FORMAT_UNSPECIFIED;
    case 1:
    case "CSV":
      return ExportRequest_Format.CSV;
    case 2:
    case "JSON":
      return ExportRequest_Format.JSON;
    case 3:
    case "SQL":
      return ExportRequest_Format.SQL;
    case 4:
    case "XLSX":
      return ExportRequest_Format.XLSX;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ExportRequest_Format.UNRECOGNIZED;
  }
}

export function exportRequest_FormatToJSON(object: ExportRequest_Format): string {
  switch (object) {
    case ExportRequest_Format.FORMAT_UNSPECIFIED:
      return "FORMAT_UNSPECIFIED";
    case ExportRequest_Format.CSV:
      return "CSV";
    case ExportRequest_Format.JSON:
      return "JSON";
    case ExportRequest_Format.SQL:
      return "SQL";
    case ExportRequest_Format.XLSX:
      return "XLSX";
    case ExportRequest_Format.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface ExportResponse {
  /** The export file content. */
  content: Uint8Array;
}

export interface QueryRequest {
  /**
   * The name is the instance name to execute the query against.
   * Format: instances/{instance}
   */
  name: string;
  /**
   * The connection database name to execute the query against.
   * For PostgreSQL, it's required.
   * For other database engines, it's optional. Use empty string to execute against without specifying a database.
   */
  connectionDatabase: string;
  /** The SQL statement to execute. */
  statement: string;
  /** The maximum number of rows to return. */
  limit: number;
  /** The timeout for the request. */
  timeout?: Duration | undefined;
}

export interface QueryResponse {
  /** The query results. */
  results: QueryResult[];
  /** The query advices. */
  advices: Advice[];
  /** The query is allowed to be exported or not. */
  allowExport: boolean;
}

export interface QueryResult {
  /** Column names of the query result. */
  columnNames: string[];
  /**
   * Column types of the query result.
   * The types come from the Golang SQL driver.
   */
  columnTypeNames: string[];
  /** Rows of the query result. */
  rows: QueryRow[];
  /** Columns are masked or not. */
  masked: boolean[];
  /** Columns are sensitive or not. */
  sensitive: boolean[];
  /** The error message if the query failed. */
  error: string;
  /** The time it takes to execute the query. */
  latency?:
    | Duration
    | undefined;
  /** The query statement for the result. */
  statement: string;
}

export interface QueryRow {
  /** Row values of the query result. */
  values: RowValue[];
}

export interface RowValue {
  nullValue?: NullValue | undefined;
  boolValue?: boolean | undefined;
  bytesValue?: Uint8Array | undefined;
  doubleValue?: number | undefined;
  floatValue?: number | undefined;
  int32Value?: number | undefined;
  int64Value?: number | undefined;
  stringValue?: string | undefined;
  uint32Value?: number | undefined;
  uint64Value?:
    | number
    | undefined;
  /** value_value is used for Spanner and TUPLE ARRAY MAP in Clickhouse only. */
  valueValue?: any | undefined;
}

export interface Advice {
  /** The advice status. */
  status: Advice_Status;
  /** The advice code. */
  code: number;
  /** The advice title. */
  title: string;
  /** The advice content. */
  content: string;
  /** The advice line number in the SQL statement. */
  line: number;
  /** The advice detail. */
  detail: string;
}

export enum Advice_Status {
  /** STATUS_UNSPECIFIED - Unspecified. */
  STATUS_UNSPECIFIED = 0,
  SUCCESS = 1,
  WARNING = 2,
  ERROR = 3,
  UNRECOGNIZED = -1,
}

export function advice_StatusFromJSON(object: any): Advice_Status {
  switch (object) {
    case 0:
    case "STATUS_UNSPECIFIED":
      return Advice_Status.STATUS_UNSPECIFIED;
    case 1:
    case "SUCCESS":
      return Advice_Status.SUCCESS;
    case 2:
    case "WARNING":
      return Advice_Status.WARNING;
    case 3:
    case "ERROR":
      return Advice_Status.ERROR;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Advice_Status.UNRECOGNIZED;
  }
}

export function advice_StatusToJSON(object: Advice_Status): string {
  switch (object) {
    case Advice_Status.STATUS_UNSPECIFIED:
      return "STATUS_UNSPECIFIED";
    case Advice_Status.SUCCESS:
      return "SUCCESS";
    case Advice_Status.WARNING:
      return "WARNING";
    case Advice_Status.ERROR:
      return "ERROR";
    case Advice_Status.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface PrettyRequest {
  engine: Engine;
  /**
   * The SDL format SQL schema information that was dumped from a database engine.
   * This information will be sorted to match the order of statements in the userSchema.
   */
  currentSchema: string;
  /** The expected SDL schema. This schema will be checked for correctness and normalized. */
  expectedSchema: string;
}

export interface PrettyResponse {
  /** The pretty-formatted version of current schema. */
  currentSchema: string;
  /** The expected SDL schema after normalizing. */
  expectedSchema: string;
}

function createBaseDifferPreviewRequest(): DifferPreviewRequest {
  return { engine: 0, oldSchema: "", newMetadata: undefined };
}

export const DifferPreviewRequest = {
  encode(message: DifferPreviewRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.engine !== 0) {
      writer.uint32(8).int32(message.engine);
    }
    if (message.oldSchema !== "") {
      writer.uint32(18).string(message.oldSchema);
    }
    if (message.newMetadata !== undefined) {
      DatabaseMetadata.encode(message.newMetadata, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DifferPreviewRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDifferPreviewRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.engine = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.oldSchema = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.newMetadata = DatabaseMetadata.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DifferPreviewRequest {
    return {
      engine: isSet(object.engine) ? engineFromJSON(object.engine) : 0,
      oldSchema: isSet(object.oldSchema) ? String(object.oldSchema) : "",
      newMetadata: isSet(object.newMetadata) ? DatabaseMetadata.fromJSON(object.newMetadata) : undefined,
    };
  },

  toJSON(message: DifferPreviewRequest): unknown {
    const obj: any = {};
    message.engine !== undefined && (obj.engine = engineToJSON(message.engine));
    message.oldSchema !== undefined && (obj.oldSchema = message.oldSchema);
    message.newMetadata !== undefined &&
      (obj.newMetadata = message.newMetadata ? DatabaseMetadata.toJSON(message.newMetadata) : undefined);
    return obj;
  },

  create(base?: DeepPartial<DifferPreviewRequest>): DifferPreviewRequest {
    return DifferPreviewRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<DifferPreviewRequest>): DifferPreviewRequest {
    const message = createBaseDifferPreviewRequest();
    message.engine = object.engine ?? 0;
    message.oldSchema = object.oldSchema ?? "";
    message.newMetadata = (object.newMetadata !== undefined && object.newMetadata !== null)
      ? DatabaseMetadata.fromPartial(object.newMetadata)
      : undefined;
    return message;
  },
};

function createBaseDifferPreviewResponse(): DifferPreviewResponse {
  return { schema: "" };
}

export const DifferPreviewResponse = {
  encode(message: DifferPreviewResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schema !== "") {
      writer.uint32(10).string(message.schema);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DifferPreviewResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDifferPreviewResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.schema = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DifferPreviewResponse {
    return { schema: isSet(object.schema) ? String(object.schema) : "" };
  },

  toJSON(message: DifferPreviewResponse): unknown {
    const obj: any = {};
    message.schema !== undefined && (obj.schema = message.schema);
    return obj;
  },

  create(base?: DeepPartial<DifferPreviewResponse>): DifferPreviewResponse {
    return DifferPreviewResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<DifferPreviewResponse>): DifferPreviewResponse {
    const message = createBaseDifferPreviewResponse();
    message.schema = object.schema ?? "";
    return message;
  },
};

function createBaseAdminExecuteRequest(): AdminExecuteRequest {
  return { name: "", connectionDatabase: "", statement: "", limit: 0, timeout: undefined };
}

export const AdminExecuteRequest = {
  encode(message: AdminExecuteRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.connectionDatabase !== "") {
      writer.uint32(18).string(message.connectionDatabase);
    }
    if (message.statement !== "") {
      writer.uint32(26).string(message.statement);
    }
    if (message.limit !== 0) {
      writer.uint32(32).int32(message.limit);
    }
    if (message.timeout !== undefined) {
      Duration.encode(message.timeout, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): AdminExecuteRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAdminExecuteRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.connectionDatabase = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.statement = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.limit = reader.int32();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.timeout = Duration.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): AdminExecuteRequest {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      connectionDatabase: isSet(object.connectionDatabase) ? String(object.connectionDatabase) : "",
      statement: isSet(object.statement) ? String(object.statement) : "",
      limit: isSet(object.limit) ? Number(object.limit) : 0,
      timeout: isSet(object.timeout) ? Duration.fromJSON(object.timeout) : undefined,
    };
  },

  toJSON(message: AdminExecuteRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.connectionDatabase !== undefined && (obj.connectionDatabase = message.connectionDatabase);
    message.statement !== undefined && (obj.statement = message.statement);
    message.limit !== undefined && (obj.limit = Math.round(message.limit));
    message.timeout !== undefined && (obj.timeout = message.timeout ? Duration.toJSON(message.timeout) : undefined);
    return obj;
  },

  create(base?: DeepPartial<AdminExecuteRequest>): AdminExecuteRequest {
    return AdminExecuteRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<AdminExecuteRequest>): AdminExecuteRequest {
    const message = createBaseAdminExecuteRequest();
    message.name = object.name ?? "";
    message.connectionDatabase = object.connectionDatabase ?? "";
    message.statement = object.statement ?? "";
    message.limit = object.limit ?? 0;
    message.timeout = (object.timeout !== undefined && object.timeout !== null)
      ? Duration.fromPartial(object.timeout)
      : undefined;
    return message;
  },
};

function createBaseAdminExecuteResponse(): AdminExecuteResponse {
  return { results: [] };
}

export const AdminExecuteResponse = {
  encode(message: AdminExecuteResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.results) {
      QueryResult.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): AdminExecuteResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAdminExecuteResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.results.push(QueryResult.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): AdminExecuteResponse {
    return { results: Array.isArray(object?.results) ? object.results.map((e: any) => QueryResult.fromJSON(e)) : [] };
  },

  toJSON(message: AdminExecuteResponse): unknown {
    const obj: any = {};
    if (message.results) {
      obj.results = message.results.map((e) => e ? QueryResult.toJSON(e) : undefined);
    } else {
      obj.results = [];
    }
    return obj;
  },

  create(base?: DeepPartial<AdminExecuteResponse>): AdminExecuteResponse {
    return AdminExecuteResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<AdminExecuteResponse>): AdminExecuteResponse {
    const message = createBaseAdminExecuteResponse();
    message.results = object.results?.map((e) => QueryResult.fromPartial(e)) || [];
    return message;
  },
};

function createBaseExportRequest(): ExportRequest {
  return { name: "", connectionDatabase: "", statement: "", limit: 0, format: 0, admin: false };
}

export const ExportRequest = {
  encode(message: ExportRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.connectionDatabase !== "") {
      writer.uint32(18).string(message.connectionDatabase);
    }
    if (message.statement !== "") {
      writer.uint32(26).string(message.statement);
    }
    if (message.limit !== 0) {
      writer.uint32(32).int32(message.limit);
    }
    if (message.format !== 0) {
      writer.uint32(40).int32(message.format);
    }
    if (message.admin === true) {
      writer.uint32(48).bool(message.admin);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ExportRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExportRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.connectionDatabase = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.statement = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.limit = reader.int32();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.format = reader.int32() as any;
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.admin = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ExportRequest {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      connectionDatabase: isSet(object.connectionDatabase) ? String(object.connectionDatabase) : "",
      statement: isSet(object.statement) ? String(object.statement) : "",
      limit: isSet(object.limit) ? Number(object.limit) : 0,
      format: isSet(object.format) ? exportRequest_FormatFromJSON(object.format) : 0,
      admin: isSet(object.admin) ? Boolean(object.admin) : false,
    };
  },

  toJSON(message: ExportRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.connectionDatabase !== undefined && (obj.connectionDatabase = message.connectionDatabase);
    message.statement !== undefined && (obj.statement = message.statement);
    message.limit !== undefined && (obj.limit = Math.round(message.limit));
    message.format !== undefined && (obj.format = exportRequest_FormatToJSON(message.format));
    message.admin !== undefined && (obj.admin = message.admin);
    return obj;
  },

  create(base?: DeepPartial<ExportRequest>): ExportRequest {
    return ExportRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ExportRequest>): ExportRequest {
    const message = createBaseExportRequest();
    message.name = object.name ?? "";
    message.connectionDatabase = object.connectionDatabase ?? "";
    message.statement = object.statement ?? "";
    message.limit = object.limit ?? 0;
    message.format = object.format ?? 0;
    message.admin = object.admin ?? false;
    return message;
  },
};

function createBaseExportResponse(): ExportResponse {
  return { content: new Uint8Array(0) };
}

export const ExportResponse = {
  encode(message: ExportResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.content.length !== 0) {
      writer.uint32(10).bytes(message.content);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ExportResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExportResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.content = reader.bytes();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ExportResponse {
    return { content: isSet(object.content) ? bytesFromBase64(object.content) : new Uint8Array(0) };
  },

  toJSON(message: ExportResponse): unknown {
    const obj: any = {};
    message.content !== undefined &&
      (obj.content = base64FromBytes(message.content !== undefined ? message.content : new Uint8Array(0)));
    return obj;
  },

  create(base?: DeepPartial<ExportResponse>): ExportResponse {
    return ExportResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ExportResponse>): ExportResponse {
    const message = createBaseExportResponse();
    message.content = object.content ?? new Uint8Array(0);
    return message;
  },
};

function createBaseQueryRequest(): QueryRequest {
  return { name: "", connectionDatabase: "", statement: "", limit: 0, timeout: undefined };
}

export const QueryRequest = {
  encode(message: QueryRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.connectionDatabase !== "") {
      writer.uint32(18).string(message.connectionDatabase);
    }
    if (message.statement !== "") {
      writer.uint32(26).string(message.statement);
    }
    if (message.limit !== 0) {
      writer.uint32(32).int32(message.limit);
    }
    if (message.timeout !== undefined) {
      Duration.encode(message.timeout, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.connectionDatabase = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.statement = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.limit = reader.int32();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.timeout = Duration.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryRequest {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      connectionDatabase: isSet(object.connectionDatabase) ? String(object.connectionDatabase) : "",
      statement: isSet(object.statement) ? String(object.statement) : "",
      limit: isSet(object.limit) ? Number(object.limit) : 0,
      timeout: isSet(object.timeout) ? Duration.fromJSON(object.timeout) : undefined,
    };
  },

  toJSON(message: QueryRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.connectionDatabase !== undefined && (obj.connectionDatabase = message.connectionDatabase);
    message.statement !== undefined && (obj.statement = message.statement);
    message.limit !== undefined && (obj.limit = Math.round(message.limit));
    message.timeout !== undefined && (obj.timeout = message.timeout ? Duration.toJSON(message.timeout) : undefined);
    return obj;
  },

  create(base?: DeepPartial<QueryRequest>): QueryRequest {
    return QueryRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<QueryRequest>): QueryRequest {
    const message = createBaseQueryRequest();
    message.name = object.name ?? "";
    message.connectionDatabase = object.connectionDatabase ?? "";
    message.statement = object.statement ?? "";
    message.limit = object.limit ?? 0;
    message.timeout = (object.timeout !== undefined && object.timeout !== null)
      ? Duration.fromPartial(object.timeout)
      : undefined;
    return message;
  },
};

function createBaseQueryResponse(): QueryResponse {
  return { results: [], advices: [], allowExport: false };
}

export const QueryResponse = {
  encode(message: QueryResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.results) {
      QueryResult.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.advices) {
      Advice.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.allowExport === true) {
      writer.uint32(24).bool(message.allowExport);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.results.push(QueryResult.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.advices.push(Advice.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.allowExport = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryResponse {
    return {
      results: Array.isArray(object?.results) ? object.results.map((e: any) => QueryResult.fromJSON(e)) : [],
      advices: Array.isArray(object?.advices) ? object.advices.map((e: any) => Advice.fromJSON(e)) : [],
      allowExport: isSet(object.allowExport) ? Boolean(object.allowExport) : false,
    };
  },

  toJSON(message: QueryResponse): unknown {
    const obj: any = {};
    if (message.results) {
      obj.results = message.results.map((e) => e ? QueryResult.toJSON(e) : undefined);
    } else {
      obj.results = [];
    }
    if (message.advices) {
      obj.advices = message.advices.map((e) => e ? Advice.toJSON(e) : undefined);
    } else {
      obj.advices = [];
    }
    message.allowExport !== undefined && (obj.allowExport = message.allowExport);
    return obj;
  },

  create(base?: DeepPartial<QueryResponse>): QueryResponse {
    return QueryResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<QueryResponse>): QueryResponse {
    const message = createBaseQueryResponse();
    message.results = object.results?.map((e) => QueryResult.fromPartial(e)) || [];
    message.advices = object.advices?.map((e) => Advice.fromPartial(e)) || [];
    message.allowExport = object.allowExport ?? false;
    return message;
  },
};

function createBaseQueryResult(): QueryResult {
  return {
    columnNames: [],
    columnTypeNames: [],
    rows: [],
    masked: [],
    sensitive: [],
    error: "",
    latency: undefined,
    statement: "",
  };
}

export const QueryResult = {
  encode(message: QueryResult, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.columnNames) {
      writer.uint32(10).string(v!);
    }
    for (const v of message.columnTypeNames) {
      writer.uint32(18).string(v!);
    }
    for (const v of message.rows) {
      QueryRow.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    writer.uint32(34).fork();
    for (const v of message.masked) {
      writer.bool(v);
    }
    writer.ldelim();
    writer.uint32(42).fork();
    for (const v of message.sensitive) {
      writer.bool(v);
    }
    writer.ldelim();
    if (message.error !== "") {
      writer.uint32(50).string(message.error);
    }
    if (message.latency !== undefined) {
      Duration.encode(message.latency, writer.uint32(58).fork()).ldelim();
    }
    if (message.statement !== "") {
      writer.uint32(66).string(message.statement);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryResult {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryResult();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.columnNames.push(reader.string());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.columnTypeNames.push(reader.string());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.rows.push(QueryRow.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag === 32) {
            message.masked.push(reader.bool());

            continue;
          }

          if (tag === 34) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.masked.push(reader.bool());
            }

            continue;
          }

          break;
        case 5:
          if (tag === 40) {
            message.sensitive.push(reader.bool());

            continue;
          }

          if (tag === 42) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.sensitive.push(reader.bool());
            }

            continue;
          }

          break;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.error = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.latency = Duration.decode(reader, reader.uint32());
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.statement = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryResult {
    return {
      columnNames: Array.isArray(object?.columnNames) ? object.columnNames.map((e: any) => String(e)) : [],
      columnTypeNames: Array.isArray(object?.columnTypeNames) ? object.columnTypeNames.map((e: any) => String(e)) : [],
      rows: Array.isArray(object?.rows) ? object.rows.map((e: any) => QueryRow.fromJSON(e)) : [],
      masked: Array.isArray(object?.masked) ? object.masked.map((e: any) => Boolean(e)) : [],
      sensitive: Array.isArray(object?.sensitive) ? object.sensitive.map((e: any) => Boolean(e)) : [],
      error: isSet(object.error) ? String(object.error) : "",
      latency: isSet(object.latency) ? Duration.fromJSON(object.latency) : undefined,
      statement: isSet(object.statement) ? String(object.statement) : "",
    };
  },

  toJSON(message: QueryResult): unknown {
    const obj: any = {};
    if (message.columnNames) {
      obj.columnNames = message.columnNames.map((e) => e);
    } else {
      obj.columnNames = [];
    }
    if (message.columnTypeNames) {
      obj.columnTypeNames = message.columnTypeNames.map((e) => e);
    } else {
      obj.columnTypeNames = [];
    }
    if (message.rows) {
      obj.rows = message.rows.map((e) => e ? QueryRow.toJSON(e) : undefined);
    } else {
      obj.rows = [];
    }
    if (message.masked) {
      obj.masked = message.masked.map((e) => e);
    } else {
      obj.masked = [];
    }
    if (message.sensitive) {
      obj.sensitive = message.sensitive.map((e) => e);
    } else {
      obj.sensitive = [];
    }
    message.error !== undefined && (obj.error = message.error);
    message.latency !== undefined && (obj.latency = message.latency ? Duration.toJSON(message.latency) : undefined);
    message.statement !== undefined && (obj.statement = message.statement);
    return obj;
  },

  create(base?: DeepPartial<QueryResult>): QueryResult {
    return QueryResult.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<QueryResult>): QueryResult {
    const message = createBaseQueryResult();
    message.columnNames = object.columnNames?.map((e) => e) || [];
    message.columnTypeNames = object.columnTypeNames?.map((e) => e) || [];
    message.rows = object.rows?.map((e) => QueryRow.fromPartial(e)) || [];
    message.masked = object.masked?.map((e) => e) || [];
    message.sensitive = object.sensitive?.map((e) => e) || [];
    message.error = object.error ?? "";
    message.latency = (object.latency !== undefined && object.latency !== null)
      ? Duration.fromPartial(object.latency)
      : undefined;
    message.statement = object.statement ?? "";
    return message;
  },
};

function createBaseQueryRow(): QueryRow {
  return { values: [] };
}

export const QueryRow = {
  encode(message: QueryRow, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.values) {
      RowValue.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryRow {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryRow();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.values.push(RowValue.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryRow {
    return { values: Array.isArray(object?.values) ? object.values.map((e: any) => RowValue.fromJSON(e)) : [] };
  },

  toJSON(message: QueryRow): unknown {
    const obj: any = {};
    if (message.values) {
      obj.values = message.values.map((e) => e ? RowValue.toJSON(e) : undefined);
    } else {
      obj.values = [];
    }
    return obj;
  },

  create(base?: DeepPartial<QueryRow>): QueryRow {
    return QueryRow.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<QueryRow>): QueryRow {
    const message = createBaseQueryRow();
    message.values = object.values?.map((e) => RowValue.fromPartial(e)) || [];
    return message;
  },
};

function createBaseRowValue(): RowValue {
  return {
    nullValue: undefined,
    boolValue: undefined,
    bytesValue: undefined,
    doubleValue: undefined,
    floatValue: undefined,
    int32Value: undefined,
    int64Value: undefined,
    stringValue: undefined,
    uint32Value: undefined,
    uint64Value: undefined,
    valueValue: undefined,
  };
}

export const RowValue = {
  encode(message: RowValue, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.nullValue !== undefined) {
      writer.uint32(8).int32(message.nullValue);
    }
    if (message.boolValue !== undefined) {
      writer.uint32(16).bool(message.boolValue);
    }
    if (message.bytesValue !== undefined) {
      writer.uint32(26).bytes(message.bytesValue);
    }
    if (message.doubleValue !== undefined) {
      writer.uint32(33).double(message.doubleValue);
    }
    if (message.floatValue !== undefined) {
      writer.uint32(45).float(message.floatValue);
    }
    if (message.int32Value !== undefined) {
      writer.uint32(48).int32(message.int32Value);
    }
    if (message.int64Value !== undefined) {
      writer.uint32(56).int64(message.int64Value);
    }
    if (message.stringValue !== undefined) {
      writer.uint32(66).string(message.stringValue);
    }
    if (message.uint32Value !== undefined) {
      writer.uint32(72).uint32(message.uint32Value);
    }
    if (message.uint64Value !== undefined) {
      writer.uint32(80).uint64(message.uint64Value);
    }
    if (message.valueValue !== undefined) {
      Value.encode(Value.wrap(message.valueValue), writer.uint32(90).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RowValue {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRowValue();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.nullValue = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.boolValue = reader.bool();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.bytesValue = reader.bytes();
          continue;
        case 4:
          if (tag !== 33) {
            break;
          }

          message.doubleValue = reader.double();
          continue;
        case 5:
          if (tag !== 45) {
            break;
          }

          message.floatValue = reader.float();
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.int32Value = reader.int32();
          continue;
        case 7:
          if (tag !== 56) {
            break;
          }

          message.int64Value = longToNumber(reader.int64() as Long);
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.stringValue = reader.string();
          continue;
        case 9:
          if (tag !== 72) {
            break;
          }

          message.uint32Value = reader.uint32();
          continue;
        case 10:
          if (tag !== 80) {
            break;
          }

          message.uint64Value = longToNumber(reader.uint64() as Long);
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.valueValue = Value.unwrap(Value.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): RowValue {
    return {
      nullValue: isSet(object.nullValue) ? nullValueFromJSON(object.nullValue) : undefined,
      boolValue: isSet(object.boolValue) ? Boolean(object.boolValue) : undefined,
      bytesValue: isSet(object.bytesValue) ? bytesFromBase64(object.bytesValue) : undefined,
      doubleValue: isSet(object.doubleValue) ? Number(object.doubleValue) : undefined,
      floatValue: isSet(object.floatValue) ? Number(object.floatValue) : undefined,
      int32Value: isSet(object.int32Value) ? Number(object.int32Value) : undefined,
      int64Value: isSet(object.int64Value) ? Number(object.int64Value) : undefined,
      stringValue: isSet(object.stringValue) ? String(object.stringValue) : undefined,
      uint32Value: isSet(object.uint32Value) ? Number(object.uint32Value) : undefined,
      uint64Value: isSet(object.uint64Value) ? Number(object.uint64Value) : undefined,
      valueValue: isSet(object?.valueValue) ? object.valueValue : undefined,
    };
  },

  toJSON(message: RowValue): unknown {
    const obj: any = {};
    message.nullValue !== undefined &&
      (obj.nullValue = message.nullValue !== undefined ? nullValueToJSON(message.nullValue) : undefined);
    message.boolValue !== undefined && (obj.boolValue = message.boolValue);
    message.bytesValue !== undefined &&
      (obj.bytesValue = message.bytesValue !== undefined ? base64FromBytes(message.bytesValue) : undefined);
    message.doubleValue !== undefined && (obj.doubleValue = message.doubleValue);
    message.floatValue !== undefined && (obj.floatValue = message.floatValue);
    message.int32Value !== undefined && (obj.int32Value = Math.round(message.int32Value));
    message.int64Value !== undefined && (obj.int64Value = Math.round(message.int64Value));
    message.stringValue !== undefined && (obj.stringValue = message.stringValue);
    message.uint32Value !== undefined && (obj.uint32Value = Math.round(message.uint32Value));
    message.uint64Value !== undefined && (obj.uint64Value = Math.round(message.uint64Value));
    message.valueValue !== undefined && (obj.valueValue = message.valueValue);
    return obj;
  },

  create(base?: DeepPartial<RowValue>): RowValue {
    return RowValue.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<RowValue>): RowValue {
    const message = createBaseRowValue();
    message.nullValue = object.nullValue ?? undefined;
    message.boolValue = object.boolValue ?? undefined;
    message.bytesValue = object.bytesValue ?? undefined;
    message.doubleValue = object.doubleValue ?? undefined;
    message.floatValue = object.floatValue ?? undefined;
    message.int32Value = object.int32Value ?? undefined;
    message.int64Value = object.int64Value ?? undefined;
    message.stringValue = object.stringValue ?? undefined;
    message.uint32Value = object.uint32Value ?? undefined;
    message.uint64Value = object.uint64Value ?? undefined;
    message.valueValue = object.valueValue ?? undefined;
    return message;
  },
};

function createBaseAdvice(): Advice {
  return { status: 0, code: 0, title: "", content: "", line: 0, detail: "" };
}

export const Advice = {
  encode(message: Advice, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.status !== 0) {
      writer.uint32(8).int32(message.status);
    }
    if (message.code !== 0) {
      writer.uint32(16).int32(message.code);
    }
    if (message.title !== "") {
      writer.uint32(26).string(message.title);
    }
    if (message.content !== "") {
      writer.uint32(34).string(message.content);
    }
    if (message.line !== 0) {
      writer.uint32(40).int32(message.line);
    }
    if (message.detail !== "") {
      writer.uint32(50).string(message.detail);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Advice {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAdvice();
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
          if (tag !== 16) {
            break;
          }

          message.code = reader.int32();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.title = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.content = reader.string();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.line = reader.int32();
          continue;
        case 6:
          if (tag !== 50) {
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

  fromJSON(object: any): Advice {
    return {
      status: isSet(object.status) ? advice_StatusFromJSON(object.status) : 0,
      code: isSet(object.code) ? Number(object.code) : 0,
      title: isSet(object.title) ? String(object.title) : "",
      content: isSet(object.content) ? String(object.content) : "",
      line: isSet(object.line) ? Number(object.line) : 0,
      detail: isSet(object.detail) ? String(object.detail) : "",
    };
  },

  toJSON(message: Advice): unknown {
    const obj: any = {};
    message.status !== undefined && (obj.status = advice_StatusToJSON(message.status));
    message.code !== undefined && (obj.code = Math.round(message.code));
    message.title !== undefined && (obj.title = message.title);
    message.content !== undefined && (obj.content = message.content);
    message.line !== undefined && (obj.line = Math.round(message.line));
    message.detail !== undefined && (obj.detail = message.detail);
    return obj;
  },

  create(base?: DeepPartial<Advice>): Advice {
    return Advice.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Advice>): Advice {
    const message = createBaseAdvice();
    message.status = object.status ?? 0;
    message.code = object.code ?? 0;
    message.title = object.title ?? "";
    message.content = object.content ?? "";
    message.line = object.line ?? 0;
    message.detail = object.detail ?? "";
    return message;
  },
};

function createBasePrettyRequest(): PrettyRequest {
  return { engine: 0, currentSchema: "", expectedSchema: "" };
}

export const PrettyRequest = {
  encode(message: PrettyRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.engine !== 0) {
      writer.uint32(8).int32(message.engine);
    }
    if (message.currentSchema !== "") {
      writer.uint32(18).string(message.currentSchema);
    }
    if (message.expectedSchema !== "") {
      writer.uint32(26).string(message.expectedSchema);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PrettyRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePrettyRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.engine = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.currentSchema = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.expectedSchema = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PrettyRequest {
    return {
      engine: isSet(object.engine) ? engineFromJSON(object.engine) : 0,
      currentSchema: isSet(object.currentSchema) ? String(object.currentSchema) : "",
      expectedSchema: isSet(object.expectedSchema) ? String(object.expectedSchema) : "",
    };
  },

  toJSON(message: PrettyRequest): unknown {
    const obj: any = {};
    message.engine !== undefined && (obj.engine = engineToJSON(message.engine));
    message.currentSchema !== undefined && (obj.currentSchema = message.currentSchema);
    message.expectedSchema !== undefined && (obj.expectedSchema = message.expectedSchema);
    return obj;
  },

  create(base?: DeepPartial<PrettyRequest>): PrettyRequest {
    return PrettyRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PrettyRequest>): PrettyRequest {
    const message = createBasePrettyRequest();
    message.engine = object.engine ?? 0;
    message.currentSchema = object.currentSchema ?? "";
    message.expectedSchema = object.expectedSchema ?? "";
    return message;
  },
};

function createBasePrettyResponse(): PrettyResponse {
  return { currentSchema: "", expectedSchema: "" };
}

export const PrettyResponse = {
  encode(message: PrettyResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.currentSchema !== "") {
      writer.uint32(10).string(message.currentSchema);
    }
    if (message.expectedSchema !== "") {
      writer.uint32(18).string(message.expectedSchema);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PrettyResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePrettyResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.currentSchema = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.expectedSchema = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PrettyResponse {
    return {
      currentSchema: isSet(object.currentSchema) ? String(object.currentSchema) : "",
      expectedSchema: isSet(object.expectedSchema) ? String(object.expectedSchema) : "",
    };
  },

  toJSON(message: PrettyResponse): unknown {
    const obj: any = {};
    message.currentSchema !== undefined && (obj.currentSchema = message.currentSchema);
    message.expectedSchema !== undefined && (obj.expectedSchema = message.expectedSchema);
    return obj;
  },

  create(base?: DeepPartial<PrettyResponse>): PrettyResponse {
    return PrettyResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PrettyResponse>): PrettyResponse {
    const message = createBasePrettyResponse();
    message.currentSchema = object.currentSchema ?? "";
    message.expectedSchema = object.expectedSchema ?? "";
    return message;
  },
};

export type SQLServiceDefinition = typeof SQLServiceDefinition;
export const SQLServiceDefinition = {
  name: "SQLService",
  fullName: "bytebase.v1.SQLService",
  methods: {
    pretty: {
      name: "Pretty",
      requestType: PrettyRequest,
      requestStream: false,
      responseType: PrettyResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([19, 58, 1, 42, 34, 14, 47, 118, 49, 47, 115, 113, 108, 47, 112, 114, 101, 116, 116, 121]),
          ],
        },
      },
    },
    query: {
      name: "Query",
      requestType: QueryRequest,
      requestStream: false,
      responseType: QueryResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              33,
              58,
              1,
              42,
              34,
              28,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
              47,
              42,
              125,
              58,
              113,
              117,
              101,
              114,
              121,
            ]),
          ],
        },
      },
    },
    export: {
      name: "Export",
      requestType: ExportRequest,
      requestStream: false,
      responseType: ExportResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              34,
              58,
              1,
              42,
              34,
              29,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
              47,
              42,
              125,
              58,
              101,
              120,
              112,
              111,
              114,
              116,
            ]),
          ],
        },
      },
    },
    adminExecute: {
      name: "AdminExecute",
      requestType: AdminExecuteRequest,
      requestStream: true,
      responseType: AdminExecuteResponse,
      responseStream: true,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([18, 18, 16, 47, 118, 49, 58, 97, 100, 109, 105, 110, 69, 120, 101, 99, 117, 116, 101]),
          ],
        },
      },
    },
    differPreview: {
      name: "DifferPreview",
      requestType: DifferPreviewRequest,
      requestStream: false,
      responseType: DifferPreviewResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              26,
              58,
              1,
              42,
              34,
              21,
              47,
              118,
              49,
              47,
              115,
              113,
              108,
              47,
              100,
              105,
              102,
              102,
              101,
              114,
              80,
              114,
              101,
              118,
              105,
              101,
              119,
            ]),
          ],
        },
      },
    },
  },
} as const;

export interface SQLServiceImplementation<CallContextExt = {}> {
  pretty(request: PrettyRequest, context: CallContext & CallContextExt): Promise<DeepPartial<PrettyResponse>>;
  query(request: QueryRequest, context: CallContext & CallContextExt): Promise<DeepPartial<QueryResponse>>;
  export(request: ExportRequest, context: CallContext & CallContextExt): Promise<DeepPartial<ExportResponse>>;
  adminExecute(
    request: AsyncIterable<AdminExecuteRequest>,
    context: CallContext & CallContextExt,
  ): ServerStreamingMethodResult<DeepPartial<AdminExecuteResponse>>;
  differPreview(
    request: DifferPreviewRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<DifferPreviewResponse>>;
}

export interface SQLServiceClient<CallOptionsExt = {}> {
  pretty(request: DeepPartial<PrettyRequest>, options?: CallOptions & CallOptionsExt): Promise<PrettyResponse>;
  query(request: DeepPartial<QueryRequest>, options?: CallOptions & CallOptionsExt): Promise<QueryResponse>;
  export(request: DeepPartial<ExportRequest>, options?: CallOptions & CallOptionsExt): Promise<ExportResponse>;
  adminExecute(
    request: AsyncIterable<DeepPartial<AdminExecuteRequest>>,
    options?: CallOptions & CallOptionsExt,
  ): AsyncIterable<AdminExecuteResponse>;
  differPreview(
    request: DeepPartial<DifferPreviewRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<DifferPreviewResponse>;
}

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

function bytesFromBase64(b64: string): Uint8Array {
  if (tsProtoGlobalThis.Buffer) {
    return Uint8Array.from(tsProtoGlobalThis.Buffer.from(b64, "base64"));
  } else {
    const bin = tsProtoGlobalThis.atob(b64);
    const arr = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; ++i) {
      arr[i] = bin.charCodeAt(i);
    }
    return arr;
  }
}

function base64FromBytes(arr: Uint8Array): string {
  if (tsProtoGlobalThis.Buffer) {
    return tsProtoGlobalThis.Buffer.from(arr).toString("base64");
  } else {
    const bin: string[] = [];
    arr.forEach((byte) => {
      bin.push(String.fromCharCode(byte));
    });
    return tsProtoGlobalThis.btoa(bin.join(""));
  }
}

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

export type ServerStreamingMethodResult<Response> = { [Symbol.asyncIterator](): AsyncIterator<Response, void> };
