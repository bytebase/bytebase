/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { PushEvent } from "./vcs";

export const protobufPackage = "bytebase.store";

export interface InstanceChangeHistoryPayload {
  pushEvent?: PushEvent | undefined;
  changedResources?: ChangedResources | undefined;
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
}

export interface ChangedResourceTable {
  name: string;
}

function createBaseInstanceChangeHistoryPayload(): InstanceChangeHistoryPayload {
  return { pushEvent: undefined, changedResources: undefined };
}

export const InstanceChangeHistoryPayload = {
  encode(message: InstanceChangeHistoryPayload, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pushEvent !== undefined) {
      PushEvent.encode(message.pushEvent, writer.uint32(10).fork()).ldelim();
    }
    if (message.changedResources !== undefined) {
      ChangedResources.encode(message.changedResources, writer.uint32(18).fork()).ldelim();
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
      pushEvent: isSet(object.pushEvent) ? PushEvent.fromJSON(object.pushEvent) : undefined,
      changedResources: isSet(object.changedResources) ? ChangedResources.fromJSON(object.changedResources) : undefined,
    };
  },

  toJSON(message: InstanceChangeHistoryPayload): unknown {
    const obj: any = {};
    message.pushEvent !== undefined &&
      (obj.pushEvent = message.pushEvent ? PushEvent.toJSON(message.pushEvent) : undefined);
    message.changedResources !== undefined &&
      (obj.changedResources = message.changedResources ? ChangedResources.toJSON(message.changedResources) : undefined);
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
      databases: Array.isArray(object?.databases)
        ? object.databases.map((e: any) => ChangedResourceDatabase.fromJSON(e))
        : [],
    };
  },

  toJSON(message: ChangedResources): unknown {
    const obj: any = {};
    if (message.databases) {
      obj.databases = message.databases.map((e) => e ? ChangedResourceDatabase.toJSON(e) : undefined);
    } else {
      obj.databases = [];
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
      name: isSet(object.name) ? String(object.name) : "",
      schemas: Array.isArray(object?.schemas) ? object.schemas.map((e: any) => ChangedResourceSchema.fromJSON(e)) : [],
    };
  },

  toJSON(message: ChangedResourceDatabase): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    if (message.schemas) {
      obj.schemas = message.schemas.map((e) => e ? ChangedResourceSchema.toJSON(e) : undefined);
    } else {
      obj.schemas = [];
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
  return { name: "", tables: [] };
}

export const ChangedResourceSchema = {
  encode(message: ChangedResourceSchema, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.tables) {
      ChangedResourceTable.encode(v!, writer.uint32(18).fork()).ldelim();
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
      name: isSet(object.name) ? String(object.name) : "",
      tables: Array.isArray(object?.tables) ? object.tables.map((e: any) => ChangedResourceTable.fromJSON(e)) : [],
    };
  },

  toJSON(message: ChangedResourceSchema): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    if (message.tables) {
      obj.tables = message.tables.map((e) => e ? ChangedResourceTable.toJSON(e) : undefined);
    } else {
      obj.tables = [];
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
    return message;
  },
};

function createBaseChangedResourceTable(): ChangedResourceTable {
  return { name: "" };
}

export const ChangedResourceTable = {
  encode(message: ChangedResourceTable, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
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
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ChangedResourceTable {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: ChangedResourceTable): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<ChangedResourceTable>): ChangedResourceTable {
    return ChangedResourceTable.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ChangedResourceTable>): ChangedResourceTable {
    const message = createBaseChangedResourceTable();
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
