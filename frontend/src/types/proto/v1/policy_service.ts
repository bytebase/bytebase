/* eslint-disable */
import _m0 from "protobufjs/minimal";
import { State, stateFromJSON, stateToJSON } from "./common";

export const protobufPackage = "bytebase.v1";

export enum PolicyType {
  POLICY_TYPE_UNSPECIFIED = 0,
  PIPELINE_APPROVAL = 1,
  BACKUP_PLAN = 2,
  SQL_REVIEW = 3,
  ENVIRONMENT_TIER = 4,
  SENSITIVE_DATA = 5,
  ACCESS_CONTROL = 6,
  UNRECOGNIZED = -1,
}

export function policyTypeFromJSON(object: any): PolicyType {
  switch (object) {
    case 0:
    case "POLICY_TYPE_UNSPECIFIED":
      return PolicyType.POLICY_TYPE_UNSPECIFIED;
    case 1:
    case "PIPELINE_APPROVAL":
      return PolicyType.PIPELINE_APPROVAL;
    case 2:
    case "BACKUP_PLAN":
      return PolicyType.BACKUP_PLAN;
    case 3:
    case "SQL_REVIEW":
      return PolicyType.SQL_REVIEW;
    case 4:
    case "ENVIRONMENT_TIER":
      return PolicyType.ENVIRONMENT_TIER;
    case 5:
    case "SENSITIVE_DATA":
      return PolicyType.SENSITIVE_DATA;
    case 6:
    case "ACCESS_CONTROL":
      return PolicyType.ACCESS_CONTROL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return PolicyType.UNRECOGNIZED;
  }
}

