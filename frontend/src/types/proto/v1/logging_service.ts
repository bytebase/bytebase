/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Timestamp } from "../google/protobuf/timestamp";

export const protobufPackage = "bytebase.v1";

export interface ListLogsRequest {
  /**
   * filter is the filter to apply on the list logs request,
   * follow the [ebnf](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form) syntax.
   * The field only support in filter:
   * - creator, example:
   *    - creator = "users/{email}"
   * - resource, example:
   *    - resource = "projects/{project resource id}"
   * - level, example:
   *    - level = "INFO"
   *    - level = "ERROR | WARN"
   * - action, example:
   *    - action = "ACTION_MEMBER_CREATE" | "ACTION_ISSUE_CREATE"
   * - create_time, example:
   *    - create_time <= "2022-01-01T12:00:00.000Z"
   *    - create_time >= "2022-01-01T12:00:00.000Z"
   * For example:
   * List the logs of type 'ACTION_ISSUE_COMMENT_CREATE' in issue/123: 'action="ACTION_ISSUE_COMMENT_CREATE", resource="issue/123"'
   */
  filter: string;
  /**
   * The order by of the log.
   * Only support order by create_time.
   * For example:
   *  - order_by = "create_time asc"
   *  - order_by = "create_time desc"
   */
  orderBy: string;
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
  /** The list of log entities. */
  logEntities: LogEntity[];
  /**
   * A token to retrieve next page of log entities.
   * Pass this value in the page_token field in the subsequent call to `ListLogs` method
   * to retrieve the next page of log entities.
   */
  nextPageToken: string;
}

export interface GetLogRequest {
  /**
   * The name of the log to retrieve.
   * Format: logs/{uid}
   */
  name: string;
}

export interface LogEntity {
  /**
   * The name of the log.
   * Format: logs/{uid}
   */
  name: string;
  /**
   * The creator of the log entity.
   * Format: users/{email}
   */
  creator: string;
  createTime?: Date | undefined;
  updateTime?: Date | undefined;
  action: LogEntity_Action;
  level: LogEntity_Level;
  /**
   * The name of the resource associated with this log entity. For example, the resource user associated with log entity type of "ACTION_MEMBER_CREATE".
   * Format:
   * For ACTION_MEMBER_*: users/{email}
   * For ACTION_ISSUE_*: issues/{issue uid}
   * For ACTION_PIPELINE_*: pipelines/{pipeline uid}
   * For ACTION_PROJECT_*: projects/{project resource id}
   * For ACTION_DATABASE_*: instances/{instance resource id}
   */
  resource: string;
  /**
   * The payload of the log entity.
   * TODO: use oneof
   */
  payload: string;
  comment: string;
}

