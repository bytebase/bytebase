import { MigrationHistoryId, QueryHistory, OnboardingGuideType } from ".";
import { InstanceUser } from "./InstanceUser";
import { Backup } from "./backup";
import { Command } from "./common";
import { DataSource } from "./dataSource";
import { Database } from "./database";
import {
  CommandId,
  DatabaseId,
  DataSourceId,
  InstanceId,
  IssueId,
  ProjectId,
  VCSId,
} from "./id";
import { Instance, MigrationHistory } from "./instance";
import { Issue } from "./issue";
import { IssueSubscriber } from "./issueSubscriber";
import { Label } from "./label";
import { Notification } from "./notification";
import { Principal } from "./principal";
import { Project } from "./project";
import { DatabaseMetadata } from "./proto/store/database";
import { SQLEditorMode } from "./sqlEditor";
import { VCS } from "./vcs";

export interface PrincipalState {
  principalList: Principal[];
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

export interface ProjectState {
  projectById: Map<ProjectId, Project>;
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
  databaseListByInstanceId: Map<string, Database[]>;
  // Used exclusively for project panel, we do this to avoid interference from databaseListByInstanceId
  // where updating databaseListByInstanceId will cause reloading project related UI due to reactivity
  databaseListByProjectId: Map<string, Database[]>;
}

export interface DBSchemaState {
  requestCache: Map<DatabaseId, Promise<DatabaseMetadata>>;
  databaseMetadataById: Map<DatabaseId, DatabaseMetadata>;
}

export interface BackupState {
  backupList: Map<DatabaseId, Backup[]>;
}

export interface VCSState {
  vcsById: Map<VCSId, VCS>;
}

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
  mode: SQLEditorMode;
}

export interface DebugState {
  isDebug: boolean;
}

export interface OnboardingGuideState {
  guideName?: OnboardingGuideType;
}
