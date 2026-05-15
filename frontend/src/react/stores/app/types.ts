import type { StateCreator } from "zustand";
import type { AppFeatures } from "@/types/appProfile";
import type { Permission } from "@/types/iam/permission";
import type { NotificationCreate } from "@/types/notification";
import type { ActuatorInfo } from "@/types/proto-es/v1/actuator_service_pb";
import type { DatabaseGroup } from "@/types/proto-es/v1/database_group_service_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { IamPolicy } from "@/types/proto-es/v1/iam_policy_pb";
import type { InstanceRole } from "@/types/proto-es/v1/instance_role_service_pb";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto-es/v1/instance_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Role } from "@/types/proto-es/v1/role_service_pb";
import type { WorkspaceProfileSetting } from "@/types/proto-es/v1/setting_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import type {
  PlanFeature,
  PlanType,
  Subscription,
} from "@/types/proto-es/v1/subscription_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import type { Workspace } from "@/types/proto-es/v1/workspace_service_pb";
import type { Environment } from "@/types/v1/environment";

export type ProjectListParams = {
  pageSize: number;
  pageToken: string;
  query?: string;
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
  NotificationSlice &
  PreferencesSlice;

export type AppSliceCreator<Slice> = StateCreator<AppStoreState, [], [], Slice>;