export enum LogEntity_Action {
  /** ACTION_UNSPECIFIED - In worksapce resource only. */
  ACTION_UNSPECIFIED = 0,
  /**
   * ACTION_MEMBER_CREATE - Member related activity types.
   * Enum value 1 - 20
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
   * ACTION_ISSUE_CREATE - Issue related activity types.
   * Enum value 21 - 40
   *
   * ACTION_ISSUE_CREATE is the type for creating a new issue.
   */
  ACTION_ISSUE_CREATE = 21,
  /** ACTION_ISSUE_COMMENT_CREATE - ACTION_ISSUE_COMMENT_CREATE is the type for creating a new comment on an issue. */
  ACTION_ISSUE_COMMENT_CREATE = 22,
  /** ACTION_ISSUE_FIELD_UPDATE - ACTION_ISSUE_FIELD_UPDATE is the type for updating an issue's field. */
  ACTION_ISSUE_FIELD_UPDATE = 23,
  /** ACTION_ISSUE_STATUS_UPDATE - ACTION_ISSUE_STATUS_UPDATE is the type for updating an issue's status. */
  ACTION_ISSUE_STATUS_UPDATE = 24,
  /** ACTION_ISSUE_APPROVAL_NOTIFY - ACTION_ISSUE_APPROVAL_NOTIFY is the type for notifying issue approval. */
  ACTION_ISSUE_APPROVAL_NOTIFY = 25,
  /** ACTION_PIPELINE_STAGE_STATUS_UPDATE - ACTION_PIPELINE_STAGE_STATUS_UPDATE represents the pipeline stage status change, including BEGIN, END for now. */
  ACTION_PIPELINE_STAGE_STATUS_UPDATE = 31,
  /** ACTION_PIPELINE_TASK_STATUS_UPDATE - ACTION_PIPELINE_TASK_STATUS_UPDATE represents the pipeline task status change, including PENDING, PENDING_APPROVAL, RUNNING, SUCCESS, FAILURE, CANCELED for now. */
  ACTION_PIPELINE_TASK_STATUS_UPDATE = 32,
  /** ACTION_PIPELINE_TASK_FILE_COMMIT - ACTION_PIPELINE_TASK_FILE_COMMIT represents the VCS trigger to commit a file to update the task statement. */
  ACTION_PIPELINE_TASK_FILE_COMMIT = 33,
  /** ACTION_PIPELINE_TASK_STATEMENT_UPDATE - ACTION_PIPELINE_TASK_STATEMENT_UPDATE represents the manual update of the task statement. */
  ACTION_PIPELINE_TASK_STATEMENT_UPDATE = 34,
  /** ACTION_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE - ACTION_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE represents the manual update of the task earliest allowed time. */
  ACTION_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE = 35,
  /**
   * ACTION_PROJECT_REPOSITORY_PUSH - Project related activity types.
   * Enum value 41 - 60
   *
   * ACTION_PROJECT_REPOSITORY_PUSH represents Bytebase receiving a push event from the project repository.
   */
  ACTION_PROJECT_REPOSITORY_PUSH = 41,
  /** ACTION_PROJECT_MEMBER_CREATE - ACTION_PROJECT_MEMBER_CREATE represents adding a member to the project. */
  ACTION_PROJECT_MEMBER_CREATE = 42,
  /** ACTION_PROJECT_MEMBER_DELETE - ACTION_PROJECT_MEMBER_DELETE represents removing a member from the project. */
  ACTION_PROJECT_MEMBER_DELETE = 43,
  /** ACTION_PROJECT_MEMBER_ROLE_UPDATE - ACTION_PROJECT_MEMBER_ROLE_UPDATE represents updating the member role, for example, from ADMIN to MEMBER. */
  ACTION_PROJECT_MEMBER_ROLE_UPDATE = 44,
  /** ACTION_PROJECT_DATABASE_RECOVERY_PITR_DONE - ACTION_PROJECT_DATABASE_RECOVERY_PITR_DONE is the type for database PITR recovery done. */
  ACTION_PROJECT_DATABASE_RECOVERY_PITR_DONE = 45,
  /** ACTION_PROJECT_DATABASE_TRANSFER - ACTION_PROJECT_DATABASE_TRANSFER represents transfering the database from one project to another. */
  ACTION_PROJECT_DATABASE_TRANSFER = 46,
  /**
   * ACTION_DATABASE_SQL_EDITOR_QUERY - Database related activity types.
   * Enum value 61 - 80
   *
   * ACTION_DATABASE_SQL_EDITOR_QUERY is the type for SQL editor query.
   */
  ACTION_DATABASE_SQL_EDITOR_QUERY = 61,
  /** ACTION_DATABASE_SQL_EXPORT - ACTION_DATABASE_SQL_EXPORT is the type for exporting SQL. */
  ACTION_DATABASE_SQL_EXPORT = 62,
  UNRECOGNIZED = -1,
}

