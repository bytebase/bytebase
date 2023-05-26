/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Engine, engineFromJSON, engineToJSON } from "./common";

export const protobufPackage = "bytebase.v1";

export enum Status {
  STATUS_SUCCESS = 0,
  STATUS_WARN = 1,
  STATUS_ERROR = 2,
  UNRECOGNIZED = -1,
}

export function statusFromJSON(object: any): Status {
  switch (object) {
    case 0:
    case "STATUS_SUCCESS":
      return Status.STATUS_SUCCESS;
    case 1:
    case "STATUS_WARN":
      return Status.STATUS_WARN;
    case 2:
    case "STATUS_ERROR":
      return Status.STATUS_ERROR;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Status.UNRECOGNIZED;
  }
}

export function statusToJSON(object: Status): string {
  switch (object) {
    case Status.STATUS_SUCCESS:
      return "STATUS_SUCCESS";
    case Status.STATUS_WARN:
      return "STATUS_WARN";
    case Status.STATUS_ERROR:
      return "STATUS_ERROR";
    case Status.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface QueryRequest {
  /**
   * The instance to execute the query against.
   * Format: instances/{instance}
   */
  instance: string;
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
}

export interface QueryResponse {
  /** The query results. */
  results: QueryResult[];
  /** The query advices. */
  advices: Advice[];
}

export interface QueryResult {
  /** The column names of the query result. */
  columnNames: string[];
  /** The column types of the query result. */
  columnTypeNames: string[];
  /** The data of the query result. */
  data: string[];
  /** The column is masked or not. */
  masked: boolean[];
  /** The error message if the query failed. */
  error: string;
}

export interface Advice {
  /** The advice status. */
  status: Status;
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

function createBaseQueryRequest(): QueryRequest {
  return { instance: "", connectionDatabase: "", statement: "", limit: 0 };
}

export const QueryRequest = {
  encode(message: QueryRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.instance !== "") {
      writer.uint32(10).string(message.instance);
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

          message.instance = reader.string();
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
      instance: isSet(object.instance) ? String(object.instance) : "",
      connectionDatabase: isSet(object.connectionDatabase) ? String(object.connectionDatabase) : "",
      statement: isSet(object.statement) ? String(object.statement) : "",
      limit: isSet(object.limit) ? Number(object.limit) : 0,
    };
  },

  toJSON(message: QueryRequest): unknown {
    const obj: any = {};
    message.instance !== undefined && (obj.instance = message.instance);
    message.connectionDatabase !== undefined && (obj.connectionDatabase = message.connectionDatabase);
    message.statement !== undefined && (obj.statement = message.statement);
    message.limit !== undefined && (obj.limit = Math.round(message.limit));
    return obj;
  },

  create(base?: DeepPartial<QueryRequest>): QueryRequest {
    return QueryRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<QueryRequest>): QueryRequest {
    const message = createBaseQueryRequest();
    message.instance = object.instance ?? "";
    message.connectionDatabase = object.connectionDatabase ?? "";
    message.statement = object.statement ?? "";
    message.limit = object.limit ?? 0;
    return message;
  },
};

function createBaseQueryResponse(): QueryResponse {
  return { results: [], advices: [] };
}

export const QueryResponse = {
  encode(message: QueryResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.results) {
      QueryResult.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.advices) {
      Advice.encode(v!, writer.uint32(18).fork()).ldelim();
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
    return obj;
  },

  create(base?: DeepPartial<QueryResponse>): QueryResponse {
    return QueryResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<QueryResponse>): QueryResponse {
    const message = createBaseQueryResponse();
    message.results = object.results?.map((e) => QueryResult.fromPartial(e)) || [];
    message.advices = object.advices?.map((e) => Advice.fromPartial(e)) || [];
    return message;
  },
};

function createBaseQueryResult(): QueryResult {
  return { columnNames: [], columnTypeNames: [], data: [], masked: [], error: "" };
}

export const QueryResult = {
  encode(message: QueryResult, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.columnNames) {
      writer.uint32(10).string(v!);
    }
    for (const v of message.columnTypeNames) {
      writer.uint32(18).string(v!);
    }
    for (const v of message.data) {
      writer.uint32(26).string(v!);
    }
    writer.uint32(34).fork();
    for (const v of message.masked) {
      writer.bool(v);
    }
    writer.ldelim();
    if (message.error !== "") {
      writer.uint32(42).string(message.error);
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

          message.data.push(reader.string());
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
          if (tag !== 42) {
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

  fromJSON(object: any): QueryResult {
    return {
      columnNames: Array.isArray(object?.columnNames) ? object.columnNames.map((e: any) => String(e)) : [],
      columnTypeNames: Array.isArray(object?.columnTypeNames) ? object.columnTypeNames.map((e: any) => String(e)) : [],
      data: Array.isArray(object?.data) ? object.data.map((e: any) => String(e)) : [],
      masked: Array.isArray(object?.masked) ? object.masked.map((e: any) => Boolean(e)) : [],
      error: isSet(object.error) ? String(object.error) : "",
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
    if (message.data) {
      obj.data = message.data.map((e) => e);
    } else {
      obj.data = [];
    }
    if (message.masked) {
      obj.masked = message.masked.map((e) => e);
    } else {
      obj.masked = [];
    }
    message.error !== undefined && (obj.error = message.error);
    return obj;
  },

  create(base?: DeepPartial<QueryResult>): QueryResult {
    return QueryResult.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<QueryResult>): QueryResult {
    const message = createBaseQueryResult();
    message.columnNames = object.columnNames?.map((e) => e) || [];
    message.columnTypeNames = object.columnTypeNames?.map((e) => e) || [];
    message.data = object.data?.map((e) => e) || [];
    message.masked = object.masked?.map((e) => e) || [];
    message.error = object.error ?? "";
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
      status: isSet(object.status) ? statusFromJSON(object.status) : 0,
      code: isSet(object.code) ? Number(object.code) : 0,
      title: isSet(object.title) ? String(object.title) : "",
      content: isSet(object.content) ? String(object.content) : "",
      line: isSet(object.line) ? Number(object.line) : 0,
      detail: isSet(object.detail) ? String(object.detail) : "",
    };
  },

  toJSON(message: Advice): unknown {
    const obj: any = {};
    message.status !== undefined && (obj.status = statusToJSON(message.status));
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
          8410: [new Uint8Array([8, 105, 110, 115, 116, 97, 110, 99, 101])],
          578365826: [
            new Uint8Array([
              37,
              58,
              1,
              42,
              34,
              32,
              47,
              118,
              49,
              47,
              123,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
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
  },
} as const;

export interface SQLServiceImplementation<CallContextExt = {}> {
  pretty(request: PrettyRequest, context: CallContext & CallContextExt): Promise<DeepPartial<PrettyResponse>>;
  query(request: QueryRequest, context: CallContext & CallContextExt): Promise<DeepPartial<QueryResponse>>;
}

export interface SQLServiceClient<CallOptionsExt = {}> {
  pretty(request: DeepPartial<PrettyRequest>, options?: CallOptions & CallOptionsExt): Promise<PrettyResponse>;
  query(request: DeepPartial<QueryRequest>, options?: CallOptions & CallOptionsExt): Promise<QueryResponse>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
