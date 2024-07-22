/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export interface AnomalyConnectionPayload {
  /** Connection failure detail */
  detail: string;
}

export interface AnomalyDatabaseSchemaDriftPayload {
  /** The schema version corresponds to the expected schema */
  version: string;
  /** The expected latest schema stored in the migration history table */
  expect: string;
  /** The actual schema dumped from the database */
  actual: string;
}

function createBaseAnomalyConnectionPayload(): AnomalyConnectionPayload {
  return { detail: "" };
}

export const AnomalyConnectionPayload = {
  encode(message: AnomalyConnectionPayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.detail !== "") {
      writer.uint32(10).string(message.detail);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): AnomalyConnectionPayload {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAnomalyConnectionPayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.detail = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): AnomalyConnectionPayload {
    return { detail: isSet(object.detail) ? globalThis.String(object.detail) : "" };
  },

  toJSON(message: AnomalyConnectionPayload): unknown {
    const obj: any = {};
    if (message.detail !== "") {
      obj.detail = message.detail;
    }
    return obj;
  },

  create(base?: DeepPartial<AnomalyConnectionPayload>): AnomalyConnectionPayload {
    return AnomalyConnectionPayload.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<AnomalyConnectionPayload>): AnomalyConnectionPayload {
    const message = createBaseAnomalyConnectionPayload();
    message.detail = object.detail ?? "";
    return message;
  },
};

function createBaseAnomalyDatabaseSchemaDriftPayload(): AnomalyDatabaseSchemaDriftPayload {
  return { version: "", expect: "", actual: "" };
}

export const AnomalyDatabaseSchemaDriftPayload = {
  encode(message: AnomalyDatabaseSchemaDriftPayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.version !== "") {
      writer.uint32(10).string(message.version);
    }
    if (message.expect !== "") {
      writer.uint32(18).string(message.expect);
    }
    if (message.actual !== "") {
      writer.uint32(26).string(message.actual);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): AnomalyDatabaseSchemaDriftPayload {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAnomalyDatabaseSchemaDriftPayload();
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

          message.expect = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.actual = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): AnomalyDatabaseSchemaDriftPayload {
    return {
      version: isSet(object.version) ? globalThis.String(object.version) : "",
      expect: isSet(object.expect) ? globalThis.String(object.expect) : "",
      actual: isSet(object.actual) ? globalThis.String(object.actual) : "",
    };
  },

  toJSON(message: AnomalyDatabaseSchemaDriftPayload): unknown {
    const obj: any = {};
    if (message.version !== "") {
      obj.version = message.version;
    }
    if (message.expect !== "") {
      obj.expect = message.expect;
    }
    if (message.actual !== "") {
      obj.actual = message.actual;
    }
    return obj;
  },

  create(base?: DeepPartial<AnomalyDatabaseSchemaDriftPayload>): AnomalyDatabaseSchemaDriftPayload {
    return AnomalyDatabaseSchemaDriftPayload.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<AnomalyDatabaseSchemaDriftPayload>): AnomalyDatabaseSchemaDriftPayload {
    const message = createBaseAnomalyDatabaseSchemaDriftPayload();
    message.version = object.version ?? "";
    message.expect = object.expect ?? "";
    message.actual = object.actual ?? "";
    return message;
  },
};

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
