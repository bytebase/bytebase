/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";

export const protobufPackage = "bytebase.v1";

export interface CreateBookmarkRequest {
  /** The bookmark to create. */
  bookmark?: Bookmark | undefined;
}

export interface DeleteBookmarkRequest {
  /**
   * The name of the bookmark to delete.
   * Format: bookmarks/{bookmark}
   */
  name: string;
}

export interface ListBookmarksRequest {
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
   * Format: bookmarks/{bookmark}, user and bookmark are server-generated unique IDs.
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
  return { bookmark: undefined };
}

export const CreateBookmarkRequest = {
  encode(message: CreateBookmarkRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.bookmark !== undefined) {
      Bookmark.encode(message.bookmark, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateBookmarkRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateBookmarkRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.bookmark = Bookmark.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateBookmarkRequest {
    return { bookmark: isSet(object.bookmark) ? Bookmark.fromJSON(object.bookmark) : undefined };
  },

  toJSON(message: CreateBookmarkRequest): unknown {
    const obj: any = {};
    if (message.bookmark !== undefined) {
      obj.bookmark = Bookmark.toJSON(message.bookmark);
    }
    return obj;
  },

  create(base?: DeepPartial<CreateBookmarkRequest>): CreateBookmarkRequest {
    return CreateBookmarkRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<CreateBookmarkRequest>): CreateBookmarkRequest {
    const message = createBaseCreateBookmarkRequest();
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteBookmarkRequest();
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

  fromJSON(object: any): DeleteBookmarkRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: DeleteBookmarkRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<DeleteBookmarkRequest>): DeleteBookmarkRequest {
    return DeleteBookmarkRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<DeleteBookmarkRequest>): DeleteBookmarkRequest {
    const message = createBaseDeleteBookmarkRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListBookmarksRequest(): ListBookmarksRequest {
  return { pageSize: 0, pageToken: "" };
}

export const ListBookmarksRequest = {
  encode(message: ListBookmarksRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pageSize !== 0) {
      writer.uint32(8).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(18).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListBookmarksRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListBookmarksRequest();
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

  fromJSON(object: any): ListBookmarksRequest {
    return {
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListBookmarksRequest): unknown {
    const obj: any = {};
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListBookmarksRequest>): ListBookmarksRequest {
    return ListBookmarksRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListBookmarksRequest>): ListBookmarksRequest {
    const message = createBaseListBookmarksRequest();
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListBookmarksResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.bookmarks.push(Bookmark.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListBookmarksResponse {
    return {
      bookmarks: Array.isArray(object?.bookmarks) ? object.bookmarks.map((e: any) => Bookmark.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListBookmarksResponse): unknown {
    const obj: any = {};
    if (message.bookmarks?.length) {
      obj.bookmarks = message.bookmarks.map((e) => Bookmark.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListBookmarksResponse>): ListBookmarksResponse {
    return ListBookmarksResponse.fromPartial(base ?? {});
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBookmark();
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

          message.link = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
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
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.link !== "") {
      obj.link = message.link;
    }
    return obj;
  },

  create(base?: DeepPartial<Bookmark>): Bookmark {
    return Bookmark.fromPartial(base ?? {});
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
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          578365826: [
            new Uint8Array([
              25,
              58,
              8,
              98,
              111,
              111,
              107,
              109,
              97,
              114,
              107,
              34,
              13,
              47,
              118,
              49,
              47,
              98,
              111,
              111,
              107,
              109,
              97,
              114,
              107,
              115,
            ]),
          ],
        },
      },
    },
    /** DeleteBookmark deletes a bookmark. */
    deleteBookmark: {
      name: "DeleteBookmark",
      requestType: DeleteBookmarkRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              24,
              42,
              22,
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
              98,
              111,
              111,
              107,
              109,
              97,
              114,
              107,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    /** ListBookmarks lists bookmarks. */
    listBookmarks: {
      name: "ListBookmarks",
      requestType: ListBookmarksRequest,
      requestStream: false,
      responseType: ListBookmarksResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          578365826: [new Uint8Array([15, 18, 13, 47, 118, 49, 47, 98, 111, 111, 107, 109, 97, 114, 107, 115])],
        },
      },
    },
  },
} as const;

export interface BookmarkServiceImplementation<CallContextExt = {}> {
  /** CreateBookmark creates a new bookmark. */
  createBookmark(request: CreateBookmarkRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Bookmark>>;
  /** DeleteBookmark deletes a bookmark. */
  deleteBookmark(request: DeleteBookmarkRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Empty>>;
  /** ListBookmarks lists bookmarks. */
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
  /** ListBookmarks lists bookmarks. */
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
