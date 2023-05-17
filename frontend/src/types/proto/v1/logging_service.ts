/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Struct } from "../google/protobuf/struct";
import { Timestamp } from "../google/protobuf/timestamp";

export const protobufPackage = "bytebase.v1";

export interface ListLogsRequest {
  /**
   * The parent resource name.
   * Format:
   * projects/{project}
   * workspaces/{workspace}
   */
  parent: string;
  /**
   * filter is the filter to apply on the list logs request,
   * follow the [ebnf](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form) syntax.
   * The field only support in filter:
   * - creator
   * - container
   * - level
   * - action
   * For example:
   * List the logs of type 'ACTION_ISSUE_COMMENT_CREATE' in issue/123: 'action="ACTION_ISSUE_COMMENT_CREATE", container="issue/123"'
   */
  filter: string;
  /**
   * Not used. The maximum number of logs to return.
   * The service may return fewer than this value.
   * If unspecified, at most 100 log entries will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * Not used. A page token, received from a previous `ListLogs` call.
   * Provide this to retrieve the subsequent page.
   */
  pageToken: string;
}

export interface ListLogsResponse {
  /** The list of log entries. */
  logEntries: LogEntry[];
  /**
   * A token to retrieve next page of log entries.
   * Pass this value in the page_token field in the subsequent call to `ListLogs` method
   * to retrieve the next page of log entries.
   */
  nextPageToken: string;
}

export interface LogEntry {
  /**
   * The creator of the log entry.
   * Format: user:{emailid}
   */
  creator: string;
  /** The timestamp when the backup resource was created initally. */
  createTime?: Date;
  /** The timestamp when the backup resource was updated. */
  updateTime?: Date;
  action: LogEntry_Action;
  level: LogEntry_Level;
  /**
   * The name of the resource associated with this log entry. For example, the resource user associated with log entry type of "ACTION_MEMBER_CREATE".
   * Format:
   * For ACTION_MEMBER_*: user:emailid
   * For ACTION_ISSUE_*: issues/{issue}
   * For ACTION_PIPELINE_*: pipelines/{pipeline}
   * For ACTION_PROJECT_*: projects/{project}
   * For ACTION_SQL_EDITOR_QUERY: workspaces/{workspace} OR projects/{project}
   */
  resourceName: string;
  /** The payload of the log entry. */
  jsonPayload?: { [key: string]: any };
}

