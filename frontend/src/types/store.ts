import {
  AuthProvider,
  DeploymentConfig,
  EnvironmentId,
  MigrationHistoryId,
  Policy,
  PolicyType,
  QueryHistory,
  View,
  Sheet,
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
import { Table } from "./table";
import { VCS } from "./vcs";
import { Label } from "./label";
import { ConnectionAtom, ConnectionContext } from "./sqlEditor";
import { TabInfo } from "./tab";
import instanceStore from "../store/modules/instance";
import sqlEditorStore from "../store/modules/sqlEditor";
import tabStore from "../store/modules/tab";
import sheetStore from "../store/modules/sheet";

export interface ActuatorState {
  serverInfo?: ServerInfo;
}

export interface AuthState {
  authProviderList: AuthProvider[];
  currentUser: Principal;
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
  bookmarkListByUser: Map<PrincipalId, Bookmark[]>;
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
}

export interface IssueSubscriberState {
  subscriberListByIssue: Map<IssueId, IssueSubscriber[]>;
}

// eslint-disable-next-line @typescript-eslint/no-empty-interface
export interface PipelineState {}

// eslint-disable-next-line @typescript-eslint/no-empty-interface
export interface StageState {}

// eslint-disable-next-line @typescript-eslint/no-empty-interface
export interface TaskState {}

export interface PolicyState {
  policyMapByEnvironmentId: Map<EnvironmentId, Map<PolicyType, Policy>>;
}

export interface ProjectState {
  projectById: Map<ProjectId, Project>;
}

export interface ProjectWebhookState {
  projectWebhookListByProjectId: Map<ProjectId, ProjectWebhook[]>;
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
export type InstanceGetters = typeof instanceStore.getters;
export type InstanceActions = typeof instanceStore.actions;
export type InstanceMutations = typeof instanceStore.mutations;

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

export interface TableState {
  tableListByDatabaseId: Map<DatabaseId, Table[]>;
}

export interface ViewState {
  viewListByDatabaseId: Map<DatabaseId, View[]>;
}

export interface BackupState {
  backupListByDatabaseId: Map<DatabaseId, Backup[]>;
}

export interface BackupSettingState {
  backupSettingByDatabaseId: Map<DatabaseId, BackupSetting>;
}

export interface VCSState {
  vcsById: Map<VCSId, VCS>;
}

export interface RepositoryState {
  // repositoryListByVCSId are used in workspace version control panel, while repositoryByProjectId are used in project version control panel.
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

// type for vuex
export interface SqlEditorState {
  connectionTree: ConnectionAtom[];
  connectionContext: ConnectionContext;
  shouldSetContent: boolean;
  shouldFormatContent: boolean;
  queryHistoryList: QueryHistory[];
  isFetchingQueryHistory: boolean;
  isExecuting: boolean;
  isFetchingSheet: boolean;
  isShowExecutingHint: boolean;
  sharedSheet: Sheet;
}
export type SqlEditorGetters = typeof sqlEditorStore.getters;
export type SqlEditorActions = typeof sqlEditorStore.actions;
export type SqlEditorMutations = typeof sqlEditorStore.mutations;

export interface TabState {
  tabList: TabInfo[];
  currentTabId: string;
}
export type TabGetters = typeof tabStore.getters;
export type TabActions = typeof tabStore.actions;
export type TabMutations = typeof tabStore.mutations;

export interface DeploymentState {
  deploymentConfigByProjectId: Map<ProjectId, DeploymentConfig>;
}

export interface SheetState {
  sheetList: Sheet[];
  sheetById: Map<SheetId, Sheet>;
}
export type SheetGetters = typeof sheetStore.getters;
export type SheetActions = typeof sheetStore.actions;
export type SheetMutations = typeof sheetStore.mutations;

export interface DebugState {
  isDebug: boolean;
}
