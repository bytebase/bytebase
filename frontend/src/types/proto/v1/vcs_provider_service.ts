/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { VCSType, vCSTypeFromJSON, vCSTypeToJSON } from "./common";
import { VCSConnector } from "./vcs_connector_service";

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

export interface SearchVCSProviderRepositoriesRequest {
  /**
   * The name of the vcs provider to retrieve the vcs provider repositories.
   * Format: vcsProviders/{vcsProvider}
   */
  name: string;
}

export interface VCSRepository {
  /**
   * The id of the repository in vcs provider.
   * e.g. In GitLab, this is the corresponding project id. e.g. 123
   */
  id: string;
  /**
   * The title of the repository in vcs provider.
   * e.g. sample-project
   */
  title: string;
  /**
   * The full_path of the repository in vcs provider.
   * e.g. bytebase/sample-project
   */
  fullPath: string;
  /**
   * Web url of the repository in vcs provider.
   * e.g. http://gitlab.bytebase.com/bytebase/sample-project
   */
  webUrl: string;
}

export interface SearchVCSProviderRepositoriesResponse {
  /** The list of repositories in vcs provider. */
  repositories: VCSRepository[];
}

export interface ListVCSConnectorsInProviderRequest {
  /**
   * The name of the vcs provider to retrieve the linked projects.
   * Format: vcsProviders/{vcsProvider}
   */
  name: string;
}

export interface ListVCSConnectorsInProviderResponse {
  /** The vcsConnectors from the specified request. */
  vcsConnectors: VCSConnector[];
}

