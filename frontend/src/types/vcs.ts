import { VCSProvider_Type } from "@/types/proto/v1/vcs_provider_service";

// Backend uses the same ENUM for GitLab/GitHub SaaS and self-hosted. Because they are based on the
// same codebase.
export type VCSType = "GITLAB" | "GITHUB" | "BITBUCKET";

// When configuring the VCS, we split the SaaS and self-hosted into two types to present optimal UX.
export type VCSUIType =
  | "GITLAB_SELF_HOST"
  | "GITLAB_COM"
  | "GITHUB_COM"
  | "GITHUB_ENTERPRISE"
  | "BITBUCKET_ORG"
  | "AZURE_DEVOPS";

export interface VCSConfig {
  type: VCSProvider_Type;
  uiType: VCSUIType;
  resourceId: string;
  name: string;
  instanceUrl: string;
  accessToken: string;
}

export type VCSFileCommit = {
  id: string;
  title: string;
  message: string;
  createdTs: number;
  url: string;
  authorName: string;
  added: string;
};

export type VCSCommit = {
  id: string;
  title: string;
  message: string;
  createdTs: number;
  url: string;
  authorName: string;
  authorEmail: string;
  addedList: string[];
  modifiedList: string[];
};

export type VCSPushEvent = {
  vcsType: VCSType;
  ref: string;
  repositoryId: string;
  repositoryUrl: string;
  repositoryFullPath: string;
  authorName: string;
  fileCommit: VCSFileCommit;
  commits: VCSCommit[];
};
