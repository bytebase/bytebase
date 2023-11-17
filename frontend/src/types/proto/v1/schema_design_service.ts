/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { Timestamp } from "../google/protobuf/timestamp";
import { Engine, engineFromJSON, engineToJSON } from "./common";
import { DatabaseMetadata } from "./database_service";

export const protobufPackage = "bytebase.v1";

export enum SchemaDesignView {
  /**
   * SCHEMA_DESIGN_VIEW_UNSPECIFIED - The default / unset value.
   * The API will default to the BASIC view.
   */
  SCHEMA_DESIGN_VIEW_UNSPECIFIED = 0,
  /** SCHEMA_DESIGN_VIEW_BASIC - Exclude schema, baseline_schema. */
  SCHEMA_DESIGN_VIEW_BASIC = 1,
  /** SCHEMA_DESIGN_VIEW_FULL - Include everything. */
  SCHEMA_DESIGN_VIEW_FULL = 2,
  UNRECOGNIZED = -1,
}

export function schemaDesignViewFromJSON(object: any): SchemaDesignView {
  switch (object) {
    case 0:
    case "SCHEMA_DESIGN_VIEW_UNSPECIFIED":
      return SchemaDesignView.SCHEMA_DESIGN_VIEW_UNSPECIFIED;
    case 1:
    case "SCHEMA_DESIGN_VIEW_BASIC":
      return SchemaDesignView.SCHEMA_DESIGN_VIEW_BASIC;
    case 2:
    case "SCHEMA_DESIGN_VIEW_FULL":
      return SchemaDesignView.SCHEMA_DESIGN_VIEW_FULL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return SchemaDesignView.UNRECOGNIZED;
  }
}

export function schemaDesignViewToJSON(object: SchemaDesignView): string {
  switch (object) {
    case SchemaDesignView.SCHEMA_DESIGN_VIEW_UNSPECIFIED:
      return "SCHEMA_DESIGN_VIEW_UNSPECIFIED";
    case SchemaDesignView.SCHEMA_DESIGN_VIEW_BASIC:
      return "SCHEMA_DESIGN_VIEW_BASIC";
    case SchemaDesignView.SCHEMA_DESIGN_VIEW_FULL:
      return "SCHEMA_DESIGN_VIEW_FULL";
    case SchemaDesignView.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface SchemaDesign {
  /**
   * The name of the schema design.
   * Format: projects/{project}/schemaDesigns/{schemaDesign}
   * {schemaDesign} should be the id of a sheet.
   */
  name: string;
  /** The title of schema design. AKA sheet's name. */
  title: string;
  /** The schema of schema design. AKA sheet's statement. */
  schema: string;
  /** The metadata of the current editing schema. */
  schemaMetadata:
    | DatabaseMetadata
    | undefined;
  /** The baseline schema. */
  baselineSchema: string;
  /** The metadata of the baseline schema. */
  baselineSchemaMetadata:
    | DatabaseMetadata
    | undefined;
  /** The database engine of the schema design. */
  engine: Engine;
  /**
   * The name of the baseline database.
   * Format: instances/{instance}/databases/{database}
   */
  baselineDatabase: string;
  /**
   * The name of the baseline sheet.
   * For main branch, its format will be: projects/{project}/sheets/{sheet}
   * For personal draft, its format will be: projects/{project}/schemaDesigns/{schemaDesign}
   */
  baselineSheetName: string;
  /** The baseline change history id. */
  baselineChangeHistoryId?:
    | string
    | undefined;
  /** The type of the schema design. */
  type: SchemaDesign_Type;
  /** The etag of the schema design. */
  etag: string;
  /** The protection of the schema design branch. */
  protection:
    | SchemaDesign_Protection
    | undefined;
  /**
   * The creator of the schema design.
   * Format: users/{email}
   */
  creator: string;
  /**
   * The updater of the schema design.
   * Format: users/{email}
   */
  updater: string;
  /** The timestamp when the schema design was created. */
  createTime:
    | Date
    | undefined;
  /** The timestamp when the schema design was last updated. */
  updateTime: Date | undefined;
}

export enum SchemaDesign_Type {
  TYPE_UNSPECIFIED = 0,
  /** MAIN_BRANCH - Main branch type is the main version of schema design. And only allow to be updated/merged with personal drafts. */
  MAIN_BRANCH = 1,
  /** PERSONAL_DRAFT - Personal draft type is a copy of the main branch type schema designs. */
  PERSONAL_DRAFT = 2,
  UNRECOGNIZED = -1,
}

export function schemaDesign_TypeFromJSON(object: any): SchemaDesign_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return SchemaDesign_Type.TYPE_UNSPECIFIED;
    case 1:
    case "MAIN_BRANCH":
      return SchemaDesign_Type.MAIN_BRANCH;
    case 2:
    case "PERSONAL_DRAFT":
      return SchemaDesign_Type.PERSONAL_DRAFT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return SchemaDesign_Type.UNRECOGNIZED;
  }
}

