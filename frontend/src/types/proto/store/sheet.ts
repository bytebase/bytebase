/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Engine, engineFromJSON, engineToJSON } from "./common";
import { DatabaseConfig } from "./database";
import { PushEvent } from "./vcs";

export const protobufPackage = "bytebase.store";

export interface SheetPayload {
  type: SheetPayload_Type;
  vcsPayload: SheetPayload_VCSPayload | undefined;
  schemaDesign:
    | SheetPayload_SchemaDesign
    | undefined;
  /** The snapshot of the database config when creating the sheet, be used to compare with the baseline_database_config and apply the diff to the database. */
  databaseConfig:
    | DatabaseConfig
    | undefined;
  /** The snapshot of the baseline database config when creating the sheet. */
  baselineDatabaseConfig: DatabaseConfig | undefined;
}

/** Type of the SheetPayload. */
export enum SheetPayload_Type {
  TYPE_UNSPECIFIED = 0,
  SCHEMA_DESIGN = 1,
  UNRECOGNIZED = -1,
}

export function sheetPayload_TypeFromJSON(object: any): SheetPayload_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return SheetPayload_Type.TYPE_UNSPECIFIED;
    case 1:
    case "SCHEMA_DESIGN":
      return SheetPayload_Type.SCHEMA_DESIGN;
    case -1:
    case "UNRECOGNIZED":
    default:
      return SheetPayload_Type.UNRECOGNIZED;
  }
}

export function sheetPayload_TypeToJSON(object: SheetPayload_Type): string {
  switch (object) {
    case SheetPayload_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case SheetPayload_Type.SCHEMA_DESIGN:
      return "SCHEMA_DESIGN";
    case SheetPayload_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface SheetPayload_VCSPayload {
  fileName: string;
  filePath: string;
  size: Long;
  author: string;
  lastCommitId: string;
  lastSyncTs: Long;
  pushEvent: PushEvent | undefined;
}

export interface SheetPayload_SchemaDesign {
  /** The type of the schema design. */
  type: SheetPayload_SchemaDesign_Type;
  /** The database instance engine of the schema design. */
  engine: Engine;
  /** The id of the baseline sheet including the baseline full schema. */
  baselineSheetId: string;
  /** The sheet id of the baseline schema design. Only valid when the schema design is a personal draft. */
  baselineSchemaDesignId: string;
}

export enum SheetPayload_SchemaDesign_Type {
  TYPE_UNSPECIFIED = 0,
  /** MAIN_BRANCH - Main branch type is the main version of schema design. And only allow to be updated/merged with personal drafts. */
  MAIN_BRANCH = 1,
  /** PERSONAL_DRAFT - Personal draft type is a copy of the main branch type schema designs. */
  PERSONAL_DRAFT = 2,
  UNRECOGNIZED = -1,
}

export function sheetPayload_SchemaDesign_TypeFromJSON(object: any): SheetPayload_SchemaDesign_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return SheetPayload_SchemaDesign_Type.TYPE_UNSPECIFIED;
    case 1:
    case "MAIN_BRANCH":
      return SheetPayload_SchemaDesign_Type.MAIN_BRANCH;
    case 2:
    case "PERSONAL_DRAFT":
      return SheetPayload_SchemaDesign_Type.PERSONAL_DRAFT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return SheetPayload_SchemaDesign_Type.UNRECOGNIZED;
  }
}

