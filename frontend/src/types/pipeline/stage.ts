// The database belongs to an instance which in turns belongs to an environment.

import { Environment } from "../environment";
import { EnvironmentID, StageID } from "../id";
import { Principal } from "../principal";
import { Pipeline } from "./pipeline";
import { Task, TaskCreate } from "./task";

// THus stage can access both instance and environment info.
export type Stage = {
  id: StageID;

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
};

export type StageCreate = {
  // Related fields
  taskList: TaskCreate[];
  environmentID: EnvironmentID;

  // Domain specific fields
  name: string;
};
