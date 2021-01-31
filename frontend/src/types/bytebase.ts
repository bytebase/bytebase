import { ResourceObject } from "./jsonapi";

// These ID format may change in the future, so we encapsulate with a type.
// Also good for readability.
export type UserId = string;

export type PipelineId = string;

export type GroupId = string;

export type ProjectId = string;

export type EnvironmentId = string;

// Models
export type User = ResourceObject & {};
export type NewUser = Omit<ResourceObject, "id">;

export type Bookmark = ResourceObject & {};
export type NewBookmark = Omit<ResourceObject, "id">;

export type Activity = ResourceObject & {};
export type NewActivity = Omit<ResourceObject, "id">;

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
export type NewPipeline = Omit<ResourceObject, "id">;

export type Environment = ResourceObject & {
  attributes: {
    name: string;
    order: number;
    host: string;
    port: string;
    username: string;
    password: string;
    database: string;
  };
};
export type NewEnvironment = Omit<ResourceObject, "id">;

export type Group = ResourceObject & {};
export type NewGroup = Omit<ResourceObject, "id">;

export type Project = ResourceObject & {};
export type NewProject = Omit<ResourceObject, "id">;

export type Repository = ResourceObject & {};
export type NewRepository = Omit<ResourceObject, "id">;

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

export interface RepositoryState {
  repositoryByProject: Map<ProjectId, Repository>;
}

// UI
export interface RouterSlug {
  pipelineId?: PipelineId;
}
