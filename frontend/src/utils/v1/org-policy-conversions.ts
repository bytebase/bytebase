import { fromJson, toJson } from "@bufbuild/protobuf";
import type { Policy as OldPolicy } from "@/types/proto/v1/org_policy_service";
import { Policy as OldPolicyProto } from "@/types/proto/v1/org_policy_service";
import type { Policy as NewPolicy } from "@/types/proto-es/v1/org_policy_service_pb";
import { PolicySchema } from "@/types/proto-es/v1/org_policy_service_pb";
import { PolicyResourceType as OldPolicyResourceType } from "@/types/proto/v1/org_policy_service";
import { PolicyResourceType as NewPolicyResourceType } from "@/types/proto-es/v1/org_policy_service_pb";
import { PolicyType as OldPolicyType } from "@/types/proto/v1/org_policy_service";
import { PolicyType as NewPolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import { SQLReviewRuleLevel as OldSQLReviewRuleLevel } from "@/types/proto/v1/org_policy_service";
import { SQLReviewRuleLevel as NewSQLReviewRuleLevel } from "@/types/proto-es/v1/org_policy_service_pb";

// Convert old proto to proto-es
export const convertOldPolicyToNew = (oldPolicy: OldPolicy): NewPolicy => {
  // Use toJSON to convert old proto to JSON, then fromJson to convert to proto-es
  const json = OldPolicyProto.toJSON(oldPolicy) as any; // Type assertion needed due to proto type incompatibility
  return fromJson(PolicySchema, json);
};

// Convert proto-es to old proto
export const convertNewPolicyToOld = (newPolicy: NewPolicy): OldPolicy => {
  // Use toJson to convert proto-es to JSON, then fromJSON to convert to old proto
  const json = toJson(PolicySchema, newPolicy);
  return OldPolicyProto.fromJSON(json);
};

// Convert old PolicyResourceType enum to new (string to numeric)
export const convertOldPolicyResourceTypeToNew = (oldType: OldPolicyResourceType): NewPolicyResourceType => {
  const mapping: Record<OldPolicyResourceType, NewPolicyResourceType> = {
    [OldPolicyResourceType.RESOURCE_TYPE_UNSPECIFIED]: NewPolicyResourceType.RESOURCE_TYPE_UNSPECIFIED,
    [OldPolicyResourceType.WORKSPACE]: NewPolicyResourceType.WORKSPACE,
    [OldPolicyResourceType.ENVIRONMENT]: NewPolicyResourceType.ENVIRONMENT,
    [OldPolicyResourceType.PROJECT]: NewPolicyResourceType.PROJECT,
    [OldPolicyResourceType.UNRECOGNIZED]: NewPolicyResourceType.RESOURCE_TYPE_UNSPECIFIED,
  };
  return mapping[oldType] ?? NewPolicyResourceType.RESOURCE_TYPE_UNSPECIFIED;
};

// Convert new PolicyResourceType enum to old (numeric to string)
export const convertNewPolicyResourceTypeToOld = (newType: NewPolicyResourceType): OldPolicyResourceType => {
  const mapping: Record<NewPolicyResourceType, OldPolicyResourceType> = {
    [NewPolicyResourceType.RESOURCE_TYPE_UNSPECIFIED]: OldPolicyResourceType.RESOURCE_TYPE_UNSPECIFIED,
    [NewPolicyResourceType.WORKSPACE]: OldPolicyResourceType.WORKSPACE,
    [NewPolicyResourceType.ENVIRONMENT]: OldPolicyResourceType.ENVIRONMENT,
    [NewPolicyResourceType.PROJECT]: OldPolicyResourceType.PROJECT,
  };
  return mapping[newType] ?? OldPolicyResourceType.UNRECOGNIZED;
};

// Convert old PolicyType enum to new (string to numeric)
export const convertOldPolicyTypeToNew = (oldType: OldPolicyType): NewPolicyType => {
  const mapping: Record<OldPolicyType, NewPolicyType> = {
    [OldPolicyType.POLICY_TYPE_UNSPECIFIED]: NewPolicyType.POLICY_TYPE_UNSPECIFIED,
    [OldPolicyType.ROLLOUT_POLICY]: NewPolicyType.ROLLOUT_POLICY,
    [OldPolicyType.DISABLE_COPY_DATA]: NewPolicyType.DISABLE_COPY_DATA,
    [OldPolicyType.MASKING_RULE]: NewPolicyType.MASKING_RULE,
    [OldPolicyType.MASKING_EXCEPTION]: NewPolicyType.MASKING_EXCEPTION,
    [OldPolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW]: NewPolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW,
    [OldPolicyType.TAG]: NewPolicyType.TAG,
    [OldPolicyType.DATA_SOURCE_QUERY]: NewPolicyType.DATA_SOURCE_QUERY,
    [OldPolicyType.DATA_EXPORT]: NewPolicyType.DATA_EXPORT,
    [OldPolicyType.DATA_QUERY]: NewPolicyType.DATA_QUERY,
    [OldPolicyType.UNRECOGNIZED]: NewPolicyType.POLICY_TYPE_UNSPECIFIED,
  };
  return mapping[oldType] ?? NewPolicyType.POLICY_TYPE_UNSPECIFIED;
};

// Convert new PolicyType enum to old (numeric to string)
export const convertNewPolicyTypeToOld = (newType: NewPolicyType): OldPolicyType => {
  const mapping: Record<NewPolicyType, OldPolicyType> = {
    [NewPolicyType.POLICY_TYPE_UNSPECIFIED]: OldPolicyType.POLICY_TYPE_UNSPECIFIED,
    [NewPolicyType.ROLLOUT_POLICY]: OldPolicyType.ROLLOUT_POLICY,
    [NewPolicyType.DISABLE_COPY_DATA]: OldPolicyType.DISABLE_COPY_DATA,
    [NewPolicyType.MASKING_RULE]: OldPolicyType.MASKING_RULE,
    [NewPolicyType.MASKING_EXCEPTION]: OldPolicyType.MASKING_EXCEPTION,
    [NewPolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW]: OldPolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW,
    [NewPolicyType.TAG]: OldPolicyType.TAG,
    [NewPolicyType.DATA_SOURCE_QUERY]: OldPolicyType.DATA_SOURCE_QUERY,
    [NewPolicyType.DATA_EXPORT]: OldPolicyType.DATA_EXPORT,
    [NewPolicyType.DATA_QUERY]: OldPolicyType.DATA_QUERY,
  };
  return mapping[newType] ?? OldPolicyType.UNRECOGNIZED;
};

// Convert old SQLReviewRuleLevel enum to new (string to numeric)
export const convertOldSQLReviewRuleLevelToNew = (oldLevel: OldSQLReviewRuleLevel): NewSQLReviewRuleLevel => {
  const mapping: Record<OldSQLReviewRuleLevel, NewSQLReviewRuleLevel> = {
    [OldSQLReviewRuleLevel.LEVEL_UNSPECIFIED]: NewSQLReviewRuleLevel.LEVEL_UNSPECIFIED,
    [OldSQLReviewRuleLevel.ERROR]: NewSQLReviewRuleLevel.ERROR,
    [OldSQLReviewRuleLevel.WARNING]: NewSQLReviewRuleLevel.WARNING,
    [OldSQLReviewRuleLevel.DISABLED]: NewSQLReviewRuleLevel.DISABLED,
    [OldSQLReviewRuleLevel.UNRECOGNIZED]: NewSQLReviewRuleLevel.LEVEL_UNSPECIFIED,
  };
  return mapping[oldLevel] ?? NewSQLReviewRuleLevel.LEVEL_UNSPECIFIED;
};

// Convert new SQLReviewRuleLevel enum to old (numeric to string)
export const convertNewSQLReviewRuleLevelToOld = (newLevel: NewSQLReviewRuleLevel): OldSQLReviewRuleLevel => {
  const mapping: Record<NewSQLReviewRuleLevel, OldSQLReviewRuleLevel> = {
    [NewSQLReviewRuleLevel.LEVEL_UNSPECIFIED]: OldSQLReviewRuleLevel.LEVEL_UNSPECIFIED,
    [NewSQLReviewRuleLevel.ERROR]: OldSQLReviewRuleLevel.ERROR,
    [NewSQLReviewRuleLevel.WARNING]: OldSQLReviewRuleLevel.WARNING,
    [NewSQLReviewRuleLevel.DISABLED]: OldSQLReviewRuleLevel.DISABLED,
  };
  return mapping[newLevel] ?? OldSQLReviewRuleLevel.UNRECOGNIZED;
};