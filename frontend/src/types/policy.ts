import { RowStatus, Environment, IssueType, PolicyId, SubsetOf } from ".";

export type PolicyType =
  | "bb.policy.pipeline-approval"
  | "bb.policy.backup-plan"
  | "bb.policy.environment-tier"
  | "bb.policy.access-control";

export type PipelineApprovalPolicyValue =
  | "MANUAL_APPROVAL_NEVER"
  | "MANUAL_APPROVAL_ALWAYS";

export type PipelineApprovalPolicyPayload = {
  value: PipelineApprovalPolicyValue;
  assigneeGroupList: AssigneeGroup[];
};

export const DefaultApprovalPolicy: PipelineApprovalPolicyValue =
  "MANUAL_APPROVAL_ALWAYS";

export type EnvironmentTier = "PROTECTED" | "UNPROTECTED";

export type EnvironmentTierPolicyPayload = {
  environmentTier: EnvironmentTier;
};

export const DefaultEnvironmentTier: EnvironmentTier = "UNPROTECTED";

export type BackupPlanPolicySchedule = "UNSET" | "DAILY" | "WEEKLY";

export type BackupPlanPolicyPayload = {
  schedule: BackupPlanPolicySchedule;
  retentionPeriodTs: number;
};

export const DefaultSchedulePolicy: BackupPlanPolicySchedule = "UNSET";

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

export type AccessControlRule = {
  fullDatabase: boolean;
};

export type AccessControlPolicyPayload = {
  disallowRuleList: AccessControlRule[];
};

export type PolicyPayload =
  | PipelineApprovalPolicyPayload
  | BackupPlanPolicyPayload
  | EnvironmentTierPolicyPayload
  | AccessControlPolicyPayload;

export type PolicyResourceType =
  | ""
  | "workspace"
  | "environment"
  | "project"
  | "instance"
  | "database";

export type Policy = {
  id: PolicyId;

  // Standard fields
  rowStatus: RowStatus;

  // Related fields
  resourceType: PolicyResourceType;
  resourceId: number;
  environment: Environment;

  // Domain specific fields
  inheritFromParent: boolean;
  type: PolicyType;
  payload: PolicyPayload;
};

export type PolicyUpsert = {
  // Standard fields
  rowStatus?: RowStatus;

  // Domain specific fields
  inheritFromParent?: boolean;
  payload?: PolicyPayload;
};
