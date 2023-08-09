import { BackupSetting } from "./backup";
import { EMPTY_ID, UNKNOWN_ID } from "./const";
import { DataSource } from "./dataSource";
import { Database } from "./database";
import { Environment } from "./environment";
import { CommandId, CommandRegisterId, PrincipalId } from "./id";
import { Instance } from "./instance";
import { Issue } from "./issue";
import { Pipeline, Stage, Task, TaskProgress } from "./pipeline";
import { Principal } from "./principal";
import { Project, ProjectMember } from "./project";
import { SQLReviewPolicy } from "./sqlReview";
import { VCS } from "./vcs";

// System bot id
export const SYSTEM_BOT_ID = 1;
// System bot email
export const SYSTEM_BOT_EMAIL = "support@bytebase.com";

// The project to hold those databases synced from the instance but haven't been assigned an application
// project yet. We can't use UNKNOWN_ID because of referential integrity.
export const DEFAULT_PROJECT_ID = 1;

export const ALL_DATABASE_NAME = "*";

// For text input, we do validation if there is no further keystroke after 1s
export const TEXT_VALIDATION_DELAY = 1000;

// Normally, we poll issue every 10s to fetch any update from the server side.
// If change occurs, then we will start the poll from 0.2s, 0.4s, 0.8s, 1.6s, 3.2s, 5s, 10s, 10s ... with jitter
// We do this because new update is more likely to happen after the initial change (e.g task gets new update after changing its status)
export const NORMAL_POLL_INTERVAL = 10000;
export const MINIMUM_POLL_INTERVAL = 200;
// Add jitter to avoid timer from different clients converging to the same polling frequency.
export const POLL_JITTER = 200;

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
  vcsSlug?: string;
  connectionSlug?: string;
  sheetSlug?: string;
  sqlReviewPolicySlug?: string;
  ssoName?: string;

  // Resource names.
  projectName?: string;
  databaseGroupName?: string;
  schemaGroupName?: string;
};

// Quick Action Type
export type Command = {
  id: CommandId;
  registerId: CommandRegisterId;
  run: () => void;
};

export type ResourceType =
  | "PRINCIPAL"
  | "ENVIRONMENT"
  | "PROJECT"
  | "PROJECT_MEMBER"
  | "INSTANCE"
  | "DATABASE"
  | "DATA_SOURCE"
  | "BACKUP_SETTING"
  | "ISSUE"
  | "PIPELINE"
  | "POLICY"
  | "STAGE"
  | "TASK_PROGRESS"
  | "TASK"
  | "ACTIVITY"
  | "INBOX"
  | "VCS"
  | "REPOSITORY"
  | "ANOMALY"
  | "DEPLOYMENT_CONFIG"
  | "SHEET"
  | "SQL_REVIEW"
  | "AUDIT_LOG";

interface ResourceMaker {
  (type: "PRINCIPAL"): Principal;
  (type: "ENVIRONMENT"): Environment;
  (type: "PROJECT"): Project;
  (type: "PROJECT_MEMBER"): ProjectMember;
  (type: "INSTANCE"): Instance;
  (type: "DATABASE"): Database;
  (type: "DATA_SOURCE"): DataSource;
  (type: "BACKUP_SETTING"): BackupSetting;
  (type: "ISSUE"): Issue;
  (type: "PIPELINE"): Pipeline;
  (type: "STAGE"): Stage;
  (type: "TASK_PROGRESS"): TaskProgress;
  (type: "TASK"): Task;
  (type: "VCS"): VCS;
  (type: "SQL_REVIEW"): SQLReviewPolicy;
}

