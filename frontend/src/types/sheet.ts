import {
  SheetId,
  Database,
  DatabaseId,
  Principal,
  Project,
  ProjectId,
  RowStatus,
  PrincipalId,
  IssueId,
} from ".";
import { Sheet_Visibility } from "@/types/proto/v1/sheet_service";

export type SheetVisibility = "PRIVATE" | "PROJECT" | "PUBLIC";

export type SheetSource =
  | "BYTEBASE"
  | "GITLAB"
  | "GITHUB"
  | "BITBUCKET"
  | "BYTEBASE_ARTIFACT";

export type SheetType = "SQL";

interface SheetVCSPayload {
  fileName: string;
  filePath: string;
  size: number;
  author: string;
  lastCommitId: string;
  lastSyncTs: number;
}

/**
 * Mark a link from sheets to issues if a sheet is created in an issue via
 * "Upload SQL"
 */
export type SheetIssueBacktracePayload = {
  type: "bb.sheet.issue-backtrace";
  issueId: IssueId;
  issueName: string;
};

// eslint-disable-next-line @typescript-eslint/ban-types
type SheetEmptyPayload = {};

export type SheetPayload =
  | SheetVCSPayload
  | SheetIssueBacktracePayload
  | SheetEmptyPayload;

export interface Sheet {
  id: SheetId;

  // Standard fields
  rowStatus: RowStatus;
  creator: Principal;
  creatorId: PrincipalId;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Related fields
  projectId: ProjectId;
  project: Project;
  databaseId?: DatabaseId;
  database?: Database;

  // Domain fields
  name: string;
  statement: string;
  visibility: SheetVisibility;
  source: SheetSource;
  type: SheetType;
  starred: boolean;
  pinned: boolean;
  payload: SheetPayload;

  // The current size of statement in bytes.
  size: number;
}

export interface SheetUpsert {
  id?: SheetId;
  projectId: ProjectId;
  databaseId?: DatabaseId;
  name: string;
  statement: string;
  visibility?: SheetVisibility;
  payload?: SheetPayload;
}

export interface SheetCreate {
  projectId: ProjectId;
  databaseId?: DatabaseId;
  name: string;
  statement: string;
  visibility: SheetVisibility;
  payload: SheetPayload;
  source: SheetSource;
}

export interface SheetPatch {
  id: SheetId;
  name?: string;
  statement?: string;
  visibility?: SheetVisibility;
  rowStatus?: RowStatus;
  payload?: SheetPayload;
}

export type AccessOption = {
  label: string;
  description: string;
  value: Sheet_Visibility;
};
