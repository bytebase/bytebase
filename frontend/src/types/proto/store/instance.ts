/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { Duration } from "../google/protobuf/duration";
import { Timestamp } from "../google/protobuf/timestamp";

export const protobufPackage = "bytebase.store";

/** InstanceOptions is the option for instances. */
export interface InstanceOptions {
  /**
   * The schema tenant mode is used to determine whether the instance is in schema tenant mode.
   * For Oracle schema tenant mode, the instance a Oracle database and the database is the Oracle schema.
   */
  schemaTenantMode: boolean;
  /** How often the instance is synced. */
  syncInterval?: Duration | undefined;
}

/** InstanceMetadata is the metadata for instances. */
export interface InstanceMetadata {
  /**
   * The lower_case_table_names config for MySQL instances.
   * It is used to determine whether the table names and database names are case sensitive.
   */
  mysqlLowerCaseTableNames: number;
  lastSyncTime?: Date | undefined;
}

function createBaseInstanceOptions(): InstanceOptions {
  return { schemaTenantMode: false, syncInterval: undefined };
}

export const InstanceOptions = {
  encode(message: InstanceOptions, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schemaTenantMode === true) {
      writer.uint32(8).bool(message.schemaTenantMode);
    }
    if (message.syncInterval !== undefined) {
      Duration.encode(message.syncInterval, writer.uint32(18).fork()).ldelim();
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
        case 2:
          if (tag !== 18) {
            break;
          }

          message.syncInterval = Duration.decode(reader, reader.uint32());
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
    return {
      schemaTenantMode: isSet(object.schemaTenantMode) ? Boolean(object.schemaTenantMode) : false,
      syncInterval: isSet(object.syncInterval) ? Duration.fromJSON(object.syncInterval) : undefined,
    };
  },

  toJSON(message: InstanceOptions): unknown {
    const obj: any = {};
    message.schemaTenantMode !== undefined && (obj.schemaTenantMode = message.schemaTenantMode);
    message.syncInterval !== undefined &&
      (obj.syncInterval = message.syncInterval ? Duration.toJSON(message.syncInterval) : undefined);
    return obj;
  },

  create(base?: DeepPartial<InstanceOptions>): InstanceOptions {
    return InstanceOptions.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<InstanceOptions>): InstanceOptions {
    const message = createBaseInstanceOptions();
    message.schemaTenantMode = object.schemaTenantMode ?? false;
    message.syncInterval = (object.syncInterval !== undefined && object.syncInterval !== null)
      ? Duration.fromPartial(object.syncInterval)
      : undefined;
    return message;
  },
};

function createBaseInstanceMetadata(): InstanceMetadata {
  return { mysqlLowerCaseTableNames: 0, lastSyncTime: undefined };
}

export const InstanceMetadata = {
  encode(message: InstanceMetadata, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.mysqlLowerCaseTableNames !== 0) {
      writer.uint32(8).int32(message.mysqlLowerCaseTableNames);
    }
    if (message.lastSyncTime !== undefined) {
      Timestamp.encode(toTimestamp(message.lastSyncTime), writer.uint32(18).fork()).ldelim();
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

          message.mysqlLowerCaseTableNames = reader.int32();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.lastSyncTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
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
      mysqlLowerCaseTableNames: isSet(object.mysqlLowerCaseTableNames) ? Number(object.mysqlLowerCaseTableNames) : 0,
      lastSyncTime: isSet(object.lastSyncTime) ? fromJsonTimestamp(object.lastSyncTime) : undefined,
    };
  },

  toJSON(message: InstanceMetadata): unknown {
    const obj: any = {};
    message.mysqlLowerCaseTableNames !== undefined &&
      (obj.mysqlLowerCaseTableNames = Math.round(message.mysqlLowerCaseTableNames));
    message.lastSyncTime !== undefined && (obj.lastSyncTime = message.lastSyncTime.toISOString());
    return obj;
  },

  create(base?: DeepPartial<InstanceMetadata>): InstanceMetadata {
    return InstanceMetadata.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<InstanceMetadata>): InstanceMetadata {
    const message = createBaseInstanceMetadata();
    message.mysqlLowerCaseTableNames = object.mysqlLowerCaseTableNames ?? 0;
    message.lastSyncTime = object.lastSyncTime ?? undefined;
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function toTimestamp(date: Date): Timestamp {
  const seconds = date.getTime() / 1_000;
  const nanos = (date.getTime() % 1_000) * 1_000_000;
  return { seconds, nanos };
}

function fromTimestamp(t: Timestamp): Date {
  let millis = (t.seconds || 0) * 1_000;
  millis += (t.nanos || 0) / 1_000_000;
  return new Date(millis);
}

function fromJsonTimestamp(o: any): Date {
  if (o instanceof Date) {
    return o;
  } else if (typeof o === "string") {
    return new Date(o);
  } else {
    return fromTimestamp(Timestamp.fromJSON(o));
  }
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
