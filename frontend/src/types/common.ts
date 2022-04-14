import { Policy } from ".";
import { Activity } from "./activity";
import { Anomaly } from "./anomaly";
import { BackupSetting } from "./backup";
import { Bookmark } from "./bookmark";
import { Database } from "./database";
import { DataSource } from "./dataSource";
import { Environment } from "./environment";
import { CommandId, CommandRegisterId, PrincipalId } from "./id";
import { Inbox } from "./inbox";
import { Instance } from "./instance";
import { Issue } from "./issue";
import { Member } from "./member";
import { Pipeline, Stage, Task } from "./pipeline";
import { Principal } from "./principal";
import {
  Project,
  ProjectMember,
  EmptyProjectRoleProviderPayload,
} from "./project";
import { ProjectWebhook } from "./projectWebhook";
import { Repository } from "./repository";
import { VCS } from "./vcs";
import { DeploymentConfig } from "./deployment";
import { DefaultApporvalPolicy } from "./policy";
import { Sheet } from "./sheet";
import { DatabaseSchemaGuide } from "./schemaSystem";

// System bot id
export const SYSTEM_BOT_ID = 1;

// The project to hold those databases synced from the instance but haven't been assigned an application
// project yet. We can't use UNKNOWN_ID because of referential integrity.
export const DEFAULT_PROJECT_ID = 1;

export const ALL_DATABASE_NAME = "*";

// For onboarding
export const ONBOARDING_ISSUE_ID = 101;

// For text input, we do validation if there is no further keystroke after 1s
export const TEXT_VALIDATION_DELAY = 1000;

// Normally, we poll issue every 30s to fetch any update from the server side.
// If change occurs, then we will start the poll from 1s, 2s, 4s, 8s, 16s, 30s, 30s ... with jitter
// We do this because new update is more likely to happen after the initial change (e.g task gets new update after changing its status)
export const NORMAL_POLL_INTERVAL = 30000;
export const POST_CHANGE_POLL_INTERVAL = 1000;
// Add jitter to avoid timer from different clients converging to the same polling frequency.
export const POLL_JITTER = 5000;

// It may take a while to perform instance related operations since we are
// connecting the remote instance. And certain operations just take longer for
// a particular database type due to its unique property (e.g. create migration schema
// is a heavier operation in TiDB than traditional RDBMS).
export const INSTANCE_OPERATION_TIMEOUT = 60000;

// RowStatus
export type RowStatus = "NORMAL" | "ARCHIVED";

// Router
export type RouterSlug = {
  principalId?: PrincipalId;
  environmentSlug?: string;
  projectSlug?: string;
  projectWebhookSlug?: string;
  issueSlug?: string;
  instanceSlug?: string;
  databaseSlug?: string;
  tableName?: string;
  dataSourceSlug?: string;
  migrationHistorySlug?: string;
  vcsSlug?: string;
  connectionSlug?: string;
  sheetSlug?: string;
  schemaGuideSlug?: string;
};

// Quick Action Type
export type Command = {
  id: CommandId;
  registerId: CommandRegisterId;
  run: () => void;
};

export type EnvironmentQuickActionType =
  | "quickaction.bb.environment.create"
  | "quickaction.bb.environment.reorder";
export type ProjectQuickActionType =
  | "quickaction.bb.project.create"
  | "quickaction.bb.project.default"
  | "quickaction.bb.project.database.transfer";
export type InstanceQuickActionType = "quickaction.bb.instance.create";
export type UserQuickActionType = "quickaction.bb.user.manage";
export type DatabaseQuickActionType =
  | "quickaction.bb.database.create" // Used by DBA and Owner
  | "quickaction.bb.database.request" // Used by Developer
  | "quickaction.bb.database.schema.update"
  | "quickaction.bb.database.data.update"
  | "quickaction.bb.database.troubleshoot";

export type QuickActionType =
  | EnvironmentQuickActionType
  | ProjectQuickActionType
  | InstanceQuickActionType
  | UserQuickActionType
  | DatabaseQuickActionType;

