import type { RowStatus } from "./common";
import type { EnvironmentId } from "./id";

export type Environment = {
  id: EnvironmentId;
  resourceId: string;

  // Standard fields
  rowStatus: RowStatus;

  // Domain specific fields
  name: string;
  order: number;
  tier: "PROTECTED" | "UNPROTECTED";
};
