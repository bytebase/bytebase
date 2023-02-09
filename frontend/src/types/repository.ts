import { RepositoryId, VCSId } from "./id";
import { Project } from "./project";
import { VCS } from "./vcs";

export type Repository = {
  id: RepositoryId;

  // Related fields
  vcs: VCS;
  project: Project;

  // Domain specific fields
  // e.g. sample-project
  name: string;
  // e.g. bytebase/sample-project
  fullPath: string;
  // e.g. http://gitlab.bytebase.com/bytebase/sample-project
  webUrl: string;
  baseDirectory: string;
  branchFilter: string;
  filePathTemplate: string;
  schemaPathTemplate: string;
  sheetPathTemplate: string;
  enableSQLReviewCI: boolean;
  sqlReviewCIPullRequestURL: string;
  // e.g. In GitLab, this is the corresponding project id.
  externalId: string;
};

export type SQLReviewCISetup = {
  pullRequestURL: string;
};

export type RepositoryCreate = {
  // Related fields
  vcsId: VCSId;

  // Domain specific fields
  name: string;
  fullPath: string;
  webUrl: string;
  branchFilter: string;
  baseDirectory: string;
  filePathTemplate: string;
  schemaPathTemplate: string;
  sheetPathTemplate: string;
  externalId: string;
  accessToken: string;
  expiresTs: number;
  refreshToken: string;
};

export type RepositoryPatch = {
  baseDirectory?: string;
  branchFilter?: string;
  filePathTemplate?: string;
  schemaPathTemplate?: string;
  sheetPathTemplate?: string;
  enableSQLReviewCI?: boolean;
};

export type RepositoryConfig = {
  baseDirectory: string;
  branchFilter: string;
  filePathTemplate: string;
  schemaPathTemplate: string;
  sheetPathTemplate: string;
  enableSQLReviewCI: boolean;
};

export type ExternalRepositoryInfo = {
  // e.g. In GitLab, this is the corresponding project id. e.g. 123
  externalId: string;
  // e.g. sample-project
  name: string;
  // e.g. bytebase/sample-project
  fullPath: string;
  // e.g. http://gitlab.bytebase.com/bytebase/sample-project
  webUrl: string;
};