export function policyTypeToJSON(object: PolicyType): string {
  switch (object) {
    case PolicyType.POLICY_TYPE_UNSPECIFIED:
      return "POLICY_TYPE_UNSPECIFIED";
    case PolicyType.PIPELINE_APPROVAL:
      return "PIPELINE_APPROVAL";
    case PolicyType.BACKUP_PLAN:
      return "BACKUP_PLAN";
    case PolicyType.SQL_REVIEW:
      return "SQL_REVIEW";
    case PolicyType.ENVIRONMENT_TIER:
      return "ENVIRONMENT_TIER";
    case PolicyType.SENSITIVE_DATA:
      return "SENSITIVE_DATA";
    case PolicyType.ACCESS_CONTROL:
      return "ACCESS_CONTROL";
    case PolicyType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface GetPolicyRequest {
  /**
   * The name of the policy to retrieve.
   * Format: policies/{policy type}/{policy resource id}
   */
  name: string;
}

export interface ListPoliciesRequest {
  /**
   * The parent, which owns this collection of policies.
   * Format: policies/{policy type}
   */
  parent: string;
  /**
   * The maximum number of policies to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 policies will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `GetPolicies` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `GetPolicies` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface ListPoliciesResponse {
  /** The policies from the specified request. */
  policies: Policy[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface Policy {
  /**
   * The name of the policy.
   * Format: policies/{policy type}/{policy resource id}
   */
  name: string;
  /** The system-assigned, unique identifier for a resource. */
  uid: string;
  state: State;
  inheritFromParent: boolean;
  type: PolicyType;
  payload: string;
}

function createBaseGetPolicyRequest(): GetPolicyRequest {
  return { name: "" };
}

export const GetPolicyRequest = {
  encode(message: GetPolicyRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetPolicyRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetPolicyRequest();
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

  fromJSON(object: any): GetPolicyRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetPolicyRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<GetPolicyRequest>, I>>(object: I): GetPolicyRequest {
    const message = createBaseGetPolicyRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListPoliciesRequest(): ListPoliciesRequest {
  return { parent: "", pageSize: 0, pageToken: "" };
}

export const ListPoliciesRequest = {
  encode(message: ListPoliciesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): ListPoliciesRequest {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListPoliciesRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.parent = reader.string();
          break;
        case 2:
          message.pageSize = reader.int32();
          break;
        case 3:
          message.pageToken = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ListPoliciesRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListPoliciesRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ListPoliciesRequest>, I>>(object: I): ListPoliciesRequest {
    const message = createBaseListPoliciesRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListPoliciesResponse(): ListPoliciesResponse {
  return { policies: [], nextPageToken: "" };
}

export const ListPoliciesResponse = {
  encode(message: ListPoliciesResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.policies) {
      Policy.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListPoliciesResponse {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListPoliciesResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.policies.push(Policy.decode(reader, reader.uint32()));
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

  fromJSON(object: any): ListPoliciesResponse {
    return {
      policies: Array.isArray(object?.policies) ? object.policies.map((e: any) => Policy.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListPoliciesResponse): unknown {
    const obj: any = {};
    if (message.policies) {
      obj.policies = message.policies.map((e) => e ? Policy.toJSON(e) : undefined);
    } else {
      obj.policies = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<ListPoliciesResponse>, I>>(object: I): ListPoliciesResponse {
    const message = createBaseListPoliciesResponse();
    message.policies = object.policies?.map((e) => Policy.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBasePolicy(): Policy {
  return { name: "", uid: "", state: 0, inheritFromParent: false, type: 0, payload: "" };
}

export const Policy = {
  encode(message: Policy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.uid !== "") {
      writer.uint32(18).string(message.uid);
    }
    if (message.state !== 0) {
      writer.uint32(24).int32(message.state);
    }
    if (message.inheritFromParent === true) {
      writer.uint32(32).bool(message.inheritFromParent);
    }
    if (message.type !== 0) {
      writer.uint32(40).int32(message.type);
    }
    if (message.payload !== "") {
      writer.uint32(50).string(message.payload);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Policy {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePolicy();
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
          message.state = reader.int32() as any;
          break;
        case 4:
          message.inheritFromParent = reader.bool();
          break;
        case 5:
          message.type = reader.int32() as any;
          break;
        case 6:
          message.payload = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Policy {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      uid: isSet(object.uid) ? String(object.uid) : "",
      state: isSet(object.state) ? stateFromJSON(object.state) : 0,
      inheritFromParent: isSet(object.inheritFromParent) ? Boolean(object.inheritFromParent) : false,
      type: isSet(object.type) ? policyTypeFromJSON(object.type) : 0,
      payload: isSet(object.payload) ? String(object.payload) : "",
    };
  },

  toJSON(message: Policy): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.uid !== undefined && (obj.uid = message.uid);
    message.state !== undefined && (obj.state = stateToJSON(message.state));
    message.inheritFromParent !== undefined && (obj.inheritFromParent = message.inheritFromParent);
    message.type !== undefined && (obj.type = policyTypeToJSON(message.type));
    message.payload !== undefined && (obj.payload = message.payload);
    return obj;
  },

  fromPartial<I extends Exact<DeepPartial<Policy>, I>>(object: I): Policy {
    const message = createBasePolicy();
    message.name = object.name ?? "";
    message.uid = object.uid ?? "";
    message.state = object.state ?? 0;
    message.inheritFromParent = object.inheritFromParent ?? false;
    message.type = object.type ?? 0;
    message.payload = object.payload ?? "";
    return message;
  },
};

export interface PolicyService {
  GetPolicy(request: GetPolicyRequest): Promise<Policy>;
  ListPolicies(request: ListPoliciesRequest): Promise<ListPoliciesResponse>;
}

export class PolicyServiceClientImpl implements PolicyService {
  private readonly rpc: Rpc;
  private readonly service: string;
  constructor(rpc: Rpc, opts?: { service?: string }) {
    this.service = opts?.service || "bytebase.v1.PolicyService";
    this.rpc = rpc;
    this.GetPolicy = this.GetPolicy.bind(this);
    this.ListPolicies = this.ListPolicies.bind(this);
  }
  GetPolicy(request: GetPolicyRequest): Promise<Policy> {
    const data = GetPolicyRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "GetPolicy", data);
    return promise.then((data) => Policy.decode(new _m0.Reader(data)));
  }

  ListPolicies(request: ListPoliciesRequest): Promise<ListPoliciesResponse> {
    const data = ListPoliciesRequest.encode(request).finish();
    const promise = this.rpc.request(this.service, "ListPolicies", data);
    return promise.then((data) => ListPoliciesResponse.decode(new _m0.Reader(data)));
  }
}

interface Rpc {
  request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

type KeysOfUnion<T> = T extends T ? keyof T : never;
export type Exact<P, I extends P> = P extends Builtin ? P
  : P & { [K in keyof P]: Exact<P[K], I[K]> } & { [K in Exclude<keyof I, KeysOfUnion<P>>]: never };

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
