import { ResourceObject } from "./jsonapi";
import { BBNotificationStyle } from "../bbkit/types";
import { FieldId } from "../plugins";

export const UNKNOWN_ID = "-1";

export type ResourceType =
  | "PRINCIPAL"
  | "EXECUTION"
  | "USER"
  | "ROLE_MAPPING"
  | "ENVIRONMENT"
  | "PROJECT"
  | "INSTANCE"
  | "DATABASE"
  | "DATA_SOURCE"
  | "TASK"
  | "ACTIVITY"
  | "MESSAGE"
  | "BOOKMARK";

// Returns as function to avoid caller accidentally mutate it.
export const unknown = (
  type: ResourceType
):
  | Execution
  | Principal
  | User
  | RoleMapping
  | Environment
  | Project
  | Instance
  | Database
  | DataSource
  | Task
  | Activity
  | Message
  | Bookmark => {
  const UNKNOWN_EXECUTION: Execution = {
    id: UNKNOWN_ID,
    status: "PENDING",
  };
  const UNKNOWN_PRINCIPAL: Principal = {
    id: UNKNOWN_ID,
    status: "UNKNOWN",
    name: "<<Unknown principal>>",
    email: "",
    role: "GUEST",
  };

  const UNKNOWN_USER: User = {
    id: UNKNOWN_ID,
    status: "UNKNOWN",
    name: "<<Unknown user>>",
    email: "unknown@example.com",
  };

  const UNKNOWN_ROLE_MAPPING: RoleMapping = {
    id: UNKNOWN_ID,
    createdTs: 0,
    lastUpdatedTs: 0,
    role: "GUEST",
    principalId: UNKNOWN_ID,
    updaterId: UNKNOWN_ID,
  };

  const UNKNOWN_ENVIRONMENT: Environment = {
    id: UNKNOWN_ID,
    rowStatus: "NORMAL",
    name: "<<Unknown environment>>",
    order: 0,
  };

  const UNKNOWN_PROJECT: Project = {
    id: UNKNOWN_ID,
    rowStatus: "NORMAL",
    name: "<<Unknown project>>",
    creator: UNKNOWN_PRINCIPAL,
    updater: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    lastUpdatedTs: 0,
  };

  const UNKNOWN_INSTANCE: Instance = {
    id: UNKNOWN_ID,
    rowStatus: "NORMAL",
    environment: UNKNOWN_ENVIRONMENT,
    creator: UNKNOWN_PRINCIPAL,
    updater: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    lastUpdatedTs: 0,
    name: "<<Unknown instance>>",
    host: "",
  };

  const UNKNOWN_DATABASE: Database = {
    id: UNKNOWN_ID,
    instance: UNKNOWN_INSTANCE,
    dataSourceList: [],
    createdTs: 0,
    lastUpdatedTs: 0,
    name: "<<Unknown database>>",
    ownerId: UNKNOWN_ID,
    syncStatus: "NOT_FOUND",
    lastSuccessfulSyncTs: 0,
    fingerprint: "",
  };

  const UNKNOWN_DATA_SOURCE: DataSource = {
    id: UNKNOWN_ID,
    instance: UNKNOWN_INSTANCE,
    database: UNKNOWN_DATABASE,
    memberList: [],
    createdTs: 0,
    lastUpdatedTs: 0,
    name: "<<Unknown data source>>",
    type: "RO",
  };

  const UNKNOWN_TASK: Task = {
    id: UNKNOWN_ID,
    createdTs: 0,
    lastUpdatedTs: 0,
    name: "<<Unknown task>>",
    status: "OPEN",
    type: "bytebase.general",
    description: "",
    stageList: [],
    creator: UNKNOWN_PRINCIPAL,
    subscriberList: [],
    payload: {},
  };

  const UNKNOWN_ACTIVITY: Activity = {
    id: UNKNOWN_ID,
    containerId: UNKNOWN_ID,
    createdTs: 0,
    lastUpdatedTs: 0,
    actionType: "bytebase.task.create",
    creator: UNKNOWN_PRINCIPAL,
    comment: "<<Unknown comment>>",
  };

  const UNKNOWN_MESSAGE: Message = {
    id: UNKNOWN_ID,
    containerId: UNKNOWN_ID,
    createdTs: 0,
    lastUpdatedTs: 0,
    type: "bb.msg.task.assign",
    status: "DELIVERED",
    description: "",
    creator: UNKNOWN_PRINCIPAL,
    receiver: UNKNOWN_PRINCIPAL,
  };

  const UNKNOWN_BOOKMARK: Bookmark = {
    id: UNKNOWN_ID,
    name: "",
    link: "",
    creatorId: UNKNOWN_ID,
  };

  switch (type) {
    case "EXECUTION":
      return UNKNOWN_EXECUTION;
    case "PRINCIPAL":
      return UNKNOWN_PRINCIPAL;
    case "USER":
      return UNKNOWN_USER;
    case "ROLE_MAPPING":
      return UNKNOWN_ROLE_MAPPING;
    case "ENVIRONMENT":
      return UNKNOWN_ENVIRONMENT;
    case "PROJECT":
      return UNKNOWN_PROJECT;
    case "INSTANCE":
      return UNKNOWN_INSTANCE;
    case "DATABASE":
      return UNKNOWN_DATABASE;
    case "DATA_SOURCE":
      return UNKNOWN_DATA_SOURCE;
    case "TASK":
      return UNKNOWN_TASK;
    case "ACTIVITY":
      return UNKNOWN_ACTIVITY;
    case "MESSAGE":
      return UNKNOWN_MESSAGE;
    case "BOOKMARK":
      return UNKNOWN_BOOKMARK;
  }
};

