/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export interface TaskRunLog {
  type: TaskRunLog_Type;
  schemaDumpStart: TaskRunLog_SchemaDumpStart | undefined;
  schemaDumpEnd: TaskRunLog_SchemaDumpEnd | undefined;
  commandExecute: TaskRunLog_CommandExecute | undefined;
  commandResponse: TaskRunLog_CommandResponse | undefined;
}

export enum TaskRunLog_Type {
  TYPE_UNSPECIFIED = "TYPE_UNSPECIFIED",
  SCHEMA_DUMP_START = "SCHEMA_DUMP_START",
  SCHEMA_DUMP_END = "SCHEMA_DUMP_END",
  COMMAND_EXECUTE = "COMMAND_EXECUTE",
  COMMAND_RESPONSE = "COMMAND_RESPONSE",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function taskRunLog_TypeFromJSON(object: any): TaskRunLog_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return TaskRunLog_Type.TYPE_UNSPECIFIED;
    case 1:
    case "SCHEMA_DUMP_START":
      return TaskRunLog_Type.SCHEMA_DUMP_START;
    case 2:
    case "SCHEMA_DUMP_END":
      return TaskRunLog_Type.SCHEMA_DUMP_END;
    case 3:
    case "COMMAND_EXECUTE":
      return TaskRunLog_Type.COMMAND_EXECUTE;
    case 4:
    case "COMMAND_RESPONSE":
      return TaskRunLog_Type.COMMAND_RESPONSE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return TaskRunLog_Type.UNRECOGNIZED;
  }
}

