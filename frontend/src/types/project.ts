import { RowStatus } from "./common";
import { MemberId, PrincipalId, ProjectId } from "./id";
import { OAuthToken } from "./oauth";
import { Principal } from "./principal";
import { ExternalRepositoryInfo, RepositoryConfig } from "./repository";
import { VCS } from "./vcs";

export type ProjectRoleType = "OWNER" | "DEVELOPER";

export type ProjectWorkflowType = "UI" | "VCS";

export type ProjectVisibility = "PUBLIC" | "PRIVATE";

export type ProjectTenantMode = "DISABLED" | "TENANT";

export type ProjectRoleProvider = "GITLAB_SELF_HOST" | "BYTEBASE";

export type ProjectRoleProviderPayload = {
  vcsRole: string;
  lastSyncTs: number;
};

export const EmptyProjectRoleProviderPayload: ProjectRoleProviderPayload = {
  vcsRole: "",
  lastSyncTs: 0,
};

// Project
export type Project = {
  id: ProjectId;

  // Standard fields
  creator: Principal;
  updater: Principal;
  createdTs: number;
  updatedTs: number;
  rowStatus: RowStatus;

  // Domain specific fields
  name: string;
  key: string;
  // Returns the member list directly because we need it quite frequently in order
  // to do various access check.
  memberList: ProjectMember[];
  workflowType: ProjectWorkflowType;
  visibility: ProjectVisibility;
  tenantMode: ProjectTenantMode;
  dbNameTemplate: string;
  roleProvider: ProjectRoleProvider;
};

export type ProjectCreate = {
  // Domain specific fields
  name: string;
  key: string;
  tenantMode: ProjectTenantMode;
  dbNameTemplate: string;
  roleProvider: ProjectRoleProvider;
};

export type ProjectPatch = {
  // Standard fields
  rowStatus?: RowStatus;

  // Domain specific fields
  name?: string;
  key?: string;
  roleProvider?: ProjectRoleProvider;
};

// Project Member
export type ProjectMember = {
  id: MemberId;

  // Related fields
  project: Project;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  role: ProjectRoleType;
  principal: Principal;
  roleProvider: ProjectRoleProvider;
  payload: ProjectRoleProviderPayload;
};

export type ProjectMemberCreate = {
  // Domain specific fields
  principalId: PrincipalId;
  role: ProjectRoleType;
  roleProvider: ProjectRoleProvider;
};

export type ProjectMemberPatch = {
  // Domain specific fields
  role: ProjectRoleType;
  roleProvider: ProjectRoleProvider;
};

export type ProjectRepositoryConfig = {
  vcs: VCS;
  // TODO(zilong): get rid of the token in the frontend.
  token: OAuthToken;
  code: string;
  repositoryInfo: ExternalRepositoryInfo;
  repositoryConfig: RepositoryConfig;
};
