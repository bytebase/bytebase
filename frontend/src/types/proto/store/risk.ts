/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { ParsedExpr } from "../google/api/expr/v1alpha1/syntax";

export const protobufPackage = "bytebase.store";

export interface RiskExpression {
  expression?: ParsedExpr;
}

function createBaseRiskExpression(): RiskExpression {
  return { expression: undefined };
}

export const RiskExpression = {
  encode(message: RiskExpression, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.expression !== undefined) {
      ParsedExpr.encode(message.expression, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RiskExpression {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRiskExpression();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.expression = ParsedExpr.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): RiskExpression {
    return { expression: isSet(object.expression) ? ParsedExpr.fromJSON(object.expression) : undefined };
  },

  toJSON(message: RiskExpression): unknown {
    const obj: any = {};
    message.expression !== undefined &&
      (obj.expression = message.expression ? ParsedExpr.toJSON(message.expression) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<RiskExpression>): RiskExpression {
    const message = createBaseRiskExpression();
    message.expression = (object.expression !== undefined && object.expression !== null)
      ? ParsedExpr.fromPartial(object.expression)
      : undefined;
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
