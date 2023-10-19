/* eslint-disable */
import _m0 from "protobufjs/minimal";
import { Duration } from "../google/protobuf/duration";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { Expr } from "../google/type/expr";
import { Engine, engineFromJSON, engineToJSON, MaskingLevel, maskingLevelFromJSON, maskingLevelToJSON } from "./common";
import { DeploymentType, deploymentTypeFromJSON, deploymentTypeToJSON } from "./deployment";
import { IamPolicy } from "./iam_policy";

export const protobufPackage = "bytebase.v1";

export enum PolicyType {
  POLICY_TYPE_UNSPECIFIED = 0,
  WORKSPACE_IAM = 1,
  DEPLOYMENT_APPROVAL = 2,
  ROLLOUT_POLICY = 11,
  BACKUP_PLAN = 3,
  SQL_REVIEW = 4,
  MASKING = 5,
  SLOW_QUERY = 7,
  DISABLE_COPY_DATA = 8,
  MASKING_RULE = 9,
  MASKING_EXCEPTION = 10,
  UNRECOGNIZED = -1,
}

export function policyTypeFromJSON(object: any): PolicyType {
  switch (object) {
    case 0:
    case "POLICY_TYPE_UNSPECIFIED":
      return PolicyType.POLICY_TYPE_UNSPECIFIED;
    case 1:
    case "WORKSPACE_IAM":
      return PolicyType.WORKSPACE_IAM;
    case 2:
    case "DEPLOYMENT_APPROVAL":
      return PolicyType.DEPLOYMENT_APPROVAL;
    case 11:
    case "ROLLOUT_POLICY":
      return PolicyType.ROLLOUT_POLICY;
    case 3:
    case "BACKUP_PLAN":
      return PolicyType.BACKUP_PLAN;
    case 4:
    case "SQL_REVIEW":
      return PolicyType.SQL_REVIEW;
    case 5:
    case "MASKING":
      return PolicyType.MASKING;
    case 7:
    case "SLOW_QUERY":
      return PolicyType.SLOW_QUERY;
    case 8:
    case "DISABLE_COPY_DATA":
      return PolicyType.DISABLE_COPY_DATA;
    case 9:
    case "MASKING_RULE":
      return PolicyType.MASKING_RULE;
    case 10:
    case "MASKING_EXCEPTION":
      return PolicyType.MASKING_EXCEPTION;
    case -1:
    case "UNRECOGNIZED":
    default:
      return PolicyType.UNRECOGNIZED;
  }
}