// unknown represents an anomaly.
// Returns as function to avoid caller accidentally mutate it.
// UNKNOWN_ID means an anomaly, it expects a resource which is missing (e.g. Keyed lookup missing).
export const UNKNOWN_ID = -1;
// EMPTY_ID means an expected behavior, it expects no resource (e.g. contains an empty value, using this technic enables
// us to declare variable as required, which leads to cleaner code)
export const EMPTY_ID = 0;

export type ResourceType =
  | "PRINCIPAL"
  | "MEMBER"
  | "ENVIRONMENT"
  | "PROJECT"
  | "PROJECT_HOOK"
  | "PROJECT_MEMBER"
  | "INSTANCE"
  | "DATABASE"
  | "DATA_SOURCE"
  | "BACKUP_SETTING"
  | "ISSUE"
  | "PIPELINE"
  | "POLICY"
  | "STAGE"
  | "TASK"
  | "ACTIVITY"
  | "INBOX"
  | "BOOKMARK"
  | "VCS"
  | "REPOSITORY"
  | "ANOMALY"
  | "DEPLOYMENT_CONFIG"
  | "SHEET"
  | "SCHEMA_GUIDE";

export const unknown = (
  type: ResourceType
):
  | Principal
  | Member
  | Environment
  | Project
  | ProjectWebhook
  | ProjectMember
  | Instance
  | Database
  | DataSource
  | BackupSetting
  | Issue
  | Pipeline
  | Policy
  | Stage
  | Task
  | Activity
  | Inbox
  | Bookmark
  | VCS
  | Repository
  | Anomaly
  | DeploymentConfig
  | Sheet
  | DatabaseSchemaGuide => {
  // Have to omit creator and updater to avoid recursion.
  const UNKNOWN_PRINCIPAL: Principal = {
    id: UNKNOWN_ID,
    creatorId: UNKNOWN_ID,
    createdTs: 0,
    updaterId: UNKNOWN_ID,
    updatedTs: 0,
    type: "END_USER",
    name: "<<Unknown principal>>",
    email: "",
    role: "DEVELOPER",
  } as Principal;

  const UNKNOWN_MEMBER: Member = {
    id: UNKNOWN_ID,
    rowStatus: "NORMAL",
    creator: UNKNOWN_PRINCIPAL,
    updater: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    status: "ACTIVE",
    role: "DEVELOPER",
    principal: UNKNOWN_PRINCIPAL,
  };

  const UNKNOWN_ENVIRONMENT: Environment = {
    id: UNKNOWN_ID,
    creator: UNKNOWN_PRINCIPAL,
    updater: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    rowStatus: "NORMAL",
    name: "<<Unknown environment>>",
    order: 0,
  };

  const UNKNOWN_PROJECT: Project = {
    id: UNKNOWN_ID,
    rowStatus: "NORMAL",
    name: "<<Unknown project>>",
    key: "UNK",
    creator: UNKNOWN_PRINCIPAL,
    updater: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    memberList: [],
    workflowType: "UI",
    visibility: "PUBLIC",
    tenantMode: "DISABLED",
    dbNameTemplate: "",
    roleProvider: "BYTEBASE",
  };

  const UNKNOWN_PROJECT_HOOK: ProjectWebhook = {
    id: UNKNOWN_ID,
    projectId: UNKNOWN_ID,
    creator: UNKNOWN_PRINCIPAL,
    updater: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    type: "",
    name: "",
    url: "",
    activityList: [],
  };

  const UNKNOWN_PROJECT_MEMBER: ProjectMember = {
    id: UNKNOWN_ID,
    project: UNKNOWN_PROJECT,
    creator: UNKNOWN_PRINCIPAL,
    updater: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    role: "DEVELOPER",
    principal: UNKNOWN_PRINCIPAL,
    roleProvider: "BYTEBASE",
    payload: EmptyProjectRoleProviderPayload,
  };

  const UNKNOWN_INSTANCE: Instance = {
    id: UNKNOWN_ID,
    rowStatus: "NORMAL",
    environment: UNKNOWN_ENVIRONMENT,
    anomalyList: [],
    dataSourceList: [],
    creator: UNKNOWN_PRINCIPAL,
    updater: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    name: "<<Unknown instance>>",
    engine: "MYSQL",
    engineVersion: "",
    host: "",
  };

  const UNKNOWN_DATABASE: Database = {
    id: UNKNOWN_ID,
    instanceId: UNKNOWN_ID,
    instance: UNKNOWN_INSTANCE,
    projectId: UNKNOWN_ID,
    project: UNKNOWN_PROJECT,
    labels: [],
    dataSourceList: [],
    anomalyList: [],
    creator: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updater: UNKNOWN_PRINCIPAL,
    updatedTs: 0,
    name: "<<Unknown database>>",
    characterSet: "",
    collation: "",
    syncStatus: "NOT_FOUND",
    lastSuccessfulSyncTs: 0,
    schemaVersion: "",
  };

  const UNKNOWN_DATA_SOURCE: DataSource = {
    id: UNKNOWN_ID,
    instanceId: UNKNOWN_ID,
    databaseId: UNKNOWN_ID,
    creator: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updater: UNKNOWN_PRINCIPAL,
    updatedTs: 0,
    name: "<<Unknown data source>>",
    type: "RO",
  };

  const UNKNOWN_BACKUP_SETTING: BackupSetting = {
    id: UNKNOWN_ID,
    databaseId: UNKNOWN_ID,
    creator: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updater: UNKNOWN_PRINCIPAL,
    updatedTs: 0,
    enabled: false,
    hour: 0,
    dayOfWeek: 0,
    hookUrl: "",
  };

  const UNKNOWN_PIPELINE: Pipeline = {
    id: UNKNOWN_ID,
    creator: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updater: UNKNOWN_PRINCIPAL,
    updatedTs: 0,
    name: "<<Unknown pipeline>>",
    status: "DONE",
    stageList: [],
  };

  const UNKNOWN_POLICY: Policy = {
    id: UNKNOWN_ID,
    creator: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updater: UNKNOWN_PRINCIPAL,
    updatedTs: 0,
    environment: UNKNOWN_ENVIRONMENT,
    type: "bb.policy.pipeline-approval",
    payload: {
      value: DefaultApporvalPolicy,
    },
  };

  const UNKNOWN_ISSUE: Issue = {
    id: UNKNOWN_ID,
    project: UNKNOWN_PROJECT,
    pipeline: UNKNOWN_PIPELINE,
    creator: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updater: UNKNOWN_PRINCIPAL,
    updatedTs: 0,
    name: "<<Unknown issue>>",
    status: "DONE",
    type: "bb.issue.general",
    description: "",
    assignee: UNKNOWN_PRINCIPAL,
    subscriberList: [],
    payload: {},
  };

  const UNKNOWN_STAGE: Stage = {
    id: UNKNOWN_ID,
    pipeline: UNKNOWN_PIPELINE,
    creator: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updater: UNKNOWN_PRINCIPAL,
    updatedTs: 0,
    name: "<<Unknown stage>>",
    environment: UNKNOWN_ENVIRONMENT,
    taskList: [],
  };

  const UNKNOWN_TASK: Task = {
    id: UNKNOWN_ID,
    pipeline: UNKNOWN_PIPELINE,
    stage: UNKNOWN_STAGE,
    creator: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updater: UNKNOWN_PRINCIPAL,
    updatedTs: 0,
    name: "<<Unknown task>>",
    type: "bb.task.general",
    status: "DONE",
    instance: UNKNOWN_INSTANCE,
    database: UNKNOWN_DATABASE,
    earliestAllowedTs: 0,
    taskRunList: [],
    taskCheckRunList: [],
  };

  const UNKNOWN_ACTIVITY: Activity = {
    id: UNKNOWN_ID,
    creator: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updater: UNKNOWN_PRINCIPAL,
    updatedTs: 0,
    containerId: UNKNOWN_ID,
    type: "bb.issue.create",
    level: "INFO",
    comment: "<<Unknown comment>>",
  };

  const UNKNOWN_INBOX: Inbox = {
    id: UNKNOWN_ID,
    receiver_id: UNKNOWN_ID,
    activity: UNKNOWN_ACTIVITY,
    status: "READ",
  };

  const UNKNOWN_BOOKMARK: Bookmark = {
    id: UNKNOWN_ID,
    creator: UNKNOWN_PRINCIPAL,
    updater: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    name: "",
    link: "",
  };

  const UNKNOWN_VCS: VCS = {
    id: UNKNOWN_ID,
    creator: UNKNOWN_PRINCIPAL,
    updater: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    name: "",
    type: "GITLAB_SELF_HOST",
    instanceUrl: "",
    apiUrl: "",
    applicationId: "",
    secret: "",
  };

  const UNKONWN_REPOSITORY: Repository = {
    id: UNKNOWN_ID,
    creator: UNKNOWN_PRINCIPAL,
    updater: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    vcs: UNKNOWN_VCS,
    project: UNKNOWN_PROJECT,
    name: "",
    fullPath: "",
    webUrl: "",
    baseDirectory: "",
    branchFilter: "",
    filePathTemplate: "",
    schemaPathTemplate: "",
    externalId: UNKNOWN_ID.toString(),
  };

  const UNKNOWN_ANOMALY: Anomaly = {
    id: UNKNOWN_ID,
    creator: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updater: UNKNOWN_PRINCIPAL,
    updatedTs: 0,
    instanceId: UNKNOWN_ID,
    instance: UNKNOWN_INSTANCE,
    databaseId: UNKNOWN_ID,
    database: UNKNOWN_DATABASE,
    type: "bb.anomaly.database.backup.policy-violation",
    severity: "MEDIUM",
    payload: {
      environmentId: UNKNOWN_ID,
      expectedSchedule: "DAILY",
      actualSchedule: "UNSET",
    },
  };

  const UNKNOWN_DEPLOYMENT_CONFIG: DeploymentConfig = {
    id: UNKNOWN_ID,
    creator: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updater: UNKNOWN_PRINCIPAL,
    updatedTs: 0,
    project: UNKNOWN_PROJECT,
    schedule: {
      deployments: [],
    },
  };

  const UNKNOWN_SHEET: Sheet = {
    id: UNKNOWN_ID,
    creator: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updater: UNKNOWN_PRINCIPAL,
    updatedTs: 0,
    projectId: UNKNOWN_ID,
    project: UNKNOWN_PROJECT,
    databaseId: UNKNOWN_ID,
    database: UNKNOWN_DATABASE,
    name: "<<Unknown sheet>>",
    statement: "",
    visibility: "PRIVATE",
  };

  const UNKNOWN_SCHEMA_GUIDE: DatabaseSchemaGuide = {
    id: UNKNOWN_ID,
    name: "",
    createdTs: 0,
    updatedTs: 0,
    environmentList: [],
    ruleList: [],
  };

  switch (type) {
    case "PRINCIPAL":
      return UNKNOWN_PRINCIPAL;
    case "MEMBER":
      return UNKNOWN_MEMBER;
    case "ENVIRONMENT":
      return UNKNOWN_ENVIRONMENT;
    case "PROJECT":
      return UNKNOWN_PROJECT;
    case "PROJECT_HOOK":
      return UNKNOWN_PROJECT_HOOK;
    case "PROJECT_MEMBER":
      return UNKNOWN_PROJECT_MEMBER;
    case "INSTANCE":
      return UNKNOWN_INSTANCE;
    case "DATABASE":
      return UNKNOWN_DATABASE;
    case "DATA_SOURCE":
      return UNKNOWN_DATA_SOURCE;
    case "BACKUP_SETTING":
      return UNKNOWN_BACKUP_SETTING;
    case "ISSUE":
      return UNKNOWN_ISSUE;
    case "PIPELINE":
      return UNKNOWN_PIPELINE;
    case "POLICY":
      return UNKNOWN_POLICY;
    case "STAGE":
      return UNKNOWN_STAGE;
    case "TASK":
      return UNKNOWN_TASK;
    case "ACTIVITY":
      return UNKNOWN_ACTIVITY;
    case "INBOX":
      return UNKNOWN_INBOX;
    case "BOOKMARK":
      return UNKNOWN_BOOKMARK;
    case "VCS":
      return UNKNOWN_VCS;
    case "REPOSITORY":
      return UNKONWN_REPOSITORY;
    case "ANOMALY":
      return UNKNOWN_ANOMALY;
    case "DEPLOYMENT_CONFIG":
      return UNKNOWN_DEPLOYMENT_CONFIG;
    case "SHEET":
      return UNKNOWN_SHEET;
    case "SCHEMA_GUIDE":
      return UNKNOWN_SCHEMA_GUIDE;
  }
};

