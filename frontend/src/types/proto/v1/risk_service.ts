/* eslint-disable */
import * as Long from "long";
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { ParsedExpr } from "../google/api/expr/v1alpha1/syntax";
import { Empty } from "../google/protobuf/empty";
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

export interface CreateRiskRequest {
  /** The risk to create. */
  risk?: Risk;
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

export interface DeleteRiskRequest {
  /**
   * The name of the risk to delete.
   * Format: risks/{risk}
   */
  name: string;
}

export interface Risk {
  /** Format: risks/{risk} */
  name: string;
  /** system-generated unique identifier. */
  uid: string;
  source: Risk_Source;
  title: string;
  level: number;
  expression?: ParsedExpr;
  active: boolean;
}

export enum Risk_Source {
  SOURCE_UNSPECIFIED = 0,
  DDL = 1,
  DML = 2,
  CREATE_DATABASE = 3,
  UNRECOGNIZED = -1,
}

export function risk_SourceFromJSON(object: any): Risk_Source {
  switch (object) {
    case 0:
    case "SOURCE_UNSPECIFIED":
      return Risk_Source.SOURCE_UNSPECIFIED;
    case 1:
    case "DDL":
      return Risk_Source.DDL;
    case 2:
    case "DML":
      return Risk_Source.DML;
    case 3:
    case "CREATE_DATABASE":
      return Risk_Source.CREATE_DATABASE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Risk_Source.UNRECOGNIZED;
  }
}

export function risk_SourceToJSON(object: Risk_Source): string {
  switch (object) {
    case Risk_Source.SOURCE_UNSPECIFIED:
      return "SOURCE_UNSPECIFIED";
    case Risk_Source.DDL:
      return "DDL";
    case Risk_Source.DML:
      return "DML";
    case Risk_Source.CREATE_DATABASE:
      return "CREATE_DATABASE";
    case Risk_Source.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListRisksRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 8) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.pageToken = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<ListRisksRequest>): ListRisksRequest {
    return ListRisksRequest.fromPartial(base ?? {});
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListRisksResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.risks.push(Risk.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.nextPageToken = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<ListRisksResponse>): ListRisksResponse {
    return ListRisksResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListRisksResponse>): ListRisksResponse {
    const message = createBaseListRisksResponse();
    message.risks = object.risks?.map((e) => Risk.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseCreateRiskRequest(): CreateRiskRequest {
  return { risk: undefined };
}

export const CreateRiskRequest = {
  encode(message: CreateRiskRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.risk !== undefined) {
      Risk.encode(message.risk, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateRiskRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateRiskRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.risk = Risk.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateRiskRequest {
    return { risk: isSet(object.risk) ? Risk.fromJSON(object.risk) : undefined };
  },

  toJSON(message: CreateRiskRequest): unknown {
    const obj: any = {};
    message.risk !== undefined && (obj.risk = message.risk ? Risk.toJSON(message.risk) : undefined);
    return obj;
  },

  create(base?: DeepPartial<CreateRiskRequest>): CreateRiskRequest {
    return CreateRiskRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<CreateRiskRequest>): CreateRiskRequest {
    const message = createBaseCreateRiskRequest();
    message.risk = (object.risk !== undefined && object.risk !== null) ? Risk.fromPartial(object.risk) : undefined;
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
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateRiskRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.risk = Risk.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
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

  create(base?: DeepPartial<UpdateRiskRequest>): UpdateRiskRequest {
    return UpdateRiskRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<UpdateRiskRequest>): UpdateRiskRequest {
    const message = createBaseUpdateRiskRequest();
    message.risk = (object.risk !== undefined && object.risk !== null) ? Risk.fromPartial(object.risk) : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseDeleteRiskRequest(): DeleteRiskRequest {
  return { name: "" };
}

export const DeleteRiskRequest = {
  encode(message: DeleteRiskRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteRiskRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteRiskRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DeleteRiskRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: DeleteRiskRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<DeleteRiskRequest>): DeleteRiskRequest {
    return DeleteRiskRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<DeleteRiskRequest>): DeleteRiskRequest {
    const message = createBaseDeleteRiskRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseRisk(): Risk {
  return { name: "", uid: "", source: 0, title: "", level: 0, expression: undefined, active: false };
}

export const Risk = {
  encode(message: Risk, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.uid !== "") {
      writer.uint32(18).string(message.uid);
    }
    if (message.source !== 0) {
      writer.uint32(24).int32(message.source);
    }
    if (message.title !== "") {
      writer.uint32(34).string(message.title);
    }
    if (message.level !== 0) {
      writer.uint32(40).int64(message.level);
    }
    if (message.expression !== undefined) {
      ParsedExpr.encode(message.expression, writer.uint32(50).fork()).ldelim();
    }
    if (message.active === true) {
      writer.uint32(56).bool(message.active);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Risk {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRisk();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.uid = reader.string();
          continue;
        case 3:
          if (tag != 24) {
            break;
          }

          message.source = reader.int32() as any;
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.title = reader.string();
          continue;
        case 5:
          if (tag != 40) {
            break;
          }

          message.level = longToNumber(reader.int64() as Long);
          continue;
        case 6:
          if (tag != 50) {
            break;
          }

          message.expression = ParsedExpr.decode(reader, reader.uint32());
          continue;
        case 7:
          if (tag != 56) {
            break;
          }

          message.active = reader.bool();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Risk {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      uid: isSet(object.uid) ? String(object.uid) : "",
      source: isSet(object.source) ? risk_SourceFromJSON(object.source) : 0,
      title: isSet(object.title) ? String(object.title) : "",
      level: isSet(object.level) ? Number(object.level) : 0,
      expression: isSet(object.expression) ? ParsedExpr.fromJSON(object.expression) : undefined,
      active: isSet(object.active) ? Boolean(object.active) : false,
    };
  },

  toJSON(message: Risk): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.source !== undefined && (obj.source = risk_SourceToJSON(message.source));
    message.title !== undefined && (obj.title = message.title);
    message.level !== undefined && (obj.level = Math.round(message.level));
    message.expression !== undefined &&
      (obj.expression = message.expression ? ParsedExpr.toJSON(message.expression) : undefined);
    message.active !== undefined && (obj.active = message.active);
    return obj;
  },

  create(base?: DeepPartial<Risk>): Risk {
    return Risk.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Risk>): Risk {
    const message = createBaseRisk();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.source = object.source ?? 0;
    message.title = object.title ?? "";
    message.level = object.level ?? 0;
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
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          578365826: [new Uint8Array([11, 18, 9, 47, 118, 49, 47, 114, 105, 115, 107, 115])],
        },
      },
    },
    createRisk: {
      name: "CreateRisk",
      requestType: CreateRiskRequest,
      requestStream: false,
      responseType: Risk,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 114, 105, 115, 107])],
          578365826: [new Uint8Array([17, 58, 4, 114, 105, 115, 107, 34, 9, 47, 118, 49, 47, 114, 105, 115, 107, 115])],
        },
      },
    },
    updateRisk: {
      name: "UpdateRisk",
      requestType: UpdateRiskRequest,
      requestStream: false,
      responseType: Risk,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([16, 114, 105, 115, 107, 44, 117, 112, 100, 97, 116, 101, 95, 109, 97, 115, 107])],
          578365826: [
            new Uint8Array([
              31,
              58,
              4,
              114,
              105,
              115,
              107,
              50,
              23,
              47,
              118,
              49,
              47,
              123,
              114,
              105,
              115,
              107,
              46,
              110,
              97,
              109,
              101,
              61,
              114,
              105,
              115,
              107,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    deleteRisk: {
      name: "DeleteRisk",
      requestType: DeleteRiskRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              20,
              42,
              18,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              114,
              105,
              115,
              107,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
  },
} as const;

export interface RiskServiceImplementation<CallContextExt = {}> {
  listRisks(request: ListRisksRequest, context: CallContext & CallContextExt): Promise<DeepPartial<ListRisksResponse>>;
  createRisk(request: CreateRiskRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Risk>>;
  updateRisk(request: UpdateRiskRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Risk>>;
  deleteRisk(request: DeleteRiskRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Empty>>;
}

export interface RiskServiceClient<CallOptionsExt = {}> {
  listRisks(request: DeepPartial<ListRisksRequest>, options?: CallOptions & CallOptionsExt): Promise<ListRisksResponse>;
  createRisk(request: DeepPartial<CreateRiskRequest>, options?: CallOptions & CallOptionsExt): Promise<Risk>;
  updateRisk(request: DeepPartial<UpdateRiskRequest>, options?: CallOptions & CallOptionsExt): Promise<Risk>;
  deleteRisk(request: DeepPartial<DeleteRiskRequest>, options?: CallOptions & CallOptionsExt): Promise<Empty>;
}

declare var self: any | undefined;
declare var window: any | undefined;
declare var global: any | undefined;
var tsProtoGlobalThis: any = (() => {
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
    throw new tsProtoGlobalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
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
