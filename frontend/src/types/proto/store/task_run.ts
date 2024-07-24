/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Position } from "./common";

export const protobufPackage = "bytebase.store";

export interface TaskRunResult {
  detail: string;
  /** Format: instances/{instance}/databases/{database}/changeHistories/{changeHistory} */
  changeHistory: string;
  version: string;
  startPosition: TaskRunResult_Position | undefined;
  endPosition:
    | TaskRunResult_Position
    | undefined;
  /** The uid of the export archive. */
  exportArchiveUid: number;
  /** The prior backup detail that will be used to rollback the task run. */
  priorBackupDetail: PriorBackupDetail | undefined;
}

/** The following fields are used for error reporting. */
export interface TaskRunResult_Position {
  line: number;
  column: number;
}

export interface PriorBackupDetail {
  items: PriorBackupDetail_Item[];
}

export interface PriorBackupDetail_Item {
  /** The original table information. */
  sourceTable:
    | PriorBackupDetail_Item_Table
    | undefined;
  /** The target backup table information. */
  targetTable: PriorBackupDetail_Item_Table | undefined;
  startPosition: Position | undefined;
  endPosition: Position | undefined;
}

export interface PriorBackupDetail_Item_Table {
  /**
   * The database information.
   * Format: instances/{instance}/databases/{database}
   */
  database: string;
  schema: string;
  table: string;
}

