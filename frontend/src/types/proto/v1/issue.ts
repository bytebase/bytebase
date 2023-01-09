/* eslint-disable */

export const protobufPackage = "bytebase.v1";

export enum IssueType {
  ISSUE_TYPE_UNSPECIFIED = 0,
  BB_ISSUE_GENERAL = 1,
  BB_ISSUE_DATABASE_CREATE = 11,
  BB_ISSUE_DATABASE_GRANT = 12,
  BB_ISSUE_DATABASE_SCHEMA_UPDATE = 13,
  BB_ISSUE_DATABASE_SCHEMA_UPDATE_GHOST = 14,
  BB_ISSUE_DATABASE_DATA_UPDATE = 15,
  BB_ISSUE_DATABASE_RESTORE_PITR = 16,
  BB_ISSUE_DATABASE_ROLLBACK = 17,
  BB_ISSUE_DATASOURCE_REQUEST = 21,
  UNRECOGNIZED = -1,
}

export function issueTypeFromJSON(object: any): IssueType {
  switch (object) {
    case 0:
    case "ISSUE_TYPE_UNSPECIFIED":
      return IssueType.ISSUE_TYPE_UNSPECIFIED;
    case 1:
    case "BB_ISSUE_GENERAL":
      return IssueType.BB_ISSUE_GENERAL;
    case 11:
    case "BB_ISSUE_DATABASE_CREATE":
      return IssueType.BB_ISSUE_DATABASE_CREATE;
    case 12:
    case "BB_ISSUE_DATABASE_GRANT":
      return IssueType.BB_ISSUE_DATABASE_GRANT;
    case 13:
    case "BB_ISSUE_DATABASE_SCHEMA_UPDATE":
      return IssueType.BB_ISSUE_DATABASE_SCHEMA_UPDATE;
    case 14:
    case "BB_ISSUE_DATABASE_SCHEMA_UPDATE_GHOST":
      return IssueType.BB_ISSUE_DATABASE_SCHEMA_UPDATE_GHOST;
    case 15:
    case "BB_ISSUE_DATABASE_DATA_UPDATE":
      return IssueType.BB_ISSUE_DATABASE_DATA_UPDATE;
    case 16:
    case "BB_ISSUE_DATABASE_RESTORE_PITR":
      return IssueType.BB_ISSUE_DATABASE_RESTORE_PITR;
    case 17:
    case "BB_ISSUE_DATABASE_ROLLBACK":
      return IssueType.BB_ISSUE_DATABASE_ROLLBACK;
    case 21:
    case "BB_ISSUE_DATASOURCE_REQUEST":
      return IssueType.BB_ISSUE_DATASOURCE_REQUEST;
    case -1:
    case "UNRECOGNIZED":
    default:
      return IssueType.UNRECOGNIZED;
  }
}

export function issueTypeToJSON(object: IssueType): string {
  switch (object) {
    case IssueType.ISSUE_TYPE_UNSPECIFIED:
      return "ISSUE_TYPE_UNSPECIFIED";
    case IssueType.BB_ISSUE_GENERAL:
      return "BB_ISSUE_GENERAL";
    case IssueType.BB_ISSUE_DATABASE_CREATE:
      return "BB_ISSUE_DATABASE_CREATE";
    case IssueType.BB_ISSUE_DATABASE_GRANT:
      return "BB_ISSUE_DATABASE_GRANT";
    case IssueType.BB_ISSUE_DATABASE_SCHEMA_UPDATE:
      return "BB_ISSUE_DATABASE_SCHEMA_UPDATE";
    case IssueType.BB_ISSUE_DATABASE_SCHEMA_UPDATE_GHOST:
      return "BB_ISSUE_DATABASE_SCHEMA_UPDATE_GHOST";
    case IssueType.BB_ISSUE_DATABASE_DATA_UPDATE:
      return "BB_ISSUE_DATABASE_DATA_UPDATE";
    case IssueType.BB_ISSUE_DATABASE_RESTORE_PITR:
      return "BB_ISSUE_DATABASE_RESTORE_PITR";
    case IssueType.BB_ISSUE_DATABASE_ROLLBACK:
      return "BB_ISSUE_DATABASE_ROLLBACK";
    case IssueType.BB_ISSUE_DATASOURCE_REQUEST:
      return "BB_ISSUE_DATASOURCE_REQUEST";
    case IssueType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
