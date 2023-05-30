/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { Timestamp } from "../google/protobuf/timestamp";

export const protobufPackage = "bytebase.v1";

export interface GetActuatorInfoRequest {
}

export interface DeleteCacheRequest {
}

/**
 * ServerInfo is the API message for server info.
 * Actuator concept is similar to the Spring Boot Actuator.
 */
export interface ActuatorInfo {
  /** version is the bytebase's server version */
  version: string;
  /** git_commit is the git commit hash of the build */
  gitCommit: string;
  /** readonly flag means if the Bytebase is running in readonly mode. */
  readonly: boolean;
  /** saas flag means if the Bytebase is running in SaaS mode, some features are not allowed to edit by users. */
  saas: boolean;
  /** demo_name specifies the demo name, empty string means no demo. */
  demoName: string;
  /** host is the Bytebase instance host. */
  host: string;
  /** port is the Bytebase instance port. */
  port: string;
  /** external_url is the URL where user or webhook callback visits Bytebase. */
  externalUrl: string;
  /** need_admin_setup flag means the Bytebase instance doesn't have any end users. */
  needAdminSetup: boolean;
  /** disallow_signup is the flag to disable self-service signup. */
  disallowSignup: boolean;
  /** last_active_time is the service last active time in UTC Time Format, any API calls will refresh this value. */
  lastActiveTime?: Date;
  /** require_2fa is the flag to require 2FA for all users. */
  require2fa: boolean;
  /** workspace_id is the identifier for the workspace. */
  workspaceId: string;
  /** gitops_webhook_url is the webhook URL for GitOps. */
  gitopsWebhookUrl: string;
}

function createBaseGetActuatorInfoRequest(): GetActuatorInfoRequest {
  return {};
}

export const GetActuatorInfoRequest = {
  encode(_: GetActuatorInfoRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetActuatorInfoRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetActuatorInfoRequest();
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

  fromJSON(_: any): GetActuatorInfoRequest {
    return {};
  },

  toJSON(_: GetActuatorInfoRequest): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<GetActuatorInfoRequest>): GetActuatorInfoRequest {
    return GetActuatorInfoRequest.fromPartial(base ?? {});
  },

  fromPartial(_: DeepPartial<GetActuatorInfoRequest>): GetActuatorInfoRequest {
    const message = createBaseGetActuatorInfoRequest();
    return message;
  },
};

function createBaseDeleteCacheRequest(): DeleteCacheRequest {
  return {};
}

export const DeleteCacheRequest = {
  encode(_: DeleteCacheRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteCacheRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteCacheRequest();
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

  fromJSON(_: any): DeleteCacheRequest {
    return {};
  },

  toJSON(_: DeleteCacheRequest): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<DeleteCacheRequest>): DeleteCacheRequest {
    return DeleteCacheRequest.fromPartial(base ?? {});
  },

  fromPartial(_: DeepPartial<DeleteCacheRequest>): DeleteCacheRequest {
    const message = createBaseDeleteCacheRequest();
    return message;
  },
};

function createBaseActuatorInfo(): ActuatorInfo {
  return {
    version: "",
    gitCommit: "",
    readonly: false,
    saas: false,
    demoName: "",
    host: "",
    port: "",
    externalUrl: "",
    needAdminSetup: false,
    disallowSignup: false,
    lastActiveTime: undefined,
    require2fa: false,
    workspaceId: "",
    gitopsWebhookUrl: "",
  };
}

