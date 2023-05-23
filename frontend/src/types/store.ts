import { RemovableRef } from "@vueuse/core";
import {
  MigrationHistoryId,
  QueryHistory,
  Sheet,
  OnboardingGuideType,
} from ".";
import { Activity } from "./activity";
import { Backup, BackupSetting } from "./backup";
import { Bookmark } from "./bookmark";
import { Command } from "./common";
import { Database } from "./database";
import { DataSource } from "./dataSource";
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
import { Notification } from "./notification";
import { Principal } from "./principal";
import { Project } from "./project";
import { Repository } from "./repository";
import { VCS } from "./vcs";
import { Label } from "./label";
import { ReleaseInfo } from "./actuator";
import type { DebugLog } from "@/types/debug";
import type { AuditLog } from "@/types/auditLog";
import { DatabaseMetadata } from "./proto/store/database";
import { ActuatorInfo } from "@/types/proto/v1/actuator_service";

export interface ActuatorState {
  serverInfo?: ActuatorInfo;
  releaseInfo: RemovableRef<ReleaseInfo>;
}

export interface AuditLogState {
  auditLogList: AuditLog[];
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
