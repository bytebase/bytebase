/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export interface TaskRunResult {
  detail: string;
  /** Format: instances/{instance}/databases/{database}/changeHistories/{changeHistory} */
  changeHistory: string;
  version: string;
  startPosition: TaskRunResult_Position | undefined;
  endPosition: TaskRunResult_Position | undefined;
}

/** The following fields are used for error reporting. */
export interface TaskRunResult_Position {
  line: number;
  column: number;
}

function createBaseTaskRunResult(): TaskRunResult {
  return { detail: "", changeHistory: "", version: "", startPosition: undefined, endPosition: undefined };
}

export const TaskRunResult = {
  encode(message: TaskRunResult, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.detail !== "") {
      writer.uint32(10).string(message.detail);
    }
    if (message.changeHistory !== "") {
      writer.uint32(18).string(message.changeHistory);
    }
    if (message.version !== "") {
      writer.uint32(26).string(message.version);
    }
    if (message.startPosition !== undefined) {
      TaskRunResult_Position.encode(message.startPosition, writer.uint32(34).fork()).ldelim();
    }
    if (message.endPosition !== undefined) {
      TaskRunResult_Position.encode(message.endPosition, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskRunResult {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTaskRunResult();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.detail = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.changeHistory = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.version = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.startPosition = TaskRunResult_Position.decode(reader, reader.uint32());
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.endPosition = TaskRunResult_Position.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TaskRunResult {
    return {
      detail: isSet(object.detail) ? globalThis.String(object.detail) : "",
      changeHistory: isSet(object.changeHistory) ? globalThis.String(object.changeHistory) : "",
      version: isSet(object.version) ? globalThis.String(object.version) : "",
      startPosition: isSet(object.startPosition) ? TaskRunResult_Position.fromJSON(object.startPosition) : undefined,
      endPosition: isSet(object.endPosition) ? TaskRunResult_Position.fromJSON(object.endPosition) : undefined,
    };
  },

  toJSON(message: TaskRunResult): unknown {
    const obj: any = {};
    if (message.detail !== "") {
      obj.detail = message.detail;
    }
    if (message.changeHistory !== "") {
      obj.changeHistory = message.changeHistory;
    }
    if (message.version !== "") {
      obj.version = message.version;
    }
    if (message.startPosition !== undefined) {
      obj.startPosition = TaskRunResult_Position.toJSON(message.startPosition);
    }
    if (message.endPosition !== undefined) {
      obj.endPosition = TaskRunResult_Position.toJSON(message.endPosition);
    }
    return obj;
  },

  create(base?: DeepPartial<TaskRunResult>): TaskRunResult {
    return TaskRunResult.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TaskRunResult>): TaskRunResult {
    const message = createBaseTaskRunResult();
    message.detail = object.detail ?? "";
    message.changeHistory = object.changeHistory ?? "";
    message.version = object.version ?? "";
    message.startPosition = (object.startPosition !== undefined && object.startPosition !== null)
      ? TaskRunResult_Position.fromPartial(object.startPosition)
      : undefined;
    message.endPosition = (object.endPosition !== undefined && object.endPosition !== null)
      ? TaskRunResult_Position.fromPartial(object.endPosition)
      : undefined;
    return message;
  },
};

function createBaseTaskRunResult_Position(): TaskRunResult_Position {
  return { line: 0, column: 0 };
}

export const TaskRunResult_Position = {
  encode(message: TaskRunResult_Position, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.line !== 0) {
      writer.uint32(8).int32(message.line);
    }
    if (message.column !== 0) {
      writer.uint32(16).int32(message.column);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskRunResult_Position {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTaskRunResult_Position();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.line = reader.int32();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.column = reader.int32();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TaskRunResult_Position {
    return {
      line: isSet(object.line) ? globalThis.Number(object.line) : 0,
      column: isSet(object.column) ? globalThis.Number(object.column) : 0,
    };
  },

  toJSON(message: TaskRunResult_Position): unknown {
    const obj: any = {};
    if (message.line !== 0) {
      obj.line = Math.round(message.line);
    }
    if (message.column !== 0) {
      obj.column = Math.round(message.column);
    }
    return obj;
  },

  create(base?: DeepPartial<TaskRunResult_Position>): TaskRunResult_Position {
    return TaskRunResult_Position.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TaskRunResult_Position>): TaskRunResult_Position {
    const message = createBaseTaskRunResult_Position();
    message.line = object.line ?? 0;
    message.column = object.column ?? 0;
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
