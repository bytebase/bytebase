// Pipeline, Stage, Task are the backbones of execution.
//
// A PIPELINE consists of multiple STAGES. A STAGE consists of multiple TASKS.
//
// Comparison with Tekton
// PIPELINE = Tekton Pipeline
// STAGE = N/A
// TASK = Tekton Task
// N/A = Tekton Step
//
// Comparison with GitLab:
// PIPELINE = GitLab Pipeline
// STAGE = GitLab Stage
// TASK = GitLab Job
// N/A = GitLab Script Step
//
// Comparison with GitHub:
// PIPELINE = GitHub Workflow
// STAGE = N/A
// TASK = GitHub Job
// N/A = GitHub Step
//
// Comparison with Octopus:
// PIPELINE = Octopus Lifecycle
// STAGE = Octopus Phase + Task
// TASK = Octopus Step
//
// Comparsion with Jenkins:
// PIPELINE = Jenkins Pipeline
// STAGE = Jenkins Stage
// TASK = Jenkins Step (but it's also called task from its doc)
//
// Comparsion with Spinnaker:
// PIPELINE = Spinnaker Pipeline
// STAGE = Spinnaker Stage
// TASK = Spinnaker Task
//
//
// Design consideration
//
// - Other mainstream products either have 3 or 4 layers.
//   We choose 3 layers omitting the most granular layer - Step. For now only GitLab has 4 layer systems
//   and its step is mostly used to model a lightweight step like shell script step. This seems like
//   an overkill for our case. BTW, Octopus employes 3 layer design which seems to be sufficient.
//
// - All products agree on the smallest querable execution unit (having a dedicated API resource endpoint):
//   Tekton Task/GitLab Job/GitHub Job/Octopus Step.
//   Thus, we also choose Task as our smallest querable execution unit.
//
// - We also have a Stage concept which is similar to GitLab Stage/Octopus Phase, in that it's a
//   container to group mulitple tasks. Stage is usually used to model a stage in the development
//   lifecycle (dev, testing, staging, prod).
//
// - Only Pipeline and Task have status, while Stage doesn't. Stage's status derives from its
//   containing Tasks.
//
// - Pipeline status is 1-to-1 mapping to the Issue status. We introduce Pipepline for decoupling
//   pipeline logic (workflow orchestration etc) from issue logic (collabration etc). And it
//   helps testing (we can mock the entire pipeline implemenation) and also allows Pipeline to
//   be reused in other situation. On the other hand, we want to reduce the complexity of
//   introducing this extra layer, thus we always try to make a fixed 1-to-1 mapping for their
//   respective fields. Client code could combine Pipeline status and its running step status (substatus)
//   to achieve more granular behavior.
//
// So we finally arrive the same conclusion as spinnaker
//
// We require a stage to associate with a database. Since database belongs to an instance, which
// in turns belongs to an environment, thus the stage is also associated with an instance and environment.
// The environment has tiers which defines rules like whether requires manual approval.

import { Database } from "./database";
import { Environment } from "./environment";
import {
  DatabaseId,
  EnvironmentId,
  PipelineId,
  PrincipalId,
  StageId,
  TaskId,
  TaskRunId,
} from "./id";
import { Principal } from "./principal";

/*
 An example
 
 An alter schema PIPELINE
  Dev STAGE (db_dev, env_dev)
    Change dev database schema
  
  Testing STAGE (db_test, env_test)
    Change testing database schema
    Verify integration test pass

  Staging STAGE (db_staging, env_staging)
    Approve change
    Change staging database schema

  Prod STAGE (db_prod, env_prod)
    Approve change
    Change prod database schema
*/
// Pipeline
export type PipelineStatus = "OPEN" | "DONE" | "CANCELED";

export type Pipeline = {
  id: PipelineId;

  // Related fields
  stageList: Stage[];

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  status: PipelineStatus;
};

export type PipelineCreate = {
  // Related fields
  stageList: StageCreate[];

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

// Stage
export type StageType =
  | "bb.stage.general"
  | "bb.stage.database.create"
  | "bb.stage.database.grant"
  | "bb.stage.schema.update";

export type StageRunnable = {
  auto: boolean;
  run: () => void;
};

// The database belongs to an instance which in turns belongs to an environment.
// THus stage can access both instance and environment info.
export type Stage = {
  id: StageId;

  // Related fields
  taskList: Task[];
  pipeline: Pipeline;
  environment: Environment;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  type: StageType;
  runnable?: StageRunnable;
};

export type StageCreate = {
  // Related fields
  taskList: TaskCreate[];
  environmentId: EnvironmentId;

  // Domain specific fields
  name: string;
  type: StageType;
};

// Task
export type TaskType = "bb.task.general" | "bb.task.database.schema.update";

export type TaskStatus =
  | "PENDING"
  | "PENDING_APPROVAL"
  | "RUNNING"
  | "DONE"
  | "FAILED"
  | "CANCELED"
  | "SKIPPED";

export type DatabaseSchemaUpdateTaskPayload = {
  sql: string;
  rollbackSql: string;
};

export type TaskPayload = DatabaseSchemaUpdateTaskPayload;

export type Task = {
  id: TaskId;

  // Related fields
  taskRunList: TaskRun[];
  pipeline: Pipeline;
  stage: Stage;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  status: TaskStatus;
  type: TaskType;
  // We may get an empty database for tasks like creating database.
  database: Database;
  payload?: TaskPayload;
};

export type TaskCreate = {
  // Domain specific fields
  name: string;
  status: TaskStatus;
  type: TaskType;
  databaseId?: DatabaseId;
};

export type TaskStatusPatch = {
  // Standard fields
  updaterId: PrincipalId;

  // Domain specific fields
  status: TaskStatus;
  comment?: string;
};

export type TaskApprove = {
  // Standard fields
  updaterId: PrincipalId;

  // Domain specific fields
  comment?: string;
};

// TaskRun is one run of a particular task
export type TaskRunStatus =
  | "PENDING"
  | "RUNNING"
  | "DONE"
  | "FAILED"
  | "CANCELED";

export type TaskRun = {
  id: TaskRunId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  status: TaskRunStatus;
  type: TaskType;
  payload?: TaskPayload;
};
