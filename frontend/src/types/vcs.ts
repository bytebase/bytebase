import { VCSId } from "./id";

export type VCSType = "GITLAB_SELF_HOST" | "GITHUB_COM";

export interface VCSConfig {
  type: VCSType;
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
  instanceUrl: string;
  apiUrl: string;
  applicationId: string;
  secret: string;
};

export type VCSCreate = {
  // Domain specific fields
  name: string;
  type: VCSType;
  instanceUrl: string;
  applicationId: string;
  secret: string;
};

export type VCSPatch = {
  // Domain specific fields
  name?: string;
  applicationId?: string;
  secret?: string;
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
  ID: string;
  Title: string;
  Message: string;
  CreatedTs: number;
  URL: string;
  AuthorName: string;
  AuthorEmail: string;
  AddedList: string[];
  ModifiedList: string[];
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
  vcsType: VCSType,
  str: string
): boolean {
  if (vcsType == "GITLAB_SELF_HOST") {
    return /^[a-zA-Z0-9_]{64}$/.test(str);
  } else if (vcsType == "GITHUB_COM") {
    return /^[a-zA-Z0-9_]{20}$|^[a-zA-Z0-9_]{40}$/.test(str);
  }
  return false;
}