export function taskRunLog_TypeToJSON(object: TaskRunLog_Type): string {
  switch (object) {
    case TaskRunLog_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case TaskRunLog_Type.SCHEMA_DUMP_START:
      return "SCHEMA_DUMP_START";
    case TaskRunLog_Type.SCHEMA_DUMP_END:
      return "SCHEMA_DUMP_END";
    case TaskRunLog_Type.COMMAND_EXECUTE:
      return "COMMAND_EXECUTE";
    case TaskRunLog_Type.COMMAND_RESPONSE:
      return "COMMAND_RESPONSE";
    case TaskRunLog_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function taskRunLog_TypeToNumber(object: TaskRunLog_Type): number {
  switch (object) {
    case TaskRunLog_Type.TYPE_UNSPECIFIED:
      return 0;
    case TaskRunLog_Type.SCHEMA_DUMP_START:
      return 1;
    case TaskRunLog_Type.SCHEMA_DUMP_END:
      return 2;
    case TaskRunLog_Type.COMMAND_EXECUTE:
      return 3;
    case TaskRunLog_Type.COMMAND_RESPONSE:
      return 4;
    case TaskRunLog_Type.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface TaskRunLog_SchemaDumpStart {
}

export interface TaskRunLog_SchemaDumpEnd {
  error: string;
}

export interface TaskRunLog_CommandExecute {
  /** Executed commands are in range [command_index, command_index + command_count). */
  commandIndex: number;
  commandCount: number;
}

export interface TaskRunLog_CommandResponse {
  /** Executed commands are in range [command_index, command_index + command_count). */
  commandIndex: number;
  commandCount: number;
  error: string;
  affectedRows: number;
  /**
   * `all_affected_rows` is the affected rows of each command.
   * `all_affected_rows` may be unavailable if the database driver doesn't support it. Caller should fallback to `affected_rows` in that case.
   */
  allAffectedRows: number[];
}

function createBaseTaskRunLog(): TaskRunLog {
  return {
    type: TaskRunLog_Type.TYPE_UNSPECIFIED,
    schemaDumpStart: undefined,
    schemaDumpEnd: undefined,
    commandExecute: undefined,
    commandResponse: undefined,
  };
}

export const TaskRunLog = {
  encode(message: TaskRunLog, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.type !== TaskRunLog_Type.TYPE_UNSPECIFIED) {
      writer.uint32(8).int32(taskRunLog_TypeToNumber(message.type));
    }
    if (message.schemaDumpStart !== undefined) {
      TaskRunLog_SchemaDumpStart.encode(message.schemaDumpStart, writer.uint32(18).fork()).ldelim();
    }
    if (message.schemaDumpEnd !== undefined) {
      TaskRunLog_SchemaDumpEnd.encode(message.schemaDumpEnd, writer.uint32(26).fork()).ldelim();
    }
    if (message.commandExecute !== undefined) {
      TaskRunLog_CommandExecute.encode(message.commandExecute, writer.uint32(34).fork()).ldelim();
    }
    if (message.commandResponse !== undefined) {
      TaskRunLog_CommandResponse.encode(message.commandResponse, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskRunLog {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTaskRunLog();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.type = taskRunLog_TypeFromJSON(reader.int32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.schemaDumpStart = TaskRunLog_SchemaDumpStart.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.schemaDumpEnd = TaskRunLog_SchemaDumpEnd.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.commandExecute = TaskRunLog_CommandExecute.decode(reader, reader.uint32());
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.commandResponse = TaskRunLog_CommandResponse.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TaskRunLog {
    return {
      type: isSet(object.type) ? taskRunLog_TypeFromJSON(object.type) : TaskRunLog_Type.TYPE_UNSPECIFIED,
      schemaDumpStart: isSet(object.schemaDumpStart)
        ? TaskRunLog_SchemaDumpStart.fromJSON(object.schemaDumpStart)
        : undefined,
      schemaDumpEnd: isSet(object.schemaDumpEnd) ? TaskRunLog_SchemaDumpEnd.fromJSON(object.schemaDumpEnd) : undefined,
      commandExecute: isSet(object.commandExecute)
        ? TaskRunLog_CommandExecute.fromJSON(object.commandExecute)
        : undefined,
      commandResponse: isSet(object.commandResponse)
        ? TaskRunLog_CommandResponse.fromJSON(object.commandResponse)
        : undefined,
    };
  },

  toJSON(message: TaskRunLog): unknown {
    const obj: any = {};
    if (message.type !== TaskRunLog_Type.TYPE_UNSPECIFIED) {
      obj.type = taskRunLog_TypeToJSON(message.type);
    }
    if (message.schemaDumpStart !== undefined) {
      obj.schemaDumpStart = TaskRunLog_SchemaDumpStart.toJSON(message.schemaDumpStart);
    }
    if (message.schemaDumpEnd !== undefined) {
      obj.schemaDumpEnd = TaskRunLog_SchemaDumpEnd.toJSON(message.schemaDumpEnd);
    }
    if (message.commandExecute !== undefined) {
      obj.commandExecute = TaskRunLog_CommandExecute.toJSON(message.commandExecute);
    }
    if (message.commandResponse !== undefined) {
      obj.commandResponse = TaskRunLog_CommandResponse.toJSON(message.commandResponse);
    }
    return obj;
  },

  create(base?: DeepPartial<TaskRunLog>): TaskRunLog {
    return TaskRunLog.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TaskRunLog>): TaskRunLog {
    const message = createBaseTaskRunLog();
    message.type = object.type ?? TaskRunLog_Type.TYPE_UNSPECIFIED;
    message.schemaDumpStart = (object.schemaDumpStart !== undefined && object.schemaDumpStart !== null)
      ? TaskRunLog_SchemaDumpStart.fromPartial(object.schemaDumpStart)
      : undefined;
    message.schemaDumpEnd = (object.schemaDumpEnd !== undefined && object.schemaDumpEnd !== null)
      ? TaskRunLog_SchemaDumpEnd.fromPartial(object.schemaDumpEnd)
      : undefined;
    message.commandExecute = (object.commandExecute !== undefined && object.commandExecute !== null)
      ? TaskRunLog_CommandExecute.fromPartial(object.commandExecute)
      : undefined;
    message.commandResponse = (object.commandResponse !== undefined && object.commandResponse !== null)
      ? TaskRunLog_CommandResponse.fromPartial(object.commandResponse)
      : undefined;
    return message;
  },
};

function createBaseTaskRunLog_SchemaDumpStart(): TaskRunLog_SchemaDumpStart {
  return {};
}

export const TaskRunLog_SchemaDumpStart = {
  encode(_: TaskRunLog_SchemaDumpStart, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskRunLog_SchemaDumpStart {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTaskRunLog_SchemaDumpStart();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(_: any): TaskRunLog_SchemaDumpStart {
    return {};
  },

  toJSON(_: TaskRunLog_SchemaDumpStart): unknown {
    const obj: any = {};
    return obj;
  },

  create(base?: DeepPartial<TaskRunLog_SchemaDumpStart>): TaskRunLog_SchemaDumpStart {
    return TaskRunLog_SchemaDumpStart.fromPartial(base ?? {});
  },
  fromPartial(_: DeepPartial<TaskRunLog_SchemaDumpStart>): TaskRunLog_SchemaDumpStart {
    const message = createBaseTaskRunLog_SchemaDumpStart();
    return message;
  },
};

function createBaseTaskRunLog_SchemaDumpEnd(): TaskRunLog_SchemaDumpEnd {
  return { error: "" };
}

export const TaskRunLog_SchemaDumpEnd = {
  encode(message: TaskRunLog_SchemaDumpEnd, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.error !== "") {
      writer.uint32(10).string(message.error);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskRunLog_SchemaDumpEnd {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTaskRunLog_SchemaDumpEnd();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.error = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TaskRunLog_SchemaDumpEnd {
    return { error: isSet(object.error) ? globalThis.String(object.error) : "" };
  },

  toJSON(message: TaskRunLog_SchemaDumpEnd): unknown {
    const obj: any = {};
    if (message.error !== "") {
      obj.error = message.error;
    }
    return obj;
  },

  create(base?: DeepPartial<TaskRunLog_SchemaDumpEnd>): TaskRunLog_SchemaDumpEnd {
    return TaskRunLog_SchemaDumpEnd.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TaskRunLog_SchemaDumpEnd>): TaskRunLog_SchemaDumpEnd {
    const message = createBaseTaskRunLog_SchemaDumpEnd();
    message.error = object.error ?? "";
    return message;
  },
};

function createBaseTaskRunLog_CommandExecute(): TaskRunLog_CommandExecute {
  return { commandIndex: 0, commandCount: 0 };
}

export const TaskRunLog_CommandExecute = {
  encode(message: TaskRunLog_CommandExecute, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.commandIndex !== 0) {
      writer.uint32(8).int32(message.commandIndex);
    }
    if (message.commandCount !== 0) {
      writer.uint32(16).int32(message.commandCount);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskRunLog_CommandExecute {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTaskRunLog_CommandExecute();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.commandIndex = reader.int32();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.commandCount = reader.int32();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TaskRunLog_CommandExecute {
    return {
      commandIndex: isSet(object.commandIndex) ? globalThis.Number(object.commandIndex) : 0,
      commandCount: isSet(object.commandCount) ? globalThis.Number(object.commandCount) : 0,
    };
  },

  toJSON(message: TaskRunLog_CommandExecute): unknown {
    const obj: any = {};
    if (message.commandIndex !== 0) {
      obj.commandIndex = Math.round(message.commandIndex);
    }
    if (message.commandCount !== 0) {
      obj.commandCount = Math.round(message.commandCount);
    }
    return obj;
  },

  create(base?: DeepPartial<TaskRunLog_CommandExecute>): TaskRunLog_CommandExecute {
    return TaskRunLog_CommandExecute.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TaskRunLog_CommandExecute>): TaskRunLog_CommandExecute {
    const message = createBaseTaskRunLog_CommandExecute();
    message.commandIndex = object.commandIndex ?? 0;
    message.commandCount = object.commandCount ?? 0;
    return message;
  },
};

function createBaseTaskRunLog_CommandResponse(): TaskRunLog_CommandResponse {
  return { commandIndex: 0, commandCount: 0, error: "", affectedRows: 0, allAffectedRows: [] };
}

export const TaskRunLog_CommandResponse = {
  encode(message: TaskRunLog_CommandResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.commandIndex !== 0) {
      writer.uint32(8).int32(message.commandIndex);
    }
    if (message.commandCount !== 0) {
      writer.uint32(16).int32(message.commandCount);
    }
    if (message.error !== "") {
      writer.uint32(26).string(message.error);
    }
    if (message.affectedRows !== 0) {
      writer.uint32(32).int32(message.affectedRows);
    }
    writer.uint32(42).fork();
    for (const v of message.allAffectedRows) {
      writer.int32(v);
    }
    writer.ldelim();
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TaskRunLog_CommandResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTaskRunLog_CommandResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.commandIndex = reader.int32();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.commandCount = reader.int32();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.error = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.affectedRows = reader.int32();
          continue;
        case 5:
          if (tag === 40) {
            message.allAffectedRows.push(reader.int32());

            continue;
          }

          if (tag === 42) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.allAffectedRows.push(reader.int32());
            }

            continue;
          }

          break;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TaskRunLog_CommandResponse {
    return {
      commandIndex: isSet(object.commandIndex) ? globalThis.Number(object.commandIndex) : 0,
      commandCount: isSet(object.commandCount) ? globalThis.Number(object.commandCount) : 0,
      error: isSet(object.error) ? globalThis.String(object.error) : "",
      affectedRows: isSet(object.affectedRows) ? globalThis.Number(object.affectedRows) : 0,
      allAffectedRows: globalThis.Array.isArray(object?.allAffectedRows)
        ? object.allAffectedRows.map((e: any) => globalThis.Number(e))
        : [],
    };
  },

  toJSON(message: TaskRunLog_CommandResponse): unknown {
    const obj: any = {};
    if (message.commandIndex !== 0) {
      obj.commandIndex = Math.round(message.commandIndex);
    }
    if (message.commandCount !== 0) {
      obj.commandCount = Math.round(message.commandCount);
    }
    if (message.error !== "") {
      obj.error = message.error;
    }
    if (message.affectedRows !== 0) {
      obj.affectedRows = Math.round(message.affectedRows);
    }
    if (message.allAffectedRows?.length) {
      obj.allAffectedRows = message.allAffectedRows.map((e) => Math.round(e));
    }
    return obj;
  },

  create(base?: DeepPartial<TaskRunLog_CommandResponse>): TaskRunLog_CommandResponse {
    return TaskRunLog_CommandResponse.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TaskRunLog_CommandResponse>): TaskRunLog_CommandResponse {
    const message = createBaseTaskRunLog_CommandResponse();
    message.commandIndex = object.commandIndex ?? 0;
    message.commandCount = object.commandCount ?? 0;
    message.error = object.error ?? "";
    message.affectedRows = object.affectedRows ?? 0;
    message.allAffectedRows = object.allAffectedRows?.map((e) => e) || [];
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
