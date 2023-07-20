/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { Expr } from "../google/type/expr";
import { State, stateFromJSON, stateToJSON } from "./common";
import { ProjectGitOpsInfo } from "./externalvs_service";
import { IamPolicy } from "./iam_policy";

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

export enum DatabaseGroupView {
  /**
   * DATABASE_GROUP_VIEW_UNSPECIFIED - The default / unset value.
   * The API will default to the BASIC view.
   */
  DATABASE_GROUP_VIEW_UNSPECIFIED = 0,
  /** DATABASE_GROUP_VIEW_BASIC - Include basic information about the database group, but exclude the list of matched databases and unmatched databases. */
  DATABASE_GROUP_VIEW_BASIC = 1,
  /** DATABASE_GROUP_VIEW_FULL - Include everything. */
  DATABASE_GROUP_VIEW_FULL = 2,
  UNRECOGNIZED = -1,
}

export function databaseGroupViewFromJSON(object: any): DatabaseGroupView {
  switch (object) {
    case 0:
    case "DATABASE_GROUP_VIEW_UNSPECIFIED":
      return DatabaseGroupView.DATABASE_GROUP_VIEW_UNSPECIFIED;
    case 1:
    case "DATABASE_GROUP_VIEW_BASIC":
      return DatabaseGroupView.DATABASE_GROUP_VIEW_BASIC;
    case 2:
    case "DATABASE_GROUP_VIEW_FULL":
      return DatabaseGroupView.DATABASE_GROUP_VIEW_FULL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return DatabaseGroupView.UNRECOGNIZED;
  }
}