export interface VCSProvider {
  /**
   * The name of the vcs provider.
   * Format: vcsProviders/{vcsProvider}
   */
  name: string;
  /** The title of the vcs provider. It is used to display in the UI. Specified by the client. */
  title: string;
  type: VCSType;
  /**
   * The url of the vcs provider. Specified by the client.
   * For example: github.com, gitlab.com, gitlab.bytebase.com.
   */
  url: string;
  /** The access token of the vcs provider. */
  accessToken: string;
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

function createBaseSearchVCSProviderRepositoriesRequest(): SearchVCSProviderRepositoriesRequest {
  return { name: "" };
}

export const SearchVCSProviderRepositoriesRequest = {
  encode(message: SearchVCSProviderRepositoriesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SearchVCSProviderRepositoriesRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSearchVCSProviderRepositoriesRequest();
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

  fromJSON(object: any): SearchVCSProviderRepositoriesRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: SearchVCSProviderRepositoriesRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<SearchVCSProviderRepositoriesRequest>): SearchVCSProviderRepositoriesRequest {
    return SearchVCSProviderRepositoriesRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SearchVCSProviderRepositoriesRequest>): SearchVCSProviderRepositoriesRequest {
    const message = createBaseSearchVCSProviderRepositoriesRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseVCSRepository(): VCSRepository {
  return { id: "", title: "", fullPath: "", webUrl: "" };
}

export const VCSRepository = {
  encode(message: VCSRepository, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.fullPath !== "") {
      writer.uint32(26).string(message.fullPath);
    }
    if (message.webUrl !== "") {
      writer.uint32(34).string(message.webUrl);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): VCSRepository {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseVCSRepository();
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

          message.fullPath = reader.string();
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

  fromJSON(object: any): VCSRepository {
    return {
      id: isSet(object.id) ? globalThis.String(object.id) : "",
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      fullPath: isSet(object.fullPath) ? globalThis.String(object.fullPath) : "",
      webUrl: isSet(object.webUrl) ? globalThis.String(object.webUrl) : "",
    };
  },

  toJSON(message: VCSRepository): unknown {
    const obj: any = {};
    if (message.id !== "") {
      obj.id = message.id;
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
    return obj;
  },

  create(base?: DeepPartial<VCSRepository>): VCSRepository {
    return VCSRepository.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<VCSRepository>): VCSRepository {
    const message = createBaseVCSRepository();
    message.id = object.id ?? "";
    message.title = object.title ?? "";
    message.fullPath = object.fullPath ?? "";
    message.webUrl = object.webUrl ?? "";
    return message;
  },
};

function createBaseSearchVCSProviderRepositoriesResponse(): SearchVCSProviderRepositoriesResponse {
  return { repositories: [] };
}

export const SearchVCSProviderRepositoriesResponse = {
  encode(message: SearchVCSProviderRepositoriesResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.repositories) {
      VCSRepository.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SearchVCSProviderRepositoriesResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSearchVCSProviderRepositoriesResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.repositories.push(VCSRepository.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SearchVCSProviderRepositoriesResponse {
    return {
      repositories: globalThis.Array.isArray(object?.repositories)
        ? object.repositories.map((e: any) => VCSRepository.fromJSON(e))
        : [],
    };
  },

  toJSON(message: SearchVCSProviderRepositoriesResponse): unknown {
    const obj: any = {};
    if (message.repositories?.length) {
      obj.repositories = message.repositories.map((e) => VCSRepository.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<SearchVCSProviderRepositoriesResponse>): SearchVCSProviderRepositoriesResponse {
    return SearchVCSProviderRepositoriesResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SearchVCSProviderRepositoriesResponse>): SearchVCSProviderRepositoriesResponse {
    const message = createBaseSearchVCSProviderRepositoriesResponse();
    message.repositories = object.repositories?.map((e) => VCSRepository.fromPartial(e)) || [];
    return message;
  },
};

function createBaseListVCSConnectorsInProviderRequest(): ListVCSConnectorsInProviderRequest {
  return { name: "" };
}

export const ListVCSConnectorsInProviderRequest = {
  encode(message: ListVCSConnectorsInProviderRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListVCSConnectorsInProviderRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListVCSConnectorsInProviderRequest();
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

  fromJSON(object: any): ListVCSConnectorsInProviderRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: ListVCSConnectorsInProviderRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<ListVCSConnectorsInProviderRequest>): ListVCSConnectorsInProviderRequest {
    return ListVCSConnectorsInProviderRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListVCSConnectorsInProviderRequest>): ListVCSConnectorsInProviderRequest {
    const message = createBaseListVCSConnectorsInProviderRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListVCSConnectorsInProviderResponse(): ListVCSConnectorsInProviderResponse {
  return { vcsConnectors: [] };
}

export const ListVCSConnectorsInProviderResponse = {
  encode(message: ListVCSConnectorsInProviderResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.vcsConnectors) {
      VCSConnector.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListVCSConnectorsInProviderResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListVCSConnectorsInProviderResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.vcsConnectors.push(VCSConnector.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListVCSConnectorsInProviderResponse {
    return {
      vcsConnectors: globalThis.Array.isArray(object?.vcsConnectors)
        ? object.vcsConnectors.map((e: any) => VCSConnector.fromJSON(e))
        : [],
    };
  },

  toJSON(message: ListVCSConnectorsInProviderResponse): unknown {
    const obj: any = {};
    if (message.vcsConnectors?.length) {
      obj.vcsConnectors = message.vcsConnectors.map((e) => VCSConnector.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<ListVCSConnectorsInProviderResponse>): ListVCSConnectorsInProviderResponse {
    return ListVCSConnectorsInProviderResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListVCSConnectorsInProviderResponse>): ListVCSConnectorsInProviderResponse {
    const message = createBaseListVCSConnectorsInProviderResponse();
    message.vcsConnectors = object.vcsConnectors?.map((e) => VCSConnector.fromPartial(e)) || [];
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
      type: isSet(object.type) ? vCSTypeFromJSON(object.type) : 0,
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
      obj.type = vCSTypeToJSON(message.type);
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
    searchVCSProviderRepositories: {
      name: "SearchVCSProviderRepositories",
      requestType: SearchVCSProviderRepositoriesRequest,
      requestStream: false,
      responseType: SearchVCSProviderRepositoriesResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              49,
              58,
              1,
              42,
              34,
              44,
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
              82,
              101,
              112,
              111,
              115,
              105,
              116,
              111,
              114,
              105,
              101,
              115,
            ]),
          ],
        },
      },
    },
    listVCSConnectorsInProvider: {
      name: "ListVCSConnectorsInProvider",
      requestType: ListVCSConnectorsInProviderRequest,
      requestStream: false,
      responseType: ListVCSConnectorsInProviderResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              41,
              18,
              39,
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
              118,
              99,
              115,
              67,
              111,
              110,
              110,
              101,
              99,
              116,
              111,
              114,
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
