/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";

export const protobufPackage = "bytebase.v1";

export interface CreateBookmarkRequest {
  /**
   * The parent resource of the bookmark.
   * Format: users/{user}, user is a server-generated unique IDs.
   */
  parent: string;
  /** The bookmark to create. */
  bookmark?: Bookmark;
}

export interface DeleteBookmarkRequest {
  /**
   * The name of the bookmark to delete.
   * Format: users/{user}/bookmarks/{bookmark}, user and bookmark are server-generated unique IDs.
   */
  name: string;
}

export interface ListBookmarksRequest {
  /**
   * The parent resource of the bookmark.
   * Format: users/{user}, user is a server-generated unique ID.
   */
  parent: string;
  /**
   * Not used. The maximum number of bookmarks to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 bookmarks will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * Not used. A page token, received from a previous `ListBookmarks` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListBookmarks` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface ListBookmarksResponse {
  /** The list of bookmarks. */
  bookmarks: Bookmark[];
  /**
   * Not used. A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface Bookmark {
  /**
   * The name of the bookmark.
   * Format: users/{user}/bookmarks/{bookmark}, user and bookmark are server-generated unique IDs.
   */
  name: string;
  /** The title of the bookmark. */
  title: string;
  /**
   * The resource link of the bookmark. Only support issue link for now.
   * Format:
   * Issue: /issue/slug(issue_name)-{issue_uid}
   * Example: /issue/start-here-add-email-column-to-employee-table-101
   */
  link: string;
}

function createBaseCreateBookmarkRequest(): CreateBookmarkRequest {
  return { parent: "", bookmark: undefined };
}

export const CreateBookmarkRequest = {
  encode(message: CreateBookmarkRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.bookmark !== undefined) {
      Bookmark.encode(message.bookmark, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateBookmarkRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateBookmarkRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.parent = reader.string();
          break;
        case 2:
          message.bookmark = Bookmark.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): CreateBookmarkRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      bookmark: isSet(object.bookmark) ? Bookmark.fromJSON(object.bookmark) : undefined,
    };
  },

  toJSON(message: CreateBookmarkRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.bookmark !== undefined && (obj.bookmark = message.bookmark ? Bookmark.toJSON(message.bookmark) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<CreateBookmarkRequest>): CreateBookmarkRequest {
    const message = createBaseCreateBookmarkRequest();
    message.parent = object.parent ?? "";
    message.bookmark = (object.bookmark !== undefined && object.bookmark !== null)
      ? Bookmark.fromPartial(object.bookmark)
      : undefined;
    return message;
  },
};

function createBaseDeleteBookmarkRequest(): DeleteBookmarkRequest {
  return { name: "" };
}

export const DeleteBookmarkRequest = {
  encode(message: DeleteBookmarkRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteBookmarkRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteBookmarkRequest();
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

  fromJSON(object: any): DeleteBookmarkRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: DeleteBookmarkRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<DeleteBookmarkRequest>): DeleteBookmarkRequest {
    const message = createBaseDeleteBookmarkRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListBookmarksRequest(): ListBookmarksRequest {
  return { parent: "", pageSize: 0, pageToken: "" };
}

export const ListBookmarksRequest = {
  encode(message: ListBookmarksRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): ListBookmarksRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListBookmarksRequest();
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
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ListBookmarksRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListBookmarksRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    return obj;
  },

  fromPartial(object: DeepPartial<ListBookmarksRequest>): ListBookmarksRequest {
    const message = createBaseListBookmarksRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListBookmarksResponse(): ListBookmarksResponse {
  return { bookmarks: [], nextPageToken: "" };
}

export const ListBookmarksResponse = {
  encode(message: ListBookmarksResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.bookmarks) {
      Bookmark.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListBookmarksResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListBookmarksResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.bookmarks.push(Bookmark.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListBookmarksResponse {
    return {
      bookmarks: Array.isArray(object?.bookmarks) ? object.bookmarks.map((e: any) => Bookmark.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListBookmarksResponse): unknown {
    const obj: any = {};
    if (message.bookmarks) {
      obj.bookmarks = message.bookmarks.map((e) => e ? Bookmark.toJSON(e) : undefined);
    } else {
      obj.bookmarks = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  fromPartial(object: DeepPartial<ListBookmarksResponse>): ListBookmarksResponse {
    const message = createBaseListBookmarksResponse();
    message.bookmarks = object.bookmarks?.map((e) => Bookmark.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseBookmark(): Bookmark {
  return { name: "", title: "", link: "" };
}

export const Bookmark = {
  encode(message: Bookmark, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.link !== "") {
      writer.uint32(26).string(message.link);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Bookmark {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBookmark();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.title = reader.string();
          break;
        case 3:
          message.link = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Bookmark {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      title: isSet(object.title) ? String(object.title) : "",
      link: isSet(object.link) ? String(object.link) : "",
    };
  },

  toJSON(message: Bookmark): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.title !== undefined && (obj.title = message.title);
    message.link !== undefined && (obj.link = message.link);
    return obj;
  },

  fromPartial(object: DeepPartial<Bookmark>): Bookmark {
    const message = createBaseBookmark();
    message.name = object.name ?? "";
    message.title = object.title ?? "";
    message.link = object.link ?? "";
    return message;
  },
};

export type BookmarkServiceDefinition = typeof BookmarkServiceDefinition;
export const BookmarkServiceDefinition = {
  name: "BookmarkService",
  fullName: "bytebase.v1.BookmarkService",
  methods: {
    /** CreateBookmark creates a new bookmark. */
    createBookmark: {
      name: "CreateBookmark",
      requestType: CreateBookmarkRequest,
      requestStream: false,
      responseType: Bookmark,
      responseStream: false,
      options: {},
    },
    /** DeleteBookmark deletes a bookmark. */
    deleteBookmark: {
      name: "DeleteBookmark",
      requestType: DeleteBookmarkRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {},
    },
    /** ListBookmark lists bookmarks. */
    listBookmarks: {
      name: "ListBookmarks",
      requestType: ListBookmarksRequest,
      requestStream: false,
      responseType: ListBookmarksResponse,
      responseStream: false,
      options: {},
    },
  },
} as const;

export interface BookmarkServiceImplementation<CallContextExt = {}> {
  /** CreateBookmark creates a new bookmark. */
  createBookmark(request: CreateBookmarkRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Bookmark>>;
  /** DeleteBookmark deletes a bookmark. */
  deleteBookmark(request: DeleteBookmarkRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Empty>>;
  /** ListBookmark lists bookmarks. */
  listBookmarks(
    request: ListBookmarksRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ListBookmarksResponse>>;
}

export interface BookmarkServiceClient<CallOptionsExt = {}> {
  /** CreateBookmark creates a new bookmark. */
  createBookmark(
    request: DeepPartial<CreateBookmarkRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Bookmark>;
  /** DeleteBookmark deletes a bookmark. */
  deleteBookmark(request: DeepPartial<DeleteBookmarkRequest>, options?: CallOptions & CallOptionsExt): Promise<Empty>;
  /** ListBookmark lists bookmarks. */
  listBookmarks(
    request: DeepPartial<ListBookmarksRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ListBookmarksResponse>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
