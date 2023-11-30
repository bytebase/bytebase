/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { DatabaseConfig, DatabaseSchemaMetadata } from "./database";

export const protobufPackage = "bytebase.store";

export interface BranchSnapshot {
  schema: string;
  metadata: DatabaseSchemaMetadata | undefined;
  databaseConfig: DatabaseConfig | undefined;
}

export interface BranchConfig {
  /**
   * The id of source database.
   * Optional.
   */
  sourceDatabaseId: Long;
  /**
   * The id of the source branch.
   * Optional.
   */
  sourceBranchId: Long;
}

function createBaseBranchSnapshot(): BranchSnapshot {
  return { schema: "", metadata: undefined, databaseConfig: undefined };
}

export const BranchSnapshot = {
  encode(message: BranchSnapshot, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schema !== "") {
      writer.uint32(10).string(message.schema);
    }
    if (message.metadata !== undefined) {
      DatabaseSchemaMetadata.encode(message.metadata, writer.uint32(18).fork()).ldelim();
    }
    if (message.databaseConfig !== undefined) {
      DatabaseConfig.encode(message.databaseConfig, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BranchSnapshot {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBranchSnapshot();
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

          message.metadata = DatabaseSchemaMetadata.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.databaseConfig = DatabaseConfig.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BranchSnapshot {
    return {
      schema: isSet(object.schema) ? globalThis.String(object.schema) : "",
      metadata: isSet(object.metadata) ? DatabaseSchemaMetadata.fromJSON(object.metadata) : undefined,
      databaseConfig: isSet(object.databaseConfig) ? DatabaseConfig.fromJSON(object.databaseConfig) : undefined,
    };
  },

  toJSON(message: BranchSnapshot): unknown {
    const obj: any = {};
    if (message.schema !== "") {
      obj.schema = message.schema;
    }
    if (message.metadata !== undefined) {
      obj.metadata = DatabaseSchemaMetadata.toJSON(message.metadata);
    }
    if (message.databaseConfig !== undefined) {
      obj.databaseConfig = DatabaseConfig.toJSON(message.databaseConfig);
    }
    return obj;
  },

  create(base?: DeepPartial<BranchSnapshot>): BranchSnapshot {
    return BranchSnapshot.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BranchSnapshot>): BranchSnapshot {
    const message = createBaseBranchSnapshot();
    message.schema = object.schema ?? "";
    message.metadata = (object.metadata !== undefined && object.metadata !== null)
      ? DatabaseSchemaMetadata.fromPartial(object.metadata)
      : undefined;
    message.databaseConfig = (object.databaseConfig !== undefined && object.databaseConfig !== null)
      ? DatabaseConfig.fromPartial(object.databaseConfig)
      : undefined;
    return message;
  },
};

function createBaseBranchConfig(): BranchConfig {
  return { sourceDatabaseId: Long.ZERO, sourceBranchId: Long.ZERO };
}

export const BranchConfig = {
  encode(message: BranchConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (!message.sourceDatabaseId.isZero()) {
      writer.uint32(8).int64(message.sourceDatabaseId);
    }
    if (!message.sourceBranchId.isZero()) {
      writer.uint32(16).int64(message.sourceBranchId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): BranchConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBranchConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.sourceDatabaseId = reader.int64() as Long;
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.sourceBranchId = reader.int64() as Long;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): BranchConfig {
    return {
      sourceDatabaseId: isSet(object.sourceDatabaseId) ? Long.fromValue(object.sourceDatabaseId) : Long.ZERO,
      sourceBranchId: isSet(object.sourceBranchId) ? Long.fromValue(object.sourceBranchId) : Long.ZERO,
    };
  },

  toJSON(message: BranchConfig): unknown {
    const obj: any = {};
    if (!message.sourceDatabaseId.isZero()) {
      obj.sourceDatabaseId = (message.sourceDatabaseId || Long.ZERO).toString();
    }
    if (!message.sourceBranchId.isZero()) {
      obj.sourceBranchId = (message.sourceBranchId || Long.ZERO).toString();
    }
    return obj;
  },

  create(base?: DeepPartial<BranchConfig>): BranchConfig {
    return BranchConfig.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BranchConfig>): BranchConfig {
    const message = createBaseBranchConfig();
    message.sourceDatabaseId = (object.sourceDatabaseId !== undefined && object.sourceDatabaseId !== null)
      ? Long.fromValue(object.sourceDatabaseId)
      : Long.ZERO;
    message.sourceBranchId = (object.sourceBranchId !== undefined && object.sourceBranchId !== null)
      ? Long.fromValue(object.sourceBranchId)
      : Long.ZERO;
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Long ? string | number | Long : T extends globalThis.Array<infer U> ? globalThis.Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