export function schemaDesign_TypeToJSON(object: SchemaDesign_Type): string {
  switch (object) {
    case SchemaDesign_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case SchemaDesign_Type.MAIN_BRANCH:
      return "MAIN_BRANCH";
    case SchemaDesign_Type.PERSONAL_DRAFT:
      return "PERSONAL_DRAFT";
    case SchemaDesign_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface SchemaDesign_Protection {
  /** Permits force pushes to the branch. */
  allowForcePushes: boolean;
}

export interface GetSchemaDesignRequest {
  /**
   * The name of the schema design to retrieve.
   * Format: projects/{project}/schemaDesigns/{schemaDesign}
   */
  name: string;
}

export interface ListSchemaDesignsRequest {
  /**
   * The parent resource of the schema design.
   * Foramt: projects/{project}
   */
  parent: string;
  /** To filter the search result. */
  filter: string;
  /**
   * The maximum number of schema designs to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 schema designs will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListSchemaDesigns` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListSchemaDesigns` must match
   * the call that provided the page token.
   */
  pageToken: string;
  view: SchemaDesignView;
}

export interface ListSchemaDesignsResponse {
  /** The schema designs from the specified request. */
  schemaDesigns: SchemaDesign[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface CreateSchemaDesignRequest {
  /**
   * The parent, which owns this collection of schema designs.
   * Format: project/{project}
   */
  parent: string;
  schemaDesign: SchemaDesign | undefined;
}

export interface UpdateSchemaDesignRequest {
  /**
   * The schema design to update.
   *
   * The schema design's `name` field is used to identify the schema design to update.
   * Format: projects/{project}/schemaDesigns/{schemaDesign}
   */
  schemaDesign:
    | SchemaDesign
    | undefined;
  /** The list of fields to update. */
  updateMask: string[] | undefined;
}

export interface MergeSchemaDesignRequest {
  /**
   * The name of the schema design to merge.
   * Format: projects/{project}/schemaDesigns/{schemaDesign}
   */
  name: string;
  /**
   * The target schema design to merge into.
   * Format: projects/{project}/schemaDesigns/{schemaDesign}
   */
  targetName: string;
}

export interface ParseSchemaStringRequest {
  /** The schema string to parse. */
  schemaString: string;
  /** The database engine of the schema string. */
  engine: Engine;
}

export interface ParseSchemaStringResponse {
  /** The metadata of the parsed schema. */
  schemaMetadata: DatabaseMetadata | undefined;
}

export interface DeleteSchemaDesignRequest {
  /**
   * The name of the schema design to delete.
   * Format: projects/{project}/schemaDesigns/{schemaDesign}
   */
  name: string;
}

export interface DiffMetadataRequest {
  /** The metadata of the source schema. */
  sourceMetadata:
    | DatabaseMetadata
    | undefined;
  /** The metadata of the target schema. */
  targetMetadata:
    | DatabaseMetadata
    | undefined;
  /** The database engine of the schema. */
  engine: Engine;
}

export interface DiffMetadataResponse {
  /** The diff of the metadata. */
  diff: string;
}

function createBaseSchemaDesign(): SchemaDesign {
  return {
    name: "",
    title: "",
    schema: "",
    schemaMetadata: undefined,
    baselineSchema: "",
    baselineSchemaMetadata: undefined,
    engine: 0,
    baselineDatabase: "",
    baselineSheetName: "",
    baselineChangeHistoryId: undefined,
    type: 0,
    etag: "",
    protection: undefined,
    creator: "",
    updater: "",
    createTime: undefined,
    updateTime: undefined,
  };
}

export const SchemaDesign = {
  encode(message: SchemaDesign, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.schema !== "") {
      writer.uint32(26).string(message.schema);
    }
    if (message.schemaMetadata !== undefined) {
      DatabaseMetadata.encode(message.schemaMetadata, writer.uint32(34).fork()).ldelim();
    }
    if (message.baselineSchema !== "") {
      writer.uint32(42).string(message.baselineSchema);
    }
    if (message.baselineSchemaMetadata !== undefined) {
      DatabaseMetadata.encode(message.baselineSchemaMetadata, writer.uint32(50).fork()).ldelim();
    }
    if (message.engine !== 0) {
      writer.uint32(56).int32(message.engine);
    }
    if (message.baselineDatabase !== "") {
      writer.uint32(66).string(message.baselineDatabase);
    }
    if (message.baselineSheetName !== "") {
      writer.uint32(74).string(message.baselineSheetName);
    }
    if (message.baselineChangeHistoryId !== undefined) {
      writer.uint32(82).string(message.baselineChangeHistoryId);
    }
    if (message.type !== 0) {
      writer.uint32(88).int32(message.type);
    }
    if (message.etag !== "") {
      writer.uint32(98).string(message.etag);
    }
    if (message.protection !== undefined) {
      SchemaDesign_Protection.encode(message.protection, writer.uint32(106).fork()).ldelim();
    }
    if (message.creator !== "") {
      writer.uint32(114).string(message.creator);
    }
    if (message.updater !== "") {
      writer.uint32(122).string(message.updater);
    }
    if (message.createTime !== undefined) {
      Timestamp.encode(toTimestamp(message.createTime), writer.uint32(130).fork()).ldelim();
    }
    if (message.updateTime !== undefined) {
      Timestamp.encode(toTimestamp(message.updateTime), writer.uint32(138).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SchemaDesign {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSchemaDesign();
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

          message.title = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.schema = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.schemaMetadata = DatabaseMetadata.decode(reader, reader.uint32());
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.baselineSchema = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.baselineSchemaMetadata = DatabaseMetadata.decode(reader, reader.uint32());
          continue;
        case 7:
          if (tag !== 56) {
            break;
          }

          message.engine = reader.int32() as any;
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.baselineDatabase = reader.string();
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.baselineSheetName = reader.string();
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.baselineChangeHistoryId = reader.string();
          continue;
        case 11:
          if (tag !== 88) {
            break;
          }

          message.type = reader.int32() as any;
          continue;
        case 12:
          if (tag !== 98) {
            break;
          }

          message.etag = reader.string();
          continue;
        case 13:
          if (tag !== 106) {
            break;
          }

          message.protection = SchemaDesign_Protection.decode(reader, reader.uint32());
          continue;
        case 14:
          if (tag !== 114) {
            break;
          }

          message.creator = reader.string();
          continue;
        case 15:
          if (tag !== 122) {
            break;
          }

          message.updater = reader.string();
          continue;
        case 16:
          if (tag !== 130) {
            break;
          }

          message.createTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
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

  fromJSON(object: any): SchemaDesign {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      schema: isSet(object.schema) ? globalThis.String(object.schema) : "",
      schemaMetadata: isSet(object.schemaMetadata) ? DatabaseMetadata.fromJSON(object.schemaMetadata) : undefined,
      baselineSchema: isSet(object.baselineSchema) ? globalThis.String(object.baselineSchema) : "",
      baselineSchemaMetadata: isSet(object.baselineSchemaMetadata)
        ? DatabaseMetadata.fromJSON(object.baselineSchemaMetadata)
        : undefined,
      engine: isSet(object.engine) ? engineFromJSON(object.engine) : 0,
      baselineDatabase: isSet(object.baselineDatabase) ? globalThis.String(object.baselineDatabase) : "",
      baselineSheetName: isSet(object.baselineSheetName) ? globalThis.String(object.baselineSheetName) : "",
      baselineChangeHistoryId: isSet(object.baselineChangeHistoryId)
        ? globalThis.String(object.baselineChangeHistoryId)
        : undefined,
      type: isSet(object.type) ? schemaDesign_TypeFromJSON(object.type) : 0,
      etag: isSet(object.etag) ? globalThis.String(object.etag) : "",
      protection: isSet(object.protection) ? SchemaDesign_Protection.fromJSON(object.protection) : undefined,
      creator: isSet(object.creator) ? globalThis.String(object.creator) : "",
      updater: isSet(object.updater) ? globalThis.String(object.updater) : "",
      createTime: isSet(object.createTime) ? fromJsonTimestamp(object.createTime) : undefined,
      updateTime: isSet(object.updateTime) ? fromJsonTimestamp(object.updateTime) : undefined,
    };
  },

  toJSON(message: SchemaDesign): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.schema !== "") {
      obj.schema = message.schema;
    }
    if (message.schemaMetadata !== undefined) {
      obj.schemaMetadata = DatabaseMetadata.toJSON(message.schemaMetadata);
    }
    if (message.baselineSchema !== "") {
      obj.baselineSchema = message.baselineSchema;
    }
    if (message.baselineSchemaMetadata !== undefined) {
      obj.baselineSchemaMetadata = DatabaseMetadata.toJSON(message.baselineSchemaMetadata);
    }
    if (message.engine !== 0) {
      obj.engine = engineToJSON(message.engine);
    }
    if (message.baselineDatabase !== "") {
      obj.baselineDatabase = message.baselineDatabase;
    }
    if (message.baselineSheetName !== "") {
      obj.baselineSheetName = message.baselineSheetName;
    }
    if (message.baselineChangeHistoryId !== undefined) {
      obj.baselineChangeHistoryId = message.baselineChangeHistoryId;
    }
    if (message.type !== 0) {
      obj.type = schemaDesign_TypeToJSON(message.type);
    }
    if (message.etag !== "") {
      obj.etag = message.etag;
    }
    if (message.protection !== undefined) {
      obj.protection = SchemaDesign_Protection.toJSON(message.protection);
    }
    if (message.creator !== "") {
      obj.creator = message.creator;
    }
    if (message.updater !== "") {
      obj.updater = message.updater;
    }
    if (message.createTime !== undefined) {
      obj.createTime = message.createTime.toISOString();
    }
    if (message.updateTime !== undefined) {
      obj.updateTime = message.updateTime.toISOString();
    }
    return obj;
  },

  create(base?: DeepPartial<SchemaDesign>): SchemaDesign {
    return SchemaDesign.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SchemaDesign>): SchemaDesign {
    const message = createBaseSchemaDesign();
    message.name = object.name ?? "";
    message.title = object.title ?? "";
    message.schema = object.schema ?? "";
    message.schemaMetadata = (object.schemaMetadata !== undefined && object.schemaMetadata !== null)
      ? DatabaseMetadata.fromPartial(object.schemaMetadata)
      : undefined;
    message.baselineSchema = object.baselineSchema ?? "";
    message.baselineSchemaMetadata =
      (object.baselineSchemaMetadata !== undefined && object.baselineSchemaMetadata !== null)
        ? DatabaseMetadata.fromPartial(object.baselineSchemaMetadata)
        : undefined;
    message.engine = object.engine ?? 0;
    message.baselineDatabase = object.baselineDatabase ?? "";
    message.baselineSheetName = object.baselineSheetName ?? "";
    message.baselineChangeHistoryId = object.baselineChangeHistoryId ?? undefined;
    message.type = object.type ?? 0;
    message.etag = object.etag ?? "";
    message.protection = (object.protection !== undefined && object.protection !== null)
      ? SchemaDesign_Protection.fromPartial(object.protection)
      : undefined;
    message.creator = object.creator ?? "";
    message.updater = object.updater ?? "";
    message.createTime = object.createTime ?? undefined;
    message.updateTime = object.updateTime ?? undefined;
    return message;
  },
};

function createBaseSchemaDesign_Protection(): SchemaDesign_Protection {
  return { allowForcePushes: false };
}

export const SchemaDesign_Protection = {
  encode(message: SchemaDesign_Protection, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.allowForcePushes === true) {
      writer.uint32(8).bool(message.allowForcePushes);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SchemaDesign_Protection {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSchemaDesign_Protection();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.allowForcePushes = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SchemaDesign_Protection {
    return { allowForcePushes: isSet(object.allowForcePushes) ? globalThis.Boolean(object.allowForcePushes) : false };
  },

  toJSON(message: SchemaDesign_Protection): unknown {
    const obj: any = {};
    if (message.allowForcePushes === true) {
      obj.allowForcePushes = message.allowForcePushes;
    }
    return obj;
  },

  create(base?: DeepPartial<SchemaDesign_Protection>): SchemaDesign_Protection {
    return SchemaDesign_Protection.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SchemaDesign_Protection>): SchemaDesign_Protection {
    const message = createBaseSchemaDesign_Protection();
    message.allowForcePushes = object.allowForcePushes ?? false;
    return message;
  },
};

function createBaseGetSchemaDesignRequest(): GetSchemaDesignRequest {
  return { name: "" };
}

export const GetSchemaDesignRequest = {
  encode(message: GetSchemaDesignRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetSchemaDesignRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetSchemaDesignRequest();
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

  fromJSON(object: any): GetSchemaDesignRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: GetSchemaDesignRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<GetSchemaDesignRequest>): GetSchemaDesignRequest {
    return GetSchemaDesignRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<GetSchemaDesignRequest>): GetSchemaDesignRequest {
    const message = createBaseGetSchemaDesignRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListSchemaDesignsRequest(): ListSchemaDesignsRequest {
  return { parent: "", filter: "", pageSize: 0, pageToken: "", view: 0 };
}

export const ListSchemaDesignsRequest = {
  encode(message: ListSchemaDesignsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
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
    if (message.view !== 0) {
      writer.uint32(40).int32(message.view);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListSchemaDesignsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListSchemaDesignsRequest();
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

          message.filter = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.pageToken = reader.string();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.view = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListSchemaDesignsRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      filter: isSet(object.filter) ? globalThis.String(object.filter) : "",
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
      view: isSet(object.view) ? schemaDesignViewFromJSON(object.view) : 0,
    };
  },

  toJSON(message: ListSchemaDesignsRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.filter !== "") {
      obj.filter = message.filter;
    }
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    if (message.view !== 0) {
      obj.view = schemaDesignViewToJSON(message.view);
    }
    return obj;
  },

  create(base?: DeepPartial<ListSchemaDesignsRequest>): ListSchemaDesignsRequest {
    return ListSchemaDesignsRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListSchemaDesignsRequest>): ListSchemaDesignsRequest {
    const message = createBaseListSchemaDesignsRequest();
    message.parent = object.parent ?? "";
    message.filter = object.filter ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    message.view = object.view ?? 0;
    return message;
  },
};

function createBaseListSchemaDesignsResponse(): ListSchemaDesignsResponse {
  return { schemaDesigns: [], nextPageToken: "" };
}

export const ListSchemaDesignsResponse = {
  encode(message: ListSchemaDesignsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.schemaDesigns) {
      SchemaDesign.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListSchemaDesignsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListSchemaDesignsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.schemaDesigns.push(SchemaDesign.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListSchemaDesignsResponse {
    return {
      schemaDesigns: globalThis.Array.isArray(object?.schemaDesigns)
        ? object.schemaDesigns.map((e: any) => SchemaDesign.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListSchemaDesignsResponse): unknown {
    const obj: any = {};
    if (message.schemaDesigns?.length) {
      obj.schemaDesigns = message.schemaDesigns.map((e) => SchemaDesign.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListSchemaDesignsResponse>): ListSchemaDesignsResponse {
    return ListSchemaDesignsResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListSchemaDesignsResponse>): ListSchemaDesignsResponse {
    const message = createBaseListSchemaDesignsResponse();
    message.schemaDesigns = object.schemaDesigns?.map((e) => SchemaDesign.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseCreateSchemaDesignRequest(): CreateSchemaDesignRequest {
  return { parent: "", schemaDesign: undefined };
}

export const CreateSchemaDesignRequest = {
  encode(message: CreateSchemaDesignRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.schemaDesign !== undefined) {
      SchemaDesign.encode(message.schemaDesign, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateSchemaDesignRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateSchemaDesignRequest();
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

          message.schemaDesign = SchemaDesign.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateSchemaDesignRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      schemaDesign: isSet(object.schemaDesign) ? SchemaDesign.fromJSON(object.schemaDesign) : undefined,
    };
  },

  toJSON(message: CreateSchemaDesignRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.schemaDesign !== undefined) {
      obj.schemaDesign = SchemaDesign.toJSON(message.schemaDesign);
    }
    return obj;
  },

  create(base?: DeepPartial<CreateSchemaDesignRequest>): CreateSchemaDesignRequest {
    return CreateSchemaDesignRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<CreateSchemaDesignRequest>): CreateSchemaDesignRequest {
    const message = createBaseCreateSchemaDesignRequest();
    message.parent = object.parent ?? "";
    message.schemaDesign = (object.schemaDesign !== undefined && object.schemaDesign !== null)
      ? SchemaDesign.fromPartial(object.schemaDesign)
      : undefined;
    return message;
  },
};

function createBaseUpdateSchemaDesignRequest(): UpdateSchemaDesignRequest {
  return { schemaDesign: undefined, updateMask: undefined };
}

export const UpdateSchemaDesignRequest = {
  encode(message: UpdateSchemaDesignRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schemaDesign !== undefined) {
      SchemaDesign.encode(message.schemaDesign, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateSchemaDesignRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateSchemaDesignRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.schemaDesign = SchemaDesign.decode(reader, reader.uint32());
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

  fromJSON(object: any): UpdateSchemaDesignRequest {
    return {
      schemaDesign: isSet(object.schemaDesign) ? SchemaDesign.fromJSON(object.schemaDesign) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateSchemaDesignRequest): unknown {
    const obj: any = {};
    if (message.schemaDesign !== undefined) {
      obj.schemaDesign = SchemaDesign.toJSON(message.schemaDesign);
    }
    if (message.updateMask !== undefined) {
      obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask));
    }
    return obj;
  },

  create(base?: DeepPartial<UpdateSchemaDesignRequest>): UpdateSchemaDesignRequest {
    return UpdateSchemaDesignRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UpdateSchemaDesignRequest>): UpdateSchemaDesignRequest {
    const message = createBaseUpdateSchemaDesignRequest();
    message.schemaDesign = (object.schemaDesign !== undefined && object.schemaDesign !== null)
      ? SchemaDesign.fromPartial(object.schemaDesign)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseMergeSchemaDesignRequest(): MergeSchemaDesignRequest {
  return { name: "", targetName: "" };
}

export const MergeSchemaDesignRequest = {
  encode(message: MergeSchemaDesignRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.targetName !== "") {
      writer.uint32(18).string(message.targetName);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MergeSchemaDesignRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMergeSchemaDesignRequest();
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

          message.targetName = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MergeSchemaDesignRequest {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      targetName: isSet(object.targetName) ? globalThis.String(object.targetName) : "",
    };
  },

  toJSON(message: MergeSchemaDesignRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.targetName !== "") {
      obj.targetName = message.targetName;
    }
    return obj;
  },

  create(base?: DeepPartial<MergeSchemaDesignRequest>): MergeSchemaDesignRequest {
    return MergeSchemaDesignRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<MergeSchemaDesignRequest>): MergeSchemaDesignRequest {
    const message = createBaseMergeSchemaDesignRequest();
    message.name = object.name ?? "";
    message.targetName = object.targetName ?? "";
    return message;
  },
};

function createBaseParseSchemaStringRequest(): ParseSchemaStringRequest {
  return { schemaString: "", engine: 0 };
}

export const ParseSchemaStringRequest = {
  encode(message: ParseSchemaStringRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schemaString !== "") {
      writer.uint32(10).string(message.schemaString);
    }
    if (message.engine !== 0) {
      writer.uint32(16).int32(message.engine);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ParseSchemaStringRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseParseSchemaStringRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.schemaString = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.engine = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ParseSchemaStringRequest {
    return {
      schemaString: isSet(object.schemaString) ? globalThis.String(object.schemaString) : "",
      engine: isSet(object.engine) ? engineFromJSON(object.engine) : 0,
    };
  },

  toJSON(message: ParseSchemaStringRequest): unknown {
    const obj: any = {};
    if (message.schemaString !== "") {
      obj.schemaString = message.schemaString;
    }
    if (message.engine !== 0) {
      obj.engine = engineToJSON(message.engine);
    }
    return obj;
  },

  create(base?: DeepPartial<ParseSchemaStringRequest>): ParseSchemaStringRequest {
    return ParseSchemaStringRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ParseSchemaStringRequest>): ParseSchemaStringRequest {
    const message = createBaseParseSchemaStringRequest();
    message.schemaString = object.schemaString ?? "";
    message.engine = object.engine ?? 0;
    return message;
  },
};

function createBaseParseSchemaStringResponse(): ParseSchemaStringResponse {
  return { schemaMetadata: undefined };
}

export const ParseSchemaStringResponse = {
  encode(message: ParseSchemaStringResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schemaMetadata !== undefined) {
      DatabaseMetadata.encode(message.schemaMetadata, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ParseSchemaStringResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseParseSchemaStringResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.schemaMetadata = DatabaseMetadata.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ParseSchemaStringResponse {
    return {
      schemaMetadata: isSet(object.schemaMetadata) ? DatabaseMetadata.fromJSON(object.schemaMetadata) : undefined,
    };
  },

  toJSON(message: ParseSchemaStringResponse): unknown {
    const obj: any = {};
    if (message.schemaMetadata !== undefined) {
      obj.schemaMetadata = DatabaseMetadata.toJSON(message.schemaMetadata);
    }
    return obj;
  },

  create(base?: DeepPartial<ParseSchemaStringResponse>): ParseSchemaStringResponse {
    return ParseSchemaStringResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ParseSchemaStringResponse>): ParseSchemaStringResponse {
    const message = createBaseParseSchemaStringResponse();
    message.schemaMetadata = (object.schemaMetadata !== undefined && object.schemaMetadata !== null)
      ? DatabaseMetadata.fromPartial(object.schemaMetadata)
      : undefined;
    return message;
  },
};

function createBaseDeleteSchemaDesignRequest(): DeleteSchemaDesignRequest {
  return { name: "" };
}

export const DeleteSchemaDesignRequest = {
  encode(message: DeleteSchemaDesignRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteSchemaDesignRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteSchemaDesignRequest();
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

  fromJSON(object: any): DeleteSchemaDesignRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: DeleteSchemaDesignRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<DeleteSchemaDesignRequest>): DeleteSchemaDesignRequest {
    return DeleteSchemaDesignRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DeleteSchemaDesignRequest>): DeleteSchemaDesignRequest {
    const message = createBaseDeleteSchemaDesignRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseDiffMetadataRequest(): DiffMetadataRequest {
  return { sourceMetadata: undefined, targetMetadata: undefined, engine: 0 };
}

export const DiffMetadataRequest = {
  encode(message: DiffMetadataRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sourceMetadata !== undefined) {
      DatabaseMetadata.encode(message.sourceMetadata, writer.uint32(10).fork()).ldelim();
    }
    if (message.targetMetadata !== undefined) {
      DatabaseMetadata.encode(message.targetMetadata, writer.uint32(18).fork()).ldelim();
    }
    if (message.engine !== 0) {
      writer.uint32(24).int32(message.engine);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DiffMetadataRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDiffMetadataRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.sourceMetadata = DatabaseMetadata.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.targetMetadata = DatabaseMetadata.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.engine = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DiffMetadataRequest {
    return {
      sourceMetadata: isSet(object.sourceMetadata) ? DatabaseMetadata.fromJSON(object.sourceMetadata) : undefined,
      targetMetadata: isSet(object.targetMetadata) ? DatabaseMetadata.fromJSON(object.targetMetadata) : undefined,
      engine: isSet(object.engine) ? engineFromJSON(object.engine) : 0,
    };
  },

  toJSON(message: DiffMetadataRequest): unknown {
    const obj: any = {};
    if (message.sourceMetadata !== undefined) {
      obj.sourceMetadata = DatabaseMetadata.toJSON(message.sourceMetadata);
    }
    if (message.targetMetadata !== undefined) {
      obj.targetMetadata = DatabaseMetadata.toJSON(message.targetMetadata);
    }
    if (message.engine !== 0) {
      obj.engine = engineToJSON(message.engine);
    }
    return obj;
  },

  create(base?: DeepPartial<DiffMetadataRequest>): DiffMetadataRequest {
    return DiffMetadataRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DiffMetadataRequest>): DiffMetadataRequest {
    const message = createBaseDiffMetadataRequest();
    message.sourceMetadata = (object.sourceMetadata !== undefined && object.sourceMetadata !== null)
      ? DatabaseMetadata.fromPartial(object.sourceMetadata)
      : undefined;
    message.targetMetadata = (object.targetMetadata !== undefined && object.targetMetadata !== null)
      ? DatabaseMetadata.fromPartial(object.targetMetadata)
      : undefined;
    message.engine = object.engine ?? 0;
    return message;
  },
};

function createBaseDiffMetadataResponse(): DiffMetadataResponse {
  return { diff: "" };
}

export const DiffMetadataResponse = {
  encode(message: DiffMetadataResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.diff !== "") {
      writer.uint32(10).string(message.diff);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DiffMetadataResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDiffMetadataResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.diff = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DiffMetadataResponse {
    return { diff: isSet(object.diff) ? globalThis.String(object.diff) : "" };
  },

  toJSON(message: DiffMetadataResponse): unknown {
    const obj: any = {};
    if (message.diff !== "") {
      obj.diff = message.diff;
    }
    return obj;
  },

  create(base?: DeepPartial<DiffMetadataResponse>): DiffMetadataResponse {
    return DiffMetadataResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DiffMetadataResponse>): DiffMetadataResponse {
    const message = createBaseDiffMetadataResponse();
    message.diff = object.diff ?? "";
    return message;
  },
};

export type SchemaDesignServiceDefinition = typeof SchemaDesignServiceDefinition;
export const SchemaDesignServiceDefinition = {
  name: "SchemaDesignService",
  fullName: "bytebase.v1.SchemaDesignService",
  methods: {
    getSchemaDesign: {
      name: "GetSchemaDesign",
      requestType: GetSchemaDesignRequest,
      requestStream: false,
      responseType: SchemaDesign,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              39,
              18,
              37,
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
              115,
              99,
              104,
              101,
              109,
              97,
              68,
              101,
              115,
              105,
              103,
              110,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    listSchemaDesigns: {
      name: "ListSchemaDesigns",
      requestType: ListSchemaDesignsRequest,
      requestStream: false,
      responseType: ListSchemaDesignsResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          578365826: [
            new Uint8Array([
              39,
              18,
              37,
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
              115,
              99,
              104,
              101,
              109,
              97,
              68,
              101,
              115,
              105,
              103,
              110,
              115,
            ]),
          ],
        },
      },
    },
    createSchemaDesign: {
      name: "CreateSchemaDesign",
      requestType: CreateSchemaDesignRequest,
      requestStream: false,
      responseType: SchemaDesign,
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
              115,
              99,
              104,
              101,
              109,
              97,
              95,
              100,
              101,
              115,
              105,
              103,
              110,
            ]),
          ],
          578365826: [
            new Uint8Array([
              53,
              58,
              13,
              115,
              99,
              104,
              101,
              109,
              97,
              95,
              100,
              101,
              115,
              105,
              103,
              110,
              34,
              36,
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
              115,
              99,
              104,
              101,
              109,
              97,
              68,
              101,
              115,
              105,
              103,
              110,
            ]),
          ],
        },
      },
    },
    updateSchemaDesign: {
      name: "UpdateSchemaDesign",
      requestType: UpdateSchemaDesignRequest,
      requestStream: false,
      responseType: SchemaDesign,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([
              25,
              115,
              99,
              104,
              101,
              109,
              97,
              95,
              100,
              101,
              115,
              105,
              103,
              110,
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
          578365826: [
            new Uint8Array([
              68,
              58,
              13,
              115,
              99,
              104,
              101,
              109,
              97,
              95,
              100,
              101,
              115,
              105,
              103,
              110,
              50,
              51,
              47,
              118,
              49,
              47,
              123,
              115,
              99,
              104,
              101,
              109,
              97,
              95,
              100,
              101,
              115,
              105,
              103,
              110,
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
              115,
              99,
              104,
              101,
              109,
              97,
              68,
              101,
              115,
              105,
              103,
              110,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    mergeSchemaDesign: {
      name: "MergeSchemaDesign",
      requestType: MergeSchemaDesignRequest,
      requestStream: false,
      responseType: SchemaDesign,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([16, 110, 97, 109, 101, 44, 116, 97, 114, 103, 101, 116, 95, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              45,
              34,
              43,
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
              115,
              99,
              104,
              101,
              109,
              97,
              68,
              101,
              115,
              105,
              103,
              110,
              115,
              47,
              42,
              125,
              58,
              109,
              101,
              114,
              103,
              101,
            ]),
          ],
        },
      },
    },
    parseSchemaString: {
      name: "ParseSchemaString",
      requestType: ParseSchemaStringRequest,
      requestStream: false,
      responseType: ParseSchemaStringResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              39,
              58,
              1,
              42,
              34,
              34,
              47,
              118,
              49,
              47,
              115,
              99,
              104,
              101,
              109,
              97,
              68,
              101,
              115,
              105,
              103,
              110,
              58,
              112,
              97,
              114,
              115,
              101,
              83,
              99,
              104,
              101,
              109,
              97,
              83,
              116,
              114,
              105,
              110,
              103,
            ]),
          ],
        },
      },
    },
    deleteSchemaDesign: {
      name: "DeleteSchemaDesign",
      requestType: DeleteSchemaDesignRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              39,
              42,
              37,
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
              115,
              99,
              104,
              101,
              109,
              97,
              68,
              101,
              115,
              105,
              103,
              110,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    diffMetadata: {
      name: "DiffMetadata",
      requestType: DiffMetadataRequest,
      requestStream: false,
      responseType: DiffMetadataResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              34,
              58,
              1,
              42,
              34,
              29,
              47,
              118,
              49,
              47,
              115,
              99,
              104,
              101,
              109,
              97,
              68,
              101,
              115,
              105,
              103,
              110,
              58,
              100,
              105,
              102,
              102,
              77,
              101,
              116,
              97,
              100,
              97,
              116,
              97,
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
