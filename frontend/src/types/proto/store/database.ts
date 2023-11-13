/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Timestamp } from "../google/protobuf/timestamp";
import { StringValue } from "../google/protobuf/wrappers";

export const protobufPackage = "bytebase.store";

/** DatabaseMetadata is the metadata for databases. */
export interface DatabaseMetadata {
  labels: { [key: string]: string };
  lastSyncTime: Date | undefined;
}

export interface DatabaseMetadata_LabelsEntry {
  key: string;
  value: string;
}

/** DatabaseSchemaMetadata is the schema metadata for databases. */
export interface DatabaseSchemaMetadata {
  name: string;
  /** The schemas is the list of schemas in a database. */
  schemas: SchemaMetadata[];
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
 * SchemaMetadata is the metadata for schemas.
 * This is the concept of schema in Postgres, but it's a no-op for MySQL.
 */
export interface SchemaMetadata {
  /**
   * The name is the schema name.
   * It is an empty string for databases without such concept such as MySQL.
   */
  name: string;
  /** The tables is the list of tables in a schema. */
  tables: TableMetadata[];
  /** The views is the list of views in a schema. */
  views: ViewMetadata[];
  /** The functions is the list of functions in a schema. */
  functions: FunctionMetadata[];
  /** The streams is the list of streams in a schema, currently, only used for Snowflake. */
  streams: StreamMetadata[];
  /** The routines is the list of routines in a schema, currently, only used for Snowflake. */
  tasks: TaskMetadata[];
}

export interface TaskMetadata {
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
  state: TaskMetadata_State;
  /** The condition of the task. */
  condition: string;
  /** The definition of the task. */
  definition: string;
}

export enum TaskMetadata_State {
  STATE_UNSPECIFIED = 0,
  STATE_STARTED = 1,
  STATE_SUSPENDED = 2,
  UNRECOGNIZED = -1,
}

export function taskMetadata_StateFromJSON(object: any): TaskMetadata_State {
  switch (object) {
    case 0:
    case "STATE_UNSPECIFIED":
      return TaskMetadata_State.STATE_UNSPECIFIED;
    case 1:
    case "STATE_STARTED":
      return TaskMetadata_State.STATE_STARTED;
    case 2:
    case "STATE_SUSPENDED":
      return TaskMetadata_State.STATE_SUSPENDED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return TaskMetadata_State.UNRECOGNIZED;
  }
}

export function taskMetadata_StateToJSON(object: TaskMetadata_State): string {
  switch (object) {
    case TaskMetadata_State.STATE_UNSPECIFIED:
      return "STATE_UNSPECIFIED";
    case TaskMetadata_State.STATE_STARTED:
      return "STATE_STARTED";
    case TaskMetadata_State.STATE_SUSPENDED:
      return "STATE_SUSPENDED";
    case TaskMetadata_State.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface StreamMetadata {
  /** The name is the name of a stream. */
  name: string;
  /** The table_name is the name of the table/view that the stream is created on. */
  tableName: string;
  /** The owner of the stream. */
  owner: string;
  /** The comment of the stream. */
  comment: string;
  /** The type of the stream. */
  type: StreamMetadata_Type;
  /** Indicates whether the stream was last read before the `stale_after` time. */
  stale: boolean;
  /** The mode of the stream. */
  mode: StreamMetadata_Mode;
  /** The definition of the stream. */
  definition: string;
}

export enum StreamMetadata_Type {
  TYPE_UNSPECIFIED = 0,
  TYPE_DELTA = 1,
  UNRECOGNIZED = -1,
}

export function streamMetadata_TypeFromJSON(object: any): StreamMetadata_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return StreamMetadata_Type.TYPE_UNSPECIFIED;
    case 1:
    case "TYPE_DELTA":
      return StreamMetadata_Type.TYPE_DELTA;
    case -1:
    case "UNRECOGNIZED":
    default:
      return StreamMetadata_Type.UNRECOGNIZED;
  }
}

export function streamMetadata_TypeToJSON(object: StreamMetadata_Type): string {
  switch (object) {
    case StreamMetadata_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case StreamMetadata_Type.TYPE_DELTA:
      return "TYPE_DELTA";
    case StreamMetadata_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum StreamMetadata_Mode {
  MODE_UNSPECIFIED = 0,
  MODE_DEFAULT = 1,
  MODE_APPEND_ONLY = 2,
  MODE_INSERT_ONLY = 3,
  UNRECOGNIZED = -1,
}

export function streamMetadata_ModeFromJSON(object: any): StreamMetadata_Mode {
  switch (object) {
    case 0:
    case "MODE_UNSPECIFIED":
      return StreamMetadata_Mode.MODE_UNSPECIFIED;
    case 1:
    case "MODE_DEFAULT":
      return StreamMetadata_Mode.MODE_DEFAULT;
    case 2:
    case "MODE_APPEND_ONLY":
      return StreamMetadata_Mode.MODE_APPEND_ONLY;
    case 3:
    case "MODE_INSERT_ONLY":
      return StreamMetadata_Mode.MODE_INSERT_ONLY;
    case -1:
    case "UNRECOGNIZED":
    default:
      return StreamMetadata_Mode.UNRECOGNIZED;
  }
}

export function streamMetadata_ModeToJSON(object: StreamMetadata_Mode): string {
  switch (object) {
    case StreamMetadata_Mode.MODE_UNSPECIFIED:
      return "MODE_UNSPECIFIED";
    case StreamMetadata_Mode.MODE_DEFAULT:
      return "MODE_DEFAULT";
    case StreamMetadata_Mode.MODE_APPEND_ONLY:
      return "MODE_APPEND_ONLY";
    case StreamMetadata_Mode.MODE_INSERT_ONLY:
      return "MODE_INSERT_ONLY";
    case StreamMetadata_Mode.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

/** TableMetadata is the metadata for tables. */
export interface TableMetadata {
  /** The name is the name of a table. */
  name: string;
  /** The columns is the ordered list of columns in a table. */
  columns: ColumnMetadata[];
  /** The indexes is the list of indexes in a table. */
  indexes: IndexMetadata[];
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
  foreignKeys: ForeignKeyMetadata[];
  /** The partitions is the list of partitions in a table. */
  partitions: TablePartitionMetadata[];
}

/** TablePartitionMetadata is the metadata for table partitions. */
export interface TablePartitionMetadata {
  /** The name is the name of a table partition. */
  name: string;
  /** The type of a table partition. */
  type: TablePartitionMetadata_Type;
  /** The expression is the expression of a table partition. */
  expression: string;
  /** The subpartitions is the list of subpartitions in a table partition. */
  subpartitions: TablePartitionMetadata[];
}

export enum TablePartitionMetadata_Type {
  TYPE_UNSPECIFIED = 0,
  RANGE = 1,
  LIST = 2,
  HASH = 3,
  UNRECOGNIZED = -1,
}

export function tablePartitionMetadata_TypeFromJSON(object: any): TablePartitionMetadata_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return TablePartitionMetadata_Type.TYPE_UNSPECIFIED;
    case 1:
    case "RANGE":
      return TablePartitionMetadata_Type.RANGE;
    case 2:
    case "LIST":
      return TablePartitionMetadata_Type.LIST;
    case 3:
    case "HASH":
      return TablePartitionMetadata_Type.HASH;
    case -1:
    case "UNRECOGNIZED":
    default:
      return TablePartitionMetadata_Type.UNRECOGNIZED;
  }
}

export function tablePartitionMetadata_TypeToJSON(object: TablePartitionMetadata_Type): string {
  switch (object) {
    case TablePartitionMetadata_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case TablePartitionMetadata_Type.RANGE:
      return "RANGE";
    case TablePartitionMetadata_Type.LIST:
      return "LIST";
    case TablePartitionMetadata_Type.HASH:
      return "HASH";
    case TablePartitionMetadata_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

/** ColumnMetadata is the metadata for columns. */
export interface ColumnMetadata {
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

/** ViewMetadata is the metadata for views. */
export interface ViewMetadata {
  /** The name is the name of a view. */
  name: string;
  /** The definition is the definition of a view. */
  definition: string;
  /** The comment is the comment of a view. */
  comment: string;
  /** The dependent_columns is the list of dependent columns of a view. */
  dependentColumns: DependentColumn[];
}

/** DependentColumn is the metadata for dependent columns. */
export interface DependentColumn {
  /** The schema is the schema of a reference column. */
  schema: string;
  /** The table is the table of a reference column. */
  table: string;
  /** The column is the name of a reference column. */
  column: string;
}

/** FunctionMetadata is the metadata for functions. */
export interface FunctionMetadata {
  /** The name is the name of a view. */
  name: string;
  /** The definition is the definition of a view. */
  definition: string;
}

/** IndexMetadata is the metadata for indexes. */
export interface IndexMetadata {
  /** The name is the name of an index. */
  name: string;
  /**
   * The expressions are the ordered columns or expressions of an index.
   * This could refer to a column or an expression.
   */
  expressions: string[];
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
}

/** ExtensionMetadata is the metadata for extensions. */
export interface ExtensionMetadata {
  /** The name is the name of an extension. */
  name: string;
  /** The schema is the extension that is installed to. But the extension usage is not limited to the schema. */
  schema: string;
  /** The version is the version of an extension. */
  version: string;
  /** The description is the description of an extension. */
  description: string;
}

/** ForeignKeyMetadata is the metadata for foreign keys. */
export interface ForeignKeyMetadata {
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

/** InstanceRoleMetadata is the message for instance role. */
export interface InstanceRoleMetadata {
  /** The role name. It's unique within the instance. */
  name: string;
  /** The grant display string on the instance. It's generated by database engine. */
  grant: string;
}

export interface Secrets {
  /** The list of secrets. */
  items: SecretItem[];
}

export interface SecretItem {
  /** The name is the name of the secret. */
  name: string;
  /** The value is the value of the secret. */
  value: string;
  /** The description is the description of the secret. */
  description: string;
}

export interface DatabaseConfig {
  name: string;
  /** The schema_configs is the list of configs for schemas in a database. */
  schemaConfigs: SchemaConfig[];
}

export interface SchemaConfig {
  /**
   * The name is the schema name.
   * It is an empty string for databases without such concept such as MySQL.
   */
  name: string;
  /** The table_configs is the list of configs for tables in a schema. */
  tableConfigs: TableConfig[];
}

export interface TableConfig {
  /** The name is the name of a table. */
  name: string;
  /** The column_configs is the ordered list of configs for columns in a table. */
  columnConfigs: ColumnConfig[];
}

export interface ColumnConfig {
  /** The name is the name of a column. */
  name: string;
  semanticTypeId: string;
  /** The user labels for a column. */
  labels: { [key: string]: string };
}

export interface ColumnConfig_LabelsEntry {
  key: string;
  value: string;
}

function createBaseDatabaseMetadata(): DatabaseMetadata {
  return { labels: {}, lastSyncTime: undefined };
}

export const DatabaseMetadata = {
  encode(message: DatabaseMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    Object.entries(message.labels).forEach(([key, value]) => {
      DatabaseMetadata_LabelsEntry.encode({ key: key as any, value }, writer.uint32(10).fork()).ldelim();
    });
    if (message.lastSyncTime !== undefined) {
      Timestamp.encode(toTimestamp(message.lastSyncTime), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DatabaseMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDatabaseMetadata();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          const entry1 = DatabaseMetadata_LabelsEntry.decode(reader, reader.uint32());
          if (entry1.value !== undefined) {
            message.labels[entry1.key] = entry1.value;
          }
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.lastSyncTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DatabaseMetadata {
    return {
      labels: isObject(object.labels)
        ? Object.entries(object.labels).reduce<{ [key: string]: string }>((acc, [key, value]) => {
          acc[key] = String(value);
          return acc;
        }, {})
        : {},
      lastSyncTime: isSet(object.lastSyncTime) ? fromJsonTimestamp(object.lastSyncTime) : undefined,
    };
  },

  toJSON(message: DatabaseMetadata): unknown {
    const obj: any = {};
    if (message.labels) {
      const entries = Object.entries(message.labels);
      if (entries.length > 0) {
        obj.labels = {};
        entries.forEach(([k, v]) => {
          obj.labels[k] = v;
        });
      }
    }
    if (message.lastSyncTime !== undefined) {
      obj.lastSyncTime = message.lastSyncTime.toISOString();
    }
    return obj;
  },

  create(base?: DeepPartial<DatabaseMetadata>): DatabaseMetadata {
    return DatabaseMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DatabaseMetadata>): DatabaseMetadata {
    const message = createBaseDatabaseMetadata();
    message.labels = Object.entries(object.labels ?? {}).reduce<{ [key: string]: string }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = globalThis.String(value);
      }
      return acc;
    }, {});
    message.lastSyncTime = object.lastSyncTime ?? undefined;
    return message;
  },
};

function createBaseDatabaseMetadata_LabelsEntry(): DatabaseMetadata_LabelsEntry {
  return { key: "", value: "" };
}

export const DatabaseMetadata_LabelsEntry = {
  encode(message: DatabaseMetadata_LabelsEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== "") {
      writer.uint32(18).string(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DatabaseMetadata_LabelsEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDatabaseMetadata_LabelsEntry();
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

  fromJSON(object: any): DatabaseMetadata_LabelsEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? globalThis.String(object.value) : "",
    };
  },

  toJSON(message: DatabaseMetadata_LabelsEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== "") {
      obj.value = message.value;
    }
    return obj;
  },

  create(base?: DeepPartial<DatabaseMetadata_LabelsEntry>): DatabaseMetadata_LabelsEntry {
    return DatabaseMetadata_LabelsEntry.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DatabaseMetadata_LabelsEntry>): DatabaseMetadata_LabelsEntry {
    const message = createBaseDatabaseMetadata_LabelsEntry();
    message.key = object.key ?? "";
    message.value = object.value ?? "";
    return message;
  },
};

function createBaseDatabaseSchemaMetadata(): DatabaseSchemaMetadata {
  return { name: "", schemas: [], characterSet: "", collation: "", extensions: [], datashare: false, serviceName: "" };
}

export const DatabaseSchemaMetadata = {
  encode(message: DatabaseSchemaMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.schemas) {
      SchemaMetadata.encode(v!, writer.uint32(18).fork()).ldelim();
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

  decode(input: _m0.Reader | Uint8Array, length?: number): DatabaseSchemaMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDatabaseSchemaMetadata();
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

          message.schemas.push(SchemaMetadata.decode(reader, reader.uint32()));
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

  fromJSON(object: any): DatabaseSchemaMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      schemas: globalThis.Array.isArray(object?.schemas)
        ? object.schemas.map((e: any) => SchemaMetadata.fromJSON(e))
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

  toJSON(message: DatabaseSchemaMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.schemas?.length) {
      obj.schemas = message.schemas.map((e) => SchemaMetadata.toJSON(e));
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

  create(base?: DeepPartial<DatabaseSchemaMetadata>): DatabaseSchemaMetadata {
    return DatabaseSchemaMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DatabaseSchemaMetadata>): DatabaseSchemaMetadata {
    const message = createBaseDatabaseSchemaMetadata();
    message.name = object.name ?? "";
    message.schemas = object.schemas?.map((e) => SchemaMetadata.fromPartial(e)) || [];
    message.characterSet = object.characterSet ?? "";
    message.collation = object.collation ?? "";
    message.extensions = object.extensions?.map((e) => ExtensionMetadata.fromPartial(e)) || [];
    message.datashare = object.datashare ?? false;
    message.serviceName = object.serviceName ?? "";
    return message;
  },
};

function createBaseSchemaMetadata(): SchemaMetadata {
  return { name: "", tables: [], views: [], functions: [], streams: [], tasks: [] };
}

export const SchemaMetadata = {
  encode(message: SchemaMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.tables) {
      TableMetadata.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.views) {
      ViewMetadata.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.functions) {
      FunctionMetadata.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.streams) {
      StreamMetadata.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    for (const v of message.tasks) {
      TaskMetadata.encode(v!, writer.uint32(50).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SchemaMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSchemaMetadata();
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

          message.tables.push(TableMetadata.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.views.push(ViewMetadata.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.functions.push(FunctionMetadata.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.streams.push(StreamMetadata.decode(reader, reader.uint32()));
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.tasks.push(TaskMetadata.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SchemaMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      tables: globalThis.Array.isArray(object?.tables) ? object.tables.map((e: any) => TableMetadata.fromJSON(e)) : [],
      views: globalThis.Array.isArray(object?.views) ? object.views.map((e: any) => ViewMetadata.fromJSON(e)) : [],
      functions: globalThis.Array.isArray(object?.functions)
        ? object.functions.map((e: any) => FunctionMetadata.fromJSON(e))
        : [],
      streams: globalThis.Array.isArray(object?.streams)
        ? object.streams.map((e: any) => StreamMetadata.fromJSON(e))
        : [],
      tasks: globalThis.Array.isArray(object?.tasks) ? object.tasks.map((e: any) => TaskMetadata.fromJSON(e)) : [],
    };
  },

  toJSON(message: SchemaMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.tables?.length) {
      obj.tables = message.tables.map((e) => TableMetadata.toJSON(e));
    }
    if (message.views?.length) {
      obj.views = message.views.map((e) => ViewMetadata.toJSON(e));
    }
    if (message.functions?.length) {
      obj.functions = message.functions.map((e) => FunctionMetadata.toJSON(e));
    }
    if (message.streams?.length) {
      obj.streams = message.streams.map((e) => StreamMetadata.toJSON(e));
    }
    if (message.tasks?.length) {
      obj.tasks = message.tasks.map((e) => TaskMetadata.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<SchemaMetadata>): SchemaMetadata {
    return SchemaMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SchemaMetadata>): SchemaMetadata {
    const message = createBaseSchemaMetadata();
    message.name = object.name ?? "";
    message.tables = object.tables?.map((e) => TableMetadata.fromPartial(e)) || [];
    message.views = object.views?.map((e) => ViewMetadata.fromPartial(e)) || [];
    message.functions = object.functions?.map((e) => FunctionMetadata.fromPartial(e)) || [];
    message.streams = object.streams?.map((e) => StreamMetadata.fromPartial(e)) || [];
    message.tasks = object.tasks?.map((e) => TaskMetadata.fromPartial(e)) || [];
    return message;
  },
};

function createBaseTaskMetadata(): TaskMetadata {
  return {
    name: "",
    id: "",
    owner: "",
    comment: "",
    warehouse: "",
    schedule: "",
    predecessors: [],
    state: 0,
    condition: "",
    definition: "",
  };
}

export const TaskMetadata = {
  encode(message: TaskMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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
    if (message.state !== 0) {
      writer.uint32(64).int32(message.state);
    }
    if (message.condition !== "") {
      writer.uint32(74).string(message.condition);
    }
    if (message.definition !== "") {
      writer.uint32(82).string(message.definition);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTaskMetadata();
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

          message.state = reader.int32() as any;
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

  fromJSON(object: any): TaskMetadata {
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
      state: isSet(object.state) ? taskMetadata_StateFromJSON(object.state) : 0,
      condition: isSet(object.condition) ? globalThis.String(object.condition) : "",
      definition: isSet(object.definition) ? globalThis.String(object.definition) : "",
    };
  },

  toJSON(message: TaskMetadata): unknown {
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
    if (message.state !== 0) {
      obj.state = taskMetadata_StateToJSON(message.state);
    }
    if (message.condition !== "") {
      obj.condition = message.condition;
    }
    if (message.definition !== "") {
      obj.definition = message.definition;
    }
    return obj;
  },

  create(base?: DeepPartial<TaskMetadata>): TaskMetadata {
    return TaskMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TaskMetadata>): TaskMetadata {
    const message = createBaseTaskMetadata();
    message.name = object.name ?? "";
    message.id = object.id ?? "";
    message.owner = object.owner ?? "";
    message.comment = object.comment ?? "";
    message.warehouse = object.warehouse ?? "";
    message.schedule = object.schedule ?? "";
    message.predecessors = object.predecessors?.map((e) => e) || [];
    message.state = object.state ?? 0;
    message.condition = object.condition ?? "";
    message.definition = object.definition ?? "";
    return message;
  },
};

function createBaseStreamMetadata(): StreamMetadata {
  return { name: "", tableName: "", owner: "", comment: "", type: 0, stale: false, mode: 0, definition: "" };
}

export const StreamMetadata = {
  encode(message: StreamMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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
    if (message.type !== 0) {
      writer.uint32(40).int32(message.type);
    }
    if (message.stale === true) {
      writer.uint32(48).bool(message.stale);
    }
    if (message.mode !== 0) {
      writer.uint32(56).int32(message.mode);
    }
    if (message.definition !== "") {
      writer.uint32(66).string(message.definition);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): StreamMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStreamMetadata();
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

          message.type = reader.int32() as any;
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

          message.mode = reader.int32() as any;
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

  fromJSON(object: any): StreamMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      tableName: isSet(object.tableName) ? globalThis.String(object.tableName) : "",
      owner: isSet(object.owner) ? globalThis.String(object.owner) : "",
      comment: isSet(object.comment) ? globalThis.String(object.comment) : "",
      type: isSet(object.type) ? streamMetadata_TypeFromJSON(object.type) : 0,
      stale: isSet(object.stale) ? globalThis.Boolean(object.stale) : false,
      mode: isSet(object.mode) ? streamMetadata_ModeFromJSON(object.mode) : 0,
      definition: isSet(object.definition) ? globalThis.String(object.definition) : "",
    };
  },

  toJSON(message: StreamMetadata): unknown {
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
    if (message.type !== 0) {
      obj.type = streamMetadata_TypeToJSON(message.type);
    }
    if (message.stale === true) {
      obj.stale = message.stale;
    }
    if (message.mode !== 0) {
      obj.mode = streamMetadata_ModeToJSON(message.mode);
    }
    if (message.definition !== "") {
      obj.definition = message.definition;
    }
    return obj;
  },

  create(base?: DeepPartial<StreamMetadata>): StreamMetadata {
    return StreamMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<StreamMetadata>): StreamMetadata {
    const message = createBaseStreamMetadata();
    message.name = object.name ?? "";
    message.tableName = object.tableName ?? "";
    message.owner = object.owner ?? "";
    message.comment = object.comment ?? "";
    message.type = object.type ?? 0;
    message.stale = object.stale ?? false;
    message.mode = object.mode ?? 0;
    message.definition = object.definition ?? "";
    return message;
  },
};

function createBaseTableMetadata(): TableMetadata {
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
  };
}

export const TableMetadata = {
  encode(message: TableMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.columns) {
      ColumnMetadata.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.indexes) {
      IndexMetadata.encode(v!, writer.uint32(26).fork()).ldelim();
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
      ForeignKeyMetadata.encode(v!, writer.uint32(98).fork()).ldelim();
    }
    for (const v of message.partitions) {
      TablePartitionMetadata.encode(v!, writer.uint32(122).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TableMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTableMetadata();
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

          message.columns.push(ColumnMetadata.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.indexes.push(IndexMetadata.decode(reader, reader.uint32()));
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

          message.foreignKeys.push(ForeignKeyMetadata.decode(reader, reader.uint32()));
          continue;
        case 15:
          if (tag !== 122) {
            break;
          }

          message.partitions.push(TablePartitionMetadata.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TableMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      columns: globalThis.Array.isArray(object?.columns)
        ? object.columns.map((e: any) => ColumnMetadata.fromJSON(e))
        : [],
      indexes: globalThis.Array.isArray(object?.indexes)
        ? object.indexes.map((e: any) => IndexMetadata.fromJSON(e))
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
        ? object.foreignKeys.map((e: any) => ForeignKeyMetadata.fromJSON(e))
        : [],
      partitions: globalThis.Array.isArray(object?.partitions)
        ? object.partitions.map((e: any) => TablePartitionMetadata.fromJSON(e))
        : [],
    };
  },

  toJSON(message: TableMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.columns?.length) {
      obj.columns = message.columns.map((e) => ColumnMetadata.toJSON(e));
    }
    if (message.indexes?.length) {
      obj.indexes = message.indexes.map((e) => IndexMetadata.toJSON(e));
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
      obj.foreignKeys = message.foreignKeys.map((e) => ForeignKeyMetadata.toJSON(e));
    }
    if (message.partitions?.length) {
      obj.partitions = message.partitions.map((e) => TablePartitionMetadata.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<TableMetadata>): TableMetadata {
    return TableMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TableMetadata>): TableMetadata {
    const message = createBaseTableMetadata();
    message.name = object.name ?? "";
    message.columns = object.columns?.map((e) => ColumnMetadata.fromPartial(e)) || [];
    message.indexes = object.indexes?.map((e) => IndexMetadata.fromPartial(e)) || [];
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
    message.foreignKeys = object.foreignKeys?.map((e) => ForeignKeyMetadata.fromPartial(e)) || [];
    message.partitions = object.partitions?.map((e) => TablePartitionMetadata.fromPartial(e)) || [];
    return message;
  },
};

function createBaseTablePartitionMetadata(): TablePartitionMetadata {
  return { name: "", type: 0, expression: "", subpartitions: [] };
}

export const TablePartitionMetadata = {
  encode(message: TablePartitionMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.type !== 0) {
      writer.uint32(16).int32(message.type);
    }
    if (message.expression !== "") {
      writer.uint32(26).string(message.expression);
    }
    for (const v of message.subpartitions) {
      TablePartitionMetadata.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TablePartitionMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTablePartitionMetadata();
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

          message.type = reader.int32() as any;
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

          message.subpartitions.push(TablePartitionMetadata.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TablePartitionMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      type: isSet(object.type) ? tablePartitionMetadata_TypeFromJSON(object.type) : 0,
      expression: isSet(object.expression) ? globalThis.String(object.expression) : "",
      subpartitions: globalThis.Array.isArray(object?.subpartitions)
        ? object.subpartitions.map((e: any) => TablePartitionMetadata.fromJSON(e))
        : [],
    };
  },

  toJSON(message: TablePartitionMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.type !== 0) {
      obj.type = tablePartitionMetadata_TypeToJSON(message.type);
    }
    if (message.expression !== "") {
      obj.expression = message.expression;
    }
    if (message.subpartitions?.length) {
      obj.subpartitions = message.subpartitions.map((e) => TablePartitionMetadata.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<TablePartitionMetadata>): TablePartitionMetadata {
    return TablePartitionMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TablePartitionMetadata>): TablePartitionMetadata {
    const message = createBaseTablePartitionMetadata();
    message.name = object.name ?? "";
    message.type = object.type ?? 0;
    message.expression = object.expression ?? "";
    message.subpartitions = object.subpartitions?.map((e) => TablePartitionMetadata.fromPartial(e)) || [];
    return message;
  },
};

function createBaseColumnMetadata(): ColumnMetadata {
  return {
    name: "",
    position: 0,
    default: undefined,
    defaultNull: undefined,
    defaultExpression: undefined,
    nullable: false,
    type: "",
    characterSet: "",
    collation: "",
    comment: "",
    classification: "",
    userComment: "",
  };
}

export const ColumnMetadata = {
  encode(message: ColumnMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): ColumnMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseColumnMetadata();
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

  fromJSON(object: any): ColumnMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      position: isSet(object.position) ? globalThis.Number(object.position) : 0,
      default: isSet(object.default) ? String(object.default) : undefined,
      defaultNull: isSet(object.defaultNull) ? globalThis.Boolean(object.defaultNull) : undefined,
      defaultExpression: isSet(object.defaultExpression) ? globalThis.String(object.defaultExpression) : undefined,
      nullable: isSet(object.nullable) ? globalThis.Boolean(object.nullable) : false,
      type: isSet(object.type) ? globalThis.String(object.type) : "",
      characterSet: isSet(object.characterSet) ? globalThis.String(object.characterSet) : "",
      collation: isSet(object.collation) ? globalThis.String(object.collation) : "",
      comment: isSet(object.comment) ? globalThis.String(object.comment) : "",
      classification: isSet(object.classification) ? globalThis.String(object.classification) : "",
      userComment: isSet(object.userComment) ? globalThis.String(object.userComment) : "",
    };
  },

  toJSON(message: ColumnMetadata): unknown {
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

  create(base?: DeepPartial<ColumnMetadata>): ColumnMetadata {
    return ColumnMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ColumnMetadata>): ColumnMetadata {
    const message = createBaseColumnMetadata();
    message.name = object.name ?? "";
    message.position = object.position ?? 0;
    message.default = object.default ?? undefined;
    message.defaultNull = object.defaultNull ?? undefined;
    message.defaultExpression = object.defaultExpression ?? undefined;
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

function createBaseViewMetadata(): ViewMetadata {
  return { name: "", definition: "", comment: "", dependentColumns: [] };
}

export const ViewMetadata = {
  encode(message: ViewMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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
      DependentColumn.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ViewMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseViewMetadata();
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

          message.dependentColumns.push(DependentColumn.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ViewMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      definition: isSet(object.definition) ? globalThis.String(object.definition) : "",
      comment: isSet(object.comment) ? globalThis.String(object.comment) : "",
      dependentColumns: globalThis.Array.isArray(object?.dependentColumns)
        ? object.dependentColumns.map((e: any) => DependentColumn.fromJSON(e))
        : [],
    };
  },

  toJSON(message: ViewMetadata): unknown {
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
      obj.dependentColumns = message.dependentColumns.map((e) => DependentColumn.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<ViewMetadata>): ViewMetadata {
    return ViewMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ViewMetadata>): ViewMetadata {
    const message = createBaseViewMetadata();
    message.name = object.name ?? "";
    message.definition = object.definition ?? "";
    message.comment = object.comment ?? "";
    message.dependentColumns = object.dependentColumns?.map((e) => DependentColumn.fromPartial(e)) || [];
    return message;
  },
};

function createBaseDependentColumn(): DependentColumn {
  return { schema: "", table: "", column: "" };
}

export const DependentColumn = {
  encode(message: DependentColumn, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): DependentColumn {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDependentColumn();
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

  fromJSON(object: any): DependentColumn {
    return {
      schema: isSet(object.schema) ? globalThis.String(object.schema) : "",
      table: isSet(object.table) ? globalThis.String(object.table) : "",
      column: isSet(object.column) ? globalThis.String(object.column) : "",
    };
  },

  toJSON(message: DependentColumn): unknown {
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

  create(base?: DeepPartial<DependentColumn>): DependentColumn {
    return DependentColumn.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DependentColumn>): DependentColumn {
    const message = createBaseDependentColumn();
    message.schema = object.schema ?? "";
    message.table = object.table ?? "";
    message.column = object.column ?? "";
    return message;
  },
};

function createBaseFunctionMetadata(): FunctionMetadata {
  return { name: "", definition: "" };
}

export const FunctionMetadata = {
  encode(message: FunctionMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.definition !== "") {
      writer.uint32(18).string(message.definition);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): FunctionMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFunctionMetadata();
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
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): FunctionMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      definition: isSet(object.definition) ? globalThis.String(object.definition) : "",
    };
  },

  toJSON(message: FunctionMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.definition !== "") {
      obj.definition = message.definition;
    }
    return obj;
  },

  create(base?: DeepPartial<FunctionMetadata>): FunctionMetadata {
    return FunctionMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<FunctionMetadata>): FunctionMetadata {
    const message = createBaseFunctionMetadata();
    message.name = object.name ?? "";
    message.definition = object.definition ?? "";
    return message;
  },
};

function createBaseIndexMetadata(): IndexMetadata {
  return { name: "", expressions: [], type: "", unique: false, primary: false, visible: false, comment: "" };
}

export const IndexMetadata = {
  encode(message: IndexMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.expressions) {
      writer.uint32(18).string(v!);
    }
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
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IndexMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIndexMetadata();
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
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): IndexMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      expressions: globalThis.Array.isArray(object?.expressions)
        ? object.expressions.map((e: any) => globalThis.String(e))
        : [],
      type: isSet(object.type) ? globalThis.String(object.type) : "",
      unique: isSet(object.unique) ? globalThis.Boolean(object.unique) : false,
      primary: isSet(object.primary) ? globalThis.Boolean(object.primary) : false,
      visible: isSet(object.visible) ? globalThis.Boolean(object.visible) : false,
      comment: isSet(object.comment) ? globalThis.String(object.comment) : "",
    };
  },

  toJSON(message: IndexMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.expressions?.length) {
      obj.expressions = message.expressions;
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
    return obj;
  },

  create(base?: DeepPartial<IndexMetadata>): IndexMetadata {
    return IndexMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<IndexMetadata>): IndexMetadata {
    const message = createBaseIndexMetadata();
    message.name = object.name ?? "";
    message.expressions = object.expressions?.map((e) => e) || [];
    message.type = object.type ?? "";
    message.unique = object.unique ?? false;
    message.primary = object.primary ?? false;
    message.visible = object.visible ?? false;
    message.comment = object.comment ?? "";
    return message;
  },
};

function createBaseExtensionMetadata(): ExtensionMetadata {
  return { name: "", schema: "", version: "", description: "" };
}

export const ExtensionMetadata = {
  encode(message: ExtensionMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): ExtensionMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExtensionMetadata();
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

  fromJSON(object: any): ExtensionMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      schema: isSet(object.schema) ? globalThis.String(object.schema) : "",
      version: isSet(object.version) ? globalThis.String(object.version) : "",
      description: isSet(object.description) ? globalThis.String(object.description) : "",
    };
  },

  toJSON(message: ExtensionMetadata): unknown {
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

  create(base?: DeepPartial<ExtensionMetadata>): ExtensionMetadata {
    return ExtensionMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ExtensionMetadata>): ExtensionMetadata {
    const message = createBaseExtensionMetadata();
    message.name = object.name ?? "";
    message.schema = object.schema ?? "";
    message.version = object.version ?? "";
    message.description = object.description ?? "";
    return message;
  },
};

function createBaseForeignKeyMetadata(): ForeignKeyMetadata {
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

export const ForeignKeyMetadata = {
  encode(message: ForeignKeyMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): ForeignKeyMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseForeignKeyMetadata();
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

  fromJSON(object: any): ForeignKeyMetadata {
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

  toJSON(message: ForeignKeyMetadata): unknown {
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

  create(base?: DeepPartial<ForeignKeyMetadata>): ForeignKeyMetadata {
    return ForeignKeyMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ForeignKeyMetadata>): ForeignKeyMetadata {
    const message = createBaseForeignKeyMetadata();
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

function createBaseInstanceRoleMetadata(): InstanceRoleMetadata {
  return { name: "", grant: "" };
}

export const InstanceRoleMetadata = {
  encode(message: InstanceRoleMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.grant !== "") {
      writer.uint32(58).string(message.grant);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): InstanceRoleMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseInstanceRoleMetadata();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 2:
          if (tag !== 18) {
            break;
          }

          message.name = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.grant = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): InstanceRoleMetadata {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      grant: isSet(object.grant) ? globalThis.String(object.grant) : "",
    };
  },

  toJSON(message: InstanceRoleMetadata): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.grant !== "") {
      obj.grant = message.grant;
    }
    return obj;
  },

  create(base?: DeepPartial<InstanceRoleMetadata>): InstanceRoleMetadata {
    return InstanceRoleMetadata.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<InstanceRoleMetadata>): InstanceRoleMetadata {
    const message = createBaseInstanceRoleMetadata();
    message.name = object.name ?? "";
    message.grant = object.grant ?? "";
    return message;
  },
};

function createBaseSecrets(): Secrets {
  return { items: [] };
}

export const Secrets = {
  encode(message: Secrets, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.items) {
      SecretItem.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Secrets {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSecrets();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.items.push(SecretItem.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Secrets {
    return {
      items: globalThis.Array.isArray(object?.items) ? object.items.map((e: any) => SecretItem.fromJSON(e)) : [],
    };
  },

  toJSON(message: Secrets): unknown {
    const obj: any = {};
    if (message.items?.length) {
      obj.items = message.items.map((e) => SecretItem.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<Secrets>): Secrets {
    return Secrets.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Secrets>): Secrets {
    const message = createBaseSecrets();
    message.items = object.items?.map((e) => SecretItem.fromPartial(e)) || [];
    return message;
  },
};

function createBaseSecretItem(): SecretItem {
  return { name: "", value: "", description: "" };
}

export const SecretItem = {
  encode(message: SecretItem, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.value !== "") {
      writer.uint32(18).string(message.value);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SecretItem {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSecretItem();
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

          message.value = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
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

  fromJSON(object: any): SecretItem {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      value: isSet(object.value) ? globalThis.String(object.value) : "",
      description: isSet(object.description) ? globalThis.String(object.description) : "",
    };
  },

  toJSON(message: SecretItem): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.value !== "") {
      obj.value = message.value;
    }
    if (message.description !== "") {
      obj.description = message.description;
    }
    return obj;
  },

  create(base?: DeepPartial<SecretItem>): SecretItem {
    return SecretItem.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SecretItem>): SecretItem {
    const message = createBaseSecretItem();
    message.name = object.name ?? "";
    message.value = object.value ?? "";
    message.description = object.description ?? "";
    return message;
  },
};

function createBaseDatabaseConfig(): DatabaseConfig {
  return { name: "", schemaConfigs: [] };
}

export const DatabaseConfig = {
  encode(message: DatabaseConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.schemaConfigs) {
      SchemaConfig.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DatabaseConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDatabaseConfig();
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

          message.schemaConfigs.push(SchemaConfig.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DatabaseConfig {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      schemaConfigs: globalThis.Array.isArray(object?.schemaConfigs)
        ? object.schemaConfigs.map((e: any) => SchemaConfig.fromJSON(e))
        : [],
    };
  },

  toJSON(message: DatabaseConfig): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.schemaConfigs?.length) {
      obj.schemaConfigs = message.schemaConfigs.map((e) => SchemaConfig.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<DatabaseConfig>): DatabaseConfig {
    return DatabaseConfig.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DatabaseConfig>): DatabaseConfig {
    const message = createBaseDatabaseConfig();
    message.name = object.name ?? "";
    message.schemaConfigs = object.schemaConfigs?.map((e) => SchemaConfig.fromPartial(e)) || [];
    return message;
  },
};

function createBaseSchemaConfig(): SchemaConfig {
  return { name: "", tableConfigs: [] };
}

export const SchemaConfig = {
  encode(message: SchemaConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.tableConfigs) {
      TableConfig.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SchemaConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSchemaConfig();
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

          message.tableConfigs.push(TableConfig.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SchemaConfig {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      tableConfigs: globalThis.Array.isArray(object?.tableConfigs)
        ? object.tableConfigs.map((e: any) => TableConfig.fromJSON(e))
        : [],
    };
  },

  toJSON(message: SchemaConfig): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.tableConfigs?.length) {
      obj.tableConfigs = message.tableConfigs.map((e) => TableConfig.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<SchemaConfig>): SchemaConfig {
    return SchemaConfig.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SchemaConfig>): SchemaConfig {
    const message = createBaseSchemaConfig();
    message.name = object.name ?? "";
    message.tableConfigs = object.tableConfigs?.map((e) => TableConfig.fromPartial(e)) || [];
    return message;
  },
};

function createBaseTableConfig(): TableConfig {
  return { name: "", columnConfigs: [] };
}

export const TableConfig = {
  encode(message: TableConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.columnConfigs) {
      ColumnConfig.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TableConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTableConfig();
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

          message.columnConfigs.push(ColumnConfig.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TableConfig {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      columnConfigs: globalThis.Array.isArray(object?.columnConfigs)
        ? object.columnConfigs.map((e: any) => ColumnConfig.fromJSON(e))
        : [],
    };
  },

  toJSON(message: TableConfig): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.columnConfigs?.length) {
      obj.columnConfigs = message.columnConfigs.map((e) => ColumnConfig.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<TableConfig>): TableConfig {
    return TableConfig.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TableConfig>): TableConfig {
    const message = createBaseTableConfig();
    message.name = object.name ?? "";
    message.columnConfigs = object.columnConfigs?.map((e) => ColumnConfig.fromPartial(e)) || [];
    return message;
  },
};

function createBaseColumnConfig(): ColumnConfig {
  return { name: "", semanticTypeId: "", labels: {} };
}

export const ColumnConfig = {
  encode(message: ColumnConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.semanticTypeId !== "") {
      writer.uint32(18).string(message.semanticTypeId);
    }
    Object.entries(message.labels).forEach(([key, value]) => {
      ColumnConfig_LabelsEntry.encode({ key: key as any, value }, writer.uint32(26).fork()).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ColumnConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseColumnConfig();
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

          message.semanticTypeId = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          const entry3 = ColumnConfig_LabelsEntry.decode(reader, reader.uint32());
          if (entry3.value !== undefined) {
            message.labels[entry3.key] = entry3.value;
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

  fromJSON(object: any): ColumnConfig {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      semanticTypeId: isSet(object.semanticTypeId) ? globalThis.String(object.semanticTypeId) : "",
      labels: isObject(object.labels)
        ? Object.entries(object.labels).reduce<{ [key: string]: string }>((acc, [key, value]) => {
          acc[key] = String(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: ColumnConfig): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.semanticTypeId !== "") {
      obj.semanticTypeId = message.semanticTypeId;
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

  create(base?: DeepPartial<ColumnConfig>): ColumnConfig {
    return ColumnConfig.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ColumnConfig>): ColumnConfig {
    const message = createBaseColumnConfig();
    message.name = object.name ?? "";
    message.semanticTypeId = object.semanticTypeId ?? "";
    message.labels = Object.entries(object.labels ?? {}).reduce<{ [key: string]: string }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = globalThis.String(value);
      }
      return acc;
    }, {});
    return message;
  },
};

function createBaseColumnConfig_LabelsEntry(): ColumnConfig_LabelsEntry {
  return { key: "", value: "" };
}

export const ColumnConfig_LabelsEntry = {
  encode(message: ColumnConfig_LabelsEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== "") {
      writer.uint32(18).string(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ColumnConfig_LabelsEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseColumnConfig_LabelsEntry();
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

  fromJSON(object: any): ColumnConfig_LabelsEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? globalThis.String(object.value) : "",
    };
  },

  toJSON(message: ColumnConfig_LabelsEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== "") {
      obj.value = message.value;
    }
    return obj;
  },

  create(base?: DeepPartial<ColumnConfig_LabelsEntry>): ColumnConfig_LabelsEntry {
    return ColumnConfig_LabelsEntry.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ColumnConfig_LabelsEntry>): ColumnConfig_LabelsEntry {
    const message = createBaseColumnConfig_LabelsEntry();
    message.key = object.key ?? "";
    message.value = object.value ?? "";
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
