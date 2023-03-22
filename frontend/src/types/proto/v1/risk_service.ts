/* eslint-disable */
import * as Long from "long";
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { ParsedExpr } from "../google/api/expr/v1alpha1/syntax";
import { FieldMask } from "../google/protobuf/field_mask";

export const protobufPackage = "bytebase.v1";

export interface ListRisksRequest {
  /**
   * The maximum number of risks to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 risks will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListRisks` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `LiskRisks` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface ListRisksResponse {
  risks: Risk[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface UpdateRiskRequest {
  /**
   * The risk to update.
   *
   * The risk's `name` field is used to identify the risk to update.
   * Format: risks/{risk}
   */
  risk?: Risk;
  /** The list of fields to update. */
  updateMask?: string[];
}

export interface Risk {
  /** Format: risks/{risk} */
  name: string;
  /** system-generated unique identifier. */
  uid: string;
  namespace: Risk_Namespace;
  title: string;
  level: number;
  actions: RiskAction[];
  rules: RiskRule[];
}

export enum Risk_Namespace {
  NAMESPACE_UNSPECIFIED = 0,
  DDL = 1,
  DML = 2,
  CREATE_DATABASE = 3,
  UNRECOGNIZED = -1,
}

export function risk_NamespaceFromJSON(object: any): Risk_Namespace {
  switch (object) {
    case 0:
    case "NAMESPACE_UNSPECIFIED":
      return Risk_Namespace.NAMESPACE_UNSPECIFIED;
    case 1:
    case "DDL":
      return Risk_Namespace.DDL;
    case 2:
    case "DML":
      return Risk_Namespace.DML;
    case 3:
    case "CREATE_DATABASE":
      return Risk_Namespace.CREATE_DATABASE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Risk_Namespace.UNRECOGNIZED;
  }
}

export function risk_NamespaceToJSON(object: Risk_Namespace): string {
  switch (object) {
    case Risk_Namespace.NAMESPACE_UNSPECIFIED:
      return "NAMESPACE_UNSPECIFIED";
    case Risk_Namespace.DDL:
      return "DDL";
    case Risk_Namespace.DML:
      return "DML";
    case Risk_Namespace.CREATE_DATABASE:
      return "CREATE_DATABASE";
    case Risk_Namespace.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface RiskAction {
  type: RiskAction_Type;
  /** Format: approvalTemplates/{approvalTemplate} */
  approvalTemplate?: string | undefined;
}

export enum RiskAction_Type {
  TYPE_UNSPECIFIED = 0,
  CHOOSE_APPROVAL_TEMPLATE = 1,
  UNRECOGNIZED = -1,
}

export function riskAction_TypeFromJSON(object: any): RiskAction_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return RiskAction_Type.TYPE_UNSPECIFIED;
    case 1:
    case "CHOOSE_APPROVAL_TEMPLATE":
      return RiskAction_Type.CHOOSE_APPROVAL_TEMPLATE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return RiskAction_Type.UNRECOGNIZED;
  }
}