export const ActuatorInfo = {
  encode(message: ActuatorInfo, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.version !== "") {
      writer.uint32(10).string(message.version);
    }
    if (message.gitCommit !== "") {
      writer.uint32(18).string(message.gitCommit);
    }
    if (message.readonly === true) {
      writer.uint32(24).bool(message.readonly);
    }
    if (message.saas === true) {
      writer.uint32(32).bool(message.saas);
    }
    if (message.demoName !== "") {
      writer.uint32(42).string(message.demoName);
    }
    if (message.host !== "") {
      writer.uint32(50).string(message.host);
    }
    if (message.port !== "") {
      writer.uint32(58).string(message.port);
    }
    if (message.externalUrl !== "") {
      writer.uint32(66).string(message.externalUrl);
    }
    if (message.needAdminSetup === true) {
      writer.uint32(72).bool(message.needAdminSetup);
    }
    if (message.disallowSignup === true) {
      writer.uint32(80).bool(message.disallowSignup);
    }
    if (message.lastActiveTime !== undefined) {
      Timestamp.encode(toTimestamp(message.lastActiveTime), writer.uint32(90).fork()).ldelim();
    }
    if (message.require2fa === true) {
      writer.uint32(96).bool(message.require2fa);
    }
    if (message.workspaceId !== "") {
      writer.uint32(106).string(message.workspaceId);
    }
    if (message.gitopsWebhookUrl !== "") {
      writer.uint32(114).string(message.gitopsWebhookUrl);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ActuatorInfo {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseActuatorInfo();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.version = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.gitCommit = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.readonly = reader.bool();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.saas = reader.bool();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.demoName = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.host = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.port = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.externalUrl = reader.string();
          continue;
        case 9:
          if (tag !== 72) {
            break;
          }

          message.needAdminSetup = reader.bool();
          continue;
        case 10:
          if (tag !== 80) {
            break;
          }

          message.disallowSignup = reader.bool();
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.lastActiveTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 12:
          if (tag !== 96) {
            break;
          }

          message.require2fa = reader.bool();
          continue;
        case 13:
          if (tag !== 106) {
            break;
          }

          message.workspaceId = reader.string();
          continue;
        case 14:
          if (tag !== 114) {
            break;
          }

          message.gitopsWebhookUrl = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ActuatorInfo {
    return {
      version: isSet(object.version) ? String(object.version) : "",
      gitCommit: isSet(object.gitCommit) ? String(object.gitCommit) : "",
      readonly: isSet(object.readonly) ? Boolean(object.readonly) : false,
      saas: isSet(object.saas) ? Boolean(object.saas) : false,
      demoName: isSet(object.demoName) ? String(object.demoName) : "",
      host: isSet(object.host) ? String(object.host) : "",
      port: isSet(object.port) ? String(object.port) : "",
      externalUrl: isSet(object.externalUrl) ? String(object.externalUrl) : "",
      needAdminSetup: isSet(object.needAdminSetup) ? Boolean(object.needAdminSetup) : false,
      disallowSignup: isSet(object.disallowSignup) ? Boolean(object.disallowSignup) : false,
      lastActiveTime: isSet(object.lastActiveTime) ? fromJsonTimestamp(object.lastActiveTime) : undefined,
      require2fa: isSet(object.require2fa) ? Boolean(object.require2fa) : false,
      workspaceId: isSet(object.workspaceId) ? String(object.workspaceId) : "",
      gitopsWebhookUrl: isSet(object.gitopsWebhookUrl) ? String(object.gitopsWebhookUrl) : "",
    };
  },

  toJSON(message: ActuatorInfo): unknown {
    const obj: any = {};
    message.version !== undefined && (obj.version = message.version);
    message.gitCommit !== undefined && (obj.gitCommit = message.gitCommit);
    message.readonly !== undefined && (obj.readonly = message.readonly);
    message.saas !== undefined && (obj.saas = message.saas);
    message.demoName !== undefined && (obj.demoName = message.demoName);
    message.host !== undefined && (obj.host = message.host);
    message.port !== undefined && (obj.port = message.port);
    message.externalUrl !== undefined && (obj.externalUrl = message.externalUrl);
    message.needAdminSetup !== undefined && (obj.needAdminSetup = message.needAdminSetup);
    message.disallowSignup !== undefined && (obj.disallowSignup = message.disallowSignup);
    message.lastActiveTime !== undefined && (obj.lastActiveTime = message.lastActiveTime.toISOString());
    message.require2fa !== undefined && (obj.require2fa = message.require2fa);
    message.workspaceId !== undefined && (obj.workspaceId = message.workspaceId);
    message.gitopsWebhookUrl !== undefined && (obj.gitopsWebhookUrl = message.gitopsWebhookUrl);
    return obj;
  },

  create(base?: DeepPartial<ActuatorInfo>): ActuatorInfo {
    return ActuatorInfo.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ActuatorInfo>): ActuatorInfo {
    const message = createBaseActuatorInfo();
    message.version = object.version ?? "";
    message.gitCommit = object.gitCommit ?? "";
    message.readonly = object.readonly ?? false;
    message.saas = object.saas ?? false;
    message.demoName = object.demoName ?? "";
    message.host = object.host ?? "";
    message.port = object.port ?? "";
    message.externalUrl = object.externalUrl ?? "";
    message.needAdminSetup = object.needAdminSetup ?? false;
    message.disallowSignup = object.disallowSignup ?? false;
    message.lastActiveTime = object.lastActiveTime ?? undefined;
    message.require2fa = object.require2fa ?? false;
    message.workspaceId = object.workspaceId ?? "";
    message.gitopsWebhookUrl = object.gitopsWebhookUrl ?? "";
    return message;
  },
};

export type ActuatorServiceDefinition = typeof ActuatorServiceDefinition;
export const ActuatorServiceDefinition = {
  name: "ActuatorService",
  fullName: "bytebase.v1.ActuatorService",
  methods: {
    getActuatorInfo: {
      name: "GetActuatorInfo",
      requestType: GetActuatorInfoRequest,
      requestStream: false,
      responseType: ActuatorInfo,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          578365826: [
            new Uint8Array([19, 18, 17, 47, 118, 49, 47, 97, 99, 116, 117, 97, 116, 111, 114, 47, 105, 110, 102, 111]),
          ],
        },
      },
    },
    deleteCache: {
      name: "DeleteCache",
      requestType: DeleteCacheRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              20,
              42,
              18,
              47,
              118,
              49,
              47,
              97,
              99,
              116,
              117,
              97,
              116,
              111,
              114,
              47,
              99,
              97,
              99,
              104,
              101,
            ]),
          ],
        },
      },
    },
  },
} as const;

export interface ActuatorServiceImplementation<CallContextExt = {}> {
  getActuatorInfo(
    request: GetActuatorInfoRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ActuatorInfo>>;
  deleteCache(request: DeleteCacheRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Empty>>;
}

export interface ActuatorServiceClient<CallOptionsExt = {}> {
  getActuatorInfo(
    request: DeepPartial<GetActuatorInfoRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ActuatorInfo>;
  deleteCache(request: DeepPartial<DeleteCacheRequest>, options?: CallOptions & CallOptionsExt): Promise<Empty>;
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
  let millis = (t.seconds || 0) * 1_000;
  millis += (t.nanos || 0) / 1_000_000;
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
