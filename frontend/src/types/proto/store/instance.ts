/* eslint-disable */
import * as _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

/** InstanceOption is the option for instances. */
export interface InstanceOption {
  /**
   * The schema tenant mode is used to determine whether the instance is in schema tenant mode.
   * For Oracle schema tenant mode, the instance a Oracle database and the database is the Oracle schema.
   */
  schemaTenantMode: boolean;
}

function createBaseInstanceOption(): InstanceOption {
  return { schemaTenantMode: false };
}

export const InstanceOption = {
  encode(message: InstanceOption, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schemaTenantMode === true) {
      writer.uint32(8).bool(message.schemaTenantMode);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): InstanceOption {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseInstanceOption();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.schemaTenantMode = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): InstanceOption {
    return { schemaTenantMode: isSet(object.schemaTenantMode) ? Boolean(object.schemaTenantMode) : false };
  },

  toJSON(message: InstanceOption): unknown {
    const obj: any = {};
    message.schemaTenantMode !== undefined && (obj.schemaTenantMode = message.schemaTenantMode);
    return obj;
  },

  create(base?: DeepPartial<InstanceOption>): InstanceOption {
    return InstanceOption.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<InstanceOption>): InstanceOption {
    const message = createBaseInstanceOption();
    message.schemaTenantMode = object.schemaTenantMode ?? false;
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
