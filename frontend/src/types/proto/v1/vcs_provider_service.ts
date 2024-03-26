/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";

export const protobufPackage = "bytebase.v1";

export interface CreateVCSProviderRequest {
  vcsProvider:
    | VCSProvider
    | undefined;
  /**
   * The ID to use for the VCS provider, which will become the final component of
   * the VCS provider's name.
   *
   * This value should be 4-63 characters, and valid characters
   * are /[a-z][0-9]-/.
   */
  vcsProviderId: string;
}

export interface GetVCSProviderRequest {
  /**
   * The name of the vcs provider to retrieve.
   * Format: vcsProviders/{vcsProvider}
   */
  name: string;
}

export interface ListVCSProvidersRequest {
  /**
   * Not used. The maximum number of vcs provider to return. The service may return fewer than this value.
   * If unspecified, at most 100 vcs provider will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * Not used. A page token, received from a previous `ListVCSProviders` call.
   * Provide this to retrieve the subsequent page.
   */
  pageToken: string;
}

export interface ListVCSProvidersResponse {
  /** The list of vcs providers. */
  vcsProviders: VCSProvider[];
  /**
   * Not used. A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface UpdateVCSProviderRequest {
  vcsProvider:
    | VCSProvider
    | undefined;
  /** The list of fields to be updated. */
  updateMask: string[] | undefined;
}

export interface DeleteVCSProviderRequest {
  /**
   * The name of the vcs provider to delete.
   * Format: vcsProviders/{vcsProvider}
   */
  name: string;
}

export interface SearchVCSProviderProjectsRequest {
  /**
   * The name of the vcs provider to retrieve the vcs provider repositories.
   * Format: vcsProviders/{vcsProvider}
   */
  name: string;
}

export interface SearchVCSProviderProjectsResponse {
  /** The list of project in vcs provider. */
  projects: SearchVCSProviderProjectsResponse_Project[];
}

export interface SearchVCSProviderProjectsResponse_Project {
  /** The id of the project in vcs provider. */
  id: string;
  /** The title of the project in vcs provider. */
  title: string;
  /** The fullpath of the project in vcs provider. */
  fullpath: string;
  /** Web url of the project in vcs provider. */
  webUrl: string;
}

export interface ListProjectGitOpsInfoRequest {
  /**
   * The name of the vcs provider to retrieve the linked projects.
   * Format: vcsProviders/{vcsProvider}
   */
  name: string;
}

export interface ListProjectGitOpsInfoResponse {
  /** The binding between vcs provider and project. */
  projectGitopsInfo: ProjectGitOpsInfo[];
}

export interface VCSProvider {
  /**
   * The name of the vcs provider.
   * Format: vcsProviders/{vcsProvider}
   */
  name: string;
  /** The title of the vcs provider. It is used to display in the UI. Specified by the client. */
  title: string;
  type: VCSProvider_Type;
  /**
   * The url of the vcs provider. Specified by the client.
   * For example: github.com, gitlab.com, gitlab.bytebase.com.
   */
  url: string;
  /** The access token of the vcs provider. */
  accessToken: string;
}

export enum VCSProvider_Type {
  TYPE_UNSPECIFIED = 0,
  /** GITHUB - GitHub type. Using for GitHub community edition(ce). */
  GITHUB = 1,
  /** GITLAB - GitLab type. Using for GitLab community edition(ce) and enterprise edition(ee). */
  GITLAB = 2,
  /** BITBUCKET - BitBucket type. Using for BitBucket cloud or BitBucket server. */
  BITBUCKET = 3,
  /** AZURE_DEVOPS - Azure DevOps. Using for Azure DevOps GitOps workflow. */
  AZURE_DEVOPS = 4,
  UNRECOGNIZED = -1,
}

export function vCSProvider_TypeFromJSON(object: any): VCSProvider_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return VCSProvider_Type.TYPE_UNSPECIFIED;
    case 1:
    case "GITHUB":
      return VCSProvider_Type.GITHUB;
    case 2:
    case "GITLAB":
      return VCSProvider_Type.GITLAB;
    case 3:
    case "BITBUCKET":
      return VCSProvider_Type.BITBUCKET;
    case 4:
    case "AZURE_DEVOPS":
      return VCSProvider_Type.AZURE_DEVOPS;
    case -1:
    case "UNRECOGNIZED":
    default:
      return VCSProvider_Type.UNRECOGNIZED;
  }
}

