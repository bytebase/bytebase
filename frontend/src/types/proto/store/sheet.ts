/* eslint-disable */
import * as Long from "long";
import * as _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export interface SheetPayload {
  vcsPayload?: SheetPayload_VCSPayload;
  /** used_by_issues link to the issues where the sheet is used. */
  usedByIssues: SheetPayload_UsedByIssue[];
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

function createBaseSheetPayload(): SheetPayload {
  return { vcsPayload: undefined, usedByIssues: [] };
}

export const SheetPayload = {
  encode(message: SheetPayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.vcsPayload !== undefined) {
      SheetPayload_VCSPayload.encode(message.vcsPayload, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.usedByIssues) {
      SheetPayload_UsedByIssue.encode(v!, writer.uint32(18).fork()).ldelim();
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
          if (tag !== 10) {
            break;
          }

          message.vcsPayload = SheetPayload_VCSPayload.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.usedByIssues.push(SheetPayload_UsedByIssue.decode(reader, reader.uint32()));
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
      vcsPayload: isSet(object.vcsPayload) ? SheetPayload_VCSPayload.fromJSON(object.vcsPayload) : undefined,
      usedByIssues: Array.isArray(object?.usedByIssues)
        ? object.usedByIssues.map((e: any) => SheetPayload_UsedByIssue.fromJSON(e))
        : [],
    };
  },

  toJSON(message: SheetPayload): unknown {
    const obj: any = {};
    message.vcsPayload !== undefined &&
      (obj.vcsPayload = message.vcsPayload ? SheetPayload_VCSPayload.toJSON(message.vcsPayload) : undefined);
    if (message.usedByIssues) {
      obj.usedByIssues = message.usedByIssues.map((e) => e ? SheetPayload_UsedByIssue.toJSON(e) : undefined);
    } else {
      obj.usedByIssues = [];
    }
    return obj;
  },

  create(base?: DeepPartial<SheetPayload>): SheetPayload {
    return SheetPayload.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SheetPayload>): SheetPayload {
    const message = createBaseSheetPayload();
    message.vcsPayload = (object.vcsPayload !== undefined && object.vcsPayload !== null)
      ? SheetPayload_VCSPayload.fromPartial(object.vcsPayload)
      : undefined;
    message.usedByIssues = object.usedByIssues?.map((e) => SheetPayload_UsedByIssue.fromPartial(e)) || [];
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
