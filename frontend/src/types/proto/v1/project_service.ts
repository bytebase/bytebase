/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { Expr } from "../google/type/expr";
import { State, stateFromJSON, stateToJSON, stateToNumber } from "./common";
import { GetIamPolicyRequest, IamPolicy, SetIamPolicyRequest } from "./iam_policy";

export const protobufPackage = "bytebase.v1";

export enum Workflow {
  WORKFLOW_UNSPECIFIED = "WORKFLOW_UNSPECIFIED",
  UI = "UI",
  VCS = "VCS",
  UNRECOGNIZED = "UNRECOGNIZED",
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

export function workflowToNumber(object: Workflow): number {
  switch (object) {
    case Workflow.WORKFLOW_UNSPECIFIED:
      return 0;
    case Workflow.UI:
      return 1;
    case Workflow.VCS:
      return 2;
    case Workflow.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum OperatorType {
  /** OPERATOR_TYPE_UNSPECIFIED - The operator is not specified. */
  OPERATOR_TYPE_UNSPECIFIED = "OPERATOR_TYPE_UNSPECIFIED",
  /** OPERATOR_TYPE_IN - The operator is "In". */
  OPERATOR_TYPE_IN = "OPERATOR_TYPE_IN",
  /** OPERATOR_TYPE_EXISTS - The operator is "Exists". */
  OPERATOR_TYPE_EXISTS = "OPERATOR_TYPE_EXISTS",
  /** OPERATOR_TYPE_NOT_IN - The operator is "Not In". */
  OPERATOR_TYPE_NOT_IN = "OPERATOR_TYPE_NOT_IN",
  UNRECOGNIZED = "UNRECOGNIZED",
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
    case 3:
    case "OPERATOR_TYPE_NOT_IN":
      return OperatorType.OPERATOR_TYPE_NOT_IN;
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
    case OperatorType.OPERATOR_TYPE_NOT_IN:
      return "OPERATOR_TYPE_NOT_IN";
    case OperatorType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function operatorTypeToNumber(object: OperatorType): number {
  switch (object) {
    case OperatorType.OPERATOR_TYPE_UNSPECIFIED:
      return 0;
    case OperatorType.OPERATOR_TYPE_IN:
      return 1;
    case OperatorType.OPERATOR_TYPE_EXISTS:
      return 2;
    case OperatorType.OPERATOR_TYPE_NOT_IN:
      return 3;
    case OperatorType.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum DatabaseGroupView {
  /**
   * DATABASE_GROUP_VIEW_UNSPECIFIED - The default / unset value.
   * The API will default to the BASIC view.
   */
  DATABASE_GROUP_VIEW_UNSPECIFIED = "DATABASE_GROUP_VIEW_UNSPECIFIED",
  /** DATABASE_GROUP_VIEW_BASIC - Include basic information about the database group, but exclude the list of matched databases and unmatched databases. */
  DATABASE_GROUP_VIEW_BASIC = "DATABASE_GROUP_VIEW_BASIC",
  /** DATABASE_GROUP_VIEW_FULL - Include everything. */
  DATABASE_GROUP_VIEW_FULL = "DATABASE_GROUP_VIEW_FULL",
  UNRECOGNIZED = "UNRECOGNIZED",
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

export function databaseGroupViewToNumber(object: DatabaseGroupView): number {
  switch (object) {
    case DatabaseGroupView.DATABASE_GROUP_VIEW_UNSPECIFIED:
      return 0;
    case DatabaseGroupView.DATABASE_GROUP_VIEW_BASIC:
      return 1;
    case DatabaseGroupView.DATABASE_GROUP_VIEW_FULL:
      return 2;
    case DatabaseGroupView.UNRECOGNIZED:
    default:
      return -1;
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
  /** Show deleted projects if specified. */
  showDeleted: boolean;
}

export interface SearchProjectsResponse {
  /** The projects from the specified request. */
  projects: Project[];
}

export interface CreateProjectRequest {
  /** The project to create. */
  project:
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
  project:
    | Project
    | undefined;
  /** The list of fields to update. */
  updateMask: string[] | undefined;
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
  policy: IamPolicy | undefined;
}

export interface GetDeploymentConfigRequest {
  /**
   * The name of the resource.
   * Format: projects/{project}/deploymentConfigs/default.
   */
  name: string;
}

export interface UpdateDeploymentConfigRequest {
  config: DeploymentConfig | undefined;
}

export interface Label {
  value: string;
  color: string;
  group: string;
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
  webhooks: Webhook[];
  dataClassificationConfigId: string;
  issueLabels: Label[];
  /** Force issue labels to be used when creating an issue. */
  forceIssueLabels: boolean;
  /** Allow modifying statement after issue is created. */
  allowModifyStatement: boolean;
  /** Enable auto resolve issue. */
  autoResolveIssue: boolean;
}

export interface AddWebhookRequest {
  /**
   * The name of the project to add the webhook to.
   * Format: projects/{project}
   */
  project: string;
  /** The webhook to add. */
  webhook: Webhook | undefined;
}

export interface UpdateWebhookRequest {
  /** The webhook to modify. */
  webhook:
    | Webhook
    | undefined;
  /** The list of fields to update. */
  updateMask: string[] | undefined;
}

export interface RemoveWebhookRequest {
  /** The webhook to remove. Identified by its url. */
  webhook: Webhook | undefined;
}

export interface TestWebhookRequest {
  /**
   * The name of the project which owns the webhook to test.
   * Format: projects/{project}
   */
  project: string;
  /** The webhook to test. Identified by its url. */
  webhook: Webhook | undefined;
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
   * if direct_message is set, the notification is sent directly
   * to the persons and url will be ignored.
   * IM integration setting should be set for this function to work.
   */
  directMessage: boolean;
  /**
   * notification_types is the list of activities types that the webhook is interested in.
   * Bytebase will only send notifications to the webhook if the activity type is in the list.
   * It should not be empty, and should be a subset of the following:
   * - TYPE_ISSUE_CREATED
   * - TYPE_ISSUE_STATUS_UPDATE
   * - TYPE_ISSUE_PIPELINE_STAGE_UPDATE
   * - TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE
   * - TYPE_ISSUE_FIELD_UPDATE
   * - TYPE_ISSUE_COMMENT_CREATE
   */
  notificationTypes: Activity_Type[];
}

export enum Webhook_Type {
  TYPE_UNSPECIFIED = "TYPE_UNSPECIFIED",
  TYPE_SLACK = "TYPE_SLACK",
  TYPE_DISCORD = "TYPE_DISCORD",
  TYPE_TEAMS = "TYPE_TEAMS",
  TYPE_DINGTALK = "TYPE_DINGTALK",
  TYPE_FEISHU = "TYPE_FEISHU",
  TYPE_WECOM = "TYPE_WECOM",
  TYPE_CUSTOM = "TYPE_CUSTOM",
  UNRECOGNIZED = "UNRECOGNIZED",
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

export function webhook_TypeToNumber(object: Webhook_Type): number {
  switch (object) {
    case Webhook_Type.TYPE_UNSPECIFIED:
      return 0;
    case Webhook_Type.TYPE_SLACK:
      return 1;
    case Webhook_Type.TYPE_DISCORD:
      return 2;
    case Webhook_Type.TYPE_TEAMS:
      return 3;
    case Webhook_Type.TYPE_DINGTALK:
      return 4;
    case Webhook_Type.TYPE_FEISHU:
      return 5;
    case Webhook_Type.TYPE_WECOM:
      return 6;
    case Webhook_Type.TYPE_CUSTOM:
      return 7;
    case Webhook_Type.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface DeploymentConfig {
  /**
   * The name of the resource.
   * Format: projects/{project}/deploymentConfigs/default.
   */
  name: string;
  /** The title of the deployment config. */
  title: string;
  schedule: Schedule | undefined;
}

export interface Schedule {
  deployments: ScheduleDeployment[];
}

export interface ScheduleDeployment {
  /** The title of the deployment (stage) in a schedule. */
  title: string;
  spec: DeploymentSpec | undefined;
}

export interface DeploymentSpec {
  labelSelector: LabelSelector | undefined;
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
  TYPE_UNSPECIFIED = "TYPE_UNSPECIFIED",
  /**
   * TYPE_NOTIFY_ISSUE_APPROVED - Notifications via webhooks.
   *
   * TYPE_NOTIFY_ISSUE_APPROVED represents the issue approved notification.
   */
  TYPE_NOTIFY_ISSUE_APPROVED = "TYPE_NOTIFY_ISSUE_APPROVED",
  /** TYPE_NOTIFY_PIPELINE_ROLLOUT - TYPE_NOTIFY_PIPELINE_ROLLOUT represents the pipeline rollout notification. */
  TYPE_NOTIFY_PIPELINE_ROLLOUT = "TYPE_NOTIFY_PIPELINE_ROLLOUT",
  /**
   * TYPE_ISSUE_CREATE - Issue related activity types.
   *
   * TYPE_ISSUE_CREATE represents creating an issue.
   */
  TYPE_ISSUE_CREATE = "TYPE_ISSUE_CREATE",
  /** TYPE_ISSUE_COMMENT_CREATE - TYPE_ISSUE_COMMENT_CREATE represents commenting on an issue. */
  TYPE_ISSUE_COMMENT_CREATE = "TYPE_ISSUE_COMMENT_CREATE",
  /** TYPE_ISSUE_FIELD_UPDATE - TYPE_ISSUE_FIELD_UPDATE represents updating the issue field, likes title, description, assignee, etc. */
  TYPE_ISSUE_FIELD_UPDATE = "TYPE_ISSUE_FIELD_UPDATE",
  /** TYPE_ISSUE_STATUS_UPDATE - TYPE_ISSUE_STATUS_UPDATE represents the issue status change, including OPEN, CLOSE, CANCEL fow now. */
  TYPE_ISSUE_STATUS_UPDATE = "TYPE_ISSUE_STATUS_UPDATE",
  /** TYPE_ISSUE_APPROVAL_NOTIFY - TYPE_ISSUE_APPROVAL_NOTIFY is the type for notifying issue approval. */
  TYPE_ISSUE_APPROVAL_NOTIFY = "TYPE_ISSUE_APPROVAL_NOTIFY",
  /** TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE - TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE represents the pipeline stage status change, including BEGIN, END for now. */
  TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE = "TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE",
  /** TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE - TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE represents the pipeline task status change, including PENDING, PENDING_APPROVAL, RUNNING, SUCCESS, FAILURE, CANCELED for now. */
  TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE = "TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE",
  /** TYPE_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE - TYPE_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE represents the pipeline task run status change, including PENDING, RUNNING, DONE, FAILED, CANCELED. */
  TYPE_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE = "TYPE_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE",
  /** TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE - TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE represents the manual update of the task statement. */
  TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE = "TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE",
  /** TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE - TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE represents the manual update of the task earliest allowed time. */
  TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE = "TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE",
  /**
   * TYPE_MEMBER_CREATE - Member related activity types.
   *
   * TYPE_MEMBER_CREATE represents creating a members.
   */
  TYPE_MEMBER_CREATE = "TYPE_MEMBER_CREATE",
  /** TYPE_MEMBER_ROLE_UPDATE - TYPE_MEMBER_ROLE_UPDATE represents updating the member role, for example, from ADMIN to MEMBER. */
  TYPE_MEMBER_ROLE_UPDATE = "TYPE_MEMBER_ROLE_UPDATE",
  /** TYPE_MEMBER_ACTIVATE - TYPE_MEMBER_ACTIVATE represents activating a deactivated member. */
  TYPE_MEMBER_ACTIVATE = "TYPE_MEMBER_ACTIVATE",
  /** TYPE_MEMBER_DEACTIVATE - TYPE_MEMBER_DEACTIVATE represents deactivating an active member. */
  TYPE_MEMBER_DEACTIVATE = "TYPE_MEMBER_DEACTIVATE",
  /**
   * TYPE_PROJECT_REPOSITORY_PUSH - Project related activity types.
   *
   * TYPE_PROJECT_REPOSITORY_PUSH represents Bytebase receiving a push event from the project repository.
   */
  TYPE_PROJECT_REPOSITORY_PUSH = "TYPE_PROJECT_REPOSITORY_PUSH",
  /** TYPE_PROJECT_DATABASE_TRANSFER - TYPE_PROJECT_DATABASE_TRANFER represents transfering the database from one project to another. */
  TYPE_PROJECT_DATABASE_TRANSFER = "TYPE_PROJECT_DATABASE_TRANSFER",
  /** TYPE_PROJECT_MEMBER_CREATE - TYPE_PROJECT_MEMBER_CREATE represents adding a member to the project. */
  TYPE_PROJECT_MEMBER_CREATE = "TYPE_PROJECT_MEMBER_CREATE",
  /** TYPE_PROJECT_MEMBER_DELETE - TYPE_PROJECT_MEMBER_DELETE represents removing a member from the project. */
  TYPE_PROJECT_MEMBER_DELETE = "TYPE_PROJECT_MEMBER_DELETE",
  /**
   * TYPE_SQL_EDITOR_QUERY - SQL Editor related activity types.
   * TYPE_SQL_EDITOR_QUERY represents executing query in SQL Editor.
   */
  TYPE_SQL_EDITOR_QUERY = "TYPE_SQL_EDITOR_QUERY",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function activity_TypeFromJSON(object: any): Activity_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return Activity_Type.TYPE_UNSPECIFIED;
    case 23:
    case "TYPE_NOTIFY_ISSUE_APPROVED":
      return Activity_Type.TYPE_NOTIFY_ISSUE_APPROVED;
    case 24:
    case "TYPE_NOTIFY_PIPELINE_ROLLOUT":
      return Activity_Type.TYPE_NOTIFY_PIPELINE_ROLLOUT;
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
    case 22:
    case "TYPE_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE":
      return Activity_Type.TYPE_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE;
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
    case 19:
    case "TYPE_SQL_EDITOR_QUERY":
      return Activity_Type.TYPE_SQL_EDITOR_QUERY;
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
    case Activity_Type.TYPE_NOTIFY_ISSUE_APPROVED:
      return "TYPE_NOTIFY_ISSUE_APPROVED";
    case Activity_Type.TYPE_NOTIFY_PIPELINE_ROLLOUT:
      return "TYPE_NOTIFY_PIPELINE_ROLLOUT";
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
    case Activity_Type.TYPE_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE:
      return "TYPE_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE";
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
    case Activity_Type.TYPE_SQL_EDITOR_QUERY:
      return "TYPE_SQL_EDITOR_QUERY";
    case Activity_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function activity_TypeToNumber(object: Activity_Type): number {
  switch (object) {
    case Activity_Type.TYPE_UNSPECIFIED:
      return 0;
    case Activity_Type.TYPE_NOTIFY_ISSUE_APPROVED:
      return 23;
    case Activity_Type.TYPE_NOTIFY_PIPELINE_ROLLOUT:
      return 24;
    case Activity_Type.TYPE_ISSUE_CREATE:
      return 1;
    case Activity_Type.TYPE_ISSUE_COMMENT_CREATE:
      return 2;
    case Activity_Type.TYPE_ISSUE_FIELD_UPDATE:
      return 3;
    case Activity_Type.TYPE_ISSUE_STATUS_UPDATE:
      return 4;
    case Activity_Type.TYPE_ISSUE_APPROVAL_NOTIFY:
      return 21;
    case Activity_Type.TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE:
      return 5;
    case Activity_Type.TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE:
      return 6;
    case Activity_Type.TYPE_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE:
      return 22;
    case Activity_Type.TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE:
      return 8;
    case Activity_Type.TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE:
      return 9;
    case Activity_Type.TYPE_MEMBER_CREATE:
      return 10;
    case Activity_Type.TYPE_MEMBER_ROLE_UPDATE:
      return 11;
    case Activity_Type.TYPE_MEMBER_ACTIVATE:
      return 12;
    case Activity_Type.TYPE_MEMBER_DEACTIVATE:
      return 13;
    case Activity_Type.TYPE_PROJECT_REPOSITORY_PUSH:
      return 14;
    case Activity_Type.TYPE_PROJECT_DATABASE_TRANSFER:
      return 15;
    case Activity_Type.TYPE_PROJECT_MEMBER_CREATE:
      return 16;
    case Activity_Type.TYPE_PROJECT_MEMBER_DELETE:
      return 17;
    case Activity_Type.TYPE_SQL_EDITOR_QUERY:
      return 19;
    case Activity_Type.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface ListDatabaseGroupsRequest {
  /**
   * The parent resource whose database groups are to be listed.
   * Format: projects/{project}
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
  databaseGroup:
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
  databaseGroup:
    | DatabaseGroup
    | undefined;
  /** The list of fields to update. */
  updateMask: string[] | undefined;
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
  databaseExpr:
    | Expr
    | undefined;
  /** The list of databases that match the database group condition. */
  matchedDatabases: DatabaseGroup_Database[];
  /** The list of databases that match the database group condition. */
  unmatchedDatabases: DatabaseGroup_Database[];
  multitenancy: boolean;
}

export interface DatabaseGroup_Database {
  /**
   * The resource name of the database.
   * Format: instances/{instance}/databases/{database}
   */
  name: string;
}

export interface GetProjectProtectionRulesRequest {
  /**
   * The name of the protection rules.
   * Format: projects/{project}/protectionRules
   */
  name: string;
}

export interface UpdateProjectProtectionRulesRequest {
  protectionRules: ProtectionRules | undefined;
}

export interface ProtectionRules {
  /**
   * The name of the protection rules.
   * Format: projects/{project}/protectionRules
   */
  name: string;
  rules: ProtectionRule[];
}

export interface ProtectionRule {
  /** A unique identifier for a node in UUID format. */
  id: string;
  target: ProtectionRule_Target;
  /** The name of the branch/changelist or wildcard. */
  nameFilter: string;
  /**
   * The roles allowed to create branches or changelists, rebase branches, delete branches.
   * Format: roles/projectOwner.
   */
  allowedRoles: string[];
  branchSource: ProtectionRule_BranchSource;
}

/** The type of target. */
export enum ProtectionRule_Target {
  PROTECTION_TARGET_UNSPECIFIED = "PROTECTION_TARGET_UNSPECIFIED",
  BRANCH = "BRANCH",
  CHANGELIST = "CHANGELIST",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function protectionRule_TargetFromJSON(object: any): ProtectionRule_Target {
  switch (object) {
    case 0:
    case "PROTECTION_TARGET_UNSPECIFIED":
      return ProtectionRule_Target.PROTECTION_TARGET_UNSPECIFIED;
    case 1:
    case "BRANCH":
      return ProtectionRule_Target.BRANCH;
    case 2:
    case "CHANGELIST":
      return ProtectionRule_Target.CHANGELIST;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ProtectionRule_Target.UNRECOGNIZED;
  }
}

export function protectionRule_TargetToJSON(object: ProtectionRule_Target): string {
  switch (object) {
    case ProtectionRule_Target.PROTECTION_TARGET_UNSPECIFIED:
      return "PROTECTION_TARGET_UNSPECIFIED";
    case ProtectionRule_Target.BRANCH:
      return "BRANCH";
    case ProtectionRule_Target.CHANGELIST:
      return "CHANGELIST";
    case ProtectionRule_Target.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function protectionRule_TargetToNumber(object: ProtectionRule_Target): number {
  switch (object) {
    case ProtectionRule_Target.PROTECTION_TARGET_UNSPECIFIED:
      return 0;
    case ProtectionRule_Target.BRANCH:
      return 1;
    case ProtectionRule_Target.CHANGELIST:
      return 2;
    case ProtectionRule_Target.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum ProtectionRule_BranchSource {
  BRANCH_SOURCE_UNSPECIFIED = "BRANCH_SOURCE_UNSPECIFIED",
  DATABASE = "DATABASE",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function protectionRule_BranchSourceFromJSON(object: any): ProtectionRule_BranchSource {
  switch (object) {
    case 0:
    case "BRANCH_SOURCE_UNSPECIFIED":
      return ProtectionRule_BranchSource.BRANCH_SOURCE_UNSPECIFIED;
    case 1:
    case "DATABASE":
      return ProtectionRule_BranchSource.DATABASE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ProtectionRule_BranchSource.UNRECOGNIZED;
  }
}

export function protectionRule_BranchSourceToJSON(object: ProtectionRule_BranchSource): string {
  switch (object) {
    case ProtectionRule_BranchSource.BRANCH_SOURCE_UNSPECIFIED:
      return "BRANCH_SOURCE_UNSPECIFIED";
    case ProtectionRule_BranchSource.DATABASE:
      return "DATABASE";
    case ProtectionRule_BranchSource.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function protectionRule_BranchSourceToNumber(object: ProtectionRule_BranchSource): number {
  switch (object) {
    case ProtectionRule_BranchSource.BRANCH_SOURCE_UNSPECIFIED:
      return 0;
    case ProtectionRule_BranchSource.DATABASE:
      return 1;
    case ProtectionRule_BranchSource.UNRECOGNIZED:
    default:
      return -1;
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
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: GetProjectRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
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
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
      showDeleted: isSet(object.showDeleted) ? globalThis.Boolean(object.showDeleted) : false,
    };
  },

  toJSON(message: ListProjectsRequest): unknown {
    const obj: any = {};
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    if (message.showDeleted === true) {
      obj.showDeleted = message.showDeleted;
    }
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
      projects: globalThis.Array.isArray(object?.projects) ? object.projects.map((e: any) => Project.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListProjectsResponse): unknown {
    const obj: any = {};
    if (message.projects?.length) {
      obj.projects = message.projects.map((e) => Project.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
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
  return { showDeleted: false };
}

export const SearchProjectsRequest = {
  encode(message: SearchProjectsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.showDeleted === true) {
      writer.uint32(8).bool(message.showDeleted);
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

  fromJSON(object: any): SearchProjectsRequest {
    return { showDeleted: isSet(object.showDeleted) ? globalThis.Boolean(object.showDeleted) : false };
  },

  toJSON(message: SearchProjectsRequest): unknown {
    const obj: any = {};
    if (message.showDeleted === true) {
      obj.showDeleted = message.showDeleted;
    }
    return obj;
  },

  create(base?: DeepPartial<SearchProjectsRequest>): SearchProjectsRequest {
    return SearchProjectsRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SearchProjectsRequest>): SearchProjectsRequest {
    const message = createBaseSearchProjectsRequest();
    message.showDeleted = object.showDeleted ?? false;
    return message;
  },
};

function createBaseSearchProjectsResponse(): SearchProjectsResponse {
  return { projects: [] };
}

export const SearchProjectsResponse = {
  encode(message: SearchProjectsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.projects) {
      Project.encode(v!, writer.uint32(10).fork()).ldelim();
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
      projects: globalThis.Array.isArray(object?.projects) ? object.projects.map((e: any) => Project.fromJSON(e)) : [],
    };
  },

  toJSON(message: SearchProjectsResponse): unknown {
    const obj: any = {};
    if (message.projects?.length) {
      obj.projects = message.projects.map((e) => Project.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<SearchProjectsResponse>): SearchProjectsResponse {
    return SearchProjectsResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SearchProjectsResponse>): SearchProjectsResponse {
    const message = createBaseSearchProjectsResponse();
    message.projects = object.projects?.map((e) => Project.fromPartial(e)) || [];
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
      projectId: isSet(object.projectId) ? globalThis.String(object.projectId) : "",
    };
  },

  toJSON(message: CreateProjectRequest): unknown {
    const obj: any = {};
    if (message.project !== undefined) {
      obj.project = Project.toJSON(message.project);
    }
    if (message.projectId !== "") {
      obj.projectId = message.projectId;
    }
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
    if (message.project !== undefined) {
      obj.project = Project.toJSON(message.project);
    }
    if (message.updateMask !== undefined) {
      obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask));
    }
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
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      force: isSet(object.force) ? globalThis.Boolean(object.force) : false,
    };
  },

  toJSON(message: DeleteProjectRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.force === true) {
      obj.force = message.force;
    }
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
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: UndeleteProjectRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
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
      scope: isSet(object.scope) ? globalThis.String(object.scope) : "",
      names: globalThis.Array.isArray(object?.names) ? object.names.map((e: any) => globalThis.String(e)) : [],
    };
  },

  toJSON(message: BatchGetIamPolicyRequest): unknown {
    const obj: any = {};
    if (message.scope !== "") {
      obj.scope = message.scope;
    }
    if (message.names?.length) {
      obj.names = message.names;
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
      policyResults: globalThis.Array.isArray(object?.policyResults)
        ? object.policyResults.map((e: any) => BatchGetIamPolicyResponse_PolicyResult.fromJSON(e))
        : [],
    };
  },

  toJSON(message: BatchGetIamPolicyResponse): unknown {
    const obj: any = {};
    if (message.policyResults?.length) {
      obj.policyResults = message.policyResults.map((e) => BatchGetIamPolicyResponse_PolicyResult.toJSON(e));
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
      project: isSet(object.project) ? globalThis.String(object.project) : "",
      policy: isSet(object.policy) ? IamPolicy.fromJSON(object.policy) : undefined,
    };
  },

  toJSON(message: BatchGetIamPolicyResponse_PolicyResult): unknown {
    const obj: any = {};
    if (message.project !== "") {
      obj.project = message.project;
    }
    if (message.policy !== undefined) {
      obj.policy = IamPolicy.toJSON(message.policy);
    }
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
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: GetDeploymentConfigRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
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
    if (message.config !== undefined) {
      obj.config = DeploymentConfig.toJSON(message.config);
    }
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

function createBaseLabel(): Label {
  return { value: "", color: "", group: "" };
}

export const Label = {
  encode(message: Label, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.value !== "") {
      writer.uint32(10).string(message.value);
    }
    if (message.color !== "") {
      writer.uint32(18).string(message.color);
    }
    if (message.group !== "") {
      writer.uint32(26).string(message.group);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Label {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseLabel();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.value = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.color = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.group = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Label {
    return {
      value: isSet(object.value) ? globalThis.String(object.value) : "",
      color: isSet(object.color) ? globalThis.String(object.color) : "",
      group: isSet(object.group) ? globalThis.String(object.group) : "",
    };
  },

  toJSON(message: Label): unknown {
    const obj: any = {};
    if (message.value !== "") {
      obj.value = message.value;
    }
    if (message.color !== "") {
      obj.color = message.color;
    }
    if (message.group !== "") {
      obj.group = message.group;
    }
    return obj;
  },

  create(base?: DeepPartial<Label>): Label {
    return Label.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Label>): Label {
    const message = createBaseLabel();
    message.value = object.value ?? "";
    message.color = object.color ?? "";
    message.group = object.group ?? "";
    return message;
  },
};

function createBaseProject(): Project {
  return {
    name: "",
    uid: "",
    state: State.STATE_UNSPECIFIED,
    title: "",
    key: "",
    workflow: Workflow.WORKFLOW_UNSPECIFIED,
    webhooks: [],
    dataClassificationConfigId: "",
    issueLabels: [],
    forceIssueLabels: false,
    allowModifyStatement: false,
    autoResolveIssue: false,
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
    if (message.state !== State.STATE_UNSPECIFIED) {
      writer.uint32(24).int32(stateToNumber(message.state));
    }
    if (message.title !== "") {
      writer.uint32(34).string(message.title);
    }
    if (message.key !== "") {
      writer.uint32(42).string(message.key);
    }
    if (message.workflow !== Workflow.WORKFLOW_UNSPECIFIED) {
      writer.uint32(48).int32(workflowToNumber(message.workflow));
    }
    for (const v of message.webhooks) {
      Webhook.encode(v!, writer.uint32(90).fork()).ldelim();
    }
    if (message.dataClassificationConfigId !== "") {
      writer.uint32(98).string(message.dataClassificationConfigId);
    }
    for (const v of message.issueLabels) {
      Label.encode(v!, writer.uint32(106).fork()).ldelim();
    }
    if (message.forceIssueLabels === true) {
      writer.uint32(112).bool(message.forceIssueLabels);
    }
    if (message.allowModifyStatement === true) {
      writer.uint32(120).bool(message.allowModifyStatement);
    }
    if (message.autoResolveIssue === true) {
      writer.uint32(128).bool(message.autoResolveIssue);
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

          message.state = stateFromJSON(reader.int32());
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

          message.workflow = workflowFromJSON(reader.int32());
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

          message.dataClassificationConfigId = reader.string();
          continue;
        case 13:
          if (tag !== 106) {
            break;
          }

          message.issueLabels.push(Label.decode(reader, reader.uint32()));
          continue;
        case 14:
          if (tag !== 112) {
            break;
          }

          message.forceIssueLabels = reader.bool();
          continue;
        case 15:
          if (tag !== 120) {
            break;
          }

          message.allowModifyStatement = reader.bool();
          continue;
        case 16:
          if (tag !== 128) {
            break;
          }

          message.autoResolveIssue = reader.bool();
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
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      uid: isSet(object.uid) ? globalThis.String(object.uid) : "",
      state: isSet(object.state) ? stateFromJSON(object.state) : State.STATE_UNSPECIFIED,
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      workflow: isSet(object.workflow) ? workflowFromJSON(object.workflow) : Workflow.WORKFLOW_UNSPECIFIED,
      webhooks: globalThis.Array.isArray(object?.webhooks) ? object.webhooks.map((e: any) => Webhook.fromJSON(e)) : [],
      dataClassificationConfigId: isSet(object.dataClassificationConfigId)
        ? globalThis.String(object.dataClassificationConfigId)
        : "",
      issueLabels: globalThis.Array.isArray(object?.issueLabels)
        ? object.issueLabels.map((e: any) => Label.fromJSON(e))
        : [],
      forceIssueLabels: isSet(object.forceIssueLabels) ? globalThis.Boolean(object.forceIssueLabels) : false,
      allowModifyStatement: isSet(object.allowModifyStatement)
        ? globalThis.Boolean(object.allowModifyStatement)
        : false,
      autoResolveIssue: isSet(object.autoResolveIssue) ? globalThis.Boolean(object.autoResolveIssue) : false,
    };
  },

  toJSON(message: Project): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.uid !== "") {
      obj.uid = message.uid;
    }
    if (message.state !== State.STATE_UNSPECIFIED) {
      obj.state = stateToJSON(message.state);
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.workflow !== Workflow.WORKFLOW_UNSPECIFIED) {
      obj.workflow = workflowToJSON(message.workflow);
    }
    if (message.webhooks?.length) {
      obj.webhooks = message.webhooks.map((e) => Webhook.toJSON(e));
    }
    if (message.dataClassificationConfigId !== "") {
      obj.dataClassificationConfigId = message.dataClassificationConfigId;
    }
    if (message.issueLabels?.length) {
      obj.issueLabels = message.issueLabels.map((e) => Label.toJSON(e));
    }
    if (message.forceIssueLabels === true) {
      obj.forceIssueLabels = message.forceIssueLabels;
    }
    if (message.allowModifyStatement === true) {
      obj.allowModifyStatement = message.allowModifyStatement;
    }
    if (message.autoResolveIssue === true) {
      obj.autoResolveIssue = message.autoResolveIssue;
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
    message.state = object.state ?? State.STATE_UNSPECIFIED;
    message.title = object.title ?? "";
    message.key = object.key ?? "";
    message.workflow = object.workflow ?? Workflow.WORKFLOW_UNSPECIFIED;
    message.webhooks = object.webhooks?.map((e) => Webhook.fromPartial(e)) || [];
    message.dataClassificationConfigId = object.dataClassificationConfigId ?? "";
    message.issueLabels = object.issueLabels?.map((e) => Label.fromPartial(e)) || [];
    message.forceIssueLabels = object.forceIssueLabels ?? false;
    message.allowModifyStatement = object.allowModifyStatement ?? false;
    message.autoResolveIssue = object.autoResolveIssue ?? false;
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
      project: isSet(object.project) ? globalThis.String(object.project) : "",
      webhook: isSet(object.webhook) ? Webhook.fromJSON(object.webhook) : undefined,
    };
  },

  toJSON(message: AddWebhookRequest): unknown {
    const obj: any = {};
    if (message.project !== "") {
      obj.project = message.project;
    }
    if (message.webhook !== undefined) {
      obj.webhook = Webhook.toJSON(message.webhook);
    }
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
    if (message.webhook !== undefined) {
      obj.webhook = Webhook.toJSON(message.webhook);
    }
    if (message.updateMask !== undefined) {
      obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask));
    }
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
    if (message.webhook !== undefined) {
      obj.webhook = Webhook.toJSON(message.webhook);
    }
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
      project: isSet(object.project) ? globalThis.String(object.project) : "",
      webhook: isSet(object.webhook) ? Webhook.fromJSON(object.webhook) : undefined,
    };
  },

  toJSON(message: TestWebhookRequest): unknown {
    const obj: any = {};
    if (message.project !== "") {
      obj.project = message.project;
    }
    if (message.webhook !== undefined) {
      obj.webhook = Webhook.toJSON(message.webhook);
    }
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
    return { error: isSet(object.error) ? globalThis.String(object.error) : "" };
  },

  toJSON(message: TestWebhookResponse): unknown {
    const obj: any = {};
    if (message.error !== "") {
      obj.error = message.error;
    }
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
  return {
    name: "",
    type: Webhook_Type.TYPE_UNSPECIFIED,
    title: "",
    url: "",
    directMessage: false,
    notificationTypes: [],
  };
}

export const Webhook = {
  encode(message: Webhook, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.type !== Webhook_Type.TYPE_UNSPECIFIED) {
      writer.uint32(16).int32(webhook_TypeToNumber(message.type));
    }
    if (message.title !== "") {
      writer.uint32(26).string(message.title);
    }
    if (message.url !== "") {
      writer.uint32(34).string(message.url);
    }
    if (message.directMessage === true) {
      writer.uint32(48).bool(message.directMessage);
    }
    writer.uint32(42).fork();
    for (const v of message.notificationTypes) {
      writer.int32(activity_TypeToNumber(v));
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

          message.type = webhook_TypeFromJSON(reader.int32());
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
        case 6:
          if (tag !== 48) {
            break;
          }

          message.directMessage = reader.bool();
          continue;
        case 5:
          if (tag === 40) {
            message.notificationTypes.push(activity_TypeFromJSON(reader.int32()));

            continue;
          }

          if (tag === 42) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.notificationTypes.push(activity_TypeFromJSON(reader.int32()));
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
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      type: isSet(object.type) ? webhook_TypeFromJSON(object.type) : Webhook_Type.TYPE_UNSPECIFIED,
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      url: isSet(object.url) ? globalThis.String(object.url) : "",
      directMessage: isSet(object.directMessage) ? globalThis.Boolean(object.directMessage) : false,
      notificationTypes: globalThis.Array.isArray(object?.notificationTypes)
        ? object.notificationTypes.map((e: any) => activity_TypeFromJSON(e))
        : [],
    };
  },

  toJSON(message: Webhook): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.type !== Webhook_Type.TYPE_UNSPECIFIED) {
      obj.type = webhook_TypeToJSON(message.type);
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.url !== "") {
      obj.url = message.url;
    }
    if (message.directMessage === true) {
      obj.directMessage = message.directMessage;
    }
    if (message.notificationTypes?.length) {
      obj.notificationTypes = message.notificationTypes.map((e) => activity_TypeToJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<Webhook>): Webhook {
    return Webhook.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Webhook>): Webhook {
    const message = createBaseWebhook();
    message.name = object.name ?? "";
    message.type = object.type ?? Webhook_Type.TYPE_UNSPECIFIED;
    message.title = object.title ?? "";
    message.url = object.url ?? "";
    message.directMessage = object.directMessage ?? false;
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
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      schedule: isSet(object.schedule) ? Schedule.fromJSON(object.schedule) : undefined,
    };
  },

  toJSON(message: DeploymentConfig): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.schedule !== undefined) {
      obj.schedule = Schedule.toJSON(message.schedule);
    }
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
      deployments: globalThis.Array.isArray(object?.deployments)
        ? object.deployments.map((e: any) => ScheduleDeployment.fromJSON(e))
        : [],
    };
  },

  toJSON(message: Schedule): unknown {
    const obj: any = {};
    if (message.deployments?.length) {
      obj.deployments = message.deployments.map((e) => ScheduleDeployment.toJSON(e));
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
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      spec: isSet(object.spec) ? DeploymentSpec.fromJSON(object.spec) : undefined,
    };
  },

  toJSON(message: ScheduleDeployment): unknown {
    const obj: any = {};
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.spec !== undefined) {
      obj.spec = DeploymentSpec.toJSON(message.spec);
    }
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
    if (message.labelSelector !== undefined) {
      obj.labelSelector = LabelSelector.toJSON(message.labelSelector);
    }
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
      matchExpressions: globalThis.Array.isArray(object?.matchExpressions)
        ? object.matchExpressions.map((e: any) => LabelSelectorRequirement.fromJSON(e))
        : [],
    };
  },

  toJSON(message: LabelSelector): unknown {
    const obj: any = {};
    if (message.matchExpressions?.length) {
      obj.matchExpressions = message.matchExpressions.map((e) => LabelSelectorRequirement.toJSON(e));
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
  return { key: "", operator: OperatorType.OPERATOR_TYPE_UNSPECIFIED, values: [] };
}

export const LabelSelectorRequirement = {
  encode(message: LabelSelectorRequirement, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.operator !== OperatorType.OPERATOR_TYPE_UNSPECIFIED) {
      writer.uint32(16).int32(operatorTypeToNumber(message.operator));
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

          message.operator = operatorTypeFromJSON(reader.int32());
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
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      operator: isSet(object.operator) ? operatorTypeFromJSON(object.operator) : OperatorType.OPERATOR_TYPE_UNSPECIFIED,
      values: globalThis.Array.isArray(object?.values) ? object.values.map((e: any) => globalThis.String(e)) : [],
    };
  },

  toJSON(message: LabelSelectorRequirement): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.operator !== OperatorType.OPERATOR_TYPE_UNSPECIFIED) {
      obj.operator = operatorTypeToJSON(message.operator);
    }
    if (message.values?.length) {
      obj.values = message.values;
    }
    return obj;
  },

  create(base?: DeepPartial<LabelSelectorRequirement>): LabelSelectorRequirement {
    return LabelSelectorRequirement.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<LabelSelectorRequirement>): LabelSelectorRequirement {
    const message = createBaseLabelSelectorRequirement();
    message.key = object.key ?? "";
    message.operator = object.operator ?? OperatorType.OPERATOR_TYPE_UNSPECIFIED;
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
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
    };
  },

  toJSON(message: ListDatabaseGroupsRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
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
      databaseGroups: globalThis.Array.isArray(object?.databaseGroups)
        ? object.databaseGroups.map((e: any) => DatabaseGroup.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListDatabaseGroupsResponse): unknown {
    const obj: any = {};
    if (message.databaseGroups?.length) {
      obj.databaseGroups = message.databaseGroups.map((e) => DatabaseGroup.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
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
  return { name: "", view: DatabaseGroupView.DATABASE_GROUP_VIEW_UNSPECIFIED };
}

export const GetDatabaseGroupRequest = {
  encode(message: GetDatabaseGroupRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.view !== DatabaseGroupView.DATABASE_GROUP_VIEW_UNSPECIFIED) {
      writer.uint32(16).int32(databaseGroupViewToNumber(message.view));
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

          message.view = databaseGroupViewFromJSON(reader.int32());
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
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      view: isSet(object.view)
        ? databaseGroupViewFromJSON(object.view)
        : DatabaseGroupView.DATABASE_GROUP_VIEW_UNSPECIFIED,
    };
  },

  toJSON(message: GetDatabaseGroupRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.view !== DatabaseGroupView.DATABASE_GROUP_VIEW_UNSPECIFIED) {
      obj.view = databaseGroupViewToJSON(message.view);
    }
    return obj;
  },

  create(base?: DeepPartial<GetDatabaseGroupRequest>): GetDatabaseGroupRequest {
    return GetDatabaseGroupRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<GetDatabaseGroupRequest>): GetDatabaseGroupRequest {
    const message = createBaseGetDatabaseGroupRequest();
    message.name = object.name ?? "";
    message.view = object.view ?? DatabaseGroupView.DATABASE_GROUP_VIEW_UNSPECIFIED;
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
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      databaseGroup: isSet(object.databaseGroup) ? DatabaseGroup.fromJSON(object.databaseGroup) : undefined,
      databaseGroupId: isSet(object.databaseGroupId) ? globalThis.String(object.databaseGroupId) : "",
      validateOnly: isSet(object.validateOnly) ? globalThis.Boolean(object.validateOnly) : false,
    };
  },

  toJSON(message: CreateDatabaseGroupRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.databaseGroup !== undefined) {
      obj.databaseGroup = DatabaseGroup.toJSON(message.databaseGroup);
    }
    if (message.databaseGroupId !== "") {
      obj.databaseGroupId = message.databaseGroupId;
    }
    if (message.validateOnly === true) {
      obj.validateOnly = message.validateOnly;
    }
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
    if (message.databaseGroup !== undefined) {
      obj.databaseGroup = DatabaseGroup.toJSON(message.databaseGroup);
    }
    if (message.updateMask !== undefined) {
      obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask));
    }
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
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: DeleteDatabaseGroupRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
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
  return {
    name: "",
    databasePlaceholder: "",
    databaseExpr: undefined,
    matchedDatabases: [],
    unmatchedDatabases: [],
    multitenancy: false,
  };
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
    if (message.multitenancy === true) {
      writer.uint32(48).bool(message.multitenancy);
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
        case 6:
          if (tag !== 48) {
            break;
          }

          message.multitenancy = reader.bool();
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
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      databasePlaceholder: isSet(object.databasePlaceholder) ? globalThis.String(object.databasePlaceholder) : "",
      databaseExpr: isSet(object.databaseExpr) ? Expr.fromJSON(object.databaseExpr) : undefined,
      matchedDatabases: globalThis.Array.isArray(object?.matchedDatabases)
        ? object.matchedDatabases.map((e: any) => DatabaseGroup_Database.fromJSON(e))
        : [],
      unmatchedDatabases: globalThis.Array.isArray(object?.unmatchedDatabases)
        ? object.unmatchedDatabases.map((e: any) => DatabaseGroup_Database.fromJSON(e))
        : [],
      multitenancy: isSet(object.multitenancy) ? globalThis.Boolean(object.multitenancy) : false,
    };
  },

  toJSON(message: DatabaseGroup): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.databasePlaceholder !== "") {
      obj.databasePlaceholder = message.databasePlaceholder;
    }
    if (message.databaseExpr !== undefined) {
      obj.databaseExpr = Expr.toJSON(message.databaseExpr);
    }
    if (message.matchedDatabases?.length) {
      obj.matchedDatabases = message.matchedDatabases.map((e) => DatabaseGroup_Database.toJSON(e));
    }
    if (message.unmatchedDatabases?.length) {
      obj.unmatchedDatabases = message.unmatchedDatabases.map((e) => DatabaseGroup_Database.toJSON(e));
    }
    if (message.multitenancy === true) {
      obj.multitenancy = message.multitenancy;
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
    message.multitenancy = object.multitenancy ?? false;
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
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: DatabaseGroup_Database): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
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

function createBaseGetProjectProtectionRulesRequest(): GetProjectProtectionRulesRequest {
  return { name: "" };
}

export const GetProjectProtectionRulesRequest = {
  encode(message: GetProjectProtectionRulesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetProjectProtectionRulesRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetProjectProtectionRulesRequest();
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

  fromJSON(object: any): GetProjectProtectionRulesRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: GetProjectProtectionRulesRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<GetProjectProtectionRulesRequest>): GetProjectProtectionRulesRequest {
    return GetProjectProtectionRulesRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<GetProjectProtectionRulesRequest>): GetProjectProtectionRulesRequest {
    const message = createBaseGetProjectProtectionRulesRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseUpdateProjectProtectionRulesRequest(): UpdateProjectProtectionRulesRequest {
  return { protectionRules: undefined };
}

export const UpdateProjectProtectionRulesRequest = {
  encode(message: UpdateProjectProtectionRulesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.protectionRules !== undefined) {
      ProtectionRules.encode(message.protectionRules, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateProjectProtectionRulesRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateProjectProtectionRulesRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 2:
          if (tag !== 18) {
            break;
          }

          message.protectionRules = ProtectionRules.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UpdateProjectProtectionRulesRequest {
    return {
      protectionRules: isSet(object.protectionRules) ? ProtectionRules.fromJSON(object.protectionRules) : undefined,
    };
  },

  toJSON(message: UpdateProjectProtectionRulesRequest): unknown {
    const obj: any = {};
    if (message.protectionRules !== undefined) {
      obj.protectionRules = ProtectionRules.toJSON(message.protectionRules);
    }
    return obj;
  },

  create(base?: DeepPartial<UpdateProjectProtectionRulesRequest>): UpdateProjectProtectionRulesRequest {
    return UpdateProjectProtectionRulesRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UpdateProjectProtectionRulesRequest>): UpdateProjectProtectionRulesRequest {
    const message = createBaseUpdateProjectProtectionRulesRequest();
    message.protectionRules = (object.protectionRules !== undefined && object.protectionRules !== null)
      ? ProtectionRules.fromPartial(object.protectionRules)
      : undefined;
    return message;
  },
};

function createBaseProtectionRules(): ProtectionRules {
  return { name: "", rules: [] };
}

export const ProtectionRules = {
  encode(message: ProtectionRules, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.rules) {
      ProtectionRule.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ProtectionRules {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseProtectionRules();
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

          message.rules.push(ProtectionRule.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ProtectionRules {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      rules: globalThis.Array.isArray(object?.rules) ? object.rules.map((e: any) => ProtectionRule.fromJSON(e)) : [],
    };
  },

  toJSON(message: ProtectionRules): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.rules?.length) {
      obj.rules = message.rules.map((e) => ProtectionRule.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<ProtectionRules>): ProtectionRules {
    return ProtectionRules.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ProtectionRules>): ProtectionRules {
    const message = createBaseProtectionRules();
    message.name = object.name ?? "";
    message.rules = object.rules?.map((e) => ProtectionRule.fromPartial(e)) || [];
    return message;
  },
};

function createBaseProtectionRule(): ProtectionRule {
  return {
    id: "",
    target: ProtectionRule_Target.PROTECTION_TARGET_UNSPECIFIED,
    nameFilter: "",
    allowedRoles: [],
    branchSource: ProtectionRule_BranchSource.BRANCH_SOURCE_UNSPECIFIED,
  };
}

export const ProtectionRule = {
  encode(message: ProtectionRule, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.target !== ProtectionRule_Target.PROTECTION_TARGET_UNSPECIFIED) {
      writer.uint32(16).int32(protectionRule_TargetToNumber(message.target));
    }
    if (message.nameFilter !== "") {
      writer.uint32(26).string(message.nameFilter);
    }
    for (const v of message.allowedRoles) {
      writer.uint32(34).string(v!);
    }
    if (message.branchSource !== ProtectionRule_BranchSource.BRANCH_SOURCE_UNSPECIFIED) {
      writer.uint32(40).int32(protectionRule_BranchSourceToNumber(message.branchSource));
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ProtectionRule {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseProtectionRule();
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
          if (tag !== 16) {
            break;
          }

          message.target = protectionRule_TargetFromJSON(reader.int32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.nameFilter = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.allowedRoles.push(reader.string());
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.branchSource = protectionRule_BranchSourceFromJSON(reader.int32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ProtectionRule {
    return {
      id: isSet(object.id) ? globalThis.String(object.id) : "",
      target: isSet(object.target)
        ? protectionRule_TargetFromJSON(object.target)
        : ProtectionRule_Target.PROTECTION_TARGET_UNSPECIFIED,
      nameFilter: isSet(object.nameFilter) ? globalThis.String(object.nameFilter) : "",
      allowedRoles: globalThis.Array.isArray(object?.allowedRoles)
        ? object.allowedRoles.map((e: any) => globalThis.String(e))
        : [],
      branchSource: isSet(object.branchSource)
        ? protectionRule_BranchSourceFromJSON(object.branchSource)
        : ProtectionRule_BranchSource.BRANCH_SOURCE_UNSPECIFIED,
    };
  },

  toJSON(message: ProtectionRule): unknown {
    const obj: any = {};
    if (message.id !== "") {
      obj.id = message.id;
    }
    if (message.target !== ProtectionRule_Target.PROTECTION_TARGET_UNSPECIFIED) {
      obj.target = protectionRule_TargetToJSON(message.target);
    }
    if (message.nameFilter !== "") {
      obj.nameFilter = message.nameFilter;
    }
    if (message.allowedRoles?.length) {
      obj.allowedRoles = message.allowedRoles;
    }
    if (message.branchSource !== ProtectionRule_BranchSource.BRANCH_SOURCE_UNSPECIFIED) {
      obj.branchSource = protectionRule_BranchSourceToJSON(message.branchSource);
    }
    return obj;
  },

  create(base?: DeepPartial<ProtectionRule>): ProtectionRule {
    return ProtectionRule.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ProtectionRule>): ProtectionRule {
    const message = createBaseProtectionRule();
    message.id = object.id ?? "";
    message.target = object.target ?? ProtectionRule_Target.PROTECTION_TARGET_UNSPECIFIED;
    message.nameFilter = object.nameFilter ?? "";
    message.allowedRoles = object.allowedRoles?.map((e) => e) || [];
    message.branchSource = object.branchSource ?? ProtectionRule_BranchSource.BRANCH_SOURCE_UNSPECIFIED;
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
          800010: [new Uint8Array([15, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 103, 101, 116])],
          800016: [new Uint8Array([2])],
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
          800010: [new Uint8Array([16, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 108, 105, 115, 116])],
          578365826: [new Uint8Array([14, 18, 12, 47, 118, 49, 47, 112, 114, 111, 106, 101, 99, 116, 115])],
        },
      },
    },
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
          800010: [
            new Uint8Array([18, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 99, 114, 101, 97, 116, 101]),
          ],
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
          800010: [
            new Uint8Array([18, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 117, 112, 100, 97, 116, 101]),
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
          800010: [
            new Uint8Array([18, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 100, 101, 108, 101, 116, 101]),
          ],
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
          800010: [
            new Uint8Array([
              20,
              98,
              98,
              46,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              46,
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
          800010: [
            new Uint8Array([
              24,
              98,
              98,
              46,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              46,
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
              114,
              101,
              115,
              111,
              117,
              114,
              99,
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
          800010: [
            new Uint8Array([
              24,
              98,
              98,
              46,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              46,
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
          800010: [
            new Uint8Array([
              24,
              98,
              98,
              46,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              46,
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
              114,
              101,
              115,
              111,
              117,
              114,
              99,
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
          800010: [new Uint8Array([15, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 103, 101, 116])],
          578365826: [
            new Uint8Array([
              43,
              18,
              41,
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
              115,
              47,
              42,
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
          800010: [
            new Uint8Array([18, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 117, 112, 100, 97, 116, 101]),
          ],
          578365826: [
            new Uint8Array([
              58,
              58,
              6,
              99,
              111,
              110,
              102,
              105,
              103,
              50,
              48,
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
              115,
              47,
              42,
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
          800010: [
            new Uint8Array([18, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 117, 112, 100, 97, 116, 101]),
          ],
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
          800010: [
            new Uint8Array([18, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 117, 112, 100, 97, 116, 101]),
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
          800010: [
            new Uint8Array([18, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 117, 112, 100, 97, 116, 101]),
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
          800010: [
            new Uint8Array([18, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 117, 112, 100, 97, 116, 101]),
          ],
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
    listDatabaseGroups: {
      name: "ListDatabaseGroups",
      requestType: ListDatabaseGroupsRequest,
      requestStream: false,
      responseType: ListDatabaseGroupsResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([6, 112, 97, 114, 101, 110, 116])],
          800010: [new Uint8Array([15, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 103, 101, 116])],
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
          800010: [new Uint8Array([15, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 103, 101, 116])],
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
          800010: [
            new Uint8Array([18, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 117, 112, 100, 97, 116, 101]),
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
          800010: [
            new Uint8Array([18, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 117, 112, 100, 97, 116, 101]),
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
          800010: [
            new Uint8Array([18, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 117, 112, 100, 97, 116, 101]),
          ],
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
    getProjectProtectionRules: {
      name: "GetProjectProtectionRules",
      requestType: GetProjectProtectionRulesRequest,
      requestStream: false,
      responseType: ProtectionRules,
      responseStream: false,
      options: {
        _unknownFields: {
          800010: [new Uint8Array([15, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 103, 101, 116])],
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
              114,
              111,
              116,
              101,
              99,
              116,
              105,
              111,
              110,
              82,
              117,
              108,
              101,
              115,
              125,
            ]),
          ],
        },
      },
    },
    updateProjectProtectionRules: {
      name: "UpdateProjectProtectionRules",
      requestType: UpdateProjectProtectionRulesRequest,
      requestStream: false,
      responseType: ProtectionRules,
      responseStream: false,
      options: {
        _unknownFields: {
          800010: [
            new Uint8Array([18, 98, 98, 46, 112, 114, 111, 106, 101, 99, 116, 115, 46, 117, 112, 100, 97, 116, 101]),
          ],
          578365826: [
            new Uint8Array([
              59,
              58,
              1,
              42,
              50,
              54,
              47,
              118,
              49,
              47,
              123,
              112,
              114,
              111,
              116,
              101,
              99,
              116,
              105,
              111,
              110,
              95,
              114,
              117,
              108,
              101,
              115,
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
              114,
              111,
              116,
              101,
              99,
              116,
              105,
              111,
              110,
              82,
              117,
              108,
              101,
              115,
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
  : T extends Long ? string | number | Long : T extends globalThis.Array<infer U> ? globalThis.Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
