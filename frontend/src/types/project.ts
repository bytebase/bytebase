import {
  OAuthToken,
  ExternalVersionControl,
} from "@/types/proto/v1/externalvs_service";
import { SchemaChange } from "@/types/proto/v1/project_service";
import { RowStatus } from "./common";
import { ProjectId } from "./id";
import { Principal } from "./principal";
import { ExternalRepositoryInfo, RepositoryConfig } from "./repository";

export type ProjectRoleType =
  | "OWNER"
  | "DEVELOPER"
  | "EXPORTER"
  | "QUERIER"
  | string;

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
  workflowType?: ProjectWorkflowType;
  dbNameTemplate?: string;
  tenantMode?: ProjectTenantMode;
};

// Project Member
export type ProjectMember = {
  id: string; // projects/%s/roles/%s/principals/%d

  // Related fields
  project: Project;

  // Domain specific fields
  role: ProjectRoleType;
  principal: Principal;
};

export type ProjectRepositoryConfig = {
  vcs: ExternalVersionControl;
  // TODO(zilong): get rid of the token in the frontend.
  token: OAuthToken;
  code: string;
  repositoryInfo: ExternalRepositoryInfo;
  repositoryConfig: RepositoryConfig;
  schemaChangeType: SchemaChange;
};
