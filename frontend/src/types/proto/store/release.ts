/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { VCSType, vCSTypeFromJSON, vCSTypeToJSON, vCSTypeToNumber } from "./common";

export const protobufPackage = "bytebase.store";

export enum ReleaseFileType {
  TYPE_UNSPECIFIED = "TYPE_UNSPECIFIED",
  VERSIONED = "VERSIONED",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function releaseFileTypeFromJSON(object: any): ReleaseFileType {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return ReleaseFileType.TYPE_UNSPECIFIED;
    case 1:
    case "VERSIONED":
      return ReleaseFileType.VERSIONED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ReleaseFileType.UNRECOGNIZED;
  }
}

export function releaseFileTypeToJSON(object: ReleaseFileType): string {
  switch (object) {
    case ReleaseFileType.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case ReleaseFileType.VERSIONED:
      return "VERSIONED";
    case ReleaseFileType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function releaseFileTypeToNumber(object: ReleaseFileType): number {
  switch (object) {
    case ReleaseFileType.TYPE_UNSPECIFIED:
      return 0;
    case ReleaseFileType.VERSIONED:
      return 1;
    case ReleaseFileType.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface ReleasePayload {
  title: string;
  files: ReleasePayload_File[];
  vcsSource: ReleasePayload_VCSSource | undefined;
}

export interface ReleasePayload_File {
  /**
   * The name of the file.
   * Expressed as a path, e.g. `2.2/V0001_create_table.sql`
   */
  name: string;
  /**
   * The sheet that holds the content.
   * Format: projects/{project}/sheets/{sheet}
   */
  sheet: string;
  /** The SHA1 hash value of the sheet. */
  sheetSha1: string;
  type: ReleaseFileType;
  version: string;
}

export interface ReleasePayload_VCSSource {
  vcsType: VCSType;
  pullRequestUrl: string;
}

function createBaseReleasePayload(): ReleasePayload {
  return { title: "", files: [], vcsSource: undefined };
}

export const ReleasePayload = {
  encode(message: ReleasePayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.title !== "") {
      writer.uint32(10).string(message.title);
    }
    for (const v of message.files) {
      ReleasePayload_File.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.vcsSource !== undefined) {
      ReleasePayload_VCSSource.encode(message.vcsSource, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ReleasePayload {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseReleasePayload();
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

          message.files.push(ReleasePayload_File.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.vcsSource = ReleasePayload_VCSSource.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ReleasePayload {
    return {
      title: isSet(object.title) ? globalThis.String(object.title) : "",
      files: globalThis.Array.isArray(object?.files)
        ? object.files.map((e: any) => ReleasePayload_File.fromJSON(e))
        : [],
      vcsSource: isSet(object.vcsSource) ? ReleasePayload_VCSSource.fromJSON(object.vcsSource) : undefined,
    };
  },

  toJSON(message: ReleasePayload): unknown {
    const obj: any = {};
    if (message.title !== "") {
      obj.title = message.title;
    }
    if (message.files?.length) {
      obj.files = message.files.map((e) => ReleasePayload_File.toJSON(e));
    }
    if (message.vcsSource !== undefined) {
      obj.vcsSource = ReleasePayload_VCSSource.toJSON(message.vcsSource);
    }
    return obj;
  },

  create(base?: DeepPartial<ReleasePayload>): ReleasePayload {
    return ReleasePayload.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ReleasePayload>): ReleasePayload {
    const message = createBaseReleasePayload();
    message.title = object.title ?? "";
    message.files = object.files?.map((e) => ReleasePayload_File.fromPartial(e)) || [];
    message.vcsSource = (object.vcsSource !== undefined && object.vcsSource !== null)
      ? ReleasePayload_VCSSource.fromPartial(object.vcsSource)
      : undefined;
    return message;
  },
};

function createBaseReleasePayload_File(): ReleasePayload_File {
  return { name: "", sheet: "", sheetSha1: "", type: ReleaseFileType.TYPE_UNSPECIFIED, version: "" };
}

export const ReleasePayload_File = {
  encode(message: ReleasePayload_File, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.sheet !== "") {
      writer.uint32(18).string(message.sheet);
    }
    if (message.sheetSha1 !== "") {
      writer.uint32(26).string(message.sheetSha1);
    }
    if (message.type !== ReleaseFileType.TYPE_UNSPECIFIED) {
      writer.uint32(32).int32(releaseFileTypeToNumber(message.type));
    }
    if (message.version !== "") {
      writer.uint32(42).string(message.version);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ReleasePayload_File {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseReleasePayload_File();
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

          message.sheet = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.sheetSha1 = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.type = releaseFileTypeFromJSON(reader.int32());
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.version = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ReleasePayload_File {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      sheet: isSet(object.sheet) ? globalThis.String(object.sheet) : "",
      sheetSha1: isSet(object.sheetSha1) ? globalThis.String(object.sheetSha1) : "",
      type: isSet(object.type) ? releaseFileTypeFromJSON(object.type) : ReleaseFileType.TYPE_UNSPECIFIED,
      version: isSet(object.version) ? globalThis.String(object.version) : "",
    };
  },

  toJSON(message: ReleasePayload_File): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.sheet !== "") {
      obj.sheet = message.sheet;
    }
    if (message.sheetSha1 !== "") {
      obj.sheetSha1 = message.sheetSha1;
    }
    if (message.type !== ReleaseFileType.TYPE_UNSPECIFIED) {
      obj.type = releaseFileTypeToJSON(message.type);
    }
    if (message.version !== "") {
      obj.version = message.version;
    }
    return obj;
  },

  create(base?: DeepPartial<ReleasePayload_File>): ReleasePayload_File {
    return ReleasePayload_File.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ReleasePayload_File>): ReleasePayload_File {
    const message = createBaseReleasePayload_File();
    message.name = object.name ?? "";
    message.sheet = object.sheet ?? "";
    message.sheetSha1 = object.sheetSha1 ?? "";
    message.type = object.type ?? ReleaseFileType.TYPE_UNSPECIFIED;
    message.version = object.version ?? "";
    return message;
  },
};

function createBaseReleasePayload_VCSSource(): ReleasePayload_VCSSource {
  return { vcsType: VCSType.VCS_TYPE_UNSPECIFIED, pullRequestUrl: "" };
}

export const ReleasePayload_VCSSource = {
  encode(message: ReleasePayload_VCSSource, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.vcsType !== VCSType.VCS_TYPE_UNSPECIFIED) {
      writer.uint32(8).int32(vCSTypeToNumber(message.vcsType));
    }
    if (message.pullRequestUrl !== "") {
      writer.uint32(18).string(message.pullRequestUrl);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ReleasePayload_VCSSource {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseReleasePayload_VCSSource();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.vcsType = vCSTypeFromJSON(reader.int32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.pullRequestUrl = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ReleasePayload_VCSSource {
    return {
      vcsType: isSet(object.vcsType) ? vCSTypeFromJSON(object.vcsType) : VCSType.VCS_TYPE_UNSPECIFIED,
      pullRequestUrl: isSet(object.pullRequestUrl) ? globalThis.String(object.pullRequestUrl) : "",
    };
  },

  toJSON(message: ReleasePayload_VCSSource): unknown {
    const obj: any = {};
    if (message.vcsType !== VCSType.VCS_TYPE_UNSPECIFIED) {
      obj.vcsType = vCSTypeToJSON(message.vcsType);
    }
    if (message.pullRequestUrl !== "") {
      obj.pullRequestUrl = message.pullRequestUrl;
    }
    return obj;
  },

  create(base?: DeepPartial<ReleasePayload_VCSSource>): ReleasePayload_VCSSource {
    return ReleasePayload_VCSSource.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ReleasePayload_VCSSource>): ReleasePayload_VCSSource {
    const message = createBaseReleasePayload_VCSSource();
    message.vcsType = object.vcsType ?? VCSType.VCS_TYPE_UNSPECIFIED;
    message.pullRequestUrl = object.pullRequestUrl ?? "";
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
