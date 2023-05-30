import { RowStatus } from "./common";
import { EnvironmentId } from "./id";

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
