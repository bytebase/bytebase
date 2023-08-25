/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { Timestamp } from "../google/protobuf/timestamp";

export const protobufPackage = "bytebase.store";

export interface PlanConfig {
  steps: PlanConfig_Step[];
}

export interface PlanConfig_Step {
  specs: PlanConfig_Spec[];
}

export interface PlanConfig_Spec {
  /** earliest_allowed_time the earliest execution time of the change. */
  earliestAllowedTime?:
    | Date
    | undefined;
  /** A UUID4 string that uniquely identifies the Spec. */
  id: string;
  createDatabaseConfig?: PlanConfig_CreateDatabaseConfig | undefined;
  changeDatabaseConfig?: PlanConfig_ChangeDatabaseConfig | undefined;
  restoreDatabaseConfig?: PlanConfig_RestoreDatabaseConfig | undefined;
}

export interface PlanConfig_CreateDatabaseConfig {
  /**
   * The resource name of the instance on which the database is created.
   * Format: instances/{instance}
   */
  target: string;
  /** The name of the database to create. */
  database: string;
  /**
   * table is the name of the table, if it is not empty, Bytebase should create a table after creating the database.
   * For example, in MongoDB, it only creates the database when we first store data in that database.
   */
  table: string;
  /** character_set is the character set of the database. */
  characterSet: string;
  /** collation is the collation of the database. */
  collation: string;
  /** cluster is the cluster of the database. This is only applicable to ClickHouse for "ON CLUSTER <<cluster>>". */
  cluster: string;
  /** owner is the owner of the database. This is only applicable to Postgres for "WITH OWNER <<owner>>". */
  owner: string;
  backup: string;
  /** labels of the database. */
  labels: { [key: string]: string };
}

export interface PlanConfig_CreateDatabaseConfig_LabelsEntry {
  key: string;
  value: string;
}

export interface PlanConfig_ChangeDatabaseConfig {
  /**
   * The resource name of the target.
   * Format: instances/{instance-id}/databases/{database-name}.
   * Format: projects/{project}/databaseGroups/{databaseGroup}
   */
  target: string;
  /**
   * The resource name of the sheet.
   * Format: projects/{project}/sheets/{sheet}
   */
  sheet: string;
  type: PlanConfig_ChangeDatabaseConfig_Type;
  /**
   * schema_version is parsed from VCS file name.
   * It is automatically generated in the UI workflow.
   */
  schemaVersion: string;
  /** If RollbackEnabled, build the RollbackSheetID of the task. */
  rollbackEnabled: boolean;
  rollbackDetail?: PlanConfig_ChangeDatabaseConfig_RollbackDetail | undefined;
}

/** Type is the database change type. */
export enum PlanConfig_ChangeDatabaseConfig_Type {
  TYPE_UNSPECIFIED = 0,
  /**
   * BASELINE - Used for establishing schema baseline, this is used when
   * 1. Onboard the database into Bytebase since Bytebase needs to know the current database schema.
   * 2. Had schema drift and need to re-establish the baseline.
   */
  BASELINE = 1,
  /** MIGRATE - Used for DDL changes including CREATE DATABASE. */
  MIGRATE = 2,
  /** MIGRATE_SDL - Used for schema changes via state-based schema migration including CREATE DATABASE. */
  MIGRATE_SDL = 3,
  /** MIGRATE_GHOST - Used for DDL changes using gh-ost. */
  MIGRATE_GHOST = 4,
  /** BRANCH - Used when restoring from a backup (the restored database branched from the original backup). */
  BRANCH = 5,
  /** DATA - Used for DML change. */
  DATA = 6,
  UNRECOGNIZED = -1,
}

