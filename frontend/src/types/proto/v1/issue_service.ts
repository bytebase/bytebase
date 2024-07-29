/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Duration } from "../google/protobuf/duration";
import { FieldMask } from "../google/protobuf/field_mask";
import { Timestamp } from "../google/protobuf/timestamp";
import { Expr } from "../google/type/expr";

export const protobufPackage = "bytebase.v1";

export enum IssueStatus {
  ISSUE_STATUS_UNSPECIFIED = "ISSUE_STATUS_UNSPECIFIED",
  OPEN = "OPEN",
  DONE = "DONE",
  CANCELED = "CANCELED",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function issueStatusFromJSON(object: any): IssueStatus {
  switch (object) {
    case 0:
    case "ISSUE_STATUS_UNSPECIFIED":
      return IssueStatus.ISSUE_STATUS_UNSPECIFIED;
    case 1:
    case "OPEN":
      return IssueStatus.OPEN;
    case 2:
    case "DONE":
      return IssueStatus.DONE;
    case 3:
    case "CANCELED":
      return IssueStatus.CANCELED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return IssueStatus.UNRECOGNIZED;
  }
}

export function issueStatusToJSON(object: IssueStatus): string {
  switch (object) {
    case IssueStatus.ISSUE_STATUS_UNSPECIFIED:
      return "ISSUE_STATUS_UNSPECIFIED";
    case IssueStatus.OPEN:
      return "OPEN";
    case IssueStatus.DONE:
      return "DONE";
    case IssueStatus.CANCELED:
      return "CANCELED";
    case IssueStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function issueStatusToNumber(object: IssueStatus): number {
  switch (object) {
    case IssueStatus.ISSUE_STATUS_UNSPECIFIED:
      return 0;
    case IssueStatus.OPEN:
      return 1;
    case IssueStatus.DONE:
      return 2;
    case IssueStatus.CANCELED:
      return 3;
    case IssueStatus.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface GetIssueRequest {
  /**
   * The name of the issue to retrieve.
   * Format: projects/{project}/issues/{issue}
   */
  name: string;
  force: boolean;
}

export interface CreateIssueRequest {
  /**
   * The parent, which owns this collection of issues.
   * Format: projects/{project}
   */
  parent: string;
  /** The issue to create. */
  issue: Issue | undefined;
}

export interface ListIssuesRequest {
  /**
   * The parent, which owns this collection of issues.
   * Format: projects/{project}
   * Use "projects/-" to list all issues from all projects.
   */
  parent: string;
  /**
   * The maximum number of issues to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 issues will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListIssues` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListIssues` must match
   * the call that provided the page token.
   */
  pageToken: string;
  /** Filter is used to filter issues returned in the list. */
  filter: string;
  /** Query is the query statement. */
  query: string;
}

export interface ListIssuesResponse {
  /** The issues from the specified request. */
  issues: Issue[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface SearchIssuesRequest {
  /**
   * The parent, which owns this collection of issues.
   * Format: projects/{project}
   * Use "projects/-" to list all issues from all projects.
   */
  parent: string;
  /**
   * The maximum number of issues to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 issues will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListIssues` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListIssues` must match
   * the call that provided the page token.
   */
  pageToken: string;
  /** Filter is used to filter issues returned in the list. */
  filter: string;
  /** Query is the query statement. */
  query: string;
}

export interface SearchIssuesResponse {
  /** The issues from the specified request. */
  issues: Issue[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface UpdateIssueRequest {
  /**
   * The issue to update.
   *
   * The issue's `name` field is used to identify the issue to update.
   * Format: projects/{project}/issues/{issue}
   */
  issue:
    | Issue
    | undefined;
  /** The list of fields to update. */
  updateMask: string[] | undefined;
}

export interface BatchUpdateIssuesStatusRequest {
  /**
   * The parent resource shared by all issues being updated.
   * Format: projects/{project}
   * If the operation spans parents, a dash (-) may be accepted as a wildcard.
   * We only support updating the status of databases for now.
   */
  parent: string;
  /**
   * The list of issues to update.
   * Format: projects/{project}/issues/{issue}
   */
  issues: string[];
  /** The new status. */
  status: IssueStatus;
  reason: string;
}

export interface BatchUpdateIssuesStatusResponse {
}

export interface ApproveIssueRequest {
  /**
   * The name of the issue to add an approver.
   * Format: projects/{project}/issues/{issue}
   */
  name: string;
  comment: string;
}

export interface RejectIssueRequest {
  /**
   * The name of the issue to add an rejection.
   * Format: projects/{project}/issues/{issue}
   */
  name: string;
  comment: string;
}

export interface RequestIssueRequest {
  /**
   * The name of the issue to request a issue.
   * Format: projects/{project}/issues/{issue}
   */
  name: string;
  comment: string;
}

export interface Issue {
  /**
   * The name of the issue.
   * Format: projects/{project}/issues/{issue}
   */
  name: string;
  /** The system-assigned, unique identifier for a resource. */
  uid: string;
  title: string;
  description: string;
  type: Issue_Type;
  status: IssueStatus;
  /** Format: users/hello@world.com */
  assignee: string;
  assigneeAttention: boolean;
  approvers: Issue_Approver[];
  approvalTemplates: ApprovalTemplate[];
  /**
   * If the value is `false`, it means that the backend is still finding matching approval templates.
   * If `true`, approval_templates & approvers & approval_finding_error are available.
   */
  approvalFindingDone: boolean;
  approvalFindingError: string;
  /**
   * The subscribers.
   * Format: users/hello@world.com
   */
  subscribers: string[];
  /** Format: users/hello@world.com */
  creator: string;
  createTime: Date | undefined;
  updateTime:
    | Date
    | undefined;
  /**
   * The plan associated with the issue.
   * Can be empty.
   * Format: projects/{project}/plans/{plan}
   */
  plan: string;
  /**
   * The rollout associated with the issue.
   * Can be empty.
   * Format: projects/{project}/rollouts/{rollout}
   */
  rollout: string;
  /** Used if the issue type is GRANT_REQUEST. */
  grantRequest:
    | GrantRequest
    | undefined;
  /**
   * The releasers of the pending stage of the issue rollout, judging
   * from the rollout policy.
   * If the policy is auto rollout, the releasers are the project owners and the issue creator.
   * Format:
   * - roles/workspaceOwner
   * - roles/workspaceDBA
   * - roles/projectOwner
   * - roles/projectReleaser
   * - users/{email}
   */
  releasers: string[];
  riskLevel: Issue_RiskLevel;
  /**
   * The status count of the issue.
   * Keys are the following:
   * - NOT_STARTED
   * - SKIPPED
   * - PENDING
   * - RUNNING
   * - DONE
   * - FAILED
   * - CANCELED
   */
  taskStatusCount: { [key: string]: number };
  labels: string[];
}

export enum Issue_Type {
  TYPE_UNSPECIFIED = "TYPE_UNSPECIFIED",
  DATABASE_CHANGE = "DATABASE_CHANGE",
  GRANT_REQUEST = "GRANT_REQUEST",
  DATABASE_DATA_EXPORT = "DATABASE_DATA_EXPORT",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function issue_TypeFromJSON(object: any): Issue_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return Issue_Type.TYPE_UNSPECIFIED;
    case 1:
    case "DATABASE_CHANGE":
      return Issue_Type.DATABASE_CHANGE;
    case 2:
    case "GRANT_REQUEST":
      return Issue_Type.GRANT_REQUEST;
    case 3:
    case "DATABASE_DATA_EXPORT":
      return Issue_Type.DATABASE_DATA_EXPORT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Issue_Type.UNRECOGNIZED;
  }
}

export function issue_TypeToJSON(object: Issue_Type): string {
  switch (object) {
    case Issue_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case Issue_Type.DATABASE_CHANGE:
      return "DATABASE_CHANGE";
    case Issue_Type.GRANT_REQUEST:
      return "GRANT_REQUEST";
    case Issue_Type.DATABASE_DATA_EXPORT:
      return "DATABASE_DATA_EXPORT";
    case Issue_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function issue_TypeToNumber(object: Issue_Type): number {
  switch (object) {
    case Issue_Type.TYPE_UNSPECIFIED:
      return 0;
    case Issue_Type.DATABASE_CHANGE:
      return 1;
    case Issue_Type.GRANT_REQUEST:
      return 2;
    case Issue_Type.DATABASE_DATA_EXPORT:
      return 3;
    case Issue_Type.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum Issue_RiskLevel {
  RISK_LEVEL_UNSPECIFIED = "RISK_LEVEL_UNSPECIFIED",
  LOW = "LOW",
  MODERATE = "MODERATE",
  HIGH = "HIGH",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function issue_RiskLevelFromJSON(object: any): Issue_RiskLevel {
  switch (object) {
    case 0:
    case "RISK_LEVEL_UNSPECIFIED":
      return Issue_RiskLevel.RISK_LEVEL_UNSPECIFIED;
    case 1:
    case "LOW":
      return Issue_RiskLevel.LOW;
    case 2:
    case "MODERATE":
      return Issue_RiskLevel.MODERATE;
    case 3:
    case "HIGH":
      return Issue_RiskLevel.HIGH;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Issue_RiskLevel.UNRECOGNIZED;
  }
}

export function issue_RiskLevelToJSON(object: Issue_RiskLevel): string {
  switch (object) {
    case Issue_RiskLevel.RISK_LEVEL_UNSPECIFIED:
      return "RISK_LEVEL_UNSPECIFIED";
    case Issue_RiskLevel.LOW:
      return "LOW";
    case Issue_RiskLevel.MODERATE:
      return "MODERATE";
    case Issue_RiskLevel.HIGH:
      return "HIGH";
    case Issue_RiskLevel.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function issue_RiskLevelToNumber(object: Issue_RiskLevel): number {
  switch (object) {
    case Issue_RiskLevel.RISK_LEVEL_UNSPECIFIED:
      return 0;
    case Issue_RiskLevel.LOW:
      return 1;
    case Issue_RiskLevel.MODERATE:
      return 2;
    case Issue_RiskLevel.HIGH:
      return 3;
    case Issue_RiskLevel.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface Issue_Approver {
  /** The new status. */
  status: Issue_Approver_Status;
  /** Format: users/hello@world.com */
  principal: string;
}

export enum Issue_Approver_Status {
  STATUS_UNSPECIFIED = "STATUS_UNSPECIFIED",
  PENDING = "PENDING",
  APPROVED = "APPROVED",
  REJECTED = "REJECTED",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function issue_Approver_StatusFromJSON(object: any): Issue_Approver_Status {
  switch (object) {
    case 0:
    case "STATUS_UNSPECIFIED":
      return Issue_Approver_Status.STATUS_UNSPECIFIED;
    case 1:
    case "PENDING":
      return Issue_Approver_Status.PENDING;
    case 2:
    case "APPROVED":
      return Issue_Approver_Status.APPROVED;
    case 3:
    case "REJECTED":
      return Issue_Approver_Status.REJECTED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Issue_Approver_Status.UNRECOGNIZED;
  }
}

export function issue_Approver_StatusToJSON(object: Issue_Approver_Status): string {
  switch (object) {
    case Issue_Approver_Status.STATUS_UNSPECIFIED:
      return "STATUS_UNSPECIFIED";
    case Issue_Approver_Status.PENDING:
      return "PENDING";
    case Issue_Approver_Status.APPROVED:
      return "APPROVED";
    case Issue_Approver_Status.REJECTED:
      return "REJECTED";
    case Issue_Approver_Status.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function issue_Approver_StatusToNumber(object: Issue_Approver_Status): number {
  switch (object) {
    case Issue_Approver_Status.STATUS_UNSPECIFIED:
      return 0;
    case Issue_Approver_Status.PENDING:
      return 1;
    case Issue_Approver_Status.APPROVED:
      return 2;
    case Issue_Approver_Status.REJECTED:
      return 3;
    case Issue_Approver_Status.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface Issue_TaskStatusCountEntry {
  key: string;
  value: number;
}

export interface GrantRequest {
  /**
   * The requested role.
   * Format: roles/EXPORTER.
   */
  role: string;
  /**
   * The user to be granted.
   * Format: users/{email}.
   */
  user: string;
  condition: Expr | undefined;
  expiration: Duration | undefined;
}

export interface ApprovalTemplate {
  flow: ApprovalFlow | undefined;
  title: string;
  description: string;
  /**
   * The name of the creator in users/{email} format.
   * TODO: we should mark it as OUTPUT_ONLY, but currently the frontend will post the approval setting with creator.
   */
  creator: string;
}

export interface ApprovalFlow {
  steps: ApprovalStep[];
}

export interface ApprovalStep {
  type: ApprovalStep_Type;
  nodes: ApprovalNode[];
}

/**
 * Type of the ApprovalStep
 * ALL means every node must be approved to proceed.
 * ANY means approving any node will proceed.
 */
export enum ApprovalStep_Type {
  TYPE_UNSPECIFIED = "TYPE_UNSPECIFIED",
  ALL = "ALL",
  ANY = "ANY",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function approvalStep_TypeFromJSON(object: any): ApprovalStep_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return ApprovalStep_Type.TYPE_UNSPECIFIED;
    case 1:
    case "ALL":
      return ApprovalStep_Type.ALL;
    case 2:
    case "ANY":
      return ApprovalStep_Type.ANY;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ApprovalStep_Type.UNRECOGNIZED;
  }
}

export function approvalStep_TypeToJSON(object: ApprovalStep_Type): string {
  switch (object) {
    case ApprovalStep_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case ApprovalStep_Type.ALL:
      return "ALL";
    case ApprovalStep_Type.ANY:
      return "ANY";
    case ApprovalStep_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function approvalStep_TypeToNumber(object: ApprovalStep_Type): number {
  switch (object) {
    case ApprovalStep_Type.TYPE_UNSPECIFIED:
      return 0;
    case ApprovalStep_Type.ALL:
      return 1;
    case ApprovalStep_Type.ANY:
      return 2;
    case ApprovalStep_Type.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface ApprovalNode {
  type: ApprovalNode_Type;
  groupValue?:
    | ApprovalNode_GroupValue
    | undefined;
  /** Format: roles/{role} */
  role?: string | undefined;
  externalNodeId?: string | undefined;
}

/**
 * Type of the ApprovalNode.
 * type determines who should approve this node.
 * ANY_IN_GROUP means the ApprovalNode can be approved by an user from our predefined user group.
 * See GroupValue below for the predefined user groups.
 */
export enum ApprovalNode_Type {
  TYPE_UNSPECIFIED = "TYPE_UNSPECIFIED",
  ANY_IN_GROUP = "ANY_IN_GROUP",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function approvalNode_TypeFromJSON(object: any): ApprovalNode_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return ApprovalNode_Type.TYPE_UNSPECIFIED;
    case 1:
    case "ANY_IN_GROUP":
      return ApprovalNode_Type.ANY_IN_GROUP;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ApprovalNode_Type.UNRECOGNIZED;
  }
}

export function approvalNode_TypeToJSON(object: ApprovalNode_Type): string {
  switch (object) {
    case ApprovalNode_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case ApprovalNode_Type.ANY_IN_GROUP:
      return "ANY_IN_GROUP";
    case ApprovalNode_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function approvalNode_TypeToNumber(object: ApprovalNode_Type): number {
  switch (object) {
    case ApprovalNode_Type.TYPE_UNSPECIFIED:
      return 0;
    case ApprovalNode_Type.ANY_IN_GROUP:
      return 1;
    case ApprovalNode_Type.UNRECOGNIZED:
    default:
      return -1;
  }
}

/**
 * The predefined user groups are:
 * - WORKSPACE_OWNER
 * - WORKSPACE_DBA
 * - PROJECT_OWNER
 * - PROJECT_MEMBER
 */
export enum ApprovalNode_GroupValue {
  GROUP_VALUE_UNSPECIFILED = "GROUP_VALUE_UNSPECIFILED",
  WORKSPACE_OWNER = "WORKSPACE_OWNER",
  WORKSPACE_DBA = "WORKSPACE_DBA",
  PROJECT_OWNER = "PROJECT_OWNER",
  PROJECT_MEMBER = "PROJECT_MEMBER",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function approvalNode_GroupValueFromJSON(object: any): ApprovalNode_GroupValue {
  switch (object) {
    case 0:
    case "GROUP_VALUE_UNSPECIFILED":
      return ApprovalNode_GroupValue.GROUP_VALUE_UNSPECIFILED;
    case 1:
    case "WORKSPACE_OWNER":
      return ApprovalNode_GroupValue.WORKSPACE_OWNER;
    case 2:
    case "WORKSPACE_DBA":
      return ApprovalNode_GroupValue.WORKSPACE_DBA;
    case 3:
    case "PROJECT_OWNER":
      return ApprovalNode_GroupValue.PROJECT_OWNER;
    case 4:
    case "PROJECT_MEMBER":
      return ApprovalNode_GroupValue.PROJECT_MEMBER;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ApprovalNode_GroupValue.UNRECOGNIZED;
  }
}

export function approvalNode_GroupValueToJSON(object: ApprovalNode_GroupValue): string {
  switch (object) {
    case ApprovalNode_GroupValue.GROUP_VALUE_UNSPECIFILED:
      return "GROUP_VALUE_UNSPECIFILED";
    case ApprovalNode_GroupValue.WORKSPACE_OWNER:
      return "WORKSPACE_OWNER";
    case ApprovalNode_GroupValue.WORKSPACE_DBA:
      return "WORKSPACE_DBA";
    case ApprovalNode_GroupValue.PROJECT_OWNER:
      return "PROJECT_OWNER";
    case ApprovalNode_GroupValue.PROJECT_MEMBER:
      return "PROJECT_MEMBER";
    case ApprovalNode_GroupValue.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function approvalNode_GroupValueToNumber(object: ApprovalNode_GroupValue): number {
  switch (object) {
    case ApprovalNode_GroupValue.GROUP_VALUE_UNSPECIFILED:
      return 0;
    case ApprovalNode_GroupValue.WORKSPACE_OWNER:
      return 1;
    case ApprovalNode_GroupValue.WORKSPACE_DBA:
      return 2;
    case ApprovalNode_GroupValue.PROJECT_OWNER:
      return 3;
    case ApprovalNode_GroupValue.PROJECT_MEMBER:
      return 4;
    case ApprovalNode_GroupValue.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface ListIssueCommentsRequest {
  /** Format: projects/{projects}/issues/{issue} */
  parent: string;
  /**
   * The maximum number of issues to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 issues will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListIssues` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListIssues` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface ListIssueCommentsResponse {
  issueComments: IssueComment[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface CreateIssueCommentRequest {
  /**
   * The issue name
   * Format: projects/{project}/issues/{issue}
   */
  parent: string;
  issueComment: IssueComment | undefined;
}

export interface UpdateIssueCommentRequest {
  /**
   * The issue name
   * Format: projects/{project}/issues/{issue}
   */
  parent: string;
  issueComment:
    | IssueComment
    | undefined;
  /** The list of fields to update. */
  updateMask: string[] | undefined;
}

export interface IssueComment {
  uid: string;
  comment: string;
  /** TODO: use struct message instead. */
  payload: string;
  createTime: Date | undefined;
  updateTime:
    | Date
    | undefined;
  /** Format: projects/{project}/issues/{issue}/issueComments/{issueComment-uid} */
  name: string;
  /** Format: users/{email} */
  creator: string;
  approval?: IssueComment_Approval | undefined;
  issueUpdate?: IssueComment_IssueUpdate | undefined;
  stageEnd?: IssueComment_StageEnd | undefined;
  taskUpdate?: IssueComment_TaskUpdate | undefined;
  taskPriorBackup?: IssueComment_TaskPriorBackup | undefined;
}

export interface IssueComment_Approval {
  status: IssueComment_Approval_Status;
}

export enum IssueComment_Approval_Status {
  STATUS_UNSPECIFIED = "STATUS_UNSPECIFIED",
  PENDING = "PENDING",
  APPROVED = "APPROVED",
  REJECTED = "REJECTED",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function issueComment_Approval_StatusFromJSON(object: any): IssueComment_Approval_Status {
  switch (object) {
    case 0:
    case "STATUS_UNSPECIFIED":
      return IssueComment_Approval_Status.STATUS_UNSPECIFIED;
    case 1:
    case "PENDING":
      return IssueComment_Approval_Status.PENDING;
    case 2:
    case "APPROVED":
      return IssueComment_Approval_Status.APPROVED;
    case 3:
    case "REJECTED":
      return IssueComment_Approval_Status.REJECTED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return IssueComment_Approval_Status.UNRECOGNIZED;
  }
}

export function issueComment_Approval_StatusToJSON(object: IssueComment_Approval_Status): string {
  switch (object) {
    case IssueComment_Approval_Status.STATUS_UNSPECIFIED:
      return "STATUS_UNSPECIFIED";
    case IssueComment_Approval_Status.PENDING:
      return "PENDING";
    case IssueComment_Approval_Status.APPROVED:
      return "APPROVED";
    case IssueComment_Approval_Status.REJECTED:
      return "REJECTED";
    case IssueComment_Approval_Status.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function issueComment_Approval_StatusToNumber(object: IssueComment_Approval_Status): number {
  switch (object) {
    case IssueComment_Approval_Status.STATUS_UNSPECIFIED:
      return 0;
    case IssueComment_Approval_Status.PENDING:
      return 1;
    case IssueComment_Approval_Status.APPROVED:
      return 2;
    case IssueComment_Approval_Status.REJECTED:
      return 3;
    case IssueComment_Approval_Status.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface IssueComment_IssueUpdate {
  fromTitle?: string | undefined;
  toTitle?: string | undefined;
  fromDescription?: string | undefined;
  toDescription?: string | undefined;
  fromStatus?: IssueStatus | undefined;
  toStatus?: IssueStatus | undefined;
  fromLabels: string[];
  toLabels: string[];
}

export interface IssueComment_StageEnd {
  stage: string;
}

export interface IssueComment_TaskUpdate {
  tasks: string[];
  /** Format: projects/{project}/sheets/{sheet} */
  fromSheet?:
    | string
    | undefined;
  /** Format: projects/{project}/sheets/{sheet} */
  toSheet?: string | undefined;
  fromEarliestAllowedTime?: Date | undefined;
  toEarliestAllowedTime?: Date | undefined;
  toStatus?: IssueComment_TaskUpdate_Status | undefined;
}

export enum IssueComment_TaskUpdate_Status {
  STATUS_UNSPECIFIED = "STATUS_UNSPECIFIED",
  PENDING = "PENDING",
  RUNNING = "RUNNING",
  DONE = "DONE",
  FAILED = "FAILED",
  SKIPPED = "SKIPPED",
  CANCELED = "CANCELED",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function issueComment_TaskUpdate_StatusFromJSON(object: any): IssueComment_TaskUpdate_Status {
  switch (object) {
    case 0:
    case "STATUS_UNSPECIFIED":
      return IssueComment_TaskUpdate_Status.STATUS_UNSPECIFIED;
    case 1:
    case "PENDING":
      return IssueComment_TaskUpdate_Status.PENDING;
    case 2:
    case "RUNNING":
      return IssueComment_TaskUpdate_Status.RUNNING;
    case 3:
    case "DONE":
      return IssueComment_TaskUpdate_Status.DONE;
    case 4:
    case "FAILED":
      return IssueComment_TaskUpdate_Status.FAILED;
    case 5:
    case "SKIPPED":
      return IssueComment_TaskUpdate_Status.SKIPPED;
    case 6:
    case "CANCELED":
      return IssueComment_TaskUpdate_Status.CANCELED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return IssueComment_TaskUpdate_Status.UNRECOGNIZED;
  }
}

export function issueComment_TaskUpdate_StatusToJSON(object: IssueComment_TaskUpdate_Status): string {
  switch (object) {
    case IssueComment_TaskUpdate_Status.STATUS_UNSPECIFIED:
      return "STATUS_UNSPECIFIED";
    case IssueComment_TaskUpdate_Status.PENDING:
      return "PENDING";
    case IssueComment_TaskUpdate_Status.RUNNING:
      return "RUNNING";
    case IssueComment_TaskUpdate_Status.DONE:
      return "DONE";
    case IssueComment_TaskUpdate_Status.FAILED:
      return "FAILED";
    case IssueComment_TaskUpdate_Status.SKIPPED:
      return "SKIPPED";
    case IssueComment_TaskUpdate_Status.CANCELED:
      return "CANCELED";
    case IssueComment_TaskUpdate_Status.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function issueComment_TaskUpdate_StatusToNumber(object: IssueComment_TaskUpdate_Status): number {
  switch (object) {
    case IssueComment_TaskUpdate_Status.STATUS_UNSPECIFIED:
      return 0;
    case IssueComment_TaskUpdate_Status.PENDING:
      return 1;
    case IssueComment_TaskUpdate_Status.RUNNING:
      return 2;
    case IssueComment_TaskUpdate_Status.DONE:
      return 3;
    case IssueComment_TaskUpdate_Status.FAILED:
      return 4;
    case IssueComment_TaskUpdate_Status.SKIPPED:
      return 5;
    case IssueComment_TaskUpdate_Status.CANCELED:
      return 6;
    case IssueComment_TaskUpdate_Status.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface IssueComment_TaskPriorBackup {
  task: string;
  tables: IssueComment_TaskPriorBackup_Table[];
  originalLine?: number | undefined;
  database: string;
}

export interface IssueComment_TaskPriorBackup_Table {
  schema: string;
  table: string;
}

function createBaseGetIssueRequest(): GetIssueRequest {
  return { name: "", force: false };
}

export const GetIssueRequest = {
  encode(message: GetIssueRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.force === true) {
      writer.uint32(16).bool(message.force);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetIssueRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetIssueRequest();
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

  fromJSON(object: any): GetIssueRequest {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      force: isSet(object.force) ? globalThis.Boolean(object.force) : false,
    };
  },

  toJSON(message: GetIssueRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.force === true) {
      obj.force = message.force;
    }
    return obj;
  },

  create(base?: DeepPartial<GetIssueRequest>): GetIssueRequest {
    return GetIssueRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<GetIssueRequest>): GetIssueRequest {
    const message = createBaseGetIssueRequest();
    message.name = object.name ?? "";
    message.force = object.force ?? false;
    return message;
  },
};

function createBaseCreateIssueRequest(): CreateIssueRequest {
  return { parent: "", issue: undefined };
}

export const CreateIssueRequest = {
  encode(message: CreateIssueRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.issue !== undefined) {
      Issue.encode(message.issue, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateIssueRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateIssueRequest();
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

          message.issue = Issue.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateIssueRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      issue: isSet(object.issue) ? Issue.fromJSON(object.issue) : undefined,
    };
  },

  toJSON(message: CreateIssueRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.issue !== undefined) {
      obj.issue = Issue.toJSON(message.issue);
    }
    return obj;
  },

  create(base?: DeepPartial<CreateIssueRequest>): CreateIssueRequest {
    return CreateIssueRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<CreateIssueRequest>): CreateIssueRequest {
    const message = createBaseCreateIssueRequest();
    message.parent = object.parent ?? "";
    message.issue = (object.issue !== undefined && object.issue !== null) ? Issue.fromPartial(object.issue) : undefined;
    return message;
  },
};

function createBaseListIssuesRequest(): ListIssuesRequest {
  return { parent: "", pageSize: 0, pageToken: "", filter: "", query: "" };
}

export const ListIssuesRequest = {
  encode(message: ListIssuesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.pageSize !== 0) {
      writer.uint32(16).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(26).string(message.pageToken);
    }
    if (message.filter !== "") {
      writer.uint32(34).string(message.filter);
    }
    if (message.query !== "") {
      writer.uint32(42).string(message.query);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListIssuesRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListIssuesRequest();
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
        case 4:
          if (tag !== 34) {
            break;
          }

          message.filter = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.query = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListIssuesRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
      filter: isSet(object.filter) ? globalThis.String(object.filter) : "",
      query: isSet(object.query) ? globalThis.String(object.query) : "",
    };
  },

  toJSON(message: ListIssuesRequest): unknown {
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
    if (message.filter !== "") {
      obj.filter = message.filter;
    }
    if (message.query !== "") {
      obj.query = message.query;
    }
    return obj;
  },

  create(base?: DeepPartial<ListIssuesRequest>): ListIssuesRequest {
    return ListIssuesRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListIssuesRequest>): ListIssuesRequest {
    const message = createBaseListIssuesRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    message.filter = object.filter ?? "";
    message.query = object.query ?? "";
    return message;
  },
};

function createBaseListIssuesResponse(): ListIssuesResponse {
  return { issues: [], nextPageToken: "" };
}

export const ListIssuesResponse = {
  encode(message: ListIssuesResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.issues) {
      Issue.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListIssuesResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListIssuesResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.issues.push(Issue.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListIssuesResponse {
    return {
      issues: globalThis.Array.isArray(object?.issues) ? object.issues.map((e: any) => Issue.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListIssuesResponse): unknown {
    const obj: any = {};
    if (message.issues?.length) {
      obj.issues = message.issues.map((e) => Issue.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListIssuesResponse>): ListIssuesResponse {
    return ListIssuesResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListIssuesResponse>): ListIssuesResponse {
    const message = createBaseListIssuesResponse();
    message.issues = object.issues?.map((e) => Issue.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseSearchIssuesRequest(): SearchIssuesRequest {
  return { parent: "", pageSize: 0, pageToken: "", filter: "", query: "" };
}

export const SearchIssuesRequest = {
  encode(message: SearchIssuesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.pageSize !== 0) {
      writer.uint32(16).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(26).string(message.pageToken);
    }
    if (message.filter !== "") {
      writer.uint32(34).string(message.filter);
    }
    if (message.query !== "") {
      writer.uint32(42).string(message.query);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SearchIssuesRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSearchIssuesRequest();
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
        case 4:
          if (tag !== 34) {
            break;
          }

          message.filter = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.query = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SearchIssuesRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
      filter: isSet(object.filter) ? globalThis.String(object.filter) : "",
      query: isSet(object.query) ? globalThis.String(object.query) : "",
    };
  },

  toJSON(message: SearchIssuesRequest): unknown {
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
    if (message.filter !== "") {
      obj.filter = message.filter;
    }
    if (message.query !== "") {
      obj.query = message.query;
    }
    return obj;
  },

  create(base?: DeepPartial<SearchIssuesRequest>): SearchIssuesRequest {
    return SearchIssuesRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SearchIssuesRequest>): SearchIssuesRequest {
    const message = createBaseSearchIssuesRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    message.filter = object.filter ?? "";
    message.query = object.query ?? "";
    return message;
  },
};

function createBaseSearchIssuesResponse(): SearchIssuesResponse {
  return { issues: [], nextPageToken: "" };
}

export const SearchIssuesResponse = {
  encode(message: SearchIssuesResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.issues) {
      Issue.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SearchIssuesResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSearchIssuesResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.issues.push(Issue.decode(reader, reader.uint32()));
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

  fromJSON(object: any): SearchIssuesResponse {
    return {
      issues: globalThis.Array.isArray(object?.issues) ? object.issues.map((e: any) => Issue.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: SearchIssuesResponse): unknown {
    const obj: any = {};
    if (message.issues?.length) {
      obj.issues = message.issues.map((e) => Issue.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<SearchIssuesResponse>): SearchIssuesResponse {
    return SearchIssuesResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SearchIssuesResponse>): SearchIssuesResponse {
    const message = createBaseSearchIssuesResponse();
    message.issues = object.issues?.map((e) => Issue.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseUpdateIssueRequest(): UpdateIssueRequest {
  return { issue: undefined, updateMask: undefined };
}

export const UpdateIssueRequest = {
  encode(message: UpdateIssueRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.issue !== undefined) {
      Issue.encode(message.issue, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateIssueRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateIssueRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.issue = Issue.decode(reader, reader.uint32());
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

  fromJSON(object: any): UpdateIssueRequest {
    return {
      issue: isSet(object.issue) ? Issue.fromJSON(object.issue) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateIssueRequest): unknown {
    const obj: any = {};
    if (message.issue !== undefined) {
      obj.issue = Issue.toJSON(message.issue);
    }
    if (message.updateMask !== undefined) {
      obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask));
    }
    return obj;
  },

  create(base?: DeepPartial<UpdateIssueRequest>): UpdateIssueRequest {
    return UpdateIssueRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UpdateIssueRequest>): UpdateIssueRequest {
    const message = createBaseUpdateIssueRequest();
    message.issue = (object.issue !== undefined && object.issue !== null) ? Issue.fromPartial(object.issue) : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseBatchUpdateIssuesStatusRequest(): BatchUpdateIssuesStatusRequest {
  return { parent: "", issues: [], status: IssueStatus.ISSUE_STATUS_UNSPECIFIED, reason: "" };
}

export const BatchUpdateIssuesStatusRequest = {
  encode(message: BatchUpdateIssuesStatusRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    for (const v of message.issues) {
      writer.uint32(18).string(v!);
    }
    if (message.status !== IssueStatus.ISSUE_STATUS_UNSPECIFIED) {
      writer.uint32(24).int32(issueStatusToNumber(message.status));
    }
    if (message.reason !== "") {
      writer.uint32(34).string(message.reason);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchUpdateIssuesStatusRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchUpdateIssuesStatusRequest();
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

          message.issues.push(reader.string());
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.status = issueStatusFromJSON(reader.int32());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.reason = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BatchUpdateIssuesStatusRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      issues: globalThis.Array.isArray(object?.issues) ? object.issues.map((e: any) => globalThis.String(e)) : [],
      status: isSet(object.status) ? issueStatusFromJSON(object.status) : IssueStatus.ISSUE_STATUS_UNSPECIFIED,
      reason: isSet(object.reason) ? globalThis.String(object.reason) : "",
    };
  },

  toJSON(message: BatchUpdateIssuesStatusRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.issues?.length) {
      obj.issues = message.issues;
    }
    if (message.status !== IssueStatus.ISSUE_STATUS_UNSPECIFIED) {
      obj.status = issueStatusToJSON(message.status);
    }
    if (message.reason !== "") {
      obj.reason = message.reason;
    }
    return obj;
  },

  create(base?: DeepPartial<BatchUpdateIssuesStatusRequest>): BatchUpdateIssuesStatusRequest {
    return BatchUpdateIssuesStatusRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BatchUpdateIssuesStatusRequest>): BatchUpdateIssuesStatusRequest {
    const message = createBaseBatchUpdateIssuesStatusRequest();
    message.parent = object.parent ?? "";
    message.issues = object.issues?.map((e) => e) || [];
    message.status = object.status ?? IssueStatus.ISSUE_STATUS_UNSPECIFIED;
    message.reason = object.reason ?? "";
    return message;
  },
};

function createBaseBatchUpdateIssuesStatusResponse(): BatchUpdateIssuesStatusResponse {
  return {};
}

export const BatchUpdateIssuesStatusResponse = {
  encode(_: BatchUpdateIssuesStatusResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchUpdateIssuesStatusResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchUpdateIssuesStatusResponse();
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

  fromJSON(_: any): BatchUpdateIssuesStatusResponse {
    return {};
  },

  toJSON(_: BatchUpdateIssuesStatusResponse): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<BatchUpdateIssuesStatusResponse>): BatchUpdateIssuesStatusResponse {
    return BatchUpdateIssuesStatusResponse.fromPartial(base ?? {});
  },
  fromPartial(_: DeepPartial<BatchUpdateIssuesStatusResponse>): BatchUpdateIssuesStatusResponse {
    const message = createBaseBatchUpdateIssuesStatusResponse();
    return message;
  },
};

function createBaseApproveIssueRequest(): ApproveIssueRequest {
  return { name: "", comment: "" };
}

export const ApproveIssueRequest = {
  encode(message: ApproveIssueRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.comment !== "") {
      writer.uint32(18).string(message.comment);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ApproveIssueRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApproveIssueRequest();
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

  fromJSON(object: any): ApproveIssueRequest {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      comment: isSet(object.comment) ? globalThis.String(object.comment) : "",
    };
  },

  toJSON(message: ApproveIssueRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.comment !== "") {
      obj.comment = message.comment;
    }
    return obj;
  },

  create(base?: DeepPartial<ApproveIssueRequest>): ApproveIssueRequest {
    return ApproveIssueRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ApproveIssueRequest>): ApproveIssueRequest {
    const message = createBaseApproveIssueRequest();
    message.name = object.name ?? "";
    message.comment = object.comment ?? "";
    return message;
  },
};

function createBaseRejectIssueRequest(): RejectIssueRequest {
  return { name: "", comment: "" };
}

export const RejectIssueRequest = {
  encode(message: RejectIssueRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.comment !== "") {
      writer.uint32(18).string(message.comment);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RejectIssueRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRejectIssueRequest();
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

  fromJSON(object: any): RejectIssueRequest {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      comment: isSet(object.comment) ? globalThis.String(object.comment) : "",
    };
  },

  toJSON(message: RejectIssueRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.comment !== "") {
      obj.comment = message.comment;
    }
    return obj;
  },

  create(base?: DeepPartial<RejectIssueRequest>): RejectIssueRequest {
    return RejectIssueRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<RejectIssueRequest>): RejectIssueRequest {
    const message = createBaseRejectIssueRequest();
    message.name = object.name ?? "";
    message.comment = object.comment ?? "";
    return message;
  },
};

function createBaseRequestIssueRequest(): RequestIssueRequest {
  return { name: "", comment: "" };
}

export const RequestIssueRequest = {
  encode(message: RequestIssueRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.comment !== "") {
      writer.uint32(18).string(message.comment);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RequestIssueRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRequestIssueRequest();
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

  fromJSON(object: any): RequestIssueRequest {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      comment: isSet(object.comment) ? globalThis.String(object.comment) : "",
    };
  },

  toJSON(message: RequestIssueRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.comment !== "") {
      obj.comment = message.comment;
    }
    return obj;
  },

  create(base?: DeepPartial<RequestIssueRequest>): RequestIssueRequest {
    return RequestIssueRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<RequestIssueRequest>): RequestIssueRequest {
    const message = createBaseRequestIssueRequest();
    message.name = object.name ?? "";
    message.comment = object.comment ?? "";
    return message;
  },
};

function createBaseIssue(): Issue {
  return {
    name: "",
    uid: "",
    title: "",
    description: "",
    type: Issue_Type.TYPE_UNSPECIFIED,
    status: IssueStatus.ISSUE_STATUS_UNSPECIFIED,
    assignee: "",
    assigneeAttention: false,
    approvers: [],
    approvalTemplates: [],
    approvalFindingDone: false,
    approvalFindingError: "",
    subscribers: [],
    creator: "",
    createTime: undefined,
    updateTime: undefined,
    plan: "",
    rollout: "",
    grantRequest: undefined,
    releasers: [],
    riskLevel: Issue_RiskLevel.RISK_LEVEL_UNSPECIFIED,
    taskStatusCount: {},
    labels: [],
  };
}

export const Issue = {
  encode(message: Issue, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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
    if (message.type !== Issue_Type.TYPE_UNSPECIFIED) {
      writer.uint32(40).int32(issue_TypeToNumber(message.type));
    }
    if (message.status !== IssueStatus.ISSUE_STATUS_UNSPECIFIED) {
      writer.uint32(48).int32(issueStatusToNumber(message.status));
    }
    if (message.assignee !== "") {
      writer.uint32(58).string(message.assignee);
    }
    if (message.assigneeAttention === true) {
      writer.uint32(64).bool(message.assigneeAttention);
    }
    for (const v of message.approvers) {
      Issue_Approver.encode(v!, writer.uint32(74).fork()).ldelim();
    }
    for (const v of message.approvalTemplates) {
      ApprovalTemplate.encode(v!, writer.uint32(82).fork()).ldelim();
    }
    if (message.approvalFindingDone === true) {
      writer.uint32(88).bool(message.approvalFindingDone);
    }
    if (message.approvalFindingError !== "") {
      writer.uint32(98).string(message.approvalFindingError);
    }
    for (const v of message.subscribers) {
      writer.uint32(106).string(v!);
    }
    if (message.creator !== "") {
      writer.uint32(114).string(message.creator);
    }
    if (message.createTime !== undefined) {
      Timestamp.encode(toTimestamp(message.createTime), writer.uint32(122).fork()).ldelim();
    }
    if (message.updateTime !== undefined) {
      Timestamp.encode(toTimestamp(message.updateTime), writer.uint32(130).fork()).ldelim();
    }
    if (message.plan !== "") {
      writer.uint32(138).string(message.plan);
    }
    if (message.rollout !== "") {
      writer.uint32(146).string(message.rollout);
    }
    if (message.grantRequest !== undefined) {
      GrantRequest.encode(message.grantRequest, writer.uint32(154).fork()).ldelim();
    }
    for (const v of message.releasers) {
      writer.uint32(162).string(v!);
    }
    if (message.riskLevel !== Issue_RiskLevel.RISK_LEVEL_UNSPECIFIED) {
      writer.uint32(168).int32(issue_RiskLevelToNumber(message.riskLevel));
    }
    Object.entries(message.taskStatusCount).forEach(([key, value]) => {
      Issue_TaskStatusCountEntry.encode({ key: key as any, value }, writer.uint32(178).fork()).ldelim();
    });
    for (const v of message.labels) {
      writer.uint32(186).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Issue {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIssue();
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
          if (tag !== 26) {
            break;
          }

          message.title = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.description = reader.string();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.type = issue_TypeFromJSON(reader.int32());
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.status = issueStatusFromJSON(reader.int32());
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.assignee = reader.string();
          continue;
        case 8:
          if (tag !== 64) {
            break;
          }

          message.assigneeAttention = reader.bool();
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.approvers.push(Issue_Approver.decode(reader, reader.uint32()));
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.approvalTemplates.push(ApprovalTemplate.decode(reader, reader.uint32()));
          continue;
        case 11:
          if (tag !== 88) {
            break;
          }

          message.approvalFindingDone = reader.bool();
          continue;
        case 12:
          if (tag !== 98) {
            break;
          }

          message.approvalFindingError = reader.string();
          continue;
        case 13:
          if (tag !== 106) {
            break;
          }

          message.subscribers.push(reader.string());
          continue;
        case 14:
          if (tag !== 114) {
            break;
          }

          message.creator = reader.string();
          continue;
        case 15:
          if (tag !== 122) {
            break;
          }

          message.createTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 16:
          if (tag !== 130) {
            break;
          }

          message.updateTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 17:
          if (tag !== 138) {
            break;
          }

          message.plan = reader.string();
          continue;
        case 18:
          if (tag !== 146) {
            break;
          }

          message.rollout = reader.string();
          continue;
        case 19:
          if (tag !== 154) {
            break;
          }

          message.grantRequest = GrantRequest.decode(reader, reader.uint32());
          continue;
        case 20:
          if (tag !== 162) {
            break;
          }

          message.releasers.push(reader.string());
          continue;
        case 21:
          if (tag !== 168) {
            break;
          }

          message.riskLevel = issue_RiskLevelFromJSON(reader.int32());
          continue;
        case 22:
          if (tag !== 178) {
            break;
          }

          const entry22 = Issue_TaskStatusCountEntry.decode(reader, reader.uint32());
          if (entry22.value !== undefined) {
            message.taskStatusCount[entry22.key] = entry22.value;
          }
          continue;
        case 23:
          if (tag !== 186) {
            break;
          }

          message.labels.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Issue {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      uid: isSet(object.uid) ? globalThis.String(object.uid) : "",
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      description: isSet(object.description) ? globalThis.String(object.description) : "",
      type: isSet(object.type) ? issue_TypeFromJSON(object.type) : Issue_Type.TYPE_UNSPECIFIED,
      status: isSet(object.status) ? issueStatusFromJSON(object.status) : IssueStatus.ISSUE_STATUS_UNSPECIFIED,
      assignee: isSet(object.assignee) ? globalThis.String(object.assignee) : "",
      assigneeAttention: isSet(object.assigneeAttention) ? globalThis.Boolean(object.assigneeAttention) : false,
      approvers: globalThis.Array.isArray(object?.approvers)
        ? object.approvers.map((e: any) => Issue_Approver.fromJSON(e))
        : [],
      approvalTemplates: globalThis.Array.isArray(object?.approvalTemplates)
        ? object.approvalTemplates.map((e: any) => ApprovalTemplate.fromJSON(e))
        : [],
      approvalFindingDone: isSet(object.approvalFindingDone) ? globalThis.Boolean(object.approvalFindingDone) : false,
      approvalFindingError: isSet(object.approvalFindingError) ? globalThis.String(object.approvalFindingError) : "",
      subscribers: globalThis.Array.isArray(object?.subscribers)
        ? object.subscribers.map((e: any) => globalThis.String(e))
        : [],
      creator: isSet(object.creator) ? globalThis.String(object.creator) : "",
      createTime: isSet(object.createTime) ? fromJsonTimestamp(object.createTime) : undefined,
      updateTime: isSet(object.updateTime) ? fromJsonTimestamp(object.updateTime) : undefined,
      plan: isSet(object.plan) ? globalThis.String(object.plan) : "",
      rollout: isSet(object.rollout) ? globalThis.String(object.rollout) : "",
      grantRequest: isSet(object.grantRequest) ? GrantRequest.fromJSON(object.grantRequest) : undefined,
      releasers: globalThis.Array.isArray(object?.releasers)
        ? object.releasers.map((e: any) => globalThis.String(e))
        : [],
      riskLevel: isSet(object.riskLevel)
        ? issue_RiskLevelFromJSON(object.riskLevel)
        : Issue_RiskLevel.RISK_LEVEL_UNSPECIFIED,
      taskStatusCount: isObject(object.taskStatusCount)
        ? Object.entries(object.taskStatusCount).reduce<{ [key: string]: number }>((acc, [key, value]) => {
          acc[key] = Number(value);
          return acc;
        }, {})
        : {},
      labels: globalThis.Array.isArray(object?.labels)
        ? object.labels.map((e: any) => globalThis.String(e))
        : [],
    };
  },

  toJSON(message: Issue): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.uid !== "") {
      obj.uid = message.uid;
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.description !== "") {
      obj.description = message.description;
    }
    if (message.type !== Issue_Type.TYPE_UNSPECIFIED) {
      obj.type = issue_TypeToJSON(message.type);
    }
    if (message.status !== IssueStatus.ISSUE_STATUS_UNSPECIFIED) {
      obj.status = issueStatusToJSON(message.status);
    }
    if (message.assignee !== "") {
      obj.assignee = message.assignee;
    }
    if (message.assigneeAttention === true) {
      obj.assigneeAttention = message.assigneeAttention;
    }
    if (message.approvers?.length) {
      obj.approvers = message.approvers.map((e) => Issue_Approver.toJSON(e));
    }
    if (message.approvalTemplates?.length) {
      obj.approvalTemplates = message.approvalTemplates.map((e) => ApprovalTemplate.toJSON(e));
    }
    if (message.approvalFindingDone === true) {
      obj.approvalFindingDone = message.approvalFindingDone;
    }
    if (message.approvalFindingError !== "") {
      obj.approvalFindingError = message.approvalFindingError;
    }
    if (message.subscribers?.length) {
      obj.subscribers = message.subscribers;
    }
    if (message.creator !== "") {
      obj.creator = message.creator;
    }
    if (message.createTime !== undefined) {
      obj.createTime = message.createTime.toISOString();
    }
    if (message.updateTime !== undefined) {
      obj.updateTime = message.updateTime.toISOString();
    }
    if (message.plan !== "") {
      obj.plan = message.plan;
    }
    if (message.rollout !== "") {
      obj.rollout = message.rollout;
    }
    if (message.grantRequest !== undefined) {
      obj.grantRequest = GrantRequest.toJSON(message.grantRequest);
    }
    if (message.releasers?.length) {
      obj.releasers = message.releasers;
    }
    if (message.riskLevel !== Issue_RiskLevel.RISK_LEVEL_UNSPECIFIED) {
      obj.riskLevel = issue_RiskLevelToJSON(message.riskLevel);
    }
    if (message.taskStatusCount) {
      const entries = Object.entries(message.taskStatusCount);
      if (entries.length > 0) {
        obj.taskStatusCount = {};
        entries.forEach(([k, v]) => {
          obj.taskStatusCount[k] = Math.round(v);
        });
      }
    }
    if (message.labels?.length) {
      obj.labels = message.labels;
    }
    return obj;
  },

  create(base?: DeepPartial<Issue>): Issue {
    return Issue.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Issue>): Issue {
    const message = createBaseIssue();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.type = object.type ?? Issue_Type.TYPE_UNSPECIFIED;
    message.status = object.status ?? IssueStatus.ISSUE_STATUS_UNSPECIFIED;
    message.assignee = object.assignee ?? "";
    message.assigneeAttention = object.assigneeAttention ?? false;
    message.approvers = object.approvers?.map((e) => Issue_Approver.fromPartial(e)) || [];
    message.approvalTemplates = object.approvalTemplates?.map((e) => ApprovalTemplate.fromPartial(e)) || [];
    message.approvalFindingDone = object.approvalFindingDone ?? false;
    message.approvalFindingError = object.approvalFindingError ?? "";
    message.subscribers = object.subscribers?.map((e) => e) || [];
    message.creator = object.creator ?? "";
    message.createTime = object.createTime ?? undefined;
    message.updateTime = object.updateTime ?? undefined;
    message.plan = object.plan ?? "";
    message.rollout = object.rollout ?? "";
    message.grantRequest = (object.grantRequest !== undefined && object.grantRequest !== null)
      ? GrantRequest.fromPartial(object.grantRequest)
      : undefined;
    message.releasers = object.releasers?.map((e) => e) || [];
    message.riskLevel = object.riskLevel ?? Issue_RiskLevel.RISK_LEVEL_UNSPECIFIED;
    message.taskStatusCount = Object.entries(object.taskStatusCount ?? {}).reduce<{ [key: string]: number }>(
      (acc, [key, value]) => {
        if (value !== undefined) {
          acc[key] = globalThis.Number(value);
        }
        return acc;
      },
      {},
    );
    message.labels = object.labels?.map((e) => e) || [];
    return message;
  },
};

function createBaseIssue_Approver(): Issue_Approver {
  return { status: Issue_Approver_Status.STATUS_UNSPECIFIED, principal: "" };
}

export const Issue_Approver = {
  encode(message: Issue_Approver, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.status !== Issue_Approver_Status.STATUS_UNSPECIFIED) {
      writer.uint32(8).int32(issue_Approver_StatusToNumber(message.status));
    }
    if (message.principal !== "") {
      writer.uint32(18).string(message.principal);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Issue_Approver {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIssue_Approver();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.status = issue_Approver_StatusFromJSON(reader.int32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.principal = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Issue_Approver {
    return {
      status: isSet(object.status)
        ? issue_Approver_StatusFromJSON(object.status)
        : Issue_Approver_Status.STATUS_UNSPECIFIED,
      principal: isSet(object.principal) ? globalThis.String(object.principal) : "",
    };
  },

  toJSON(message: Issue_Approver): unknown {
    const obj: any = {};
    if (message.status !== Issue_Approver_Status.STATUS_UNSPECIFIED) {
      obj.status = issue_Approver_StatusToJSON(message.status);
    }
    if (message.principal !== "") {
      obj.principal = message.principal;
    }
    return obj;
  },

  create(base?: DeepPartial<Issue_Approver>): Issue_Approver {
    return Issue_Approver.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Issue_Approver>): Issue_Approver {
    const message = createBaseIssue_Approver();
    message.status = object.status ?? Issue_Approver_Status.STATUS_UNSPECIFIED;
    message.principal = object.principal ?? "";
    return message;
  },
};

function createBaseIssue_TaskStatusCountEntry(): Issue_TaskStatusCountEntry {
  return { key: "", value: 0 };
}

export const Issue_TaskStatusCountEntry = {
  encode(message: Issue_TaskStatusCountEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== 0) {
      writer.uint32(16).int32(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Issue_TaskStatusCountEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIssue_TaskStatusCountEntry();
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

          message.value = reader.int32();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Issue_TaskStatusCountEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? globalThis.Number(object.value) : 0,
    };
  },

  toJSON(message: Issue_TaskStatusCountEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== 0) {
      obj.value = Math.round(message.value);
    }
    return obj;
  },

  create(base?: DeepPartial<Issue_TaskStatusCountEntry>): Issue_TaskStatusCountEntry {
    return Issue_TaskStatusCountEntry.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Issue_TaskStatusCountEntry>): Issue_TaskStatusCountEntry {
    const message = createBaseIssue_TaskStatusCountEntry();
    message.key = object.key ?? "";
    message.value = object.value ?? 0;
    return message;
  },
};

function createBaseGrantRequest(): GrantRequest {
  return { role: "", user: "", condition: undefined, expiration: undefined };
}

export const GrantRequest = {
  encode(message: GrantRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.role !== "") {
      writer.uint32(10).string(message.role);
    }
    if (message.user !== "") {
      writer.uint32(18).string(message.user);
    }
    if (message.condition !== undefined) {
      Expr.encode(message.condition, writer.uint32(26).fork()).ldelim();
    }
    if (message.expiration !== undefined) {
      Duration.encode(message.expiration, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GrantRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGrantRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.role = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.user = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.condition = Expr.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.expiration = Duration.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GrantRequest {
    return {
      role: isSet(object.role) ? globalThis.String(object.role) : "",
      user: isSet(object.user) ? globalThis.String(object.user) : "",
      condition: isSet(object.condition) ? Expr.fromJSON(object.condition) : undefined,
      expiration: isSet(object.expiration) ? Duration.fromJSON(object.expiration) : undefined,
    };
  },

  toJSON(message: GrantRequest): unknown {
    const obj: any = {};
    if (message.role !== "") {
      obj.role = message.role;
    }
    if (message.user !== "") {
      obj.user = message.user;
    }
    if (message.condition !== undefined) {
      obj.condition = Expr.toJSON(message.condition);
    }
    if (message.expiration !== undefined) {
      obj.expiration = Duration.toJSON(message.expiration);
    }
    return obj;
  },

  create(base?: DeepPartial<GrantRequest>): GrantRequest {
    return GrantRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<GrantRequest>): GrantRequest {
    const message = createBaseGrantRequest();
    message.role = object.role ?? "";
    message.user = object.user ?? "";
    message.condition = (object.condition !== undefined && object.condition !== null)
      ? Expr.fromPartial(object.condition)
      : undefined;
    message.expiration = (object.expiration !== undefined && object.expiration !== null)
      ? Duration.fromPartial(object.expiration)
      : undefined;
    return message;
  },
};

function createBaseApprovalTemplate(): ApprovalTemplate {
  return { flow: undefined, title: "", description: "", creator: "" };
}

export const ApprovalTemplate = {
  encode(message: ApprovalTemplate, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.flow !== undefined) {
      ApprovalFlow.encode(message.flow, writer.uint32(10).fork()).ldelim();
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    if (message.creator !== "") {
      writer.uint32(34).string(message.creator);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ApprovalTemplate {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApprovalTemplate();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.flow = ApprovalFlow.decode(reader, reader.uint32());
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

          message.description = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.creator = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ApprovalTemplate {
    return {
      flow: isSet(object.flow) ? ApprovalFlow.fromJSON(object.flow) : undefined,
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      description: isSet(object.description) ? globalThis.String(object.description) : "",
      creator: isSet(object.creator) ? globalThis.String(object.creator) : "",
    };
  },

  toJSON(message: ApprovalTemplate): unknown {
    const obj: any = {};
    if (message.flow !== undefined) {
      obj.flow = ApprovalFlow.toJSON(message.flow);
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.description !== "") {
      obj.description = message.description;
    }
    if (message.creator !== "") {
      obj.creator = message.creator;
    }
    return obj;
  },

  create(base?: DeepPartial<ApprovalTemplate>): ApprovalTemplate {
    return ApprovalTemplate.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ApprovalTemplate>): ApprovalTemplate {
    const message = createBaseApprovalTemplate();
    message.flow = (object.flow !== undefined && object.flow !== null)
      ? ApprovalFlow.fromPartial(object.flow)
      : undefined;
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.creator = object.creator ?? "";
    return message;
  },
};

function createBaseApprovalFlow(): ApprovalFlow {
  return { steps: [] };
}

export const ApprovalFlow = {
  encode(message: ApprovalFlow, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.steps) {
      ApprovalStep.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ApprovalFlow {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApprovalFlow();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.steps.push(ApprovalStep.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ApprovalFlow {
    return {
      steps: globalThis.Array.isArray(object?.steps) ? object.steps.map((e: any) => ApprovalStep.fromJSON(e)) : [],
    };
  },

  toJSON(message: ApprovalFlow): unknown {
    const obj: any = {};
    if (message.steps?.length) {
      obj.steps = message.steps.map((e) => ApprovalStep.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<ApprovalFlow>): ApprovalFlow {
    return ApprovalFlow.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ApprovalFlow>): ApprovalFlow {
    const message = createBaseApprovalFlow();
    message.steps = object.steps?.map((e) => ApprovalStep.fromPartial(e)) || [];
    return message;
  },
};

function createBaseApprovalStep(): ApprovalStep {
  return { type: ApprovalStep_Type.TYPE_UNSPECIFIED, nodes: [] };
}

export const ApprovalStep = {
  encode(message: ApprovalStep, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.type !== ApprovalStep_Type.TYPE_UNSPECIFIED) {
      writer.uint32(8).int32(approvalStep_TypeToNumber(message.type));
    }
    for (const v of message.nodes) {
      ApprovalNode.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ApprovalStep {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApprovalStep();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.type = approvalStep_TypeFromJSON(reader.int32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.nodes.push(ApprovalNode.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ApprovalStep {
    return {
      type: isSet(object.type) ? approvalStep_TypeFromJSON(object.type) : ApprovalStep_Type.TYPE_UNSPECIFIED,
      nodes: globalThis.Array.isArray(object?.nodes) ? object.nodes.map((e: any) => ApprovalNode.fromJSON(e)) : [],
    };
  },

  toJSON(message: ApprovalStep): unknown {
    const obj: any = {};
    if (message.type !== ApprovalStep_Type.TYPE_UNSPECIFIED) {
      obj.type = approvalStep_TypeToJSON(message.type);
    }
    if (message.nodes?.length) {
      obj.nodes = message.nodes.map((e) => ApprovalNode.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<ApprovalStep>): ApprovalStep {
    return ApprovalStep.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ApprovalStep>): ApprovalStep {
    const message = createBaseApprovalStep();
    message.type = object.type ?? ApprovalStep_Type.TYPE_UNSPECIFIED;
    message.nodes = object.nodes?.map((e) => ApprovalNode.fromPartial(e)) || [];
    return message;
  },
};

function createBaseApprovalNode(): ApprovalNode {
  return {
    type: ApprovalNode_Type.TYPE_UNSPECIFIED,
    groupValue: undefined,
    role: undefined,
    externalNodeId: undefined,
  };
}

export const ApprovalNode = {
  encode(message: ApprovalNode, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.type !== ApprovalNode_Type.TYPE_UNSPECIFIED) {
      writer.uint32(8).int32(approvalNode_TypeToNumber(message.type));
    }
    if (message.groupValue !== undefined) {
      writer.uint32(16).int32(approvalNode_GroupValueToNumber(message.groupValue));
    }
    if (message.role !== undefined) {
      writer.uint32(26).string(message.role);
    }
    if (message.externalNodeId !== undefined) {
      writer.uint32(34).string(message.externalNodeId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ApprovalNode {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseApprovalNode();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.type = approvalNode_TypeFromJSON(reader.int32());
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.groupValue = approvalNode_GroupValueFromJSON(reader.int32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.role = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.externalNodeId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ApprovalNode {
    return {
      type: isSet(object.type) ? approvalNode_TypeFromJSON(object.type) : ApprovalNode_Type.TYPE_UNSPECIFIED,
      groupValue: isSet(object.groupValue) ? approvalNode_GroupValueFromJSON(object.groupValue) : undefined,
      role: isSet(object.role) ? globalThis.String(object.role) : undefined,
      externalNodeId: isSet(object.externalNodeId) ? globalThis.String(object.externalNodeId) : undefined,
    };
  },

  toJSON(message: ApprovalNode): unknown {
    const obj: any = {};
    if (message.type !== ApprovalNode_Type.TYPE_UNSPECIFIED) {
      obj.type = approvalNode_TypeToJSON(message.type);
    }
    if (message.groupValue !== undefined) {
      obj.groupValue = approvalNode_GroupValueToJSON(message.groupValue);
    }
    if (message.role !== undefined) {
      obj.role = message.role;
    }
    if (message.externalNodeId !== undefined) {
      obj.externalNodeId = message.externalNodeId;
    }
    return obj;
  },

  create(base?: DeepPartial<ApprovalNode>): ApprovalNode {
    return ApprovalNode.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ApprovalNode>): ApprovalNode {
    const message = createBaseApprovalNode();
    message.type = object.type ?? ApprovalNode_Type.TYPE_UNSPECIFIED;
    message.groupValue = object.groupValue ?? undefined;
    message.role = object.role ?? undefined;
    message.externalNodeId = object.externalNodeId ?? undefined;
    return message;
  },
};

function createBaseListIssueCommentsRequest(): ListIssueCommentsRequest {
  return { parent: "", pageSize: 0, pageToken: "" };
}

export const ListIssueCommentsRequest = {
  encode(message: ListIssueCommentsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): ListIssueCommentsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListIssueCommentsRequest();
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

  fromJSON(object: any): ListIssueCommentsRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
    };
  },

  toJSON(message: ListIssueCommentsRequest): unknown {
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

  create(base?: DeepPartial<ListIssueCommentsRequest>): ListIssueCommentsRequest {
    return ListIssueCommentsRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListIssueCommentsRequest>): ListIssueCommentsRequest {
    const message = createBaseListIssueCommentsRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListIssueCommentsResponse(): ListIssueCommentsResponse {
  return { issueComments: [], nextPageToken: "" };
}

export const ListIssueCommentsResponse = {
  encode(message: ListIssueCommentsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.issueComments) {
      IssueComment.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListIssueCommentsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListIssueCommentsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.issueComments.push(IssueComment.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListIssueCommentsResponse {
    return {
      issueComments: globalThis.Array.isArray(object?.issueComments)
        ? object.issueComments.map((e: any) => IssueComment.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListIssueCommentsResponse): unknown {
    const obj: any = {};
    if (message.issueComments?.length) {
      obj.issueComments = message.issueComments.map((e) => IssueComment.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListIssueCommentsResponse>): ListIssueCommentsResponse {
    return ListIssueCommentsResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListIssueCommentsResponse>): ListIssueCommentsResponse {
    const message = createBaseListIssueCommentsResponse();
    message.issueComments = object.issueComments?.map((e) => IssueComment.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseCreateIssueCommentRequest(): CreateIssueCommentRequest {
  return { parent: "", issueComment: undefined };
}

export const CreateIssueCommentRequest = {
  encode(message: CreateIssueCommentRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.issueComment !== undefined) {
      IssueComment.encode(message.issueComment, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateIssueCommentRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateIssueCommentRequest();
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

          message.issueComment = IssueComment.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateIssueCommentRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      issueComment: isSet(object.issueComment) ? IssueComment.fromJSON(object.issueComment) : undefined,
    };
  },

  toJSON(message: CreateIssueCommentRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.issueComment !== undefined) {
      obj.issueComment = IssueComment.toJSON(message.issueComment);
    }
    return obj;
  },

  create(base?: DeepPartial<CreateIssueCommentRequest>): CreateIssueCommentRequest {
    return CreateIssueCommentRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<CreateIssueCommentRequest>): CreateIssueCommentRequest {
    const message = createBaseCreateIssueCommentRequest();
    message.parent = object.parent ?? "";
    message.issueComment = (object.issueComment !== undefined && object.issueComment !== null)
      ? IssueComment.fromPartial(object.issueComment)
      : undefined;
    return message;
  },
};

function createBaseUpdateIssueCommentRequest(): UpdateIssueCommentRequest {
  return { parent: "", issueComment: undefined, updateMask: undefined };
}

export const UpdateIssueCommentRequest = {
  encode(message: UpdateIssueCommentRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.issueComment !== undefined) {
      IssueComment.encode(message.issueComment, writer.uint32(18).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateIssueCommentRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateIssueCommentRequest();
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

          message.issueComment = IssueComment.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
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

  fromJSON(object: any): UpdateIssueCommentRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      issueComment: isSet(object.issueComment) ? IssueComment.fromJSON(object.issueComment) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateIssueCommentRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.issueComment !== undefined) {
      obj.issueComment = IssueComment.toJSON(message.issueComment);
    }
    if (message.updateMask !== undefined) {
      obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask));
    }
    return obj;
  },

  create(base?: DeepPartial<UpdateIssueCommentRequest>): UpdateIssueCommentRequest {
    return UpdateIssueCommentRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UpdateIssueCommentRequest>): UpdateIssueCommentRequest {
    const message = createBaseUpdateIssueCommentRequest();
    message.parent = object.parent ?? "";
    message.issueComment = (object.issueComment !== undefined && object.issueComment !== null)
      ? IssueComment.fromPartial(object.issueComment)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseIssueComment(): IssueComment {
  return {
    uid: "",
    comment: "",
    payload: "",
    createTime: undefined,
    updateTime: undefined,
    name: "",
    creator: "",
    approval: undefined,
    issueUpdate: undefined,
    stageEnd: undefined,
    taskUpdate: undefined,
    taskPriorBackup: undefined,
  };
}

export const IssueComment = {
  encode(message: IssueComment, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.uid !== "") {
      writer.uint32(10).string(message.uid);
    }
    if (message.comment !== "") {
      writer.uint32(18).string(message.comment);
    }
    if (message.payload !== "") {
      writer.uint32(26).string(message.payload);
    }
    if (message.createTime !== undefined) {
      Timestamp.encode(toTimestamp(message.createTime), writer.uint32(34).fork()).ldelim();
    }
    if (message.updateTime !== undefined) {
      Timestamp.encode(toTimestamp(message.updateTime), writer.uint32(42).fork()).ldelim();
    }
    if (message.name !== "") {
      writer.uint32(50).string(message.name);
    }
    if (message.creator !== "") {
      writer.uint32(58).string(message.creator);
    }
    if (message.approval !== undefined) {
      IssueComment_Approval.encode(message.approval, writer.uint32(66).fork()).ldelim();
    }
    if (message.issueUpdate !== undefined) {
      IssueComment_IssueUpdate.encode(message.issueUpdate, writer.uint32(74).fork()).ldelim();
    }
    if (message.stageEnd !== undefined) {
      IssueComment_StageEnd.encode(message.stageEnd, writer.uint32(82).fork()).ldelim();
    }
    if (message.taskUpdate !== undefined) {
      IssueComment_TaskUpdate.encode(message.taskUpdate, writer.uint32(90).fork()).ldelim();
    }
    if (message.taskPriorBackup !== undefined) {
      IssueComment_TaskPriorBackup.encode(message.taskPriorBackup, writer.uint32(98).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IssueComment {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIssueComment();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.uid = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.comment = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.payload = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.createTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.updateTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.name = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.creator = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.approval = IssueComment_Approval.decode(reader, reader.uint32());
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.issueUpdate = IssueComment_IssueUpdate.decode(reader, reader.uint32());
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.stageEnd = IssueComment_StageEnd.decode(reader, reader.uint32());
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.taskUpdate = IssueComment_TaskUpdate.decode(reader, reader.uint32());
          continue;
        case 12:
          if (tag !== 98) {
            break;
          }

          message.taskPriorBackup = IssueComment_TaskPriorBackup.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): IssueComment {
    return {
      uid: isSet(object.uid) ? globalThis.String(object.uid) : "",
      comment: isSet(object.comment) ? globalThis.String(object.comment) : "",
      payload: isSet(object.payload) ? globalThis.String(object.payload) : "",
      createTime: isSet(object.createTime) ? fromJsonTimestamp(object.createTime) : undefined,
      updateTime: isSet(object.updateTime) ? fromJsonTimestamp(object.updateTime) : undefined,
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      creator: isSet(object.creator) ? globalThis.String(object.creator) : "",
      approval: isSet(object.approval) ? IssueComment_Approval.fromJSON(object.approval) : undefined,
      issueUpdate: isSet(object.issueUpdate) ? IssueComment_IssueUpdate.fromJSON(object.issueUpdate) : undefined,
      stageEnd: isSet(object.stageEnd) ? IssueComment_StageEnd.fromJSON(object.stageEnd) : undefined,
      taskUpdate: isSet(object.taskUpdate) ? IssueComment_TaskUpdate.fromJSON(object.taskUpdate) : undefined,
      taskPriorBackup: isSet(object.taskPriorBackup)
        ? IssueComment_TaskPriorBackup.fromJSON(object.taskPriorBackup)
        : undefined,
    };
  },

  toJSON(message: IssueComment): unknown {
    const obj: any = {};
    if (message.uid !== "") {
      obj.uid = message.uid;
    }
    if (message.comment !== "") {
      obj.comment = message.comment;
    }
    if (message.payload !== "") {
      obj.payload = message.payload;
    }
    if (message.createTime !== undefined) {
      obj.createTime = message.createTime.toISOString();
    }
    if (message.updateTime !== undefined) {
      obj.updateTime = message.updateTime.toISOString();
    }
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.creator !== "") {
      obj.creator = message.creator;
    }
    if (message.approval !== undefined) {
      obj.approval = IssueComment_Approval.toJSON(message.approval);
    }
    if (message.issueUpdate !== undefined) {
      obj.issueUpdate = IssueComment_IssueUpdate.toJSON(message.issueUpdate);
    }
    if (message.stageEnd !== undefined) {
      obj.stageEnd = IssueComment_StageEnd.toJSON(message.stageEnd);
    }
    if (message.taskUpdate !== undefined) {
      obj.taskUpdate = IssueComment_TaskUpdate.toJSON(message.taskUpdate);
    }
    if (message.taskPriorBackup !== undefined) {
      obj.taskPriorBackup = IssueComment_TaskPriorBackup.toJSON(message.taskPriorBackup);
    }
    return obj;
  },

  create(base?: DeepPartial<IssueComment>): IssueComment {
    return IssueComment.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<IssueComment>): IssueComment {
    const message = createBaseIssueComment();
    message.uid = object.uid ?? "";
    message.comment = object.comment ?? "";
    message.payload = object.payload ?? "";
    message.createTime = object.createTime ?? undefined;
    message.updateTime = object.updateTime ?? undefined;
    message.name = object.name ?? "";
    message.creator = object.creator ?? "";
    message.approval = (object.approval !== undefined && object.approval !== null)
      ? IssueComment_Approval.fromPartial(object.approval)
      : undefined;
    message.issueUpdate = (object.issueUpdate !== undefined && object.issueUpdate !== null)
      ? IssueComment_IssueUpdate.fromPartial(object.issueUpdate)
      : undefined;
    message.stageEnd = (object.stageEnd !== undefined && object.stageEnd !== null)
      ? IssueComment_StageEnd.fromPartial(object.stageEnd)
      : undefined;
    message.taskUpdate = (object.taskUpdate !== undefined && object.taskUpdate !== null)
      ? IssueComment_TaskUpdate.fromPartial(object.taskUpdate)
      : undefined;
    message.taskPriorBackup = (object.taskPriorBackup !== undefined && object.taskPriorBackup !== null)
      ? IssueComment_TaskPriorBackup.fromPartial(object.taskPriorBackup)
      : undefined;
    return message;
  },
};

function createBaseIssueComment_Approval(): IssueComment_Approval {
  return { status: IssueComment_Approval_Status.STATUS_UNSPECIFIED };
}

export const IssueComment_Approval = {
  encode(message: IssueComment_Approval, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.status !== IssueComment_Approval_Status.STATUS_UNSPECIFIED) {
      writer.uint32(8).int32(issueComment_Approval_StatusToNumber(message.status));
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IssueComment_Approval {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIssueComment_Approval();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.status = issueComment_Approval_StatusFromJSON(reader.int32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): IssueComment_Approval {
    return {
      status: isSet(object.status)
        ? issueComment_Approval_StatusFromJSON(object.status)
        : IssueComment_Approval_Status.STATUS_UNSPECIFIED,
    };
  },

  toJSON(message: IssueComment_Approval): unknown {
    const obj: any = {};
    if (message.status !== IssueComment_Approval_Status.STATUS_UNSPECIFIED) {
      obj.status = issueComment_Approval_StatusToJSON(message.status);
    }
    return obj;
  },

  create(base?: DeepPartial<IssueComment_Approval>): IssueComment_Approval {
    return IssueComment_Approval.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<IssueComment_Approval>): IssueComment_Approval {
    const message = createBaseIssueComment_Approval();
    message.status = object.status ?? IssueComment_Approval_Status.STATUS_UNSPECIFIED;
    return message;
  },
};

function createBaseIssueComment_IssueUpdate(): IssueComment_IssueUpdate {
  return {
    fromTitle: undefined,
    toTitle: undefined,
    fromDescription: undefined,
    toDescription: undefined,
    fromStatus: undefined,
    toStatus: undefined,
    fromLabels: [],
    toLabels: [],
  };
}

export const IssueComment_IssueUpdate = {
  encode(message: IssueComment_IssueUpdate, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.fromTitle !== undefined) {
      writer.uint32(10).string(message.fromTitle);
    }
    if (message.toTitle !== undefined) {
      writer.uint32(18).string(message.toTitle);
    }
    if (message.fromDescription !== undefined) {
      writer.uint32(26).string(message.fromDescription);
    }
    if (message.toDescription !== undefined) {
      writer.uint32(34).string(message.toDescription);
    }
    if (message.fromStatus !== undefined) {
      writer.uint32(40).int32(issueStatusToNumber(message.fromStatus));
    }
    if (message.toStatus !== undefined) {
      writer.uint32(48).int32(issueStatusToNumber(message.toStatus));
    }
    for (const v of message.fromLabels) {
      writer.uint32(74).string(v!);
    }
    for (const v of message.toLabels) {
      writer.uint32(82).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IssueComment_IssueUpdate {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIssueComment_IssueUpdate();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.fromTitle = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.toTitle = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.fromDescription = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.toDescription = reader.string();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.fromStatus = issueStatusFromJSON(reader.int32());
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.toStatus = issueStatusFromJSON(reader.int32());
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.fromLabels.push(reader.string());
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.toLabels.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): IssueComment_IssueUpdate {
    return {
      fromTitle: isSet(object.fromTitle) ? globalThis.String(object.fromTitle) : undefined,
      toTitle: isSet(object.toTitle) ? globalThis.String(object.toTitle) : undefined,
      fromDescription: isSet(object.fromDescription) ? globalThis.String(object.fromDescription) : undefined,
      toDescription: isSet(object.toDescription) ? globalThis.String(object.toDescription) : undefined,
      fromStatus: isSet(object.fromStatus) ? issueStatusFromJSON(object.fromStatus) : undefined,
      toStatus: isSet(object.toStatus) ? issueStatusFromJSON(object.toStatus) : undefined,
      fromLabels: globalThis.Array.isArray(object?.fromLabels)
        ? object.fromLabels.map((e: any) => globalThis.String(e))
        : [],
      toLabels: globalThis.Array.isArray(object?.toLabels) ? object.toLabels.map((e: any) => globalThis.String(e)) : [],
    };
  },

  toJSON(message: IssueComment_IssueUpdate): unknown {
    const obj: any = {};
    if (message.fromTitle !== undefined) {
      obj.fromTitle = message.fromTitle;
    }
    if (message.toTitle !== undefined) {
      obj.toTitle = message.toTitle;
    }
    if (message.fromDescription !== undefined) {
      obj.fromDescription = message.fromDescription;
    }
    if (message.toDescription !== undefined) {
      obj.toDescription = message.toDescription;
    }
    if (message.fromStatus !== undefined) {
      obj.fromStatus = issueStatusToJSON(message.fromStatus);
    }
    if (message.toStatus !== undefined) {
      obj.toStatus = issueStatusToJSON(message.toStatus);
    }
    if (message.fromLabels?.length) {
      obj.fromLabels = message.fromLabels;
    }
    if (message.toLabels?.length) {
      obj.toLabels = message.toLabels;
    }
    return obj;
  },

  create(base?: DeepPartial<IssueComment_IssueUpdate>): IssueComment_IssueUpdate {
    return IssueComment_IssueUpdate.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<IssueComment_IssueUpdate>): IssueComment_IssueUpdate {
    const message = createBaseIssueComment_IssueUpdate();
    message.fromTitle = object.fromTitle ?? undefined;
    message.toTitle = object.toTitle ?? undefined;
    message.fromDescription = object.fromDescription ?? undefined;
    message.toDescription = object.toDescription ?? undefined;
    message.fromStatus = object.fromStatus ?? undefined;
    message.toStatus = object.toStatus ?? undefined;
    message.fromLabels = object.fromLabels?.map((e) => e) || [];
    message.toLabels = object.toLabels?.map((e) => e) || [];
    return message;
  },
};

function createBaseIssueComment_StageEnd(): IssueComment_StageEnd {
  return { stage: "" };
}

export const IssueComment_StageEnd = {
  encode(message: IssueComment_StageEnd, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.stage !== "") {
      writer.uint32(10).string(message.stage);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IssueComment_StageEnd {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIssueComment_StageEnd();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.stage = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): IssueComment_StageEnd {
    return { stage: isSet(object.stage) ? globalThis.String(object.stage) : "" };
  },

  toJSON(message: IssueComment_StageEnd): unknown {
    const obj: any = {};
    if (message.stage !== "") {
      obj.stage = message.stage;
    }
    return obj;
  },

  create(base?: DeepPartial<IssueComment_StageEnd>): IssueComment_StageEnd {
    return IssueComment_StageEnd.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<IssueComment_StageEnd>): IssueComment_StageEnd {
    const message = createBaseIssueComment_StageEnd();
    message.stage = object.stage ?? "";
    return message;
  },
};

function createBaseIssueComment_TaskUpdate(): IssueComment_TaskUpdate {
  return {
    tasks: [],
    fromSheet: undefined,
    toSheet: undefined,
    fromEarliestAllowedTime: undefined,
    toEarliestAllowedTime: undefined,
    toStatus: undefined,
  };
}

export const IssueComment_TaskUpdate = {
  encode(message: IssueComment_TaskUpdate, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.tasks) {
      writer.uint32(10).string(v!);
    }
    if (message.fromSheet !== undefined) {
      writer.uint32(18).string(message.fromSheet);
    }
    if (message.toSheet !== undefined) {
      writer.uint32(26).string(message.toSheet);
    }
    if (message.fromEarliestAllowedTime !== undefined) {
      Timestamp.encode(toTimestamp(message.fromEarliestAllowedTime), writer.uint32(34).fork()).ldelim();
    }
    if (message.toEarliestAllowedTime !== undefined) {
      Timestamp.encode(toTimestamp(message.toEarliestAllowedTime), writer.uint32(42).fork()).ldelim();
    }
    if (message.toStatus !== undefined) {
      writer.uint32(48).int32(issueComment_TaskUpdate_StatusToNumber(message.toStatus));
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IssueComment_TaskUpdate {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIssueComment_TaskUpdate();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.tasks.push(reader.string());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.fromSheet = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.toSheet = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.fromEarliestAllowedTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.toEarliestAllowedTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.toStatus = issueComment_TaskUpdate_StatusFromJSON(reader.int32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): IssueComment_TaskUpdate {
    return {
      tasks: globalThis.Array.isArray(object?.tasks) ? object.tasks.map((e: any) => globalThis.String(e)) : [],
      fromSheet: isSet(object.fromSheet) ? globalThis.String(object.fromSheet) : undefined,
      toSheet: isSet(object.toSheet) ? globalThis.String(object.toSheet) : undefined,
      fromEarliestAllowedTime: isSet(object.fromEarliestAllowedTime)
        ? fromJsonTimestamp(object.fromEarliestAllowedTime)
        : undefined,
      toEarliestAllowedTime: isSet(object.toEarliestAllowedTime)
        ? fromJsonTimestamp(object.toEarliestAllowedTime)
        : undefined,
      toStatus: isSet(object.toStatus) ? issueComment_TaskUpdate_StatusFromJSON(object.toStatus) : undefined,
    };
  },

  toJSON(message: IssueComment_TaskUpdate): unknown {
    const obj: any = {};
    if (message.tasks?.length) {
      obj.tasks = message.tasks;
    }
    if (message.fromSheet !== undefined) {
      obj.fromSheet = message.fromSheet;
    }
    if (message.toSheet !== undefined) {
      obj.toSheet = message.toSheet;
    }
    if (message.fromEarliestAllowedTime !== undefined) {
      obj.fromEarliestAllowedTime = message.fromEarliestAllowedTime.toISOString();
    }
    if (message.toEarliestAllowedTime !== undefined) {
      obj.toEarliestAllowedTime = message.toEarliestAllowedTime.toISOString();
    }
    if (message.toStatus !== undefined) {
      obj.toStatus = issueComment_TaskUpdate_StatusToJSON(message.toStatus);
    }
    return obj;
  },

  create(base?: DeepPartial<IssueComment_TaskUpdate>): IssueComment_TaskUpdate {
    return IssueComment_TaskUpdate.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<IssueComment_TaskUpdate>): IssueComment_TaskUpdate {
    const message = createBaseIssueComment_TaskUpdate();
    message.tasks = object.tasks?.map((e) => e) || [];
    message.fromSheet = object.fromSheet ?? undefined;
    message.toSheet = object.toSheet ?? undefined;
    message.fromEarliestAllowedTime = object.fromEarliestAllowedTime ?? undefined;
    message.toEarliestAllowedTime = object.toEarliestAllowedTime ?? undefined;
    message.toStatus = object.toStatus ?? undefined;
    return message;
  },
};

function createBaseIssueComment_TaskPriorBackup(): IssueComment_TaskPriorBackup {
  return { task: "", tables: [], originalLine: undefined, database: "" };
}

export const IssueComment_TaskPriorBackup = {
  encode(message: IssueComment_TaskPriorBackup, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.task !== "") {
      writer.uint32(10).string(message.task);
    }
    for (const v of message.tables) {
      IssueComment_TaskPriorBackup_Table.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.originalLine !== undefined) {
      writer.uint32(24).int32(message.originalLine);
    }
    if (message.database !== "") {
      writer.uint32(34).string(message.database);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IssueComment_TaskPriorBackup {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIssueComment_TaskPriorBackup();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.task = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.tables.push(IssueComment_TaskPriorBackup_Table.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.originalLine = reader.int32();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.database = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): IssueComment_TaskPriorBackup {
    return {
      task: isSet(object.task) ? globalThis.String(object.task) : "",
      tables: globalThis.Array.isArray(object?.tables)
        ? object.tables.map((e: any) => IssueComment_TaskPriorBackup_Table.fromJSON(e))
        : [],
      originalLine: isSet(object.originalLine) ? globalThis.Number(object.originalLine) : undefined,
      database: isSet(object.database) ? globalThis.String(object.database) : "",
    };
  },

  toJSON(message: IssueComment_TaskPriorBackup): unknown {
    const obj: any = {};
    if (message.task !== "") {
      obj.task = message.task;
    }
    if (message.tables?.length) {
      obj.tables = message.tables.map((e) => IssueComment_TaskPriorBackup_Table.toJSON(e));
    }
    if (message.originalLine !== undefined) {
      obj.originalLine = Math.round(message.originalLine);
    }
    if (message.database !== "") {
      obj.database = message.database;
    }
    return obj;
  },

  create(base?: DeepPartial<IssueComment_TaskPriorBackup>): IssueComment_TaskPriorBackup {
    return IssueComment_TaskPriorBackup.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<IssueComment_TaskPriorBackup>): IssueComment_TaskPriorBackup {
    const message = createBaseIssueComment_TaskPriorBackup();
    message.task = object.task ?? "";
    message.tables = object.tables?.map((e) => IssueComment_TaskPriorBackup_Table.fromPartial(e)) || [];
    message.originalLine = object.originalLine ?? undefined;
    message.database = object.database ?? "";
    return message;
  },
};

function createBaseIssueComment_TaskPriorBackup_Table(): IssueComment_TaskPriorBackup_Table {
  return { schema: "", table: "" };
}

export const IssueComment_TaskPriorBackup_Table = {
  encode(message: IssueComment_TaskPriorBackup_Table, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schema !== "") {
      writer.uint32(10).string(message.schema);
    }
    if (message.table !== "") {
      writer.uint32(18).string(message.table);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IssueComment_TaskPriorBackup_Table {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIssueComment_TaskPriorBackup_Table();
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
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): IssueComment_TaskPriorBackup_Table {
    return {
      schema: isSet(object.schema) ? globalThis.String(object.schema) : "",
      table: isSet(object.table) ? globalThis.String(object.table) : "",
    };
  },

  toJSON(message: IssueComment_TaskPriorBackup_Table): unknown {
    const obj: any = {};
    if (message.schema !== "") {
      obj.schema = message.schema;
    }
    if (message.table !== "") {
      obj.table = message.table;
    }
    return obj;
  },

  create(base?: DeepPartial<IssueComment_TaskPriorBackup_Table>): IssueComment_TaskPriorBackup_Table {
    return IssueComment_TaskPriorBackup_Table.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<IssueComment_TaskPriorBackup_Table>): IssueComment_TaskPriorBackup_Table {
    const message = createBaseIssueComment_TaskPriorBackup_Table();
    message.schema = object.schema ?? "";
    message.table = object.table ?? "";
    return message;
  },
};

export type IssueServiceDefinition = typeof IssueServiceDefinition;
export const IssueServiceDefinition = {
  name: "IssueService",
  fullName: "bytebase.v1.IssueService",
  methods: {
    getIssue: {
      name: "GetIssue",
      requestType: GetIssueRequest,
      requestStream: false,
      responseType: Issue,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          800010: [new Uint8Array([13, 98, 98, 46, 105, 115, 115, 117, 101, 115, 46, 103, 101, 116])],
          800016: [new Uint8Array([2])],
          578365826: [
            new Uint8Array([
              32,
              18,
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
              47,
              105,
              115,
              115,
              117,
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
    createIssue: {
      name: "CreateIssue",
      requestType: CreateIssueRequest,
      requestStream: false,
      responseType: Issue,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([12, 112, 97, 114, 101, 110, 116, 44, 105, 115, 115, 117, 101])],
          800010: [new Uint8Array([16, 98, 98, 46, 105, 115, 115, 117, 101, 115, 46, 99, 114, 101, 97, 116, 101])],
          800016: [new Uint8Array([2])],
          578365826: [
            new Uint8Array([
              39,
              58,
              5,
              105,
              115,
              115,
              117,
              101,
              34,
              30,
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
              105,
              115,
              115,
              117,
              101,
              115,
            ]),
          ],
        },
      },
    },
    listIssues: {
      name: "ListIssues",
      requestType: ListIssuesRequest,
      requestStream: false,
      responseType: ListIssuesResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([6, 112, 97, 114, 101, 110, 116])],
          800010: [new Uint8Array([14, 98, 98, 46, 105, 115, 115, 117, 101, 115, 46, 108, 105, 115, 116])],
          800016: [new Uint8Array([2])],
          578365826: [
            new Uint8Array([
              32,
              18,
              30,
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
              105,
              115,
              115,
              117,
              101,
              115,
            ]),
          ],
        },
      },
    },
    /** Search for issues that the caller has the bb.issues.get permission on and also satisfy the specified filter & query. */
    searchIssues: {
      name: "SearchIssues",
      requestType: SearchIssuesRequest,
      requestStream: false,
      responseType: SearchIssuesResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          800010: [new Uint8Array([13, 98, 98, 46, 105, 115, 115, 117, 101, 115, 46, 103, 101, 116])],
          800016: [new Uint8Array([2])],
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
              105,
              115,
              115,
              117,
              101,
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
    updateIssue: {
      name: "UpdateIssue",
      requestType: UpdateIssueRequest,
      requestStream: false,
      responseType: Issue,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([17, 105, 115, 115, 117, 101, 44, 117, 112, 100, 97, 116, 101, 95, 109, 97, 115, 107])],
          800010: [new Uint8Array([16, 98, 98, 46, 105, 115, 115, 117, 101, 115, 46, 117, 112, 100, 97, 116, 101])],
          800016: [new Uint8Array([2])],
          578365826: [
            new Uint8Array([
              45,
              58,
              5,
              105,
              115,
              115,
              117,
              101,
              50,
              36,
              47,
              118,
              49,
              47,
              123,
              105,
              115,
              115,
              117,
              101,
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
              105,
              115,
              115,
              117,
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
    listIssueComments: {
      name: "ListIssueComments",
      requestType: ListIssueCommentsRequest,
      requestStream: false,
      responseType: ListIssueCommentsResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([6, 112, 97, 114, 101, 110, 116])],
          800010: [
            new Uint8Array([
              21,
              98,
              98,
              46,
              105,
              115,
              115,
              117,
              101,
              67,
              111,
              109,
              109,
              101,
              110,
              116,
              115,
              46,
              108,
              105,
              115,
              116,
            ]),
          ],
          800016: [new Uint8Array([2])],
          578365826: [
            new Uint8Array([
              48,
              18,
              46,
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
              105,
              115,
              115,
              117,
              101,
              115,
              47,
              42,
              125,
              47,
              105,
              115,
              115,
              117,
              101,
              67,
              111,
              109,
              109,
              101,
              110,
              116,
              115,
            ]),
          ],
        },
      },
    },
    createIssueComment: {
      name: "CreateIssueComment",
      requestType: CreateIssueCommentRequest,
      requestStream: false,
      responseType: IssueComment,
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
              105,
              115,
              115,
              117,
              101,
              95,
              99,
              111,
              109,
              109,
              101,
              110,
              116,
            ]),
          ],
          800010: [
            new Uint8Array([
              23,
              98,
              98,
              46,
              105,
              115,
              115,
              117,
              101,
              67,
              111,
              109,
              109,
              101,
              110,
              116,
              115,
              46,
              99,
              114,
              101,
              97,
              116,
              101,
            ]),
          ],
          800016: [new Uint8Array([2])],
          578365826: [
            new Uint8Array([
              57,
              58,
              13,
              105,
              115,
              115,
              117,
              101,
              95,
              99,
              111,
              109,
              109,
              101,
              110,
              116,
              34,
              40,
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
              105,
              115,
              115,
              117,
              101,
              115,
              47,
              42,
              125,
              58,
              99,
              111,
              109,
              109,
              101,
              110,
              116,
            ]),
          ],
        },
      },
    },
    updateIssueComment: {
      name: "UpdateIssueComment",
      requestType: UpdateIssueCommentRequest,
      requestStream: false,
      responseType: IssueComment,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([
              32,
              112,
              97,
              114,
              101,
              110,
              116,
              44,
              105,
              115,
              115,
              117,
              101,
              95,
              99,
              111,
              109,
              109,
              101,
              110,
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
            new Uint8Array([
              23,
              98,
              98,
              46,
              105,
              115,
              115,
              117,
              101,
              67,
              111,
              109,
              109,
              101,
              110,
              116,
              115,
              46,
              117,
              112,
              100,
              97,
              116,
              101,
            ]),
          ],
          800016: [new Uint8Array([2])],
          578365826: [
            new Uint8Array([
              57,
              58,
              13,
              105,
              115,
              115,
              117,
              101,
              95,
              99,
              111,
              109,
              109,
              101,
              110,
              116,
              50,
              40,
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
              105,
              115,
              115,
              117,
              101,
              115,
              47,
              42,
              125,
              58,
              99,
              111,
              109,
              109,
              101,
              110,
              116,
            ]),
          ],
        },
      },
    },
    batchUpdateIssuesStatus: {
      name: "BatchUpdateIssuesStatus",
      requestType: BatchUpdateIssuesStatusRequest,
      requestStream: false,
      responseType: BatchUpdateIssuesStatusResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          800010: [new Uint8Array([16, 98, 98, 46, 105, 115, 115, 117, 101, 115, 46, 117, 112, 100, 97, 116, 101])],
          800016: [new Uint8Array([2])],
          578365826: [
            new Uint8Array([
              53,
              58,
              1,
              42,
              34,
              48,
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
              105,
              115,
              115,
              117,
              101,
              115,
              58,
              98,
              97,
              116,
              99,
              104,
              85,
              112,
              100,
              97,
              116,
              101,
              83,
              116,
              97,
              116,
              117,
              115,
            ]),
          ],
        },
      },
    },
    approveIssue: {
      name: "ApproveIssue",
      requestType: ApproveIssueRequest,
      requestStream: false,
      responseType: Issue,
      responseStream: false,
      options: {
        _unknownFields: {
          800016: [new Uint8Array([2])],
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
              105,
              115,
              115,
              117,
              101,
              115,
              47,
              42,
              125,
              58,
              97,
              112,
              112,
              114,
              111,
              118,
              101,
            ]),
          ],
        },
      },
    },
    rejectIssue: {
      name: "RejectIssue",
      requestType: RejectIssueRequest,
      requestStream: false,
      responseType: Issue,
      responseStream: false,
      options: {
        _unknownFields: {
          800016: [new Uint8Array([2])],
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
              105,
              115,
              115,
              117,
              101,
              115,
              47,
              42,
              125,
              58,
              114,
              101,
              106,
              101,
              99,
              116,
            ]),
          ],
        },
      },
    },
    requestIssue: {
      name: "RequestIssue",
      requestType: RequestIssueRequest,
      requestStream: false,
      responseType: Issue,
      responseStream: false,
      options: {
        _unknownFields: {
          800016: [new Uint8Array([2])],
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
              105,
              115,
              115,
              117,
              101,
              115,
              47,
              42,
              125,
              58,
              114,
              101,
              113,
              117,
              101,
              115,
              116,
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

function toTimestamp(date: Date): Timestamp {
  const seconds = numberToLong(date.getTime() / 1_000);
  const nanos = (date.getTime() % 1_000) * 1_000_000;
  return { seconds, nanos };
}

function fromTimestamp(t: Timestamp): Date {
  let millis = (t.seconds.toNumber() || 0) * 1_000;
  millis += (t.nanos || 0) / 1_000_000;
  return new globalThis.Date(millis);
}

function fromJsonTimestamp(o: any): Date {
  if (o instanceof globalThis.Date) {
    return o;
  } else if (typeof o === "string") {
    return new globalThis.Date(o);
  } else {
    return fromTimestamp(Timestamp.fromJSON(o));
  }
}

function numberToLong(number: number) {
  return Long.fromNumber(number);
}

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isObject(value: any): boolean {
  return typeof value === "object" && value !== null;
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