function createBaseTaskRunResult(): TaskRunResult {
  return {
    detail: "",
    changeHistory: "",
    version: "",
    startPosition: undefined,
    endPosition: undefined,
    exportArchiveUid: 0,
    priorBackupDetail: undefined,
  };
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
    if (message.exportArchiveUid !== 0) {
      writer.uint32(48).int32(message.exportArchiveUid);
    }
    if (message.priorBackupDetail !== undefined) {
      PriorBackupDetail.encode(message.priorBackupDetail, writer.uint32(58).fork()).ldelim();
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
        case 6:
          if (tag !== 48) {
            break;
          }

          message.exportArchiveUid = reader.int32();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.priorBackupDetail = PriorBackupDetail.decode(reader, reader.uint32());
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
      exportArchiveUid: isSet(object.exportArchiveUid) ? globalThis.Number(object.exportArchiveUid) : 0,
      priorBackupDetail: isSet(object.priorBackupDetail)
        ? PriorBackupDetail.fromJSON(object.priorBackupDetail)
        : undefined,
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
    if (message.exportArchiveUid !== 0) {
      obj.exportArchiveUid = Math.round(message.exportArchiveUid);
    }
    if (message.priorBackupDetail !== undefined) {
      obj.priorBackupDetail = PriorBackupDetail.toJSON(message.priorBackupDetail);
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
    message.exportArchiveUid = object.exportArchiveUid ?? 0;
    message.priorBackupDetail = (object.priorBackupDetail !== undefined && object.priorBackupDetail !== null)
      ? PriorBackupDetail.fromPartial(object.priorBackupDetail)
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

function createBasePriorBackupDetail(): PriorBackupDetail {
  return { items: [] };
}

export const PriorBackupDetail = {
  encode(message: PriorBackupDetail, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.items) {
      PriorBackupDetail_Item.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PriorBackupDetail {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePriorBackupDetail();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.items.push(PriorBackupDetail_Item.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PriorBackupDetail {
    return {
      items: globalThis.Array.isArray(object?.items)
        ? object.items.map((e: any) => PriorBackupDetail_Item.fromJSON(e))
        : [],
    };
  },

  toJSON(message: PriorBackupDetail): unknown {
    const obj: any = {};
    if (message.items?.length) {
      obj.items = message.items.map((e) => PriorBackupDetail_Item.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<PriorBackupDetail>): PriorBackupDetail {
    return PriorBackupDetail.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<PriorBackupDetail>): PriorBackupDetail {
    const message = createBasePriorBackupDetail();
    message.items = object.items?.map((e) => PriorBackupDetail_Item.fromPartial(e)) || [];
    return message;
  },
};

function createBasePriorBackupDetail_Item(): PriorBackupDetail_Item {
  return { sourceTable: undefined, targetTable: undefined, startPosition: undefined, endPosition: undefined };
}

export const PriorBackupDetail_Item = {
  encode(message: PriorBackupDetail_Item, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sourceTable !== undefined) {
      PriorBackupDetail_Item_Table.encode(message.sourceTable, writer.uint32(10).fork()).ldelim();
    }
    if (message.targetTable !== undefined) {
      PriorBackupDetail_Item_Table.encode(message.targetTable, writer.uint32(18).fork()).ldelim();
    }
    if (message.startPosition !== undefined) {
      Position.encode(message.startPosition, writer.uint32(26).fork()).ldelim();
    }
    if (message.endPosition !== undefined) {
      Position.encode(message.endPosition, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PriorBackupDetail_Item {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePriorBackupDetail_Item();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.sourceTable = PriorBackupDetail_Item_Table.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.targetTable = PriorBackupDetail_Item_Table.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.startPosition = Position.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.endPosition = Position.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PriorBackupDetail_Item {
    return {
      sourceTable: isSet(object.sourceTable) ? PriorBackupDetail_Item_Table.fromJSON(object.sourceTable) : undefined,
      targetTable: isSet(object.targetTable) ? PriorBackupDetail_Item_Table.fromJSON(object.targetTable) : undefined,
      startPosition: isSet(object.startPosition) ? Position.fromJSON(object.startPosition) : undefined,
      endPosition: isSet(object.endPosition) ? Position.fromJSON(object.endPosition) : undefined,
    };
  },

  toJSON(message: PriorBackupDetail_Item): unknown {
    const obj: any = {};
    if (message.sourceTable !== undefined) {
      obj.sourceTable = PriorBackupDetail_Item_Table.toJSON(message.sourceTable);
    }
    if (message.targetTable !== undefined) {
      obj.targetTable = PriorBackupDetail_Item_Table.toJSON(message.targetTable);
    }
    if (message.startPosition !== undefined) {
      obj.startPosition = Position.toJSON(message.startPosition);
    }
    if (message.endPosition !== undefined) {
      obj.endPosition = Position.toJSON(message.endPosition);
    }
    return obj;
  },

  create(base?: DeepPartial<PriorBackupDetail_Item>): PriorBackupDetail_Item {
    return PriorBackupDetail_Item.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<PriorBackupDetail_Item>): PriorBackupDetail_Item {
    const message = createBasePriorBackupDetail_Item();
    message.sourceTable = (object.sourceTable !== undefined && object.sourceTable !== null)
      ? PriorBackupDetail_Item_Table.fromPartial(object.sourceTable)
      : undefined;
    message.targetTable = (object.targetTable !== undefined && object.targetTable !== null)
      ? PriorBackupDetail_Item_Table.fromPartial(object.targetTable)
      : undefined;
    message.startPosition = (object.startPosition !== undefined && object.startPosition !== null)
      ? Position.fromPartial(object.startPosition)
      : undefined;
    message.endPosition = (object.endPosition !== undefined && object.endPosition !== null)
      ? Position.fromPartial(object.endPosition)
      : undefined;
    return message;
  },
};

function createBasePriorBackupDetail_Item_Table(): PriorBackupDetail_Item_Table {
  return { database: "", schema: "", table: "" };
}

export const PriorBackupDetail_Item_Table = {
  encode(message: PriorBackupDetail_Item_Table, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.database !== "") {
      writer.uint32(10).string(message.database);
    }
    if (message.schema !== "") {
      writer.uint32(18).string(message.schema);
    }
    if (message.table !== "") {
      writer.uint32(26).string(message.table);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PriorBackupDetail_Item_Table {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePriorBackupDetail_Item_Table();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.database = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.schema = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.table = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PriorBackupDetail_Item_Table {
    return {
      database: isSet(object.database) ? globalThis.String(object.database) : "",
      schema: isSet(object.schema) ? globalThis.String(object.schema) : "",
      table: isSet(object.table) ? globalThis.String(object.table) : "",
    };
  },

  toJSON(message: PriorBackupDetail_Item_Table): unknown {
    const obj: any = {};
    if (message.database !== "") {
      obj.database = message.database;
    }
    if (message.schema !== "") {
      obj.schema = message.schema;
    }
    if (message.table !== "") {
      obj.table = message.table;
    }
    return obj;
  },

  create(base?: DeepPartial<PriorBackupDetail_Item_Table>): PriorBackupDetail_Item_Table {
    return PriorBackupDetail_Item_Table.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<PriorBackupDetail_Item_Table>): PriorBackupDetail_Item_Table {
    const message = createBasePriorBackupDetail_Item_Table();
    message.database = object.database ?? "";
    message.schema = object.schema ?? "";
    message.table = object.table ?? "";
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
