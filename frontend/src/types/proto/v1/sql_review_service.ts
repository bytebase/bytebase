/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { Timestamp } from "../google/protobuf/timestamp";
import { SQLReviewRule } from "./org_policy_service";

export const protobufPackage = "bytebase.v1";

export interface ListSQLReviewsRequest {
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

export interface ListSQLReviewsResponse {
  /** The sql review from the specified request. */
  sqlReviews: SQLReview[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface CreateSQLReviewRequest {
  /** The sql review to create. */
  sqlReview: SQLReview | undefined;
}

export interface UpdateSQLReviewRequest {
  /**
   * The sql review toupdate.
   *
   * The name field is used to identify the sql review to update.
   */
  sqlReview:
    | SQLReview
    | undefined;
  /** The list of fields to update. */
  updateMask: string[] | undefined;
}

export interface GetSQLReviewRequest {
  /**
   * The name of the sql review to retrieve.
   * Format: sqlReviews/{uid}
   */
  name: string;
}

export interface DeleteSQLReviewRequest {
  /**
   * The name of the sql review to delete.
   * Format: sqlReviews/{uid}
   */
  name: string;
}

export interface SQLReview {
  /**
   * The name of the sql review to retrieve.
   * Format: sqlReviews/{uid}
   */
  name: string;
  title: string;
  enforce: boolean;
  /** Format: users/hello@world.com */
  creator: string;
  createTime: Date | undefined;
  updateTime: Date | undefined;
  rules: SQLReviewRule[];
}

function createBaseListSQLReviewsRequest(): ListSQLReviewsRequest {
  return { pageSize: 0, pageToken: "" };
}

export const ListSQLReviewsRequest = {
  encode(message: ListSQLReviewsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pageSize !== 0) {
      writer.uint32(8).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(18).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListSQLReviewsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListSQLReviewsRequest();
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

  fromJSON(object: any): ListSQLReviewsRequest {
    return {
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
    };
  },

  toJSON(message: ListSQLReviewsRequest): unknown {
    const obj: any = {};
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListSQLReviewsRequest>): ListSQLReviewsRequest {
    return ListSQLReviewsRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListSQLReviewsRequest>): ListSQLReviewsRequest {
    const message = createBaseListSQLReviewsRequest();
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListSQLReviewsResponse(): ListSQLReviewsResponse {
  return { sqlReviews: [], nextPageToken: "" };
}

export const ListSQLReviewsResponse = {
  encode(message: ListSQLReviewsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.sqlReviews) {
      SQLReview.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListSQLReviewsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListSQLReviewsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.sqlReviews.push(SQLReview.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListSQLReviewsResponse {
    return {
      sqlReviews: globalThis.Array.isArray(object?.sqlReviews)
        ? object.sqlReviews.map((e: any) => SQLReview.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListSQLReviewsResponse): unknown {
    const obj: any = {};
    if (message.sqlReviews?.length) {
      obj.sqlReviews = message.sqlReviews.map((e) => SQLReview.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListSQLReviewsResponse>): ListSQLReviewsResponse {
    return ListSQLReviewsResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListSQLReviewsResponse>): ListSQLReviewsResponse {
    const message = createBaseListSQLReviewsResponse();
    message.sqlReviews = object.sqlReviews?.map((e) => SQLReview.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseCreateSQLReviewRequest(): CreateSQLReviewRequest {
  return { sqlReview: undefined };
}

export const CreateSQLReviewRequest = {
  encode(message: CreateSQLReviewRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sqlReview !== undefined) {
      SQLReview.encode(message.sqlReview, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateSQLReviewRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateSQLReviewRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.sqlReview = SQLReview.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateSQLReviewRequest {
    return { sqlReview: isSet(object.sqlReview) ? SQLReview.fromJSON(object.sqlReview) : undefined };
  },

  toJSON(message: CreateSQLReviewRequest): unknown {
    const obj: any = {};
    if (message.sqlReview !== undefined) {
      obj.sqlReview = SQLReview.toJSON(message.sqlReview);
    }
    return obj;
  },

  create(base?: DeepPartial<CreateSQLReviewRequest>): CreateSQLReviewRequest {
    return CreateSQLReviewRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<CreateSQLReviewRequest>): CreateSQLReviewRequest {
    const message = createBaseCreateSQLReviewRequest();
    message.sqlReview = (object.sqlReview !== undefined && object.sqlReview !== null)
      ? SQLReview.fromPartial(object.sqlReview)
      : undefined;
    return message;
  },
};

function createBaseUpdateSQLReviewRequest(): UpdateSQLReviewRequest {
  return { sqlReview: undefined, updateMask: undefined };
}

export const UpdateSQLReviewRequest = {
  encode(message: UpdateSQLReviewRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sqlReview !== undefined) {
      SQLReview.encode(message.sqlReview, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateSQLReviewRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateSQLReviewRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.sqlReview = SQLReview.decode(reader, reader.uint32());
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

  fromJSON(object: any): UpdateSQLReviewRequest {
    return {
      sqlReview: isSet(object.sqlReview) ? SQLReview.fromJSON(object.sqlReview) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateSQLReviewRequest): unknown {
    const obj: any = {};
    if (message.sqlReview !== undefined) {
      obj.sqlReview = SQLReview.toJSON(message.sqlReview);
    }
    if (message.updateMask !== undefined) {
      obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask));
    }
    return obj;
  },

  create(base?: DeepPartial<UpdateSQLReviewRequest>): UpdateSQLReviewRequest {
    return UpdateSQLReviewRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UpdateSQLReviewRequest>): UpdateSQLReviewRequest {
    const message = createBaseUpdateSQLReviewRequest();
    message.sqlReview = (object.sqlReview !== undefined && object.sqlReview !== null)
      ? SQLReview.fromPartial(object.sqlReview)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseGetSQLReviewRequest(): GetSQLReviewRequest {
  return { name: "" };
}

export const GetSQLReviewRequest = {
  encode(message: GetSQLReviewRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetSQLReviewRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetSQLReviewRequest();
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

  fromJSON(object: any): GetSQLReviewRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: GetSQLReviewRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<GetSQLReviewRequest>): GetSQLReviewRequest {
    return GetSQLReviewRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<GetSQLReviewRequest>): GetSQLReviewRequest {
    const message = createBaseGetSQLReviewRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseDeleteSQLReviewRequest(): DeleteSQLReviewRequest {
  return { name: "" };
}

export const DeleteSQLReviewRequest = {
  encode(message: DeleteSQLReviewRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteSQLReviewRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteSQLReviewRequest();
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

  fromJSON(object: any): DeleteSQLReviewRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: DeleteSQLReviewRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<DeleteSQLReviewRequest>): DeleteSQLReviewRequest {
    return DeleteSQLReviewRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DeleteSQLReviewRequest>): DeleteSQLReviewRequest {
    const message = createBaseDeleteSQLReviewRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseSQLReview(): SQLReview {
  return { name: "", title: "", enforce: false, creator: "", createTime: undefined, updateTime: undefined, rules: [] };
}

export const SQLReview = {
  encode(message: SQLReview, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.enforce === true) {
      writer.uint32(24).bool(message.enforce);
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

  decode(input: _m0.Reader | Uint8Array, length?: number): SQLReview {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSQLReview();
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

          message.enforce = reader.bool();
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

  fromJSON(object: any): SQLReview {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      enforce: isSet(object.enforce) ? globalThis.Boolean(object.enforce) : false,
      creator: isSet(object.creator) ? globalThis.String(object.creator) : "",
      createTime: isSet(object.createTime) ? fromJsonTimestamp(object.createTime) : undefined,
      updateTime: isSet(object.updateTime) ? fromJsonTimestamp(object.updateTime) : undefined,
      rules: globalThis.Array.isArray(object?.rules) ? object.rules.map((e: any) => SQLReviewRule.fromJSON(e)) : [],
    };
  },

  toJSON(message: SQLReview): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.enforce === true) {
      obj.enforce = message.enforce;
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

  create(base?: DeepPartial<SQLReview>): SQLReview {
    return SQLReview.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SQLReview>): SQLReview {
    const message = createBaseSQLReview();
    message.name = object.name ?? "";
    message.title = object.title ?? "";
    message.enforce = object.enforce ?? false;
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
    createSQLReview: {
      name: "CreateSQLReview",
      requestType: CreateSQLReviewRequest,
      requestStream: false,
      responseType: SQLReview,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          578365826: [
            new Uint8Array([
              28,
              58,
              10,
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
              34,
              14,
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
              115,
            ]),
          ],
        },
      },
    },
    listSQLReviews: {
      name: "ListSQLReviews",
      requestType: ListSQLReviewsRequest,
      requestStream: false,
      responseType: ListSQLReviewsResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          578365826: [new Uint8Array([16, 18, 14, 47, 118, 49, 47, 115, 113, 108, 82, 101, 118, 105, 101, 119, 115])],
        },
      },
    },
    getSQLReview: {
      name: "GetSQLReview",
      requestType: GetSQLReviewRequest,
      requestStream: false,
      responseType: SQLReview,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              25,
              18,
              23,
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
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    updateSQLReview: {
      name: "UpdateSQLReview",
      requestType: UpdateSQLReviewRequest,
      requestStream: false,
      responseType: SQLReview,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([
              22,
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
              48,
              58,
              10,
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
              50,
              34,
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
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    deleteSQLReview: {
      name: "DeleteSQLReview",
      requestType: DeleteSQLReviewRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              25,
              42,
              23,
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
