// The database belongs to an instance which in turns belongs to an environment.
import { Environment } from "../environment";
import { EnvironmentId, StageId } from "../id";
import { Pipeline } from "./pipeline";
import { Task, TaskCreate, TaskStatus } from "./task";

// THus stage can access both instance and environment info.
export type Stage = {
  id: StageId;

  // Related fields
  taskList: Task[];
  pipeline: Pipeline;
  environment: Environment;

  // Domain specific fields
  name: string;
};

export type StageCreate = {
  // Related fields
  taskList: TaskCreate[];
  environmentId: EnvironmentId;

  // Domain specific fields
  name: string;
};

export type StageAllTaskStatusPatch = {
  id: StageId;

  status: TaskStatus;
  comment?: string;

  updatedTs: number;
};

export type StageStatusUpdateType = "BEGIN" | "END";
