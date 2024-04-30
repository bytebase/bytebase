/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Duration } from "../google/protobuf/duration";
import { NullValue, nullValueFromJSON, nullValueToJSON, Value } from "../google/protobuf/struct";
import { Timestamp } from "../google/protobuf/timestamp";
import { Engine, engineFromJSON, engineToJSON, ExportFormat, exportFormatFromJSON, exportFormatToJSON } from "./common";
import { DatabaseMetadata } from "./database_service";

export const protobufPackage = "bytebase.v1";

export interface ExecuteRequest {
  /**
   * The name is the instance name to execute the query against.
   * Format: instances/{instance}/databases/{databaseName}
   */
  name: string;
  /** The SQL statement to execute. */
  statement: string;
  /** The maximum number of rows to return. */
  limit: number;
  /** The timeout for the request. */
  timeout: Duration | undefined;
}

export interface ExecuteResponse {
  /** The execute results. */
  results: QueryResult[];
  /** The execute advices. */
  advices: Advice[];
}

export interface AdminExecuteRequest {
  /**
   * The name is the instance name to execute the query against.
   * Format: instances/{instance}/databases/{databaseName}
   */
  name: string;
  /** @deprecated */
  connectionDatabase: string;
  /** The SQL statement to execute. */
  statement: string;
  /** The maximum number of rows to return. */
  limit: number;
  /** The timeout for the request. */
  timeout: Duration | undefined;
}

export interface AdminExecuteResponse {
  /** The query results. */
  results: QueryResult[];
}

