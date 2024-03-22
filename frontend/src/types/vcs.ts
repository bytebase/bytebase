import { VCSProvider_Type } from "@/types/proto/v1/vcs_provider_service";
import { VCSId } from "./id";

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
  vcsType: VCSProvider_Type,
  str: string
): boolean {
  if (vcsType == VCSProvider_Type.GITLAB) {
    return /^(.{1,}-){0,1}[a-zA-Z0-9_]{64}$/.test(str);
  } else if (vcsType == VCSProvider_Type.GITHUB) {
    return /^[a-zA-Z0-9_]{20}$|^[a-zA-Z0-9_]{40}$/.test(str);
  } else if (vcsType == VCSProvider_Type.BITBUCKET) {
    return /^[a-zA-Z0-9_]{18}$|^[a-zA-Z0-9_]{32}$/.test(str);
  } else if (vcsType == VCSProvider_Type.AZURE_DEVOPS) {
    // TODO: Azure App id is uuid but the secret is random string. We may need to distinguish them.
    return /^[a-zA-Z0-9-_.]{1,}$/.test(str);
  }
  return false;
}