export function databaseGroupViewToJSON(object: DatabaseGroupView): string {
  switch (object) {
    case DatabaseGroupView.DATABASE_GROUP_VIEW_UNSPECIFIED:
      return "DATABASE_GROUP_VIEW_UNSPECIFIED";
    case DatabaseGroupView.DATABASE_GROUP_VIEW_BASIC:
      return "DATABASE_GROUP_VIEW_BASIC";
    case DatabaseGroupView.DATABASE_GROUP_VIEW_FULL:
      return "DATABASE_GROUP_VIEW_FULL";
    case DatabaseGroupView.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum SchemaGroupView {
  /**
   * SCHEMA_GROUP_VIEW_UNSPECIFIED - The default / unset value.
   * The API will default to the BASIC view.
   */
  SCHEMA_GROUP_VIEW_UNSPECIFIED = 0,
  /** SCHEMA_GROUP_VIEW_BASIC - Include basic information about the schema group, but exclude the list of matched tables and unmatched tables. */
  SCHEMA_GROUP_VIEW_BASIC = 1,
  /** SCHEMA_GROUP_VIEW_FULL - Include everything. */
  SCHEMA_GROUP_VIEW_FULL = 2,
  UNRECOGNIZED = -1,
}

export function schemaGroupViewFromJSON(object: any): SchemaGroupView {
  switch (object) {
    case 0:
    case "SCHEMA_GROUP_VIEW_UNSPECIFIED":
      return SchemaGroupView.SCHEMA_GROUP_VIEW_UNSPECIFIED;
    case 1:
    case "SCHEMA_GROUP_VIEW_BASIC":
      return SchemaGroupView.SCHEMA_GROUP_VIEW_BASIC;
    case 2:
    case "SCHEMA_GROUP_VIEW_FULL":
      return SchemaGroupView.SCHEMA_GROUP_VIEW_FULL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return SchemaGroupView.UNRECOGNIZED;
  }
}

export function schemaGroupViewToJSON(object: SchemaGroupView): string {
  switch (object) {
    case SchemaGroupView.SCHEMA_GROUP_VIEW_UNSPECIFIED:
      return "SCHEMA_GROUP_VIEW_UNSPECIFIED";
    case SchemaGroupView.SCHEMA_GROUP_VIEW_BASIC:
      return "SCHEMA_GROUP_VIEW_BASIC";
    case SchemaGroupView.SCHEMA_GROUP_VIEW_FULL:
      return "SCHEMA_GROUP_VIEW_FULL";
    case SchemaGroupView.UNRECOGNIZED:
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

export interface SearchProjectsRequest {
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
  /** Filter is used to filter projects returned in the list. */
  filter: string;
}

export interface SearchProjectsResponse {
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
  project?:
    | Project
    | undefined;
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
  project?:
    | Project
    | undefined;
  /** The list of fields to update. */
  updateMask?: string[] | undefined;
}

export interface DeleteProjectRequest {
  /**
   * The name of the project to delete.
   * Format: projects/{project}
   */
  name: string;
  /** If set to true, any databases and sheets from this project will also be moved to default project, and all open issues will be closed. */
  force: boolean;
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

export interface BatchGetIamPolicyRequest {
  /** The scope of the batch get. Typically it's "projects/-". */
  scope: string;
  names: string[];
}

export interface BatchGetIamPolicyResponse {
  policyResults: BatchGetIamPolicyResponse_PolicyResult[];
}

export interface BatchGetIamPolicyResponse_PolicyResult {
  project: string;
  policy?: IamPolicy | undefined;
}

export interface SetIamPolicyRequest {
  /**
   * The name of the project to set the IAM policy.
   * Format: projects/{project}
   */
  project: string;
  policy?: IamPolicy | undefined;
}

export interface GetDeploymentConfigRequest {
  /**
   * The name of the resource.
   * Format: projects/{project}/deploymentConfig
   */
  name: string;
}

export interface UpdateDeploymentConfigRequest {
  config?: DeploymentConfig | undefined;
}

export interface UpdateProjectGitOpsInfoRequest {
  /** The binding for the project and external version control. */
  projectGitopsInfo?:
    | ProjectGitOpsInfo
    | undefined;
  /** The mask of the fields to be updated. */
  updateMask?:
    | string[]
    | undefined;
  /** If true, the gitops will be created if it does not exist. */
  allowMissing: boolean;
}

export interface UnsetProjectGitOpsInfoRequest {
  /**
   * The name of the GitOps info.
   * Format: projects/{project}/gitOpsInfo
   */
  name: string;
}

export interface GetProjectGitOpsInfoRequest {
  /**
   * The name of the GitOps info.
   * Format: projects/{project}/gitOpsInfo
   */
  name: string;
}

export interface SetupSQLReviewCIRequest {
  /**
   * The name of the GitOps info.
   * Format: projects/{project}/gitOpsInfo
   */
  name: string;
}

export interface SetupSQLReviewCIResponse {
  /** The CI setup PR URL for the repository. */
  pullRequestUrl: string;
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
  schemaChange: SchemaChange;
  webhooks: Webhook[];
  dataCategoryConfigId: string;
}

export interface AddWebhookRequest {
  /**
   * The name of the project to add the webhook to.
   * Format: projects/{project}
   */
  project: string;
  /** The webhook to add. */
  webhook?: Webhook | undefined;
}

export interface UpdateWebhookRequest {
  /** The webhook to modify. */
  webhook?:
    | Webhook
    | undefined;
  /** The list of fields to update. */
  updateMask?: string[] | undefined;
}

export interface RemoveWebhookRequest {
  /** The webhook to remove. Identified by its url. */
  webhook?: Webhook | undefined;
}

export interface TestWebhookRequest {
  /**
   * The name of the project which owns the webhook to test.
   * Format: projects/{project}
   */
  project: string;
  /** The webhook to test. Identified by its url. */
  webhook?: Webhook | undefined;
}

export interface TestWebhookResponse {
  /** The result of the test, empty if the test is successful. */
  error: string;
}

export interface Webhook {
  /**
   * name is the name of the webhook, generated by the server.
   * format: projects/{project}/webhooks/{webhook}
   */
  name: string;
  /** type is the type of the webhook. */
  type: Webhook_Type;
  /** title is the title of the webhook. */
  title: string;
  /** url is the url of the webhook, should be unique within the project. */
  url: string;
  /**
   * notification_types is the list of activities types that the webhook is interested in.
   * Bytebase will only send notifications to the webhook if the activity type is in the list.
   * It should not be empty, and shoule be a subset of the following:
   * - TYPE_ISSUE_CREATED
   * - TYPE_ISSUE_STATUS_UPDATE
   * - TYPE_ISSUE_PIPELINE_STAGE_UPDATE
   * - TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE
   * - TYPE_ISSUE_FIELD_UPDATE
   * - TYPE_ISSUE_COMMENT_CREAT
   */
  notificationTypes: Activity_Type[];
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

export interface DeploymentConfig {
  /**
   * The name of the resource.
   * Format: projects/{project}/deploymentConfig
   */
  name: string;
  /** The title of the deployment config. */
  title: string;
  schedule?: Schedule | undefined;
}

export interface Schedule {
  deployments: ScheduleDeployment[];
}

export interface ScheduleDeployment {
  /** The title of the deployment (stage) in a schedule. */
  title: string;
  spec?: DeploymentSpec | undefined;
}

export interface DeploymentSpec {
  labelSelector?: LabelSelector | undefined;
}

export interface LabelSelector {
  matchExpressions: LabelSelectorRequirement[];
}

export interface LabelSelectorRequirement {
  key: string;
  operator: OperatorType;
  values: string[];
}

/** TODO(zp): move to activity later. */
export interface Activity {
}

export enum Activity_Type {
  TYPE_UNSPECIFIED = 0,
  /**
   * TYPE_ISSUE_CREATE - Issue related activity types.
   *
   * TYPE_ISSUE_CREATE represents creating an issue.
   */
  TYPE_ISSUE_CREATE = 1,
  /** TYPE_ISSUE_COMMENT_CREATE - TYPE_ISSUE_COMMENT_CREATE represents commenting on an issue. */
  TYPE_ISSUE_COMMENT_CREATE = 2,
  /** TYPE_ISSUE_FIELD_UPDATE - TYPE_ISSUE_FIELD_UPDATE represents updating the issue field, likes title, description, assignee, etc. */
  TYPE_ISSUE_FIELD_UPDATE = 3,
  /** TYPE_ISSUE_STATUS_UPDATE - TYPE_ISSUE_STATUS_UPDATE represents the issue status change, including OPEN, CLOSE, CANCEL fow now. */
  TYPE_ISSUE_STATUS_UPDATE = 4,
  /** TYPE_ISSUE_APPROVAL_NOTIFY - TYPE_ISSUE_APPROVAL_NOTIFY is the type for notifying issue approval. */
  TYPE_ISSUE_APPROVAL_NOTIFY = 21,
  /** TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE - TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE represents the pipeline stage status change, including BEGIN, END for now. */
  TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE = 5,
  /** TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE - TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE represents the pipeline task status change, including PENDING, PENDING_APPROVAL, RUNNING, SUCCESS, FAILURE, CANCELED for now. */
  TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE = 6,
  /** TYPE_ISSUE_PIPELINE_TASK_FILE_COMMIT - TYPE_ISSUE_PIPELINE_TASK_FILE_COMMIT represents the VCS trigger to commit a file to update the task statement. */
  TYPE_ISSUE_PIPELINE_TASK_FILE_COMMIT = 7,
  /** TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE - TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE represents the manual update of the task statement. */
  TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE = 8,
  /** TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE - TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE represents the manual update of the task earliest allowed time. */
  TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE = 9,
  /**
   * TYPE_MEMBER_CREATE - Member related activity types.
   *
   * TYPE_MEMBER_CREATE represents creating a members.
   */
  TYPE_MEMBER_CREATE = 10,
  /** TYPE_MEMBER_ROLE_UPDATE - TYPE_MEMBER_ROLE_UPDATE represents updating the member role, for example, from ADMIN to MEMBER. */
  TYPE_MEMBER_ROLE_UPDATE = 11,
  /** TYPE_MEMBER_ACTIVATE - TYPE_MEMBER_ACTIVATE represents activating a deactivated member. */
  TYPE_MEMBER_ACTIVATE = 12,
  /** TYPE_MEMBER_DEACTIVATE - TYPE_MEMBER_DEACTIVATE represents deactivating an active member. */
  TYPE_MEMBER_DEACTIVATE = 13,
  /**
   * TYPE_PROJECT_REPOSITORY_PUSH - Project related activity types.
   *
   * TYPE_PROJECT_REPOSITORY_PUSH represents Bytebase receiving a push event from the project repository.
   */
  TYPE_PROJECT_REPOSITORY_PUSH = 14,
  /** TYPE_PROJECT_DATABASE_TRANSFER - TYPE_PROJECT_DATABASE_TRANFER represents transfering the database from one project to another. */
  TYPE_PROJECT_DATABASE_TRANSFER = 15,
  /** TYPE_PROJECT_MEMBER_CREATE - TYPE_PROJECT_MEMBER_CREATE represents adding a member to the project. */
  TYPE_PROJECT_MEMBER_CREATE = 16,
  /** TYPE_PROJECT_MEMBER_DELETE - TYPE_PROJECT_MEMBER_DELETE represents removing a member from the project. */
  TYPE_PROJECT_MEMBER_DELETE = 17,
  /** TYPE_PROJECT_MEMBER_ROLE_UPDATE - TYPE_PROJECT_MEMBER_ROLE_UPDATE represents updating the member role, for example, from ADMIN to MEMBER. */
  TYPE_PROJECT_MEMBER_ROLE_UPDATE = 18,
  /**
   * TYPE_SQL_EDITOR_QUERY - SQL Editor related activity types.
   * TYPE_SQL_EDITOR_QUERY represents executing query in SQL Editor.
   */
  TYPE_SQL_EDITOR_QUERY = 19,
  /**
   * TYPE_DATABASE_RECOVERY_PITR_DONE - Database related activity types.
   * TYPE_DATABASE_RECOVERY_PITR_DONE represents the database recovery to a point in time is done.
   */
  TYPE_DATABASE_RECOVERY_PITR_DONE = 20,
  UNRECOGNIZED = -1,
}

export function activity_TypeFromJSON(object: any): Activity_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return Activity_Type.TYPE_UNSPECIFIED;
    case 1:
    case "TYPE_ISSUE_CREATE":
      return Activity_Type.TYPE_ISSUE_CREATE;
    case 2:
    case "TYPE_ISSUE_COMMENT_CREATE":
      return Activity_Type.TYPE_ISSUE_COMMENT_CREATE;
    case 3:
    case "TYPE_ISSUE_FIELD_UPDATE":
      return Activity_Type.TYPE_ISSUE_FIELD_UPDATE;
    case 4:
    case "TYPE_ISSUE_STATUS_UPDATE":
      return Activity_Type.TYPE_ISSUE_STATUS_UPDATE;
    case 21:
    case "TYPE_ISSUE_APPROVAL_NOTIFY":
      return Activity_Type.TYPE_ISSUE_APPROVAL_NOTIFY;
    case 5:
    case "TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE":
      return Activity_Type.TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE;
    case 6:
    case "TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE":
      return Activity_Type.TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE;
    case 7:
    case "TYPE_ISSUE_PIPELINE_TASK_FILE_COMMIT":
      return Activity_Type.TYPE_ISSUE_PIPELINE_TASK_FILE_COMMIT;
    case 8:
    case "TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE":
      return Activity_Type.TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE;
    case 9:
    case "TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE":
      return Activity_Type.TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE;
    case 10:
    case "TYPE_MEMBER_CREATE":
      return Activity_Type.TYPE_MEMBER_CREATE;
    case 11:
    case "TYPE_MEMBER_ROLE_UPDATE":
      return Activity_Type.TYPE_MEMBER_ROLE_UPDATE;
    case 12:
    case "TYPE_MEMBER_ACTIVATE":
      return Activity_Type.TYPE_MEMBER_ACTIVATE;
    case 13:
    case "TYPE_MEMBER_DEACTIVATE":
      return Activity_Type.TYPE_MEMBER_DEACTIVATE;
    case 14:
    case "TYPE_PROJECT_REPOSITORY_PUSH":
      return Activity_Type.TYPE_PROJECT_REPOSITORY_PUSH;
    case 15:
    case "TYPE_PROJECT_DATABASE_TRANSFER":
      return Activity_Type.TYPE_PROJECT_DATABASE_TRANSFER;
    case 16:
    case "TYPE_PROJECT_MEMBER_CREATE":
      return Activity_Type.TYPE_PROJECT_MEMBER_CREATE;
    case 17:
    case "TYPE_PROJECT_MEMBER_DELETE":
      return Activity_Type.TYPE_PROJECT_MEMBER_DELETE;
    case 18:
    case "TYPE_PROJECT_MEMBER_ROLE_UPDATE":
      return Activity_Type.TYPE_PROJECT_MEMBER_ROLE_UPDATE;
    case 19:
    case "TYPE_SQL_EDITOR_QUERY":
      return Activity_Type.TYPE_SQL_EDITOR_QUERY;
    case 20:
    case "TYPE_DATABASE_RECOVERY_PITR_DONE":
      return Activity_Type.TYPE_DATABASE_RECOVERY_PITR_DONE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Activity_Type.UNRECOGNIZED;
  }
}

export function activity_TypeToJSON(object: Activity_Type): string {
  switch (object) {
    case Activity_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case Activity_Type.TYPE_ISSUE_CREATE:
      return "TYPE_ISSUE_CREATE";
    case Activity_Type.TYPE_ISSUE_COMMENT_CREATE:
      return "TYPE_ISSUE_COMMENT_CREATE";
    case Activity_Type.TYPE_ISSUE_FIELD_UPDATE:
      return "TYPE_ISSUE_FIELD_UPDATE";
    case Activity_Type.TYPE_ISSUE_STATUS_UPDATE:
      return "TYPE_ISSUE_STATUS_UPDATE";
    case Activity_Type.TYPE_ISSUE_APPROVAL_NOTIFY:
      return "TYPE_ISSUE_APPROVAL_NOTIFY";
    case Activity_Type.TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE:
      return "TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE";
    case Activity_Type.TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE:
      return "TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE";
    case Activity_Type.TYPE_ISSUE_PIPELINE_TASK_FILE_COMMIT:
      return "TYPE_ISSUE_PIPELINE_TASK_FILE_COMMIT";
    case Activity_Type.TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE:
      return "TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE";
    case Activity_Type.TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE:
      return "TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE";
    case Activity_Type.TYPE_MEMBER_CREATE:
      return "TYPE_MEMBER_CREATE";
    case Activity_Type.TYPE_MEMBER_ROLE_UPDATE:
      return "TYPE_MEMBER_ROLE_UPDATE";
    case Activity_Type.TYPE_MEMBER_ACTIVATE:
      return "TYPE_MEMBER_ACTIVATE";
    case Activity_Type.TYPE_MEMBER_DEACTIVATE:
      return "TYPE_MEMBER_DEACTIVATE";
    case Activity_Type.TYPE_PROJECT_REPOSITORY_PUSH:
      return "TYPE_PROJECT_REPOSITORY_PUSH";
    case Activity_Type.TYPE_PROJECT_DATABASE_TRANSFER:
      return "TYPE_PROJECT_DATABASE_TRANSFER";
    case Activity_Type.TYPE_PROJECT_MEMBER_CREATE:
      return "TYPE_PROJECT_MEMBER_CREATE";
    case Activity_Type.TYPE_PROJECT_MEMBER_DELETE:
      return "TYPE_PROJECT_MEMBER_DELETE";
    case Activity_Type.TYPE_PROJECT_MEMBER_ROLE_UPDATE:
      return "TYPE_PROJECT_MEMBER_ROLE_UPDATE";
    case Activity_Type.TYPE_SQL_EDITOR_QUERY:
      return "TYPE_SQL_EDITOR_QUERY";
    case Activity_Type.TYPE_DATABASE_RECOVERY_PITR_DONE:
      return "TYPE_DATABASE_RECOVERY_PITR_DONE";
    case Activity_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface ListDatabaseGroupsRequest {
  /**
   * The parent resource whose database groups are to be listed.
   * Format: projects/{project}
   * Using "projects/-" will list database groups across all projects.
   */
  parent: string;
  /**
   * Not used. The maximum number of anomalies to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 anomalies will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * Not used. A page token, received from a previous `ListDatabaseGroups` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListDatabaseGroups` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface ListDatabaseGroupsResponse {
  /** database_groups is the list of database groups. */
  databaseGroups: DatabaseGroup[];
  /**
   * Not used. A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface GetDatabaseGroupRequest {
  /**
   * The name of the database group to retrieve.
   * Format: projects/{project}/databaseGroups/{databaseGroup}
   */
  name: string;
  /** The view to return. Defaults to DATABASE_GROUP_VIEW_BASIC. */
  view: DatabaseGroupView;
}

export interface CreateDatabaseGroupRequest {
  /**
   * The parent resource where this database group will be created.
   * Format: projects/{project}
   */
  parent: string;
  /** The database group to create. */
  databaseGroup?:
    | DatabaseGroup
    | undefined;
  /**
   * The ID to use for the database group, which will become the final component of
   * the database group's resource name.
   *
   * This value should be 4-63 characters, and valid characters
   * are /[a-z][0-9]-/.
   */
  databaseGroupId: string;
  /** If set, validate the create request and preview the full database group response, but do not actually create it. */
  validateOnly: boolean;
}

export interface UpdateDatabaseGroupRequest {
  /**
   * The database group to update.
   *
   * The database group's `name` field is used to identify the database group to update.
   * Format: projects/{project}/databaseGroups/{databaseGroup}
   */
  databaseGroup?:
    | DatabaseGroup
    | undefined;
  /** The list of fields to update. */
  updateMask?: string[] | undefined;
}

export interface DeleteDatabaseGroupRequest {
  /**
   * The name of the database group to delete.
   * Format: projects/{project}/databaseGroups/{databaseGroup}
   */
  name: string;
}

export interface DatabaseGroup {
  /**
   * The name of the database group.
   * Format: projects/{project}/databaseGroups/{databaseGroup}
   */
  name: string;
  /**
   * The short name used in actual databases specified by users.
   * For example, the placeholder for db1_2010, db1_2021, db1_2023 will be "db1".
   */
  databasePlaceholder: string;
  /** The condition that is associated with this database group. */
  databaseExpr?:
    | Expr
    | undefined;
  /** The list of databases that match the database group condition. */
  matchedDatabases: DatabaseGroup_Database[];
  /** The list of databases that match the database group condition. */
  unmatchedDatabases: DatabaseGroup_Database[];
}

export interface DatabaseGroup_Database {
  /**
   * The resource name of the database.
   * Format: instances/{instance}/databases/{database}
   */
  name: string;
}

export interface CreateSchemaGroupRequest {
  /**
   * The parent resource where this schema group will be created.
   * Format: projects/{project}/databaseGroups/{databaseGroup}
   */
  parent: string;
  /** The schema group to create. */
  schemaGroup?:
    | SchemaGroup
    | undefined;
  /**
   * The ID to use for the schema group, which will become the final component of
   * the schema group's resource name.
   *
   * This value should be 4-63 characters, and valid characters
   * are /[a-z][0-9]-/.
   */
  schemaGroupId: string;
  /** If set, validate the create request and preview the full schema group response, but do not actually create it. */
  validateOnly: boolean;
}

export interface UpdateSchemaGroupRequest {
  /**
   * The schema group to update.
   *
   * The schema group's `name` field is used to identify the schema group to update.
   * Format: projects/{project}/databaseGroups/{databaseGroup}/schemaGroups/{schemaGroup}
   */
  schemaGroup?:
    | SchemaGroup
    | undefined;
  /** The list of fields to update. */
  updateMask?: string[] | undefined;
}

export interface DeleteSchemaGroupRequest {
  /**
   * The name of the schema group to delete.
   * Format: projects/{project}/databaseGroups/{databaseGroup}/schemaGroups/{schemaGroup}
   */
  name: string;
}

export interface ListSchemaGroupsRequest {
  /**
   * The parent resource whose schema groups are to be listed.
   * Format: projects/{project}/schemaGroups/{schemaGroup}
   */
  parent: string;
  /**
   * Not used. The maximum number of anomalies to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 anomalies will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * Not used. A page token, received from a previous `ListSchemaGroups` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListSchemaGroups` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface ListSchemaGroupsResponse {
  /** schema_groups is the list of schema groups. */
  schemaGroups: SchemaGroup[];
  /**
   * Not used. A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface GetSchemaGroupRequest {
  /**
   * The name of the database group to retrieve.
   * Format: projects/{project}/databaseGroups/{databaseGroup}/schemaGroups/{schemaGroup}
   */
  name: string;
  /** The view to return. Defaults to SCHEMA_GROUP_VIEW_BASIC. */
  view: SchemaGroupView;
}

export interface SchemaGroup {
  /**
   * The name of the schema group.
   * Format: projects/{project}/databaseGroups/{databaseGroup}/schemaGroups/{schemaGroup}
   */
  name: string;
  /**
   * The table condition that is associated with this schema group.
   * The table_placeholder in the sheet script will be rendered to the actual table name.
   */
  tableExpr?:
    | Expr
    | undefined;
  /**
   * The table placeholder used for rendering. For example, if set to "tbl", all the table name
   * "tbl" in the SQL script will be rendered to the actual table name.
   */
  tablePlaceholder: string;
  /** The list of databases that match the database group condition. */
  matchedTables: SchemaGroup_Table[];
  /** The list of databases that match the database group condition. */
  unmatchedTables: SchemaGroup_Table[];
}

/**
 * In the future, we can introduce schema_expr if users use schema (Postgres schema) for groups.
 * Its keyword will be {{SCHEMA}}.
 * All the expressions will be used to filter the schema objects in DatabaseSchema.
 */
export interface SchemaGroup_Table {
  /**
   * The resource name of the database.
   * Format: instances/{instance}/databases/{database}
   */
  database: string;
  schema: string;
  table: string;
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetProjectRequest();
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

  fromJSON(object: any): GetProjectRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetProjectRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<GetProjectRequest>): GetProjectRequest {
    return GetProjectRequest.fromPartial(base ?? {});
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListProjectsRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.pageToken = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
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

  create(base?: DeepPartial<ListProjectsRequest>): ListProjectsRequest {
    return ListProjectsRequest.fromPartial(base ?? {});
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListProjectsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.projects.push(Project.decode(reader, reader.uint32()));
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

  create(base?: DeepPartial<ListProjectsResponse>): ListProjectsResponse {
    return ListProjectsResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListProjectsResponse>): ListProjectsResponse {
    const message = createBaseListProjectsResponse();
    message.projects = object.projects?.map((e) => Project.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseSearchProjectsRequest(): SearchProjectsRequest {
  return { pageSize: 0, pageToken: "", filter: "" };
}

export const SearchProjectsRequest = {
  encode(message: SearchProjectsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pageSize !== 0) {
      writer.uint32(8).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(18).string(message.pageToken);
    }
    if (message.filter !== "") {
      writer.uint32(26).string(message.filter);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SearchProjectsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSearchProjectsRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.pageToken = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.filter = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SearchProjectsRequest {
    return {
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
      filter: isSet(object.filter) ? String(object.filter) : "",
    };
  },

  toJSON(message: SearchProjectsRequest): unknown {
    const obj: any = {};
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    message.filter !== undefined && (obj.filter = message.filter);
    return obj;
  },

  create(base?: DeepPartial<SearchProjectsRequest>): SearchProjectsRequest {
    return SearchProjectsRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SearchProjectsRequest>): SearchProjectsRequest {
    const message = createBaseSearchProjectsRequest();
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    message.filter = object.filter ?? "";
    return message;
  },
};

function createBaseSearchProjectsResponse(): SearchProjectsResponse {
  return { projects: [], nextPageToken: "" };
}

export const SearchProjectsResponse = {
  encode(message: SearchProjectsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.projects) {
      Project.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SearchProjectsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSearchProjectsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.projects.push(Project.decode(reader, reader.uint32()));
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

  fromJSON(object: any): SearchProjectsResponse {
    return {
      projects: Array.isArray(object?.projects) ? object.projects.map((e: any) => Project.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: SearchProjectsResponse): unknown {
    const obj: any = {};
    if (message.projects) {
      obj.projects = message.projects.map((e) => e ? Project.toJSON(e) : undefined);
    } else {
      obj.projects = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  create(base?: DeepPartial<SearchProjectsResponse>): SearchProjectsResponse {
    return SearchProjectsResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SearchProjectsResponse>): SearchProjectsResponse {
    const message = createBaseSearchProjectsResponse();
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateProjectRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.project = Project.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.projectId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<CreateProjectRequest>): CreateProjectRequest {
    return CreateProjectRequest.fromPartial(base ?? {});
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateProjectRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.project = Project.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<UpdateProjectRequest>): UpdateProjectRequest {
    return UpdateProjectRequest.fromPartial(base ?? {});
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
  return { name: "", force: false };
}

export const DeleteProjectRequest = {
  encode(message: DeleteProjectRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.force === true) {
      writer.uint32(16).bool(message.force);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteProjectRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteProjectRequest();
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
          if (tag !== 16) {
            break;
          }

          message.force = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DeleteProjectRequest {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      force: isSet(object.force) ? Boolean(object.force) : false,
    };
  },

  toJSON(message: DeleteProjectRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.force !== undefined && (obj.force = message.force);
    return obj;
  },

  create(base?: DeepPartial<DeleteProjectRequest>): DeleteProjectRequest {
    return DeleteProjectRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<DeleteProjectRequest>): DeleteProjectRequest {
    const message = createBaseDeleteProjectRequest();
    message.name = object.name ?? "";
    message.force = object.force ?? false;
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUndeleteProjectRequest();
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

  fromJSON(object: any): UndeleteProjectRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: UndeleteProjectRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<UndeleteProjectRequest>): UndeleteProjectRequest {
    return UndeleteProjectRequest.fromPartial(base ?? {});
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetIamPolicyRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.project = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<GetIamPolicyRequest>): GetIamPolicyRequest {
    return GetIamPolicyRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<GetIamPolicyRequest>): GetIamPolicyRequest {
    const message = createBaseGetIamPolicyRequest();
    message.project = object.project ?? "";
    return message;
  },
};

function createBaseBatchGetIamPolicyRequest(): BatchGetIamPolicyRequest {
  return { scope: "", names: [] };
}

export const BatchGetIamPolicyRequest = {
  encode(message: BatchGetIamPolicyRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.scope !== "") {
      writer.uint32(10).string(message.scope);
    }
    for (const v of message.names) {
      writer.uint32(18).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchGetIamPolicyRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchGetIamPolicyRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.scope = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.names.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BatchGetIamPolicyRequest {
    return {
      scope: isSet(object.scope) ? String(object.scope) : "",
      names: Array.isArray(object?.names) ? object.names.map((e: any) => String(e)) : [],
    };
  },

  toJSON(message: BatchGetIamPolicyRequest): unknown {
    const obj: any = {};
    message.scope !== undefined && (obj.scope = message.scope);
    if (message.names) {
      obj.names = message.names.map((e) => e);
    } else {
      obj.names = [];
    }
    return obj;
  },

  create(base?: DeepPartial<BatchGetIamPolicyRequest>): BatchGetIamPolicyRequest {
    return BatchGetIamPolicyRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<BatchGetIamPolicyRequest>): BatchGetIamPolicyRequest {
    const message = createBaseBatchGetIamPolicyRequest();
    message.scope = object.scope ?? "";
    message.names = object.names?.map((e) => e) || [];
    return message;
  },
};

function createBaseBatchGetIamPolicyResponse(): BatchGetIamPolicyResponse {
  return { policyResults: [] };
}

export const BatchGetIamPolicyResponse = {
  encode(message: BatchGetIamPolicyResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.policyResults) {
      BatchGetIamPolicyResponse_PolicyResult.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchGetIamPolicyResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchGetIamPolicyResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.policyResults.push(BatchGetIamPolicyResponse_PolicyResult.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BatchGetIamPolicyResponse {
    return {
      policyResults: Array.isArray(object?.policyResults)
        ? object.policyResults.map((e: any) => BatchGetIamPolicyResponse_PolicyResult.fromJSON(e))
        : [],
    };
  },

  toJSON(message: BatchGetIamPolicyResponse): unknown {
    const obj: any = {};
    if (message.policyResults) {
      obj.policyResults = message.policyResults.map((e) =>
        e ? BatchGetIamPolicyResponse_PolicyResult.toJSON(e) : undefined
      );
    } else {
      obj.policyResults = [];
    }
    return obj;
  },

  create(base?: DeepPartial<BatchGetIamPolicyResponse>): BatchGetIamPolicyResponse {
    return BatchGetIamPolicyResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<BatchGetIamPolicyResponse>): BatchGetIamPolicyResponse {
    const message = createBaseBatchGetIamPolicyResponse();
    message.policyResults = object.policyResults?.map((e) => BatchGetIamPolicyResponse_PolicyResult.fromPartial(e)) ||
      [];
    return message;
  },
};

function createBaseBatchGetIamPolicyResponse_PolicyResult(): BatchGetIamPolicyResponse_PolicyResult {
  return { project: "", policy: undefined };
}

export const BatchGetIamPolicyResponse_PolicyResult = {
  encode(message: BatchGetIamPolicyResponse_PolicyResult, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.project !== "") {
      writer.uint32(10).string(message.project);
    }
    if (message.policy !== undefined) {
      IamPolicy.encode(message.policy, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchGetIamPolicyResponse_PolicyResult {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchGetIamPolicyResponse_PolicyResult();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.project = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.policy = IamPolicy.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BatchGetIamPolicyResponse_PolicyResult {
    return {
      project: isSet(object.project) ? String(object.project) : "",
      policy: isSet(object.policy) ? IamPolicy.fromJSON(object.policy) : undefined,
    };
  },

  toJSON(message: BatchGetIamPolicyResponse_PolicyResult): unknown {
    const obj: any = {};
    message.project !== undefined && (obj.project = message.project);
    message.policy !== undefined && (obj.policy = message.policy ? IamPolicy.toJSON(message.policy) : undefined);
    return obj;
  },

  create(base?: DeepPartial<BatchGetIamPolicyResponse_PolicyResult>): BatchGetIamPolicyResponse_PolicyResult {
    return BatchGetIamPolicyResponse_PolicyResult.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<BatchGetIamPolicyResponse_PolicyResult>): BatchGetIamPolicyResponse_PolicyResult {
    const message = createBaseBatchGetIamPolicyResponse_PolicyResult();
    message.project = object.project ?? "";
    message.policy = (object.policy !== undefined && object.policy !== null)
      ? IamPolicy.fromPartial(object.policy)
      : undefined;
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSetIamPolicyRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.project = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.policy = IamPolicy.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<SetIamPolicyRequest>): SetIamPolicyRequest {
    return SetIamPolicyRequest.fromPartial(base ?? {});
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetDeploymentConfigRequest();
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

  fromJSON(object: any): GetDeploymentConfigRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetDeploymentConfigRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<GetDeploymentConfigRequest>): GetDeploymentConfigRequest {
    return GetDeploymentConfigRequest.fromPartial(base ?? {});
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateDeploymentConfigRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.config = DeploymentConfig.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<UpdateDeploymentConfigRequest>): UpdateDeploymentConfigRequest {
    return UpdateDeploymentConfigRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<UpdateDeploymentConfigRequest>): UpdateDeploymentConfigRequest {
    const message = createBaseUpdateDeploymentConfigRequest();
    message.config = (object.config !== undefined && object.config !== null)
      ? DeploymentConfig.fromPartial(object.config)
      : undefined;
    return message;
  },
};

function createBaseUpdateProjectGitOpsInfoRequest(): UpdateProjectGitOpsInfoRequest {
  return { projectGitopsInfo: undefined, updateMask: undefined, allowMissing: false };
}

export const UpdateProjectGitOpsInfoRequest = {
  encode(message: UpdateProjectGitOpsInfoRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.projectGitopsInfo !== undefined) {
      ProjectGitOpsInfo.encode(message.projectGitopsInfo, writer.uint32(18).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(26).fork()).ldelim();
    }
    if (message.allowMissing === true) {
      writer.uint32(32).bool(message.allowMissing);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateProjectGitOpsInfoRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateProjectGitOpsInfoRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 2:
          if (tag !== 18) {
            break;
          }

          message.projectGitopsInfo = ProjectGitOpsInfo.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag !== 32) {
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

  fromJSON(object: any): UpdateProjectGitOpsInfoRequest {
    return {
      projectGitopsInfo: isSet(object.projectGitopsInfo)
        ? ProjectGitOpsInfo.fromJSON(object.projectGitopsInfo)
        : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
      allowMissing: isSet(object.allowMissing) ? Boolean(object.allowMissing) : false,
    };
  },

  toJSON(message: UpdateProjectGitOpsInfoRequest): unknown {
    const obj: any = {};
    message.projectGitopsInfo !== undefined && (obj.projectGitopsInfo = message.projectGitopsInfo
      ? ProjectGitOpsInfo.toJSON(message.projectGitopsInfo)
      : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    message.allowMissing !== undefined && (obj.allowMissing = message.allowMissing);
    return obj;
  },

  create(base?: DeepPartial<UpdateProjectGitOpsInfoRequest>): UpdateProjectGitOpsInfoRequest {
    return UpdateProjectGitOpsInfoRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<UpdateProjectGitOpsInfoRequest>): UpdateProjectGitOpsInfoRequest {
    const message = createBaseUpdateProjectGitOpsInfoRequest();
    message.projectGitopsInfo = (object.projectGitopsInfo !== undefined && object.projectGitopsInfo !== null)
      ? ProjectGitOpsInfo.fromPartial(object.projectGitopsInfo)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    message.allowMissing = object.allowMissing ?? false;
    return message;
  },
};

function createBaseUnsetProjectGitOpsInfoRequest(): UnsetProjectGitOpsInfoRequest {
  return { name: "" };
}

export const UnsetProjectGitOpsInfoRequest = {
  encode(message: UnsetProjectGitOpsInfoRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UnsetProjectGitOpsInfoRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUnsetProjectGitOpsInfoRequest();
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

  fromJSON(object: any): UnsetProjectGitOpsInfoRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: UnsetProjectGitOpsInfoRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<UnsetProjectGitOpsInfoRequest>): UnsetProjectGitOpsInfoRequest {
    return UnsetProjectGitOpsInfoRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<UnsetProjectGitOpsInfoRequest>): UnsetProjectGitOpsInfoRequest {
    const message = createBaseUnsetProjectGitOpsInfoRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseGetProjectGitOpsInfoRequest(): GetProjectGitOpsInfoRequest {
  return { name: "" };
}

export const GetProjectGitOpsInfoRequest = {
  encode(message: GetProjectGitOpsInfoRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetProjectGitOpsInfoRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetProjectGitOpsInfoRequest();
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

  fromJSON(object: any): GetProjectGitOpsInfoRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetProjectGitOpsInfoRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<GetProjectGitOpsInfoRequest>): GetProjectGitOpsInfoRequest {
    return GetProjectGitOpsInfoRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<GetProjectGitOpsInfoRequest>): GetProjectGitOpsInfoRequest {
    const message = createBaseGetProjectGitOpsInfoRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseSetupSQLReviewCIRequest(): SetupSQLReviewCIRequest {
  return { name: "" };
}

export const SetupSQLReviewCIRequest = {
  encode(message: SetupSQLReviewCIRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SetupSQLReviewCIRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSetupSQLReviewCIRequest();
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

  fromJSON(object: any): SetupSQLReviewCIRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: SetupSQLReviewCIRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<SetupSQLReviewCIRequest>): SetupSQLReviewCIRequest {
    return SetupSQLReviewCIRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SetupSQLReviewCIRequest>): SetupSQLReviewCIRequest {
    const message = createBaseSetupSQLReviewCIRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseSetupSQLReviewCIResponse(): SetupSQLReviewCIResponse {
  return { pullRequestUrl: "" };
}

export const SetupSQLReviewCIResponse = {
  encode(message: SetupSQLReviewCIResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pullRequestUrl !== "") {
      writer.uint32(10).string(message.pullRequestUrl);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SetupSQLReviewCIResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSetupSQLReviewCIResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.pullRequestUrl = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SetupSQLReviewCIResponse {
    return { pullRequestUrl: isSet(object.pullRequestUrl) ? String(object.pullRequestUrl) : "" };
  },

  toJSON(message: SetupSQLReviewCIResponse): unknown {
    const obj: any = {};
    message.pullRequestUrl !== undefined && (obj.pullRequestUrl = message.pullRequestUrl);
    return obj;
  },

  create(base?: DeepPartial<SetupSQLReviewCIResponse>): SetupSQLReviewCIResponse {
    return SetupSQLReviewCIResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SetupSQLReviewCIResponse>): SetupSQLReviewCIResponse {
    const message = createBaseSetupSQLReviewCIResponse();
    message.pullRequestUrl = object.pullRequestUrl ?? "";
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
    schemaChange: 0,
    webhooks: [],
    dataCategoryConfigId: "",
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
    if (message.schemaChange !== 0) {
      writer.uint32(80).int32(message.schemaChange);
    }
    for (const v of message.webhooks) {
      Webhook.encode(v!, writer.uint32(90).fork()).ldelim();
    }
    if (message.dataCategoryConfigId !== "") {
      writer.uint32(98).string(message.dataCategoryConfigId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Project {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseProject();
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
        case 3:
          if (tag !== 24) {
            break;
          }

          message.state = reader.int32() as any;
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.title = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.key = reader.string();
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.workflow = reader.int32() as any;
          continue;
        case 7:
          if (tag !== 56) {
            break;
          }

          message.visibility = reader.int32() as any;
          continue;
        case 8:
          if (tag !== 64) {
            break;
          }

          message.tenantMode = reader.int32() as any;
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.dbNameTemplate = reader.string();
          continue;
        case 10:
          if (tag !== 80) {
            break;
          }

          message.schemaChange = reader.int32() as any;
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.webhooks.push(Webhook.decode(reader, reader.uint32()));
          continue;
        case 12:
          if (tag !== 98) {
            break;
          }

          message.dataCategoryConfigId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
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
      schemaChange: isSet(object.schemaChange) ? schemaChangeFromJSON(object.schemaChange) : 0,
      webhooks: Array.isArray(object?.webhooks) ? object.webhooks.map((e: any) => Webhook.fromJSON(e)) : [],
      dataCategoryConfigId: isSet(object.dataCategoryConfigId) ? String(object.dataCategoryConfigId) : "",
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
    message.schemaChange !== undefined && (obj.schemaChange = schemaChangeToJSON(message.schemaChange));
    if (message.webhooks) {
      obj.webhooks = message.webhooks.map((e) => e ? Webhook.toJSON(e) : undefined);
    } else {
      obj.webhooks = [];
    }
    message.dataCategoryConfigId !== undefined && (obj.dataCategoryConfigId = message.dataCategoryConfigId);
    return obj;
  },

  create(base?: DeepPartial<Project>): Project {
    return Project.fromPartial(base ?? {});
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
    message.schemaChange = object.schemaChange ?? 0;
    message.webhooks = object.webhooks?.map((e) => Webhook.fromPartial(e)) || [];
    message.dataCategoryConfigId = object.dataCategoryConfigId ?? "";
    return message;
  },
};

function createBaseAddWebhookRequest(): AddWebhookRequest {
  return { project: "", webhook: undefined };
}

export const AddWebhookRequest = {
  encode(message: AddWebhookRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.project !== "") {
      writer.uint32(10).string(message.project);
    }
    if (message.webhook !== undefined) {
      Webhook.encode(message.webhook, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): AddWebhookRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAddWebhookRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.project = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.webhook = Webhook.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): AddWebhookRequest {
    return {
      project: isSet(object.project) ? String(object.project) : "",
      webhook: isSet(object.webhook) ? Webhook.fromJSON(object.webhook) : undefined,
    };
  },

  toJSON(message: AddWebhookRequest): unknown {
    const obj: any = {};
    message.project !== undefined && (obj.project = message.project);
    message.webhook !== undefined && (obj.webhook = message.webhook ? Webhook.toJSON(message.webhook) : undefined);
    return obj;
  },

  create(base?: DeepPartial<AddWebhookRequest>): AddWebhookRequest {
    return AddWebhookRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<AddWebhookRequest>): AddWebhookRequest {
    const message = createBaseAddWebhookRequest();
    message.project = object.project ?? "";
    message.webhook = (object.webhook !== undefined && object.webhook !== null)
      ? Webhook.fromPartial(object.webhook)
      : undefined;
    return message;
  },
};

function createBaseUpdateWebhookRequest(): UpdateWebhookRequest {
  return { webhook: undefined, updateMask: undefined };
}

export const UpdateWebhookRequest = {
  encode(message: UpdateWebhookRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.webhook !== undefined) {
      Webhook.encode(message.webhook, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateWebhookRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateWebhookRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.webhook = Webhook.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UpdateWebhookRequest {
    return {
      webhook: isSet(object.webhook) ? Webhook.fromJSON(object.webhook) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateWebhookRequest): unknown {
    const obj: any = {};
    message.webhook !== undefined && (obj.webhook = message.webhook ? Webhook.toJSON(message.webhook) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  create(base?: DeepPartial<UpdateWebhookRequest>): UpdateWebhookRequest {
    return UpdateWebhookRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<UpdateWebhookRequest>): UpdateWebhookRequest {
    const message = createBaseUpdateWebhookRequest();
    message.webhook = (object.webhook !== undefined && object.webhook !== null)
      ? Webhook.fromPartial(object.webhook)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseRemoveWebhookRequest(): RemoveWebhookRequest {
  return { webhook: undefined };
}

export const RemoveWebhookRequest = {
  encode(message: RemoveWebhookRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.webhook !== undefined) {
      Webhook.encode(message.webhook, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RemoveWebhookRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRemoveWebhookRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.webhook = Webhook.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): RemoveWebhookRequest {
    return { webhook: isSet(object.webhook) ? Webhook.fromJSON(object.webhook) : undefined };
  },

  toJSON(message: RemoveWebhookRequest): unknown {
    const obj: any = {};
    message.webhook !== undefined && (obj.webhook = message.webhook ? Webhook.toJSON(message.webhook) : undefined);
    return obj;
  },

  create(base?: DeepPartial<RemoveWebhookRequest>): RemoveWebhookRequest {
    return RemoveWebhookRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<RemoveWebhookRequest>): RemoveWebhookRequest {
    const message = createBaseRemoveWebhookRequest();
    message.webhook = (object.webhook !== undefined && object.webhook !== null)
      ? Webhook.fromPartial(object.webhook)
      : undefined;
    return message;
  },
};

function createBaseTestWebhookRequest(): TestWebhookRequest {
  return { project: "", webhook: undefined };
}

export const TestWebhookRequest = {
  encode(message: TestWebhookRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.project !== "") {
      writer.uint32(10).string(message.project);
    }
    if (message.webhook !== undefined) {
      Webhook.encode(message.webhook, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TestWebhookRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTestWebhookRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.project = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.webhook = Webhook.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TestWebhookRequest {
    return {
      project: isSet(object.project) ? String(object.project) : "",
      webhook: isSet(object.webhook) ? Webhook.fromJSON(object.webhook) : undefined,
    };
  },

  toJSON(message: TestWebhookRequest): unknown {
    const obj: any = {};
    message.project !== undefined && (obj.project = message.project);
    message.webhook !== undefined && (obj.webhook = message.webhook ? Webhook.toJSON(message.webhook) : undefined);
    return obj;
  },

  create(base?: DeepPartial<TestWebhookRequest>): TestWebhookRequest {
    return TestWebhookRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<TestWebhookRequest>): TestWebhookRequest {
    const message = createBaseTestWebhookRequest();
    message.project = object.project ?? "";
    message.webhook = (object.webhook !== undefined && object.webhook !== null)
      ? Webhook.fromPartial(object.webhook)
      : undefined;
    return message;
  },
};

function createBaseTestWebhookResponse(): TestWebhookResponse {
  return { error: "" };
}

export const TestWebhookResponse = {
  encode(message: TestWebhookResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.error !== "") {
      writer.uint32(10).string(message.error);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TestWebhookResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTestWebhookResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.error = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TestWebhookResponse {
    return { error: isSet(object.error) ? String(object.error) : "" };
  },

  toJSON(message: TestWebhookResponse): unknown {
    const obj: any = {};
    message.error !== undefined && (obj.error = message.error);
    return obj;
  },

  create(base?: DeepPartial<TestWebhookResponse>): TestWebhookResponse {
    return TestWebhookResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<TestWebhookResponse>): TestWebhookResponse {
    const message = createBaseTestWebhookResponse();
    message.error = object.error ?? "";
    return message;
  },
};

function createBaseWebhook(): Webhook {
  return { name: "", type: 0, title: "", url: "", notificationTypes: [] };
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
    for (const v of message.notificationTypes) {
      writer.int32(v);
    }
    writer.ldelim();
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Webhook {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseWebhook();
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
          if (tag !== 16) {
            break;
          }

          message.type = reader.int32() as any;
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.title = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.url = reader.string();
          continue;
        case 5:
          if (tag === 40) {
            message.notificationTypes.push(reader.int32() as any);

            continue;
          }

          if (tag === 42) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.notificationTypes.push(reader.int32() as any);
            }

            continue;
          }

          break;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Webhook {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      type: isSet(object.type) ? webhook_TypeFromJSON(object.type) : 0,
      title: isSet(object.title) ? String(object.title) : "",
      url: isSet(object.url) ? String(object.url) : "",
      notificationTypes: Array.isArray(object?.notificationTypes)
        ? object.notificationTypes.map((e: any) => activity_TypeFromJSON(e))
        : [],
    };
  },

  toJSON(message: Webhook): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.type !== undefined && (obj.type = webhook_TypeToJSON(message.type));
    message.title !== undefined && (obj.title = message.title);
    message.url !== undefined && (obj.url = message.url);
    if (message.notificationTypes) {
      obj.notificationTypes = message.notificationTypes.map((e) => activity_TypeToJSON(e));
    } else {
      obj.notificationTypes = [];
    }
    return obj;
  },

  create(base?: DeepPartial<Webhook>): Webhook {
    return Webhook.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Webhook>): Webhook {
    const message = createBaseWebhook();
    message.name = object.name ?? "";
    message.type = object.type ?? 0;
    message.title = object.title ?? "";
    message.url = object.url ?? "";
    message.notificationTypes = object.notificationTypes?.map((e) => e) || [];
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeploymentConfig();
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

          message.title = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.schedule = Schedule.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<DeploymentConfig>): DeploymentConfig {
    return DeploymentConfig.fromPartial(base ?? {});
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSchedule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.deployments.push(ScheduleDeployment.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<Schedule>): Schedule {
    return Schedule.fromPartial(base ?? {});
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseScheduleDeployment();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.title = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.spec = DeploymentSpec.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<ScheduleDeployment>): ScheduleDeployment {
    return ScheduleDeployment.fromPartial(base ?? {});
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeploymentSpec();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.labelSelector = LabelSelector.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<DeploymentSpec>): DeploymentSpec {
    return DeploymentSpec.fromPartial(base ?? {});
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseLabelSelector();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.matchExpressions.push(LabelSelectorRequirement.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<LabelSelector>): LabelSelector {
    return LabelSelector.fromPartial(base ?? {});
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseLabelSelectorRequirement();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.operator = reader.int32() as any;
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.values.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<LabelSelectorRequirement>): LabelSelectorRequirement {
    return LabelSelectorRequirement.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<LabelSelectorRequirement>): LabelSelectorRequirement {
    const message = createBaseLabelSelectorRequirement();
    message.key = object.key ?? "";
    message.operator = object.operator ?? 0;
    message.values = object.values?.map((e) => e) || [];
    return message;
  },
};

function createBaseActivity(): Activity {
  return {};
}

export const Activity = {
  encode(_: Activity, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Activity {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseActivity();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): Activity {
    return {};
  },

  toJSON(_: Activity): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<Activity>): Activity {
    return Activity.fromPartial(base ?? {});
  },

  fromPartial(_: DeepPartial<Activity>): Activity {
    const message = createBaseActivity();
    return message;
  },
};

function createBaseListDatabaseGroupsRequest(): ListDatabaseGroupsRequest {
  return { parent: "", pageSize: 0, pageToken: "" };
}

export const ListDatabaseGroupsRequest = {
  encode(message: ListDatabaseGroupsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): ListDatabaseGroupsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListDatabaseGroupsRequest();
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

          message.pageSize = reader.int32();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.pageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListDatabaseGroupsRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListDatabaseGroupsRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    return obj;
  },

  create(base?: DeepPartial<ListDatabaseGroupsRequest>): ListDatabaseGroupsRequest {
    return ListDatabaseGroupsRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListDatabaseGroupsRequest>): ListDatabaseGroupsRequest {
    const message = createBaseListDatabaseGroupsRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListDatabaseGroupsResponse(): ListDatabaseGroupsResponse {
  return { databaseGroups: [], nextPageToken: "" };
}

export const ListDatabaseGroupsResponse = {
  encode(message: ListDatabaseGroupsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.databaseGroups) {
      DatabaseGroup.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListDatabaseGroupsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListDatabaseGroupsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.databaseGroups.push(DatabaseGroup.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListDatabaseGroupsResponse {
    return {
      databaseGroups: Array.isArray(object?.databaseGroups)
        ? object.databaseGroups.map((e: any) => DatabaseGroup.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListDatabaseGroupsResponse): unknown {
    const obj: any = {};
    if (message.databaseGroups) {
      obj.databaseGroups = message.databaseGroups.map((e) => e ? DatabaseGroup.toJSON(e) : undefined);
    } else {
      obj.databaseGroups = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  create(base?: DeepPartial<ListDatabaseGroupsResponse>): ListDatabaseGroupsResponse {
    return ListDatabaseGroupsResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListDatabaseGroupsResponse>): ListDatabaseGroupsResponse {
    const message = createBaseListDatabaseGroupsResponse();
    message.databaseGroups = object.databaseGroups?.map((e) => DatabaseGroup.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseGetDatabaseGroupRequest(): GetDatabaseGroupRequest {
  return { name: "", view: 0 };
}

export const GetDatabaseGroupRequest = {
  encode(message: GetDatabaseGroupRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.view !== 0) {
      writer.uint32(16).int32(message.view);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetDatabaseGroupRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetDatabaseGroupRequest();
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
          if (tag !== 16) {
            break;
          }

          message.view = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GetDatabaseGroupRequest {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      view: isSet(object.view) ? databaseGroupViewFromJSON(object.view) : 0,
    };
  },

  toJSON(message: GetDatabaseGroupRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.view !== undefined && (obj.view = databaseGroupViewToJSON(message.view));
    return obj;
  },

  create(base?: DeepPartial<GetDatabaseGroupRequest>): GetDatabaseGroupRequest {
    return GetDatabaseGroupRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<GetDatabaseGroupRequest>): GetDatabaseGroupRequest {
    const message = createBaseGetDatabaseGroupRequest();
    message.name = object.name ?? "";
    message.view = object.view ?? 0;
    return message;
  },
};

function createBaseCreateDatabaseGroupRequest(): CreateDatabaseGroupRequest {
  return { parent: "", databaseGroup: undefined, databaseGroupId: "", validateOnly: false };
}

export const CreateDatabaseGroupRequest = {
  encode(message: CreateDatabaseGroupRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.databaseGroup !== undefined) {
      DatabaseGroup.encode(message.databaseGroup, writer.uint32(18).fork()).ldelim();
    }
    if (message.databaseGroupId !== "") {
      writer.uint32(26).string(message.databaseGroupId);
    }
    if (message.validateOnly === true) {
      writer.uint32(32).bool(message.validateOnly);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateDatabaseGroupRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateDatabaseGroupRequest();
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

          message.databaseGroup = DatabaseGroup.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.databaseGroupId = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.validateOnly = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateDatabaseGroupRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      databaseGroup: isSet(object.databaseGroup) ? DatabaseGroup.fromJSON(object.databaseGroup) : undefined,
      databaseGroupId: isSet(object.databaseGroupId) ? String(object.databaseGroupId) : "",
      validateOnly: isSet(object.validateOnly) ? Boolean(object.validateOnly) : false,
    };
  },

  toJSON(message: CreateDatabaseGroupRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.databaseGroup !== undefined &&
      (obj.databaseGroup = message.databaseGroup ? DatabaseGroup.toJSON(message.databaseGroup) : undefined);
    message.databaseGroupId !== undefined && (obj.databaseGroupId = message.databaseGroupId);
    message.validateOnly !== undefined && (obj.validateOnly = message.validateOnly);
    return obj;
  },

  create(base?: DeepPartial<CreateDatabaseGroupRequest>): CreateDatabaseGroupRequest {
    return CreateDatabaseGroupRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<CreateDatabaseGroupRequest>): CreateDatabaseGroupRequest {
    const message = createBaseCreateDatabaseGroupRequest();
    message.parent = object.parent ?? "";
    message.databaseGroup = (object.databaseGroup !== undefined && object.databaseGroup !== null)
      ? DatabaseGroup.fromPartial(object.databaseGroup)
      : undefined;
    message.databaseGroupId = object.databaseGroupId ?? "";
    message.validateOnly = object.validateOnly ?? false;
    return message;
  },
};

function createBaseUpdateDatabaseGroupRequest(): UpdateDatabaseGroupRequest {
  return { databaseGroup: undefined, updateMask: undefined };
}

export const UpdateDatabaseGroupRequest = {
  encode(message: UpdateDatabaseGroupRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.databaseGroup !== undefined) {
      DatabaseGroup.encode(message.databaseGroup, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateDatabaseGroupRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateDatabaseGroupRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.databaseGroup = DatabaseGroup.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UpdateDatabaseGroupRequest {
    return {
      databaseGroup: isSet(object.databaseGroup) ? DatabaseGroup.fromJSON(object.databaseGroup) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateDatabaseGroupRequest): unknown {
    const obj: any = {};
    message.databaseGroup !== undefined &&
      (obj.databaseGroup = message.databaseGroup ? DatabaseGroup.toJSON(message.databaseGroup) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  create(base?: DeepPartial<UpdateDatabaseGroupRequest>): UpdateDatabaseGroupRequest {
    return UpdateDatabaseGroupRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<UpdateDatabaseGroupRequest>): UpdateDatabaseGroupRequest {
    const message = createBaseUpdateDatabaseGroupRequest();
    message.databaseGroup = (object.databaseGroup !== undefined && object.databaseGroup !== null)
      ? DatabaseGroup.fromPartial(object.databaseGroup)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseDeleteDatabaseGroupRequest(): DeleteDatabaseGroupRequest {
  return { name: "" };
}

export const DeleteDatabaseGroupRequest = {
  encode(message: DeleteDatabaseGroupRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteDatabaseGroupRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteDatabaseGroupRequest();
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

  fromJSON(object: any): DeleteDatabaseGroupRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: DeleteDatabaseGroupRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<DeleteDatabaseGroupRequest>): DeleteDatabaseGroupRequest {
    return DeleteDatabaseGroupRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<DeleteDatabaseGroupRequest>): DeleteDatabaseGroupRequest {
    const message = createBaseDeleteDatabaseGroupRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseDatabaseGroup(): DatabaseGroup {
  return { name: "", databasePlaceholder: "", databaseExpr: undefined, matchedDatabases: [], unmatchedDatabases: [] };
}

export const DatabaseGroup = {
  encode(message: DatabaseGroup, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.databasePlaceholder !== "") {
      writer.uint32(18).string(message.databasePlaceholder);
    }
    if (message.databaseExpr !== undefined) {
      Expr.encode(message.databaseExpr, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.matchedDatabases) {
      DatabaseGroup_Database.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.unmatchedDatabases) {
      DatabaseGroup_Database.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DatabaseGroup {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDatabaseGroup();
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

          message.databasePlaceholder = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.databaseExpr = Expr.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.matchedDatabases.push(DatabaseGroup_Database.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.unmatchedDatabases.push(DatabaseGroup_Database.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DatabaseGroup {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      databasePlaceholder: isSet(object.databasePlaceholder) ? String(object.databasePlaceholder) : "",
      databaseExpr: isSet(object.databaseExpr) ? Expr.fromJSON(object.databaseExpr) : undefined,
      matchedDatabases: Array.isArray(object?.matchedDatabases)
        ? object.matchedDatabases.map((e: any) => DatabaseGroup_Database.fromJSON(e))
        : [],
      unmatchedDatabases: Array.isArray(object?.unmatchedDatabases)
        ? object.unmatchedDatabases.map((e: any) => DatabaseGroup_Database.fromJSON(e))
        : [],
    };
  },

  toJSON(message: DatabaseGroup): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.databasePlaceholder !== undefined && (obj.databasePlaceholder = message.databasePlaceholder);
    message.databaseExpr !== undefined &&
      (obj.databaseExpr = message.databaseExpr ? Expr.toJSON(message.databaseExpr) : undefined);
    if (message.matchedDatabases) {
      obj.matchedDatabases = message.matchedDatabases.map((e) => e ? DatabaseGroup_Database.toJSON(e) : undefined);
    } else {
      obj.matchedDatabases = [];
    }
    if (message.unmatchedDatabases) {
      obj.unmatchedDatabases = message.unmatchedDatabases.map((e) => e ? DatabaseGroup_Database.toJSON(e) : undefined);
    } else {
      obj.unmatchedDatabases = [];
    }
    return obj;
  },

  create(base?: DeepPartial<DatabaseGroup>): DatabaseGroup {
    return DatabaseGroup.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<DatabaseGroup>): DatabaseGroup {
    const message = createBaseDatabaseGroup();
    message.name = object.name ?? "";
    message.databasePlaceholder = object.databasePlaceholder ?? "";
    message.databaseExpr = (object.databaseExpr !== undefined && object.databaseExpr !== null)
      ? Expr.fromPartial(object.databaseExpr)
      : undefined;
    message.matchedDatabases = object.matchedDatabases?.map((e) => DatabaseGroup_Database.fromPartial(e)) || [];
    message.unmatchedDatabases = object.unmatchedDatabases?.map((e) => DatabaseGroup_Database.fromPartial(e)) || [];
    return message;
  },
};

function createBaseDatabaseGroup_Database(): DatabaseGroup_Database {
  return { name: "" };
}

export const DatabaseGroup_Database = {
  encode(message: DatabaseGroup_Database, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DatabaseGroup_Database {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDatabaseGroup_Database();
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

  fromJSON(object: any): DatabaseGroup_Database {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: DatabaseGroup_Database): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<DatabaseGroup_Database>): DatabaseGroup_Database {
    return DatabaseGroup_Database.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<DatabaseGroup_Database>): DatabaseGroup_Database {
    const message = createBaseDatabaseGroup_Database();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseCreateSchemaGroupRequest(): CreateSchemaGroupRequest {
  return { parent: "", schemaGroup: undefined, schemaGroupId: "", validateOnly: false };
}

export const CreateSchemaGroupRequest = {
  encode(message: CreateSchemaGroupRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.schemaGroup !== undefined) {
      SchemaGroup.encode(message.schemaGroup, writer.uint32(18).fork()).ldelim();
    }
    if (message.schemaGroupId !== "") {
      writer.uint32(26).string(message.schemaGroupId);
    }
    if (message.validateOnly === true) {
      writer.uint32(32).bool(message.validateOnly);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateSchemaGroupRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateSchemaGroupRequest();
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

          message.schemaGroup = SchemaGroup.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.schemaGroupId = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.validateOnly = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateSchemaGroupRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      schemaGroup: isSet(object.schemaGroup) ? SchemaGroup.fromJSON(object.schemaGroup) : undefined,
      schemaGroupId: isSet(object.schemaGroupId) ? String(object.schemaGroupId) : "",
      validateOnly: isSet(object.validateOnly) ? Boolean(object.validateOnly) : false,
    };
  },

  toJSON(message: CreateSchemaGroupRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.schemaGroup !== undefined &&
      (obj.schemaGroup = message.schemaGroup ? SchemaGroup.toJSON(message.schemaGroup) : undefined);
    message.schemaGroupId !== undefined && (obj.schemaGroupId = message.schemaGroupId);
    message.validateOnly !== undefined && (obj.validateOnly = message.validateOnly);
    return obj;
  },

  create(base?: DeepPartial<CreateSchemaGroupRequest>): CreateSchemaGroupRequest {
    return CreateSchemaGroupRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<CreateSchemaGroupRequest>): CreateSchemaGroupRequest {
    const message = createBaseCreateSchemaGroupRequest();
    message.parent = object.parent ?? "";
    message.schemaGroup = (object.schemaGroup !== undefined && object.schemaGroup !== null)
      ? SchemaGroup.fromPartial(object.schemaGroup)
      : undefined;
    message.schemaGroupId = object.schemaGroupId ?? "";
    message.validateOnly = object.validateOnly ?? false;
    return message;
  },
};

function createBaseUpdateSchemaGroupRequest(): UpdateSchemaGroupRequest {
  return { schemaGroup: undefined, updateMask: undefined };
}

export const UpdateSchemaGroupRequest = {
  encode(message: UpdateSchemaGroupRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schemaGroup !== undefined) {
      SchemaGroup.encode(message.schemaGroup, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateSchemaGroupRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateSchemaGroupRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.schemaGroup = SchemaGroup.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UpdateSchemaGroupRequest {
    return {
      schemaGroup: isSet(object.schemaGroup) ? SchemaGroup.fromJSON(object.schemaGroup) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateSchemaGroupRequest): unknown {
    const obj: any = {};
    message.schemaGroup !== undefined &&
      (obj.schemaGroup = message.schemaGroup ? SchemaGroup.toJSON(message.schemaGroup) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  create(base?: DeepPartial<UpdateSchemaGroupRequest>): UpdateSchemaGroupRequest {
    return UpdateSchemaGroupRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<UpdateSchemaGroupRequest>): UpdateSchemaGroupRequest {
    const message = createBaseUpdateSchemaGroupRequest();
    message.schemaGroup = (object.schemaGroup !== undefined && object.schemaGroup !== null)
      ? SchemaGroup.fromPartial(object.schemaGroup)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseDeleteSchemaGroupRequest(): DeleteSchemaGroupRequest {
  return { name: "" };
}

export const DeleteSchemaGroupRequest = {
  encode(message: DeleteSchemaGroupRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteSchemaGroupRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteSchemaGroupRequest();
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

  fromJSON(object: any): DeleteSchemaGroupRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: DeleteSchemaGroupRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<DeleteSchemaGroupRequest>): DeleteSchemaGroupRequest {
    return DeleteSchemaGroupRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<DeleteSchemaGroupRequest>): DeleteSchemaGroupRequest {
    const message = createBaseDeleteSchemaGroupRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListSchemaGroupsRequest(): ListSchemaGroupsRequest {
  return { parent: "", pageSize: 0, pageToken: "" };
}

export const ListSchemaGroupsRequest = {
  encode(message: ListSchemaGroupsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): ListSchemaGroupsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListSchemaGroupsRequest();
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

          message.pageSize = reader.int32();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.pageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListSchemaGroupsRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListSchemaGroupsRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    return obj;
  },

  create(base?: DeepPartial<ListSchemaGroupsRequest>): ListSchemaGroupsRequest {
    return ListSchemaGroupsRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListSchemaGroupsRequest>): ListSchemaGroupsRequest {
    const message = createBaseListSchemaGroupsRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListSchemaGroupsResponse(): ListSchemaGroupsResponse {
  return { schemaGroups: [], nextPageToken: "" };
}

export const ListSchemaGroupsResponse = {
  encode(message: ListSchemaGroupsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.schemaGroups) {
      SchemaGroup.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListSchemaGroupsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListSchemaGroupsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.schemaGroups.push(SchemaGroup.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListSchemaGroupsResponse {
    return {
      schemaGroups: Array.isArray(object?.schemaGroups)
        ? object.schemaGroups.map((e: any) => SchemaGroup.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListSchemaGroupsResponse): unknown {
    const obj: any = {};
    if (message.schemaGroups) {
      obj.schemaGroups = message.schemaGroups.map((e) => e ? SchemaGroup.toJSON(e) : undefined);
    } else {
      obj.schemaGroups = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  create(base?: DeepPartial<ListSchemaGroupsResponse>): ListSchemaGroupsResponse {
    return ListSchemaGroupsResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListSchemaGroupsResponse>): ListSchemaGroupsResponse {
    const message = createBaseListSchemaGroupsResponse();
    message.schemaGroups = object.schemaGroups?.map((e) => SchemaGroup.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseGetSchemaGroupRequest(): GetSchemaGroupRequest {
  return { name: "", view: 0 };
}

export const GetSchemaGroupRequest = {
  encode(message: GetSchemaGroupRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.view !== 0) {
      writer.uint32(16).int32(message.view);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetSchemaGroupRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetSchemaGroupRequest();
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
          if (tag !== 16) {
            break;
          }

          message.view = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GetSchemaGroupRequest {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      view: isSet(object.view) ? schemaGroupViewFromJSON(object.view) : 0,
    };
  },

  toJSON(message: GetSchemaGroupRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.view !== undefined && (obj.view = schemaGroupViewToJSON(message.view));
    return obj;
  },

  create(base?: DeepPartial<GetSchemaGroupRequest>): GetSchemaGroupRequest {
    return GetSchemaGroupRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<GetSchemaGroupRequest>): GetSchemaGroupRequest {
    const message = createBaseGetSchemaGroupRequest();
    message.name = object.name ?? "";
    message.view = object.view ?? 0;
    return message;
  },
};

function createBaseSchemaGroup(): SchemaGroup {
  return { name: "", tableExpr: undefined, tablePlaceholder: "", matchedTables: [], unmatchedTables: [] };
}

export const SchemaGroup = {
  encode(message: SchemaGroup, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.tableExpr !== undefined) {
      Expr.encode(message.tableExpr, writer.uint32(18).fork()).ldelim();
    }
    if (message.tablePlaceholder !== "") {
      writer.uint32(26).string(message.tablePlaceholder);
    }
    for (const v of message.matchedTables) {
      SchemaGroup_Table.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.unmatchedTables) {
      SchemaGroup_Table.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SchemaGroup {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSchemaGroup();
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

          message.tableExpr = Expr.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.tablePlaceholder = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.matchedTables.push(SchemaGroup_Table.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.unmatchedTables.push(SchemaGroup_Table.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SchemaGroup {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      tableExpr: isSet(object.tableExpr) ? Expr.fromJSON(object.tableExpr) : undefined,
      tablePlaceholder: isSet(object.tablePlaceholder) ? String(object.tablePlaceholder) : "",
      matchedTables: Array.isArray(object?.matchedTables)
        ? object.matchedTables.map((e: any) => SchemaGroup_Table.fromJSON(e))
        : [],
      unmatchedTables: Array.isArray(object?.unmatchedTables)
        ? object.unmatchedTables.map((e: any) => SchemaGroup_Table.fromJSON(e))
        : [],
    };
  },

  toJSON(message: SchemaGroup): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.tableExpr !== undefined && (obj.tableExpr = message.tableExpr ? Expr.toJSON(message.tableExpr) : undefined);
    message.tablePlaceholder !== undefined && (obj.tablePlaceholder = message.tablePlaceholder);
    if (message.matchedTables) {
      obj.matchedTables = message.matchedTables.map((e) => e ? SchemaGroup_Table.toJSON(e) : undefined);
    } else {
      obj.matchedTables = [];
    }
    if (message.unmatchedTables) {
      obj.unmatchedTables = message.unmatchedTables.map((e) => e ? SchemaGroup_Table.toJSON(e) : undefined);
    } else {
      obj.unmatchedTables = [];
    }
    return obj;
  },

  create(base?: DeepPartial<SchemaGroup>): SchemaGroup {
    return SchemaGroup.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SchemaGroup>): SchemaGroup {
    const message = createBaseSchemaGroup();
    message.name = object.name ?? "";
    message.tableExpr = (object.tableExpr !== undefined && object.tableExpr !== null)
      ? Expr.fromPartial(object.tableExpr)
      : undefined;
    message.tablePlaceholder = object.tablePlaceholder ?? "";
    message.matchedTables = object.matchedTables?.map((e) => SchemaGroup_Table.fromPartial(e)) || [];
    message.unmatchedTables = object.unmatchedTables?.map((e) => SchemaGroup_Table.fromPartial(e)) || [];
    return message;
  },
};

function createBaseSchemaGroup_Table(): SchemaGroup_Table {
  return { database: "", schema: "", table: "" };
}

export const SchemaGroup_Table = {
  encode(message: SchemaGroup_Table, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.database !== "") {
      writer.uint32(10).string(message.database);
    }
    if (message.schema !== "") {
      writer.uint32(18).string(message.schema);
    }
    if (message.table !== "") {
      writer.uint32(26).string(message.table);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SchemaGroup_Table {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSchemaGroup_Table();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.database = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.schema = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.table = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SchemaGroup_Table {
    return {
      database: isSet(object.database) ? String(object.database) : "",
      schema: isSet(object.schema) ? String(object.schema) : "",
      table: isSet(object.table) ? String(object.table) : "",
    };
  },

  toJSON(message: SchemaGroup_Table): unknown {
    const obj: any = {};
    message.database !== undefined && (obj.database = message.database);
    message.schema !== undefined && (obj.schema = message.schema);
    message.table !== undefined && (obj.table = message.table);
    return obj;
  },

  create(base?: DeepPartial<SchemaGroup_Table>): SchemaGroup_Table {
    return SchemaGroup_Table.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SchemaGroup_Table>): SchemaGroup_Table {
    const message = createBaseSchemaGroup_Table();
    message.database = object.database ?? "";
    message.schema = object.schema ?? "";
    message.table = object.table ?? "";
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
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              23,
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
            ]),
          ],
        },
      },
    },
    listProjects: {
      name: "ListProjects",
      requestType: ListProjectsRequest,
      requestStream: false,
      responseType: ListProjectsResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          578365826: [new Uint8Array([14, 18, 12, 47, 118, 49, 47, 112, 114, 111, 106, 101, 99, 116, 115])],
        },
      },
    },
    /** Search for projects that the caller has both projects.get permission on, and also satisfy the specified query. */
    searchProjects: {
      name: "SearchProjects",
      requestType: SearchProjectsRequest,
      requestStream: false,
      responseType: SearchProjectsResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          578365826: [
            new Uint8Array([
              21,
              18,
              19,
              47,
              118,
              49,
              47,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              58,
              115,
              101,
              97,
              114,
              99,
              104,
            ]),
          ],
        },
      },
    },
    createProject: {
      name: "CreateProject",
      requestType: CreateProjectRequest,
      requestStream: false,
      responseType: Project,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          578365826: [
            new Uint8Array([
              23,
              58,
              7,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              34,
              12,
              47,
              118,
              49,
              47,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
            ]),
          ],
        },
      },
    },
    updateProject: {
      name: "UpdateProject",
      requestType: UpdateProjectRequest,
      requestStream: false,
      responseType: Project,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([
              19,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              44,
              117,
              112,
              100,
              97,
              116,
              101,
              95,
              109,
              97,
              115,
              107,
            ]),
          ],
          578365826: [
            new Uint8Array([
              40,
              58,
              7,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              50,
              29,
              47,
              118,
              49,
              47,
              123,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
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
              125,
            ]),
          ],
        },
      },
    },
    deleteProject: {
      name: "DeleteProject",
      requestType: DeleteProjectRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              23,
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
            ]),
          ],
        },
      },
    },
    undeleteProject: {
      name: "UndeleteProject",
      requestType: UndeleteProjectRequest,
      requestStream: false,
      responseType: Project,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              35,
              58,
              1,
              42,
              34,
              30,
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
              125,
              58,
              117,
              110,
              100,
              101,
              108,
              101,
              116,
              101,
            ]),
          ],
        },
      },
    },
    getIamPolicy: {
      name: "GetIamPolicy",
      requestType: GetIamPolicyRequest,
      requestStream: false,
      responseType: IamPolicy,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              39,
              18,
              37,
              47,
              118,
              49,
              47,
              123,
              112,
              114,
              111,
              106,
              101,
              99,
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
              58,
              103,
              101,
              116,
              73,
              97,
              109,
              80,
              111,
              108,
              105,
              99,
              121,
            ]),
          ],
        },
      },
    },
    batchGetIamPolicy: {
      name: "BatchGetIamPolicy",
      requestType: BatchGetIamPolicyRequest,
      requestStream: false,
      responseType: BatchGetIamPolicyResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              38,
              18,
              36,
              47,
              118,
              49,
              47,
              123,
              115,
              99,
              111,
              112,
              101,
              61,
              42,
              47,
              42,
              125,
              47,
              105,
              97,
              109,
              80,
              111,
              108,
              105,
              99,
              105,
              101,
              115,
              58,
              98,
              97,
              116,
              99,
              104,
              71,
              101,
              116,
            ]),
          ],
        },
      },
    },
    setIamPolicy: {
      name: "SetIamPolicy",
      requestType: SetIamPolicyRequest,
      requestStream: false,
      responseType: IamPolicy,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              42,
              58,
              1,
              42,
              34,
              37,
              47,
              118,
              49,
              47,
              123,
              112,
              114,
              111,
              106,
              101,
              99,
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
              58,
              115,
              101,
              116,
              73,
              97,
              109,
              80,
              111,
              108,
              105,
              99,
              121,
            ]),
          ],
        },
      },
    },
    getDeploymentConfig: {
      name: "GetDeploymentConfig",
      requestType: GetDeploymentConfigRequest,
      requestStream: false,
      responseType: DeploymentConfig,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              40,
              18,
              38,
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
              100,
              101,
              112,
              108,
              111,
              121,
              109,
              101,
              110,
              116,
              67,
              111,
              110,
              102,
              105,
              103,
              125,
            ]),
          ],
        },
      },
    },
    updateDeploymentConfig: {
      name: "UpdateDeploymentConfig",
      requestType: UpdateDeploymentConfigRequest,
      requestStream: false,
      responseType: DeploymentConfig,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              55,
              58,
              6,
              99,
              111,
              110,
              102,
              105,
              103,
              50,
              45,
              47,
              118,
              49,
              47,
              123,
              99,
              111,
              110,
              102,
              105,
              103,
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
              100,
              101,
              112,
              108,
              111,
              121,
              109,
              101,
              110,
              116,
              67,
              111,
              110,
              102,
              105,
              103,
              125,
            ]),
          ],
        },
      },
    },
    addWebhook: {
      name: "AddWebhook",
      requestType: AddWebhookRequest,
      requestStream: false,
      responseType: Project,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              40,
              58,
              1,
              42,
              34,
              35,
              47,
              118,
              49,
              47,
              123,
              112,
              114,
              111,
              106,
              101,
              99,
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
              58,
              97,
              100,
              100,
              87,
              101,
              98,
              104,
              111,
              111,
              107,
            ]),
          ],
        },
      },
    },
    updateWebhook: {
      name: "UpdateWebhook",
      requestType: UpdateWebhookRequest,
      requestStream: false,
      responseType: Project,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([
              19,
              119,
              101,
              98,
              104,
              111,
              111,
              107,
              44,
              117,
              112,
              100,
              97,
              116,
              101,
              95,
              109,
              97,
              115,
              107,
            ]),
          ],
          578365826: [
            new Uint8Array([
              59,
              58,
              1,
              42,
              34,
              54,
              47,
              118,
              49,
              47,
              123,
              119,
              101,
              98,
              104,
              111,
              111,
              107,
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
              119,
              101,
              98,
              104,
              111,
              111,
              107,
              115,
              47,
              42,
              125,
              58,
              117,
              112,
              100,
              97,
              116,
              101,
              87,
              101,
              98,
              104,
              111,
              111,
              107,
            ]),
          ],
        },
      },
    },
    removeWebhook: {
      name: "RemoveWebhook",
      requestType: RemoveWebhookRequest,
      requestStream: false,
      responseType: Project,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              59,
              58,
              1,
              42,
              34,
              54,
              47,
              118,
              49,
              47,
              123,
              119,
              101,
              98,
              104,
              111,
              111,
              107,
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
              119,
              101,
              98,
              104,
              111,
              111,
              107,
              115,
              47,
              42,
              125,
              58,
              114,
              101,
              109,
              111,
              118,
              101,
              87,
              101,
              98,
              104,
              111,
              111,
              107,
            ]),
          ],
        },
      },
    },
    testWebhook: {
      name: "TestWebhook",
      requestType: TestWebhookRequest,
      requestStream: false,
      responseType: TestWebhookResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              41,
              58,
              1,
              42,
              34,
              36,
              47,
              118,
              49,
              47,
              123,
              112,
              114,
              111,
              106,
              101,
              99,
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
              58,
              116,
              101,
              115,
              116,
              87,
              101,
              98,
              104,
              111,
              111,
              107,
            ]),
          ],
        },
      },
    },
    updateProjectGitOpsInfo: {
      name: "UpdateProjectGitOpsInfo",
      requestType: UpdateProjectGitOpsInfoRequest,
      requestStream: false,
      responseType: ProjectGitOpsInfo,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              57,
              58,
              1,
              42,
              34,
              52,
              47,
              118,
              49,
              47,
              123,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              95,
              103,
              105,
              116,
              111,
              112,
              115,
              95,
              105,
              110,
              102,
              111,
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
              103,
              105,
              116,
              79,
              112,
              115,
              73,
              110,
              102,
              111,
              125,
            ]),
          ],
        },
      },
    },
    unsetProjectGitOpsInfo: {
      name: "UnsetProjectGitOpsInfo",
      requestType: UnsetProjectGitOpsInfoRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
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
              103,
              105,
              116,
              79,
              112,
              115,
              73,
              110,
              102,
              111,
              125,
            ]),
          ],
        },
      },
    },
    setupProjectSQLReviewCI: {
      name: "SetupProjectSQLReviewCI",
      requestType: SetupSQLReviewCIRequest,
      requestStream: false,
      responseType: SetupSQLReviewCIResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              54,
              58,
              1,
              42,
              50,
              49,
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
              103,
              105,
              116,
              79,
              112,
              115,
              73,
              110,
              102,
              111,
              125,
              58,
              115,
              101,
              116,
              117,
              112,
              83,
              81,
              76,
              82,
              101,
              118,
              105,
              101,
              119,
              67,
              73,
            ]),
          ],
        },
      },
    },
    getProjectGitOpsInfo: {
      name: "GetProjectGitOpsInfo",
      requestType: GetProjectGitOpsInfoRequest,
      requestStream: false,
      responseType: ProjectGitOpsInfo,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
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
              103,
              105,
              116,
              79,
              112,
              115,
              73,
              110,
              102,
              111,
              125,
            ]),
          ],
        },
      },
    },
    listDatabaseGroups: {
      name: "ListDatabaseGroups",
      requestType: ListDatabaseGroupsRequest,
      requestStream: false,
      responseType: ListDatabaseGroupsResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([6, 112, 97, 114, 101, 110, 116])],
          578365826: [
            new Uint8Array([
              40,
              18,
              38,
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
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              71,
              114,
              111,
              117,
              112,
              115,
            ]),
          ],
        },
      },
    },
    getDatabaseGroup: {
      name: "GetDatabaseGroup",
      requestType: GetDatabaseGroupRequest,
      requestStream: false,
      responseType: DatabaseGroup,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              40,
              18,
              38,
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
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              71,
              114,
              111,
              117,
              112,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    createDatabaseGroup: {
      name: "CreateDatabaseGroup",
      requestType: CreateDatabaseGroupRequest,
      requestStream: false,
      responseType: DatabaseGroup,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([
              20,
              112,
              97,
              114,
              101,
              110,
              116,
              44,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              71,
              114,
              111,
              117,
              112,
            ]),
          ],
          578365826: [
            new Uint8Array([
              56,
              58,
              14,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              95,
              103,
              114,
              111,
              117,
              112,
              34,
              38,
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
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              71,
              114,
              111,
              117,
              112,
              115,
            ]),
          ],
        },
      },
    },
    updateDatabaseGroup: {
      name: "UpdateDatabaseGroup",
      requestType: UpdateDatabaseGroupRequest,
      requestStream: false,
      responseType: DatabaseGroup,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([
              26,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              95,
              103,
              114,
              111,
              117,
              112,
              44,
              117,
              112,
              100,
              97,
              116,
              101,
              95,
              109,
              97,
              115,
              107,
            ]),
          ],
          578365826: [
            new Uint8Array([
              71,
              58,
              14,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              95,
              103,
              114,
              111,
              117,
              112,
              50,
              53,
              47,
              118,
              49,
              47,
              123,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              95,
              103,
              114,
              111,
              117,
              112,
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
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              71,
              114,
              111,
              117,
              112,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    deleteDatabaseGroup: {
      name: "DeleteDatabaseGroup",
      requestType: DeleteDatabaseGroupRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              40,
              42,
              38,
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
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              71,
              114,
              111,
              117,
              112,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    listSchemaGroups: {
      name: "ListSchemaGroups",
      requestType: ListSchemaGroupsRequest,
      requestStream: false,
      responseType: ListSchemaGroupsResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([6, 112, 97, 114, 101, 110, 116])],
          578365826: [
            new Uint8Array([
              55,
              18,
              53,
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
              47,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              71,
              114,
              111,
              117,
              112,
              115,
              47,
              42,
              125,
              47,
              115,
              99,
              104,
              101,
              109,
              97,
              71,
              114,
              111,
              117,
              112,
              115,
            ]),
          ],
        },
      },
    },
    getSchemaGroup: {
      name: "GetSchemaGroup",
      requestType: GetSchemaGroupRequest,
      requestStream: false,
      responseType: SchemaGroup,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              55,
              18,
              53,
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
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              71,
              114,
              111,
              117,
              112,
              115,
              47,
              42,
              47,
              115,
              99,
              104,
              101,
              109,
              97,
              71,
              114,
              111,
              117,
              112,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    createSchemaGroup: {
      name: "CreateSchemaGroup",
      requestType: CreateSchemaGroupRequest,
      requestStream: false,
      responseType: SchemaGroup,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([
              20,
              112,
              97,
              114,
              101,
              110,
              116,
              44,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              71,
              114,
              111,
              117,
              112,
            ]),
          ],
          578365826: [
            new Uint8Array([
              69,
              58,
              12,
              115,
              99,
              104,
              101,
              109,
              97,
              95,
              103,
              114,
              111,
              117,
              112,
              34,
              53,
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
              47,
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              71,
              114,
              111,
              117,
              112,
              115,
              47,
              42,
              125,
              47,
              115,
              99,
              104,
              101,
              109,
              97,
              71,
              114,
              111,
              117,
              112,
              115,
            ]),
          ],
        },
      },
    },
    updateSchemaGroup: {
      name: "UpdateSchemaGroup",
      requestType: UpdateSchemaGroupRequest,
      requestStream: false,
      responseType: SchemaGroup,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([
              24,
              115,
              99,
              104,
              101,
              109,
              97,
              95,
              103,
              114,
              111,
              117,
              112,
              44,
              117,
              112,
              100,
              97,
              116,
              101,
              95,
              109,
              97,
              115,
              107,
            ]),
          ],
          578365826: [
            new Uint8Array([
              82,
              58,
              12,
              115,
              99,
              104,
              101,
              109,
              97,
              95,
              103,
              114,
              111,
              117,
              112,
              50,
              66,
              47,
              118,
              49,
              47,
              123,
              115,
              99,
              104,
              101,
              109,
              97,
              95,
              103,
              114,
              111,
              117,
              112,
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
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              71,
              114,
              111,
              117,
              112,
              115,
              47,
              42,
              47,
              115,
              99,
              104,
              101,
              109,
              97,
              71,
              114,
              111,
              117,
              112,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    deleteSchemaGroup: {
      name: "DeleteSchemaGroup",
      requestType: DeleteSchemaGroupRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              55,
              42,
              53,
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
              100,
              97,
              116,
              97,
              98,
              97,
              115,
              101,
              71,
              114,
              111,
              117,
              112,
              115,
              47,
              42,
              47,
              115,
              99,
              104,
              101,
              109,
              97,
              71,
              114,
              111,
              117,
              112,
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

export interface ProjectServiceImplementation<CallContextExt = {}> {
  getProject(request: GetProjectRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Project>>;
  listProjects(
    request: ListProjectsRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ListProjectsResponse>>;
  /** Search for projects that the caller has both projects.get permission on, and also satisfy the specified query. */
  searchProjects(
    request: SearchProjectsRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<SearchProjectsResponse>>;
  createProject(request: CreateProjectRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Project>>;
  updateProject(request: UpdateProjectRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Project>>;
  deleteProject(request: DeleteProjectRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Empty>>;
  undeleteProject(
    request: UndeleteProjectRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Project>>;
  getIamPolicy(request: GetIamPolicyRequest, context: CallContext & CallContextExt): Promise<DeepPartial<IamPolicy>>;
  batchGetIamPolicy(
    request: BatchGetIamPolicyRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<BatchGetIamPolicyResponse>>;
  setIamPolicy(request: SetIamPolicyRequest, context: CallContext & CallContextExt): Promise<DeepPartial<IamPolicy>>;
  getDeploymentConfig(
    request: GetDeploymentConfigRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<DeploymentConfig>>;
  updateDeploymentConfig(
    request: UpdateDeploymentConfigRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<DeploymentConfig>>;
  addWebhook(request: AddWebhookRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Project>>;
  updateWebhook(request: UpdateWebhookRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Project>>;
  removeWebhook(request: RemoveWebhookRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Project>>;
  testWebhook(
    request: TestWebhookRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<TestWebhookResponse>>;
  updateProjectGitOpsInfo(
    request: UpdateProjectGitOpsInfoRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ProjectGitOpsInfo>>;
  unsetProjectGitOpsInfo(
    request: UnsetProjectGitOpsInfoRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Empty>>;
  setupProjectSQLReviewCI(
    request: SetupSQLReviewCIRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<SetupSQLReviewCIResponse>>;
  getProjectGitOpsInfo(
    request: GetProjectGitOpsInfoRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ProjectGitOpsInfo>>;
  listDatabaseGroups(
    request: ListDatabaseGroupsRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ListDatabaseGroupsResponse>>;
  getDatabaseGroup(
    request: GetDatabaseGroupRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<DatabaseGroup>>;
  createDatabaseGroup(
    request: CreateDatabaseGroupRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<DatabaseGroup>>;
  updateDatabaseGroup(
    request: UpdateDatabaseGroupRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<DatabaseGroup>>;
  deleteDatabaseGroup(
    request: DeleteDatabaseGroupRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Empty>>;
  listSchemaGroups(
    request: ListSchemaGroupsRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ListSchemaGroupsResponse>>;
  getSchemaGroup(
    request: GetSchemaGroupRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<SchemaGroup>>;
  createSchemaGroup(
    request: CreateSchemaGroupRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<SchemaGroup>>;
  updateSchemaGroup(
    request: UpdateSchemaGroupRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<SchemaGroup>>;
  deleteSchemaGroup(
    request: DeleteSchemaGroupRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Empty>>;
}

export interface ProjectServiceClient<CallOptionsExt = {}> {
  getProject(request: DeepPartial<GetProjectRequest>, options?: CallOptions & CallOptionsExt): Promise<Project>;
  listProjects(
    request: DeepPartial<ListProjectsRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ListProjectsResponse>;
  /** Search for projects that the caller has both projects.get permission on, and also satisfy the specified query. */
  searchProjects(
    request: DeepPartial<SearchProjectsRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<SearchProjectsResponse>;
  createProject(request: DeepPartial<CreateProjectRequest>, options?: CallOptions & CallOptionsExt): Promise<Project>;
  updateProject(request: DeepPartial<UpdateProjectRequest>, options?: CallOptions & CallOptionsExt): Promise<Project>;
  deleteProject(request: DeepPartial<DeleteProjectRequest>, options?: CallOptions & CallOptionsExt): Promise<Empty>;
  undeleteProject(
    request: DeepPartial<UndeleteProjectRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Project>;
  getIamPolicy(request: DeepPartial<GetIamPolicyRequest>, options?: CallOptions & CallOptionsExt): Promise<IamPolicy>;
  batchGetIamPolicy(
    request: DeepPartial<BatchGetIamPolicyRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<BatchGetIamPolicyResponse>;
  setIamPolicy(request: DeepPartial<SetIamPolicyRequest>, options?: CallOptions & CallOptionsExt): Promise<IamPolicy>;
  getDeploymentConfig(
    request: DeepPartial<GetDeploymentConfigRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<DeploymentConfig>;
  updateDeploymentConfig(
    request: DeepPartial<UpdateDeploymentConfigRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<DeploymentConfig>;
  addWebhook(request: DeepPartial<AddWebhookRequest>, options?: CallOptions & CallOptionsExt): Promise<Project>;
  updateWebhook(request: DeepPartial<UpdateWebhookRequest>, options?: CallOptions & CallOptionsExt): Promise<Project>;
  removeWebhook(request: DeepPartial<RemoveWebhookRequest>, options?: CallOptions & CallOptionsExt): Promise<Project>;
  testWebhook(
    request: DeepPartial<TestWebhookRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<TestWebhookResponse>;
  updateProjectGitOpsInfo(
    request: DeepPartial<UpdateProjectGitOpsInfoRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ProjectGitOpsInfo>;
  unsetProjectGitOpsInfo(
    request: DeepPartial<UnsetProjectGitOpsInfoRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Empty>;
  setupProjectSQLReviewCI(
    request: DeepPartial<SetupSQLReviewCIRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<SetupSQLReviewCIResponse>;
  getProjectGitOpsInfo(
    request: DeepPartial<GetProjectGitOpsInfoRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ProjectGitOpsInfo>;
  listDatabaseGroups(
    request: DeepPartial<ListDatabaseGroupsRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ListDatabaseGroupsResponse>;
  getDatabaseGroup(
    request: DeepPartial<GetDatabaseGroupRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<DatabaseGroup>;
  createDatabaseGroup(
    request: DeepPartial<CreateDatabaseGroupRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<DatabaseGroup>;
  updateDatabaseGroup(
    request: DeepPartial<UpdateDatabaseGroupRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<DatabaseGroup>;
  deleteDatabaseGroup(
    request: DeepPartial<DeleteDatabaseGroupRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Empty>;
  listSchemaGroups(
    request: DeepPartial<ListSchemaGroupsRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ListSchemaGroupsResponse>;
  getSchemaGroup(
    request: DeepPartial<GetSchemaGroupRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<SchemaGroup>;
  createSchemaGroup(
    request: DeepPartial<CreateSchemaGroupRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<SchemaGroup>;
  updateSchemaGroup(
    request: DeepPartial<UpdateSchemaGroupRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<SchemaGroup>;
  deleteSchemaGroup(
    request: DeepPartial<DeleteSchemaGroupRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Empty>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
