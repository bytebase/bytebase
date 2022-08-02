import {
  RowStatus,
  Environment,
  PolicyId,
  Principal,
  RuleType,
  RuleLevel,
} from ".";

export type PolicyType =
  | "bb.policy.pipeline-approval"
  | "bb.policy.backup-plan"
  | "bb.policy.sql-review";

export type PipelineApprovalPolicyValue =
  | "MANUAL_APPROVAL_NEVER"
  | "MANUAL_APPROVAL_ALWAYS";

export type PipelineApporvalPolicyPayload = {
  value: PipelineApprovalPolicyValue;
};

export const DefaultApporvalPolicy: PipelineApprovalPolicyValue =
  "MANUAL_APPROVAL_ALWAYS";

export type BackupPlanPolicySchedule = "UNSET" | "DAILY" | "WEEKLY";

export type BackupPlanPolicyPayload = {
  schedule: BackupPlanPolicySchedule;
};

export const DefaultSchedulePolicy: BackupPlanPolicySchedule = "UNSET";

// SQLReviewPolicyPayload is the payload for SQL review policy in the backend.
export type SQLReviewPolicyPayload = {
  name: string;
  ruleList: {
    type: RuleType;
    level: RuleLevel;
    payload: string;
  }[];
};

export type PolicyPayload =
  | PipelineApporvalPolicyPayload
  | BackupPlanPolicyPayload
  | SQLReviewPolicyPayload;

export type Policy = {
  id: PolicyId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;
  rowStatus: RowStatus;

  // Related fields
  environment: Environment;

  // Domain specific fields
  type: PolicyType;
  payload: PolicyPayload;
};

export type PolicyUpsert = {
  // Standard fields
  rowStatus?: RowStatus;

  // Domain specific fields
  payload?: PolicyPayload;
};
