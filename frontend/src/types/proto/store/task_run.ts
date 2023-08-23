/* eslint-disable */
import * as _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export interface TaskRunResult {
  detail: string;
  /** Format: instances/{instance}/databases/{database}/changeHistories/{changeHistory} */
  changeHistory: string;
  version: string;
}

function createBaseTaskRunResult(): TaskRunResult {
  return { detail: "", changeHistory: "", version: "" };
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
      detail: isSet(object.detail) ? String(object.detail) : "",
      changeHistory: isSet(object.changeHistory) ? String(object.changeHistory) : "",
      version: isSet(object.version) ? String(object.version) : "",
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