export enum LogEntry_Action {
  ACTION_UNSPECIFIED = 0,
  /**
   * ACTION_MEMBER_CREATE - In worksapce resource only.
   *
   * ACTION_MEMBER_CREATE is the type for creating a new member.
   */
  ACTION_MEMBER_CREATE = 1,
  /** ACTION_MEMBER_ROLE_UPDATE - ACTION_MEMBER_ROLE_UPDATE is the type for updating a member's role. */
  ACTION_MEMBER_ROLE_UPDATE = 2,
  /** ACTION_MEMBER_ACTIVATE - ACTION_MEMBER_ACTIVATE_UPDATE is the type for activating members. */
  ACTION_MEMBER_ACTIVATE = 3,
  /** ACTION_MEMBER_DEACTIVE - ACTION_MEMBER_DEACTIVE is the type for deactiving members. */
  ACTION_MEMBER_DEACTIVE = 4,
  /**
   * ACTION_ISSUE_CREATE - In project resource only.
   *
   * ACTION_ISSUE_CREATE is the type for creating a new issue.
   */
  ACTION_ISSUE_CREATE = 5,
  /** ACTION_ISSUE_COMMENT_CREATE - ACTION_ISSUE_COMMENT_CREATE is the type for creating a new comment on an issue. */
  ACTION_ISSUE_COMMENT_CREATE = 6,
  /** ACTION_ISSUE_FIELD_UPDATE - ACTION_ISSUE_FIELD_UPDATE is the type for updating an issue's field. */
  ACTION_ISSUE_FIELD_UPDATE = 7,
  /** ACTION_ISSUE_STATUS_UPDATE - ACTION_ISSUE_STATUS_UPDATE is the type for updating an issue's status. */
  ACTION_ISSUE_STATUS_UPDATE = 8,
  /** ACTION_PIPELINE_STAGE_STATUS_UPDATE - ACTION_PIPELINE_STAGE_STATUS_UPDATE is the type for stage begins or ends. */
  ACTION_PIPELINE_STAGE_STATUS_UPDATE = 9,
  /** ACTION_PIPELINE_TASK_STATUS_UPDATE - ACTION_PIPELINE_TASK_STATUS_UPDATE is the type for updating pipeline task status. */
  ACTION_PIPELINE_TASK_STATUS_UPDATE = 10,
  /** ACTION_PIPELINE_TASK_FILE_COMMIT - ACTION_PIPELINE_TASK_FILE_COMMIT is the type for committing pipeline task files. */
  ACTION_PIPELINE_TASK_FILE_COMMIT = 11,
  /** ACTION_PIPELINE_TASK_STATEMENT_UPDATE - ACTION_PIPELINE_TASK_STATEMENT_UPDATE is the type for updating pipeline task SQL statement. */
  ACTION_PIPELINE_TASK_STATEMENT_UPDATE = 12,
  /** ACITON_PIPELINE_TASK_EARLIEST_ALLOWED_DATE_UPDATE - ACTION_PIPELINE_TASK_EARLIEST_ALLOWED_DATE_UPDATE is the type for updating pipeline task the earliest allowed time. */
  ACITON_PIPELINE_TASK_EARLIEST_ALLOWED_DATE_UPDATE = 13,
  /** ACTION_PROJECT_MEMBER_CREATE - ACTION_PROJECT_MEMBER_CREATE is the type for creating a new project member. */
  ACTION_PROJECT_MEMBER_CREATE = 14,
  /** ACTION_PROJECT_MEMBER_ROLE_UPDATE - ACTION_PROJECT_MEMBER_ROLE_UPDATE is the type for updating a project member's role. */
  ACTION_PROJECT_MEMBER_ROLE_UPDATE = 15,
  /** ACTION_PROJECT_MEMBER_DELETE - ACTION_PROJECT_MEMBER_DELETE is the type for deleting a project member. */
  ACTION_PROJECT_MEMBER_DELETE = 16,
  /** ACTION_PROJECT_REPOSITORY_PUSH - ACTION_PROJECT_REPOSITORY_PUSH is the type for pushing to a project repository. */
  ACTION_PROJECT_REPOSITORY_PUSH = 17,
  /** ACTION_PROJECT_DTABASE_TRANSFER - ACTION_PROJECT_DATABASE_TRANSFER is the type for transferring a database to a project. */
  ACTION_PROJECT_DTABASE_TRANSFER = 18,
  /** ACTION_PROJECT_DATABASE_RECOVERY_PITR_DONE - ACTION_PROJECT_DATABASE_RECOVERY_PITR_DONE is the type for database PITR recovery done. */
  ACTION_PROJECT_DATABASE_RECOVERY_PITR_DONE = 19,
  /**
   * ACTION_SQL_EDITOR_QUERY - Both in workspace and project resource.
   *
   * ACTION_SQL_EDITOR_QUERY is the type for SQL editor query.
   * If user runs SQL in Read-only mode, this action will belong to project resource.
   * If user runs SQL in Read-write mode, this action will belong to workspace resource.
   */
  ACTION_SQL_EDITOR_QUERY = 20,
  UNRECOGNIZED = -1,
}