export function policyTypeToJSON(object: PolicyType): string {
  switch (object) {
    case PolicyType.POLICY_TYPE_UNSPECIFIED:
      return "POLICY_TYPE_UNSPECIFIED";
    case PolicyType.WORKSPACE_IAM:
      return "WORKSPACE_IAM";
    case PolicyType.DEPLOYMENT_APPROVAL:
      return "DEPLOYMENT_APPROVAL";
    case PolicyType.ROLLOUT_POLICY:
      return "ROLLOUT_POLICY";
    case PolicyType.BACKUP_PLAN:
      return "BACKUP_PLAN";
    case PolicyType.SQL_REVIEW:
      return "SQL_REVIEW";
    case PolicyType.MASKING:
      return "MASKING";
    case PolicyType.SLOW_QUERY:
      return "SLOW_QUERY";
    case PolicyType.DISABLE_COPY_DATA:
      return "DISABLE_COPY_DATA";
    case PolicyType.MASKING_RULE:
      return "MASKING_RULE";
    case PolicyType.MASKING_EXCEPTION:
      return "MASKING_EXCEPTION";
    case PolicyType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum PolicyResourceType {
  RESOURCE_TYPE_UNSPECIFIED = 0,
  WORKSPACE = 1,
  ENVIRONMENT = 2,
  PROJECT = 3,
  INSTANCE = 4,
  DATABASE = 5,
  UNRECOGNIZED = -1,
}

export function policyResourceTypeFromJSON(object: any): PolicyResourceType {
  switch (object) {
    case 0:
    case "RESOURCE_TYPE_UNSPECIFIED":
      return PolicyResourceType.RESOURCE_TYPE_UNSPECIFIED;
    case 1:
    case "WORKSPACE":
      return PolicyResourceType.WORKSPACE;
    case 2:
    case "ENVIRONMENT":
      return PolicyResourceType.ENVIRONMENT;
    case 3:
    case "PROJECT":
      return PolicyResourceType.PROJECT;
    case 4:
    case "INSTANCE":
      return PolicyResourceType.INSTANCE;
    case 5:
    case "DATABASE":
      return PolicyResourceType.DATABASE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return PolicyResourceType.UNRECOGNIZED;
  }
}

export function policyResourceTypeToJSON(object: PolicyResourceType): string {
  switch (object) {
    case PolicyResourceType.RESOURCE_TYPE_UNSPECIFIED:
      return "RESOURCE_TYPE_UNSPECIFIED";
    case PolicyResourceType.WORKSPACE:
      return "WORKSPACE";
    case PolicyResourceType.ENVIRONMENT:
      return "ENVIRONMENT";
    case PolicyResourceType.PROJECT:
      return "PROJECT";
    case PolicyResourceType.INSTANCE:
      return "INSTANCE";
    case PolicyResourceType.DATABASE:
      return "DATABASE";
    case PolicyResourceType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum ApprovalGroup {
  ASSIGNEE_GROUP_UNSPECIFIED = 0,
  APPROVAL_GROUP_DBA = 1,
  APPROVAL_GROUP_PROJECT_OWNER = 2,
  UNRECOGNIZED = -1,
}

export function approvalGroupFromJSON(object: any): ApprovalGroup {
  switch (object) {
    case 0:
    case "ASSIGNEE_GROUP_UNSPECIFIED":
      return ApprovalGroup.ASSIGNEE_GROUP_UNSPECIFIED;
    case 1:
    case "APPROVAL_GROUP_DBA":
      return ApprovalGroup.APPROVAL_GROUP_DBA;
    case 2:
    case "APPROVAL_GROUP_PROJECT_OWNER":
      return ApprovalGroup.APPROVAL_GROUP_PROJECT_OWNER;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ApprovalGroup.UNRECOGNIZED;
  }
}

export function approvalGroupToJSON(object: ApprovalGroup): string {
  switch (object) {
    case ApprovalGroup.ASSIGNEE_GROUP_UNSPECIFIED:
      return "ASSIGNEE_GROUP_UNSPECIFIED";
    case ApprovalGroup.APPROVAL_GROUP_DBA:
      return "APPROVAL_GROUP_DBA";
    case ApprovalGroup.APPROVAL_GROUP_PROJECT_OWNER:
      return "APPROVAL_GROUP_PROJECT_OWNER";
    case ApprovalGroup.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum ApprovalStrategy {
  APPROVAL_STRATEGY_UNSPECIFIED = 0,
  AUTOMATIC = 1,
  MANUAL = 2,
  UNRECOGNIZED = -1,
}

export function approvalStrategyFromJSON(object: any): ApprovalStrategy {
  switch (object) {
    case 0:
    case "APPROVAL_STRATEGY_UNSPECIFIED":
      return ApprovalStrategy.APPROVAL_STRATEGY_UNSPECIFIED;
    case 1:
    case "AUTOMATIC":
      return ApprovalStrategy.AUTOMATIC;
    case 2:
    case "MANUAL":
      return ApprovalStrategy.MANUAL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ApprovalStrategy.UNRECOGNIZED;
  }
}

export function approvalStrategyToJSON(object: ApprovalStrategy): string {
  switch (object) {
    case ApprovalStrategy.APPROVAL_STRATEGY_UNSPECIFIED:
      return "APPROVAL_STRATEGY_UNSPECIFIED";
    case ApprovalStrategy.AUTOMATIC:
      return "AUTOMATIC";
    case ApprovalStrategy.MANUAL:
      return "MANUAL";
    case ApprovalStrategy.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum BackupPlanSchedule {
  SCHEDULE_UNSPECIFIED = 0,
  UNSET = 1,
  DAILY = 2,
  WEEKLY = 3,
  UNRECOGNIZED = -1,
}

export function backupPlanScheduleFromJSON(object: any): BackupPlanSchedule {
  switch (object) {
    case 0:
    case "SCHEDULE_UNSPECIFIED":
      return BackupPlanSchedule.SCHEDULE_UNSPECIFIED;
    case 1:
    case "UNSET":
      return BackupPlanSchedule.UNSET;
    case 2:
    case "DAILY":
      return BackupPlanSchedule.DAILY;
    case 3:
    case "WEEKLY":
      return BackupPlanSchedule.WEEKLY;
    case -1:
    case "UNRECOGNIZED":
    default:
      return BackupPlanSchedule.UNRECOGNIZED;
  }
}

export function backupPlanScheduleToJSON(object: BackupPlanSchedule): string {
  switch (object) {
    case BackupPlanSchedule.SCHEDULE_UNSPECIFIED:
      return "SCHEDULE_UNSPECIFIED";
    case BackupPlanSchedule.UNSET:
      return "UNSET";
    case BackupPlanSchedule.DAILY:
      return "DAILY";
    case BackupPlanSchedule.WEEKLY:
      return "WEEKLY";
    case BackupPlanSchedule.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum SQLReviewRuleLevel {
  LEVEL_UNSPECIFIED = 0,
  ERROR = 1,
  WARNING = 2,
  DISABLED = 3,
  UNRECOGNIZED = -1,
}

export function sQLReviewRuleLevelFromJSON(object: any): SQLReviewRuleLevel {
  switch (object) {
    case 0:
    case "LEVEL_UNSPECIFIED":
      return SQLReviewRuleLevel.LEVEL_UNSPECIFIED;
    case 1:
    case "ERROR":
      return SQLReviewRuleLevel.ERROR;
    case 2:
    case "WARNING":
      return SQLReviewRuleLevel.WARNING;
    case 3:
    case "DISABLED":
      return SQLReviewRuleLevel.DISABLED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return SQLReviewRuleLevel.UNRECOGNIZED;
  }
}

export function sQLReviewRuleLevelToJSON(object: SQLReviewRuleLevel): string {
  switch (object) {
    case SQLReviewRuleLevel.LEVEL_UNSPECIFIED:
      return "LEVEL_UNSPECIFIED";
    case SQLReviewRuleLevel.ERROR:
      return "ERROR";
    case SQLReviewRuleLevel.WARNING:
      return "WARNING";
    case SQLReviewRuleLevel.DISABLED:
      return "DISABLED";
    case SQLReviewRuleLevel.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface CreatePolicyRequest {
  /**
   * The parent resource where this instance will be created.
   * Workspace resource name: "".
   * Environment resource name: environments/environment-id.
   * Instance resource name: instances/instance-id.
   * Database resource name: instances/instance-id/databases/database-name.
   */
  parent: string;
  /** The policy to create. */
  policy: Policy | undefined;
  type: PolicyType;
}

export interface UpdatePolicyRequest {
  /**
   * The policy to update.
   *
   * The policy's `name` field is used to identify the instance to update.
   * Format: {resource name}/policies/{policy type}
   * Workspace resource name: "".
   * Environment resource name: environments/environment-id.
   * Instance resource name: instances/instance-id.
   * Database resource name: instances/instance-id/databases/database-name.
   */
  policy:
    | Policy
    | undefined;
  /** The list of fields to update. */
  updateMask:
    | string[]
    | undefined;
  /**
   * If set to true, and the policy is not found, a new policy will be created.
   * In this situation, `update_mask` is ignored.
   */
  allowMissing: boolean;
}

export interface DeletePolicyRequest {
  /**
   * The policy's `name` field is used to identify the instance to update.
   * Format: {resource name}/policies/{policy type}
   * Workspace resource name: "".
   * Environment resource name: environments/environment-id.
   * Instance resource name: instances/instance-id.
   * Database resource name: instances/instance-id/databases/database-name.
   */
  name: string;
}

export interface GetPolicyRequest {
  /**
   * The name of the policy to retrieve.
   * Format: {resource type}/{resource id}/policies/{policy type}
   */
  name: string;
}

export interface ListPoliciesRequest {
  /**
   * The parent, which owns this collection of policies.
   * Format: {resource type}/{resource id}/policies/{policy type}
   */
  parent: string;
  policyType?:
    | PolicyType
    | undefined;
  /**
   * The maximum number of policies to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 policies will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `GetPolicies` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `GetPolicies` must match
   * the call that provided the page token.
   */
  pageToken: string;
  /** Show deleted policies if specified. */
  showDeleted: boolean;
}

export interface ListPoliciesResponse {
  /** The policies from the specified request. */
  policies: Policy[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface Policy {
  /**
   * The name of the policy.
   * Format: {resource name}/policies/{policy type}
   * Workspace resource name: "".
   * Environment resource name: environments/environment-id.
   * Instance resource name: instances/instance-id.
   * Database resource name: instances/instance-id/databases/database-name.
   */
  name: string;
  /** The system-assigned, unique identifier for a resource. */
  uid: string;
  inheritFromParent: boolean;
  type: PolicyType;
  workspaceIamPolicy?: IamPolicy | undefined;
  deploymentApprovalPolicy?: DeploymentApprovalPolicy | undefined;
  rolloutPolicy?: RolloutPolicy | undefined;
  backupPlanPolicy?: BackupPlanPolicy | undefined;
  maskingPolicy?: MaskingPolicy | undefined;
  sqlReviewPolicy?: SQLReviewPolicy | undefined;
  slowQueryPolicy?: SlowQueryPolicy | undefined;
  disableCopyDataPolicy?: DisableCopyDataPolicy | undefined;
  maskingRulePolicy?: MaskingRulePolicy | undefined;
  maskingExceptionPolicy?: MaskingExceptionPolicy | undefined;
  enforce: boolean;
  /** The resource type for the policy. */
  resourceType: PolicyResourceType;
  /** The system-assigned, unique identifier for the resource. */
  resourceUid: string;
}

export interface DeploymentApprovalPolicy {
  defaultStrategy: ApprovalStrategy;
  deploymentApprovalStrategies: DeploymentApprovalStrategy[];
}

export interface RolloutPolicy {
  automatic: boolean;
  workspaceRoles: string[];
  projectRoles: string[];
  /**
   * roles/LAST_APPROVER
   * roles/CREATOR
   */
  issueRoles: string[];
}

export interface DeploymentApprovalStrategy {
  deploymentType: DeploymentType;
  approvalGroup: ApprovalGroup;
  approvalStrategy: ApprovalStrategy;
}

export interface BackupPlanPolicy {
  schedule: BackupPlanSchedule;
  retentionDuration: Duration | undefined;
}

export interface SlowQueryPolicy {
  active: boolean;
}

export interface DisableCopyDataPolicy {
  active: boolean;
}

export interface MaskingPolicy {
  maskData: MaskData[];
}

export interface MaskData {
  schema: string;
  table: string;
  column: string;
  maskingLevel: MaskingLevel;
  fullMaskingAlgorithmId: string;
  partialMaskingAlgorithmId: string;
}

export interface SQLReviewPolicy {
  name: string;
  rules: SQLReviewRule[];
}

export interface SQLReviewRule {
  type: string;
  level: SQLReviewRuleLevel;
  payload: string;
  engine: Engine;
  comment: string;
}

/** MaskingExceptionPolicy is the allowlist of users who can access sensitive data. */
export interface MaskingExceptionPolicy {
  maskingExceptions: MaskingExceptionPolicy_MaskingException[];
}

export interface MaskingExceptionPolicy_MaskingException {
  /** action is the action that the user can access sensitive data. */
  action: MaskingExceptionPolicy_MaskingException_Action;
  /** Level is the masking level that the user can access sensitive data. */
  maskingLevel: MaskingLevel;
  /**
   * Member is the principal who bind to this exception policy instance.
   *
   * * `user:{emailid}`: An email address that represents a specific Bytebase account. For example, `alice@example.com`.
   */
  member: string;
  /** The condition that is associated with this exception policy instance. */
  condition: Expr | undefined;
}

export enum MaskingExceptionPolicy_MaskingException_Action {
  ACTION_UNSPECIFIED = 0,
  QUERY = 1,
  EXPORT = 2,
  UNRECOGNIZED = -1,
}

export function maskingExceptionPolicy_MaskingException_ActionFromJSON(
  object: any,
): MaskingExceptionPolicy_MaskingException_Action {
  switch (object) {
    case 0:
    case "ACTION_UNSPECIFIED":
      return MaskingExceptionPolicy_MaskingException_Action.ACTION_UNSPECIFIED;
    case 1:
    case "QUERY":
      return MaskingExceptionPolicy_MaskingException_Action.QUERY;
    case 2:
    case "EXPORT":
      return MaskingExceptionPolicy_MaskingException_Action.EXPORT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return MaskingExceptionPolicy_MaskingException_Action.UNRECOGNIZED;
  }
}

export function maskingExceptionPolicy_MaskingException_ActionToJSON(
  object: MaskingExceptionPolicy_MaskingException_Action,
): string {
  switch (object) {
    case MaskingExceptionPolicy_MaskingException_Action.ACTION_UNSPECIFIED:
      return "ACTION_UNSPECIFIED";
    case MaskingExceptionPolicy_MaskingException_Action.QUERY:
      return "QUERY";
    case MaskingExceptionPolicy_MaskingException_Action.EXPORT:
      return "EXPORT";
    case MaskingExceptionPolicy_MaskingException_Action.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface MaskingRulePolicy {
  rules: MaskingRulePolicy_MaskingRule[];
}

export interface MaskingRulePolicy_MaskingRule {
  /** A unique identifier for a node in UUID format. */
  id: string;
  condition: Expr | undefined;
  maskingLevel: MaskingLevel;
}

function createBaseCreatePolicyRequest(): CreatePolicyRequest {
  return { parent: "", policy: undefined, type: 0 };
}

export const CreatePolicyRequest = {
  encode(message: CreatePolicyRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.policy !== undefined) {
      Policy.encode(message.policy, writer.uint32(18).fork()).ldelim();
    }
    if (message.type !== 0) {
      writer.uint32(24).int32(message.type);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreatePolicyRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreatePolicyRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.parent = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.policy = Policy.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.type = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreatePolicyRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      policy: isSet(object.policy) ? Policy.fromJSON(object.policy) : undefined,
      type: isSet(object.type) ? policyTypeFromJSON(object.type) : 0,
    };
  },

  toJSON(message: CreatePolicyRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.policy !== undefined && (obj.policy = message.policy ? Policy.toJSON(message.policy) : undefined);
    message.type !== undefined && (obj.type = policyTypeToJSON(message.type));
    return obj;
  },

  create(base?: DeepPartial<CreatePolicyRequest>): CreatePolicyRequest {
    return CreatePolicyRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<CreatePolicyRequest>): CreatePolicyRequest {
    const message = createBaseCreatePolicyRequest();
    message.parent = object.parent ?? "";
    message.policy = (object.policy !== undefined && object.policy !== null)
      ? Policy.fromPartial(object.policy)
      : undefined;
    message.type = object.type ?? 0;
    return message;
  },
};

function createBaseUpdatePolicyRequest(): UpdatePolicyRequest {
  return { policy: undefined, updateMask: undefined, allowMissing: false };
}

export const UpdatePolicyRequest = {
  encode(message: UpdatePolicyRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.policy !== undefined) {
      Policy.encode(message.policy, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    if (message.allowMissing === true) {
      writer.uint32(24).bool(message.allowMissing);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdatePolicyRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdatePolicyRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.policy = Policy.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.allowMissing = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UpdatePolicyRequest {
    return {
      policy: isSet(object.policy) ? Policy.fromJSON(object.policy) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
      allowMissing: isSet(object.allowMissing) ? Boolean(object.allowMissing) : false,
    };
  },

  toJSON(message: UpdatePolicyRequest): unknown {
    const obj: any = {};
    message.policy !== undefined && (obj.policy = message.policy ? Policy.toJSON(message.policy) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    message.allowMissing !== undefined && (obj.allowMissing = message.allowMissing);
    return obj;
  },

  create(base?: DeepPartial<UpdatePolicyRequest>): UpdatePolicyRequest {
    return UpdatePolicyRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<UpdatePolicyRequest>): UpdatePolicyRequest {
    const message = createBaseUpdatePolicyRequest();
    message.policy = (object.policy !== undefined && object.policy !== null)
      ? Policy.fromPartial(object.policy)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    message.allowMissing = object.allowMissing ?? false;
    return message;
  },
};

function createBaseDeletePolicyRequest(): DeletePolicyRequest {
  return { name: "" };
}

export const DeletePolicyRequest = {
  encode(message: DeletePolicyRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeletePolicyRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeletePolicyRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DeletePolicyRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: DeletePolicyRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<DeletePolicyRequest>): DeletePolicyRequest {
    return DeletePolicyRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<DeletePolicyRequest>): DeletePolicyRequest {
    const message = createBaseDeletePolicyRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseGetPolicyRequest(): GetPolicyRequest {
  return { name: "" };
}

export const GetPolicyRequest = {
  encode(message: GetPolicyRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetPolicyRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetPolicyRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GetPolicyRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetPolicyRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<GetPolicyRequest>): GetPolicyRequest {
    return GetPolicyRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<GetPolicyRequest>): GetPolicyRequest {
    const message = createBaseGetPolicyRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListPoliciesRequest(): ListPoliciesRequest {
  return { parent: "", policyType: undefined, pageSize: 0, pageToken: "", showDeleted: false };
}

export const ListPoliciesRequest = {
  encode(message: ListPoliciesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.policyType !== undefined) {
      writer.uint32(16).int32(message.policyType);
    }
    if (message.pageSize !== 0) {
      writer.uint32(24).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(34).string(message.pageToken);
    }
    if (message.showDeleted === true) {
      writer.uint32(40).bool(message.showDeleted);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListPoliciesRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListPoliciesRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.parent = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.policyType = reader.int32() as any;
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.pageToken = reader.string();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.showDeleted = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListPoliciesRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      policyType: isSet(object.policyType) ? policyTypeFromJSON(object.policyType) : undefined,
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
      showDeleted: isSet(object.showDeleted) ? Boolean(object.showDeleted) : false,
    };
  },

  toJSON(message: ListPoliciesRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.policyType !== undefined &&
      (obj.policyType = message.policyType !== undefined ? policyTypeToJSON(message.policyType) : undefined);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    message.showDeleted !== undefined && (obj.showDeleted = message.showDeleted);
    return obj;
  },

  create(base?: DeepPartial<ListPoliciesRequest>): ListPoliciesRequest {
    return ListPoliciesRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListPoliciesRequest>): ListPoliciesRequest {
    const message = createBaseListPoliciesRequest();
    message.parent = object.parent ?? "";
    message.policyType = object.policyType ?? undefined;
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    message.showDeleted = object.showDeleted ?? false;
    return message;
  },
};

function createBaseListPoliciesResponse(): ListPoliciesResponse {
  return { policies: [], nextPageToken: "" };
}

export const ListPoliciesResponse = {
  encode(message: ListPoliciesResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.policies) {
      Policy.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListPoliciesResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListPoliciesResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.policies.push(Policy.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.nextPageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListPoliciesResponse {
    return {
      policies: Array.isArray(object?.policies) ? object.policies.map((e: any) => Policy.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListPoliciesResponse): unknown {
    const obj: any = {};
    if (message.policies) {
      obj.policies = message.policies.map((e) => e ? Policy.toJSON(e) : undefined);
    } else {
      obj.policies = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  create(base?: DeepPartial<ListPoliciesResponse>): ListPoliciesResponse {
    return ListPoliciesResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListPoliciesResponse>): ListPoliciesResponse {
    const message = createBaseListPoliciesResponse();
    message.policies = object.policies?.map((e) => Policy.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBasePolicy(): Policy {
  return {
    name: "",
    uid: "",
    inheritFromParent: false,
    type: 0,
    workspaceIamPolicy: undefined,
    deploymentApprovalPolicy: undefined,
    rolloutPolicy: undefined,
    backupPlanPolicy: undefined,
    maskingPolicy: undefined,
    sqlReviewPolicy: undefined,
    slowQueryPolicy: undefined,
    disableCopyDataPolicy: undefined,
    maskingRulePolicy: undefined,
    maskingExceptionPolicy: undefined,
    enforce: false,
    resourceType: 0,
    resourceUid: "",
  };
}

export const Policy = {
  encode(message: Policy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.uid !== "") {
      writer.uint32(18).string(message.uid);
    }
    if (message.inheritFromParent === true) {
      writer.uint32(32).bool(message.inheritFromParent);
    }
    if (message.type !== 0) {
      writer.uint32(40).int32(message.type);
    }
    if (message.workspaceIamPolicy !== undefined) {
      IamPolicy.encode(message.workspaceIamPolicy, writer.uint32(50).fork()).ldelim();
    }
    if (message.deploymentApprovalPolicy !== undefined) {
      DeploymentApprovalPolicy.encode(message.deploymentApprovalPolicy, writer.uint32(58).fork()).ldelim();
    }
    if (message.rolloutPolicy !== undefined) {
      RolloutPolicy.encode(message.rolloutPolicy, writer.uint32(154).fork()).ldelim();
    }
    if (message.backupPlanPolicy !== undefined) {
      BackupPlanPolicy.encode(message.backupPlanPolicy, writer.uint32(66).fork()).ldelim();
    }
    if (message.maskingPolicy !== undefined) {
      MaskingPolicy.encode(message.maskingPolicy, writer.uint32(74).fork()).ldelim();
    }
    if (message.sqlReviewPolicy !== undefined) {
      SQLReviewPolicy.encode(message.sqlReviewPolicy, writer.uint32(90).fork()).ldelim();
    }
    if (message.slowQueryPolicy !== undefined) {
      SlowQueryPolicy.encode(message.slowQueryPolicy, writer.uint32(98).fork()).ldelim();
    }
    if (message.disableCopyDataPolicy !== undefined) {
      DisableCopyDataPolicy.encode(message.disableCopyDataPolicy, writer.uint32(130).fork()).ldelim();
    }
    if (message.maskingRulePolicy !== undefined) {
      MaskingRulePolicy.encode(message.maskingRulePolicy, writer.uint32(138).fork()).ldelim();
    }
    if (message.maskingExceptionPolicy !== undefined) {
      MaskingExceptionPolicy.encode(message.maskingExceptionPolicy, writer.uint32(146).fork()).ldelim();
    }
    if (message.enforce === true) {
      writer.uint32(104).bool(message.enforce);
    }
    if (message.resourceType !== 0) {
      writer.uint32(112).int32(message.resourceType);
    }
    if (message.resourceUid !== "") {
      writer.uint32(122).string(message.resourceUid);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Policy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.uid = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.inheritFromParent = reader.bool();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.type = reader.int32() as any;
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.workspaceIamPolicy = IamPolicy.decode(reader, reader.uint32());
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.deploymentApprovalPolicy = DeploymentApprovalPolicy.decode(reader, reader.uint32());
          continue;
        case 19:
          if (tag !== 154) {
            break;
          }

          message.rolloutPolicy = RolloutPolicy.decode(reader, reader.uint32());
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.backupPlanPolicy = BackupPlanPolicy.decode(reader, reader.uint32());
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.maskingPolicy = MaskingPolicy.decode(reader, reader.uint32());
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.sqlReviewPolicy = SQLReviewPolicy.decode(reader, reader.uint32());
          continue;
        case 12:
          if (tag !== 98) {
            break;
          }

          message.slowQueryPolicy = SlowQueryPolicy.decode(reader, reader.uint32());
          continue;
        case 16:
          if (tag !== 130) {
            break;
          }

          message.disableCopyDataPolicy = DisableCopyDataPolicy.decode(reader, reader.uint32());
          continue;
        case 17:
          if (tag !== 138) {
            break;
          }

          message.maskingRulePolicy = MaskingRulePolicy.decode(reader, reader.uint32());
          continue;
        case 18:
          if (tag !== 146) {
            break;
          }

          message.maskingExceptionPolicy = MaskingExceptionPolicy.decode(reader, reader.uint32());
          continue;
        case 13:
          if (tag !== 104) {
            break;
          }

          message.enforce = reader.bool();
          continue;
        case 14:
          if (tag !== 112) {
            break;
          }

          message.resourceType = reader.int32() as any;
          continue;
        case 15:
          if (tag !== 122) {
            break;
          }

          message.resourceUid = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Policy {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      uid: isSet(object.uid) ? String(object.uid) : "",
      inheritFromParent: isSet(object.inheritFromParent) ? Boolean(object.inheritFromParent) : false,
      type: isSet(object.type) ? policyTypeFromJSON(object.type) : 0,
      workspaceIamPolicy: isSet(object.workspaceIamPolicy) ? IamPolicy.fromJSON(object.workspaceIamPolicy) : undefined,
      deploymentApprovalPolicy: isSet(object.deploymentApprovalPolicy)
        ? DeploymentApprovalPolicy.fromJSON(object.deploymentApprovalPolicy)
        : undefined,
      rolloutPolicy: isSet(object.rolloutPolicy) ? RolloutPolicy.fromJSON(object.rolloutPolicy) : undefined,
      backupPlanPolicy: isSet(object.backupPlanPolicy) ? BackupPlanPolicy.fromJSON(object.backupPlanPolicy) : undefined,
      maskingPolicy: isSet(object.maskingPolicy) ? MaskingPolicy.fromJSON(object.maskingPolicy) : undefined,
      sqlReviewPolicy: isSet(object.sqlReviewPolicy) ? SQLReviewPolicy.fromJSON(object.sqlReviewPolicy) : undefined,
      slowQueryPolicy: isSet(object.slowQueryPolicy) ? SlowQueryPolicy.fromJSON(object.slowQueryPolicy) : undefined,
      disableCopyDataPolicy: isSet(object.disableCopyDataPolicy)
        ? DisableCopyDataPolicy.fromJSON(object.disableCopyDataPolicy)
        : undefined,
      maskingRulePolicy: isSet(object.maskingRulePolicy)
        ? MaskingRulePolicy.fromJSON(object.maskingRulePolicy)
        : undefined,
      maskingExceptionPolicy: isSet(object.maskingExceptionPolicy)
        ? MaskingExceptionPolicy.fromJSON(object.maskingExceptionPolicy)
        : undefined,
      enforce: isSet(object.enforce) ? Boolean(object.enforce) : false,
      resourceType: isSet(object.resourceType) ? policyResourceTypeFromJSON(object.resourceType) : 0,
      resourceUid: isSet(object.resourceUid) ? String(object.resourceUid) : "",
    };
  },

  toJSON(message: Policy): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.inheritFromParent !== undefined && (obj.inheritFromParent = message.inheritFromParent);
    message.type !== undefined && (obj.type = policyTypeToJSON(message.type));
    message.workspaceIamPolicy !== undefined &&
      (obj.workspaceIamPolicy = message.workspaceIamPolicy ? IamPolicy.toJSON(message.workspaceIamPolicy) : undefined);
    message.deploymentApprovalPolicy !== undefined && (obj.deploymentApprovalPolicy = message.deploymentApprovalPolicy
      ? DeploymentApprovalPolicy.toJSON(message.deploymentApprovalPolicy)
      : undefined);
    message.rolloutPolicy !== undefined &&
      (obj.rolloutPolicy = message.rolloutPolicy ? RolloutPolicy.toJSON(message.rolloutPolicy) : undefined);
    message.backupPlanPolicy !== undefined &&
      (obj.backupPlanPolicy = message.backupPlanPolicy ? BackupPlanPolicy.toJSON(message.backupPlanPolicy) : undefined);
    message.maskingPolicy !== undefined &&
      (obj.maskingPolicy = message.maskingPolicy ? MaskingPolicy.toJSON(message.maskingPolicy) : undefined);
    message.sqlReviewPolicy !== undefined &&
      (obj.sqlReviewPolicy = message.sqlReviewPolicy ? SQLReviewPolicy.toJSON(message.sqlReviewPolicy) : undefined);
    message.slowQueryPolicy !== undefined &&
      (obj.slowQueryPolicy = message.slowQueryPolicy ? SlowQueryPolicy.toJSON(message.slowQueryPolicy) : undefined);
    message.disableCopyDataPolicy !== undefined && (obj.disableCopyDataPolicy = message.disableCopyDataPolicy
      ? DisableCopyDataPolicy.toJSON(message.disableCopyDataPolicy)
      : undefined);
    message.maskingRulePolicy !== undefined && (obj.maskingRulePolicy = message.maskingRulePolicy
      ? MaskingRulePolicy.toJSON(message.maskingRulePolicy)
      : undefined);
    message.maskingExceptionPolicy !== undefined && (obj.maskingExceptionPolicy = message.maskingExceptionPolicy
      ? MaskingExceptionPolicy.toJSON(message.maskingExceptionPolicy)
      : undefined);
    message.enforce !== undefined && (obj.enforce = message.enforce);
    message.resourceType !== undefined && (obj.resourceType = policyResourceTypeToJSON(message.resourceType));
    message.resourceUid !== undefined && (obj.resourceUid = message.resourceUid);
    return obj;
  },

  create(base?: DeepPartial<Policy>): Policy {
    return Policy.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Policy>): Policy {
    const message = createBasePolicy();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.inheritFromParent = object.inheritFromParent ?? false;
    message.type = object.type ?? 0;
    message.workspaceIamPolicy = (object.workspaceIamPolicy !== undefined && object.workspaceIamPolicy !== null)
      ? IamPolicy.fromPartial(object.workspaceIamPolicy)
      : undefined;
    message.deploymentApprovalPolicy =
      (object.deploymentApprovalPolicy !== undefined && object.deploymentApprovalPolicy !== null)
        ? DeploymentApprovalPolicy.fromPartial(object.deploymentApprovalPolicy)
        : undefined;
    message.rolloutPolicy = (object.rolloutPolicy !== undefined && object.rolloutPolicy !== null)
      ? RolloutPolicy.fromPartial(object.rolloutPolicy)
      : undefined;
    message.backupPlanPolicy = (object.backupPlanPolicy !== undefined && object.backupPlanPolicy !== null)
      ? BackupPlanPolicy.fromPartial(object.backupPlanPolicy)
      : undefined;
    message.maskingPolicy = (object.maskingPolicy !== undefined && object.maskingPolicy !== null)
      ? MaskingPolicy.fromPartial(object.maskingPolicy)
      : undefined;
    message.sqlReviewPolicy = (object.sqlReviewPolicy !== undefined && object.sqlReviewPolicy !== null)
      ? SQLReviewPolicy.fromPartial(object.sqlReviewPolicy)
      : undefined;
    message.slowQueryPolicy = (object.slowQueryPolicy !== undefined && object.slowQueryPolicy !== null)
      ? SlowQueryPolicy.fromPartial(object.slowQueryPolicy)
      : undefined;
    message.disableCopyDataPolicy =
      (object.disableCopyDataPolicy !== undefined && object.disableCopyDataPolicy !== null)
        ? DisableCopyDataPolicy.fromPartial(object.disableCopyDataPolicy)
        : undefined;
    message.maskingRulePolicy = (object.maskingRulePolicy !== undefined && object.maskingRulePolicy !== null)
      ? MaskingRulePolicy.fromPartial(object.maskingRulePolicy)
      : undefined;
    message.maskingExceptionPolicy =
      (object.maskingExceptionPolicy !== undefined && object.maskingExceptionPolicy !== null)
        ? MaskingExceptionPolicy.fromPartial(object.maskingExceptionPolicy)
        : undefined;
    message.enforce = object.enforce ?? false;
    message.resourceType = object.resourceType ?? 0;
    message.resourceUid = object.resourceUid ?? "";
    return message;
  },
};

function createBaseDeploymentApprovalPolicy(): DeploymentApprovalPolicy {
  return { defaultStrategy: 0, deploymentApprovalStrategies: [] };
}

export const DeploymentApprovalPolicy = {
  encode(message: DeploymentApprovalPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.defaultStrategy !== 0) {
      writer.uint32(8).int32(message.defaultStrategy);
    }
    for (const v of message.deploymentApprovalStrategies) {
      DeploymentApprovalStrategy.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeploymentApprovalPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeploymentApprovalPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.defaultStrategy = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.deploymentApprovalStrategies.push(DeploymentApprovalStrategy.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DeploymentApprovalPolicy {
    return {
      defaultStrategy: isSet(object.defaultStrategy) ? approvalStrategyFromJSON(object.defaultStrategy) : 0,
      deploymentApprovalStrategies: Array.isArray(object?.deploymentApprovalStrategies)
        ? object.deploymentApprovalStrategies.map((e: any) => DeploymentApprovalStrategy.fromJSON(e))
        : [],
    };
  },

  toJSON(message: DeploymentApprovalPolicy): unknown {
    const obj: any = {};
    message.defaultStrategy !== undefined && (obj.defaultStrategy = approvalStrategyToJSON(message.defaultStrategy));
    if (message.deploymentApprovalStrategies) {
      obj.deploymentApprovalStrategies = message.deploymentApprovalStrategies.map((e) =>
        e ? DeploymentApprovalStrategy.toJSON(e) : undefined
      );
    } else {
      obj.deploymentApprovalStrategies = [];
    }
    return obj;
  },

  create(base?: DeepPartial<DeploymentApprovalPolicy>): DeploymentApprovalPolicy {
    return DeploymentApprovalPolicy.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<DeploymentApprovalPolicy>): DeploymentApprovalPolicy {
    const message = createBaseDeploymentApprovalPolicy();
    message.defaultStrategy = object.defaultStrategy ?? 0;
    message.deploymentApprovalStrategies =
      object.deploymentApprovalStrategies?.map((e) => DeploymentApprovalStrategy.fromPartial(e)) || [];
    return message;
  },
};

function createBaseRolloutPolicy(): RolloutPolicy {
  return { automatic: false, workspaceRoles: [], projectRoles: [], issueRoles: [] };
}

export const RolloutPolicy = {
  encode(message: RolloutPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.automatic === true) {
      writer.uint32(8).bool(message.automatic);
    }
    for (const v of message.workspaceRoles) {
      writer.uint32(18).string(v!);
    }
    for (const v of message.projectRoles) {
      writer.uint32(26).string(v!);
    }
    for (const v of message.issueRoles) {
      writer.uint32(34).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RolloutPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRolloutPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.automatic = reader.bool();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.workspaceRoles.push(reader.string());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.projectRoles.push(reader.string());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.issueRoles.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): RolloutPolicy {
    return {
      automatic: isSet(object.automatic) ? Boolean(object.automatic) : false,
      workspaceRoles: Array.isArray(object?.workspaceRoles) ? object.workspaceRoles.map((e: any) => String(e)) : [],
      projectRoles: Array.isArray(object?.projectRoles) ? object.projectRoles.map((e: any) => String(e)) : [],
      issueRoles: Array.isArray(object?.issueRoles) ? object.issueRoles.map((e: any) => String(e)) : [],
    };
  },

  toJSON(message: RolloutPolicy): unknown {
    const obj: any = {};
    message.automatic !== undefined && (obj.automatic = message.automatic);
    if (message.workspaceRoles) {
      obj.workspaceRoles = message.workspaceRoles.map((e) => e);
    } else {
      obj.workspaceRoles = [];
    }
    if (message.projectRoles) {
      obj.projectRoles = message.projectRoles.map((e) => e);
    } else {
      obj.projectRoles = [];
    }
    if (message.issueRoles) {
      obj.issueRoles = message.issueRoles.map((e) => e);
    } else {
      obj.issueRoles = [];
    }
    return obj;
  },

  create(base?: DeepPartial<RolloutPolicy>): RolloutPolicy {
    return RolloutPolicy.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<RolloutPolicy>): RolloutPolicy {
    const message = createBaseRolloutPolicy();
    message.automatic = object.automatic ?? false;
    message.workspaceRoles = object.workspaceRoles?.map((e) => e) || [];
    message.projectRoles = object.projectRoles?.map((e) => e) || [];
    message.issueRoles = object.issueRoles?.map((e) => e) || [];
    return message;
  },
};

function createBaseDeploymentApprovalStrategy(): DeploymentApprovalStrategy {
  return { deploymentType: 0, approvalGroup: 0, approvalStrategy: 0 };
}

export const DeploymentApprovalStrategy = {
  encode(message: DeploymentApprovalStrategy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.deploymentType !== 0) {
      writer.uint32(8).int32(message.deploymentType);
    }
    if (message.approvalGroup !== 0) {
      writer.uint32(16).int32(message.approvalGroup);
    }
    if (message.approvalStrategy !== 0) {
      writer.uint32(24).int32(message.approvalStrategy);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeploymentApprovalStrategy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeploymentApprovalStrategy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.deploymentType = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.approvalGroup = reader.int32() as any;
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.approvalStrategy = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DeploymentApprovalStrategy {
    return {
      deploymentType: isSet(object.deploymentType) ? deploymentTypeFromJSON(object.deploymentType) : 0,
      approvalGroup: isSet(object.approvalGroup) ? approvalGroupFromJSON(object.approvalGroup) : 0,
      approvalStrategy: isSet(object.approvalStrategy) ? approvalStrategyFromJSON(object.approvalStrategy) : 0,
    };
  },

  toJSON(message: DeploymentApprovalStrategy): unknown {
    const obj: any = {};
    message.deploymentType !== undefined && (obj.deploymentType = deploymentTypeToJSON(message.deploymentType));
    message.approvalGroup !== undefined && (obj.approvalGroup = approvalGroupToJSON(message.approvalGroup));
    message.approvalStrategy !== undefined && (obj.approvalStrategy = approvalStrategyToJSON(message.approvalStrategy));
    return obj;
  },

  create(base?: DeepPartial<DeploymentApprovalStrategy>): DeploymentApprovalStrategy {
    return DeploymentApprovalStrategy.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<DeploymentApprovalStrategy>): DeploymentApprovalStrategy {
    const message = createBaseDeploymentApprovalStrategy();
    message.deploymentType = object.deploymentType ?? 0;
    message.approvalGroup = object.approvalGroup ?? 0;
    message.approvalStrategy = object.approvalStrategy ?? 0;
    return message;
  },
};

function createBaseBackupPlanPolicy(): BackupPlanPolicy {
  return { schedule: 0, retentionDuration: undefined };
}

export const BackupPlanPolicy = {
  encode(message: BackupPlanPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schedule !== 0) {
      writer.uint32(8).int32(message.schedule);
    }
    if (message.retentionDuration !== undefined) {
      Duration.encode(message.retentionDuration, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BackupPlanPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBackupPlanPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.schedule = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.retentionDuration = Duration.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BackupPlanPolicy {
    return {
      schedule: isSet(object.schedule) ? backupPlanScheduleFromJSON(object.schedule) : 0,
      retentionDuration: isSet(object.retentionDuration) ? Duration.fromJSON(object.retentionDuration) : undefined,
    };
  },

  toJSON(message: BackupPlanPolicy): unknown {
    const obj: any = {};
    message.schedule !== undefined && (obj.schedule = backupPlanScheduleToJSON(message.schedule));
    message.retentionDuration !== undefined &&
      (obj.retentionDuration = message.retentionDuration ? Duration.toJSON(message.retentionDuration) : undefined);
    return obj;
  },

  create(base?: DeepPartial<BackupPlanPolicy>): BackupPlanPolicy {
    return BackupPlanPolicy.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<BackupPlanPolicy>): BackupPlanPolicy {
    const message = createBaseBackupPlanPolicy();
    message.schedule = object.schedule ?? 0;
    message.retentionDuration = (object.retentionDuration !== undefined && object.retentionDuration !== null)
      ? Duration.fromPartial(object.retentionDuration)
      : undefined;
    return message;
  },
};

function createBaseSlowQueryPolicy(): SlowQueryPolicy {
  return { active: false };
}

export const SlowQueryPolicy = {
  encode(message: SlowQueryPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.active === true) {
      writer.uint32(8).bool(message.active);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SlowQueryPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSlowQueryPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.active = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SlowQueryPolicy {
    return { active: isSet(object.active) ? Boolean(object.active) : false };
  },

  toJSON(message: SlowQueryPolicy): unknown {
    const obj: any = {};
    message.active !== undefined && (obj.active = message.active);
    return obj;
  },

  create(base?: DeepPartial<SlowQueryPolicy>): SlowQueryPolicy {
    return SlowQueryPolicy.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SlowQueryPolicy>): SlowQueryPolicy {
    const message = createBaseSlowQueryPolicy();
    message.active = object.active ?? false;
    return message;
  },
};

function createBaseDisableCopyDataPolicy(): DisableCopyDataPolicy {
  return { active: false };
}

export const DisableCopyDataPolicy = {
  encode(message: DisableCopyDataPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.active === true) {
      writer.uint32(8).bool(message.active);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DisableCopyDataPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDisableCopyDataPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.active = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DisableCopyDataPolicy {
    return { active: isSet(object.active) ? Boolean(object.active) : false };
  },

  toJSON(message: DisableCopyDataPolicy): unknown {
    const obj: any = {};
    message.active !== undefined && (obj.active = message.active);
    return obj;
  },

  create(base?: DeepPartial<DisableCopyDataPolicy>): DisableCopyDataPolicy {
    return DisableCopyDataPolicy.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<DisableCopyDataPolicy>): DisableCopyDataPolicy {
    const message = createBaseDisableCopyDataPolicy();
    message.active = object.active ?? false;
    return message;
  },
};

function createBaseMaskingPolicy(): MaskingPolicy {
  return { maskData: [] };
}

export const MaskingPolicy = {
  encode(message: MaskingPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.maskData) {
      MaskData.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MaskingPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMaskingPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.maskData.push(MaskData.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MaskingPolicy {
    return { maskData: Array.isArray(object?.maskData) ? object.maskData.map((e: any) => MaskData.fromJSON(e)) : [] };
  },

  toJSON(message: MaskingPolicy): unknown {
    const obj: any = {};
    if (message.maskData) {
      obj.maskData = message.maskData.map((e) => e ? MaskData.toJSON(e) : undefined);
    } else {
      obj.maskData = [];
    }
    return obj;
  },

  create(base?: DeepPartial<MaskingPolicy>): MaskingPolicy {
    return MaskingPolicy.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<MaskingPolicy>): MaskingPolicy {
    const message = createBaseMaskingPolicy();
    message.maskData = object.maskData?.map((e) => MaskData.fromPartial(e)) || [];
    return message;
  },
};

function createBaseMaskData(): MaskData {
  return {
    schema: "",
    table: "",
    column: "",
    maskingLevel: 0,
    fullMaskingAlgorithmId: "",
    partialMaskingAlgorithmId: "",
  };
}

export const MaskData = {
  encode(message: MaskData, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schema !== "") {
      writer.uint32(10).string(message.schema);
    }
    if (message.table !== "") {
      writer.uint32(18).string(message.table);
    }
    if (message.column !== "") {
      writer.uint32(26).string(message.column);
    }
    if (message.maskingLevel !== 0) {
      writer.uint32(32).int32(message.maskingLevel);
    }
    if (message.fullMaskingAlgorithmId !== "") {
      writer.uint32(42).string(message.fullMaskingAlgorithmId);
    }
    if (message.partialMaskingAlgorithmId !== "") {
      writer.uint32(50).string(message.partialMaskingAlgorithmId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MaskData {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMaskData();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.schema = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.table = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.column = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.maskingLevel = reader.int32() as any;
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.fullMaskingAlgorithmId = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.partialMaskingAlgorithmId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MaskData {
    return {
      schema: isSet(object.schema) ? String(object.schema) : "",
      table: isSet(object.table) ? String(object.table) : "",
      column: isSet(object.column) ? String(object.column) : "",
      maskingLevel: isSet(object.maskingLevel) ? maskingLevelFromJSON(object.maskingLevel) : 0,
      fullMaskingAlgorithmId: isSet(object.fullMaskingAlgorithmId) ? String(object.fullMaskingAlgorithmId) : "",
      partialMaskingAlgorithmId: isSet(object.partialMaskingAlgorithmId)
        ? String(object.partialMaskingAlgorithmId)
        : "",
    };
  },

  toJSON(message: MaskData): unknown {
    const obj: any = {};
    message.schema !== undefined && (obj.schema = message.schema);
    message.table !== undefined && (obj.table = message.table);
    message.column !== undefined && (obj.column = message.column);
    message.maskingLevel !== undefined && (obj.maskingLevel = maskingLevelToJSON(message.maskingLevel));
    message.fullMaskingAlgorithmId !== undefined && (obj.fullMaskingAlgorithmId = message.fullMaskingAlgorithmId);
    message.partialMaskingAlgorithmId !== undefined &&
      (obj.partialMaskingAlgorithmId = message.partialMaskingAlgorithmId);
    return obj;
  },

  create(base?: DeepPartial<MaskData>): MaskData {
    return MaskData.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<MaskData>): MaskData {
    const message = createBaseMaskData();
    message.schema = object.schema ?? "";
    message.table = object.table ?? "";
    message.column = object.column ?? "";
    message.maskingLevel = object.maskingLevel ?? 0;
    message.fullMaskingAlgorithmId = object.fullMaskingAlgorithmId ?? "";
    message.partialMaskingAlgorithmId = object.partialMaskingAlgorithmId ?? "";
    return message;
  },
};

function createBaseSQLReviewPolicy(): SQLReviewPolicy {
  return { name: "", rules: [] };
}

export const SQLReviewPolicy = {
  encode(message: SQLReviewPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.rules) {
      SQLReviewRule.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SQLReviewPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSQLReviewPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.rules.push(SQLReviewRule.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SQLReviewPolicy {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      rules: Array.isArray(object?.rules) ? object.rules.map((e: any) => SQLReviewRule.fromJSON(e)) : [],
    };
  },

  toJSON(message: SQLReviewPolicy): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    if (message.rules) {
      obj.rules = message.rules.map((e) => e ? SQLReviewRule.toJSON(e) : undefined);
    } else {
      obj.rules = [];
    }
    return obj;
  },

  create(base?: DeepPartial<SQLReviewPolicy>): SQLReviewPolicy {
    return SQLReviewPolicy.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SQLReviewPolicy>): SQLReviewPolicy {
    const message = createBaseSQLReviewPolicy();
    message.name = object.name ?? "";
    message.rules = object.rules?.map((e) => SQLReviewRule.fromPartial(e)) || [];
    return message;
  },
};

function createBaseSQLReviewRule(): SQLReviewRule {
  return { type: "", level: 0, payload: "", engine: 0, comment: "" };
}

export const SQLReviewRule = {
  encode(message: SQLReviewRule, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.type !== "") {
      writer.uint32(10).string(message.type);
    }
    if (message.level !== 0) {
      writer.uint32(16).int32(message.level);
    }
    if (message.payload !== "") {
      writer.uint32(26).string(message.payload);
    }
    if (message.engine !== 0) {
      writer.uint32(32).int32(message.engine);
    }
    if (message.comment !== "") {
      writer.uint32(42).string(message.comment);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SQLReviewRule {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSQLReviewRule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.type = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.level = reader.int32() as any;
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.payload = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.engine = reader.int32() as any;
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.comment = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SQLReviewRule {
    return {
      type: isSet(object.type) ? String(object.type) : "",
      level: isSet(object.level) ? sQLReviewRuleLevelFromJSON(object.level) : 0,
      payload: isSet(object.payload) ? String(object.payload) : "",
      engine: isSet(object.engine) ? engineFromJSON(object.engine) : 0,
      comment: isSet(object.comment) ? String(object.comment) : "",
    };
  },

  toJSON(message: SQLReviewRule): unknown {
    const obj: any = {};
    message.type !== undefined && (obj.type = message.type);
    message.level !== undefined && (obj.level = sQLReviewRuleLevelToJSON(message.level));
    message.payload !== undefined && (obj.payload = message.payload);
    message.engine !== undefined && (obj.engine = engineToJSON(message.engine));
    message.comment !== undefined && (obj.comment = message.comment);
    return obj;
  },

  create(base?: DeepPartial<SQLReviewRule>): SQLReviewRule {
    return SQLReviewRule.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SQLReviewRule>): SQLReviewRule {
    const message = createBaseSQLReviewRule();
    message.type = object.type ?? "";
    message.level = object.level ?? 0;
    message.payload = object.payload ?? "";
    message.engine = object.engine ?? 0;
    message.comment = object.comment ?? "";
    return message;
  },
};

function createBaseMaskingExceptionPolicy(): MaskingExceptionPolicy {
  return { maskingExceptions: [] };
}

export const MaskingExceptionPolicy = {
  encode(message: MaskingExceptionPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.maskingExceptions) {
      MaskingExceptionPolicy_MaskingException.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MaskingExceptionPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMaskingExceptionPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.maskingExceptions.push(MaskingExceptionPolicy_MaskingException.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MaskingExceptionPolicy {
    return {
      maskingExceptions: Array.isArray(object?.maskingExceptions)
        ? object.maskingExceptions.map((e: any) => MaskingExceptionPolicy_MaskingException.fromJSON(e))
        : [],
    };
  },

  toJSON(message: MaskingExceptionPolicy): unknown {
    const obj: any = {};
    if (message.maskingExceptions) {
      obj.maskingExceptions = message.maskingExceptions.map((e) =>
        e ? MaskingExceptionPolicy_MaskingException.toJSON(e) : undefined
      );
    } else {
      obj.maskingExceptions = [];
    }
    return obj;
  },

  create(base?: DeepPartial<MaskingExceptionPolicy>): MaskingExceptionPolicy {
    return MaskingExceptionPolicy.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<MaskingExceptionPolicy>): MaskingExceptionPolicy {
    const message = createBaseMaskingExceptionPolicy();
    message.maskingExceptions =
      object.maskingExceptions?.map((e) => MaskingExceptionPolicy_MaskingException.fromPartial(e)) || [];
    return message;
  },
};

function createBaseMaskingExceptionPolicy_MaskingException(): MaskingExceptionPolicy_MaskingException {
  return { action: 0, maskingLevel: 0, member: "", condition: undefined };
}

export const MaskingExceptionPolicy_MaskingException = {
  encode(message: MaskingExceptionPolicy_MaskingException, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.action !== 0) {
      writer.uint32(8).int32(message.action);
    }
    if (message.maskingLevel !== 0) {
      writer.uint32(16).int32(message.maskingLevel);
    }
    if (message.member !== "") {
      writer.uint32(26).string(message.member);
    }
    if (message.condition !== undefined) {
      Expr.encode(message.condition, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MaskingExceptionPolicy_MaskingException {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMaskingExceptionPolicy_MaskingException();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.action = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.maskingLevel = reader.int32() as any;
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.member = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.condition = Expr.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MaskingExceptionPolicy_MaskingException {
    return {
      action: isSet(object.action) ? maskingExceptionPolicy_MaskingException_ActionFromJSON(object.action) : 0,
      maskingLevel: isSet(object.maskingLevel) ? maskingLevelFromJSON(object.maskingLevel) : 0,
      member: isSet(object.member) ? String(object.member) : "",
      condition: isSet(object.condition) ? Expr.fromJSON(object.condition) : undefined,
    };
  },

  toJSON(message: MaskingExceptionPolicy_MaskingException): unknown {
    const obj: any = {};
    message.action !== undefined && (obj.action = maskingExceptionPolicy_MaskingException_ActionToJSON(message.action));
    message.maskingLevel !== undefined && (obj.maskingLevel = maskingLevelToJSON(message.maskingLevel));
    message.member !== undefined && (obj.member = message.member);
    message.condition !== undefined && (obj.condition = message.condition ? Expr.toJSON(message.condition) : undefined);
    return obj;
  },

  create(base?: DeepPartial<MaskingExceptionPolicy_MaskingException>): MaskingExceptionPolicy_MaskingException {
    return MaskingExceptionPolicy_MaskingException.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<MaskingExceptionPolicy_MaskingException>): MaskingExceptionPolicy_MaskingException {
    const message = createBaseMaskingExceptionPolicy_MaskingException();
    message.action = object.action ?? 0;
    message.maskingLevel = object.maskingLevel ?? 0;
    message.member = object.member ?? "";
    message.condition = (object.condition !== undefined && object.condition !== null)
      ? Expr.fromPartial(object.condition)
      : undefined;
    return message;
  },
};

function createBaseMaskingRulePolicy(): MaskingRulePolicy {
  return { rules: [] };
}

export const MaskingRulePolicy = {
  encode(message: MaskingRulePolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.rules) {
      MaskingRulePolicy_MaskingRule.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MaskingRulePolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMaskingRulePolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.rules.push(MaskingRulePolicy_MaskingRule.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MaskingRulePolicy {
    return {
      rules: Array.isArray(object?.rules)
        ? object.rules.map((e: any) => MaskingRulePolicy_MaskingRule.fromJSON(e))
        : [],
    };
  },

  toJSON(message: MaskingRulePolicy): unknown {
    const obj: any = {};
    if (message.rules) {
      obj.rules = message.rules.map((e) => e ? MaskingRulePolicy_MaskingRule.toJSON(e) : undefined);
    } else {
      obj.rules = [];
    }
    return obj;
  },

  create(base?: DeepPartial<MaskingRulePolicy>): MaskingRulePolicy {
    return MaskingRulePolicy.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<MaskingRulePolicy>): MaskingRulePolicy {
    const message = createBaseMaskingRulePolicy();
    message.rules = object.rules?.map((e) => MaskingRulePolicy_MaskingRule.fromPartial(e)) || [];
    return message;
  },
};

function createBaseMaskingRulePolicy_MaskingRule(): MaskingRulePolicy_MaskingRule {
  return { id: "", condition: undefined, maskingLevel: 0 };
}

export const MaskingRulePolicy_MaskingRule = {
  encode(message: MaskingRulePolicy_MaskingRule, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.condition !== undefined) {
      Expr.encode(message.condition, writer.uint32(18).fork()).ldelim();
    }
    if (message.maskingLevel !== 0) {
      writer.uint32(24).int32(message.maskingLevel);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MaskingRulePolicy_MaskingRule {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMaskingRulePolicy_MaskingRule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.id = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.condition = Expr.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.maskingLevel = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MaskingRulePolicy_MaskingRule {
    return {
      id: isSet(object.id) ? String(object.id) : "",
      condition: isSet(object.condition) ? Expr.fromJSON(object.condition) : undefined,
      maskingLevel: isSet(object.maskingLevel) ? maskingLevelFromJSON(object.maskingLevel) : 0,
    };
  },

  toJSON(message: MaskingRulePolicy_MaskingRule): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.condition !== undefined && (obj.condition = message.condition ? Expr.toJSON(message.condition) : undefined);
    message.maskingLevel !== undefined && (obj.maskingLevel = maskingLevelToJSON(message.maskingLevel));
    return obj;
  },

  create(base?: DeepPartial<MaskingRulePolicy_MaskingRule>): MaskingRulePolicy_MaskingRule {
    return MaskingRulePolicy_MaskingRule.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<MaskingRulePolicy_MaskingRule>): MaskingRulePolicy_MaskingRule {
    const message = createBaseMaskingRulePolicy_MaskingRule();
    message.id = object.id ?? "";
    message.condition = (object.condition !== undefined && object.condition !== null)
      ? Expr.fromPartial(object.condition)
      : undefined;
    message.maskingLevel = object.maskingLevel ?? 0;
    return message;
  },
};

export type OrgPolicyServiceDefinition = typeof OrgPolicyServiceDefinition;
export const OrgPolicyServiceDefinition = {
  name: "OrgPolicyService",
  fullName: "bytebase.v1.OrgPolicyService",
  methods: {
    getPolicy: {
      name: "GetPolicy",
      requestType: GetPolicyRequest,
      requestStream: false,
      responseType: Policy,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              185,
              1,
              90,
              34,
              18,
              32,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              47,
              42,
              125,
              90,
              38,
              18,
              36,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              101,
              110,
              118,
              105,
              114,
              111,
              110,
              109,
              101,
              110,
              116,
              115,
              47,
              42,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              47,
              42,
              125,
              90,
              35,
              18,
              33,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
              47,
              42,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              47,
              42,
              125,
              90,
              47,
              18,
              45,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
              47,
              42,
              47,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              115,
              47,
              42,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              47,
              42,
              125,
              18,
              21,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    listPolicies: {
      name: "ListPolicies",
      requestType: ListPoliciesRequest,
      requestStream: false,
      responseType: ListPoliciesResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          578365826: [
            new Uint8Array([
              176,
              1,
              90,
              34,
              18,
              32,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              125,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              90,
              38,
              18,
              36,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              101,
              110,
              118,
              105,
              114,
              111,
              110,
              109,
              101,
              110,
              116,
              115,
              47,
              42,
              125,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              90,
              35,
              18,
              33,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
              47,
              42,
              125,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              90,
              47,
              18,
              45,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
              47,
              42,
              47,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              115,
              47,
              42,
              125,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              18,
              12,
              47,
              118,
              49,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
            ]),
          ],
        },
      },
    },
    createPolicy: {
      name: "CreatePolicy",
      requestType: CreatePolicyRequest,
      requestStream: false,
      responseType: Policy,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([13, 112, 97, 114, 101, 110, 116, 44, 112, 111, 108, 105, 99, 121])],
          578365826: [
            new Uint8Array([
              216,
              1,
              58,
              6,
              112,
              111,
              108,
              105,
              99,
              121,
              90,
              42,
              58,
              6,
              112,
              111,
              108,
              105,
              99,
              121,
              34,
              32,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              125,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              90,
              46,
              58,
              6,
              112,
              111,
              108,
              105,
              99,
              121,
              34,
              36,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              101,
              110,
              118,
              105,
              114,
              111,
              110,
              109,
              101,
              110,
              116,
              115,
              47,
              42,
              125,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              90,
              43,
              58,
              6,
              112,
              111,
              108,
              105,
              99,
              121,
              34,
              33,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
              47,
              42,
              125,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              90,
              55,
              58,
              6,
              112,
              111,
              108,
              105,
              99,
              121,
              34,
              45,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
              47,
              42,
              47,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              115,
              47,
              42,
              125,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              34,
              12,
              47,
              118,
              49,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
            ]),
          ],
        },
      },
    },
    updatePolicy: {
      name: "UpdatePolicy",
      requestType: UpdatePolicyRequest,
      requestStream: false,
      responseType: Policy,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([18, 112, 111, 108, 105, 99, 121, 44, 117, 112, 100, 97, 116, 101, 95, 109, 97, 115, 107]),
          ],
          578365826: [
            new Uint8Array([
              132,
              2,
              58,
              6,
              112,
              111,
              108,
              105,
              99,
              121,
              90,
              49,
              58,
              6,
              112,
              111,
              108,
              105,
              99,
              121,
              50,
              39,
              47,
              118,
              49,
              47,
              123,
              112,
              111,
              108,
              105,
              99,
              121,
              46,
              110,
              97,
              109,
              101,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              47,
              42,
              125,
              90,
              53,
              58,
              6,
              112,
              111,
              108,
              105,
              99,
              121,
              50,
              43,
              47,
              118,
              49,
              47,
              123,
              112,
              111,
              108,
              105,
              99,
              121,
              46,
              110,
              97,
              109,
              101,
              61,
              101,
              110,
              118,
              105,
              114,
              111,
              110,
              109,
              101,
              110,
              116,
              115,
              47,
              42,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              47,
              42,
              125,
              90,
              50,
              58,
              6,
              112,
              111,
              108,
              105,
              99,
              121,
              50,
              40,
              47,
              118,
              49,
              47,
              123,
              112,
              111,
              108,
              105,
              99,
              121,
              46,
              110,
              97,
              109,
              101,
              61,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
              47,
              42,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              47,
              42,
              125,
              90,
              62,
              58,
              6,
              112,
              111,
              108,
              105,
              99,
              121,
              50,
              52,
              47,
              118,
              49,
              47,
              123,
              112,
              111,
              108,
              105,
              99,
              121,
              46,
              110,
              97,
              109,
              101,
              61,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
              47,
              42,
              47,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              115,
              47,
              42,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              47,
              42,
              125,
              50,
              28,
              47,
              118,
              49,
              47,
              123,
              112,
              111,
              108,
              105,
              99,
              121,
              46,
              110,
              97,
              109,
              101,
              61,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    deletePolicy: {
      name: "DeletePolicy",
      requestType: DeletePolicyRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              185,
              1,
              90,
              34,
              42,
              32,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              47,
              42,
              125,
              90,
              38,
              42,
              36,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              101,
              110,
              118,
              105,
              114,
              111,
              110,
              109,
              101,
              110,
              116,
              115,
              47,
              42,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              47,
              42,
              125,
              90,
              35,
              42,
              33,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
              47,
              42,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              47,
              42,
              125,
              90,
              47,
              42,
              45,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              105,
              110,
              115,
              116,
              97,
              110,
              99,
              101,
              115,
              47,
              42,
              47,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              115,
              47,
              42,
              47,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              47,
              42,
              125,
              42,
              21,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              112,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
  },
} as const;

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