// These ID format may change in the future, so we encapsulate with a type.
// Also good for readability.

export type ExecutionId = string;

export type UserId = string;

// For now, Principal is equal to UserId, in the future it may contain other id such as application, bot etc.
export type PrincipalId = UserId;

export type RoleMappingId = string;

export type BookmarkId = string;

export type ProjectId = string;

export type StageId = string;

export type TaskId = string;

export type ActivityId = string;

export type MessageId = string;

export type EnvironmentId = string;

export type InstanceId = string;

export type DataSourceId = string;

export type DatabaseId = string;
// This is a placeholder database id which will later be converted to
// the actual database id referencing all databases.
export const ALL_DATABASE_PLACEHOLDER_ID = "0";
export const ALL_DATABASE_NAME: string = "*";

export type CommandId = string;
export type CommandRegisterId = string;

// This references to the object id, which can be used as a container.
// Currently, only task can be used a container.
// The type is used by Activity and Message
export type ContainerId = TaskId;

export type BatchUpdate = {
  idList: string[];
  fieldMaskList: string[];
  rowValueList: any[][];
};

// Persistent State Models

// RowStatus
export type RowStatus = "NORMAL" | "ARCHIVED" | "PENDING_DELETE";

// Execution
export type ExecutionStatus = "PENDING" | "RUNNING" | "DONE" | "FAILED";

export type Execution = {
  id: ExecutionId;
  status: ExecutionStatus;
  link?: string;
};

// RoleMapping
export type RoleType = "OWNER" | "DBA" | "DEVELOPER" | "GUEST";

export type RoleMapping = {
  id: RoleMappingId;
  createdTs: number;
  lastUpdatedTs: number;
  role: RoleType;
  principalId: PrincipalId;
  updaterId: PrincipalId;
};

export type RoleMappingNew = {
  principalId: PrincipalId;
  role: RoleType;
  updaterId: PrincipalId;
};

export type RoleMappingPatch = {
  id: RoleMappingId;
  role: RoleType;
};

// Principal
// This is a facet of the underlying identity entity.
// For now, there is only user type. In the future,
// we may support application/bot identity.
export type PrincipalStatus = "UNKNOWN" | "INVITED" | "ACTIVE";

export type Principal = {
  id: PrincipalId;
  status: PrincipalStatus;
  name: string;
  email: string;
  role: RoleType;
};

