import { ResourceObject } from "./jsonapi";
import { BBNotificationStyle } from "../bbkit/types";
import { TaskFieldId } from "../plugins";

export const UNKNOWN_ID = "-1";

// These ID format may change in the future, so we encapsulate with a type.
// Also good for readability.
export type UserId = string;

// For now, Principal is equal to UserId, in the future it may contain other id such as application, bot etc.
export type PrincipalId = UserId;

export type RoleMappingId = string;

export type BookmarkId = string;

export type StageId = string;

export type TaskId = string;

export type ActivityId = string;

export type EnvironmentId = string;

export type InstanceId = string;

export type DataSourceId = string;

export type DataSourceMemberId = string;

export type DatabaseId = string;
export const ALL_DATABASE_ID: DatabaseId = "-1";

export type CommandId = string;
export type CommandRegisterId = string;

// Persistent State Models
// User
export type UserStatus = "INVITED" | "ACTIVE";

export type User = {
  id: UserId;
  status: UserStatus;
  name: string;
  email: string;
};
export type NewUser = Omit<User, "id">;

// Principal
export type PrincipalStatus = "UNKNOWN" | "INVITED" | "ACTIVE";

export type Principal = {
  id: PrincipalId;
  status: PrincipalStatus;
  name: string;
  email: string;
};

export type PrincipalNew = {
  email: string;
};

export type PrincipalPatch = {
  id: PrincipalId;
  name?: string;
};

// RoleMapping
export type RoleType = "OWNER" | "DBA" | "DEVELOPER";

export type RoleMapping = {
  id: RoleMappingId;
  createdTs: number;
  lastUpdatedTs: number;
  role: RoleType;
  principal: Principal;
  updater: Principal;
};

export type RoleMappingNew = {
  principalId: PrincipalId;
  email: string;
  role: RoleType;
  updaterId: PrincipalId;
};

export type RoleMappingPatch = {
  id: RoleMappingId;
  role: RoleType;
};

// Bookmark
export type Bookmark = {
  id: BookmarkId;
  name: string;
  link: string;
  creatorId: UserId;
};
export type BookmarkNew = Omit<Bookmark, "id">;

// Stage
export type StageType = "SIMPLE" | "ENVIRONMENT";

export type StageStatus = "PENDING" | "RUNNING" | "DONE" | "FAILED" | "SKIPPED";

export type StageRunnable = {
  auto: boolean;
  run: () => void;
};

export type Stage = {
  id: StageId;
  name: string;
  type: StageType;
  environmentId?: EnvironmentId;
  runnable?: StageRunnable;
};

export type StageProgress = Stage & {
  status: StageStatus;
};

export type StageProgressPatch = {
  id: StageId;
  status: StageStatus;
};

// Task
type TaskTypeGeneral = "bytebase.general";

type TaskTypeDataSource =
  | "bytebase.datasource.request"
  | "bytebase.datasource.schema.update";

export type TaskType = TaskTypeGeneral | TaskTypeDataSource;

export type TaskStatus = "OPEN" | "DONE" | "CANCELED";

export type TaskPayload = { [key: string]: any };

export type Task = {
  id: TaskId;
  name: string;
  createdTs: number;
  lastUpdatedTs: number;
  status: TaskStatus;
  category: "DDL" | "DML" | "OPS";
  type: TaskType;
  description: string;
  stageProgressList: StageProgress[];
  creator: Principal;
  assignee?: Principal;
  subscriberIdList: Array<string>;
  payload: TaskPayload;
};

export type TaskNew = {
  name: string;
  type: TaskType;
  description: string;
  stageProgressList: StageProgress[];
  creatorId: PrincipalId;
  assigneeId?: PrincipalId;
  payload: TaskPayload;
};

export type TaskPatch = {
  name?: string;
  status?: TaskStatus;
  description?: string;
  assigneeId?: PrincipalId;
  stageProgressList?: StageProgressPatch[];
  payload?: TaskPayload;
};

// Activity
export type TaskActionType =
  | "bytebase.task.create"
  | "bytebase.task.comment.create"
  | "bytebase.task.field.update";

export type ActionType = TaskActionType;

export type ActionTaskCommentCreatePayload = {
  comment: string;
};

