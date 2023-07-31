/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { PushEvent } from "./vcs";

export const protobufPackage = "bytebase.store";

export interface InstanceChangeHistoryPayload {
  pushEvent?: PushEvent | undefined;
  changeResources?: ChangeResources | undefined;
}

export interface ChangeResources {
  databases: ChangeResourceDatabase[];
}

export interface ChangeResourceDatabase {
  name: string;
  schemas: ChangeResourceSchema[];
}

export interface ChangeResourceSchema {
  name: string;
  tables: ChangeResourceTable[];
}

export interface ChangeResourceTable {
  name: string;
}

function createBaseInstanceChangeHistoryPayload(): InstanceChangeHistoryPayload {
  return { pushEvent: undefined, changeResources: undefined };
}

export const InstanceChangeHistoryPayload = {
  encode(message: InstanceChangeHistoryPayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pushEvent !== undefined) {
      PushEvent.encode(message.pushEvent, writer.uint32(10).fork()).ldelim();
    }
    if (message.changeResources !== undefined) {
      ChangeResources.encode(message.changeResources, writer.uint32(18).fork()).ldelim();
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

          message.pushEvent = PushEvent.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.changeResources = ChangeResources.decode(reader, reader.uint32());
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
      pushEvent: isSet(object.pushEvent) ? PushEvent.fromJSON(object.pushEvent) : undefined,
      changeResources: isSet(object.changeResources) ? ChangeResources.fromJSON(object.changeResources) : undefined,
    };
  },

  toJSON(message: InstanceChangeHistoryPayload): unknown {
    const obj: any = {};
    message.pushEvent !== undefined &&
      (obj.pushEvent = message.pushEvent ? PushEvent.toJSON(message.pushEvent) : undefined);
    message.changeResources !== undefined &&
      (obj.changeResources = message.changeResources ? ChangeResources.toJSON(message.changeResources) : undefined);
    return obj;
  },

  create(base?: DeepPartial<InstanceChangeHistoryPayload>): InstanceChangeHistoryPayload {
    return InstanceChangeHistoryPayload.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<InstanceChangeHistoryPayload>): InstanceChangeHistoryPayload {
    const message = createBaseInstanceChangeHistoryPayload();
    message.pushEvent = (object.pushEvent !== undefined && object.pushEvent !== null)
      ? PushEvent.fromPartial(object.pushEvent)
      : undefined;
    message.changeResources = (object.changeResources !== undefined && object.changeResources !== null)
      ? ChangeResources.fromPartial(object.changeResources)
      : undefined;
    return message;
  },
};

function createBaseChangeResources(): ChangeResources {
  return { databases: [] };
}

export const ChangeResources = {
  encode(message: ChangeResources, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.databases) {
      ChangeResourceDatabase.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ChangeResources {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseChangeResources();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.databases.push(ChangeResourceDatabase.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ChangeResources {
    return {
      databases: Array.isArray(object?.databases)
        ? object.databases.map((e: any) => ChangeResourceDatabase.fromJSON(e))
        : [],
    };
  },

  toJSON(message: ChangeResources): unknown {
    const obj: any = {};
    if (message.databases) {
      obj.databases = message.databases.map((e) => e ? ChangeResourceDatabase.toJSON(e) : undefined);
    } else {
      obj.databases = [];
    }
    return obj;
  },

  create(base?: DeepPartial<ChangeResources>): ChangeResources {
    return ChangeResources.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ChangeResources>): ChangeResources {
    const message = createBaseChangeResources();
    message.databases = object.databases?.map((e) => ChangeResourceDatabase.fromPartial(e)) || [];
    return message;
  },
};

function createBaseChangeResourceDatabase(): ChangeResourceDatabase {
  return { name: "", schemas: [] };
}

export const ChangeResourceDatabase = {
  encode(message: ChangeResourceDatabase, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.schemas) {
      ChangeResourceSchema.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ChangeResourceDatabase {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseChangeResourceDatabase();
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

          message.schemas.push(ChangeResourceSchema.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ChangeResourceDatabase {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      schemas: Array.isArray(object?.schemas) ? object.schemas.map((e: any) => ChangeResourceSchema.fromJSON(e)) : [],
    };
  },

  toJSON(message: ChangeResourceDatabase): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    if (message.schemas) {
      obj.schemas = message.schemas.map((e) => e ? ChangeResourceSchema.toJSON(e) : undefined);
    } else {
      obj.schemas = [];
    }
    return obj;
  },

  create(base?: DeepPartial<ChangeResourceDatabase>): ChangeResourceDatabase {
    return ChangeResourceDatabase.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ChangeResourceDatabase>): ChangeResourceDatabase {
    const message = createBaseChangeResourceDatabase();
    message.name = object.name ?? "";
    message.schemas = object.schemas?.map((e) => ChangeResourceSchema.fromPartial(e)) || [];
    return message;
  },
};

function createBaseChangeResourceSchema(): ChangeResourceSchema {
  return { name: "", tables: [] };
}

export const ChangeResourceSchema = {
  encode(message: ChangeResourceSchema, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.tables) {
      ChangeResourceTable.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ChangeResourceSchema {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseChangeResourceSchema();
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

          message.tables.push(ChangeResourceTable.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ChangeResourceSchema {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      tables: Array.isArray(object?.tables) ? object.tables.map((e: any) => ChangeResourceTable.fromJSON(e)) : [],
    };
  },

  toJSON(message: ChangeResourceSchema): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    if (message.tables) {
      obj.tables = message.tables.map((e) => e ? ChangeResourceTable.toJSON(e) : undefined);
    } else {
      obj.tables = [];
    }
    return obj;
  },

  create(base?: DeepPartial<ChangeResourceSchema>): ChangeResourceSchema {
    return ChangeResourceSchema.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ChangeResourceSchema>): ChangeResourceSchema {
    const message = createBaseChangeResourceSchema();
    message.name = object.name ?? "";
    message.tables = object.tables?.map((e) => ChangeResourceTable.fromPartial(e)) || [];
    return message;
  },
};

function createBaseChangeResourceTable(): ChangeResourceTable {
  return { name: "" };
}

export const ChangeResourceTable = {
  encode(message: ChangeResourceTable, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ChangeResourceTable {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseChangeResourceTable();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.name = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ChangeResourceTable {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: ChangeResourceTable): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<ChangeResourceTable>): ChangeResourceTable {
    return ChangeResourceTable.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ChangeResourceTable>): ChangeResourceTable {
    const message = createBaseChangeResourceTable();
    message.name = object.name ?? "";
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
