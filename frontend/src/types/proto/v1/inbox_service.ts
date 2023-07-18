/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { FieldMask } from "../google/protobuf/field_mask";
import { LogEntity } from "./logging_service";

export const protobufPackage = "bytebase.v1";

export interface ListInboxRequest {
  /**
   * filter is the filter to apply on the list inbox request,
   * follow the [ebnf](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form) syntax.
   * The field only support in filter:
   * - "create_time" with ">=" operator, example:
   *    - create_time >= "2022-01-01T12:00:00.000Z"
   */
  filter: string;
  /**
   * Not used. The maximum number of inbox to return.
   * The service may return fewer than this value.
   * If unspecified, at most 100 log entries will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * Not used. A page token, received from a previous `ListInbox` call.
   * Provide this to retrieve the subsequent page.
   */
  pageToken: string;
}

export interface ListInboxResponse {
  /** The list of inbox messages. */
  inboxMessages: InboxMessage[];
  /**
   * A token to retrieve next page of inbox.
   * Pass this value in the page_token field in the subsequent call to `ListLogs` method
   * to retrieve the next page of log entities.
   */
  nextPageToken: string;
}

export interface GetInboxSummaryRequest {
}

export interface UpdateInboxRequest {
  /** The inbox message to update. */
  inboxMessage?:
    | InboxMessage
    | undefined;
  /** The list of fields to update. */
  updateMask?: string[] | undefined;
}

export interface InboxMessage {
  /** The message name in inbox/{uid} format. */
  name: string;
  activityUid: string;
  status: InboxMessage_Status;
  activity?: LogEntity | undefined;
}

export enum InboxMessage_Status {
  STATUS_UNSPECIFIED = 0,
  STATUS_UNREAD = 1,
  STATUS_READ = 2,
  UNRECOGNIZED = -1,
}

export function inboxMessage_StatusFromJSON(object: any): InboxMessage_Status {
  switch (object) {
    case 0:
    case "STATUS_UNSPECIFIED":
      return InboxMessage_Status.STATUS_UNSPECIFIED;
    case 1:
    case "STATUS_UNREAD":
      return InboxMessage_Status.STATUS_UNREAD;
    case 2:
    case "STATUS_READ":
      return InboxMessage_Status.STATUS_READ;
    case -1:
    case "UNRECOGNIZED":
    default:
      return InboxMessage_Status.UNRECOGNIZED;
  }
}

