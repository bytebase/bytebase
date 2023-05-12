import { RowStatus, Environment, PolicyId } from ".";

export type PolicyType = "bb.policy.environment-tier";

export type EnvironmentTier = "PROTECTED" | "UNPROTECTED";

export type EnvironmentTierPolicyPayload = {
  environmentTier: EnvironmentTier;
};

export const DefaultEnvironmentTier: EnvironmentTier = "UNPROTECTED";

export type BackupPlanPolicySchedule = "UNSET" | "DAILY" | "WEEKLY";

export type PolicyPayload = EnvironmentTierPolicyPayload;

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
