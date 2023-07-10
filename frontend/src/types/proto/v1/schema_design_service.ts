/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { Timestamp } from "../google/protobuf/timestamp";
import { Engine, engineFromJSON, engineToJSON } from "./common";
import { DatabaseMetadata } from "./database_service";

export const protobufPackage = "bytebase.v1";

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
  schemaMetadata?: DatabaseMetadata;
  /** The baseline schema. */
  baselineSchema: string;
  /** The metadata of the baseline schema. */
  baselineSchemaMetadata?: DatabaseMetadata;
  /** The database engine of the schema design. */
  engine: Engine;
  /**
   * The name of the baseline database.
   * Format: instances/{instance}/databases/{database}
   */
  baselineDatabase: string;
  /**
   * The selected schema version of the baseline database.
   * If not specified, the latest schema of database will be used as baseline schema.
   */
  schemaVersion: string;
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
  createTime?: Date;
  /** The timestamp when the schema design was last updated. */
  updateTime?: Date;
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
  /**
   * To filter the search result.
   * Format: only support the following spec for now:
   * - `creator = users/{email}`, `creator != users/{email}`
   * - `starred = true`, `starred = false`.
   * Not support empty filter for now.
   */
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
   * Use "projects/-" to list all schema designs.
   */
  parent: string;
  schemaDesign?: SchemaDesign;
}

export interface UpdateSchemaDesignRequest {
  /**
   * The schema design to update.
   *
   * The schema design's `name` field is used to identify the schema design to update.
   * Format: projects/{project}/schemaDesigns/{schemaDesign}
   */
  schemaDesign?: SchemaDesign;
  /** The list of fields to update. */
  updateMask?: string[];
}

export interface DeleteSchemaDesignRequest {
  /**
   * The name of the schema design to delete.
   * Format: projects/{project}/schemaDesigns/{schemaDesign}
   */
  name: string;
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
    schemaVersion: "",
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
    if (message.schemaVersion !== "") {
      writer.uint32(74).string(message.schemaVersion);
    }
    if (message.creator !== "") {
      writer.uint32(82).string(message.creator);
    }
    if (message.updater !== "") {
      writer.uint32(90).string(message.updater);
    }
    if (message.createTime !== undefined) {
      Timestamp.encode(toTimestamp(message.createTime), writer.uint32(98).fork()).ldelim();
    }
    if (message.updateTime !== undefined) {
      Timestamp.encode(toTimestamp(message.updateTime), writer.uint32(106).fork()).ldelim();
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

          message.schemaVersion = reader.string();
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.creator = reader.string();
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.updater = reader.string();
          continue;
        case 12:
          if (tag !== 98) {
            break;
          }

          message.createTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 13:
          if (tag !== 106) {
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
      name: isSet(object.name) ? String(object.name) : "",
      title: isSet(object.title) ? String(object.title) : "",
      schema: isSet(object.schema) ? String(object.schema) : "",
      schemaMetadata: isSet(object.schemaMetadata) ? DatabaseMetadata.fromJSON(object.schemaMetadata) : undefined,
      baselineSchema: isSet(object.baselineSchema) ? String(object.baselineSchema) : "",
      baselineSchemaMetadata: isSet(object.baselineSchemaMetadata)
        ? DatabaseMetadata.fromJSON(object.baselineSchemaMetadata)
        : undefined,
      engine: isSet(object.engine) ? engineFromJSON(object.engine) : 0,
      baselineDatabase: isSet(object.baselineDatabase) ? String(object.baselineDatabase) : "",
      schemaVersion: isSet(object.schemaVersion) ? String(object.schemaVersion) : "",
      creator: isSet(object.creator) ? String(object.creator) : "",
      updater: isSet(object.updater) ? String(object.updater) : "",
      createTime: isSet(object.createTime) ? fromJsonTimestamp(object.createTime) : undefined,
      updateTime: isSet(object.updateTime) ? fromJsonTimestamp(object.updateTime) : undefined,
    };
  },