export type PrincipalNew = {
  name: string;
  email: string;
};

export type PrincipalPatch = {
  name?: string;
};

export type User = {
  id: UserId;
  status: PrincipalStatus;
  name: string;
  email: string;
};

export type UserPatch = {
  name?: string;
};

// Bookmark
export type Bookmark = {
  id: BookmarkId;
  name: string;
  link: string;
  creatorId: UserId;
};
export type BookmarkNew = Omit<Bookmark, "id">;

// Project
export type Project = {
  id: ProjectId;
  rowStatus: RowStatus;
  name: string;
  creator: Principal;
  updater: Principal;
  createdTs: number;
  lastUpdatedTs: number;
};

export type ProjectNew = {
  name: string;
  creatorId: PrincipalId;
};

export type ProjectPatch = {
  name?: string;
};

// Stage
export type StageType =
  | "bytebase.stage.general"
  | "bytebase.stage.transition"
  | "bytebase.stage.database.create"
  | "bytebase.stage.database.grant"
  | "bytebase.stage.schema.update";

export type StageStatus = "PENDING" | "RUNNING" | "DONE" | "FAILED" | "SKIPPED";

export type StageRunnable = {
  auto: boolean;
  run: () => void;
};

export type Stage = {
  id: StageId;
  name: string;
  type: StageType;
  status: StageStatus;
  environmentId?: EnvironmentId;
  databaseId?: DatabaseId;
  runnable?: StageRunnable;
};

export type StageProgressPatch = {
  id: StageId;
  status: StageStatus;
};

// Task
type TaskTypeGeneral = "bytebase.general";

type TaskTypeDatabase =
  | "bytebase.database.create"
  | "bytebase.database.grant"
  | "bytebase.database.schema.update";

type TaskTypeDataSource = "bytebase.datasource.request";

export type TaskType = TaskTypeGeneral | TaskTypeDatabase | TaskTypeDataSource;

export type TaskStatus = "OPEN" | "DONE" | "CANCELED";

export type TaskPayload = { [key: string]: any };

export type Task = {
  id: TaskId;
  createdTs: number;
  lastUpdatedTs: number;
  name: string;
  status: TaskStatus;
  type: TaskType;
  description: string;
  stageList: Stage[];
  creator: Principal;
  assignee?: Principal;
  subscriberList: Principal[];
  sql?: string;
  rollbackSql?: string;
  payload: TaskPayload;
};

export type TaskNew = {
  name: string;
  type: TaskType;
  description: string;
  stageList: Stage[];
  creatorId: PrincipalId;
  assigneeId?: PrincipalId;
  subscriberIdList: PrincipalId[];
  sql?: string;
  rollbackSql?: string;
  payload: TaskPayload;
};

export type TaskPatch = {
  updaterId: PrincipalId;
  name?: string;
  status?: TaskStatus;
  description?: string;
  subscriberIdList?: PrincipalId[];
  sql?: string;
  rollbackSql?: string;
  assigneeId?: PrincipalId;
  stage?: StageProgressPatch;
  payload?: TaskPayload;
  comment?: string;
};

export type TaskStatusTransitionType = "RESOLVE" | "ABORT" | "REOPEN";

export interface TaskStatusTransition {
  type: TaskStatusTransitionType;
  actionName: string;
  to: TaskStatus;
}

export const TASK_STATUS_TRANSITION_LIST: Map<
  TaskStatusTransitionType,
  TaskStatusTransition
> = new Map([
  [
    "RESOLVE",
    {
      type: "RESOLVE",
      actionName: "Resolve",
      to: "DONE",
    },
  ],
  [
    "ABORT",
    {
      type: "ABORT",
      actionName: "Abort",
      to: "CANCELED",
    },
  ],
  [
    "REOPEN",
    {
      type: "REOPEN",
      actionName: "Reopen",
      to: "OPEN",
    },
  ],
]);

export type StageStatusTransitionType = "RUN" | "RETRY" | "STOP" | "SKIP";

export interface StageStatusTransition {
  type: StageStatusTransitionType;
  actionName: string;
  requireRunnable: boolean;
  to: StageStatus;
}

