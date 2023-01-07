import { RowStatus } from "./common";
import { EnvironmentId } from "./id";
import { EnvironmentTier } from "./policy";

export type Environment = {
  id: EnvironmentId;
  resourceId: string;

  // Standard fields
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