// empty represents an expected behavior.
export const empty = (
  type: ResourceType
):
  | Principal
  | Member
  | Environment
  | Project
  | ProjectWebhook
  | ProjectMember
  | Instance
  | Database
  | DataSource
  | BackupSetting
  | Issue
  | Pipeline
  | Policy
  | Stage
  | Task
  | Activity
  | Inbox
  | Bookmark
  | VCS
  | Repository
  | Anomaly
  | DeploymentConfig
  | Sheet
  | DatabaseSchemaGuide => {
  // Have to omit creator and updater to avoid recursion.
  const EMPTY_PRINCIPAL: Principal = {
    id: EMPTY_ID,
    createdTs: 0,
    updatedTs: 0,
    type: "END_USER",
    name: "",
    email: "",
    role: "DEVELOPER",
  } as Principal;

  const EMPTY_MEMBER: Member = {
    id: EMPTY_ID,
    rowStatus: "NORMAL",
    creator: EMPTY_PRINCIPAL,
    updater: EMPTY_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    status: "ACTIVE",
    role: "DEVELOPER",
    principal: EMPTY_PRINCIPAL,
  };

  const EMPTY_ENVIRONMENT: Environment = {
    id: EMPTY_ID,
    creator: EMPTY_PRINCIPAL,
    updater: EMPTY_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    rowStatus: "NORMAL",
    name: "",
    order: 0,
  };

  const EMPTY_PROJECT: Project = {
    id: EMPTY_ID,
    rowStatus: "NORMAL",
    name: "",
    key: "",
    creator: EMPTY_PRINCIPAL,
    updater: EMPTY_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    memberList: [],
    workflowType: "UI",
    visibility: "PUBLIC",
    tenantMode: "DISABLED",
    dbNameTemplate: "",
    roleProvider: "BYTEBASE",
  };

  const EMPTY_PROJECT_HOOK: ProjectWebhook = {
    id: EMPTY_ID,
    projectId: EMPTY_ID,
    creator: EMPTY_PRINCIPAL,
    updater: EMPTY_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    type: "",
    name: "",
    url: "",
    activityList: [],
  };

  const EMPTY_PROJECT_MEMBER: ProjectMember = {
    id: EMPTY_ID,
    project: EMPTY_PROJECT,
    creator: EMPTY_PRINCIPAL,
    updater: EMPTY_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    role: "DEVELOPER",
    principal: EMPTY_PRINCIPAL,
    roleProvider: "BYTEBASE",
    payload: EmptyProjectRoleProviderPayload,
  };

  const EMPTY_INSTANCE: Instance = {
    id: EMPTY_ID,
    rowStatus: "NORMAL",
    environment: EMPTY_ENVIRONMENT,
    anomalyList: [],
    dataSourceList: [],
    creator: EMPTY_PRINCIPAL,
    updater: EMPTY_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    name: "",
    engine: "MYSQL",
    engineVersion: "",
    host: "",
  };

  const EMPTY_DATABASE: Database = {
    id: EMPTY_ID,
    instanceId: UNKNOWN_ID,
    instance: EMPTY_INSTANCE,
    projectId: UNKNOWN_ID,
    project: EMPTY_PROJECT,
    dataSourceList: [],
    anomalyList: [],
    labels: [],
    creator: EMPTY_PRINCIPAL,
    createdTs: 0,
    updater: EMPTY_PRINCIPAL,
    updatedTs: 0,
    name: "",
    characterSet: "",
    collation: "",
    syncStatus: "NOT_FOUND",
    lastSuccessfulSyncTs: 0,
    schemaVersion: "",
  };

  const EMPTY_DATA_SOURCE: DataSource = {
    id: EMPTY_ID,
    instanceId: UNKNOWN_ID,
    databaseId: UNKNOWN_ID,
    creator: EMPTY_PRINCIPAL,
    createdTs: 0,
    updater: EMPTY_PRINCIPAL,
    updatedTs: 0,
    name: "",
    type: "RO",
  };

  const EMPTY_BACKUP_SETTING: BackupSetting = {
    id: EMPTY_ID,
    databaseId: UNKNOWN_ID,
    creator: EMPTY_PRINCIPAL,
    createdTs: 0,
    updater: EMPTY_PRINCIPAL,
    updatedTs: 0,
    enabled: false,
    hour: 0,
    dayOfWeek: 0,
    hookUrl: "",
  };

  const EMPTY_PIPELINE: Pipeline = {
    id: EMPTY_ID,
    creator: EMPTY_PRINCIPAL,
    createdTs: 0,
    updater: EMPTY_PRINCIPAL,
    updatedTs: 0,
    name: "",
    status: "DONE",
    stageList: [],
  };

  const EMPTY_POLICY: Policy = {
    id: EMPTY_ID,
    creator: EMPTY_PRINCIPAL,
    createdTs: 0,
    updater: EMPTY_PRINCIPAL,
    updatedTs: 0,
    environment: EMPTY_ENVIRONMENT,
    type: "bb.policy.pipeline-approval",
    payload: {
      value: DefaultApporvalPolicy,
    },
  };

  const EMPTY_ISSUE: Issue = {
    id: EMPTY_ID,
    pipeline: EMPTY_PIPELINE,
    project: EMPTY_PROJECT,
    creator: EMPTY_PRINCIPAL,
    createdTs: 0,
    updater: EMPTY_PRINCIPAL,
    updatedTs: 0,
    name: "",
    status: "DONE",
    type: "bb.issue.general",
    description: "",
    assignee: EMPTY_PRINCIPAL,
    subscriberList: [],
    payload: {},
  };

  const EMPTY_STAGE: Stage = {
    id: EMPTY_ID,
    pipeline: EMPTY_PIPELINE,
    creator: EMPTY_PRINCIPAL,
    createdTs: 0,
    updater: EMPTY_PRINCIPAL,
    updatedTs: 0,
    name: "",
    environment: EMPTY_ENVIRONMENT,
    taskList: [],
  };

  const EMPTY_TASK: Task = {
    id: EMPTY_ID,
    pipeline: EMPTY_PIPELINE,
    stage: EMPTY_STAGE,
    creator: EMPTY_PRINCIPAL,
    createdTs: 0,
    updater: EMPTY_PRINCIPAL,
    updatedTs: 0,
    name: "",
    type: "bb.task.general",
    status: "DONE",
    instance: EMPTY_INSTANCE,
    database: EMPTY_DATABASE,
    taskRunList: [],
    taskCheckRunList: [],
    earliestAllowedTs: 0,
  };

  const EMPTY_ACTIVITY: Activity = {
    id: EMPTY_ID,
    creator: EMPTY_PRINCIPAL,
    createdTs: 0,
    updater: EMPTY_PRINCIPAL,
    updatedTs: 0,
    containerId: EMPTY_ID,
    type: "bb.issue.create",
    level: "INFO",
    comment: "",
  };

  const EMPTY_INBOX: Inbox = {
    id: EMPTY_ID,
    receiver_id: EMPTY_ID,
    activity: EMPTY_ACTIVITY,
    status: "READ",
  };

  const EMPTY_BOOKMARK: Bookmark = {
    id: EMPTY_ID,
    creator: EMPTY_PRINCIPAL,
    updater: EMPTY_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    name: "",
    link: "",
  };

  const EMPTY_VCS: VCS = {
    id: EMPTY_ID,
    creator: EMPTY_PRINCIPAL,
    updater: EMPTY_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    name: "",
    type: "GITLAB_SELF_HOST",
    instanceUrl: "",
    apiUrl: "",
    applicationId: "",
    secret: "",
  };

  const EMPTY_REPOSITORY: Repository = {
    id: EMPTY_ID,
    creator: EMPTY_PRINCIPAL,
    updater: EMPTY_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    vcs: EMPTY_VCS,
    project: EMPTY_PROJECT,
    name: "",
    fullPath: "",
    webUrl: "",
    baseDirectory: "",
    branchFilter: "",
    filePathTemplate: "",
    schemaPathTemplate: "",
    externalId: EMPTY_ID.toString(),
  };

  const EMPTY_ANOMALY: Anomaly = {
    id: EMPTY_ID,
    creator: EMPTY_PRINCIPAL,
    createdTs: 0,
    updater: EMPTY_PRINCIPAL,
    updatedTs: 0,
    instanceId: EMPTY_ID,
    instance: EMPTY_INSTANCE,
    databaseId: EMPTY_ID,
    database: EMPTY_DATABASE,
    type: "bb.anomaly.database.backup.policy-violation",
    severity: "MEDIUM",
    payload: {
      environmentId: EMPTY_ID,
      expectedSchedule: "DAILY",
      actualSchedule: "UNSET",
    },
  };

  const EMPTY_DEPLOYMENT_CONFIG: DeploymentConfig = {
    id: EMPTY_ID,
    creator: EMPTY_PRINCIPAL,
    createdTs: 0,
    updater: EMPTY_PRINCIPAL,
    updatedTs: 0,
    project: EMPTY_PROJECT,
    schedule: {
      deployments: [],
    },
  };

  const EMPTY_SHEET: Sheet = {
    id: EMPTY_ID,
    creator: EMPTY_PRINCIPAL,
    createdTs: 0,
    updater: EMPTY_PRINCIPAL,
    updatedTs: 0,
    projectId: EMPTY_ID,
    project: EMPTY_PROJECT,
    databaseId: EMPTY_ID,
    database: EMPTY_DATABASE,
    name: "<<Empty sheet>>",
    statement: "",
    visibility: "PRIVATE",
  };

  const EMPTY_SCHEMA_GUIDE: DatabaseSchemaGuide = {
    id: EMPTY_ID,
    name: "",
    createdTs: 0,
    updatedTs: 0,
    environmentList: [],
    ruleList: [],
  };

  switch (type) {
    case "PRINCIPAL":
      return EMPTY_PRINCIPAL;
    case "MEMBER":
      return EMPTY_MEMBER;
    case "ENVIRONMENT":
      return EMPTY_ENVIRONMENT;
    case "PROJECT":
      return EMPTY_PROJECT;
    case "PROJECT_HOOK":
      return EMPTY_PROJECT_HOOK;
    case "PROJECT_MEMBER":
      return EMPTY_PROJECT_MEMBER;
    case "INSTANCE":
      return EMPTY_INSTANCE;
    case "DATABASE":
      return EMPTY_DATABASE;
    case "DATA_SOURCE":
      return EMPTY_DATA_SOURCE;
    case "BACKUP_SETTING":
      return EMPTY_BACKUP_SETTING;
    case "ISSUE":
      return EMPTY_ISSUE;
    case "PIPELINE":
      return EMPTY_PIPELINE;
    case "POLICY":
      return EMPTY_POLICY;
    case "STAGE":
      return EMPTY_STAGE;
    case "TASK":
      return EMPTY_TASK;
    case "ACTIVITY":
      return EMPTY_ACTIVITY;
    case "INBOX":
      return EMPTY_INBOX;
    case "BOOKMARK":
      return EMPTY_BOOKMARK;
    case "VCS":
      return EMPTY_VCS;
    case "REPOSITORY":
      return EMPTY_REPOSITORY;
    case "ANOMALY":
      return EMPTY_ANOMALY;
    case "DEPLOYMENT_CONFIG":
      return EMPTY_DEPLOYMENT_CONFIG;
    case "SHEET":
      return EMPTY_SHEET;
    case "SCHEMA_GUIDE":
      return EMPTY_SCHEMA_GUIDE;
  }
};
