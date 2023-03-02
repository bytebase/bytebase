/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.v1";

export interface GetActuatorInfoRequest {
}

/**
 * ServerInfo is the API message for server info.
 * Actuator concept is similar to the Spring Boot Actuator.
 */
export interface ActuatorInfo {
  version: string;
  gitCommit: string;
  readonly: boolean;
  saas: boolean;
  demoName: string;
  host: string;
  port: string;
  externalUrl: string;
  needAdminSetup: boolean;
  disallowSignup: boolean;
}

function createBaseGetActuatorInfoRequest(): GetActuatorInfoRequest {
  return {};
}

export const GetActuatorInfoRequest = {
  encode(_: GetActuatorInfoRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetActuatorInfoRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetActuatorInfoRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
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

  fromPartial(_: DeepPartial<GetActuatorInfoRequest>): GetActuatorInfoRequest {
    const message = createBaseGetActuatorInfoRequest();
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
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ActuatorInfo {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseActuatorInfo();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.version = reader.string();
          break;
        case 2:
          message.gitCommit = reader.string();
          break;
        case 3:
          message.readonly = reader.bool();
          break;
        case 4:
          message.saas = reader.bool();
          break;
        case 5:
          message.demoName = reader.string();
          break;
        case 6:
          message.host = reader.string();
          break;
        case 7:
          message.port = reader.string();
          break;
        case 8:
          message.externalUrl = reader.string();
          break;
        case 9:
          message.needAdminSetup = reader.bool();
          break;
        case 10:
          message.disallowSignup = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
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
    return obj;
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
      options: {},
    },
  },
} as const;

export interface ActuatorServiceImplementation<CallContextExt = {}> {
  getActuatorInfo(
    request: GetActuatorInfoRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ActuatorInfo>>;
}

export interface ActuatorServiceClient<CallOptionsExt = {}> {
  getActuatorInfo(
    request: DeepPartial<GetActuatorInfoRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ActuatorInfo>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
