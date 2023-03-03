/* eslint-disable */
import * as _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.v1.activity";

export interface Activity {
}

export enum Activity_Type {
  TYPE_UNSPECIFIED = 0,
  /**
   * TYPE_ISSUE_CREATE - Issue related activity types.
   *
   * TYPE_ISSUE_CREATE represents creating an issue.
   */
  TYPE_ISSUE_CREATE = 1,
  /** TYPE_ISSUE_COMMENT_CREATE - TYPE_ISSUE_COMMENT_CREATE represents commenting on an issue. */
  TYPE_ISSUE_COMMENT_CREATE = 2,
  /** TYPE_ISSUE_FIELD_UPDATE - TYPE_ISSUE_FIELD_UPDATE represents updating the issue field, likes title, description, assignee, etc. */
  TYPE_ISSUE_FIELD_UPDATE = 3,
  /** TYPE_ISSUE_STATUS_UPDATE - TYPE_ISSUE_STATUS_UPDATE represents the issue status change, including OPEN, CLOSE, CANCEL fow now. */
  TYPE_ISSUE_STATUS_UPDATE = 4,
  /** TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE - TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE represents the pipeline stage status change, including BEGIN, END for now. */
  TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE = 5,
  /** TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE - TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE represents the pipeline task status change, including PENDING, PENDING_APPROVAL, RUNNING, SUCCESS, FAILURE, CANCELED for now. */
  TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE = 6,
  /** TYPE_ISSUE_PIPELINE_TASK_FILE_COMMIT - TYPE_ISSUE_PIPELINE_TASK_FILE_COMMIT represents the VCS trigger to commit a file to update the task statement. */
  TYPE_ISSUE_PIPELINE_TASK_FILE_COMMIT = 7,
  /** TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE - TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE represents the manual update of the task statement. */
  TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE = 8,
  /** TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE - TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE represents the manual update of the task earliest allowed time. */
  TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE = 9,
  /**
   * TYPE_MEMBER_CREATE - Member related activity types.
   *
   * TYPE_MEMBER_CREATE represents creating a members.
   */
  TYPE_MEMBER_CREATE = 10,
  /** TYPE_MEMBER_ROLE_UPDATE - TYPE_MEMBER_ROLE_UPDATE represents updating the member role, for example, from ADMIN to MEMBER. */
  TYPE_MEMBER_ROLE_UPDATE = 11,
  /** TYPE_MEMBER_ACTIVATE - TYPE_MEMBER_ACTIVATE represents activating a deactivated member. */
  TYPE_MEMBER_ACTIVATE = 12,
  /** TYPE_MEMBER_DEACTIVATE - TYPE_MEMBER_DEACTIVATE represents deactivating an active member. */
  TYPE_MEMBER_DEACTIVATE = 13,
  /**
   * TYPE_PROJECT_REPOSITORY_PUSH - Project related activity types.
   *
   * TYPE_PROJECT_REPOSITORY_PUSH represents Bytebase receiving a push event from the project repository.
   */
  TYPE_PROJECT_REPOSITORY_PUSH = 14,
  /** TYPE_PROJECT_DATABASE_TRANSFER - TYPE_PROJECT_DATABASE_TRANFER represents transfering the database from one project to another. */
  TYPE_PROJECT_DATABASE_TRANSFER = 15,
  /** TYPE_PROJECT_MEMBER_CREATE - TYPE_PROJECT_MEMBER_CREATE represents adding a member to the project. */
  TYPE_PROJECT_MEMBER_CREATE = 16,
  /** TYPE_PROJECT_MEMBER_DELETE - TYPE_PROJECT_MEMBER_DELETE represents removing a member from the project. */
  TYPE_PROJECT_MEMBER_DELETE = 17,
  /** TYPE_PROJECT_MEMBER_ROLE_UPDATE - TYPE_PROJECT_MEMBER_ROLE_UPDATE represents updating the member role, for example, from ADMIN to MEMBER. */
  TYPE_PROJECT_MEMBER_ROLE_UPDATE = 18,
  /**
   * TYPE_SQL_EDITOR_QUERY - SQL Editor related activity types.
   * TYPE_SQL_EDITOR_QUERY represents executing query in SQL Editor.
   */
  TYPE_SQL_EDITOR_QUERY = 19,
  /**
   * TYPE_DATABASE_RECOVERY_PITR_DONE - Database related activity types.
   * TYPE_DATABASE_RECOVERY_PITR_DONE represents the database recovery to a point in time is done.
   */
  TYPE_DATABASE_RECOVERY_PITR_DONE = 20,
  UNRECOGNIZED = -1,
}