export function logEntry_ActionFromJSON(object: any): LogEntry_Action {
  switch (object) {
    case 0:
    case "ACTION_UNSPECIFIED":
      return LogEntry_Action.ACTION_UNSPECIFIED;
    case 1:
    case "ACTION_MEMBER_CREATE":
      return LogEntry_Action.ACTION_MEMBER_CREATE;
    case 2:
    case "ACTION_MEMBER_ROLE_UPDATE":
      return LogEntry_Action.ACTION_MEMBER_ROLE_UPDATE;
    case 3:
    case "ACTION_MEMBER_ACTIVATE":
      return LogEntry_Action.ACTION_MEMBER_ACTIVATE;
    case 4:
    case "ACTION_MEMBER_DEACTIVE":
      return LogEntry_Action.ACTION_MEMBER_DEACTIVE;
    case 5:
    case "ACTION_ISSUE_CREATE":
      return LogEntry_Action.ACTION_ISSUE_CREATE;
    case 6:
    case "ACTION_ISSUE_COMMENT_CREATE":
      return LogEntry_Action.ACTION_ISSUE_COMMENT_CREATE;
    case 7:
    case "ACTION_ISSUE_FIELD_UPDATE":
      return LogEntry_Action.ACTION_ISSUE_FIELD_UPDATE;
    case 8:
    case "ACTION_ISSUE_STATUS_UPDATE":
      return LogEntry_Action.ACTION_ISSUE_STATUS_UPDATE;
    case 9:
    case "ACTION_PIPELINE_STAGE_STATUS_UPDATE":
      return LogEntry_Action.ACTION_PIPELINE_STAGE_STATUS_UPDATE;
    case 10:
    case "ACTION_PIPELINE_TASK_STATUS_UPDATE":
      return LogEntry_Action.ACTION_PIPELINE_TASK_STATUS_UPDATE;
    case 11:
    case "ACTION_PIPELINE_TASK_FILE_COMMIT":
      return LogEntry_Action.ACTION_PIPELINE_TASK_FILE_COMMIT;
    case 12:
    case "ACTION_PIPELINE_TASK_STATEMENT_UPDATE":
      return LogEntry_Action.ACTION_PIPELINE_TASK_STATEMENT_UPDATE;
    case 13:
    case "ACITON_PIPELINE_TASK_EARLIEST_ALLOWED_DATE_UPDATE":
      return LogEntry_Action.ACITON_PIPELINE_TASK_EARLIEST_ALLOWED_DATE_UPDATE;
    case 14:
    case "ACTION_PROJECT_MEMBER_CREATE":
      return LogEntry_Action.ACTION_PROJECT_MEMBER_CREATE;
    case 15:
    case "ACTION_PROJECT_MEMBER_ROLE_UPDATE":
      return LogEntry_Action.ACTION_PROJECT_MEMBER_ROLE_UPDATE;
    case 16:
    case "ACTION_PROJECT_MEMBER_DELETE":
      return LogEntry_Action.ACTION_PROJECT_MEMBER_DELETE;
    case 17:
    case "ACTION_PROJECT_REPOSITORY_PUSH":
      return LogEntry_Action.ACTION_PROJECT_REPOSITORY_PUSH;
    case 18:
    case "ACTION_PROJECT_DTABASE_TRANSFER":
      return LogEntry_Action.ACTION_PROJECT_DTABASE_TRANSFER;
    case 19:
    case "ACTION_PROJECT_DATABASE_RECOVERY_PITR_DONE":
      return LogEntry_Action.ACTION_PROJECT_DATABASE_RECOVERY_PITR_DONE;
    case 20:
    case "ACTION_SQL_EDITOR_QUERY":
      return LogEntry_Action.ACTION_SQL_EDITOR_QUERY;
    case -1:
    case "UNRECOGNIZED":
    default:
      return LogEntry_Action.UNRECOGNIZED;
  }
}

