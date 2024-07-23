import { EMPTY_ID, UNKNOWN_ID } from "./const";
import type { DataSource } from "./dataSource";
import type { Database } from "./database";
import type { Environment } from "./environment";
import type { CommandId, CommandRegisterId } from "./id";
import type { Instance } from "./instance";
import type { Principal } from "./principal";
import type { Project, ProjectMember } from "./project";
import type { SQLReviewPolicy } from "./sqlReview";

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

// Link to inquire enterprise plan
export const ENTERPRISE_INQUIRE_LINK =
  "https://www.bytebase.com/contact-us?source=console";

// RowStatus
export type RowStatus = "NORMAL" | "ARCHIVED";

// Router
export type RouterSlug = {
  issueSlug?: string;
  connectionSlug?: string;
  sheetSlug?: string;
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
  | "POLICY"
  | "ACTIVITY"
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
      authenticationPrivateKey: "",
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

  const UNKNOWN_SQL_REVIEW_POLICY: SQLReviewPolicy = {
    id: `${UNKNOWN_ID}`,
    enforce: false,
    name: "",
    ruleList: [],
    resources: [],
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
    case "SQL_REVIEW":
      return UNKNOWN_SQL_REVIEW_POLICY;
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
      authenticationPrivateKey: "",
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
  }
};

export const empty = makeEmpty as ResourceMaker;
