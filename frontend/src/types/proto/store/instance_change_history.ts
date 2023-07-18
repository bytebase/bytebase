/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { PushEvent } from "./vcs";

export const protobufPackage = "bytebase.store";

export interface InstanceChangeHistoryPayload {
  pushEvent?: PushEvent | undefined;
}

function createBaseInstanceChangeHistoryPayload(): InstanceChangeHistoryPayload {
  return { pushEvent: undefined };
}

export const InstanceChangeHistoryPayload = {
  encode(message: InstanceChangeHistoryPayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pushEvent !== undefined) {
      PushEvent.encode(message.pushEvent, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): InstanceChangeHistoryPayload {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseInstanceChangeHistoryPayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.pushEvent = PushEvent.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): InstanceChangeHistoryPayload {
    return { pushEvent: isSet(object.pushEvent) ? PushEvent.fromJSON(object.pushEvent) : undefined };
  },

  toJSON(message: InstanceChangeHistoryPayload): unknown {
    const obj: any = {};
    if (message.pushEvent !== undefined) {
      obj.pushEvent = PushEvent.toJSON(message.pushEvent);
    }
    return obj;
  },

  create(base?: DeepPartial<InstanceChangeHistoryPayload>): InstanceChangeHistoryPayload {
    return InstanceChangeHistoryPayload.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<InstanceChangeHistoryPayload>): InstanceChangeHistoryPayload {
    const message = createBaseInstanceChangeHistoryPayload();
    message.pushEvent = (object.pushEvent !== undefined && object.pushEvent !== null)
      ? PushEvent.fromPartial(object.pushEvent)
      : undefined;
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
