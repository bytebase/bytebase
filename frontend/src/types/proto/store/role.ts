/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export interface RolePermissions {
  items: string[];
}

function createBaseRolePermissions(): RolePermissions {
  return { items: [] };
}

export const RolePermissions = {
  encode(message: RolePermissions, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.items) {
      writer.uint32(10).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RolePermissions {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRolePermissions();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.items.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): RolePermissions {
    return { items: globalThis.Array.isArray(object?.items) ? object.items.map((e: any) => globalThis.String(e)) : [] };
  },

  toJSON(message: RolePermissions): unknown {
    const obj: any = {};
    if (message.items?.length) {
      obj.items = message.items;
    }
    return obj;
  },

  create(base?: DeepPartial<RolePermissions>): RolePermissions {
    return RolePermissions.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<RolePermissions>): RolePermissions {
    const message = createBaseRolePermissions();
    message.items = object.items?.map((e) => e) || [];
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
