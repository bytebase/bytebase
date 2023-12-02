/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export interface Changelist {
  description: string;
  changes: Changelist_Change[];
}

export interface Changelist_Change {
  /** The name of a sheet. */
  sheet: string;
  /**
   * The source of origin.
   * 1) change history: instances/{instance}/databases/{database}/changeHistories/{changeHistory}.
   * 2) branch: projects/{project}/branches/{branch}.
   * 3) raw SQL if empty.
   */
  source: string;
}

function createBaseChangelist(): Changelist {
  return { description: "", changes: [] };
}

export const Changelist = {
  encode(message: Changelist, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.description !== "") {
      writer.uint32(10).string(message.description);
    }
    for (const v of message.changes) {
      Changelist_Change.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Changelist {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseChangelist();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.description = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.changes.push(Changelist_Change.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Changelist {
    return {
      description: isSet(object.description) ? globalThis.String(object.description) : "",
      changes: globalThis.Array.isArray(object?.changes)
        ? object.changes.map((e: any) => Changelist_Change.fromJSON(e))
        : [],
    };
  },

  toJSON(message: Changelist): unknown {
    const obj: any = {};
    if (message.description !== "") {
      obj.description = message.description;
    }
    if (message.changes?.length) {
      obj.changes = message.changes.map((e) => Changelist_Change.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<Changelist>): Changelist {
    return Changelist.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Changelist>): Changelist {
    const message = createBaseChangelist();
    message.description = object.description ?? "";
    message.changes = object.changes?.map((e) => Changelist_Change.fromPartial(e)) || [];
    return message;
  },
};

function createBaseChangelist_Change(): Changelist_Change {
  return { sheet: "", source: "" };
}

export const Changelist_Change = {
  encode(message: Changelist_Change, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.sheet !== "") {
      writer.uint32(10).string(message.sheet);
    }
    if (message.source !== "") {
      writer.uint32(18).string(message.source);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Changelist_Change {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseChangelist_Change();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.sheet = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.source = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Changelist_Change {
    return {
      sheet: isSet(object.sheet) ? globalThis.String(object.sheet) : "",
      source: isSet(object.source) ? globalThis.String(object.source) : "",
    };
  },

  toJSON(message: Changelist_Change): unknown {
    const obj: any = {};
    if (message.sheet !== "") {
      obj.sheet = message.sheet;
    }
    if (message.source !== "") {
      obj.source = message.source;
    }
    return obj;
  },

  create(base?: DeepPartial<Changelist_Change>): Changelist_Change {
    return Changelist_Change.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Changelist_Change>): Changelist_Change {
    const message = createBaseChangelist_Change();
    message.sheet = object.sheet ?? "";
    message.source = object.source ?? "";
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
