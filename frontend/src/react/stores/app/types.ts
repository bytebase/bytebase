import type { FieldMask } from "@bufbuild/protobuf/wkt";
import type { StateCreator } from "zustand";
import type { ConditionGroupExpr } from "@/plugins/cel";
import type { DatabaseFilter } from "@/react/lib/databaseFilter";
import type { AppFeatures } from "@/types/appProfile";
import type { Permission } from "@/types/iam/permission";
import type { NotificationCreate } from "@/types/notification";
import type { AccessGrant } from "@/types/proto-es/v1/access_grant_service_pb";
import type { ActuatorInfo } from "@/types/proto-es/v1/actuator_service_pb";
import type { LoginRequest } from "@/types/proto-es/v1/auth_service_pb";
import type { Engine, State } from "@/types/proto-es/v1/common_pb";
import type { DatabaseCatalog } from "@/types/proto-es/v1/database_catalog_service_pb";
import type {
  DatabaseGroup,
  DatabaseGroupView,
} from "@/types/proto-es/v1/database_group_service_pb";
import type {
  BatchUpdateDatabasesRequest,
  Changelog,
  ChangelogView,
  Database,
  DatabaseMetadata,
  DatabaseSchema,
  DiffSchemaRequest,
  DiffSchemaResponse,
  ExtensionMetadata,
  ExternalTableMetadata,
  FunctionMetadata,
  GetChangelogRequest,
  ListChangelogsRequest,
  SchemaMetadata,
  TableMetadata,
  UpdateDatabaseRequest,
  ViewMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import type { IamPolicy } from "@/types/proto-es/v1/iam_policy_pb";
import type { IdentityProvider } from "@/types/proto-es/v1/idp_service_pb";
import type { InstanceRole } from "@/types/proto-es/v1/instance_role_service_pb";
import type {
  DataSource,
  Instance,
  InstanceResource,
  ListInstanceDatabaseResponse,
  SyncInstanceResponse,
  UpdateInstanceRequest,
} from "@/types/proto-es/v1/instance_service_pb";
import type {
  Issue,
  IssueComment,
  ListIssueCommentsRequest,
} from "@/types/proto-es/v1/issue_service_pb";
import type {
  Policy,
  PolicyType,
  QueryDataPolicy,
} from "@/types/proto-es/v1/org_policy_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Project, Webhook } from "@/types/proto-es/v1/project_service_pb";
import type { Release } from "@/types/proto-es/v1/release_service_pb";
import type { Revision } from "@/types/proto-es/v1/revision_service_pb";
import type { Role } from "@/types/proto-es/v1/role_service_pb";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import type { ServiceAccount } from "@/types/proto-es/v1/service_account_service_pb";
import type {
  DataClassificationSetting_DataClassificationConfig,
  Setting,
  Setting_SettingName,
  SettingValue,
  WorkspaceProfileSetting,
} from "@/types/proto-es/v1/setting_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import type {
  ExportRequest,
  QueryRequest,
} from "@/types/proto-es/v1/sql_service_pb";
import type {
  BillingInterval,
  PaymentInfo,
  PlanFeature,
  PlanType,
  PurchasePlan,
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
import type { IssueFilter } from "@/types/v1/issue/issue";
import type { SQLResultSetV1 } from "@/types/v1/sql";
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
  // Substring search via `query.contains(...)`. Use for free-text search
  // UIs. Backend ILIKE-matches the stored grant query (whitespace
  // normalized), so callers can pass any substring.
  statement?: string;
  // Exact-match via `query == ...`. Use for authorization-eligibility
  // checks where the frontend needs to mirror the backend JIT match
  // (e.g. "can the user export this exact statement?"). Backend trims
  // boundary whitespace on both sides — internal whitespace preserved
  // byte-for-byte. PR #20491 bot review #3349385091.
  statementExact?: string;
  creator?: string;
  status?: AccessGrantFilterStatus[];
  issue?: string;
  target?: string;
  createdTsAfter?: number;
  createdTsBefore?: number;
  unmask?: boolean;
  export?: boolean;
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
  // Resource name `users/{email}` of the signed-in user. Mirrors the legacy
  // Pinia auth store's `currentUserName`; drives `isLoggedIn`.
  currentUserName?: string;
  unauthenticatedOccurred: boolean;
  authSessionKey: string;
  isSelfEmailUpdate: boolean;
  loadCurrentUser: () => Promise<User | undefined>;
  isLoggedIn: () => boolean;
  requireResetPassword: () => boolean;
  setRequireResetPassword: (value: boolean) => void;
  setUnauthenticatedOccurred: (value: boolean) => void;
  fetchCurrentUser: () => Promise<User | undefined>;
  login: (params: {
    request: LoginRequest;
    redirect?: boolean;
    redirectUrl?: string;
  }) => Promise<void>;
  signup: (request: Partial<User>) => Promise<void>;
  logout: () => Promise<void>;
  sendEmailLoginCode: (email: string, workspace?: string) => Promise<void>;
  updateCurrentUserNameForEmailChange: (newName: string) => void;
  setIsSelfEmailUpdate: (value: boolean) => void;
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
  // General-purpose setting cache, mirrors the legacy Pinia
  // `useSettingV1Store` API. Keyed by the setting's resource name
  // (`settings/{Setting_SettingName}`).
  settingsByName: Record<string, Setting>;
  settingRequests: Record<string, Promise<Setting | undefined>>;
  appFeatures: AppFeatures;
  subscription?: Subscription;
  subscriptionRequest?: Promise<Subscription | undefined>;
  // Subscription purchase metadata (SaaS). Mirrors the legacy Pinia
  // `useSubscriptionV1Store` `purchasePlans` / `paymentInfo` refs.
  purchasePlans: PurchasePlan[];
  paymentInfo?: PaymentInfo;
  loadServerInfo: () => Promise<ActuatorInfo | undefined>;
  refreshServerInfo: () => Promise<ActuatorInfo | undefined>;
  // Alias for the legacy Pinia `actuatorStore.fetchServerInfo(workspace?)`
  // so consumers map 1:1. Delegates to the slice's refresh path.
  fetchServerInfo: (
    workspaceResourceName?: string
  ) => Promise<ActuatorInfo | undefined>;
  loadWorkspace: () => Promise<Workspace | undefined>;
  loadWorkspaceList: () => Promise<Workspace[]>;
  updateWorkspace: (
    workspace: Workspace,
    updateMask: string[]
  ) => Promise<Workspace>;
  switchWorkspace: (workspaceName: string, redirect?: boolean) => Promise<void>;
  loadWorkspaceProfile: (
    force?: boolean
  ) => Promise<WorkspaceProfileSetting | undefined>;
  loadEnvironmentList: (force?: boolean) => Promise<Environment[]>;
  refreshEnvironmentList: () => Promise<Environment[]>;
  getEnvironmentByName: (name: string, fallback?: boolean) => Environment;
  getSettingByName: (name: Setting_SettingName) => Setting | undefined;
  getOrFetchSettingByName: (
    name: Setting_SettingName,
    silent?: boolean
  ) => Promise<Setting | undefined>;
  // Bridge: lets the legacy Pinia `useSettingV1Store.upsertSetting` push
  // updates into the app store after a save, so still-app-store consumers
  // (e.g. SQL editor's `OpenAIButton`) see fresh values without a refresh.
  setSettingByName: (setting: Setting) => void;
  // Writes a setting to the server and updates the cache. Mirrors the legacy
  // Pinia `useSettingV1Store().upsertSetting`.
  upsertSetting: (params: {
    name: Setting_SettingName;
    value: SettingValue;
    validateOnly?: boolean;
    updateMask?: FieldMask;
  }) => Promise<Setting>;
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
  activeVcsUserCount: () => number;
  activeUserCount: () => number;
  enableOnboarding: () => boolean;
  quickStartEnabled: () => boolean;
  setupSample: () => Promise<void>;
  // Always returns a profile (never undefined), mirroring the Pinia
  // `workspaceProfile` getter so consumers read fields without null checks.
  getWorkspaceProfile: () => WorkspaceProfileSetting;
  // Data-classification config from the DATA_CLASSIFICATION setting cache.
  classification: () => DataClassificationSetting_DataClassificationConfig[];
  getProjectClassification: (
    classificationId: string
  ) => DataClassificationSetting_DataClassificationConfig | undefined;
  updateWorkspaceProfile: (params: {
    payload: Partial<WorkspaceProfileSetting>;
    updateMask: FieldMask;
  }) => Promise<void>;
  fetchEnvironments: (silent?: boolean) => Promise<void>;
  createEnvironment: (
    environment: Partial<Environment>
  ) => Promise<Environment>;
  updateEnvironment: (update: Partial<Environment>) => Promise<Environment>;
  deleteEnvironment: (name: string) => Promise<void>;
  reorderEnvironmentList: (
    orderedEnvironmentList: Environment[]
  ) => Promise<Environment[]>;
  setSubscription: (subscription: Subscription) => void;
  hasSplitInstanceLicense: () => boolean;
  pollSubscriptionUntil: (
    predicate: (subscription: Subscription) => boolean,
    options?: { timeoutMs?: number; intervalMs?: number; signal?: AbortSignal }
  ) => Promise<Subscription | undefined>;
  createPurchase: (
    plan: PlanType,
    interval: BillingInterval,
    seats: number
  ) => Promise<string>;
  updatePurchase: (
    plan: PlanType,
    interval: BillingInterval,
    seats: number,
    etag: string
  ) => Promise<string>;
  cancelPurchase: (feedback: string, comment: string) => Promise<void>;
  fetchPaymentInfo: () => Promise<PaymentInfo | undefined>;
  verifyCheckoutSession: (sessionId: string) => Promise<string>;
  fetchPurchasePlans: () => Promise<PurchasePlan[] | undefined>;
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
  getProjectIamPolicy: (project: string) => IamPolicy;
  updateProjectIamPolicy: (
    project: string,
    policy: IamPolicy
  ) => Promise<IamPolicy>;
  fetchWorkspaceIamPolicy: () => Promise<IamPolicy>;
  patchWorkspaceIamPolicy: (
    batchPatch: { member: string; roles: string[] }[]
  ) => Promise<void>;
  workspaceRoleMapToUsers: () => Map<string, Set<string>>;
  workspaceUserMapToRoles: () => Map<string, Set<string>>;
  findWorkspaceRolesByMember: (member: string) => string[];
  getWorkspaceRolesByName: (name: string) => Set<string>;
  hasWorkspacePermission: (permission: Permission) => boolean;
  hasProjectPermission: (project: Project, permission: Permission) => boolean;
};