export const STAGE_TRANSITION_LIST: Map<
  StageStatusTransitionType,
  StageStatusTransition
> = new Map([
  [
    "RUN",
    {
      type: "RUN",
      actionName: "Run",
      requireRunnable: true,
      to: "RUNNING",
    },
  ],
  [
    "RETRY",
    {
      type: "RETRY",
      actionName: "Rerun",
      requireRunnable: true,
      to: "RUNNING",
    },
  ],
  [
    "STOP",
    {
      type: "STOP",
      actionName: "Stop",
      requireRunnable: true,
      to: "PENDING",
    },
  ],
  [
    "SKIP",
    {
      type: "SKIP",
      actionName: "Skip",
      requireRunnable: false,
      to: "SKIPPED",
    },
  ],
]);

// Activity
export type TaskActionType =
  | "bytebase.task.create"
  | "bytebase.task.comment.create"
  | "bytebase.task.field.update";

export type ActionType = TaskActionType;

export type ActionTaskFieldUpdatePayload = {
  changeList: {
    fieldId: FieldId;
    oldValue?: string;
    newValue?: string;
  }[];
};

export type ActionPayloadType = ActionTaskFieldUpdatePayload;

export type Activity = {
  id: ActivityId;
  // The object where this activity belongs
  // e.g if actionType is "bytebase.task.xxx", then this field refers to the corresponding task's id.
  containerId: ContainerId;
  createdTs: number;
  lastUpdatedTs: number;
  actionType: ActionType;
  creator: Principal;
  comment: string;
  payload?: ActionPayloadType;
};

export type ActivityNew = {
  containerId: ContainerId;
  actionType: ActionType;
  creatorId: PrincipalId;
  comment: string;
  payload?: ActionPayloadType;
};

export type ActivityPatch = {
  payload: any;
};

// Message
export type MemberMessageType =
  | "bb.msg.member.create"
  | "bb.msg.member.invite"
  | "bb.msg.member.join"
  | "bb.msg.member.revoke"
  | "bb.msg.member.updaterole";

export type EnvironmentMessageType =
  | "bb.msg.environment.create"
  | "bb.msg.environment.update"
  | "bb.msg.environment.delete"
  | "bb.msg.environment.archive"
  | "bb.msg.environment.restore";

export type InstanceMessageType =
  | "bb.msg.instance.create"
  | "bb.msg.instance.update"
  | "bb.msg.instance.delete"
  | "bb.msg.instance.archive"
  | "bb.msg.instance.restore";

export type TaskMessageType =
  | "bb.msg.task.assign"
  | "bb.msg.task.updatestatus"
  | "bb.msg.task.comment";

export type MessageType =
  | MemberMessageType
  | EnvironmentMessageType
  | InstanceMessageType
  | TaskMessageType;

export type MemberMessagePayload = {
  principalId: PrincipalId;
  oldRole?: RoleType;
  newRole?: RoleType;
};

export type EnvironmentUpdateMessagePayload = {
  environmentName: string;
};

export enum EnvironmentBuiltinFieldId {
  ROW_STATUS = "1",
  NAME = "2",
}

export type EnvironmentMessagePayload = {
  environmentName: string;
  changeList: {
    fieldId: EnvironmentBuiltinFieldId;
    oldValue?: any;
    newValue?: any;
  }[];
};

export enum InstanceBuiltinFieldId {
  ROW_STATUS = "1",
  NAME = "2",
  ENVIRONMENT = "3",
  EXTERNAL_LINK = "4",
  HOST = "5",
  PORT = "6",
}

export type InstanceMessagePaylaod = {
  instanceName: string;
};

export type TaskAssignMessagePayload = {
  taskName: string;
  oldAssigneeId: PrincipalId;
  newAssigneeId: PrincipalId;
};

export type TaskUpdateStatusMessagePayload = {
  taskName: string;
  oldStatus: TaskStatus;
  newStatus: TaskStatus;
};

export type TaskCommentMessagePayload = {
  taskName: string;
  commentId: ActivityId;
};

