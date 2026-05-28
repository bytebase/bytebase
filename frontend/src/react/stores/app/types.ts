import type { StateCreator } from "zustand";
import type { AppFeatures } from "@/types/appProfile";
import type { Permission } from "@/types/iam/permission";
import type { NotificationCreate } from "@/types/notification";
import type { AccessGrant } from "@/types/proto-es/v1/access_grant_service_pb";
import type { ActuatorInfo } from "@/types/proto-es/v1/actuator_service_pb";
import type { State } from "@/types/proto-es/v1/common_pb";
import type { DatabaseGroup } from "@/types/proto-es/v1/database_group_service_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import type { IamPolicy } from "@/types/proto-es/v1/iam_policy_pb";
import type { IdentityProvider } from "@/types/proto-es/v1/idp_service_pb";
import type { InstanceRole } from "@/types/proto-es/v1/instance_role_service_pb";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto-es/v1/instance_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Role } from "@/types/proto-es/v1/role_service_pb";
import type { ServiceAccount } from "@/types/proto-es/v1/service_account_service_pb";
import type { WorkspaceProfileSetting } from "@/types/proto-es/v1/setting_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import type {
  PlanFeature,
  PlanType,
  Subscription,
} from "@/types/proto-es/v1/subscription_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import type { WorkloadIdentity } from "@/types/proto-es/v1/workload_identity_service_pb";
import type { Workspace } from "@/types/proto-es/v1/workspace_service_pb";
import type { Environment } from "@/types/v1/environment";
import type { AccessGrantFilterStatus } from "@/utils";

export type ProjectListParams = {
  pageSize: number;
  pageToken: string;
  query?: string;
};

export type GroupFilter = {
  query?: string;
  project?: string;
};

export type AccountFilter = {
  query?: string;
  state?: State;
};

export type ListServiceAccountsParams = {
  parent: string;
  pageSize: number;
  pageToken?: string;
  showDeleted: boolean;
  filter?: AccountFilter;
};

export type ListWorkloadIdentitiesParams = {
  parent: string;
  pageSize: number;
  pageToken?: string;
  showDeleted: boolean;
  filter?: AccountFilter;
};

export type AccessGrantFilter = {
  name?: string;
  statement?: string;
  creator?: string;
  status?: AccessGrantFilterStatus[];
  issue?: string;
  target?: string;
  createdTsAfter?: number;
  createdTsBefore?: number;
};

export type ListAccessGrantsParams = {
  parent: string;
  filter?: AccessGrantFilter;
  pageSize?: number;
  pageToken?: string;
  orderBy?: string;
};

export type AuthSlice = {
  currentUser?: User;
  currentUserRequest?: Promise<User | undefined>;
  loadCurrentUser: () => Promise<User | undefined>;
  logout: (signinUrl: string) => Promise<void>;
};

export type WorkspaceSlice = {
  serverInfo?: ActuatorInfo;
  serverInfoTs: number;
  serverInfoRequest?: Promise<ActuatorInfo | undefined>;
  workspace?: Workspace;
  workspaceList: Workspace[];
  workspaceRequest?: Promise<Workspace | undefined>;
  workspaceProfile?: WorkspaceProfileSetting;
  workspaceProfileRequest?: Promise<WorkspaceProfileSetting | undefined>;
  environmentList: Environment[];
  environmentRequest?: Promise<Environment[]>;
  appFeatures: AppFeatures;
  subscription?: Subscription;
  subscriptionRequest?: Promise<Subscription | undefined>;
  loadServerInfo: () => Promise<ActuatorInfo | undefined>;
  refreshServerInfo: () => Promise<ActuatorInfo | undefined>;
  loadWorkspace: () => Promise<Workspace | undefined>;
  loadWorkspaceList: () => Promise<Workspace[]>;
  switchWorkspace: (workspaceName: string) => Promise<void>;
  loadWorkspaceProfile: (
    force?: boolean
  ) => Promise<WorkspaceProfileSetting | undefined>;
  loadEnvironmentList: (force?: boolean) => Promise<Environment[]>;
  refreshEnvironmentList: () => Promise<Environment[]>;
  loadSubscription: () => Promise<Subscription | undefined>;
  refreshSubscription: () => Promise<Subscription | undefined>;
  uploadLicense: (license: string) => Promise<Subscription | undefined>;
  currentPlan: () => PlanType;
  isFreePlan: () => boolean;
  isTrialing: () => boolean;
  isExpired: () => boolean;
  daysBeforeExpire: () => number;
  trialingDays: () => number;
  showTrial: () => boolean;
  expireAt: () => string;
  instanceCountLimit: () => number;
  userCountLimit: () => number;
  instanceLicenseCount: () => number;
  hasUnifiedInstanceLicense: () => boolean;
  hasFeature: (feature: PlanFeature) => boolean;
  hasInstanceFeature: (
    feature: PlanFeature,
    instance?: Instance | InstanceResource
  ) => boolean;
  instanceMissingLicense: (
    feature: PlanFeature,
    instance?: Instance | InstanceResource
  ) => boolean;
  getMinimumRequiredPlan: (feature: PlanFeature) => PlanType;
  isSaaSMode: () => boolean;
  workspaceResourceName: () => string;
  externalUrl: () => string;
  needConfigureExternalUrl: () => boolean;
  version: () => string;
  changelogURL: () => string;
  activatedInstanceCount: () => number;
  totalInstanceCount: () => number;
  userCountInIam: () => number;
};