export interface ProjectFilter {
  query?: string;
  excludeDefault?: boolean;
  state?: State;
  // label should be "{label key}:{label value}" format
  labels?: string[];
}

export type ProjectSlice = {
  projectsByName: Record<string, Project>;
  projectRequests: Record<string, Promise<Project | undefined>>;
  projectErrorsByName: Record<string, Error | undefined>;
  getProjectByName: (name: string) => Project;
  fetchProject: (
    name: string,
    silent?: boolean
  ) => Promise<Project | undefined>;
  getOrFetchProjectByName: (name: string, silent?: boolean) => Promise<Project>;
  batchFetchProjects: (names: string[]) => Promise<Project[]>;
  // Returns ALL requested projects (fetching the missing ones first),
  // resolved through `getProjectByName` so the placeholder is filled in.
  batchGetOrFetchProjects: (names: string[]) => Promise<Project[]>;
  searchProjects: (params: ProjectListParams) => Promise<{
    projects: Project[];
    nextPageToken?: string;
  }>;
  fetchProjectList: (params: {
    pageSize?: number;
    pageToken?: string;
    silent?: boolean;
    filter?: ProjectFilter;
    orderBy?: string;
    cache?: boolean;
  }) => Promise<{ projects: Project[]; nextPageToken?: string }>;
  createProject: (title: string, resourceId: string) => Promise<Project>;
  updateProject: (project: Project, updateMask: string[]) => Promise<Project>;
  archiveProject: (project: Project) => Promise<void>;
  restoreProject: (project: Project) => Promise<void>;
  deleteProject: (project: string) => Promise<void>;
  batchDeleteProjects: (projectNames: string[]) => Promise<void>;
  batchPurgeProjects: (projectNames: string[]) => Promise<void>;
  // Immutably upsert a single project into the by-name cache.
  updateProjectCache: (project: Project) => void;
  resetProjects: () => void;
};

