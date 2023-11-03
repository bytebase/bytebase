/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { ParsedExpr } from "../google/api/expr/v1alpha1/syntax";

export const protobufPackage = "bytebase.v1";

export interface BatchParseRequest {
  expressions: string[];
}

export interface BatchParseResponse {
  expressions: ParsedExpr[];
}

export interface BatchDeparseRequest {
  expressions: ParsedExpr[];
}

export interface BatchDeparseResponse {
  expressions: string[];
}

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
    return {
      expressions: globalThis.Array.isArray(object?.expressions)
        ? object.expressions.map((e: any) => globalThis.String(e))
        : [],
    };
  },

  toJSON(message: BatchParseRequest): unknown {
    const obj: any = {};
    if (message.expressions?.length) {
      obj.expressions = message.expressions;
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
      expressions: globalThis.Array.isArray(object?.expressions)
        ? object.expressions.map((e: any) => ParsedExpr.fromJSON(e))
        : [],
    };
  },

  toJSON(message: BatchParseResponse): unknown {
    const obj: any = {};
    if (message.expressions?.length) {
      obj.expressions = message.expressions.map((e) => ParsedExpr.toJSON(e));
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
      expressions: globalThis.Array.isArray(object?.expressions)
        ? object.expressions.map((e: any) => ParsedExpr.fromJSON(e))
        : [],
    };
  },

  toJSON(message: BatchDeparseRequest): unknown {
    const obj: any = {};
    if (message.expressions?.length) {
      obj.expressions = message.expressions.map((e) => ParsedExpr.toJSON(e));
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
    return {
      expressions: globalThis.Array.isArray(object?.expressions)
        ? object.expressions.map((e: any) => globalThis.String(e))
        : [],
    };
  },

  toJSON(message: BatchDeparseResponse): unknown {
    const obj: any = {};
    if (message.expressions?.length) {
      obj.expressions = message.expressions;
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
  : T extends Long ? string | number | Long : T extends globalThis.Array<infer U> ? globalThis.Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}
