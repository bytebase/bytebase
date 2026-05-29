import type { StateCreator } from "zustand";
import type { DatabaseFilter } from "@/store/modules/v1/database";
import type { AppFeatures } from "@/types/appProfile";
import type { Permission } from "@/types/iam/permission";
import type { NotificationCreate } from "@/types/notification";
import type { AccessGrant } from "@/types/proto-es/v1/access_grant_service_pb";
import type { ActuatorInfo } from "@/types/proto-es/v1/actuator_service_pb";
import type { State } from "@/types/proto-es/v1/common_pb";
import type {
  DatabaseGroup,
  DatabaseGroupView,
} from "@/types/proto-es/v1/database_group_service_pb";
import type {
  Changelog,
  ChangelogView,
  Database,
  GetChangelogRequest,
  ListChangelogsRequest,
} from "@/types/proto-es/v1/database_service_pb";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import type { IamPolicy } from "@/types/proto-es/v1/iam_policy_pb";
import type { IdentityProvider } from "@/types/proto-es/v1/idp_service_pb";
import type { InstanceRole } from "@/types/proto-es/v1/instance_role_service_pb";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto-es/v1/instance_service_pb";
import type { Project, Webhook } from "@/types/proto-es/v1/project_service_pb";
import type { Release } from "@/types/proto-es/v1/release_service_pb";
import type { Revision } from "@/types/proto-es/v1/revision_service_pb";
import type { Role } from "@/types/proto-es/v1/role_service_pb";
import type { ServiceAccount } from "@/types/proto-es/v1/service_account_service_pb";
import type { WorkspaceProfileSetting } from "@/types/proto-es/v1/setting_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import type {
  PlanFeature,
  PlanType,
  Subscription,
} from "@/types/proto-es/v1/subscription_service_pb";
import type {
  UpdateUserRequest,
  User,
} from "@/types/proto-es/v1/user_service_pb";
import type { WorkloadIdentity } from "@/types/proto-es/v1/workload_identity_service_pb";
import type {
  Worksheet,
  WorksheetOrganizer,
} from "@/types/proto-es/v1/worksheet_service_pb";
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

export type UserFilter = {
  query?: string;
  project?: string;
  state?: State;
};

export type ListUsersParams = {
  pageSize: number;
  pageToken?: string;
  filter?: UserFilter;
  showDeleted?: boolean;
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
  getEnvironmentByName: (name: string, fallback?: boolean) => Environment;
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
  getProjectByName: (name: string) => Project;
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
  // Either a pre-built CEL filter string or a structured `DatabaseFilter`
  // (built via `buildDatabaseFilter`), matching the legacy Pinia store.
  filter?: string | DatabaseFilter;
  orderBy?: string;
  silent?: boolean;
};

export type DatabaseSlice = {
  databasesByName: Record<string, Database>;
  databaseRequests: Record<string, Promise<Database | undefined>>;
  databaseErrorsByName: Record<string, Error | undefined>;
  // Synchronous read with the `unknownDatabase` fallback (never null), so
  // callers can read `.project` / `.instanceResource` without null checks.
  getDatabaseByName: (name: string) => Database;
  fetchDatabase: (name: string) => Promise<Database | undefined>;
  // Async read that resolves to the cached/fetched database, or the
  // `unknownDatabase` fallback for invalid names / fetch failures.
  getOrFetchDatabaseByName: (
    name: string,
    silent?: boolean
  ) => Promise<Database>;
  batchFetchDatabases: (names: string[]) => Promise<Database[]>;
  batchGetOrFetchDatabases: (names: string[]) => Promise<Database[]>;
  fetchDatabases: (params: DatabaseListParams) => Promise<{
    databases: Database[];
    nextPageToken: string;
  }>;
  syncDatabase: (name: string, refresh?: boolean) => Promise<void>;
};

export type DBGroupSlice = {
  dbGroupsByName: Record<string, DatabaseGroup>;
  // Tracks which view (BASIC/FULL) the cached entry was fetched with, so a
  // FULL request (needs `matchedDatabases`) refetches when only BASIC is
  // cached.
  dbGroupViewByName: Record<string, DatabaseGroupView>;
  dbGroupRequests: Record<string, Promise<DatabaseGroup | undefined>>;
  dbGroupErrorsByName: Record<string, Error | undefined>;
  fetchDBGroup: (
    name: string,
    view?: DatabaseGroupView
  ) => Promise<DatabaseGroup | undefined>;
  listDBGroupsForProject: (project: string) => Promise<DatabaseGroup[]>;
};

export type SheetSlice = {
  sheetsByName: Record<string, Sheet>;
  sheetRequests: Record<string, Promise<Sheet | undefined>>;
  sheetErrorsByName: Record<string, Error | undefined>;
  fetchSheet: (name: string, raw?: boolean) => Promise<Sheet | undefined>;
  createSheet: (parent: string, sheet: Sheet) => Promise<Sheet>;
};

export type WorksheetView = "FULL" | "BASIC";

