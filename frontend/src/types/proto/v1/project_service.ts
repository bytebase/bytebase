/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { Timestamp } from "../google/protobuf/timestamp";
import { Activity_Type, activity_TypeFromJSON, activity_TypeToJSON } from "./activity_service";
import { State, stateFromJSON, stateToJSON } from "./common";

export const protobufPackage = "bytebase.v1";

export enum Workflow {
  WORKFLOW_UNSPECIFIED = 0,
  UI = 1,
  VCS = 2,
  UNRECOGNIZED = -1,
}

export function workflowFromJSON(object: any): Workflow {
  switch (object) {
    case 0:
    case "WORKFLOW_UNSPECIFIED":
      return Workflow.WORKFLOW_UNSPECIFIED;
    case 1:
    case "UI":
      return Workflow.UI;
    case 2:
    case "VCS":
      return Workflow.VCS;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Workflow.UNRECOGNIZED;
  }
}

export function workflowToJSON(object: Workflow): string {
  switch (object) {
    case Workflow.WORKFLOW_UNSPECIFIED:
      return "WORKFLOW_UNSPECIFIED";
    case Workflow.UI:
      return "UI";
    case Workflow.VCS:
      return "VCS";
    case Workflow.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum Visibility {
  VISIBILITY_UNSPECIFIED = 0,
  VISIBILITY_PUBLIC = 1,
  VISIBILITY_PRIVATE = 2,
  UNRECOGNIZED = -1,
}

export function visibilityFromJSON(object: any): Visibility {
  switch (object) {
    case 0:
    case "VISIBILITY_UNSPECIFIED":
      return Visibility.VISIBILITY_UNSPECIFIED;
    case 1:
    case "VISIBILITY_PUBLIC":
      return Visibility.VISIBILITY_PUBLIC;
    case 2:
    case "VISIBILITY_PRIVATE":
      return Visibility.VISIBILITY_PRIVATE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Visibility.UNRECOGNIZED;
  }
}

export function visibilityToJSON(object: Visibility): string {
  switch (object) {
    case Visibility.VISIBILITY_UNSPECIFIED:
      return "VISIBILITY_UNSPECIFIED";
    case Visibility.VISIBILITY_PUBLIC:
      return "VISIBILITY_PUBLIC";
    case Visibility.VISIBILITY_PRIVATE:
      return "VISIBILITY_PRIVATE";
    case Visibility.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum TenantMode {
  TENANT_MODE_UNSPECIFIED = 0,
  TENANT_MODE_DISABLED = 1,
  TENANT_MODE_ENABLED = 2,
  UNRECOGNIZED = -1,
}

export function tenantModeFromJSON(object: any): TenantMode {
  switch (object) {
    case 0:
    case "TENANT_MODE_UNSPECIFIED":
      return TenantMode.TENANT_MODE_UNSPECIFIED;
    case 1:
    case "TENANT_MODE_DISABLED":
      return TenantMode.TENANT_MODE_DISABLED;
    case 2:
    case "TENANT_MODE_ENABLED":
      return TenantMode.TENANT_MODE_ENABLED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return TenantMode.UNRECOGNIZED;
  }
}

export function tenantModeToJSON(object: TenantMode): string {
  switch (object) {
    case TenantMode.TENANT_MODE_UNSPECIFIED:
      return "TENANT_MODE_UNSPECIFIED";
    case TenantMode.TENANT_MODE_DISABLED:
      return "TENANT_MODE_DISABLED";
    case TenantMode.TENANT_MODE_ENABLED:
      return "TENANT_MODE_ENABLED";
    case TenantMode.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum SchemaVersion {
  SCHEMA_VERSION_UNSPECIFIED = 0,
  TIMESTAMP = 1,
  SEMANTIC = 2,
  UNRECOGNIZED = -1,
}

export function schemaVersionFromJSON(object: any): SchemaVersion {
  switch (object) {
    case 0:
    case "SCHEMA_VERSION_UNSPECIFIED":
      return SchemaVersion.SCHEMA_VERSION_UNSPECIFIED;
    case 1:
    case "TIMESTAMP":
      return SchemaVersion.TIMESTAMP;
    case 2:
    case "SEMANTIC":
      return SchemaVersion.SEMANTIC;
    case -1:
    case "UNRECOGNIZED":
    default:
      return SchemaVersion.UNRECOGNIZED;
  }
}

export function schemaVersionToJSON(object: SchemaVersion): string {
  switch (object) {
    case SchemaVersion.SCHEMA_VERSION_UNSPECIFIED:
      return "SCHEMA_VERSION_UNSPECIFIED";
    case SchemaVersion.TIMESTAMP:
      return "TIMESTAMP";
    case SchemaVersion.SEMANTIC:
      return "SEMANTIC";
    case SchemaVersion.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum SchemaChange {
  SCHEMA_CHANGE_UNSPECIFIED = 0,
  DDL = 1,
  SDL = 2,
  UNRECOGNIZED = -1,
}

export function schemaChangeFromJSON(object: any): SchemaChange {
  switch (object) {
    case 0:
    case "SCHEMA_CHANGE_UNSPECIFIED":
      return SchemaChange.SCHEMA_CHANGE_UNSPECIFIED;
    case 1:
    case "DDL":
      return SchemaChange.DDL;
    case 2:
    case "SDL":
      return SchemaChange.SDL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return SchemaChange.UNRECOGNIZED;
  }
}

export function schemaChangeToJSON(object: SchemaChange): string {
  switch (object) {
    case SchemaChange.SCHEMA_CHANGE_UNSPECIFIED:
      return "SCHEMA_CHANGE_UNSPECIFIED";
    case SchemaChange.DDL:
      return "DDL";
    case SchemaChange.SDL:
      return "SDL";
    case SchemaChange.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum LgtmCheck {
  LGTM_CHECK_UNSPECIFIED = 0,
  LGTM_CHECK_DISABLED = 1,
  LGTM_CHECK_PROJECT_OWNER = 2,
  LGTM_CHECK_PROJECT_MEMBER = 3,
  UNRECOGNIZED = -1,
}

export function lgtmCheckFromJSON(object: any): LgtmCheck {
  switch (object) {
    case 0:
    case "LGTM_CHECK_UNSPECIFIED":
      return LgtmCheck.LGTM_CHECK_UNSPECIFIED;
    case 1:
    case "LGTM_CHECK_DISABLED":
      return LgtmCheck.LGTM_CHECK_DISABLED;
    case 2:
    case "LGTM_CHECK_PROJECT_OWNER":
      return LgtmCheck.LGTM_CHECK_PROJECT_OWNER;
    case 3:
    case "LGTM_CHECK_PROJECT_MEMBER":
      return LgtmCheck.LGTM_CHECK_PROJECT_MEMBER;
    case -1:
    case "UNRECOGNIZED":
    default:
      return LgtmCheck.UNRECOGNIZED;
  }
}

export function lgtmCheckToJSON(object: LgtmCheck): string {
  switch (object) {
    case LgtmCheck.LGTM_CHECK_UNSPECIFIED:
      return "LGTM_CHECK_UNSPECIFIED";
    case LgtmCheck.LGTM_CHECK_DISABLED:
      return "LGTM_CHECK_DISABLED";
    case LgtmCheck.LGTM_CHECK_PROJECT_OWNER:
      return "LGTM_CHECK_PROJECT_OWNER";
    case LgtmCheck.LGTM_CHECK_PROJECT_MEMBER:
      return "LGTM_CHECK_PROJECT_MEMBER";
    case LgtmCheck.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum ProjectRole {
  PROJECT_ROLE_UNSPECIFIED = 0,
  PROJECT_ROLE_OWNER = 1,
  PROJECT_ROLE_DEVELOPER = 2,
  UNRECOGNIZED = -1,
}

export function projectRoleFromJSON(object: any): ProjectRole {
  switch (object) {
    case 0:
    case "PROJECT_ROLE_UNSPECIFIED":
      return ProjectRole.PROJECT_ROLE_UNSPECIFIED;
    case 1:
    case "PROJECT_ROLE_OWNER":
      return ProjectRole.PROJECT_ROLE_OWNER;
    case 2:
    case "PROJECT_ROLE_DEVELOPER":
      return ProjectRole.PROJECT_ROLE_DEVELOPER;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ProjectRole.UNRECOGNIZED;
  }
}

export function projectRoleToJSON(object: ProjectRole): string {
  switch (object) {
    case ProjectRole.PROJECT_ROLE_UNSPECIFIED:
      return "PROJECT_ROLE_UNSPECIFIED";
    case ProjectRole.PROJECT_ROLE_OWNER:
      return "PROJECT_ROLE_OWNER";
    case ProjectRole.PROJECT_ROLE_DEVELOPER:
      return "PROJECT_ROLE_DEVELOPER";
    case ProjectRole.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum ReviewStatus {
  REVIEW_STATUS_UNSPECIFIED = 0,
  OPEN = 1,
  DONE = 2,
  CANCELED = 3,
  UNRECOGNIZED = -1,
}

export function reviewStatusFromJSON(object: any): ReviewStatus {
  switch (object) {
    case 0:
    case "REVIEW_STATUS_UNSPECIFIED":
      return ReviewStatus.REVIEW_STATUS_UNSPECIFIED;
    case 1:
    case "OPEN":
      return ReviewStatus.OPEN;
    case 2:
    case "DONE":
      return ReviewStatus.DONE;
    case 3:
    case "CANCELED":
      return ReviewStatus.CANCELED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ReviewStatus.UNRECOGNIZED;
  }
}

export function reviewStatusToJSON(object: ReviewStatus): string {
  switch (object) {
    case ReviewStatus.REVIEW_STATUS_UNSPECIFIED:
      return "REVIEW_STATUS_UNSPECIFIED";
    case ReviewStatus.OPEN:
      return "OPEN";
    case ReviewStatus.DONE:
      return "DONE";
    case ReviewStatus.CANCELED:
      return "CANCELED";
    case ReviewStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum OperatorType {
  /** OPERATOR_TYPE_UNSPECIFIED - The operator is not specified. */
  OPERATOR_TYPE_UNSPECIFIED = 0,
  /** OPERATOR_TYPE_IN - The operator is "In". */
  OPERATOR_TYPE_IN = 1,
  /** OPERATOR_TYPE_EXISTS - The operator is "Exists". */
  OPERATOR_TYPE_EXISTS = 2,
  UNRECOGNIZED = -1,
}

export function operatorTypeFromJSON(object: any): OperatorType {
  switch (object) {
    case 0:
    case "OPERATOR_TYPE_UNSPECIFIED":
      return OperatorType.OPERATOR_TYPE_UNSPECIFIED;
    case 1:
    case "OPERATOR_TYPE_IN":
      return OperatorType.OPERATOR_TYPE_IN;
    case 2:
    case "OPERATOR_TYPE_EXISTS":
      return OperatorType.OPERATOR_TYPE_EXISTS;
    case -1:
    case "UNRECOGNIZED":
    default:
      return OperatorType.UNRECOGNIZED;
  }
}

export function operatorTypeToJSON(object: OperatorType): string {
  switch (object) {
    case OperatorType.OPERATOR_TYPE_UNSPECIFIED:
      return "OPERATOR_TYPE_UNSPECIFIED";
    case OperatorType.OPERATOR_TYPE_IN:
      return "OPERATOR_TYPE_IN";
    case OperatorType.OPERATOR_TYPE_EXISTS:
      return "OPERATOR_TYPE_EXISTS";
    case OperatorType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface GetProjectRequest {
  /**
   * The name of the project to retrieve.
   * Format: projects/{project}
   */
  name: string;
}

export interface ListProjectsRequest {
  /**
   * The maximum number of projects to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 projects will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListProjects` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListProjects` must match
   * the call that provided the page token.
   */
  pageToken: string;
  /** Show deleted projects if specified. */
  showDeleted: boolean;
}

export interface ListProjectsResponse {
  /** The projects from the specified request. */
  projects: Project[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface CreateProjectRequest {
  /** The project to create. */
  project?: Project;
  /**
   * The ID to use for the project, which will become the final component of
   * the project's resource name.
   *
   * This value should be 4-63 characters, and valid characters
   * are /[a-z][0-9]-/.
   */
  projectId: string;
}

export interface UpdateProjectRequest {
  /**
   * The project to update.
   *
   * The project's `name` field is used to identify the project to update.
   * Format: projects/{project}
   */
  project?: Project;
  /** The list of fields to update. */
  updateMask?: string[];
}

export interface DeleteProjectRequest {
  /**
   * The name of the project to delete.
   * Format: projects/{project}
   */
  name: string;
}

export interface UndeleteProjectRequest {
  /**
   * The name of the deleted project.
   * Format: projects/{project}
   */
  name: string;
}

export interface GetIamPolicyRequest {
  /**
   * The name of the project to get the IAM policy.
   * Format: projects/{project}
   */
  project: string;
}

export interface SetIamPolicyRequest {
  /**
   * The name of the project to set the IAM policy.
   * Format: projects/{project}
   */
  project: string;
  policy?: IamPolicy;
}

export interface GetDeploymentConfigRequest {
  /**
   * The name of the resource.
   * Format: projects/{project}/deploymentConfig
   */
  name: string;
}

export interface UpdateDeploymentConfigRequest {
  config?: DeploymentConfig;
}

export interface Project {
  /**
   * The name of the project.
   * Format: projects/{project}
   */
  name: string;
  /** The system-assigned, unique identifier for a resource. */
  uid: string;
  state: State;
  /** The title or name of a project. It's not unique within the workspace. */
  title: string;
  /** The key is a short and upper-case identifier for a project. It's unique within the workspace. */
  key: string;
  workflow: Workflow;
  visibility: Visibility;
  tenantMode: TenantMode;
  dbNameTemplate: string;
  schemaVersion: SchemaVersion;
  schemaChange: SchemaChange;
  lgtmCheck: LgtmCheck;
}

export interface IamPolicy {
  /**
   * Collection of binding.
   * A binding binds one or more project members to a single project role.
   */
  bindings: Binding[];
}

export interface Binding {
  /** The project role that is assigned to the members. */
  role: ProjectRole;
  /**
   * Specifies the principals requesting access for a Bytebase resource.
   * `members` can have the following values:
   *
   * * `user:{emailid}`: An email address that represents a specific Bytebase
   *    account. For example, `alice@example.com` .
   */
  members: string[];
}

export interface GetReviewRequest {
  /**
   * The name of the review to retrieve.
   * Format: projects/{project}/reviews/{review}
   */
  name: string;
}

export interface ListReviewsRequest {
  /**
   * The parent, which owns this collection of reviews.
   * Format: projects/{project}
   * Use "projects/-" to list all reviews from all projects.
   */
  parent: string;
  /**
   * The maximum number of reviews to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 reviews will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListReviews` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListReviews` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface ListReviewsResponse {
  /** The reviews from the specified request. */
  reviews: Review[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface UpdateReviewRequest {
  /**
   * The review to update.
   *
   * The review's `name` field is used to identify the review to update.
   * Format: projects/{project}/reviews/{review}
   */
  review?: Review;
  /** The list of fields to update. */
  updateMask?: string[];
}

export interface BatchUpdateReviewsRequest {
  /**
   * The parent resource shared by all reviews being updated.
   * Format: projects/{project}
   * If the operation spans parents, a dash (-) may be accepted as a wildcard.
   * We only support updating the status of databases for now.
   */
  parent: string;
  /**
   * The request message specifying the resources to update.
   * A maximum of 1000 databases can be modified in a batch.
   */
  requests: UpdateReviewRequest[];
}

export interface BatchUpdateReviewsResponse {
  /** Reviews updated. */
  reviews: Review[];
}

export interface ListWebhooksRequest {
  /**
   * The parent, which owns this collection of webhooks.
   * Format: projects/{project}
   */
  parent: string;
  /**
   * Not used. The maximum number of reviews to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 reviews will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * Not used. A page token, received from a previous `ListReviews` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListReviews` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface ListWebhooksResponse {
  /** The webhooks from the specified request. */
  webhooks: Webhook[];
  /**
   * Not used. A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface GetWebhookRequest {
  /**
   * The name of the webhook to retrieve.
   * Format: projects/{project}/webhooks/{webhook}
   */
  name: string;
}

export interface AddWebhookRequest {
  /**
   * The parent, which owns this collection of webhooks.
   * Format: projects/{project}
   */
  parent: string;
  /** The webhook to add. */
  webhook?: Webhook;
}

export interface ModifyWebhookRequest {
  /**
   * The webhook to modify.
   *
   * The webhook's `name` field is used to identify the webhook to modify.
   * Format: projects/{project}/webhooks/{webhook}
   */
  webhook?: Webhook;
  /** The list of fields to update. */
  updateMask?: string[];
}

export interface RemoveWebhookRequest {
  /**
   * The name of the webhook to remove.
   * Format: projects/{project}/webhooks/{webhook}
   */
  name: string;
}

export interface Webhook {
  /**
   * The name of the Webhook, generated by server. And it's unique within the project.
   * Format: projects/{project}/webhooks/{webhook}
   */
  name: string;
  /** type is the type of the webhook. */
  type: Webhook_Type;
  /** title is the title of the webhook. */
  title: string;
  /** url is the url of the webhook. */
  url: string;
  /**
   * sub_types is the list of activities types that the webhook is interested in.
   * It should not be empty, and shoule be a subset of the following:
   * - TYPE_ISSUE_CREATED
   * - TYPE_ISSUE_STATUS_UPDATE
   * - TYPE_ISSUE_PIPELINE_STAGE_UPDATE
   * - TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE
   * - TYPE_ISSUE_FIELD_UPDATE
   * - TYPE_ISSUE_COMMENT_CREAT
   */
  subTypes: Activity_Type[];
}

export enum Webhook_Type {
  TYPE_UNSPECIFIED = 0,
  TYPE_SLACK = 1,
  TYPE_DISCORD = 2,
  TYPE_TEAMS = 3,
  TYPE_DINGTALK = 4,
  TYPE_FEISHU = 5,
  TYPE_WECOM = 6,
  TYPE_CUSTOM = 7,
  UNRECOGNIZED = -1,
}

export function webhook_TypeFromJSON(object: any): Webhook_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return Webhook_Type.TYPE_UNSPECIFIED;
    case 1:
    case "TYPE_SLACK":
      return Webhook_Type.TYPE_SLACK;
    case 2:
    case "TYPE_DISCORD":
      return Webhook_Type.TYPE_DISCORD;
    case 3:
    case "TYPE_TEAMS":
      return Webhook_Type.TYPE_TEAMS;
    case 4:
    case "TYPE_DINGTALK":
      return Webhook_Type.TYPE_DINGTALK;
    case 5:
    case "TYPE_FEISHU":
      return Webhook_Type.TYPE_FEISHU;
    case 6:
    case "TYPE_WECOM":
      return Webhook_Type.TYPE_WECOM;
    case 7:
    case "TYPE_CUSTOM":
      return Webhook_Type.TYPE_CUSTOM;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Webhook_Type.UNRECOGNIZED;
  }
}

export function webhook_TypeToJSON(object: Webhook_Type): string {
  switch (object) {
    case Webhook_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case Webhook_Type.TYPE_SLACK:
      return "TYPE_SLACK";
    case Webhook_Type.TYPE_DISCORD:
      return "TYPE_DISCORD";
    case Webhook_Type.TYPE_TEAMS:
      return "TYPE_TEAMS";
    case Webhook_Type.TYPE_DINGTALK:
      return "TYPE_DINGTALK";
    case Webhook_Type.TYPE_FEISHU:
      return "TYPE_FEISHU";
    case Webhook_Type.TYPE_WECOM:
      return "TYPE_WECOM";
    case Webhook_Type.TYPE_CUSTOM:
      return "TYPE_CUSTOM";
    case Webhook_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface Review {
  /**
   * The name of the review.
   * `review` is a system generated ID.
   * Format: projects/{project}/reviews/{review}
   */
  name: string;
  /** The system-assigned, unique identifier for a resource. */
  uid: string;
  title: string;
  description: string;
  status: ReviewStatus;
  /** Format: user:hello@world.com */
  assignee: string;
  assigneeAttention: boolean;
  /**
   * The subscribers.
   * Format: user:hello@world.com
   */
  subscribers: string[];
  /** Format: user:hello@world.com */
  creator: string;
  createTime?: Date;
  updateTime?: Date;
}

export interface DeploymentConfig {
  /**
   * The name of the resource.
   * Format: projects/{project}/deploymentConfig
   */
  name: string;
  /** The title of the deployment config. */
  title: string;
  schedule?: Schedule;
}

export interface Schedule {
  deployments: ScheduleDeployment[];
}

export interface ScheduleDeployment {
  /** The title of the deployment (stage) in a schedule. */
  title: string;
  spec?: DeploymentSpec;
}

export interface DeploymentSpec {
  labelSelector?: LabelSelector;
}

export interface LabelSelector {
  matchExpressions: LabelSelectorRequirement[];
}

export interface LabelSelectorRequirement {
  key: string;
  operator: OperatorType;
  values: string[];
}

function createBaseGetProjectRequest(): GetProjectRequest {
  return { name: "" };
}

export const GetProjectRequest = {
  encode(message: GetProjectRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetProjectRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetProjectRequest();
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

  fromJSON(object: any): GetProjectRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetProjectRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<GetProjectRequest>): GetProjectRequest {
    const message = createBaseGetProjectRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListProjectsRequest(): ListProjectsRequest {
  return { pageSize: 0, pageToken: "", showDeleted: false };
}

export const ListProjectsRequest = {
  encode(message: ListProjectsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pageSize !== 0) {
      writer.uint32(8).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(18).string(message.pageToken);
    }
    if (message.showDeleted === true) {
      writer.uint32(24).bool(message.showDeleted);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListProjectsRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListProjectsRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pageSize = reader.int32();
          break;
        case 2:
          message.pageToken = reader.string();
          break;
        case 3:
          message.showDeleted = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ListProjectsRequest {
    return {
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
      showDeleted: isSet(object.showDeleted) ? Boolean(object.showDeleted) : false,
    };
  },

  toJSON(message: ListProjectsRequest): unknown {
    const obj: any = {};
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    message.showDeleted !== undefined && (obj.showDeleted = message.showDeleted);
    return obj;
  },

  fromPartial(object: DeepPartial<ListProjectsRequest>): ListProjectsRequest {
    const message = createBaseListProjectsRequest();
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    message.showDeleted = object.showDeleted ?? false;
    return message;
  },
};

function createBaseListProjectsResponse(): ListProjectsResponse {
  return { projects: [], nextPageToken: "" };
}

export const ListProjectsResponse = {
  encode(message: ListProjectsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.projects) {
      Project.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListProjectsResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListProjectsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.projects.push(Project.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListProjectsResponse {
    return {
      projects: Array.isArray(object?.projects) ? object.projects.map((e: any) => Project.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListProjectsResponse): unknown {
    const obj: any = {};
    if (message.projects) {
      obj.projects = message.projects.map((e) => e ? Project.toJSON(e) : undefined);
    } else {
      obj.projects = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  fromPartial(object: DeepPartial<ListProjectsResponse>): ListProjectsResponse {
    const message = createBaseListProjectsResponse();
    message.projects = object.projects?.map((e) => Project.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseCreateProjectRequest(): CreateProjectRequest {
  return { project: undefined, projectId: "" };
}

export const CreateProjectRequest = {
  encode(message: CreateProjectRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.project !== undefined) {
      Project.encode(message.project, writer.uint32(10).fork()).ldelim();
    }
    if (message.projectId !== "") {
      writer.uint32(18).string(message.projectId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateProjectRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateProjectRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.project = Project.decode(reader, reader.uint32());
          break;
        case 2:
          message.projectId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): CreateProjectRequest {
    return {
      project: isSet(object.project) ? Project.fromJSON(object.project) : undefined,
      projectId: isSet(object.projectId) ? String(object.projectId) : "",
    };
  },

  toJSON(message: CreateProjectRequest): unknown {
    const obj: any = {};
    message.project !== undefined && (obj.project = message.project ? Project.toJSON(message.project) : undefined);
    message.projectId !== undefined && (obj.projectId = message.projectId);
    return obj;
  },

  fromPartial(object: DeepPartial<CreateProjectRequest>): CreateProjectRequest {
    const message = createBaseCreateProjectRequest();
    message.project = (object.project !== undefined && object.project !== null)
      ? Project.fromPartial(object.project)
      : undefined;
    message.projectId = object.projectId ?? "";
    return message;
  },
};

function createBaseUpdateProjectRequest(): UpdateProjectRequest {
  return { project: undefined, updateMask: undefined };
}

export const UpdateProjectRequest = {
  encode(message: UpdateProjectRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.project !== undefined) {
      Project.encode(message.project, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateProjectRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateProjectRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.project = Project.decode(reader, reader.uint32());
          break;
        case 2:
          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): UpdateProjectRequest {
    return {
      project: isSet(object.project) ? Project.fromJSON(object.project) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateProjectRequest): unknown {
    const obj: any = {};
    message.project !== undefined && (obj.project = message.project ? Project.toJSON(message.project) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  fromPartial(object: DeepPartial<UpdateProjectRequest>): UpdateProjectRequest {
    const message = createBaseUpdateProjectRequest();
    message.project = (object.project !== undefined && object.project !== null)
      ? Project.fromPartial(object.project)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseDeleteProjectRequest(): DeleteProjectRequest {
  return { name: "" };
}

export const DeleteProjectRequest = {
  encode(message: DeleteProjectRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteProjectRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteProjectRequest();
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

  fromJSON(object: any): DeleteProjectRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: DeleteProjectRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<DeleteProjectRequest>): DeleteProjectRequest {
    const message = createBaseDeleteProjectRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseUndeleteProjectRequest(): UndeleteProjectRequest {
  return { name: "" };
}

export const UndeleteProjectRequest = {
  encode(message: UndeleteProjectRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UndeleteProjectRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUndeleteProjectRequest();
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

  fromJSON(object: any): UndeleteProjectRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: UndeleteProjectRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<UndeleteProjectRequest>): UndeleteProjectRequest {
    const message = createBaseUndeleteProjectRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseGetIamPolicyRequest(): GetIamPolicyRequest {
  return { project: "" };
}

export const GetIamPolicyRequest = {
  encode(message: GetIamPolicyRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.project !== "") {
      writer.uint32(10).string(message.project);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetIamPolicyRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetIamPolicyRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.project = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): GetIamPolicyRequest {
    return { project: isSet(object.project) ? String(object.project) : "" };
  },

  toJSON(message: GetIamPolicyRequest): unknown {
    const obj: any = {};
    message.project !== undefined && (obj.project = message.project);
    return obj;
  },

  fromPartial(object: DeepPartial<GetIamPolicyRequest>): GetIamPolicyRequest {
    const message = createBaseGetIamPolicyRequest();
    message.project = object.project ?? "";
    return message;
  },
};

function createBaseSetIamPolicyRequest(): SetIamPolicyRequest {
  return { project: "", policy: undefined };
}

export const SetIamPolicyRequest = {
  encode(message: SetIamPolicyRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.project !== "") {
      writer.uint32(10).string(message.project);
    }
    if (message.policy !== undefined) {
      IamPolicy.encode(message.policy, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SetIamPolicyRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSetIamPolicyRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.project = reader.string();
          break;
        case 2:
          message.policy = IamPolicy.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): SetIamPolicyRequest {
    return {
      project: isSet(object.project) ? String(object.project) : "",
      policy: isSet(object.policy) ? IamPolicy.fromJSON(object.policy) : undefined,
    };
  },

  toJSON(message: SetIamPolicyRequest): unknown {
    const obj: any = {};
    message.project !== undefined && (obj.project = message.project);
    message.policy !== undefined && (obj.policy = message.policy ? IamPolicy.toJSON(message.policy) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<SetIamPolicyRequest>): SetIamPolicyRequest {
    const message = createBaseSetIamPolicyRequest();
    message.project = object.project ?? "";
    message.policy = (object.policy !== undefined && object.policy !== null)
      ? IamPolicy.fromPartial(object.policy)
      : undefined;
    return message;
  },
};

function createBaseGetDeploymentConfigRequest(): GetDeploymentConfigRequest {
  return { name: "" };
}

export const GetDeploymentConfigRequest = {
  encode(message: GetDeploymentConfigRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetDeploymentConfigRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetDeploymentConfigRequest();
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

  fromJSON(object: any): GetDeploymentConfigRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetDeploymentConfigRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<GetDeploymentConfigRequest>): GetDeploymentConfigRequest {
    const message = createBaseGetDeploymentConfigRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseUpdateDeploymentConfigRequest(): UpdateDeploymentConfigRequest {
  return { config: undefined };
}

export const UpdateDeploymentConfigRequest = {
  encode(message: UpdateDeploymentConfigRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.config !== undefined) {
      DeploymentConfig.encode(message.config, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateDeploymentConfigRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateDeploymentConfigRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.config = DeploymentConfig.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): UpdateDeploymentConfigRequest {
    return { config: isSet(object.config) ? DeploymentConfig.fromJSON(object.config) : undefined };
  },

  toJSON(message: UpdateDeploymentConfigRequest): unknown {
    const obj: any = {};
    message.config !== undefined && (obj.config = message.config ? DeploymentConfig.toJSON(message.config) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<UpdateDeploymentConfigRequest>): UpdateDeploymentConfigRequest {
    const message = createBaseUpdateDeploymentConfigRequest();
    message.config = (object.config !== undefined && object.config !== null)
      ? DeploymentConfig.fromPartial(object.config)
      : undefined;
    return message;
  },
};

function createBaseProject(): Project {
  return {
    name: "",
    uid: "",
    state: 0,
    title: "",
    key: "",
    workflow: 0,
    visibility: 0,
    tenantMode: 0,
    dbNameTemplate: "",
    schemaVersion: 0,
    schemaChange: 0,
    lgtmCheck: 0,
  };
}

export const Project = {
  encode(message: Project, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.uid !== "") {
      writer.uint32(18).string(message.uid);
    }
    if (message.state !== 0) {
      writer.uint32(24).int32(message.state);
    }
    if (message.title !== "") {
      writer.uint32(34).string(message.title);
    }
    if (message.key !== "") {
      writer.uint32(42).string(message.key);
    }
    if (message.workflow !== 0) {
      writer.uint32(48).int32(message.workflow);
    }
    if (message.visibility !== 0) {
      writer.uint32(56).int32(message.visibility);
    }
    if (message.tenantMode !== 0) {
      writer.uint32(64).int32(message.tenantMode);
    }
    if (message.dbNameTemplate !== "") {
      writer.uint32(74).string(message.dbNameTemplate);
    }
    if (message.schemaVersion !== 0) {
      writer.uint32(80).int32(message.schemaVersion);
    }
    if (message.schemaChange !== 0) {
      writer.uint32(88).int32(message.schemaChange);
    }
    if (message.lgtmCheck !== 0) {
      writer.uint32(96).int32(message.lgtmCheck);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Project {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseProject();
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
          message.title = reader.string();
          break;
        case 5:
          message.key = reader.string();
          break;
        case 6:
          message.workflow = reader.int32() as any;
          break;
        case 7:
          message.visibility = reader.int32() as any;
          break;
        case 8:
          message.tenantMode = reader.int32() as any;
          break;
        case 9:
          message.dbNameTemplate = reader.string();
          break;
        case 10:
          message.schemaVersion = reader.int32() as any;
          break;
        case 11:
          message.schemaChange = reader.int32() as any;
          break;
        case 12:
          message.lgtmCheck = reader.int32() as any;
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Project {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      uid: isSet(object.uid) ? String(object.uid) : "",
      state: isSet(object.state) ? stateFromJSON(object.state) : 0,
      title: isSet(object.title) ? String(object.title) : "",
      key: isSet(object.key) ? String(object.key) : "",
      workflow: isSet(object.workflow) ? workflowFromJSON(object.workflow) : 0,
      visibility: isSet(object.visibility) ? visibilityFromJSON(object.visibility) : 0,
      tenantMode: isSet(object.tenantMode) ? tenantModeFromJSON(object.tenantMode) : 0,
      dbNameTemplate: isSet(object.dbNameTemplate) ? String(object.dbNameTemplate) : "",
      schemaVersion: isSet(object.schemaVersion) ? schemaVersionFromJSON(object.schemaVersion) : 0,
      schemaChange: isSet(object.schemaChange) ? schemaChangeFromJSON(object.schemaChange) : 0,
      lgtmCheck: isSet(object.lgtmCheck) ? lgtmCheckFromJSON(object.lgtmCheck) : 0,
    };
  },

  toJSON(message: Project): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.state !== undefined && (obj.state = stateToJSON(message.state));
    message.title !== undefined && (obj.title = message.title);
    message.key !== undefined && (obj.key = message.key);
    message.workflow !== undefined && (obj.workflow = workflowToJSON(message.workflow));
    message.visibility !== undefined && (obj.visibility = visibilityToJSON(message.visibility));
    message.tenantMode !== undefined && (obj.tenantMode = tenantModeToJSON(message.tenantMode));
    message.dbNameTemplate !== undefined && (obj.dbNameTemplate = message.dbNameTemplate);
    message.schemaVersion !== undefined && (obj.schemaVersion = schemaVersionToJSON(message.schemaVersion));
    message.schemaChange !== undefined && (obj.schemaChange = schemaChangeToJSON(message.schemaChange));
    message.lgtmCheck !== undefined && (obj.lgtmCheck = lgtmCheckToJSON(message.lgtmCheck));
    return obj;
  },

  fromPartial(object: DeepPartial<Project>): Project {
    const message = createBaseProject();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.state = object.state ?? 0;
    message.title = object.title ?? "";
    message.key = object.key ?? "";
    message.workflow = object.workflow ?? 0;
    message.visibility = object.visibility ?? 0;
    message.tenantMode = object.tenantMode ?? 0;
    message.dbNameTemplate = object.dbNameTemplate ?? "";
    message.schemaVersion = object.schemaVersion ?? 0;
    message.schemaChange = object.schemaChange ?? 0;
    message.lgtmCheck = object.lgtmCheck ?? 0;
    return message;
  },
};

function createBaseIamPolicy(): IamPolicy {
  return { bindings: [] };
}

export const IamPolicy = {
  encode(message: IamPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.bindings) {
      Binding.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IamPolicy {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIamPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.bindings.push(Binding.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): IamPolicy {
    return { bindings: Array.isArray(object?.bindings) ? object.bindings.map((e: any) => Binding.fromJSON(e)) : [] };
  },

  toJSON(message: IamPolicy): unknown {
    const obj: any = {};
    if (message.bindings) {
      obj.bindings = message.bindings.map((e) => e ? Binding.toJSON(e) : undefined);
    } else {
      obj.bindings = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<IamPolicy>): IamPolicy {
    const message = createBaseIamPolicy();
    message.bindings = object.bindings?.map((e) => Binding.fromPartial(e)) || [];
    return message;
  },
};

function createBaseBinding(): Binding {
  return { role: 0, members: [] };
}

export const Binding = {
  encode(message: Binding, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.role !== 0) {
      writer.uint32(8).int32(message.role);
    }
    for (const v of message.members) {
      writer.uint32(18).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Binding {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBinding();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.role = reader.int32() as any;
          break;
        case 2:
          message.members.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Binding {
    return {
      role: isSet(object.role) ? projectRoleFromJSON(object.role) : 0,
      members: Array.isArray(object?.members) ? object.members.map((e: any) => String(e)) : [],
    };
  },

  toJSON(message: Binding): unknown {
    const obj: any = {};
    message.role !== undefined && (obj.role = projectRoleToJSON(message.role));
    if (message.members) {
      obj.members = message.members.map((e) => e);
    } else {
      obj.members = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<Binding>): Binding {
    const message = createBaseBinding();
    message.role = object.role ?? 0;
    message.members = object.members?.map((e) => e) || [];
    return message;
  },
};

function createBaseGetReviewRequest(): GetReviewRequest {
  return { name: "" };
}

export const GetReviewRequest = {
  encode(message: GetReviewRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetReviewRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetReviewRequest();
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

  fromJSON(object: any): GetReviewRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetReviewRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<GetReviewRequest>): GetReviewRequest {
    const message = createBaseGetReviewRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListReviewsRequest(): ListReviewsRequest {
  return { parent: "", pageSize: 0, pageToken: "" };
}

export const ListReviewsRequest = {
  encode(message: ListReviewsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): ListReviewsRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListReviewsRequest();
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

  fromJSON(object: any): ListReviewsRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListReviewsRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    return obj;
  },

  fromPartial(object: DeepPartial<ListReviewsRequest>): ListReviewsRequest {
    const message = createBaseListReviewsRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListReviewsResponse(): ListReviewsResponse {
  return { reviews: [], nextPageToken: "" };
}

export const ListReviewsResponse = {
  encode(message: ListReviewsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.reviews) {
      Review.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListReviewsResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListReviewsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.reviews.push(Review.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListReviewsResponse {
    return {
      reviews: Array.isArray(object?.reviews) ? object.reviews.map((e: any) => Review.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListReviewsResponse): unknown {
    const obj: any = {};
    if (message.reviews) {
      obj.reviews = message.reviews.map((e) => e ? Review.toJSON(e) : undefined);
    } else {
      obj.reviews = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  fromPartial(object: DeepPartial<ListReviewsResponse>): ListReviewsResponse {
    const message = createBaseListReviewsResponse();
    message.reviews = object.reviews?.map((e) => Review.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseUpdateReviewRequest(): UpdateReviewRequest {
  return { review: undefined, updateMask: undefined };
}

export const UpdateReviewRequest = {
  encode(message: UpdateReviewRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.review !== undefined) {
      Review.encode(message.review, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateReviewRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateReviewRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.review = Review.decode(reader, reader.uint32());
          break;
        case 2:
          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): UpdateReviewRequest {
    return {
      review: isSet(object.review) ? Review.fromJSON(object.review) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateReviewRequest): unknown {
    const obj: any = {};
    message.review !== undefined && (obj.review = message.review ? Review.toJSON(message.review) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  fromPartial(object: DeepPartial<UpdateReviewRequest>): UpdateReviewRequest {
    const message = createBaseUpdateReviewRequest();
    message.review = (object.review !== undefined && object.review !== null)
      ? Review.fromPartial(object.review)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseBatchUpdateReviewsRequest(): BatchUpdateReviewsRequest {
  return { parent: "", requests: [] };
}

export const BatchUpdateReviewsRequest = {
  encode(message: BatchUpdateReviewsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    for (const v of message.requests) {
      UpdateReviewRequest.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchUpdateReviewsRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchUpdateReviewsRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.parent = reader.string();
          break;
        case 2:
          message.requests.push(UpdateReviewRequest.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): BatchUpdateReviewsRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      requests: Array.isArray(object?.requests) ? object.requests.map((e: any) => UpdateReviewRequest.fromJSON(e)) : [],
    };
  },

  toJSON(message: BatchUpdateReviewsRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    if (message.requests) {
      obj.requests = message.requests.map((e) => e ? UpdateReviewRequest.toJSON(e) : undefined);
    } else {
      obj.requests = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<BatchUpdateReviewsRequest>): BatchUpdateReviewsRequest {
    const message = createBaseBatchUpdateReviewsRequest();
    message.parent = object.parent ?? "";
    message.requests = object.requests?.map((e) => UpdateReviewRequest.fromPartial(e)) || [];
    return message;
  },
};

function createBaseBatchUpdateReviewsResponse(): BatchUpdateReviewsResponse {
  return { reviews: [] };
}

export const BatchUpdateReviewsResponse = {
  encode(message: BatchUpdateReviewsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.reviews) {
      Review.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchUpdateReviewsResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchUpdateReviewsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.reviews.push(Review.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): BatchUpdateReviewsResponse {
    return { reviews: Array.isArray(object?.reviews) ? object.reviews.map((e: any) => Review.fromJSON(e)) : [] };
  },

  toJSON(message: BatchUpdateReviewsResponse): unknown {
    const obj: any = {};
    if (message.reviews) {
      obj.reviews = message.reviews.map((e) => e ? Review.toJSON(e) : undefined);
    } else {
      obj.reviews = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<BatchUpdateReviewsResponse>): BatchUpdateReviewsResponse {
    const message = createBaseBatchUpdateReviewsResponse();
    message.reviews = object.reviews?.map((e) => Review.fromPartial(e)) || [];
    return message;
  },
};

function createBaseListWebhooksRequest(): ListWebhooksRequest {
  return { parent: "", pageSize: 0, pageToken: "" };
}

export const ListWebhooksRequest = {
  encode(message: ListWebhooksRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): ListWebhooksRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListWebhooksRequest();
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

  fromJSON(object: any): ListWebhooksRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListWebhooksRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    return obj;
  },

  fromPartial(object: DeepPartial<ListWebhooksRequest>): ListWebhooksRequest {
    const message = createBaseListWebhooksRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListWebhooksResponse(): ListWebhooksResponse {
  return { webhooks: [], nextPageToken: "" };
}

export const ListWebhooksResponse = {
  encode(message: ListWebhooksResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.webhooks) {
      Webhook.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListWebhooksResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListWebhooksResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.webhooks.push(Webhook.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListWebhooksResponse {
    return {
      webhooks: Array.isArray(object?.webhooks) ? object.webhooks.map((e: any) => Webhook.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListWebhooksResponse): unknown {
    const obj: any = {};
    if (message.webhooks) {
      obj.webhooks = message.webhooks.map((e) => e ? Webhook.toJSON(e) : undefined);
    } else {
      obj.webhooks = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  fromPartial(object: DeepPartial<ListWebhooksResponse>): ListWebhooksResponse {
    const message = createBaseListWebhooksResponse();
    message.webhooks = object.webhooks?.map((e) => Webhook.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseGetWebhookRequest(): GetWebhookRequest {
  return { name: "" };
}

export const GetWebhookRequest = {
  encode(message: GetWebhookRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetWebhookRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetWebhookRequest();
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

  fromJSON(object: any): GetWebhookRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetWebhookRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<GetWebhookRequest>): GetWebhookRequest {
    const message = createBaseGetWebhookRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseAddWebhookRequest(): AddWebhookRequest {
  return { parent: "", webhook: undefined };
}

export const AddWebhookRequest = {
  encode(message: AddWebhookRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.webhook !== undefined) {
      Webhook.encode(message.webhook, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): AddWebhookRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAddWebhookRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.parent = reader.string();
          break;
        case 2:
          message.webhook = Webhook.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): AddWebhookRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      webhook: isSet(object.webhook) ? Webhook.fromJSON(object.webhook) : undefined,
    };
  },

  toJSON(message: AddWebhookRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.webhook !== undefined && (obj.webhook = message.webhook ? Webhook.toJSON(message.webhook) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<AddWebhookRequest>): AddWebhookRequest {
    const message = createBaseAddWebhookRequest();
    message.parent = object.parent ?? "";
    message.webhook = (object.webhook !== undefined && object.webhook !== null)
      ? Webhook.fromPartial(object.webhook)
      : undefined;
    return message;
  },
};

function createBaseModifyWebhookRequest(): ModifyWebhookRequest {
  return { webhook: undefined, updateMask: undefined };
}

export const ModifyWebhookRequest = {
  encode(message: ModifyWebhookRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.webhook !== undefined) {
      Webhook.encode(message.webhook, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ModifyWebhookRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseModifyWebhookRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.webhook = Webhook.decode(reader, reader.uint32());
          break;
        case 2:
          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ModifyWebhookRequest {
    return {
      webhook: isSet(object.webhook) ? Webhook.fromJSON(object.webhook) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: ModifyWebhookRequest): unknown {
    const obj: any = {};
    message.webhook !== undefined && (obj.webhook = message.webhook ? Webhook.toJSON(message.webhook) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  fromPartial(object: DeepPartial<ModifyWebhookRequest>): ModifyWebhookRequest {
    const message = createBaseModifyWebhookRequest();
    message.webhook = (object.webhook !== undefined && object.webhook !== null)
      ? Webhook.fromPartial(object.webhook)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseRemoveWebhookRequest(): RemoveWebhookRequest {
  return { name: "" };
}

export const RemoveWebhookRequest = {
  encode(message: RemoveWebhookRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RemoveWebhookRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRemoveWebhookRequest();
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

  fromJSON(object: any): RemoveWebhookRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: RemoveWebhookRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<RemoveWebhookRequest>): RemoveWebhookRequest {
    const message = createBaseRemoveWebhookRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseWebhook(): Webhook {
  return { name: "", type: 0, title: "", url: "", subTypes: [] };
}

export const Webhook = {
  encode(message: Webhook, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.type !== 0) {
      writer.uint32(16).int32(message.type);
    }
    if (message.title !== "") {
      writer.uint32(26).string(message.title);
    }
    if (message.url !== "") {
      writer.uint32(34).string(message.url);
    }
    writer.uint32(42).fork();
    for (const v of message.subTypes) {
      writer.int32(v);
    }
    writer.ldelim();
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Webhook {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseWebhook();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.type = reader.int32() as any;
          break;
        case 3:
          message.title = reader.string();
          break;
        case 4:
          message.url = reader.string();
          break;
        case 5:
          if ((tag & 7) === 2) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.subTypes.push(reader.int32() as any);
            }
          } else {
            message.subTypes.push(reader.int32() as any);
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Webhook {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      type: isSet(object.type) ? webhook_TypeFromJSON(object.type) : 0,
      title: isSet(object.title) ? String(object.title) : "",
      url: isSet(object.url) ? String(object.url) : "",
      subTypes: Array.isArray(object?.subTypes) ? object.subTypes.map((e: any) => activity_TypeFromJSON(e)) : [],
    };
  },

  toJSON(message: Webhook): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.type !== undefined && (obj.type = webhook_TypeToJSON(message.type));
    message.title !== undefined && (obj.title = message.title);
    message.url !== undefined && (obj.url = message.url);
    if (message.subTypes) {
      obj.subTypes = message.subTypes.map((e) => activity_TypeToJSON(e));
    } else {
      obj.subTypes = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<Webhook>): Webhook {
    const message = createBaseWebhook();
    message.name = object.name ?? "";
    message.type = object.type ?? 0;
    message.title = object.title ?? "";
    message.url = object.url ?? "";
    message.subTypes = object.subTypes?.map((e) => e) || [];
    return message;
  },
};

function createBaseReview(): Review {
  return {
    name: "",
    uid: "",
    title: "",
    description: "",
    status: 0,
    assignee: "",
    assigneeAttention: false,
    subscribers: [],
    creator: "",
    createTime: undefined,
    updateTime: undefined,
  };
}

export const Review = {
  encode(message: Review, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.uid !== "") {
      writer.uint32(18).string(message.uid);
    }
    if (message.title !== "") {
      writer.uint32(26).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(34).string(message.description);
    }
    if (message.status !== 0) {
      writer.uint32(40).int32(message.status);
    }
    if (message.assignee !== "") {
      writer.uint32(50).string(message.assignee);
    }
    if (message.assigneeAttention === true) {
      writer.uint32(56).bool(message.assigneeAttention);
    }
    for (const v of message.subscribers) {
      writer.uint32(66).string(v!);
    }
    if (message.creator !== "") {
      writer.uint32(74).string(message.creator);
    }
    if (message.createTime !== undefined) {
      Timestamp.encode(toTimestamp(message.createTime), writer.uint32(82).fork()).ldelim();
    }
    if (message.updateTime !== undefined) {
      Timestamp.encode(toTimestamp(message.updateTime), writer.uint32(90).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Review {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseReview();
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
          message.title = reader.string();
          break;
        case 4:
          message.description = reader.string();
          break;
        case 5:
          message.status = reader.int32() as any;
          break;
        case 6:
          message.assignee = reader.string();
          break;
        case 7:
          message.assigneeAttention = reader.bool();
          break;
        case 8:
          message.subscribers.push(reader.string());
          break;
        case 9:
          message.creator = reader.string();
          break;
        case 10:
          message.createTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          break;
        case 11:
          message.updateTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Review {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      uid: isSet(object.uid) ? String(object.uid) : "",
      title: isSet(object.title) ? String(object.title) : "",
      description: isSet(object.description) ? String(object.description) : "",
      status: isSet(object.status) ? reviewStatusFromJSON(object.status) : 0,
      assignee: isSet(object.assignee) ? String(object.assignee) : "",
      assigneeAttention: isSet(object.assigneeAttention) ? Boolean(object.assigneeAttention) : false,
      subscribers: Array.isArray(object?.subscribers) ? object.subscribers.map((e: any) => String(e)) : [],
      creator: isSet(object.creator) ? String(object.creator) : "",
      createTime: isSet(object.createTime) ? fromJsonTimestamp(object.createTime) : undefined,
      updateTime: isSet(object.updateTime) ? fromJsonTimestamp(object.updateTime) : undefined,
    };
  },

  toJSON(message: Review): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.title !== undefined && (obj.title = message.title);
    message.description !== undefined && (obj.description = message.description);
    message.status !== undefined && (obj.status = reviewStatusToJSON(message.status));
    message.assignee !== undefined && (obj.assignee = message.assignee);
    message.assigneeAttention !== undefined && (obj.assigneeAttention = message.assigneeAttention);
    if (message.subscribers) {
      obj.subscribers = message.subscribers.map((e) => e);
    } else {
      obj.subscribers = [];
    }
    message.creator !== undefined && (obj.creator = message.creator);
    message.createTime !== undefined && (obj.createTime = message.createTime.toISOString());
    message.updateTime !== undefined && (obj.updateTime = message.updateTime.toISOString());
    return obj;
  },

  fromPartial(object: DeepPartial<Review>): Review {
    const message = createBaseReview();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.status = object.status ?? 0;
    message.assignee = object.assignee ?? "";
    message.assigneeAttention = object.assigneeAttention ?? false;
    message.subscribers = object.subscribers?.map((e) => e) || [];
    message.creator = object.creator ?? "";
    message.createTime = object.createTime ?? undefined;
    message.updateTime = object.updateTime ?? undefined;
    return message;
  },
};

function createBaseDeploymentConfig(): DeploymentConfig {
  return { name: "", title: "", schedule: undefined };
}

export const DeploymentConfig = {
  encode(message: DeploymentConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.schedule !== undefined) {
      Schedule.encode(message.schedule, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeploymentConfig {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeploymentConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.title = reader.string();
          break;
        case 3:
          message.schedule = Schedule.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DeploymentConfig {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      title: isSet(object.title) ? String(object.title) : "",
      schedule: isSet(object.schedule) ? Schedule.fromJSON(object.schedule) : undefined,
    };
  },

  toJSON(message: DeploymentConfig): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.title !== undefined && (obj.title = message.title);
    message.schedule !== undefined && (obj.schedule = message.schedule ? Schedule.toJSON(message.schedule) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<DeploymentConfig>): DeploymentConfig {
    const message = createBaseDeploymentConfig();
    message.name = object.name ?? "";
    message.title = object.title ?? "";
    message.schedule = (object.schedule !== undefined && object.schedule !== null)
      ? Schedule.fromPartial(object.schedule)
      : undefined;
    return message;
  },
};

function createBaseSchedule(): Schedule {
  return { deployments: [] };
}

export const Schedule = {
  encode(message: Schedule, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.deployments) {
      ScheduleDeployment.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Schedule {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSchedule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.deployments.push(ScheduleDeployment.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Schedule {
    return {
      deployments: Array.isArray(object?.deployments)
        ? object.deployments.map((e: any) => ScheduleDeployment.fromJSON(e))
        : [],
    };
  },

  toJSON(message: Schedule): unknown {
    const obj: any = {};
    if (message.deployments) {
      obj.deployments = message.deployments.map((e) => e ? ScheduleDeployment.toJSON(e) : undefined);
    } else {
      obj.deployments = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<Schedule>): Schedule {
    const message = createBaseSchedule();
    message.deployments = object.deployments?.map((e) => ScheduleDeployment.fromPartial(e)) || [];
    return message;
  },
};

function createBaseScheduleDeployment(): ScheduleDeployment {
  return { title: "", spec: undefined };
}

export const ScheduleDeployment = {
  encode(message: ScheduleDeployment, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.title !== "") {
      writer.uint32(10).string(message.title);
    }
    if (message.spec !== undefined) {
      DeploymentSpec.encode(message.spec, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ScheduleDeployment {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseScheduleDeployment();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.title = reader.string();
          break;
        case 2:
          message.spec = DeploymentSpec.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ScheduleDeployment {
    return {
      title: isSet(object.title) ? String(object.title) : "",
      spec: isSet(object.spec) ? DeploymentSpec.fromJSON(object.spec) : undefined,
    };
  },

  toJSON(message: ScheduleDeployment): unknown {
    const obj: any = {};
    message.title !== undefined && (obj.title = message.title);
    message.spec !== undefined && (obj.spec = message.spec ? DeploymentSpec.toJSON(message.spec) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<ScheduleDeployment>): ScheduleDeployment {
    const message = createBaseScheduleDeployment();
    message.title = object.title ?? "";
    message.spec = (object.spec !== undefined && object.spec !== null)
      ? DeploymentSpec.fromPartial(object.spec)
      : undefined;
    return message;
  },
};

function createBaseDeploymentSpec(): DeploymentSpec {
  return { labelSelector: undefined };
}

export const DeploymentSpec = {
  encode(message: DeploymentSpec, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.labelSelector !== undefined) {
      LabelSelector.encode(message.labelSelector, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeploymentSpec {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeploymentSpec();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.labelSelector = LabelSelector.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): DeploymentSpec {
    return { labelSelector: isSet(object.labelSelector) ? LabelSelector.fromJSON(object.labelSelector) : undefined };
  },

  toJSON(message: DeploymentSpec): unknown {
    const obj: any = {};
    message.labelSelector !== undefined &&
      (obj.labelSelector = message.labelSelector ? LabelSelector.toJSON(message.labelSelector) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<DeploymentSpec>): DeploymentSpec {
    const message = createBaseDeploymentSpec();
    message.labelSelector = (object.labelSelector !== undefined && object.labelSelector !== null)
      ? LabelSelector.fromPartial(object.labelSelector)
      : undefined;
    return message;
  },
};

function createBaseLabelSelector(): LabelSelector {
  return { matchExpressions: [] };
}

export const LabelSelector = {
  encode(message: LabelSelector, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.matchExpressions) {
      LabelSelectorRequirement.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): LabelSelector {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseLabelSelector();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.matchExpressions.push(LabelSelectorRequirement.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): LabelSelector {
    return {
      matchExpressions: Array.isArray(object?.matchExpressions)
        ? object.matchExpressions.map((e: any) => LabelSelectorRequirement.fromJSON(e))
        : [],
    };
  },

  toJSON(message: LabelSelector): unknown {
    const obj: any = {};
    if (message.matchExpressions) {
      obj.matchExpressions = message.matchExpressions.map((e) => e ? LabelSelectorRequirement.toJSON(e) : undefined);
    } else {
      obj.matchExpressions = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<LabelSelector>): LabelSelector {
    const message = createBaseLabelSelector();
    message.matchExpressions = object.matchExpressions?.map((e) => LabelSelectorRequirement.fromPartial(e)) || [];
    return message;
  },
};

function createBaseLabelSelectorRequirement(): LabelSelectorRequirement {
  return { key: "", operator: 0, values: [] };
}

export const LabelSelectorRequirement = {
  encode(message: LabelSelectorRequirement, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.operator !== 0) {
      writer.uint32(16).int32(message.operator);
    }
    for (const v of message.values) {
      writer.uint32(26).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): LabelSelectorRequirement {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseLabelSelectorRequirement();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.operator = reader.int32() as any;
          break;
        case 3:
          message.values.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): LabelSelectorRequirement {
    return {
      key: isSet(object.key) ? String(object.key) : "",
      operator: isSet(object.operator) ? operatorTypeFromJSON(object.operator) : 0,
      values: Array.isArray(object?.values) ? object.values.map((e: any) => String(e)) : [],
    };
  },

  toJSON(message: LabelSelectorRequirement): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.operator !== undefined && (obj.operator = operatorTypeToJSON(message.operator));
    if (message.values) {
      obj.values = message.values.map((e) => e);
    } else {
      obj.values = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<LabelSelectorRequirement>): LabelSelectorRequirement {
    const message = createBaseLabelSelectorRequirement();
    message.key = object.key ?? "";
    message.operator = object.operator ?? 0;
    message.values = object.values?.map((e) => e) || [];
    return message;
  },
};

export type ProjectServiceDefinition = typeof ProjectServiceDefinition;
export const ProjectServiceDefinition = {
  name: "ProjectService",
  fullName: "bytebase.v1.ProjectService",
  methods: {
    getProject: {
      name: "GetProject",
      requestType: GetProjectRequest,
      requestStream: false,
      responseType: Project,
      responseStream: false,
      options: {},
    },
    listProjects: {
      name: "ListProjects",
      requestType: ListProjectsRequest,
      requestStream: false,
      responseType: ListProjectsResponse,
      responseStream: false,
      options: {},
    },
    createProject: {
      name: "CreateProject",
      requestType: CreateProjectRequest,
      requestStream: false,
      responseType: Project,
      responseStream: false,
      options: {},
    },
    updateProject: {
      name: "UpdateProject",
      requestType: UpdateProjectRequest,
      requestStream: false,
      responseType: Project,
      responseStream: false,
      options: {},
    },
    deleteProject: {
      name: "DeleteProject",
      requestType: DeleteProjectRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {},
    },
    undeleteProject: {
      name: "UndeleteProject",
      requestType: UndeleteProjectRequest,
      requestStream: false,
      responseType: Project,
      responseStream: false,
      options: {},
    },
    getIamPolicy: {
      name: "GetIamPolicy",
      requestType: GetIamPolicyRequest,
      requestStream: false,
      responseType: IamPolicy,
      responseStream: false,
      options: {},
    },
    setIamPolicy: {
      name: "SetIamPolicy",
      requestType: SetIamPolicyRequest,
      requestStream: false,
      responseType: IamPolicy,
      responseStream: false,
      options: {},
    },
    getReview: {
      name: "GetReview",
      requestType: GetReviewRequest,
      requestStream: false,
      responseType: Review,
      responseStream: false,
      options: {},
    },
    listReviews: {
      name: "ListReviews",
      requestType: ListReviewsRequest,
      requestStream: false,
      responseType: ListReviewsResponse,
      responseStream: false,
      options: {},
    },
    updateReview: {
      name: "UpdateReview",
      requestType: UpdateReviewRequest,
      requestStream: false,
      responseType: Review,
      responseStream: false,
      options: {},
    },
    batchUpdateReviews: {
      name: "BatchUpdateReviews",
      requestType: BatchUpdateReviewsRequest,
      requestStream: false,
      responseType: BatchUpdateReviewsResponse,
      responseStream: false,
      options: {},
    },
    getDeploymentConfig: {
      name: "GetDeploymentConfig",
      requestType: GetDeploymentConfigRequest,
      requestStream: false,
      responseType: DeploymentConfig,
      responseStream: false,
      options: {},
    },
    updateDeploymentConfig: {
      name: "UpdateDeploymentConfig",
      requestType: UpdateDeploymentConfigRequest,
      requestStream: false,
      responseType: DeploymentConfig,
      responseStream: false,
      options: {},
    },
    listWebhooks: {
      name: "ListWebhooks",
      requestType: ListWebhooksRequest,
      requestStream: false,
      responseType: ListWebhooksResponse,
      responseStream: false,
      options: {},
    },
    getWebhook: {
      name: "GetWebhook",
      requestType: GetWebhookRequest,
      requestStream: false,
      responseType: Webhook,
      responseStream: false,
      options: {},
    },
    addWebhook: {
      name: "AddWebhook",
      requestType: AddWebhookRequest,
      requestStream: false,
      responseType: Webhook,
      responseStream: false,
      options: {},
    },
    modifyWebhook: {
      name: "ModifyWebhook",
      requestType: ModifyWebhookRequest,
      requestStream: false,
      responseType: Webhook,
      responseStream: false,
      options: {},
    },
    removeWebhook: {
      name: "RemoveWebhook",
      requestType: RemoveWebhookRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {},
    },
  },
} as const;

export interface ProjectServiceImplementation<CallContextExt = {}> {
  getProject(request: GetProjectRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Project>>;
  listProjects(
    request: ListProjectsRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ListProjectsResponse>>;
  createProject(request: CreateProjectRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Project>>;
  updateProject(request: UpdateProjectRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Project>>;
  deleteProject(request: DeleteProjectRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Empty>>;
  undeleteProject(
    request: UndeleteProjectRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Project>>;
  getIamPolicy(request: GetIamPolicyRequest, context: CallContext & CallContextExt): Promise<DeepPartial<IamPolicy>>;
  setIamPolicy(request: SetIamPolicyRequest, context: CallContext & CallContextExt): Promise<DeepPartial<IamPolicy>>;
  getReview(request: GetReviewRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Review>>;
  listReviews(
    request: ListReviewsRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ListReviewsResponse>>;
  updateReview(request: UpdateReviewRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Review>>;
  batchUpdateReviews(
    request: BatchUpdateReviewsRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<BatchUpdateReviewsResponse>>;
  getDeploymentConfig(
    request: GetDeploymentConfigRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<DeploymentConfig>>;
  updateDeploymentConfig(
    request: UpdateDeploymentConfigRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<DeploymentConfig>>;
  listWebhooks(
    request: ListWebhooksRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ListWebhooksResponse>>;
  getWebhook(request: GetWebhookRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Webhook>>;
  addWebhook(request: AddWebhookRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Webhook>>;
  modifyWebhook(request: ModifyWebhookRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Webhook>>;
  removeWebhook(request: RemoveWebhookRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Empty>>;
}

export interface ProjectServiceClient<CallOptionsExt = {}> {
  getProject(request: DeepPartial<GetProjectRequest>, options?: CallOptions & CallOptionsExt): Promise<Project>;
  listProjects(
    request: DeepPartial<ListProjectsRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ListProjectsResponse>;
  createProject(request: DeepPartial<CreateProjectRequest>, options?: CallOptions & CallOptionsExt): Promise<Project>;
  updateProject(request: DeepPartial<UpdateProjectRequest>, options?: CallOptions & CallOptionsExt): Promise<Project>;
  deleteProject(request: DeepPartial<DeleteProjectRequest>, options?: CallOptions & CallOptionsExt): Promise<Empty>;
  undeleteProject(
    request: DeepPartial<UndeleteProjectRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Project>;
  getIamPolicy(request: DeepPartial<GetIamPolicyRequest>, options?: CallOptions & CallOptionsExt): Promise<IamPolicy>;
  setIamPolicy(request: DeepPartial<SetIamPolicyRequest>, options?: CallOptions & CallOptionsExt): Promise<IamPolicy>;
  getReview(request: DeepPartial<GetReviewRequest>, options?: CallOptions & CallOptionsExt): Promise<Review>;
  listReviews(
    request: DeepPartial<ListReviewsRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ListReviewsResponse>;
  updateReview(request: DeepPartial<UpdateReviewRequest>, options?: CallOptions & CallOptionsExt): Promise<Review>;
  batchUpdateReviews(
    request: DeepPartial<BatchUpdateReviewsRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<BatchUpdateReviewsResponse>;
  getDeploymentConfig(
    request: DeepPartial<GetDeploymentConfigRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<DeploymentConfig>;
  updateDeploymentConfig(
    request: DeepPartial<UpdateDeploymentConfigRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<DeploymentConfig>;
  listWebhooks(
    request: DeepPartial<ListWebhooksRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ListWebhooksResponse>;
  getWebhook(request: DeepPartial<GetWebhookRequest>, options?: CallOptions & CallOptionsExt): Promise<Webhook>;
  addWebhook(request: DeepPartial<AddWebhookRequest>, options?: CallOptions & CallOptionsExt): Promise<Webhook>;
  modifyWebhook(request: DeepPartial<ModifyWebhookRequest>, options?: CallOptions & CallOptionsExt): Promise<Webhook>;
  removeWebhook(request: DeepPartial<RemoveWebhookRequest>, options?: CallOptions & CallOptionsExt): Promise<Empty>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function toTimestamp(date: Date): Timestamp {
  const seconds = date.getTime() / 1_000;
  const nanos = (date.getTime() % 1_000) * 1_000_000;
  return { seconds, nanos };
}

function fromTimestamp(t: Timestamp): Date {
  let millis = t.seconds * 1_000;
  millis += t.nanos / 1_000_000;
  return new Date(millis);
}

function fromJsonTimestamp(o: any): Date {
  if (o instanceof Date) {
    return o;
  } else if (typeof o === "string") {
    return new Date(o);
  } else {
    return fromTimestamp(Timestamp.fromJSON(o));
  }
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
