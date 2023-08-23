/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { ParsedExpr } from "../google/api/expr/v1alpha1/syntax";

export const protobufPackage = "bytebase.v1";

export interface ParseRequest {
  expression: string;
}

export interface ParseResponse {
  expression?: ParsedExpr | undefined;
}

export interface DeparseRequest {
  expression?: ParsedExpr | undefined;
}

export interface DeparseResponse {
  expression: string;
}

function createBaseParseRequest(): ParseRequest {
  return { expression: "" };
}

export const ParseRequest = {
  encode(message: ParseRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.expression !== "") {
      writer.uint32(10).string(message.expression);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ParseRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseParseRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.expression = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ParseRequest {
    return { expression: isSet(object.expression) ? String(object.expression) : "" };
  },

  toJSON(message: ParseRequest): unknown {
    const obj: any = {};
    if (message.expression !== "") {
      obj.expression = message.expression;
    }
    return obj;
  },

  create(base?: DeepPartial<ParseRequest>): ParseRequest {
    return ParseRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ParseRequest>): ParseRequest {
    const message = createBaseParseRequest();
    message.expression = object.expression ?? "";
    return message;
  },
};

function createBaseParseResponse(): ParseResponse {
  return { expression: undefined };
}

export const ParseResponse = {
  encode(message: ParseResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.expression !== undefined) {
      ParsedExpr.encode(message.expression, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ParseResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseParseResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.expression = ParsedExpr.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ParseResponse {
    return { expression: isSet(object.expression) ? ParsedExpr.fromJSON(object.expression) : undefined };
  },

  toJSON(message: ParseResponse): unknown {
    const obj: any = {};
    if (message.expression !== undefined) {
      obj.expression = ParsedExpr.toJSON(message.expression);
    }
    return obj;
  },

  create(base?: DeepPartial<ParseResponse>): ParseResponse {
    return ParseResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ParseResponse>): ParseResponse {
    const message = createBaseParseResponse();
    message.expression = (object.expression !== undefined && object.expression !== null)
      ? ParsedExpr.fromPartial(object.expression)
      : undefined;
    return message;
  },
};

function createBaseDeparseRequest(): DeparseRequest {
  return { expression: undefined };
}

export const DeparseRequest = {
  encode(message: DeparseRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.expression !== undefined) {
      ParsedExpr.encode(message.expression, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeparseRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeparseRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.expression = ParsedExpr.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DeparseRequest {
    return { expression: isSet(object.expression) ? ParsedExpr.fromJSON(object.expression) : undefined };
  },

  toJSON(message: DeparseRequest): unknown {
    const obj: any = {};
    if (message.expression !== undefined) {
      obj.expression = ParsedExpr.toJSON(message.expression);
    }
    return obj;
  },

  create(base?: DeepPartial<DeparseRequest>): DeparseRequest {
    return DeparseRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DeparseRequest>): DeparseRequest {
    const message = createBaseDeparseRequest();
    message.expression = (object.expression !== undefined && object.expression !== null)
      ? ParsedExpr.fromPartial(object.expression)
      : undefined;
    return message;
  },
};

function createBaseDeparseResponse(): DeparseResponse {
  return { expression: "" };
}

export const DeparseResponse = {
  encode(message: DeparseResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.expression !== "") {
      writer.uint32(10).string(message.expression);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeparseResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeparseResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.expression = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DeparseResponse {
    return { expression: isSet(object.expression) ? String(object.expression) : "" };
  },

  toJSON(message: DeparseResponse): unknown {
    const obj: any = {};
    if (message.expression !== "") {
      obj.expression = message.expression;
    }
    return obj;
  },

  create(base?: DeepPartial<DeparseResponse>): DeparseResponse {
    return DeparseResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DeparseResponse>): DeparseResponse {
    const message = createBaseDeparseResponse();
    message.expression = object.expression ?? "";
    return message;
  },
};

export type CelServiceDefinition = typeof CelServiceDefinition;
export const CelServiceDefinition = {
  name: "CelService",
  fullName: "bytebase.v1.CelService",
  methods: {
    parse: {
      name: "Parse",
      requestType: ParseRequest,
      requestStream: false,
      responseType: ParseResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([18, 58, 1, 42, 34, 13, 47, 118, 49, 47, 99, 101, 108, 47, 112, 97, 114, 115, 101]),
          ],
        },
      },
    },
    deparse: {
      name: "Deparse",
      requestType: DeparseRequest,
      requestStream: false,
      responseType: DeparseResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              20,
              58,
              1,
              42,
              34,
              15,
              47,
              118,
              49,
              47,
              99,
              101,
              108,
              47,
              100,
              101,
              112,
              97,
              114,
              115,
              101,
            ]),
          ],
        },
      },
    },
  },
} as const;

export interface CelServiceImplementation<CallContextExt = {}> {
  parse(request: ParseRequest, context: CallContext & CallContextExt): Promise<DeepPartial<ParseResponse>>;
  deparse(request: DeparseRequest, context: CallContext & CallContextExt): Promise<DeepPartial<DeparseResponse>>;
}

export interface CelServiceClient<CallOptionsExt = {}> {
  parse(request: DeepPartial<ParseRequest>, options?: CallOptions & CallOptionsExt): Promise<ParseResponse>;
  deparse(request: DeepPartial<DeparseRequest>, options?: CallOptions & CallOptionsExt): Promise<DeparseResponse>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
