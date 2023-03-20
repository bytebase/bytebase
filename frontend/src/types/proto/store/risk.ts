/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { ParsedExpr } from "../google/api/expr/v1alpha1/syntax";

export const protobufPackage = "bytebase.store";

export interface RiskRule {
  rules: RiskRule_Rule[];
}

export interface RiskRule_Rule {
  title: string;
  expression?: ParsedExpr;
  active: boolean;
}

export interface RiskAction {
  actions: RiskAction_Action[];
}

export interface RiskAction_Action {
  type: RiskAction_Action_Type;
  /** Format: approvalTemplates/{approvalTemplate} */
  approvalTemplate?: string | undefined;
}

export enum RiskAction_Action_Type {
  TYPE_UNSPECIFIED = 0,
  CHOOSE_APPROVAL_TEMPLATE = 1,
  UNRECOGNIZED = -1,
}

export function riskAction_Action_TypeFromJSON(object: any): RiskAction_Action_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return RiskAction_Action_Type.TYPE_UNSPECIFIED;
    case 1:
    case "CHOOSE_APPROVAL_TEMPLATE":
      return RiskAction_Action_Type.CHOOSE_APPROVAL_TEMPLATE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return RiskAction_Action_Type.UNRECOGNIZED;
  }
}

export function riskAction_Action_TypeToJSON(object: RiskAction_Action_Type): string {
  switch (object) {
    case RiskAction_Action_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case RiskAction_Action_Type.CHOOSE_APPROVAL_TEMPLATE:
      return "CHOOSE_APPROVAL_TEMPLATE";
    case RiskAction_Action_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

function createBaseRiskRule(): RiskRule {
  return { rules: [] };
}

export const RiskRule = {
  encode(message: RiskRule, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.rules) {
      RiskRule_Rule.encode(v!, writer.uint32(10).fork()).ldelim();
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
          message.rules.push(RiskRule_Rule.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): RiskRule {
    return { rules: Array.isArray(object?.rules) ? object.rules.map((e: any) => RiskRule_Rule.fromJSON(e)) : [] };
  },

  toJSON(message: RiskRule): unknown {
    const obj: any = {};
    if (message.rules) {
      obj.rules = message.rules.map((e) => e ? RiskRule_Rule.toJSON(e) : undefined);
    } else {
      obj.rules = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<RiskRule>): RiskRule {
    const message = createBaseRiskRule();
    message.rules = object.rules?.map((e) => RiskRule_Rule.fromPartial(e)) || [];
    return message;
  },
};

function createBaseRiskRule_Rule(): RiskRule_Rule {
  return { title: "", expression: undefined, active: false };
}

export const RiskRule_Rule = {
  encode(message: RiskRule_Rule, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): RiskRule_Rule {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRiskRule_Rule();
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

  fromJSON(object: any): RiskRule_Rule {
    return {
      title: isSet(object.title) ? String(object.title) : "",
      expression: isSet(object.expression) ? ParsedExpr.fromJSON(object.expression) : undefined,
      active: isSet(object.active) ? Boolean(object.active) : false,
    };
  },

  toJSON(message: RiskRule_Rule): unknown {
    const obj: any = {};
    message.title !== undefined && (obj.title = message.title);
    message.expression !== undefined &&
      (obj.expression = message.expression ? ParsedExpr.toJSON(message.expression) : undefined);
    message.active !== undefined && (obj.active = message.active);
    return obj;
  },

  fromPartial(object: DeepPartial<RiskRule_Rule>): RiskRule_Rule {
    const message = createBaseRiskRule_Rule();
    message.title = object.title ?? "";
    message.expression = (object.expression !== undefined && object.expression !== null)
      ? ParsedExpr.fromPartial(object.expression)
      : undefined;
    message.active = object.active ?? false;
    return message;
  },
};

function createBaseRiskAction(): RiskAction {
  return { actions: [] };
}

export const RiskAction = {
  encode(message: RiskAction, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.actions) {
      RiskAction_Action.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RiskAction {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRiskAction();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.actions.push(RiskAction_Action.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): RiskAction {
    return {
      actions: Array.isArray(object?.actions) ? object.actions.map((e: any) => RiskAction_Action.fromJSON(e)) : [],
    };
  },

  toJSON(message: RiskAction): unknown {
    const obj: any = {};
    if (message.actions) {
      obj.actions = message.actions.map((e) => e ? RiskAction_Action.toJSON(e) : undefined);
    } else {
      obj.actions = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<RiskAction>): RiskAction {
    const message = createBaseRiskAction();
    message.actions = object.actions?.map((e) => RiskAction_Action.fromPartial(e)) || [];
    return message;
  },
};

function createBaseRiskAction_Action(): RiskAction_Action {
  return { type: 0, approvalTemplate: undefined };
}

export const RiskAction_Action = {
  encode(message: RiskAction_Action, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.type !== 0) {
      writer.uint32(8).int32(message.type);
    }
    if (message.approvalTemplate !== undefined) {
      writer.uint32(18).string(message.approvalTemplate);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RiskAction_Action {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRiskAction_Action();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.type = reader.int32() as any;
          break;
        case 2:
          message.approvalTemplate = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): RiskAction_Action {
    return {
      type: isSet(object.type) ? riskAction_Action_TypeFromJSON(object.type) : 0,
      approvalTemplate: isSet(object.approvalTemplate) ? String(object.approvalTemplate) : undefined,
    };
  },

  toJSON(message: RiskAction_Action): unknown {
    const obj: any = {};
    message.type !== undefined && (obj.type = riskAction_Action_TypeToJSON(message.type));
    message.approvalTemplate !== undefined && (obj.approvalTemplate = message.approvalTemplate);
    return obj;
  },

  fromPartial(object: DeepPartial<RiskAction_Action>): RiskAction_Action {
    const message = createBaseRiskAction_Action();
    message.type = object.type ?? 0;
    message.approvalTemplate = object.approvalTemplate ?? undefined;
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
