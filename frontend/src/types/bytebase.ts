import { ResourceObject } from "./jsonapi";
import { BBNotificationStyle } from "../bbkit/types";
import { FieldId } from "../plugins";

// The project to hold those databases synced from the instance but haven't been assigned an application
// project yet. We can't use UNKNOWN_ID because of referential integrity.
export const DEFAULT_PROJECT_ID = "1";

export const ALL_DATABASE_NAME = "*";

// UNKNOWN_ID means an anomaly, it expects a resource which is missing (e.g. Keyed lookup missing).
export const UNKNOWN_ID = "-1";
// EMPTY_ID means an expected behavior, it expects no resource (e.g. contains an empty value, using this technic enables
// us to declare variable as required, which leads to cleaner code)
export const EMPTY_ID = "0";

export type ResourceType =
  | "PRINCIPAL"
  | "EXECUTION"
  | "USER"
  | "MEMBER"
  | "ENVIRONMENT"
  | "PROJECT"
  | "PROJECT_MEMBER"
  | "INSTANCE"
  | "DATABASE"
  | "DATA_SOURCE"
  | "ISSUE"
  | "PIPELINE"
  | "TASK"
  | "STEP"
  | "ACTIVITY"
  | "MESSAGE"
  | "BOOKMARK";

// unknown represents an anomaly.
// Returns as function to avoid caller accidentally mutate it.
export const unknown = (
  type: ResourceType
):
  | Execution
  | Principal
  | User
  | Member
  | Environment
  | Project
  | ProjectMember
  | Instance
  | Database
  | DataSource
  | Issue
  | Pipeline
  | Task
  | Step
  | Activity
  | Message
  | Bookmark => {
  const UNKNOWN_EXECUTION: Execution = {
    id: UNKNOWN_ID,
    status: "PENDING",
  };

  // Have to omit creator and updater to avoid recursion.
  const UNKNOWN_PRINCIPAL: Principal = {
    id: UNKNOWN_ID,
    createdTs: 0,
    updatedTs: 0,
    status: "UNKNOWN",
    name: "<<Unknown principal>>",
    email: "",
    role: "GUEST",
  } as Principal;

  const UNKNOWN_USER: User = {
    id: UNKNOWN_ID,
    createdTs: 0,
    updatedTs: 0,
    status: "UNKNOWN",
    name: "<<Unknown user>>",
    email: "unknown@example.com",
  };

  const UNKNOWN_MEMBER: Member = {
    id: UNKNOWN_ID,
    creator: UNKNOWN_PRINCIPAL,
    updater: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    role: "GUEST",
    principalId: UNKNOWN_ID,
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
  };

  const UNKNOWN_INSTANCE: Instance = {
    id: UNKNOWN_ID,
    rowStatus: "NORMAL",
    environment: UNKNOWN_ENVIRONMENT,
    creator: UNKNOWN_PRINCIPAL,
    updater: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    name: "<<Unknown instance>>",
    host: "",
  };

  const UNKNOWN_DATABASE: Database = {
    id: UNKNOWN_ID,
    instance: UNKNOWN_INSTANCE,
    project: UNKNOWN_PROJECT,
    dataSourceList: [],
    creator: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updater: UNKNOWN_PRINCIPAL,
    updatedTs: 0,
    name: "<<Unknown database>>",
    syncStatus: "NOT_FOUND",
    lastSuccessfulSyncTs: 0,
    fingerprint: "",
  };

  const UNKNOWN_DATA_SOURCE: DataSource = {
    id: UNKNOWN_ID,
    instance: UNKNOWN_INSTANCE,
    database: UNKNOWN_DATABASE,
    creator: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updater: UNKNOWN_PRINCIPAL,
    updatedTs: 0,
    memberList: [],
    name: "<<Unknown data source>>",
    type: "RO",
  };

  const UNKNOWN_PIPELINE: Pipeline = {
    id: UNKNOWN_ID,
    creator: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updater: UNKNOWN_PRINCIPAL,
    updatedTs: 0,
    name: "<<Unknown pipeline>>",
    status: "DONE",
    taskList: [],
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
    type: "bytebase.general",
    description: "",
    subscriberList: [],
    payload: {},
  };

  const UNKNOWN_TASK: Task = {
    id: UNKNOWN_ID,
    pipeline: UNKNOWN_PIPELINE,
    creator: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updater: UNKNOWN_PRINCIPAL,
    updatedTs: 0,
    name: "<<Unknown task>>",
    type: "bytebase.task.unknown",
    status: "DONE",
    environment: UNKNOWN_ENVIRONMENT,
    database: UNKNOWN_DATABASE,
    stepList: [],
  };

  const UNKNOWN_STEP: Step = {
    id: UNKNOWN_ID,
    pipeline: UNKNOWN_PIPELINE,
    task: UNKNOWN_TASK,
    creator: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updater: UNKNOWN_PRINCIPAL,
    updatedTs: 0,
    name: "<<Unknown step>>",
    type: "bytebase.step.unknown",
    status: "DONE",
  };

  const UNKNOWN_ACTIVITY: Activity = {
    id: UNKNOWN_ID,
    containerId: UNKNOWN_ID,
    createdTs: 0,
    updatedTs: 0,
    actionType: "bytebase.issue.create",
    creator: UNKNOWN_PRINCIPAL,
    updater: UNKNOWN_PRINCIPAL,
    comment: "<<Unknown comment>>",
  };

  const UNKNOWN_MESSAGE: Message = {
    id: UNKNOWN_ID,
    containerId: UNKNOWN_ID,
    creator: UNKNOWN_PRINCIPAL,
    createdTs: 0,
    updater: UNKNOWN_PRINCIPAL,
    updatedTs: 0,
    type: "bb.msg.issue.assign",
    status: "CONSUMED",
    description: "",
    receiver: UNKNOWN_PRINCIPAL,
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

  switch (type) {
    case "EXECUTION":
      return UNKNOWN_EXECUTION;
    case "PRINCIPAL":
      return UNKNOWN_PRINCIPAL;
    case "USER":
      return UNKNOWN_USER;
    case "MEMBER":
      return UNKNOWN_MEMBER;
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
    case "ISSUE":
      return UNKNOWN_ISSUE;
    case "PIPELINE":
      return UNKNOWN_PIPELINE;
    case "TASK":
      return UNKNOWN_TASK;
    case "STEP":
      return UNKNOWN_STEP;
    case "ACTIVITY":
      return UNKNOWN_ACTIVITY;
    case "MESSAGE":
      return UNKNOWN_MESSAGE;
    case "BOOKMARK":
      return UNKNOWN_BOOKMARK;
  }
};

// empty represents an expected behavior.
export const empty = (
  type: ResourceType
):
  | Execution
  | Principal
  | User
  | Member
  | Environment
  | Project
  | ProjectMember
  | Instance
  | Database
  | DataSource
  | Issue
  | Pipeline
  | Task
  | Step
  | Activity
  | Message
  | Bookmark => {
  const EMPTY_EXECUTION: Execution = {
    id: EMPTY_ID,
    status: "PENDING",
  };

  // Have to omit creator and updater to avoid recursion.
  const EMPTY_PRINCIPAL: Principal = {
    id: EMPTY_ID,
    createdTs: 0,
    updatedTs: 0,
    status: "UNKNOWN",
    name: "",
    email: "",
    role: "GUEST",
  } as Principal;

  const EMPTY_USER: User = {
    id: EMPTY_ID,
    createdTs: 0,
    updatedTs: 0,
    status: "UNKNOWN",
    name: "",
    email: "",
  };

  const EMPTY_MEMBER: Member = {
    id: EMPTY_ID,
    creator: EMPTY_PRINCIPAL,
    updater: EMPTY_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    role: "GUEST",
    principalId: EMPTY_ID,
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
  };

  const EMPTY_INSTANCE: Instance = {
    id: EMPTY_ID,
    rowStatus: "NORMAL",
    environment: EMPTY_ENVIRONMENT,
    creator: EMPTY_PRINCIPAL,
    updater: EMPTY_PRINCIPAL,
    createdTs: 0,
    updatedTs: 0,
    name: "",
    host: "",
  };

  const EMPTY_DATABASE: Database = {
    id: EMPTY_ID,
    instance: EMPTY_INSTANCE,
    project: EMPTY_PROJECT,
    dataSourceList: [],
    creator: EMPTY_PRINCIPAL,
    createdTs: 0,
    updater: EMPTY_PRINCIPAL,
    updatedTs: 0,
    name: "",
    syncStatus: "NOT_FOUND",
    lastSuccessfulSyncTs: 0,
    fingerprint: "",
  };

  const EMPTY_DATA_SOURCE: DataSource = {
    id: EMPTY_ID,
    instance: EMPTY_INSTANCE,
    database: EMPTY_DATABASE,
    creator: EMPTY_PRINCIPAL,
    createdTs: 0,
    updater: EMPTY_PRINCIPAL,
    updatedTs: 0,
    memberList: [],
    name: "",
    type: "RO",
  };

  const EMPTY_PIPELINE: Pipeline = {
    id: EMPTY_ID,
    creator: EMPTY_PRINCIPAL,
    createdTs: 0,
    updater: EMPTY_PRINCIPAL,
    updatedTs: 0,
    name: "",
    status: "DONE",
    taskList: [],
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
    type: "bytebase.general",
    description: "",
    subscriberList: [],
    payload: {},
  };

  const EMPTY_TASK: Task = {
    id: EMPTY_ID,
    pipeline: EMPTY_PIPELINE,
    creator: EMPTY_PRINCIPAL,
    createdTs: 0,
    updater: EMPTY_PRINCIPAL,
    updatedTs: 0,
    name: "",
    type: "bytebase.task.unknown",
    status: "DONE",
    environment: EMPTY_ENVIRONMENT,
    database: EMPTY_DATABASE,
    stepList: [],
  };

  const EMPTY_STEP: Step = {
    id: EMPTY_ID,
    pipeline: EMPTY_PIPELINE,
    task: EMPTY_TASK,
    creator: EMPTY_PRINCIPAL,
    createdTs: 0,
    updater: EMPTY_PRINCIPAL,
    updatedTs: 0,
    name: "",
    type: "bytebase.step.unknown",
    status: "DONE",
  };

  const EMPTY_ACTIVITY: Activity = {
    id: EMPTY_ID,
    containerId: EMPTY_ID,
    createdTs: 0,
    updatedTs: 0,
    actionType: "bytebase.issue.create",
    creator: EMPTY_PRINCIPAL,
    updater: EMPTY_PRINCIPAL,
    comment: "",
  };

  const EMPTY_MESSAGE: Message = {
    id: EMPTY_ID,
    containerId: EMPTY_ID,
    creator: EMPTY_PRINCIPAL,
    createdTs: 0,
    updater: EMPTY_PRINCIPAL,
    updatedTs: 0,
    type: "bb.msg.issue.assign",
    status: "DELIVERED",
    description: "",
    receiver: EMPTY_PRINCIPAL,
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

  switch (type) {
    case "EXECUTION":
      return EMPTY_EXECUTION;
    case "PRINCIPAL":
      return EMPTY_PRINCIPAL;
    case "USER":
      return EMPTY_USER;
    case "MEMBER":
      return EMPTY_MEMBER;
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
    case "ISSUE":
      return EMPTY_ISSUE;
    case "PIPELINE":
      return EMPTY_PIPELINE;
    case "TASK":
      return EMPTY_TASK;
    case "STEP":
      return EMPTY_STEP;
    case "ACTIVITY":
      return EMPTY_ACTIVITY;
    case "MESSAGE":
      return EMPTY_MESSAGE;
    case "BOOKMARK":
      return EMPTY_BOOKMARK;
  }
};

export const FINAL_TASK: Task = {
  id: EMPTY_ID,
  pipeline: empty("PIPELINE") as Pipeline,
  environment: empty("ENVIRONMENT") as Environment,
  database: empty("DATABASE") as Database,
  stepList: [],
  creator: empty("PRINCIPAL") as Principal,
  createdTs: 0,
  updater: empty("PRINCIPAL") as Principal,
  updatedTs: 0,
  name: "Final task",
  type: "bytebase.task.final",
  status: "PENDING",
};

export const FINAL_STEP: Step = {
  id: EMPTY_ID,
  pipeline: empty("PIPELINE") as Pipeline,
  task: FINAL_TASK,
  creator: empty("PRINCIPAL") as Principal,
  createdTs: 0,
  updater: empty("PRINCIPAL") as Principal,
  updatedTs: 0,
  name: "Final step",
  type: "bytebase.step.final",
  status: "PENDING",
};

// These ID format may change in the future, so we encapsulate with a type.
// Also good for readability.

export type ExecutionId = string;

export type UserId = string;

// For now, Principal is equal to UserId, in the future it may contain other id such as application, bot etc.
export type PrincipalId = UserId;

export type MemberId = string;

export type BookmarkId = string;

export type ProjectId = string;

export type IssueId = string;

export type PipelineId = string;

export type TaskId = string;

export type StepId = string;

export type ActivityId = string;

export type MessageId = string;

export type EnvironmentId = string;

export type InstanceId = string;

export type DataSourceId = string;

export type DatabaseId = string;

export type CommandId = string;
export type CommandRegisterId = string;

// This references to the object id, which can be used as a container.
// Currently, only issue can be used a container.
// The type is used by Activity and Message
export type ContainerId = IssueId;

export type BatchUpdate = {
  updaterId: PrincipalId;
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

// Member
export type RoleType = "OWNER" | "DBA" | "DEVELOPER" | "GUEST";

export type Member = {
  id: MemberId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  role: RoleType;
  principalId: PrincipalId;
};

export type MemberNew = {
  // Standard fields
  creatorId: PrincipalId;

  // Domain specific fields
  principalId: PrincipalId;
  role: RoleType;
};

export type MemberPatch = {
  // Standard fields
  updaterId: PrincipalId;

  // Domain specific fields
  role: RoleType;
};

// ProjectMember
export type ProjectRoleType = "OWNER" | "DEVELOPER";

export type ProjectMember = {
  id: MemberId;

  // Related fields
  project: Project;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  role: ProjectRoleType;
  principal: Principal;
};

export type ProjectMemberNew = {
  // Standard fields
  creatorId: PrincipalId;

  // Domain specific fields
  principalId: PrincipalId;
  role: ProjectRoleType;
};

export type ProjectMemberPatch = {
  // Standard fields
  updaterId: PrincipalId;

  // Domain specific fields
  role: ProjectRoleType;
};

// Principal
// This is a facet of the underlying identity entity.
// For now, there is only user type. In the future,
// we may support application/bot identity.
export type PrincipalStatus = "UNKNOWN" | "INVITED" | "ACTIVE";

export type Principal = {
  id: PrincipalId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  status: PrincipalStatus;
  name: string;
  email: string;
  role: RoleType;
};

export type PrincipalNew = {
  // Standard fields
  creatorId: PrincipalId;

  // Domain specific fields
  name: string;
  email: string;
};

export type PrincipalPatch = {
  // Standard fields
  updaterId: PrincipalId;
  rowStatus?: RowStatus;

  // Domain specific fields
  name?: string;
};

export type User = {
  id: UserId;

  // Standard fields
  // [TODO] User doesn't have updater, creator fields because of bootstrap issue.
  // Who is the updater, creator for the 1st user?
  createdTs: number;
  updatedTs: number;

  // Domain specific fields
  status: PrincipalStatus;
  name: string;
  email: string;
};

export type UserPatch = {
  // Standard fields
  updaterId: PrincipalId;

  // Domain specific fields
  name?: string;
};

// Bookmark
export type Bookmark = {
  id: BookmarkId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  link: string;
};

export type BookmarkNew = {
  // Standard fields
  creatorId: PrincipalId;

  // Domain specific fields
  name: string;
  link: string;
};

// Project
export type Project = {
  id: ProjectId;

  // Standard fields
  creator: Principal;
  updater: Principal;
  createdTs: number;
  updatedTs: number;
  rowStatus: RowStatus;

  // Domain specific fields
  name: string;
  key: string;
  // Returns the member list directly because we need it quite frequently in order
  // to do various access check.
  memberList: ProjectMember[];
};

export type ProjectNew = {
  // Standard fields
  creatorId: PrincipalId;

  // Domain specific fields
  name: string;
  key: string;
};

export type ProjectPatch = {
  // Standard fields
  updaterId: PrincipalId;
  rowStatus?: RowStatus;

  // Domain specific fields
  name?: string;
  key?: string;
};

// Issue
type IssueTypeGeneral = "bytebase.general";

type IssueTypeDatabase =
  | "bytebase.database.create"
  | "bytebase.database.grant"
  | "bytebase.database.schema.update";

type IssueTypeDataSource = "bytebase.data-source.request";

export type IssueType =
  | IssueTypeGeneral
  | IssueTypeDatabase
  | IssueTypeDataSource;

export type IssueStatus = "OPEN" | "DONE" | "CANCELED";

export type IssuePayload = { [key: string]: any };

export type Issue = {
  id: IssueId;

  // Related fields
  project: Project;
  pipeline: Pipeline;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  status: IssueStatus;
  type: IssueType;
  description: string;
  assignee?: Principal;
  subscriberList: Principal[];
  sql?: string;
  rollbackSql?: string;
  payload: IssuePayload;
};

export type IssueNew = {
  // Related fields
  projectId: ProjectId;
  pipeline?: PipelineNew;

  // Standard fields
  creatorId: PrincipalId;

  // Domain specific fields
  name: string;
  type: IssueType;
  description: string;
  assigneeId?: PrincipalId;
  sql?: string;
  rollbackSql?: string;
  payload: IssuePayload;
};

export type IssuePatch = {
  // Related fields
  projectId?: ProjectId;

  // Standard fields
  updaterId: PrincipalId;

  // Domain specific fields
  name?: string;
  description?: string;
  subscriberIdList?: PrincipalId[];
  sql?: string;
  rollbackSql?: string;
  assigneeId?: PrincipalId;
  payload?: IssuePayload;
};

export type IssueStatusPatch = {
  // Standard fields
  updaterId: PrincipalId;

  // Domain specific fields
  status: IssueStatus;
  comment?: string;
};

export type IssueStatusTransitionType = "NEXT" | "RESOLVE" | "ABORT" | "REOPEN";

export interface IssueStatusTransition {
  type: IssueStatusTransitionType;
  actionName: string;
  to: IssueStatus;
}

export const ISSUE_STATUS_TRANSITION_LIST: Map<
  IssueStatusTransitionType,
  IssueStatusTransition
> = new Map([
  [
    "NEXT",
    {
      type: "NEXT",
      actionName: "Next",
      to: "OPEN",
    },
  ],
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

// The first transition in the list is the primary action and the rests are
// the normal action. For now there are at most 1 primary 1 normal action.
export const CREATOR_APPLICABLE_ACTION_LIST: Map<
  IssueStatus,
  IssueStatusTransitionType[]
> = new Map([
  ["OPEN", ["ABORT"]],
  ["DONE", ["REOPEN"]],
  ["CANCELED", ["REOPEN"]],
]);

export const ASSIGNEE_APPLICABLE_ACTION_LIST: Map<
  IssueStatus,
  IssueStatusTransitionType[]
> = new Map([
  ["OPEN", ["NEXT", "RESOLVE", "ABORT"]],
  ["DONE", ["REOPEN"]],
  ["CANCELED", ["REOPEN"]],
]);

// Pipeline, Task, Step are the backbones of execution.
//
// A PIPELINE consists of multiple TASKS. A TASK consists of multiple STEPS.
//
// Comparison with Tekton
// PIPELINE = Tekton Pipeline
// TASK = Tekton Task
// STEP = Tekton Step
//
// Comparison with GitLab:
// PIPELINE = GitLab Pipeline
// TASK = GitLab Stage
// STEP = GitLab Job
//
// Comparison with Octopus:
// PIPELINE = Octopus Lifecycle
// TASK = Octopus Phase
// STEP = Octopus Step

// We require a task to associate with a database. Since database belongs to an instance, which
// in turns belongs to an environment, thus the task is also associated with an instance and environment.
// The environment has tiers which defines rules like whether requires manual approval.

/*
 An example
 
 An alter schema PIPELINE
  Dev TASK (db_dev, env_dev)
    Change dev database schema
  
  Testing TASK (db_test, env_test)
    Change testing database schema
    Verify integration test pass

  Staging TASK (db_staging, env_staging)
    Approve change
    Change staging database schema

  Prod TASK (db_prod, env_prod)
    Approve change
    Change prod database schema
*/
// Pipeline
export type PipelineStatus = "OPEN" | "DONE" | "CANCELED";

export type Pipeline = {
  id: PipelineId;

  // Related fields
  taskList: Task[];

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  status: PipelineStatus;
};

export type PipelineNew = {
  // Related fields
  taskList: TaskNew[];

  // Standard fields
  creatorId: PrincipalId;

  // Domain specific fields
  name: string;
};

export type PipelineStatusPatch = {
  // Standard fields
  updaterId: PrincipalId;

  // Domain specific fields
  status: PipelineStatus;
  comment?: string;
};

// Task
export type TaskType =
  | "bytebase.task.unknown"
  | "bytebase.task.final"
  | "bytebase.task.transition"
  | "bytebase.task.database.create"
  | "bytebase.task.database.grant"
  | "bytebase.task.schema.update";

export type TaskStatus = "PENDING" | "RUNNING" | "DONE" | "FAILED" | "SKIPPED";

export type TaskRunnable = {
  auto: boolean;
  run: () => void;
};

// The database belongs to an instance which in turns belongs to an environment.
// THus task can access both instance and environment info.
export type Task = {
  id: TaskId;

  // Related fields
  stepList: Step[];
  pipeline: Pipeline;
  environment: Environment;
  // We may get an empty database for tasks like creating database.
  database: Database;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  type: TaskType;
  status: TaskStatus;
  runnable?: TaskRunnable;
};

export type TaskNew = {
  // Related fields
  stepList: StepNew[];
  environmentId: EnvironmentId;
  databaseId: DatabaseId;

  // Domain specific fields
  name: string;
  type: TaskType;
};

export type TaskStatusPatch = {
  updaterId: PrincipalId;
  status: TaskStatus;
  comment?: string;
};

export type TaskStatusTransitionType = "RUN" | "RETRY" | "STOP" | "SKIP";

export interface TaskStatusTransition {
  type: TaskStatusTransitionType;
  actionName: string;
  requireRunnable: boolean;
  to: TaskStatus;
}

export const TASK_TRANSITION_LIST: Map<
  TaskStatusTransitionType,
  TaskStatusTransition
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

// Step
export type StepType =
  | "bytebase.step.unknown"
  | "bytebase.step.final"
  | "bytebase.step.resolve"
  | "bytebase.step.approve"
  | "bytebase.step.database.schema.update";

export type StepStatus = "PENDING" | "RUNNING" | "DONE" | "FAILED" | "SKIPPED";

export type DatabaseSchemaUpdateStepPayload = {
  sql: string;
  rollbackSql: string;
};

export type StepPayload = DatabaseSchemaUpdateStepPayload;

export type Step = {
  id: StepId;

  // Related fields
  pipeline: Pipeline;
  task: Task;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  type: StepType;
  status: StepStatus;
  payload?: StepPayload;
};

export type StepNew = {
  // Domain specific fields
  name: string;
  type: StepType;
};

export type StepPatch = {
  // Standard fields
  updaterId: PrincipalId;

  // Domain specific fields
  status: StepStatus;
};

export type StepStatusPatch = {
  updaterId: PrincipalId;
  status: StepStatus;
  comment?: string;
};

// Activity
export type IssueActionType =
  | "bytebase.issue.create"
  | "bytebase.issue.comment.create"
  | "bytebase.issue.field.update"
  | "bytebase.issue.status.update"
  | "bytebase.issue.task.status.update"
  | "bytebase.issue.task.step.status.update";

export type ActionType = IssueActionType;

export type ActionIssueFieldUpdatePayload = {
  changeList: {
    fieldId: FieldId;
    oldValue?: string;
    newValue?: string;
  }[];
};

export type ActionPayloadType = ActionIssueFieldUpdatePayload;

export type Activity = {
  id: ActivityId;

  // Related fields
  // The object where this activity belongs
  // e.g if actionType is "bytebase.issue.xxx", then this field refers to the corresponding issue's id.
  containerId: ContainerId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  actionType: ActionType;
  comment: string;
  payload?: ActionPayloadType;
};

export type ActivityNew = {
  // Related fields
  containerId: ContainerId;

  // Standard fields
  creatorId: PrincipalId;

  // Domain specific fields
  actionType: ActionType;
  comment: string;
  payload?: ActionPayloadType;
};

export type ActivityPatch = {
  // Standard fields
  updaterId: PrincipalId;

  // Domain specific fields
  comment: string;
};

// Message
export type MemberMessageType =
  | "bb.msg.member.create"
  | "bb.msg.member.invite"
  | "bb.msg.member.join"
  | "bb.msg.member.revoke"
  | "bb.msg.member.updaterole";

export type ProjectMemberMessageType =
  | "bb.msg.project.member.create"
  | "bb.msg.project.member.revoke"
  | "bb.msg.project.member.updaterole";

export type EnvironmentMessageType =
  | "bb.msg.environment.create"
  | "bb.msg.environment.update"
  | "bb.msg.environment.delete"
  | "bb.msg.environment.archive"
  | "bb.msg.environment.restore"
  | "bb.msg.environment.reorder";

export type InstanceMessageType =
  | "bb.msg.instance.create"
  | "bb.msg.instance.update"
  | "bb.msg.instance.delete"
  | "bb.msg.instance.archive"
  | "bb.msg.instance.restore";

export type IssueMessageType =
  | "bb.msg.issue.assign"
  | "bb.msg.issue.status.update"
  | "bb.msg.issue.task.status.update"
  | "bb.msg.issue.comment";

export type MessageType =
  | MemberMessageType
  | EnvironmentMessageType
  | InstanceMessageType
  | IssueMessageType;

export type MemberMessagePayload = {
  principalId: PrincipalId;
  oldRole?: RoleType;
  newRole?: RoleType;
};

export type ProjectMemberMessagePayload = {
  principalId: PrincipalId;
  oldRole?: ProjectRoleType;
  newRole?: ProjectRoleType;
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
  username = "7",
  password = "8",
}

export type InstanceMessagePaylaod = {
  instanceName: string;
};

export type IssueAssignMessagePayload = {
  issueName: string;
  oldAssigneeId: PrincipalId;
  newAssigneeId: PrincipalId;
};

export type IssueUpdateStatusMessagePayload = {
  issueName: string;
  oldStatus: IssueStatus;
  newStatus: IssueStatus;
};

export type IssueCommentMessagePayload = {
  issueName: string;
  commentId: ActivityId;
};

export type MessagePayload =
  | MemberMessagePayload
  | EnvironmentMessagePayload
  | EnvironmentUpdateMessagePayload
  | InstanceMessagePaylaod
  | IssueAssignMessagePayload
  | IssueUpdateStatusMessagePayload
  | IssueCommentMessagePayload;

export type MessageStatus = "DELIVERED" | "CONSUMED";

export type Message = {
  id: MessageId;

  // Related fields
  // The object where this message originates, simliar to containerId in Activity
  containerId: ContainerId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  type: MessageType;
  status: MessageStatus;
  description: string;
  receiver: Principal;
  payload?: MessagePayload;
};
export type MessageNew = Omit<Message, "id" | "createdTs" | "updatedTs">;

export type MessagePatch = {
  updaterId: PrincipalId;
  status: MessageStatus;
};

// Environment
// TODO: Introduce an environment tier to explicitly define which environment is prod/staging/test etc
export type Environment = {
  id: EnvironmentId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;
  rowStatus: RowStatus;

  // Domain specific fields
  name: string;
  order: number;
};

export type EnvironmentNew = {
  // Standard fields
  creatorId: PrincipalId;

  // Domain specific fields
  name: string;
};

export type EnvironmentPatch = {
  updaterId: PrincipalId;
  rowStatus?: RowStatus;

  // Domain specific fields
  name?: string;
};

// Instance
export type Instance = {
  id: InstanceId;

  // Related fields
  environment: Environment;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;
  rowStatus: RowStatus;

  // Domain specific fields
  name: string;
  externalLink?: string;
  host: string;
  port?: string;
  // In mysql, username can be empty which means anonymous user
  username?: string;
  password?: string;
};

export type InstanceNew = {
  // Related fields
  environmentId: EnvironmentId;

  // Standard fields
  creatorId: PrincipalId;

  // Domain specific fields
  name: string;
  externalLink?: string;
  host: string;
  port?: string;
  // In mysql, username can be empty which means anonymous user
  username?: string;
  password?: string;
};

export type InstancePatch = {
  // Standard fields
  updaterId: PrincipalId;
  rowStatus?: RowStatus;

  // Domain specific fields
  name?: string;
  externalLink?: string;
  host?: string;
  port?: string;
  username?: string;
  password?: string;
};

// Database

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

  // Related fields
  instance: Instance;
  project: Project;
  dataSourceList: DataSource[];

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  syncStatus: DatabaseSyncStatus;
  lastSuccessfulSyncTs: number;
  fingerprint: string;
};

export type DatabaseNew = {
  // Related fields
  instanceId: InstanceId;
  projectId: ProjectId;

  // Standard fields
  creatorId: PrincipalId;

  // Domain specific fields
  name: string;
  issueId?: IssueId;
};

export type DatabasePatch = {
  // Related fields
  projectId: ProjectId;

  // Standard fields
  updaterId: PrincipalId;
};

// Data Source

// For now the ADMIN requires the same database privilege as RW.
// The seperation is to make it explicit which one serves as the ADMIN data source,
// which from the ops perspective, having different meaning from the normal RW data source.
export type DataSourceType = "ADMIN" | "RW" | "RO";

export type DataSource = {
  id: DataSourceId;

  // Related fields
  database: Database;
  instance: Instance;
  // Returns the member list directly because we need it quite frequently in order
  // to do various access check.
  memberList: DataSourceMember[];

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  type: DataSourceType;
  // In mysql, username can be empty which means anonymous user
  username?: string;
  password?: string;
};

export type DataSourceNew = {
  // Related fields
  databaseId: DatabaseId;
  instanceId: InstanceId;
  memberList: DataSourceMemberNew[];

  // Standard fields
  creatorId: PrincipalId;

  // Domain specific fields
  name: string;
  type: DataSourceType;
  username?: string;
  password?: string;
};

export type DataSourcePatch = {
  // Standard fields
  updaterId: PrincipalId;

  // Domain specific fields
  name?: string;
  username?: string;
  password?: string;
};

export type DataSourceMember = {
  // Standard fields
  createdTs: number;

  // Domain specific fields
  principal: Principal;
  issueId?: IssueId;
};

export type DataSourceMemberNew = {
  // Domain specific fields
  principalId: PrincipalId;
  issueId?: IssueId;
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
  // Support Owner and DBA role at the workspace level
  | "bytebase.admin"
  // Support DBA workflow, including
  // 1. Developers can't create database directly, they need to do this via a request db issue.
  // 2. Allow developers to submit troubleshooting ticket.
  | "bytebase.dba-workflow"
  // Support defining extra data source for a database and exposing the related data source UI.
  | "bytebase.data-source";

export enum PlanType {
  FREE = 0,
  TEAM = 1,
  ENTERPRISE = 2,
}

// UI State Models
export type RouterSlug = {
  environmentSlug?: string;
  projectSlug?: string;
  issueSlug?: string;
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

export interface MemberState {
  memberList: Member[];
}

export interface PrincipalState {
  principalList: Principal[];
}

export interface BookmarkState {
  bookmarkListByUser: Map<UserId, Bookmark[]>;
}

export interface ActivityState {
  activityListByUser: Map<UserId, Activity[]>;
  activityListByIssue: Map<IssueId, Activity[]>;
}

export interface MessageState {
  messageListByUser: Map<UserId, Message[]>;
}

export interface IssueState {
  // [NOTE] This is only used by the issue list view. We don't
  // update the entry here if any issue is changed (the updated issue only gets updated in issueById).
  // Instead, we always fetch the list every time we display the issue list view.
  issueListByUser: Map<UserId, Issue[]>;
  issueById: Map<IssueId, Issue>;
}

export interface PipelineState {}

export interface TaskState {}

export interface StepState {}

export interface ProjectState {
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
