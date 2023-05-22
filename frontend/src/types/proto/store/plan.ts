/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { Timestamp } from "../google/protobuf/timestamp";

export const protobufPackage = "bytebase.store";

export interface PlanWorkflow {
  steps: PlanWorkflow_Step[];
}

export interface PlanWorkflow_Step {
  specs: PlanWorkflow_Spec[];
}

export interface PlanWorkflow_Spec {
  /** earliest_allowed_time the earliest execution time of the change. */
  earliestAllowedTime?: Date;
  createDatabaseConfig?: PlanWorkflow_CreateDatabaseConfig | undefined;
  changeDatabaseConfig?: PlanWorkflow_ChangeDatabaseConfig | undefined;
  restoreDatabaseConfig?: PlanWorkflow_RestoreDatabaseConfig | undefined;
}

export interface PlanWorkflow_CreateDatabaseConfig {
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
  /**
   * backup is the resource name of the backup.
   * FIXME: backup v1 API is not ready yet, write the format here when it's ready.
   */
  backup: string;
  /** labels of the database. */
  labels: { [key: string]: string };
}

export interface PlanWorkflow_CreateDatabaseConfig_LabelsEntry {
  key: string;
  value: string;
}

export interface PlanWorkflow_ChangeDatabaseConfig {
  /**
   * The resource name of the target.
   * Format: projects/{project}/logicalDatabases/{ldb1}.
   * Format: projects/{project}/logicalDatabases/{ldb1}/logicalTables/{ltb1}.
   * Format: instances/{xxx}/databases/{db1}.
   */
  target: string;
  /**
   * The resource name of the sheet.
   * Format: sheets/{sheet}
   */
  sheet: string;
  type: PlanWorkflow_ChangeDatabaseConfig_Type;
  /**
   * schema_version is parsed from VCS file name.
   * It is automatically generated in the UI workflow.
   */
  schemaVersion: string;
  /** If RollbackEnabled, build the RollbackSheetID of the task. */
  rollbackEnabled: boolean;
}

/** Type is the database change type. */
export enum PlanWorkflow_ChangeDatabaseConfig_Type {
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

export function planWorkflow_ChangeDatabaseConfig_TypeFromJSON(object: any): PlanWorkflow_ChangeDatabaseConfig_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return PlanWorkflow_ChangeDatabaseConfig_Type.TYPE_UNSPECIFIED;
    case 1:
    case "BASELINE":
      return PlanWorkflow_ChangeDatabaseConfig_Type.BASELINE;
    case 2:
    case "MIGRATE":
      return PlanWorkflow_ChangeDatabaseConfig_Type.MIGRATE;
    case 3:
    case "MIGRATE_SDL":
      return PlanWorkflow_ChangeDatabaseConfig_Type.MIGRATE_SDL;
    case 4:
    case "MIGRATE_GHOST":
      return PlanWorkflow_ChangeDatabaseConfig_Type.MIGRATE_GHOST;
    case 5:
    case "BRANCH":
      return PlanWorkflow_ChangeDatabaseConfig_Type.BRANCH;
    case 6:
    case "DATA":
      return PlanWorkflow_ChangeDatabaseConfig_Type.DATA;
    case -1:
    case "UNRECOGNIZED":
    default:
      return PlanWorkflow_ChangeDatabaseConfig_Type.UNRECOGNIZED;
  }
}