export function logEntry_ActionToJSON(object: LogEntry_Action): string {
  switch (object) {
    case LogEntry_Action.ACTION_UNSPECIFIED:
      return "ACTION_UNSPECIFIED";
    case LogEntry_Action.ACTION_MEMBER_CREATE:
      return "ACTION_MEMBER_CREATE";
    case LogEntry_Action.ACTION_MEMBER_ROLE_UPDATE:
      return "ACTION_MEMBER_ROLE_UPDATE";
    case LogEntry_Action.ACTION_MEMBER_ACTIVATE:
      return "ACTION_MEMBER_ACTIVATE";
    case LogEntry_Action.ACTION_MEMBER_DEACTIVE:
      return "ACTION_MEMBER_DEACTIVE";
    case LogEntry_Action.ACTION_ISSUE_CREATE:
      return "ACTION_ISSUE_CREATE";
    case LogEntry_Action.ACTION_ISSUE_COMMENT_CREATE:
      return "ACTION_ISSUE_COMMENT_CREATE";
    case LogEntry_Action.ACTION_ISSUE_FIELD_UPDATE:
      return "ACTION_ISSUE_FIELD_UPDATE";
    case LogEntry_Action.ACTION_ISSUE_STATUS_UPDATE:
      return "ACTION_ISSUE_STATUS_UPDATE";
    case LogEntry_Action.ACTION_PIPELINE_STAGE_STATUS_UPDATE:
      return "ACTION_PIPELINE_STAGE_STATUS_UPDATE";
    case LogEntry_Action.ACTION_PIPELINE_TASK_STATUS_UPDATE:
      return "ACTION_PIPELINE_TASK_STATUS_UPDATE";
    case LogEntry_Action.ACTION_PIPELINE_TASK_FILE_COMMIT:
      return "ACTION_PIPELINE_TASK_FILE_COMMIT";
    case LogEntry_Action.ACTION_PIPELINE_TASK_STATEMENT_UPDATE:
      return "ACTION_PIPELINE_TASK_STATEMENT_UPDATE";
    case LogEntry_Action.ACITON_PIPELINE_TASK_EARLIEST_ALLOWED_DATE_UPDATE:
      return "ACITON_PIPELINE_TASK_EARLIEST_ALLOWED_DATE_UPDATE";
    case LogEntry_Action.ACTION_PROJECT_MEMBER_CREATE:
      return "ACTION_PROJECT_MEMBER_CREATE";
    case LogEntry_Action.ACTION_PROJECT_MEMBER_ROLE_UPDATE:
      return "ACTION_PROJECT_MEMBER_ROLE_UPDATE";
    case LogEntry_Action.ACTION_PROJECT_MEMBER_DELETE:
      return "ACTION_PROJECT_MEMBER_DELETE";
    case LogEntry_Action.ACTION_PROJECT_REPOSITORY_PUSH:
      return "ACTION_PROJECT_REPOSITORY_PUSH";
    case LogEntry_Action.ACTION_PROJECT_DTABASE_TRANSFER:
      return "ACTION_PROJECT_DTABASE_TRANSFER";
    case LogEntry_Action.ACTION_PROJECT_DATABASE_RECOVERY_PITR_DONE:
      return "ACTION_PROJECT_DATABASE_RECOVERY_PITR_DONE";
    case LogEntry_Action.ACTION_SQL_EDITOR_QUERY:
      return "ACTION_SQL_EDITOR_QUERY";
    case LogEntry_Action.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum LogEntry_Level {
  LEVEL_UNSPECIFIED = 0,
  /** LEVEL_INFO - LEVEL_INFO is the type for information. */
  LEVEL_INFO = 1,
  /** LEVEL_WARNING - LEVEL_WARNING is the type for warning. */
  LEVEL_WARNING = 2,
  /** LEVEL_ERROR - LEVEL_ERROR is the type for error. */
  LEVEL_ERROR = 3,
  UNRECOGNIZED = -1,
}

export function logEntry_LevelFromJSON(object: any): LogEntry_Level {
  switch (object) {
    case 0:
    case "LEVEL_UNSPECIFIED":
      return LogEntry_Level.LEVEL_UNSPECIFIED;
    case 1:
    case "LEVEL_INFO":
      return LogEntry_Level.LEVEL_INFO;
    case 2:
    case "LEVEL_WARNING":
      return LogEntry_Level.LEVEL_WARNING;
    case 3:
    case "LEVEL_ERROR":
      return LogEntry_Level.LEVEL_ERROR;
    case -1:
    case "UNRECOGNIZED":
    default:
      return LogEntry_Level.UNRECOGNIZED;
  }
}

export function logEntry_LevelToJSON(object: LogEntry_Level): string {
  switch (object) {
    case LogEntry_Level.LEVEL_UNSPECIFIED:
      return "LEVEL_UNSPECIFIED";
    case LogEntry_Level.LEVEL_INFO:
      return "LEVEL_INFO";
    case LogEntry_Level.LEVEL_WARNING:
      return "LEVEL_WARNING";
    case LogEntry_Level.LEVEL_ERROR:
      return "LEVEL_ERROR";
    case LogEntry_Level.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

function createBaseListLogsRequest(): ListLogsRequest {
  return { parent: "", filter: "", pageSize: 0, pageToken: "" };
}

export const ListLogsRequest = {
  encode(message: ListLogsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): ListLogsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListLogsRequest();
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

  fromJSON(object: any): ListLogsRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      filter: isSet(object.filter) ? String(object.filter) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListLogsRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.filter !== undefined && (obj.filter = message.filter);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    return obj;
  },

  create(base?: DeepPartial<ListLogsRequest>): ListLogsRequest {
    return ListLogsRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListLogsRequest>): ListLogsRequest {
    const message = createBaseListLogsRequest();
    message.parent = object.parent ?? "";
    message.filter = object.filter ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListLogsResponse(): ListLogsResponse {
  return { logEntries: [], nextPageToken: "" };
}

export const ListLogsResponse = {
  encode(message: ListLogsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.logEntries) {
      LogEntry.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListLogsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListLogsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.logEntries.push(LogEntry.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListLogsResponse {
    return {
      logEntries: Array.isArray(object?.logEntries) ? object.logEntries.map((e: any) => LogEntry.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListLogsResponse): unknown {
    const obj: any = {};
    if (message.logEntries) {
      obj.logEntries = message.logEntries.map((e) => e ? LogEntry.toJSON(e) : undefined);
    } else {
      obj.logEntries = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  create(base?: DeepPartial<ListLogsResponse>): ListLogsResponse {
    return ListLogsResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListLogsResponse>): ListLogsResponse {
    const message = createBaseListLogsResponse();
    message.logEntries = object.logEntries?.map((e) => LogEntry.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseLogEntry(): LogEntry {
  return {
    creator: "",
    createTime: undefined,
    updateTime: undefined,
    action: 0,
    level: 0,
    resourceName: "",
    jsonPayload: undefined,
  };
}

export const LogEntry = {
  encode(message: LogEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.createTime !== undefined) {
      Timestamp.encode(toTimestamp(message.createTime), writer.uint32(18).fork()).ldelim();
    }
    if (message.updateTime !== undefined) {
      Timestamp.encode(toTimestamp(message.updateTime), writer.uint32(26).fork()).ldelim();
    }
    if (message.action !== 0) {
      writer.uint32(32).int32(message.action);
    }
    if (message.level !== 0) {
      writer.uint32(40).int32(message.level);
    }
    if (message.resourceName !== "") {
      writer.uint32(50).string(message.resourceName);
    }
    if (message.jsonPayload !== undefined) {
      Struct.encode(Struct.wrap(message.jsonPayload), writer.uint32(58).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): LogEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseLogEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.creator = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.createTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.updateTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.action = reader.int32() as any;
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.level = reader.int32() as any;
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.resourceName = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.jsonPayload = Struct.unwrap(Struct.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): LogEntry {
    return {
      creator: isSet(object.creator) ? String(object.creator) : "",
      createTime: isSet(object.createTime) ? fromJsonTimestamp(object.createTime) : undefined,
      updateTime: isSet(object.updateTime) ? fromJsonTimestamp(object.updateTime) : undefined,
      action: isSet(object.action) ? logEntry_ActionFromJSON(object.action) : 0,
      level: isSet(object.level) ? logEntry_LevelFromJSON(object.level) : 0,
      resourceName: isSet(object.resourceName) ? String(object.resourceName) : "",
      jsonPayload: isObject(object.jsonPayload) ? object.jsonPayload : undefined,
    };
  },

  toJSON(message: LogEntry): unknown {
    const obj: any = {};
    message.creator !== undefined && (obj.creator = message.creator);
    message.createTime !== undefined && (obj.createTime = message.createTime.toISOString());
    message.updateTime !== undefined && (obj.updateTime = message.updateTime.toISOString());
    message.action !== undefined && (obj.action = logEntry_ActionToJSON(message.action));
    message.level !== undefined && (obj.level = logEntry_LevelToJSON(message.level));
    message.resourceName !== undefined && (obj.resourceName = message.resourceName);
    message.jsonPayload !== undefined && (obj.jsonPayload = message.jsonPayload);
    return obj;
  },

  create(base?: DeepPartial<LogEntry>): LogEntry {
    return LogEntry.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<LogEntry>): LogEntry {
    const message = createBaseLogEntry();
    message.creator = object.creator ?? "";
    message.createTime = object.createTime ?? undefined;
    message.updateTime = object.updateTime ?? undefined;
    message.action = object.action ?? 0;
    message.level = object.level ?? 0;
    message.resourceName = object.resourceName ?? "";
    message.jsonPayload = object.jsonPayload ?? undefined;
    return message;
  },
};

export type LoggingServiceDefinition = typeof LoggingServiceDefinition;
export const LoggingServiceDefinition = {
  name: "LoggingService",
  fullName: "bytebase.v1.LoggingService",
  methods: {
    listLogs: {
      name: "ListLogs",
      requestType: ListLogsRequest,
      requestStream: false,
      responseType: ListLogsResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([6, 112, 97, 114, 101, 110, 116])],
          578365826: [
            new Uint8Array([
              64,
              90,
              32,
              34,
              30,
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
              119,
              111,
              114,
              107,
              115,
              112,
              97,
              99,
              101,
              115,
              47,
              42,
              125,
              47,
              108,
              111,
              103,
              115,
              34,
              28,
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
              108,
              111,
              103,
              115,
            ]),
          ],
        },
      },
    },
  },
} as const;

export interface LoggingServiceImplementation<CallContextExt = {}> {
  listLogs(request: ListLogsRequest, context: CallContext & CallContextExt): Promise<DeepPartial<ListLogsResponse>>;
}

export interface LoggingServiceClient<CallOptionsExt = {}> {
  listLogs(request: DeepPartial<ListLogsRequest>, options?: CallOptions & CallOptionsExt): Promise<ListLogsResponse>;
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

function isObject(value: any): boolean {
  return typeof value === "object" && value !== null;
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
