/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Timestamp } from "../google/protobuf/timestamp";

export const protobufPackage = "bytebase.v1";

export interface SearchAnomaliesRequest {
  /**
   * The resource that is the target of the operation.
   * Format:
   * - Instance: environments/{environment}/instnaces/{instance}
   * - Database: environments/{environment}/instnaces/{instance}/databases/{database}
   */
  resourceName: string;
  /**
   * filter is the filter to apply on the list anomaly request,
   * follow the [google cel-spec](https://github.com/google/cel-spec) syntax.
   * For example:
   * List the instance anomalies of a specific instance: 'anomaly.resource_name="environments/{environemnt}/instances/{instance}" && anomaly.instance_only=true'
   * List the specified type anomalies: 'anomaly.type="DATABASE_BACKUP_POLICY_VIOLATION"'
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
   * Not used. A page token, received from a previous `ListAnomalies` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListAnomalies` must match
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
   * - Instance: environments/{environment}/instnaces/{instance}
   * - Database: environments/{environment}/instnaces/{instance}/databases/{database}
   */
  resourceName: string;
  /** type is the type of the anomaly. */
  type: Anomaly_AnomalyType;
  /** serverity is the serverity of the anomaly. */
  serverity: Anomaly_AnomalyServerity;
  instanceConnectionDetail?: Anomaly_InstanceConnectionDetail | undefined;
  databaseConnectionDetail?: Anomaly_DatabaseConnectionDetail | undefined;
  databaseBackupPolicyViolationDetail?: Anomaly_DatabaseBackupPolicyViolationDetail | undefined;
  databaseBackupMissingDetail?: Anomaly_DatabaseBackupMissingDetail | undefined;
  databaseSchemaDriftDetail?:
    | Anomaly_DatabaseSchemaDriftDetail
    | undefined;
  /**
   * instance_only is the flag to indicate if the anomaly is only for the instance.
   * If true, the anomaly is only for the instance, and the database is not specified.
   * If false, the anomaly is for the database.
   */
  instanceOnly: boolean;
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
   * DATABASE_BACKUP_POLICY_VIOLATION - Database level anomaly.
   *
   * DATABASE_BACKUP_POLICY_VIOLATION is the anomaly type for database backup policy violation,
   * e.g. the database backup policy is not meet the environment backup policy.
   */
  DATABASE_BACKUP_POLICY_VIOLATION = 3,
  /** DATABASE_BACKUP_MISSING - DATABASE_BACKUP_MISSING is the anomaly type for the backup missing, e.g. the backup is missing. */
  DATABASE_BACKUP_MISSING = 4,
  /** DATABASE_CONNECTION - DATABASE_CONNECTION is the anomaly type for database connection, e.g. the database had been deleted. */
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
    case 3:
    case "DATABASE_BACKUP_POLICY_VIOLATION":
      return Anomaly_AnomalyType.DATABASE_BACKUP_POLICY_VIOLATION;
    case 4:
    case "DATABASE_BACKUP_MISSING":
      return Anomaly_AnomalyType.DATABASE_BACKUP_MISSING;
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
    case Anomaly_AnomalyType.DATABASE_BACKUP_POLICY_VIOLATION:
      return "DATABASE_BACKUP_POLICY_VIOLATION";
    case Anomaly_AnomalyType.DATABASE_BACKUP_MISSING:
      return "DATABASE_BACKUP_MISSING";
    case Anomaly_AnomalyType.DATABASE_CONNECTION:
      return "DATABASE_CONNECTION";
    case Anomaly_AnomalyType.DATABASE_SCHEMA_DRIFT:
      return "DATABASE_SCHEMA_DRIFT";
    case Anomaly_AnomalyType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

/** AnomalyServerity is the severity of the anomaly. */
export enum Anomaly_AnomalyServerity {
  /** ANOMALY_SERVERITY_UNSPECIFIED - Unspecified anomaly serverity. */
  ANOMALY_SERVERITY_UNSPECIFIED = 0,
  /** MEDIUM - MEDIUM is the info level anomaly serverity. */
  MEDIUM = 1,
  /** HIGH - HIGH is the warning level anomaly serverity. */
  HIGH = 2,
  /** CRITICAL - CRITICAL is the critical level anomaly serverity. */
  CRITICAL = 3,
  UNRECOGNIZED = -1,
}

export function anomaly_AnomalyServerityFromJSON(object: any): Anomaly_AnomalyServerity {
  switch (object) {
    case 0:
    case "ANOMALY_SERVERITY_UNSPECIFIED":
      return Anomaly_AnomalyServerity.ANOMALY_SERVERITY_UNSPECIFIED;
    case 1:
    case "MEDIUM":
      return Anomaly_AnomalyServerity.MEDIUM;
    case 2:
    case "HIGH":
      return Anomaly_AnomalyServerity.HIGH;
    case 3:
    case "CRITICAL":
      return Anomaly_AnomalyServerity.CRITICAL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Anomaly_AnomalyServerity.UNRECOGNIZED;
  }
}

export function anomaly_AnomalyServerityToJSON(object: Anomaly_AnomalyServerity): string {
  switch (object) {
    case Anomaly_AnomalyServerity.ANOMALY_SERVERITY_UNSPECIFIED:
      return "ANOMALY_SERVERITY_UNSPECIFIED";
    case Anomaly_AnomalyServerity.MEDIUM:
      return "MEDIUM";
    case Anomaly_AnomalyServerity.HIGH:
      return "HIGH";
    case Anomaly_AnomalyServerity.CRITICAL:
      return "CRITICAL";
    case Anomaly_AnomalyServerity.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

/** BackupPlanSchedule is the backup plan schedule. */
export enum Anomaly_BackupPlanSchedule {
  /** BACKUP_PLAN_SCHEDULE_UNSPECIFIED - Unspecified backup plan schedule. */
  BACKUP_PLAN_SCHEDULE_UNSPECIFIED = 0,
  /** UNSET - UNSET is the unset backup plan schedule. */
  UNSET = 1,
  /** DAILY - DAILY is the daily backup plan schedule. */
  DAILY = 2,
  /** WEEKLY - WEEKLY is the weekly backup plan schedule. */
  WEEKLY = 3,
  UNRECOGNIZED = -1,
}

export function anomaly_BackupPlanScheduleFromJSON(object: any): Anomaly_BackupPlanSchedule {
  switch (object) {
    case 0:
    case "BACKUP_PLAN_SCHEDULE_UNSPECIFIED":
      return Anomaly_BackupPlanSchedule.BACKUP_PLAN_SCHEDULE_UNSPECIFIED;
    case 1:
    case "UNSET":
      return Anomaly_BackupPlanSchedule.UNSET;
    case 2:
    case "DAILY":
      return Anomaly_BackupPlanSchedule.DAILY;
    case 3:
    case "WEEKLY":
      return Anomaly_BackupPlanSchedule.WEEKLY;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Anomaly_BackupPlanSchedule.UNRECOGNIZED;
  }
}

export function anomaly_BackupPlanScheduleToJSON(object: Anomaly_BackupPlanSchedule): string {
  switch (object) {
    case Anomaly_BackupPlanSchedule.BACKUP_PLAN_SCHEDULE_UNSPECIFIED:
      return "BACKUP_PLAN_SCHEDULE_UNSPECIFIED";
    case Anomaly_BackupPlanSchedule.UNSET:
      return "UNSET";
    case Anomaly_BackupPlanSchedule.DAILY:
      return "DAILY";
    case Anomaly_BackupPlanSchedule.WEEKLY:
      return "WEEKLY";
    case Anomaly_BackupPlanSchedule.UNRECOGNIZED:
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

/** DatabaseBackupPolicyViolationDetail is the detail for database backup policy violation anomaly. */
export interface Anomaly_DatabaseBackupPolicyViolationDetail {
  /**
   * parent is the parent of the database.
   * Format: environments/{environment}
   */
  parent: string;
  /** expected_schedule is the expected backup plan schedule in the parent. */
  expectedSchedule: Anomaly_BackupPlanSchedule;
  /** actual_schedule is the actual backup plan schedule in the database. */
  actualSchedule: Anomaly_BackupPlanSchedule;
}

/** DatabaseBackupMissingDetail is the detail for database backup missing anomaly. */
export interface Anomaly_DatabaseBackupMissingDetail {
  /** expected_schedule is the expected backup plan schedule in the database. */
  expectedSchedule: Anomaly_BackupPlanSchedule;
  /** last_backup_time is the last backup time in the database. */
  lastBackupTime?: Date;
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
  return { resourceName: "", filter: "", pageSize: 0, pageToken: "" };
}

export const SearchAnomaliesRequest = {
  encode(message: SearchAnomaliesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.resourceName !== "") {
      writer.uint32(10).string(message.resourceName);
    }
    if (message.filter !== "") {
      writer.uint32(18).string(message.filter);
    }
    if (message.pageSize !== 0) {
      writer.uint32(24).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(34).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SearchAnomaliesRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSearchAnomaliesRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.resourceName = reader.string();
          break;
        case 2:
          message.filter = reader.string();
          break;
        case 3:
          message.pageSize = reader.int32();
          break;
        case 4:
          message.pageToken = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): SearchAnomaliesRequest {
    return {
      resourceName: isSet(object.resourceName) ? String(object.resourceName) : "",
      filter: isSet(object.filter) ? String(object.filter) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: SearchAnomaliesRequest): unknown {
    const obj: any = {};
    message.resourceName !== undefined && (obj.resourceName = message.resourceName);
    message.filter !== undefined && (obj.filter = message.filter);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    return obj;
  },

  fromPartial(object: DeepPartial<SearchAnomaliesRequest>): SearchAnomaliesRequest {
    const message = createBaseSearchAnomaliesRequest();
    message.resourceName = object.resourceName ?? "";
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
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSearchAnomaliesResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.anomalies.push(Anomaly.decode(reader, reader.uint32()));
          break;
        case 2:
          message.nextPageToken = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): SearchAnomaliesResponse {
    return {
      anomalies: Array.isArray(object?.anomalies) ? object.anomalies.map((e: any) => Anomaly.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: SearchAnomaliesResponse): unknown {
    const obj: any = {};
    if (message.anomalies) {
      obj.anomalies = message.anomalies.map((e) => e ? Anomaly.toJSON(e) : undefined);
    } else {
      obj.anomalies = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
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
    resourceName: "",
    type: 0,
    serverity: 0,
    instanceConnectionDetail: undefined,
    databaseConnectionDetail: undefined,
    databaseBackupPolicyViolationDetail: undefined,
    databaseBackupMissingDetail: undefined,
    databaseSchemaDriftDetail: undefined,
    instanceOnly: false,
  };
}

export const Anomaly = {
  encode(message: Anomaly, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.resourceName !== "") {
      writer.uint32(10).string(message.resourceName);
    }
    if (message.type !== 0) {
      writer.uint32(16).int32(message.type);
    }
    if (message.serverity !== 0) {
      writer.uint32(24).int32(message.serverity);
    }
    if (message.instanceConnectionDetail !== undefined) {
      Anomaly_InstanceConnectionDetail.encode(message.instanceConnectionDetail, writer.uint32(34).fork()).ldelim();
    }
    if (message.databaseConnectionDetail !== undefined) {
      Anomaly_DatabaseConnectionDetail.encode(message.databaseConnectionDetail, writer.uint32(42).fork()).ldelim();
    }
    if (message.databaseBackupPolicyViolationDetail !== undefined) {
      Anomaly_DatabaseBackupPolicyViolationDetail.encode(
        message.databaseBackupPolicyViolationDetail,
        writer.uint32(50).fork(),
      ).ldelim();
    }
    if (message.databaseBackupMissingDetail !== undefined) {
      Anomaly_DatabaseBackupMissingDetail.encode(message.databaseBackupMissingDetail, writer.uint32(58).fork())
        .ldelim();
    }
    if (message.databaseSchemaDriftDetail !== undefined) {
      Anomaly_DatabaseSchemaDriftDetail.encode(message.databaseSchemaDriftDetail, writer.uint32(66).fork()).ldelim();
    }
    if (message.instanceOnly === true) {
      writer.uint32(72).bool(message.instanceOnly);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Anomaly {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAnomaly();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.resourceName = reader.string();
          break;
        case 2:
          message.type = reader.int32() as any;
          break;
        case 3:
          message.serverity = reader.int32() as any;
          break;
        case 4:
          message.instanceConnectionDetail = Anomaly_InstanceConnectionDetail.decode(reader, reader.uint32());
          break;
        case 5:
          message.databaseConnectionDetail = Anomaly_DatabaseConnectionDetail.decode(reader, reader.uint32());
          break;
        case 6:
          message.databaseBackupPolicyViolationDetail = Anomaly_DatabaseBackupPolicyViolationDetail.decode(
            reader,
            reader.uint32(),
          );
          break;
        case 7:
          message.databaseBackupMissingDetail = Anomaly_DatabaseBackupMissingDetail.decode(reader, reader.uint32());
          break;
        case 8:
          message.databaseSchemaDriftDetail = Anomaly_DatabaseSchemaDriftDetail.decode(reader, reader.uint32());
          break;
        case 9:
          message.instanceOnly = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Anomaly {
    return {
      resourceName: isSet(object.resourceName) ? String(object.resourceName) : "",
      type: isSet(object.type) ? anomaly_AnomalyTypeFromJSON(object.type) : 0,
      serverity: isSet(object.serverity) ? anomaly_AnomalyServerityFromJSON(object.serverity) : 0,
      instanceConnectionDetail: isSet(object.instanceConnectionDetail)
        ? Anomaly_InstanceConnectionDetail.fromJSON(object.instanceConnectionDetail)
        : undefined,
      databaseConnectionDetail: isSet(object.databaseConnectionDetail)
        ? Anomaly_DatabaseConnectionDetail.fromJSON(object.databaseConnectionDetail)
        : undefined,
      databaseBackupPolicyViolationDetail: isSet(object.databaseBackupPolicyViolationDetail)
        ? Anomaly_DatabaseBackupPolicyViolationDetail.fromJSON(object.databaseBackupPolicyViolationDetail)
        : undefined,
      databaseBackupMissingDetail: isSet(object.databaseBackupMissingDetail)
        ? Anomaly_DatabaseBackupMissingDetail.fromJSON(object.databaseBackupMissingDetail)
        : undefined,
      databaseSchemaDriftDetail: isSet(object.databaseSchemaDriftDetail)
        ? Anomaly_DatabaseSchemaDriftDetail.fromJSON(object.databaseSchemaDriftDetail)
        : undefined,
      instanceOnly: isSet(object.instanceOnly) ? Boolean(object.instanceOnly) : false,
    };
  },

  toJSON(message: Anomaly): unknown {
    const obj: any = {};
    message.resourceName !== undefined && (obj.resourceName = message.resourceName);
    message.type !== undefined && (obj.type = anomaly_AnomalyTypeToJSON(message.type));
    message.serverity !== undefined && (obj.serverity = anomaly_AnomalyServerityToJSON(message.serverity));
    message.instanceConnectionDetail !== undefined && (obj.instanceConnectionDetail = message.instanceConnectionDetail
      ? Anomaly_InstanceConnectionDetail.toJSON(message.instanceConnectionDetail)
      : undefined);
    message.databaseConnectionDetail !== undefined && (obj.databaseConnectionDetail = message.databaseConnectionDetail
      ? Anomaly_DatabaseConnectionDetail.toJSON(message.databaseConnectionDetail)
      : undefined);
    message.databaseBackupPolicyViolationDetail !== undefined &&
      (obj.databaseBackupPolicyViolationDetail = message.databaseBackupPolicyViolationDetail
        ? Anomaly_DatabaseBackupPolicyViolationDetail.toJSON(message.databaseBackupPolicyViolationDetail)
        : undefined);
    message.databaseBackupMissingDetail !== undefined &&
      (obj.databaseBackupMissingDetail = message.databaseBackupMissingDetail
        ? Anomaly_DatabaseBackupMissingDetail.toJSON(message.databaseBackupMissingDetail)
        : undefined);
    message.databaseSchemaDriftDetail !== undefined &&
      (obj.databaseSchemaDriftDetail = message.databaseSchemaDriftDetail
        ? Anomaly_DatabaseSchemaDriftDetail.toJSON(message.databaseSchemaDriftDetail)
        : undefined);
    message.instanceOnly !== undefined && (obj.instanceOnly = message.instanceOnly);
    return obj;
  },

  fromPartial(object: DeepPartial<Anomaly>): Anomaly {
    const message = createBaseAnomaly();
    message.resourceName = object.resourceName ?? "";
    message.type = object.type ?? 0;
    message.serverity = object.serverity ?? 0;
    message.instanceConnectionDetail =
      (object.instanceConnectionDetail !== undefined && object.instanceConnectionDetail !== null)
        ? Anomaly_InstanceConnectionDetail.fromPartial(object.instanceConnectionDetail)
        : undefined;
    message.databaseConnectionDetail =
      (object.databaseConnectionDetail !== undefined && object.databaseConnectionDetail !== null)
        ? Anomaly_DatabaseConnectionDetail.fromPartial(object.databaseConnectionDetail)
        : undefined;
    message.databaseBackupPolicyViolationDetail =
      (object.databaseBackupPolicyViolationDetail !== undefined && object.databaseBackupPolicyViolationDetail !== null)
        ? Anomaly_DatabaseBackupPolicyViolationDetail.fromPartial(object.databaseBackupPolicyViolationDetail)
        : undefined;
    message.databaseBackupMissingDetail =
      (object.databaseBackupMissingDetail !== undefined && object.databaseBackupMissingDetail !== null)
        ? Anomaly_DatabaseBackupMissingDetail.fromPartial(object.databaseBackupMissingDetail)
        : undefined;
    message.databaseSchemaDriftDetail =
      (object.databaseSchemaDriftDetail !== undefined && object.databaseSchemaDriftDetail !== null)
        ? Anomaly_DatabaseSchemaDriftDetail.fromPartial(object.databaseSchemaDriftDetail)
        : undefined;
    message.instanceOnly = object.instanceOnly ?? false;
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
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAnomaly_InstanceConnectionDetail();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.detail = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Anomaly_InstanceConnectionDetail {
    return { detail: isSet(object.detail) ? String(object.detail) : "" };
  },

  toJSON(message: Anomaly_InstanceConnectionDetail): unknown {
    const obj: any = {};
    message.detail !== undefined && (obj.detail = message.detail);
    return obj;
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
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAnomaly_DatabaseConnectionDetail();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.detail = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Anomaly_DatabaseConnectionDetail {
    return { detail: isSet(object.detail) ? String(object.detail) : "" };
  },

  toJSON(message: Anomaly_DatabaseConnectionDetail): unknown {
    const obj: any = {};
    message.detail !== undefined && (obj.detail = message.detail);
    return obj;
  },

  fromPartial(object: DeepPartial<Anomaly_DatabaseConnectionDetail>): Anomaly_DatabaseConnectionDetail {
    const message = createBaseAnomaly_DatabaseConnectionDetail();
    message.detail = object.detail ?? "";
    return message;
  },
};

function createBaseAnomaly_DatabaseBackupPolicyViolationDetail(): Anomaly_DatabaseBackupPolicyViolationDetail {
  return { parent: "", expectedSchedule: 0, actualSchedule: 0 };
}

export const Anomaly_DatabaseBackupPolicyViolationDetail = {
  encode(message: Anomaly_DatabaseBackupPolicyViolationDetail, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.expectedSchedule !== 0) {
      writer.uint32(16).int32(message.expectedSchedule);
    }
    if (message.actualSchedule !== 0) {
      writer.uint32(24).int32(message.actualSchedule);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Anomaly_DatabaseBackupPolicyViolationDetail {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAnomaly_DatabaseBackupPolicyViolationDetail();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.parent = reader.string();
          break;
        case 2:
          message.expectedSchedule = reader.int32() as any;
          break;
        case 3:
          message.actualSchedule = reader.int32() as any;
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Anomaly_DatabaseBackupPolicyViolationDetail {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      expectedSchedule: isSet(object.expectedSchedule)
        ? anomaly_BackupPlanScheduleFromJSON(object.expectedSchedule)
        : 0,
      actualSchedule: isSet(object.actualSchedule) ? anomaly_BackupPlanScheduleFromJSON(object.actualSchedule) : 0,
    };
  },

  toJSON(message: Anomaly_DatabaseBackupPolicyViolationDetail): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.expectedSchedule !== undefined &&
      (obj.expectedSchedule = anomaly_BackupPlanScheduleToJSON(message.expectedSchedule));
    message.actualSchedule !== undefined &&
      (obj.actualSchedule = anomaly_BackupPlanScheduleToJSON(message.actualSchedule));
    return obj;
  },

  fromPartial(
    object: DeepPartial<Anomaly_DatabaseBackupPolicyViolationDetail>,
  ): Anomaly_DatabaseBackupPolicyViolationDetail {
    const message = createBaseAnomaly_DatabaseBackupPolicyViolationDetail();
    message.parent = object.parent ?? "";
    message.expectedSchedule = object.expectedSchedule ?? 0;
    message.actualSchedule = object.actualSchedule ?? 0;
    return message;
  },
};

function createBaseAnomaly_DatabaseBackupMissingDetail(): Anomaly_DatabaseBackupMissingDetail {
  return { expectedSchedule: 0, lastBackupTime: undefined };
}

export const Anomaly_DatabaseBackupMissingDetail = {
  encode(message: Anomaly_DatabaseBackupMissingDetail, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.expectedSchedule !== 0) {
      writer.uint32(8).int32(message.expectedSchedule);
    }
    if (message.lastBackupTime !== undefined) {
      Timestamp.encode(toTimestamp(message.lastBackupTime), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Anomaly_DatabaseBackupMissingDetail {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAnomaly_DatabaseBackupMissingDetail();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.expectedSchedule = reader.int32() as any;
          break;
        case 2:
          message.lastBackupTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Anomaly_DatabaseBackupMissingDetail {
    return {
      expectedSchedule: isSet(object.expectedSchedule)
        ? anomaly_BackupPlanScheduleFromJSON(object.expectedSchedule)
        : 0,
      lastBackupTime: isSet(object.lastBackupTime) ? fromJsonTimestamp(object.lastBackupTime) : undefined,
    };
  },

  toJSON(message: Anomaly_DatabaseBackupMissingDetail): unknown {
    const obj: any = {};
    message.expectedSchedule !== undefined &&
      (obj.expectedSchedule = anomaly_BackupPlanScheduleToJSON(message.expectedSchedule));
    message.lastBackupTime !== undefined && (obj.lastBackupTime = message.lastBackupTime.toISOString());
    return obj;
  },

  fromPartial(object: DeepPartial<Anomaly_DatabaseBackupMissingDetail>): Anomaly_DatabaseBackupMissingDetail {
    const message = createBaseAnomaly_DatabaseBackupMissingDetail();
    message.expectedSchedule = object.expectedSchedule ?? 0;
    message.lastBackupTime = object.lastBackupTime ?? undefined;
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
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAnomaly_DatabaseSchemaDriftDetail();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.recordVersion = reader.string();
          break;
        case 2:
          message.expectedSchema = reader.string();
          break;
        case 3:
          message.actualSchema = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Anomaly_DatabaseSchemaDriftDetail {
    return {
      recordVersion: isSet(object.recordVersion) ? String(object.recordVersion) : "",
      expectedSchema: isSet(object.expectedSchema) ? String(object.expectedSchema) : "",
      actualSchema: isSet(object.actualSchema) ? String(object.actualSchema) : "",
    };
  },

  toJSON(message: Anomaly_DatabaseSchemaDriftDetail): unknown {
    const obj: any = {};
    message.recordVersion !== undefined && (obj.recordVersion = message.recordVersion);
    message.expectedSchema !== undefined && (obj.expectedSchema = message.expectedSchema);
    message.actualSchema !== undefined && (obj.actualSchema = message.actualSchema);
    return obj;
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
      options: {},
    },
  },
} as const;

export interface AnomalyServiceImplementation<CallContextExt = {}> {
  searchAnomalies(
    request: SearchAnomaliesRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<SearchAnomaliesResponse>>;
}

export interface AnomalyServiceClient<CallOptionsExt = {}> {
  searchAnomalies(
    request: DeepPartial<SearchAnomaliesRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<SearchAnomaliesResponse>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function toTimestamp(date: Date): Timestamp {
  const seconds = date.getTime() / 1_000;
  const nanos = (date.getTime() % 1_000) * 1_000_000;
  return { seconds, nanos };
}

function fromTimestamp(t: Timestamp): Date {
  let millis = t.seconds * 1_000;
  millis += t.nanos / 1_000_000;
  return new Date(millis);
}

function fromJsonTimestamp(o: any): Date {
  if (o instanceof Date) {
    return o;
  } else if (typeof o === "string") {
    return new Date(o);
  } else {
    return fromTimestamp(Timestamp.fromJSON(o));
  }
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
