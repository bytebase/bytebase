/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Timestamp } from "../google/protobuf/timestamp";

export const protobufPackage = "bytebase.v1";

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
  ref: string;
  before: string;
  after: string;
  repositoryId: string;
  repositoryUrl: string;
  repositoryFullPath: string;
  authorName: string;
  commits: Commit[];
}

export interface Commit {
  id: string;
  title: string;
  message: string;
  createdTime: Date | undefined;
  url: string;
  authorName: string;
  authorEmail: string;
  addedList: string[];
  modifiedList: string[];
}

function createBasePushEvent(): PushEvent {
  return {
    vcsType: 0,
    ref: "",
    before: "",
    after: "",
    repositoryId: "",
    repositoryUrl: "",
    repositoryFullPath: "",
    authorName: "",
    commits: [],
  };
}

export const PushEvent = {
  encode(message: PushEvent, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.vcsType !== 0) {
      writer.uint32(8).int32(message.vcsType);
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
      ref: isSet(object.ref) ? globalThis.String(object.ref) : "",
      before: isSet(object.before) ? globalThis.String(object.before) : "",
      after: isSet(object.after) ? globalThis.String(object.after) : "",
      repositoryId: isSet(object.repositoryId) ? globalThis.String(object.repositoryId) : "",
      repositoryUrl: isSet(object.repositoryUrl) ? globalThis.String(object.repositoryUrl) : "",
      repositoryFullPath: isSet(object.repositoryFullPath) ? globalThis.String(object.repositoryFullPath) : "",
      authorName: isSet(object.authorName) ? globalThis.String(object.authorName) : "",
      commits: globalThis.Array.isArray(object?.commits) ? object.commits.map((e: any) => Commit.fromJSON(e)) : [],
    };
  },

  toJSON(message: PushEvent): unknown {
    const obj: any = {};
    if (message.vcsType !== 0) {
      obj.vcsType = vcsTypeToJSON(message.vcsType);
    }
    if (message.ref !== "") {
      obj.ref = message.ref;
    }
    if (message.before !== "") {
      obj.before = message.before;
    }
    if (message.after !== "") {
      obj.after = message.after;
    }
    if (message.repositoryId !== "") {
      obj.repositoryId = message.repositoryId;
    }
    if (message.repositoryUrl !== "") {
      obj.repositoryUrl = message.repositoryUrl;
    }
    if (message.repositoryFullPath !== "") {
      obj.repositoryFullPath = message.repositoryFullPath;
    }
    if (message.authorName !== "") {
      obj.authorName = message.authorName;
    }
    if (message.commits?.length) {
      obj.commits = message.commits.map((e) => Commit.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<PushEvent>): PushEvent {
    return PushEvent.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<PushEvent>): PushEvent {
    const message = createBasePushEvent();
    message.vcsType = object.vcsType ?? 0;
    message.ref = object.ref ?? "";
    message.before = object.before ?? "";
    message.after = object.after ?? "";
    message.repositoryId = object.repositoryId ?? "";
    message.repositoryUrl = object.repositoryUrl ?? "";
    message.repositoryFullPath = object.repositoryFullPath ?? "";
    message.authorName = object.authorName ?? "";
    message.commits = object.commits?.map((e) => Commit.fromPartial(e)) || [];
    return message;
  },
};

function createBaseCommit(): Commit {
  return {
    id: "",
    title: "",
    message: "",
    createdTime: undefined,
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
    if (message.createdTime !== undefined) {
      Timestamp.encode(toTimestamp(message.createdTime), writer.uint32(34).fork()).ldelim();
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
          if (tag !== 34) {
            break;
          }

          message.createdTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
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
      id: isSet(object.id) ? globalThis.String(object.id) : "",
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      message: isSet(object.message) ? globalThis.String(object.message) : "",
      createdTime: isSet(object.createdTime) ? fromJsonTimestamp(object.createdTime) : undefined,
      url: isSet(object.url) ? globalThis.String(object.url) : "",
      authorName: isSet(object.authorName) ? globalThis.String(object.authorName) : "",
      authorEmail: isSet(object.authorEmail) ? globalThis.String(object.authorEmail) : "",
      addedList: globalThis.Array.isArray(object?.addedList)
        ? object.addedList.map((e: any) => globalThis.String(e))
        : [],
      modifiedList: globalThis.Array.isArray(object?.modifiedList)
        ? object.modifiedList.map((e: any) => globalThis.String(e))
        : [],
    };
  },

  toJSON(message: Commit): unknown {
    const obj: any = {};
    if (message.id !== "") {
      obj.id = message.id;
    }
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.message !== "") {
      obj.message = message.message;
    }
    if (message.createdTime !== undefined) {
      obj.createdTime = message.createdTime.toISOString();
    }
    if (message.url !== "") {
      obj.url = message.url;
    }
    if (message.authorName !== "") {
      obj.authorName = message.authorName;
    }
    if (message.authorEmail !== "") {
      obj.authorEmail = message.authorEmail;
    }
    if (message.addedList?.length) {
      obj.addedList = message.addedList;
    }
    if (message.modifiedList?.length) {
      obj.modifiedList = message.modifiedList;
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
    message.createdTime = object.createdTime ?? undefined;
    message.url = object.url ?? "";
    message.authorName = object.authorName ?? "";
    message.authorEmail = object.authorEmail ?? "";
    message.addedList = object.addedList?.map((e) => e) || [];
    message.modifiedList = object.modifiedList?.map((e) => e) || [];
    return message;
  },
};

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

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
