import { ResourceObject } from "./jsonapi";
import { BBNotificationStyle } from "../bbkit/types";

// These ID format may change in the future, so we encapsulate with a type.
// Also good for readability.
export type UserId = string;

export type TaskId = string;

export type GroupId = string;

export type ProjectId = string;

export type EnvironmentId = string;

export type InstanceId = string;

export type DataSourceId = {
  id: string;
  instanceId: InstanceId;
};

export type CommandId = string;
export type CommandRegisterId = string;

// Persistent State Models
export type User = ResourceObject & {};
export type NewUser = Omit<User, "id">;

export type Bookmark = ResourceObject & {};
export type NewBookmark = Omit<Bookmark, "id">;

export type Activity = ResourceObject & {};
export type ActivityNew = Omit<Activity, "id">;

export type Task = ResourceObject & {
  attributes: {
    title: string;
    createdTs: number;
    lastUpdatedTs: number;
    status: "PENDING" | "RUNNING" | "DONE" | "FAILED" | "CANCELED";
    type: "DDL" | "DML" | "OPS";
    content: string;
    currentStageId: string;
    stageProgressList: {
      stageId: string;
      stageName: string;
      status:
        | "CREATED"
        | "PENDING"
        | "RUNNING"
        | "DONE"
        | "FAILED"
        | "CANCELED"
        | "SKIPPED";
    }[];
    creator: {
      id: string;
      name: string;
    };
    assignee: {
      id: string;
      name: string;
    };
    subscriberIdList: Array<string>;
  };
};
export type TaskNew = Omit<Task, "id">;

export type Environment = ResourceObject & {
  attributes: {
    name: string;
    order: number;
  };
};
export type EnvironmentNew = Omit<Environment, "id">;

export type Instance = ResourceObject & {
  attributes: {
    name: string;
    environmentId: string;
    externalLink?: string;
    host: string;
    port?: string;
  };
};
export type InstanceNew = Omit<Instance, "id">;

export type DataSource = ResourceObject & {
  attributes: {
    name: string;
    type: "ADMIN" | "NORMAL";
    // In mysql, username can be empty which means anonymous user
    username?: string;
    password?: string;
  };
};
export type DataSourceNew = Omit<DataSource, "id">;

export type Group = ResourceObject & {};
export type GroupNew = Omit<Group, "id">;

export type Project = ResourceObject & {};
export type ProjectNew = Omit<Project, "id">;

export type Repository = ResourceObject & {};
export type RepositoryNew = Omit<Repository, "id">;

export type LoginInfo = Omit<
  ResourceObject & {
    attributes: {
      username: string;
      password: string;
    };
  },
  "id"
>;

export type SignupInfo = Omit<
  ResourceObject & {
    attributes: {
      username: string;
      password: string;
    };
  },
  "id"
>;

// UI State Models
export type RouterSlug = {
  taskId?: TaskId;
  instanceSlug?: string;
};

export type Notification = {
  id: string;
  createdTs: number;
  module: string;
  style: BBNotificationStyle;
  title: string;
  description?: string;
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
};

// Store
export interface AuthState {
  currentUser: User | null;
}

export interface BookmarkState {
  bookmarkListByUser: Map<UserId, Bookmark[]>;
}

export interface ActivityState {
  activityListByUser: Map<UserId, Activity[]>;
}

export interface TaskState {
  taskListByUser: Map<UserId, Task[]>;
  taskById: Map<TaskId, Task>;
}

export interface GroupState {
  groupListByUser: Map<UserId, Group[]>;
}

export interface ProjectState {
  projectListByGroup: Map<GroupId, Project[]>;
  projectListByUser: Map<UserId, Project[]>;
}

export interface EnvironmentState {
  environmentList: Environment[];
}

export interface InstanceState {
  instanceList: Instance[];
  instanceById: Map<InstanceId, Instance>;
}

export interface DataSourceState {
  dataSourceListByInstanceId: Map<InstanceId, DataSource[]>;
  dataSourceById: Map<DataSourceId, DataSource>;
}

export interface RepositoryState {
  repositoryByProject: Map<ProjectId, Repository>;
}

export interface NotificationState {
  notificationByModule: Map<string, Notification[]>;
}

export interface CommandState {
  commandListById: Map<CommandId, Command[]>;
}
