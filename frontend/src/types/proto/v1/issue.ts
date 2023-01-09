/* eslint-disable */

export const protobufPackage = "bytebase.v1";

export enum IssueType {
  ISSUE_TYPE_UNSPECIFIED = 0,
  BB_ISSUE_GENERAL = 1,
  BB_ISSUE_DATABASE_CREATE = 2,
  BB_ISSUE_DATABASE_SCHEMA_UPDATE = 3,
  BB_ISSUE_DATABASE_SCHEMA_UPDATE_GHOST = 4,
  BB_ISSUE_DATABASE_DATA_UPDATE = 5,
  BB_ISSUE_DATABASE_RESTORE_PITR = 6,
  BB_ISSUE_DATABASE_ROLLBACK = 7,
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
    case 2:
    case "BB_ISSUE_DATABASE_CREATE":
      return IssueType.BB_ISSUE_DATABASE_CREATE;
    case 3:
    case "BB_ISSUE_DATABASE_SCHEMA_UPDATE":
      return IssueType.BB_ISSUE_DATABASE_SCHEMA_UPDATE;
    case 4:
    case "BB_ISSUE_DATABASE_SCHEMA_UPDATE_GHOST":
      return IssueType.BB_ISSUE_DATABASE_SCHEMA_UPDATE_GHOST;
    case 5:
    case "BB_ISSUE_DATABASE_DATA_UPDATE":
      return IssueType.BB_ISSUE_DATABASE_DATA_UPDATE;
    case 6:
    case "BB_ISSUE_DATABASE_RESTORE_PITR":
      return IssueType.BB_ISSUE_DATABASE_RESTORE_PITR;
    case 7:
    case "BB_ISSUE_DATABASE_ROLLBACK":
      return IssueType.BB_ISSUE_DATABASE_ROLLBACK;
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
    case IssueType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
