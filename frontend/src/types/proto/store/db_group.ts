/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export interface DatabaseGroupPayload {
  multitenancy: boolean;
}

function createBaseDatabaseGroupPayload(): DatabaseGroupPayload {
  return { multitenancy: false };
}

export const DatabaseGroupPayload = {
  encode(message: DatabaseGroupPayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.multitenancy === true) {
      writer.uint32(8).bool(message.multitenancy);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DatabaseGroupPayload {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDatabaseGroupPayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.multitenancy = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DatabaseGroupPayload {
    return { multitenancy: isSet(object.multitenancy) ? globalThis.Boolean(object.multitenancy) : false };
  },

  toJSON(message: DatabaseGroupPayload): unknown {
    const obj: any = {};
    if (message.multitenancy === true) {
      obj.multitenancy = message.multitenancy;
    }
    return obj;
  },

  create(base?: DeepPartial<DatabaseGroupPayload>): DatabaseGroupPayload {
    return DatabaseGroupPayload.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DatabaseGroupPayload>): DatabaseGroupPayload {
    const message = createBaseDatabaseGroupPayload();
    message.multitenancy = object.multitenancy ?? false;
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
