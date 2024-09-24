/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { VCSType, vCSTypeFromJSON, vCSTypeToJSON, vCSTypeToNumber } from "./common";

export const protobufPackage = "bytebase.v1";

export interface GetReleaseRequest {
  /** Format: projects/{project}/releases/{release} */
  name: string;
}

export interface ListReleasesRequest {
  /** Format: projects/{project} */
  parent: string;
  /**
   * The maximum number of change histories to return. The service may return fewer than this value.
   * If unspecified, at most 10 change histories will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * Not used. A page token, received from a previous `ListReleasesRequest` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListReleasesRequest` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface ListReleasesResponse {
  releases: Release[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

export interface CreateReleaseRequest {
  /** Format: projects/{project} */
  parent: string;
  /** The release to create. */
  release: Release | undefined;
}

export interface Release {
  /** Format: projects/{project}/releases/{release} */
  name: string;
  files: Release_File[];
  vcsSource: Release_VCSSource | undefined;
}

export interface Release_File {
  filename: string;
  /**
   * The sheet that holds the statement.
   * Format: projects/{project}/sheets/{sheet}
   */
  sheet: string;
  /** The SHA1 hash value of the sheet. */
  sheetSha1: string;
  type: Release_File_Type;
  version: string;
}

export enum Release_File_Type {
  TYPE_UNSPECIFIED = "TYPE_UNSPECIFIED",
  VERSIONED = "VERSIONED",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function release_File_TypeFromJSON(object: any): Release_File_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return Release_File_Type.TYPE_UNSPECIFIED;
    case 1:
    case "VERSIONED":
      return Release_File_Type.VERSIONED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Release_File_Type.UNRECOGNIZED;
  }
}

