/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { Duration } from "../google/protobuf/duration";
import { Timestamp } from "../google/protobuf/timestamp";
import Long = require("long");

export const protobufPackage = "bytebase.store";

/** SlowQueryStatistics is the slow query statistics. */
export interface SlowQueryStatistics {
  /** Items is the list of slow query statistics. */
  items: SlowQueryStatisticsItem[];
}

/** SlowQueryStatisticsItem is the item of slow query statistics. */
export interface SlowQueryStatisticsItem {
  /** sql_fingerprint is the fingerprint of the slow query. */
  sqlFingerprint: string;
  /** count is the number of slow queries with the same fingerprint. */
  count: number;
  /** latest_log_time is the time of the latest slow query with the same fingerprint. */
  latestLogTime?:
    | Date
    | undefined;
  /** The total query time of the slow query log. */
  totalQueryTime?:
    | Duration
    | undefined;
  /** The maximum query time of the slow query log. */
  maximumQueryTime?:
    | Duration
    | undefined;
  /** The total rows sent of the slow query log. */
  totalRowsSent: number;
  /** The maximum rows sent of the slow query log. */
  maximumRowsSent: number;
  /** The total rows examined of the slow query log. */
  totalRowsExamined: number;
  /** The maximum rows examined of the slow query log. */
  maximumRowsExamined: number;
  /** samples are the details of the sample slow queries with the same fingerprint. */
  samples: SlowQueryDetails[];
}

/** SlowQueryDetails is the details of a slow query. */
export interface SlowQueryDetails {
  /** start_time is the start time of the slow query. */
  startTime?:
    | Date
    | undefined;
  /** query_time is the query time of the slow query. */
  queryTime?:
    | Duration
    | undefined;
  /** lock_time is the lock time of the slow query. */
  lockTime?:
    | Duration
    | undefined;
  /** rows_sent is the number of rows sent by the slow query. */
  rowsSent: number;
  /** rows_examined is the number of rows examined by the slow query. */
  rowsExamined: number;
  /** sql_text is the SQL text of the slow query. */
  sqlText: string;
}

function createBaseSlowQueryStatistics(): SlowQueryStatistics {
  return { items: [] };
}

