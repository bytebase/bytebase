// Code generated by protoc-gen-ts_proto. DO NOT EDIT.
// versions:
//   protoc-gen-ts_proto  v2.3.0
//   protoc               unknown
// source: store/revision.proto

/* eslint-disable */
import { BinaryReader, BinaryWriter } from "@bufbuild/protobuf/wire";
import Long from "long";

export const protobufPackage = "bytebase.store";

export interface RevisionPayload {
  /**
   * Format: projects/{project}/releases/{release}
   * Can be empty.
   */
  release: string;
  /**
   * Format: projects/{project}/releases/{release}/files/{id}
   * Can be empty.
   */
  file: string;
  /**
   * The sheet that holds the content.
   * Format: projects/{project}/sheets/{sheet}
   */
  sheet: string;
  /** The SHA256 hash value of the sheet. */
  sheetSha256: string;
  /**
   * The task run associated with the revision.
   * Can be empty.
   * Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun}
   */
  taskRun: string;
}

function createBaseRevisionPayload(): RevisionPayload {
  return { release: "", file: "", sheet: "", sheetSha256: "", taskRun: "" };
}

export const RevisionPayload: MessageFns<RevisionPayload> = {
  encode(message: RevisionPayload, writer: BinaryWriter = new BinaryWriter()): BinaryWriter {
    if (message.release !== "") {
      writer.uint32(10).string(message.release);
    }
    if (message.file !== "") {
      writer.uint32(18).string(message.file);
    }
    if (message.sheet !== "") {
      writer.uint32(26).string(message.sheet);
    }
    if (message.sheetSha256 !== "") {
      writer.uint32(34).string(message.sheetSha256);
    }
    if (message.taskRun !== "") {
      writer.uint32(42).string(message.taskRun);
    }
    return writer;
  },

  decode(input: BinaryReader | Uint8Array, length?: number): RevisionPayload {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRevisionPayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1: {
          if (tag !== 10) {
            break;
          }

          message.release = reader.string();
          continue;
        }
        case 2: {
          if (tag !== 18) {
            break;
          }

          message.file = reader.string();
          continue;
        }
        case 3: {
          if (tag !== 26) {
            break;
          }

          message.sheet = reader.string();
          continue;
        }
        case 4: {
          if (tag !== 34) {
            break;
          }

          message.sheetSha256 = reader.string();
          continue;
        }
        case 5: {
          if (tag !== 42) {
            break;
          }

          message.taskRun = reader.string();
          continue;
        }
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skip(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): RevisionPayload {
    return {
      release: isSet(object.release) ? globalThis.String(object.release) : "",
      file: isSet(object.file) ? globalThis.String(object.file) : "",
      sheet: isSet(object.sheet) ? globalThis.String(object.sheet) : "",
      sheetSha256: isSet(object.sheetSha256) ? globalThis.String(object.sheetSha256) : "",
      taskRun: isSet(object.taskRun) ? globalThis.String(object.taskRun) : "",
    };
  },

  toJSON(message: RevisionPayload): unknown {
    const obj: any = {};
    if (message.release !== "") {
      obj.release = message.release;
    }
    if (message.file !== "") {
      obj.file = message.file;
    }
    if (message.sheet !== "") {
      obj.sheet = message.sheet;
    }
    if (message.sheetSha256 !== "") {
      obj.sheetSha256 = message.sheetSha256;
    }
    if (message.taskRun !== "") {
      obj.taskRun = message.taskRun;
    }
    return obj;
  },

  create(base?: DeepPartial<RevisionPayload>): RevisionPayload {
    return RevisionPayload.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<RevisionPayload>): RevisionPayload {
    const message = createBaseRevisionPayload();
    message.release = object.release ?? "";
    message.file = object.file ?? "";
    message.sheet = object.sheet ?? "";
    message.sheetSha256 = object.sheetSha256 ?? "";
    message.taskRun = object.taskRun ?? "";
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Long ? string | number | Long : T extends globalThis.Array<infer U> ? globalThis.Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}

export interface MessageFns<T> {
  encode(message: T, writer?: BinaryWriter): BinaryWriter;
  decode(input: BinaryReader | Uint8Array, length?: number): T;
  fromJSON(object: any): T;
  toJSON(message: T): unknown;
  create(base?: DeepPartial<T>): T;
  fromPartial(object: DeepPartial<T>): T;
}
