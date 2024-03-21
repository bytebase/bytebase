/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Timestamp } from "../google/protobuf/timestamp";

export const protobufPackage = "bytebase.store";

export interface PlanConfig {
  steps: PlanConfig_Step[];
}

export interface PlanConfig_Step {
  /**
   * Use the title if set.
   * Use a generated title if empty.
   */
  title: string;
  specs: PlanConfig_Spec[];
}

export interface PlanConfig_Spec {
  /** earliest_allowed_time the earliest execution time of the change. */
  earliestAllowedTime:
    | Date
    | undefined;
  /** A UUID4 string that uniquely identifies the Spec. */
  id: string;
  /** IDs of the specs that this spec depends on. */
  dependsOnSpecs: string[];
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
  /**
   * backup is the resource name of the backup.
   * Format: instances/{instance}/databases/{database}/backups/{backup-name}
   */
  backup: string;
  /**
   * The environment resource.
   * Format: environments/prod where prod is the environment resource ID.
   */
  environment: string;
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
   * Format: projects/{project}/databaseGroups/{databaseGroup}.
   * Format: projects/{project}/deploymentConfigs/default. The plan should
   * have a single step and single spec for the deployment configuration type.
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
  /** If RollbackEnabled, build the RollbackSheetID of the task after the task is completed. */
  rollbackEnabled: boolean;
  rollbackDetail?: PlanConfig_ChangeDatabaseConfig_RollbackDetail | undefined;
  ghostFlags: { [key: string]: string };
  /** If set, a backup of the modified data will be created automatically before any changes are applied. */
  preUpdateBackupDetail?: PlanConfig_ChangeDatabaseConfig_PreUpdateBackupDetail | undefined;
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

export interface PlanConfig_ChangeDatabaseConfig_GhostFlagsEntry {
  key: string;
  value: string;
}

export interface PlanConfig_ChangeDatabaseConfig_PreUpdateBackupDetail {
  /**
   * The database for keeping the backup data.
   * Format: instances/{instance}/databases/{database}
   */
  database: string;
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
    return {
      steps: globalThis.Array.isArray(object?.steps) ? object.steps.map((e: any) => PlanConfig_Step.fromJSON(e)) : [],
    };
  },

