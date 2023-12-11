/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Timestamp } from "../google/protobuf/timestamp";

export const protobufPackage = "bytebase.v1";

export enum PlanType {
  PLAN_TYPE_UNSPECIFIED = 0,
  FREE = 1,
  TEAM = 2,
  ENTERPRISE = 3,
  UNRECOGNIZED = -1,
}

export function planTypeFromJSON(object: any): PlanType {
  switch (object) {
    case 0:
    case "PLAN_TYPE_UNSPECIFIED":
      return PlanType.PLAN_TYPE_UNSPECIFIED;
    case 1:
    case "FREE":
      return PlanType.FREE;
    case 2:
    case "TEAM":
      return PlanType.TEAM;
    case 3:
    case "ENTERPRISE":
      return PlanType.ENTERPRISE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return PlanType.UNRECOGNIZED;
  }
}

export function planTypeToJSON(object: PlanType): string {
  switch (object) {
    case PlanType.PLAN_TYPE_UNSPECIFIED:
      return "PLAN_TYPE_UNSPECIFIED";
    case PlanType.FREE:
      return "FREE";
    case PlanType.TEAM:
      return "TEAM";
    case PlanType.ENTERPRISE:
      return "ENTERPRISE";
    case PlanType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface GetSubscriptionRequest {
}

export interface GetFeatureMatrixRequest {
}

export interface UpdateSubscriptionRequest {
  patch: PatchSubscription | undefined;
}

export interface PatchSubscription {
  license: string;
}

export interface Subscription {
  instanceCount: number;
  expiresTime: Date | undefined;
  startedTime: Date | undefined;
  plan: PlanType;
  trialing: boolean;
  orgId: string;
  orgName: string;
}

export interface FeatureMatrix {
  features: Feature[];
}

export interface Feature {
  /** Name is the feature name. */
  name: string;
  /** Matrix is the feature matrix for different plan. The key is the plan enum in string value. */
  matrix: { [key: string]: boolean };
}

export interface Feature_MatrixEntry {
  key: string;
  value: boolean;
}

function createBaseGetSubscriptionRequest(): GetSubscriptionRequest {
  return {};
}

export const GetSubscriptionRequest = {
  encode(_: GetSubscriptionRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetSubscriptionRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetSubscriptionRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): GetSubscriptionRequest {
    return {};
  },

  toJSON(_: GetSubscriptionRequest): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<GetSubscriptionRequest>): GetSubscriptionRequest {
    return GetSubscriptionRequest.fromPartial(base ?? {});
  },
  fromPartial(_: DeepPartial<GetSubscriptionRequest>): GetSubscriptionRequest {
    const message = createBaseGetSubscriptionRequest();
    return message;
  },
};

function createBaseGetFeatureMatrixRequest(): GetFeatureMatrixRequest {
  return {};
}

export const GetFeatureMatrixRequest = {
  encode(_: GetFeatureMatrixRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetFeatureMatrixRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetFeatureMatrixRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): GetFeatureMatrixRequest {
    return {};
  },

  toJSON(_: GetFeatureMatrixRequest): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<GetFeatureMatrixRequest>): GetFeatureMatrixRequest {
    return GetFeatureMatrixRequest.fromPartial(base ?? {});
  },
  fromPartial(_: DeepPartial<GetFeatureMatrixRequest>): GetFeatureMatrixRequest {
    const message = createBaseGetFeatureMatrixRequest();
    return message;
  },
};

function createBaseUpdateSubscriptionRequest(): UpdateSubscriptionRequest {
  return { patch: undefined };
}