export function planConfig_ChangeDatabaseConfig_TypeFromJSON(object: any): PlanConfig_ChangeDatabaseConfig_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return PlanConfig_ChangeDatabaseConfig_Type.TYPE_UNSPECIFIED;
    case 1:
    case "BASELINE":
      return PlanConfig_ChangeDatabaseConfig_Type.BASELINE;
    case 2:
    case "MIGRATE":
      return PlanConfig_ChangeDatabaseConfig_Type.MIGRATE;
    case 3:
    case "MIGRATE_SDL":
      return PlanConfig_ChangeDatabaseConfig_Type.MIGRATE_SDL;
    case 4:
    case "MIGRATE_GHOST":
      return PlanConfig_ChangeDatabaseConfig_Type.MIGRATE_GHOST;
    case 5:
    case "BRANCH":
      return PlanConfig_ChangeDatabaseConfig_Type.BRANCH;
    case 6:
    case "DATA":
      return PlanConfig_ChangeDatabaseConfig_Type.DATA;
    case -1:
    case "UNRECOGNIZED":
    default:
      return PlanConfig_ChangeDatabaseConfig_Type.UNRECOGNIZED;
  }
}

export function planConfig_ChangeDatabaseConfig_TypeToJSON(object: PlanConfig_ChangeDatabaseConfig_Type): string {
  switch (object) {
    case PlanConfig_ChangeDatabaseConfig_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case PlanConfig_ChangeDatabaseConfig_Type.BASELINE:
      return "BASELINE";
    case PlanConfig_ChangeDatabaseConfig_Type.MIGRATE:
      return "MIGRATE";
    case PlanConfig_ChangeDatabaseConfig_Type.MIGRATE_SDL:
      return "MIGRATE_SDL";
    case PlanConfig_ChangeDatabaseConfig_Type.MIGRATE_GHOST:
      return "MIGRATE_GHOST";
    case PlanConfig_ChangeDatabaseConfig_Type.BRANCH:
      return "BRANCH";
    case PlanConfig_ChangeDatabaseConfig_Type.DATA:
      return "DATA";
    case PlanConfig_ChangeDatabaseConfig_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface PlanConfig_ChangeDatabaseConfig_RollbackDetail {
  /**
   * rollback_from_task is the task from which the rollback SQL statement is generated for this task.
   * Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}
   */
  rollbackFromTask: string;
  /**
   * rollback_from_issue is the issue containing the original task from which the rollback SQL statement is generated for this task.
   * Format: projects/{project}/issues/{issue}
   */
  rollbackFromIssue: string;
}

export interface PlanConfig_RestoreDatabaseConfig {
  /**
   * The resource name of the target to restore.
   * Format: instances/{instance}/databases/{database}
   */
  target: string;
  /** create_database_config is present if the user wants to restore to a new database. */
  createDatabaseConfig?:
    | PlanConfig_CreateDatabaseConfig
    | undefined;
  /**
   * Restore from a backup.
   * Format: instances/{instance}/databases/{database}/backups/{backup-name}
   */
  backup?:
    | string
    | undefined;
  /** After the PITR operations, the database will be recovered to the state at this time. */
  pointInTime?: Date | undefined;
}

function createBasePlanConfig(): PlanConfig {
  return { steps: [] };
}

export const PlanConfig = {
  encode(message: PlanConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.steps) {
      PlanConfig_Step.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.steps.push(PlanConfig_Step.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanConfig {
    return { steps: Array.isArray(object?.steps) ? object.steps.map((e: any) => PlanConfig_Step.fromJSON(e)) : [] };
  },

  toJSON(message: PlanConfig): unknown {
    const obj: any = {};
    if (message.steps) {
      obj.steps = message.steps.map((e) => e ? PlanConfig_Step.toJSON(e) : undefined);
    } else {
      obj.steps = [];
    }
    return obj;
  },

  create(base?: DeepPartial<PlanConfig>): PlanConfig {
    return PlanConfig.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanConfig>): PlanConfig {
    const message = createBasePlanConfig();
    message.steps = object.steps?.map((e) => PlanConfig_Step.fromPartial(e)) || [];
    return message;
  },
};

function createBasePlanConfig_Step(): PlanConfig_Step {
  return { specs: [] };
}

export const PlanConfig_Step = {
  encode(message: PlanConfig_Step, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.specs) {
      PlanConfig_Spec.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanConfig_Step {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanConfig_Step();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.specs.push(PlanConfig_Spec.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanConfig_Step {
    return { specs: Array.isArray(object?.specs) ? object.specs.map((e: any) => PlanConfig_Spec.fromJSON(e)) : [] };
  },

  toJSON(message: PlanConfig_Step): unknown {
    const obj: any = {};
    if (message.specs) {
      obj.specs = message.specs.map((e) => e ? PlanConfig_Spec.toJSON(e) : undefined);
    } else {
      obj.specs = [];
    }
    return obj;
  },

  create(base?: DeepPartial<PlanConfig_Step>): PlanConfig_Step {
    return PlanConfig_Step.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanConfig_Step>): PlanConfig_Step {
    const message = createBasePlanConfig_Step();
    message.specs = object.specs?.map((e) => PlanConfig_Spec.fromPartial(e)) || [];
    return message;
  },
};

function createBasePlanConfig_Spec(): PlanConfig_Spec {
  return {
    earliestAllowedTime: undefined,
    id: "",
    createDatabaseConfig: undefined,
    changeDatabaseConfig: undefined,
    restoreDatabaseConfig: undefined,
  };
}

export const PlanConfig_Spec = {
  encode(message: PlanConfig_Spec, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.earliestAllowedTime !== undefined) {
      Timestamp.encode(toTimestamp(message.earliestAllowedTime), writer.uint32(34).fork()).ldelim();
    }
    if (message.id !== "") {
      writer.uint32(42).string(message.id);
    }
    if (message.createDatabaseConfig !== undefined) {
      PlanConfig_CreateDatabaseConfig.encode(message.createDatabaseConfig, writer.uint32(10).fork()).ldelim();
    }
    if (message.changeDatabaseConfig !== undefined) {
      PlanConfig_ChangeDatabaseConfig.encode(message.changeDatabaseConfig, writer.uint32(18).fork()).ldelim();
    }
    if (message.restoreDatabaseConfig !== undefined) {
      PlanConfig_RestoreDatabaseConfig.encode(message.restoreDatabaseConfig, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanConfig_Spec {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanConfig_Spec();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 4:
          if (tag !== 34) {
            break;
          }

          message.earliestAllowedTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.id = reader.string();
          continue;
        case 1:
          if (tag !== 10) {
            break;
          }

          message.createDatabaseConfig = PlanConfig_CreateDatabaseConfig.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.changeDatabaseConfig = PlanConfig_ChangeDatabaseConfig.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.restoreDatabaseConfig = PlanConfig_RestoreDatabaseConfig.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanConfig_Spec {
    return {
      earliestAllowedTime: isSet(object.earliestAllowedTime)
        ? fromJsonTimestamp(object.earliestAllowedTime)
        : undefined,
      id: isSet(object.id) ? String(object.id) : "",
      createDatabaseConfig: isSet(object.createDatabaseConfig)
        ? PlanConfig_CreateDatabaseConfig.fromJSON(object.createDatabaseConfig)
        : undefined,
      changeDatabaseConfig: isSet(object.changeDatabaseConfig)
        ? PlanConfig_ChangeDatabaseConfig.fromJSON(object.changeDatabaseConfig)
        : undefined,
      restoreDatabaseConfig: isSet(object.restoreDatabaseConfig)
        ? PlanConfig_RestoreDatabaseConfig.fromJSON(object.restoreDatabaseConfig)
        : undefined,
    };
  },

  toJSON(message: PlanConfig_Spec): unknown {
    const obj: any = {};
    message.earliestAllowedTime !== undefined && (obj.earliestAllowedTime = message.earliestAllowedTime.toISOString());
    message.id !== undefined && (obj.id = message.id);
    message.createDatabaseConfig !== undefined && (obj.createDatabaseConfig = message.createDatabaseConfig
      ? PlanConfig_CreateDatabaseConfig.toJSON(message.createDatabaseConfig)
      : undefined);
    message.changeDatabaseConfig !== undefined && (obj.changeDatabaseConfig = message.changeDatabaseConfig
      ? PlanConfig_ChangeDatabaseConfig.toJSON(message.changeDatabaseConfig)
      : undefined);
    message.restoreDatabaseConfig !== undefined && (obj.restoreDatabaseConfig = message.restoreDatabaseConfig
      ? PlanConfig_RestoreDatabaseConfig.toJSON(message.restoreDatabaseConfig)
      : undefined);
    return obj;
  },

  create(base?: DeepPartial<PlanConfig_Spec>): PlanConfig_Spec {
    return PlanConfig_Spec.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanConfig_Spec>): PlanConfig_Spec {
    const message = createBasePlanConfig_Spec();
    message.earliestAllowedTime = object.earliestAllowedTime ?? undefined;
    message.id = object.id ?? "";
    message.createDatabaseConfig = (object.createDatabaseConfig !== undefined && object.createDatabaseConfig !== null)
      ? PlanConfig_CreateDatabaseConfig.fromPartial(object.createDatabaseConfig)
      : undefined;
    message.changeDatabaseConfig = (object.changeDatabaseConfig !== undefined && object.changeDatabaseConfig !== null)
      ? PlanConfig_ChangeDatabaseConfig.fromPartial(object.changeDatabaseConfig)
      : undefined;
    message.restoreDatabaseConfig =
      (object.restoreDatabaseConfig !== undefined && object.restoreDatabaseConfig !== null)
        ? PlanConfig_RestoreDatabaseConfig.fromPartial(object.restoreDatabaseConfig)
        : undefined;
    return message;
  },
};

function createBasePlanConfig_CreateDatabaseConfig(): PlanConfig_CreateDatabaseConfig {
  return {
    target: "",
    database: "",
    table: "",
    characterSet: "",
    collation: "",
    cluster: "",
    owner: "",
    backup: "",
    labels: {},
  };
}

export const PlanConfig_CreateDatabaseConfig = {
  encode(message: PlanConfig_CreateDatabaseConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.target !== "") {
      writer.uint32(10).string(message.target);
    }
    if (message.database !== "") {
      writer.uint32(18).string(message.database);
    }
    if (message.table !== "") {
      writer.uint32(26).string(message.table);
    }
    if (message.characterSet !== "") {
      writer.uint32(34).string(message.characterSet);
    }
    if (message.collation !== "") {
      writer.uint32(42).string(message.collation);
    }
    if (message.cluster !== "") {
      writer.uint32(50).string(message.cluster);
    }
    if (message.owner !== "") {
      writer.uint32(58).string(message.owner);
    }
    if (message.backup !== "") {
      writer.uint32(66).string(message.backup);
    }
    Object.entries(message.labels).forEach(([key, value]) => {
      PlanConfig_CreateDatabaseConfig_LabelsEntry.encode({ key: key as any, value }, writer.uint32(74).fork()).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanConfig_CreateDatabaseConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanConfig_CreateDatabaseConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.target = reader.string();
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

          message.table = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.characterSet = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.collation = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.cluster = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.owner = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.backup = reader.string();
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          const entry9 = PlanConfig_CreateDatabaseConfig_LabelsEntry.decode(reader, reader.uint32());
          if (entry9.value !== undefined) {
            message.labels[entry9.key] = entry9.value;
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

  fromJSON(object: any): PlanConfig_CreateDatabaseConfig {
    return {
      target: isSet(object.target) ? String(object.target) : "",
      database: isSet(object.database) ? String(object.database) : "",
      table: isSet(object.table) ? String(object.table) : "",
      characterSet: isSet(object.characterSet) ? String(object.characterSet) : "",
      collation: isSet(object.collation) ? String(object.collation) : "",
      cluster: isSet(object.cluster) ? String(object.cluster) : "",
      owner: isSet(object.owner) ? String(object.owner) : "",
      backup: isSet(object.backup) ? String(object.backup) : "",
      labels: isObject(object.labels)
        ? Object.entries(object.labels).reduce<{ [key: string]: string }>((acc, [key, value]) => {
          acc[key] = String(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: PlanConfig_CreateDatabaseConfig): unknown {
    const obj: any = {};
    message.target !== undefined && (obj.target = message.target);
    message.database !== undefined && (obj.database = message.database);
    message.table !== undefined && (obj.table = message.table);
    message.characterSet !== undefined && (obj.characterSet = message.characterSet);
    message.collation !== undefined && (obj.collation = message.collation);
    message.cluster !== undefined && (obj.cluster = message.cluster);
    message.owner !== undefined && (obj.owner = message.owner);
    message.backup !== undefined && (obj.backup = message.backup);
    obj.labels = {};
    if (message.labels) {
      Object.entries(message.labels).forEach(([k, v]) => {
        obj.labels[k] = v;
      });
    }
    return obj;
  },

  create(base?: DeepPartial<PlanConfig_CreateDatabaseConfig>): PlanConfig_CreateDatabaseConfig {
    return PlanConfig_CreateDatabaseConfig.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanConfig_CreateDatabaseConfig>): PlanConfig_CreateDatabaseConfig {
    const message = createBasePlanConfig_CreateDatabaseConfig();
    message.target = object.target ?? "";
    message.database = object.database ?? "";
    message.table = object.table ?? "";
    message.characterSet = object.characterSet ?? "";
    message.collation = object.collation ?? "";
    message.cluster = object.cluster ?? "";
    message.owner = object.owner ?? "";
    message.backup = object.backup ?? "";
    message.labels = Object.entries(object.labels ?? {}).reduce<{ [key: string]: string }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = String(value);
      }
      return acc;
    }, {});
    return message;
  },
};

function createBasePlanConfig_CreateDatabaseConfig_LabelsEntry(): PlanConfig_CreateDatabaseConfig_LabelsEntry {
  return { key: "", value: "" };
}

export const PlanConfig_CreateDatabaseConfig_LabelsEntry = {
  encode(message: PlanConfig_CreateDatabaseConfig_LabelsEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== "") {
      writer.uint32(18).string(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanConfig_CreateDatabaseConfig_LabelsEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanConfig_CreateDatabaseConfig_LabelsEntry();
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

  fromJSON(object: any): PlanConfig_CreateDatabaseConfig_LabelsEntry {
    return { key: isSet(object.key) ? String(object.key) : "", value: isSet(object.value) ? String(object.value) : "" };
  },

  toJSON(message: PlanConfig_CreateDatabaseConfig_LabelsEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined && (obj.value = message.value);
    return obj;
  },

  create(base?: DeepPartial<PlanConfig_CreateDatabaseConfig_LabelsEntry>): PlanConfig_CreateDatabaseConfig_LabelsEntry {
    return PlanConfig_CreateDatabaseConfig_LabelsEntry.fromPartial(base ?? {});
  },

  fromPartial(
    object: DeepPartial<PlanConfig_CreateDatabaseConfig_LabelsEntry>,
  ): PlanConfig_CreateDatabaseConfig_LabelsEntry {
    const message = createBasePlanConfig_CreateDatabaseConfig_LabelsEntry();
    message.key = object.key ?? "";
    message.value = object.value ?? "";
    return message;
  },
};

function createBasePlanConfig_ChangeDatabaseConfig(): PlanConfig_ChangeDatabaseConfig {
  return { target: "", sheet: "", type: 0, schemaVersion: "", rollbackEnabled: false, rollbackDetail: undefined };
}

export const PlanConfig_ChangeDatabaseConfig = {
  encode(message: PlanConfig_ChangeDatabaseConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.target !== "") {
      writer.uint32(10).string(message.target);
    }
    if (message.sheet !== "") {
      writer.uint32(18).string(message.sheet);
    }
    if (message.type !== 0) {
      writer.uint32(24).int32(message.type);
    }
    if (message.schemaVersion !== "") {
      writer.uint32(34).string(message.schemaVersion);
    }
    if (message.rollbackEnabled === true) {
      writer.uint32(40).bool(message.rollbackEnabled);
    }
    if (message.rollbackDetail !== undefined) {
      PlanConfig_ChangeDatabaseConfig_RollbackDetail.encode(message.rollbackDetail, writer.uint32(50).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanConfig_ChangeDatabaseConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanConfig_ChangeDatabaseConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.target = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.sheet = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.type = reader.int32() as any;
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.schemaVersion = reader.string();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.rollbackEnabled = reader.bool();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.rollbackDetail = PlanConfig_ChangeDatabaseConfig_RollbackDetail.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanConfig_ChangeDatabaseConfig {
    return {
      target: isSet(object.target) ? String(object.target) : "",
      sheet: isSet(object.sheet) ? String(object.sheet) : "",
      type: isSet(object.type) ? planConfig_ChangeDatabaseConfig_TypeFromJSON(object.type) : 0,
      schemaVersion: isSet(object.schemaVersion) ? String(object.schemaVersion) : "",
      rollbackEnabled: isSet(object.rollbackEnabled) ? Boolean(object.rollbackEnabled) : false,
      rollbackDetail: isSet(object.rollbackDetail)
        ? PlanConfig_ChangeDatabaseConfig_RollbackDetail.fromJSON(object.rollbackDetail)
        : undefined,
    };
  },

  toJSON(message: PlanConfig_ChangeDatabaseConfig): unknown {
    const obj: any = {};
    message.target !== undefined && (obj.target = message.target);
    message.sheet !== undefined && (obj.sheet = message.sheet);
    message.type !== undefined && (obj.type = planConfig_ChangeDatabaseConfig_TypeToJSON(message.type));
    message.schemaVersion !== undefined && (obj.schemaVersion = message.schemaVersion);
    message.rollbackEnabled !== undefined && (obj.rollbackEnabled = message.rollbackEnabled);
    message.rollbackDetail !== undefined && (obj.rollbackDetail = message.rollbackDetail
      ? PlanConfig_ChangeDatabaseConfig_RollbackDetail.toJSON(message.rollbackDetail)
      : undefined);
    return obj;
  },

  create(base?: DeepPartial<PlanConfig_ChangeDatabaseConfig>): PlanConfig_ChangeDatabaseConfig {
    return PlanConfig_ChangeDatabaseConfig.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanConfig_ChangeDatabaseConfig>): PlanConfig_ChangeDatabaseConfig {
    const message = createBasePlanConfig_ChangeDatabaseConfig();
    message.target = object.target ?? "";
    message.sheet = object.sheet ?? "";
    message.type = object.type ?? 0;
    message.schemaVersion = object.schemaVersion ?? "";
    message.rollbackEnabled = object.rollbackEnabled ?? false;
    message.rollbackDetail = (object.rollbackDetail !== undefined && object.rollbackDetail !== null)
      ? PlanConfig_ChangeDatabaseConfig_RollbackDetail.fromPartial(object.rollbackDetail)
      : undefined;
    return message;
  },
};

function createBasePlanConfig_ChangeDatabaseConfig_RollbackDetail(): PlanConfig_ChangeDatabaseConfig_RollbackDetail {
  return { rollbackFromTask: "", rollbackFromIssue: "" };
}

export const PlanConfig_ChangeDatabaseConfig_RollbackDetail = {
  encode(
    message: PlanConfig_ChangeDatabaseConfig_RollbackDetail,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.rollbackFromTask !== "") {
      writer.uint32(10).string(message.rollbackFromTask);
    }
    if (message.rollbackFromIssue !== "") {
      writer.uint32(18).string(message.rollbackFromIssue);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanConfig_ChangeDatabaseConfig_RollbackDetail {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanConfig_ChangeDatabaseConfig_RollbackDetail();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.rollbackFromTask = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.rollbackFromIssue = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanConfig_ChangeDatabaseConfig_RollbackDetail {
    return {
      rollbackFromTask: isSet(object.rollbackFromTask) ? String(object.rollbackFromTask) : "",
      rollbackFromIssue: isSet(object.rollbackFromIssue) ? String(object.rollbackFromIssue) : "",
    };
  },

  toJSON(message: PlanConfig_ChangeDatabaseConfig_RollbackDetail): unknown {
    const obj: any = {};
    message.rollbackFromTask !== undefined && (obj.rollbackFromTask = message.rollbackFromTask);
    message.rollbackFromIssue !== undefined && (obj.rollbackFromIssue = message.rollbackFromIssue);
    return obj;
  },

  create(
    base?: DeepPartial<PlanConfig_ChangeDatabaseConfig_RollbackDetail>,
  ): PlanConfig_ChangeDatabaseConfig_RollbackDetail {
    return PlanConfig_ChangeDatabaseConfig_RollbackDetail.fromPartial(base ?? {});
  },

  fromPartial(
    object: DeepPartial<PlanConfig_ChangeDatabaseConfig_RollbackDetail>,
  ): PlanConfig_ChangeDatabaseConfig_RollbackDetail {
    const message = createBasePlanConfig_ChangeDatabaseConfig_RollbackDetail();
    message.rollbackFromTask = object.rollbackFromTask ?? "";
    message.rollbackFromIssue = object.rollbackFromIssue ?? "";
    return message;
  },
};

function createBasePlanConfig_RestoreDatabaseConfig(): PlanConfig_RestoreDatabaseConfig {
  return { target: "", createDatabaseConfig: undefined, backup: undefined, pointInTime: undefined };
}

export const PlanConfig_RestoreDatabaseConfig = {
  encode(message: PlanConfig_RestoreDatabaseConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.target !== "") {
      writer.uint32(10).string(message.target);
    }
    if (message.createDatabaseConfig !== undefined) {
      PlanConfig_CreateDatabaseConfig.encode(message.createDatabaseConfig, writer.uint32(18).fork()).ldelim();
    }
    if (message.backup !== undefined) {
      writer.uint32(26).string(message.backup);
    }
    if (message.pointInTime !== undefined) {
      Timestamp.encode(toTimestamp(message.pointInTime), writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanConfig_RestoreDatabaseConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanConfig_RestoreDatabaseConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.target = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.createDatabaseConfig = PlanConfig_CreateDatabaseConfig.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.backup = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.pointInTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanConfig_RestoreDatabaseConfig {
    return {
      target: isSet(object.target) ? String(object.target) : "",
      createDatabaseConfig: isSet(object.createDatabaseConfig)
        ? PlanConfig_CreateDatabaseConfig.fromJSON(object.createDatabaseConfig)
        : undefined,
      backup: isSet(object.backup) ? String(object.backup) : undefined,
      pointInTime: isSet(object.pointInTime) ? fromJsonTimestamp(object.pointInTime) : undefined,
    };
  },

  toJSON(message: PlanConfig_RestoreDatabaseConfig): unknown {
    const obj: any = {};
    message.target !== undefined && (obj.target = message.target);
    message.createDatabaseConfig !== undefined && (obj.createDatabaseConfig = message.createDatabaseConfig
      ? PlanConfig_CreateDatabaseConfig.toJSON(message.createDatabaseConfig)
      : undefined);
    message.backup !== undefined && (obj.backup = message.backup);
    message.pointInTime !== undefined && (obj.pointInTime = message.pointInTime.toISOString());
    return obj;
  },

  create(base?: DeepPartial<PlanConfig_RestoreDatabaseConfig>): PlanConfig_RestoreDatabaseConfig {
    return PlanConfig_RestoreDatabaseConfig.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanConfig_RestoreDatabaseConfig>): PlanConfig_RestoreDatabaseConfig {
    const message = createBasePlanConfig_RestoreDatabaseConfig();
    message.target = object.target ?? "";
    message.createDatabaseConfig = (object.createDatabaseConfig !== undefined && object.createDatabaseConfig !== null)
      ? PlanConfig_CreateDatabaseConfig.fromPartial(object.createDatabaseConfig)
      : undefined;
    message.backup = object.backup ?? undefined;
    message.pointInTime = object.pointInTime ?? undefined;
    return message;
  },
};

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
  let millis = (t.seconds || 0) * 1_000;
  millis += (t.nanos || 0) / 1_000_000;
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

function isObject(value: any): boolean {
  return typeof value === "object" && value !== null;
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