export type MessagePayload =
  | MemberMessagePayload
  | EnvironmentMessagePayload
  | EnvironmentUpdateMessagePayload
  | InstanceMessagePaylaod
  | TaskAssignMessagePayload
  | TaskUpdateStatusMessagePayload
  | TaskCommentMessagePayload;

export type MessageStatus = "DELIVERED" | "CONSUMED";

export type Message = {
  id: MessageId;
  // The object where this message originates
  containerId: ContainerId;
  createdTs: number;
  lastUpdatedTs: number;
  type: MessageType;
  status: MessageStatus;
  description: string;
  creator: Principal;
  receiver: Principal;
  payload?: MessagePayload;
};
export type MessageNew = Omit<Message, "id" | "createdTs" | "lastUpdatedTs">;

export type MessagePatch = {
  status: MessageStatus;
};

// Environment
export type Environment = {
  id: EnvironmentId;
  rowStatus: RowStatus;
  name: string;
  order: number;
};

export type EnvironmentNew = {
  name: string;
};

export type EnvironmentPatch = {
  rowStatus?: RowStatus;
  name?: string;
};

// Instance
export type Instance = {
  id: InstanceId;
  rowStatus: RowStatus;
  environment: Environment;
  creator: Principal;
  updater: Principal;
  createdTs: number;
  lastUpdatedTs: number;
  name: string;
  externalLink?: string;
  host: string;
  port?: string;
};

export type InstanceNew = {
  environmentId: EnvironmentId;
  name: string;
  externalLink?: string;
  host: string;
  port?: string;
};

export type InstancePatch = {
  rowStatus?: RowStatus;
  name?: string;
  externalLink?: string;
  host?: string;
  port?: string;
};

export type DataSourceType = "RW" | "RO";
// Data Source
export type DataSource = {
  id: DataSourceId;
  database: Database;
  instance: Instance;
  // Returns the member list directly because we need it quite frequently in order
  // to do various access check.
  memberList: DataSourceMember[];
  createdTs: number;
  lastUpdatedTs: number;
  name: string;
  type: DataSourceType;
  // In mysql, username can be empty which means anonymous user
  username?: string;
  password?: string;
};

export type DataSourceNew = {
  name: string;
  databaseId: DatabaseId;
  instanceId: InstanceId;
  memberList: DataSourceMemberNew[];
  type: DataSourceType;
  username?: string;
  password?: string;
};

export type DataSourceMember = {
  principal: Principal;
  taskId?: TaskId;
  createdTs: number;
};

export type DataSourceMemberNew = {
  principalId: PrincipalId;
  taskId?: TaskId;
};

// We periodically sync the underlying db schema and stores those info
// in the "database" object.
// Physically, a database belongs to an instance. Logically, it belongs to a project.

// "OK" means find the exact match
// "DRIFTED" means we find the database with the same name, but the fingerprint is different,
//            this usually indicates the underlying database has been recreated (might for a entirely different purpose)
// "NOT_FOUND" means no matching database name found, this ususally means someone changes
//            the underlying db name.
export type DatabaseSyncStatus = "OK" | "DRIFTED" | "NOT_FOUND";
// Database
export type Database = {
  id: DatabaseId;
  instance: Instance;
  project: Project;
  dataSourceList: DataSource[];
  createdTs: number;
  lastUpdatedTs: number;
  name: string;
  ownerId: PrincipalId;
  syncStatus: DatabaseSyncStatus;
  lastSuccessfulSyncTs: number;
  fingerprint: string;
};

export type DatabaseNew = {
  name: string;
  instanceId: InstanceId;
  projectId: ProjectId;
  ownerId: PrincipalId;
  creatorId: PrincipalId;
  taskId?: TaskId;
};

export type DatabasePatch = {
  projectId: ProjectId;
  ownerId: PrincipalId;
};

// Auth
export type LoginInfo = {
  email: string;
  password: string;
};

export type SignupInfo = {
  email: string;
  password: string;
  name: string;
};