  toJSON(message: PlanConfig): unknown {
    const obj: any = {};
    if (message.steps?.length) {
      obj.steps = message.steps.map((e) => PlanConfig_Step.toJSON(e));
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
  return { title: "", specs: [] };
}

export const PlanConfig_Step = {
  encode(message: PlanConfig_Step, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
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
        case 2:
          if (tag !== 18) {
            break;
          }

          message.title = reader.string();
          continue;
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
    return {
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      specs: globalThis.Array.isArray(object?.specs) ? object.specs.map((e: any) => PlanConfig_Spec.fromJSON(e)) : [],
    };
  },

  toJSON(message: PlanConfig_Step): unknown {
    const obj: any = {};
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.specs?.length) {
      obj.specs = message.specs.map((e) => PlanConfig_Spec.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<PlanConfig_Step>): PlanConfig_Step {
    return PlanConfig_Step.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<PlanConfig_Step>): PlanConfig_Step {
    const message = createBasePlanConfig_Step();
    message.title = object.title ?? "";
    message.specs = object.specs?.map((e) => PlanConfig_Spec.fromPartial(e)) || [];
    return message;
  },
};

function createBasePlanConfig_Spec(): PlanConfig_Spec {
  return {
    earliestAllowedTime: undefined,
    id: "",
    dependsOnSpecs: [],
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
    for (const v of message.dependsOnSpecs) {
      writer.uint32(50).string(v!);
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
        case 6:
          if (tag !== 50) {
            break;
          }

          message.dependsOnSpecs.push(reader.string());
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
      id: isSet(object.id) ? globalThis.String(object.id) : "",
      dependsOnSpecs: globalThis.Array.isArray(object?.dependsOnSpecs)
        ? object.dependsOnSpecs.map((e: any) => globalThis.String(e))
        : [],
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
    if (message.earliestAllowedTime !== undefined) {
      obj.earliestAllowedTime = message.earliestAllowedTime.toISOString();
    }
    if (message.id !== "") {
      obj.id = message.id;
    }
    if (message.dependsOnSpecs?.length) {
      obj.dependsOnSpecs = message.dependsOnSpecs;
    }
    if (message.createDatabaseConfig !== undefined) {
      obj.createDatabaseConfig = PlanConfig_CreateDatabaseConfig.toJSON(message.createDatabaseConfig);
    }
    if (message.changeDatabaseConfig !== undefined) {
      obj.changeDatabaseConfig = PlanConfig_ChangeDatabaseConfig.toJSON(message.changeDatabaseConfig);
    }
    if (message.restoreDatabaseConfig !== undefined) {
      obj.restoreDatabaseConfig = PlanConfig_RestoreDatabaseConfig.toJSON(message.restoreDatabaseConfig);
    }
    return obj;
  },

  create(base?: DeepPartial<PlanConfig_Spec>): PlanConfig_Spec {
    return PlanConfig_Spec.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<PlanConfig_Spec>): PlanConfig_Spec {
    const message = createBasePlanConfig_Spec();
    message.earliestAllowedTime = object.earliestAllowedTime ?? undefined;
    message.id = object.id ?? "";
    message.dependsOnSpecs = object.dependsOnSpecs?.map((e) => e) || [];
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
    environment: "",
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
    if (message.environment !== "") {
      writer.uint32(74).string(message.environment);
    }
    Object.entries(message.labels).forEach(([key, value]) => {
      PlanConfig_CreateDatabaseConfig_LabelsEntry.encode({ key: key as any, value }, writer.uint32(82).fork()).ldelim();
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

          message.environment = reader.string();
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          const entry10 = PlanConfig_CreateDatabaseConfig_LabelsEntry.decode(reader, reader.uint32());
          if (entry10.value !== undefined) {
            message.labels[entry10.key] = entry10.value;
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
      target: isSet(object.target) ? globalThis.String(object.target) : "",
      database: isSet(object.database) ? globalThis.String(object.database) : "",
      table: isSet(object.table) ? globalThis.String(object.table) : "",
      characterSet: isSet(object.characterSet) ? globalThis.String(object.characterSet) : "",
      collation: isSet(object.collation) ? globalThis.String(object.collation) : "",
      cluster: isSet(object.cluster) ? globalThis.String(object.cluster) : "",
      owner: isSet(object.owner) ? globalThis.String(object.owner) : "",
      backup: isSet(object.backup) ? globalThis.String(object.backup) : "",
      environment: isSet(object.environment) ? globalThis.String(object.environment) : "",
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
    if (message.target !== "") {
      obj.target = message.target;
    }
    if (message.database !== "") {
      obj.database = message.database;
    }
    if (message.table !== "") {
      obj.table = message.table;
    }
    if (message.characterSet !== "") {
      obj.characterSet = message.characterSet;
    }
    if (message.collation !== "") {
      obj.collation = message.collation;
    }
    if (message.cluster !== "") {
      obj.cluster = message.cluster;
    }
    if (message.owner !== "") {
      obj.owner = message.owner;
    }
    if (message.backup !== "") {
      obj.backup = message.backup;
    }
    if (message.environment !== "") {
      obj.environment = message.environment;
    }
    if (message.labels) {
      const entries = Object.entries(message.labels);
      if (entries.length > 0) {
        obj.labels = {};
        entries.forEach(([k, v]) => {
          obj.labels[k] = v;
        });
      }
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
    message.environment = object.environment ?? "";
    message.labels = Object.entries(object.labels ?? {}).reduce<{ [key: string]: string }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = globalThis.String(value);
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
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? globalThis.String(object.value) : "",
    };
  },

  toJSON(message: PlanConfig_CreateDatabaseConfig_LabelsEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== "") {
      obj.value = message.value;
    }
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
  return {
    target: "",
    sheet: "",
    type: 0,
    schemaVersion: "",
    rollbackEnabled: false,
    rollbackDetail: undefined,
    ghostFlags: {},
    preUpdateBackupDetail: undefined,
  };
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
    Object.entries(message.ghostFlags).forEach(([key, value]) => {
      PlanConfig_ChangeDatabaseConfig_GhostFlagsEntry.encode({ key: key as any, value }, writer.uint32(58).fork())
        .ldelim();
    });
    if (message.preUpdateBackupDetail !== undefined) {
      PlanConfig_ChangeDatabaseConfig_PreUpdateBackupDetail.encode(
        message.preUpdateBackupDetail,
        writer.uint32(66).fork(),
      ).ldelim();
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
        case 7:
          if (tag !== 58) {
            break;
          }

          const entry7 = PlanConfig_ChangeDatabaseConfig_GhostFlagsEntry.decode(reader, reader.uint32());
          if (entry7.value !== undefined) {
            message.ghostFlags[entry7.key] = entry7.value;
          }
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.preUpdateBackupDetail = PlanConfig_ChangeDatabaseConfig_PreUpdateBackupDetail.decode(
            reader,
            reader.uint32(),
          );
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
      target: isSet(object.target) ? globalThis.String(object.target) : "",
      sheet: isSet(object.sheet) ? globalThis.String(object.sheet) : "",
      type: isSet(object.type) ? planConfig_ChangeDatabaseConfig_TypeFromJSON(object.type) : 0,
      schemaVersion: isSet(object.schemaVersion) ? globalThis.String(object.schemaVersion) : "",
      rollbackEnabled: isSet(object.rollbackEnabled) ? globalThis.Boolean(object.rollbackEnabled) : false,
      rollbackDetail: isSet(object.rollbackDetail)
        ? PlanConfig_ChangeDatabaseConfig_RollbackDetail.fromJSON(object.rollbackDetail)
        : undefined,
      ghostFlags: isObject(object.ghostFlags)
        ? Object.entries(object.ghostFlags).reduce<{ [key: string]: string }>((acc, [key, value]) => {
          acc[key] = String(value);
          return acc;
        }, {})
        : {},
      preUpdateBackupDetail: isSet(object.preUpdateBackupDetail)
        ? PlanConfig_ChangeDatabaseConfig_PreUpdateBackupDetail.fromJSON(object.preUpdateBackupDetail)
        : undefined,
    };
  },

  toJSON(message: PlanConfig_ChangeDatabaseConfig): unknown {
    const obj: any = {};
    if (message.target !== "") {
      obj.target = message.target;
    }
    if (message.sheet !== "") {
      obj.sheet = message.sheet;
    }
    if (message.type !== 0) {
      obj.type = planConfig_ChangeDatabaseConfig_TypeToJSON(message.type);
    }
    if (message.schemaVersion !== "") {
      obj.schemaVersion = message.schemaVersion;
    }
    if (message.rollbackEnabled === true) {
      obj.rollbackEnabled = message.rollbackEnabled;
    }
    if (message.rollbackDetail !== undefined) {
      obj.rollbackDetail = PlanConfig_ChangeDatabaseConfig_RollbackDetail.toJSON(message.rollbackDetail);
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
    if (message.preUpdateBackupDetail !== undefined) {
      obj.preUpdateBackupDetail = PlanConfig_ChangeDatabaseConfig_PreUpdateBackupDetail.toJSON(
        message.preUpdateBackupDetail,
      );
    }
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
    message.ghostFlags = Object.entries(object.ghostFlags ?? {}).reduce<{ [key: string]: string }>(
      (acc, [key, value]) => {
        if (value !== undefined) {
          acc[key] = globalThis.String(value);
        }
        return acc;
      },
      {},
    );
    message.preUpdateBackupDetail =
      (object.preUpdateBackupDetail !== undefined && object.preUpdateBackupDetail !== null)
        ? PlanConfig_ChangeDatabaseConfig_PreUpdateBackupDetail.fromPartial(object.preUpdateBackupDetail)
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
      rollbackFromTask: isSet(object.rollbackFromTask) ? globalThis.String(object.rollbackFromTask) : "",
      rollbackFromIssue: isSet(object.rollbackFromIssue) ? globalThis.String(object.rollbackFromIssue) : "",
    };
  },

  toJSON(message: PlanConfig_ChangeDatabaseConfig_RollbackDetail): unknown {
    const obj: any = {};
    if (message.rollbackFromTask !== "") {
      obj.rollbackFromTask = message.rollbackFromTask;
    }
    if (message.rollbackFromIssue !== "") {
      obj.rollbackFromIssue = message.rollbackFromIssue;
    }
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

function createBasePlanConfig_ChangeDatabaseConfig_GhostFlagsEntry(): PlanConfig_ChangeDatabaseConfig_GhostFlagsEntry {
  return { key: "", value: "" };
}

export const PlanConfig_ChangeDatabaseConfig_GhostFlagsEntry = {
  encode(
    message: PlanConfig_ChangeDatabaseConfig_GhostFlagsEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== "") {
      writer.uint32(18).string(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanConfig_ChangeDatabaseConfig_GhostFlagsEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanConfig_ChangeDatabaseConfig_GhostFlagsEntry();
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

  fromJSON(object: any): PlanConfig_ChangeDatabaseConfig_GhostFlagsEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? globalThis.String(object.value) : "",
    };
  },

  toJSON(message: PlanConfig_ChangeDatabaseConfig_GhostFlagsEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== "") {
      obj.value = message.value;
    }
    return obj;
  },

  create(
    base?: DeepPartial<PlanConfig_ChangeDatabaseConfig_GhostFlagsEntry>,
  ): PlanConfig_ChangeDatabaseConfig_GhostFlagsEntry {
    return PlanConfig_ChangeDatabaseConfig_GhostFlagsEntry.fromPartial(base ?? {});
  },
  fromPartial(
    object: DeepPartial<PlanConfig_ChangeDatabaseConfig_GhostFlagsEntry>,
  ): PlanConfig_ChangeDatabaseConfig_GhostFlagsEntry {
    const message = createBasePlanConfig_ChangeDatabaseConfig_GhostFlagsEntry();
    message.key = object.key ?? "";
    message.value = object.value ?? "";
    return message;
  },
};

function createBasePlanConfig_ChangeDatabaseConfig_PreUpdateBackupDetail(): PlanConfig_ChangeDatabaseConfig_PreUpdateBackupDetail {
  return { database: "" };
}

export const PlanConfig_ChangeDatabaseConfig_PreUpdateBackupDetail = {
  encode(
    message: PlanConfig_ChangeDatabaseConfig_PreUpdateBackupDetail,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.database !== "") {
      writer.uint32(10).string(message.database);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlanConfig_ChangeDatabaseConfig_PreUpdateBackupDetail {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlanConfig_ChangeDatabaseConfig_PreUpdateBackupDetail();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.database = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlanConfig_ChangeDatabaseConfig_PreUpdateBackupDetail {
    return { database: isSet(object.database) ? globalThis.String(object.database) : "" };
  },

  toJSON(message: PlanConfig_ChangeDatabaseConfig_PreUpdateBackupDetail): unknown {
    const obj: any = {};
    if (message.database !== "") {
      obj.database = message.database;
    }
    return obj;
  },

  create(
    base?: DeepPartial<PlanConfig_ChangeDatabaseConfig_PreUpdateBackupDetail>,
  ): PlanConfig_ChangeDatabaseConfig_PreUpdateBackupDetail {
    return PlanConfig_ChangeDatabaseConfig_PreUpdateBackupDetail.fromPartial(base ?? {});
  },
  fromPartial(
    object: DeepPartial<PlanConfig_ChangeDatabaseConfig_PreUpdateBackupDetail>,
  ): PlanConfig_ChangeDatabaseConfig_PreUpdateBackupDetail {
    const message = createBasePlanConfig_ChangeDatabaseConfig_PreUpdateBackupDetail();
    message.database = object.database ?? "";
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
      target: isSet(object.target) ? globalThis.String(object.target) : "",
      createDatabaseConfig: isSet(object.createDatabaseConfig)
        ? PlanConfig_CreateDatabaseConfig.fromJSON(object.createDatabaseConfig)
        : undefined,
      backup: isSet(object.backup) ? globalThis.String(object.backup) : undefined,
      pointInTime: isSet(object.pointInTime) ? fromJsonTimestamp(object.pointInTime) : undefined,
    };
  },

  toJSON(message: PlanConfig_RestoreDatabaseConfig): unknown {
    const obj: any = {};
    if (message.target !== "") {
      obj.target = message.target;
    }
    if (message.createDatabaseConfig !== undefined) {
      obj.createDatabaseConfig = PlanConfig_CreateDatabaseConfig.toJSON(message.createDatabaseConfig);
    }
    if (message.backup !== undefined) {
      obj.backup = message.backup;
    }
    if (message.pointInTime !== undefined) {
      obj.pointInTime = message.pointInTime.toISOString();
    }
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

function isObject(value: any): boolean {
  return typeof value === "object" && value !== null;
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
