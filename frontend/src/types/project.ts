import { RowStatus } from "./common";
import { MemberId, PrincipalId, ProjectId, ResourceId } from "./id";
import { OAuthToken } from "./oauth";
import { Principal } from "./principal";
import { ExternalRepositoryInfo, RepositoryConfig } from "./repository";
import { VCS } from "./vcs";

export type ProjectRoleType = "OWNER" | "DEVELOPER";

export type ProjectWorkflowType = "UI" | "VCS";

export type ProjectVisibility = "PUBLIC" | "PRIVATE";

export type ProjectTenantMode = "DISABLED" | "TENANT";

export type SchemaChangeType = "DDL" | "SDL";

export type LGTMCheckValue = "DISABLED" | "PROJECT_OWNER" | "PROJECT_MEMBER";

export type LGTMCheckSetting = {
  value: LGTMCheckValue;
};

// Project
export type Project = {
  id: ProjectId;
  resourceId: string;
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
  schemaChangeType: SchemaChangeType;
  lgtmCheckSetting: LGTMCheckSetting;
};

export const getDefaultLGTMCheckSetting = (): LGTMCheckSetting => {
  return {
    value: "DISABLED",
  };
};

export type ProjectCreate = {
  resourceId: ResourceId;

  // Domain specific fields
  name: string;
  key: string;
  tenantMode: ProjectTenantMode;
  dbNameTemplate: string;
  schemaChangeType: SchemaChangeType;
};

export type ProjectPatch = {
  // Standard fields
  rowStatus?: RowStatus;

  // Domain specific fields
  name?: string;
  key?: string;
  schemaChangeType?: SchemaChangeType;
  lgtmCheckSetting?: LGTMCheckSetting;
  workflowType?: ProjectWorkflowType;
  dbNameTemplate?: string;
  tenantMode?: ProjectTenantMode;
};

// Project Member
export type ProjectMember = {
  id: MemberId;

  // Related fields
  project: Project;

  // Domain specific fields
  role: ProjectRoleType;
  principal: Principal;
};

export type ProjectMemberCreate = {
  // Domain specific fields
  principalId: PrincipalId;
  role: ProjectRoleType;
};

export type ProjectMemberPatch = {
  // Domain specific fields
  role: ProjectRoleType;
};

export type ProjectRepositoryConfig = {
  vcs: VCS;
  // TODO(zilong): get rid of the token in the frontend.
  token: OAuthToken;
  code: string;
  repositoryInfo: ExternalRepositoryInfo;
  repositoryConfig: RepositoryConfig;
  schemaChangeType: SchemaChangeType;
};