export type ActivateInfo = {
  email: string;
  password: string;
  name: string;
  token: string;
};

// Plan
export type FeatureType =
  // - Task can only be assigned to DBA and Owner
  "bytebase.admin";

export enum PlanType {
  FREE = 0,
  TEAM = 1,
  ENTERPRISE = 2,
}

// UI State Models
export type RouterSlug = {
  environmentSlug?: string;
  projectSlug?: string;
  taskSlug?: string;
  instanceSlug?: string;
  databaseSlug?: string;
  dataSourceSlug?: string;
  principalId?: PrincipalId;
};

export type Notification = {
  id: string;
  createdTs: number;
  module: string;
  style: BBNotificationStyle;
  title: string;
  description?: string;
  link?: string;
  linkTitle?: string;
  manualHide?: boolean;
};

export type Command = {
  id: CommandId;
  registerId: CommandRegisterId;
  run: () => void;
};

// Notification
// "id" and "createdTs" is auto generated upon the notification store
// receives.
export type NewNotification = Omit<Notification, "id" | "createdTs">;

export type NotificationFilter = {
  module: string;
  id?: string;
};

// Quick Action Type
export type EnvironmentQuickActionType =
  | "quickaction.bytebase.environment.create"
  | "quickaction.bytebase.environment.reorder";
export type ProjectQuickActionType = "quickaction.bytebase.project.create";
export type InstanceQuickActionType = "quickaction.bytebase.instance.create";
export type UserQuickActionType = "quickaction.bytebase.user.manage";
export type DatabaseQuickActionType =
  | "quickaction.bytebase.database.create" // Used by DBA and Owner
  | "quickaction.bytebase.database.request" // Used by Developer
  | "quickaction.bytebase.database.schema.update"
  | "quickaction.bytebase.database.troubleshoot";

export type QuickActionType =
  | EnvironmentQuickActionType
  | ProjectQuickActionType
  | InstanceQuickActionType
  | UserQuickActionType
  | DatabaseQuickActionType;

// Store
export interface AuthState {
  currentUser: Principal;
}

export interface PlanState {
  plan: PlanType;
}

export interface RoleMappingState {
  roleMappingList: RoleMapping[];
}

export interface PrincipalState {
  principalList: Principal[];
}

export interface BookmarkState {
  bookmarkListByUser: Map<UserId, Bookmark[]>;
}

export interface ActivityState {
  activityListByUser: Map<UserId, Activity[]>;
  activityListByTask: Map<TaskId, Activity[]>;
}

export interface MessageState {
  messageListByUser: Map<UserId, Message[]>;
}

export interface TaskState {
  // [NOTE] This is only used by the task list view. We don't
  // update the entry here if any task is changed (the updated task only gets updated in taskById).
  // Instead, we always fetch the list every time we display the task list view.
  taskListByUser: Map<UserId, Task[]>;
  taskById: Map<TaskId, Task>;
}

export interface ProjectState {
  // [NOTE] This is only used by the project list view. We don't
  // update the entry here if any project is changed (the updated project only gets updated in projectById).
  // Instead, we always fetch the list every time we display the project list view.
  projectListByUser: Map<UserId, Project[]>;
  projectById: Map<ProjectId, Project>;
}

export interface EnvironmentState {
  environmentList: Environment[];
}

export interface InstanceState {
  instanceById: Map<InstanceId, Instance>;
}

export interface DataSourceState {
  dataSourceById: Map<DataSourceId, DataSource>;
}

export interface DatabaseState {
  // UI may fetch the database list from different dimension (by user, by environment).
  // In those cases, we will iterate through this map and compute the list on the fly.
  // By keeping a single map, we avoid caching inconsistency issue.
  // We save it by instance because database belongs to instance and saving this way
  // follows that hierarchy.
  // If this causes performance issue, we will add caching later (and deal with the consistency)
  databaseListByInstanceId: Map<InstanceId, Database[]>;
}

export interface NotificationState {
  notificationByModule: Map<string, Notification[]>;
}

export interface CommandState {
  commandListById: Map<CommandId, Command[]>;
}
