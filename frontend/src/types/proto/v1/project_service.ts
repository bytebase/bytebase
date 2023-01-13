/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { Timestamp } from "../google/protobuf/timestamp";
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

export enum RoleProvider {
  ROLE_PROVIDER_UNSPECIFIED = 0,
  BYTEBASE = 1,
  GITLAB_SELF_HOST = 2,
  GITHUB_COM = 3,
  UNRECOGNIZED = -1,
}

export function roleProviderFromJSON(object: any): RoleProvider {
  switch (object) {
    case 0:
    case "ROLE_PROVIDER_UNSPECIFIED":
      return RoleProvider.ROLE_PROVIDER_UNSPECIFIED;
    case 1:
    case "BYTEBASE":
      return RoleProvider.BYTEBASE;
    case 2:
    case "GITLAB_SELF_HOST":
      return RoleProvider.GITLAB_SELF_HOST;
    case 3:
    case "GITHUB_COM":
      return RoleProvider.GITHUB_COM;
    case -1:
    case "UNRECOGNIZED":
    default:
      return RoleProvider.UNRECOGNIZED;
  }
}

export function roleProviderToJSON(object: RoleProvider): string {
  switch (object) {
    case RoleProvider.ROLE_PROVIDER_UNSPECIFIED:
      return "ROLE_PROVIDER_UNSPECIFIED";
    case RoleProvider.BYTEBASE:
      return "BYTEBASE";
    case RoleProvider.GITLAB_SELF_HOST:
      return "GITLAB_SELF_HOST";
    case RoleProvider.GITHUB_COM:
      return "GITHUB_COM";
    case RoleProvider.UNRECOGNIZED:
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

export interface SyncExternalIamPolicyRequest {
  /**
   * The name of the project to set the IAM policy.
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
  roleProvider: RoleProvider;
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

function createBaseSyncExternalIamPolicyRequest(): SyncExternalIamPolicyRequest {
  return { project: "" };
}

export const SyncExternalIamPolicyRequest = {
  encode(message: SyncExternalIamPolicyRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.project !== "") {
      writer.uint32(10).string(message.project);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SyncExternalIamPolicyRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSyncExternalIamPolicyRequest();
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

  fromJSON(object: any): SyncExternalIamPolicyRequest {
    return { project: isSet(object.project) ? String(object.project) : "" };
  },

  toJSON(message: SyncExternalIamPolicyRequest): unknown {
    const obj: any = {};
    message.project !== undefined && (obj.project = message.project);
    return obj;
  },

  fromPartial(object: DeepPartial<SyncExternalIamPolicyRequest>): SyncExternalIamPolicyRequest {
    const message = createBaseSyncExternalIamPolicyRequest();
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
    roleProvider: 0,
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
    if (message.roleProvider !== 0) {
      writer.uint32(80).int32(message.roleProvider);
    }
    if (message.schemaVersion !== 0) {
      writer.uint32(88).int32(message.schemaVersion);
    }
    if (message.schemaChange !== 0) {
      writer.uint32(96).int32(message.schemaChange);
    }
    if (message.lgtmCheck !== 0) {
      writer.uint32(104).int32(message.lgtmCheck);
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
          message.roleProvider = reader.int32() as any;
          break;
        case 11:
          message.schemaVersion = reader.int32() as any;
          break;
        case 12:
          message.schemaChange = reader.int32() as any;
          break;
        case 13:
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
      roleProvider: isSet(object.roleProvider) ? roleProviderFromJSON(object.roleProvider) : 0,
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
    message.roleProvider !== undefined && (obj.roleProvider = roleProviderToJSON(message.roleProvider));
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
    message.roleProvider = object.roleProvider ?? 0;
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
    syncExternalIamPolicy: {
      name: "SyncExternalIamPolicy",
      requestType: SyncExternalIamPolicyRequest,
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
  syncExternalIamPolicy(
    request: SyncExternalIamPolicyRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<IamPolicy>>;
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
  syncExternalIamPolicy(
    request: DeepPartial<SyncExternalIamPolicyRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<IamPolicy>;
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
