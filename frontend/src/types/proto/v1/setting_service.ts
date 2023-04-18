/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.v1";

/** The request message for getting a setting. */
export interface GetSettingRequest {
  /** The resource name of the setting. */
  name: string;
}

/** The response message for getting a setting. */
export interface GetSettingResponse {
  setting?: Setting;
}

/** The request message for updating a setting. */
export interface SetSettingRequest {
  /** The setting to update. */
  setting?: Setting;
  /**
   * validate_only is a flag to indicate whether to validate the setting value,
   * server would not persist the setting value if it is true.
   */
  validateOnly: boolean;
}

/** The schema of setting. */
export interface Setting {
  /**
   * The resource name of the setting. Must be one of the following forms:
   *
   * - `setting/{setting_name}`
   * For example, "settings/bb.branding.logo"
   */
  name: string;
  /** The value of the setting. */
  value?: Value;
}

/** The data in setting value. */
export interface Value {
  /** Defines this value as being a string value. */
  stringValue?: string | undefined;
  smtpMailDeliverySettingValue?: SMTPMailDeliverySettingValue | undefined;
}

export interface SMTPMailDeliverySettingValue {
  /** The SMTP server address. */
  server: string;
  /** The SMTP server port. */
  port: number;
  /** The SMTP server encryption. */
  encryption: SMTPMailDeliverySettingValue_Encryption;
  /**
   * The CA, KEY, and CERT for the SMTP server.
   * Not used.
   */
  ca?: string | undefined;
  key?: string | undefined;
  cert?: string | undefined;
  authentication: SMTPMailDeliverySettingValue_Authentication;
  username: string;
  /** If not specified, server will use the existed password. */
  password?:
    | string
    | undefined;
  /** The sender email address. */
  from: string;
  /** The recipient email address, used with validate_only to send test email. */
  to: string;
}

/** We support three types of SMTP encryption: NONE, STARTTLS, and SSL/TLS. */
export enum SMTPMailDeliverySettingValue_Encryption {
  ENCRYPTION_UNSPECIFIED = 0,
  ENCRYPTION_NONE = 1,
  ENCRYPTION_STARTTLS = 2,
  ENCRYPTION_SSL_TLS = 3,
  UNRECOGNIZED = -1,
}

export function sMTPMailDeliverySettingValue_EncryptionFromJSON(object: any): SMTPMailDeliverySettingValue_Encryption {
  switch (object) {
    case 0:
    case "ENCRYPTION_UNSPECIFIED":
      return SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_UNSPECIFIED;
    case 1:
    case "ENCRYPTION_NONE":
      return SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_NONE;
    case 2:
    case "ENCRYPTION_STARTTLS":
      return SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_STARTTLS;
    case 3:
    case "ENCRYPTION_SSL_TLS":
      return SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_SSL_TLS;
    case -1:
    case "UNRECOGNIZED":
    default:
      return SMTPMailDeliverySettingValue_Encryption.UNRECOGNIZED;
  }
}

