/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { Any } from "../protobuf/any";

export const protobufPackage = "google.rpc";

/**
 * The `Status` type defines a logical error model that is suitable for
 * different programming environments, including REST APIs and RPC APIs. It is
 * used by [gRPC](https://github.com/grpc). Each `Status` message contains
 * three pieces of data: error code, error message, and error details.
 *
 * You can find out more about this error model and how to work with it in the
 * [API Design Guide](https://cloud.google.com/apis/design/errors).
 */
export interface Status {
  /**
   * The status code, which should be an enum value of
   * [google.rpc.Code][google.rpc.Code].
   */
  code: number;
  /**
   * A developer-facing error message, which should be in English. Any
   * user-facing error message should be localized and sent in the
   * [google.rpc.Status.details][google.rpc.Status.details] field, or localized
   * by the client.
   */
  message: string;
  /**
   * A list of messages that carry the error details.  There is a common set of
   * message types for APIs to use.
   */
  details: Any[];
}

function createBaseStatus(): Status {
  return { code: 0, message: "", details: [] };
}

export const Status = {
  encode(message: Status, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.code !== 0) {
      writer.uint32(8).int32(message.code);
    }
    if (message.message !== "") {
      writer.uint32(18).string(message.message);
    }
    for (const v of message.details) {
      Any.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Status {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStatus();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.code = reader.int32();
          break;
        case 2:
          message.message = reader.string();
          break;
        case 3:
          message.details.push(Any.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Status {
    return {
      code: isSet(object.code) ? Number(object.code) : 0,
      message: isSet(object.message) ? String(object.message) : "",
      details: Array.isArray(object?.details) ? object.details.map((e: any) => Any.fromJSON(e)) : [],
    };
  },

  toJSON(message: Status): unknown {
    const obj: any = {};
    message.code !== undefined && (obj.code = Math.round(message.code));
    message.message !== undefined && (obj.message = message.message);
    if (message.details) {
      obj.details = message.details.map((e) => e ? Any.toJSON(e) : undefined);
    } else {
      obj.details = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<Status>): Status {
    const message = createBaseStatus();
    message.code = object.code ?? 0;
    message.message = object.message ?? "";
    message.details = object.details?.map((e) => Any.fromPartial(e)) || [];
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
