/* eslint-disable */
import * as Long from "long";
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { ParsedExpr } from "../google/api/expr/v1alpha1/syntax";
import { FieldMask } from "../google/protobuf/field_mask";

export const protobufPackage = "bytebase.v1";

export interface GetRiskRequest {
  /**
   * The name of the risk to retrieve.
   * Format: risks/{risk}
   */
  name: string;
}

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
  /** Show deleted risks if specified. */
  showDeleted: boolean;
}

export interface ListRisksResponse {
  risks: Risk[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface AddRiskConditionRequest {
  /**
   * The name of the risk to add the risk condition to.
   * Format: risks/{risk}
   */
  risk: string;
  /** The risk condition to add. */
  riskCondition?: RiskCondition;
}

export interface RemoveRiskConditionRequest {
  /**
   * The name of the risk to remove the risk condition from.
   * Format: risks/{risk}
   */
  risk: string;
  /** The risk condition to remove. Identified by its name. */
  riskCondition?: RiskCondition;
}

export interface UpdateRiskConditionRequest {
  /**
   * The name of the risk which owns the risk condition to be updated.
   * Format: risks/{risk}
   */
  risk: string;
  /**
   * The risk condition to modify.
   * Identified by its name.
   */
  riskCondition?: RiskCondition;
  /** The list of fields to update. */
  updateMask?: string[];
}

export interface Risk {
  /** Format: risks/{risk} */
  name: string;
  /** system-generated unique identifier. */
  uid: string;
  title: string;
  description: string;
  level: number;
  actions: RiskAction[];
  conditions: RiskCondition[];
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

export interface RiskCondition {
  /** Format: risks/{risk}/riskConditions/{riskCondition} */
  name: string;
  uid: string;
  title: string;
  description: string;
  expression?: ParsedExpr;
}

function createBaseGetRiskRequest(): GetRiskRequest {
  return { name: "" };
}

export const GetRiskRequest = {
  encode(message: GetRiskRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetRiskRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetRiskRequest();
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

  fromJSON(object: any): GetRiskRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetRiskRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<GetRiskRequest>): GetRiskRequest {
    const message = createBaseGetRiskRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListRisksRequest(): ListRisksRequest {
  return { pageSize: 0, pageToken: "", showDeleted: false };
}

export const ListRisksRequest = {
  encode(message: ListRisksRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pageSize !== 0) {
      writer.uint32(8).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(18).string(message.pageToken);
    }
    if (message.showDeleted === true) {
      writer.uint32(24).bool(message.showDeleted);
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
        case 3:
          message.showDeleted = reader.bool();
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
      showDeleted: isSet(object.showDeleted) ? Boolean(object.showDeleted) : false,
    };
  },

  toJSON(message: ListRisksRequest): unknown {
    const obj: any = {};
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    message.showDeleted !== undefined && (obj.showDeleted = message.showDeleted);
    return obj;
  },

  fromPartial(object: DeepPartial<ListRisksRequest>): ListRisksRequest {
    const message = createBaseListRisksRequest();
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    message.showDeleted = object.showDeleted ?? false;
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

function createBaseAddRiskConditionRequest(): AddRiskConditionRequest {
  return { risk: "", riskCondition: undefined };
}

export const AddRiskConditionRequest = {
  encode(message: AddRiskConditionRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.risk !== "") {
      writer.uint32(10).string(message.risk);
    }
    if (message.riskCondition !== undefined) {
      RiskCondition.encode(message.riskCondition, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): AddRiskConditionRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAddRiskConditionRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.risk = reader.string();
          break;
        case 2:
          message.riskCondition = RiskCondition.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): AddRiskConditionRequest {
    return {
      risk: isSet(object.risk) ? String(object.risk) : "",
      riskCondition: isSet(object.riskCondition) ? RiskCondition.fromJSON(object.riskCondition) : undefined,
    };
  },

  toJSON(message: AddRiskConditionRequest): unknown {
    const obj: any = {};
    message.risk !== undefined && (obj.risk = message.risk);
    message.riskCondition !== undefined &&
      (obj.riskCondition = message.riskCondition ? RiskCondition.toJSON(message.riskCondition) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<AddRiskConditionRequest>): AddRiskConditionRequest {
    const message = createBaseAddRiskConditionRequest();
    message.risk = object.risk ?? "";
    message.riskCondition = (object.riskCondition !== undefined && object.riskCondition !== null)
      ? RiskCondition.fromPartial(object.riskCondition)
      : undefined;
    return message;
  },
};

function createBaseRemoveRiskConditionRequest(): RemoveRiskConditionRequest {
  return { risk: "", riskCondition: undefined };
}

export const RemoveRiskConditionRequest = {
  encode(message: RemoveRiskConditionRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.risk !== "") {
      writer.uint32(10).string(message.risk);
    }
    if (message.riskCondition !== undefined) {
      RiskCondition.encode(message.riskCondition, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RemoveRiskConditionRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRemoveRiskConditionRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.risk = reader.string();
          break;
        case 2:
          message.riskCondition = RiskCondition.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): RemoveRiskConditionRequest {
    return {
      risk: isSet(object.risk) ? String(object.risk) : "",
      riskCondition: isSet(object.riskCondition) ? RiskCondition.fromJSON(object.riskCondition) : undefined,
    };
  },

  toJSON(message: RemoveRiskConditionRequest): unknown {
    const obj: any = {};
    message.risk !== undefined && (obj.risk = message.risk);
    message.riskCondition !== undefined &&
      (obj.riskCondition = message.riskCondition ? RiskCondition.toJSON(message.riskCondition) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<RemoveRiskConditionRequest>): RemoveRiskConditionRequest {
    const message = createBaseRemoveRiskConditionRequest();
    message.risk = object.risk ?? "";
    message.riskCondition = (object.riskCondition !== undefined && object.riskCondition !== null)
      ? RiskCondition.fromPartial(object.riskCondition)
      : undefined;
    return message;
  },
};

function createBaseUpdateRiskConditionRequest(): UpdateRiskConditionRequest {
  return { risk: "", riskCondition: undefined, updateMask: undefined };
}

export const UpdateRiskConditionRequest = {
  encode(message: UpdateRiskConditionRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.risk !== "") {
      writer.uint32(10).string(message.risk);
    }
    if (message.riskCondition !== undefined) {
      RiskCondition.encode(message.riskCondition, writer.uint32(18).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateRiskConditionRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateRiskConditionRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.risk = reader.string();
          break;
        case 2:
          message.riskCondition = RiskCondition.decode(reader, reader.uint32());
          break;
        case 3:
          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): UpdateRiskConditionRequest {
    return {
      risk: isSet(object.risk) ? String(object.risk) : "",
      riskCondition: isSet(object.riskCondition) ? RiskCondition.fromJSON(object.riskCondition) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateRiskConditionRequest): unknown {
    const obj: any = {};
    message.risk !== undefined && (obj.risk = message.risk);
    message.riskCondition !== undefined &&
      (obj.riskCondition = message.riskCondition ? RiskCondition.toJSON(message.riskCondition) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  fromPartial(object: DeepPartial<UpdateRiskConditionRequest>): UpdateRiskConditionRequest {
    const message = createBaseUpdateRiskConditionRequest();
    message.risk = object.risk ?? "";
    message.riskCondition = (object.riskCondition !== undefined && object.riskCondition !== null)
      ? RiskCondition.fromPartial(object.riskCondition)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseRisk(): Risk {
  return { name: "", uid: "", title: "", description: "", level: 0, actions: [], conditions: [] };
}

export const Risk = {
  encode(message: Risk, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.uid !== "") {
      writer.uint32(18).string(message.uid);
    }
    if (message.title !== "") {
      writer.uint32(26).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(34).string(message.description);
    }
    if (message.level !== 0) {
      writer.uint32(40).int64(message.level);
    }
    for (const v of message.actions) {
      RiskAction.encode(v!, writer.uint32(50).fork()).ldelim();
    }
    for (const v of message.conditions) {
      RiskCondition.encode(v!, writer.uint32(58).fork()).ldelim();
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
          message.title = reader.string();
          break;
        case 4:
          message.description = reader.string();
          break;
        case 5:
          message.level = longToNumber(reader.int64() as Long);
          break;
        case 6:
          message.actions.push(RiskAction.decode(reader, reader.uint32()));
          break;
        case 7:
          message.conditions.push(RiskCondition.decode(reader, reader.uint32()));
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
      title: isSet(object.title) ? String(object.title) : "",
      description: isSet(object.description) ? String(object.description) : "",
      level: isSet(object.level) ? Number(object.level) : 0,
      actions: Array.isArray(object?.actions) ? object.actions.map((e: any) => RiskAction.fromJSON(e)) : [],
      conditions: Array.isArray(object?.conditions) ? object.conditions.map((e: any) => RiskCondition.fromJSON(e)) : [],
    };
  },

  toJSON(message: Risk): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.title !== undefined && (obj.title = message.title);
    message.description !== undefined && (obj.description = message.description);
    message.level !== undefined && (obj.level = Math.round(message.level));
    if (message.actions) {
      obj.actions = message.actions.map((e) => e ? RiskAction.toJSON(e) : undefined);
    } else {
      obj.actions = [];
    }
    if (message.conditions) {
      obj.conditions = message.conditions.map((e) => e ? RiskCondition.toJSON(e) : undefined);
    } else {
      obj.conditions = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<Risk>): Risk {
    const message = createBaseRisk();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.level = object.level ?? 0;
    message.actions = object.actions?.map((e) => RiskAction.fromPartial(e)) || [];
    message.conditions = object.conditions?.map((e) => RiskCondition.fromPartial(e)) || [];
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

function createBaseRiskCondition(): RiskCondition {
  return { name: "", uid: "", title: "", description: "", expression: undefined };
}

export const RiskCondition = {
  encode(message: RiskCondition, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.uid !== "") {
      writer.uint32(18).string(message.uid);
    }
    if (message.title !== "") {
      writer.uint32(26).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(34).string(message.description);
    }
    if (message.expression !== undefined) {
      ParsedExpr.encode(message.expression, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RiskCondition {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRiskCondition();
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
          message.title = reader.string();
          break;
        case 4:
          message.description = reader.string();
          break;
        case 5:
          message.expression = ParsedExpr.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): RiskCondition {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      uid: isSet(object.uid) ? String(object.uid) : "",
      title: isSet(object.title) ? String(object.title) : "",
      description: isSet(object.description) ? String(object.description) : "",
      expression: isSet(object.expression) ? ParsedExpr.fromJSON(object.expression) : undefined,
    };
  },

  toJSON(message: RiskCondition): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.title !== undefined && (obj.title = message.title);
    message.description !== undefined && (obj.description = message.description);
    message.expression !== undefined &&
      (obj.expression = message.expression ? ParsedExpr.toJSON(message.expression) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<RiskCondition>): RiskCondition {
    const message = createBaseRiskCondition();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.expression = (object.expression !== undefined && object.expression !== null)
      ? ParsedExpr.fromPartial(object.expression)
      : undefined;
    return message;
  },
};

export type RiskServiceDefinition = typeof RiskServiceDefinition;
export const RiskServiceDefinition = {
  name: "RiskService",
  fullName: "bytebase.v1.RiskService",
  methods: {
    getRisk: {
      name: "GetRisk",
      requestType: GetRiskRequest,
      requestStream: false,
      responseType: Risk,
      responseStream: false,
      options: {},
    },
    listRisks: {
      name: "ListRisks",
      requestType: ListRisksRequest,
      requestStream: false,
      responseType: ListRisksResponse,
      responseStream: false,
      options: {},
    },
    addRiskCondition: {
      name: "AddRiskCondition",
      requestType: AddRiskConditionRequest,
      requestStream: false,
      responseType: Risk,
      responseStream: false,
      options: {},
    },
    removeRiskCondition: {
      name: "RemoveRiskCondition",
      requestType: RemoveRiskConditionRequest,
      requestStream: false,
      responseType: Risk,
      responseStream: false,
      options: {},
    },
    updateRiskCondition: {
      name: "UpdateRiskCondition",
      requestType: UpdateRiskConditionRequest,
      requestStream: false,
      responseType: Risk,
      responseStream: false,
      options: {},
    },
  },
} as const;

export interface RiskServiceImplementation<CallContextExt = {}> {
  getRisk(request: GetRiskRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Risk>>;
  listRisks(request: ListRisksRequest, context: CallContext & CallContextExt): Promise<DeepPartial<ListRisksResponse>>;
  addRiskCondition(request: AddRiskConditionRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Risk>>;
  removeRiskCondition(
    request: RemoveRiskConditionRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Risk>>;
  updateRiskCondition(
    request: UpdateRiskConditionRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<Risk>>;
}

export interface RiskServiceClient<CallOptionsExt = {}> {
  getRisk(request: DeepPartial<GetRiskRequest>, options?: CallOptions & CallOptionsExt): Promise<Risk>;
  listRisks(request: DeepPartial<ListRisksRequest>, options?: CallOptions & CallOptionsExt): Promise<ListRisksResponse>;
  addRiskCondition(
    request: DeepPartial<AddRiskConditionRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Risk>;
  removeRiskCondition(
    request: DeepPartial<RemoveRiskConditionRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Risk>;
  updateRiskCondition(
    request: DeepPartial<UpdateRiskConditionRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<Risk>;
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
