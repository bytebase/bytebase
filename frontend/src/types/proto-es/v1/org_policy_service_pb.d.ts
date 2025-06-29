// @generated by protoc-gen-es v2.5.2
// @generated from file v1/org_policy_service.proto (package bytebase.v1, syntax proto3)
/* eslint-disable */

import type { GenEnum, GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import type { Message } from "@bufbuild/protobuf";
import type { Duration, EmptySchema, FieldMask } from "@bufbuild/protobuf/wkt";
import type { Engine } from "./common_pb";
import type { Expr } from "../google/type/expr_pb";

/**
 * Describes the file v1/org_policy_service.proto.
 */
export declare const file_v1_org_policy_service: GenFile;

/**
 * @generated from message bytebase.v1.CreatePolicyRequest
 */
export declare type CreatePolicyRequest = Message<"bytebase.v1.CreatePolicyRequest"> & {
  /**
   * The parent resource where this instance will be created.
   * Workspace resource name: "".
   * Environment resource name: environments/environment-id.
   * Instance resource name: instances/instance-id.
   * Database resource name: instances/instance-id/databases/database-name.
   *
   * @generated from field: string parent = 1;
   */
  parent: string;

  /**
   * The policy to create.
   *
   * @generated from field: bytebase.v1.Policy policy = 2;
   */
  policy?: Policy;

  /**
   * @generated from field: bytebase.v1.PolicyType type = 3;
   */
  type: PolicyType;
};

/**
 * Describes the message bytebase.v1.CreatePolicyRequest.
 * Use `create(CreatePolicyRequestSchema)` to create a new message.
 */
export declare const CreatePolicyRequestSchema: GenMessage<CreatePolicyRequest>;

/**
 * @generated from message bytebase.v1.UpdatePolicyRequest
 */
export declare type UpdatePolicyRequest = Message<"bytebase.v1.UpdatePolicyRequest"> & {
  /**
   * The policy to update.
   *
   * The policy's `name` field is used to identify the instance to update.
   * Format: {resource name}/policies/{policy type}
   * Workspace resource name: "".
   * Environment resource name: environments/environment-id.
   * Instance resource name: instances/instance-id.
   * Database resource name: instances/instance-id/databases/database-name.
   *
   * @generated from field: bytebase.v1.Policy policy = 1;
   */
  policy?: Policy;

  /**
   * The list of fields to update.
   *
   * @generated from field: google.protobuf.FieldMask update_mask = 2;
   */
  updateMask?: FieldMask;

  /**
   * If set to true, and the policy is not found, a new policy will be created.
   * In this situation, `update_mask` is ignored.
   *
   * @generated from field: bool allow_missing = 3;
   */
  allowMissing: boolean;
};

/**
 * Describes the message bytebase.v1.UpdatePolicyRequest.
 * Use `create(UpdatePolicyRequestSchema)` to create a new message.
 */
export declare const UpdatePolicyRequestSchema: GenMessage<UpdatePolicyRequest>;

/**
 * @generated from message bytebase.v1.DeletePolicyRequest
 */
export declare type DeletePolicyRequest = Message<"bytebase.v1.DeletePolicyRequest"> & {
  /**
   * The policy's `name` field is used to identify the instance to update.
   * Format: {resource name}/policies/{policy type}
   * Workspace resource name: "".
   * Environment resource name: environments/environment-id.
   * Instance resource name: instances/instance-id.
   * Database resource name: instances/instance-id/databases/database-name.
   *
   * @generated from field: string name = 1;
   */
  name: string;
};

/**
 * Describes the message bytebase.v1.DeletePolicyRequest.
 * Use `create(DeletePolicyRequestSchema)` to create a new message.
 */
export declare const DeletePolicyRequestSchema: GenMessage<DeletePolicyRequest>;

/**
 * @generated from message bytebase.v1.GetPolicyRequest
 */
export declare type GetPolicyRequest = Message<"bytebase.v1.GetPolicyRequest"> & {
  /**
   * The name of the policy to retrieve.
   * Format: {resource type}/{resource id}/policies/{policy type}
   *
   * @generated from field: string name = 1;
   */
  name: string;
};

/**
 * Describes the message bytebase.v1.GetPolicyRequest.
 * Use `create(GetPolicyRequestSchema)` to create a new message.
 */
export declare const GetPolicyRequestSchema: GenMessage<GetPolicyRequest>;

/**
 * @generated from message bytebase.v1.ListPoliciesRequest
 */
export declare type ListPoliciesRequest = Message<"bytebase.v1.ListPoliciesRequest"> & {
  /**
   * The parent, which owns this collection of policies.
   * Format: {resource type}/{resource id}
   *
   * @generated from field: string parent = 1;
   */
  parent: string;

  /**
   * @generated from field: optional bytebase.v1.PolicyType policy_type = 2;
   */
  policyType?: PolicyType;

  /**
   * Not used.
   * The maximum number of policies to return. The service may return fewer than
   * this value.
   * If unspecified, at most 10 policies will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   *
   * @generated from field: int32 page_size = 3;
   */
  pageSize: number;

  /**
   * Not used.
   * A page token, received from a previous `ListPolicies` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListPolicies` must match
   * the call that provided the page token.
   *
   * @generated from field: string page_token = 4;
   */
  pageToken: string;

  /**
   * Show deleted policies if specified.
   *
   * @generated from field: bool show_deleted = 5;
   */
  showDeleted: boolean;
};

/**
 * Describes the message bytebase.v1.ListPoliciesRequest.
 * Use `create(ListPoliciesRequestSchema)` to create a new message.
 */
export declare const ListPoliciesRequestSchema: GenMessage<ListPoliciesRequest>;

/**
 * @generated from message bytebase.v1.ListPoliciesResponse
 */
export declare type ListPoliciesResponse = Message<"bytebase.v1.ListPoliciesResponse"> & {
  /**
   * The policies from the specified request.
   *
   * @generated from field: repeated bytebase.v1.Policy policies = 1;
   */
  policies: Policy[];

  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   *
   * @generated from field: string next_page_token = 2;
   */
  nextPageToken: string;
};

/**
 * Describes the message bytebase.v1.ListPoliciesResponse.
 * Use `create(ListPoliciesResponseSchema)` to create a new message.
 */
export declare const ListPoliciesResponseSchema: GenMessage<ListPoliciesResponse>;

/**
 * @generated from message bytebase.v1.Policy
 */
export declare type Policy = Message<"bytebase.v1.Policy"> & {
  /**
   * The name of the policy.
   * Format: {resource name}/policies/{policy type}
   * Workspace resource name: "".
   * Environment resource name: environments/environment-id.
   * Instance resource name: instances/instance-id.
   * Database resource name: instances/instance-id/databases/database-name.
   *
   * @generated from field: string name = 1;
   */
  name: string;

  /**
   * @generated from field: bool inherit_from_parent = 4;
   */
  inheritFromParent: boolean;

  /**
   * @generated from field: bytebase.v1.PolicyType type = 5;
   */
  type: PolicyType;

  /**
   * @generated from oneof bytebase.v1.Policy.policy
   */
  policy: {
    /**
     * @generated from field: bytebase.v1.RolloutPolicy rollout_policy = 19;
     */
    value: RolloutPolicy;
    case: "rolloutPolicy";
  } | {
    /**
     * @generated from field: bytebase.v1.DisableCopyDataPolicy disable_copy_data_policy = 16;
     */
    value: DisableCopyDataPolicy;
    case: "disableCopyDataPolicy";
  } | {
    /**
     * @generated from field: bytebase.v1.MaskingRulePolicy masking_rule_policy = 17;
     */
    value: MaskingRulePolicy;
    case: "maskingRulePolicy";
  } | {
    /**
     * @generated from field: bytebase.v1.MaskingExceptionPolicy masking_exception_policy = 18;
     */
    value: MaskingExceptionPolicy;
    case: "maskingExceptionPolicy";
  } | {
    /**
     * @generated from field: bytebase.v1.RestrictIssueCreationForSQLReviewPolicy restrict_issue_creation_for_sql_review_policy = 20;
     */
    value: RestrictIssueCreationForSQLReviewPolicy;
    case: "restrictIssueCreationForSqlReviewPolicy";
  } | {
    /**
     * @generated from field: bytebase.v1.TagPolicy tag_policy = 21;
     */
    value: TagPolicy;
    case: "tagPolicy";
  } | {
    /**
     * @generated from field: bytebase.v1.DataSourceQueryPolicy data_source_query_policy = 22;
     */
    value: DataSourceQueryPolicy;
    case: "dataSourceQueryPolicy";
  } | {
    /**
     * @generated from field: bytebase.v1.ExportDataPolicy export_data_policy = 23;
     */
    value: ExportDataPolicy;
    case: "exportDataPolicy";
  } | {
    /**
     * @generated from field: bytebase.v1.QueryDataPolicy query_data_policy = 24;
     */
    value: QueryDataPolicy;
    case: "queryDataPolicy";
  } | { case: undefined; value?: undefined };

  /**
   * @generated from field: bool enforce = 13;
   */
  enforce: boolean;

  /**
   * The resource type for the policy.
   *
   * @generated from field: bytebase.v1.PolicyResourceType resource_type = 14;
   */
  resourceType: PolicyResourceType;
};

/**
 * Describes the message bytebase.v1.Policy.
 * Use `create(PolicySchema)` to create a new message.
 */
export declare const PolicySchema: GenMessage<Policy>;

/**
 * @generated from message bytebase.v1.RolloutPolicy
 */
export declare type RolloutPolicy = Message<"bytebase.v1.RolloutPolicy"> & {
  /**
   * @generated from field: bool automatic = 1;
   */
  automatic: boolean;

  /**
   * @generated from field: repeated string roles = 2;
   */
  roles: string[];

  /**
   * roles/LAST_APPROVER
   * roles/CREATOR
   *
   * @generated from field: repeated string issue_roles = 3;
   */
  issueRoles: string[];
};

/**
 * Describes the message bytebase.v1.RolloutPolicy.
 * Use `create(RolloutPolicySchema)` to create a new message.
 */
export declare const RolloutPolicySchema: GenMessage<RolloutPolicy>;

/**
 * @generated from message bytebase.v1.DisableCopyDataPolicy
 */
export declare type DisableCopyDataPolicy = Message<"bytebase.v1.DisableCopyDataPolicy"> & {
  /**
   * @generated from field: bool active = 1;
   */
  active: boolean;
};

/**
 * Describes the message bytebase.v1.DisableCopyDataPolicy.
 * Use `create(DisableCopyDataPolicySchema)` to create a new message.
 */
export declare const DisableCopyDataPolicySchema: GenMessage<DisableCopyDataPolicy>;

/**
 * @generated from message bytebase.v1.ExportDataPolicy
 */
export declare type ExportDataPolicy = Message<"bytebase.v1.ExportDataPolicy"> & {
  /**
   * @generated from field: bool disable = 1;
   */
  disable: boolean;
};

/**
 * Describes the message bytebase.v1.ExportDataPolicy.
 * Use `create(ExportDataPolicySchema)` to create a new message.
 */
export declare const ExportDataPolicySchema: GenMessage<ExportDataPolicy>;

/**
 * QueryDataPolicy is the policy configuration for querying data.
 *
 * @generated from message bytebase.v1.QueryDataPolicy
 */
export declare type QueryDataPolicy = Message<"bytebase.v1.QueryDataPolicy"> & {
  /**
   * The query timeout duration.
   *
   * @generated from field: google.protobuf.Duration timeout = 1;
   */
  timeout?: Duration;
};

/**
 * Describes the message bytebase.v1.QueryDataPolicy.
 * Use `create(QueryDataPolicySchema)` to create a new message.
 */
export declare const QueryDataPolicySchema: GenMessage<QueryDataPolicy>;

/**
 * The SQL review rules. Check the SQL_REVIEW_RULES_DOCUMENTATION.md for details.
 *
 * @generated from message bytebase.v1.SQLReviewRule
 */
export declare type SQLReviewRule = Message<"bytebase.v1.SQLReviewRule"> & {
  /**
   * @generated from field: string type = 1;
   */
  type: string;

  /**
   * @generated from field: bytebase.v1.SQLReviewRuleLevel level = 2;
   */
  level: SQLReviewRuleLevel;

  /**
   * The payload is a JSON string that varies by rule type.
   *
   * @generated from field: string payload = 3;
   */
  payload: string;

  /**
   * @generated from field: bytebase.v1.Engine engine = 4;
   */
  engine: Engine;

  /**
   * @generated from field: string comment = 5;
   */
  comment: string;
};

/**
 * Describes the message bytebase.v1.SQLReviewRule.
 * Use `create(SQLReviewRuleSchema)` to create a new message.
 */
export declare const SQLReviewRuleSchema: GenMessage<SQLReviewRule>;

/**
 * MaskingExceptionPolicy is the allowlist of users who can access sensitive data.
 *
 * @generated from message bytebase.v1.MaskingExceptionPolicy
 */
export declare type MaskingExceptionPolicy = Message<"bytebase.v1.MaskingExceptionPolicy"> & {
  /**
   * @generated from field: repeated bytebase.v1.MaskingExceptionPolicy.MaskingException masking_exceptions = 1;
   */
  maskingExceptions: MaskingExceptionPolicy_MaskingException[];
};

/**
 * Describes the message bytebase.v1.MaskingExceptionPolicy.
 * Use `create(MaskingExceptionPolicySchema)` to create a new message.
 */
export declare const MaskingExceptionPolicySchema: GenMessage<MaskingExceptionPolicy>;

/**
 * @generated from message bytebase.v1.MaskingExceptionPolicy.MaskingException
 */
export declare type MaskingExceptionPolicy_MaskingException = Message<"bytebase.v1.MaskingExceptionPolicy.MaskingException"> & {
  /**
   * action is the action that the user can access sensitive data.
   *
   * @generated from field: bytebase.v1.MaskingExceptionPolicy.MaskingException.Action action = 1;
   */
  action: MaskingExceptionPolicy_MaskingException_Action;

  /**
   * Member is the principal who bind to this exception policy instance.
   *
   * - `user:{email}`: An email address that represents a specific Bytebase account. For example, `alice@example.com`.
   * - `group:{email}`: An email address for group.
   *
   * @generated from field: string member = 3;
   */
  member: string;

  /**
   * The condition that is associated with this exception policy instance.
   * The syntax and semantics of CEL are documented at https://github.com/google/cel-spec
   * If the condition is empty, means the user can access all databases without expiration.
   *
   * Support variables:
   * resource.instance_id: the instance resource id. Only support "==" operation.
   * resource.database_name: the database name. Only support "==" operation.
   * resource.schema_name: the schema name. Only support "==" operation.
   * resource.table_name: the table name. Only support "==" operation.
   * resource.column_name: the column name. Only support "==" operation.
   * request.time: the expiration. Only support "<" operation in `request.time < timestamp("{ISO datetime string format}")`
   * All variables should join with "&&" condition.
   *
   * For example:
   * resource.instance_id == "local" && resource.database_name == "employee" && request.time < timestamp("2025-04-30T11:10:39.000Z")
   * resource.instance_id == "local" && resource.database_name == "employee"
   *
   * @generated from field: google.type.Expr condition = 4;
   */
  condition?: Expr;
};

/**
 * Describes the message bytebase.v1.MaskingExceptionPolicy.MaskingException.
 * Use `create(MaskingExceptionPolicy_MaskingExceptionSchema)` to create a new message.
 */
export declare const MaskingExceptionPolicy_MaskingExceptionSchema: GenMessage<MaskingExceptionPolicy_MaskingException>;

/**
 * @generated from enum bytebase.v1.MaskingExceptionPolicy.MaskingException.Action
 */
export enum MaskingExceptionPolicy_MaskingException_Action {
  /**
   * @generated from enum value: ACTION_UNSPECIFIED = 0;
   */
  ACTION_UNSPECIFIED = 0,

  /**
   * @generated from enum value: QUERY = 1;
   */
  QUERY = 1,

  /**
   * @generated from enum value: EXPORT = 2;
   */
  EXPORT = 2,
}

/**
 * Describes the enum bytebase.v1.MaskingExceptionPolicy.MaskingException.Action.
 */
export declare const MaskingExceptionPolicy_MaskingException_ActionSchema: GenEnum<MaskingExceptionPolicy_MaskingException_Action>;

/**
 * @generated from message bytebase.v1.MaskingRulePolicy
 */
export declare type MaskingRulePolicy = Message<"bytebase.v1.MaskingRulePolicy"> & {
  /**
   * @generated from field: repeated bytebase.v1.MaskingRulePolicy.MaskingRule rules = 1;
   */
  rules: MaskingRulePolicy_MaskingRule[];
};

/**
 * Describes the message bytebase.v1.MaskingRulePolicy.
 * Use `create(MaskingRulePolicySchema)` to create a new message.
 */
export declare const MaskingRulePolicySchema: GenMessage<MaskingRulePolicy>;

/**
 * @generated from message bytebase.v1.MaskingRulePolicy.MaskingRule
 */
export declare type MaskingRulePolicy_MaskingRule = Message<"bytebase.v1.MaskingRulePolicy.MaskingRule"> & {
  /**
   * A unique identifier for a node in UUID format.
   *
   * @generated from field: string id = 1;
   */
  id: string;

  /**
   * The condition for the masking rule.
   * The syntax and semantics of CEL are documented at https://github.com/google/cel-spec
   *
   * Support variables:
   * environment_id: the environment resource id.
   * project_id: the project resource id.
   * instance_id: the instance resource id.
   * database_name: the database name.
   * table_name: the table name.
   * column_name: the column name.
   * classification_level: the classification level.
   *
   * Each variable support following operations:
   * ==: the value equals the target.
   * !=: the value not equals the target.
   * in: the value matches one of the targets.
   * !(in): the value not matches any of the targets.
   *
   * For example:
   * environment_id == "test" && project_id == "sample-project"
   * instance_id == "sample-instance" && database_name == "employee" && table_name in ["table1", "table2"]
   * environment_id != "test" || !(project_id in ["poject1", "prject2"])
   * instance_id == "sample-instance" && (database_name == "db1" || database_name == "db2")
   *
   * @generated from field: google.type.Expr condition = 2;
   */
  condition?: Expr;

  /**
   * @generated from field: string semantic_type = 3;
   */
  semanticType: string;
};

/**
 * Describes the message bytebase.v1.MaskingRulePolicy.MaskingRule.
 * Use `create(MaskingRulePolicy_MaskingRuleSchema)` to create a new message.
 */
export declare const MaskingRulePolicy_MaskingRuleSchema: GenMessage<MaskingRulePolicy_MaskingRule>;

/**
 * @generated from message bytebase.v1.RestrictIssueCreationForSQLReviewPolicy
 */
export declare type RestrictIssueCreationForSQLReviewPolicy = Message<"bytebase.v1.RestrictIssueCreationForSQLReviewPolicy"> & {
  /**
   * @generated from field: bool disallow = 1;
   */
  disallow: boolean;
};

/**
 * Describes the message bytebase.v1.RestrictIssueCreationForSQLReviewPolicy.
 * Use `create(RestrictIssueCreationForSQLReviewPolicySchema)` to create a new message.
 */
export declare const RestrictIssueCreationForSQLReviewPolicySchema: GenMessage<RestrictIssueCreationForSQLReviewPolicy>;

/**
 * @generated from message bytebase.v1.TagPolicy
 */
export declare type TagPolicy = Message<"bytebase.v1.TagPolicy"> & {
  /**
   * tags is the key - value map for resources.
   * for example, the environment resource can have the sql review config tag, like "bb.tag.review_config": "reviewConfigs/{review config resource id}"
   *
   * @generated from field: map<string, string> tags = 1;
   */
  tags: { [key: string]: string };
};

/**
 * Describes the message bytebase.v1.TagPolicy.
 * Use `create(TagPolicySchema)` to create a new message.
 */
export declare const TagPolicySchema: GenMessage<TagPolicy>;

/**
 * DataSourceQueryPolicy is the policy configuration for running statements in the SQL editor.
 *
 * @generated from message bytebase.v1.DataSourceQueryPolicy
 */
export declare type DataSourceQueryPolicy = Message<"bytebase.v1.DataSourceQueryPolicy"> & {
  /**
   * @generated from field: bytebase.v1.DataSourceQueryPolicy.Restriction admin_data_source_restriction = 1;
   */
  adminDataSourceRestriction: DataSourceQueryPolicy_Restriction;

  /**
   * Disallow running DDL statements in the SQL editor.
   *
   * @generated from field: bool disallow_ddl = 2;
   */
  disallowDdl: boolean;

  /**
   * Disallow running DML statements in the SQL editor.
   *
   * @generated from field: bool disallow_dml = 3;
   */
  disallowDml: boolean;
};

/**
 * Describes the message bytebase.v1.DataSourceQueryPolicy.
 * Use `create(DataSourceQueryPolicySchema)` to create a new message.
 */
export declare const DataSourceQueryPolicySchema: GenMessage<DataSourceQueryPolicy>;

/**
 * @generated from enum bytebase.v1.DataSourceQueryPolicy.Restriction
 */
export enum DataSourceQueryPolicy_Restriction {
  /**
   * @generated from enum value: RESTRICTION_UNSPECIFIED = 0;
   */
  RESTRICTION_UNSPECIFIED = 0,

  /**
   * Allow to query admin data sources when there is no read-only data source.
   *
   * @generated from enum value: FALLBACK = 1;
   */
  FALLBACK = 1,

  /**
   * Disallow to query admin data sources.
   *
   * @generated from enum value: DISALLOW = 2;
   */
  DISALLOW = 2,
}

/**
 * Describes the enum bytebase.v1.DataSourceQueryPolicy.Restriction.
 */
export declare const DataSourceQueryPolicy_RestrictionSchema: GenEnum<DataSourceQueryPolicy_Restriction>;

/**
 * @generated from enum bytebase.v1.PolicyType
 */
export enum PolicyType {
  /**
   * @generated from enum value: POLICY_TYPE_UNSPECIFIED = 0;
   */
  POLICY_TYPE_UNSPECIFIED = 0,

  /**
   * @generated from enum value: ROLLOUT_POLICY = 11;
   */
  ROLLOUT_POLICY = 11,

  /**
   * @generated from enum value: DISABLE_COPY_DATA = 8;
   */
  DISABLE_COPY_DATA = 8,

  /**
   * @generated from enum value: MASKING_RULE = 9;
   */
  MASKING_RULE = 9,

  /**
   * @generated from enum value: MASKING_EXCEPTION = 10;
   */
  MASKING_EXCEPTION = 10,

  /**
   * @generated from enum value: RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW = 12;
   */
  RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW = 12,

  /**
   * @generated from enum value: TAG = 13;
   */
  TAG = 13,

  /**
   * @generated from enum value: DATA_SOURCE_QUERY = 14;
   */
  DATA_SOURCE_QUERY = 14,

  /**
   * @generated from enum value: DATA_EXPORT = 15;
   */
  DATA_EXPORT = 15,

  /**
   * @generated from enum value: DATA_QUERY = 16;
   */
  DATA_QUERY = 16,
}

/**
 * Describes the enum bytebase.v1.PolicyType.
 */
export declare const PolicyTypeSchema: GenEnum<PolicyType>;

/**
 * @generated from enum bytebase.v1.PolicyResourceType
 */
export enum PolicyResourceType {
  /**
   * @generated from enum value: RESOURCE_TYPE_UNSPECIFIED = 0;
   */
  RESOURCE_TYPE_UNSPECIFIED = 0,

  /**
   * @generated from enum value: WORKSPACE = 1;
   */
  WORKSPACE = 1,

  /**
   * @generated from enum value: ENVIRONMENT = 2;
   */
  ENVIRONMENT = 2,

  /**
   * @generated from enum value: PROJECT = 3;
   */
  PROJECT = 3,
}

/**
 * Describes the enum bytebase.v1.PolicyResourceType.
 */
export declare const PolicyResourceTypeSchema: GenEnum<PolicyResourceType>;

/**
 * @generated from enum bytebase.v1.SQLReviewRuleLevel
 */
export enum SQLReviewRuleLevel {
  /**
   * @generated from enum value: LEVEL_UNSPECIFIED = 0;
   */
  LEVEL_UNSPECIFIED = 0,

  /**
   * @generated from enum value: ERROR = 1;
   */
  ERROR = 1,

  /**
   * @generated from enum value: WARNING = 2;
   */
  WARNING = 2,

  /**
   * @generated from enum value: DISABLED = 3;
   */
  DISABLED = 3,
}

/**
 * Describes the enum bytebase.v1.SQLReviewRuleLevel.
 */
export declare const SQLReviewRuleLevelSchema: GenEnum<SQLReviewRuleLevel>;

/**
 * @generated from service bytebase.v1.OrgPolicyService
 */
export declare const OrgPolicyService: GenService<{
  /**
   * Permissions required: bb.policies.get
   *
   * @generated from rpc bytebase.v1.OrgPolicyService.GetPolicy
   */
  getPolicy: {
    methodKind: "unary";
    input: typeof GetPolicyRequestSchema;
    output: typeof PolicySchema;
  },
  /**
   * Permissions required: bb.policies.list
   *
   * @generated from rpc bytebase.v1.OrgPolicyService.ListPolicies
   */
  listPolicies: {
    methodKind: "unary";
    input: typeof ListPoliciesRequestSchema;
    output: typeof ListPoliciesResponseSchema;
  },
  /**
   * Permissions required: bb.policies.create
   *
   * @generated from rpc bytebase.v1.OrgPolicyService.CreatePolicy
   */
  createPolicy: {
    methodKind: "unary";
    input: typeof CreatePolicyRequestSchema;
    output: typeof PolicySchema;
  },
  /**
   * Permissions required: bb.policies.update
   *
   * @generated from rpc bytebase.v1.OrgPolicyService.UpdatePolicy
   */
  updatePolicy: {
    methodKind: "unary";
    input: typeof UpdatePolicyRequestSchema;
    output: typeof PolicySchema;
  },
  /**
   * Permissions required: bb.policies.delete
   *
   * @generated from rpc bytebase.v1.OrgPolicyService.DeletePolicy
   */
  deletePolicy: {
    methodKind: "unary";
    input: typeof DeletePolicyRequestSchema;
    output: typeof EmptySchema;
  },
}>;

