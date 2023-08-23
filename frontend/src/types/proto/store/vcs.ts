/* eslint-disable */
import * as Long from "long";
import * as _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export enum VcsType {
  VCS_TYPE_UNSPECIFIED = 0,
  GITLAB = 1,
  GITHUB = 2,
  BITBUCKET = 3,
  UNRECOGNIZED = -1,
}

export function vcsTypeFromJSON(object: any): VcsType {
  switch (object) {
    case 0:
    case "VCS_TYPE_UNSPECIFIED":
      return VcsType.VCS_TYPE_UNSPECIFIED;
    case 1:
    case "GITLAB":
      return VcsType.GITLAB;
    case 2:
    case "GITHUB":
      return VcsType.GITHUB;
    case 3:
    case "BITBUCKET":
      return VcsType.BITBUCKET;
    case -1:
    case "UNRECOGNIZED":
    default:
      return VcsType.UNRECOGNIZED;
  }
}

export function vcsTypeToJSON(object: VcsType): string {
  switch (object) {
    case VcsType.VCS_TYPE_UNSPECIFIED:
      return "VCS_TYPE_UNSPECIFIED";
    case VcsType.GITLAB:
      return "GITLAB";
    case VcsType.GITHUB:
      return "GITHUB";
    case VcsType.BITBUCKET:
      return "BITBUCKET";
    case VcsType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface PushEvent {
  vcsType: VcsType;
  baseDir: string;
  ref: string;
  before: string;
  after: string;
  repositoryId: string;
  repositoryUrl: string;
  repositoryFullPath: string;
  authorName: string;
  commits: Commit[];
  fileCommit?: FileCommit | undefined;
}

export interface Commit {
  id: string;
  title: string;
  message: string;
  createdTs: number;
  url: string;
  authorName: string;
  authorEmail: string;
  addedList: string[];
  modifiedList: string[];
}

export interface FileCommit {
  id: string;
  title: string;
  message: string;
  createdTs: number;
  url: string;
  authorName: string;
  authorEmail: string;
  added: string;
}

function createBasePushEvent(): PushEvent {
  return {
    vcsType: 0,
    baseDir: "",
    ref: "",
    before: "",
    after: "",
    repositoryId: "",
    repositoryUrl: "",
    repositoryFullPath: "",
    authorName: "",
    commits: [],
    fileCommit: undefined,
  };
}

export const PushEvent = {
  encode(message: PushEvent, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.vcsType !== 0) {
      writer.uint32(8).int32(message.vcsType);
    }
    if (message.baseDir !== "") {
      writer.uint32(18).string(message.baseDir);
    }
    if (message.ref !== "") {
      writer.uint32(26).string(message.ref);
    }
    if (message.before !== "") {
      writer.uint32(34).string(message.before);
    }
    if (message.after !== "") {
      writer.uint32(42).string(message.after);
    }
    if (message.repositoryId !== "") {
      writer.uint32(50).string(message.repositoryId);
    }
    if (message.repositoryUrl !== "") {
      writer.uint32(58).string(message.repositoryUrl);
    }
    if (message.repositoryFullPath !== "") {
      writer.uint32(66).string(message.repositoryFullPath);
    }
    if (message.authorName !== "") {
      writer.uint32(74).string(message.authorName);
    }
    for (const v of message.commits) {
      Commit.encode(v!, writer.uint32(82).fork()).ldelim();
    }
    if (message.fileCommit !== undefined) {
      FileCommit.encode(message.fileCommit, writer.uint32(90).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PushEvent {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePushEvent();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.vcsType = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.baseDir = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.ref = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.before = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.after = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.repositoryId = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.repositoryUrl = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.repositoryFullPath = reader.string();
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.authorName = reader.string();
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.commits.push(Commit.decode(reader, reader.uint32()));
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.fileCommit = FileCommit.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PushEvent {
    return {
      vcsType: isSet(object.vcsType) ? vcsTypeFromJSON(object.vcsType) : 0,
      baseDir: isSet(object.baseDir) ? String(object.baseDir) : "",
      ref: isSet(object.ref) ? String(object.ref) : "",
      before: isSet(object.before) ? String(object.before) : "",
      after: isSet(object.after) ? String(object.after) : "",
      repositoryId: isSet(object.repositoryId) ? String(object.repositoryId) : "",
      repositoryUrl: isSet(object.repositoryUrl) ? String(object.repositoryUrl) : "",
      repositoryFullPath: isSet(object.repositoryFullPath) ? String(object.repositoryFullPath) : "",
      authorName: isSet(object.authorName) ? String(object.authorName) : "",
      commits: Array.isArray(object?.commits) ? object.commits.map((e: any) => Commit.fromJSON(e)) : [],
      fileCommit: isSet(object.fileCommit) ? FileCommit.fromJSON(object.fileCommit) : undefined,
    };
  },

  toJSON(message: PushEvent): unknown {
    const obj: any = {};
    message.vcsType !== undefined && (obj.vcsType = vcsTypeToJSON(message.vcsType));
    message.baseDir !== undefined && (obj.baseDir = message.baseDir);
    message.ref !== undefined && (obj.ref = message.ref);
    message.before !== undefined && (obj.before = message.before);
    message.after !== undefined && (obj.after = message.after);
    message.repositoryId !== undefined && (obj.repositoryId = message.repositoryId);
    message.repositoryUrl !== undefined && (obj.repositoryUrl = message.repositoryUrl);
    message.repositoryFullPath !== undefined && (obj.repositoryFullPath = message.repositoryFullPath);
    message.authorName !== undefined && (obj.authorName = message.authorName);
    if (message.commits) {
      obj.commits = message.commits.map((e) => e ? Commit.toJSON(e) : undefined);
    } else {
      obj.commits = [];
    }
    message.fileCommit !== undefined &&
      (obj.fileCommit = message.fileCommit ? FileCommit.toJSON(message.fileCommit) : undefined);
    return obj;
  },

  create(base?: DeepPartial<PushEvent>): PushEvent {
    return PushEvent.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PushEvent>): PushEvent {
    const message = createBasePushEvent();
    message.vcsType = object.vcsType ?? 0;
    message.baseDir = object.baseDir ?? "";
    message.ref = object.ref ?? "";
    message.before = object.before ?? "";
    message.after = object.after ?? "";
    message.repositoryId = object.repositoryId ?? "";
    message.repositoryUrl = object.repositoryUrl ?? "";
    message.repositoryFullPath = object.repositoryFullPath ?? "";
    message.authorName = object.authorName ?? "";
    message.commits = object.commits?.map((e) => Commit.fromPartial(e)) || [];
    message.fileCommit = (object.fileCommit !== undefined && object.fileCommit !== null)
      ? FileCommit.fromPartial(object.fileCommit)
      : undefined;
    return message;
  },
};

function createBaseCommit(): Commit {
  return {
    id: "",
    title: "",
    message: "",
    createdTs: 0,
    url: "",
    authorName: "",
    authorEmail: "",
    addedList: [],
    modifiedList: [],
  };
}

export const Commit = {
  encode(message: Commit, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.message !== "") {
      writer.uint32(26).string(message.message);
    }
    if (message.createdTs !== 0) {
      writer.uint32(32).int64(message.createdTs);
    }
    if (message.url !== "") {
      writer.uint32(42).string(message.url);
    }
    if (message.authorName !== "") {
      writer.uint32(50).string(message.authorName);
    }
    if (message.authorEmail !== "") {
      writer.uint32(58).string(message.authorEmail);
    }
    for (const v of message.addedList) {
      writer.uint32(66).string(v!);
    }
    for (const v of message.modifiedList) {
      writer.uint32(74).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Commit {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCommit();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.id = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.title = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.message = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.createdTs = longToNumber(reader.int64() as Long);
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.url = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.authorName = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.authorEmail = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.addedList.push(reader.string());
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.modifiedList.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Commit {
    return {
      id: isSet(object.id) ? String(object.id) : "",
      title: isSet(object.title) ? String(object.title) : "",
      message: isSet(object.message) ? String(object.message) : "",
      createdTs: isSet(object.createdTs) ? Number(object.createdTs) : 0,
      url: isSet(object.url) ? String(object.url) : "",
      authorName: isSet(object.authorName) ? String(object.authorName) : "",
      authorEmail: isSet(object.authorEmail) ? String(object.authorEmail) : "",
      addedList: Array.isArray(object?.addedList) ? object.addedList.map((e: any) => String(e)) : [],
      modifiedList: Array.isArray(object?.modifiedList) ? object.modifiedList.map((e: any) => String(e)) : [],
    };
  },

  toJSON(message: Commit): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.title !== undefined && (obj.title = message.title);
    message.message !== undefined && (obj.message = message.message);
    message.createdTs !== undefined && (obj.createdTs = Math.round(message.createdTs));
    message.url !== undefined && (obj.url = message.url);
    message.authorName !== undefined && (obj.authorName = message.authorName);
    message.authorEmail !== undefined && (obj.authorEmail = message.authorEmail);
    if (message.addedList) {
      obj.addedList = message.addedList.map((e) => e);
    } else {
      obj.addedList = [];
    }
    if (message.modifiedList) {
      obj.modifiedList = message.modifiedList.map((e) => e);
    } else {
      obj.modifiedList = [];
    }
    return obj;
  },

  create(base?: DeepPartial<Commit>): Commit {
    return Commit.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Commit>): Commit {
    const message = createBaseCommit();
    message.id = object.id ?? "";
    message.title = object.title ?? "";
    message.message = object.message ?? "";
    message.createdTs = object.createdTs ?? 0;
    message.url = object.url ?? "";
    message.authorName = object.authorName ?? "";
    message.authorEmail = object.authorEmail ?? "";
    message.addedList = object.addedList?.map((e) => e) || [];
    message.modifiedList = object.modifiedList?.map((e) => e) || [];
    return message;
  },
};

function createBaseFileCommit(): FileCommit {
  return { id: "", title: "", message: "", createdTs: 0, url: "", authorName: "", authorEmail: "", added: "" };
}

export const FileCommit = {
  encode(message: FileCommit, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.message !== "") {
      writer.uint32(26).string(message.message);
    }
    if (message.createdTs !== 0) {
      writer.uint32(32).int64(message.createdTs);
    }
    if (message.url !== "") {
      writer.uint32(42).string(message.url);
    }
    if (message.authorName !== "") {
      writer.uint32(50).string(message.authorName);
    }
    if (message.authorEmail !== "") {
      writer.uint32(58).string(message.authorEmail);
    }
    if (message.added !== "") {
      writer.uint32(66).string(message.added);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): FileCommit {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFileCommit();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.id = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.title = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.message = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.createdTs = longToNumber(reader.int64() as Long);
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.url = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.authorName = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.authorEmail = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.added = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): FileCommit {
    return {
      id: isSet(object.id) ? String(object.id) : "",
      title: isSet(object.title) ? String(object.title) : "",
      message: isSet(object.message) ? String(object.message) : "",
      createdTs: isSet(object.createdTs) ? Number(object.createdTs) : 0,
      url: isSet(object.url) ? String(object.url) : "",
      authorName: isSet(object.authorName) ? String(object.authorName) : "",
      authorEmail: isSet(object.authorEmail) ? String(object.authorEmail) : "",
      added: isSet(object.added) ? String(object.added) : "",
    };
  },

  toJSON(message: FileCommit): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.title !== undefined && (obj.title = message.title);
    message.message !== undefined && (obj.message = message.message);
    message.createdTs !== undefined && (obj.createdTs = Math.round(message.createdTs));
    message.url !== undefined && (obj.url = message.url);
    message.authorName !== undefined && (obj.authorName = message.authorName);
    message.authorEmail !== undefined && (obj.authorEmail = message.authorEmail);
    message.added !== undefined && (obj.added = message.added);
    return obj;
  },

  create(base?: DeepPartial<FileCommit>): FileCommit {
    return FileCommit.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<FileCommit>): FileCommit {
    const message = createBaseFileCommit();
    message.id = object.id ?? "";
    message.title = object.title ?? "";
    message.message = object.message ?? "";
    message.createdTs = object.createdTs ?? 0;
    message.url = object.url ?? "";
    message.authorName = object.authorName ?? "";
    message.authorEmail = object.authorEmail ?? "";
    message.added = object.added ?? "";
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
