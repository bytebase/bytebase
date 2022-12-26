/* eslint-disable */
import _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";

export const protobufPackage = "bytebase.v1";

export enum Workflow {
  WORKFLOW_UNSPECIFIED = 0,
  WORKFLOW_UI = 1,
  WORKFLOW_VCS = 2,
  UNRECOGNIZED = -1,
}

export function workflowFromJSON(object: any): Workflow {
  switch (object) {
    case 0:
    case "WORKFLOW_UNSPECIFIED":
      return Workflow.WORKFLOW_UNSPECIFIED;
    case 1:
    case "WORKFLOW_UI":
      return Workflow.WORKFLOW_UI;
    case 2:
    case "WORKFLOW_VCS":
      return Workflow.WORKFLOW_VCS;
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
    case Workflow.WORKFLOW_UI:
      return "WORKFLOW_UI";
    case Workflow.WORKFLOW_VCS:
      return "WORKFLOW_VCS";
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
  ROLE_PROVIDER_BYTEBASE = 1,
  ROLE_PROVIDER_GITLAB_SELF_HOST = 2,
  ROLE_PROVIDER_GITHUB_COM = 3,
  UNRECOGNIZED = -1,
}

export function roleProviderFromJSON(object: any): RoleProvider {
  switch (object) {
    case 0:
    case "ROLE_PROVIDER_UNSPECIFIED":
      return RoleProvider.ROLE_PROVIDER_UNSPECIFIED;
    case 1:
    case "ROLE_PROVIDER_BYTEBASE":
      return RoleProvider.ROLE_PROVIDER_BYTEBASE;
    case 2:
    case "ROLE_PROVIDER_GITLAB_SELF_HOST":
      return RoleProvider.ROLE_PROVIDER_GITLAB_SELF_HOST;
    case 3:
    case "ROLE_PROVIDER_GITHUB_COM":
      return RoleProvider.ROLE_PROVIDER_GITHUB_COM;
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
    case RoleProvider.ROLE_PROVIDER_BYTEBASE:
      return "ROLE_PROVIDER_BYTEBASE";
    case RoleProvider.ROLE_PROVIDER_GITLAB_SELF_HOST:
      return "ROLE_PROVIDER_GITLAB_SELF_HOST";
    case RoleProvider.ROLE_PROVIDER_GITHUB_COM:
      return "ROLE_PROVIDER_GITHUB_COM";
    case RoleProvider.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum SchemaVersion {
  SCHEMA_VERSION_UNSPECIFIED = 0,
  SCHEMA_VERSION_TIMESTAMP = 1,
  SCHEMA_VERSION_SEMANTIC = 2,
  UNRECOGNIZED = -1,
}

export function schemaVersionFromJSON(object: any): SchemaVersion {
  switch (object) {
    case 0:
    case "SCHEMA_VERSION_UNSPECIFIED":
      return SchemaVersion.SCHEMA_VERSION_UNSPECIFIED;
    case 1:
    case "SCHEMA_VERSION_TIMESTAMP":
      return SchemaVersion.SCHEMA_VERSION_TIMESTAMP;
    case 2:
    case "SCHEMA_VERSION_SEMANTIC":
      return SchemaVersion.SCHEMA_VERSION_SEMANTIC;
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
    case SchemaVersion.SCHEMA_VERSION_TIMESTAMP:
      return "SCHEMA_VERSION_TIMESTAMP";
    case SchemaVersion.SCHEMA_VERSION_SEMANTIC:
      return "SCHEMA_VERSION_SEMANTIC";
    case SchemaVersion.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum SchemaChange {
  SCHEMA_CHANGE_UNSPECIFIED = 0,
  SCHEMA_CHANGE_DDL = 1,
  SCHEMA_CHANGE_SDL = 2,
  UNRECOGNIZED = -1,
}

export function schemaChangeFromJSON(object: any): SchemaChange {
  switch (object) {
    case 0:
    case "SCHEMA_CHANGE_UNSPECIFIED":
      return SchemaChange.SCHEMA_CHANGE_UNSPECIFIED;
    case 1:
    case "SCHEMA_CHANGE_DDL":
      return SchemaChange.SCHEMA_CHANGE_DDL;
    case 2:
    case "SCHEMA_CHANGE_SDL":
      return SchemaChange.SCHEMA_CHANGE_SDL;
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
    case SchemaChange.SCHEMA_CHANGE_DDL:
      return "SCHEMA_CHANGE_DDL";
    case SchemaChange.SCHEMA_CHANGE_SDL:
      return "SCHEMA_CHANGE_SDL";
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
  /** The projects from the specified publisher. */
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

export interface Project {
  /**
   * The name of the project.
   * Format: projects/{project}
   */
  name: string;
  title: string;
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

  fromPartial<I extends Exact<DeepPartial<GetProjectRequest>, I>>(object: I): GetProjectRequest {
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

  fromPartial<I extends Exact<DeepPartial<ListProjectsRequest>, I>>(object: I): ListProjectsRequest {
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

  fromPartial<I extends Exact<DeepPartial<ListProjectsResponse>, I>>(object: I): ListProjectsResponse {
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

  fromPartial<I extends Exact<DeepPartial<CreateProjectRequest>, I>>(object: I): CreateProjectRequest {
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

  fromPartial<I extends Exact<DeepPartial<UpdateProjectRequest>, I>>(object: I): UpdateProjectRequest {
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

  fromPartial<I extends Exact<DeepPartial<DeleteProjectRequest>, I>>(object: I): DeleteProjectRequest {
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

  fromPartial<I extends Exact<DeepPartial<UndeleteProjectRequest>, I>>(object: I): UndeleteProjectRequest {
    const message = createBaseUndeleteProjectRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseProject(): Project {
  return {
    name: "",
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
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.key !== "") {
      writer.uint32(26).string(message.key);
    }
    if (message.workflow !== 0) {
      writer.uint32(32).int32(message.workflow);
    }
    if (message.visibility !== 0) {
      writer.uint32(40).int32(message.visibility);
    }
    if (message.tenantMode !== 0) {
      writer.uint32(48).int32(message.tenantMode);
    }
    if (message.dbNameTemplate !== "") {
      writer.uint32(58).string(message.dbNameTemplate);
    }
    if (message.roleProvider !== 0) {
      writer.uint32(64).int32(message.roleProvider);
    }
    if (message.schemaVersion !== 0) {
      writer.uint32(72).int32(message.schemaVersion);
    }
    if (message.schemaChange !== 0) {
      writer.uint32(80).int32(message.schemaChange);
    }
    if (message.lgtmCheck !== 0) {
      writer.uint32(88).int32(message.lgtmCheck);
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
          message.title = reader.string();
          break;
        case 3:
          message.key = reader.string();
          break;
        case 4:
          message.workflow = reader.int32() as any;
          break;
        case 5:
          message.visibility = reader.int32() as any;
          break;
        case 6:
          message.tenantMode = reader.int32() as any;
          break;
        case 7:
          message.dbNameTemplate = reader.string();
          break;
        case 8:
          message.roleProvider = reader.int32() as any;
          break;
        case 9:
          message.schemaVersion = reader.int32() as any;
          break;
        case 10:
          message.schemaChange = reader.int32() as any;
          break;
        case 11:
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

  fromPartial<I extends Exact<DeepPartial<Project>, I>>(object: I): Project {
    const message = createBaseProject();
    message.name = object.name ?? "";
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

export interface ProjectService {
  GetProject(request: GetProjectRequest): Promise<Project>;
  ListProjects(request: ListProjectsRequest): Promise<ListProjectsResponse>;
  CreateProject(request: CreateProjectRequest): Promise<Project>;
  UpdateProject(request: UpdateProjectRequest): Promise<Project>;
  DeleteProject(request: DeleteProjectRequest): Promise<Empty>;
  UndeleteProject(request: UndeleteProjectRequest): Promise<Project>;
}

export class ProjectServiceClientImpl implements ProjectService {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "bytebase.v1.ProjectService";
    this.rpc = rpc;
    this.GetProject = this.GetProject.bind(this);
    this.ListProjects = this.ListProjects.bind(this);
    this.CreateProject = this.CreateProject.bind(this);
    this.UpdateProject = this.UpdateProject.bind(this);
    this.DeleteProject = this.DeleteProject.bind(this);
    this.UndeleteProject = this.UndeleteProject.bind(this);
  }
  GetProject(request: GetProjectRequest): Promise<Project> {
    const data = GetProjectRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "GetProject", data);
    return promise.then((data) => Project.decode(new _m0.Reader(data)));
  }

  ListProjects(request: ListProjectsRequest): Promise<ListProjectsResponse> {
    const data = ListProjectsRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "ListProjects", data);
    return promise.then((data) => ListProjectsResponse.decode(new _m0.Reader(data)));
  }

  CreateProject(request: CreateProjectRequest): Promise<Project> {
    const data = CreateProjectRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "CreateProject", data);
    return promise.then((data) => Project.decode(new _m0.Reader(data)));
  }

  UpdateProject(request: UpdateProjectRequest): Promise<Project> {
    const data = UpdateProjectRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "UpdateProject", data);
    return promise.then((data) => Project.decode(new _m0.Reader(data)));
  }

  DeleteProject(request: DeleteProjectRequest): Promise<Empty> {
    const data = DeleteProjectRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "DeleteProject", data);
    return promise.then((data) => Empty.decode(new _m0.Reader(data)));
  }

  UndeleteProject(request: UndeleteProjectRequest): Promise<Project> {
    const data = UndeleteProjectRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "UndeleteProject", data);
    return promise.then((data) => Project.decode(new _m0.Reader(data)));
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
