/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { Timestamp } from "../google/protobuf/timestamp";

export const protobufPackage = "bytebase.v1";

export interface CreateVCSConnectorRequest {
  /**
   * The parent resource where this vcsConnector will be created.
   * Format: projects/{project}
   */
  parent: string;
  /** The vcsConnector to create. */
  vcsConnector:
    | VCSConnector
    | undefined;
  /**
   * The ID to use for the vcsConnector, which will become the final component of
   * the vcsConnector's resource name.
   *
   * This value should be 4-63 characters, and valid characters
   * are /[a-z][0-9]-/.
   */
  vcsConnectorId: string;
}

export interface GetVCSConnectorRequest {
  /**
   * The name of the vcsConnector to retrieve.
   * Format: projects/{project}/vcsConnectors/{vcsConnector}
   */
  name: string;
}

export interface ListVCSConnectorsRequest {
  /**
   * The parent, which owns this collection of vcsConnectors.
   * Format: projects/{project}
   * Use "projects/-" to list all vcsConnectors.
   */
  parent: string;
  /**
   * The maximum number of databases to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 databases will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListDatabases` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListDatabases` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface ListVCSConnectorsResponse {
  /** The vcsConnectors from the specified request. */
  vcsConnectors: VCSConnector[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface UpdateVCSConnectorRequest {
  /**
   * The vcsConnector to update.
   *
   * The vcsConnector's `name` field is used to identify the vcsConnector to update.
   * Format: projects/{project}/vcsConnectors/{vcsConnector}
   */
  vcsConnector:
    | VCSConnector
    | undefined;
  /** The list of fields to be updated. */
  updateMask: string[] | undefined;
}

export interface DeleteVCSConnectorRequest {
  /**
   * The name of the vcsConnector to delete.
   * Format: projects/{project}/vcsConnectors/{vcsConnector}
   */
  name: string;
}

export interface VCSConnector {
  /**
   * The name of the vcsConnector resource.
   * Canonical parent is project.
   * Format: projects/{project}/vcsConnectors/{vcsConnector}
   */
  name: string;
  /** The title of the vcs connector. */
  title: string;
  /**
   * The creator of the vcsConnector.
   * Format: users/{email}
   */
  creator: string;
  /**
   * The updater of the vcsConnector.
   * Format: users/{email}
   */
  updater: string;
  /** The create time of the vcsConnector. */
  createTime:
    | Date
    | undefined;
  /** The last update time of the vcsConnector. */
  updateTime:
    | Date
    | undefined;
  /**
   * The name of the VCS.
   * Format: vcsProviders/{vcsProvider}
   */
  vcsProvider: string;
  /** The reposition external id in target VCS. */
  externalId: string;
  /** The root directory where Bytebase observes the file change. If empty, then it observes the entire repository. */
  baseDirectory: string;
  /** The branch Bytebase listens to for changes. For example: main. */
  branch: string;
  /** The webhook endpoint ID of the repository. */
  webhookEndpointId: string;
  /**
   * TODO(d): move these to create VCS connector API.
   * The full_path of the repository. For example: bytebase/sample.
   */
  fullPath: string;
  /** The web url of the repository. For axample: https://gitlab.bytebase.com/bytebase/sample. */
  webUrl: string;
}

function createBaseCreateVCSConnectorRequest(): CreateVCSConnectorRequest {
  return { parent: "", vcsConnector: undefined, vcsConnectorId: "" };
}

export const CreateVCSConnectorRequest = {
  encode(message: CreateVCSConnectorRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.vcsConnector !== undefined) {
      VCSConnector.encode(message.vcsConnector, writer.uint32(18).fork()).ldelim();
    }
    if (message.vcsConnectorId !== "") {
      writer.uint32(26).string(message.vcsConnectorId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateVCSConnectorRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateVCSConnectorRequest();
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

          message.vcsConnector = VCSConnector.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.vcsConnectorId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateVCSConnectorRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      vcsConnector: isSet(object.vcsConnector) ? VCSConnector.fromJSON(object.vcsConnector) : undefined,
      vcsConnectorId: isSet(object.vcsConnectorId) ? globalThis.String(object.vcsConnectorId) : "",
    };
  },

  toJSON(message: CreateVCSConnectorRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.vcsConnector !== undefined) {
      obj.vcsConnector = VCSConnector.toJSON(message.vcsConnector);
    }
    if (message.vcsConnectorId !== "") {
      obj.vcsConnectorId = message.vcsConnectorId;
    }
    return obj;
  },

  create(base?: DeepPartial<CreateVCSConnectorRequest>): CreateVCSConnectorRequest {
    return CreateVCSConnectorRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<CreateVCSConnectorRequest>): CreateVCSConnectorRequest {
    const message = createBaseCreateVCSConnectorRequest();
    message.parent = object.parent ?? "";
    message.vcsConnector = (object.vcsConnector !== undefined && object.vcsConnector !== null)
      ? VCSConnector.fromPartial(object.vcsConnector)
      : undefined;
    message.vcsConnectorId = object.vcsConnectorId ?? "";
    return message;
  },
};

function createBaseGetVCSConnectorRequest(): GetVCSConnectorRequest {
  return { name: "" };
}

export const GetVCSConnectorRequest = {
  encode(message: GetVCSConnectorRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetVCSConnectorRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetVCSConnectorRequest();
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

  fromJSON(object: any): GetVCSConnectorRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: GetVCSConnectorRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<GetVCSConnectorRequest>): GetVCSConnectorRequest {
    return GetVCSConnectorRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<GetVCSConnectorRequest>): GetVCSConnectorRequest {
    const message = createBaseGetVCSConnectorRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListVCSConnectorsRequest(): ListVCSConnectorsRequest {
  return { parent: "", pageSize: 0, pageToken: "" };
}

export const ListVCSConnectorsRequest = {
  encode(message: ListVCSConnectorsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): ListVCSConnectorsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListVCSConnectorsRequest();
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

  fromJSON(object: any): ListVCSConnectorsRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
    };
  },

  toJSON(message: ListVCSConnectorsRequest): unknown {
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

  create(base?: DeepPartial<ListVCSConnectorsRequest>): ListVCSConnectorsRequest {
    return ListVCSConnectorsRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListVCSConnectorsRequest>): ListVCSConnectorsRequest {
    const message = createBaseListVCSConnectorsRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListVCSConnectorsResponse(): ListVCSConnectorsResponse {
  return { vcsConnectors: [], nextPageToken: "" };
}

export const ListVCSConnectorsResponse = {
  encode(message: ListVCSConnectorsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.vcsConnectors) {
      VCSConnector.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListVCSConnectorsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListVCSConnectorsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.vcsConnectors.push(VCSConnector.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListVCSConnectorsResponse {
    return {
      vcsConnectors: globalThis.Array.isArray(object?.vcsConnectors)
        ? object.vcsConnectors.map((e: any) => VCSConnector.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListVCSConnectorsResponse): unknown {
    const obj: any = {};
    if (message.vcsConnectors?.length) {
      obj.vcsConnectors = message.vcsConnectors.map((e) => VCSConnector.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListVCSConnectorsResponse>): ListVCSConnectorsResponse {
    return ListVCSConnectorsResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListVCSConnectorsResponse>): ListVCSConnectorsResponse {
    const message = createBaseListVCSConnectorsResponse();
    message.vcsConnectors = object.vcsConnectors?.map((e) => VCSConnector.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseUpdateVCSConnectorRequest(): UpdateVCSConnectorRequest {
  return { vcsConnector: undefined, updateMask: undefined };
}

export const UpdateVCSConnectorRequest = {
  encode(message: UpdateVCSConnectorRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.vcsConnector !== undefined) {
      VCSConnector.encode(message.vcsConnector, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateVCSConnectorRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateVCSConnectorRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.vcsConnector = VCSConnector.decode(reader, reader.uint32());
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

  fromJSON(object: any): UpdateVCSConnectorRequest {
    return {
      vcsConnector: isSet(object.vcsConnector) ? VCSConnector.fromJSON(object.vcsConnector) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateVCSConnectorRequest): unknown {
    const obj: any = {};
    if (message.vcsConnector !== undefined) {
      obj.vcsConnector = VCSConnector.toJSON(message.vcsConnector);
    }
    if (message.updateMask !== undefined) {
      obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask));
    }
    return obj;
  },

  create(base?: DeepPartial<UpdateVCSConnectorRequest>): UpdateVCSConnectorRequest {
    return UpdateVCSConnectorRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UpdateVCSConnectorRequest>): UpdateVCSConnectorRequest {
    const message = createBaseUpdateVCSConnectorRequest();
    message.vcsConnector = (object.vcsConnector !== undefined && object.vcsConnector !== null)
      ? VCSConnector.fromPartial(object.vcsConnector)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseDeleteVCSConnectorRequest(): DeleteVCSConnectorRequest {
  return { name: "" };
}

export const DeleteVCSConnectorRequest = {
  encode(message: DeleteVCSConnectorRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteVCSConnectorRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteVCSConnectorRequest();
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

  fromJSON(object: any): DeleteVCSConnectorRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: DeleteVCSConnectorRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<DeleteVCSConnectorRequest>): DeleteVCSConnectorRequest {
    return DeleteVCSConnectorRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DeleteVCSConnectorRequest>): DeleteVCSConnectorRequest {
    const message = createBaseDeleteVCSConnectorRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseVCSConnector(): VCSConnector {
  return {
    name: "",
    title: "",
    creator: "",
    updater: "",
    createTime: undefined,
    updateTime: undefined,
    vcsProvider: "",
    externalId: "",
    baseDirectory: "",
    branch: "",
    webhookEndpointId: "",
    fullPath: "",
    webUrl: "",
  };
}

export const VCSConnector = {
  encode(message: VCSConnector, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.creator !== "") {
      writer.uint32(26).string(message.creator);
    }
    if (message.updater !== "") {
      writer.uint32(34).string(message.updater);
    }
    if (message.createTime !== undefined) {
      Timestamp.encode(toTimestamp(message.createTime), writer.uint32(42).fork()).ldelim();
    }
    if (message.updateTime !== undefined) {
      Timestamp.encode(toTimestamp(message.updateTime), writer.uint32(50).fork()).ldelim();
    }
    if (message.vcsProvider !== "") {
      writer.uint32(58).string(message.vcsProvider);
    }
    if (message.externalId !== "") {
      writer.uint32(66).string(message.externalId);
    }
    if (message.baseDirectory !== "") {
      writer.uint32(74).string(message.baseDirectory);
    }
    if (message.branch !== "") {
      writer.uint32(82).string(message.branch);
    }
    if (message.webhookEndpointId !== "") {
      writer.uint32(90).string(message.webhookEndpointId);
    }
    if (message.fullPath !== "") {
      writer.uint32(98).string(message.fullPath);
    }
    if (message.webUrl !== "") {
      writer.uint32(106).string(message.webUrl);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): VCSConnector {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseVCSConnector();
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

          message.creator = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.updater = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.createTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.updateTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.vcsProvider = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.externalId = reader.string();
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.baseDirectory = reader.string();
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.branch = reader.string();
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.webhookEndpointId = reader.string();
          continue;
        case 12:
          if (tag !== 98) {
            break;
          }

          message.fullPath = reader.string();
          continue;
        case 13:
          if (tag !== 106) {
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

  fromJSON(object: any): VCSConnector {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      creator: isSet(object.creator) ? globalThis.String(object.creator) : "",
      updater: isSet(object.updater) ? globalThis.String(object.updater) : "",
      createTime: isSet(object.createTime) ? fromJsonTimestamp(object.createTime) : undefined,
      updateTime: isSet(object.updateTime) ? fromJsonTimestamp(object.updateTime) : undefined,
      vcsProvider: isSet(object.vcsProvider) ? globalThis.String(object.vcsProvider) : "",
      externalId: isSet(object.externalId) ? globalThis.String(object.externalId) : "",
      baseDirectory: isSet(object.baseDirectory) ? globalThis.String(object.baseDirectory) : "",
      branch: isSet(object.branch) ? globalThis.String(object.branch) : "",
      webhookEndpointId: isSet(object.webhookEndpointId) ? globalThis.String(object.webhookEndpointId) : "",
      fullPath: isSet(object.fullPath) ? globalThis.String(object.fullPath) : "",
      webUrl: isSet(object.webUrl) ? globalThis.String(object.webUrl) : "",
    };
  },

  toJSON(message: VCSConnector): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.creator !== "") {
      obj.creator = message.creator;
    }
    if (message.updater !== "") {
      obj.updater = message.updater;
    }
    if (message.createTime !== undefined) {
      obj.createTime = message.createTime.toISOString();
    }
    if (message.updateTime !== undefined) {
      obj.updateTime = message.updateTime.toISOString();
    }
    if (message.vcsProvider !== "") {
      obj.vcsProvider = message.vcsProvider;
    }
    if (message.externalId !== "") {
      obj.externalId = message.externalId;
    }
    if (message.baseDirectory !== "") {
      obj.baseDirectory = message.baseDirectory;
    }
    if (message.branch !== "") {
      obj.branch = message.branch;
    }
    if (message.webhookEndpointId !== "") {
      obj.webhookEndpointId = message.webhookEndpointId;
    }
    if (message.fullPath !== "") {
      obj.fullPath = message.fullPath;
    }
    if (message.webUrl !== "") {
      obj.webUrl = message.webUrl;
    }
    return obj;
  },

  create(base?: DeepPartial<VCSConnector>): VCSConnector {
    return VCSConnector.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<VCSConnector>): VCSConnector {
    const message = createBaseVCSConnector();
    message.name = object.name ?? "";
    message.title = object.title ?? "";
    message.creator = object.creator ?? "";
    message.updater = object.updater ?? "";
    message.createTime = object.createTime ?? undefined;
    message.updateTime = object.updateTime ?? undefined;
    message.vcsProvider = object.vcsProvider ?? "";
    message.externalId = object.externalId ?? "";
    message.baseDirectory = object.baseDirectory ?? "";
    message.branch = object.branch ?? "";
    message.webhookEndpointId = object.webhookEndpointId ?? "";
    message.fullPath = object.fullPath ?? "";
    message.webUrl = object.webUrl ?? "";
    return message;
  },
};

export type VCSConnectorServiceDefinition = typeof VCSConnectorServiceDefinition;
export const VCSConnectorServiceDefinition = {
  name: "VCSConnectorService",
  fullName: "bytebase.v1.VCSConnectorService",
  methods: {
    createVCSConnector: {
      name: "CreateVCSConnector",
      requestType: CreateVCSConnectorRequest,
      requestStream: false,
      responseType: VCSConnector,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([
              19,
              112,
              97,
              114,
              101,
              110,
              116,
              44,
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
            ]),
          ],
          578365826: [
            new Uint8Array([
              54,
              58,
              13,
              118,
              99,
              115,
              95,
              99,
              111,
              110,
              110,
              101,
              99,
              116,
              111,
              114,
              34,
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
    getVCSConnector: {
      name: "GetVCSConnector",
      requestType: GetVCSConnectorRequest,
      requestStream: false,
      responseType: VCSConnector,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
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
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    listVCSConnectors: {
      name: "ListVCSConnectors",
      requestType: ListVCSConnectorsRequest,
      requestStream: false,
      responseType: ListVCSConnectorsResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([6, 112, 97, 114, 101, 110, 116])],
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
    updateVCSConnector: {
      name: "UpdateVCSConnector",
      requestType: UpdateVCSConnectorRequest,
      requestStream: false,
      responseType: VCSConnector,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([
              25,
              118,
              99,
              115,
              95,
              99,
              111,
              110,
              110,
              101,
              99,
              116,
              111,
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
              68,
              58,
              13,
              118,
              99,
              115,
              95,
              99,
              111,
              110,
              110,
              101,
              99,
              116,
              111,
              114,
              50,
              51,
              47,
              118,
              49,
              47,
              123,
              118,
              99,
              115,
              95,
              99,
              111,
              110,
              110,
              101,
              99,
              116,
              111,
              114,
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
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    deleteVCSConnector: {
      name: "DeleteVCSConnector",
      requestType: DeleteVCSConnectorRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              39,
              42,
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

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