export function activity_TypeFromJSON(object: any): Activity_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return Activity_Type.TYPE_UNSPECIFIED;
    case 1:
    case "TYPE_ISSUE_CREATE":
      return Activity_Type.TYPE_ISSUE_CREATE;
    case 2:
    case "TYPE_ISSUE_COMMENT_CREATE":
      return Activity_Type.TYPE_ISSUE_COMMENT_CREATE;
    case 3:
    case "TYPE_ISSUE_FIELD_UPDATE":
      return Activity_Type.TYPE_ISSUE_FIELD_UPDATE;
    case 4:
    case "TYPE_ISSUE_STATUS_UPDATE":
      return Activity_Type.TYPE_ISSUE_STATUS_UPDATE;
    case 5:
    case "TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE":
      return Activity_Type.TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE;
    case 6:
    case "TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE":
      return Activity_Type.TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE;
    case 7:
    case "TYPE_ISSUE_PIPELINE_TASK_FILE_COMMIT":
      return Activity_Type.TYPE_ISSUE_PIPELINE_TASK_FILE_COMMIT;
    case 8:
    case "TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE":
      return Activity_Type.TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE;
    case 9:
    case "TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE":
      return Activity_Type.TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE;
    case 10:
    case "TYPE_MEMBER_CREATE":
      return Activity_Type.TYPE_MEMBER_CREATE;
    case 11:
    case "TYPE_MEMBER_ROLE_UPDATE":
      return Activity_Type.TYPE_MEMBER_ROLE_UPDATE;
    case 12:
    case "TYPE_MEMBER_ACTIVATE":
      return Activity_Type.TYPE_MEMBER_ACTIVATE;
    case 13:
    case "TYPE_MEMBER_DEACTIVATE":
      return Activity_Type.TYPE_MEMBER_DEACTIVATE;
    case 14:
    case "TYPE_PROJECT_REPOSITORY_PUSH":
      return Activity_Type.TYPE_PROJECT_REPOSITORY_PUSH;
    case 15:
    case "TYPE_PROJECT_DATABASE_TRANSFER":
      return Activity_Type.TYPE_PROJECT_DATABASE_TRANSFER;
    case 16:
    case "TYPE_PROJECT_MEMBER_CREATE":
      return Activity_Type.TYPE_PROJECT_MEMBER_CREATE;
    case 17:
    case "TYPE_PROJECT_MEMBER_DELETE":
      return Activity_Type.TYPE_PROJECT_MEMBER_DELETE;
    case 18:
    case "TYPE_PROJECT_MEMBER_ROLE_UPDATE":
      return Activity_Type.TYPE_PROJECT_MEMBER_ROLE_UPDATE;
    case 19:
    case "TYPE_SQL_EDITOR_QUERY":
      return Activity_Type.TYPE_SQL_EDITOR_QUERY;
    case 20:
    case "TYPE_DATABASE_RECOVERY_PITR_DONE":
      return Activity_Type.TYPE_DATABASE_RECOVERY_PITR_DONE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Activity_Type.UNRECOGNIZED;
  }
}

export function activity_TypeToJSON(object: Activity_Type): string {
  switch (object) {
    case Activity_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case Activity_Type.TYPE_ISSUE_CREATE:
      return "TYPE_ISSUE_CREATE";
    case Activity_Type.TYPE_ISSUE_COMMENT_CREATE:
      return "TYPE_ISSUE_COMMENT_CREATE";
    case Activity_Type.TYPE_ISSUE_FIELD_UPDATE:
      return "TYPE_ISSUE_FIELD_UPDATE";
    case Activity_Type.TYPE_ISSUE_STATUS_UPDATE:
      return "TYPE_ISSUE_STATUS_UPDATE";
    case Activity_Type.TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE:
      return "TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE";
    case Activity_Type.TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE:
      return "TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE";
    case Activity_Type.TYPE_ISSUE_PIPELINE_TASK_FILE_COMMIT:
      return "TYPE_ISSUE_PIPELINE_TASK_FILE_COMMIT";
    case Activity_Type.TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE:
      return "TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE";
    case Activity_Type.TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE:
      return "TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE";
    case Activity_Type.TYPE_MEMBER_CREATE:
      return "TYPE_MEMBER_CREATE";
    case Activity_Type.TYPE_MEMBER_ROLE_UPDATE:
      return "TYPE_MEMBER_ROLE_UPDATE";
    case Activity_Type.TYPE_MEMBER_ACTIVATE:
      return "TYPE_MEMBER_ACTIVATE";
    case Activity_Type.TYPE_MEMBER_DEACTIVATE:
      return "TYPE_MEMBER_DEACTIVATE";
    case Activity_Type.TYPE_PROJECT_REPOSITORY_PUSH:
      return "TYPE_PROJECT_REPOSITORY_PUSH";
    case Activity_Type.TYPE_PROJECT_DATABASE_TRANSFER:
      return "TYPE_PROJECT_DATABASE_TRANSFER";
    case Activity_Type.TYPE_PROJECT_MEMBER_CREATE:
      return "TYPE_PROJECT_MEMBER_CREATE";
    case Activity_Type.TYPE_PROJECT_MEMBER_DELETE:
      return "TYPE_PROJECT_MEMBER_DELETE";
    case Activity_Type.TYPE_PROJECT_MEMBER_ROLE_UPDATE:
      return "TYPE_PROJECT_MEMBER_ROLE_UPDATE";
    case Activity_Type.TYPE_SQL_EDITOR_QUERY:
      return "TYPE_SQL_EDITOR_QUERY";
    case Activity_Type.TYPE_DATABASE_RECOVERY_PITR_DONE:
      return "TYPE_DATABASE_RECOVERY_PITR_DONE";
    case Activity_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

function createBaseActivity(): Activity {
  return {};
}

export const Activity = {
  encode(_: Activity, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Activity {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseActivity();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(_: any): Activity {
    return {};
  },

  toJSON(_: Activity): unknown {
    const obj: any = {};
    return obj;
  },

  fromPartial(_: DeepPartial<Activity>): Activity {
    const message = createBaseActivity();
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;