export function riskAction_TypeToJSON(object: RiskAction_Type): string {
  switch (object) {
    case RiskAction_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case RiskAction_Type.CHOOSE_APPROVAL_TEMPLATE:
      return "CHOOSE_APPROVAL_TEMPLATE";
    case RiskAction_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface RiskRule {
  title: string;
  expression?: ParsedExpr;
  active: boolean;
}

function createBaseListRisksRequest(): ListRisksRequest {
  return { pageSize: 0, pageToken: "" };
}

export const ListRisksRequest = {
  encode(message: ListRisksRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pageSize !== 0) {
      writer.uint32(8).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(18).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListRisksRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListRisksRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pageSize = reader.int32();
          break;
        case 2:
          message.pageToken = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ListRisksRequest {
    return {
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListRisksRequest): unknown {
    const obj: any = {};
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    return obj;
  },

  fromPartial(object: DeepPartial<ListRisksRequest>): ListRisksRequest {
    const message = createBaseListRisksRequest();
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListRisksResponse(): ListRisksResponse {
  return { risks: [], nextPageToken: "" };
}

export const ListRisksResponse = {
  encode(message: ListRisksResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.risks) {
      Risk.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListRisksResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListRisksResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.risks.push(Risk.decode(reader, reader.uint32()));
          break;
        case 2:
          message.nextPageToken = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ListRisksResponse {
    return {
      risks: Array.isArray(object?.risks) ? object.risks.map((e: any) => Risk.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListRisksResponse): unknown {
    const obj: any = {};
    if (message.risks) {
      obj.risks = message.risks.map((e) => e ? Risk.toJSON(e) : undefined);
    } else {
      obj.risks = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  fromPartial(object: DeepPartial<ListRisksResponse>): ListRisksResponse {
    const message = createBaseListRisksResponse();
    message.risks = object.risks?.map((e) => Risk.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseUpdateRiskRequest(): UpdateRiskRequest {
  return { risk: undefined, updateMask: undefined };
}

export const UpdateRiskRequest = {
  encode(message: UpdateRiskRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.risk !== undefined) {
      Risk.encode(message.risk, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateRiskRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateRiskRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.risk = Risk.decode(reader, reader.uint32());
          break;
        case 2:
          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): UpdateRiskRequest {
    return {
      risk: isSet(object.risk) ? Risk.fromJSON(object.risk) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateRiskRequest): unknown {
    const obj: any = {};
    message.risk !== undefined && (obj.risk = message.risk ? Risk.toJSON(message.risk) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  fromPartial(object: DeepPartial<UpdateRiskRequest>): UpdateRiskRequest {
    const message = createBaseUpdateRiskRequest();
    message.risk = (object.risk !== undefined && object.risk !== null) ? Risk.fromPartial(object.risk) : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseRisk(): Risk {
  return { name: "", uid: "", namespace: 0, title: "", level: 0, actions: [], rules: [] };
}

export const Risk = {
  encode(message: Risk, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.uid !== "") {
      writer.uint32(18).string(message.uid);
    }
    if (message.namespace !== 0) {
      writer.uint32(24).int32(message.namespace);
    }
    if (message.title !== "") {
      writer.uint32(34).string(message.title);
    }
    if (message.level !== 0) {
      writer.uint32(40).int64(message.level);
    }
    for (const v of message.actions) {
      RiskAction.encode(v!, writer.uint32(50).fork()).ldelim();
    }
    for (const v of message.rules) {
      RiskRule.encode(v!, writer.uint32(58).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Risk {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRisk();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.uid = reader.string();
          break;
        case 3:
          message.namespace = reader.int32() as any;
          break;
        case 4:
          message.title = reader.string();
          break;
        case 5:
          message.level = longToNumber(reader.int64() as Long);
          break;
        case 6:
          message.actions.push(RiskAction.decode(reader, reader.uint32()));
          break;
        case 7:
          message.rules.push(RiskRule.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Risk {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      uid: isSet(object.uid) ? String(object.uid) : "",
      namespace: isSet(object.namespace) ? risk_NamespaceFromJSON(object.namespace) : 0,
      title: isSet(object.title) ? String(object.title) : "",
      level: isSet(object.level) ? Number(object.level) : 0,
      actions: Array.isArray(object?.actions) ? object.actions.map((e: any) => RiskAction.fromJSON(e)) : [],
      rules: Array.isArray(object?.rules) ? object.rules.map((e: any) => RiskRule.fromJSON(e)) : [],
    };
  },

  toJSON(message: Risk): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.namespace !== undefined && (obj.namespace = risk_NamespaceToJSON(message.namespace));
    message.title !== undefined && (obj.title = message.title);
    message.level !== undefined && (obj.level = Math.round(message.level));
    if (message.actions) {
      obj.actions = message.actions.map((e) => e ? RiskAction.toJSON(e) : undefined);
    } else {
      obj.actions = [];
    }
    if (message.rules) {
      obj.rules = message.rules.map((e) => e ? RiskRule.toJSON(e) : undefined);
    } else {
      obj.rules = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<Risk>): Risk {
    const message = createBaseRisk();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.namespace = object.namespace ?? 0;
    message.title = object.title ?? "";
    message.level = object.level ?? 0;
    message.actions = object.actions?.map((e) => RiskAction.fromPartial(e)) || [];
    message.rules = object.rules?.map((e) => RiskRule.fromPartial(e)) || [];
    return message;
  },
};

function createBaseRiskAction(): RiskAction {
  return { type: 0, approvalTemplate: undefined };
}

export const RiskAction = {
  encode(message: RiskAction, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.type !== 0) {
      writer.uint32(8).int32(message.type);
    }
    if (message.approvalTemplate !== undefined) {
      writer.uint32(18).string(message.approvalTemplate);
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

  fromJSON(object: any): RiskAction {
    return {
      type: isSet(object.type) ? riskAction_TypeFromJSON(object.type) : 0,
      approvalTemplate: isSet(object.approvalTemplate) ? String(object.approvalTemplate) : undefined,
    };
  },

  toJSON(message: RiskAction): unknown {
    const obj: any = {};
    message.type !== undefined && (obj.type = riskAction_TypeToJSON(message.type));
    message.approvalTemplate !== undefined && (obj.approvalTemplate = message.approvalTemplate);
    return obj;
  },

  fromPartial(object: DeepPartial<RiskAction>): RiskAction {
    const message = createBaseRiskAction();
    message.type = object.type ?? 0;
    message.approvalTemplate = object.approvalTemplate ?? undefined;
    return message;
  },
};

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

export type RiskServiceDefinition = typeof RiskServiceDefinition;
export const RiskServiceDefinition = {
  name: "RiskService",
  fullName: "bytebase.v1.RiskService",
  methods: {
    listRisks: {
      name: "ListRisks",
      requestType: ListRisksRequest,
      requestStream: false,
      responseType: ListRisksResponse,
      responseStream: false,
      options: {},
    },
    updateRisk: {
      name: "UpdateRisk",
      requestType: UpdateRiskRequest,
      requestStream: false,
      responseType: Risk,
      responseStream: false,
      options: {},
    },
  },
} as const;

export interface RiskServiceImplementation<CallContextExt = {}> {
  listRisks(request: ListRisksRequest, context: CallContext & CallContextExt): Promise<DeepPartial<ListRisksResponse>>;
  updateRisk(request: UpdateRiskRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Risk>>;
}

export interface RiskServiceClient<CallOptionsExt = {}> {
  listRisks(request: DeepPartial<ListRisksRequest>, options?: CallOptions & CallOptionsExt): Promise<ListRisksResponse>;
  updateRisk(request: DeepPartial<UpdateRiskRequest>, options?: CallOptions & CallOptionsExt): Promise<Risk>;
}

declare var self: any | undefined;
declare var window: any | undefined;
declare var global: any | undefined;
var globalThis: any = (() => {
  if (typeof globalThis !== "undefined") {
    return globalThis;
  }
  if (typeof self !== "undefined") {
    return self;
  }
  if (typeof window !== "undefined") {
    return window;
  }
  if (typeof global !== "undefined") {
    return global;
  }
  throw "Unable to locate global object";
})();

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function longToNumber(long: Long): number {
  if (long.gt(Number.MAX_SAFE_INTEGER)) {
    throw new globalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
  }
  return long.toNumber();
}

// If you get a compile-error about 'Constructor<Long> and ... have no overlap',
// add '--ts_proto_opt=esModuleInterop=true' as a flag when calling 'protoc'.
if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
