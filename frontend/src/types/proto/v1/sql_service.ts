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
  dumpedSDL: string;
  /** The user-defined SDL schema. This schema will be checked for correctness and normalized. */
  userSDL: string;
}

export interface PrettyResponse {
  /** The pretty-formatted version of dumpedSDL. */
  prettyDumpedSDL: string;
  /** The user-defined SDL schema after normalizing. */
  prettyUserSDL: string;
}

function createBasePrettyRequest(): PrettyRequest {
  return { engine: 0, dumpedSDL: "", userSDL: "" };
}

export const PrettyRequest = {
  encode(message: PrettyRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.engine !== 0) {
      writer.uint32(8).int32(message.engine);
    }
    if (message.dumpedSDL !== "") {
      writer.uint32(18).string(message.dumpedSDL);
    }
    if (message.userSDL !== "") {
      writer.uint32(26).string(message.userSDL);
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
          message.dumpedSDL = reader.string();
          break;
        case 3:
          message.userSDL = reader.string();
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
      dumpedSDL: isSet(object.dumpedSDL) ? String(object.dumpedSDL) : "",
      userSDL: isSet(object.userSDL) ? String(object.userSDL) : "",
    };
  },

  toJSON(message: PrettyRequest): unknown {
    const obj: any = {};
    message.engine !== undefined && (obj.engine = engineToJSON(message.engine));
    message.dumpedSDL !== undefined && (obj.dumpedSDL = message.dumpedSDL);
    message.userSDL !== undefined && (obj.userSDL = message.userSDL);
    return obj;
  },

  fromPartial(object: DeepPartial<PrettyRequest>): PrettyRequest {
    const message = createBasePrettyRequest();
    message.engine = object.engine ?? 0;
    message.dumpedSDL = object.dumpedSDL ?? "";
    message.userSDL = object.userSDL ?? "";
    return message;
  },
};

function createBasePrettyResponse(): PrettyResponse {
  return { prettyDumpedSDL: "", prettyUserSDL: "" };
}

export const PrettyResponse = {
  encode(message: PrettyResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.prettyDumpedSDL !== "") {
      writer.uint32(10).string(message.prettyDumpedSDL);
    }
    if (message.prettyUserSDL !== "") {
      writer.uint32(18).string(message.prettyUserSDL);
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
          message.prettyDumpedSDL = reader.string();
          break;
        case 2:
          message.prettyUserSDL = reader.string();
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
      prettyDumpedSDL: isSet(object.prettyDumpedSDL) ? String(object.prettyDumpedSDL) : "",
      prettyUserSDL: isSet(object.prettyUserSDL) ? String(object.prettyUserSDL) : "",
    };
  },

  toJSON(message: PrettyResponse): unknown {
    const obj: any = {};
    message.prettyDumpedSDL !== undefined && (obj.prettyDumpedSDL = message.prettyDumpedSDL);
    message.prettyUserSDL !== undefined && (obj.prettyUserSDL = message.prettyUserSDL);
    return obj;
  },

  fromPartial(object: DeepPartial<PrettyResponse>): PrettyResponse {
    const message = createBasePrettyResponse();
    message.prettyDumpedSDL = object.prettyDumpedSDL ?? "";
    message.prettyUserSDL = object.prettyUserSDL ?? "";
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
