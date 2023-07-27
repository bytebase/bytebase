/* eslint-disable */
import * as _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

/** InstanceOptions is the option for instances. */
export interface InstanceOptions {
  /**
   * The schema tenant mode is used to determine whether the instance is in schema tenant mode.
   * For Oracle schema tenant mode, the instance a Oracle database and the database is the Oracle schema.
   */
  schemaTenantMode: boolean;
}

/** InstanceMetadata is the metadata for instances. */
export interface InstanceMetadata {
  /**
   * The lower_case_table_name config for MySQL instances.
   * It is used to determine whether the table name and database name are case sensitive.
   */
  mysqlLowerCaseTableName: number;
}

function createBaseInstanceOptions(): InstanceOptions {
  return { schemaTenantMode: false };
}

export const InstanceOptions = {
  encode(message: InstanceOptions, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schemaTenantMode === true) {
      writer.uint32(8).bool(message.schemaTenantMode);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): InstanceOptions {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseInstanceOptions();
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

  fromJSON(object: any): InstanceOptions {
    return { schemaTenantMode: isSet(object.schemaTenantMode) ? Boolean(object.schemaTenantMode) : false };
  },

  toJSON(message: InstanceOptions): unknown {
    const obj: any = {};
    message.schemaTenantMode !== undefined && (obj.schemaTenantMode = message.schemaTenantMode);
    return obj;
  },

  create(base?: DeepPartial<InstanceOptions>): InstanceOptions {
    return InstanceOptions.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<InstanceOptions>): InstanceOptions {
    const message = createBaseInstanceOptions();
    message.schemaTenantMode = object.schemaTenantMode ?? false;
    return message;
  },
};

function createBaseInstanceMetadata(): InstanceMetadata {
  return { mysqlLowerCaseTableName: 0 };
}

export const InstanceMetadata = {
  encode(message: InstanceMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.mysqlLowerCaseTableName !== 0) {
      writer.uint32(8).int32(message.mysqlLowerCaseTableName);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): InstanceMetadata {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseInstanceMetadata();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.mysqlLowerCaseTableName = reader.int32();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): InstanceMetadata {
    return {
      mysqlLowerCaseTableName: isSet(object.mysqlLowerCaseTableName) ? Number(object.mysqlLowerCaseTableName) : 0,
    };
  },

  toJSON(message: InstanceMetadata): unknown {
    const obj: any = {};
    message.mysqlLowerCaseTableName !== undefined &&
      (obj.mysqlLowerCaseTableName = Math.round(message.mysqlLowerCaseTableName));
    return obj;
  },

  create(base?: DeepPartial<InstanceMetadata>): InstanceMetadata {
    return InstanceMetadata.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<InstanceMetadata>): InstanceMetadata {
    const message = createBaseInstanceMetadata();
    message.mysqlLowerCaseTableName = object.mysqlLowerCaseTableName ?? 0;
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
