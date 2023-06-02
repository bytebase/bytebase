import { VCSId } from "./id";
import { ExternalVersionControl_Type } from "@/types/proto/v1/externalvs_service";

// Backend uses the same ENUM for GitLab/GitHub SaaS and self-hosted. Because they are based on the
// same codebase.
export type VCSType = "GITLAB" | "GITHUB" | "BITBUCKET";

// When configuring the VCS, we split the SaaS and self-hosted into two types to present optimal UX.
export type VCSUIType =
  | "GITLAB_SELF_HOST"
  | "GITLAB_COM"
  | "GITHUB_COM"
  | "BITBUCKET_ORG";

export interface VCSConfig {
  type: ExternalVersionControl_Type;
  uiType: VCSUIType;
  name: string;
  instanceUrl: string;
  applicationId: string;
  secret: string;
}

export type VCS = {
  id: VCSId;

  // Domain specific fields
  name: string;
  type: VCSType;
  uiType: VCSUIType;
  instanceUrl: string;
  apiUrl: string;
  applicationId: string;
  secret: string;
};

export type VCSTokenCreate = {
  code: string;
  redirectUrl: string;
};

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

export function isValidVCSApplicationIdOrSecret(
  vcsType: ExternalVersionControl_Type,
  str: string
): boolean {
  if (vcsType == ExternalVersionControl_Type.GITLAB) {
    return /^[a-zA-Z0-9_]{64}$/.test(str);
  } else if (vcsType == ExternalVersionControl_Type.GITHUB) {
    return /^[a-zA-Z0-9_]{20}$|^[a-zA-Z0-9_]{40}$/.test(str);
  } else if (vcsType == ExternalVersionControl_Type.BITBUCKET) {
    return /^[a-zA-Z0-9_]{18}$|^[a-zA-Z0-9_]{32}$/.test(str);
  }
  return false;
}