export function inboxMessage_StatusToJSON(object: InboxMessage_Status): string {
  switch (object) {
    case InboxMessage_Status.STATUS_UNSPECIFIED:
      return "STATUS_UNSPECIFIED";
    case InboxMessage_Status.STATUS_UNREAD:
      return "STATUS_UNREAD";
    case InboxMessage_Status.STATUS_READ:
      return "STATUS_READ";
    case InboxMessage_Status.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface InboxSummary {
  unread: number;
  unreadError: number;
}

function createBaseListInboxRequest(): ListInboxRequest {
  return { filter: "", pageSize: 0, pageToken: "" };
}

export const ListInboxRequest = {
  encode(message: ListInboxRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.filter !== "") {
      writer.uint32(10).string(message.filter);
    }
    if (message.pageSize !== 0) {
      writer.uint32(16).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(26).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListInboxRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListInboxRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.filter = reader.string();
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

  fromJSON(object: any): ListInboxRequest {
    return {
      filter: isSet(object.filter) ? String(object.filter) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListInboxRequest): unknown {
    const obj: any = {};
    if (message.filter !== "") {
      obj.filter = message.filter;
    }
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListInboxRequest>): ListInboxRequest {
    return ListInboxRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListInboxRequest>): ListInboxRequest {
    const message = createBaseListInboxRequest();
    message.filter = object.filter ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListInboxResponse(): ListInboxResponse {
  return { inboxMessages: [], nextPageToken: "" };
}

export const ListInboxResponse = {
  encode(message: ListInboxResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.inboxMessages) {
      InboxMessage.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListInboxResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListInboxResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.inboxMessages.push(InboxMessage.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListInboxResponse {
    return {
      inboxMessages: Array.isArray(object?.inboxMessages)
        ? object.inboxMessages.map((e: any) => InboxMessage.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListInboxResponse): unknown {
    const obj: any = {};
    if (message.inboxMessages?.length) {
      obj.inboxMessages = message.inboxMessages.map((e) => InboxMessage.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListInboxResponse>): ListInboxResponse {
    return ListInboxResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListInboxResponse>): ListInboxResponse {
    const message = createBaseListInboxResponse();
    message.inboxMessages = object.inboxMessages?.map((e) => InboxMessage.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseGetInboxSummaryRequest(): GetInboxSummaryRequest {
  return {};
}

export const GetInboxSummaryRequest = {
  encode(_: GetInboxSummaryRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetInboxSummaryRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetInboxSummaryRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): GetInboxSummaryRequest {
    return {};
  },

  toJSON(_: GetInboxSummaryRequest): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<GetInboxSummaryRequest>): GetInboxSummaryRequest {
    return GetInboxSummaryRequest.fromPartial(base ?? {});
  },

  fromPartial(_: DeepPartial<GetInboxSummaryRequest>): GetInboxSummaryRequest {
    const message = createBaseGetInboxSummaryRequest();
    return message;
  },
};

function createBaseUpdateInboxRequest(): UpdateInboxRequest {
  return { inboxMessage: undefined, updateMask: undefined };
}

export const UpdateInboxRequest = {
  encode(message: UpdateInboxRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.inboxMessage !== undefined) {
      InboxMessage.encode(message.inboxMessage, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateInboxRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateInboxRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.inboxMessage = InboxMessage.decode(reader, reader.uint32());
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

  fromJSON(object: any): UpdateInboxRequest {
    return {
      inboxMessage: isSet(object.inboxMessage) ? InboxMessage.fromJSON(object.inboxMessage) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateInboxRequest): unknown {
    const obj: any = {};
    if (message.inboxMessage !== undefined) {
      obj.inboxMessage = InboxMessage.toJSON(message.inboxMessage);
    }
    if (message.updateMask !== undefined) {
      obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask));
    }
    return obj;
  },

  create(base?: DeepPartial<UpdateInboxRequest>): UpdateInboxRequest {
    return UpdateInboxRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<UpdateInboxRequest>): UpdateInboxRequest {
    const message = createBaseUpdateInboxRequest();
    message.inboxMessage = (object.inboxMessage !== undefined && object.inboxMessage !== null)
      ? InboxMessage.fromPartial(object.inboxMessage)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseInboxMessage(): InboxMessage {
  return { name: "", activityUid: "", status: 0, activity: undefined };
}

export const InboxMessage = {
  encode(message: InboxMessage, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.activityUid !== "") {
      writer.uint32(18).string(message.activityUid);
    }
    if (message.status !== 0) {
      writer.uint32(24).int32(message.status);
    }
    if (message.activity !== undefined) {
      LogEntity.encode(message.activity, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): InboxMessage {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseInboxMessage();
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

          message.activityUid = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.status = reader.int32() as any;
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.activity = LogEntity.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): InboxMessage {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      activityUid: isSet(object.activityUid) ? String(object.activityUid) : "",
      status: isSet(object.status) ? inboxMessage_StatusFromJSON(object.status) : 0,
      activity: isSet(object.activity) ? LogEntity.fromJSON(object.activity) : undefined,
    };
  },

  toJSON(message: InboxMessage): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.activityUid !== "") {
      obj.activityUid = message.activityUid;
    }
    if (message.status !== 0) {
      obj.status = inboxMessage_StatusToJSON(message.status);
    }
    if (message.activity !== undefined) {
      obj.activity = LogEntity.toJSON(message.activity);
    }
    return obj;
  },

  create(base?: DeepPartial<InboxMessage>): InboxMessage {
    return InboxMessage.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<InboxMessage>): InboxMessage {
    const message = createBaseInboxMessage();
    message.name = object.name ?? "";
    message.activityUid = object.activityUid ?? "";
    message.status = object.status ?? 0;
    message.activity = (object.activity !== undefined && object.activity !== null)
      ? LogEntity.fromPartial(object.activity)
      : undefined;
    return message;
  },
};

function createBaseInboxSummary(): InboxSummary {
  return { unread: 0, unreadError: 0 };
}

export const InboxSummary = {
  encode(message: InboxSummary, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.unread !== 0) {
      writer.uint32(8).int32(message.unread);
    }
    if (message.unreadError !== 0) {
      writer.uint32(16).int32(message.unreadError);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): InboxSummary {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseInboxSummary();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.unread = reader.int32();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.unreadError = reader.int32();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): InboxSummary {
    return {
      unread: isSet(object.unread) ? Number(object.unread) : 0,
      unreadError: isSet(object.unreadError) ? Number(object.unreadError) : 0,
    };
  },

  toJSON(message: InboxSummary): unknown {
    const obj: any = {};
    if (message.unread !== 0) {
      obj.unread = Math.round(message.unread);
    }
    if (message.unreadError !== 0) {
      obj.unreadError = Math.round(message.unreadError);
    }
    return obj;
  },

  create(base?: DeepPartial<InboxSummary>): InboxSummary {
    return InboxSummary.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<InboxSummary>): InboxSummary {
    const message = createBaseInboxSummary();
    message.unread = object.unread ?? 0;
    message.unreadError = object.unreadError ?? 0;
    return message;
  },
};

export type InboxServiceDefinition = typeof InboxServiceDefinition;
export const InboxServiceDefinition = {
  name: "InboxService",
  fullName: "bytebase.v1.InboxService",
  methods: {
    listInbox: {
      name: "ListInbox",
      requestType: ListInboxRequest,
      requestStream: false,
      responseType: ListInboxResponse,
      responseStream: false,
      options: {
        _unknownFields: { 578365826: [new Uint8Array([11, 18, 9, 47, 118, 49, 47, 105, 110, 98, 111, 120])] },
      },
    },
    getInboxSummary: {
      name: "GetInboxSummary",
      requestType: GetInboxSummaryRequest,
      requestStream: false,
      responseType: InboxSummary,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([19, 18, 17, 47, 118, 49, 47, 105, 110, 98, 111, 120, 58, 115, 117, 109, 109, 97, 114, 121]),
          ],
        },
      },
    },
    updateInbox: {
      name: "UpdateInbox",
      requestType: UpdateInboxRequest,
      requestStream: false,
      responseType: InboxMessage,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([
              25,
              105,
              110,
              98,
              111,
              120,
              95,
              109,
              101,
              115,
              115,
              97,
              103,
              101,
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
              49,
              58,
              13,
              105,
              110,
              98,
              111,
              120,
              95,
              109,
              101,
              115,
              115,
              97,
              103,
              101,
              50,
              32,
              47,
              118,
              49,
              47,
              123,
              105,
              110,
              98,
              111,
              120,
              95,
              109,
              101,
              115,
              115,
              97,
              103,
              101,
              46,
              110,
              97,
              109,
              101,
              61,
              105,
              110,
              98,
              111,
              120,
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

export interface InboxServiceImplementation<CallContextExt = {}> {
  listInbox(request: ListInboxRequest, context: CallContext & CallContextExt): Promise<DeepPartial<ListInboxResponse>>;
  getInboxSummary(
    request: GetInboxSummaryRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<InboxSummary>>;
  updateInbox(request: UpdateInboxRequest, context: CallContext & CallContextExt): Promise<DeepPartial<InboxMessage>>;
}

export interface InboxServiceClient<CallOptionsExt = {}> {
  listInbox(request: DeepPartial<ListInboxRequest>, options?: CallOptions & CallOptionsExt): Promise<ListInboxResponse>;
  getInboxSummary(
    request: DeepPartial<GetInboxSummaryRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<InboxSummary>;
  updateInbox(request: DeepPartial<UpdateInboxRequest>, options?: CallOptions & CallOptionsExt): Promise<InboxMessage>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