const makeUnknown = (type: ResourceType) => {
  // Have to omit creator and updater to avoid recursion.
  const UNKNOWN_PRINCIPAL: Principal = {
    id: UNKNOWN_ID,
    type: "END_USER",
    name: "<<Unknown principal>>",
    email: "",
    role: "DEVELOPER",
  } as Principal;

  const UNKNOWN_ENVIRONMENT: Environment = {
    id: UNKNOWN_ID,
    resourceId: "",
    rowStatus: "NORMAL",
    name: "<<Unknown environment>>",
    order: 0,
    tier: "UNPROTECTED",
  };

  const UNKNOWN_PROJECT: Project = {
    id: UNKNOWN_ID,
    resourceId: "",
    rowStatus: "NORMAL",
    name: "<<Unknown project>>",
    key: "UNK",
    memberList: [],
    workflowType: "UI",
    visibility: "PUBLIC",
    tenantMode: "DISABLED",
    dbNameTemplate: "",
    schemaChangeType: "DDL",
  };

  const UNKNOWN_PROJECT_MEMBER: ProjectMember = {
    id: `projects/${UNKNOWN_ID}/roles/${UNKNOWN_ID}/principals/${UNKNOWN_ID}`,
    project: UNKNOWN_PROJECT,
    role: "DEVELOPER",
    principal: UNKNOWN_PRINCIPAL,
  };

  const UNKNOWN_INSTANCE: Instance = {
    id: UNKNOWN_ID,
    resourceId: "",
    rowStatus: "NORMAL",
    environment: UNKNOWN_ENVIRONMENT,
    dataSourceList: [],
    name: "<<Unknown instance>>",
    engine: "MYSQL",
    engineVersion: "",
    externalLink: "",
    srv: false,
    authenticationDatabase: "",
  };

  const UNKNOWN_DATABASE: Database = {
    id: UNKNOWN_ID,
    instanceId: UNKNOWN_ID,
    instance: UNKNOWN_INSTANCE,
    projectId: UNKNOWN_ID,
    project: UNKNOWN_PROJECT,
    labels: [],
    dataSourceList: [],
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
    name: "<<Unknown data source>>",
    type: "RO",
    username: "",
    password: "",
    sslCa: "",
    sslCert: "",
    sslKey: "",
    host: "",
    port: "",
    database: "",
    options: {
      srv: false,
      authenticationDatabase: "",
      sid: "",
      serviceName: "",
      sshHost: "",
      sshPort: "",
      sshUser: "",
      sshPassword: "",
      sshPrivateKey: "",
    },
    // UI-only fields
    updateSsl: false,
  };

  const UNKNOWN_BACKUP_SETTING: BackupSetting = {
    id: UNKNOWN_ID,
    databaseId: UNKNOWN_ID,
    enabled: false,
    hour: 0,
    dayOfWeek: 0,
    hookUrl: "",
    retentionPeriodTs: 0,
  };

  const UNKNOWN_PIPELINE: Pipeline = {
    id: UNKNOWN_ID,
    name: "<<Unknown pipeline>>",
    stageList: [],
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
    assigneeNeedAttention: false,
    subscriberList: [],
    payload: {},
  };

  const UNKNOWN_STAGE: Stage = {
    id: UNKNOWN_ID,
    pipeline: UNKNOWN_PIPELINE,
    name: "<<Unknown stage>>",
    environment: UNKNOWN_ENVIRONMENT,
    taskList: [],
  };

  const UNKNOWN_TASK_PROGRESS: TaskProgress = {
    totalUnit: 0,
    completedUnit: 0,
    createdTs: 0,
    updatedTs: 0,
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
    blockedBy: [],
    progress: { ...UNKNOWN_TASK_PROGRESS },
  };

  const UNKNOWN_VCS: VCS = {
    id: UNKNOWN_ID,
    name: "",
    type: "GITLAB",
    uiType: "GITLAB_SELF_HOST",
    instanceUrl: "",
    apiUrl: "",
    applicationId: "",
    secret: "",
  };

  switch (type) {
    case "PRINCIPAL":
      return UNKNOWN_PRINCIPAL;
    case "ENVIRONMENT":
      return UNKNOWN_ENVIRONMENT;
    case "PROJECT":
      return UNKNOWN_PROJECT;
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
    case "STAGE":
      return UNKNOWN_STAGE;
    case "TASK_PROGRESS":
      return UNKNOWN_TASK_PROGRESS;
    case "TASK":
      return UNKNOWN_TASK;
    case "VCS":
      return UNKNOWN_VCS;
  }
};

export const unknown = makeUnknown as ResourceMaker;

