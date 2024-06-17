/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { Timestamp } from "../google/protobuf/timestamp";
import { SQLReviewRule } from "./org_policy_service";

export const protobufPackage = "bytebase.v1";

export interface ListSQLReviewConfigsRequest {
  /**
   * The maximum number of sql review to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 sql review will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListSQLReviews` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface ListSQLReviewConfigsResponse {
  /** The sql review from the specified request. */
  sqlReviewConfigs: SQLReviewConfig[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface CreateSQLReviewConfigRequest {
  /** The sql review to create. */
  sqlReviewConfig: SQLReviewConfig | undefined;
}

export interface UpdateSQLReviewConfigRequest {
  /**
   * The sql review toupdate.
   *
   * The name field is used to identify the sql review to update.
   */
  sqlReviewConfig:
    | SQLReviewConfig
    | undefined;
  /** The list of fields to update. */
  updateMask: string[] | undefined;
}

export interface GetSQLReviewConfigRequest {
  /**
   * The name of the sql review to retrieve.
   * Format: sqlReviewConfigs/{uid}
   */
  name: string;
}

export interface DeleteSQLReviewConfigRequest {
  /**
   * The name of the sql review to delete.
   * Format: sqlReviewConfigs/{uid}
   */
  name: string;
}

export interface SQLReviewConfig {
  /**
   * The name of the sql review to retrieve.
   * Format: sqlReviewConfigs/{uid}
   */
  name: string;
  title: string;
  enabled: boolean;
  /** Format: users/hello@world.com */
  creator: string;
  createTime: Date | undefined;
  updateTime: Date | undefined;
  rules: SQLReviewRule[];
}

function createBaseListSQLReviewConfigsRequest(): ListSQLReviewConfigsRequest {
  return { pageSize: 0, pageToken: "" };
}

export const ListSQLReviewConfigsRequest = {
  encode(message: ListSQLReviewConfigsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pageSize !== 0) {
      writer.uint32(8).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(18).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListSQLReviewConfigsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListSQLReviewConfigsRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 2:
          if (tag !== 18) {
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

  fromJSON(object: any): ListSQLReviewConfigsRequest {
    return {
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
    };
  },

  toJSON(message: ListSQLReviewConfigsRequest): unknown {
    const obj: any = {};
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListSQLReviewConfigsRequest>): ListSQLReviewConfigsRequest {
    return ListSQLReviewConfigsRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListSQLReviewConfigsRequest>): ListSQLReviewConfigsRequest {
    const message = createBaseListSQLReviewConfigsRequest();
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListSQLReviewConfigsResponse(): ListSQLReviewConfigsResponse {
  return { sqlReviewConfigs: [], nextPageToken: "" };
}

export const ListSQLReviewConfigsResponse = {
  encode(message: ListSQLReviewConfigsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.sqlReviewConfigs) {
      SQLReviewConfig.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListSQLReviewConfigsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListSQLReviewConfigsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.sqlReviewConfigs.push(SQLReviewConfig.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListSQLReviewConfigsResponse {
    return {
      sqlReviewConfigs: globalThis.Array.isArray(object?.sqlReviewConfigs)
        ? object.sqlReviewConfigs.map((e: any) => SQLReviewConfig.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListSQLReviewConfigsResponse): unknown {
    const obj: any = {};
    if (message.sqlReviewConfigs?.length) {
      obj.sqlReviewConfigs = message.sqlReviewConfigs.map((e) => SQLReviewConfig.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListSQLReviewConfigsResponse>): ListSQLReviewConfigsResponse {
    return ListSQLReviewConfigsResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListSQLReviewConfigsResponse>): ListSQLReviewConfigsResponse {
    const message = createBaseListSQLReviewConfigsResponse();
    message.sqlReviewConfigs = object.sqlReviewConfigs?.map((e) => SQLReviewConfig.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseCreateSQLReviewConfigRequest(): CreateSQLReviewConfigRequest {
  return { sqlReviewConfig: undefined };
}

export const CreateSQLReviewConfigRequest = {
  encode(message: CreateSQLReviewConfigRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sqlReviewConfig !== undefined) {
      SQLReviewConfig.encode(message.sqlReviewConfig, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateSQLReviewConfigRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateSQLReviewConfigRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.sqlReviewConfig = SQLReviewConfig.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateSQLReviewConfigRequest {
    return {
      sqlReviewConfig: isSet(object.sqlReviewConfig) ? SQLReviewConfig.fromJSON(object.sqlReviewConfig) : undefined,
    };
  },

  toJSON(message: CreateSQLReviewConfigRequest): unknown {
    const obj: any = {};
    if (message.sqlReviewConfig !== undefined) {
      obj.sqlReviewConfig = SQLReviewConfig.toJSON(message.sqlReviewConfig);
    }
    return obj;
  },

  create(base?: DeepPartial<CreateSQLReviewConfigRequest>): CreateSQLReviewConfigRequest {
    return CreateSQLReviewConfigRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<CreateSQLReviewConfigRequest>): CreateSQLReviewConfigRequest {
    const message = createBaseCreateSQLReviewConfigRequest();
    message.sqlReviewConfig = (object.sqlReviewConfig !== undefined && object.sqlReviewConfig !== null)
      ? SQLReviewConfig.fromPartial(object.sqlReviewConfig)
      : undefined;
    return message;
  },
};

function createBaseUpdateSQLReviewConfigRequest(): UpdateSQLReviewConfigRequest {
  return { sqlReviewConfig: undefined, updateMask: undefined };
}

export const UpdateSQLReviewConfigRequest = {
  encode(message: UpdateSQLReviewConfigRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sqlReviewConfig !== undefined) {
      SQLReviewConfig.encode(message.sqlReviewConfig, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateSQLReviewConfigRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateSQLReviewConfigRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.sqlReviewConfig = SQLReviewConfig.decode(reader, reader.uint32());
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

  fromJSON(object: any): UpdateSQLReviewConfigRequest {
    return {
      sqlReviewConfig: isSet(object.sqlReviewConfig) ? SQLReviewConfig.fromJSON(object.sqlReviewConfig) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateSQLReviewConfigRequest): unknown {
    const obj: any = {};
    if (message.sqlReviewConfig !== undefined) {
      obj.sqlReviewConfig = SQLReviewConfig.toJSON(message.sqlReviewConfig);
    }
    if (message.updateMask !== undefined) {
      obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask));
    }
    return obj;
  },

  create(base?: DeepPartial<UpdateSQLReviewConfigRequest>): UpdateSQLReviewConfigRequest {
    return UpdateSQLReviewConfigRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UpdateSQLReviewConfigRequest>): UpdateSQLReviewConfigRequest {
    const message = createBaseUpdateSQLReviewConfigRequest();
    message.sqlReviewConfig = (object.sqlReviewConfig !== undefined && object.sqlReviewConfig !== null)
      ? SQLReviewConfig.fromPartial(object.sqlReviewConfig)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseGetSQLReviewConfigRequest(): GetSQLReviewConfigRequest {
  return { name: "" };
}

export const GetSQLReviewConfigRequest = {
  encode(message: GetSQLReviewConfigRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetSQLReviewConfigRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetSQLReviewConfigRequest();
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

  fromJSON(object: any): GetSQLReviewConfigRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: GetSQLReviewConfigRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<GetSQLReviewConfigRequest>): GetSQLReviewConfigRequest {
    return GetSQLReviewConfigRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<GetSQLReviewConfigRequest>): GetSQLReviewConfigRequest {
    const message = createBaseGetSQLReviewConfigRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseDeleteSQLReviewConfigRequest(): DeleteSQLReviewConfigRequest {
  return { name: "" };
}

export const DeleteSQLReviewConfigRequest = {
  encode(message: DeleteSQLReviewConfigRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteSQLReviewConfigRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteSQLReviewConfigRequest();
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

  fromJSON(object: any): DeleteSQLReviewConfigRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: DeleteSQLReviewConfigRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<DeleteSQLReviewConfigRequest>): DeleteSQLReviewConfigRequest {
    return DeleteSQLReviewConfigRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DeleteSQLReviewConfigRequest>): DeleteSQLReviewConfigRequest {
    const message = createBaseDeleteSQLReviewConfigRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseSQLReviewConfig(): SQLReviewConfig {
  return { name: "", title: "", enabled: false, creator: "", createTime: undefined, updateTime: undefined, rules: [] };
}

export const SQLReviewConfig = {
  encode(message: SQLReviewConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.enabled === true) {
      writer.uint32(24).bool(message.enabled);
    }
    if (message.creator !== "") {
      writer.uint32(34).string(message.creator);
    }
    if (message.createTime !== undefined) {
      Timestamp.encode(toTimestamp(message.createTime), writer.uint32(42).fork()).ldelim();
    }
    if (message.updateTime !== undefined) {
      Timestamp.encode(toTimestamp(message.updateTime), writer.uint32(50).fork()).ldelim();
    }
    for (const v of message.rules) {
      SQLReviewRule.encode(v!, writer.uint32(58).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SQLReviewConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSQLReviewConfig();
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
          if (tag !== 24) {
            break;
          }

          message.enabled = reader.bool();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.creator = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.createTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.updateTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.rules.push(SQLReviewRule.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SQLReviewConfig {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      enabled: isSet(object.enabled) ? globalThis.Boolean(object.enabled) : false,
      creator: isSet(object.creator) ? globalThis.String(object.creator) : "",
      createTime: isSet(object.createTime) ? fromJsonTimestamp(object.createTime) : undefined,
      updateTime: isSet(object.updateTime) ? fromJsonTimestamp(object.updateTime) : undefined,
      rules: globalThis.Array.isArray(object?.rules) ? object.rules.map((e: any) => SQLReviewRule.fromJSON(e)) : [],
    };
  },

  toJSON(message: SQLReviewConfig): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.enabled === true) {
      obj.enabled = message.enabled;
    }
    if (message.creator !== "") {
      obj.creator = message.creator;
    }
    if (message.createTime !== undefined) {
      obj.createTime = message.createTime.toISOString();
    }
    if (message.updateTime !== undefined) {
      obj.updateTime = message.updateTime.toISOString();
    }
    if (message.rules?.length) {
      obj.rules = message.rules.map((e) => SQLReviewRule.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<SQLReviewConfig>): SQLReviewConfig {
    return SQLReviewConfig.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SQLReviewConfig>): SQLReviewConfig {
    const message = createBaseSQLReviewConfig();
    message.name = object.name ?? "";
    message.title = object.title ?? "";
    message.enabled = object.enabled ?? false;
    message.creator = object.creator ?? "";
    message.createTime = object.createTime ?? undefined;
    message.updateTime = object.updateTime ?? undefined;
    message.rules = object.rules?.map((e) => SQLReviewRule.fromPartial(e)) || [];
    return message;
  },
};

export type SQLReviewServiceDefinition = typeof SQLReviewServiceDefinition;
export const SQLReviewServiceDefinition = {
  name: "SQLReviewService",
  fullName: "bytebase.v1.SQLReviewService",
  methods: {
    createSQLReviewConfig: {
      name: "CreateSQLReviewConfig",
      requestType: CreateSQLReviewConfigRequest,
      requestStream: false,
      responseType: SQLReviewConfig,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          578365826: [
            new Uint8Array([
              41,
              58,
              17,
              115,
              113,
              108,
              95,
              114,
              101,
              118,
              105,
              101,
              119,
              95,
              99,
              111,
              110,
              102,
              105,
              103,
              34,
              20,
              47,
              118,
              49,
              47,
              115,
              113,
              108,
              82,
              101,
              118,
              105,
              101,
              119,
              67,
              111,
              110,
              102,
              105,
              103,
              115,
            ]),
          ],
        },
      },
    },
    listSQLReviewConfigs: {
      name: "ListSQLReviewConfigs",
      requestType: ListSQLReviewConfigsRequest,
      requestStream: false,
      responseType: ListSQLReviewConfigsResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          578365826: [
            new Uint8Array([
              22,
              18,
              20,
              47,
              118,
              49,
              47,
              115,
              113,
              108,
              82,
              101,
              118,
              105,
              101,
              119,
              67,
              111,
              110,
              102,
              105,
              103,
              115,
            ]),
          ],
        },
      },
    },
    getSQLReviewConfig: {
      name: "GetSQLReviewConfig",
      requestType: GetSQLReviewConfigRequest,
      requestStream: false,
      responseType: SQLReviewConfig,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              31,
              18,
              29,
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
              115,
              113,
              108,
              82,
              101,
              118,
              105,
              101,
              119,
              67,
              111,
              110,
              102,
              105,
              103,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    updateSQLReviewConfig: {
      name: "UpdateSQLReviewConfig",
      requestType: UpdateSQLReviewConfigRequest,
      requestStream: false,
      responseType: SQLReviewConfig,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([
              29,
              115,
              113,
              108,
              95,
              114,
              101,
              118,
              105,
              101,
              119,
              95,
              99,
              111,
              110,
              102,
              105,
              103,
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
              17,
              115,
              113,
              108,
              95,
              114,
              101,
              118,
              105,
              101,
              119,
              95,
              99,
              111,
              110,
              102,
              105,
              103,
              50,
              47,
              47,
              118,
              49,
              47,
              123,
              115,
              113,
              108,
              95,
              114,
              101,
              118,
              105,
              101,
              119,
              95,
              99,
              111,
              110,
              102,
              105,
              103,
              46,
              110,
              97,
              109,
              101,
              61,
              115,
              113,
              108,
              82,
              101,
              118,
              105,
              101,
              119,
              67,
              111,
              110,
              102,
              105,
              103,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    deleteSQLReviewConfig: {
      name: "DeleteSQLReviewConfig",
      requestType: DeleteSQLReviewConfigRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              31,
              42,
              29,
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
              115,
              113,
              108,
              82,
              101,
              118,
              105,
              101,
              119,
              67,
              111,
              110,
              102,
              105,
              103,
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
