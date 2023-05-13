import { RowStatus } from "./common";
import { EnvironmentId } from "./id";
import { EnvironmentTier as EnvironmentTierV1 } from "@/types/proto/v1/environment_service";

export type Environment = {
  id: EnvironmentId;
  resourceId: string;

  // Standard fields
  rowStatus: RowStatus;

  // Domain specific fields
  name: string;
  order: number;
  tier: EnvironmentTierV1;
};

export type EnvironmentCreate = {
  name: string;
  // Domain specific fields
  title: string;
};

export type EnvironmentPatch = {
  // Standard fields
  rowStatus?: RowStatus;

  // Domain specific fields
  title?: string;
  order?: number;
  tier?: EnvironmentTierV1;
};
