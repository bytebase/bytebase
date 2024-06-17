/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { SQLReviewRule } from "./policy";

export const protobufPackage = "bytebase.store";

export interface SQLReviewPayload {
  ruleList: SQLReviewRule[];
}

function createBaseSQLReviewPayload(): SQLReviewPayload {
  return { ruleList: [] };
}

export const SQLReviewPayload = {
  encode(message: SQLReviewPayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.ruleList) {
      SQLReviewRule.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SQLReviewPayload {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSQLReviewPayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.ruleList.push(SQLReviewRule.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SQLReviewPayload {
    return {
      ruleList: globalThis.Array.isArray(object?.ruleList)
        ? object.ruleList.map((e: any) => SQLReviewRule.fromJSON(e))
        : [],
    };
  },

  toJSON(message: SQLReviewPayload): unknown {
    const obj: any = {};
    if (message.ruleList?.length) {
      obj.ruleList = message.ruleList.map((e) => SQLReviewRule.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<SQLReviewPayload>): SQLReviewPayload {
    return SQLReviewPayload.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SQLReviewPayload>): SQLReviewPayload {
    const message = createBaseSQLReviewPayload();
    message.ruleList = object.ruleList?.map((e) => SQLReviewRule.fromPartial(e)) || [];
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
