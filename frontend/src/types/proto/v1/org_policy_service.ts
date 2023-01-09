/* eslint-disable */
import _m0 from "protobufjs/minimal";
import { Duration } from "../google/protobuf/duration";
import { State, stateFromJSON, stateToJSON } from "./common";
import { DeploymentType, deploymentTypeFromJSON, deploymentTypeToJSON } from "./deployment";

export const protobufPackage = "bytebase.v1";

export enum PolicyType {
  POLICY_TYPE_UNSPECIFIED = 0,
  DEPLOYMENT_APPROVAL = 1,
  BACKUP_PLAN = 2,
  SQL_REVIEW = 3,
  SENSITIVE_DATA = 4,
  ACCESS_CONTROL = 5,
  UNRECOGNIZED = -1,
}

export function policyTypeFromJSON(object: any): PolicyType {
  switch (object) {
    case 0:
    case "POLICY_TYPE_UNSPECIFIED":
      return PolicyType.POLICY_TYPE_UNSPECIFIED;
    case 1:
    case "DEPLOYMENT_APPROVAL":
      return PolicyType.DEPLOYMENT_APPROVAL;
    case 2:
    case "BACKUP_PLAN":
      return PolicyType.BACKUP_PLAN;
    case 3:
    case "SQL_REVIEW":
      return PolicyType.SQL_REVIEW;
    case 4:
    case "SENSITIVE_DATA":
      return PolicyType.SENSITIVE_DATA;
    case 5:
    case "ACCESS_CONTROL":
      return PolicyType.ACCESS_CONTROL;
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
    case PolicyType.DEPLOYMENT_APPROVAL:
      return "DEPLOYMENT_APPROVAL";
    case PolicyType.BACKUP_PLAN:
      return "BACKUP_PLAN";
    case PolicyType.SQL_REVIEW:
      return "SQL_REVIEW";
    case PolicyType.SENSITIVE_DATA:
      return "SENSITIVE_DATA";
    case PolicyType.ACCESS_CONTROL:
      return "ACCESS_CONTROL";
    case PolicyType.UNRECOGNIZED:
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

export enum SensitiveDataMaskType {
  MASK_TYPE_UNSPECIFIED = 0,
  DEFAULT = 1,
  UNRECOGNIZED = -1,
}

export function sensitiveDataMaskTypeFromJSON(object: any): SensitiveDataMaskType {
  switch (object) {
    case 0:
    case "MASK_TYPE_UNSPECIFIED":
      return SensitiveDataMaskType.MASK_TYPE_UNSPECIFIED;
    case 1:
    case "DEFAULT":
      return SensitiveDataMaskType.DEFAULT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return SensitiveDataMaskType.UNRECOGNIZED;
  }
}

export function sensitiveDataMaskTypeToJSON(object: SensitiveDataMaskType): string {
  switch (object) {
    case SensitiveDataMaskType.MASK_TYPE_UNSPECIFIED:
      return "MASK_TYPE_UNSPECIFIED";
    case SensitiveDataMaskType.DEFAULT:
      return "DEFAULT";
    case SensitiveDataMaskType.UNRECOGNIZED:
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
   * Instance resource name: environments/environment-id/instances/instance-id.
   * Database resource name: environments/environment-id/instances/instance-id/databases/database-name.
   */
  name: string;
  /** The system-assigned, unique identifier for a resource. */
  uid: string;
  state: State;
  inheritFromParent: boolean;
  type: PolicyType;
  deploymentApprovalPolicy?: DeploymentApprovalPolicy | undefined;
  backupPlanPolicy?: BackupPlanPolicy | undefined;
  sensitiveDataPolicy?: SensitiveDataPolicy | undefined;
  accessControlPolicy?: AccessControlPolicy | undefined;
  sqlReviewPolicy?: SQLReviewPolicy | undefined;
}

export interface DeploymentApprovalPolicy {
  defaultStrategy: ApprovalStrategy;
  deploymentApprovalStrategies: DeploymentApprovalStrategy[];
}

export interface DeploymentApprovalStrategy {
  deploymentType: DeploymentType;
  approvalGroup: ApprovalGroup;
  approvalStrategy: ApprovalStrategy;
}

export interface BackupPlanPolicy {
  schedule: BackupPlanSchedule;
  retentionDuration?: Duration;
}

export interface SensitiveDataPolicy {
  sensitiveData: SensitiveData[];
}

export interface SensitiveData {
  schema: string;
  table: string;
  column: string;
  maskType: SensitiveDataMaskType;
}

export interface AccessControlPolicy {
  disallowRules: AccessControlRule[];
}

export interface AccessControlRule {
  fullDatabase: boolean;
}

export interface SQLReviewPolicy {
  title: string;
  rules: SQLReviewRule[];
}

export interface SQLReviewRule {
  type: string;
  level: SQLReviewRuleLevel;
  payload: string;
}

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
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetPolicyRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
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

  fromPartial<I extends Exact<DeepPartial<GetPolicyRequest>, I>>(object: I): GetPolicyRequest {
    const message = createBaseGetPolicyRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListPoliciesRequest(): ListPoliciesRequest {
  return { parent: "", pageSize: 0, pageToken: "" };
}

export const ListPoliciesRequest = {
  encode(message: ListPoliciesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.pageSize !== 0) {
      writer.uint32(16).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(26).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListPoliciesRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListPoliciesRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.parent = reader.string();
          break;
        case 2:
          message.pageSize = reader.int32();
          break;
        case 3:
          message.pageToken = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ListPoliciesRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListPoliciesRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ListPoliciesRequest>, I>>(object: I): ListPoliciesRequest {
    const message = createBaseListPoliciesRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
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
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListPoliciesResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.policies.push(Policy.decode(reader, reader.uint32()));
          break;
        case 2:
          message.nextPageToken = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
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

  fromPartial<I extends Exact<DeepPartial<ListPoliciesResponse>, I>>(object: I): ListPoliciesResponse {
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
    state: 0,
    inheritFromParent: false,
    type: 0,
    deploymentApprovalPolicy: undefined,
    backupPlanPolicy: undefined,
    sensitiveDataPolicy: undefined,
    accessControlPolicy: undefined,
    sqlReviewPolicy: undefined,
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
    if (message.state !== 0) {
      writer.uint32(24).int32(message.state);
    }
    if (message.inheritFromParent === true) {
      writer.uint32(32).bool(message.inheritFromParent);
    }
    if (message.type !== 0) {
      writer.uint32(40).int32(message.type);
    }
    if (message.deploymentApprovalPolicy !== undefined) {
      DeploymentApprovalPolicy.encode(message.deploymentApprovalPolicy, writer.uint32(50).fork()).ldelim();
    }
    if (message.backupPlanPolicy !== undefined) {
      BackupPlanPolicy.encode(message.backupPlanPolicy, writer.uint32(58).fork()).ldelim();
    }
    if (message.sensitiveDataPolicy !== undefined) {
      SensitiveDataPolicy.encode(message.sensitiveDataPolicy, writer.uint32(66).fork()).ldelim();
    }
    if (message.accessControlPolicy !== undefined) {
      AccessControlPolicy.encode(message.accessControlPolicy, writer.uint32(74).fork()).ldelim();
    }
    if (message.sqlReviewPolicy !== undefined) {
      SQLReviewPolicy.encode(message.sqlReviewPolicy, writer.uint32(82).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Policy {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.uid = reader.string();
          break;
        case 3:
          message.state = reader.int32() as any;
          break;
        case 4:
          message.inheritFromParent = reader.bool();
          break;
        case 5:
          message.type = reader.int32() as any;
          break;
        case 6:
          message.deploymentApprovalPolicy = DeploymentApprovalPolicy.decode(reader, reader.uint32());
          break;
        case 7:
          message.backupPlanPolicy = BackupPlanPolicy.decode(reader, reader.uint32());
          break;
        case 8:
          message.sensitiveDataPolicy = SensitiveDataPolicy.decode(reader, reader.uint32());
          break;
        case 9:
          message.accessControlPolicy = AccessControlPolicy.decode(reader, reader.uint32());
          break;
        case 10:
          message.sqlReviewPolicy = SQLReviewPolicy.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Policy {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      uid: isSet(object.uid) ? String(object.uid) : "",
      state: isSet(object.state) ? stateFromJSON(object.state) : 0,
      inheritFromParent: isSet(object.inheritFromParent) ? Boolean(object.inheritFromParent) : false,
      type: isSet(object.type) ? policyTypeFromJSON(object.type) : 0,
      deploymentApprovalPolicy: isSet(object.deploymentApprovalPolicy)
        ? DeploymentApprovalPolicy.fromJSON(object.deploymentApprovalPolicy)
        : undefined,
      backupPlanPolicy: isSet(object.backupPlanPolicy) ? BackupPlanPolicy.fromJSON(object.backupPlanPolicy) : undefined,
      sensitiveDataPolicy: isSet(object.sensitiveDataPolicy)
        ? SensitiveDataPolicy.fromJSON(object.sensitiveDataPolicy)
        : undefined,
      accessControlPolicy: isSet(object.accessControlPolicy)
        ? AccessControlPolicy.fromJSON(object.accessControlPolicy)
        : undefined,
      sqlReviewPolicy: isSet(object.sqlReviewPolicy) ? SQLReviewPolicy.fromJSON(object.sqlReviewPolicy) : undefined,
    };
  },

  toJSON(message: Policy): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.state !== undefined && (obj.state = stateToJSON(message.state));
    message.inheritFromParent !== undefined && (obj.inheritFromParent = message.inheritFromParent);
    message.type !== undefined && (obj.type = policyTypeToJSON(message.type));
    message.deploymentApprovalPolicy !== undefined && (obj.deploymentApprovalPolicy = message.deploymentApprovalPolicy
      ? DeploymentApprovalPolicy.toJSON(message.deploymentApprovalPolicy)
      : undefined);
    message.backupPlanPolicy !== undefined &&
      (obj.backupPlanPolicy = message.backupPlanPolicy ? BackupPlanPolicy.toJSON(message.backupPlanPolicy) : undefined);
    message.sensitiveDataPolicy !== undefined && (obj.sensitiveDataPolicy = message.sensitiveDataPolicy
      ? SensitiveDataPolicy.toJSON(message.sensitiveDataPolicy)
      : undefined);
    message.accessControlPolicy !== undefined && (obj.accessControlPolicy = message.accessControlPolicy
      ? AccessControlPolicy.toJSON(message.accessControlPolicy)
      : undefined);
    message.sqlReviewPolicy !== undefined &&
      (obj.sqlReviewPolicy = message.sqlReviewPolicy ? SQLReviewPolicy.toJSON(message.sqlReviewPolicy) : undefined);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<Policy>, I>>(object: I): Policy {
    const message = createBasePolicy();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.state = object.state ?? 0;
    message.inheritFromParent = object.inheritFromParent ?? false;
    message.type = object.type ?? 0;
    message.deploymentApprovalPolicy =
      (object.deploymentApprovalPolicy !== undefined && object.deploymentApprovalPolicy !== null)
        ? DeploymentApprovalPolicy.fromPartial(object.deploymentApprovalPolicy)
        : undefined;
    message.backupPlanPolicy = (object.backupPlanPolicy !== undefined && object.backupPlanPolicy !== null)
      ? BackupPlanPolicy.fromPartial(object.backupPlanPolicy)
      : undefined;
    message.sensitiveDataPolicy = (object.sensitiveDataPolicy !== undefined && object.sensitiveDataPolicy !== null)
      ? SensitiveDataPolicy.fromPartial(object.sensitiveDataPolicy)
      : undefined;
    message.accessControlPolicy = (object.accessControlPolicy !== undefined && object.accessControlPolicy !== null)
      ? AccessControlPolicy.fromPartial(object.accessControlPolicy)
      : undefined;
    message.sqlReviewPolicy = (object.sqlReviewPolicy !== undefined && object.sqlReviewPolicy !== null)
      ? SQLReviewPolicy.fromPartial(object.sqlReviewPolicy)
      : undefined;
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
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeploymentApprovalPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.defaultStrategy = reader.int32() as any;
          break;
        case 2:
          message.deploymentApprovalStrategies.push(DeploymentApprovalStrategy.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
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

  fromPartial<I extends Exact<DeepPartial<DeploymentApprovalPolicy>, I>>(object: I): DeploymentApprovalPolicy {
    const message = createBaseDeploymentApprovalPolicy();
    message.defaultStrategy = object.defaultStrategy ?? 0;
    message.deploymentApprovalStrategies =
      object.deploymentApprovalStrategies?.map((e) => DeploymentApprovalStrategy.fromPartial(e)) || [];
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
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeploymentApprovalStrategy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.deploymentType = reader.int32() as any;
          break;
        case 2:
          message.approvalGroup = reader.int32() as any;
          break;
        case 3:
          message.approvalStrategy = reader.int32() as any;
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
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

  fromPartial<I extends Exact<DeepPartial<DeploymentApprovalStrategy>, I>>(object: I): DeploymentApprovalStrategy {
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
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBackupPlanPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.schedule = reader.int32() as any;
          break;
        case 2:
          message.retentionDuration = Duration.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
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

  fromPartial<I extends Exact<DeepPartial<BackupPlanPolicy>, I>>(object: I): BackupPlanPolicy {
    const message = createBaseBackupPlanPolicy();
    message.schedule = object.schedule ?? 0;
    message.retentionDuration = (object.retentionDuration !== undefined && object.retentionDuration !== null)
      ? Duration.fromPartial(object.retentionDuration)
      : undefined;
    return message;
  },
};

function createBaseSensitiveDataPolicy(): SensitiveDataPolicy {
  return { sensitiveData: [] };
}

export const SensitiveDataPolicy = {
  encode(message: SensitiveDataPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.sensitiveData) {
      SensitiveData.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SensitiveDataPolicy {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSensitiveDataPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sensitiveData.push(SensitiveData.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): SensitiveDataPolicy {
    return {
      sensitiveData: Array.isArray(object?.sensitiveData)
        ? object.sensitiveData.map((e: any) => SensitiveData.fromJSON(e))
        : [],
    };
  },

  toJSON(message: SensitiveDataPolicy): unknown {
    const obj: any = {};
    if (message.sensitiveData) {
      obj.sensitiveData = message.sensitiveData.map((e) => e ? SensitiveData.toJSON(e) : undefined);
    } else {
      obj.sensitiveData = [];
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<SensitiveDataPolicy>, I>>(object: I): SensitiveDataPolicy {
    const message = createBaseSensitiveDataPolicy();
    message.sensitiveData = object.sensitiveData?.map((e) => SensitiveData.fromPartial(e)) || [];
    return message;
  },
};

function createBaseSensitiveData(): SensitiveData {
  return { schema: "", table: "", column: "", maskType: 0 };
}

export const SensitiveData = {
  encode(message: SensitiveData, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schema !== "") {
      writer.uint32(10).string(message.schema);
    }
    if (message.table !== "") {
      writer.uint32(18).string(message.table);
    }
    if (message.column !== "") {
      writer.uint32(26).string(message.column);
    }
    if (message.maskType !== 0) {
      writer.uint32(32).int32(message.maskType);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SensitiveData {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSensitiveData();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.schema = reader.string();
          break;
        case 2:
          message.table = reader.string();
          break;
        case 3:
          message.column = reader.string();
          break;
        case 4:
          message.maskType = reader.int32() as any;
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): SensitiveData {
    return {
      schema: isSet(object.schema) ? String(object.schema) : "",
      table: isSet(object.table) ? String(object.table) : "",
      column: isSet(object.column) ? String(object.column) : "",
      maskType: isSet(object.maskType) ? sensitiveDataMaskTypeFromJSON(object.maskType) : 0,
    };
  },

  toJSON(message: SensitiveData): unknown {
    const obj: any = {};
    message.schema !== undefined && (obj.schema = message.schema);
    message.table !== undefined && (obj.table = message.table);
    message.column !== undefined && (obj.column = message.column);
    message.maskType !== undefined && (obj.maskType = sensitiveDataMaskTypeToJSON(message.maskType));
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<SensitiveData>, I>>(object: I): SensitiveData {
    const message = createBaseSensitiveData();
    message.schema = object.schema ?? "";
    message.table = object.table ?? "";
    message.column = object.column ?? "";
    message.maskType = object.maskType ?? 0;
    return message;
  },
};

function createBaseAccessControlPolicy(): AccessControlPolicy {
  return { disallowRules: [] };
}

export const AccessControlPolicy = {
  encode(message: AccessControlPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.disallowRules) {
      AccessControlRule.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): AccessControlPolicy {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAccessControlPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.disallowRules.push(AccessControlRule.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): AccessControlPolicy {
    return {
      disallowRules: Array.isArray(object?.disallowRules)
        ? object.disallowRules.map((e: any) => AccessControlRule.fromJSON(e))
        : [],
    };
  },

  toJSON(message: AccessControlPolicy): unknown {
    const obj: any = {};
    if (message.disallowRules) {
      obj.disallowRules = message.disallowRules.map((e) => e ? AccessControlRule.toJSON(e) : undefined);
    } else {
      obj.disallowRules = [];
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<AccessControlPolicy>, I>>(object: I): AccessControlPolicy {
    const message = createBaseAccessControlPolicy();
    message.disallowRules = object.disallowRules?.map((e) => AccessControlRule.fromPartial(e)) || [];
    return message;
  },
};

function createBaseAccessControlRule(): AccessControlRule {
  return { fullDatabase: false };
}

export const AccessControlRule = {
  encode(message: AccessControlRule, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.fullDatabase === true) {
      writer.uint32(8).bool(message.fullDatabase);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): AccessControlRule {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAccessControlRule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.fullDatabase = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): AccessControlRule {
    return { fullDatabase: isSet(object.fullDatabase) ? Boolean(object.fullDatabase) : false };
  },

  toJSON(message: AccessControlRule): unknown {
    const obj: any = {};
    message.fullDatabase !== undefined && (obj.fullDatabase = message.fullDatabase);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<AccessControlRule>, I>>(object: I): AccessControlRule {
    const message = createBaseAccessControlRule();
    message.fullDatabase = object.fullDatabase ?? false;
    return message;
  },
};

function createBaseSQLReviewPolicy(): SQLReviewPolicy {
  return { title: "", rules: [] };
}

export const SQLReviewPolicy = {
  encode(message: SQLReviewPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.title !== "") {
      writer.uint32(10).string(message.title);
    }
    for (const v of message.rules) {
      SQLReviewRule.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SQLReviewPolicy {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSQLReviewPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.title = reader.string();
          break;
        case 2:
          message.rules.push(SQLReviewRule.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): SQLReviewPolicy {
    return {
      title: isSet(object.title) ? String(object.title) : "",
      rules: Array.isArray(object?.rules) ? object.rules.map((e: any) => SQLReviewRule.fromJSON(e)) : [],
    };
  },

  toJSON(message: SQLReviewPolicy): unknown {
    const obj: any = {};
    message.title !== undefined && (obj.title = message.title);
    if (message.rules) {
      obj.rules = message.rules.map((e) => e ? SQLReviewRule.toJSON(e) : undefined);
    } else {
      obj.rules = [];
    }
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<SQLReviewPolicy>, I>>(object: I): SQLReviewPolicy {
    const message = createBaseSQLReviewPolicy();
    message.title = object.title ?? "";
    message.rules = object.rules?.map((e) => SQLReviewRule.fromPartial(e)) || [];
    return message;
  },
};

function createBaseSQLReviewRule(): SQLReviewRule {
  return { type: "", level: 0, payload: "" };
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
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SQLReviewRule {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSQLReviewRule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.type = reader.string();
          break;
        case 2:
          message.level = reader.int32() as any;
          break;
        case 3:
          message.payload = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): SQLReviewRule {
    return {
      type: isSet(object.type) ? String(object.type) : "",
      level: isSet(object.level) ? sQLReviewRuleLevelFromJSON(object.level) : 0,
      payload: isSet(object.payload) ? String(object.payload) : "",
    };
  },

  toJSON(message: SQLReviewRule): unknown {
    const obj: any = {};
    message.type !== undefined && (obj.type = message.type);
    message.level !== undefined && (obj.level = sQLReviewRuleLevelToJSON(message.level));
    message.payload !== undefined && (obj.payload = message.payload);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<SQLReviewRule>, I>>(object: I): SQLReviewRule {
    const message = createBaseSQLReviewRule();
    message.type = object.type ?? "";
    message.level = object.level ?? 0;
    message.payload = object.payload ?? "";
    return message;
  },
};

export interface OrgPolicyService {
  GetPolicy(request: GetPolicyRequest): Promise<Policy>;
  ListPolicies(request: ListPoliciesRequest): Promise<ListPoliciesResponse>;
}

export class OrgPolicyServiceClientImpl implements OrgPolicyService {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "bytebase.v1.OrgPolicyService";
    this.rpc = rpc;
    this.GetPolicy = this.GetPolicy.bind(this);
    this.ListPolicies = this.ListPolicies.bind(this);
  }
  GetPolicy(request: GetPolicyRequest): Promise<Policy> {
    const data = GetPolicyRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "GetPolicy", data);
    return promise.then((data) => Policy.decode(new _m0.Reader(data)));
  }

  ListPolicies(request: ListPoliciesRequest): Promise<ListPoliciesResponse> {
    const data = ListPoliciesRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "ListPolicies", data);
    return promise.then((data) => ListPoliciesResponse.decode(new _m0.Reader(data)));
  }
}

interface Rpc {
  request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

type KeysOfUnion<T> = T extends T ? keyof T : never;
export type Exact<P, I extends P> = P extends Builtin ? P
  : P & { [K in keyof P]: Exact<P[K], I[K]> } & { [K in Exclude<keyof I, KeysOfUnion<P>>]: never };

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