export interface InstanceFilter {
  environment?: string;
  project?: string;
  host?: string;
  port?: string;
  query?: string;
  engines?: Engine[];
  state?: State;
  labels?: string[];
}

export type InstanceSlice = {
  instancesByName: Record<string, Instance>;
  instanceRequests: Record<string, Promise<Instance | undefined>>;
  instanceErrorsByName: Record<string, Error | undefined>;
  resetInstances: () => void;
  fetchInstance: (name: string) => Promise<Instance | undefined>;
  getInstanceByName: (name: string) => Instance;
  getOrFetchInstanceByName: (
    name: string,
    silent?: boolean
  ) => Promise<Instance>;
  createInstance: (
    instance: Instance,
    validateOnly?: boolean
  ) => Promise<Instance>;
  updateInstance: (
    instance: Instance,
    updateMask: string[]
  ) => Promise<Instance>;
  archiveInstance: (instance: Instance, force?: boolean) => Promise<Instance>;
  restoreInstance: (instance: Instance) => Promise<Instance>;
  deleteInstance: (instance: string) => Promise<void>;
  syncInstance: (
    instance: string,
    enableFullSync: boolean
  ) => Promise<SyncInstanceResponse>;
  batchSyncInstances: (
    instanceNameList: string[],
    enableFullSync: boolean
  ) => Promise<void>;
  batchUpdateInstances: (
    requests: UpdateInstanceRequest[]
  ) => Promise<Instance[]>;
  createDataSource: (params: {
    instance: string;
    dataSource: DataSource;
    validateOnly?: boolean;
  }) => Promise<Instance>;
  updateDataSource: (params: {
    instance: string;
    dataSource: DataSource;
    updateMask: string[];
    validateOnly?: boolean;
  }) => Promise<Instance>;
  deleteDataSource: (
    instance: Instance,
    dataSource: DataSource
  ) => Promise<Instance>;
  listInstanceDatabases: (
    name: string,
    instance?: Instance
  ) => Promise<ListInstanceDatabaseResponse>;
  fetchInstanceList: (params: {
    pageSize?: number;
    pageToken?: string;
    orderBy?: string;
    filter?: InstanceFilter;
    silent?: boolean;
  }) => Promise<{ instances: Instance[]; nextPageToken: string }>;
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
  // When listing by instance parent, stale cached databases for that instance
  // are evicted first unless this is set (e.g. paginated "load more").
  skipCacheRemoval?: boolean;
};