export function logEntity_ActionFromJSON(object: any): LogEntity_Action {
  switch (object) {
    case 0:
    case "ACTION_UNSPECIFIED":
      return LogEntity_Action.ACTION_UNSPECIFIED;
    case 1:
    case "ACTION_MEMBER_CREATE":
      return LogEntity_Action.ACTION_MEMBER_CREATE;
    case 2:
    case "ACTION_MEMBER_ROLE_UPDATE":
      return LogEntity_Action.ACTION_MEMBER_ROLE_UPDATE;
    case 3:
    case "ACTION_MEMBER_ACTIVATE":
      return LogEntity_Action.ACTION_MEMBER_ACTIVATE;
    case 4:
    case "ACTION_MEMBER_DEACTIVE":
      return LogEntity_Action.ACTION_MEMBER_DEACTIVE;
    case 21:
    case "ACTION_ISSUE_CREATE":
      return LogEntity_Action.ACTION_ISSUE_CREATE;
    case 22:
    case "ACTION_ISSUE_COMMENT_CREATE":
      return LogEntity_Action.ACTION_ISSUE_COMMENT_CREATE;
    case 23:
    case "ACTION_ISSUE_FIELD_UPDATE":
      return LogEntity_Action.ACTION_ISSUE_FIELD_UPDATE;
    case 24:
    case "ACTION_ISSUE_STATUS_UPDATE":
      return LogEntity_Action.ACTION_ISSUE_STATUS_UPDATE;
    case 25:
    case "ACTION_ISSUE_APPROVAL_NOTIFY":
      return LogEntity_Action.ACTION_ISSUE_APPROVAL_NOTIFY;
    case 31:
    case "ACTION_PIPELINE_STAGE_STATUS_UPDATE":
      return LogEntity_Action.ACTION_PIPELINE_STAGE_STATUS_UPDATE;
    case 32:
    case "ACTION_PIPELINE_TASK_STATUS_UPDATE":
      return LogEntity_Action.ACTION_PIPELINE_TASK_STATUS_UPDATE;
    case 33:
    case "ACTION_PIPELINE_TASK_FILE_COMMIT":
      return LogEntity_Action.ACTION_PIPELINE_TASK_FILE_COMMIT;
    case 34:
    case "ACTION_PIPELINE_TASK_STATEMENT_UPDATE":
      return LogEntity_Action.ACTION_PIPELINE_TASK_STATEMENT_UPDATE;
    case 35:
    case "ACTION_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE":
      return LogEntity_Action.ACTION_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE;
    case 41:
    case "ACTION_PROJECT_REPOSITORY_PUSH":
      return LogEntity_Action.ACTION_PROJECT_REPOSITORY_PUSH;
    case 42:
    case "ACTION_PROJECT_MEMBER_CREATE":
      return LogEntity_Action.ACTION_PROJECT_MEMBER_CREATE;
    case 43:
    case "ACTION_PROJECT_MEMBER_DELETE":
      return LogEntity_Action.ACTION_PROJECT_MEMBER_DELETE;
    case 44:
    case "ACTION_PROJECT_MEMBER_ROLE_UPDATE":
      return LogEntity_Action.ACTION_PROJECT_MEMBER_ROLE_UPDATE;
    case 45:
    case "ACTION_PROJECT_DATABASE_RECOVERY_PITR_DONE":
      return LogEntity_Action.ACTION_PROJECT_DATABASE_RECOVERY_PITR_DONE;
    case 46:
    case "ACTION_PROJECT_DATABASE_TRANSFER":
      return LogEntity_Action.ACTION_PROJECT_DATABASE_TRANSFER;
    case 61:
    case "ACTION_DATABASE_SQL_EDITOR_QUERY":
      return LogEntity_Action.ACTION_DATABASE_SQL_EDITOR_QUERY;
    case 62:
    case "ACTION_DATABASE_SQL_EXPORT":
      return LogEntity_Action.ACTION_DATABASE_SQL_EXPORT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return LogEntity_Action.UNRECOGNIZED;
  }
}