export type ActionTaskFieldUpdatePayload = {
  changeList: {
    fieldId: TaskFieldId;
    oldValue?: string;
    newValue?: string;
  }[];
};

export type ActionPayloadType =
  | ActionTaskCommentCreatePayload
  | ActionTaskFieldUpdatePayload;

export type Activity = {
  id: ActivityId;
  createdTs: number;
  lastUpdatedTs: number;
  actionType: ActionType;
  // The object where this activity belongs
  // e.g if actionType is "bytebase.task.xxx", then this field refers to the corresponding task's id.
  containerId: TaskId;
  creator: Principal;
  payload?: ActionPayloadType;
};
export type ActivityNew = Omit<Activity, "id">;

export type ActivityPatch = {
  payload: any;
};

// Environment
export type Environment = {
  id: EnvironmentId;
  name: string;
  order: number;
};
export type EnvironmentNew = Omit<Environment, "id">;

// Instance
export type Instance = {
  id: InstanceId;
  createdTs: number;
  lastUpdatedTs: number;
  name: string;
  environment: Environment;
  externalLink?: string;
  host: string;
  port?: string;
};
export type InstanceNew = Omit<Instance, "id">;

export type DataSourceType = "RW" | "RO";
// Data Source
export type DataSource = {
  id: DataSourceId;
  instanceId: InstanceId;
  name: string;
  createdTs: number;
  lastUpdatedTs: number;
  type: DataSourceType;
  // In mysql, username can be empty which means anonymous user
  username?: string;
  password?: string;
  // If empty, it means it can access all databases from an instance.
  // TODO: unlike other objects like environment, here we don't expand
  // to the database object, this is due to timing issue during rendering.
  // Unlike environment which is a global state that we can load upon
  // startup, the database info is per instance, and the author haven't
  // figured out a elegant way to guarantee it's loaded before the router
  // fetches the specific data source and requires the database info for
  // the conversion
  databaseId?: DatabaseId;
};

export type DataSourceNew = {
  name: string;
  type: DataSourceType;
  databaseId?: string;
  username?: string;
  password?: string;
};

export type DataSourceMember = {
  id: DataSourceMemberId;
  principal: Principal;
  taskId?: TaskId;
  createdTs: number;
};

// We periodically sync the underlying db schema and stores those info
// in the "database" object.

// "OK" means find the exact match
// "MISMATCH" means we find the database with the same name, but the fingerprint is different,
//            this usually indicates the underlying database has been recreated (might for a entirely different purpose)
// "NOT_FOUND" means no matching database name found, this ususally means someone changes
//            the underlying db name.
export type DatabaseSyncStatus = "OK" | "MISMATCH" | "NOT_FOUND";
// Database
export type Database = {
  id: DatabaseId;
  name: string;
  createdTs: number;
  lastUpdatedTs: number;
  syncStatus: DatabaseSyncStatus;
  fingerprint: string;
  instance: Instance;
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

// UI State Models
export type RouterSlug = {
  taskSlug?: string;
  instanceSlug?: string;
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
  manualHide?: boolean;
};

export type Command = {
  id: CommandId;
  registerId: CommandRegisterId;
  run: () => void;
};

// "id" and "createdTs" is auto generated upon the notification store
// receives.
export type NewNotification = Omit<Notification, "id" | "createdTs">;

export type NotificationFilter = {
  module: string;
  id?: string;
};

// Store
export interface AuthState {
  currentUser: User | undefined;
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

export interface TaskState {
  taskListByUser: Map<UserId, Task[]>;
  taskById: Map<TaskId, Task>;
}

export interface EnvironmentState {
  environmentList: Environment[];
}

export interface InstanceState {
  instanceById: Map<InstanceId, Instance>;
}

export interface DataSourceState {
  dataSourceListByInstanceId: Map<InstanceId, DataSource[]>;
  memberListById: Map<DataSourceId, DataSourceMember[]>;
}

export interface DatabaseState {
  databaseListByInstanceId: Map<InstanceId, Database[]>;
}

export interface NotificationState {
  notificationByModule: Map<string, Notification[]>;
}

export interface CommandState {
  commandListById: Map<CommandId, Command[]>;
}
