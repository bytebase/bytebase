/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Duration } from "../google/protobuf/duration";

export const protobufPackage = "bytebase.store";

export interface QueryHistoryPayload {
  error?: string | undefined;
  duration: Duration | undefined;
}

function createBaseQueryHistoryPayload(): QueryHistoryPayload {
  return { error: undefined, duration: undefined };
}

export const QueryHistoryPayload = {
  encode(message: QueryHistoryPayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.error !== undefined) {
      writer.uint32(10).string(message.error);
    }
    if (message.duration !== undefined) {
      Duration.encode(message.duration, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): QueryHistoryPayload {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQueryHistoryPayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.error = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.duration = Duration.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): QueryHistoryPayload {
    return {
      error: isSet(object.error) ? globalThis.String(object.error) : undefined,
      duration: isSet(object.duration) ? Duration.fromJSON(object.duration) : undefined,
    };
  },

  toJSON(message: QueryHistoryPayload): unknown {
    const obj: any = {};
    if (message.error !== undefined) {
      obj.error = message.error;
    }
    if (message.duration !== undefined) {
      obj.duration = Duration.toJSON(message.duration);
    }
    return obj;
  },

  create(base?: DeepPartial<QueryHistoryPayload>): QueryHistoryPayload {
    return QueryHistoryPayload.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<QueryHistoryPayload>): QueryHistoryPayload {
    const message = createBaseQueryHistoryPayload();
    message.error = object.error ?? undefined;
    message.duration = (object.duration !== undefined && object.duration !== null)
      ? Duration.fromPartial(object.duration)
      : undefined;
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
