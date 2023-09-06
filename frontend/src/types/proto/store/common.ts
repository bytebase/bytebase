/* eslint-disable */
import * as _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export enum Engine {
  ENGINE_UNSPECIFIED = 0,
  CLICKHOUSE = 1,
  MYSQL = 2,
  POSTGRES = 3,
  SNOWFLAKE = 4,
  SQLITE = 5,
  TIDB = 6,
  MONGODB = 7,
  REDIS = 8,
  ORACLE = 9,
  SPANNER = 10,
  MSSQL = 11,
  REDSHIFT = 12,
  MARIADB = 13,
  OCEANBASE = 14,
  DM = 15,
  UNRECOGNIZED = -1,
}

export function engineFromJSON(object: any): Engine {
  switch (object) {
    case 0:
    case "ENGINE_UNSPECIFIED":
      return Engine.ENGINE_UNSPECIFIED;
    case 1:
    case "CLICKHOUSE":
      return Engine.CLICKHOUSE;
    case 2:
    case "MYSQL":
      return Engine.MYSQL;
    case 3:
    case "POSTGRES":
      return Engine.POSTGRES;
    case 4:
    case "SNOWFLAKE":
      return Engine.SNOWFLAKE;
    case 5:
    case "SQLITE":
      return Engine.SQLITE;
    case 6:
    case "TIDB":
      return Engine.TIDB;
    case 7:
    case "MONGODB":
      return Engine.MONGODB;
    case 8:
    case "REDIS":
      return Engine.REDIS;
    case 9:
    case "ORACLE":
      return Engine.ORACLE;
    case 10:
    case "SPANNER":
      return Engine.SPANNER;
    case 11:
    case "MSSQL":
      return Engine.MSSQL;
    case 12:
    case "REDSHIFT":
      return Engine.REDSHIFT;
    case 13:
    case "MARIADB":
      return Engine.MARIADB;
    case 14:
    case "OCEANBASE":
      return Engine.OCEANBASE;
    case 15:
    case "DM":
      return Engine.DM;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Engine.UNRECOGNIZED;
  }
}

export function engineToJSON(object: Engine): string {
  switch (object) {
    case Engine.ENGINE_UNSPECIFIED:
      return "ENGINE_UNSPECIFIED";
    case Engine.CLICKHOUSE:
      return "CLICKHOUSE";
    case Engine.MYSQL:
      return "MYSQL";
    case Engine.POSTGRES:
      return "POSTGRES";
    case Engine.SNOWFLAKE:
      return "SNOWFLAKE";
    case Engine.SQLITE:
      return "SQLITE";
    case Engine.TIDB:
      return "TIDB";
    case Engine.MONGODB:
      return "MONGODB";
    case Engine.REDIS:
      return "REDIS";
    case Engine.ORACLE:
      return "ORACLE";
    case Engine.SPANNER:
      return "SPANNER";
    case Engine.MSSQL:
      return "MSSQL";
    case Engine.REDSHIFT:
      return "REDSHIFT";
    case Engine.MARIADB:
      return "MARIADB";
    case Engine.OCEANBASE:
      return "OCEANBASE";
    case Engine.DM:
      return "DM";
    case Engine.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

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

export enum MaskingLevel {
  MASKING_LEVEL_UNSPECIFIED = 0,
  NONE = 1,
  PARTIAL = 2,
  FULL = 3,
  UNRECOGNIZED = -1,
}

export function maskingLevelFromJSON(object: any): MaskingLevel {
  switch (object) {
    case 0:
    case "MASKING_LEVEL_UNSPECIFIED":
      return MaskingLevel.MASKING_LEVEL_UNSPECIFIED;
    case 1:
    case "NONE":
      return MaskingLevel.NONE;
    case 2:
    case "PARTIAL":
      return MaskingLevel.PARTIAL;
    case 3:
    case "FULL":
      return MaskingLevel.FULL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return MaskingLevel.UNRECOGNIZED;
  }
}

export function maskingLevelToJSON(object: MaskingLevel): string {
  switch (object) {
    case MaskingLevel.MASKING_LEVEL_UNSPECIFIED:
      return "MASKING_LEVEL_UNSPECIFIED";
    case MaskingLevel.NONE:
      return "NONE";
    case MaskingLevel.PARTIAL:
      return "PARTIAL";
    case MaskingLevel.FULL:
      return "FULL";
    case MaskingLevel.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

/** Used internally for obfuscating the page token. */
export interface PageToken {
  limit: number;
  offset: number;
}

function createBasePageToken(): PageToken {
  return { limit: 0, offset: 0 };
}

export const PageToken = {
  encode(message: PageToken, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.limit !== 0) {
      writer.uint32(8).int32(message.limit);
    }
    if (message.offset !== 0) {
      writer.uint32(16).int32(message.offset);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PageToken {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePageToken();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.limit = reader.int32();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.offset = reader.int32();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PageToken {
    return {
      limit: isSet(object.limit) ? Number(object.limit) : 0,
      offset: isSet(object.offset) ? Number(object.offset) : 0,
    };
  },

  toJSON(message: PageToken): unknown {
    const obj: any = {};
    message.limit !== undefined && (obj.limit = Math.round(message.limit));
    message.offset !== undefined && (obj.offset = Math.round(message.offset));
    return obj;
  },

  create(base?: DeepPartial<PageToken>): PageToken {
    return PageToken.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<PageToken>): PageToken {
    const message = createBasePageToken();
    message.limit = object.limit ?? 0;
    message.offset = object.offset ?? 0;
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