export function planWorkflow_ChangeDatabaseConfig_TypeToJSON(object: PlanWorkflow_ChangeDatabaseConfig_Type): string {
  switch (object) {
    case PlanWorkflow_ChangeDatabaseConfig_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case PlanWorkflow_ChangeDatabaseConfig_Type.BASELINE:
      return "BASELINE";
    case PlanWorkflow_ChangeDatabaseConfig_Type.MIGRATE:
      return "MIGRATE";
    case PlanWorkflow_ChangeDatabaseConfig_Type.MIGRATE_SDL:
      return "MIGRATE_SDL";
    case PlanWorkflow_ChangeDatabaseConfig_Type.MIGRATE_GHOST:
      return "MIGRATE_GHOST";
    case PlanWorkflow_ChangeDatabaseConfig_Type.BRANCH:
      return "BRANCH";
    case PlanWorkflow_ChangeDatabaseConfig_Type.DATA:
      return "DATA";
    case PlanWorkflow_ChangeDatabaseConfig_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface PlanWorkflow_RestoreDatabaseConfig {
  /**
   * The resource name of the target to restore.
   * Format: instances/{instance}/databases/{database}
   */
  target: string;
  /** create_database_config is present if the user wants to restore to a new database. */
  createDatabaseConfig?:
    | PlanWorkflow_CreateDatabaseConfig
    | undefined;
  /**
   * FIXME: format TBD
   * Restore from a backup.
   */
  backup?:
    | string
    | undefined;
  /** After the PITR operations, the database will be recovered to the state at this time. */
  pointInTime?: Date | undefined;
}

function createBasePlanWorkflow(): PlanWorkflow {
  return { steps: [] };
}

export const PlanWorkflow = {
  encode(message: PlanWorkflow, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.steps) {
      PlanWorkflow_Step.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanWorkflow {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanWorkflow();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.steps.push(PlanWorkflow_Step.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanWorkflow {
    return { steps: Array.isArray(object?.steps) ? object.steps.map((e: any) => PlanWorkflow_Step.fromJSON(e)) : [] };
  },

  toJSON(message: PlanWorkflow): unknown {
    const obj: any = {};
    if (message.steps) {
      obj.steps = message.steps.map((e) => e ? PlanWorkflow_Step.toJSON(e) : undefined);
    } else {
      obj.steps = [];
    }
    return obj;
  },

  create(base?: DeepPartial<PlanWorkflow>): PlanWorkflow {
    return PlanWorkflow.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanWorkflow>): PlanWorkflow {
    const message = createBasePlanWorkflow();
    message.steps = object.steps?.map((e) => PlanWorkflow_Step.fromPartial(e)) || [];
    return message;
  },
};

function createBasePlanWorkflow_Step(): PlanWorkflow_Step {
  return { specs: [] };
}

export const PlanWorkflow_Step = {
  encode(message: PlanWorkflow_Step, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.specs) {
      PlanWorkflow_Spec.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanWorkflow_Step {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanWorkflow_Step();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.specs.push(PlanWorkflow_Spec.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanWorkflow_Step {
    return { specs: Array.isArray(object?.specs) ? object.specs.map((e: any) => PlanWorkflow_Spec.fromJSON(e)) : [] };
  },

  toJSON(message: PlanWorkflow_Step): unknown {
    const obj: any = {};
    if (message.specs) {
      obj.specs = message.specs.map((e) => e ? PlanWorkflow_Spec.toJSON(e) : undefined);
    } else {
      obj.specs = [];
    }
    return obj;
  },

  create(base?: DeepPartial<PlanWorkflow_Step>): PlanWorkflow_Step {
    return PlanWorkflow_Step.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanWorkflow_Step>): PlanWorkflow_Step {
    const message = createBasePlanWorkflow_Step();
    message.specs = object.specs?.map((e) => PlanWorkflow_Spec.fromPartial(e)) || [];
    return message;
  },
};

function createBasePlanWorkflow_Spec(): PlanWorkflow_Spec {
  return {
    earliestAllowedTime: undefined,
    createDatabaseConfig: undefined,
    changeDatabaseConfig: undefined,
    restoreDatabaseConfig: undefined,
  };
}

export const PlanWorkflow_Spec = {
  encode(message: PlanWorkflow_Spec, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.earliestAllowedTime !== undefined) {
      Timestamp.encode(toTimestamp(message.earliestAllowedTime), writer.uint32(34).fork()).ldelim();
    }
    if (message.createDatabaseConfig !== undefined) {
      PlanWorkflow_CreateDatabaseConfig.encode(message.createDatabaseConfig, writer.uint32(10).fork()).ldelim();
    }
    if (message.changeDatabaseConfig !== undefined) {
      PlanWorkflow_ChangeDatabaseConfig.encode(message.changeDatabaseConfig, writer.uint32(18).fork()).ldelim();
    }
    if (message.restoreDatabaseConfig !== undefined) {
      PlanWorkflow_RestoreDatabaseConfig.encode(message.restoreDatabaseConfig, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanWorkflow_Spec {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanWorkflow_Spec();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 4:
          if (tag !== 34) {
            break;
          }

          message.earliestAllowedTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 1:
          if (tag !== 10) {
            break;
          }

          message.createDatabaseConfig = PlanWorkflow_CreateDatabaseConfig.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.changeDatabaseConfig = PlanWorkflow_ChangeDatabaseConfig.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.restoreDatabaseConfig = PlanWorkflow_RestoreDatabaseConfig.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanWorkflow_Spec {
    return {
      earliestAllowedTime: isSet(object.earliestAllowedTime)
        ? fromJsonTimestamp(object.earliestAllowedTime)
        : undefined,
      createDatabaseConfig: isSet(object.createDatabaseConfig)
        ? PlanWorkflow_CreateDatabaseConfig.fromJSON(object.createDatabaseConfig)
        : undefined,
      changeDatabaseConfig: isSet(object.changeDatabaseConfig)
        ? PlanWorkflow_ChangeDatabaseConfig.fromJSON(object.changeDatabaseConfig)
        : undefined,
      restoreDatabaseConfig: isSet(object.restoreDatabaseConfig)
        ? PlanWorkflow_RestoreDatabaseConfig.fromJSON(object.restoreDatabaseConfig)
        : undefined,
    };
  },

  toJSON(message: PlanWorkflow_Spec): unknown {
    const obj: any = {};
    message.earliestAllowedTime !== undefined && (obj.earliestAllowedTime = message.earliestAllowedTime.toISOString());
    message.createDatabaseConfig !== undefined && (obj.createDatabaseConfig = message.createDatabaseConfig
      ? PlanWorkflow_CreateDatabaseConfig.toJSON(message.createDatabaseConfig)
      : undefined);
    message.changeDatabaseConfig !== undefined && (obj.changeDatabaseConfig = message.changeDatabaseConfig
      ? PlanWorkflow_ChangeDatabaseConfig.toJSON(message.changeDatabaseConfig)
      : undefined);
    message.restoreDatabaseConfig !== undefined && (obj.restoreDatabaseConfig = message.restoreDatabaseConfig
      ? PlanWorkflow_RestoreDatabaseConfig.toJSON(message.restoreDatabaseConfig)
      : undefined);
    return obj;
  },

  create(base?: DeepPartial<PlanWorkflow_Spec>): PlanWorkflow_Spec {
    return PlanWorkflow_Spec.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanWorkflow_Spec>): PlanWorkflow_Spec {
    const message = createBasePlanWorkflow_Spec();
    message.earliestAllowedTime = object.earliestAllowedTime ?? undefined;
    message.createDatabaseConfig = (object.createDatabaseConfig !== undefined && object.createDatabaseConfig !== null)
      ? PlanWorkflow_CreateDatabaseConfig.fromPartial(object.createDatabaseConfig)
      : undefined;
    message.changeDatabaseConfig = (object.changeDatabaseConfig !== undefined && object.changeDatabaseConfig !== null)
      ? PlanWorkflow_ChangeDatabaseConfig.fromPartial(object.changeDatabaseConfig)
      : undefined;
    message.restoreDatabaseConfig =
      (object.restoreDatabaseConfig !== undefined && object.restoreDatabaseConfig !== null)
        ? PlanWorkflow_RestoreDatabaseConfig.fromPartial(object.restoreDatabaseConfig)
        : undefined;
    return message;
  },
};

function createBasePlanWorkflow_CreateDatabaseConfig(): PlanWorkflow_CreateDatabaseConfig {
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

export const PlanWorkflow_CreateDatabaseConfig = {
  encode(message: PlanWorkflow_CreateDatabaseConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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
      PlanWorkflow_CreateDatabaseConfig_LabelsEntry.encode({ key: key as any, value }, writer.uint32(74).fork())
        .ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanWorkflow_CreateDatabaseConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanWorkflow_CreateDatabaseConfig();
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

          const entry9 = PlanWorkflow_CreateDatabaseConfig_LabelsEntry.decode(reader, reader.uint32());
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

  fromJSON(object: any): PlanWorkflow_CreateDatabaseConfig {
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

  toJSON(message: PlanWorkflow_CreateDatabaseConfig): unknown {
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

  create(base?: DeepPartial<PlanWorkflow_CreateDatabaseConfig>): PlanWorkflow_CreateDatabaseConfig {
    return PlanWorkflow_CreateDatabaseConfig.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanWorkflow_CreateDatabaseConfig>): PlanWorkflow_CreateDatabaseConfig {
    const message = createBasePlanWorkflow_CreateDatabaseConfig();
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

function createBasePlanWorkflow_CreateDatabaseConfig_LabelsEntry(): PlanWorkflow_CreateDatabaseConfig_LabelsEntry {
  return { key: "", value: "" };
}

export const PlanWorkflow_CreateDatabaseConfig_LabelsEntry = {
  encode(message: PlanWorkflow_CreateDatabaseConfig_LabelsEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== "") {
      writer.uint32(18).string(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanWorkflow_CreateDatabaseConfig_LabelsEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanWorkflow_CreateDatabaseConfig_LabelsEntry();
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

  fromJSON(object: any): PlanWorkflow_CreateDatabaseConfig_LabelsEntry {
    return { key: isSet(object.key) ? String(object.key) : "", value: isSet(object.value) ? String(object.value) : "" };
  },

  toJSON(message: PlanWorkflow_CreateDatabaseConfig_LabelsEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined && (obj.value = message.value);
    return obj;
  },

  create(
    base?: DeepPartial<PlanWorkflow_CreateDatabaseConfig_LabelsEntry>,
  ): PlanWorkflow_CreateDatabaseConfig_LabelsEntry {
    return PlanWorkflow_CreateDatabaseConfig_LabelsEntry.fromPartial(base ?? {});
  },

  fromPartial(
    object: DeepPartial<PlanWorkflow_CreateDatabaseConfig_LabelsEntry>,
  ): PlanWorkflow_CreateDatabaseConfig_LabelsEntry {
    const message = createBasePlanWorkflow_CreateDatabaseConfig_LabelsEntry();
    message.key = object.key ?? "";
    message.value = object.value ?? "";
    return message;
  },
};

function createBasePlanWorkflow_ChangeDatabaseConfig(): PlanWorkflow_ChangeDatabaseConfig {
  return { target: "", sheet: "", type: 0, schemaVersion: "", rollbackEnabled: false };
}

export const PlanWorkflow_ChangeDatabaseConfig = {
  encode(message: PlanWorkflow_ChangeDatabaseConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanWorkflow_ChangeDatabaseConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanWorkflow_ChangeDatabaseConfig();
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
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanWorkflow_ChangeDatabaseConfig {
    return {
      target: isSet(object.target) ? String(object.target) : "",
      sheet: isSet(object.sheet) ? String(object.sheet) : "",
      type: isSet(object.type) ? planWorkflow_ChangeDatabaseConfig_TypeFromJSON(object.type) : 0,
      schemaVersion: isSet(object.schemaVersion) ? String(object.schemaVersion) : "",
      rollbackEnabled: isSet(object.rollbackEnabled) ? Boolean(object.rollbackEnabled) : false,
    };
  },

  toJSON(message: PlanWorkflow_ChangeDatabaseConfig): unknown {
    const obj: any = {};
    message.target !== undefined && (obj.target = message.target);
    message.sheet !== undefined && (obj.sheet = message.sheet);
    message.type !== undefined && (obj.type = planWorkflow_ChangeDatabaseConfig_TypeToJSON(message.type));
    message.schemaVersion !== undefined && (obj.schemaVersion = message.schemaVersion);
    message.rollbackEnabled !== undefined && (obj.rollbackEnabled = message.rollbackEnabled);
    return obj;
  },

  create(base?: DeepPartial<PlanWorkflow_ChangeDatabaseConfig>): PlanWorkflow_ChangeDatabaseConfig {
    return PlanWorkflow_ChangeDatabaseConfig.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanWorkflow_ChangeDatabaseConfig>): PlanWorkflow_ChangeDatabaseConfig {
    const message = createBasePlanWorkflow_ChangeDatabaseConfig();
    message.target = object.target ?? "";
    message.sheet = object.sheet ?? "";
    message.type = object.type ?? 0;
    message.schemaVersion = object.schemaVersion ?? "";
    message.rollbackEnabled = object.rollbackEnabled ?? false;
    return message;
  },
};

function createBasePlanWorkflow_RestoreDatabaseConfig(): PlanWorkflow_RestoreDatabaseConfig {
  return { target: "", createDatabaseConfig: undefined, backup: undefined, pointInTime: undefined };
}

export const PlanWorkflow_RestoreDatabaseConfig = {
  encode(message: PlanWorkflow_RestoreDatabaseConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.target !== "") {
      writer.uint32(10).string(message.target);
    }
    if (message.createDatabaseConfig !== undefined) {
      PlanWorkflow_CreateDatabaseConfig.encode(message.createDatabaseConfig, writer.uint32(18).fork()).ldelim();
    }
    if (message.backup !== undefined) {
      writer.uint32(26).string(message.backup);
    }
    if (message.pointInTime !== undefined) {
      Timestamp.encode(toTimestamp(message.pointInTime), writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanWorkflow_RestoreDatabaseConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanWorkflow_RestoreDatabaseConfig();
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

          message.createDatabaseConfig = PlanWorkflow_CreateDatabaseConfig.decode(reader, reader.uint32());
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

  fromJSON(object: any): PlanWorkflow_RestoreDatabaseConfig {
    return {
      target: isSet(object.target) ? String(object.target) : "",
      createDatabaseConfig: isSet(object.createDatabaseConfig)
        ? PlanWorkflow_CreateDatabaseConfig.fromJSON(object.createDatabaseConfig)
        : undefined,
      backup: isSet(object.backup) ? String(object.backup) : undefined,
      pointInTime: isSet(object.pointInTime) ? fromJsonTimestamp(object.pointInTime) : undefined,
    };
  },

  toJSON(message: PlanWorkflow_RestoreDatabaseConfig): unknown {
    const obj: any = {};
    message.target !== undefined && (obj.target = message.target);
    message.createDatabaseConfig !== undefined && (obj.createDatabaseConfig = message.createDatabaseConfig
      ? PlanWorkflow_CreateDatabaseConfig.toJSON(message.createDatabaseConfig)
      : undefined);
    message.backup !== undefined && (obj.backup = message.backup);
    message.pointInTime !== undefined && (obj.pointInTime = message.pointInTime.toISOString());
    return obj;
  },

  create(base?: DeepPartial<PlanWorkflow_RestoreDatabaseConfig>): PlanWorkflow_RestoreDatabaseConfig {
    return PlanWorkflow_RestoreDatabaseConfig.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PlanWorkflow_RestoreDatabaseConfig>): PlanWorkflow_RestoreDatabaseConfig {
    const message = createBasePlanWorkflow_RestoreDatabaseConfig();
    message.target = object.target ?? "";
    message.createDatabaseConfig = (object.createDatabaseConfig !== undefined && object.createDatabaseConfig !== null)
      ? PlanWorkflow_CreateDatabaseConfig.fromPartial(object.createDatabaseConfig)
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
