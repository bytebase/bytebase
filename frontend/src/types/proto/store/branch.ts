/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Timestamp } from "../google/protobuf/timestamp";
import { StringValue } from "../google/protobuf/wrappers";
import { DatabaseConfig, DatabaseSchemaMetadata, ExtensionMetadata } from "./database";

export const protobufPackage = "bytebase.store";

export interface BranchSnapshot {
  metadata: DatabaseSchemaMetadata | undefined;
  databaseConfig: DatabaseConfig | undefined;
}

export interface BranchConfig {
  /**
   * The name of source database.
   * Optional.
   * Example: instances/instance-id/databases/database-name.
   */
  sourceDatabase: string;
  /**
   * The name of the source branch.
   * Optional.
   * Example: projects/project-id/branches/branch-id.
   */
  sourceBranch: string;
}

/** BranchDatabaseSchemaMetadata is the schema metadata for databases used in branch. */
export interface BranchDatabaseSchemaMetadata {
  name: string;
  /** The schemas is the list of schemas in a database. */
  schemas: BranchSchemaMetadata[];
  /** The character_set is the character set of a database. */
  characterSet: string;
  /** The collation is the collation of a database. */
  collation: string;
  /** The extensions is the list of extensions in a database. */
  extensions: ExtensionMetadata[];
  /** The database belongs to a datashare. */
  datashare: boolean;
  /** The service name of the database. It's the Oracle specific concept. */
  serviceName: string;
}

/**
 * BranchSchemaMetadata is the metadata for schemas used in branch.
 * This is the concept of schema in Postgres, but it's a no-op for MySQL.
 */
export interface BranchSchemaMetadata {
  /**
   * The name is the schema name.
   * It is an empty string for databases without such concept such as MySQL.
   */
  name: string;
  /** The tables is the list of tables in a schema. */
  tables: BranchTableMetadata[];
  /** The external_tables is the list of external tables in a schema. */
  externalTables: BranchExternalTableMetadata[];
  /** The views is the list of views in a schema. */
  views: BranchViewMetadata[];
  /** The functions is the list of functions in a schema. */
  functions: BranchFunctionMetadata[];
  /** The procedures is the list of procedures in a schema. */
  procedures: BranchProcedureMetadata[];
  /** The streams is the list of streams in a schema, currently, only used for Snowflake. */
  streams: BranchStreamMetadata[];
  /** The routines is the list of routines in a schema, currently, only used for Snowflake. */
  tasks: BranchTaskMetadata[];
  /** The materialized_views is the list of materialized views in a schema. */
  materializedViews: BranchMaterializedViewMetadata[];
  /**
   * The last updater of the object.
   * Format: users/{email}
   */
  updater: string;
  /** The timestamp when the object was last updated. */
  updateTime: Date | undefined;
}

export interface BranchTaskMetadata {
  /** The name is the name of a task. */
  name: string;
  /**
   * The id is the snowflake-generated id of a task.
   * Example: 01ad32a0-1bb6-5e93-0000-000000000001
   */
  id: string;
  /** The owner of the task. */
  owner: string;
  /** The comment of the task. */
  comment: string;
  /** The warehouse of the task. */
  warehouse: string;
  /** The schedule interval of the task. */
  schedule: string;
  /** The predecessor tasks of the task. */
  predecessors: string[];
  /** The state of the task. */
  state: BranchTaskMetadata_State;
  /** The condition of the task. */
  condition: string;
  /** The definition of the task. */
  definition: string;
}

