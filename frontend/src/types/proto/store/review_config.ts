/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { SQLReviewRule } from "./policy";

export const protobufPackage = "bytebase.store";

export interface ReviewConfigPayload {
  sqlReviewRules: SQLReviewRule[];
}

function createBaseReviewConfigPayload(): ReviewConfigPayload {
  return { sqlReviewRules: [] };
}

export const ReviewConfigPayload = {
  encode(message: ReviewConfigPayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.sqlReviewRules) {
      SQLReviewRule.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ReviewConfigPayload {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseReviewConfigPayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.sqlReviewRules.push(SQLReviewRule.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ReviewConfigPayload {
    return {
      sqlReviewRules: globalThis.Array.isArray(object?.sqlReviewRules)
        ? object.sqlReviewRules.map((e: any) => SQLReviewRule.fromJSON(e))
        : [],
    };
  },

  toJSON(message: ReviewConfigPayload): unknown {
    const obj: any = {};
    if (message.sqlReviewRules?.length) {
      obj.sqlReviewRules = message.sqlReviewRules.map((e) => SQLReviewRule.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<ReviewConfigPayload>): ReviewConfigPayload {
    return ReviewConfigPayload.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ReviewConfigPayload>): ReviewConfigPayload {
    const message = createBaseReviewConfigPayload();
    message.sqlReviewRules = object.sqlReviewRules?.map((e) => SQLReviewRule.fromPartial(e)) || [];
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
