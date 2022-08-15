import {
  RowStatus,
  Environment,
  IssueType,
  PolicyId,
  Principal,
  RuleType,
  RuleLevel,
  SubsetOf,
} from ".";

export type PolicyType =
  | "bb.policy.pipeline-approval"
  | "bb.policy.backup-plan"
  | "bb.policy.sql-review";

export type PipelineApprovalPolicyValue =
  | "MANUAL_APPROVAL_NEVER"
  | "MANUAL_APPROVAL_ALWAYS";

export type PipelineApprovalPolicyPayload = {
  value: PipelineApprovalPolicyValue;
  assigneeGroupList: AssigneeGroup[];
};

export const DefaultApprovalPolicy: PipelineApprovalPolicyValue =
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

export type AssigneeGroupValue = "WORKSPACE_OWNER_OR_DBA" | "PROJECT_OWNER";

export const DefaultAssigneeGroup: AssigneeGroupValue =
  "WORKSPACE_OWNER_OR_DBA";

export type AssigneeGroup = {
  issueType: SubsetOf<
    IssueType,
    | "bb.issue.database.schema.update"
    | "bb.issue.database.data.update"
    | "bb.issue.database.schema.update.ghost"
  >;
  value: AssigneeGroupValue;
};

export type PolicyPayload =
  | PipelineApprovalPolicyPayload
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