export type IamSlice = {
  workspacePolicy?: IamPolicy;
  workspacePolicyRequest?: Promise<IamPolicy | undefined>;
  projectPoliciesByName: Record<string, IamPolicy>;
  projectPolicyRequests: Record<string, Promise<IamPolicy | undefined>>;
  roles: Role[];
  rolesRequest?: Promise<Role[]>;
  loadWorkspacePermissionState: () => Promise<void>;
  loadProjectIamPolicy: (project: string) => Promise<IamPolicy | undefined>;
  hasWorkspacePermission: (permission: Permission) => boolean;
  hasProjectPermission: (project: Project, permission: Permission) => boolean;
};

export type ProjectSlice = {
  projectsByName: Record<string, Project>;
  projectRequests: Record<string, Promise<Project | undefined>>;
  projectErrorsByName: Record<string, Error | undefined>;
  fetchProject: (name: string) => Promise<Project | undefined>;
  batchFetchProjects: (names: string[]) => Promise<Project[]>;
  searchProjects: (params: ProjectListParams) => Promise<{
    projects: Project[];
    nextPageToken?: string;
  }>;
  createProject: (title: string, resourceId: string) => Promise<Project>;
};

export type InstanceSlice = {
  instancesByName: Record<string, Instance>;
  instanceRequests: Record<string, Promise<Instance | undefined>>;
  instanceErrorsByName: Record<string, Error | undefined>;
  fetchInstance: (name: string) => Promise<Instance | undefined>;
};

export type DatabaseListParams = {
  parent: string;
  pageSize: number;
  pageToken?: string;
  filter?: string;
  orderBy?: string;
};

export type DatabaseSlice = {
  databasesByName: Record<string, Database>;
  databaseRequests: Record<string, Promise<Database | undefined>>;
  databaseErrorsByName: Record<string, Error | undefined>;
  fetchDatabase: (name: string) => Promise<Database | undefined>;
  batchFetchDatabases: (names: string[]) => Promise<Database[]>;
  fetchDatabases: (params: DatabaseListParams) => Promise<{
    databases: Database[];
    nextPageToken: string;
  }>;
};

export type DBGroupSlice = {
  dbGroupsByName: Record<string, DatabaseGroup>;
  dbGroupRequests: Record<string, Promise<DatabaseGroup | undefined>>;
  dbGroupErrorsByName: Record<string, Error | undefined>;
  fetchDBGroup: (name: string) => Promise<DatabaseGroup | undefined>;
  listDBGroupsForProject: (project: string) => Promise<DatabaseGroup[]>;
};

export type SheetSlice = {
  sheetsByName: Record<string, Sheet>;
  sheetRequests: Record<string, Promise<Sheet | undefined>>;
  sheetErrorsByName: Record<string, Error | undefined>;
  fetchSheet: (name: string, raw?: boolean) => Promise<Sheet | undefined>;
  createSheet: (parent: string, sheet: Sheet) => Promise<Sheet>;
};

export type InstanceRoleSlice = {
  rolesByInstance: Record<string, InstanceRole[]>;
  roleRequests: Record<string, Promise<InstanceRole[]>>;
  fetchInstanceRoles: (instance: string) => Promise<InstanceRole[]>;
};

export type GroupSlice = {
  groupsByName: Record<string, Group>;
  groupRequests: Record<string, Promise<Group | undefined>>;
  groupErrorsByName: Record<string, Error | undefined>;
  listGroups: (params: {
    pageSize: number;
    pageToken?: string;
    filter?: GroupFilter;
  }) => Promise<{ groups: Group[]; nextPageToken: string }>;
  batchFetchGroups: (names: string[]) => Promise<Group[]>;
  batchGetOrFetchGroups: (names: string[]) => Promise<(Group | undefined)[]>;
  fetchGroup: (id: string) => Promise<Group | undefined>;
  getGroupByIdentifier: (id: string) => Group | undefined;
  createGroup: (group: Group) => Promise<Group>;
  updateGroup: (group: Group) => Promise<Group>;
  deleteGroup: (name: string) => Promise<void>;
};

