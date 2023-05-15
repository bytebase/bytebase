/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { FieldMask } from "../google/protobuf/field_mask";

export const protobufPackage = "bytebase.v1";

export interface GetPlanRequest {
  /**
   * The name of the plan to retrieve.
   * Format: projects/{project}/plans/{plan}
   */
  name: string;
}

export interface ListPlansRequest {
  /**
   * The parent, which owns this collection of plans.
   * Format: projects/{project}
   * Use "projects/-" to list all plans from all projects.
   */
  parent: string;
  /**
   * The maximum number of plans to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 plans will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListPlans` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListPlans` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface ListPlansResponse {
  /** The plans from the specified request. */
  plans: Plan[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface UpdatePlanRequest {
  /**
   * The plan to update.
   *
   * The plan's `name` field is used to identify the plan to update.
   * Format: projects/{project}/plans/{plan}
   */
  plan?: Plan;
  /** The list of fields to update. */
  updateMask?: string[];
}

export interface Plan {
  /**
   * The name of the plan.
   * `plan` is a system generated ID.
   * Format: projects/{project}/plans/{plan}
   */
  name: string;
  /** The system-assigned, unique identifier for a resource. */
  uid: string;
  title: string;
  description: string;
}

function createBaseGetPlanRequest(): GetPlanRequest {
  return { name: "" };
}

export const GetPlanRequest = {
  encode(message: GetPlanRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetPlanRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetPlanRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GetPlanRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetPlanRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<GetPlanRequest>): GetPlanRequest {
    return GetPlanRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<GetPlanRequest>): GetPlanRequest {
    const message = createBaseGetPlanRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListPlansRequest(): ListPlansRequest {
  return { parent: "", pageSize: 0, pageToken: "" };
}

export const ListPlansRequest = {
  encode(message: ListPlansRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.pageSize !== 0) {
      writer.uint32(16).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(26).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListPlansRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListPlansRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.parent = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.pageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListPlansRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListPlansRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    return obj;
  },

  create(base?: DeepPartial<ListPlansRequest>): ListPlansRequest {
    return ListPlansRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListPlansRequest>): ListPlansRequest {
    const message = createBaseListPlansRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListPlansResponse(): ListPlansResponse {
  return { plans: [], nextPageToken: "" };
}

export const ListPlansResponse = {
  encode(message: ListPlansResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.plans) {
      Plan.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListPlansResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListPlansResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.plans.push(Plan.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.nextPageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListPlansResponse {
    return {
      plans: Array.isArray(object?.plans) ? object.plans.map((e: any) => Plan.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListPlansResponse): unknown {
    const obj: any = {};
    if (message.plans) {
      obj.plans = message.plans.map((e) => e ? Plan.toJSON(e) : undefined);
    } else {
      obj.plans = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  create(base?: DeepPartial<ListPlansResponse>): ListPlansResponse {
    return ListPlansResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListPlansResponse>): ListPlansResponse {
    const message = createBaseListPlansResponse();
    message.plans = object.plans?.map((e) => Plan.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseUpdatePlanRequest(): UpdatePlanRequest {
  return { plan: undefined, updateMask: undefined };
}

export const UpdatePlanRequest = {
  encode(message: UpdatePlanRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.plan !== undefined) {
      Plan.encode(message.plan, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdatePlanRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdatePlanRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.plan = Plan.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UpdatePlanRequest {
    return {
      plan: isSet(object.plan) ? Plan.fromJSON(object.plan) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdatePlanRequest): unknown {
    const obj: any = {};
    message.plan !== undefined && (obj.plan = message.plan ? Plan.toJSON(message.plan) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  create(base?: DeepPartial<UpdatePlanRequest>): UpdatePlanRequest {
    return UpdatePlanRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<UpdatePlanRequest>): UpdatePlanRequest {
    const message = createBaseUpdatePlanRequest();
    message.plan = (object.plan !== undefined && object.plan !== null) ? Plan.fromPartial(object.plan) : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBasePlan(): Plan {
  return { name: "", uid: "", title: "", description: "" };
}

export const Plan = {
  encode(message: Plan, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Plan {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlan();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.uid = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.title = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.description = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Plan {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      uid: isSet(object.uid) ? String(object.uid) : "",
      title: isSet(object.title) ? String(object.title) : "",
      description: isSet(object.description) ? String(object.description) : "",
    };
  },

  toJSON(message: Plan): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.title !== undefined && (obj.title = message.title);
    message.description !== undefined && (obj.description = message.description);
    return obj;
  },

  create(base?: DeepPartial<Plan>): Plan {
    return Plan.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Plan>): Plan {
    const message = createBasePlan();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    return message;
  },
};

export type RolloutServiceDefinition = typeof RolloutServiceDefinition;
export const RolloutServiceDefinition = {
  name: "RolloutService",
  fullName: "bytebase.v1.RolloutService",
  methods: {
    getPlan: {
      name: "GetPlan",
      requestType: GetPlanRequest,
      requestStream: false,
      responseType: Plan,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              31,
              18,
              29,
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
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              47,
              112,
              108,
              97,
              110,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    listPlans: {
      name: "ListPlans",
      requestType: ListPlansRequest,
      requestStream: false,
      responseType: ListPlansResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([6, 112, 97, 114, 101, 110, 116])],
          578365826: [
            new Uint8Array([
              31,
              18,
              29,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              125,
              47,
              112,
              108,
              97,
              110,
              115,
            ]),
          ],
        },
      },
    },
    updatePlan: {
      name: "UpdatePlan",
      requestType: UpdatePlanRequest,
      requestStream: false,
      responseType: Plan,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([16, 112, 108, 97, 110, 44, 117, 112, 100, 97, 116, 101, 95, 109, 97, 115, 107])],
          578365826: [
            new Uint8Array([
              42,
              58,
              4,
              112,
              108,
              97,
              110,
              50,
              34,
              47,
              118,
              49,
              47,
              123,
              112,
              108,
              97,
              110,
              46,
              110,
              97,
              109,
              101,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              47,
              112,
              108,
              97,
              110,
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

export interface RolloutServiceImplementation<CallContextExt = {}> {
  getPlan(request: GetPlanRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Plan>>;
  listPlans(request: ListPlansRequest, context: CallContext & CallContextExt): Promise<DeepPartial<ListPlansResponse>>;
  updatePlan(request: UpdatePlanRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Plan>>;
}

export interface RolloutServiceClient<CallOptionsExt = {}> {
  getPlan(request: DeepPartial<GetPlanRequest>, options?: CallOptions & CallOptionsExt): Promise<Plan>;
  listPlans(request: DeepPartial<ListPlansRequest>, options?: CallOptions & CallOptionsExt): Promise<ListPlansResponse>;
  updatePlan(request: DeepPartial<UpdatePlanRequest>, options?: CallOptions & CallOptionsExt): Promise<Plan>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