export type DatabaseSlice = {
  databasesByName: Record<string, Database>;
  databaseRequests: Record<string, Promise<Database | undefined>>;
  databaseErrorsByName: Record<string, Error | undefined>;
  resetDatabases: () => void;
  getDatabaseList: () => Database[];
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
  batchSyncDatabases: (databases: string[]) => Promise<void>;
  batchUpdateDatabases: (
    params: BatchUpdateDatabasesRequest
  ) => Promise<Database[]>;
  updateDatabase: (params: UpdateDatabaseRequest) => Promise<Database>;
  // Drops cached databases (and their schema metadata) for the given instance.
  removeCacheByInstance: (instance: string) => void;
  // Patches the cached `instanceResource` of every database under `instance`.
  updateDatabaseInstance: (instance: Instance) => void;
  fetchDatabaseSchema: (database: string) => Promise<DatabaseSchema>;
  diffSchema: (params: DiffSchemaRequest) => Promise<DiffSchemaResponse>;
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
  // Synchronous cache read. Returns a stable unknownDatabaseGroup when absent or
  // when a FULL view is requested but only BASIC is cached.
  getDBGroupByName: (name: string, view?: DatabaseGroupView) => DatabaseGroup;
  getOrFetchDBGroupByName: (
    name: string,
    options?: {
      skipCache?: boolean;
      silent?: boolean;
      view?: DatabaseGroupView;
    }
  ) => Promise<DatabaseGroup>;
  fetchDBGroupListByProjectName: (
    projectName: string,
    view: DatabaseGroupView
  ) => Promise<DatabaseGroup[]>;
  createDatabaseGroup: (params: {
    projectName: string;
    databaseGroup: Pick<
      DatabaseGroup,
      "$typeName" | "name" | "title" | "databaseExpr"
    >;
    databaseGroupId: string;
    validateOnly?: boolean;
  }) => Promise<DatabaseGroup>;
  updateDatabaseGroup: (
    databaseGroup: DatabaseGroup,
    updateMask: string[]
  ) => Promise<DatabaseGroup>;
  deleteDatabaseGroup: (name: string) => Promise<void>;
  fetchDatabaseGroupMatchList: (params: {
    projectName: string;
    expr: ConditionGroupExpr;
  }) => Promise<string[]>;
};

