/* eslint-disable */
import * as Long from "long";
import * as _m0 from "protobufjs/minimal";
import { Engine, engineFromJSON, engineToJSON } from "./common";

export const protobufPackage = "bytebase.store";

export interface SheetPayload {
  type: SheetPayload_Type;
  vcsPayload?:
    | SheetPayload_VCSPayload
    | undefined;
  /** used_by_issues link to the issues where the sheet is used. */
  usedByIssues: SheetPayload_UsedByIssue[];
  schemaDesign?: SheetPayload_SchemaDesign | undefined;
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
  size: number;
  author: string;
  lastCommitId: string;
  lastSyncTs: number;
}

export interface SheetPayload_UsedByIssue {
  issueId: number;
  issueTitle: string;
}

export interface SheetPayload_SchemaDesign {
  /** The type of the schema design. */
  type: SheetPayload_SchemaDesign_Type;
  /** The database instance engine of the schema design. */
  engine: Engine;
  /** The sheet id of the baseline schema. */
  baselineSchemaSheetId: string;
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
  return { type: 0, vcsPayload: undefined, usedByIssues: [], schemaDesign: undefined };
}

export const SheetPayload = {
  encode(message: SheetPayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.type !== 0) {
      writer.uint32(8).int32(message.type);
    }
    if (message.vcsPayload !== undefined) {
      SheetPayload_VCSPayload.encode(message.vcsPayload, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.usedByIssues) {
      SheetPayload_UsedByIssue.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    if (message.schemaDesign !== undefined) {
      SheetPayload_SchemaDesign.encode(message.schemaDesign, writer.uint32(34).fork()).ldelim();
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

          message.usedByIssues.push(SheetPayload_UsedByIssue.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.schemaDesign = SheetPayload_SchemaDesign.decode(reader, reader.uint32());
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
      usedByIssues: Array.isArray(object?.usedByIssues)
        ? object.usedByIssues.map((e: any) => SheetPayload_UsedByIssue.fromJSON(e))
        : [],
      schemaDesign: isSet(object.schemaDesign) ? SheetPayload_SchemaDesign.fromJSON(object.schemaDesign) : undefined,
    };
  },

  toJSON(message: SheetPayload): unknown {
    const obj: any = {};
    message.type !== undefined && (obj.type = sheetPayload_TypeToJSON(message.type));
    message.vcsPayload !== undefined &&
      (obj.vcsPayload = message.vcsPayload ? SheetPayload_VCSPayload.toJSON(message.vcsPayload) : undefined);
    if (message.usedByIssues) {
      obj.usedByIssues = message.usedByIssues.map((e) => e ? SheetPayload_UsedByIssue.toJSON(e) : undefined);
    } else {
      obj.usedByIssues = [];
    }
    message.schemaDesign !== undefined &&
      (obj.schemaDesign = message.schemaDesign ? SheetPayload_SchemaDesign.toJSON(message.schemaDesign) : undefined);
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
    message.usedByIssues = object.usedByIssues?.map((e) => SheetPayload_UsedByIssue.fromPartial(e)) || [];
    message.schemaDesign = (object.schemaDesign !== undefined && object.schemaDesign !== null)
      ? SheetPayload_SchemaDesign.fromPartial(object.schemaDesign)
      : undefined;
    return message;
  },
};

function createBaseSheetPayload_VCSPayload(): SheetPayload_VCSPayload {
  return { fileName: "", filePath: "", size: 0, author: "", lastCommitId: "", lastSyncTs: 0 };
}

export const SheetPayload_VCSPayload = {
  encode(message: SheetPayload_VCSPayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.fileName !== "") {
      writer.uint32(10).string(message.fileName);
    }
    if (message.filePath !== "") {
      writer.uint32(18).string(message.filePath);
    }
    if (message.size !== 0) {
      writer.uint32(24).int64(message.size);
    }
    if (message.author !== "") {
      writer.uint32(34).string(message.author);
    }
    if (message.lastCommitId !== "") {
      writer.uint32(42).string(message.lastCommitId);
    }
    if (message.lastSyncTs !== 0) {
      writer.uint32(48).int64(message.lastSyncTs);
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

          message.size = longToNumber(reader.int64() as Long);
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

          message.lastSyncTs = longToNumber(reader.int64() as Long);
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
      fileName: isSet(object.fileName) ? String(object.fileName) : "",
      filePath: isSet(object.filePath) ? String(object.filePath) : "",
      size: isSet(object.size) ? Number(object.size) : 0,
      author: isSet(object.author) ? String(object.author) : "",
      lastCommitId: isSet(object.lastCommitId) ? String(object.lastCommitId) : "",
      lastSyncTs: isSet(object.lastSyncTs) ? Number(object.lastSyncTs) : 0,
    };
  },

  toJSON(message: SheetPayload_VCSPayload): unknown {
    const obj: any = {};
    message.fileName !== undefined && (obj.fileName = message.fileName);
    message.filePath !== undefined && (obj.filePath = message.filePath);
    message.size !== undefined && (obj.size = Math.round(message.size));
    message.author !== undefined && (obj.author = message.author);
    message.lastCommitId !== undefined && (obj.lastCommitId = message.lastCommitId);
    message.lastSyncTs !== undefined && (obj.lastSyncTs = Math.round(message.lastSyncTs));
    return obj;
  },

  create(base?: DeepPartial<SheetPayload_VCSPayload>): SheetPayload_VCSPayload {
    return SheetPayload_VCSPayload.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SheetPayload_VCSPayload>): SheetPayload_VCSPayload {
    const message = createBaseSheetPayload_VCSPayload();
    message.fileName = object.fileName ?? "";
    message.filePath = object.filePath ?? "";
    message.size = object.size ?? 0;
    message.author = object.author ?? "";
    message.lastCommitId = object.lastCommitId ?? "";
    message.lastSyncTs = object.lastSyncTs ?? 0;
    return message;
  },
};

function createBaseSheetPayload_UsedByIssue(): SheetPayload_UsedByIssue {
  return { issueId: 0, issueTitle: "" };
}

export const SheetPayload_UsedByIssue = {
  encode(message: SheetPayload_UsedByIssue, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.issueId !== 0) {
      writer.uint32(8).int64(message.issueId);
    }
    if (message.issueTitle !== "") {
      writer.uint32(18).string(message.issueTitle);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SheetPayload_UsedByIssue {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSheetPayload_UsedByIssue();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.issueId = longToNumber(reader.int64() as Long);
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.issueTitle = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SheetPayload_UsedByIssue {
    return {
      issueId: isSet(object.issueId) ? Number(object.issueId) : 0,
      issueTitle: isSet(object.issueTitle) ? String(object.issueTitle) : "",
    };
  },

  toJSON(message: SheetPayload_UsedByIssue): unknown {
    const obj: any = {};
    message.issueId !== undefined && (obj.issueId = Math.round(message.issueId));
    message.issueTitle !== undefined && (obj.issueTitle = message.issueTitle);
    return obj;
  },

  create(base?: DeepPartial<SheetPayload_UsedByIssue>): SheetPayload_UsedByIssue {
    return SheetPayload_UsedByIssue.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SheetPayload_UsedByIssue>): SheetPayload_UsedByIssue {
    const message = createBaseSheetPayload_UsedByIssue();
    message.issueId = object.issueId ?? 0;
    message.issueTitle = object.issueTitle ?? "";
    return message;
  },
};

function createBaseSheetPayload_SchemaDesign(): SheetPayload_SchemaDesign {
  return { type: 0, engine: 0, baselineSchemaSheetId: "" };
}

export const SheetPayload_SchemaDesign = {
  encode(message: SheetPayload_SchemaDesign, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.type !== 0) {
      writer.uint32(8).int32(message.type);
    }
    if (message.engine !== 0) {
      writer.uint32(16).int32(message.engine);
    }
    if (message.baselineSchemaSheetId !== "") {
      writer.uint32(26).string(message.baselineSchemaSheetId);
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

          message.baselineSchemaSheetId = reader.string();
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
      baselineSchemaSheetId: isSet(object.baselineSchemaSheetId) ? String(object.baselineSchemaSheetId) : "",
    };
  },

  toJSON(message: SheetPayload_SchemaDesign): unknown {
    const obj: any = {};
    message.type !== undefined && (obj.type = sheetPayload_SchemaDesign_TypeToJSON(message.type));
    message.engine !== undefined && (obj.engine = engineToJSON(message.engine));
    message.baselineSchemaSheetId !== undefined && (obj.baselineSchemaSheetId = message.baselineSchemaSheetId);
    return obj;
  },

  create(base?: DeepPartial<SheetPayload_SchemaDesign>): SheetPayload_SchemaDesign {
    return SheetPayload_SchemaDesign.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SheetPayload_SchemaDesign>): SheetPayload_SchemaDesign {
    const message = createBaseSheetPayload_SchemaDesign();
    message.type = object.type ?? 0;
    message.engine = object.engine ?? 0;
    message.baselineSchemaSheetId = object.baselineSchemaSheetId ?? "";
    return message;
  },
};

declare const self: any | undefined;
declare const window: any | undefined;
declare const global: any | undefined;
const tsProtoGlobalThis: any = (() => {
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
