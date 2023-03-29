/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { State, stateFromJSON, stateToJSON } from "./common";
import { ProjectGitOpsInfo } from "./externalvs_service";

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

export interface SetProjectGitOpsInfoRequest {
  /**
   * The name of the project.
   * Format: projects/{project}
   */
  project: string;
  /** The binding for the project and external version control. */
  projectGitopsInfo?: ProjectGitOpsInfo;
}

export interface GetProjectGitOpsInfoRequest {
  /**
   * The name of the project.
   * Format: projects/{project}
   */
  project: string;
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
  webhooks: Webhook[];
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

export interface AddWebhookRequest {
  /**
   * The name of the project to add the webhook to.
   * Format: projects/{project}
   */
  project: string;
  /** The webhook to add. */
  webhook?: Webhook;
}

export interface UpdateWebhookRequest {
  /**
   * The name of the project which owns the webhook to be updated.
   * Format: projects/{project}
   */
  project: string;
  /**
   * The webhook to modify.
   * Identified by its url.
   */
  webhook?: Webhook;
  /** The list of fields to update. */
  updateMask?: string[];
}

export interface RemoveWebhookRequest {
  /**
   * The name of the project to remove the webhook from.
   * Format: projects/{project}
   */
  project: string;
  /** The webhook to remove. Identified by its url. */
  webhook?: Webhook;
}

export interface TestWebhookRequest {
  /**
   * The name of the project which owns the webhook to test.
   * Format: projects/{project}
   */
  project: string;
  /** The webhook to test. Identified by its url. */
  webhook?: Webhook;
}

export interface TestWebhookResponse {
  /** The result of the test, empty if the test is successful. */
  error: string;
}

export interface Webhook {
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
          if (tag != 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
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
          if (tag != 8) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.pageToken = reader.string();
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.showDeleted = reader.bool();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
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
          if (tag != 10) {
            break;
          }

          message.projects.push(Project.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.nextPageToken = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
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
          if (tag != 10) {
            break;
          }

          message.project = Project.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.projectId = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
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
          if (tag != 10) {
            break;
          }

          message.project = Project.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteProjectRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<DeleteProjectRequest>): DeleteProjectRequest {
    return DeleteProjectRequest.fromPartial(base ?? {});
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUndeleteProjectRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
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
          if (tag != 10) {
            break;
          }

          message.project = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
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
          if (tag != 10) {
            break;
          }

          message.project = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.policy = IamPolicy.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
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
          if (tag != 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
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
          if (tag != 10) {
            break;
          }

          message.config = DeploymentConfig.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
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

function createBaseSetProjectGitOpsInfoRequest(): SetProjectGitOpsInfoRequest {
  return { project: "", projectGitopsInfo: undefined };
}

export const SetProjectGitOpsInfoRequest = {
  encode(message: SetProjectGitOpsInfoRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.project !== "") {
      writer.uint32(10).string(message.project);
    }
    if (message.projectGitopsInfo !== undefined) {
      ProjectGitOpsInfo.encode(message.projectGitopsInfo, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SetProjectGitOpsInfoRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSetProjectGitOpsInfoRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.project = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.projectGitopsInfo = ProjectGitOpsInfo.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SetProjectGitOpsInfoRequest {
    return {
      project: isSet(object.project) ? String(object.project) : "",
      projectGitopsInfo: isSet(object.projectGitopsInfo)
        ? ProjectGitOpsInfo.fromJSON(object.projectGitopsInfo)
        : undefined,
    };
  },

  toJSON(message: SetProjectGitOpsInfoRequest): unknown {
    const obj: any = {};
    message.project !== undefined && (obj.project = message.project);
    message.projectGitopsInfo !== undefined && (obj.projectGitopsInfo = message.projectGitopsInfo
      ? ProjectGitOpsInfo.toJSON(message.projectGitopsInfo)
      : undefined);
    return obj;
  },

  create(base?: DeepPartial<SetProjectGitOpsInfoRequest>): SetProjectGitOpsInfoRequest {
    return SetProjectGitOpsInfoRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SetProjectGitOpsInfoRequest>): SetProjectGitOpsInfoRequest {
    const message = createBaseSetProjectGitOpsInfoRequest();
    message.project = object.project ?? "";
    message.projectGitopsInfo = (object.projectGitopsInfo !== undefined && object.projectGitopsInfo !== null)
      ? ProjectGitOpsInfo.fromPartial(object.projectGitopsInfo)
      : undefined;
    return message;
  },
};

function createBaseGetProjectGitOpsInfoRequest(): GetProjectGitOpsInfoRequest {
  return { project: "" };
}

export const GetProjectGitOpsInfoRequest = {
  encode(message: GetProjectGitOpsInfoRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.project !== "") {
      writer.uint32(10).string(message.project);
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
          if (tag != 10) {
            break;
          }

          message.project = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GetProjectGitOpsInfoRequest {
    return { project: isSet(object.project) ? String(object.project) : "" };
  },

  toJSON(message: GetProjectGitOpsInfoRequest): unknown {
    const obj: any = {};
    message.project !== undefined && (obj.project = message.project);
    return obj;
  },

  create(base?: DeepPartial<GetProjectGitOpsInfoRequest>): GetProjectGitOpsInfoRequest {
    return GetProjectGitOpsInfoRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<GetProjectGitOpsInfoRequest>): GetProjectGitOpsInfoRequest {
    const message = createBaseGetProjectGitOpsInfoRequest();
    message.project = object.project ?? "";
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
    webhooks: [],
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
    for (const v of message.webhooks) {
      Webhook.encode(v!, writer.uint32(98).fork()).ldelim();
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
          if (tag != 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.uid = reader.string();
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.state = reader.int32() as any;
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.title = reader.string();
          continue;
        case 5:
          if (tag != 42) {
            break;
          }

          message.key = reader.string();
          continue;
        case 6:
          if (tag != 48) {
            break;
          }

          message.workflow = reader.int32() as any;
          continue;
        case 7:
          if (tag != 56) {
            break;
          }

          message.visibility = reader.int32() as any;
          continue;
        case 8:
          if (tag != 64) {
            break;
          }

          message.tenantMode = reader.int32() as any;
          continue;
        case 9:
          if (tag != 74) {
            break;
          }

          message.dbNameTemplate = reader.string();
          continue;
        case 10:
          if (tag != 80) {
            break;
          }

          message.schemaVersion = reader.int32() as any;
          continue;
        case 11:
          if (tag != 88) {
            break;
          }

          message.schemaChange = reader.int32() as any;
          continue;
        case 12:
          if (tag != 98) {
            break;
          }

          message.webhooks.push(Webhook.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
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
      schemaVersion: isSet(object.schemaVersion) ? schemaVersionFromJSON(object.schemaVersion) : 0,
      schemaChange: isSet(object.schemaChange) ? schemaChangeFromJSON(object.schemaChange) : 0,
      webhooks: Array.isArray(object?.webhooks) ? object.webhooks.map((e: any) => Webhook.fromJSON(e)) : [],
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
    if (message.webhooks) {
      obj.webhooks = message.webhooks.map((e) => e ? Webhook.toJSON(e) : undefined);
    } else {
      obj.webhooks = [];
    }
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
    message.schemaVersion = object.schemaVersion ?? 0;
    message.schemaChange = object.schemaChange ?? 0;
    message.webhooks = object.webhooks?.map((e) => Webhook.fromPartial(e)) || [];
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIamPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.bindings.push(Binding.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<IamPolicy>): IamPolicy {
    return IamPolicy.fromPartial(base ?? {});
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBinding();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 8) {
            break;
          }

          message.role = reader.int32() as any;
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.members.push(reader.string());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<Binding>): Binding {
    return Binding.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Binding>): Binding {
    const message = createBaseBinding();
    message.role = object.role ?? 0;
    message.members = object.members?.map((e) => e) || [];
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
          if (tag != 10) {
            break;
          }

          message.project = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.webhook = Webhook.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
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
  return { project: "", webhook: undefined, updateMask: undefined };
}

export const UpdateWebhookRequest = {
  encode(message: UpdateWebhookRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.project !== "") {
      writer.uint32(10).string(message.project);
    }
    if (message.webhook !== undefined) {
      Webhook.encode(message.webhook, writer.uint32(18).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(26).fork()).ldelim();
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
          if (tag != 10) {
            break;
          }

          message.project = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.webhook = Webhook.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UpdateWebhookRequest {
    return {
      project: isSet(object.project) ? String(object.project) : "",
      webhook: isSet(object.webhook) ? Webhook.fromJSON(object.webhook) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateWebhookRequest): unknown {
    const obj: any = {};
    message.project !== undefined && (obj.project = message.project);
    message.webhook !== undefined && (obj.webhook = message.webhook ? Webhook.toJSON(message.webhook) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  create(base?: DeepPartial<UpdateWebhookRequest>): UpdateWebhookRequest {
    return UpdateWebhookRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<UpdateWebhookRequest>): UpdateWebhookRequest {
    const message = createBaseUpdateWebhookRequest();
    message.project = object.project ?? "";
    message.webhook = (object.webhook !== undefined && object.webhook !== null)
      ? Webhook.fromPartial(object.webhook)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseRemoveWebhookRequest(): RemoveWebhookRequest {
  return { project: "", webhook: undefined };
}

export const RemoveWebhookRequest = {
  encode(message: RemoveWebhookRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.project !== "") {
      writer.uint32(10).string(message.project);
    }
    if (message.webhook !== undefined) {
      Webhook.encode(message.webhook, writer.uint32(18).fork()).ldelim();
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
          if (tag != 10) {
            break;
          }

          message.project = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.webhook = Webhook.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): RemoveWebhookRequest {
    return {
      project: isSet(object.project) ? String(object.project) : "",
      webhook: isSet(object.webhook) ? Webhook.fromJSON(object.webhook) : undefined,
    };
  },

  toJSON(message: RemoveWebhookRequest): unknown {
    const obj: any = {};
    message.project !== undefined && (obj.project = message.project);
    message.webhook !== undefined && (obj.webhook = message.webhook ? Webhook.toJSON(message.webhook) : undefined);
    return obj;
  },

  create(base?: DeepPartial<RemoveWebhookRequest>): RemoveWebhookRequest {
    return RemoveWebhookRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<RemoveWebhookRequest>): RemoveWebhookRequest {
    const message = createBaseRemoveWebhookRequest();
    message.project = object.project ?? "";
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
          if (tag != 10) {
            break;
          }

          message.project = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.webhook = Webhook.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
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
          if (tag != 10) {
            break;
          }

          message.error = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
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
  return { type: 0, title: "", url: "", notificationTypes: [] };
}

export const Webhook = {
  encode(message: Webhook, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.type !== 0) {
      writer.uint32(8).int32(message.type);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.url !== "") {
      writer.uint32(26).string(message.url);
    }
    writer.uint32(34).fork();
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
          if (tag != 8) {
            break;
          }

          message.type = reader.int32() as any;
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.title = reader.string();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.url = reader.string();
          continue;
        case 4:
          if (tag == 32) {
            message.notificationTypes.push(reader.int32() as any);
            continue;
          }

          if (tag == 34) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.notificationTypes.push(reader.int32() as any);
            }

            continue;
          }

          break;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Webhook {
    return {
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
          if (tag != 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.title = reader.string();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.schedule = Schedule.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
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
          if (tag != 10) {
            break;
          }

          message.deployments.push(ScheduleDeployment.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
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
          if (tag != 10) {
            break;
          }

          message.title = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.spec = DeploymentSpec.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
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
          if (tag != 10) {
            break;
          }

          message.labelSelector = LabelSelector.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
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
          if (tag != 10) {
            break;
          }

          message.matchExpressions.push(LabelSelectorRequirement.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
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
          if (tag != 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag != 16) {
            break;
          }

          message.operator = reader.int32() as any;
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.values.push(reader.string());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
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
      if ((tag & 7) == 4 || tag == 0) {
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
          578365826: [
            new Uint8Array([
              43,
              58,
              1,
              42,
              50,
              38,
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
              43,
              58,
              1,
              42,
              34,
              38,
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
    setProjectGitOpsInfo: {
      name: "SetProjectGitOpsInfo",
      requestType: SetProjectGitOpsInfoRequest,
      requestStream: false,
      responseType: ProjectGitOpsInfo,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              40,
              58,
              1,
              42,
              26,
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
            ]),
          ],
        },
      },
    },
    getProjectGitOpsInfo: {
      name: "GetProjectGitOpsInfo",
      requestType: SetProjectGitOpsInfoRequest,
      requestStream: false,
      responseType: ProjectGitOpsInfo,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              37,
              18,
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
  createProject(request: CreateProjectRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Project>>;
  updateProject(request: UpdateProjectRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Project>>;
  deleteProject(request: DeleteProjectRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Empty>>;
  undeleteProject(
    request: UndeleteProjectRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Project>>;
  getIamPolicy(request: GetIamPolicyRequest, context: CallContext & CallContextExt): Promise<DeepPartial<IamPolicy>>;
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
  setProjectGitOpsInfo(
    request: SetProjectGitOpsInfoRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ProjectGitOpsInfo>>;
  getProjectGitOpsInfo(
    request: SetProjectGitOpsInfoRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ProjectGitOpsInfo>>;
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
  setProjectGitOpsInfo(
    request: DeepPartial<SetProjectGitOpsInfoRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ProjectGitOpsInfo>;
  getProjectGitOpsInfo(
    request: DeepPartial<SetProjectGitOpsInfoRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ProjectGitOpsInfo>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
