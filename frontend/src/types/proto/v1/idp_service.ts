/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { State, stateFromJSON, stateToJSON } from "./common";

export const protobufPackage = "bytebase.v1";

export enum IdentityProviderType {
  IDENTITY_PROVIDER_UNSPECIFIED = 0,
  OAUTH2 = 1,
  OIDC = 2,
  UNRECOGNIZED = -1,
}

export function identityProviderTypeFromJSON(object: any): IdentityProviderType {
  switch (object) {
    case 0:
    case "IDENTITY_PROVIDER_UNSPECIFIED":
      return IdentityProviderType.IDENTITY_PROVIDER_UNSPECIFIED;
    case 1:
    case "OAUTH2":
      return IdentityProviderType.OAUTH2;
    case 2:
    case "OIDC":
      return IdentityProviderType.OIDC;
    case -1:
    case "UNRECOGNIZED":
    default:
      return IdentityProviderType.UNRECOGNIZED;
  }
}

export function identityProviderTypeToJSON(object: IdentityProviderType): string {
  switch (object) {
    case IdentityProviderType.IDENTITY_PROVIDER_UNSPECIFIED:
      return "IDENTITY_PROVIDER_UNSPECIFIED";
    case IdentityProviderType.OAUTH2:
      return "OAUTH2";
    case IdentityProviderType.OIDC:
      return "OIDC";
    case IdentityProviderType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface GetIdentityProviderRequest {
  name: string;
}

export interface ListIdentityProvidersRequest {
  /**
   * The maximum number of identity providers to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListIdentityProviders` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListIdentityProviders` must match
   * the call that provided the page token.
   */
  pageToken: string;
  /** Show deleted identity providers if specified. */
  showDeleted: boolean;
}

export interface ListIdentityProvidersResponse {
  /** The identity providers from the specified request. */
  identityProviders: IdentityProvider[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface CreateIdentityProviderRequest {
  /** The identity provider to create. */
  identityProvider?: IdentityProvider;
  /**
   * The ID to use for the identity provider, which will become the final component of
   * the identity provider's resource name.
   *
   * This value should be 4-63 characters, and valid characters
   * are /[a-z][0-9]-/.
   */
  identityProviderId: string;
}

export interface UpdateIdentityProviderRequest {
  /**
   * The identity provider to update.
   *
   * The identity provider's `name` field is used to identify the identity provider to update.
   * Format: idps/{identity_provider}
   */
  identityProvider?: IdentityProvider;
  /** The list of fields to update. */
  updateMask?: string[];
}

export interface DeleteIdentityProviderRequest {
  /**
   * The name of the identity provider to delete.
   * Format: idps/{identity_provider}
   */
  name: string;
}

export interface UndeleteIdentityProviderRequest {
  /**
   * The name of the deleted identity provider.
   * Format: idps/{identity_provider}
   */
  name: string;
}

export interface IdentityProvider {
  /**
   * The name of the identity provider.
   * Format: idps/{identity_provider}
   */
  name: string;
  /** The system-assigned, unique identifier for a resource. */
  uid: string;
  state: State;
  title: string;
  domain: string;
  type: IdentityProviderType;
  config: string;
}

function createBaseGetIdentityProviderRequest(): GetIdentityProviderRequest {
  return { name: "" };
}

export const GetIdentityProviderRequest = {
  encode(message: GetIdentityProviderRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetIdentityProviderRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetIdentityProviderRequest();
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

  fromJSON(object: any): GetIdentityProviderRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetIdentityProviderRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<GetIdentityProviderRequest>): GetIdentityProviderRequest {
    const message = createBaseGetIdentityProviderRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListIdentityProvidersRequest(): ListIdentityProvidersRequest {
  return { pageSize: 0, pageToken: "", showDeleted: false };
}

export const ListIdentityProvidersRequest = {
  encode(message: ListIdentityProvidersRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): ListIdentityProvidersRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListIdentityProvidersRequest();
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

  fromJSON(object: any): ListIdentityProvidersRequest {
    return {
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
      showDeleted: isSet(object.showDeleted) ? Boolean(object.showDeleted) : false,
    };
  },

  toJSON(message: ListIdentityProvidersRequest): unknown {
    const obj: any = {};
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    message.showDeleted !== undefined && (obj.showDeleted = message.showDeleted);
    return obj;
  },

  fromPartial(object: DeepPartial<ListIdentityProvidersRequest>): ListIdentityProvidersRequest {
    const message = createBaseListIdentityProvidersRequest();
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    message.showDeleted = object.showDeleted ?? false;
    return message;
  },
};

function createBaseListIdentityProvidersResponse(): ListIdentityProvidersResponse {
  return { identityProviders: [], nextPageToken: "" };
}

export const ListIdentityProvidersResponse = {
  encode(message: ListIdentityProvidersResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.identityProviders) {
      IdentityProvider.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListIdentityProvidersResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListIdentityProvidersResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.identityProviders.push(IdentityProvider.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListIdentityProvidersResponse {
    return {
      identityProviders: Array.isArray(object?.identityProviders)
        ? object.identityProviders.map((e: any) => IdentityProvider.fromJSON(e))
        : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListIdentityProvidersResponse): unknown {
    const obj: any = {};
    if (message.identityProviders) {
      obj.identityProviders = message.identityProviders.map((e) => e ? IdentityProvider.toJSON(e) : undefined);
    } else {
      obj.identityProviders = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  fromPartial(object: DeepPartial<ListIdentityProvidersResponse>): ListIdentityProvidersResponse {
    const message = createBaseListIdentityProvidersResponse();
    message.identityProviders = object.identityProviders?.map((e) => IdentityProvider.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseCreateIdentityProviderRequest(): CreateIdentityProviderRequest {
  return { identityProvider: undefined, identityProviderId: "" };
}

export const CreateIdentityProviderRequest = {
  encode(message: CreateIdentityProviderRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.identityProvider !== undefined) {
      IdentityProvider.encode(message.identityProvider, writer.uint32(10).fork()).ldelim();
    }
    if (message.identityProviderId !== "") {
      writer.uint32(18).string(message.identityProviderId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateIdentityProviderRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateIdentityProviderRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.identityProvider = IdentityProvider.decode(reader, reader.uint32());
          break;
        case 2:
          message.identityProviderId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): CreateIdentityProviderRequest {
    return {
      identityProvider: isSet(object.identityProvider) ? IdentityProvider.fromJSON(object.identityProvider) : undefined,
      identityProviderId: isSet(object.identityProviderId) ? String(object.identityProviderId) : "",
    };
  },

  toJSON(message: CreateIdentityProviderRequest): unknown {
    const obj: any = {};
    message.identityProvider !== undefined &&
      (obj.identityProvider = message.identityProvider ? IdentityProvider.toJSON(message.identityProvider) : undefined);
    message.identityProviderId !== undefined && (obj.identityProviderId = message.identityProviderId);
    return obj;
  },

  fromPartial(object: DeepPartial<CreateIdentityProviderRequest>): CreateIdentityProviderRequest {
    const message = createBaseCreateIdentityProviderRequest();
    message.identityProvider = (object.identityProvider !== undefined && object.identityProvider !== null)
      ? IdentityProvider.fromPartial(object.identityProvider)
      : undefined;
    message.identityProviderId = object.identityProviderId ?? "";
    return message;
  },
};

function createBaseUpdateIdentityProviderRequest(): UpdateIdentityProviderRequest {
  return { identityProvider: undefined, updateMask: undefined };
}

export const UpdateIdentityProviderRequest = {
  encode(message: UpdateIdentityProviderRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.identityProvider !== undefined) {
      IdentityProvider.encode(message.identityProvider, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateIdentityProviderRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateIdentityProviderRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.identityProvider = IdentityProvider.decode(reader, reader.uint32());
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

  fromJSON(object: any): UpdateIdentityProviderRequest {
    return {
      identityProvider: isSet(object.identityProvider) ? IdentityProvider.fromJSON(object.identityProvider) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateIdentityProviderRequest): unknown {
    const obj: any = {};
    message.identityProvider !== undefined &&
      (obj.identityProvider = message.identityProvider ? IdentityProvider.toJSON(message.identityProvider) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  fromPartial(object: DeepPartial<UpdateIdentityProviderRequest>): UpdateIdentityProviderRequest {
    const message = createBaseUpdateIdentityProviderRequest();
    message.identityProvider = (object.identityProvider !== undefined && object.identityProvider !== null)
      ? IdentityProvider.fromPartial(object.identityProvider)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseDeleteIdentityProviderRequest(): DeleteIdentityProviderRequest {
  return { name: "" };
}

export const DeleteIdentityProviderRequest = {
  encode(message: DeleteIdentityProviderRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteIdentityProviderRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteIdentityProviderRequest();
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

  fromJSON(object: any): DeleteIdentityProviderRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: DeleteIdentityProviderRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<DeleteIdentityProviderRequest>): DeleteIdentityProviderRequest {
    const message = createBaseDeleteIdentityProviderRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseUndeleteIdentityProviderRequest(): UndeleteIdentityProviderRequest {
  return { name: "" };
}

export const UndeleteIdentityProviderRequest = {
  encode(message: UndeleteIdentityProviderRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UndeleteIdentityProviderRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUndeleteIdentityProviderRequest();
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

  fromJSON(object: any): UndeleteIdentityProviderRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: UndeleteIdentityProviderRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<UndeleteIdentityProviderRequest>): UndeleteIdentityProviderRequest {
    const message = createBaseUndeleteIdentityProviderRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseIdentityProvider(): IdentityProvider {
  return { name: "", uid: "", state: 0, title: "", domain: "", type: 0, config: "" };
}

export const IdentityProvider = {
  encode(message: IdentityProvider, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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
    if (message.domain !== "") {
      writer.uint32(42).string(message.domain);
    }
    if (message.type !== 0) {
      writer.uint32(48).int32(message.type);
    }
    if (message.config !== "") {
      writer.uint32(58).string(message.config);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IdentityProvider {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIdentityProvider();
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
          message.domain = reader.string();
          break;
        case 6:
          message.type = reader.int32() as any;
          break;
        case 7:
          message.config = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): IdentityProvider {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      uid: isSet(object.uid) ? String(object.uid) : "",
      state: isSet(object.state) ? stateFromJSON(object.state) : 0,
      title: isSet(object.title) ? String(object.title) : "",
      domain: isSet(object.domain) ? String(object.domain) : "",
      type: isSet(object.type) ? identityProviderTypeFromJSON(object.type) : 0,
      config: isSet(object.config) ? String(object.config) : "",
    };
  },

  toJSON(message: IdentityProvider): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.state !== undefined && (obj.state = stateToJSON(message.state));
    message.title !== undefined && (obj.title = message.title);
    message.domain !== undefined && (obj.domain = message.domain);
    message.type !== undefined && (obj.type = identityProviderTypeToJSON(message.type));
    message.config !== undefined && (obj.config = message.config);
    return obj;
  },

  fromPartial(object: DeepPartial<IdentityProvider>): IdentityProvider {
    const message = createBaseIdentityProvider();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.state = object.state ?? 0;
    message.title = object.title ?? "";
    message.domain = object.domain ?? "";
    message.type = object.type ?? 0;
    message.config = object.config ?? "";
    return message;
  },
};

export type IdentityProviderServiceDefinition = typeof IdentityProviderServiceDefinition;
export const IdentityProviderServiceDefinition = {
  name: "IdentityProviderService",
  fullName: "bytebase.v1.IdentityProviderService",
  methods: {
    getIdentityProvider: {
      name: "GetIdentityProvider",
      requestType: GetIdentityProviderRequest,
      requestStream: false,
      responseType: IdentityProvider,
      responseStream: false,
      options: {},
    },
    listIdentityProviders: {
      name: "ListIdentityProviders",
      requestType: ListIdentityProvidersRequest,
      requestStream: false,
      responseType: ListIdentityProvidersResponse,
      responseStream: false,
      options: {},
    },
    createIdentityProvider: {
      name: "CreateIdentityProvider",
      requestType: CreateIdentityProviderRequest,
      requestStream: false,
      responseType: IdentityProvider,
      responseStream: false,
      options: {},
    },
    updateIdentityProvider: {
      name: "UpdateIdentityProvider",
      requestType: UpdateIdentityProviderRequest,
      requestStream: false,
      responseType: IdentityProvider,
      responseStream: false,
      options: {},
    },
    deleteIdentityProvider: {
      name: "DeleteIdentityProvider",
      requestType: DeleteIdentityProviderRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {},
    },
    undeleteIdentityProvider: {
      name: "UndeleteIdentityProvider",
      requestType: UndeleteIdentityProviderRequest,
      requestStream: false,
      responseType: IdentityProvider,
      responseStream: false,
      options: {},
    },
  },
} as const;

export interface IdentityProviderServiceImplementation<CallContextExt = {}> {
  getIdentityProvider(
    request: GetIdentityProviderRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<IdentityProvider>>;
  listIdentityProviders(
    request: ListIdentityProvidersRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ListIdentityProvidersResponse>>;
  createIdentityProvider(
    request: CreateIdentityProviderRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<IdentityProvider>>;
  updateIdentityProvider(
    request: UpdateIdentityProviderRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<IdentityProvider>>;
  deleteIdentityProvider(
    request: DeleteIdentityProviderRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Empty>>;
  undeleteIdentityProvider(
    request: UndeleteIdentityProviderRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<IdentityProvider>>;
}

export interface IdentityProviderServiceClient<CallOptionsExt = {}> {
  getIdentityProvider(
    request: DeepPartial<GetIdentityProviderRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<IdentityProvider>;
  listIdentityProviders(
    request: DeepPartial<ListIdentityProvidersRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ListIdentityProvidersResponse>;
  createIdentityProvider(
    request: DeepPartial<CreateIdentityProviderRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<IdentityProvider>;
  updateIdentityProvider(
    request: DeepPartial<UpdateIdentityProviderRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<IdentityProvider>;
  deleteIdentityProvider(
    request: DeepPartial<DeleteIdentityProviderRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Empty>;
  undeleteIdentityProvider(
    request: DeepPartial<UndeleteIdentityProviderRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<IdentityProvider>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