export type WorksheetSlice = {
  // Keyed by `${uid}:${view}` (mirrors the legacy Pinia cache, which kept
  // FULL and BASIC views separately — BASIC list entries omit the
  // statement, FULL entries carry it).
  worksheetsByKey: Record<string, Worksheet>;
  worksheetRequests: Record<string, Promise<Worksheet | undefined>>;
  getWorksheetByName: (
    name: string,
    view?: WorksheetView
  ) => Worksheet | undefined;
  getOrFetchWorksheetByName: (
    name: string,
    silent?: boolean
  ) => Promise<Worksheet | undefined>;
  fetchWorksheetList: (parent: string, filter: string) => Promise<Worksheet[]>;
  createWorksheet: (worksheet: Worksheet) => Promise<Worksheet>;
  patchWorksheet: (
    worksheet: Worksheet,
    updateMask: string[],
    signal?: AbortSignal
  ) => Promise<Worksheet | undefined>;
  deleteWorksheetByName: (name: string) => Promise<void>;
  upsertWorksheetOrganizer: (
    organizer: Partial<WorksheetOrganizer>,
    updateMask: string[]
  ) => Promise<void>;
  batchUpsertWorksheetOrganizers: (
    requests: { organizer: Partial<WorksheetOrganizer>; updateMask: string[] }[]
  ) => Promise<void>;
  worksheetList: () => Worksheet[];
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

export type UserSlice = {
  usersByName: Record<string, User>;
  userRequests: Record<string, Promise<User | undefined>>;
  listUsers: (
    params: ListUsersParams
  ) => Promise<{ users: User[]; nextPageToken: string }>;
  fetchUser: (name: string, silent?: boolean) => Promise<User | undefined>;
  batchGetOrFetchUsers: (names: string[]) => Promise<User[]>;
  getOrFetchUserByIdentifier: (params: {
    identifier: string;
    silent?: boolean;
    fallback?: boolean;
  }) => Promise<User>;
  getUserByIdentifier: (identifier: string) => User | undefined;
  createUser: (user: User) => Promise<User>;
  updateUser: (request: UpdateUserRequest) => Promise<User>;
  updateEmail: (oldEmail: string, newEmail: string) => Promise<User>;
  archiveUser: (name: string) => Promise<void>;
  restoreUser: (name: string) => Promise<User>;
};

export type RoleSlice = {
  roleList: Role[];
  listRoles: () => Promise<Role[]>;
  getRoleByName: (name: string) => Role | undefined;
  upsertRole: (role: Role) => Promise<Role>;
  deleteRole: (role: Role) => Promise<void>;
};

export type ReleaseSlice = {
  releasesByName: Record<string, Release>;
  releaseRequests: Record<string, Promise<Release | undefined>>;
  listReleasesByProject: (
    project: string,
    pagination?: { pageSize?: number; pageToken?: string },
    showDeleted?: boolean,
    filter?: string
  ) => Promise<{ releases: Release[]; nextPageToken: string }>;
  fetchRelease: (
    name: string,
    silent?: boolean
  ) => Promise<Release | undefined>;
  getReleasesByProject: (project: string) => Release[];
  getReleaseByName: (name: string) => Release;
  updateRelease: (
    release: Partial<Release>,
    updateMask: string[]
  ) => Promise<Release>;
  deleteRelease: (name: string) => Promise<void>;
  undeleteRelease: (name: string) => Promise<Release>;
};

export type RevisionSlice = {
  revisionsByName: Record<string, Revision>;
  listRevisionsByDatabase: (
    database: string,
    pagination?: { pageSize?: number; pageToken?: string }
  ) => Promise<{ revisions: Revision[]; nextPageToken: string }>;
  listAllRevisionsByDatabase: (
    database: string,
    pagination?: { pageSize?: number }
  ) => Promise<Revision[]>;
  fetchRevision: (name: string) => Promise<Revision>;
  getRevisionsByDatabase: (database: string) => Revision[];
  getRevisionByName: (name: string) => Revision | undefined;
  deleteRevision: (name: string) => Promise<void>;
};

export type ChangelogSlice = {
  changelogsByCacheKey: Record<string, Changelog>;
  changelogsByDatabase: Record<string, Changelog[]>;
  changelogRequests: Record<string, Promise<Changelog | undefined>>;
  clearChangelogCache: (parent: string) => void;
  listChangelogs: (
    params: Partial<ListChangelogsRequest>
  ) => Promise<{ changelogs: Changelog[]; nextPageToken: string }>;
  getOrFetchChangelogListOfDatabase: (
    database: string,
    pageSize: number,
    view?: ChangelogView
  ) => Promise<Changelog[]>;
  changelogListByDatabase: (database: string) => Changelog[];
  fetchChangelog: (
    params: Partial<GetChangelogRequest>
  ) => Promise<Changelog | undefined>;
  getOrFetchChangelogByName: (
    name: string,
    view?: ChangelogView
  ) => Promise<Changelog | undefined>;
  getChangelogByName: (
    name: string,
    view?: ChangelogView
  ) => Changelog | undefined;
  fetchPreviousChangelog: (name: string) => Promise<Changelog | undefined>;
};

export type ProjectWebhookSlice = {
  getProjectWebhookFromProjectById: (
    project: Project,
    webhookId: string
  ) => Webhook | undefined;
  createProjectWebhook: (project: string, webhook: Webhook) => Promise<Project>;
  updateProjectWebhook: (
    webhook: Webhook,
    updateMask: string[]
  ) => Promise<Project>;
  deleteProjectWebhook: (webhook: Webhook) => Promise<Project>;
  testProjectWebhook: (
    project: Project,
    webhook: Webhook
  ) => Promise<{ error: string }>;
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
  WorksheetSlice &
  InstanceRoleSlice &
  GroupSlice &
  ServiceAccountSlice &
  WorkloadIdentitySlice &
  IdentityProviderSlice &
  AccessGrantSlice &
  UserSlice &
  RoleSlice &
  ReleaseSlice &
  RevisionSlice &
  ChangelogSlice &
  ProjectWebhookSlice &
  NotificationSlice &
  PreferencesSlice;

export type AppSliceCreator<Slice> = StateCreator<AppStoreState, [], [], Slice>;
