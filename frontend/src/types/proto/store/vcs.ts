/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { VcsType, vcsTypeFromJSON, vcsTypeToJSON } from "./common";

export const protobufPackage = "bytebase.store";

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
  fileCommit: FileCommit | undefined;
}

export interface Commit {
  id: string;
  title: string;
  message: string;
  createdTs: Long;
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
  createdTs: Long;
  url: string;
  authorName: string;
  authorEmail: string;
  added: string;
}

export interface VCSConnector {
  /** The title or display name of the VCS connector. */
  title: string;
  /**
   * Full path from the corresponding VCS provider.
   * For GitLab, this is the project full path. e.g. group1/project-1
   */
  fullPath: string;
  /**
   * Web url from the corresponding VCS provider.
   * For GitLab, this is the project web url. e.g. https://gitlab.example.com/group1/project-1
   */
  webUrl: string;
  /** Branch to listen to. */
  branch: string;
  /** Base working directory we are interested. */
  baseDirectory: string;
  /**
   * Repository id from the corresponding VCS provider.
   * For GitLab, this is the project id. e.g. 123
   */
  externalId: string;
  /**
   * Push webhook id from the corresponding VCS provider.
   * For GitLab, this is the project webhook id. e.g. 123
   */
  externalWebhookId: string;
  /** For GitLab, webhook request contains this in the 'X-Gitlab-Token" header and we compare it with the one stored in db to validate it sends to the expected endpoint. */
  webhookSecretToken: string;
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
      baseDir: isSet(object.baseDir) ? globalThis.String(object.baseDir) : "",
      ref: isSet(object.ref) ? globalThis.String(object.ref) : "",
      before: isSet(object.before) ? globalThis.String(object.before) : "",
      after: isSet(object.after) ? globalThis.String(object.after) : "",
      repositoryId: isSet(object.repositoryId) ? globalThis.String(object.repositoryId) : "",
      repositoryUrl: isSet(object.repositoryUrl) ? globalThis.String(object.repositoryUrl) : "",
      repositoryFullPath: isSet(object.repositoryFullPath) ? globalThis.String(object.repositoryFullPath) : "",
      authorName: isSet(object.authorName) ? globalThis.String(object.authorName) : "",
      commits: globalThis.Array.isArray(object?.commits) ? object.commits.map((e: any) => Commit.fromJSON(e)) : [],
      fileCommit: isSet(object.fileCommit) ? FileCommit.fromJSON(object.fileCommit) : undefined,
    };
  },

  toJSON(message: PushEvent): unknown {
    const obj: any = {};
    if (message.vcsType !== 0) {
      obj.vcsType = vcsTypeToJSON(message.vcsType);
    }
    if (message.baseDir !== "") {
      obj.baseDir = message.baseDir;
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
    if (message.fileCommit !== undefined) {
      obj.fileCommit = FileCommit.toJSON(message.fileCommit);
    }
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
    createdTs: Long.ZERO,
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
    if (!message.createdTs.isZero()) {
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

          message.createdTs = reader.int64() as Long;
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
      createdTs: isSet(object.createdTs) ? Long.fromValue(object.createdTs) : Long.ZERO,
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
    if (!message.createdTs.isZero()) {
      obj.createdTs = (message.createdTs || Long.ZERO).toString();
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
    message.createdTs = (object.createdTs !== undefined && object.createdTs !== null)
      ? Long.fromValue(object.createdTs)
      : Long.ZERO;
    message.url = object.url ?? "";
    message.authorName = object.authorName ?? "";
    message.authorEmail = object.authorEmail ?? "";
    message.addedList = object.addedList?.map((e) => e) || [];
    message.modifiedList = object.modifiedList?.map((e) => e) || [];
    return message;
  },
};

function createBaseFileCommit(): FileCommit {
  return { id: "", title: "", message: "", createdTs: Long.ZERO, url: "", authorName: "", authorEmail: "", added: "" };
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
    if (!message.createdTs.isZero()) {
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

          message.createdTs = reader.int64() as Long;
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
      id: isSet(object.id) ? globalThis.String(object.id) : "",
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      message: isSet(object.message) ? globalThis.String(object.message) : "",
      createdTs: isSet(object.createdTs) ? Long.fromValue(object.createdTs) : Long.ZERO,
      url: isSet(object.url) ? globalThis.String(object.url) : "",
      authorName: isSet(object.authorName) ? globalThis.String(object.authorName) : "",
      authorEmail: isSet(object.authorEmail) ? globalThis.String(object.authorEmail) : "",
      added: isSet(object.added) ? globalThis.String(object.added) : "",
    };
  },

  toJSON(message: FileCommit): unknown {
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
    if (!message.createdTs.isZero()) {
      obj.createdTs = (message.createdTs || Long.ZERO).toString();
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
    if (message.added !== "") {
      obj.added = message.added;
    }
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
    message.createdTs = (object.createdTs !== undefined && object.createdTs !== null)
      ? Long.fromValue(object.createdTs)
      : Long.ZERO;
    message.url = object.url ?? "";
    message.authorName = object.authorName ?? "";
    message.authorEmail = object.authorEmail ?? "";
    message.added = object.added ?? "";
    return message;
  },
};

function createBaseVCSConnector(): VCSConnector {
  return {
    title: "",
    fullPath: "",
    webUrl: "",
    branch: "",
    baseDirectory: "",
    externalId: "",
    externalWebhookId: "",
    webhookSecretToken: "",
  };
}

export const VCSConnector = {
  encode(message: VCSConnector, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.title !== "") {
      writer.uint32(10).string(message.title);
    }
    if (message.fullPath !== "") {
      writer.uint32(18).string(message.fullPath);
    }
    if (message.webUrl !== "") {
      writer.uint32(26).string(message.webUrl);
    }
    if (message.branch !== "") {
      writer.uint32(34).string(message.branch);
    }
    if (message.baseDirectory !== "") {
      writer.uint32(42).string(message.baseDirectory);
    }
    if (message.externalId !== "") {
      writer.uint32(50).string(message.externalId);
    }
    if (message.externalWebhookId !== "") {
      writer.uint32(58).string(message.externalWebhookId);
    }
    if (message.webhookSecretToken !== "") {
      writer.uint32(66).string(message.webhookSecretToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): VCSConnector {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseVCSConnector();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.title = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.fullPath = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.webUrl = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.branch = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.baseDirectory = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.externalId = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.externalWebhookId = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.webhookSecretToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): VCSConnector {
    return {
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      fullPath: isSet(object.fullPath) ? globalThis.String(object.fullPath) : "",
      webUrl: isSet(object.webUrl) ? globalThis.String(object.webUrl) : "",
      branch: isSet(object.branch) ? globalThis.String(object.branch) : "",
      baseDirectory: isSet(object.baseDirectory) ? globalThis.String(object.baseDirectory) : "",
      externalId: isSet(object.externalId) ? globalThis.String(object.externalId) : "",
      externalWebhookId: isSet(object.externalWebhookId) ? globalThis.String(object.externalWebhookId) : "",
      webhookSecretToken: isSet(object.webhookSecretToken) ? globalThis.String(object.webhookSecretToken) : "",
    };
  },

  toJSON(message: VCSConnector): unknown {
    const obj: any = {};
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.fullPath !== "") {
      obj.fullPath = message.fullPath;
    }
    if (message.webUrl !== "") {
      obj.webUrl = message.webUrl;
    }
    if (message.branch !== "") {
      obj.branch = message.branch;
    }
    if (message.baseDirectory !== "") {
      obj.baseDirectory = message.baseDirectory;
    }
    if (message.externalId !== "") {
      obj.externalId = message.externalId;
    }
    if (message.externalWebhookId !== "") {
      obj.externalWebhookId = message.externalWebhookId;
    }
    if (message.webhookSecretToken !== "") {
      obj.webhookSecretToken = message.webhookSecretToken;
    }
    return obj;
  },

  create(base?: DeepPartial<VCSConnector>): VCSConnector {
    return VCSConnector.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<VCSConnector>): VCSConnector {
    const message = createBaseVCSConnector();
    message.title = object.title ?? "";
    message.fullPath = object.fullPath ?? "";
    message.webUrl = object.webUrl ?? "";
    message.branch = object.branch ?? "";
    message.baseDirectory = object.baseDirectory ?? "";
    message.externalId = object.externalId ?? "";
    message.externalWebhookId = object.externalWebhookId ?? "";
    message.webhookSecretToken = object.webhookSecretToken ?? "";
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
