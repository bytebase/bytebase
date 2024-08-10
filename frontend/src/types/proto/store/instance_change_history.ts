/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Range } from "./common";

export const protobufPackage = "bytebase.store";

export interface InstanceChangeHistoryPayload {
  changedResources: ChangedResources | undefined;
}

export interface ChangedResources {
  databases: ChangedResourceDatabase[];
}

export interface ChangedResourceDatabase {
  name: string;
  schemas: ChangedResourceSchema[];
}

export interface ChangedResourceSchema {
  name: string;
  tables: ChangedResourceTable[];
  views: ChangedResourceView[];
  functions: ChangedResourceFunction[];
  procedures: ChangedResourceProcedure[];
}

export interface ChangedResourceTable {
  name: string;
  /** estimated row count of the table */
  tableRows: Long;
  /** The ranges of sub-strings correspond to the statements on the sheet. */
  ranges: Range[];
}

export interface ChangedResourceView {
  name: string;
  /** The ranges of sub-strings correspond to the statements on the sheet. */
  ranges: Range[];
}

export interface ChangedResourceFunction {
  name: string;
  /** The ranges of sub-strings correspond to the statements on the sheet. */
  ranges: Range[];
}

export interface ChangedResourceProcedure {
  name: string;
  /** The ranges of sub-strings correspond to the statements on the sheet. */
  ranges: Range[];
}

function createBaseInstanceChangeHistoryPayload(): InstanceChangeHistoryPayload {
  return { changedResources: undefined };
}

