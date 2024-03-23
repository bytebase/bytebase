/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Timestamp } from "../google/protobuf/timestamp";

export const protobufPackage = "bytebase.v1";

export interface SearchAnomaliesRequest {
  /**
   * filter is the filter to apply on the search anomaly request,
   * follow the [ebnf](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form) syntax.
   * Only support filter by resource and type for now.
   * For example:
   * Search the anomalies of a specific resource: 'resource="instances/{instance}".'
   * Search the specified types of anomalies: 'type="MIGRATION_SCHEMA".'
   */
  filter: string;
  /**
   * Not used. The maximum number of anomalies to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 anomalies will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * Not used. A page token, received from a previous `SearchAnomalies` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `SearchAnomalies` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface SearchAnomaliesResponse {
  /** anomalies is the list of anomalies. */
  anomalies: Anomaly[];
  /**
   * Not used. A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface Anomaly {
  /**
   * The resource that is the target of the operation.
   * Format:
   * - Instance: instnaces/{instance}
   * - Database: instnaces/{instance}/databases/{database}
   */
  resource: string;
  /** type is the type of the anomaly. */
  type: Anomaly_AnomalyType;
  /** severity is the severity of the anomaly. */
  severity: Anomaly_AnomalySeverity;
  instanceConnectionDetail?: Anomaly_InstanceConnectionDetail | undefined;
  databaseConnectionDetail?: Anomaly_DatabaseConnectionDetail | undefined;
  databaseSchemaDriftDetail?: Anomaly_DatabaseSchemaDriftDetail | undefined;
  createTime: Date | undefined;
  updateTime: Date | undefined;
}

/** AnomalyType is the type of the anomaly. */
export enum Anomaly_AnomalyType {
  /** ANOMALY_TYPE_UNSPECIFIED - Unspecified anomaly type. */
  ANOMALY_TYPE_UNSPECIFIED = 0,
  /**
   * INSTANCE_CONNECTION - Instance level anomaly.
   *
   * INSTANCE_CONNECTION is the anomaly type for instance connection, e.g. the instance is down.
   */
  INSTANCE_CONNECTION = 1,
  /** MIGRATION_SCHEMA - MIGRATION_SCHEMA is the anomaly type for migration schema, e.g. the migration schema in the instance is missing. */
  MIGRATION_SCHEMA = 2,
  /**
   * DATABASE_CONNECTION - Database level anomaly.
   *
   * DATABASE_CONNECTION is the anomaly type for database connection, e.g. the database had been deleted.
   */
  DATABASE_CONNECTION = 5,
  /**
   * DATABASE_SCHEMA_DRIFT - DATABASE_SCHEMA_DRIFT is the anomaly type for database schema drift,
   * e.g. the database schema had been changed without bytebase migration.
   */
  DATABASE_SCHEMA_DRIFT = 6,
  UNRECOGNIZED = -1,
}

export function anomaly_AnomalyTypeFromJSON(object: any): Anomaly_AnomalyType {
  switch (object) {
    case 0:
    case "ANOMALY_TYPE_UNSPECIFIED":
      return Anomaly_AnomalyType.ANOMALY_TYPE_UNSPECIFIED;
    case 1:
    case "INSTANCE_CONNECTION":
      return Anomaly_AnomalyType.INSTANCE_CONNECTION;
    case 2:
    case "MIGRATION_SCHEMA":
      return Anomaly_AnomalyType.MIGRATION_SCHEMA;
    case 5:
    case "DATABASE_CONNECTION":
      return Anomaly_AnomalyType.DATABASE_CONNECTION;
    case 6:
    case "DATABASE_SCHEMA_DRIFT":
      return Anomaly_AnomalyType.DATABASE_SCHEMA_DRIFT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Anomaly_AnomalyType.UNRECOGNIZED;
  }
}