export type SheetSlice = {
  sheetsByName: Record<string, Sheet>;
  sheetRequests: Record<string, Promise<Sheet | undefined>>;
  sheetErrorsByName: Record<string, Error | undefined>;
  fetchSheet: (name: string, raw?: boolean) => Promise<Sheet | undefined>;
  createSheet: (parent: string, sheet: Sheet) => Promise<Sheet>;
  getSheetByName: (name: string) => Sheet | undefined;
  getOrFetchSheetByName: (name: string) => Promise<Sheet | undefined>;
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
  getReleaseByName: (name: string) => Release | undefined;
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
  // Bumped on every intro-state write so selectors reading `getIntroStateByKey`
  // re-run (the flags live in localStorage, not Zustand state).
  introStateVersion: number;
  setRecentProject: (name: string) => void;
  recordRecentVisit: (path: string) => void;
  removeRecentVisit: (path: string) => void;
  resetQuickstartProgress: () => void;
  getIntroStateByKey: (key: string) => boolean;
  saveIntroStateByKey: (params: { key: string; newState: boolean }) => void;
};

// Org policy slice (mirrors the SQL-editor-used subset of the Pinia
// `usePolicyV1Store`): keyed by the policy resource name. Async fetchers
// dedupe via `policyRequests`. `getQueryDataPolicyByParent` returns a stable
// empty fallback when the policy isn't cached.
export type PolicySlice = {
  policyMapByName: Record<string, Policy>;
  policyRequests: Record<string, Promise<Policy | undefined>>;
  getPolicyByName: (name: string) => Policy | undefined;
  getOrFetchPolicyByName: (
    name: string,
    refresh?: boolean
  ) => Promise<Policy | undefined>;
  getPolicyByParentAndType: (params: {
    parentPath: string;
    policyType: PolicyType;
  }) => Policy | undefined;
  getOrFetchPolicyByParentAndType: (params: {
    parentPath: string;
    policyType: PolicyType;
    refresh?: boolean;
  }) => Promise<Policy | undefined>;
  getQueryDataPolicyByParent: (parent: string) => QueryDataPolicy;
  upsertPolicy: (params: {
    parentPath: string;
    policy: Partial<Policy>;
  }) => Promise<Policy>;
  deletePolicy: (name: string) => Promise<void>;
};

// Stateless issue service slice (mirrors the legacy Pinia `useIssueV1Store`):
// thin wrapper around `issueServiceClientConnect.getIssue`. Returns the
// fresh issue (no cache) and pre-fetches the owning project into the app
// store so downstream code can read it synchronously.
export type ListIssueParams = {
  find: IssueFilter;
  pageSize?: number;
  pageToken?: string;
};

export type IssueSlice = {
  fetchIssueByName: (name: string, silent?: boolean) => Promise<Issue>;
  listIssues: (
    params: ListIssueParams
  ) => Promise<{ nextPageToken: string; issues: Issue[] }>;
};

// Stateless SQL service slice (mirrors the legacy Pinia `useSQLStore`):
// thin wrappers around `sqlServiceClientConnect.query` / `.export` with the
// SQL editor's permission-denied / silent context conventions.
export type SQLSlice = {
  query: (params: QueryRequest, signal: AbortSignal) => Promise<SQLResultSetV1>;
  exportData: (params: ExportRequest) => Promise<Uint8Array>;
};

export interface GetOrFetchDatabaseMetadataParams {
  database: string;
  skipCache?: boolean;
  silent?: boolean;
  // Limit the number of returned tables per schema.
  limit?: number;
  // CEL filter, e.g. `schema == "public" && table.contains("user")`.
  filter?: string;
}

