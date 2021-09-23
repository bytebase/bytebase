import { Environment, PolicyId, Principal } from ".";

export type PolicyType =
  | "bb.policy.pipeline-approval"
  | "bb.policy.backup-plan";

export type PolicyApporvalPolicyPayload = {};

export type BackupPlanPolicySchedule = "UNSET" | "DAILY" | "WEEKLY";

export type PolicyBackupPlanPolicyPayload = {
  schedule: BackupPlanPolicySchedule;
};

export type PolicyPayload =
  | PolicyApporvalPolicyPayload
  | PolicyBackupPlanPolicyPayload;

export type Policy = {
  id: PolicyId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Related fields
  environment: Environment;

  // Domain specific fields
  type: PolicyType;
  payload: PolicyPayload;
};

export type PolicyUpsert = {
  // Domain specific fields
  payload?: PolicyPayload;
};
