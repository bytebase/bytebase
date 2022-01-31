import { Environment, PolicyId, Principal } from ".";

export type PolicyType =
  | "bb.policy.pipeline-approval"
  | "bb.policy.backup-plan";

export type PipelineApprovalPolicyValue =
  | "MANUAL_APPROVAL_NEVER"
  | "MANUAL_APPROVAL_ALWAYS";

export type PipelineApporvalPolicyPayload = {
  value: PipelineApprovalPolicyValue;
};

export const DefaultApporvalPolicy: PipelineApprovalPolicyValue =
  "MANUAL_APPROVAL_ALWAYS";

export type BackupPlanPolicySchedule = "UNSET" | "DAILY" | "WEEKLY";

export type PolicyBackupPlanPolicyPayload = {
  schedule: BackupPlanPolicySchedule;
};

export const DefaultSchedulePolicy: BackupPlanPolicySchedule = "UNSET";

export type PolicyPayload =
  | PipelineApporvalPolicyPayload
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