export function sheetPayload_SchemaDesign_TypeToJSON(object: SheetPayload_SchemaDesign_Type): string {
  switch (object) {
    case SheetPayload_SchemaDesign_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case SheetPayload_SchemaDesign_Type.MAIN_BRANCH:
      return "MAIN_BRANCH";
    case SheetPayload_SchemaDesign_Type.PERSONAL_DRAFT:
      return "PERSONAL_DRAFT";
    case SheetPayload_SchemaDesign_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

function createBaseSheetPayload(): SheetPayload {
  return {
    type: 0,
    vcsPayload: undefined,
    schemaDesign: undefined,
    databaseConfig: undefined,
    baselineDatabaseConfig: undefined,
  };
}

export const SheetPayload = {
  encode(message: SheetPayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.type !== 0) {
      writer.uint32(8).int32(message.type);
    }
    if (message.vcsPayload !== undefined) {
      SheetPayload_VCSPayload.encode(message.vcsPayload, writer.uint32(18).fork()).ldelim();
    }
    if (message.schemaDesign !== undefined) {
      SheetPayload_SchemaDesign.encode(message.schemaDesign, writer.uint32(26).fork()).ldelim();
    }
    if (message.databaseConfig !== undefined) {
      DatabaseConfig.encode(message.databaseConfig, writer.uint32(34).fork()).ldelim();
    }
    if (message.baselineDatabaseConfig !== undefined) {
      DatabaseConfig.encode(message.baselineDatabaseConfig, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SheetPayload {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSheetPayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.type = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.vcsPayload = SheetPayload_VCSPayload.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.schemaDesign = SheetPayload_SchemaDesign.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.databaseConfig = DatabaseConfig.decode(reader, reader.uint32());
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.baselineDatabaseConfig = DatabaseConfig.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SheetPayload {
    return {
      type: isSet(object.type) ? sheetPayload_TypeFromJSON(object.type) : 0,
      vcsPayload: isSet(object.vcsPayload) ? SheetPayload_VCSPayload.fromJSON(object.vcsPayload) : undefined,
      schemaDesign: isSet(object.schemaDesign) ? SheetPayload_SchemaDesign.fromJSON(object.schemaDesign) : undefined,
      databaseConfig: isSet(object.databaseConfig) ? DatabaseConfig.fromJSON(object.databaseConfig) : undefined,
      baselineDatabaseConfig: isSet(object.baselineDatabaseConfig)
        ? DatabaseConfig.fromJSON(object.baselineDatabaseConfig)
        : undefined,
    };
  },

  toJSON(message: SheetPayload): unknown {
    const obj: any = {};
    if (message.type !== 0) {
      obj.type = sheetPayload_TypeToJSON(message.type);
    }
    if (message.vcsPayload !== undefined) {
      obj.vcsPayload = SheetPayload_VCSPayload.toJSON(message.vcsPayload);
    }
    if (message.schemaDesign !== undefined) {
      obj.schemaDesign = SheetPayload_SchemaDesign.toJSON(message.schemaDesign);
    }
    if (message.databaseConfig !== undefined) {
      obj.databaseConfig = DatabaseConfig.toJSON(message.databaseConfig);
    }
    if (message.baselineDatabaseConfig !== undefined) {
      obj.baselineDatabaseConfig = DatabaseConfig.toJSON(message.baselineDatabaseConfig);
    }
    return obj;
  },

  create(base?: DeepPartial<SheetPayload>): SheetPayload {
    return SheetPayload.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SheetPayload>): SheetPayload {
    const message = createBaseSheetPayload();
    message.type = object.type ?? 0;
    message.vcsPayload = (object.vcsPayload !== undefined && object.vcsPayload !== null)
      ? SheetPayload_VCSPayload.fromPartial(object.vcsPayload)
      : undefined;
    message.schemaDesign = (object.schemaDesign !== undefined && object.schemaDesign !== null)
      ? SheetPayload_SchemaDesign.fromPartial(object.schemaDesign)
      : undefined;
    message.databaseConfig = (object.databaseConfig !== undefined && object.databaseConfig !== null)
      ? DatabaseConfig.fromPartial(object.databaseConfig)
      : undefined;
    message.baselineDatabaseConfig =
      (object.baselineDatabaseConfig !== undefined && object.baselineDatabaseConfig !== null)
        ? DatabaseConfig.fromPartial(object.baselineDatabaseConfig)
        : undefined;
    return message;
  },
};

function createBaseSheetPayload_VCSPayload(): SheetPayload_VCSPayload {
  return {
    fileName: "",
    filePath: "",
    size: Long.ZERO,
    author: "",
    lastCommitId: "",
    lastSyncTs: Long.ZERO,
    pushEvent: undefined,
  };
}

export const SheetPayload_VCSPayload = {
  encode(message: SheetPayload_VCSPayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.fileName !== "") {
      writer.uint32(10).string(message.fileName);
    }
    if (message.filePath !== "") {
      writer.uint32(18).string(message.filePath);
    }
    if (!message.size.isZero()) {
      writer.uint32(24).int64(message.size);
    }
    if (message.author !== "") {
      writer.uint32(34).string(message.author);
    }
    if (message.lastCommitId !== "") {
      writer.uint32(42).string(message.lastCommitId);
    }
    if (!message.lastSyncTs.isZero()) {
      writer.uint32(48).int64(message.lastSyncTs);
    }
    if (message.pushEvent !== undefined) {
      PushEvent.encode(message.pushEvent, writer.uint32(58).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SheetPayload_VCSPayload {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSheetPayload_VCSPayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.fileName = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.filePath = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.size = reader.int64() as Long;
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.author = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.lastCommitId = reader.string();
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.lastSyncTs = reader.int64() as Long;
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.pushEvent = PushEvent.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SheetPayload_VCSPayload {
    return {
      fileName: isSet(object.fileName) ? globalThis.String(object.fileName) : "",
      filePath: isSet(object.filePath) ? globalThis.String(object.filePath) : "",
      size: isSet(object.size) ? Long.fromValue(object.size) : Long.ZERO,
      author: isSet(object.author) ? globalThis.String(object.author) : "",
      lastCommitId: isSet(object.lastCommitId) ? globalThis.String(object.lastCommitId) : "",
      lastSyncTs: isSet(object.lastSyncTs) ? Long.fromValue(object.lastSyncTs) : Long.ZERO,
      pushEvent: isSet(object.pushEvent) ? PushEvent.fromJSON(object.pushEvent) : undefined,
    };
  },

  toJSON(message: SheetPayload_VCSPayload): unknown {
    const obj: any = {};
    if (message.fileName !== "") {
      obj.fileName = message.fileName;
    }
    if (message.filePath !== "") {
      obj.filePath = message.filePath;
    }
    if (!message.size.isZero()) {
      obj.size = (message.size || Long.ZERO).toString();
    }
    if (message.author !== "") {
      obj.author = message.author;
    }
    if (message.lastCommitId !== "") {
      obj.lastCommitId = message.lastCommitId;
    }
    if (!message.lastSyncTs.isZero()) {
      obj.lastSyncTs = (message.lastSyncTs || Long.ZERO).toString();
    }
    if (message.pushEvent !== undefined) {
      obj.pushEvent = PushEvent.toJSON(message.pushEvent);
    }
    return obj;
  },

  create(base?: DeepPartial<SheetPayload_VCSPayload>): SheetPayload_VCSPayload {
    return SheetPayload_VCSPayload.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SheetPayload_VCSPayload>): SheetPayload_VCSPayload {
    const message = createBaseSheetPayload_VCSPayload();
    message.fileName = object.fileName ?? "";
    message.filePath = object.filePath ?? "";
    message.size = (object.size !== undefined && object.size !== null) ? Long.fromValue(object.size) : Long.ZERO;
    message.author = object.author ?? "";
    message.lastCommitId = object.lastCommitId ?? "";
    message.lastSyncTs = (object.lastSyncTs !== undefined && object.lastSyncTs !== null)
      ? Long.fromValue(object.lastSyncTs)
      : Long.ZERO;
    message.pushEvent = (object.pushEvent !== undefined && object.pushEvent !== null)
      ? PushEvent.fromPartial(object.pushEvent)
      : undefined;
    return message;
  },
};

function createBaseSheetPayload_SchemaDesign(): SheetPayload_SchemaDesign {
  return { type: 0, engine: 0, baselineSheetId: "", baselineSchemaDesignId: "" };
}

export const SheetPayload_SchemaDesign = {
  encode(message: SheetPayload_SchemaDesign, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.type !== 0) {
      writer.uint32(8).int32(message.type);
    }
    if (message.engine !== 0) {
      writer.uint32(16).int32(message.engine);
    }
    if (message.baselineSheetId !== "") {
      writer.uint32(26).string(message.baselineSheetId);
    }
    if (message.baselineSchemaDesignId !== "") {
      writer.uint32(34).string(message.baselineSchemaDesignId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SheetPayload_SchemaDesign {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSheetPayload_SchemaDesign();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.type = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.engine = reader.int32() as any;
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.baselineSheetId = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.baselineSchemaDesignId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SheetPayload_SchemaDesign {
    return {
      type: isSet(object.type) ? sheetPayload_SchemaDesign_TypeFromJSON(object.type) : 0,
      engine: isSet(object.engine) ? engineFromJSON(object.engine) : 0,
      baselineSheetId: isSet(object.baselineSheetId) ? globalThis.String(object.baselineSheetId) : "",
      baselineSchemaDesignId: isSet(object.baselineSchemaDesignId)
        ? globalThis.String(object.baselineSchemaDesignId)
        : "",
    };
  },

  toJSON(message: SheetPayload_SchemaDesign): unknown {
    const obj: any = {};
    if (message.type !== 0) {
      obj.type = sheetPayload_SchemaDesign_TypeToJSON(message.type);
    }
    if (message.engine !== 0) {
      obj.engine = engineToJSON(message.engine);
    }
    if (message.baselineSheetId !== "") {
      obj.baselineSheetId = message.baselineSheetId;
    }
    if (message.baselineSchemaDesignId !== "") {
      obj.baselineSchemaDesignId = message.baselineSchemaDesignId;
    }
    return obj;
  },

  create(base?: DeepPartial<SheetPayload_SchemaDesign>): SheetPayload_SchemaDesign {
    return SheetPayload_SchemaDesign.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SheetPayload_SchemaDesign>): SheetPayload_SchemaDesign {
    const message = createBaseSheetPayload_SchemaDesign();
    message.type = object.type ?? 0;
    message.engine = object.engine ?? 0;
    message.baselineSheetId = object.baselineSheetId ?? "";
    message.baselineSchemaDesignId = object.baselineSchemaDesignId ?? "";
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