export function vCSProvider_TypeToJSON(object: VCSProvider_Type): string {
  switch (object) {
    case VCSProvider_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case VCSProvider_Type.GITHUB:
      return "GITHUB";
    case VCSProvider_Type.GITLAB:
      return "GITLAB";
    case VCSProvider_Type.BITBUCKET:
      return "BITBUCKET";
    case VCSProvider_Type.AZURE_DEVOPS:
      return "AZURE_DEVOPS";
    case VCSProvider_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface ProjectGitOpsInfo {
  /**
   * The name of the linked repository, generated by the server, and also is the unique identifier.
   * Format: projects/{project}/gitOpsInfo
   */
  name: string;
  /** The name of the VCS. */
  vcs: string;
  /** The title of the repository. For axample: sample. */
  title: string;
  /** The full_path of the repository. For example: bytebase/sample. */
  fullPath: string;
  /** The web url of the repository. For axample: https://gitlab.bytebase.com/bytebase/sample. */
  webUrl: string;
  /** The branch Bytebase listens to for changes. For example: main. */
  branchFilter: string;
  /** The root directory where Bytebase observes the file change. If empty, then it observes the entire repository. */
  baseDirectory: string;
  /** The reposition external id in target VCS. */
  externalId: string;
  /** The webhook endpoint ID of the repository. */
  webhookEndpointId: string;
}

function createBaseCreateVCSProviderRequest(): CreateVCSProviderRequest {
  return { vcsProvider: undefined, vcsProviderId: "" };
}

export const CreateVCSProviderRequest = {
  encode(message: CreateVCSProviderRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.vcsProvider !== undefined) {
      VCSProvider.encode(message.vcsProvider, writer.uint32(10).fork()).ldelim();
    }
    if (message.vcsProviderId !== "") {
      writer.uint32(18).string(message.vcsProviderId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateVCSProviderRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateVCSProviderRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.vcsProvider = VCSProvider.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.vcsProviderId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateVCSProviderRequest {
    return {
      vcsProvider: isSet(object.vcsProvider) ? VCSProvider.fromJSON(object.vcsProvider) : undefined,
      vcsProviderId: isSet(object.vcsProviderId) ? globalThis.String(object.vcsProviderId) : "",
    };
  },

  toJSON(message: CreateVCSProviderRequest): unknown {
    const obj: any = {};
    if (message.vcsProvider !== undefined) {
      obj.vcsProvider = VCSProvider.toJSON(message.vcsProvider);
    }
    if (message.vcsProviderId !== "") {
      obj.vcsProviderId = message.vcsProviderId;
    }
    return obj;
  },

  create(base?: DeepPartial<CreateVCSProviderRequest>): CreateVCSProviderRequest {
    return CreateVCSProviderRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<CreateVCSProviderRequest>): CreateVCSProviderRequest {
    const message = createBaseCreateVCSProviderRequest();
    message.vcsProvider = (object.vcsProvider !== undefined && object.vcsProvider !== null)
      ? VCSProvider.fromPartial(object.vcsProvider)
      : undefined;
    message.vcsProviderId = object.vcsProviderId ?? "";
    return message;
  },
};

function createBaseGetVCSProviderRequest(): GetVCSProviderRequest {
  return { name: "" };
}

export const GetVCSProviderRequest = {
  encode(message: GetVCSProviderRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetVCSProviderRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetVCSProviderRequest();
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

  fromJSON(object: any): GetVCSProviderRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: GetVCSProviderRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<GetVCSProviderRequest>): GetVCSProviderRequest {
    return GetVCSProviderRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<GetVCSProviderRequest>): GetVCSProviderRequest {
    const message = createBaseGetVCSProviderRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListVCSProvidersRequest(): ListVCSProvidersRequest {
  return { pageSize: 0, pageToken: "" };
}

export const ListVCSProvidersRequest = {
  encode(message: ListVCSProvidersRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pageSize !== 0) {
      writer.uint32(8).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(18).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListVCSProvidersRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListVCSProvidersRequest();
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
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListVCSProvidersRequest {
    return {
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
    };
  },

  toJSON(message: ListVCSProvidersRequest): unknown {
    const obj: any = {};
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListVCSProvidersRequest>): ListVCSProvidersRequest {
    return ListVCSProvidersRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListVCSProvidersRequest>): ListVCSProvidersRequest {
    const message = createBaseListVCSProvidersRequest();
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListVCSProvidersResponse(): ListVCSProvidersResponse {
  return { vcsProviders: [], nextPageToken: "" };
}

export const ListVCSProvidersResponse = {
  encode(message: ListVCSProvidersResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.vcsProviders) {
      VCSProvider.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListVCSProvidersResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListVCSProvidersResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.vcsProviders.push(VCSProvider.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListVCSProvidersResponse {
    return {
      vcsProviders: globalThis.Array.isArray(object?.vcsProviders)
        ? object.vcsProviders.map((e: any) => VCSProvider.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListVCSProvidersResponse): unknown {
    const obj: any = {};
    if (message.vcsProviders?.length) {
      obj.vcsProviders = message.vcsProviders.map((e) => VCSProvider.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListVCSProvidersResponse>): ListVCSProvidersResponse {
    return ListVCSProvidersResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListVCSProvidersResponse>): ListVCSProvidersResponse {
    const message = createBaseListVCSProvidersResponse();
    message.vcsProviders = object.vcsProviders?.map((e) => VCSProvider.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseUpdateVCSProviderRequest(): UpdateVCSProviderRequest {
  return { vcsProvider: undefined, updateMask: undefined };
}

export const UpdateVCSProviderRequest = {
  encode(message: UpdateVCSProviderRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.vcsProvider !== undefined) {
      VCSProvider.encode(message.vcsProvider, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateVCSProviderRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateVCSProviderRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.vcsProvider = VCSProvider.decode(reader, reader.uint32());
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

  fromJSON(object: any): UpdateVCSProviderRequest {
    return {
      vcsProvider: isSet(object.vcsProvider) ? VCSProvider.fromJSON(object.vcsProvider) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateVCSProviderRequest): unknown {
    const obj: any = {};
    if (message.vcsProvider !== undefined) {
      obj.vcsProvider = VCSProvider.toJSON(message.vcsProvider);
    }
    if (message.updateMask !== undefined) {
      obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask));
    }
    return obj;
  },

  create(base?: DeepPartial<UpdateVCSProviderRequest>): UpdateVCSProviderRequest {
    return UpdateVCSProviderRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UpdateVCSProviderRequest>): UpdateVCSProviderRequest {
    const message = createBaseUpdateVCSProviderRequest();
    message.vcsProvider = (object.vcsProvider !== undefined && object.vcsProvider !== null)
      ? VCSProvider.fromPartial(object.vcsProvider)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseDeleteVCSProviderRequest(): DeleteVCSProviderRequest {
  return { name: "" };
}

export const DeleteVCSProviderRequest = {
  encode(message: DeleteVCSProviderRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteVCSProviderRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteVCSProviderRequest();
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

  fromJSON(object: any): DeleteVCSProviderRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: DeleteVCSProviderRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<DeleteVCSProviderRequest>): DeleteVCSProviderRequest {
    return DeleteVCSProviderRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DeleteVCSProviderRequest>): DeleteVCSProviderRequest {
    const message = createBaseDeleteVCSProviderRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseSearchVCSProviderProjectsRequest(): SearchVCSProviderProjectsRequest {
  return { name: "" };
}

export const SearchVCSProviderProjectsRequest = {
  encode(message: SearchVCSProviderProjectsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SearchVCSProviderProjectsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSearchVCSProviderProjectsRequest();
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

  fromJSON(object: any): SearchVCSProviderProjectsRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: SearchVCSProviderProjectsRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<SearchVCSProviderProjectsRequest>): SearchVCSProviderProjectsRequest {
    return SearchVCSProviderProjectsRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SearchVCSProviderProjectsRequest>): SearchVCSProviderProjectsRequest {
    const message = createBaseSearchVCSProviderProjectsRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseSearchVCSProviderProjectsResponse(): SearchVCSProviderProjectsResponse {
  return { projects: [] };
}

export const SearchVCSProviderProjectsResponse = {
  encode(message: SearchVCSProviderProjectsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.projects) {
      SearchVCSProviderProjectsResponse_Project.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SearchVCSProviderProjectsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSearchVCSProviderProjectsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.projects.push(SearchVCSProviderProjectsResponse_Project.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SearchVCSProviderProjectsResponse {
    return {
      projects: globalThis.Array.isArray(object?.projects)
        ? object.projects.map((e: any) => SearchVCSProviderProjectsResponse_Project.fromJSON(e))
        : [],
    };
  },

  toJSON(message: SearchVCSProviderProjectsResponse): unknown {
    const obj: any = {};
    if (message.projects?.length) {
      obj.projects = message.projects.map((e) => SearchVCSProviderProjectsResponse_Project.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<SearchVCSProviderProjectsResponse>): SearchVCSProviderProjectsResponse {
    return SearchVCSProviderProjectsResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SearchVCSProviderProjectsResponse>): SearchVCSProviderProjectsResponse {
    const message = createBaseSearchVCSProviderProjectsResponse();
    message.projects = object.projects?.map((e) => SearchVCSProviderProjectsResponse_Project.fromPartial(e)) || [];
    return message;
  },
};

function createBaseSearchVCSProviderProjectsResponse_Project(): SearchVCSProviderProjectsResponse_Project {
  return { id: "", title: "", fullpath: "", webUrl: "" };
}

export const SearchVCSProviderProjectsResponse_Project = {
  encode(message: SearchVCSProviderProjectsResponse_Project, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.fullpath !== "") {
      writer.uint32(26).string(message.fullpath);
    }
    if (message.webUrl !== "") {
      writer.uint32(34).string(message.webUrl);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SearchVCSProviderProjectsResponse_Project {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSearchVCSProviderProjectsResponse_Project();
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

          message.title = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.fullpath = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.webUrl = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SearchVCSProviderProjectsResponse_Project {
    return {
      id: isSet(object.id) ? globalThis.String(object.id) : "",
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      fullpath: isSet(object.fullpath) ? globalThis.String(object.fullpath) : "",
      webUrl: isSet(object.webUrl) ? globalThis.String(object.webUrl) : "",
    };
  },

  toJSON(message: SearchVCSProviderProjectsResponse_Project): unknown {
    const obj: any = {};
    if (message.id !== "") {
      obj.id = message.id;
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.fullpath !== "") {
      obj.fullpath = message.fullpath;
    }
    if (message.webUrl !== "") {
      obj.webUrl = message.webUrl;
    }
    return obj;
  },

  create(base?: DeepPartial<SearchVCSProviderProjectsResponse_Project>): SearchVCSProviderProjectsResponse_Project {
    return SearchVCSProviderProjectsResponse_Project.fromPartial(base ?? {});
  },
  fromPartial(
    object: DeepPartial<SearchVCSProviderProjectsResponse_Project>,
  ): SearchVCSProviderProjectsResponse_Project {
    const message = createBaseSearchVCSProviderProjectsResponse_Project();
    message.id = object.id ?? "";
    message.title = object.title ?? "";
    message.fullpath = object.fullpath ?? "";
    message.webUrl = object.webUrl ?? "";
    return message;
  },
};

function createBaseListProjectGitOpsInfoRequest(): ListProjectGitOpsInfoRequest {
  return { name: "" };
}

export const ListProjectGitOpsInfoRequest = {
  encode(message: ListProjectGitOpsInfoRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListProjectGitOpsInfoRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListProjectGitOpsInfoRequest();
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

  fromJSON(object: any): ListProjectGitOpsInfoRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: ListProjectGitOpsInfoRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<ListProjectGitOpsInfoRequest>): ListProjectGitOpsInfoRequest {
    return ListProjectGitOpsInfoRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListProjectGitOpsInfoRequest>): ListProjectGitOpsInfoRequest {
    const message = createBaseListProjectGitOpsInfoRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListProjectGitOpsInfoResponse(): ListProjectGitOpsInfoResponse {
  return { projectGitopsInfo: [] };
}

export const ListProjectGitOpsInfoResponse = {
  encode(message: ListProjectGitOpsInfoResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.projectGitopsInfo) {
      ProjectGitOpsInfo.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListProjectGitOpsInfoResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListProjectGitOpsInfoResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.projectGitopsInfo.push(ProjectGitOpsInfo.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListProjectGitOpsInfoResponse {
    return {
      projectGitopsInfo: globalThis.Array.isArray(object?.projectGitopsInfo)
        ? object.projectGitopsInfo.map((e: any) => ProjectGitOpsInfo.fromJSON(e))
        : [],
    };
  },

  toJSON(message: ListProjectGitOpsInfoResponse): unknown {
    const obj: any = {};
    if (message.projectGitopsInfo?.length) {
      obj.projectGitopsInfo = message.projectGitopsInfo.map((e) => ProjectGitOpsInfo.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<ListProjectGitOpsInfoResponse>): ListProjectGitOpsInfoResponse {
    return ListProjectGitOpsInfoResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListProjectGitOpsInfoResponse>): ListProjectGitOpsInfoResponse {
    const message = createBaseListProjectGitOpsInfoResponse();
    message.projectGitopsInfo = object.projectGitopsInfo?.map((e) => ProjectGitOpsInfo.fromPartial(e)) || [];
    return message;
  },
};

function createBaseVCSProvider(): VCSProvider {
  return { name: "", title: "", type: 0, url: "", accessToken: "" };
}

export const VCSProvider = {
  encode(message: VCSProvider, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.type !== 0) {
      writer.uint32(24).int32(message.type);
    }
    if (message.url !== "") {
      writer.uint32(34).string(message.url);
    }
    if (message.accessToken !== "") {
      writer.uint32(42).string(message.accessToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): VCSProvider {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseVCSProvider();
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
          if (tag !== 24) {
            break;
          }

          message.type = reader.int32() as any;
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.url = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.accessToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): VCSProvider {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      type: isSet(object.type) ? vCSProvider_TypeFromJSON(object.type) : 0,
      url: isSet(object.url) ? globalThis.String(object.url) : "",
      accessToken: isSet(object.accessToken) ? globalThis.String(object.accessToken) : "",
    };
  },

  toJSON(message: VCSProvider): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.type !== 0) {
      obj.type = vCSProvider_TypeToJSON(message.type);
    }
    if (message.url !== "") {
      obj.url = message.url;
    }
    if (message.accessToken !== "") {
      obj.accessToken = message.accessToken;
    }
    return obj;
  },

  create(base?: DeepPartial<VCSProvider>): VCSProvider {
    return VCSProvider.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<VCSProvider>): VCSProvider {
    const message = createBaseVCSProvider();
    message.name = object.name ?? "";
    message.title = object.title ?? "";
    message.type = object.type ?? 0;
    message.url = object.url ?? "";
    message.accessToken = object.accessToken ?? "";
    return message;
  },
};

function createBaseProjectGitOpsInfo(): ProjectGitOpsInfo {
  return {
    name: "",
    vcs: "",
    title: "",
    fullPath: "",
    webUrl: "",
    branchFilter: "",
    baseDirectory: "",
    externalId: "",
    webhookEndpointId: "",
  };
}

export const ProjectGitOpsInfo = {
  encode(message: ProjectGitOpsInfo, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.vcs !== "") {
      writer.uint32(18).string(message.vcs);
    }
    if (message.title !== "") {
      writer.uint32(26).string(message.title);
    }
    if (message.fullPath !== "") {
      writer.uint32(34).string(message.fullPath);
    }
    if (message.webUrl !== "") {
      writer.uint32(42).string(message.webUrl);
    }
    if (message.branchFilter !== "") {
      writer.uint32(50).string(message.branchFilter);
    }
    if (message.baseDirectory !== "") {
      writer.uint32(58).string(message.baseDirectory);
    }
    if (message.externalId !== "") {
      writer.uint32(90).string(message.externalId);
    }
    if (message.webhookEndpointId !== "") {
      writer.uint32(106).string(message.webhookEndpointId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ProjectGitOpsInfo {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseProjectGitOpsInfo();
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

          message.vcs = reader.string();
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

          message.fullPath = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.webUrl = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.branchFilter = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.baseDirectory = reader.string();
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.externalId = reader.string();
          continue;
        case 13:
          if (tag !== 106) {
            break;
          }

          message.webhookEndpointId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ProjectGitOpsInfo {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      vcs: isSet(object.vcs) ? globalThis.String(object.vcs) : "",
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      fullPath: isSet(object.fullPath) ? globalThis.String(object.fullPath) : "",
      webUrl: isSet(object.webUrl) ? globalThis.String(object.webUrl) : "",
      branchFilter: isSet(object.branchFilter) ? globalThis.String(object.branchFilter) : "",
      baseDirectory: isSet(object.baseDirectory) ? globalThis.String(object.baseDirectory) : "",
      externalId: isSet(object.externalId) ? globalThis.String(object.externalId) : "",
      webhookEndpointId: isSet(object.webhookEndpointId) ? globalThis.String(object.webhookEndpointId) : "",
    };
  },

  toJSON(message: ProjectGitOpsInfo): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.vcs !== "") {
      obj.vcs = message.vcs;
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.fullPath !== "") {
      obj.fullPath = message.fullPath;
    }
    if (message.webUrl !== "") {
      obj.webUrl = message.webUrl;
    }
    if (message.branchFilter !== "") {
      obj.branchFilter = message.branchFilter;
    }
    if (message.baseDirectory !== "") {
      obj.baseDirectory = message.baseDirectory;
    }
    if (message.externalId !== "") {
      obj.externalId = message.externalId;
    }
    if (message.webhookEndpointId !== "") {
      obj.webhookEndpointId = message.webhookEndpointId;
    }
    return obj;
  },

  create(base?: DeepPartial<ProjectGitOpsInfo>): ProjectGitOpsInfo {
    return ProjectGitOpsInfo.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ProjectGitOpsInfo>): ProjectGitOpsInfo {
    const message = createBaseProjectGitOpsInfo();
    message.name = object.name ?? "";
    message.vcs = object.vcs ?? "";
    message.title = object.title ?? "";
    message.fullPath = object.fullPath ?? "";
    message.webUrl = object.webUrl ?? "";
    message.branchFilter = object.branchFilter ?? "";
    message.baseDirectory = object.baseDirectory ?? "";
    message.externalId = object.externalId ?? "";
    message.webhookEndpointId = object.webhookEndpointId ?? "";
    return message;
  },
};

export type VCSProviderServiceDefinition = typeof VCSProviderServiceDefinition;
export const VCSProviderServiceDefinition = {
  name: "VCSProviderService",
  fullName: "bytebase.v1.VCSProviderService",
  methods: {
    getVCSProvider: {
      name: "GetVCSProvider",
      requestType: GetVCSProviderRequest,
      requestStream: false,
      responseType: VCSProvider,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              27,
              18,
              25,
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
              118,
              99,
              115,
              80,
              114,
              111,
              118,
              105,
              100,
              101,
              114,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    listVCSProviders: {
      name: "ListVCSProviders",
      requestType: ListVCSProvidersRequest,
      requestStream: false,
      responseType: ListVCSProvidersResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          578365826: [
            new Uint8Array([18, 18, 16, 47, 118, 49, 47, 118, 99, 115, 80, 114, 111, 118, 105, 100, 101, 114, 115]),
          ],
        },
      },
    },
    createVCSProvider: {
      name: "CreateVCSProvider",
      requestType: CreateVCSProviderRequest,
      requestStream: false,
      responseType: VCSProvider,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          578365826: [
            new Uint8Array([
              32,
              58,
              12,
              118,
              99,
              115,
              95,
              112,
              114,
              111,
              118,
              105,
              100,
              101,
              114,
              34,
              16,
              47,
              118,
              49,
              47,
              118,
              99,
              115,
              80,
              114,
              111,
              118,
              105,
              100,
              101,
              114,
              115,
            ]),
          ],
        },
      },
    },
    updateVCSProvider: {
      name: "UpdateVCSProvider",
      requestType: UpdateVCSProviderRequest,
      requestStream: false,
      responseType: VCSProvider,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([
              24,
              118,
              99,
              115,
              95,
              112,
              114,
              111,
              118,
              105,
              100,
              101,
              114,
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
              54,
              58,
              12,
              118,
              99,
              115,
              95,
              112,
              114,
              111,
              118,
              105,
              100,
              101,
              114,
              50,
              38,
              47,
              118,
              49,
              47,
              123,
              118,
              99,
              115,
              95,
              112,
              114,
              111,
              118,
              105,
              100,
              101,
              114,
              46,
              110,
              97,
              109,
              101,
              61,
              118,
              99,
              115,
              80,
              114,
              111,
              118,
              105,
              100,
              101,
              114,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    deleteVCSProvider: {
      name: "DeleteVCSProvider",
      requestType: DeleteVCSProviderRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              27,
              42,
              25,
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
              118,
              99,
              115,
              80,
              114,
              111,
              118,
              105,
              100,
              101,
              114,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    searchVCSProviderProjects: {
      name: "SearchVCSProviderProjects",
      requestType: SearchVCSProviderProjectsRequest,
      requestStream: false,
      responseType: SearchVCSProviderProjectsResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              45,
              58,
              1,
              42,
              34,
              40,
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
              118,
              99,
              115,
              80,
              114,
              111,
              118,
              105,
              100,
              101,
              114,
              115,
              47,
              42,
              125,
              58,
              115,
              101,
              97,
              114,
              99,
              104,
              80,
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
    listProjectGitOpsInfo: {
      name: "ListProjectGitOpsInfo",
      requestType: ListProjectGitOpsInfoRequest,
      requestStream: false,
      responseType: ListProjectGitOpsInfoResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              36,
              18,
              34,
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
              118,
              99,
              115,
              80,
              114,
              111,
              118,
              105,
              100,
              101,
              114,
              115,
              47,
              42,
              125,
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
