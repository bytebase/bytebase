/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { ParsedExpr } from "../google/api/expr/v1alpha1/syntax";

export const protobufPackage = "bytebase.store";

export interface RiskRule {
  title: string;
  expression?: ParsedExpr;
  active: boolean;
}

function createBaseRiskRule(): RiskRule {
  return { title: "", expression: undefined, active: false };
}

export const RiskRule = {
  encode(message: RiskRule, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.title !== "") {
      writer.uint32(10).string(message.title);
    }
    if (message.expression !== undefined) {
      ParsedExpr.encode(message.expression, writer.uint32(18).fork()).ldelim();
    }
    if (message.active === true) {
      writer.uint32(24).bool(message.active);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RiskRule {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRiskRule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.title = reader.string();
          break;
        case 2:
          message.expression = ParsedExpr.decode(reader, reader.uint32());
          break;
        case 3:
          message.active = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): RiskRule {
    return {
      title: isSet(object.title) ? String(object.title) : "",
      expression: isSet(object.expression) ? ParsedExpr.fromJSON(object.expression) : undefined,
      active: isSet(object.active) ? Boolean(object.active) : false,
    };
  },

  toJSON(message: RiskRule): unknown {
    const obj: any = {};
    message.title !== undefined && (obj.title = message.title);
    message.expression !== undefined &&
      (obj.expression = message.expression ? ParsedExpr.toJSON(message.expression) : undefined);
    message.active !== undefined && (obj.active = message.active);
    return obj;
  },

  fromPartial(object: DeepPartial<RiskRule>): RiskRule {
    const message = createBaseRiskRule();
    message.title = object.title ?? "";
    message.expression = (object.expression !== undefined && object.expression !== null)
      ? ParsedExpr.fromPartial(object.expression)
      : undefined;
    message.active = object.active ?? false;
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
