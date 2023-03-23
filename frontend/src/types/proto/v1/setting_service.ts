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
  return { setting: undefined };
}

export const SetSettingRequest = {
  encode(message: SetSettingRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.setting !== undefined) {
      Setting.encode(message.setting, writer.uint32(10).fork()).ldelim();
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
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SetSettingRequest {
    return { setting: isSet(object.setting) ? Setting.fromJSON(object.setting) : undefined };
  },

  toJSON(message: SetSettingRequest): unknown {
    const obj: any = {};
    message.setting !== undefined && (obj.setting = message.setting ? Setting.toJSON(message.setting) : undefined);
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
  return { stringValue: undefined };
}

export const Value = {
  encode(message: Value, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.stringValue !== undefined) {
      writer.uint32(10).string(message.stringValue);
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
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Value {
    return { stringValue: isSet(object.stringValue) ? String(object.stringValue) : undefined };
  },

  toJSON(message: Value): unknown {
    const obj: any = {};
    message.stringValue !== undefined && (obj.stringValue = message.stringValue);
    return obj;
  },

  create(base?: DeepPartial<Value>): Value {
    return Value.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Value>): Value {
    const message = createBaseValue();
    message.stringValue = object.stringValue ?? undefined;
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