export const UpdateSubscriptionRequest = {
  encode(message: UpdateSubscriptionRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.patch !== undefined) {
      PatchSubscription.encode(message.patch, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateSubscriptionRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateSubscriptionRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.patch = PatchSubscription.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UpdateSubscriptionRequest {
    return { patch: isSet(object.patch) ? PatchSubscription.fromJSON(object.patch) : undefined };
  },

  toJSON(message: UpdateSubscriptionRequest): unknown {
    const obj: any = {};
    if (message.patch !== undefined) {
      obj.patch = PatchSubscription.toJSON(message.patch);
    }
    return obj;
  },

  create(base?: DeepPartial<UpdateSubscriptionRequest>): UpdateSubscriptionRequest {
    return UpdateSubscriptionRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<UpdateSubscriptionRequest>): UpdateSubscriptionRequest {
    const message = createBaseUpdateSubscriptionRequest();
    message.patch = (object.patch !== undefined && object.patch !== null)
      ? PatchSubscription.fromPartial(object.patch)
      : undefined;
    return message;
  },
};

function createBasePatchSubscription(): PatchSubscription {
  return { license: "" };
}

export const PatchSubscription = {
  encode(message: PatchSubscription, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.license !== "") {
      writer.uint32(10).string(message.license);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PatchSubscription {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePatchSubscription();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.license = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PatchSubscription {
    return { license: isSet(object.license) ? globalThis.String(object.license) : "" };
  },

  toJSON(message: PatchSubscription): unknown {
    const obj: any = {};
    if (message.license !== "") {
      obj.license = message.license;
    }
    return obj;
  },

  create(base?: DeepPartial<PatchSubscription>): PatchSubscription {
    return PatchSubscription.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<PatchSubscription>): PatchSubscription {
    const message = createBasePatchSubscription();
    message.license = object.license ?? "";
    return message;
  },
};

function createBaseSubscription(): Subscription {
  return {
    instanceCount: 0,
    expiresTime: undefined,
    startedTime: undefined,
    plan: 0,
    trialing: false,
    orgId: "",
    orgName: "",
  };
}

export const Subscription = {
  encode(message: Subscription, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.instanceCount !== 0) {
      writer.uint32(16).int32(message.instanceCount);
    }
    if (message.expiresTime !== undefined) {
      Timestamp.encode(toTimestamp(message.expiresTime), writer.uint32(26).fork()).ldelim();
    }
    if (message.startedTime !== undefined) {
      Timestamp.encode(toTimestamp(message.startedTime), writer.uint32(34).fork()).ldelim();
    }
    if (message.plan !== 0) {
      writer.uint32(40).int32(message.plan);
    }
    if (message.trialing === true) {
      writer.uint32(48).bool(message.trialing);
    }
    if (message.orgId !== "") {
      writer.uint32(58).string(message.orgId);
    }
    if (message.orgName !== "") {
      writer.uint32(66).string(message.orgName);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Subscription {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSubscription();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 2:
          if (tag !== 16) {
            break;
          }

          message.instanceCount = reader.int32();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.expiresTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.startedTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.plan = reader.int32() as any;
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.trialing = reader.bool();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.orgId = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.orgName = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Subscription {
    return {
      instanceCount: isSet(object.instanceCount) ? globalThis.Number(object.instanceCount) : 0,
      expiresTime: isSet(object.expiresTime) ? fromJsonTimestamp(object.expiresTime) : undefined,
      startedTime: isSet(object.startedTime) ? fromJsonTimestamp(object.startedTime) : undefined,
      plan: isSet(object.plan) ? planTypeFromJSON(object.plan) : 0,
      trialing: isSet(object.trialing) ? globalThis.Boolean(object.trialing) : false,
      orgId: isSet(object.orgId) ? globalThis.String(object.orgId) : "",
      orgName: isSet(object.orgName) ? globalThis.String(object.orgName) : "",
    };
  },

  toJSON(message: Subscription): unknown {
    const obj: any = {};
    if (message.instanceCount !== 0) {
      obj.instanceCount = Math.round(message.instanceCount);
    }
    if (message.expiresTime !== undefined) {
      obj.expiresTime = message.expiresTime.toISOString();
    }
    if (message.startedTime !== undefined) {
      obj.startedTime = message.startedTime.toISOString();
    }
    if (message.plan !== 0) {
      obj.plan = planTypeToJSON(message.plan);
    }
    if (message.trialing === true) {
      obj.trialing = message.trialing;
    }
    if (message.orgId !== "") {
      obj.orgId = message.orgId;
    }
    if (message.orgName !== "") {
      obj.orgName = message.orgName;
    }
    return obj;
  },

  create(base?: DeepPartial<Subscription>): Subscription {
    return Subscription.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Subscription>): Subscription {
    const message = createBaseSubscription();
    message.instanceCount = object.instanceCount ?? 0;
    message.expiresTime = object.expiresTime ?? undefined;
    message.startedTime = object.startedTime ?? undefined;
    message.plan = object.plan ?? 0;
    message.trialing = object.trialing ?? false;
    message.orgId = object.orgId ?? "";
    message.orgName = object.orgName ?? "";
    return message;
  },
};

function createBaseFeatureMatrix(): FeatureMatrix {
  return { features: [] };
}

export const FeatureMatrix = {
  encode(message: FeatureMatrix, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.features) {
      Feature.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): FeatureMatrix {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFeatureMatrix();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.features.push(Feature.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): FeatureMatrix {
    return {
      features: globalThis.Array.isArray(object?.features) ? object.features.map((e: any) => Feature.fromJSON(e)) : [],
    };
  },

  toJSON(message: FeatureMatrix): unknown {
    const obj: any = {};
    if (message.features?.length) {
      obj.features = message.features.map((e) => Feature.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<FeatureMatrix>): FeatureMatrix {
    return FeatureMatrix.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<FeatureMatrix>): FeatureMatrix {
    const message = createBaseFeatureMatrix();
    message.features = object.features?.map((e) => Feature.fromPartial(e)) || [];
    return message;
  },
};

function createBaseFeature(): Feature {
  return { name: "", matrix: {} };
}

export const Feature = {
  encode(message: Feature, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    Object.entries(message.matrix).forEach(([key, value]) => {
      Feature_MatrixEntry.encode({ key: key as any, value }, writer.uint32(18).fork()).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Feature {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFeature();
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

          const entry2 = Feature_MatrixEntry.decode(reader, reader.uint32());
          if (entry2.value !== undefined) {
            message.matrix[entry2.key] = entry2.value;
          }
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Feature {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      matrix: isObject(object.matrix)
        ? Object.entries(object.matrix).reduce<{ [key: string]: boolean }>((acc, [key, value]) => {
          acc[key] = Boolean(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: Feature): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.matrix) {
      const entries = Object.entries(message.matrix);
      if (entries.length > 0) {
        obj.matrix = {};
        entries.forEach(([k, v]) => {
          obj.matrix[k] = v;
        });
      }
    }
    return obj;
  },

  create(base?: DeepPartial<Feature>): Feature {
    return Feature.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Feature>): Feature {
    const message = createBaseFeature();
    message.name = object.name ?? "";
    message.matrix = Object.entries(object.matrix ?? {}).reduce<{ [key: string]: boolean }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = globalThis.Boolean(value);
      }
      return acc;
    }, {});
    return message;
  },
};

function createBaseFeature_MatrixEntry(): Feature_MatrixEntry {
  return { key: "", value: false };
}

export const Feature_MatrixEntry = {
  encode(message: Feature_MatrixEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value === true) {
      writer.uint32(16).bool(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Feature_MatrixEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFeature_MatrixEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.value = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Feature_MatrixEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? globalThis.Boolean(object.value) : false,
    };
  },

  toJSON(message: Feature_MatrixEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value === true) {
      obj.value = message.value;
    }
    return obj;
  },

  create(base?: DeepPartial<Feature_MatrixEntry>): Feature_MatrixEntry {
    return Feature_MatrixEntry.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Feature_MatrixEntry>): Feature_MatrixEntry {
    const message = createBaseFeature_MatrixEntry();
    message.key = object.key ?? "";
    message.value = object.value ?? false;
    return message;
  },
};

export type SubscriptionServiceDefinition = typeof SubscriptionServiceDefinition;
export const SubscriptionServiceDefinition = {
  name: "SubscriptionService",
  fullName: "bytebase.v1.SubscriptionService",
  methods: {
    getSubscription: {
      name: "GetSubscription",
      requestType: GetSubscriptionRequest,
      requestStream: false,
      responseType: Subscription,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          578365826: [
            new Uint8Array([18, 18, 16, 47, 118, 49, 47, 115, 117, 98, 115, 99, 114, 105, 112, 116, 105, 111, 110]),
          ],
        },
      },
    },
    getFeatureMatrix: {
      name: "GetFeatureMatrix",
      requestType: GetFeatureMatrixRequest,
      requestStream: false,
      responseType: FeatureMatrix,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          578365826: [new Uint8Array([13, 18, 11, 47, 118, 49, 47, 102, 101, 97, 116, 117, 114, 101])],
        },
      },
    },
    updateSubscription: {
      name: "UpdateSubscription",
      requestType: UpdateSubscriptionRequest,
      requestStream: false,
      responseType: Subscription,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([5, 112, 97, 116, 99, 104])],
          578365826: [
            new Uint8Array([
              25,
              58,
              5,
              112,
              97,
              116,
              99,
              104,
              50,
              16,
              47,
              118,
              49,
              47,
              115,
              117,
              98,
              115,
              99,
              114,
              105,
              112,
              116,
              105,
              111,
              110,
            ]),
          ],
        },
      },
    },
  },
} as const;

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Long ? string | number | Long : T extends globalThis.Array<infer U> ? globalThis.Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function toTimestamp(date: Date): Timestamp {
  const seconds = numberToLong(date.getTime() / 1_000);
  const nanos = (date.getTime() % 1_000) * 1_000_000;
  return { seconds, nanos };
}

function fromTimestamp(t: Timestamp): Date {
  let millis = (t.seconds.toNumber() || 0) * 1_000;
  millis += (t.nanos || 0) / 1_000_000;
  return new globalThis.Date(millis);
}

function fromJsonTimestamp(o: any): Date {
  if (o instanceof globalThis.Date) {
    return o;
  } else if (typeof o === "string") {
    return new globalThis.Date(o);
  } else {
    return fromTimestamp(Timestamp.fromJSON(o));
  }
}

function numberToLong(number: number) {
  return Long.fromNumber(number);
}

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isObject(value: any): boolean {
  return typeof value === "object" && value !== null;
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