export type DBSchemaSlice = {
  // Cache key: `${metadataResourceName}::${filter}::${limit}` — mirrors the
  // Pinia store's `[name, filter, limit]` triple so filtered/sliced fetches
  // don't collide with full-metadata fetches.
  metadataByName: Record<string, DatabaseMetadata>;
  metadataRequests: Record<string, Promise<DatabaseMetadata>>;

  getDatabaseMetadata: (database: string) => DatabaseMetadata;
  // Returns the cached metadata reference (or undefined when uncached)
  // without the fresh-placeholder fallback `getDatabaseMetadata` adds.
  // Mirrors the legacy Pinia `getDatabaseMetadataWithoutDefault` — used by
  // consumers (e.g. SchemaPane) that need to distinguish "not loaded
  // yet" from "loaded but empty".
  getCachedDatabaseMetadata: (database: string) => DatabaseMetadata | undefined;
  getSchemaList: (database: string) => SchemaMetadata[];
  getSchemaMetadata: (params: {
    database: string;
    schema: string;
  }) => SchemaMetadata | undefined;
  // List getters mirror the legacy Pinia store API. Each composes from
  // the cached `DatabaseMetadata` and falls back to an empty array if
  // metadata isn't loaded yet (matching the legacy contract).
  getTableList: (params: {
    database: string;
    schema?: string;
  }) => TableMetadata[];
  getViewList: (params: {
    database: string;
    schema?: string;
  }) => ViewMetadata[];
  getExternalTableList: (params: {
    database: string;
    schema?: string;
  }) => ExternalTableMetadata[];
  getFunctionList: (params: {
    database: string;
    schema?: string;
  }) => FunctionMetadata[];
  getExtensionList: (database: string) => ExtensionMetadata[];
  // Invalidates all cache entries (across filter/limit variants) for a
  // database. Used by list pages that want a fresh metadata fetch on
  // next access (mirrors Pinia `removeCache`).
  removeDatabaseMetadataCache: (database: string) => void;
  getTableMetadata: (params: {
    database: string;
    table: string;
    schema?: string;
  }) => TableMetadata;
  getExternalTableMetadata: (params: {
    database: string;
    schema?: string;
    externalTable: string;
  }) => ExternalTableMetadata;
  getViewMetadata: (params: {
    database: string;
    schema?: string;
    view: string;
  }) => ViewMetadata;

  getOrFetchDatabaseMetadata: (
    params: GetOrFetchDatabaseMetadataParams
  ) => Promise<DatabaseMetadata>;
};

export type DatabaseCatalogSlice = {
  catalogsByName: Record<string, DatabaseCatalog>;
  catalogRequests: Record<string, Promise<DatabaseCatalog>>;
  getDatabaseCatalog: (database: string) => DatabaseCatalog;
  getOrFetchDatabaseCatalog: (params: {
    database: string;
    skipCache?: boolean;
    silent?: boolean;
  }) => Promise<DatabaseCatalog>;
  updateDatabaseCatalog: (catalog: DatabaseCatalog) => Promise<DatabaseCatalog>;
};

export type IssueCommentSlice = {
  // Cache keyed by issue resource name → its comment list.
  issueCommentsByIssue: Record<string, IssueComment[]>;
  listIssueComments: (
    request: ListIssueCommentsRequest
  ) => Promise<{ nextPageToken: string; issueComments: IssueComment[] }>;
  createIssueComment: (params: {
    issueName: string;
    comment: string;
  }) => Promise<void>;
  updateIssueComment: (params: {
    issueCommentName: string;
    comment: string;
  }) => Promise<void>;
  // Synchronous cache read; returns a stable empty array on miss.
  getIssueComments: (issueName: string) => IssueComment[];
};

export interface PlanFind {
  project: string;
  query?: string;
  creator?: string;
  createdTsAfter?: number;
  createdTsBefore?: number;
  hasIssue?: boolean;
  hasRollout?: boolean;
  specType?: string;
  state?: "ACTIVE" | "DELETED";
}

export type ListPlanParams = {
  find: PlanFind;
  pageSize?: number;
  pageToken?: string;
};

export type PlanSlice = {
  listPlans: (
    params: ListPlanParams
  ) => Promise<{ plans: Plan[]; nextPageToken: string }>;
};

export type RolloutSlice = {
  rolloutsByName: Record<string, Rollout>;
  fetchRolloutByName: (name: string, silent?: boolean) => Promise<Rollout>;
  // Synchronous cache read; returns a stable unknownRollout on miss.
  getRolloutByName: (name: string) => Rollout;
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
  PreferencesSlice &
  SQLSlice &
  IssueSlice &
  PolicySlice &
  DBSchemaSlice &
  RolloutSlice &
  PlanSlice &
  IssueCommentSlice &
  DatabaseCatalogSlice;

export type AppSliceCreator<Slice> = StateCreator<AppStoreState, [], [], Slice>;
