/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { Duration } from "../google/protobuf/duration";
import { Timestamp } from "../google/protobuf/timestamp";

export const protobufPackage = "bytebase.store";

/** SlowQueryStatistics is a summary of slow queries. */
export interface SlowQueryStatistics {
  /** sql_fingerprint is the fingerprint of the slow query. */
  sqlFingerprint: string;
  /** count is the number of slow queries with the same fingerprint. */
  count: number;
  /** latest_log_time is the time of the latest slow query with the same fingerprint. */
  latestLogTime?: Date;
  /** samples are the details of the sample slow queries with the same fingerprint. */
  samples: SlowQueryDetails[];
}

/** SlowQueryDetails is the details of a slow query. */
export interface SlowQueryDetails {
  /** start_time is the start time of the slow query. */
  startTime?: Date;
  /** query_time is the query time of the slow query. */
  queryTime?: Duration;
  /** lock_time is the lock time of the slow query. */
  lockTime?: Duration;
  /** rows_sent is the number of rows sent by the slow query. */
  rowsSent: number;
  /** rows_examined is the number of rows examined by the slow query. */
  rowsExamined: number;
  /** sql_text is the SQL text of the slow query. */
  sqlText: string;
}

function createBaseSlowQueryStatistics(): SlowQueryStatistics {
  return { sqlFingerprint: "", count: 0, latestLogTime: undefined, samples: [] };
}

export const SlowQueryStatistics = {
  encode(message: SlowQueryStatistics, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sqlFingerprint !== "") {
      writer.uint32(10).string(message.sqlFingerprint);
    }
    if (message.count !== 0) {
      writer.uint32(16).int32(message.count);
    }
    if (message.latestLogTime !== undefined) {
      Timestamp.encode(toTimestamp(message.latestLogTime), writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.samples) {
      SlowQueryDetails.encode(v!, writer.uint32(34).fork()).ldelim();
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
          if (tag != 10) {
            break;
          }

          message.sqlFingerprint = reader.string();
          continue;
        case 2:
          if (tag != 16) {
            break;
          }

          message.count = reader.int32();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.latestLogTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.samples.push(SlowQueryDetails.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SlowQueryStatistics {
    return {
      sqlFingerprint: isSet(object.sqlFingerprint) ? String(object.sqlFingerprint) : "",
      count: isSet(object.count) ? Number(object.count) : 0,
      latestLogTime: isSet(object.latestLogTime) ? fromJsonTimestamp(object.latestLogTime) : undefined,
      samples: Array.isArray(object?.samples) ? object.samples.map((e: any) => SlowQueryDetails.fromJSON(e)) : [],
    };
  },

  toJSON(message: SlowQueryStatistics): unknown {
    const obj: any = {};
    message.sqlFingerprint !== undefined && (obj.sqlFingerprint = message.sqlFingerprint);
    message.count !== undefined && (obj.count = Math.round(message.count));
    message.latestLogTime !== undefined && (obj.latestLogTime = message.latestLogTime.toISOString());
    if (message.samples) {
      obj.samples = message.samples.map((e) => e ? SlowQueryDetails.toJSON(e) : undefined);
    } else {
      obj.samples = [];
    }
    return obj;
  },

  create(base?: DeepPartial<SlowQueryStatistics>): SlowQueryStatistics {
    return SlowQueryStatistics.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SlowQueryStatistics>): SlowQueryStatistics {
    const message = createBaseSlowQueryStatistics();
    message.sqlFingerprint = object.sqlFingerprint ?? "";
    message.count = object.count ?? 0;
    message.latestLogTime = object.latestLogTime ?? undefined;
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
      writer.uint32(32).int32(message.rowsSent);
    }
    if (message.rowsExamined !== 0) {
      writer.uint32(40).int32(message.rowsExamined);
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
          if (tag != 10) {
            break;
          }

          message.startTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.queryTime = Duration.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.lockTime = Duration.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag != 32) {
            break;
          }

          message.rowsSent = reader.int32();
          continue;
        case 5:
          if (tag != 40) {
            break;
          }

          message.rowsExamined = reader.int32();
          continue;
        case 6:
          if (tag != 50) {
            break;
          }

          message.sqlText = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
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
    message.startTime !== undefined && (obj.startTime = message.startTime.toISOString());
    message.queryTime !== undefined &&
      (obj.queryTime = message.queryTime ? Duration.toJSON(message.queryTime) : undefined);
    message.lockTime !== undefined && (obj.lockTime = message.lockTime ? Duration.toJSON(message.lockTime) : undefined);
    message.rowsSent !== undefined && (obj.rowsSent = Math.round(message.rowsSent));
    message.rowsExamined !== undefined && (obj.rowsExamined = Math.round(message.rowsExamined));
    message.sqlText !== undefined && (obj.sqlText = message.sqlText);
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
  let millis = t.seconds * 1_000;
  millis += t.nanos / 1_000_000;
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
