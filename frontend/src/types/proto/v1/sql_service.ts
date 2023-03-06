/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Engine, engineFromJSON, engineToJSON } from "./common";

export const protobufPackage = "bytebase.v1";

export interface PrettyRequest {
  engine: Engine;
  /**
   * The SDL format SQL schema information that was dumped from a database engine.
   * This information will be sorted to match the order of statements in the userSchema.
   */
  currentSchema: string;
  /** The expected SDL schema. This schema will be checked for correctness and normalized. */
  expectedSchema: string;
}

export interface PrettyResponse {
  /** The pretty-formatted version of current schema. */
  currentSchema: string;
  /** The expected SDL schema after normalizing. */
  expectedSchema: string;
}

function createBasePrettyRequest(): PrettyRequest {
  return { engine: 0, currentSchema: "", expectedSchema: "" };
}

export const PrettyRequest = {
  encode(message: PrettyRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.engine !== 0) {
      writer.uint32(8).int32(message.engine);
    }
    if (message.currentSchema !== "") {
      writer.uint32(18).string(message.currentSchema);
    }
    if (message.expectedSchema !== "") {
      writer.uint32(26).string(message.expectedSchema);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PrettyRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePrettyRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.engine = reader.int32() as any;
          break;
        case 2:
          message.currentSchema = reader.string();
          break;
        case 3:
          message.expectedSchema = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PrettyRequest {
    return {
      engine: isSet(object.engine) ? engineFromJSON(object.engine) : 0,
      currentSchema: isSet(object.currentSchema) ? String(object.currentSchema) : "",
      expectedSchema: isSet(object.expectedSchema) ? String(object.expectedSchema) : "",
    };
  },

  toJSON(message: PrettyRequest): unknown {
    const obj: any = {};
    message.engine !== undefined && (obj.engine = engineToJSON(message.engine));
    message.currentSchema !== undefined && (obj.currentSchema = message.currentSchema);
    message.expectedSchema !== undefined && (obj.expectedSchema = message.expectedSchema);
    return obj;
  },

  fromPartial(object: DeepPartial<PrettyRequest>): PrettyRequest {
    const message = createBasePrettyRequest();
    message.engine = object.engine ?? 0;
    message.currentSchema = object.currentSchema ?? "";
    message.expectedSchema = object.expectedSchema ?? "";
    return message;
  },
};

function createBasePrettyResponse(): PrettyResponse {
  return { currentSchema: "", expectedSchema: "" };
}

export const PrettyResponse = {
  encode(message: PrettyResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.currentSchema !== "") {
      writer.uint32(10).string(message.currentSchema);
    }
    if (message.expectedSchema !== "") {
      writer.uint32(18).string(message.expectedSchema);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PrettyResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePrettyResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.currentSchema = reader.string();
          break;
        case 2:
          message.expectedSchema = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): PrettyResponse {
    return {
      currentSchema: isSet(object.currentSchema) ? String(object.currentSchema) : "",
      expectedSchema: isSet(object.expectedSchema) ? String(object.expectedSchema) : "",
    };
  },

  toJSON(message: PrettyResponse): unknown {
    const obj: any = {};
    message.currentSchema !== undefined && (obj.currentSchema = message.currentSchema);
    message.expectedSchema !== undefined && (obj.expectedSchema = message.expectedSchema);
    return obj;
  },

  fromPartial(object: DeepPartial<PrettyResponse>): PrettyResponse {
    const message = createBasePrettyResponse();
    message.currentSchema = object.currentSchema ?? "";
    message.expectedSchema = object.expectedSchema ?? "";
    return message;
  },
};

export type SQLServiceDefinition = typeof SQLServiceDefinition;
export const SQLServiceDefinition = {
  name: "SQLService",
  fullName: "bytebase.v1.SQLService",
  methods: {
    pretty: {
      name: "Pretty",
      requestType: PrettyRequest,
      requestStream: false,
      responseType: PrettyResponse,
      responseStream: false,
      options: {},
    },
  },
} as const;

export interface SQLServiceImplementation<CallContextExt = {}> {
  pretty(request: PrettyRequest, context: CallContext & CallContextExt): Promise<DeepPartial<PrettyResponse>>;
}

export interface SQLServiceClient<CallOptionsExt = {}> {
  pretty(request: DeepPartial<PrettyRequest>, options?: CallOptions & CallOptionsExt): Promise<PrettyResponse>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
