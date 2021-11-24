import { EnvironmentID, MigrationHistoryID, Policy, PolicyType, View } from ".";
import { Activity } from "./activity";
import { ServerInfo } from "./actuator";
import { Backup, BackupSetting } from "./backup";
import { Bookmark } from "./bookmark";
import { Command } from "./common";
import { Database } from "./database";
import { DataSource } from "./dataSource";
import { Environment } from "./environment";
import {
  CommandID,
  DatabaseID,
  DataSourceID,
  InstanceID,
  IssueID,
  PrincipalID,
  ProjectID,
  VCSID,
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

export interface ActuatorState {
  serverInfo?: ServerInfo;
}

export interface AuthState {
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
  bookmarkListByUser: Map<PrincipalID, Bookmark[]>;
}

export interface ActivityState {
  activityListByUser: Map<PrincipalID, Activity[]>;
  activityListByIssue: Map<IssueID, Activity[]>;
}

export interface InboxState {
  inboxListByUser: Map<PrincipalID, Inbox[]>;
  inboxSummaryByUser: Map<PrincipalID, InboxSummary>;
}

export interface IssueState {
  issueByID: Map<IssueID, Issue>;
}

export interface IssueSubscriberState {
  subscriberListByIssue: Map<IssueID, IssueSubscriber[]>;
}

// eslint-disable-next-line @typescript-eslint/no-empty-interface
export interface PipelineState {}

// eslint-disable-next-line @typescript-eslint/no-empty-interface
export interface StageState {}

// eslint-disable-next-line @typescript-eslint/no-empty-interface
export interface TaskState {}

export interface PolicyState {
  policyMapByEnvironmentID: Map<EnvironmentID, Map<PolicyType, Policy>>;
}

export interface ProjectState {
  projectByID: Map<ProjectID, Project>;
}

export interface ProjectWebhookState {
  projectWebhookListByProjectID: Map<ProjectID, ProjectWebhook[]>;
}

export interface EnvironmentState {
  environmentList: Environment[];
}

export interface InstanceState {
  instanceByID: Map<InstanceID, Instance>;
  instanceUserListByID: Map<InstanceID, InstanceUser[]>;
  migrationHistoryByID: Map<MigrationHistoryID, MigrationHistory>;
  // The key is a concatenation of instance id and database name
  migrationHistoryListByIDAndDatabaseName: Map<string, MigrationHistory[]>;
}

export interface DataSourceState {
  dataSourceByID: Map<DataSourceID, DataSource>;
}

export interface DatabaseState {
  // UI may fetch the database list from different dimension (by user, by environment).
  // In those cases, we will iterate through this map and compute the list on the fly.
  // We save it by instance because database belongs to instance and saving this way
  // follows that hierarchy.
  databaseListByInstanceID: Map<InstanceID, Database[]>;
  // Used exclusively for project panel, we do this to avoid interference from databaseListByInstanceID
  // where updating databaseListByInstanceID will cause reloading project related UI due to reactivity
  databaseListByProjectID: Map<ProjectID, Database[]>;
}

export interface TableState {
  tableListByDatabaseID: Map<DatabaseID, Table[]>;
}

export interface ViewState {
  viewListByDatabaseID: Map<DatabaseID, View[]>;
}

export interface BackupState {
  backupListByDatabaseID: Map<DatabaseID, Backup[]>;
}

export interface BackupSettingState {
  backupSettingByDatabaseID: Map<DatabaseID, BackupSetting>;
}

export interface VCSState {
  vcsByID: Map<VCSID, VCS>;
}

export interface RepositoryState {
  // repositoryListByVCSID are used in workspace version control panel, while repositoryByProjectID are used in project version control panel.
  // Because they are used separately, so we don't need to worry about repository inconsistency issue between them.
  repositoryListByVCSID: Map<VCSID, Repository[]>;
  repositoryByProjectID: Map<ProjectID, Repository>;
}

// eslint-disable-next-line @typescript-eslint/no-empty-interface
export interface AnomalyState {}

export interface NotificationState {
  notificationByModule: Map<string, Notification[]>;
}

export interface CommandState {
  commandListByID: Map<CommandID, Command[]>;
}
