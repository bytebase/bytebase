import { SchemaChange } from "@/types/proto/v1/project_service";
import { OAuthToken, VCSProvider } from "@/types/proto/v1/vcs_provider_service";
import { RowStatus } from "./common";
import { ProjectId } from "./id";
import { Principal } from "./principal";
import { ExternalRepositoryInfo, RepositoryConfig } from "./repository";

export type ProjectWorkflowType = "UI" | "VCS";

export type ProjectVisibility = "PUBLIC" | "PRIVATE";

export type ProjectTenantMode = "DISABLED" | "TENANT";

export type SchemaChangeType = "DDL" | "SDL";

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
  schemaChangeType: SchemaChangeType;
};

// Project Member
export type ProjectMember = {
  id: string; // projects/%s/roles/%s/principals/%d

  // Related fields
  project: Project;

  // Domain specific fields
  role: string;
  principal: Principal;
};

export type ProjectRepositoryConfig = {
  vcs: VCSProvider;
  // TODO(zilong): get rid of the token in the frontend.
  token: OAuthToken;
  code: string;
  repositoryInfo: ExternalRepositoryInfo;
  repositoryConfig: RepositoryConfig;
  schemaChangeType: SchemaChange;
};
