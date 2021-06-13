import { RepositoryId, VCSId, ProjectId } from "./id";
import { Principal } from "./principal";
import { Project } from "./project";
import { VCS } from "./vcs";

export type Repository = {
  id: RepositoryId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Related fields
  vcs: VCS;
  project: Project;

  // Domain specific fields
  // e.g. sample-project
  name: string;
  // e.g. bytebase/sample-project
  fullPath: string;
  // e.g. http://gitlab.bytebase.com/bytebase/sample-project
  webURL: string;
  baseDirectory: string;
  branchFilter: string;
  // e.g. In GitLab, this is the corresponding project id.
  externalId: string;
};

export type RepositoryCreate = {
  // Related fields
  vcsId: VCSId;
  projectId: ProjectId;

  // Domain specific fields
  name: string;
  fullPath: string;
  webURL: string;
  baseDirectory: string;
  branchFilter: string;
  externalId: string;
  webhookId: string;
  webhookURL: string;
};

export type RepositoryConfig = {
  baseDirectory: string;
  branchFilter: string;
};

export type ExternalRepositoryInfo = {
  // e.g. In GitLab, this is the corresponding project id. e.g. 123
  externalId: string;
  // e.g. sample-project
  name: string;
  // e.g. bytebase/sample-project
  fullPath: string;
  // e.g. http://gitlab.bytebase.com/bytebase/sample-project
  webURL: string;
};

export type WebhookInfo = {
  id: string;
  url: string;
};
