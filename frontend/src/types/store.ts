import { RemovableRef } from "@vueuse/core";
import {
  AuthProvider,
  DeploymentConfig,
  EnvironmentId,
  MigrationHistoryId,
  Policy,
  PolicyType,
  QueryHistory,
  Sheet,
  OnboardingGuideType,
  PolicyResourceType,
} from ".";
import { Activity } from "./activity";
import { ServerInfo } from "./actuator";
import { Backup, BackupSetting } from "./backup";
import { Bookmark } from "./bookmark";
import { Command } from "./common";
import { Database } from "./database";
import { DataSource } from "./dataSource";
import { Environment } from "./environment";
import {
  CommandId,
  DatabaseId,
  DataSourceId,
  InstanceId,
  IssueId,
  PrincipalId,
  ProjectId,
  VCSId,
  SheetId,
} from "./id";
import { Inbox, InboxSummary } from "./inbox";
import { Instance, MigrationHistory } from "./instance";
import { InstanceUser } from "./InstanceUser";
import { Issue } from "./issue";
import { IssueSubscriber } from "./issueSubscriber";
import { Member } from "./member";
import { Notification } from "./notification";
import { PlanType } from "./plan";
import { Principal } from "./principal";
import { Project } from "./project";
import { ProjectWebhook } from "./projectWebhook";
import { Repository } from "./repository";
import { Setting, SettingName } from "./setting";
import { VCS } from "./vcs";
import { Label } from "./label";
import { ReleaseInfo } from "./actuator";
import type { DebugLog } from "@/types/debug";
import type { AuditLog } from "@/types/auditLog";
import { DatabaseMetadata } from "./proto/store/database";

export interface ActuatorState {
  serverInfo?: ServerInfo;
  releaseInfo: RemovableRef<ReleaseInfo>;
}

export interface AuthState {
  authProviderList: AuthProvider[];
  currentUser: Principal;
}

export interface AuditLogState {
  auditLogList: AuditLog[];
}

export interface SettingState {
  settingByName: Map<SettingName, Setting>;
}

export interface PlanState {
  plan: PlanType;
}

export interface MemberState {
  memberList: Member[];
}

export interface PrincipalState {
  principalList: Principal[];
}

export interface BookmarkState {
  bookmarkList: Map<PrincipalId, Bookmark[]>;
}

export interface ActivityState {
  activityListByUser: Map<PrincipalId, Activity[]>;
  activityListByIssue: Map<IssueId, Activity[]>;
}

export interface InboxState {
  inboxListByUser: Map<PrincipalId, Inbox[]>;
  inboxSummaryByUser: Map<PrincipalId, InboxSummary>;
}

export interface IssueState {
  issueById: Map<IssueId, Issue>;
  isCreatingIssue: boolean;
}

export interface IssueSubscriberState {
  subscriberList: Map<IssueId, IssueSubscriber[]>;
}

// eslint-disable-next-line @typescript-eslint/no-empty-interface
export interface PipelineState {}

// eslint-disable-next-line @typescript-eslint/no-empty-interface
export interface StageState {}

// eslint-disable-next-line @typescript-eslint/no-empty-interface
export interface TaskState {}

export interface PolicyState {
  policyMapByEnvironmentId: Map<EnvironmentId, Map<PolicyType, Policy>>;
  policyMapByDatabaseId: Map<DatabaseId, Map<PolicyType, Policy>>;
  policyMapByResourceType: Map<PolicyResourceType, Map<PolicyType, Policy[]>>;
}

export interface ProjectState {
  projectById: Map<ProjectId, Project>;
}

export interface ProjectWebhookState {
  projectWebhookList: Map<ProjectId, ProjectWebhook[]>;
}

export interface EnvironmentState {
  environmentList: Environment[];
}

export interface InstanceState {
  instanceById: Map<InstanceId, Instance>;
  instanceUserListById: Map<InstanceId, InstanceUser[]>;
  migrationHistoryById: Map<MigrationHistoryId, MigrationHistory>;
  // The key is a concatenation of instance id and database name
  migrationHistoryListByIdAndDatabaseName: Map<string, MigrationHistory[]>;
}

export interface DataSourceState {
  dataSourceById: Map<DataSourceId, DataSource>;
}

export interface DatabaseState {
  // UI may fetch the database list from different dimension (by user, by environment).
  // In those cases, we will iterate through this map and compute the list on the fly.
  // We save it by instance because database belongs to instance and saving this way
  // follows that hierarchy.
  databaseListByInstanceId: Map<InstanceId, Database[]>;
  // Used exclusively for project panel, we do this to avoid interference from databaseListByInstanceId
  // where updating databaseListByInstanceId will cause reloading project related UI due to reactivity
  databaseListByProjectId: Map<ProjectId, Database[]>;
}

export interface DBSchemaState {
  requestCache: Map<DatabaseId, Promise<DatabaseMetadata>>;
  databaseMetadataById: Map<DatabaseId, DatabaseMetadata>;
}

export interface BackupState {
  backupList: Map<DatabaseId, Backup[]>;
}

export interface BackupSettingState {
  backupSetting: Map<DatabaseId, BackupSetting>;
}

export interface VCSState {
  vcsById: Map<VCSId, VCS>;
}

export interface RepositoryState {
  // repositoryListByVCSId are used in workspace GitOps panel, while repositoryByProjectId are used in project GitOps panel.
  // Because they are used separately, so we don't need to worry about repository inconsistency issue between them.
  repositoryListByVCSId: Map<VCSId, Repository[]>;
  repositoryByProjectId: Map<ProjectId, Repository>;
}

// eslint-disable-next-line @typescript-eslint/no-empty-interface
export interface AnomalyState {}

export interface NotificationState {
  notificationByModule: Map<string, Notification[]>;
}

export interface CommandState {
  commandListById: Map<CommandId, Command[]>;
}

export interface LabelState {
  labelList: Label[];
}

export interface SQLEditorState {
  shouldFormatContent: boolean;
  queryHistoryList: QueryHistory[];
  isFetchingQueryHistory: boolean;
  isFetchingSheet: boolean;
  isShowExecutingHint: boolean;
}

export interface DeploymentState {
  deploymentConfigByProjectId: Map<ProjectId, DeploymentConfig>;
}

export interface SheetState {
  sheetList: Sheet[];
  sheetById: Map<SheetId, Sheet>;
}

export interface DebugState {
  isDebug: boolean;
}
export interface DebugLogState {
  debugLogList: DebugLog[];
}

export interface OnboardingGuideState {
  guideName?: OnboardingGuideType;
}