export function release_File_TypeToJSON(object: Release_File_Type): string {
  switch (object) {
    case Release_File_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case Release_File_Type.VERSIONED:
      return "VERSIONED";
    case Release_File_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function release_File_TypeToNumber(object: Release_File_Type): number {
  switch (object) {
    case Release_File_Type.TYPE_UNSPECIFIED:
      return 0;
    case Release_File_Type.VERSIONED:
      return 1;
    case Release_File_Type.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface Release_VCSSource {
  vcsType: VCSType;
  pullRequestUrl: string;
}

function createBaseGetReleaseRequest(): GetReleaseRequest {
  return { name: "" };
}

export const GetReleaseRequest = {
  encode(message: GetReleaseRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetReleaseRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetReleaseRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GetReleaseRequest {
    return { name: isSet(object.name) ? globalThis.String(object.name) : "" };
  },

  toJSON(message: GetReleaseRequest): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    return obj;
  },

  create(base?: DeepPartial<GetReleaseRequest>): GetReleaseRequest {
    return GetReleaseRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<GetReleaseRequest>): GetReleaseRequest {
    const message = createBaseGetReleaseRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseListReleasesRequest(): ListReleasesRequest {
  return { parent: "", pageSize: 0, pageToken: "" };
}

export const ListReleasesRequest = {
  encode(message: ListReleasesRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.pageSize !== 0) {
      writer.uint32(16).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(26).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListReleasesRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListReleasesRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.parent = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.pageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListReleasesRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      pageSize: isSet(object.pageSize) ? globalThis.Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? globalThis.String(object.pageToken) : "",
    };
  },

  toJSON(message: ListReleasesRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.pageSize !== 0) {
      obj.pageSize = Math.round(message.pageSize);
    }
    if (message.pageToken !== "") {
      obj.pageToken = message.pageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListReleasesRequest>): ListReleasesRequest {
    return ListReleasesRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListReleasesRequest>): ListReleasesRequest {
    const message = createBaseListReleasesRequest();
    message.parent = object.parent ?? "";
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListReleasesResponse(): ListReleasesResponse {
  return { releases: [], nextPageToken: "" };
}

export const ListReleasesResponse = {
  encode(message: ListReleasesResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.releases) {
      Release.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListReleasesResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListReleasesResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.releases.push(Release.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.nextPageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListReleasesResponse {
    return {
      releases: globalThis.Array.isArray(object?.releases) ? object.releases.map((e: any) => Release.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? globalThis.String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListReleasesResponse): unknown {
    const obj: any = {};
    if (message.releases?.length) {
      obj.releases = message.releases.map((e) => Release.toJSON(e));
    }
    if (message.nextPageToken !== "") {
      obj.nextPageToken = message.nextPageToken;
    }
    return obj;
  },

  create(base?: DeepPartial<ListReleasesResponse>): ListReleasesResponse {
    return ListReleasesResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ListReleasesResponse>): ListReleasesResponse {
    const message = createBaseListReleasesResponse();
    message.releases = object.releases?.map((e) => Release.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseCreateReleaseRequest(): CreateReleaseRequest {
  return { parent: "", release: undefined };
}

export const CreateReleaseRequest = {
  encode(message: CreateReleaseRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.release !== undefined) {
      Release.encode(message.release, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateReleaseRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateReleaseRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.parent = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.release = Release.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateReleaseRequest {
    return {
      parent: isSet(object.parent) ? globalThis.String(object.parent) : "",
      release: isSet(object.release) ? Release.fromJSON(object.release) : undefined,
    };
  },

  toJSON(message: CreateReleaseRequest): unknown {
    const obj: any = {};
    if (message.parent !== "") {
      obj.parent = message.parent;
    }
    if (message.release !== undefined) {
      obj.release = Release.toJSON(message.release);
    }
    return obj;
  },

  create(base?: DeepPartial<CreateReleaseRequest>): CreateReleaseRequest {
    return CreateReleaseRequest.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<CreateReleaseRequest>): CreateReleaseRequest {
    const message = createBaseCreateReleaseRequest();
    message.parent = object.parent ?? "";
    message.release = (object.release !== undefined && object.release !== null)
      ? Release.fromPartial(object.release)
      : undefined;
    return message;
  },
};

function createBaseRelease(): Release {
  return { name: "", files: [], vcsSource: undefined };
}

export const Release = {
  encode(message: Release, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.files) {
      Release_File.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.vcsSource !== undefined) {
      Release_VCSSource.encode(message.vcsSource, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Release {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRelease();
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

          message.files.push(Release_File.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.vcsSource = Release_VCSSource.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Release {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      files: globalThis.Array.isArray(object?.files) ? object.files.map((e: any) => Release_File.fromJSON(e)) : [],
      vcsSource: isSet(object.vcsSource) ? Release_VCSSource.fromJSON(object.vcsSource) : undefined,
    };
  },

  toJSON(message: Release): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.files?.length) {
      obj.files = message.files.map((e) => Release_File.toJSON(e));
    }
    if (message.vcsSource !== undefined) {
      obj.vcsSource = Release_VCSSource.toJSON(message.vcsSource);
    }
    return obj;
  },

  create(base?: DeepPartial<Release>): Release {
    return Release.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Release>): Release {
    const message = createBaseRelease();
    message.name = object.name ?? "";
    message.files = object.files?.map((e) => Release_File.fromPartial(e)) || [];
    message.vcsSource = (object.vcsSource !== undefined && object.vcsSource !== null)
      ? Release_VCSSource.fromPartial(object.vcsSource)
      : undefined;
    return message;
  },
};

function createBaseRelease_File(): Release_File {
  return { filename: "", sheet: "", sheetSha1: "", type: Release_File_Type.TYPE_UNSPECIFIED, version: "" };
}

export const Release_File = {
  encode(message: Release_File, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.filename !== "") {
      writer.uint32(10).string(message.filename);
    }
    if (message.sheet !== "") {
      writer.uint32(18).string(message.sheet);
    }
    if (message.sheetSha1 !== "") {
      writer.uint32(26).string(message.sheetSha1);
    }
    if (message.type !== Release_File_Type.TYPE_UNSPECIFIED) {
      writer.uint32(32).int32(release_File_TypeToNumber(message.type));
    }
    if (message.version !== "") {
      writer.uint32(42).string(message.version);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Release_File {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRelease_File();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.filename = reader.string();
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

          message.type = release_File_TypeFromJSON(reader.int32());
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

  fromJSON(object: any): Release_File {
    return {
      filename: isSet(object.filename) ? globalThis.String(object.filename) : "",
      sheet: isSet(object.sheet) ? globalThis.String(object.sheet) : "",
      sheetSha1: isSet(object.sheetSha1) ? globalThis.String(object.sheetSha1) : "",
      type: isSet(object.type) ? release_File_TypeFromJSON(object.type) : Release_File_Type.TYPE_UNSPECIFIED,
      version: isSet(object.version) ? globalThis.String(object.version) : "",
    };
  },

  toJSON(message: Release_File): unknown {
    const obj: any = {};
    if (message.filename !== "") {
      obj.filename = message.filename;
    }
    if (message.sheet !== "") {
      obj.sheet = message.sheet;
    }
    if (message.sheetSha1 !== "") {
      obj.sheetSha1 = message.sheetSha1;
    }
    if (message.type !== Release_File_Type.TYPE_UNSPECIFIED) {
      obj.type = release_File_TypeToJSON(message.type);
    }
    if (message.version !== "") {
      obj.version = message.version;
    }
    return obj;
  },

  create(base?: DeepPartial<Release_File>): Release_File {
    return Release_File.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Release_File>): Release_File {
    const message = createBaseRelease_File();
    message.filename = object.filename ?? "";
    message.sheet = object.sheet ?? "";
    message.sheetSha1 = object.sheetSha1 ?? "";
    message.type = object.type ?? Release_File_Type.TYPE_UNSPECIFIED;
    message.version = object.version ?? "";
    return message;
  },
};

function createBaseRelease_VCSSource(): Release_VCSSource {
  return { vcsType: VCSType.VCS_TYPE_UNSPECIFIED, pullRequestUrl: "" };
}

export const Release_VCSSource = {
  encode(message: Release_VCSSource, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.vcsType !== VCSType.VCS_TYPE_UNSPECIFIED) {
      writer.uint32(8).int32(vCSTypeToNumber(message.vcsType));
    }
    if (message.pullRequestUrl !== "") {
      writer.uint32(18).string(message.pullRequestUrl);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Release_VCSSource {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRelease_VCSSource();
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

  fromJSON(object: any): Release_VCSSource {
    return {
      vcsType: isSet(object.vcsType) ? vCSTypeFromJSON(object.vcsType) : VCSType.VCS_TYPE_UNSPECIFIED,
      pullRequestUrl: isSet(object.pullRequestUrl) ? globalThis.String(object.pullRequestUrl) : "",
    };
  },

  toJSON(message: Release_VCSSource): unknown {
    const obj: any = {};
    if (message.vcsType !== VCSType.VCS_TYPE_UNSPECIFIED) {
      obj.vcsType = vCSTypeToJSON(message.vcsType);
    }
    if (message.pullRequestUrl !== "") {
      obj.pullRequestUrl = message.pullRequestUrl;
    }
    return obj;
  },

  create(base?: DeepPartial<Release_VCSSource>): Release_VCSSource {
    return Release_VCSSource.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Release_VCSSource>): Release_VCSSource {
    const message = createBaseRelease_VCSSource();
    message.vcsType = object.vcsType ?? VCSType.VCS_TYPE_UNSPECIFIED;
    message.pullRequestUrl = object.pullRequestUrl ?? "";
    return message;
  },
};

export type ReleaseServiceDefinition = typeof ReleaseServiceDefinition;
export const ReleaseServiceDefinition = {
  name: "ReleaseService",
  fullName: "bytebase.v1.ReleaseService",
  methods: {
    getRelease: {
      name: "GetRelease",
      requestType: GetReleaseRequest,
      requestStream: false,
      responseType: Release,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              34,
              18,
              32,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              47,
              114,
              101,
              108,
              101,
              97,
              115,
              101,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    listReleases: {
      name: "ListReleases",
      requestType: ListReleasesRequest,
      requestStream: false,
      responseType: ListReleasesResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([6, 112, 97, 114, 101, 110, 116])],
          578365826: [
            new Uint8Array([
              34,
              18,
              32,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              125,
              47,
              114,
              101,
              108,
              101,
              97,
              115,
              101,
              115,
            ]),
          ],
        },
      },
    },
    createRelease: {
      name: "CreateRelease",
      requestType: CreateReleaseRequest,
      requestStream: false,
      responseType: Release,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([14, 112, 97, 114, 101, 110, 116, 44, 114, 101, 108, 101, 97, 115, 101])],
          578365826: [
            new Uint8Array([
              43,
              58,
              7,
              114,
              101,
              108,
              101,
              97,
              115,
              101,
              34,
              32,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              125,
              47,
              114,
              101,
              108,
              101,
              97,
              115,
              101,
              115,
            ]),
          ],
        },
      },
    },
  },
} as const;

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
