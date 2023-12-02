/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { DatabaseConfig, DatabaseSchemaMetadata } from "./database";

export const protobufPackage = "bytebase.store";

export interface BranchSnapshot {
  schema: Uint8Array;
  metadata: DatabaseSchemaMetadata | undefined;
  databaseConfig: DatabaseConfig | undefined;
}

export interface BranchConfig {
  /**
   * The name of source database.
   * Optional.
   * Example: instances/instance-id/databases/database-name.
   */
  sourceDatabase: string;
  /**
   * The name of the source branch.
   * Optional.
   * Example: projects/project-id/branches/branch-id.
   */
  sourceBranch: string;
}

function createBaseBranchSnapshot(): BranchSnapshot {
  return { schema: new Uint8Array(0), metadata: undefined, databaseConfig: undefined };
}

export const BranchSnapshot = {
  encode(message: BranchSnapshot, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schema.length !== 0) {
      writer.uint32(10).bytes(message.schema);
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

          message.schema = reader.bytes();
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
      schema: isSet(object.schema) ? bytesFromBase64(object.schema) : new Uint8Array(0),
      metadata: isSet(object.metadata) ? DatabaseSchemaMetadata.fromJSON(object.metadata) : undefined,
      databaseConfig: isSet(object.databaseConfig) ? DatabaseConfig.fromJSON(object.databaseConfig) : undefined,
    };
  },

  toJSON(message: BranchSnapshot): unknown {
    const obj: any = {};
    if (message.schema.length !== 0) {
      obj.schema = base64FromBytes(message.schema);
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
    message.schema = object.schema ?? new Uint8Array(0);
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
  return { sourceDatabase: "", sourceBranch: "" };
}

export const BranchConfig = {
  encode(message: BranchConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sourceDatabase !== "") {
      writer.uint32(10).string(message.sourceDatabase);
    }
    if (message.sourceBranch !== "") {
      writer.uint32(18).string(message.sourceBranch);
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
          if (tag !== 10) {
            break;
          }

          message.sourceDatabase = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.sourceBranch = reader.string();
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
      sourceDatabase: isSet(object.sourceDatabase) ? globalThis.String(object.sourceDatabase) : "",
      sourceBranch: isSet(object.sourceBranch) ? globalThis.String(object.sourceBranch) : "",
    };
  },

  toJSON(message: BranchConfig): unknown {
    const obj: any = {};
    if (message.sourceDatabase !== "") {
      obj.sourceDatabase = message.sourceDatabase;
    }
    if (message.sourceBranch !== "") {
      obj.sourceBranch = message.sourceBranch;
    }
    return obj;
  },

  create(base?: DeepPartial<BranchConfig>): BranchConfig {
    return BranchConfig.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<BranchConfig>): BranchConfig {
    const message = createBaseBranchConfig();
    message.sourceDatabase = object.sourceDatabase ?? "";
    message.sourceBranch = object.sourceBranch ?? "";
    return message;
  },
};

function bytesFromBase64(b64: string): Uint8Array {
  if (globalThis.Buffer) {
    return Uint8Array.from(globalThis.Buffer.from(b64, "base64"));
  } else {
    const bin = globalThis.atob(b64);
    const arr = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; ++i) {
      arr[i] = bin.charCodeAt(i);
    }
    return arr;
  }
}

function base64FromBytes(arr: Uint8Array): string {
  if (globalThis.Buffer) {
    return globalThis.Buffer.from(arr).toString("base64");
  } else {
    const bin: string[] = [];
    arr.forEach((byte) => {
      bin.push(globalThis.String.fromCharCode(byte));
    });
    return globalThis.btoa(bin.join(""));
  }
}

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
