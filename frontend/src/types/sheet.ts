import {
  SheetId,
  Database,
  DatabaseId,
  Principal,
  Project,
  ProjectId,
  RowStatus,
  PrincipalId,
} from ".";

export type SheetVisibility = "PRIVATE" | "PROJECT" | "PUBLIC";

export type SheetSource = "BYTEBASE" | "GITLAB_SELF_HOST" | "GITHUB_COM";

export type SheetType = "SQL";

interface SheetVCSPayload {
  fileName: string;
  filePath: string;
  size: number;
  author: string;
  lastCommitId: string;
  lastSyncTs: number;
}

// eslint-disable-next-line @typescript-eslint/ban-types
type SheetEmptyPayload = {};

export type SheetPayload = SheetVCSPayload | SheetEmptyPayload;

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
  value: SheetVisibility;
};