export type ServiceAccountSlice = {
  serviceAccountsByName: Record<string, ServiceAccount>;
  serviceAccountRequests: Record<string, Promise<ServiceAccount | undefined>>;
  listServiceAccounts: (
    params: ListServiceAccountsParams
  ) => Promise<{ serviceAccounts: ServiceAccount[]; nextPageToken: string }>;
  fetchServiceAccount: (
    name: string,
    silent?: boolean
  ) => Promise<ServiceAccount | undefined>;
  getServiceAccount: (name: string) => ServiceAccount;
  createServiceAccount: (
    serviceAccountId: string,
    serviceAccount: Partial<ServiceAccount>,
    parent: string
  ) => Promise<ServiceAccount>;
  updateServiceAccount: (
    serviceAccount: Partial<ServiceAccount>,
    updateMask: { paths: string[] }
  ) => Promise<ServiceAccount>;
  deleteServiceAccount: (name: string) => Promise<void>;
  undeleteServiceAccount: (name: string) => Promise<ServiceAccount>;
};

export type WorkloadIdentitySlice = {
  workloadIdentitiesByName: Record<string, WorkloadIdentity>;
  workloadIdentityRequests: Record<
    string,
    Promise<WorkloadIdentity | undefined>
  >;
  listWorkloadIdentities: (params: ListWorkloadIdentitiesParams) => Promise<{
    workloadIdentities: WorkloadIdentity[];
    nextPageToken: string;
  }>;
  fetchWorkloadIdentity: (
    name: string,
    silent?: boolean
  ) => Promise<WorkloadIdentity | undefined>;
  getWorkloadIdentity: (name: string) => WorkloadIdentity;
  createWorkloadIdentity: (
    workloadIdentityId: string,
    workloadIdentity: Partial<WorkloadIdentity>,
    parent: string
  ) => Promise<WorkloadIdentity>;
  updateWorkloadIdentity: (
    workloadIdentity: Partial<WorkloadIdentity>,
    updateMask: { paths: string[] }
  ) => Promise<WorkloadIdentity>;
  deleteWorkloadIdentity: (name: string) => Promise<void>;
  undeleteWorkloadIdentity: (name: string) => Promise<WorkloadIdentity>;
};

export type IdentityProviderSlice = {
  identityProvidersByName: Record<string, IdentityProvider>;
  identityProviderRequests: Record<
    string,
    Promise<IdentityProvider | undefined>
  >;
  identityProviderList: () => IdentityProvider[];
  listIdentityProviders: (parent?: string) => Promise<IdentityProvider[]>;
  fetchIdentityProvider: (
    name: string,
    silent?: boolean
  ) => Promise<IdentityProvider | undefined>;
  getIdentityProvider: (name: string) => IdentityProvider | undefined;
  createIdentityProvider: (
    identityProvider: IdentityProvider
  ) => Promise<IdentityProvider>;
  updateIdentityProvider: (
    update: Partial<IdentityProvider>
  ) => Promise<IdentityProvider>;
  deleteIdentityProvider: (name: string) => Promise<void>;
};

export type AccessGrantSlice = {
  accessGrantsByName: Record<string, AccessGrant>;
  accessGrantRequests: Record<string, Promise<AccessGrant | undefined>>;
  fetchAccessGrant: (name: string) => Promise<AccessGrant | undefined>;
  searchMyAccessGrants: (
    params: ListAccessGrantsParams
  ) => Promise<{ accessGrants: AccessGrant[]; nextPageToken: string }>;
  listAccessGrants: (
    params: ListAccessGrantsParams
  ) => Promise<{ accessGrants: AccessGrant[]; nextPageToken: string }>;
  createAccessGrant: (
    parent: string,
    accessGrant: AccessGrant
  ) => Promise<AccessGrant>;
  activateAccessGrant: (name: string) => Promise<AccessGrant>;
  revokeAccessGrant: (name: string) => Promise<AccessGrant>;
};

export type NotificationSlice = {
  notify: (notification: NotificationCreate) => void;
};

export type PreferencesSlice = {
  setRecentProject: (name: string) => void;
  recordRecentVisit: (path: string) => void;
  removeRecentVisit: (path: string) => void;
  resetQuickstartProgress: () => void;
};

export type AppStoreState = AuthSlice &
  WorkspaceSlice &
  IamSlice &
  ProjectSlice &
  InstanceSlice &
  DatabaseSlice &
  DBGroupSlice &
  SheetSlice &
  InstanceRoleSlice &
  GroupSlice &
  ServiceAccountSlice &
  WorkloadIdentitySlice &
  IdentityProviderSlice &
  AccessGrantSlice &
  NotificationSlice &
  PreferencesSlice;

export type AppSliceCreator<Slice> = StateCreator<AppStoreState, [], [], Slice>;