export const InstanceChangeHistoryPayload = {
  encode(message: InstanceChangeHistoryPayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.changedResources !== undefined) {
      ChangedResources.encode(message.changedResources, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): InstanceChangeHistoryPayload {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseInstanceChangeHistoryPayload();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.changedResources = ChangedResources.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): InstanceChangeHistoryPayload {
    return {
      changedResources: isSet(object.changedResources) ? ChangedResources.fromJSON(object.changedResources) : undefined,
    };
  },

  toJSON(message: InstanceChangeHistoryPayload): unknown {
    const obj: any = {};
    if (message.changedResources !== undefined) {
      obj.changedResources = ChangedResources.toJSON(message.changedResources);
    }
    return obj;
  },

  create(base?: DeepPartial<InstanceChangeHistoryPayload>): InstanceChangeHistoryPayload {
    return InstanceChangeHistoryPayload.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<InstanceChangeHistoryPayload>): InstanceChangeHistoryPayload {
    const message = createBaseInstanceChangeHistoryPayload();
    message.changedResources = (object.changedResources !== undefined && object.changedResources !== null)
      ? ChangedResources.fromPartial(object.changedResources)
      : undefined;
    return message;
  },
};

function createBaseChangedResources(): ChangedResources {
  return { databases: [] };
}

export const ChangedResources = {
  encode(message: ChangedResources, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.databases) {
      ChangedResourceDatabase.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ChangedResources {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseChangedResources();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.databases.push(ChangedResourceDatabase.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ChangedResources {
    return {
      databases: globalThis.Array.isArray(object?.databases)
        ? object.databases.map((e: any) => ChangedResourceDatabase.fromJSON(e))
        : [],
    };
  },

  toJSON(message: ChangedResources): unknown {
    const obj: any = {};
    if (message.databases?.length) {
      obj.databases = message.databases.map((e) => ChangedResourceDatabase.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<ChangedResources>): ChangedResources {
    return ChangedResources.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ChangedResources>): ChangedResources {
    const message = createBaseChangedResources();
    message.databases = object.databases?.map((e) => ChangedResourceDatabase.fromPartial(e)) || [];
    return message;
  },
};

function createBaseChangedResourceDatabase(): ChangedResourceDatabase {
  return { name: "", schemas: [] };
}

export const ChangedResourceDatabase = {
  encode(message: ChangedResourceDatabase, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.schemas) {
      ChangedResourceSchema.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ChangedResourceDatabase {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseChangedResourceDatabase();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.schemas.push(ChangedResourceSchema.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ChangedResourceDatabase {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      schemas: globalThis.Array.isArray(object?.schemas)
        ? object.schemas.map((e: any) => ChangedResourceSchema.fromJSON(e))
        : [],
    };
  },

  toJSON(message: ChangedResourceDatabase): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.schemas?.length) {
      obj.schemas = message.schemas.map((e) => ChangedResourceSchema.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<ChangedResourceDatabase>): ChangedResourceDatabase {
    return ChangedResourceDatabase.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ChangedResourceDatabase>): ChangedResourceDatabase {
    const message = createBaseChangedResourceDatabase();
    message.name = object.name ?? "";
    message.schemas = object.schemas?.map((e) => ChangedResourceSchema.fromPartial(e)) || [];
    return message;
  },
};

function createBaseChangedResourceSchema(): ChangedResourceSchema {
  return { name: "", tables: [], views: [], functions: [], procedures: [] };
}

export const ChangedResourceSchema = {
  encode(message: ChangedResourceSchema, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.tables) {
      ChangedResourceTable.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.views) {
      ChangedResourceView.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.functions) {
      ChangedResourceFunction.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.procedures) {
      ChangedResourceProcedure.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ChangedResourceSchema {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseChangedResourceSchema();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.tables.push(ChangedResourceTable.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.views.push(ChangedResourceView.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.functions.push(ChangedResourceFunction.decode(reader, reader.uint32()));
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.procedures.push(ChangedResourceProcedure.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ChangedResourceSchema {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      tables: globalThis.Array.isArray(object?.tables)
        ? object.tables.map((e: any) => ChangedResourceTable.fromJSON(e))
        : [],
      views: globalThis.Array.isArray(object?.views)
        ? object.views.map((e: any) => ChangedResourceView.fromJSON(e))
        : [],
      functions: globalThis.Array.isArray(object?.functions)
        ? object.functions.map((e: any) => ChangedResourceFunction.fromJSON(e))
        : [],
      procedures: globalThis.Array.isArray(object?.procedures)
        ? object.procedures.map((e: any) => ChangedResourceProcedure.fromJSON(e))
        : [],
    };
  },

  toJSON(message: ChangedResourceSchema): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.tables?.length) {
      obj.tables = message.tables.map((e) => ChangedResourceTable.toJSON(e));
    }
    if (message.views?.length) {
      obj.views = message.views.map((e) => ChangedResourceView.toJSON(e));
    }
    if (message.functions?.length) {
      obj.functions = message.functions.map((e) => ChangedResourceFunction.toJSON(e));
    }
    if (message.procedures?.length) {
      obj.procedures = message.procedures.map((e) => ChangedResourceProcedure.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<ChangedResourceSchema>): ChangedResourceSchema {
    return ChangedResourceSchema.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ChangedResourceSchema>): ChangedResourceSchema {
    const message = createBaseChangedResourceSchema();
    message.name = object.name ?? "";
    message.tables = object.tables?.map((e) => ChangedResourceTable.fromPartial(e)) || [];
    message.views = object.views?.map((e) => ChangedResourceView.fromPartial(e)) || [];
    message.functions = object.functions?.map((e) => ChangedResourceFunction.fromPartial(e)) || [];
    message.procedures = object.procedures?.map((e) => ChangedResourceProcedure.fromPartial(e)) || [];
    return message;
  },
};

function createBaseChangedResourceTable(): ChangedResourceTable {
  return { name: "", tableRows: Long.ZERO, ranges: [] };
}

export const ChangedResourceTable = {
  encode(message: ChangedResourceTable, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (!message.tableRows.isZero()) {
      writer.uint32(16).int64(message.tableRows);
    }
    for (const v of message.ranges) {
      Range.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ChangedResourceTable {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseChangedResourceTable();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.tableRows = reader.int64() as Long;
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.ranges.push(Range.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ChangedResourceTable {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      tableRows: isSet(object.tableRows) ? Long.fromValue(object.tableRows) : Long.ZERO,
      ranges: globalThis.Array.isArray(object?.ranges) ? object.ranges.map((e: any) => Range.fromJSON(e)) : [],
    };
  },

  toJSON(message: ChangedResourceTable): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (!message.tableRows.isZero()) {
      obj.tableRows = (message.tableRows || Long.ZERO).toString();
    }
    if (message.ranges?.length) {
      obj.ranges = message.ranges.map((e) => Range.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<ChangedResourceTable>): ChangedResourceTable {
    return ChangedResourceTable.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ChangedResourceTable>): ChangedResourceTable {
    const message = createBaseChangedResourceTable();
    message.name = object.name ?? "";
    message.tableRows = (object.tableRows !== undefined && object.tableRows !== null)
      ? Long.fromValue(object.tableRows)
      : Long.ZERO;
    message.ranges = object.ranges?.map((e) => Range.fromPartial(e)) || [];
    return message;
  },
};

function createBaseChangedResourceView(): ChangedResourceView {
  return { name: "", ranges: [] };
}

export const ChangedResourceView = {
  encode(message: ChangedResourceView, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.ranges) {
      Range.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ChangedResourceView {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseChangedResourceView();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.ranges.push(Range.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ChangedResourceView {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      ranges: globalThis.Array.isArray(object?.ranges) ? object.ranges.map((e: any) => Range.fromJSON(e)) : [],
    };
  },

  toJSON(message: ChangedResourceView): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.ranges?.length) {
      obj.ranges = message.ranges.map((e) => Range.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<ChangedResourceView>): ChangedResourceView {
    return ChangedResourceView.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ChangedResourceView>): ChangedResourceView {
    const message = createBaseChangedResourceView();
    message.name = object.name ?? "";
    message.ranges = object.ranges?.map((e) => Range.fromPartial(e)) || [];
    return message;
  },
};

function createBaseChangedResourceFunction(): ChangedResourceFunction {
  return { name: "", ranges: [] };
}

export const ChangedResourceFunction = {
  encode(message: ChangedResourceFunction, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.ranges) {
      Range.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ChangedResourceFunction {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseChangedResourceFunction();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.ranges.push(Range.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ChangedResourceFunction {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      ranges: globalThis.Array.isArray(object?.ranges) ? object.ranges.map((e: any) => Range.fromJSON(e)) : [],
    };
  },

  toJSON(message: ChangedResourceFunction): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.ranges?.length) {
      obj.ranges = message.ranges.map((e) => Range.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<ChangedResourceFunction>): ChangedResourceFunction {
    return ChangedResourceFunction.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ChangedResourceFunction>): ChangedResourceFunction {
    const message = createBaseChangedResourceFunction();
    message.name = object.name ?? "";
    message.ranges = object.ranges?.map((e) => Range.fromPartial(e)) || [];
    return message;
  },
};

function createBaseChangedResourceProcedure(): ChangedResourceProcedure {
  return { name: "", ranges: [] };
}

export const ChangedResourceProcedure = {
  encode(message: ChangedResourceProcedure, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.ranges) {
      Range.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ChangedResourceProcedure {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseChangedResourceProcedure();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.ranges.push(Range.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ChangedResourceProcedure {
    return {
      name: isSet(object.name) ? globalThis.String(object.name) : "",
      ranges: globalThis.Array.isArray(object?.ranges) ? object.ranges.map((e: any) => Range.fromJSON(e)) : [],
    };
  },

  toJSON(message: ChangedResourceProcedure): unknown {
    const obj: any = {};
    if (message.name !== "") {
      obj.name = message.name;
    }
    if (message.ranges?.length) {
      obj.ranges = message.ranges.map((e) => Range.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<ChangedResourceProcedure>): ChangedResourceProcedure {
    return ChangedResourceProcedure.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ChangedResourceProcedure>): ChangedResourceProcedure {
    const message = createBaseChangedResourceProcedure();
    message.name = object.name ?? "";
    message.ranges = object.ranges?.map((e) => Range.fromPartial(e)) || [];
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
