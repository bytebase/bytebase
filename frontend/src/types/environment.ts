import { RowStatus } from "./common";
import { EnvironmentId } from "./id";
import { EnvironmentTier } from "./policy";
import { Principal } from "./principal";

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
  tier: EnvironmentTier;
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
