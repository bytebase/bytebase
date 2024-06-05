/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export interface ProjectWebhookPayload {
  /**
   * if direct_message is set, the notification is sent directly
   * to the persons and url will be ignored.
   * IM integration setting should be set for this function to work.
   */
  directMessage: boolean;
}

function createBaseProjectWebhookPayload(): ProjectWebhookPayload {
  return { directMessage: false };
}

export const ProjectWebhookPayload = {
  encode(message: ProjectWebhookPayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.directMessage === true) {
      writer.uint32(8).bool(message.directMessage);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ProjectWebhookPayload {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseProjectWebhookPayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.directMessage = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ProjectWebhookPayload {
    return { directMessage: isSet(object.directMessage) ? globalThis.Boolean(object.directMessage) : false };
  },

  toJSON(message: ProjectWebhookPayload): unknown {
    const obj: any = {};
    if (message.directMessage === true) {
      obj.directMessage = message.directMessage;
    }
    return obj;
  },

  create(base?: DeepPartial<ProjectWebhookPayload>): ProjectWebhookPayload {
    return ProjectWebhookPayload.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ProjectWebhookPayload>): ProjectWebhookPayload {
    const message = createBaseProjectWebhookPayload();
    message.directMessage = object.directMessage ?? false;
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
