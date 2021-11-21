import { VCSID } from "./id";
import { Principal } from "./principal";

export type VCSType = "GITLAB_SELF_HOST";

export interface VCSConfig {
  type: VCSType;
  name: string;
  instanceURL: string;
  applicationID: string;
  secret: string;
}

export type VCS = {
  id: VCSID;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  type: VCSType;
  instanceURL: string;
  apiURL: string;
  applicationID: string;
  secret: string;
};

export type VCSCreate = {
  // Domain specific fields
  name: string;
  type: VCSType;
  instanceURL: string;
  applicationID: string;
  secret: string;
};

export type VCSPatch = {
  // Domain specific fields
  name?: string;
  applicationID?: string;
  secret?: string;
};

export type VCSTokenCreate = {
  code: string;
  redirectURL: string;
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

export type VCSPushEvent = {
  vcsType: VCSType;
  ref: string;
  repositoryID: string;
  repositoryUrl: string;
  repositoryFullPath: string;
  authorName: string;
  fileCommit: VCSFileCommit;
};

export function isValidVCSApplicationIDOrSecret(str: string): boolean {
  return /^[a-zA-Z0-9_]{64}$/.test(str);
}