export function logEntity_ActionToJSON(object: LogEntity_Action): string {
  switch (object) {
    case LogEntity_Action.ACTION_UNSPECIFIED:
      return "ACTION_UNSPECIFIED";
    case LogEntity_Action.ACTION_MEMBER_CREATE:
      return "ACTION_MEMBER_CREATE";
    case LogEntity_Action.ACTION_MEMBER_ROLE_UPDATE:
      return "ACTION_MEMBER_ROLE_UPDATE";
    case LogEntity_Action.ACTION_MEMBER_ACTIVATE:
      return "ACTION_MEMBER_ACTIVATE";
    case LogEntity_Action.ACTION_MEMBER_DEACTIVE:
      return "ACTION_MEMBER_DEACTIVE";
    case LogEntity_Action.ACTION_ISSUE_CREATE:
      return "ACTION_ISSUE_CREATE";
    case LogEntity_Action.ACTION_ISSUE_COMMENT_CREATE:
      return "ACTION_ISSUE_COMMENT_CREATE";
    case LogEntity_Action.ACTION_ISSUE_FIELD_UPDATE:
      return "ACTION_ISSUE_FIELD_UPDATE";
    case LogEntity_Action.ACTION_ISSUE_STATUS_UPDATE:
      return "ACTION_ISSUE_STATUS_UPDATE";
    case LogEntity_Action.ACTION_ISSUE_APPROVAL_NOTIFY:
      return "ACTION_ISSUE_APPROVAL_NOTIFY";
    case LogEntity_Action.ACTION_PIPELINE_STAGE_STATUS_UPDATE:
      return "ACTION_PIPELINE_STAGE_STATUS_UPDATE";
    case LogEntity_Action.ACTION_PIPELINE_TASK_STATUS_UPDATE:
      return "ACTION_PIPELINE_TASK_STATUS_UPDATE";
    case LogEntity_Action.ACTION_PIPELINE_TASK_FILE_COMMIT:
      return "ACTION_PIPELINE_TASK_FILE_COMMIT";
    case LogEntity_Action.ACTION_PIPELINE_TASK_STATEMENT_UPDATE:
      return "ACTION_PIPELINE_TASK_STATEMENT_UPDATE";
    case LogEntity_Action.ACTION_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE:
      return "ACTION_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE";
    case LogEntity_Action.ACTION_PROJECT_REPOSITORY_PUSH:
      return "ACTION_PROJECT_REPOSITORY_PUSH";
    case LogEntity_Action.ACTION_PROJECT_MEMBER_CREATE:
      return "ACTION_PROJECT_MEMBER_CREATE";
    case LogEntity_Action.ACTION_PROJECT_MEMBER_DELETE:
      return "ACTION_PROJECT_MEMBER_DELETE";
    case LogEntity_Action.ACTION_PROJECT_MEMBER_ROLE_UPDATE:
      return "ACTION_PROJECT_MEMBER_ROLE_UPDATE";
    case LogEntity_Action.ACTION_PROJECT_DATABASE_RECOVERY_PITR_DONE:
      return "ACTION_PROJECT_DATABASE_RECOVERY_PITR_DONE";
    case LogEntity_Action.ACTION_PROJECT_DATABASE_TRANSFER:
      return "ACTION_PROJECT_DATABASE_TRANSFER";
    case LogEntity_Action.ACTION_DATABASE_SQL_EDITOR_QUERY:
      return "ACTION_DATABASE_SQL_EDITOR_QUERY";
    case LogEntity_Action.ACTION_DATABASE_SQL_EXPORT:
      return "ACTION_DATABASE_SQL_EXPORT";
    case LogEntity_Action.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum LogEntity_Level {
  LEVEL_UNSPECIFIED = 0,
  /** LEVEL_INFO - LEVEL_INFO is the type for information. */
  LEVEL_INFO = 1,
  /** LEVEL_WARNING - LEVEL_WARNING is the type for warning. */
  LEVEL_WARNING = 2,
  /** LEVEL_ERROR - LEVEL_ERROR is the type for error. */
  LEVEL_ERROR = 3,
  UNRECOGNIZED = -1,
}

export function logEntity_LevelFromJSON(object: any): LogEntity_Level {
  switch (object) {
    case 0:
    case "LEVEL_UNSPECIFIED":
      return LogEntity_Level.LEVEL_UNSPECIFIED;
    case 1:
    case "LEVEL_INFO":
      return LogEntity_Level.LEVEL_INFO;
    case 2:
    case "LEVEL_WARNING":
      return LogEntity_Level.LEVEL_WARNING;
    case 3:
    case "LEVEL_ERROR":
      return LogEntity_Level.LEVEL_ERROR;
    case -1:
    case "UNRECOGNIZED":
    default:
      return LogEntity_Level.UNRECOGNIZED;
  }
}

export function logEntity_LevelToJSON(object: LogEntity_Level): string {
  switch (object) {
    case LogEntity_Level.LEVEL_UNSPECIFIED:
      return "LEVEL_UNSPECIFIED";
    case LogEntity_Level.LEVEL_INFO:
      return "LEVEL_INFO";
    case LogEntity_Level.LEVEL_WARNING:
      return "LEVEL_WARNING";
    case LogEntity_Level.LEVEL_ERROR:
      return "LEVEL_ERROR";
    case LogEntity_Level.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

function createBaseListLogsRequest(): ListLogsRequest {
  return { filter: "", orderBy: "", pageSize: 0, pageToken: "" };
}

export const ListLogsRequest = {
  encode(message: ListLogsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.filter !== "") {
      writer.uint32(10).string(message.filter);
    }
    if (message.orderBy !== "") {
      writer.uint32(18).string(message.orderBy);
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

          message.filter = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.orderBy = reader.string();
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
      filter: isSet(object.filter) ? String(object.filter) : "",
      orderBy: isSet(object.orderBy) ? String(object.orderBy) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListLogsRequest): unknown {
    const obj: any = {};
    if (message.filter !== "") {
      obj.filter = message.filter;
    }
    if (message.orderBy !== "") {
      obj.orderBy = message.orderBy;
    }
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListLogsRequest>): ListLogsRequest {
    return ListLogsRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListLogsRequest>): ListLogsRequest {
    const message = createBaseListLogsRequest();
    message.filter = object.filter ?? "";
    message.orderBy = object.orderBy ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListLogsResponse(): ListLogsResponse {
  return { logEntities: [], nextPageToken: "" };
}

export const ListLogsResponse = {
  encode(message: ListLogsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.logEntities) {
      LogEntity.encode(v!, writer.uint32(10).fork()).ldelim();
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

          message.logEntities.push(LogEntity.decode(reader, reader.uint32()));
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
      logEntities: Array.isArray(object?.logEntities) ? object.logEntities.map((e: any) => LogEntity.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListLogsResponse): unknown {
    const obj: any = {};
    if (message.logEntities?.length) {
      obj.logEntities = message.logEntities.map((e) => LogEntity.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListLogsResponse>): ListLogsResponse {
    return ListLogsResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListLogsResponse>): ListLogsResponse {
    const message = createBaseListLogsResponse();
    message.logEntities = object.logEntities?.map((e) => LogEntity.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseGetLogRequest(): GetLogRequest {
  return { name: "" };
}

export const GetLogRequest = {
  encode(message: GetLogRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetLogRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetLogRequest();
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

  fromJSON(object: any): GetLogRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetLogRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<GetLogRequest>): GetLogRequest {
    return GetLogRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<GetLogRequest>): GetLogRequest {
    const message = createBaseGetLogRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseLogEntity(): LogEntity {
  return {
    name: "",
    creator: "",
    createTime: undefined,
    updateTime: undefined,
    action: 0,
    level: 0,
    resource: "",
    payload: "",
    comment: "",
  };
}

export const LogEntity = {
  encode(message: LogEntity, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.creator !== "") {
      writer.uint32(18).string(message.creator);
    }
    if (message.createTime !== undefined) {
      Timestamp.encode(toTimestamp(message.createTime), writer.uint32(26).fork()).ldelim();
    }
    if (message.updateTime !== undefined) {
      Timestamp.encode(toTimestamp(message.updateTime), writer.uint32(34).fork()).ldelim();
    }
    if (message.action !== 0) {
      writer.uint32(40).int32(message.action);
    }
    if (message.level !== 0) {
      writer.uint32(48).int32(message.level);
    }
    if (message.resource !== "") {
      writer.uint32(58).string(message.resource);
    }
    if (message.payload !== "") {
      writer.uint32(66).string(message.payload);
    }
    if (message.comment !== "") {
      writer.uint32(74).string(message.comment);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): LogEntity {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseLogEntity();
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

          message.creator = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.createTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.updateTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.action = reader.int32() as any;
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.level = reader.int32() as any;
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.resource = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.payload = reader.string();
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.comment = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): LogEntity {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      creator: isSet(object.creator) ? String(object.creator) : "",
      createTime: isSet(object.createTime) ? fromJsonTimestamp(object.createTime) : undefined,
      updateTime: isSet(object.updateTime) ? fromJsonTimestamp(object.updateTime) : undefined,
      action: isSet(object.action) ? logEntity_ActionFromJSON(object.action) : 0,
      level: isSet(object.level) ? logEntity_LevelFromJSON(object.level) : 0,
      resource: isSet(object.resource) ? String(object.resource) : "",
      payload: isSet(object.payload) ? String(object.payload) : "",
      comment: isSet(object.comment) ? String(object.comment) : "",
    };
  },

  toJSON(message: LogEntity): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
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
    if (message.action !== 0) {
      obj.action = logEntity_ActionToJSON(message.action);
    }
    if (message.level !== 0) {
      obj.level = logEntity_LevelToJSON(message.level);
    }
    if (message.resource !== "") {
      obj.resource = message.resource;
    }
    if (message.payload !== "") {
      obj.payload = message.payload;
    }
    if (message.comment !== "") {
      obj.comment = message.comment;
    }
    return obj;
  },

  create(base?: DeepPartial<LogEntity>): LogEntity {
    return LogEntity.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<LogEntity>): LogEntity {
    const message = createBaseLogEntity();
    message.name = object.name ?? "";
    message.creator = object.creator ?? "";
    message.createTime = object.createTime ?? undefined;
    message.updateTime = object.updateTime ?? undefined;
    message.action = object.action ?? 0;
    message.level = object.level ?? 0;
    message.resource = object.resource ?? "";
    message.payload = object.payload ?? "";
    message.comment = object.comment ?? "";
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
          578365826: [
            new Uint8Array([17, 18, 15, 47, 118, 49, 47, 108, 111, 103, 115, 58, 115, 101, 97, 114, 99, 104]),
          ],
        },
      },
    },
    getLog: {
      name: "GetLog",
      requestType: GetLogRequest,
      requestStream: false,
      responseType: LogEntity,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([19, 18, 17, 47, 118, 49, 47, 123, 110, 97, 109, 101, 61, 108, 111, 103, 115, 47, 42, 125]),
          ],
        },
      },
    },
  },
} as const;

export interface LoggingServiceImplementation<CallContextExt = {}> {
  listLogs(request: ListLogsRequest, context: CallContext & CallContextExt): Promise<DeepPartial<ListLogsResponse>>;
  getLog(request: GetLogRequest, context: CallContext & CallContextExt): Promise<DeepPartial<LogEntity>>;
}

export interface LoggingServiceClient<CallOptionsExt = {}> {
  listLogs(request: DeepPartial<ListLogsRequest>, options?: CallOptions & CallOptionsExt): Promise<ListLogsResponse>;
  getLog(request: DeepPartial<GetLogRequest>, options?: CallOptions & CallOptionsExt): Promise<LogEntity>;
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