const makeEmpty = (type: ResourceType) => {
  // Have to omit creator and updater to avoid recursion.
  const EMPTY_PRINCIPAL: Principal = {
    id: EMPTY_ID,
    type: "END_USER",
    name: "",
    email: "",
    role: "DEVELOPER",
  } as Principal;

  const EMPTY_ENVIRONMENT: Environment = {
    id: EMPTY_ID,
    resourceId: "",
    rowStatus: "NORMAL",
    name: "",
    order: 0,
    tier: "UNPROTECTED",
  };

  const EMPTY_PROJECT: Project = {
    id: EMPTY_ID,
    resourceId: "",
    rowStatus: "NORMAL",
    name: "",
    key: "",
    memberList: [],
    workflowType: "UI",
    visibility: "PUBLIC",
    tenantMode: "DISABLED",
    dbNameTemplate: "",
    schemaChangeType: "DDL",
  };

  const EMPTY_PROJECT_MEMBER: ProjectMember = {
    id: `projects/${EMPTY_ID}/roles/${EMPTY_ID}/principals/${EMPTY_ID}`,
    project: EMPTY_PROJECT,
    role: "DEVELOPER",
    principal: EMPTY_PRINCIPAL,
  };

  const EMPTY_INSTANCE: Instance = {
    id: EMPTY_ID,
    resourceId: "",
    rowStatus: "NORMAL",
    environment: EMPTY_ENVIRONMENT,
    dataSourceList: [],
    name: "",
    engine: "MYSQL",
    engineVersion: "",
    externalLink: "",
    srv: false,
    authenticationDatabase: "",
  };

  const EMPTY_DATABASE: Database = {
    id: EMPTY_ID,
    instanceId: UNKNOWN_ID,
    instance: EMPTY_INSTANCE,
    projectId: UNKNOWN_ID,
    project: EMPTY_PROJECT,
    dataSourceList: [],
    labels: [],
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
    name: "",
    type: "RO",
    username: "",
    password: "",
    sslCa: "",
    sslCert: "",
    sslKey: "",
    host: "",
    port: "",
    database: "",
    options: {
      srv: false,
      authenticationDatabase: "",
      sid: "",
      serviceName: "",
      sshHost: "",
      sshPort: "",
      sshUser: "",
      sshPassword: "",
      sshPrivateKey: "",
    },
    // UI-only fields
    updateSsl: false,
  };

  const EMPTY_BACKUP_SETTING: BackupSetting = {
    id: EMPTY_ID,
    databaseId: UNKNOWN_ID,
    enabled: false,
    hour: 0,
    dayOfWeek: 0,
    hookUrl: "",
    retentionPeriodTs: 0,
  };

  const EMPTY_PIPELINE: Pipeline = {
    id: EMPTY_ID,
    name: "",
    stageList: [],
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
    assigneeNeedAttention: false,
    subscriberList: [],
    payload: {},
  };

  const EMPTY_STAGE: Stage = {
    id: EMPTY_ID,
    pipeline: EMPTY_PIPELINE,
    name: "",
    environment: EMPTY_ENVIRONMENT,
    taskList: [],
  };

  const EMPTY_TASK_PROGRESS: TaskProgress = {
    totalUnit: 0,
    completedUnit: 0,
    createdTs: 0,
    updatedTs: 0,
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
    blockedBy: [],
    progress: { ...EMPTY_TASK_PROGRESS },
  };

  const EMPTY_VCS: VCS = {
    id: EMPTY_ID,
    name: "",
    type: "GITLAB",
    uiType: "GITLAB_SELF_HOST",
    instanceUrl: "",
    apiUrl: "",
    applicationId: "",
    secret: "",
  };

  switch (type) {
    case "PRINCIPAL":
      return EMPTY_PRINCIPAL;
    case "ENVIRONMENT":
      return EMPTY_ENVIRONMENT;
    case "PROJECT":
      return EMPTY_PROJECT;
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
    case "STAGE":
      return EMPTY_STAGE;
    case "TASK_PROGRESS":
      return EMPTY_TASK_PROGRESS;
    case "TASK":
      return EMPTY_TASK;
    case "VCS":
      return EMPTY_VCS;
  }
};

export const empty = makeEmpty as ResourceMaker;
