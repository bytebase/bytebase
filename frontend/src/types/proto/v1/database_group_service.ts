/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { Expr } from "../google/type/expr";

export const protobufPackage = "bytebase.v1";

export enum DatabaseGroupView {
  /**
   * DATABASE_GROUP_VIEW_UNSPECIFIED - The default / unset value.
   * The API will default to the BASIC view.
   */
  DATABASE_GROUP_VIEW_UNSPECIFIED = "DATABASE_GROUP_VIEW_UNSPECIFIED",
  /** DATABASE_GROUP_VIEW_BASIC - Include basic information about the database group, but exclude the list of matched databases and unmatched databases. */
  DATABASE_GROUP_VIEW_BASIC = "DATABASE_GROUP_VIEW_BASIC",
  /** DATABASE_GROUP_VIEW_FULL - Include everything. */
  DATABASE_GROUP_VIEW_FULL = "DATABASE_GROUP_VIEW_FULL",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function databaseGroupViewFromJSON(object: any): DatabaseGroupView {
  switch (object) {
    case 0:
    case "DATABASE_GROUP_VIEW_UNSPECIFIED":
      return DatabaseGroupView.DATABASE_GROUP_VIEW_UNSPECIFIED;
    case 1:
    case "DATABASE_GROUP_VIEW_BASIC":
      return DatabaseGroupView.DATABASE_GROUP_VIEW_BASIC;
    case 2:
    case "DATABASE_GROUP_VIEW_FULL":
      return DatabaseGroupView.DATABASE_GROUP_VIEW_FULL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return DatabaseGroupView.UNRECOGNIZED;
  }
}

export function databaseGroupViewToJSON(object: DatabaseGroupView): string {
  switch (object) {
    case DatabaseGroupView.DATABASE_GROUP_VIEW_UNSPECIFIED:
      return "DATABASE_GROUP_VIEW_UNSPECIFIED";
    case DatabaseGroupView.DATABASE_GROUP_VIEW_BASIC:
      return "DATABASE_GROUP_VIEW_BASIC";
    case DatabaseGroupView.DATABASE_GROUP_VIEW_FULL:
      return "DATABASE_GROUP_VIEW_FULL";
    case DatabaseGroupView.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function databaseGroupViewToNumber(object: DatabaseGroupView): number {
  switch (object) {
    case DatabaseGroupView.DATABASE_GROUP_VIEW_UNSPECIFIED:
      return 0;
    case DatabaseGroupView.DATABASE_GROUP_VIEW_BASIC:
      return 1;
    case DatabaseGroupView.DATABASE_GROUP_VIEW_FULL:
      return 2;
    case DatabaseGroupView.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface ListDatabaseGroupsRequest {
  /**
   * The parent resource whose database groups are to be listed.
   * Format: projects/{project}
   */
  parent: string;
  /**
   * Not used. The maximum number of anomalies to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 anomalies will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * Not used. A page token, received from a previous `ListDatabaseGroups` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListDatabaseGroups` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface ListDatabaseGroupsResponse {
  /** database_groups is the list of database groups. */
  databaseGroups: DatabaseGroup[];
  /**
   * Not used. A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface GetDatabaseGroupRequest {
  /**
   * The name of the database group to retrieve.
   * Format: projects/{project}/databaseGroups/{databaseGroup}
   */
  name: string;
  /** The view to return. Defaults to DATABASE_GROUP_VIEW_BASIC. */
  view: DatabaseGroupView;
}

export interface CreateDatabaseGroupRequest {
  /**
   * The parent resource where this database group will be created.
   * Format: projects/{project}
   */
  parent: string;
  /** The database group to create. */
  databaseGroup:
    | DatabaseGroup
    | undefined;
  /**
   * The ID to use for the database group, which will become the final component of
   * the database group's resource name.
   *
   * This value should be 4-63 characters, and valid characters
   * are /[a-z][0-9]-/.
   */
  databaseGroupId: string;
  /** If set, validate the create request and preview the full database group response, but do not actually create it. */
  validateOnly: boolean;
}

export interface UpdateDatabaseGroupRequest {
  /**
   * The database group to update.
   *
   * The database group's `name` field is used to identify the database group to update.
   * Format: projects/{project}/databaseGroups/{databaseGroup}
   */
  databaseGroup:
    | DatabaseGroup
    | undefined;
  /** The list of fields to update. */
  updateMask: string[] | undefined;
}

export interface DeleteDatabaseGroupRequest {
  /**
   * The name of the database group to delete.
   * Format: projects/{project}/databaseGroups/{databaseGroup}
   */
  name: string;
}

export interface DatabaseGroup {
  /**
   * The name of the database group.
   * Format: projects/{project}/databaseGroups/{databaseGroup}
   */
  name: string;
  /**
   * The short name used in actual databases specified by users.
   * For example, the placeholder for db1_2010, db1_2021, db1_2023 will be "db1".
   */
  databasePlaceholder: string;
  /** The condition that is associated with this database group. */
  databaseExpr:
    | Expr
    | undefined;
  /** The list of databases that match the database group condition. */
  matchedDatabases: DatabaseGroup_Database[];
  /** The list of databases that match the database group condition. */
  unmatchedDatabases: DatabaseGroup_Database[];
  multitenancy: boolean;
}

export interface DatabaseGroup_Database {
  /**
   * The resource name of the database.
   * Format: instances/{instance}/databases/{database}
   */
  name: string;
}

function createBaseListDatabaseGroupsRequest(): ListDatabaseGroupsRequest {
  return { parent: "", pageSize: 0, pageToken: "" };
}

export const ListDatabaseGroupsRequest = {
  encode(message: ListDatabaseGroupsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.pageSize !== 0) {
      writer.uint32(16).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(26).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListDatabaseGroupsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListDatabaseGroupsRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.parent = reader.string();
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

  fromJSON(object: any): ListDatabaseGroupsRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
    };
  },

  toJSON(message: ListDatabaseGroupsRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListDatabaseGroupsRequest>): ListDatabaseGroupsRequest {
    return ListDatabaseGroupsRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListDatabaseGroupsRequest>): ListDatabaseGroupsRequest {
    const message = createBaseListDatabaseGroupsRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListDatabaseGroupsResponse(): ListDatabaseGroupsResponse {
  return { databaseGroups: [], nextPageToken: "" };
}

export const ListDatabaseGroupsResponse = {
  encode(message: ListDatabaseGroupsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.databaseGroups) {
      DatabaseGroup.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListDatabaseGroupsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListDatabaseGroupsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.databaseGroups.push(DatabaseGroup.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListDatabaseGroupsResponse {
    return {
      databaseGroups: globalThis.Array.isArray(object?.databaseGroups)
        ? object.databaseGroups.map((e: any) => DatabaseGroup.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListDatabaseGroupsResponse): unknown {
    const obj: any = {};
    if (message.databaseGroups?.length) {
      obj.databaseGroups = message.databaseGroups.map((e) => DatabaseGroup.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListDatabaseGroupsResponse>): ListDatabaseGroupsResponse {
    return ListDatabaseGroupsResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListDatabaseGroupsResponse>): ListDatabaseGroupsResponse {
    const message = createBaseListDatabaseGroupsResponse();
    message.databaseGroups = object.databaseGroups?.map((e) => DatabaseGroup.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseGetDatabaseGroupRequest(): GetDatabaseGroupRequest {
  return { name: "", view: DatabaseGroupView.DATABASE_GROUP_VIEW_UNSPECIFIED };
}

export const GetDatabaseGroupRequest = {
  encode(message: GetDatabaseGroupRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.view !== DatabaseGroupView.DATABASE_GROUP_VIEW_UNSPECIFIED) {
      writer.uint32(16).int32(databaseGroupViewToNumber(message.view));
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetDatabaseGroupRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetDatabaseGroupRequest();
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

          message.view = databaseGroupViewFromJSON(reader.int32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GetDatabaseGroupRequest {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      view: isSet(object.view)
        ? databaseGroupViewFromJSON(object.view)
        : DatabaseGroupView.DATABASE_GROUP_VIEW_UNSPECIFIED,
    };
  },

  toJSON(message: GetDatabaseGroupRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.view !== DatabaseGroupView.DATABASE_GROUP_VIEW_UNSPECIFIED) {
      obj.view = databaseGroupViewToJSON(message.view);
    }
    return obj;
  },

  create(base?: DeepPartial<GetDatabaseGroupRequest>): GetDatabaseGroupRequest {
    return GetDatabaseGroupRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<GetDatabaseGroupRequest>): GetDatabaseGroupRequest {
    const message = createBaseGetDatabaseGroupRequest();
    message.name = object.name ?? "";
    message.view = object.view ?? DatabaseGroupView.DATABASE_GROUP_VIEW_UNSPECIFIED;
    return message;
  },
};

function createBaseCreateDatabaseGroupRequest(): CreateDatabaseGroupRequest {
  return { parent: "", databaseGroup: undefined, databaseGroupId: "", validateOnly: false };
}

export const CreateDatabaseGroupRequest = {
  encode(message: CreateDatabaseGroupRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.databaseGroup !== undefined) {
      DatabaseGroup.encode(message.databaseGroup, writer.uint32(18).fork()).ldelim();
    }
    if (message.databaseGroupId !== "") {
      writer.uint32(26).string(message.databaseGroupId);
    }
    if (message.validateOnly === true) {
      writer.uint32(32).bool(message.validateOnly);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateDatabaseGroupRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateDatabaseGroupRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.parent = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.databaseGroup = DatabaseGroup.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.databaseGroupId = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.validateOnly = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateDatabaseGroupRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      databaseGroup: isSet(object.databaseGroup) ? DatabaseGroup.fromJSON(object.databaseGroup) : undefined,
      databaseGroupId: isSet(object.databaseGroupId) ? globalThis.String(object.databaseGroupId) : "",
      validateOnly: isSet(object.validateOnly) ? globalThis.Boolean(object.validateOnly) : false,
    };
  },

  toJSON(message: CreateDatabaseGroupRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.databaseGroup !== undefined) {
      obj.databaseGroup = DatabaseGroup.toJSON(message.databaseGroup);
    }
    if (message.databaseGroupId !== "") {
      obj.databaseGroupId = message.databaseGroupId;
    }
    if (message.validateOnly === true) {
      obj.validateOnly = message.validateOnly;
    }
    return obj;
  },

  create(base?: DeepPartial<CreateDatabaseGroupRequest>): CreateDatabaseGroupRequest {
    return CreateDatabaseGroupRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<CreateDatabaseGroupRequest>): CreateDatabaseGroupRequest {
    const message = createBaseCreateDatabaseGroupRequest();
    message.parent = object.parent ?? "";
    message.databaseGroup = (object.databaseGroup !== undefined && object.databaseGroup !== null)
      ? DatabaseGroup.fromPartial(object.databaseGroup)
      : undefined;
    message.databaseGroupId = object.databaseGroupId ?? "";
    message.validateOnly = object.validateOnly ?? false;
    return message;
  },
};

function createBaseUpdateDatabaseGroupRequest(): UpdateDatabaseGroupRequest {
  return { databaseGroup: undefined, updateMask: undefined };
}

export const UpdateDatabaseGroupRequest = {
  encode(message: UpdateDatabaseGroupRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.databaseGroup !== undefined) {
      DatabaseGroup.encode(message.databaseGroup, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateDatabaseGroupRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateDatabaseGroupRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.databaseGroup = DatabaseGroup.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UpdateDatabaseGroupRequest {
    return {
      databaseGroup: isSet(object.databaseGroup) ? DatabaseGroup.fromJSON(object.databaseGroup) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateDatabaseGroupRequest): unknown {
    const obj: any = {};
    if (message.databaseGroup !== undefined) {
      obj.databaseGroup = DatabaseGroup.toJSON(message.databaseGroup);
    }
    if (message.updateMask !== undefined) {
      obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask));
    }
    return obj;
  },

  create(base?: DeepPartial<UpdateDatabaseGroupRequest>): UpdateDatabaseGroupRequest {
    return UpdateDatabaseGroupRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UpdateDatabaseGroupRequest>): UpdateDatabaseGroupRequest {
    const message = createBaseUpdateDatabaseGroupRequest();
    message.databaseGroup = (object.databaseGroup !== undefined && object.databaseGroup !== null)
      ? DatabaseGroup.fromPartial(object.databaseGroup)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseDeleteDatabaseGroupRequest(): DeleteDatabaseGroupRequest {
  return { name: "" };
}

export const DeleteDatabaseGroupRequest = {
  encode(message: DeleteDatabaseGroupRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteDatabaseGroupRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteDatabaseGroupRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DeleteDatabaseGroupRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: DeleteDatabaseGroupRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<DeleteDatabaseGroupRequest>): DeleteDatabaseGroupRequest {
    return DeleteDatabaseGroupRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DeleteDatabaseGroupRequest>): DeleteDatabaseGroupRequest {
    const message = createBaseDeleteDatabaseGroupRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseDatabaseGroup(): DatabaseGroup {
  return {
    name: "",
    databasePlaceholder: "",
    databaseExpr: undefined,
    matchedDatabases: [],
    unmatchedDatabases: [],
    multitenancy: false,
  };
}

export const DatabaseGroup = {
  encode(message: DatabaseGroup, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.databasePlaceholder !== "") {
      writer.uint32(18).string(message.databasePlaceholder);
    }
    if (message.databaseExpr !== undefined) {
      Expr.encode(message.databaseExpr, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.matchedDatabases) {
      DatabaseGroup_Database.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.unmatchedDatabases) {
      DatabaseGroup_Database.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    if (message.multitenancy === true) {
      writer.uint32(48).bool(message.multitenancy);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DatabaseGroup {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDatabaseGroup();
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

          message.databasePlaceholder = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.databaseExpr = Expr.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.matchedDatabases.push(DatabaseGroup_Database.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.unmatchedDatabases.push(DatabaseGroup_Database.decode(reader, reader.uint32()));
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.multitenancy = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DatabaseGroup {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      databasePlaceholder: isSet(object.databasePlaceholder) ? globalThis.String(object.databasePlaceholder) : "",
      databaseExpr: isSet(object.databaseExpr) ? Expr.fromJSON(object.databaseExpr) : undefined,
      matchedDatabases: globalThis.Array.isArray(object?.matchedDatabases)
        ? object.matchedDatabases.map((e: any) => DatabaseGroup_Database.fromJSON(e))
        : [],
      unmatchedDatabases: globalThis.Array.isArray(object?.unmatchedDatabases)
        ? object.unmatchedDatabases.map((e: any) => DatabaseGroup_Database.fromJSON(e))
        : [],
      multitenancy: isSet(object.multitenancy) ? globalThis.Boolean(object.multitenancy) : false,
    };
  },

  toJSON(message: DatabaseGroup): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.databasePlaceholder !== "") {
      obj.databasePlaceholder = message.databasePlaceholder;
    }
    if (message.databaseExpr !== undefined) {
      obj.databaseExpr = Expr.toJSON(message.databaseExpr);
    }
    if (message.matchedDatabases?.length) {
      obj.matchedDatabases = message.matchedDatabases.map((e) => DatabaseGroup_Database.toJSON(e));
    }
    if (message.unmatchedDatabases?.length) {
      obj.unmatchedDatabases = message.unmatchedDatabases.map((e) => DatabaseGroup_Database.toJSON(e));
    }
    if (message.multitenancy === true) {
      obj.multitenancy = message.multitenancy;
    }
    return obj;
  },

  create(base?: DeepPartial<DatabaseGroup>): DatabaseGroup {
    return DatabaseGroup.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DatabaseGroup>): DatabaseGroup {
    const message = createBaseDatabaseGroup();
    message.name = object.name ?? "";
    message.databasePlaceholder = object.databasePlaceholder ?? "";
    message.databaseExpr = (object.databaseExpr !== undefined && object.databaseExpr !== null)
      ? Expr.fromPartial(object.databaseExpr)
      : undefined;
    message.matchedDatabases = object.matchedDatabases?.map((e) => DatabaseGroup_Database.fromPartial(e)) || [];
    message.unmatchedDatabases = object.unmatchedDatabases?.map((e) => DatabaseGroup_Database.fromPartial(e)) || [];
    message.multitenancy = object.multitenancy ?? false;
    return message;
  },
};

function createBaseDatabaseGroup_Database(): DatabaseGroup_Database {
  return { name: "" };
}

export const DatabaseGroup_Database = {
  encode(message: DatabaseGroup_Database, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DatabaseGroup_Database {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDatabaseGroup_Database();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DatabaseGroup_Database {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: DatabaseGroup_Database): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<DatabaseGroup_Database>): DatabaseGroup_Database {
    return DatabaseGroup_Database.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DatabaseGroup_Database>): DatabaseGroup_Database {
    const message = createBaseDatabaseGroup_Database();
    message.name = object.name ?? "";
    return message;
  },
};

export type DatabaseGroupServiceDefinition = typeof DatabaseGroupServiceDefinition;
export const DatabaseGroupServiceDefinition = {
  name: "DatabaseGroupService",
  fullName: "bytebase.v1.DatabaseGroupService",
  methods: {
    listDatabaseGroups: {
      name: "ListDatabaseGroups",
      requestType: ListDatabaseGroupsRequest,
      requestStream: false,
      responseType: ListDatabaseGroupsResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([6, 112, 97, 114, 101, 110, 116])],
          800010: [new Uint8Array([15, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 103, 101, 116])],
          578365826: [
            new Uint8Array([
              40,
              18,
              38,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
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
              125,
              47,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              71,
              114,
              111,
              117,
              112,
              115,
            ]),
          ],
        },
      },
    },
    getDatabaseGroup: {
      name: "GetDatabaseGroup",
      requestType: GetDatabaseGroupRequest,
      requestStream: false,
      responseType: DatabaseGroup,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          800010: [new Uint8Array([15, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 103, 101, 116])],
          578365826: [
            new Uint8Array([
              40,
              18,
              38,
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
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              71,
              114,
              111,
              117,
              112,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    createDatabaseGroup: {
      name: "CreateDatabaseGroup",
      requestType: CreateDatabaseGroupRequest,
      requestStream: false,
      responseType: DatabaseGroup,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([
              20,
              112,
              97,
              114,
              101,
              110,
              116,
              44,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              71,
              114,
              111,
              117,
              112,
            ]),
          ],
          800010: [
            new Uint8Array([18, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 117, 112, 100, 97, 116, 101]),
          ],
          578365826: [
            new Uint8Array([
              56,
              58,
              14,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              95,
              103,
              114,
              111,
              117,
              112,
              34,
              38,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
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
              125,
              47,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              71,
              114,
              111,
              117,
              112,
              115,
            ]),
          ],
        },
      },
    },
    updateDatabaseGroup: {
      name: "UpdateDatabaseGroup",
      requestType: UpdateDatabaseGroupRequest,
      requestStream: false,
      responseType: DatabaseGroup,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([
              26,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              95,
              103,
              114,
              111,
              117,
              112,
              44,
              117,
              112,
              100,
              97,
              116,
              101,
              95,
              109,
              97,
              115,
              107,
            ]),
          ],
          800010: [
            new Uint8Array([18, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 117, 112, 100, 97, 116, 101]),
          ],
          578365826: [
            new Uint8Array([
              71,
              58,
              14,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              95,
              103,
              114,
              111,
              117,
              112,
              50,
              53,
              47,
              118,
              49,
              47,
              123,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              95,
              103,
              114,
              111,
              117,
              112,
              46,
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
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              71,
              114,
              111,
              117,
              112,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    deleteDatabaseGroup: {
      name: "DeleteDatabaseGroup",
      requestType: DeleteDatabaseGroupRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          800010: [
            new Uint8Array([18, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 117, 112, 100, 97, 116, 101]),
          ],
          578365826: [
            new Uint8Array([
              40,
              42,
              38,
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
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              71,
              114,
              111,
              117,
              112,
              115,
              47,
              42,
              125,
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

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
