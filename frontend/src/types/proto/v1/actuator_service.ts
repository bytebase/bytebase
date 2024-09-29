/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { Timestamp } from "../google/protobuf/timestamp";
import { PasswordRestrictionSetting } from "./setting_service";

export const protobufPackage = "bytebase.v1";

/** The request message for getting the theme resource. */
export interface GetResourcePackageRequest {
}

/** The theme resources. */
export interface ResourcePackage {
  /** The branding logo. */
  logo: Uint8Array;
}

export interface GetActuatorInfoRequest {
}

export interface UpdateActuatorInfoRequest {
  /** The actuator to update. */
  actuator:
    | ActuatorInfo
    | undefined;
  /** The list of fields to update. */
  updateMask: string[] | undefined;
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
  lastActiveTime:
    | Date
    | undefined;
  /** require_2fa is the flag to require 2FA for all users. */
  require2fa: boolean;
  /** workspace_id is the identifier for the workspace. */
  workspaceId: string;
  /** gitops_webhook_url is the webhook URL for GitOps. */
  gitopsWebhookUrl: string;
  /** debug flag means if the debug mode is enabled. */
  debug: boolean;
  /** lsp is the enablement of lsp in SQL Editor. */
  lsp: boolean;
  /** pre_update_backup is the enablement of data backup prior to data update. */
  preUpdateBackup: boolean;
  /** iam_guard is the enablement of IAM checks. */
  iamGuard: boolean;
  unlicensedFeatures: string[];
  /** disallow_password_signin is the flag to disallow user signin with email&password. (except workspace admins) */
  disallowPasswordSignin: boolean;
  passwordRestriction: PasswordRestrictionSetting | undefined;
}

function createBaseGetResourcePackageRequest(): GetResourcePackageRequest {
  return {};
}

export const GetResourcePackageRequest = {
  encode(_: GetResourcePackageRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetResourcePackageRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetResourcePackageRequest();
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

  fromJSON(_: any): GetResourcePackageRequest {
    return {};
  },

  toJSON(_: GetResourcePackageRequest): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<GetResourcePackageRequest>): GetResourcePackageRequest {
    return GetResourcePackageRequest.fromPartial(base ?? {});
  },
  fromPartial(_: DeepPartial<GetResourcePackageRequest>): GetResourcePackageRequest {
    const message = createBaseGetResourcePackageRequest();
    return message;
  },
};

function createBaseResourcePackage(): ResourcePackage {
  return { logo: new Uint8Array(0) };
}