export enum BranchTaskMetadata_State {
  STATE_UNSPECIFIED = "STATE_UNSPECIFIED",
  STATE_STARTED = "STATE_STARTED",
  STATE_SUSPENDED = "STATE_SUSPENDED",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function branchTaskMetadata_StateFromJSON(object: any): BranchTaskMetadata_State {
  switch (object) {
    case 0:
    case "STATE_UNSPECIFIED":
      return BranchTaskMetadata_State.STATE_UNSPECIFIED;
    case 1:
    case "STATE_STARTED":
      return BranchTaskMetadata_State.STATE_STARTED;
    case 2:
    case "STATE_SUSPENDED":
      return BranchTaskMetadata_State.STATE_SUSPENDED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return BranchTaskMetadata_State.UNRECOGNIZED;
  }
}

export function branchTaskMetadata_StateToJSON(object: BranchTaskMetadata_State): string {
  switch (object) {
    case BranchTaskMetadata_State.STATE_UNSPECIFIED:
      return "STATE_UNSPECIFIED";
    case BranchTaskMetadata_State.STATE_STARTED:
      return "STATE_STARTED";
    case BranchTaskMetadata_State.STATE_SUSPENDED:
      return "STATE_SUSPENDED";
    case BranchTaskMetadata_State.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function branchTaskMetadata_StateToNumber(object: BranchTaskMetadata_State): number {
  switch (object) {
    case BranchTaskMetadata_State.STATE_UNSPECIFIED:
      return 0;
    case BranchTaskMetadata_State.STATE_STARTED:
      return 1;
    case BranchTaskMetadata_State.STATE_SUSPENDED:
      return 2;
    case BranchTaskMetadata_State.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface BranchStreamMetadata {
  /** The name is the name of a stream. */
  name: string;
  /** The table_name is the name of the table/view that the stream is created on. */
  tableName: string;
  /** The owner of the stream. */
  owner: string;
  /** The comment of the stream. */
  comment: string;
  /** The type of the stream. */
  type: BranchStreamMetadata_Type;
  /** Indicates whether the stream was last read before the `stale_after` time. */
  stale: boolean;
  /** The mode of the stream. */
  mode: BranchStreamMetadata_Mode;
  /** The definition of the stream. */
  definition: string;
}

export enum BranchStreamMetadata_Type {
  TYPE_UNSPECIFIED = "TYPE_UNSPECIFIED",
  TYPE_DELTA = "TYPE_DELTA",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function branchStreamMetadata_TypeFromJSON(object: any): BranchStreamMetadata_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return BranchStreamMetadata_Type.TYPE_UNSPECIFIED;
    case 1:
    case "TYPE_DELTA":
      return BranchStreamMetadata_Type.TYPE_DELTA;
    case -1:
    case "UNRECOGNIZED":
    default:
      return BranchStreamMetadata_Type.UNRECOGNIZED;
  }
}

export function branchStreamMetadata_TypeToJSON(object: BranchStreamMetadata_Type): string {
  switch (object) {
    case BranchStreamMetadata_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case BranchStreamMetadata_Type.TYPE_DELTA:
      return "TYPE_DELTA";
    case BranchStreamMetadata_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function branchStreamMetadata_TypeToNumber(object: BranchStreamMetadata_Type): number {
  switch (object) {
    case BranchStreamMetadata_Type.TYPE_UNSPECIFIED:
      return 0;
    case BranchStreamMetadata_Type.TYPE_DELTA:
      return 1;
    case BranchStreamMetadata_Type.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum BranchStreamMetadata_Mode {
  MODE_UNSPECIFIED = "MODE_UNSPECIFIED",
  MODE_DEFAULT = "MODE_DEFAULT",
  MODE_APPEND_ONLY = "MODE_APPEND_ONLY",
  MODE_INSERT_ONLY = "MODE_INSERT_ONLY",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function branchStreamMetadata_ModeFromJSON(object: any): BranchStreamMetadata_Mode {
  switch (object) {
    case 0:
    case "MODE_UNSPECIFIED":
      return BranchStreamMetadata_Mode.MODE_UNSPECIFIED;
    case 1:
    case "MODE_DEFAULT":
      return BranchStreamMetadata_Mode.MODE_DEFAULT;
    case 2:
    case "MODE_APPEND_ONLY":
      return BranchStreamMetadata_Mode.MODE_APPEND_ONLY;
    case 3:
    case "MODE_INSERT_ONLY":
      return BranchStreamMetadata_Mode.MODE_INSERT_ONLY;
    case -1:
    case "UNRECOGNIZED":
    default:
      return BranchStreamMetadata_Mode.UNRECOGNIZED;
  }
}

export function branchStreamMetadata_ModeToJSON(object: BranchStreamMetadata_Mode): string {
  switch (object) {
    case BranchStreamMetadata_Mode.MODE_UNSPECIFIED:
      return "MODE_UNSPECIFIED";
    case BranchStreamMetadata_Mode.MODE_DEFAULT:
      return "MODE_DEFAULT";
    case BranchStreamMetadata_Mode.MODE_APPEND_ONLY:
      return "MODE_APPEND_ONLY";
    case BranchStreamMetadata_Mode.MODE_INSERT_ONLY:
      return "MODE_INSERT_ONLY";
    case BranchStreamMetadata_Mode.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function branchStreamMetadata_ModeToNumber(object: BranchStreamMetadata_Mode): number {
  switch (object) {
    case BranchStreamMetadata_Mode.MODE_UNSPECIFIED:
      return 0;
    case BranchStreamMetadata_Mode.MODE_DEFAULT:
      return 1;
    case BranchStreamMetadata_Mode.MODE_APPEND_ONLY:
      return 2;
    case BranchStreamMetadata_Mode.MODE_INSERT_ONLY:
      return 3;
    case BranchStreamMetadata_Mode.UNRECOGNIZED:
    default:
      return -1;
  }
}

/** BranchTableMetadata is the metadata for tables used in branch. */
export interface BranchTableMetadata {
  /** The name is the name of a table. */
  name: string;
  /** The columns is the ordered list of columns in a table. */
  columns: BranchColumnMetadata[];
  /** The indexes is the list of indexes in a table. */
  indexes: BranchIndexMetadata[];
  /** The engine is the engine of a table. */
  engine: string;
  /** The collation is the collation of a table. */
  collation: string;
  /** The row_count is the estimated number of rows of a table. */
  rowCount: Long;
  /** The data_size is the estimated data size of a table. */
  dataSize: Long;
  /** The index_size is the estimated index size of a table. */
  indexSize: Long;
  /** The data_free is the estimated free data size of a table. */
  dataFree: Long;
  /** The create_options is the create option of a table. */
  createOptions: string;
  /**
   * The comment is the comment of a table.
   * classification and user_comment is parsed from the comment.
   */
  comment: string;
  /** The classification is the classification of a table parsed from the comment. */
  classification: string;
  /** The user_comment is the user comment of a table parsed from the comment. */
  userComment: string;
  /** The foreign_keys is the list of foreign keys in a table. */
  foreignKeys: BranchForeignKeyMetadata[];
  /** The partitions is the list of partitions in a table. */
  partitions: BranchTablePartitionMetadata[];
  /**
   * The last updater of the object.
   * Format: users/{email}
   */
  updater: string;
  /** The timestamp when the object was last updated. */
  updateTime: Date | undefined;
}

export interface BranchExternalTableMetadata {
  /** The name is the name of a external table. */
  name: string;
  /** The external_server_name is the name of the external server. */
  externalServerName: string;
  /** The external_database_name is the name of the external database. */
  externalDatabaseName: string;
  /** The columns is the ordered list of columns in a foreign table. */
  columns: BranchColumnMetadata[];
}

/** BranchTablePartitionMetadata is the metadata for table partitions used in branch. */
export interface BranchTablePartitionMetadata {
  /** The name is the name of a table partition. */
  name: string;
  /** The type of a table partition. */
  type: BranchTablePartitionMetadata_Type;
  /**
   * The expression is the expression of a table partition.
   * For PostgreSQL, the expression is the text of {FOR VALUES partition_bound_spec}, see https://www.postgresql.org/docs/current/sql-createtable.html.
   * For MySQL, the expression is the `expr` or `column_list` of the following syntax.
   * PARTITION BY
   *    { [LINEAR] HASH(expr)
   *    | [LINEAR] KEY [ALGORITHM={1 | 2}] (column_list)
   *    | RANGE{(expr) | COLUMNS(column_list)}
   *    | LIST{(expr) | COLUMNS(column_list)} }.
   */
  expression: string;
  /**
   * The value is the value of a table partition.
   * For MySQL, the value is for RANGE and LIST partition types,
   * - For a RANGE partition, it contains the value set in the partition's VALUES LESS THAN clause, which can be either an integer or MAXVALUE.
   * - For a LIST partition, this column contains the values defined in the partition's VALUES IN clause, which is a list of comma-separated integer values.
   * - For others, it's an empty string.
   */
  value: string;
  /**
   * The use_default is whether the users use the default partition, it stores the different value for different database engines.
   * For MySQL, it's [INT] type, 0 means not use default partition, otherwise, it's equals to number in syntax [SUB]PARTITION {number}.
   */
  useDefault: string;
  /** The subpartitions is the list of subpartitions in a table partition. */
  subpartitions: BranchTablePartitionMetadata[];
}

/**
 * Type is the type of a table partition, some database engines may not support all types.
 * Only avilable for the following database engines now:
 * MySQL: RANGE, RANGE COLUMNS, LIST, LIST COLUMNS, HASH, LINEAR HASH, KEY, LINEAR_KEY (https://dev.mysql.com/doc/refman/8.0/en/partitioning-types.html)
 * TiDB: RANGE, RANGE COLUMNS, LIST, LIST COLUMNS, HASH, KEY
 * PostgreSQL: RANGE, LIST, HASH (https://www.postgresql.org/docs/current/ddl-partitioning.html)
 */
export enum BranchTablePartitionMetadata_Type {
  TYPE_UNSPECIFIED = "TYPE_UNSPECIFIED",
  RANGE = "RANGE",
  RANGE_COLUMNS = "RANGE_COLUMNS",
  LIST = "LIST",
  LIST_COLUMNS = "LIST_COLUMNS",
  HASH = "HASH",
  LINEAR_HASH = "LINEAR_HASH",
  KEY = "KEY",
  LINEAR_KEY = "LINEAR_KEY",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function branchTablePartitionMetadata_TypeFromJSON(object: any): BranchTablePartitionMetadata_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return BranchTablePartitionMetadata_Type.TYPE_UNSPECIFIED;
    case 1:
    case "RANGE":
      return BranchTablePartitionMetadata_Type.RANGE;
    case 2:
    case "RANGE_COLUMNS":
      return BranchTablePartitionMetadata_Type.RANGE_COLUMNS;
    case 3:
    case "LIST":
      return BranchTablePartitionMetadata_Type.LIST;
    case 4:
    case "LIST_COLUMNS":
      return BranchTablePartitionMetadata_Type.LIST_COLUMNS;
    case 5:
    case "HASH":
      return BranchTablePartitionMetadata_Type.HASH;
    case 6:
    case "LINEAR_HASH":
      return BranchTablePartitionMetadata_Type.LINEAR_HASH;
    case 7:
    case "KEY":
      return BranchTablePartitionMetadata_Type.KEY;
    case 8:
    case "LINEAR_KEY":
      return BranchTablePartitionMetadata_Type.LINEAR_KEY;
    case -1:
    case "UNRECOGNIZED":
    default:
      return BranchTablePartitionMetadata_Type.UNRECOGNIZED;
  }
}

export function branchTablePartitionMetadata_TypeToJSON(object: BranchTablePartitionMetadata_Type): string {
  switch (object) {
    case BranchTablePartitionMetadata_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case BranchTablePartitionMetadata_Type.RANGE:
      return "RANGE";
    case BranchTablePartitionMetadata_Type.RANGE_COLUMNS:
      return "RANGE_COLUMNS";
    case BranchTablePartitionMetadata_Type.LIST:
      return "LIST";
    case BranchTablePartitionMetadata_Type.LIST_COLUMNS:
      return "LIST_COLUMNS";
    case BranchTablePartitionMetadata_Type.HASH:
      return "HASH";
    case BranchTablePartitionMetadata_Type.LINEAR_HASH:
      return "LINEAR_HASH";
    case BranchTablePartitionMetadata_Type.KEY:
      return "KEY";
    case BranchTablePartitionMetadata_Type.LINEAR_KEY:
      return "LINEAR_KEY";
    case BranchTablePartitionMetadata_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function branchTablePartitionMetadata_TypeToNumber(object: BranchTablePartitionMetadata_Type): number {
  switch (object) {
    case BranchTablePartitionMetadata_Type.TYPE_UNSPECIFIED:
      return 0;
    case BranchTablePartitionMetadata_Type.RANGE:
      return 1;
    case BranchTablePartitionMetadata_Type.RANGE_COLUMNS:
      return 2;
    case BranchTablePartitionMetadata_Type.LIST:
      return 3;
    case BranchTablePartitionMetadata_Type.LIST_COLUMNS:
      return 4;
    case BranchTablePartitionMetadata_Type.HASH:
      return 5;
    case BranchTablePartitionMetadata_Type.LINEAR_HASH:
      return 6;
    case BranchTablePartitionMetadata_Type.KEY:
      return 7;
    case BranchTablePartitionMetadata_Type.LINEAR_KEY:
      return 8;
    case BranchTablePartitionMetadata_Type.UNRECOGNIZED:
    default:
      return -1;
  }
}

/** BranchColumnMetadata is the metadata for columns used in branch. */
export interface BranchColumnMetadata {
  /** The name is the name of a column. */
  name: string;
  /** The position is the position in columns. */
  position: number;
  /** The default is the default of a column. Use google.protobuf.StringValue to distinguish between an empty string default value or no default. */
  default?: string | undefined;
  defaultNull?: boolean | undefined;
  defaultExpression?:
    | string
    | undefined;
  /**
   * The on_update is the on update action of a column.
   * For MySQL like databases, it's only supported for TIMESTAMP columns with CURRENT_TIMESTAMP as on update value.
   */
  onUpdate: string;
  /** The nullable is the nullable of a column. */
  nullable: boolean;
  /** The type is the type of a column. */
  type: string;
  /** The character_set is the character_set of a column. */
  characterSet: string;
  /** The collation is the collation of a column. */
  collation: string;
  /**
   * The comment is the comment of a column.
   * classification and user_comment is parsed from the comment.
   */
  comment: string;
  /** The classification is the classification of a table parsed from the comment. */
  classification: string;
  /** The user_comment is the user comment of a table parsed from the comment. */
  userComment: string;
}

/** BranchViewMetadata is the metadata for views used in branch. */
export interface BranchViewMetadata {
  /** The name is the name of a view. */
  name: string;
  /** The definition is the definition of a view. */
  definition: string;
  /** The comment is the comment of a view. */
  comment: string;
  /** The dependent_columns is the list of dependent columns of a view. */
  dependentColumns: BranchDependentColumn[];
  /**
   * The last updater of the object.
   * Format: users/{email}
   */
  updater: string;
  /** The timestamp when the object was last updated. */
  updateTime: Date | undefined;
}

/** BranchDependentColumn is the metadata for dependent columns used in branch. */
export interface BranchDependentColumn {
  /** The schema is the schema of a reference column. */
  schema: string;
  /** The table is the table of a reference column. */
  table: string;
  /** The column is the name of a reference column. */
  column: string;
}

/** BranchMaterializedViewMetadata is the metadata for materialized views used in branch. */
export interface BranchMaterializedViewMetadata {
  /** The name is the name of a view. */
  name: string;
  /** The definition is the definition of a view. */
  definition: string;
  /** The comment is the comment of a view. */
  comment: string;
  /** The dependent_columns is the list of dependent columns of a view. */
  dependentColumns: BranchDependentColumn[];
}

/** BranchFunctionMetadata is the metadata for functions used in branch. */
export interface BranchFunctionMetadata {
  /** The name is the name of a function. */
  name: string;
  /** The definition is the definition of a function. */
  definition: string;
  /**
   * The last updater of the object.
   * Format: users/{email}
   */
  updater: string;
  /** The timestamp when the object was last updated. */
  updateTime: Date | undefined;
}

/** BranchProcedureMetadata is the metadata for procedures used in branch. */
export interface BranchProcedureMetadata {
  /** The name is the name of a procedure. */
  name: string;
  /** The definition is the definition of a procedure. */
  definition: string;
  /**
   * The last updater of the object.
   * Format: users/{email}
   */
  updater: string;
  /** The timestamp when the object was last updated. */
  updateTime: Date | undefined;
}

/** BranchIndexMetadata is the metadata for indexes used in branch. */
export interface BranchIndexMetadata {
  /** The name is the name of an index. */
  name: string;
  /**
   * The expressions are the ordered columns or expressions of an index.
   * This could refer to a column or an expression.
   */
  expressions: string[];
  /**
   * The key_lengths are the ordered key lengths of an index.
   * If the key length is not specified, it's -1.
   */
  keyLength: Long[];
  /** The type is the type of an index. */
  type: string;
  /** The unique is whether the index is unique. */
  unique: boolean;
  /** The primary is whether the index is a primary key index. */
  primary: boolean;
  /** The visible is whether the index is visible. */
  visible: boolean;
  /** The comment is the comment of an index. */
  comment: string;
  /** The definition of an index. */
  definition: string;
}

/** BranchExtensionMetadata is the metadata for extensions used in branch. */
export interface BranchExtensionMetadata {
  /** The name is the name of an extension. */
  name: string;
  /** The schema is the extension that is installed to. But the extension usage is not limited to the schema. */
  schema: string;
  /** The version is the version of an extension. */
  version: string;
  /** The description is the description of an extension. */
  description: string;
}

/** BranchForeignKeyMetadata is the metadata for foreign keys used in branch. */
export interface BranchForeignKeyMetadata {
  /** The name is the name of a foreign key. */
  name: string;
  /** The columns are the ordered referencing columns of a foreign key. */
  columns: string[];
  /**
   * The referenced_schema is the referenced schema name of a foreign key.
   * It is an empty string for databases without such concept such as MySQL.
   */
  referencedSchema: string;
  /** The referenced_table is the referenced table name of a foreign key. */
  referencedTable: string;
  /** The referenced_columns are the ordered referenced columns of a foreign key. */
  referencedColumns: string[];
  /** The on_delete is the on delete action of a foreign key. */
  onDelete: string;
  /** The on_update is the on update action of a foreign key. */
  onUpdate: string;
  /**
   * The match_type is the match type of a foreign key.
   * The match_type is the PostgreSQL specific field.
   * It's empty string for other databases.
   */
  matchType: string;
}

function createBaseBranchSnapshot(): BranchSnapshot {
  return { metadata: undefined, databaseConfig: undefined };
}

export const BranchSnapshot = {
  encode(message: BranchSnapshot, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.metadata !== undefined) {
      DatabaseSchemaMetadata.encode(message.metadata, writer.uint32(10).fork()).ldelim();
    }
    if (message.databaseConfig !== undefined) {
      DatabaseConfig.encode(message.databaseConfig, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BranchSnapshot {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBranchSnapshot();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.metadata = DatabaseSchemaMetadata.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.databaseConfig = DatabaseConfig.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BranchSnapshot {
    return {
      metadata: isSet(object.metadata) ? DatabaseSchemaMetadata.fromJSON(object.metadata) : undefined,
      databaseConfig: isSet(object.databaseConfig) ? DatabaseConfig.fromJSON(object.databaseConfig) : undefined,
    };
  },

  toJSON(message: BranchSnapshot): unknown {
    const obj: any = {};
    if (message.metadata !== undefined) {
      obj.metadata = DatabaseSchemaMetadata.toJSON(message.metadata);
    }
    if (message.databaseConfig !== undefined) {
      obj.databaseConfig = DatabaseConfig.toJSON(message.databaseConfig);
    }
    return obj;
  },

  create(base?: DeepPartial<BranchSnapshot>): BranchSnapshot {
    return BranchSnapshot.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BranchSnapshot>): BranchSnapshot {
    const message = createBaseBranchSnapshot();
    message.metadata = (object.metadata !== undefined && object.metadata !== null)
      ? DatabaseSchemaMetadata.fromPartial(object.metadata)
      : undefined;
    message.databaseConfig = (object.databaseConfig !== undefined && object.databaseConfig !== null)
      ? DatabaseConfig.fromPartial(object.databaseConfig)
      : undefined;
    return message;
  },
};

function createBaseBranchConfig(): BranchConfig {
  return { sourceDatabase: "", sourceBranch: "" };
}

export const BranchConfig = {
  encode(message: BranchConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sourceDatabase !== "") {
      writer.uint32(10).string(message.sourceDatabase);
    }
    if (message.sourceBranch !== "") {
      writer.uint32(18).string(message.sourceBranch);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BranchConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBranchConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.sourceDatabase = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.sourceBranch = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BranchConfig {
    return {
      sourceDatabase: isSet(object.sourceDatabase) ? globalThis.String(object.sourceDatabase) : "",
      sourceBranch: isSet(object.sourceBranch) ? globalThis.String(object.sourceBranch) : "",
    };
  },

  toJSON(message: BranchConfig): unknown {
    const obj: any = {};
    if (message.sourceDatabase !== "") {
      obj.sourceDatabase = message.sourceDatabase;
    }
    if (message.sourceBranch !== "") {
      obj.sourceBranch = message.sourceBranch;
    }
    return obj;
  },

  create(base?: DeepPartial<BranchConfig>): BranchConfig {
    return BranchConfig.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BranchConfig>): BranchConfig {
    const message = createBaseBranchConfig();
    message.sourceDatabase = object.sourceDatabase ?? "";
    message.sourceBranch = object.sourceBranch ?? "";
    return message;
  },
};

function createBaseBranchDatabaseSchemaMetadata(): BranchDatabaseSchemaMetadata {
  return { name: "", schemas: [], characterSet: "", collation: "", extensions: [], datashare: false, serviceName: "" };
}

export const BranchDatabaseSchemaMetadata = {
  encode(message: BranchDatabaseSchemaMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.schemas) {
      BranchSchemaMetadata.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.characterSet !== "") {
      writer.uint32(26).string(message.characterSet);
    }
    if (message.collation !== "") {
      writer.uint32(34).string(message.collation);
    }
    for (const v of message.extensions) {
      ExtensionMetadata.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    if (message.datashare === true) {
      writer.uint32(48).bool(message.datashare);
    }
    if (message.serviceName !== "") {
      writer.uint32(58).string(message.serviceName);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BranchDatabaseSchemaMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBranchDatabaseSchemaMetadata();
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

          message.schemas.push(BranchSchemaMetadata.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.characterSet = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.collation = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.extensions.push(ExtensionMetadata.decode(reader, reader.uint32()));
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.datashare = reader.bool();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.serviceName = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BranchDatabaseSchemaMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      schemas: globalThis.Array.isArray(object?.schemas)
        ? object.schemas.map((e: any) => BranchSchemaMetadata.fromJSON(e))
        : [],
      characterSet: isSet(object.characterSet) ? globalThis.String(object.characterSet) : "",
      collation: isSet(object.collation) ? globalThis.String(object.collation) : "",
      extensions: globalThis.Array.isArray(object?.extensions)
        ? object.extensions.map((e: any) => ExtensionMetadata.fromJSON(e))
        : [],
      datashare: isSet(object.datashare) ? globalThis.Boolean(object.datashare) : false,
      serviceName: isSet(object.serviceName) ? globalThis.String(object.serviceName) : "",
    };
  },

  toJSON(message: BranchDatabaseSchemaMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.schemas?.length) {
      obj.schemas = message.schemas.map((e) => BranchSchemaMetadata.toJSON(e));
    }
    if (message.characterSet !== "") {
      obj.characterSet = message.characterSet;
    }
    if (message.collation !== "") {
      obj.collation = message.collation;
    }
    if (message.extensions?.length) {
      obj.extensions = message.extensions.map((e) => ExtensionMetadata.toJSON(e));
    }
    if (message.datashare === true) {
      obj.datashare = message.datashare;
    }
    if (message.serviceName !== "") {
      obj.serviceName = message.serviceName;
    }
    return obj;
  },

  create(base?: DeepPartial<BranchDatabaseSchemaMetadata>): BranchDatabaseSchemaMetadata {
    return BranchDatabaseSchemaMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BranchDatabaseSchemaMetadata>): BranchDatabaseSchemaMetadata {
    const message = createBaseBranchDatabaseSchemaMetadata();
    message.name = object.name ?? "";
    message.schemas = object.schemas?.map((e) => BranchSchemaMetadata.fromPartial(e)) || [];
    message.characterSet = object.characterSet ?? "";
    message.collation = object.collation ?? "";
    message.extensions = object.extensions?.map((e) => ExtensionMetadata.fromPartial(e)) || [];
    message.datashare = object.datashare ?? false;
    message.serviceName = object.serviceName ?? "";
    return message;
  },
};

function createBaseBranchSchemaMetadata(): BranchSchemaMetadata {
  return {
    name: "",
    tables: [],
    externalTables: [],
    views: [],
    functions: [],
    procedures: [],
    streams: [],
    tasks: [],
    materializedViews: [],
    updater: "",
    updateTime: undefined,
  };
}

export const BranchSchemaMetadata = {
  encode(message: BranchSchemaMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.tables) {
      BranchTableMetadata.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.externalTables) {
      BranchExternalTableMetadata.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.views) {
      BranchViewMetadata.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.functions) {
      BranchFunctionMetadata.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    for (const v of message.procedures) {
      BranchProcedureMetadata.encode(v!, writer.uint32(50).fork()).ldelim();
    }
    for (const v of message.streams) {
      BranchStreamMetadata.encode(v!, writer.uint32(58).fork()).ldelim();
    }
    for (const v of message.tasks) {
      BranchTaskMetadata.encode(v!, writer.uint32(66).fork()).ldelim();
    }
    for (const v of message.materializedViews) {
      BranchMaterializedViewMetadata.encode(v!, writer.uint32(74).fork()).ldelim();
    }
    if (message.updater !== "") {
      writer.uint32(82).string(message.updater);
    }
    if (message.updateTime !== undefined) {
      Timestamp.encode(toTimestamp(message.updateTime), writer.uint32(90).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BranchSchemaMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBranchSchemaMetadata();
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

          message.tables.push(BranchTableMetadata.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.externalTables.push(BranchExternalTableMetadata.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.views.push(BranchViewMetadata.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.functions.push(BranchFunctionMetadata.decode(reader, reader.uint32()));
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.procedures.push(BranchProcedureMetadata.decode(reader, reader.uint32()));
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.streams.push(BranchStreamMetadata.decode(reader, reader.uint32()));
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.tasks.push(BranchTaskMetadata.decode(reader, reader.uint32()));
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.materializedViews.push(BranchMaterializedViewMetadata.decode(reader, reader.uint32()));
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.updater = reader.string();
          continue;
        case 11:
          if (tag !== 90) {
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

  fromJSON(object: any): BranchSchemaMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      tables: globalThis.Array.isArray(object?.tables)
        ? object.tables.map((e: any) => BranchTableMetadata.fromJSON(e))
        : [],
      externalTables: globalThis.Array.isArray(object?.externalTables)
        ? object.externalTables.map((e: any) => BranchExternalTableMetadata.fromJSON(e))
        : [],
      views: globalThis.Array.isArray(object?.views)
        ? object.views.map((e: any) => BranchViewMetadata.fromJSON(e))
        : [],
      functions: globalThis.Array.isArray(object?.functions)
        ? object.functions.map((e: any) => BranchFunctionMetadata.fromJSON(e))
        : [],
      procedures: globalThis.Array.isArray(object?.procedures)
        ? object.procedures.map((e: any) => BranchProcedureMetadata.fromJSON(e))
        : [],
      streams: globalThis.Array.isArray(object?.streams)
        ? object.streams.map((e: any) => BranchStreamMetadata.fromJSON(e))
        : [],
      tasks: globalThis.Array.isArray(object?.tasks)
        ? object.tasks.map((e: any) => BranchTaskMetadata.fromJSON(e))
        : [],
      materializedViews: globalThis.Array.isArray(object?.materializedViews)
        ? object.materializedViews.map((e: any) => BranchMaterializedViewMetadata.fromJSON(e))
        : [],
      updater: isSet(object.updater) ? globalThis.String(object.updater) : "",
      updateTime: isSet(object.updateTime) ? fromJsonTimestamp(object.updateTime) : undefined,
    };
  },

  toJSON(message: BranchSchemaMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.tables?.length) {
      obj.tables = message.tables.map((e) => BranchTableMetadata.toJSON(e));
    }
    if (message.externalTables?.length) {
      obj.externalTables = message.externalTables.map((e) => BranchExternalTableMetadata.toJSON(e));
    }
    if (message.views?.length) {
      obj.views = message.views.map((e) => BranchViewMetadata.toJSON(e));
    }
    if (message.functions?.length) {
      obj.functions = message.functions.map((e) => BranchFunctionMetadata.toJSON(e));
    }
    if (message.procedures?.length) {
      obj.procedures = message.procedures.map((e) => BranchProcedureMetadata.toJSON(e));
    }
    if (message.streams?.length) {
      obj.streams = message.streams.map((e) => BranchStreamMetadata.toJSON(e));
    }
    if (message.tasks?.length) {
      obj.tasks = message.tasks.map((e) => BranchTaskMetadata.toJSON(e));
    }
    if (message.materializedViews?.length) {
      obj.materializedViews = message.materializedViews.map((e) => BranchMaterializedViewMetadata.toJSON(e));
    }
    if (message.updater !== "") {
      obj.updater = message.updater;
    }
    if (message.updateTime !== undefined) {
      obj.updateTime = message.updateTime.toISOString();
    }
    return obj;
  },

  create(base?: DeepPartial<BranchSchemaMetadata>): BranchSchemaMetadata {
    return BranchSchemaMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BranchSchemaMetadata>): BranchSchemaMetadata {
    const message = createBaseBranchSchemaMetadata();
    message.name = object.name ?? "";
    message.tables = object.tables?.map((e) => BranchTableMetadata.fromPartial(e)) || [];
    message.externalTables = object.externalTables?.map((e) => BranchExternalTableMetadata.fromPartial(e)) || [];
    message.views = object.views?.map((e) => BranchViewMetadata.fromPartial(e)) || [];
    message.functions = object.functions?.map((e) => BranchFunctionMetadata.fromPartial(e)) || [];
    message.procedures = object.procedures?.map((e) => BranchProcedureMetadata.fromPartial(e)) || [];
    message.streams = object.streams?.map((e) => BranchStreamMetadata.fromPartial(e)) || [];
    message.tasks = object.tasks?.map((e) => BranchTaskMetadata.fromPartial(e)) || [];
    message.materializedViews = object.materializedViews?.map((e) => BranchMaterializedViewMetadata.fromPartial(e)) ||
      [];
    message.updater = object.updater ?? "";
    message.updateTime = object.updateTime ?? undefined;
    return message;
  },
};

function createBaseBranchTaskMetadata(): BranchTaskMetadata {
  return {
    name: "",
    id: "",
    owner: "",
    comment: "",
    warehouse: "",
    schedule: "",
    predecessors: [],
    state: BranchTaskMetadata_State.STATE_UNSPECIFIED,
    condition: "",
    definition: "",
  };
}

export const BranchTaskMetadata = {
  encode(message: BranchTaskMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.id !== "") {
      writer.uint32(18).string(message.id);
    }
    if (message.owner !== "") {
      writer.uint32(26).string(message.owner);
    }
    if (message.comment !== "") {
      writer.uint32(34).string(message.comment);
    }
    if (message.warehouse !== "") {
      writer.uint32(42).string(message.warehouse);
    }
    if (message.schedule !== "") {
      writer.uint32(50).string(message.schedule);
    }
    for (const v of message.predecessors) {
      writer.uint32(58).string(v!);
    }
    if (message.state !== BranchTaskMetadata_State.STATE_UNSPECIFIED) {
      writer.uint32(64).int32(branchTaskMetadata_StateToNumber(message.state));
    }
    if (message.condition !== "") {
      writer.uint32(74).string(message.condition);
    }
    if (message.definition !== "") {
      writer.uint32(82).string(message.definition);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BranchTaskMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBranchTaskMetadata();
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

          message.id = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.owner = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.comment = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.warehouse = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.schedule = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.predecessors.push(reader.string());
          continue;
        case 8:
          if (tag !== 64) {
            break;
          }

          message.state = branchTaskMetadata_StateFromJSON(reader.int32());
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.condition = reader.string();
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.definition = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BranchTaskMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      id: isSet(object.id) ? globalThis.String(object.id) : "",
      owner: isSet(object.owner) ? globalThis.String(object.owner) : "",
      comment: isSet(object.comment) ? globalThis.String(object.comment) : "",
      warehouse: isSet(object.warehouse) ? globalThis.String(object.warehouse) : "",
      schedule: isSet(object.schedule) ? globalThis.String(object.schedule) : "",
      predecessors: globalThis.Array.isArray(object?.predecessors)
        ? object.predecessors.map((e: any) => globalThis.String(e))
        : [],
      state: isSet(object.state)
        ? branchTaskMetadata_StateFromJSON(object.state)
        : BranchTaskMetadata_State.STATE_UNSPECIFIED,
      condition: isSet(object.condition) ? globalThis.String(object.condition) : "",
      definition: isSet(object.definition) ? globalThis.String(object.definition) : "",
    };
  },

  toJSON(message: BranchTaskMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.id !== "") {
      obj.id = message.id;
    }
    if (message.owner !== "") {
      obj.owner = message.owner;
    }
    if (message.comment !== "") {
      obj.comment = message.comment;
    }
    if (message.warehouse !== "") {
      obj.warehouse = message.warehouse;
    }
    if (message.schedule !== "") {
      obj.schedule = message.schedule;
    }
    if (message.predecessors?.length) {
      obj.predecessors = message.predecessors;
    }
    if (message.state !== BranchTaskMetadata_State.STATE_UNSPECIFIED) {
      obj.state = branchTaskMetadata_StateToJSON(message.state);
    }
    if (message.condition !== "") {
      obj.condition = message.condition;
    }
    if (message.definition !== "") {
      obj.definition = message.definition;
    }
    return obj;
  },

  create(base?: DeepPartial<BranchTaskMetadata>): BranchTaskMetadata {
    return BranchTaskMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BranchTaskMetadata>): BranchTaskMetadata {
    const message = createBaseBranchTaskMetadata();
    message.name = object.name ?? "";
    message.id = object.id ?? "";
    message.owner = object.owner ?? "";
    message.comment = object.comment ?? "";
    message.warehouse = object.warehouse ?? "";
    message.schedule = object.schedule ?? "";
    message.predecessors = object.predecessors?.map((e) => e) || [];
    message.state = object.state ?? BranchTaskMetadata_State.STATE_UNSPECIFIED;
    message.condition = object.condition ?? "";
    message.definition = object.definition ?? "";
    return message;
  },
};

function createBaseBranchStreamMetadata(): BranchStreamMetadata {
  return {
    name: "",
    tableName: "",
    owner: "",
    comment: "",
    type: BranchStreamMetadata_Type.TYPE_UNSPECIFIED,
    stale: false,
    mode: BranchStreamMetadata_Mode.MODE_UNSPECIFIED,
    definition: "",
  };
}

export const BranchStreamMetadata = {
  encode(message: BranchStreamMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.tableName !== "") {
      writer.uint32(18).string(message.tableName);
    }
    if (message.owner !== "") {
      writer.uint32(26).string(message.owner);
    }
    if (message.comment !== "") {
      writer.uint32(34).string(message.comment);
    }
    if (message.type !== BranchStreamMetadata_Type.TYPE_UNSPECIFIED) {
      writer.uint32(40).int32(branchStreamMetadata_TypeToNumber(message.type));
    }
    if (message.stale === true) {
      writer.uint32(48).bool(message.stale);
    }
    if (message.mode !== BranchStreamMetadata_Mode.MODE_UNSPECIFIED) {
      writer.uint32(56).int32(branchStreamMetadata_ModeToNumber(message.mode));
    }
    if (message.definition !== "") {
      writer.uint32(66).string(message.definition);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BranchStreamMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBranchStreamMetadata();
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

          message.tableName = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.owner = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.comment = reader.string();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.type = branchStreamMetadata_TypeFromJSON(reader.int32());
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.stale = reader.bool();
          continue;
        case 7:
          if (tag !== 56) {
            break;
          }

          message.mode = branchStreamMetadata_ModeFromJSON(reader.int32());
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.definition = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BranchStreamMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      tableName: isSet(object.tableName) ? globalThis.String(object.tableName) : "",
      owner: isSet(object.owner) ? globalThis.String(object.owner) : "",
      comment: isSet(object.comment) ? globalThis.String(object.comment) : "",
      type: isSet(object.type)
        ? branchStreamMetadata_TypeFromJSON(object.type)
        : BranchStreamMetadata_Type.TYPE_UNSPECIFIED,
      stale: isSet(object.stale) ? globalThis.Boolean(object.stale) : false,
      mode: isSet(object.mode)
        ? branchStreamMetadata_ModeFromJSON(object.mode)
        : BranchStreamMetadata_Mode.MODE_UNSPECIFIED,
      definition: isSet(object.definition) ? globalThis.String(object.definition) : "",
    };
  },

  toJSON(message: BranchStreamMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.tableName !== "") {
      obj.tableName = message.tableName;
    }
    if (message.owner !== "") {
      obj.owner = message.owner;
    }
    if (message.comment !== "") {
      obj.comment = message.comment;
    }
    if (message.type !== BranchStreamMetadata_Type.TYPE_UNSPECIFIED) {
      obj.type = branchStreamMetadata_TypeToJSON(message.type);
    }
    if (message.stale === true) {
      obj.stale = message.stale;
    }
    if (message.mode !== BranchStreamMetadata_Mode.MODE_UNSPECIFIED) {
      obj.mode = branchStreamMetadata_ModeToJSON(message.mode);
    }
    if (message.definition !== "") {
      obj.definition = message.definition;
    }
    return obj;
  },

  create(base?: DeepPartial<BranchStreamMetadata>): BranchStreamMetadata {
    return BranchStreamMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BranchStreamMetadata>): BranchStreamMetadata {
    const message = createBaseBranchStreamMetadata();
    message.name = object.name ?? "";
    message.tableName = object.tableName ?? "";
    message.owner = object.owner ?? "";
    message.comment = object.comment ?? "";
    message.type = object.type ?? BranchStreamMetadata_Type.TYPE_UNSPECIFIED;
    message.stale = object.stale ?? false;
    message.mode = object.mode ?? BranchStreamMetadata_Mode.MODE_UNSPECIFIED;
    message.definition = object.definition ?? "";
    return message;
  },
};

function createBaseBranchTableMetadata(): BranchTableMetadata {
  return {
    name: "",
    columns: [],
    indexes: [],
    engine: "",
    collation: "",
    rowCount: Long.ZERO,
    dataSize: Long.ZERO,
    indexSize: Long.ZERO,
    dataFree: Long.ZERO,
    createOptions: "",
    comment: "",
    classification: "",
    userComment: "",
    foreignKeys: [],
    partitions: [],
    updater: "",
    updateTime: undefined,
  };
}

export const BranchTableMetadata = {
  encode(message: BranchTableMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.columns) {
      BranchColumnMetadata.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.indexes) {
      BranchIndexMetadata.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    if (message.engine !== "") {
      writer.uint32(34).string(message.engine);
    }
    if (message.collation !== "") {
      writer.uint32(42).string(message.collation);
    }
    if (!message.rowCount.isZero()) {
      writer.uint32(48).int64(message.rowCount);
    }
    if (!message.dataSize.isZero()) {
      writer.uint32(56).int64(message.dataSize);
    }
    if (!message.indexSize.isZero()) {
      writer.uint32(64).int64(message.indexSize);
    }
    if (!message.dataFree.isZero()) {
      writer.uint32(72).int64(message.dataFree);
    }
    if (message.createOptions !== "") {
      writer.uint32(82).string(message.createOptions);
    }
    if (message.comment !== "") {
      writer.uint32(90).string(message.comment);
    }
    if (message.classification !== "") {
      writer.uint32(106).string(message.classification);
    }
    if (message.userComment !== "") {
      writer.uint32(114).string(message.userComment);
    }
    for (const v of message.foreignKeys) {
      BranchForeignKeyMetadata.encode(v!, writer.uint32(98).fork()).ldelim();
    }
    for (const v of message.partitions) {
      BranchTablePartitionMetadata.encode(v!, writer.uint32(122).fork()).ldelim();
    }
    if (message.updater !== "") {
      writer.uint32(130).string(message.updater);
    }
    if (message.updateTime !== undefined) {
      Timestamp.encode(toTimestamp(message.updateTime), writer.uint32(138).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BranchTableMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBranchTableMetadata();
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

          message.columns.push(BranchColumnMetadata.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.indexes.push(BranchIndexMetadata.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.engine = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.collation = reader.string();
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.rowCount = reader.int64() as Long;
          continue;
        case 7:
          if (tag !== 56) {
            break;
          }

          message.dataSize = reader.int64() as Long;
          continue;
        case 8:
          if (tag !== 64) {
            break;
          }

          message.indexSize = reader.int64() as Long;
          continue;
        case 9:
          if (tag !== 72) {
            break;
          }

          message.dataFree = reader.int64() as Long;
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.createOptions = reader.string();
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.comment = reader.string();
          continue;
        case 13:
          if (tag !== 106) {
            break;
          }

          message.classification = reader.string();
          continue;
        case 14:
          if (tag !== 114) {
            break;
          }

          message.userComment = reader.string();
          continue;
        case 12:
          if (tag !== 98) {
            break;
          }

          message.foreignKeys.push(BranchForeignKeyMetadata.decode(reader, reader.uint32()));
          continue;
        case 15:
          if (tag !== 122) {
            break;
          }

          message.partitions.push(BranchTablePartitionMetadata.decode(reader, reader.uint32()));
          continue;
        case 16:
          if (tag !== 130) {
            break;
          }

          message.updater = reader.string();
          continue;
        case 17:
          if (tag !== 138) {
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

  fromJSON(object: any): BranchTableMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      columns: globalThis.Array.isArray(object?.columns)
        ? object.columns.map((e: any) => BranchColumnMetadata.fromJSON(e))
        : [],
      indexes: globalThis.Array.isArray(object?.indexes)
        ? object.indexes.map((e: any) => BranchIndexMetadata.fromJSON(e))
        : [],
      engine: isSet(object.engine) ? globalThis.String(object.engine) : "",
      collation: isSet(object.collation) ? globalThis.String(object.collation) : "",
      rowCount: isSet(object.rowCount) ? Long.fromValue(object.rowCount) : Long.ZERO,
      dataSize: isSet(object.dataSize) ? Long.fromValue(object.dataSize) : Long.ZERO,
      indexSize: isSet(object.indexSize) ? Long.fromValue(object.indexSize) : Long.ZERO,
      dataFree: isSet(object.dataFree) ? Long.fromValue(object.dataFree) : Long.ZERO,
      createOptions: isSet(object.createOptions) ? globalThis.String(object.createOptions) : "",
      comment: isSet(object.comment) ? globalThis.String(object.comment) : "",
      classification: isSet(object.classification) ? globalThis.String(object.classification) : "",
      userComment: isSet(object.userComment) ? globalThis.String(object.userComment) : "",
      foreignKeys: globalThis.Array.isArray(object?.foreignKeys)
        ? object.foreignKeys.map((e: any) => BranchForeignKeyMetadata.fromJSON(e))
        : [],
      partitions: globalThis.Array.isArray(object?.partitions)
        ? object.partitions.map((e: any) => BranchTablePartitionMetadata.fromJSON(e))
        : [],
      updater: isSet(object.updater) ? globalThis.String(object.updater) : "",
      updateTime: isSet(object.updateTime) ? fromJsonTimestamp(object.updateTime) : undefined,
    };
  },

  toJSON(message: BranchTableMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.columns?.length) {
      obj.columns = message.columns.map((e) => BranchColumnMetadata.toJSON(e));
    }
    if (message.indexes?.length) {
      obj.indexes = message.indexes.map((e) => BranchIndexMetadata.toJSON(e));
    }
    if (message.engine !== "") {
      obj.engine = message.engine;
    }
    if (message.collation !== "") {
      obj.collation = message.collation;
    }
    if (!message.rowCount.isZero()) {
      obj.rowCount = (message.rowCount || Long.ZERO).toString();
    }
    if (!message.dataSize.isZero()) {
      obj.dataSize = (message.dataSize || Long.ZERO).toString();
    }
    if (!message.indexSize.isZero()) {
      obj.indexSize = (message.indexSize || Long.ZERO).toString();
    }
    if (!message.dataFree.isZero()) {
      obj.dataFree = (message.dataFree || Long.ZERO).toString();
    }
    if (message.createOptions !== "") {
      obj.createOptions = message.createOptions;
    }
    if (message.comment !== "") {
      obj.comment = message.comment;
    }
    if (message.classification !== "") {
      obj.classification = message.classification;
    }
    if (message.userComment !== "") {
      obj.userComment = message.userComment;
    }
    if (message.foreignKeys?.length) {
      obj.foreignKeys = message.foreignKeys.map((e) => BranchForeignKeyMetadata.toJSON(e));
    }
    if (message.partitions?.length) {
      obj.partitions = message.partitions.map((e) => BranchTablePartitionMetadata.toJSON(e));
    }
    if (message.updater !== "") {
      obj.updater = message.updater;
    }
    if (message.updateTime !== undefined) {
      obj.updateTime = message.updateTime.toISOString();
    }
    return obj;
  },

  create(base?: DeepPartial<BranchTableMetadata>): BranchTableMetadata {
    return BranchTableMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BranchTableMetadata>): BranchTableMetadata {
    const message = createBaseBranchTableMetadata();
    message.name = object.name ?? "";
    message.columns = object.columns?.map((e) => BranchColumnMetadata.fromPartial(e)) || [];
    message.indexes = object.indexes?.map((e) => BranchIndexMetadata.fromPartial(e)) || [];
    message.engine = object.engine ?? "";
    message.collation = object.collation ?? "";
    message.rowCount = (object.rowCount !== undefined && object.rowCount !== null)
      ? Long.fromValue(object.rowCount)
      : Long.ZERO;
    message.dataSize = (object.dataSize !== undefined && object.dataSize !== null)
      ? Long.fromValue(object.dataSize)
      : Long.ZERO;
    message.indexSize = (object.indexSize !== undefined && object.indexSize !== null)
      ? Long.fromValue(object.indexSize)
      : Long.ZERO;
    message.dataFree = (object.dataFree !== undefined && object.dataFree !== null)
      ? Long.fromValue(object.dataFree)
      : Long.ZERO;
    message.createOptions = object.createOptions ?? "";
    message.comment = object.comment ?? "";
    message.classification = object.classification ?? "";
    message.userComment = object.userComment ?? "";
    message.foreignKeys = object.foreignKeys?.map((e) => BranchForeignKeyMetadata.fromPartial(e)) || [];
    message.partitions = object.partitions?.map((e) => BranchTablePartitionMetadata.fromPartial(e)) || [];
    message.updater = object.updater ?? "";
    message.updateTime = object.updateTime ?? undefined;
    return message;
  },
};

function createBaseBranchExternalTableMetadata(): BranchExternalTableMetadata {
  return { name: "", externalServerName: "", externalDatabaseName: "", columns: [] };
}

export const BranchExternalTableMetadata = {
  encode(message: BranchExternalTableMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.externalServerName !== "") {
      writer.uint32(18).string(message.externalServerName);
    }
    if (message.externalDatabaseName !== "") {
      writer.uint32(26).string(message.externalDatabaseName);
    }
    for (const v of message.columns) {
      BranchColumnMetadata.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BranchExternalTableMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBranchExternalTableMetadata();
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

          message.externalServerName = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.externalDatabaseName = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.columns.push(BranchColumnMetadata.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BranchExternalTableMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      externalServerName: isSet(object.externalServerName) ? globalThis.String(object.externalServerName) : "",
      externalDatabaseName: isSet(object.externalDatabaseName) ? globalThis.String(object.externalDatabaseName) : "",
      columns: globalThis.Array.isArray(object?.columns)
        ? object.columns.map((e: any) => BranchColumnMetadata.fromJSON(e))
        : [],
    };
  },

  toJSON(message: BranchExternalTableMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.externalServerName !== "") {
      obj.externalServerName = message.externalServerName;
    }
    if (message.externalDatabaseName !== "") {
      obj.externalDatabaseName = message.externalDatabaseName;
    }
    if (message.columns?.length) {
      obj.columns = message.columns.map((e) => BranchColumnMetadata.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<BranchExternalTableMetadata>): BranchExternalTableMetadata {
    return BranchExternalTableMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BranchExternalTableMetadata>): BranchExternalTableMetadata {
    const message = createBaseBranchExternalTableMetadata();
    message.name = object.name ?? "";
    message.externalServerName = object.externalServerName ?? "";
    message.externalDatabaseName = object.externalDatabaseName ?? "";
    message.columns = object.columns?.map((e) => BranchColumnMetadata.fromPartial(e)) || [];
    return message;
  },
};

function createBaseBranchTablePartitionMetadata(): BranchTablePartitionMetadata {
  return {
    name: "",
    type: BranchTablePartitionMetadata_Type.TYPE_UNSPECIFIED,
    expression: "",
    value: "",
    useDefault: "",
    subpartitions: [],
  };
}

export const BranchTablePartitionMetadata = {
  encode(message: BranchTablePartitionMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.type !== BranchTablePartitionMetadata_Type.TYPE_UNSPECIFIED) {
      writer.uint32(16).int32(branchTablePartitionMetadata_TypeToNumber(message.type));
    }
    if (message.expression !== "") {
      writer.uint32(26).string(message.expression);
    }
    if (message.value !== "") {
      writer.uint32(34).string(message.value);
    }
    if (message.useDefault !== "") {
      writer.uint32(42).string(message.useDefault);
    }
    for (const v of message.subpartitions) {
      BranchTablePartitionMetadata.encode(v!, writer.uint32(50).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BranchTablePartitionMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBranchTablePartitionMetadata();
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
          if (tag !== 16) {
            break;
          }

          message.type = branchTablePartitionMetadata_TypeFromJSON(reader.int32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.expression = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.value = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.useDefault = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.subpartitions.push(BranchTablePartitionMetadata.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BranchTablePartitionMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      type: isSet(object.type)
        ? branchTablePartitionMetadata_TypeFromJSON(object.type)
        : BranchTablePartitionMetadata_Type.TYPE_UNSPECIFIED,
      expression: isSet(object.expression) ? globalThis.String(object.expression) : "",
      value: isSet(object.value) ? globalThis.String(object.value) : "",
      useDefault: isSet(object.useDefault) ? globalThis.String(object.useDefault) : "",
      subpartitions: globalThis.Array.isArray(object?.subpartitions)
        ? object.subpartitions.map((e: any) => BranchTablePartitionMetadata.fromJSON(e))
        : [],
    };
  },

  toJSON(message: BranchTablePartitionMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.type !== BranchTablePartitionMetadata_Type.TYPE_UNSPECIFIED) {
      obj.type = branchTablePartitionMetadata_TypeToJSON(message.type);
    }
    if (message.expression !== "") {
      obj.expression = message.expression;
    }
    if (message.value !== "") {
      obj.value = message.value;
    }
    if (message.useDefault !== "") {
      obj.useDefault = message.useDefault;
    }
    if (message.subpartitions?.length) {
      obj.subpartitions = message.subpartitions.map((e) => BranchTablePartitionMetadata.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<BranchTablePartitionMetadata>): BranchTablePartitionMetadata {
    return BranchTablePartitionMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BranchTablePartitionMetadata>): BranchTablePartitionMetadata {
    const message = createBaseBranchTablePartitionMetadata();
    message.name = object.name ?? "";
    message.type = object.type ?? BranchTablePartitionMetadata_Type.TYPE_UNSPECIFIED;
    message.expression = object.expression ?? "";
    message.value = object.value ?? "";
    message.useDefault = object.useDefault ?? "";
    message.subpartitions = object.subpartitions?.map((e) => BranchTablePartitionMetadata.fromPartial(e)) || [];
    return message;
  },
};

function createBaseBranchColumnMetadata(): BranchColumnMetadata {
  return {
    name: "",
    position: 0,
    default: undefined,
    defaultNull: undefined,
    defaultExpression: undefined,
    onUpdate: "",
    nullable: false,
    type: "",
    characterSet: "",
    collation: "",
    comment: "",
    classification: "",
    userComment: "",
  };
}

export const BranchColumnMetadata = {
  encode(message: BranchColumnMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.position !== 0) {
      writer.uint32(16).int32(message.position);
    }
    if (message.default !== undefined) {
      StringValue.encode({ value: message.default! }, writer.uint32(26).fork()).ldelim();
    }
    if (message.defaultNull !== undefined) {
      writer.uint32(32).bool(message.defaultNull);
    }
    if (message.defaultExpression !== undefined) {
      writer.uint32(42).string(message.defaultExpression);
    }
    if (message.onUpdate !== "") {
      writer.uint32(106).string(message.onUpdate);
    }
    if (message.nullable === true) {
      writer.uint32(48).bool(message.nullable);
    }
    if (message.type !== "") {
      writer.uint32(58).string(message.type);
    }
    if (message.characterSet !== "") {
      writer.uint32(66).string(message.characterSet);
    }
    if (message.collation !== "") {
      writer.uint32(74).string(message.collation);
    }
    if (message.comment !== "") {
      writer.uint32(82).string(message.comment);
    }
    if (message.classification !== "") {
      writer.uint32(90).string(message.classification);
    }
    if (message.userComment !== "") {
      writer.uint32(98).string(message.userComment);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BranchColumnMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBranchColumnMetadata();
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
          if (tag !== 16) {
            break;
          }

          message.position = reader.int32();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.default = StringValue.decode(reader, reader.uint32()).value;
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.defaultNull = reader.bool();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.defaultExpression = reader.string();
          continue;
        case 13:
          if (tag !== 106) {
            break;
          }

          message.onUpdate = reader.string();
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.nullable = reader.bool();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.type = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.characterSet = reader.string();
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.collation = reader.string();
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.comment = reader.string();
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.classification = reader.string();
          continue;
        case 12:
          if (tag !== 98) {
            break;
          }

          message.userComment = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BranchColumnMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      position: isSet(object.position) ? globalThis.Number(object.position) : 0,
      default: isSet(object.default) ? String(object.default) : undefined,
      defaultNull: isSet(object.defaultNull) ? globalThis.Boolean(object.defaultNull) : undefined,
      defaultExpression: isSet(object.defaultExpression) ? globalThis.String(object.defaultExpression) : undefined,
      onUpdate: isSet(object.onUpdate) ? globalThis.String(object.onUpdate) : "",
      nullable: isSet(object.nullable) ? globalThis.Boolean(object.nullable) : false,
      type: isSet(object.type) ? globalThis.String(object.type) : "",
      characterSet: isSet(object.characterSet) ? globalThis.String(object.characterSet) : "",
      collation: isSet(object.collation) ? globalThis.String(object.collation) : "",
      comment: isSet(object.comment) ? globalThis.String(object.comment) : "",
      classification: isSet(object.classification) ? globalThis.String(object.classification) : "",
      userComment: isSet(object.userComment) ? globalThis.String(object.userComment) : "",
    };
  },

  toJSON(message: BranchColumnMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.position !== 0) {
      obj.position = Math.round(message.position);
    }
    if (message.default !== undefined) {
      obj.default = message.default;
    }
    if (message.defaultNull !== undefined) {
      obj.defaultNull = message.defaultNull;
    }
    if (message.defaultExpression !== undefined) {
      obj.defaultExpression = message.defaultExpression;
    }
    if (message.onUpdate !== "") {
      obj.onUpdate = message.onUpdate;
    }
    if (message.nullable === true) {
      obj.nullable = message.nullable;
    }
    if (message.type !== "") {
      obj.type = message.type;
    }
    if (message.characterSet !== "") {
      obj.characterSet = message.characterSet;
    }
    if (message.collation !== "") {
      obj.collation = message.collation;
    }
    if (message.comment !== "") {
      obj.comment = message.comment;
    }
    if (message.classification !== "") {
      obj.classification = message.classification;
    }
    if (message.userComment !== "") {
      obj.userComment = message.userComment;
    }
    return obj;
  },

  create(base?: DeepPartial<BranchColumnMetadata>): BranchColumnMetadata {
    return BranchColumnMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BranchColumnMetadata>): BranchColumnMetadata {
    const message = createBaseBranchColumnMetadata();
    message.name = object.name ?? "";
    message.position = object.position ?? 0;
    message.default = object.default ?? undefined;
    message.defaultNull = object.defaultNull ?? undefined;
    message.defaultExpression = object.defaultExpression ?? undefined;
    message.onUpdate = object.onUpdate ?? "";
    message.nullable = object.nullable ?? false;
    message.type = object.type ?? "";
    message.characterSet = object.characterSet ?? "";
    message.collation = object.collation ?? "";
    message.comment = object.comment ?? "";
    message.classification = object.classification ?? "";
    message.userComment = object.userComment ?? "";
    return message;
  },
};

function createBaseBranchViewMetadata(): BranchViewMetadata {
  return { name: "", definition: "", comment: "", dependentColumns: [], updater: "", updateTime: undefined };
}

export const BranchViewMetadata = {
  encode(message: BranchViewMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.definition !== "") {
      writer.uint32(18).string(message.definition);
    }
    if (message.comment !== "") {
      writer.uint32(26).string(message.comment);
    }
    for (const v of message.dependentColumns) {
      BranchDependentColumn.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    if (message.updater !== "") {
      writer.uint32(42).string(message.updater);
    }
    if (message.updateTime !== undefined) {
      Timestamp.encode(toTimestamp(message.updateTime), writer.uint32(50).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BranchViewMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBranchViewMetadata();
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

          message.definition = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.comment = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.dependentColumns.push(BranchDependentColumn.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.updater = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
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

  fromJSON(object: any): BranchViewMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      definition: isSet(object.definition) ? globalThis.String(object.definition) : "",
      comment: isSet(object.comment) ? globalThis.String(object.comment) : "",
      dependentColumns: globalThis.Array.isArray(object?.dependentColumns)
        ? object.dependentColumns.map((e: any) => BranchDependentColumn.fromJSON(e))
        : [],
      updater: isSet(object.updater) ? globalThis.String(object.updater) : "",
      updateTime: isSet(object.updateTime) ? fromJsonTimestamp(object.updateTime) : undefined,
    };
  },

  toJSON(message: BranchViewMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.definition !== "") {
      obj.definition = message.definition;
    }
    if (message.comment !== "") {
      obj.comment = message.comment;
    }
    if (message.dependentColumns?.length) {
      obj.dependentColumns = message.dependentColumns.map((e) => BranchDependentColumn.toJSON(e));
    }
    if (message.updater !== "") {
      obj.updater = message.updater;
    }
    if (message.updateTime !== undefined) {
      obj.updateTime = message.updateTime.toISOString();
    }
    return obj;
  },

  create(base?: DeepPartial<BranchViewMetadata>): BranchViewMetadata {
    return BranchViewMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BranchViewMetadata>): BranchViewMetadata {
    const message = createBaseBranchViewMetadata();
    message.name = object.name ?? "";
    message.definition = object.definition ?? "";
    message.comment = object.comment ?? "";
    message.dependentColumns = object.dependentColumns?.map((e) => BranchDependentColumn.fromPartial(e)) || [];
    message.updater = object.updater ?? "";
    message.updateTime = object.updateTime ?? undefined;
    return message;
  },
};

function createBaseBranchDependentColumn(): BranchDependentColumn {
  return { schema: "", table: "", column: "" };
}

export const BranchDependentColumn = {
  encode(message: BranchDependentColumn, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schema !== "") {
      writer.uint32(10).string(message.schema);
    }
    if (message.table !== "") {
      writer.uint32(18).string(message.table);
    }
    if (message.column !== "") {
      writer.uint32(26).string(message.column);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BranchDependentColumn {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBranchDependentColumn();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.schema = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.table = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.column = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BranchDependentColumn {
    return {
      schema: isSet(object.schema) ? globalThis.String(object.schema) : "",
      table: isSet(object.table) ? globalThis.String(object.table) : "",
      column: isSet(object.column) ? globalThis.String(object.column) : "",
    };
  },

  toJSON(message: BranchDependentColumn): unknown {
    const obj: any = {};
    if (message.schema !== "") {
      obj.schema = message.schema;
    }
    if (message.table !== "") {
      obj.table = message.table;
    }
    if (message.column !== "") {
      obj.column = message.column;
    }
    return obj;
  },

  create(base?: DeepPartial<BranchDependentColumn>): BranchDependentColumn {
    return BranchDependentColumn.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BranchDependentColumn>): BranchDependentColumn {
    const message = createBaseBranchDependentColumn();
    message.schema = object.schema ?? "";
    message.table = object.table ?? "";
    message.column = object.column ?? "";
    return message;
  },
};

function createBaseBranchMaterializedViewMetadata(): BranchMaterializedViewMetadata {
  return { name: "", definition: "", comment: "", dependentColumns: [] };
}

export const BranchMaterializedViewMetadata = {
  encode(message: BranchMaterializedViewMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.definition !== "") {
      writer.uint32(18).string(message.definition);
    }
    if (message.comment !== "") {
      writer.uint32(26).string(message.comment);
    }
    for (const v of message.dependentColumns) {
      BranchDependentColumn.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BranchMaterializedViewMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBranchMaterializedViewMetadata();
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

          message.definition = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.comment = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.dependentColumns.push(BranchDependentColumn.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BranchMaterializedViewMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      definition: isSet(object.definition) ? globalThis.String(object.definition) : "",
      comment: isSet(object.comment) ? globalThis.String(object.comment) : "",
      dependentColumns: globalThis.Array.isArray(object?.dependentColumns)
        ? object.dependentColumns.map((e: any) => BranchDependentColumn.fromJSON(e))
        : [],
    };
  },

  toJSON(message: BranchMaterializedViewMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.definition !== "") {
      obj.definition = message.definition;
    }
    if (message.comment !== "") {
      obj.comment = message.comment;
    }
    if (message.dependentColumns?.length) {
      obj.dependentColumns = message.dependentColumns.map((e) => BranchDependentColumn.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<BranchMaterializedViewMetadata>): BranchMaterializedViewMetadata {
    return BranchMaterializedViewMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BranchMaterializedViewMetadata>): BranchMaterializedViewMetadata {
    const message = createBaseBranchMaterializedViewMetadata();
    message.name = object.name ?? "";
    message.definition = object.definition ?? "";
    message.comment = object.comment ?? "";
    message.dependentColumns = object.dependentColumns?.map((e) => BranchDependentColumn.fromPartial(e)) || [];
    return message;
  },
};

function createBaseBranchFunctionMetadata(): BranchFunctionMetadata {
  return { name: "", definition: "", updater: "", updateTime: undefined };
}

export const BranchFunctionMetadata = {
  encode(message: BranchFunctionMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.definition !== "") {
      writer.uint32(18).string(message.definition);
    }
    if (message.updater !== "") {
      writer.uint32(26).string(message.updater);
    }
    if (message.updateTime !== undefined) {
      Timestamp.encode(toTimestamp(message.updateTime), writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BranchFunctionMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBranchFunctionMetadata();
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

          message.definition = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.updater = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
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

  fromJSON(object: any): BranchFunctionMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      definition: isSet(object.definition) ? globalThis.String(object.definition) : "",
      updater: isSet(object.updater) ? globalThis.String(object.updater) : "",
      updateTime: isSet(object.updateTime) ? fromJsonTimestamp(object.updateTime) : undefined,
    };
  },

  toJSON(message: BranchFunctionMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.definition !== "") {
      obj.definition = message.definition;
    }
    if (message.updater !== "") {
      obj.updater = message.updater;
    }
    if (message.updateTime !== undefined) {
      obj.updateTime = message.updateTime.toISOString();
    }
    return obj;
  },

  create(base?: DeepPartial<BranchFunctionMetadata>): BranchFunctionMetadata {
    return BranchFunctionMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BranchFunctionMetadata>): BranchFunctionMetadata {
    const message = createBaseBranchFunctionMetadata();
    message.name = object.name ?? "";
    message.definition = object.definition ?? "";
    message.updater = object.updater ?? "";
    message.updateTime = object.updateTime ?? undefined;
    return message;
  },
};

function createBaseBranchProcedureMetadata(): BranchProcedureMetadata {
  return { name: "", definition: "", updater: "", updateTime: undefined };
}

export const BranchProcedureMetadata = {
  encode(message: BranchProcedureMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.definition !== "") {
      writer.uint32(18).string(message.definition);
    }
    if (message.updater !== "") {
      writer.uint32(26).string(message.updater);
    }
    if (message.updateTime !== undefined) {
      Timestamp.encode(toTimestamp(message.updateTime), writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BranchProcedureMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBranchProcedureMetadata();
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

          message.definition = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.updater = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
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

  fromJSON(object: any): BranchProcedureMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      definition: isSet(object.definition) ? globalThis.String(object.definition) : "",
      updater: isSet(object.updater) ? globalThis.String(object.updater) : "",
      updateTime: isSet(object.updateTime) ? fromJsonTimestamp(object.updateTime) : undefined,
    };
  },

  toJSON(message: BranchProcedureMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.definition !== "") {
      obj.definition = message.definition;
    }
    if (message.updater !== "") {
      obj.updater = message.updater;
    }
    if (message.updateTime !== undefined) {
      obj.updateTime = message.updateTime.toISOString();
    }
    return obj;
  },

  create(base?: DeepPartial<BranchProcedureMetadata>): BranchProcedureMetadata {
    return BranchProcedureMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BranchProcedureMetadata>): BranchProcedureMetadata {
    const message = createBaseBranchProcedureMetadata();
    message.name = object.name ?? "";
    message.definition = object.definition ?? "";
    message.updater = object.updater ?? "";
    message.updateTime = object.updateTime ?? undefined;
    return message;
  },
};

function createBaseBranchIndexMetadata(): BranchIndexMetadata {
  return {
    name: "",
    expressions: [],
    keyLength: [],
    type: "",
    unique: false,
    primary: false,
    visible: false,
    comment: "",
    definition: "",
  };
}

export const BranchIndexMetadata = {
  encode(message: BranchIndexMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.expressions) {
      writer.uint32(18).string(v!);
    }
    writer.uint32(74).fork();
    for (const v of message.keyLength) {
      writer.int64(v);
    }
    writer.ldelim();
    if (message.type !== "") {
      writer.uint32(26).string(message.type);
    }
    if (message.unique === true) {
      writer.uint32(32).bool(message.unique);
    }
    if (message.primary === true) {
      writer.uint32(40).bool(message.primary);
    }
    if (message.visible === true) {
      writer.uint32(48).bool(message.visible);
    }
    if (message.comment !== "") {
      writer.uint32(58).string(message.comment);
    }
    if (message.definition !== "") {
      writer.uint32(66).string(message.definition);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BranchIndexMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBranchIndexMetadata();
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

          message.expressions.push(reader.string());
          continue;
        case 9:
          if (tag === 72) {
            message.keyLength.push(reader.int64() as Long);

            continue;
          }

          if (tag === 74) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.keyLength.push(reader.int64() as Long);
            }

            continue;
          }

          break;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.type = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.unique = reader.bool();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.primary = reader.bool();
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.visible = reader.bool();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.comment = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.definition = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BranchIndexMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      expressions: globalThis.Array.isArray(object?.expressions)
        ? object.expressions.map((e: any) => globalThis.String(e))
        : [],
      keyLength: globalThis.Array.isArray(object?.keyLength) ? object.keyLength.map((e: any) => Long.fromValue(e)) : [],
      type: isSet(object.type) ? globalThis.String(object.type) : "",
      unique: isSet(object.unique) ? globalThis.Boolean(object.unique) : false,
      primary: isSet(object.primary) ? globalThis.Boolean(object.primary) : false,
      visible: isSet(object.visible) ? globalThis.Boolean(object.visible) : false,
      comment: isSet(object.comment) ? globalThis.String(object.comment) : "",
      definition: isSet(object.definition) ? globalThis.String(object.definition) : "",
    };
  },

  toJSON(message: BranchIndexMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.expressions?.length) {
      obj.expressions = message.expressions;
    }
    if (message.keyLength?.length) {
      obj.keyLength = message.keyLength.map((e) => (e || Long.ZERO).toString());
    }
    if (message.type !== "") {
      obj.type = message.type;
    }
    if (message.unique === true) {
      obj.unique = message.unique;
    }
    if (message.primary === true) {
      obj.primary = message.primary;
    }
    if (message.visible === true) {
      obj.visible = message.visible;
    }
    if (message.comment !== "") {
      obj.comment = message.comment;
    }
    if (message.definition !== "") {
      obj.definition = message.definition;
    }
    return obj;
  },

  create(base?: DeepPartial<BranchIndexMetadata>): BranchIndexMetadata {
    return BranchIndexMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BranchIndexMetadata>): BranchIndexMetadata {
    const message = createBaseBranchIndexMetadata();
    message.name = object.name ?? "";
    message.expressions = object.expressions?.map((e) => e) || [];
    message.keyLength = object.keyLength?.map((e) => Long.fromValue(e)) || [];
    message.type = object.type ?? "";
    message.unique = object.unique ?? false;
    message.primary = object.primary ?? false;
    message.visible = object.visible ?? false;
    message.comment = object.comment ?? "";
    message.definition = object.definition ?? "";
    return message;
  },
};

function createBaseBranchExtensionMetadata(): BranchExtensionMetadata {
  return { name: "", schema: "", version: "", description: "" };
}

export const BranchExtensionMetadata = {
  encode(message: BranchExtensionMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.schema !== "") {
      writer.uint32(18).string(message.schema);
    }
    if (message.version !== "") {
      writer.uint32(26).string(message.version);
    }
    if (message.description !== "") {
      writer.uint32(34).string(message.description);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BranchExtensionMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBranchExtensionMetadata();
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

          message.schema = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.version = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.description = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BranchExtensionMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      schema: isSet(object.schema) ? globalThis.String(object.schema) : "",
      version: isSet(object.version) ? globalThis.String(object.version) : "",
      description: isSet(object.description) ? globalThis.String(object.description) : "",
    };
  },

  toJSON(message: BranchExtensionMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.schema !== "") {
      obj.schema = message.schema;
    }
    if (message.version !== "") {
      obj.version = message.version;
    }
    if (message.description !== "") {
      obj.description = message.description;
    }
    return obj;
  },

  create(base?: DeepPartial<BranchExtensionMetadata>): BranchExtensionMetadata {
    return BranchExtensionMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BranchExtensionMetadata>): BranchExtensionMetadata {
    const message = createBaseBranchExtensionMetadata();
    message.name = object.name ?? "";
    message.schema = object.schema ?? "";
    message.version = object.version ?? "";
    message.description = object.description ?? "";
    return message;
  },
};

function createBaseBranchForeignKeyMetadata(): BranchForeignKeyMetadata {
  return {
    name: "",
    columns: [],
    referencedSchema: "",
    referencedTable: "",
    referencedColumns: [],
    onDelete: "",
    onUpdate: "",
    matchType: "",
  };
}

export const BranchForeignKeyMetadata = {
  encode(message: BranchForeignKeyMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.columns) {
      writer.uint32(18).string(v!);
    }
    if (message.referencedSchema !== "") {
      writer.uint32(26).string(message.referencedSchema);
    }
    if (message.referencedTable !== "") {
      writer.uint32(34).string(message.referencedTable);
    }
    for (const v of message.referencedColumns) {
      writer.uint32(42).string(v!);
    }
    if (message.onDelete !== "") {
      writer.uint32(50).string(message.onDelete);
    }
    if (message.onUpdate !== "") {
      writer.uint32(58).string(message.onUpdate);
    }
    if (message.matchType !== "") {
      writer.uint32(66).string(message.matchType);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BranchForeignKeyMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBranchForeignKeyMetadata();
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

          message.columns.push(reader.string());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.referencedSchema = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.referencedTable = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.referencedColumns.push(reader.string());
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.onDelete = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.onUpdate = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.matchType = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BranchForeignKeyMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      columns: globalThis.Array.isArray(object?.columns) ? object.columns.map((e: any) => globalThis.String(e)) : [],
      referencedSchema: isSet(object.referencedSchema) ? globalThis.String(object.referencedSchema) : "",
      referencedTable: isSet(object.referencedTable) ? globalThis.String(object.referencedTable) : "",
      referencedColumns: globalThis.Array.isArray(object?.referencedColumns)
        ? object.referencedColumns.map((e: any) => globalThis.String(e))
        : [],
      onDelete: isSet(object.onDelete) ? globalThis.String(object.onDelete) : "",
      onUpdate: isSet(object.onUpdate) ? globalThis.String(object.onUpdate) : "",
      matchType: isSet(object.matchType) ? globalThis.String(object.matchType) : "",
    };
  },

  toJSON(message: BranchForeignKeyMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.columns?.length) {
      obj.columns = message.columns;
    }
    if (message.referencedSchema !== "") {
      obj.referencedSchema = message.referencedSchema;
    }
    if (message.referencedTable !== "") {
      obj.referencedTable = message.referencedTable;
    }
    if (message.referencedColumns?.length) {
      obj.referencedColumns = message.referencedColumns;
    }
    if (message.onDelete !== "") {
      obj.onDelete = message.onDelete;
    }
    if (message.onUpdate !== "") {
      obj.onUpdate = message.onUpdate;
    }
    if (message.matchType !== "") {
      obj.matchType = message.matchType;
    }
    return obj;
  },

  create(base?: DeepPartial<BranchForeignKeyMetadata>): BranchForeignKeyMetadata {
    return BranchForeignKeyMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BranchForeignKeyMetadata>): BranchForeignKeyMetadata {
    const message = createBaseBranchForeignKeyMetadata();
    message.name = object.name ?? "";
    message.columns = object.columns?.map((e) => e) || [];
    message.referencedSchema = object.referencedSchema ?? "";
    message.referencedTable = object.referencedTable ?? "";
    message.referencedColumns = object.referencedColumns?.map((e) => e) || [];
    message.onDelete = object.onDelete ?? "";
    message.onUpdate = object.onUpdate ?? "";
    message.matchType = object.matchType ?? "";
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

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
