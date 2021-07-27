import { RowStatus } from "./common";
import { EnvironmentId } from "./id";
import { Principal } from "./principal";

// Approval policy
export type ApprovalPolicy = "MANUAL_APPROVAL_NEVER" | "MANUAL_APPROVAL_ALWAYS";

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
  approvalPolicy: ApprovalPolicy;
};

export type EnvironmentCreate = {
  // Domain specific fields
  name: string;
  approvalPolicy: ApprovalPolicy;
};

export type EnvironmentPatch = {
  // Standard fields
  rowStatus?: RowStatus;

  // Domain specific fields
  name?: string;
  order?: number;
  approvalPolicy?: ApprovalPolicy;
};