export interface QueryRequest {
  /**
   * The name is the instance name to execute the query against.
   * Format: instances/{instance}/databases/{databaseName}
   */
  name: string;
  /** @deprecated */
  connectionDatabase: string;
  /** The SQL statement to execute. */
  statement: string;
  /** The maximum number of rows to return. */
  limit: number;
  /** The timeout for the request. */
  timeout?:
    | Duration
    | undefined;
  /**
   * The id of data source.
   * It is used for querying admin data source even if the instance has read-only data sources.
   * Or it can be used to query a specific read-only data source.
   */
  dataSourceId: string;
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
  latency:
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
  int64Value?: Long | undefined;
  stringValue?: string | undefined;
  uint32Value?: number | undefined;
  uint64Value?:
    | Long
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
  /** The advice column number in the SQL statement. */
  column: number;
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

export interface ExportRequest {
  /**
   * The name is the instance name to execute the query against.
   * Format: instances/{instance}/databases/{databaseName}
   * Format: projects/{project}/issues/{issue} for data export issue.
   */
  name: string;
  /** @deprecated */
  connectionDatabase: string;
  /** The SQL statement to execute. */
  statement: string;
  /** The maximum number of rows to return. */
  limit: number;
  /** The export format. */
  format: ExportFormat;
  /**
   * The admin is used for workspace owner and DBA for exporting data from SQL Editor Admin mode.
   * The exported data is not masked.
   */
  admin: boolean;
  /** The zip password provide by users. */
  password: string;
}

export interface ExportResponse {
  /** The export file content. */
  content: Uint8Array;
}

export interface DifferPreviewRequest {
  engine: Engine;
  oldSchema: string;
  newMetadata: DatabaseMetadata | undefined;
}

export interface DifferPreviewResponse {
  schema: string;
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

export interface CheckRequest {
  statement: string;
  /**
   * The database name to check against.
   * Format: instances/{instance}/databases/{databaseName}
   */
  database: string;
  /**
   * The database metadata to check against. It can be used to check against an uncommitted metadata.
   * If not provided, the database metadata will be fetched from the database.
   */
  metadata: DatabaseMetadata | undefined;
  changeType: CheckRequest_ChangeType;
}

export enum CheckRequest_ChangeType {
  CHANGE_TYPE_UNSPECIFIED = 0,
  DDL = 1,
  DDL_GHOST = 2,
  DML = 3,
  UNRECOGNIZED = -1,
}

export function checkRequest_ChangeTypeFromJSON(object: any): CheckRequest_ChangeType {
  switch (object) {
    case 0:
    case "CHANGE_TYPE_UNSPECIFIED":
      return CheckRequest_ChangeType.CHANGE_TYPE_UNSPECIFIED;
    case 1:
    case "DDL":
      return CheckRequest_ChangeType.DDL;
    case 2:
    case "DDL_GHOST":
      return CheckRequest_ChangeType.DDL_GHOST;
    case 3:
    case "DML":
      return CheckRequest_ChangeType.DML;
    case -1:
    case "UNRECOGNIZED":
    default:
      return CheckRequest_ChangeType.UNRECOGNIZED;
  }
}

export function checkRequest_ChangeTypeToJSON(object: CheckRequest_ChangeType): string {
  switch (object) {
    case CheckRequest_ChangeType.CHANGE_TYPE_UNSPECIFIED:
      return "CHANGE_TYPE_UNSPECIFIED";
    case CheckRequest_ChangeType.DDL:
      return "DDL";
    case CheckRequest_ChangeType.DDL_GHOST:
      return "DDL_GHOST";
    case CheckRequest_ChangeType.DML:
      return "DML";
    case CheckRequest_ChangeType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface CheckResponse {
  advices: Advice[];
}

export interface ParseMyBatisMapperRequest {
  content: Uint8Array;
}

export interface ParseMyBatisMapperResponse {
  statements: string[];
}

export interface StringifyMetadataRequest {
  metadata:
    | DatabaseMetadata
    | undefined;
  /** The database engine of the schema string. */
  engine: Engine;
}

export interface StringifyMetadataResponse {
  schema: string;
}

export interface SearchQueryHistoriesRequest {
  /**
   * Not used. The maximum number of histories to return.
   * The service may return fewer than this value.
   * If unspecified, at most 100 history entries will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * Not used. A page token, received from a previous `ListQueryHistory` call.
   * Provide this to retrieve the subsequent page.
   */
  pageToken: string;
  /**
   * filter is the filter to apply on the search query history,
   * follow the [ebnf](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form) syntax.
   * Support filter by:
   * - database, for example:
   *    database = "instances/{instance}/databases/{database}"
   * - instance, for example:
   *    instance = "instance/{instance}"
   * - type, for example:
   *    type = "QUERY"
   */
  filter: string;
}

export interface SearchQueryHistoriesResponse {
  /** The list of history. */
  queryHistories: QueryHistory[];
  /**
   * A token to retrieve next page of history.
   * Pass this value in the page_token field in the subsequent call to `ListQueryHistory` method
   * to retrieve the next page of history.
   */
  nextPageToken: string;
}

export interface QueryHistory {
  /**
   * The name for the query history.
   * Format: queryHistories/{uid}
   */
  name: string;
  /**
   * The database name to execute the query.
   * Format: instances/{instance}/databases/{databaseName}
   */
  database: string;
  creator: string;
  createTime: Date | undefined;
  statement: string;
  error?: string | undefined;
  duration: Duration | undefined;
  type: QueryHistory_Type;
}

export enum QueryHistory_Type {
  TYPE_UNSPECIFIED = 0,
  QUERY = 1,
  EXPORT = 2,
  UNRECOGNIZED = -1,
}

export function queryHistory_TypeFromJSON(object: any): QueryHistory_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return QueryHistory_Type.TYPE_UNSPECIFIED;
    case 1:
    case "QUERY":
      return QueryHistory_Type.QUERY;
    case 2:
    case "EXPORT":
      return QueryHistory_Type.EXPORT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return QueryHistory_Type.UNRECOGNIZED;
  }
}

export function queryHistory_TypeToJSON(object: QueryHistory_Type): string {
  switch (object) {
    case QueryHistory_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case QueryHistory_Type.QUERY:
      return "QUERY";
    case QueryHistory_Type.EXPORT:
      return "EXPORT";
    case QueryHistory_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

function createBaseExecuteRequest(): ExecuteRequest {
  return { name: "", statement: "", limit: 0, timeout: undefined };
}

export const ExecuteRequest = {
  encode(message: ExecuteRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.statement !== "") {
      writer.uint32(18).string(message.statement);
    }
    if (message.limit !== 0) {
      writer.uint32(24).int32(message.limit);
    }
    if (message.timeout !== undefined) {
      Duration.encode(message.timeout, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ExecuteRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExecuteRequest();
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

          message.statement = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.limit = reader.int32();
          continue;
        case 4:
          if (tag !== 34) {
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

  fromJSON(object: any): ExecuteRequest {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      statement: isSet(object.statement) ? globalThis.String(object.statement) : "",
      limit: isSet(object.limit) ? globalThis.Number(object.limit) : 0,
      timeout: isSet(object.timeout) ? Duration.fromJSON(object.timeout) : undefined,
    };
  },

  toJSON(message: ExecuteRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.statement !== "") {
      obj.statement = message.statement;
    }
    if (message.limit !== 0) {
      obj.limit = Math.round(message.limit);
    }
    if (message.timeout !== undefined) {
      obj.timeout = Duration.toJSON(message.timeout);
    }
    return obj;
  },

  create(base?: DeepPartial<ExecuteRequest>): ExecuteRequest {
    return ExecuteRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ExecuteRequest>): ExecuteRequest {
    const message = createBaseExecuteRequest();
    message.name = object.name ?? "";
    message.statement = object.statement ?? "";
    message.limit = object.limit ?? 0;
    message.timeout = (object.timeout !== undefined && object.timeout !== null)
      ? Duration.fromPartial(object.timeout)
      : undefined;
    return message;
  },
};

function createBaseExecuteResponse(): ExecuteResponse {
  return { results: [], advices: [] };
}

export const ExecuteResponse = {
  encode(message: ExecuteResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.results) {
      QueryResult.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.advices) {
      Advice.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ExecuteResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExecuteResponse();
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
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ExecuteResponse {
    return {
      results: globalThis.Array.isArray(object?.results) ? object.results.map((e: any) => QueryResult.fromJSON(e)) : [],
      advices: globalThis.Array.isArray(object?.advices) ? object.advices.map((e: any) => Advice.fromJSON(e)) : [],
    };
  },

  toJSON(message: ExecuteResponse): unknown {
    const obj: any = {};
    if (message.results?.length) {
      obj.results = message.results.map((e) => QueryResult.toJSON(e));
    }
    if (message.advices?.length) {
      obj.advices = message.advices.map((e) => Advice.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<ExecuteResponse>): ExecuteResponse {
    return ExecuteResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ExecuteResponse>): ExecuteResponse {
    const message = createBaseExecuteResponse();
    message.results = object.results?.map((e) => QueryResult.fromPartial(e)) || [];
    message.advices = object.advices?.map((e) => Advice.fromPartial(e)) || [];
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
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      connectionDatabase: isSet(object.connectionDatabase) ? globalThis.String(object.connectionDatabase) : "",
      statement: isSet(object.statement) ? globalThis.String(object.statement) : "",
      limit: isSet(object.limit) ? globalThis.Number(object.limit) : 0,
      timeout: isSet(object.timeout) ? Duration.fromJSON(object.timeout) : undefined,
    };
  },

  toJSON(message: AdminExecuteRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.connectionDatabase !== "") {
      obj.connectionDatabase = message.connectionDatabase;
    }
    if (message.statement !== "") {
      obj.statement = message.statement;
    }
    if (message.limit !== 0) {
      obj.limit = Math.round(message.limit);
    }
    if (message.timeout !== undefined) {
      obj.timeout = Duration.toJSON(message.timeout);
    }
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
    return {
      results: globalThis.Array.isArray(object?.results) ? object.results.map((e: any) => QueryResult.fromJSON(e)) : [],
    };
  },

  toJSON(message: AdminExecuteResponse): unknown {
    const obj: any = {};
    if (message.results?.length) {
      obj.results = message.results.map((e) => QueryResult.toJSON(e));
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

function createBaseQueryRequest(): QueryRequest {
  return { name: "", connectionDatabase: "", statement: "", limit: 0, timeout: undefined, dataSourceId: "" };
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
    if (message.dataSourceId !== "") {
      writer.uint32(50).string(message.dataSourceId);
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
        case 6:
          if (tag !== 50) {
            break;
          }

          message.dataSourceId = reader.string();
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
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      connectionDatabase: isSet(object.connectionDatabase) ? globalThis.String(object.connectionDatabase) : "",
      statement: isSet(object.statement) ? globalThis.String(object.statement) : "",
      limit: isSet(object.limit) ? globalThis.Number(object.limit) : 0,
      timeout: isSet(object.timeout) ? Duration.fromJSON(object.timeout) : undefined,
      dataSourceId: isSet(object.dataSourceId) ? globalThis.String(object.dataSourceId) : "",
    };
  },

  toJSON(message: QueryRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.connectionDatabase !== "") {
      obj.connectionDatabase = message.connectionDatabase;
    }
    if (message.statement !== "") {
      obj.statement = message.statement;
    }
    if (message.limit !== 0) {
      obj.limit = Math.round(message.limit);
    }
    if (message.timeout !== undefined) {
      obj.timeout = Duration.toJSON(message.timeout);
    }
    if (message.dataSourceId !== "") {
      obj.dataSourceId = message.dataSourceId;
    }
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
    message.dataSourceId = object.dataSourceId ?? "";
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
      results: globalThis.Array.isArray(object?.results) ? object.results.map((e: any) => QueryResult.fromJSON(e)) : [],
      advices: globalThis.Array.isArray(object?.advices) ? object.advices.map((e: any) => Advice.fromJSON(e)) : [],
      allowExport: isSet(object.allowExport) ? globalThis.Boolean(object.allowExport) : false,
    };
  },

  toJSON(message: QueryResponse): unknown {
    const obj: any = {};
    if (message.results?.length) {
      obj.results = message.results.map((e) => QueryResult.toJSON(e));
    }
    if (message.advices?.length) {
      obj.advices = message.advices.map((e) => Advice.toJSON(e));
    }
    if (message.allowExport === true) {
      obj.allowExport = message.allowExport;
    }
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
      columnNames: globalThis.Array.isArray(object?.columnNames)
        ? object.columnNames.map((e: any) => globalThis.String(e))
        : [],
      columnTypeNames: globalThis.Array.isArray(object?.columnTypeNames)
        ? object.columnTypeNames.map((e: any) => globalThis.String(e))
        : [],
      rows: globalThis.Array.isArray(object?.rows) ? object.rows.map((e: any) => QueryRow.fromJSON(e)) : [],
      masked: globalThis.Array.isArray(object?.masked) ? object.masked.map((e: any) => globalThis.Boolean(e)) : [],
      sensitive: globalThis.Array.isArray(object?.sensitive)
        ? object.sensitive.map((e: any) => globalThis.Boolean(e))
        : [],
      error: isSet(object.error) ? globalThis.String(object.error) : "",
      latency: isSet(object.latency) ? Duration.fromJSON(object.latency) : undefined,
      statement: isSet(object.statement) ? globalThis.String(object.statement) : "",
    };
  },

  toJSON(message: QueryResult): unknown {
    const obj: any = {};
    if (message.columnNames?.length) {
      obj.columnNames = message.columnNames;
    }
    if (message.columnTypeNames?.length) {
      obj.columnTypeNames = message.columnTypeNames;
    }
    if (message.rows?.length) {
      obj.rows = message.rows.map((e) => QueryRow.toJSON(e));
    }
    if (message.masked?.length) {
      obj.masked = message.masked;
    }
    if (message.sensitive?.length) {
      obj.sensitive = message.sensitive;
    }
    if (message.error !== "") {
      obj.error = message.error;
    }
    if (message.latency !== undefined) {
      obj.latency = Duration.toJSON(message.latency);
    }
    if (message.statement !== "") {
      obj.statement = message.statement;
    }
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
    return {
      values: globalThis.Array.isArray(object?.values) ? object.values.map((e: any) => RowValue.fromJSON(e)) : [],
    };
  },

  toJSON(message: QueryRow): unknown {
    const obj: any = {};
    if (message.values?.length) {
      obj.values = message.values.map((e) => RowValue.toJSON(e));
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

          message.int64Value = reader.int64() as Long;
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

          message.uint64Value = reader.uint64() as Long;
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
      boolValue: isSet(object.boolValue) ? globalThis.Boolean(object.boolValue) : undefined,
      bytesValue: isSet(object.bytesValue) ? bytesFromBase64(object.bytesValue) : undefined,
      doubleValue: isSet(object.doubleValue) ? globalThis.Number(object.doubleValue) : undefined,
      floatValue: isSet(object.floatValue) ? globalThis.Number(object.floatValue) : undefined,
      int32Value: isSet(object.int32Value) ? globalThis.Number(object.int32Value) : undefined,
      int64Value: isSet(object.int64Value) ? Long.fromValue(object.int64Value) : undefined,
      stringValue: isSet(object.stringValue) ? globalThis.String(object.stringValue) : undefined,
      uint32Value: isSet(object.uint32Value) ? globalThis.Number(object.uint32Value) : undefined,
      uint64Value: isSet(object.uint64Value) ? Long.fromValue(object.uint64Value) : undefined,
      valueValue: isSet(object?.valueValue) ? object.valueValue : undefined,
    };
  },

  toJSON(message: RowValue): unknown {
    const obj: any = {};
    if (message.nullValue !== undefined) {
      obj.nullValue = nullValueToJSON(message.nullValue);
    }
    if (message.boolValue !== undefined) {
      obj.boolValue = message.boolValue;
    }
    if (message.bytesValue !== undefined) {
      obj.bytesValue = base64FromBytes(message.bytesValue);
    }
    if (message.doubleValue !== undefined) {
      obj.doubleValue = message.doubleValue;
    }
    if (message.floatValue !== undefined) {
      obj.floatValue = message.floatValue;
    }
    if (message.int32Value !== undefined) {
      obj.int32Value = Math.round(message.int32Value);
    }
    if (message.int64Value !== undefined) {
      obj.int64Value = (message.int64Value || Long.ZERO).toString();
    }
    if (message.stringValue !== undefined) {
      obj.stringValue = message.stringValue;
    }
    if (message.uint32Value !== undefined) {
      obj.uint32Value = Math.round(message.uint32Value);
    }
    if (message.uint64Value !== undefined) {
      obj.uint64Value = (message.uint64Value || Long.UZERO).toString();
    }
    if (message.valueValue !== undefined) {
      obj.valueValue = message.valueValue;
    }
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
    message.int64Value = (object.int64Value !== undefined && object.int64Value !== null)
      ? Long.fromValue(object.int64Value)
      : undefined;
    message.stringValue = object.stringValue ?? undefined;
    message.uint32Value = object.uint32Value ?? undefined;
    message.uint64Value = (object.uint64Value !== undefined && object.uint64Value !== null)
      ? Long.fromValue(object.uint64Value)
      : undefined;
    message.valueValue = object.valueValue ?? undefined;
    return message;
  },
};

function createBaseAdvice(): Advice {
  return { status: 0, code: 0, title: "", content: "", line: 0, column: 0, detail: "" };
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
    if (message.column !== 0) {
      writer.uint32(48).int32(message.column);
    }
    if (message.detail !== "") {
      writer.uint32(58).string(message.detail);
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
          if (tag !== 48) {
            break;
          }

          message.column = reader.int32();
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

  fromJSON(object: any): Advice {
    return {
      status: isSet(object.status) ? advice_StatusFromJSON(object.status) : 0,
      code: isSet(object.code) ? globalThis.Number(object.code) : 0,
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      content: isSet(object.content) ? globalThis.String(object.content) : "",
      line: isSet(object.line) ? globalThis.Number(object.line) : 0,
      column: isSet(object.column) ? globalThis.Number(object.column) : 0,
      detail: isSet(object.detail) ? globalThis.String(object.detail) : "",
    };
  },

  toJSON(message: Advice): unknown {
    const obj: any = {};
    if (message.status !== 0) {
      obj.status = advice_StatusToJSON(message.status);
    }
    if (message.code !== 0) {
      obj.code = Math.round(message.code);
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.content !== "") {
      obj.content = message.content;
    }
    if (message.line !== 0) {
      obj.line = Math.round(message.line);
    }
    if (message.column !== 0) {
      obj.column = Math.round(message.column);
    }
    if (message.detail !== "") {
      obj.detail = message.detail;
    }
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
    message.column = object.column ?? 0;
    message.detail = object.detail ?? "";
    return message;
  },
};

function createBaseExportRequest(): ExportRequest {
  return { name: "", connectionDatabase: "", statement: "", limit: 0, format: 0, admin: false, password: "" };
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
    if (message.password !== "") {
      writer.uint32(58).string(message.password);
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
        case 7:
          if (tag !== 58) {
            break;
          }

          message.password = reader.string();
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
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      connectionDatabase: isSet(object.connectionDatabase) ? globalThis.String(object.connectionDatabase) : "",
      statement: isSet(object.statement) ? globalThis.String(object.statement) : "",
      limit: isSet(object.limit) ? globalThis.Number(object.limit) : 0,
      format: isSet(object.format) ? exportFormatFromJSON(object.format) : 0,
      admin: isSet(object.admin) ? globalThis.Boolean(object.admin) : false,
      password: isSet(object.password) ? globalThis.String(object.password) : "",
    };
  },

  toJSON(message: ExportRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.connectionDatabase !== "") {
      obj.connectionDatabase = message.connectionDatabase;
    }
    if (message.statement !== "") {
      obj.statement = message.statement;
    }
    if (message.limit !== 0) {
      obj.limit = Math.round(message.limit);
    }
    if (message.format !== 0) {
      obj.format = exportFormatToJSON(message.format);
    }
    if (message.admin === true) {
      obj.admin = message.admin;
    }
    if (message.password !== "") {
      obj.password = message.password;
    }
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
    message.password = object.password ?? "";
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
    if (message.content.length !== 0) {
      obj.content = base64FromBytes(message.content);
    }
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
      oldSchema: isSet(object.oldSchema) ? globalThis.String(object.oldSchema) : "",
      newMetadata: isSet(object.newMetadata) ? DatabaseMetadata.fromJSON(object.newMetadata) : undefined,
    };
  },

  toJSON(message: DifferPreviewRequest): unknown {
    const obj: any = {};
    if (message.engine !== 0) {
      obj.engine = engineToJSON(message.engine);
    }
    if (message.oldSchema !== "") {
      obj.oldSchema = message.oldSchema;
    }
    if (message.newMetadata !== undefined) {
      obj.newMetadata = DatabaseMetadata.toJSON(message.newMetadata);
    }
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
    return { schema: isSet(object.schema) ? globalThis.String(object.schema) : "" };
  },

  toJSON(message: DifferPreviewResponse): unknown {
    const obj: any = {};
    if (message.schema !== "") {
      obj.schema = message.schema;
    }
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
      currentSchema: isSet(object.currentSchema) ? globalThis.String(object.currentSchema) : "",
      expectedSchema: isSet(object.expectedSchema) ? globalThis.String(object.expectedSchema) : "",
    };
  },

  toJSON(message: PrettyRequest): unknown {
    const obj: any = {};
    if (message.engine !== 0) {
      obj.engine = engineToJSON(message.engine);
    }
    if (message.currentSchema !== "") {
      obj.currentSchema = message.currentSchema;
    }
    if (message.expectedSchema !== "") {
      obj.expectedSchema = message.expectedSchema;
    }
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
      currentSchema: isSet(object.currentSchema) ? globalThis.String(object.currentSchema) : "",
      expectedSchema: isSet(object.expectedSchema) ? globalThis.String(object.expectedSchema) : "",
    };
  },

