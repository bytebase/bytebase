import { ResourceObject } from "./jsonapi";

// These ID format may change in the future, so we encapsulate with a type.
// Also good for readability.
export type UserId = string;

export type PipelineId = string;

export type GroupId = string;

export type ProjectId = string;

export type EnvironmentId = string;

export type InstanceId = string;

export type DataSourceId = {
  id: string;
  instanceId: InstanceId;
};

// Models
export type User = ResourceObject & {};
export type NewUser = Omit<User, "id">;

export type Bookmark = ResourceObject & {};
export type NewBookmark = Omit<Bookmark, "id">;

export type Activity = ResourceObject & {};
export type NewActivity = Omit<Activity, "id">;

export type Pipeline = ResourceObject & {
  attributes: {
    title: string;
    createdTs: number;
    lastUpdatedTs: number;
    status: "PENDING" | "RUNNING" | "DONE" | "FAILED" | "CANCELED";
    type: "DDL" | "DML" | "OPS";
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
export type NewPipeline = Omit<Pipeline, "id">;

export type Environment = ResourceObject & {
  attributes: {
    name: string;
    order: number;
  };
};
export type NewEnvironment = Omit<Environment, "id">;

export type Instance = ResourceObject & {
  attributes: {
    name: string;
    environmentId: string;
    externalLink?: string;
    host: string;
    port?: string;
  };
};
export type NewInstance = Omit<Instance, "id">;

export type DataSource = ResourceObject & {
  attributes: {
    name: string;
    type: "ADMIN" | "NORMAL";
    // In mysql, username can be empty which means anonymous user
    username?: string;
    password?: string;
  };
};
export type NewDataSource = Omit<DataSource, "id">;

export type Group = ResourceObject & {};
export type NewGroup = Omit<Group, "id">;

export type Project = ResourceObject & {};
export type NewProject = Omit<Project, "id">;

export type Repository = ResourceObject & {};
export type NewRepository = Omit<Repository, "id">;

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

export interface PipelineState {
  pipelineListByUser: Map<UserId, Pipeline[]>;
  pipelineById: Map<PipelineId, Pipeline>;
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

// UI
export interface RouterSlug {
  pipelineId?: PipelineId;
  instanceId?: InstanceId;
}