export function anomaly_AnomalyTypeToJSON(object: Anomaly_AnomalyType): string {
  switch (object) {
    case Anomaly_AnomalyType.ANOMALY_TYPE_UNSPECIFIED:
      return "ANOMALY_TYPE_UNSPECIFIED";
    case Anomaly_AnomalyType.INSTANCE_CONNECTION:
      return "INSTANCE_CONNECTION";
    case Anomaly_AnomalyType.MIGRATION_SCHEMA:
      return "MIGRATION_SCHEMA";
    case Anomaly_AnomalyType.DATABASE_CONNECTION:
      return "DATABASE_CONNECTION";
    case Anomaly_AnomalyType.DATABASE_SCHEMA_DRIFT:
      return "DATABASE_SCHEMA_DRIFT";
    case Anomaly_AnomalyType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

/** AnomalySeverity is the severity of the anomaly. */
export enum Anomaly_AnomalySeverity {
  /** ANOMALY_SEVERITY_UNSPECIFIED - Unspecified anomaly severity. */
  ANOMALY_SEVERITY_UNSPECIFIED = 0,
  /** MEDIUM - MEDIUM is the info level anomaly severity. */
  MEDIUM = 1,
  /** HIGH - HIGH is the warning level anomaly severity. */
  HIGH = 2,
  /** CRITICAL - CRITICAL is the critical level anomaly severity. */
  CRITICAL = 3,
  UNRECOGNIZED = -1,
}

export function anomaly_AnomalySeverityFromJSON(object: any): Anomaly_AnomalySeverity {
  switch (object) {
    case 0:
    case "ANOMALY_SEVERITY_UNSPECIFIED":
      return Anomaly_AnomalySeverity.ANOMALY_SEVERITY_UNSPECIFIED;
    case 1:
    case "MEDIUM":
      return Anomaly_AnomalySeverity.MEDIUM;
    case 2:
    case "HIGH":
      return Anomaly_AnomalySeverity.HIGH;
    case 3:
    case "CRITICAL":
      return Anomaly_AnomalySeverity.CRITICAL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Anomaly_AnomalySeverity.UNRECOGNIZED;
  }
}

export function anomaly_AnomalySeverityToJSON(object: Anomaly_AnomalySeverity): string {
  switch (object) {
    case Anomaly_AnomalySeverity.ANOMALY_SEVERITY_UNSPECIFIED:
      return "ANOMALY_SEVERITY_UNSPECIFIED";
    case Anomaly_AnomalySeverity.MEDIUM:
      return "MEDIUM";
    case Anomaly_AnomalySeverity.HIGH:
      return "HIGH";
    case Anomaly_AnomalySeverity.CRITICAL:
      return "CRITICAL";
    case Anomaly_AnomalySeverity.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

/**
 * Instance level anomaly detail.
 *
 * InstanceConnectionDetail is the detail for instance connection anomaly.
 */
export interface Anomaly_InstanceConnectionDetail {
  /** detail is the detail of the instance connection failure. */
  detail: string;
}

/**
 * Database level anomaly detial.
 *
 * DatbaaseConnectionDetail is the detail for database connection anomaly.
 */
export interface Anomaly_DatabaseConnectionDetail {
  /** detail is the detail of the database connection failure. */
  detail: string;
}

/** DatabaseSchemaDriftDetail is the detail for database schema drift anomaly. */
export interface Anomaly_DatabaseSchemaDriftDetail {
  /** record_version is the record version of the database schema drift. */
  recordVersion: string;
  /** expected_schema is the expected schema in the database. */
  expectedSchema: string;
  /** actual_schema is the actual schema in the database. */
  actualSchema: string;
}

function createBaseSearchAnomaliesRequest(): SearchAnomaliesRequest {
  return { filter: "", pageSize: 0, pageToken: "" };
}

export const SearchAnomaliesRequest = {
  encode(message: SearchAnomaliesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.filter !== "") {
      writer.uint32(10).string(message.filter);
    }
    if (message.pageSize !== 0) {
      writer.uint32(16).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(26).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SearchAnomaliesRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSearchAnomaliesRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.filter = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.pageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SearchAnomaliesRequest {
    return {
      filter: isSet(object.filter) ? globalThis.String(object.filter) : "",
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
    };
  },

  toJSON(message: SearchAnomaliesRequest): unknown {
    const obj: any = {};
    if (message.filter !== "") {
      obj.filter = message.filter;
    }
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<SearchAnomaliesRequest>): SearchAnomaliesRequest {
    return SearchAnomaliesRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SearchAnomaliesRequest>): SearchAnomaliesRequest {
    const message = createBaseSearchAnomaliesRequest();
    message.filter = object.filter ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseSearchAnomaliesResponse(): SearchAnomaliesResponse {
  return { anomalies: [], nextPageToken: "" };
}

export const SearchAnomaliesResponse = {
  encode(message: SearchAnomaliesResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.anomalies) {
      Anomaly.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SearchAnomaliesResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSearchAnomaliesResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.anomalies.push(Anomaly.decode(reader, reader.uint32()));
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

  fromJSON(object: any): SearchAnomaliesResponse {
    return {
      anomalies: globalThis.Array.isArray(object?.anomalies)
        ? object.anomalies.map((e: any) => Anomaly.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: SearchAnomaliesResponse): unknown {
    const obj: any = {};
    if (message.anomalies?.length) {
      obj.anomalies = message.anomalies.map((e) => Anomaly.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<SearchAnomaliesResponse>): SearchAnomaliesResponse {
    return SearchAnomaliesResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SearchAnomaliesResponse>): SearchAnomaliesResponse {
    const message = createBaseSearchAnomaliesResponse();
    message.anomalies = object.anomalies?.map((e) => Anomaly.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseAnomaly(): Anomaly {
  return {
    resource: "",
    type: 0,
    severity: 0,
    instanceConnectionDetail: undefined,
    databaseConnectionDetail: undefined,
    databaseSchemaDriftDetail: undefined,
    createTime: undefined,
    updateTime: undefined,
  };
}

export const Anomaly = {
  encode(message: Anomaly, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.resource !== "") {
      writer.uint32(10).string(message.resource);
    }
    if (message.type !== 0) {
      writer.uint32(16).int32(message.type);
    }
    if (message.severity !== 0) {
      writer.uint32(24).int32(message.severity);
    }
    if (message.instanceConnectionDetail !== undefined) {
      Anomaly_InstanceConnectionDetail.encode(message.instanceConnectionDetail, writer.uint32(34).fork()).ldelim();
    }
    if (message.databaseConnectionDetail !== undefined) {
      Anomaly_DatabaseConnectionDetail.encode(message.databaseConnectionDetail, writer.uint32(42).fork()).ldelim();
    }
    if (message.databaseSchemaDriftDetail !== undefined) {
      Anomaly_DatabaseSchemaDriftDetail.encode(message.databaseSchemaDriftDetail, writer.uint32(66).fork()).ldelim();
    }
    if (message.createTime !== undefined) {
      Timestamp.encode(toTimestamp(message.createTime), writer.uint32(74).fork()).ldelim();
    }
    if (message.updateTime !== undefined) {
      Timestamp.encode(toTimestamp(message.updateTime), writer.uint32(82).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Anomaly {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAnomaly();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.resource = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.type = reader.int32() as any;
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.severity = reader.int32() as any;
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.instanceConnectionDetail = Anomaly_InstanceConnectionDetail.decode(reader, reader.uint32());
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.databaseConnectionDetail = Anomaly_DatabaseConnectionDetail.decode(reader, reader.uint32());
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.databaseSchemaDriftDetail = Anomaly_DatabaseSchemaDriftDetail.decode(reader, reader.uint32());
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.createTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.updateTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Anomaly {
    return {
      resource: isSet(object.resource) ? globalThis.String(object.resource) : "",
      type: isSet(object.type) ? anomaly_AnomalyTypeFromJSON(object.type) : 0,
      severity: isSet(object.severity) ? anomaly_AnomalySeverityFromJSON(object.severity) : 0,
      instanceConnectionDetail: isSet(object.instanceConnectionDetail)
        ? Anomaly_InstanceConnectionDetail.fromJSON(object.instanceConnectionDetail)
        : undefined,
      databaseConnectionDetail: isSet(object.databaseConnectionDetail)
        ? Anomaly_DatabaseConnectionDetail.fromJSON(object.databaseConnectionDetail)
        : undefined,
      databaseSchemaDriftDetail: isSet(object.databaseSchemaDriftDetail)
        ? Anomaly_DatabaseSchemaDriftDetail.fromJSON(object.databaseSchemaDriftDetail)
        : undefined,
      createTime: isSet(object.createTime) ? fromJsonTimestamp(object.createTime) : undefined,
      updateTime: isSet(object.updateTime) ? fromJsonTimestamp(object.updateTime) : undefined,
    };
  },

  toJSON(message: Anomaly): unknown {
    const obj: any = {};
    if (message.resource !== "") {
      obj.resource = message.resource;
    }
    if (message.type !== 0) {
      obj.type = anomaly_AnomalyTypeToJSON(message.type);
    }
    if (message.severity !== 0) {
      obj.severity = anomaly_AnomalySeverityToJSON(message.severity);
    }
    if (message.instanceConnectionDetail !== undefined) {
      obj.instanceConnectionDetail = Anomaly_InstanceConnectionDetail.toJSON(message.instanceConnectionDetail);
    }
    if (message.databaseConnectionDetail !== undefined) {
      obj.databaseConnectionDetail = Anomaly_DatabaseConnectionDetail.toJSON(message.databaseConnectionDetail);
    }
    if (message.databaseSchemaDriftDetail !== undefined) {
      obj.databaseSchemaDriftDetail = Anomaly_DatabaseSchemaDriftDetail.toJSON(message.databaseSchemaDriftDetail);
    }
    if (message.createTime !== undefined) {
      obj.createTime = message.createTime.toISOString();
    }
    if (message.updateTime !== undefined) {
      obj.updateTime = message.updateTime.toISOString();
    }
    return obj;
  },

  create(base?: DeepPartial<Anomaly>): Anomaly {
    return Anomaly.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Anomaly>): Anomaly {
    const message = createBaseAnomaly();
    message.resource = object.resource ?? "";
    message.type = object.type ?? 0;
    message.severity = object.severity ?? 0;
    message.instanceConnectionDetail =
      (object.instanceConnectionDetail !== undefined && object.instanceConnectionDetail !== null)
        ? Anomaly_InstanceConnectionDetail.fromPartial(object.instanceConnectionDetail)
        : undefined;
    message.databaseConnectionDetail =
      (object.databaseConnectionDetail !== undefined && object.databaseConnectionDetail !== null)
        ? Anomaly_DatabaseConnectionDetail.fromPartial(object.databaseConnectionDetail)
        : undefined;
    message.databaseSchemaDriftDetail =
      (object.databaseSchemaDriftDetail !== undefined && object.databaseSchemaDriftDetail !== null)
        ? Anomaly_DatabaseSchemaDriftDetail.fromPartial(object.databaseSchemaDriftDetail)
        : undefined;
    message.createTime = object.createTime ?? undefined;
    message.updateTime = object.updateTime ?? undefined;
    return message;
  },
};

function createBaseAnomaly_InstanceConnectionDetail(): Anomaly_InstanceConnectionDetail {
  return { detail: "" };
}

export const Anomaly_InstanceConnectionDetail = {
  encode(message: Anomaly_InstanceConnectionDetail, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.detail !== "") {
      writer.uint32(10).string(message.detail);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Anomaly_InstanceConnectionDetail {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAnomaly_InstanceConnectionDetail();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
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

  fromJSON(object: any): Anomaly_InstanceConnectionDetail {
    return { detail: isSet(object.detail) ? globalThis.String(object.detail) : "" };
  },

  toJSON(message: Anomaly_InstanceConnectionDetail): unknown {
    const obj: any = {};
    if (message.detail !== "") {
      obj.detail = message.detail;
    }
    return obj;
  },

  create(base?: DeepPartial<Anomaly_InstanceConnectionDetail>): Anomaly_InstanceConnectionDetail {
    return Anomaly_InstanceConnectionDetail.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Anomaly_InstanceConnectionDetail>): Anomaly_InstanceConnectionDetail {
    const message = createBaseAnomaly_InstanceConnectionDetail();
    message.detail = object.detail ?? "";
    return message;
  },
};

function createBaseAnomaly_DatabaseConnectionDetail(): Anomaly_DatabaseConnectionDetail {
  return { detail: "" };
}

export const Anomaly_DatabaseConnectionDetail = {
  encode(message: Anomaly_DatabaseConnectionDetail, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.detail !== "") {
      writer.uint32(10).string(message.detail);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Anomaly_DatabaseConnectionDetail {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAnomaly_DatabaseConnectionDetail();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
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

  fromJSON(object: any): Anomaly_DatabaseConnectionDetail {
    return { detail: isSet(object.detail) ? globalThis.String(object.detail) : "" };
  },

  toJSON(message: Anomaly_DatabaseConnectionDetail): unknown {
    const obj: any = {};
    if (message.detail !== "") {
      obj.detail = message.detail;
    }
    return obj;
  },

  create(base?: DeepPartial<Anomaly_DatabaseConnectionDetail>): Anomaly_DatabaseConnectionDetail {
    return Anomaly_DatabaseConnectionDetail.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Anomaly_DatabaseConnectionDetail>): Anomaly_DatabaseConnectionDetail {
    const message = createBaseAnomaly_DatabaseConnectionDetail();
    message.detail = object.detail ?? "";
    return message;
  },
};

function createBaseAnomaly_DatabaseSchemaDriftDetail(): Anomaly_DatabaseSchemaDriftDetail {
  return { recordVersion: "", expectedSchema: "", actualSchema: "" };
}

export const Anomaly_DatabaseSchemaDriftDetail = {
  encode(message: Anomaly_DatabaseSchemaDriftDetail, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.recordVersion !== "") {
      writer.uint32(10).string(message.recordVersion);
    }
    if (message.expectedSchema !== "") {
      writer.uint32(18).string(message.expectedSchema);
    }
    if (message.actualSchema !== "") {
      writer.uint32(26).string(message.actualSchema);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Anomaly_DatabaseSchemaDriftDetail {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAnomaly_DatabaseSchemaDriftDetail();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.recordVersion = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.expectedSchema = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.actualSchema = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Anomaly_DatabaseSchemaDriftDetail {
    return {
      recordVersion: isSet(object.recordVersion) ? globalThis.String(object.recordVersion) : "",
      expectedSchema: isSet(object.expectedSchema) ? globalThis.String(object.expectedSchema) : "",
      actualSchema: isSet(object.actualSchema) ? globalThis.String(object.actualSchema) : "",
    };
  },

  toJSON(message: Anomaly_DatabaseSchemaDriftDetail): unknown {
    const obj: any = {};
    if (message.recordVersion !== "") {
      obj.recordVersion = message.recordVersion;
    }
    if (message.expectedSchema !== "") {
      obj.expectedSchema = message.expectedSchema;
    }
    if (message.actualSchema !== "") {
      obj.actualSchema = message.actualSchema;
    }
    return obj;
  },

  create(base?: DeepPartial<Anomaly_DatabaseSchemaDriftDetail>): Anomaly_DatabaseSchemaDriftDetail {
    return Anomaly_DatabaseSchemaDriftDetail.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Anomaly_DatabaseSchemaDriftDetail>): Anomaly_DatabaseSchemaDriftDetail {
    const message = createBaseAnomaly_DatabaseSchemaDriftDetail();
    message.recordVersion = object.recordVersion ?? "";
    message.expectedSchema = object.expectedSchema ?? "";
    message.actualSchema = object.actualSchema ?? "";
    return message;
  },
};

export type AnomalyServiceDefinition = typeof AnomalyServiceDefinition;
export const AnomalyServiceDefinition = {
  name: "AnomalyService",
  fullName: "bytebase.v1.AnomalyService",
  methods: {
    searchAnomalies: {
      name: "SearchAnomalies",
      requestType: SearchAnomaliesRequest,
      requestStream: false,
      responseType: SearchAnomaliesResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              22,
              18,
              20,
              47,
              118,
              49,
              47,
              97,
              110,
              111,
              109,
              97,
              108,
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
  },
} as const;

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
