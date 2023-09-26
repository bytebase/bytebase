/* eslint-disable */
import _m0 from "protobufjs/minimal";
import { ParsedExpr } from "../google/api/expr/v1alpha1/syntax";

export const protobufPackage = "bytebase.v1";

export interface ParseRequest {
  expression: string;
}

export interface BatchParseRequest {
  expressions: string[];
}

export interface ParseResponse {
  expression: ParsedExpr | undefined;
}

export interface BatchParseResponse {
  expressions: ParsedExpr[];
}

export interface DeparseRequest {
  expression: ParsedExpr | undefined;
}

export interface BatchDeparseRequest {
  expressions: ParsedExpr[];
}

export interface DeparseResponse {
  expression: string;
}

export interface BatchDeparseResponse {
  expressions: string[];
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
    message.expression !== undefined && (obj.expression = message.expression);
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

function createBaseBatchParseRequest(): BatchParseRequest {
  return { expressions: [] };
}

export const BatchParseRequest = {
  encode(message: BatchParseRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.expressions) {
      writer.uint32(10).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchParseRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchParseRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.expressions.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BatchParseRequest {
    return { expressions: Array.isArray(object?.expressions) ? object.expressions.map((e: any) => String(e)) : [] };
  },

  toJSON(message: BatchParseRequest): unknown {
    const obj: any = {};
    if (message.expressions) {
      obj.expressions = message.expressions.map((e) => e);
    } else {
      obj.expressions = [];
    }
    return obj;
  },

  create(base?: DeepPartial<BatchParseRequest>): BatchParseRequest {
    return BatchParseRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<BatchParseRequest>): BatchParseRequest {
    const message = createBaseBatchParseRequest();
    message.expressions = object.expressions?.map((e) => e) || [];
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
    message.expression !== undefined &&
      (obj.expression = message.expression ? ParsedExpr.toJSON(message.expression) : undefined);
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

function createBaseBatchParseResponse(): BatchParseResponse {
  return { expressions: [] };
}

export const BatchParseResponse = {
  encode(message: BatchParseResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.expressions) {
      ParsedExpr.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchParseResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchParseResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.expressions.push(ParsedExpr.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BatchParseResponse {
    return {
      expressions: Array.isArray(object?.expressions) ? object.expressions.map((e: any) => ParsedExpr.fromJSON(e)) : [],
    };
  },

  toJSON(message: BatchParseResponse): unknown {
    const obj: any = {};
    if (message.expressions) {
      obj.expressions = message.expressions.map((e) => e ? ParsedExpr.toJSON(e) : undefined);
    } else {
      obj.expressions = [];
    }
    return obj;
  },

  create(base?: DeepPartial<BatchParseResponse>): BatchParseResponse {
    return BatchParseResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<BatchParseResponse>): BatchParseResponse {
    const message = createBaseBatchParseResponse();
    message.expressions = object.expressions?.map((e) => ParsedExpr.fromPartial(e)) || [];
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
    message.expression !== undefined &&
      (obj.expression = message.expression ? ParsedExpr.toJSON(message.expression) : undefined);
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

function createBaseBatchDeparseRequest(): BatchDeparseRequest {
  return { expressions: [] };
}

export const BatchDeparseRequest = {
  encode(message: BatchDeparseRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.expressions) {
      ParsedExpr.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchDeparseRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchDeparseRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.expressions.push(ParsedExpr.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BatchDeparseRequest {
    return {
      expressions: Array.isArray(object?.expressions) ? object.expressions.map((e: any) => ParsedExpr.fromJSON(e)) : [],
    };
  },

  toJSON(message: BatchDeparseRequest): unknown {
    const obj: any = {};
    if (message.expressions) {
      obj.expressions = message.expressions.map((e) => e ? ParsedExpr.toJSON(e) : undefined);
    } else {
      obj.expressions = [];
    }
    return obj;
  },

  create(base?: DeepPartial<BatchDeparseRequest>): BatchDeparseRequest {
    return BatchDeparseRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<BatchDeparseRequest>): BatchDeparseRequest {
    const message = createBaseBatchDeparseRequest();
    message.expressions = object.expressions?.map((e) => ParsedExpr.fromPartial(e)) || [];
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
    message.expression !== undefined && (obj.expression = message.expression);
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

function createBaseBatchDeparseResponse(): BatchDeparseResponse {
  return { expressions: [] };
}

export const BatchDeparseResponse = {
  encode(message: BatchDeparseResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.expressions) {
      writer.uint32(10).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BatchDeparseResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBatchDeparseResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.expressions.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BatchDeparseResponse {
    return { expressions: Array.isArray(object?.expressions) ? object.expressions.map((e: any) => String(e)) : [] };
  },

  toJSON(message: BatchDeparseResponse): unknown {
    const obj: any = {};
    if (message.expressions) {
      obj.expressions = message.expressions.map((e) => e);
    } else {
      obj.expressions = [];
    }
    return obj;
  },

  create(base?: DeepPartial<BatchDeparseResponse>): BatchDeparseResponse {
    return BatchDeparseResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<BatchDeparseResponse>): BatchDeparseResponse {
    const message = createBaseBatchDeparseResponse();
    message.expressions = object.expressions?.map((e) => e) || [];
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
    batchParse: {
      name: "BatchParse",
      requestType: BatchParseRequest,
      requestStream: false,
      responseType: BatchParseResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              23,
              58,
              1,
              42,
              34,
              18,
              47,
              118,
              49,
              47,
              99,
              101,
              108,
              47,
              98,
              97,
              116,
              99,
              104,
              80,
              97,
              114,
              115,
              101,
            ]),
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
    batchDeparse: {
      name: "BatchDeparse",
      requestType: BatchDeparseRequest,
      requestStream: false,
      responseType: BatchDeparseResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              25,
              58,
              1,
              42,
              34,
              20,
              47,
              118,
              49,
              47,
              99,
              101,
              108,
              47,
              98,
              97,
              116,
              99,
              104,
              68,
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

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