export const SlowQueryStatistics = {
  encode(message: SlowQueryStatistics, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.items) {
      SlowQueryStatisticsItem.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SlowQueryStatistics {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSlowQueryStatistics();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.items.push(SlowQueryStatisticsItem.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SlowQueryStatistics {
    return {
      items: Array.isArray(object?.items) ? object.items.map((e: any) => SlowQueryStatisticsItem.fromJSON(e)) : [],
    };
  },

  toJSON(message: SlowQueryStatistics): unknown {
    const obj: any = {};
    if (message.items?.length) {
      obj.items = message.items.map((e) => SlowQueryStatisticsItem.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<SlowQueryStatistics>): SlowQueryStatistics {
    return SlowQueryStatistics.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SlowQueryStatistics>): SlowQueryStatistics {
    const message = createBaseSlowQueryStatistics();
    message.items = object.items?.map((e) => SlowQueryStatisticsItem.fromPartial(e)) || [];
    return message;
  },
};

function createBaseSlowQueryStatisticsItem(): SlowQueryStatisticsItem {
  return {
    sqlFingerprint: "",
    count: 0,
    latestLogTime: undefined,
    totalQueryTime: undefined,
    maximumQueryTime: undefined,
    totalRowsSent: 0,
    maximumRowsSent: 0,
    totalRowsExamined: 0,
    maximumRowsExamined: 0,
    samples: [],
  };
}

export const SlowQueryStatisticsItem = {
  encode(message: SlowQueryStatisticsItem, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sqlFingerprint !== "") {
      writer.uint32(10).string(message.sqlFingerprint);
    }
    if (message.count !== 0) {
      writer.uint32(16).int64(message.count);
    }
    if (message.latestLogTime !== undefined) {
      Timestamp.encode(toTimestamp(message.latestLogTime), writer.uint32(26).fork()).ldelim();
    }
    if (message.totalQueryTime !== undefined) {
      Duration.encode(message.totalQueryTime, writer.uint32(34).fork()).ldelim();
    }
    if (message.maximumQueryTime !== undefined) {
      Duration.encode(message.maximumQueryTime, writer.uint32(42).fork()).ldelim();
    }
    if (message.totalRowsSent !== 0) {
      writer.uint32(48).int64(message.totalRowsSent);
    }
    if (message.maximumRowsSent !== 0) {
      writer.uint32(56).int64(message.maximumRowsSent);
    }
    if (message.totalRowsExamined !== 0) {
      writer.uint32(64).int64(message.totalRowsExamined);
    }
    if (message.maximumRowsExamined !== 0) {
      writer.uint32(72).int64(message.maximumRowsExamined);
    }
    for (const v of message.samples) {
      SlowQueryDetails.encode(v!, writer.uint32(82).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SlowQueryStatisticsItem {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSlowQueryStatisticsItem();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.sqlFingerprint = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.count = longToNumber(reader.int64() as Long);
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.latestLogTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.totalQueryTime = Duration.decode(reader, reader.uint32());
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.maximumQueryTime = Duration.decode(reader, reader.uint32());
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.totalRowsSent = longToNumber(reader.int64() as Long);
          continue;
        case 7:
          if (tag !== 56) {
            break;
          }

          message.maximumRowsSent = longToNumber(reader.int64() as Long);
          continue;
        case 8:
          if (tag !== 64) {
            break;
          }

          message.totalRowsExamined = longToNumber(reader.int64() as Long);
          continue;
        case 9:
          if (tag !== 72) {
            break;
          }

          message.maximumRowsExamined = longToNumber(reader.int64() as Long);
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.samples.push(SlowQueryDetails.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SlowQueryStatisticsItem {
    return {
      sqlFingerprint: isSet(object.sqlFingerprint) ? String(object.sqlFingerprint) : "",
      count: isSet(object.count) ? Number(object.count) : 0,
      latestLogTime: isSet(object.latestLogTime) ? fromJsonTimestamp(object.latestLogTime) : undefined,
      totalQueryTime: isSet(object.totalQueryTime) ? Duration.fromJSON(object.totalQueryTime) : undefined,
      maximumQueryTime: isSet(object.maximumQueryTime) ? Duration.fromJSON(object.maximumQueryTime) : undefined,
      totalRowsSent: isSet(object.totalRowsSent) ? Number(object.totalRowsSent) : 0,
      maximumRowsSent: isSet(object.maximumRowsSent) ? Number(object.maximumRowsSent) : 0,
      totalRowsExamined: isSet(object.totalRowsExamined) ? Number(object.totalRowsExamined) : 0,
      maximumRowsExamined: isSet(object.maximumRowsExamined) ? Number(object.maximumRowsExamined) : 0,
      samples: Array.isArray(object?.samples) ? object.samples.map((e: any) => SlowQueryDetails.fromJSON(e)) : [],
    };
  },

  toJSON(message: SlowQueryStatisticsItem): unknown {
    const obj: any = {};
    if (message.sqlFingerprint !== "") {
      obj.sqlFingerprint = message.sqlFingerprint;
    }
    if (message.count !== 0) {
      obj.count = Math.round(message.count);
    }
    if (message.latestLogTime !== undefined) {
      obj.latestLogTime = message.latestLogTime.toISOString();
    }
    if (message.totalQueryTime !== undefined) {
      obj.totalQueryTime = Duration.toJSON(message.totalQueryTime);
    }
    if (message.maximumQueryTime !== undefined) {
      obj.maximumQueryTime = Duration.toJSON(message.maximumQueryTime);
    }
    if (message.totalRowsSent !== 0) {
      obj.totalRowsSent = Math.round(message.totalRowsSent);
    }
    if (message.maximumRowsSent !== 0) {
      obj.maximumRowsSent = Math.round(message.maximumRowsSent);
    }
    if (message.totalRowsExamined !== 0) {
      obj.totalRowsExamined = Math.round(message.totalRowsExamined);
    }
    if (message.maximumRowsExamined !== 0) {
      obj.maximumRowsExamined = Math.round(message.maximumRowsExamined);
    }
    if (message.samples?.length) {
      obj.samples = message.samples.map((e) => SlowQueryDetails.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<SlowQueryStatisticsItem>): SlowQueryStatisticsItem {
    return SlowQueryStatisticsItem.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SlowQueryStatisticsItem>): SlowQueryStatisticsItem {
    const message = createBaseSlowQueryStatisticsItem();
    message.sqlFingerprint = object.sqlFingerprint ?? "";
    message.count = object.count ?? 0;
    message.latestLogTime = object.latestLogTime ?? undefined;
    message.totalQueryTime = (object.totalQueryTime !== undefined && object.totalQueryTime !== null)
      ? Duration.fromPartial(object.totalQueryTime)
      : undefined;
    message.maximumQueryTime = (object.maximumQueryTime !== undefined && object.maximumQueryTime !== null)
      ? Duration.fromPartial(object.maximumQueryTime)
      : undefined;
    message.totalRowsSent = object.totalRowsSent ?? 0;
    message.maximumRowsSent = object.maximumRowsSent ?? 0;
    message.totalRowsExamined = object.totalRowsExamined ?? 0;
    message.maximumRowsExamined = object.maximumRowsExamined ?? 0;
    message.samples = object.samples?.map((e) => SlowQueryDetails.fromPartial(e)) || [];
    return message;
  },
};

function createBaseSlowQueryDetails(): SlowQueryDetails {
  return { startTime: undefined, queryTime: undefined, lockTime: undefined, rowsSent: 0, rowsExamined: 0, sqlText: "" };
}

export const SlowQueryDetails = {
  encode(message: SlowQueryDetails, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.startTime !== undefined) {
      Timestamp.encode(toTimestamp(message.startTime), writer.uint32(10).fork()).ldelim();
    }
    if (message.queryTime !== undefined) {
      Duration.encode(message.queryTime, writer.uint32(18).fork()).ldelim();
    }
    if (message.lockTime !== undefined) {
      Duration.encode(message.lockTime, writer.uint32(26).fork()).ldelim();
    }
    if (message.rowsSent !== 0) {
      writer.uint32(32).int64(message.rowsSent);
    }
    if (message.rowsExamined !== 0) {
      writer.uint32(40).int64(message.rowsExamined);
    }
    if (message.sqlText !== "") {
      writer.uint32(50).string(message.sqlText);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SlowQueryDetails {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSlowQueryDetails();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.startTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.queryTime = Duration.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.lockTime = Duration.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.rowsSent = longToNumber(reader.int64() as Long);
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.rowsExamined = longToNumber(reader.int64() as Long);
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.sqlText = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SlowQueryDetails {
    return {
      startTime: isSet(object.startTime) ? fromJsonTimestamp(object.startTime) : undefined,
      queryTime: isSet(object.queryTime) ? Duration.fromJSON(object.queryTime) : undefined,
      lockTime: isSet(object.lockTime) ? Duration.fromJSON(object.lockTime) : undefined,
      rowsSent: isSet(object.rowsSent) ? Number(object.rowsSent) : 0,
      rowsExamined: isSet(object.rowsExamined) ? Number(object.rowsExamined) : 0,
      sqlText: isSet(object.sqlText) ? String(object.sqlText) : "",
    };
  },

  toJSON(message: SlowQueryDetails): unknown {
    const obj: any = {};
    if (message.startTime !== undefined) {
      obj.startTime = message.startTime.toISOString();
    }
    if (message.queryTime !== undefined) {
      obj.queryTime = Duration.toJSON(message.queryTime);
    }
    if (message.lockTime !== undefined) {
      obj.lockTime = Duration.toJSON(message.lockTime);
    }
    if (message.rowsSent !== 0) {
      obj.rowsSent = Math.round(message.rowsSent);
    }
    if (message.rowsExamined !== 0) {
      obj.rowsExamined = Math.round(message.rowsExamined);
    }
    if (message.sqlText !== "") {
      obj.sqlText = message.sqlText;
    }
    return obj;
  },

  create(base?: DeepPartial<SlowQueryDetails>): SlowQueryDetails {
    return SlowQueryDetails.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SlowQueryDetails>): SlowQueryDetails {
    const message = createBaseSlowQueryDetails();
    message.startTime = object.startTime ?? undefined;
    message.queryTime = (object.queryTime !== undefined && object.queryTime !== null)
      ? Duration.fromPartial(object.queryTime)
      : undefined;
    message.lockTime = (object.lockTime !== undefined && object.lockTime !== null)
      ? Duration.fromPartial(object.lockTime)
      : undefined;
    message.rowsSent = object.rowsSent ?? 0;
    message.rowsExamined = object.rowsExamined ?? 0;
    message.sqlText = object.sqlText ?? "";
    return message;
  },
};

declare const self: any | undefined;
declare const window: any | undefined;
declare const global: any | undefined;
const tsProtoGlobalThis: any = (() => {
  if (typeof globalThis !== "undefined") {
    return globalThis;
  }
  if (typeof self !== "undefined") {
    return self;
  }
  if (typeof window !== "undefined") {
    return window;
  }
  if (typeof global !== "undefined") {
    return global;
  }
  throw "Unable to locate global object";
})();

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

function longToNumber(long: Long): number {
  if (long.gt(Number.MAX_SAFE_INTEGER)) {
    throw new tsProtoGlobalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
  }
  return long.toNumber();
}

if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
