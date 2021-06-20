import { RowStatus } from "./common";
import { PrincipalId, EnvironmentId } from "./id";
import { Principal } from "./principal";

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

export type EnvironmentCreate = {
  // Domain specific fields
  name: string;
};

export type EnvironmentPatch = {
  // Standard fields
  rowStatus?: RowStatus;

  // Domain specific fields
  name?: string;
  order?: number;
};