export function sMTPMailDeliverySettingValue_EncryptionToJSON(object: SMTPMailDeliverySettingValue_Encryption): string {
  switch (object) {
    case SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_UNSPECIFIED:
      return "ENCRYPTION_UNSPECIFIED";
    case SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_NONE:
      return "ENCRYPTION_NONE";
    case SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_STARTTLS:
      return "ENCRYPTION_STARTTLS";
    case SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_SSL_TLS:
      return "ENCRYPTION_SSL_TLS";
    case SMTPMailDeliverySettingValue_Encryption.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

/** We support four types of SMTP authentication: NONE, PLAIN, LOGIN, and CRAM-MD5. */
export enum SMTPMailDeliverySettingValue_Authentication {
  AUTHENTICATION_UNSPECIFIED = 0,
  AUTHENTICATION_NONE = 1,
  AUTHENTICATION_PLAIN = 2,
  AUTHENTICATION_LOGIN = 3,
  AUTHENTICATION_CRAM_MD5 = 4,
  UNRECOGNIZED = -1,
}

export function sMTPMailDeliverySettingValue_AuthenticationFromJSON(
  object: any,
): SMTPMailDeliverySettingValue_Authentication {
  switch (object) {
    case 0:
    case "AUTHENTICATION_UNSPECIFIED":
      return SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_UNSPECIFIED;
    case 1:
    case "AUTHENTICATION_NONE":
      return SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_NONE;
    case 2:
    case "AUTHENTICATION_PLAIN":
      return SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_PLAIN;
    case 3:
    case "AUTHENTICATION_LOGIN":
      return SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_LOGIN;
    case 4:
    case "AUTHENTICATION_CRAM_MD5":
      return SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_CRAM_MD5;
    case -1:
    case "UNRECOGNIZED":
    default:
      return SMTPMailDeliverySettingValue_Authentication.UNRECOGNIZED;
  }
}

export function sMTPMailDeliverySettingValue_AuthenticationToJSON(
  object: SMTPMailDeliverySettingValue_Authentication,
): string {
  switch (object) {
    case SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_UNSPECIFIED:
      return "AUTHENTICATION_UNSPECIFIED";
    case SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_NONE:
      return "AUTHENTICATION_NONE";
    case SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_PLAIN:
      return "AUTHENTICATION_PLAIN";
    case SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_LOGIN:
      return "AUTHENTICATION_LOGIN";
    case SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_CRAM_MD5:
      return "AUTHENTICATION_CRAM_MD5";
    case SMTPMailDeliverySettingValue_Authentication.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

function createBaseGetSettingRequest(): GetSettingRequest {
  return { name: "" };
}

export const GetSettingRequest = {
  encode(message: GetSettingRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetSettingRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetSettingRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GetSettingRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetSettingRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<GetSettingRequest>): GetSettingRequest {
    return GetSettingRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<GetSettingRequest>): GetSettingRequest {
    const message = createBaseGetSettingRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseGetSettingResponse(): GetSettingResponse {
  return { setting: undefined };
}

export const GetSettingResponse = {
  encode(message: GetSettingResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.setting !== undefined) {
      Setting.encode(message.setting, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetSettingResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetSettingResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.setting = Setting.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GetSettingResponse {
    return { setting: isSet(object.setting) ? Setting.fromJSON(object.setting) : undefined };
  },

  toJSON(message: GetSettingResponse): unknown {
    const obj: any = {};
    message.setting !== undefined && (obj.setting = message.setting ? Setting.toJSON(message.setting) : undefined);
    return obj;
  },

  create(base?: DeepPartial<GetSettingResponse>): GetSettingResponse {
    return GetSettingResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<GetSettingResponse>): GetSettingResponse {
    const message = createBaseGetSettingResponse();
    message.setting = (object.setting !== undefined && object.setting !== null)
      ? Setting.fromPartial(object.setting)
      : undefined;
    return message;
  },
};

function createBaseSetSettingRequest(): SetSettingRequest {
  return { setting: undefined, validateOnly: false };
}

export const SetSettingRequest = {
  encode(message: SetSettingRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.setting !== undefined) {
      Setting.encode(message.setting, writer.uint32(10).fork()).ldelim();
    }
    if (message.validateOnly === true) {
      writer.uint32(16).bool(message.validateOnly);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SetSettingRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSetSettingRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.setting = Setting.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag != 16) {
            break;
          }

          message.validateOnly = reader.bool();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SetSettingRequest {
    return {
      setting: isSet(object.setting) ? Setting.fromJSON(object.setting) : undefined,
      validateOnly: isSet(object.validateOnly) ? Boolean(object.validateOnly) : false,
    };
  },

  toJSON(message: SetSettingRequest): unknown {
    const obj: any = {};
    message.setting !== undefined && (obj.setting = message.setting ? Setting.toJSON(message.setting) : undefined);
    message.validateOnly !== undefined && (obj.validateOnly = message.validateOnly);
    return obj;
  },

  create(base?: DeepPartial<SetSettingRequest>): SetSettingRequest {
    return SetSettingRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SetSettingRequest>): SetSettingRequest {
    const message = createBaseSetSettingRequest();
    message.setting = (object.setting !== undefined && object.setting !== null)
      ? Setting.fromPartial(object.setting)
      : undefined;
    message.validateOnly = object.validateOnly ?? false;
    return message;
  },
};

function createBaseSetting(): Setting {
  return { name: "", value: undefined };
}

export const Setting = {
  encode(message: Setting, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.value !== undefined) {
      Value.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Setting {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSetting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.value = Value.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Setting {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      value: isSet(object.value) ? Value.fromJSON(object.value) : undefined,
    };
  },

  toJSON(message: Setting): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.value !== undefined && (obj.value = message.value ? Value.toJSON(message.value) : undefined);
    return obj;
  },

  create(base?: DeepPartial<Setting>): Setting {
    return Setting.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Setting>): Setting {
    const message = createBaseSetting();
    message.name = object.name ?? "";
    message.value = (object.value !== undefined && object.value !== null) ? Value.fromPartial(object.value) : undefined;
    return message;
  },
};

function createBaseValue(): Value {
  return { stringValue: undefined, smtpMailDeliverySettingValue: undefined };
}

export const Value = {
  encode(message: Value, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.stringValue !== undefined) {
      writer.uint32(10).string(message.stringValue);
    }
    if (message.smtpMailDeliverySettingValue !== undefined) {
      SMTPMailDeliverySettingValue.encode(message.smtpMailDeliverySettingValue, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Value {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseValue();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.stringValue = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.smtpMailDeliverySettingValue = SMTPMailDeliverySettingValue.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Value {
    return {
      stringValue: isSet(object.stringValue) ? String(object.stringValue) : undefined,
      smtpMailDeliverySettingValue: isSet(object.smtpMailDeliverySettingValue)
        ? SMTPMailDeliverySettingValue.fromJSON(object.smtpMailDeliverySettingValue)
        : undefined,
    };
  },

  toJSON(message: Value): unknown {
    const obj: any = {};
    message.stringValue !== undefined && (obj.stringValue = message.stringValue);
    message.smtpMailDeliverySettingValue !== undefined &&
      (obj.smtpMailDeliverySettingValue = message.smtpMailDeliverySettingValue
        ? SMTPMailDeliverySettingValue.toJSON(message.smtpMailDeliverySettingValue)
        : undefined);
    return obj;
  },

  create(base?: DeepPartial<Value>): Value {
    return Value.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Value>): Value {
    const message = createBaseValue();
    message.stringValue = object.stringValue ?? undefined;
    message.smtpMailDeliverySettingValue =
      (object.smtpMailDeliverySettingValue !== undefined && object.smtpMailDeliverySettingValue !== null)
        ? SMTPMailDeliverySettingValue.fromPartial(object.smtpMailDeliverySettingValue)
        : undefined;
    return message;
  },
};

function createBaseSMTPMailDeliverySettingValue(): SMTPMailDeliverySettingValue {
  return {
    server: "",
    port: 0,
    encryption: 0,
    ca: undefined,
    key: undefined,
    cert: undefined,
    authentication: 0,
    username: "",
    password: undefined,
    from: "",
    to: "",
  };
}

export const SMTPMailDeliverySettingValue = {
  encode(message: SMTPMailDeliverySettingValue, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.server !== "") {
      writer.uint32(10).string(message.server);
    }
    if (message.port !== 0) {
      writer.uint32(16).int32(message.port);
    }
    if (message.encryption !== 0) {
      writer.uint32(24).int32(message.encryption);
    }
    if (message.ca !== undefined) {
      writer.uint32(34).string(message.ca);
    }
    if (message.key !== undefined) {
      writer.uint32(42).string(message.key);
    }
    if (message.cert !== undefined) {
      writer.uint32(50).string(message.cert);
    }
    if (message.authentication !== 0) {
      writer.uint32(56).int32(message.authentication);
    }
    if (message.username !== "") {
      writer.uint32(66).string(message.username);
    }
    if (message.password !== undefined) {
      writer.uint32(74).string(message.password);
    }
    if (message.from !== "") {
      writer.uint32(82).string(message.from);
    }
    if (message.to !== "") {
      writer.uint32(90).string(message.to);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SMTPMailDeliverySettingValue {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSMTPMailDeliverySettingValue();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.server = reader.string();
          continue;
        case 2:
          if (tag != 16) {
            break;
          }

          message.port = reader.int32();
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.encryption = reader.int32() as any;
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.ca = reader.string();
          continue;
        case 5:
          if (tag != 42) {
            break;
          }

          message.key = reader.string();
          continue;
        case 6:
          if (tag != 50) {
            break;
          }

          message.cert = reader.string();
          continue;
        case 7:
          if (tag != 56) {
            break;
          }

          message.authentication = reader.int32() as any;
          continue;
        case 8:
          if (tag != 66) {
            break;
          }

          message.username = reader.string();
          continue;
        case 9:
          if (tag != 74) {
            break;
          }

          message.password = reader.string();
          continue;
        case 10:
          if (tag != 82) {
            break;
          }

          message.from = reader.string();
          continue;
        case 11:
          if (tag != 90) {
            break;
          }

          message.to = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SMTPMailDeliverySettingValue {
    return {
      server: isSet(object.server) ? String(object.server) : "",
      port: isSet(object.port) ? Number(object.port) : 0,
      encryption: isSet(object.encryption) ? sMTPMailDeliverySettingValue_EncryptionFromJSON(object.encryption) : 0,
      ca: isSet(object.ca) ? String(object.ca) : undefined,
      key: isSet(object.key) ? String(object.key) : undefined,
      cert: isSet(object.cert) ? String(object.cert) : undefined,
      authentication: isSet(object.authentication)
        ? sMTPMailDeliverySettingValue_AuthenticationFromJSON(object.authentication)
        : 0,
      username: isSet(object.username) ? String(object.username) : "",
      password: isSet(object.password) ? String(object.password) : undefined,
      from: isSet(object.from) ? String(object.from) : "",
      to: isSet(object.to) ? String(object.to) : "",
    };
  },

  toJSON(message: SMTPMailDeliverySettingValue): unknown {
    const obj: any = {};
    message.server !== undefined && (obj.server = message.server);
    message.port !== undefined && (obj.port = Math.round(message.port));
    message.encryption !== undefined &&
      (obj.encryption = sMTPMailDeliverySettingValue_EncryptionToJSON(message.encryption));
    message.ca !== undefined && (obj.ca = message.ca);
    message.key !== undefined && (obj.key = message.key);
    message.cert !== undefined && (obj.cert = message.cert);
    message.authentication !== undefined &&
      (obj.authentication = sMTPMailDeliverySettingValue_AuthenticationToJSON(message.authentication));
    message.username !== undefined && (obj.username = message.username);
    message.password !== undefined && (obj.password = message.password);
    message.from !== undefined && (obj.from = message.from);
    message.to !== undefined && (obj.to = message.to);
    return obj;
  },

  create(base?: DeepPartial<SMTPMailDeliverySettingValue>): SMTPMailDeliverySettingValue {
    return SMTPMailDeliverySettingValue.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SMTPMailDeliverySettingValue>): SMTPMailDeliverySettingValue {
    const message = createBaseSMTPMailDeliverySettingValue();
    message.server = object.server ?? "";
    message.port = object.port ?? 0;
    message.encryption = object.encryption ?? 0;
    message.ca = object.ca ?? undefined;
    message.key = object.key ?? undefined;
    message.cert = object.cert ?? undefined;
    message.authentication = object.authentication ?? 0;
    message.username = object.username ?? "";
    message.password = object.password ?? undefined;
    message.from = object.from ?? "";
    message.to = object.to ?? "";
    return message;
  },
};

export type SettingServiceDefinition = typeof SettingServiceDefinition;
export const SettingServiceDefinition = {
  name: "SettingService",
  fullName: "bytebase.v1.SettingService",
  methods: {
    getSetting: {
      name: "GetSetting",
      requestType: GetSettingRequest,
      requestStream: false,
      responseType: Setting,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              23,
              18,
              21,
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
              115,
              101,
              116,
              116,
              105,
              110,
              103,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    setSetting: {
      name: "SetSetting",
      requestType: SetSettingRequest,
      requestStream: false,
      responseType: Setting,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              40,
              58,
              7,
              115,
              101,
              116,
              116,
              105,
              110,
              103,
              50,
              29,
              47,
              118,
              49,
              47,
              123,
              115,
              101,
              116,
              116,
              105,
              110,
              103,
              46,
              110,
              97,
              109,
              101,
              61,
              115,
              101,
              116,
              116,
              105,
              110,
              103,
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

export interface SettingServiceImplementation<CallContextExt = {}> {
  getSetting(request: GetSettingRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Setting>>;
  setSetting(request: SetSettingRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Setting>>;
}

export interface SettingServiceClient<CallOptionsExt = {}> {
  getSetting(request: DeepPartial<GetSettingRequest>, options?: CallOptions & CallOptionsExt): Promise<Setting>;
  setSetting(request: DeepPartial<SetSettingRequest>, options?: CallOptions & CallOptionsExt): Promise<Setting>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
