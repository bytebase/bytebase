/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { Expr } from "../google/type/expr";
import { MaskingLevel, maskingLevelFromJSON, maskingLevelToJSON } from "./common";

export const protobufPackage = "bytebase.store";

export interface IamPolicy {
  /** Collection of binding. */
  bindings: Binding[];
}

/** Reference: https://cloud.google.com/pubsub/docs/reference/rpc/google.iam.v1#binding */
export interface Binding {
  /**
   * Role that is assigned to the list of members.
   * Format: roles/{role}
   */
  role: string;
  /**
   * Specifies the principals requesting access for a Bytebase resource.
   * `members` can have the following values:
   *
   * * `allUsers`: A special identifier that represents anyone.
   * * `user:{emailid}`: An email address that represents a specific Bytebase account. For example, `alice@example.com`.
   */
  members: string[];
  /**
   * The condition that is associated with this binding.
   * If the condition evaluates to true, then this binding applies to the current request.
   * If the condition evaluates to false, then this binding does not apply to the current request. However, a different role binding might grant the same role to one or more of the principals in this binding.
   */
  condition?: Expr | undefined;
}

export interface MaskingPolicy {
  maskData: MaskData[];
}

export interface MaskData {
  schema: string;
  table: string;
  column: string;
  semanticCategoryId: string;
  maskingLevel: MaskingLevel;
}

function createBaseIamPolicy(): IamPolicy {
  return { bindings: [] };
}

export const IamPolicy = {
  encode(message: IamPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.bindings) {
      Binding.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IamPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIamPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.bindings.push(Binding.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): IamPolicy {
    return { bindings: Array.isArray(object?.bindings) ? object.bindings.map((e: any) => Binding.fromJSON(e)) : [] };
  },

  toJSON(message: IamPolicy): unknown {
    const obj: any = {};
    if (message.bindings?.length) {
      obj.bindings = message.bindings.map((e) => Binding.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<IamPolicy>): IamPolicy {
    return IamPolicy.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<IamPolicy>): IamPolicy {
    const message = createBaseIamPolicy();
    message.bindings = object.bindings?.map((e) => Binding.fromPartial(e)) || [];
    return message;
  },
};

function createBaseBinding(): Binding {
  return { role: "", members: [], condition: undefined };
}

export const Binding = {
  encode(message: Binding, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.role !== "") {
      writer.uint32(10).string(message.role);
    }
    for (const v of message.members) {
      writer.uint32(18).string(v!);
    }
    if (message.condition !== undefined) {
      Expr.encode(message.condition, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Binding {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBinding();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.role = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.members.push(reader.string());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.condition = Expr.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Binding {
    return {
      role: isSet(object.role) ? String(object.role) : "",
      members: Array.isArray(object?.members) ? object.members.map((e: any) => String(e)) : [],
      condition: isSet(object.condition) ? Expr.fromJSON(object.condition) : undefined,
    };
  },

  toJSON(message: Binding): unknown {
    const obj: any = {};
    if (message.role !== "") {
      obj.role = message.role;
    }
    if (message.members?.length) {
      obj.members = message.members;
    }
    if (message.condition !== undefined) {
      obj.condition = Expr.toJSON(message.condition);
    }
    return obj;
  },

  create(base?: DeepPartial<Binding>): Binding {
    return Binding.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Binding>): Binding {
    const message = createBaseBinding();
    message.role = object.role ?? "";
    message.members = object.members?.map((e) => e) || [];
    message.condition = (object.condition !== undefined && object.condition !== null)
      ? Expr.fromPartial(object.condition)
      : undefined;
    return message;
  },
};

function createBaseMaskingPolicy(): MaskingPolicy {
  return { maskData: [] };
}

export const MaskingPolicy = {
  encode(message: MaskingPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.maskData) {
      MaskData.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MaskingPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMaskingPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.maskData.push(MaskData.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MaskingPolicy {
    return { maskData: Array.isArray(object?.maskData) ? object.maskData.map((e: any) => MaskData.fromJSON(e)) : [] };
  },

  toJSON(message: MaskingPolicy): unknown {
    const obj: any = {};
    if (message.maskData?.length) {
      obj.maskData = message.maskData.map((e) => MaskData.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<MaskingPolicy>): MaskingPolicy {
    return MaskingPolicy.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<MaskingPolicy>): MaskingPolicy {
    const message = createBaseMaskingPolicy();
    message.maskData = object.maskData?.map((e) => MaskData.fromPartial(e)) || [];
    return message;
  },
};

function createBaseMaskData(): MaskData {
  return { schema: "", table: "", column: "", semanticCategoryId: "", maskingLevel: 0 };
}

export const MaskData = {
  encode(message: MaskData, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schema !== "") {
      writer.uint32(10).string(message.schema);
    }
    if (message.table !== "") {
      writer.uint32(18).string(message.table);
    }
    if (message.column !== "") {
      writer.uint32(26).string(message.column);
    }
    if (message.semanticCategoryId !== "") {
      writer.uint32(34).string(message.semanticCategoryId);
    }
    if (message.maskingLevel !== 0) {
      writer.uint32(40).int32(message.maskingLevel);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MaskData {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMaskData();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.schema = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.table = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.column = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.semanticCategoryId = reader.string();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.maskingLevel = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MaskData {
    return {
      schema: isSet(object.schema) ? String(object.schema) : "",
      table: isSet(object.table) ? String(object.table) : "",
      column: isSet(object.column) ? String(object.column) : "",
      semanticCategoryId: isSet(object.semanticCategoryId) ? String(object.semanticCategoryId) : "",
      maskingLevel: isSet(object.maskingLevel) ? maskingLevelFromJSON(object.maskingLevel) : 0,
    };
  },

  toJSON(message: MaskData): unknown {
    const obj: any = {};
    if (message.schema !== "") {
      obj.schema = message.schema;
    }
    if (message.table !== "") {
      obj.table = message.table;
    }
    if (message.column !== "") {
      obj.column = message.column;
    }
    if (message.semanticCategoryId !== "") {
      obj.semanticCategoryId = message.semanticCategoryId;
    }
    if (message.maskingLevel !== 0) {
      obj.maskingLevel = maskingLevelToJSON(message.maskingLevel);
    }
    return obj;
  },

  create(base?: DeepPartial<MaskData>): MaskData {
    return MaskData.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<MaskData>): MaskData {
    const message = createBaseMaskData();
    message.schema = object.schema ?? "";
    message.table = object.table ?? "";
    message.column = object.column ?? "";
    message.semanticCategoryId = object.semanticCategoryId ?? "";
    message.maskingLevel = object.maskingLevel ?? 0;
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