  toJSON(message: PrettyResponse): unknown {
    const obj: any = {};
    if (message.currentSchema !== "") {
      obj.currentSchema = message.currentSchema;
    }
    if (message.expectedSchema !== "") {
      obj.expectedSchema = message.expectedSchema;
    }
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

function createBaseCheckRequest(): CheckRequest {
  return { statement: "", database: "", metadata: undefined, changeType: 0 };
}

export const CheckRequest = {
  encode(message: CheckRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.statement !== "") {
      writer.uint32(10).string(message.statement);
    }
    if (message.database !== "") {
      writer.uint32(18).string(message.database);
    }
    if (message.metadata !== undefined) {
      DatabaseMetadata.encode(message.metadata, writer.uint32(26).fork()).ldelim();
    }
    if (message.changeType !== 0) {
      writer.uint32(32).int32(message.changeType);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CheckRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCheckRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.statement = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.database = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.metadata = DatabaseMetadata.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.changeType = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CheckRequest {
    return {
      statement: isSet(object.statement) ? globalThis.String(object.statement) : "",
      database: isSet(object.database) ? globalThis.String(object.database) : "",
      metadata: isSet(object.metadata) ? DatabaseMetadata.fromJSON(object.metadata) : undefined,
      changeType: isSet(object.changeType) ? checkRequest_ChangeTypeFromJSON(object.changeType) : 0,
    };
  },

  toJSON(message: CheckRequest): unknown {
    const obj: any = {};
    if (message.statement !== "") {
      obj.statement = message.statement;
    }
    if (message.database !== "") {
      obj.database = message.database;
    }
    if (message.metadata !== undefined) {
      obj.metadata = DatabaseMetadata.toJSON(message.metadata);
    }
    if (message.changeType !== 0) {
      obj.changeType = checkRequest_ChangeTypeToJSON(message.changeType);
    }
    return obj;
  },

  create(base?: DeepPartial<CheckRequest>): CheckRequest {
    return CheckRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<CheckRequest>): CheckRequest {
    const message = createBaseCheckRequest();
    message.statement = object.statement ?? "";
    message.database = object.database ?? "";
    message.metadata = (object.metadata !== undefined && object.metadata !== null)
      ? DatabaseMetadata.fromPartial(object.metadata)
      : undefined;
    message.changeType = object.changeType ?? 0;
    return message;
  },
};

function createBaseCheckResponse(): CheckResponse {
  return { advices: [] };
}

export const CheckResponse = {
  encode(message: CheckResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.advices) {
      Advice.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CheckResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCheckResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.advices.push(Advice.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CheckResponse {
    return {
      advices: globalThis.Array.isArray(object?.advices) ? object.advices.map((e: any) => Advice.fromJSON(e)) : [],
    };
  },

  toJSON(message: CheckResponse): unknown {
    const obj: any = {};
    if (message.advices?.length) {
      obj.advices = message.advices.map((e) => Advice.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<CheckResponse>): CheckResponse {
    return CheckResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<CheckResponse>): CheckResponse {
    const message = createBaseCheckResponse();
    message.advices = object.advices?.map((e) => Advice.fromPartial(e)) || [];
    return message;
  },
};

function createBaseParseMyBatisMapperRequest(): ParseMyBatisMapperRequest {
  return { content: new Uint8Array(0) };
}

export const ParseMyBatisMapperRequest = {
  encode(message: ParseMyBatisMapperRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.content.length !== 0) {
      writer.uint32(10).bytes(message.content);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ParseMyBatisMapperRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseParseMyBatisMapperRequest();
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

  fromJSON(object: any): ParseMyBatisMapperRequest {
    return { content: isSet(object.content) ? bytesFromBase64(object.content) : new Uint8Array(0) };
  },

  toJSON(message: ParseMyBatisMapperRequest): unknown {
    const obj: any = {};
    if (message.content.length !== 0) {
      obj.content = base64FromBytes(message.content);
    }
    return obj;
  },

  create(base?: DeepPartial<ParseMyBatisMapperRequest>): ParseMyBatisMapperRequest {
    return ParseMyBatisMapperRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ParseMyBatisMapperRequest>): ParseMyBatisMapperRequest {
    const message = createBaseParseMyBatisMapperRequest();
    message.content = object.content ?? new Uint8Array(0);
    return message;
  },
};

function createBaseParseMyBatisMapperResponse(): ParseMyBatisMapperResponse {
  return { statements: [] };
}

export const ParseMyBatisMapperResponse = {
  encode(message: ParseMyBatisMapperResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.statements) {
      writer.uint32(10).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ParseMyBatisMapperResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseParseMyBatisMapperResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.statements.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ParseMyBatisMapperResponse {
    return {
      statements: globalThis.Array.isArray(object?.statements)
        ? object.statements.map((e: any) => globalThis.String(e))
        : [],
    };
  },

  toJSON(message: ParseMyBatisMapperResponse): unknown {
    const obj: any = {};
    if (message.statements?.length) {
      obj.statements = message.statements;
    }
    return obj;
  },

  create(base?: DeepPartial<ParseMyBatisMapperResponse>): ParseMyBatisMapperResponse {
    return ParseMyBatisMapperResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ParseMyBatisMapperResponse>): ParseMyBatisMapperResponse {
    const message = createBaseParseMyBatisMapperResponse();
    message.statements = object.statements?.map((e) => e) || [];
    return message;
  },
};

function createBaseStringifyMetadataRequest(): StringifyMetadataRequest {
  return { metadata: undefined, engine: 0 };
}

export const StringifyMetadataRequest = {
  encode(message: StringifyMetadataRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.metadata !== undefined) {
      DatabaseMetadata.encode(message.metadata, writer.uint32(10).fork()).ldelim();
    }
    if (message.engine !== 0) {
      writer.uint32(16).int32(message.engine);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): StringifyMetadataRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStringifyMetadataRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.metadata = DatabaseMetadata.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.engine = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): StringifyMetadataRequest {
    return {
      metadata: isSet(object.metadata) ? DatabaseMetadata.fromJSON(object.metadata) : undefined,
      engine: isSet(object.engine) ? engineFromJSON(object.engine) : 0,
    };
  },

  toJSON(message: StringifyMetadataRequest): unknown {
    const obj: any = {};
    if (message.metadata !== undefined) {
      obj.metadata = DatabaseMetadata.toJSON(message.metadata);
    }
    if (message.engine !== 0) {
      obj.engine = engineToJSON(message.engine);
    }
    return obj;
  },

  create(base?: DeepPartial<StringifyMetadataRequest>): StringifyMetadataRequest {
    return StringifyMetadataRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<StringifyMetadataRequest>): StringifyMetadataRequest {
    const message = createBaseStringifyMetadataRequest();
    message.metadata = (object.metadata !== undefined && object.metadata !== null)
      ? DatabaseMetadata.fromPartial(object.metadata)
      : undefined;
    message.engine = object.engine ?? 0;
    return message;
  },
};

function createBaseStringifyMetadataResponse(): StringifyMetadataResponse {
  return { schema: "" };
}

export const StringifyMetadataResponse = {
  encode(message: StringifyMetadataResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schema !== "") {
      writer.uint32(10).string(message.schema);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): StringifyMetadataResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStringifyMetadataResponse();
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

  fromJSON(object: any): StringifyMetadataResponse {
    return { schema: isSet(object.schema) ? globalThis.String(object.schema) : "" };
  },

  toJSON(message: StringifyMetadataResponse): unknown {
    const obj: any = {};
    if (message.schema !== "") {
      obj.schema = message.schema;
    }
    return obj;
  },

  create(base?: DeepPartial<StringifyMetadataResponse>): StringifyMetadataResponse {
    return StringifyMetadataResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<StringifyMetadataResponse>): StringifyMetadataResponse {
    const message = createBaseStringifyMetadataResponse();
    message.schema = object.schema ?? "";
    return message;
  },
};

function createBaseSearchQueryHistoriesRequest(): SearchQueryHistoriesRequest {
  return { pageSize: 0, pageToken: "", filter: "" };
}

export const SearchQueryHistoriesRequest = {
  encode(message: SearchQueryHistoriesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pageSize !== 0) {
      writer.uint32(8).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(18).string(message.pageToken);
    }
    if (message.filter !== "") {
      writer.uint32(26).string(message.filter);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SearchQueryHistoriesRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSearchQueryHistoriesRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.pageToken = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.filter = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SearchQueryHistoriesRequest {
    return {
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
      filter: isSet(object.filter) ? globalThis.String(object.filter) : "",
    };
  },

  toJSON(message: SearchQueryHistoriesRequest): unknown {
    const obj: any = {};
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    if (message.filter !== "") {
      obj.filter = message.filter;
    }
    return obj;
  },

  create(base?: DeepPartial<SearchQueryHistoriesRequest>): SearchQueryHistoriesRequest {
    return SearchQueryHistoriesRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SearchQueryHistoriesRequest>): SearchQueryHistoriesRequest {
    const message = createBaseSearchQueryHistoriesRequest();
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    message.filter = object.filter ?? "";
    return message;
  },
};

function createBaseSearchQueryHistoriesResponse(): SearchQueryHistoriesResponse {
  return { queryHistories: [], nextPageToken: "" };
}

export const SearchQueryHistoriesResponse = {
  encode(message: SearchQueryHistoriesResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.queryHistories) {
      QueryHistory.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SearchQueryHistoriesResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSearchQueryHistoriesResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.queryHistories.push(QueryHistory.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.nextPageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SearchQueryHistoriesResponse {
    return {
      queryHistories: globalThis.Array.isArray(object?.queryHistories)
        ? object.queryHistories.map((e: any) => QueryHistory.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: SearchQueryHistoriesResponse): unknown {
    const obj: any = {};
    if (message.queryHistories?.length) {
      obj.queryHistories = message.queryHistories.map((e) => QueryHistory.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<SearchQueryHistoriesResponse>): SearchQueryHistoriesResponse {
    return SearchQueryHistoriesResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SearchQueryHistoriesResponse>): SearchQueryHistoriesResponse {
    const message = createBaseSearchQueryHistoriesResponse();
    message.queryHistories = object.queryHistories?.map((e) => QueryHistory.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseQueryHistory(): QueryHistory {
  return {
    name: "",
    database: "",
    creator: "",
    createTime: undefined,
    statement: "",
    error: undefined,
    duration: undefined,
    type: 0,
  };
}

export const QueryHistory = {
  encode(message: QueryHistory, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.database !== "") {
      writer.uint32(18).string(message.database);
    }
    if (message.creator !== "") {
      writer.uint32(26).string(message.creator);
    }
    if (message.createTime !== undefined) {
      Timestamp.encode(toTimestamp(message.createTime), writer.uint32(34).fork()).ldelim();
    }
    if (message.statement !== "") {
      writer.uint32(42).string(message.statement);
    }
    if (message.error !== undefined) {
      writer.uint32(50).string(message.error);
    }
    if (message.duration !== undefined) {
      Duration.encode(message.duration, writer.uint32(58).fork()).ldelim();
    }
    if (message.type !== 0) {
      writer.uint32(64).int32(message.type);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryHistory {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryHistory();
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

          message.database = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.creator = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.createTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.statement = reader.string();
          continue;
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

          message.duration = Duration.decode(reader, reader.uint32());
          continue;
        case 8:
          if (tag !== 64) {
            break;
          }

          message.type = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryHistory {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      database: isSet(object.database) ? globalThis.String(object.database) : "",
      creator: isSet(object.creator) ? globalThis.String(object.creator) : "",
      createTime: isSet(object.createTime) ? fromJsonTimestamp(object.createTime) : undefined,
      statement: isSet(object.statement) ? globalThis.String(object.statement) : "",
      error: isSet(object.error) ? globalThis.String(object.error) : undefined,
      duration: isSet(object.duration) ? Duration.fromJSON(object.duration) : undefined,
      type: isSet(object.type) ? queryHistory_TypeFromJSON(object.type) : 0,
    };
  },

  toJSON(message: QueryHistory): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.database !== "") {
      obj.database = message.database;
    }
    if (message.creator !== "") {
      obj.creator = message.creator;
    }
    if (message.createTime !== undefined) {
      obj.createTime = message.createTime.toISOString();
    }
    if (message.statement !== "") {
      obj.statement = message.statement;
    }
    if (message.error !== undefined) {
      obj.error = message.error;
    }
    if (message.duration !== undefined) {
      obj.duration = Duration.toJSON(message.duration);
    }
    if (message.type !== 0) {
      obj.type = queryHistory_TypeToJSON(message.type);
    }
    return obj;
  },

  create(base?: DeepPartial<QueryHistory>): QueryHistory {
    return QueryHistory.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<QueryHistory>): QueryHistory {
    const message = createBaseQueryHistory();
    message.name = object.name ?? "";
    message.database = object.database ?? "";
    message.creator = object.creator ?? "";
    message.createTime = object.createTime ?? undefined;
    message.statement = object.statement ?? "";
    message.error = object.error ?? undefined;
    message.duration = (object.duration !== undefined && object.duration !== null)
      ? Duration.fromPartial(object.duration)
      : undefined;
    message.type = object.type ?? 0;
    return message;
  },
};

export type SQLServiceDefinition = typeof SQLServiceDefinition;
export const SQLServiceDefinition = {
  name: "SQLService",
  fullName: "bytebase.v1.SQLService",
  methods: {
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
              80,
              58,
              1,
              42,
              90,
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
              34,
              40,
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
              47,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
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
    execute: {
      name: "Execute",
      requestType: ExecuteRequest,
      requestStream: false,
      responseType: ExecuteResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              84,
              58,
              1,
              42,
              90,
              35,
              58,
              1,
              42,
              34,
              30,
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
              101,
              99,
              117,
              116,
              101,
              34,
              42,
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
              47,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              115,
              47,
              42,
              125,
              58,
              101,
              120,
              101,
              99,
              117,
              116,
              101,
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
    searchQueryHistories: {
      name: "SearchQueryHistories",
      requestType: SearchQueryHistoriesRequest,
      requestStream: false,
      responseType: SearchQueryHistoriesResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              27,
              18,
              25,
              47,
              118,
              49,
              47,
              113,
              117,
              101,
              114,
              121,
              72,
              105,
              115,
              116,
              111,
              114,
              105,
              101,
              115,
              58,
              115,
              101,
              97,
              114,
              99,
              104,
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
              126,
              58,
              1,
              42,
              90,
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
              90,
              42,
              58,
              1,
              42,
              34,
              37,
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
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              47,
              105,
              115,
              115,
              117,
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
              34,
              41,
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
              47,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
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
    check: {
      name: "Check",
      requestType: CheckRequest,
      requestStream: false,
      responseType: CheckResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([18, 58, 1, 42, 34, 13, 47, 118, 49, 47, 115, 113, 108, 47, 99, 104, 101, 99, 107]),
          ],
        },
      },
    },
    parseMyBatisMapper: {
      name: "ParseMyBatisMapper",
      requestType: ParseMyBatisMapperRequest,
      requestStream: false,
      responseType: ParseMyBatisMapperResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              31,
              58,
              1,
              42,
              34,
              26,
              47,
              118,
              49,
              47,
              115,
              113,
              108,
              47,
              112,
              97,
              114,
              115,
              101,
              77,
              121,
              66,
              97,
              116,
              105,
              115,
              77,
              97,
              112,
              112,
              101,
              114,
            ]),
          ],
        },
      },
    },
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
    stringifyMetadata: {
      name: "StringifyMetadata",
      requestType: StringifyMetadataRequest,
      requestStream: false,
      responseType: StringifyMetadataResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              39,
              58,
              1,
              42,
              34,
              34,
              47,
              118,
              49,
              47,
              115,
              99,
              104,
              101,
              109,
              97,
              68,
              101,
              115,
              105,
              103,
              110,
              58,
              115,
              116,
              114,
              105,
              110,
              103,
              105,
              102,
              121,
              77,
              101,
              116,
              97,
              100,
              97,
              116,
              97,
            ]),
          ],
        },
      },
    },
  },
} as const;

function bytesFromBase64(b64: string): Uint8Array {
  if (globalThis.Buffer) {
    return Uint8Array.from(globalThis.Buffer.from(b64, "base64"));
  } else {
    const bin = globalThis.atob(b64);
    const arr = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; ++i) {
      arr[i] = bin.charCodeAt(i);
    }
    return arr;
  }
}

function base64FromBytes(arr: Uint8Array): string {
  if (globalThis.Buffer) {
    return globalThis.Buffer.from(arr).toString("base64");
  } else {
    const bin: string[] = [];
    arr.forEach((byte) => {
      bin.push(globalThis.String.fromCharCode(byte));
    });
    return globalThis.btoa(bin.join(""));
  }
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Long ? string | number | Long : T extends globalThis.Array<infer U> ? globalThis.Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function toTimestamp(date: Date): Timestamp {
  const seconds = numberToLong(date.getTime() / 1_000);
  const nanos = (date.getTime() % 1_000) * 1_000_000;
  return { seconds, nanos };
}

function fromTimestamp(t: Timestamp): Date {
  let millis = (t.seconds.toNumber() || 0) * 1_000;
  millis += (t.nanos || 0) / 1_000_000;
  return new globalThis.Date(millis);
}

function fromJsonTimestamp(o: any): Date {
  if (o instanceof globalThis.Date) {
    return o;
  } else if (typeof o === "string") {
    return new globalThis.Date(o);
  } else {
    return fromTimestamp(Timestamp.fromJSON(o));
  }
}

function numberToLong(number: number) {
  return Long.fromNumber(number);
}

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
