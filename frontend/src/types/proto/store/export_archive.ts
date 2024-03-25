/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { ExportFormat, exportFormatFromJSON, exportFormatToJSON } from "./common";

export const protobufPackage = "bytebase.store";

export interface ExportArchivePayload {
  /** The exported file format. e.g. JSON, CSV, SQL */
  fileFormat: ExportFormat;
}

function createBaseExportArchivePayload(): ExportArchivePayload {
  return { fileFormat: 0 };
}

export const ExportArchivePayload = {
  encode(message: ExportArchivePayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.fileFormat !== 0) {
      writer.uint32(8).int32(message.fileFormat);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ExportArchivePayload {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExportArchivePayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.fileFormat = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ExportArchivePayload {
    return { fileFormat: isSet(object.fileFormat) ? exportFormatFromJSON(object.fileFormat) : 0 };
  },

  toJSON(message: ExportArchivePayload): unknown {
    const obj: any = {};
    if (message.fileFormat !== 0) {
      obj.fileFormat = exportFormatToJSON(message.fileFormat);
    }
    return obj;
  },

  create(base?: DeepPartial<ExportArchivePayload>): ExportArchivePayload {
    return ExportArchivePayload.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ExportArchivePayload>): ExportArchivePayload {
    const message = createBaseExportArchivePayload();
    message.fileFormat = object.fileFormat ?? 0;
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