export const ResourcePackage = {
  encode(message: ResourcePackage, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.logo.length !== 0) {
      writer.uint32(10).bytes(message.logo);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ResourcePackage {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseResourcePackage();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.logo = reader.bytes();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ResourcePackage {
    return { logo: isSet(object.logo) ? bytesFromBase64(object.logo) : new Uint8Array(0) };
  },

  toJSON(message: ResourcePackage): unknown {
    const obj: any = {};
    if (message.logo.length !== 0) {
      obj.logo = base64FromBytes(message.logo);
    }
    return obj;
  },

  create(base?: DeepPartial<ResourcePackage>): ResourcePackage {
    return ResourcePackage.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ResourcePackage>): ResourcePackage {
    const message = createBaseResourcePackage();
    message.logo = object.logo ?? new Uint8Array(0);
    return message;
  },
};

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

function createBaseUpdateActuatorInfoRequest(): UpdateActuatorInfoRequest {
  return { actuator: undefined, updateMask: undefined };
}

export const UpdateActuatorInfoRequest = {
  encode(message: UpdateActuatorInfoRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.actuator !== undefined) {
      ActuatorInfo.encode(message.actuator, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateActuatorInfoRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateActuatorInfoRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.actuator = ActuatorInfo.decode(reader, reader.uint32());
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

  fromJSON(object: any): UpdateActuatorInfoRequest {
    return {
      actuator: isSet(object.actuator) ? ActuatorInfo.fromJSON(object.actuator) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateActuatorInfoRequest): unknown {
    const obj: any = {};
    if (message.actuator !== undefined) {
      obj.actuator = ActuatorInfo.toJSON(message.actuator);
    }
    if (message.updateMask !== undefined) {
      obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask));
    }
    return obj;
  },

  create(base?: DeepPartial<UpdateActuatorInfoRequest>): UpdateActuatorInfoRequest {
    return UpdateActuatorInfoRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UpdateActuatorInfoRequest>): UpdateActuatorInfoRequest {
    const message = createBaseUpdateActuatorInfoRequest();
    message.actuator = (object.actuator !== undefined && object.actuator !== null)
      ? ActuatorInfo.fromPartial(object.actuator)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
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
    debug: false,
    lsp: false,
    preUpdateBackup: false,
    iamGuard: false,
    unlicensedFeatures: [],
    disallowPasswordSignin: false,
    passwordRestriction: undefined,
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
    if (message.debug === true) {
      writer.uint32(120).bool(message.debug);
    }
    if (message.lsp === true) {
      writer.uint32(128).bool(message.lsp);
    }
    if (message.preUpdateBackup === true) {
      writer.uint32(136).bool(message.preUpdateBackup);
    }
    if (message.iamGuard === true) {
      writer.uint32(144).bool(message.iamGuard);
    }
    for (const v of message.unlicensedFeatures) {
      writer.uint32(154).string(v!);
    }
    if (message.disallowPasswordSignin === true) {
      writer.uint32(160).bool(message.disallowPasswordSignin);
    }
    if (message.passwordRestriction !== undefined) {
      PasswordRestrictionSetting.encode(message.passwordRestriction, writer.uint32(170).fork()).ldelim();
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
        case 15:
          if (tag !== 120) {
            break;
          }

          message.debug = reader.bool();
          continue;
        case 16:
          if (tag !== 128) {
            break;
          }

          message.lsp = reader.bool();
          continue;
        case 17:
          if (tag !== 136) {
            break;
          }

          message.preUpdateBackup = reader.bool();
          continue;
        case 18:
          if (tag !== 144) {
            break;
          }

          message.iamGuard = reader.bool();
          continue;
        case 19:
          if (tag !== 154) {
            break;
          }

          message.unlicensedFeatures.push(reader.string());
          continue;
        case 20:
          if (tag !== 160) {
            break;
          }

          message.disallowPasswordSignin = reader.bool();
          continue;
        case 21:
          if (tag !== 170) {
            break;
          }

          message.passwordRestriction = PasswordRestrictionSetting.decode(reader, reader.uint32());
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
      version: isSet(object.version) ? globalThis.String(object.version) : "",
      gitCommit: isSet(object.gitCommit) ? globalThis.String(object.gitCommit) : "",
      readonly: isSet(object.readonly) ? globalThis.Boolean(object.readonly) : false,
      saas: isSet(object.saas) ? globalThis.Boolean(object.saas) : false,
      demoName: isSet(object.demoName) ? globalThis.String(object.demoName) : "",
      host: isSet(object.host) ? globalThis.String(object.host) : "",
      port: isSet(object.port) ? globalThis.String(object.port) : "",
      externalUrl: isSet(object.externalUrl) ? globalThis.String(object.externalUrl) : "",
      needAdminSetup: isSet(object.needAdminSetup) ? globalThis.Boolean(object.needAdminSetup) : false,
      disallowSignup: isSet(object.disallowSignup) ? globalThis.Boolean(object.disallowSignup) : false,
      lastActiveTime: isSet(object.lastActiveTime) ? fromJsonTimestamp(object.lastActiveTime) : undefined,
      require2fa: isSet(object.require2fa) ? globalThis.Boolean(object.require2fa) : false,
      workspaceId: isSet(object.workspaceId) ? globalThis.String(object.workspaceId) : "",
      gitopsWebhookUrl: isSet(object.gitopsWebhookUrl) ? globalThis.String(object.gitopsWebhookUrl) : "",
      debug: isSet(object.debug) ? globalThis.Boolean(object.debug) : false,
      lsp: isSet(object.lsp) ? globalThis.Boolean(object.lsp) : false,
      preUpdateBackup: isSet(object.preUpdateBackup) ? globalThis.Boolean(object.preUpdateBackup) : false,
      iamGuard: isSet(object.iamGuard) ? globalThis.Boolean(object.iamGuard) : false,
      unlicensedFeatures: globalThis.Array.isArray(object?.unlicensedFeatures)
        ? object.unlicensedFeatures.map((e: any) => globalThis.String(e))
        : [],
      disallowPasswordSignin: isSet(object.disallowPasswordSignin)
        ? globalThis.Boolean(object.disallowPasswordSignin)
        : false,
      passwordRestriction: isSet(object.passwordRestriction)
        ? PasswordRestrictionSetting.fromJSON(object.passwordRestriction)
        : undefined,
    };
  },

  toJSON(message: ActuatorInfo): unknown {
    const obj: any = {};
    if (message.version !== "") {
      obj.version = message.version;
    }
    if (message.gitCommit !== "") {
      obj.gitCommit = message.gitCommit;
    }
    if (message.readonly === true) {
      obj.readonly = message.readonly;
    }
    if (message.saas === true) {
      obj.saas = message.saas;
    }
    if (message.demoName !== "") {
      obj.demoName = message.demoName;
    }
    if (message.host !== "") {
      obj.host = message.host;
    }
    if (message.port !== "") {
      obj.port = message.port;
    }
    if (message.externalUrl !== "") {
      obj.externalUrl = message.externalUrl;
    }
    if (message.needAdminSetup === true) {
      obj.needAdminSetup = message.needAdminSetup;
    }
    if (message.disallowSignup === true) {
      obj.disallowSignup = message.disallowSignup;
    }
    if (message.lastActiveTime !== undefined) {
      obj.lastActiveTime = message.lastActiveTime.toISOString();
    }
    if (message.require2fa === true) {
      obj.require2fa = message.require2fa;
    }
    if (message.workspaceId !== "") {
      obj.workspaceId = message.workspaceId;
    }
    if (message.gitopsWebhookUrl !== "") {
      obj.gitopsWebhookUrl = message.gitopsWebhookUrl;
    }
    if (message.debug === true) {
      obj.debug = message.debug;
    }
    if (message.lsp === true) {
      obj.lsp = message.lsp;
    }
    if (message.preUpdateBackup === true) {
      obj.preUpdateBackup = message.preUpdateBackup;
    }
    if (message.iamGuard === true) {
      obj.iamGuard = message.iamGuard;
    }
    if (message.unlicensedFeatures?.length) {
      obj.unlicensedFeatures = message.unlicensedFeatures;
    }
    if (message.disallowPasswordSignin === true) {
      obj.disallowPasswordSignin = message.disallowPasswordSignin;
    }
    if (message.passwordRestriction !== undefined) {
      obj.passwordRestriction = PasswordRestrictionSetting.toJSON(message.passwordRestriction);
    }
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
    message.debug = object.debug ?? false;
    message.lsp = object.lsp ?? false;
    message.preUpdateBackup = object.preUpdateBackup ?? false;
    message.iamGuard = object.iamGuard ?? false;
    message.unlicensedFeatures = object.unlicensedFeatures?.map((e) => e) || [];
    message.disallowPasswordSignin = object.disallowPasswordSignin ?? false;
    message.passwordRestriction = (object.passwordRestriction !== undefined && object.passwordRestriction !== null)
      ? PasswordRestrictionSetting.fromPartial(object.passwordRestriction)
      : undefined;
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
          800000: [new Uint8Array([1])],
          578365826: [
            new Uint8Array([19, 18, 17, 47, 118, 49, 47, 97, 99, 116, 117, 97, 116, 111, 114, 47, 105, 110, 102, 111]),
          ],
        },
      },
    },
    updateActuatorInfo: {
      name: "UpdateActuatorInfo",
      requestType: UpdateActuatorInfoRequest,
      requestStream: false,
      responseType: ActuatorInfo,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([
              20,
              97,
              99,
              116,
              117,
              97,
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
          800010: [new Uint8Array([15, 98, 98, 46, 115, 101, 116, 116, 105, 110, 103, 115, 46, 115, 101, 116])],
          800016: [new Uint8Array([1])],
          578365826: [
            new Uint8Array([
              29,
              58,
              8,
              97,
              99,
              116,
              117,
              97,
              116,
              111,
              114,
              50,
              17,
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
              105,
              110,
              102,
              111,
            ]),
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
          800000: [new Uint8Array([1])],
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
    getResourcePackage: {
      name: "GetResourcePackage",
      requestType: GetResourcePackageRequest,
      requestStream: false,
      responseType: ResourcePackage,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          800000: [new Uint8Array([1])],
          578365826: [
            new Uint8Array([
              24,
              18,
              22,
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
              114,
              101,
              115,
              111,
              117,
              114,
              99,
              101,
              115,
            ]),
          ],
        },
      },
    },
  },
} as const;

function bytesFromBase64(b64: string): Uint8Array {
  if (globalThis.Buffer) {
    return Uint8Array.from(globalThis.Buffer.from(b64, "base64"));
  } else {
    const bin = globalThis.atob(b64);
    const arr = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; ++i) {
      arr[i] = bin.charCodeAt(i);
    }
    return arr;
  }
}

function base64FromBytes(arr: Uint8Array): string {
  if (globalThis.Buffer) {
    return globalThis.Buffer.from(arr).toString("base64");
  } else {
    const bin: string[] = [];
    arr.forEach((byte) => {
      bin.push(globalThis.String.fromCharCode(byte));
    });
    return globalThis.btoa(bin.join(""));
  }
}

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
