/* eslint-disable */
import _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.v1";

export interface HelloRequest {
  name: string;
}

export interface HelloResponse {
  answer: string;
}

function createBaseHelloRequest(): HelloRequest {
  return { name: "" };
}

export const HelloRequest = {
  encode(message: HelloRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): HelloRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseHelloRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): HelloRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: HelloRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<HelloRequest>, I>>(object: I): HelloRequest {
    const message = createBaseHelloRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseHelloResponse(): HelloResponse {
  return { answer: "" };
}

export const HelloResponse = {
  encode(message: HelloResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.answer !== "") {
      writer.uint32(10).string(message.answer);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): HelloResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseHelloResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.answer = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): HelloResponse {
    return { answer: isSet(object.answer) ? String(object.answer) : "" };
  },

  toJSON(message: HelloResponse): unknown {
    const obj: any = {};
    message.answer !== undefined && (obj.answer = message.answer);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<HelloResponse>, I>>(object: I): HelloResponse {
    const message = createBaseHelloResponse();
    message.answer = object.answer ?? "";
    return message;
  },
};

export interface GreeterService {
  Hello(request: HelloRequest): Promise<HelloResponse>;
}

export class GreeterServiceClientImpl implements GreeterService {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "bytebase.v1.GreeterService";
    this.rpc = rpc;
    this.Hello = this.Hello.bind(this);
  }
  Hello(request: HelloRequest): Promise<HelloResponse> {
    const data = HelloRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "Hello", data);
    return promise.then((data) => HelloResponse.decode(new _m0.Reader(data)));
  }
}

interface Rpc {
  request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

type KeysOfUnion<T> = T extends T ? keyof T : never;
export type Exact<P, I extends P> = P extends Builtin ? P
  : P & { [K in keyof P]: Exact<P[K], I[K]> } & { [K in Exclude<keyof I, KeysOfUnion<P>>]: never };

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