  toJSON(message: SchemaDesign): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.title !== undefined && (obj.title = message.title);
    message.schema !== undefined && (obj.schema = message.schema);
    message.schemaMetadata !== undefined &&
      (obj.schemaMetadata = message.schemaMetadata ? DatabaseMetadata.toJSON(message.schemaMetadata) : undefined);
    message.baselineSchema !== undefined && (obj.baselineSchema = message.baselineSchema);
    message.baselineSchemaMetadata !== undefined && (obj.baselineSchemaMetadata = message.baselineSchemaMetadata
      ? DatabaseMetadata.toJSON(message.baselineSchemaMetadata)
      : undefined);
    message.engine !== undefined && (obj.engine = engineToJSON(message.engine));
    message.baselineDatabase !== undefined && (obj.baselineDatabase = message.baselineDatabase);
    message.schemaVersion !== undefined && (obj.schemaVersion = message.schemaVersion);
    message.creator !== undefined && (obj.creator = message.creator);
    message.updater !== undefined && (obj.updater = message.updater);
    message.createTime !== undefined && (obj.createTime = message.createTime.toISOString());
    message.updateTime !== undefined && (obj.updateTime = message.updateTime.toISOString());
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
    message.schemaVersion = object.schemaVersion ?? "";
    message.creator = object.creator ?? "";
    message.updater = object.updater ?? "";
    message.createTime = object.createTime ?? undefined;
    message.updateTime = object.updateTime ?? undefined;
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
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetSchemaDesignRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
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
  return { parent: "", filter: "", pageSize: 0, pageToken: "" };
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
      parent: isSet(object.parent) ? String(object.parent) : "",
      filter: isSet(object.filter) ? String(object.filter) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListSchemaDesignsRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.filter !== undefined && (obj.filter = message.filter);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
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
      schemaDesigns: Array.isArray(object?.schemaDesigns)
        ? object.schemaDesigns.map((e: any) => SchemaDesign.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListSchemaDesignsResponse): unknown {
    const obj: any = {};
    if (message.schemaDesigns) {
      obj.schemaDesigns = message.schemaDesigns.map((e) => e ? SchemaDesign.toJSON(e) : undefined);
    } else {
      obj.schemaDesigns = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
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
      parent: isSet(object.parent) ? String(object.parent) : "",
      schemaDesign: isSet(object.schemaDesign) ? SchemaDesign.fromJSON(object.schemaDesign) : undefined,
    };
  },

  toJSON(message: CreateSchemaDesignRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.schemaDesign !== undefined &&
      (obj.schemaDesign = message.schemaDesign ? SchemaDesign.toJSON(message.schemaDesign) : undefined);
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
    message.schemaDesign !== undefined &&
      (obj.schemaDesign = message.schemaDesign ? SchemaDesign.toJSON(message.schemaDesign) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
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
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: DeleteSchemaDesignRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
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
              19,
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
              68,
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
              24,
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
  },
} as const;

export interface SchemaDesignServiceImplementation<CallContextExt = {}> {
  getSchemaDesign(
    request: GetSchemaDesignRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<SchemaDesign>>;
  listSchemaDesigns(
    request: ListSchemaDesignsRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ListSchemaDesignsResponse>>;
  createSchemaDesign(
    request: CreateSchemaDesignRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<SchemaDesign>>;
  updateSchemaDesign(
    request: UpdateSchemaDesignRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<SchemaDesign>>;
  deleteSchemaDesign(
    request: DeleteSchemaDesignRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Empty>>;
}

export interface SchemaDesignServiceClient<CallOptionsExt = {}> {
  getSchemaDesign(
    request: DeepPartial<GetSchemaDesignRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<SchemaDesign>;
  listSchemaDesigns(
    request: DeepPartial<ListSchemaDesignsRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ListSchemaDesignsResponse>;
  createSchemaDesign(
    request: DeepPartial<CreateSchemaDesignRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<SchemaDesign>;
  updateSchemaDesign(
    request: DeepPartial<UpdateSchemaDesignRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<SchemaDesign>;
  deleteSchemaDesign(
    request: DeepPartial<DeleteSchemaDesignRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Empty>;
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

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
