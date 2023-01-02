/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { FieldMask } from "../google/protobuf/field_mask";
import { Timestamp } from "../google/protobuf/timestamp";
import { StringValue } from "../google/protobuf/wrappers";
import { State, stateFromJSON, stateToJSON } from "./common";

export const protobufPackage = "bytebase.v1";

export interface ListDatabasesRequest {
  /**
   * The parent, which owns this collection of databases.
   * Format: environments/{environment}/instances/{instance}
   * Use "environments/-/instances/-" to list all databases from all environments.
   */
  parent: string;
  /**
   * The maximum number of databases to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 databases will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListDatabases` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListDatabases` must match
   * the call that provided the page token.
   */
  pageToken: string;
  /**
   * Filter is used to filter databases returned in the list.
   * For example, "project = projects/{project}" can be used to list databases in a project.
   */
  filter: string;
}

export interface ListDatabasesResponse {
  /** The databases from the specified request. */
  databases: Database[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface UpdateDatabaseRequest {
  /**
   * The database to update.
   *
   * The database's `name` field is used to identify the database to update.
   * Format: environments/{environment}/instances/{instance}/databases/{database}
   */
  database?: Database;
  /** The list of fields to update. */
  updateMask?: string[];
}

export interface BatchUpdateDatabasesRequest {
  /**
   * The parent resource shared by all databases being updated.
   * Format: environments/{environment}/instances/{instance}
   * If this is set, the parent field in the UpdateDatabaseRequest messages
   * must either be empty or match this field.
   * If the operation spans parents, a dash (-) may be accepted as a wildcard.
   */
  parent: string;
  /**
   * The request message specifying the resources to update.
   * A maximum of 1000 databases can be modified in a batch.
   */
  requests: UpdateDatabaseRequest[];
}

export interface BatchUpdateDatabasesResponse {
  /** Databases updated. */
  databases: Database[];
}

export interface Database {
  /**
   * The name of the database.
   * Format: environments/{environment}/instances/{instance}/databases/{database}
   * {database} is the database name in the instance.
   */
  name: string;
  /** The existence of a database on latest sync. */
  syncState: State;
  /** The latest synchronization time. */
  successfulSyncTime?: Date;
  /**
   * The project for a database.
   * Format: projects/{project}
   */
  project: string;
  characterSet: string;
  collation: string;
  /** The version of database schema. */
  schemaVersion: string;
  /** Labels will be used for deployment and policy control. */
  labels: { [key: string]: string };
}

export interface Database_LabelsEntry {
  key: string;
  value: string;
}

export interface GetDatabaseMetadataRequest {
  /**
   * The name of the database to retrieve metadata.
   * Format: environments/{environment}/instances/{instance}/databases/{database}
   */
  name: string;
}

export interface GetDatabaseSchemaRequest {
  /**
   * The name of the database to retrieve schema.
   * Format: environments/{environment}/instances/{instance}/databases/{database}
   */
  name: string;
}

/** DatabaseMetadata is the metadata for databases. */
export interface DatabaseMetadata {
  name: string;
  /** The schemas is the list of schemas in a database. */
  schemas: SchemaMetadata[];
  /** The character_set is the character set of a database. */
  characterSet: string;
  /** The collation is the collation of a database. */
  collation: string;
  /** The extensions is the list of extensions in a database. */
  extensions: ExtensionMetadata[];
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
  rowCount: number;
  /** The data_size is the estimated data size of a table. */
  dataSize: number;
  /** The index_size is the estimated index size of a table. */
  indexSize: number;
  /** The data_free is the estimated free data size of a table. */
  dataFree: number;
  /** The create_options is the create option of a table. */
  createOptions: string;
  /** The comment is the comment of a table. */
  comment: string;
  /** The foreign_keys is the list of foreign keys in a table. */
  foreignKeys: ForeignKeyMetadata[];
}

/** ColumnMetadata is the metadata for columns. */
export interface ColumnMetadata {
  /** The name is the name of a column. */
  name: string;
  /** The position is the position in columns. */
  position: number;
  /** The default is the default of a column. Use google.protobuf.StringValue to distinguish between an empty string default value or no default. */
  default?: string;
  /** The nullable is the nullable of a column. */
  nullable: boolean;
  /** The type is the type of a column. */
  type: string;
  /** The character_set is the character_set of a column. */
  characterSet: string;
  /** The collation is the collation of a column. */
  collation: string;
  /** The comment is the comment of a column. */
  comment: string;
}

/** ViewMetadata is the metadata for views. */
export interface ViewMetadata {
  /** The name is the name of a view. */
  name: string;
  /** The definition is the definition of a view. */
  definition: string;
  /** The comment is the comment of a view. */
  comment: string;
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

/** DatabaseMetadata is the metadata for databases. */
export interface DatabaseSchema {
  /** The schema dump from database. */
  schema: string;
}

function createBaseListDatabasesRequest(): ListDatabasesRequest {
  return { parent: "", pageSize: 0, pageToken: "", filter: "" };
}

export const ListDatabasesRequest = {
  encode(message: ListDatabasesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.pageSize !== 0) {
      writer.uint32(16).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(26).string(message.pageToken);
    }
    if (message.filter !== "") {
      writer.uint32(34).string(message.filter);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListDatabasesRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListDatabasesRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.parent = reader.string();
          break;
        case 2:
          message.pageSize = reader.int32();
          break;
        case 3:
          message.pageToken = reader.string();
          break;
        case 4:
          message.filter = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ListDatabasesRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
      filter: isSet(object.filter) ? String(object.filter) : "",
    };
  },

  toJSON(message: ListDatabasesRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    message.filter !== undefined && (obj.filter = message.filter);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ListDatabasesRequest>, I>>(object: I): ListDatabasesRequest {
    const message = createBaseListDatabasesRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    message.filter = object.filter ?? "";
    return message;
  },
};

function createBaseListDatabasesResponse(): ListDatabasesResponse {
  return { databases: [], nextPageToken: "" };
}

export const ListDatabasesResponse = {
  encode(message: ListDatabasesResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.databases) {
      Database.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListDatabasesResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListDatabasesResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.databases.push(Database.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListDatabasesResponse {
    return {
      databases: Array.isArray(object?.databases) ? object.databases.map((e: any) => Database.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListDatabasesResponse): unknown {
    const obj: any = {};
    if (message.databases) {
      obj.databases = message.databases.map((e) => e ? Database.toJSON(e) : undefined);
    } else {
      obj.databases = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ListDatabasesResponse>, I>>(object: I): ListDatabasesResponse {
    const message = createBaseListDatabasesResponse();
    message.databases = object.databases?.map((e) => Database.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseUpdateDatabaseRequest(): UpdateDatabaseRequest {
  return { database: undefined, updateMask: undefined };
}

export const UpdateDatabaseRequest = {
  encode(message: UpdateDatabaseRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.database !== undefined) {
      Database.encode(message.database, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateDatabaseRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateDatabaseRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.database = Database.decode(reader, reader.uint32());
          break;
        case 2:
          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): UpdateDatabaseRequest {
    return {
      database: isSet(object.database) ? Database.fromJSON(object.database) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateDatabaseRequest): unknown {
    const obj: any = {};
    message.database !== undefined && (obj.database = message.database ? Database.toJSON(message.database) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<UpdateDatabaseRequest>, I>>(object: I): UpdateDatabaseRequest {
    const message = createBaseUpdateDatabaseRequest();
    message.database = (object.database !== undefined && object.database !== null)
      ? Database.fromPartial(object.database)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseBatchUpdateDatabasesRequest(): BatchUpdateDatabasesRequest {
  return { parent: "", requests: [] };
}

export const BatchUpdateDatabasesRequest = {
  encode(message: BatchUpdateDatabasesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    for (const v of message.requests) {
      UpdateDatabaseRequest.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchUpdateDatabasesRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchUpdateDatabasesRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.parent = reader.string();
          break;
        case 2:
          message.requests.push(UpdateDatabaseRequest.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): BatchUpdateDatabasesRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      requests: Array.isArray(object?.requests)
        ? object.requests.map((e: any) => UpdateDatabaseRequest.fromJSON(e))
        : [],
    };
  },

  toJSON(message: BatchUpdateDatabasesRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    if (message.requests) {
      obj.requests = message.requests.map((e) => e ? UpdateDatabaseRequest.toJSON(e) : undefined);
    } else {
      obj.requests = [];
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<BatchUpdateDatabasesRequest>, I>>(object: I): BatchUpdateDatabasesRequest {
    const message = createBaseBatchUpdateDatabasesRequest();
    message.parent = object.parent ?? "";
    message.requests = object.requests?.map((e) => UpdateDatabaseRequest.fromPartial(e)) || [];
    return message;
  },
};

function createBaseBatchUpdateDatabasesResponse(): BatchUpdateDatabasesResponse {
  return { databases: [] };
}

export const BatchUpdateDatabasesResponse = {
  encode(message: BatchUpdateDatabasesResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.databases) {
      Database.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchUpdateDatabasesResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchUpdateDatabasesResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.databases.push(Database.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): BatchUpdateDatabasesResponse {
    return {
      databases: Array.isArray(object?.databases) ? object.databases.map((e: any) => Database.fromJSON(e)) : [],
    };
  },

  toJSON(message: BatchUpdateDatabasesResponse): unknown {
    const obj: any = {};
    if (message.databases) {
      obj.databases = message.databases.map((e) => e ? Database.toJSON(e) : undefined);
    } else {
      obj.databases = [];
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<BatchUpdateDatabasesResponse>, I>>(object: I): BatchUpdateDatabasesResponse {
    const message = createBaseBatchUpdateDatabasesResponse();
    message.databases = object.databases?.map((e) => Database.fromPartial(e)) || [];
    return message;
  },
};

function createBaseDatabase(): Database {
  return {
    name: "",
    syncState: 0,
    successfulSyncTime: undefined,
    project: "",
    characterSet: "",
    collation: "",
    schemaVersion: "",
    labels: {},
  };
}

export const Database = {
  encode(message: Database, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.syncState !== 0) {
      writer.uint32(16).int32(message.syncState);
    }
    if (message.successfulSyncTime !== undefined) {
      Timestamp.encode(toTimestamp(message.successfulSyncTime), writer.uint32(26).fork()).ldelim();
    }
    if (message.project !== "") {
      writer.uint32(34).string(message.project);
    }
    if (message.characterSet !== "") {
      writer.uint32(42).string(message.characterSet);
    }
    if (message.collation !== "") {
      writer.uint32(50).string(message.collation);
    }
    if (message.schemaVersion !== "") {
      writer.uint32(58).string(message.schemaVersion);
    }
    Object.entries(message.labels).forEach(([key, value]) => {
      Database_LabelsEntry.encode({ key: key as any, value }, writer.uint32(66).fork()).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Database {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDatabase();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.syncState = reader.int32() as any;
          break;
        case 3:
          message.successfulSyncTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          break;
        case 4:
          message.project = reader.string();
          break;
        case 5:
          message.characterSet = reader.string();
          break;
        case 6:
          message.collation = reader.string();
          break;
        case 7:
          message.schemaVersion = reader.string();
          break;
        case 8:
          const entry8 = Database_LabelsEntry.decode(reader, reader.uint32());
          if (entry8.value !== undefined) {
            message.labels[entry8.key] = entry8.value;
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Database {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      syncState: isSet(object.syncState) ? stateFromJSON(object.syncState) : 0,
      successfulSyncTime: isSet(object.successfulSyncTime) ? fromJsonTimestamp(object.successfulSyncTime) : undefined,
      project: isSet(object.project) ? String(object.project) : "",
      characterSet: isSet(object.characterSet) ? String(object.characterSet) : "",
      collation: isSet(object.collation) ? String(object.collation) : "",
      schemaVersion: isSet(object.schemaVersion) ? String(object.schemaVersion) : "",
      labels: isObject(object.labels)
        ? Object.entries(object.labels).reduce<{ [key: string]: string }>((acc, [key, value]) => {
          acc[key] = String(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: Database): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.syncState !== undefined && (obj.syncState = stateToJSON(message.syncState));
    message.successfulSyncTime !== undefined && (obj.successfulSyncTime = message.successfulSyncTime.toISOString());
    message.project !== undefined && (obj.project = message.project);
    message.characterSet !== undefined && (obj.characterSet = message.characterSet);
    message.collation !== undefined && (obj.collation = message.collation);
    message.schemaVersion !== undefined && (obj.schemaVersion = message.schemaVersion);
    obj.labels = {};
    if (message.labels) {
      Object.entries(message.labels).forEach(([k, v]) => {
        obj.labels[k] = v;
      });
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<Database>, I>>(object: I): Database {
    const message = createBaseDatabase();
    message.name = object.name ?? "";
    message.syncState = object.syncState ?? 0;
    message.successfulSyncTime = object.successfulSyncTime ?? undefined;
    message.project = object.project ?? "";
    message.characterSet = object.characterSet ?? "";
    message.collation = object.collation ?? "";
    message.schemaVersion = object.schemaVersion ?? "";
    message.labels = Object.entries(object.labels ?? {}).reduce<{ [key: string]: string }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = String(value);
      }
      return acc;
    }, {});
    return message;
  },
};

function createBaseDatabase_LabelsEntry(): Database_LabelsEntry {
  return { key: "", value: "" };
}

export const Database_LabelsEntry = {
  encode(message: Database_LabelsEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== "") {
      writer.uint32(18).string(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Database_LabelsEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDatabase_LabelsEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Database_LabelsEntry {
    return { key: isSet(object.key) ? String(object.key) : "", value: isSet(object.value) ? String(object.value) : "" };
  },

  toJSON(message: Database_LabelsEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined && (obj.value = message.value);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<Database_LabelsEntry>, I>>(object: I): Database_LabelsEntry {
    const message = createBaseDatabase_LabelsEntry();
    message.key = object.key ?? "";
    message.value = object.value ?? "";
    return message;
  },
};

function createBaseGetDatabaseMetadataRequest(): GetDatabaseMetadataRequest {
  return { name: "" };
}

export const GetDatabaseMetadataRequest = {
  encode(message: GetDatabaseMetadataRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetDatabaseMetadataRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetDatabaseMetadataRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GetDatabaseMetadataRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetDatabaseMetadataRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<GetDatabaseMetadataRequest>, I>>(object: I): GetDatabaseMetadataRequest {
    const message = createBaseGetDatabaseMetadataRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseGetDatabaseSchemaRequest(): GetDatabaseSchemaRequest {
  return { name: "" };
}

export const GetDatabaseSchemaRequest = {
  encode(message: GetDatabaseSchemaRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetDatabaseSchemaRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetDatabaseSchemaRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GetDatabaseSchemaRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetDatabaseSchemaRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<GetDatabaseSchemaRequest>, I>>(object: I): GetDatabaseSchemaRequest {
    const message = createBaseGetDatabaseSchemaRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseDatabaseMetadata(): DatabaseMetadata {
  return { name: "", schemas: [], characterSet: "", collation: "", extensions: [] };
}

export const DatabaseMetadata = {
  encode(message: DatabaseMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DatabaseMetadata {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDatabaseMetadata();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.schemas.push(SchemaMetadata.decode(reader, reader.uint32()));
          break;
        case 3:
          message.characterSet = reader.string();
          break;
        case 4:
          message.collation = reader.string();
          break;
        case 5:
          message.extensions.push(ExtensionMetadata.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DatabaseMetadata {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      schemas: Array.isArray(object?.schemas) ? object.schemas.map((e: any) => SchemaMetadata.fromJSON(e)) : [],
      characterSet: isSet(object.characterSet) ? String(object.characterSet) : "",
      collation: isSet(object.collation) ? String(object.collation) : "",
      extensions: Array.isArray(object?.extensions)
        ? object.extensions.map((e: any) => ExtensionMetadata.fromJSON(e))
        : [],
    };
  },

  toJSON(message: DatabaseMetadata): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    if (message.schemas) {
      obj.schemas = message.schemas.map((e) => e ? SchemaMetadata.toJSON(e) : undefined);
    } else {
      obj.schemas = [];
    }
    message.characterSet !== undefined && (obj.characterSet = message.characterSet);
    message.collation !== undefined && (obj.collation = message.collation);
    if (message.extensions) {
      obj.extensions = message.extensions.map((e) => e ? ExtensionMetadata.toJSON(e) : undefined);
    } else {
      obj.extensions = [];
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DatabaseMetadata>, I>>(object: I): DatabaseMetadata {
    const message = createBaseDatabaseMetadata();
    message.name = object.name ?? "";
    message.schemas = object.schemas?.map((e) => SchemaMetadata.fromPartial(e)) || [];
    message.characterSet = object.characterSet ?? "";
    message.collation = object.collation ?? "";
    message.extensions = object.extensions?.map((e) => ExtensionMetadata.fromPartial(e)) || [];
    return message;
  },
};

function createBaseSchemaMetadata(): SchemaMetadata {
  return { name: "", tables: [], views: [] };
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
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SchemaMetadata {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSchemaMetadata();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.tables.push(TableMetadata.decode(reader, reader.uint32()));
          break;
        case 3:
          message.views.push(ViewMetadata.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): SchemaMetadata {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      tables: Array.isArray(object?.tables) ? object.tables.map((e: any) => TableMetadata.fromJSON(e)) : [],
      views: Array.isArray(object?.views) ? object.views.map((e: any) => ViewMetadata.fromJSON(e)) : [],
    };
  },

  toJSON(message: SchemaMetadata): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    if (message.tables) {
      obj.tables = message.tables.map((e) => e ? TableMetadata.toJSON(e) : undefined);
    } else {
      obj.tables = [];
    }
    if (message.views) {
      obj.views = message.views.map((e) => e ? ViewMetadata.toJSON(e) : undefined);
    } else {
      obj.views = [];
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<SchemaMetadata>, I>>(object: I): SchemaMetadata {
    const message = createBaseSchemaMetadata();
    message.name = object.name ?? "";
    message.tables = object.tables?.map((e) => TableMetadata.fromPartial(e)) || [];
    message.views = object.views?.map((e) => ViewMetadata.fromPartial(e)) || [];
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
    rowCount: 0,
    dataSize: 0,
    indexSize: 0,
    dataFree: 0,
    createOptions: "",
    comment: "",
    foreignKeys: [],
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
    if (message.rowCount !== 0) {
      writer.uint32(48).int64(message.rowCount);
    }
    if (message.dataSize !== 0) {
      writer.uint32(56).int64(message.dataSize);
    }
    if (message.indexSize !== 0) {
      writer.uint32(64).int64(message.indexSize);
    }
    if (message.dataFree !== 0) {
      writer.uint32(72).int64(message.dataFree);
    }
    if (message.createOptions !== "") {
      writer.uint32(82).string(message.createOptions);
    }
    if (message.comment !== "") {
      writer.uint32(90).string(message.comment);
    }
    for (const v of message.foreignKeys) {
      ForeignKeyMetadata.encode(v!, writer.uint32(98).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TableMetadata {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTableMetadata();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.columns.push(ColumnMetadata.decode(reader, reader.uint32()));
          break;
        case 3:
          message.indexes.push(IndexMetadata.decode(reader, reader.uint32()));
          break;
        case 4:
          message.engine = reader.string();
          break;
        case 5:
          message.collation = reader.string();
          break;
        case 6:
          message.rowCount = longToNumber(reader.int64() as Long);
          break;
        case 7:
          message.dataSize = longToNumber(reader.int64() as Long);
          break;
        case 8:
          message.indexSize = longToNumber(reader.int64() as Long);
          break;
        case 9:
          message.dataFree = longToNumber(reader.int64() as Long);
          break;
        case 10:
          message.createOptions = reader.string();
          break;
        case 11:
          message.comment = reader.string();
          break;
        case 12:
          message.foreignKeys.push(ForeignKeyMetadata.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): TableMetadata {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      columns: Array.isArray(object?.columns) ? object.columns.map((e: any) => ColumnMetadata.fromJSON(e)) : [],
      indexes: Array.isArray(object?.indexes) ? object.indexes.map((e: any) => IndexMetadata.fromJSON(e)) : [],
      engine: isSet(object.engine) ? String(object.engine) : "",
      collation: isSet(object.collation) ? String(object.collation) : "",
      rowCount: isSet(object.rowCount) ? Number(object.rowCount) : 0,
      dataSize: isSet(object.dataSize) ? Number(object.dataSize) : 0,
      indexSize: isSet(object.indexSize) ? Number(object.indexSize) : 0,
      dataFree: isSet(object.dataFree) ? Number(object.dataFree) : 0,
      createOptions: isSet(object.createOptions) ? String(object.createOptions) : "",
      comment: isSet(object.comment) ? String(object.comment) : "",
      foreignKeys: Array.isArray(object?.foreignKeys)
        ? object.foreignKeys.map((e: any) => ForeignKeyMetadata.fromJSON(e))
        : [],
    };
  },

  toJSON(message: TableMetadata): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    if (message.columns) {
      obj.columns = message.columns.map((e) => e ? ColumnMetadata.toJSON(e) : undefined);
    } else {
      obj.columns = [];
    }
    if (message.indexes) {
      obj.indexes = message.indexes.map((e) => e ? IndexMetadata.toJSON(e) : undefined);
    } else {
      obj.indexes = [];
    }
    message.engine !== undefined && (obj.engine = message.engine);
    message.collation !== undefined && (obj.collation = message.collation);
    message.rowCount !== undefined && (obj.rowCount = Math.round(message.rowCount));
    message.dataSize !== undefined && (obj.dataSize = Math.round(message.dataSize));
    message.indexSize !== undefined && (obj.indexSize = Math.round(message.indexSize));
    message.dataFree !== undefined && (obj.dataFree = Math.round(message.dataFree));
    message.createOptions !== undefined && (obj.createOptions = message.createOptions);
    message.comment !== undefined && (obj.comment = message.comment);
    if (message.foreignKeys) {
      obj.foreignKeys = message.foreignKeys.map((e) => e ? ForeignKeyMetadata.toJSON(e) : undefined);
    } else {
      obj.foreignKeys = [];
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<TableMetadata>, I>>(object: I): TableMetadata {
    const message = createBaseTableMetadata();
    message.name = object.name ?? "";
    message.columns = object.columns?.map((e) => ColumnMetadata.fromPartial(e)) || [];
    message.indexes = object.indexes?.map((e) => IndexMetadata.fromPartial(e)) || [];
    message.engine = object.engine ?? "";
    message.collation = object.collation ?? "";
    message.rowCount = object.rowCount ?? 0;
    message.dataSize = object.dataSize ?? 0;
    message.indexSize = object.indexSize ?? 0;
    message.dataFree = object.dataFree ?? 0;
    message.createOptions = object.createOptions ?? "";
    message.comment = object.comment ?? "";
    message.foreignKeys = object.foreignKeys?.map((e) => ForeignKeyMetadata.fromPartial(e)) || [];
    return message;
  },
};

function createBaseColumnMetadata(): ColumnMetadata {
  return {
    name: "",
    position: 0,
    default: undefined,
    nullable: false,
    type: "",
    characterSet: "",
    collation: "",
    comment: "",
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
    if (message.nullable === true) {
      writer.uint32(32).bool(message.nullable);
    }
    if (message.type !== "") {
      writer.uint32(42).string(message.type);
    }
    if (message.characterSet !== "") {
      writer.uint32(50).string(message.characterSet);
    }
    if (message.collation !== "") {
      writer.uint32(58).string(message.collation);
    }
    if (message.comment !== "") {
      writer.uint32(66).string(message.comment);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ColumnMetadata {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseColumnMetadata();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.position = reader.int32();
          break;
        case 3:
          message.default = StringValue.decode(reader, reader.uint32()).value;
          break;
        case 4:
          message.nullable = reader.bool();
          break;
        case 5:
          message.type = reader.string();
          break;
        case 6:
          message.characterSet = reader.string();
          break;
        case 7:
          message.collation = reader.string();
          break;
        case 8:
          message.comment = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ColumnMetadata {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      position: isSet(object.position) ? Number(object.position) : 0,
      default: isSet(object.default) ? String(object.default) : undefined,
      nullable: isSet(object.nullable) ? Boolean(object.nullable) : false,
      type: isSet(object.type) ? String(object.type) : "",
      characterSet: isSet(object.characterSet) ? String(object.characterSet) : "",
      collation: isSet(object.collation) ? String(object.collation) : "",
      comment: isSet(object.comment) ? String(object.comment) : "",
    };
  },

  toJSON(message: ColumnMetadata): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.position !== undefined && (obj.position = Math.round(message.position));
    message.default !== undefined && (obj.default = message.default);
    message.nullable !== undefined && (obj.nullable = message.nullable);
    message.type !== undefined && (obj.type = message.type);
    message.characterSet !== undefined && (obj.characterSet = message.characterSet);
    message.collation !== undefined && (obj.collation = message.collation);
    message.comment !== undefined && (obj.comment = message.comment);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ColumnMetadata>, I>>(object: I): ColumnMetadata {
    const message = createBaseColumnMetadata();
    message.name = object.name ?? "";
    message.position = object.position ?? 0;
    message.default = object.default ?? undefined;
    message.nullable = object.nullable ?? false;
    message.type = object.type ?? "";
    message.characterSet = object.characterSet ?? "";
    message.collation = object.collation ?? "";
    message.comment = object.comment ?? "";
    return message;
  },
};

function createBaseViewMetadata(): ViewMetadata {
  return { name: "", definition: "", comment: "" };
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
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ViewMetadata {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseViewMetadata();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.definition = reader.string();
          break;
        case 3:
          message.comment = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ViewMetadata {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      definition: isSet(object.definition) ? String(object.definition) : "",
      comment: isSet(object.comment) ? String(object.comment) : "",
    };
  },

  toJSON(message: ViewMetadata): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.definition !== undefined && (obj.definition = message.definition);
    message.comment !== undefined && (obj.comment = message.comment);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ViewMetadata>, I>>(object: I): ViewMetadata {
    const message = createBaseViewMetadata();
    message.name = object.name ?? "";
    message.definition = object.definition ?? "";
    message.comment = object.comment ?? "";
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
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIndexMetadata();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.expressions.push(reader.string());
          break;
        case 3:
          message.type = reader.string();
          break;
        case 4:
          message.unique = reader.bool();
          break;
        case 5:
          message.primary = reader.bool();
          break;
        case 6:
          message.visible = reader.bool();
          break;
        case 7:
          message.comment = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): IndexMetadata {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      expressions: Array.isArray(object?.expressions) ? object.expressions.map((e: any) => String(e)) : [],
      type: isSet(object.type) ? String(object.type) : "",
      unique: isSet(object.unique) ? Boolean(object.unique) : false,
      primary: isSet(object.primary) ? Boolean(object.primary) : false,
      visible: isSet(object.visible) ? Boolean(object.visible) : false,
      comment: isSet(object.comment) ? String(object.comment) : "",
    };
  },

  toJSON(message: IndexMetadata): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    if (message.expressions) {
      obj.expressions = message.expressions.map((e) => e);
    } else {
      obj.expressions = [];
    }
    message.type !== undefined && (obj.type = message.type);
    message.unique !== undefined && (obj.unique = message.unique);
    message.primary !== undefined && (obj.primary = message.primary);
    message.visible !== undefined && (obj.visible = message.visible);
    message.comment !== undefined && (obj.comment = message.comment);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<IndexMetadata>, I>>(object: I): IndexMetadata {
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
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExtensionMetadata();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.schema = reader.string();
          break;
        case 3:
          message.version = reader.string();
          break;
        case 4:
          message.description = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ExtensionMetadata {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      schema: isSet(object.schema) ? String(object.schema) : "",
      version: isSet(object.version) ? String(object.version) : "",
      description: isSet(object.description) ? String(object.description) : "",
    };
  },

  toJSON(message: ExtensionMetadata): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.schema !== undefined && (obj.schema = message.schema);
    message.version !== undefined && (obj.version = message.version);
    message.description !== undefined && (obj.description = message.description);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ExtensionMetadata>, I>>(object: I): ExtensionMetadata {
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
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseForeignKeyMetadata();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.columns.push(reader.string());
          break;
        case 3:
          message.referencedSchema = reader.string();
          break;
        case 4:
          message.referencedTable = reader.string();
          break;
        case 5:
          message.referencedColumns.push(reader.string());
          break;
        case 6:
          message.onDelete = reader.string();
          break;
        case 7:
          message.onUpdate = reader.string();
          break;
        case 8:
          message.matchType = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ForeignKeyMetadata {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      columns: Array.isArray(object?.columns) ? object.columns.map((e: any) => String(e)) : [],
      referencedSchema: isSet(object.referencedSchema) ? String(object.referencedSchema) : "",
      referencedTable: isSet(object.referencedTable) ? String(object.referencedTable) : "",
      referencedColumns: Array.isArray(object?.referencedColumns)
        ? object.referencedColumns.map((e: any) => String(e))
        : [],
      onDelete: isSet(object.onDelete) ? String(object.onDelete) : "",
      onUpdate: isSet(object.onUpdate) ? String(object.onUpdate) : "",
      matchType: isSet(object.matchType) ? String(object.matchType) : "",
    };
  },

  toJSON(message: ForeignKeyMetadata): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    if (message.columns) {
      obj.columns = message.columns.map((e) => e);
    } else {
      obj.columns = [];
    }
    message.referencedSchema !== undefined && (obj.referencedSchema = message.referencedSchema);
    message.referencedTable !== undefined && (obj.referencedTable = message.referencedTable);
    if (message.referencedColumns) {
      obj.referencedColumns = message.referencedColumns.map((e) => e);
    } else {
      obj.referencedColumns = [];
    }
    message.onDelete !== undefined && (obj.onDelete = message.onDelete);
    message.onUpdate !== undefined && (obj.onUpdate = message.onUpdate);
    message.matchType !== undefined && (obj.matchType = message.matchType);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ForeignKeyMetadata>, I>>(object: I): ForeignKeyMetadata {
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

function createBaseDatabaseSchema(): DatabaseSchema {
  return { schema: "" };
}

export const DatabaseSchema = {
  encode(message: DatabaseSchema, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schema !== "") {
      writer.uint32(10).string(message.schema);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DatabaseSchema {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDatabaseSchema();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.schema = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DatabaseSchema {
    return { schema: isSet(object.schema) ? String(object.schema) : "" };
  },

  toJSON(message: DatabaseSchema): unknown {
    const obj: any = {};
    message.schema !== undefined && (obj.schema = message.schema);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<DatabaseSchema>, I>>(object: I): DatabaseSchema {
    const message = createBaseDatabaseSchema();
    message.schema = object.schema ?? "";
    return message;
  },
};

export interface DatabaseService {
  ListDatabases(request: ListDatabasesRequest): Promise<ListDatabasesResponse>;
  UpdateDatabase(request: UpdateDatabaseRequest): Promise<Database>;
  BatchUpdateDatabases(request: BatchUpdateDatabasesRequest): Promise<BatchUpdateDatabasesResponse>;
  GetDatabaseMetadata(request: GetDatabaseMetadataRequest): Promise<DatabaseMetadata>;
  GetDatabaseSchema(request: GetDatabaseSchemaRequest): Promise<DatabaseSchema>;
}

export class DatabaseServiceClientImpl implements DatabaseService {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "bytebase.v1.DatabaseService";
    this.rpc = rpc;
    this.ListDatabases = this.ListDatabases.bind(this);
    this.UpdateDatabase = this.UpdateDatabase.bind(this);
    this.BatchUpdateDatabases = this.BatchUpdateDatabases.bind(this);
    this.GetDatabaseMetadata = this.GetDatabaseMetadata.bind(this);
    this.GetDatabaseSchema = this.GetDatabaseSchema.bind(this);
  }
  ListDatabases(request: ListDatabasesRequest): Promise<ListDatabasesResponse> {
    const data = ListDatabasesRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "ListDatabases", data);
    return promise.then((data) => ListDatabasesResponse.decode(new _m0.Reader(data)));
  }

  UpdateDatabase(request: UpdateDatabaseRequest): Promise<Database> {
    const data = UpdateDatabaseRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "UpdateDatabase", data);
    return promise.then((data) => Database.decode(new _m0.Reader(data)));
  }

  BatchUpdateDatabases(request: BatchUpdateDatabasesRequest): Promise<BatchUpdateDatabasesResponse> {
    const data = BatchUpdateDatabasesRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "BatchUpdateDatabases", data);
    return promise.then((data) => BatchUpdateDatabasesResponse.decode(new _m0.Reader(data)));
  }

  GetDatabaseMetadata(request: GetDatabaseMetadataRequest): Promise<DatabaseMetadata> {
    const data = GetDatabaseMetadataRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "GetDatabaseMetadata", data);
    return promise.then((data) => DatabaseMetadata.decode(new _m0.Reader(data)));
  }

  GetDatabaseSchema(request: GetDatabaseSchemaRequest): Promise<DatabaseSchema> {
    const data = GetDatabaseSchemaRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "GetDatabaseSchema", data);
    return promise.then((data) => DatabaseSchema.decode(new _m0.Reader(data)));
  }
}

interface Rpc {
  request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}

declare var self: any | undefined;
declare var window: any | undefined;
declare var global: any | undefined;
var globalThis: any = (() => {
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

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

type KeysOfUnion<T> = T extends T ? keyof T : never;
export type Exact<P, I extends P> = P extends Builtin ? P
  : P & { [K in keyof P]: Exact<P[K], I[K]> } & { [K in Exclude<keyof I, KeysOfUnion<P>>]: never };

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

function longToNumber(long: Long): number {
  if (long.gt(Number.MAX_SAFE_INTEGER)) {
    throw new globalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
  }
  return long.toNumber();
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
