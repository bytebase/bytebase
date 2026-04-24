import type { StateCreator } from "zustand";
import type { AppFeatures } from "@/types/appProfile";
import type { Permission } from "@/types/iam/permission";
import type { NotificationCreate } from "@/types/notification";
import type { ActuatorInfo } from "@/types/proto-es/v1/actuator_service_pb";
import type { IamPolicy } from "@/types/proto-es/v1/iam_policy_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Role } from "@/types/proto-es/v1/role_service_pb";
import type { WorkspaceProfileSetting } from "@/types/proto-es/v1/setting_service_pb";
import type { Subscription } from "@/types/proto-es/v1/subscription_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import type { Workspace } from "@/types/proto-es/v1/workspace_service_pb";

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
  serverInfoRequest?: Promise<ActuatorInfo | undefined>;
  workspace?: Workspace;
  workspaceRequest?: Promise<Workspace | undefined>;
  workspaceProfile?: WorkspaceProfileSetting;
  workspaceProfileRequest?: Promise<WorkspaceProfileSetting | undefined>;
  appFeatures: AppFeatures;
  subscription?: Subscription;
  subscriptionRequest?: Promise<Subscription | undefined>;
  loadServerInfo: () => Promise<ActuatorInfo | undefined>;
  loadWorkspace: () => Promise<Workspace | undefined>;
  loadWorkspaceProfile: () => Promise<WorkspaceProfileSetting | undefined>;
  loadSubscription: () => Promise<Subscription | undefined>;
  uploadLicense: (license: string) => Promise<Subscription | undefined>;
};

export type IamSlice = {
  workspacePolicy?: IamPolicy;
  workspacePolicyRequest?: Promise<IamPolicy | undefined>;
  roles: Role[];
  rolesRequest?: Promise<Role[]>;
  loadWorkspacePermissionState: () => Promise<void>;
  hasWorkspacePermission: (permission: Permission) => boolean;
};

export type ProjectSlice = {
  projectsByName: Record<string, Project>;
  projectRequests: Record<string, Promise<Project | undefined>>;
  fetchProject: (name: string) => Promise<Project | undefined>;
  batchFetchProjects: (names: string[]) => Promise<Project[]>;
  searchProjects: (params: ProjectListParams) => Promise<{
    projects: Project[];
    nextPageToken?: string;
  }>;
  createProject: (title: string, resourceId: string) => Promise<Project>;
};

export type NotificationSlice = {
  notifications: NotificationCreate[];
  notify: (notification: NotificationCreate) => void;
};

export type PreferencesSlice = {
  setRecentProject: (name: string) => void;
  recordRecentVisit: (path: string) => void;
  resetQuickstartProgress: () => void;
};

export type AppStoreState = AuthSlice &
  WorkspaceSlice &
  IamSlice &
  ProjectSlice &
  NotificationSlice &
  PreferencesSlice;

export type AppSliceCreator<Slice> = StateCreator<AppStoreState, [], [], Slice>;
