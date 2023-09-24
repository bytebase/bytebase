/* eslint-disable */
import _m0 from "protobufjs/minimal";
import { Empty } from "../google/protobuf/empty";
import { FieldMask } from "../google/protobuf/field_mask";
import { Timestamp } from "../google/protobuf/timestamp";

export const protobufPackage = "bytebase.v1";

export interface CreateChangelistRequest {
  /**
   * The parent resource where this changelist will be created.
   * Foramt: projects/{project}
   */
  parent: string;
  /** The changelist to create. */
  changelist:
    | Changelist
    | undefined;
  /**
   * The ID to use for the changelist, which will become the final component of
   * the changelist's resource name.
   *
   * This value should be 4-63 characters, and valid characters
   * are /[a-z][0-9]-/.
   */
  changelistId: string;
}

export interface GetChangelistRequest {
  /**
   * The name of the changelist to retrieve.
   * Format: projects/{project}/changelists/{changelist}
   */
  name: string;
}

export interface UpdateChangelistRequest {
  /**
   * The changelist to update.
   *
   * The changelist's `name` field is used to identify the changelist to update.
   * Format: projects/{project}/changelists/{changelist}
   */
  changelist:
    | Changelist
    | undefined;
  /** The list of fields to be updated. */
  updateMask: string[] | undefined;
}

export interface DeleteChangelistRequest {
  /**
   * The name of the changelist to delete.
   * Format: projects/{project}/changelists/{changelist}
   */
  name: string;
}

export interface Changelist {
  /**
   * The name of the changelist resource.
   * Canonical parent is project.
   * Format: projects/{project}/changelists/{changelist}
   */
  name: string;
  description: string;
  /**
   * The creator of the changelist.
   * Format: users/{email}
   */
  creator: string;
  /**
   * The updater of the changelist.
   * Format: users/{email}
   */
  updater: string;
  /** The create time of the changelist. */
  createTime:
    | Date
    | undefined;
  /** The last update time of the changelist. */
  updateTime: Date | undefined;
  changes: Changelist_Change[];
}

export interface Changelist_Change {
  /** The name of a sheet. */
  sheet: string;
  /**
   * The source of origin.
   * 1) change history: instances/{instance}/databases/{database}/changeHistories/{changeHistory}.
   * 2) branch: projects/{project}/schemaDesigns/{schemaDesign}.
   * 3) raw SQL if empty.
   */
  source: string;
}

function createBaseCreateChangelistRequest(): CreateChangelistRequest {
  return { parent: "", changelist: undefined, changelistId: "" };
}

export const CreateChangelistRequest = {
  encode(message: CreateChangelistRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.parent !== "") {
      writer.uint32(10).string(message.parent);
    }
    if (message.changelist !== undefined) {
      Changelist.encode(message.changelist, writer.uint32(18).fork()).ldelim();
    }
    if (message.changelistId !== "") {
      writer.uint32(26).string(message.changelistId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): CreateChangelistRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCreateChangelistRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.parent = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.changelist = Changelist.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.changelistId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): CreateChangelistRequest {
    return {
      parent: isSet(object.parent) ? String(object.parent) : "",
      changelist: isSet(object.changelist) ? Changelist.fromJSON(object.changelist) : undefined,
      changelistId: isSet(object.changelistId) ? String(object.changelistId) : "",
    };
  },

  toJSON(message: CreateChangelistRequest): unknown {
    const obj: any = {};
    message.parent !== undefined && (obj.parent = message.parent);
    message.changelist !== undefined &&
      (obj.changelist = message.changelist ? Changelist.toJSON(message.changelist) : undefined);
    message.changelistId !== undefined && (obj.changelistId = message.changelistId);
    return obj;
  },

  create(base?: DeepPartial<CreateChangelistRequest>): CreateChangelistRequest {
    return CreateChangelistRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<CreateChangelistRequest>): CreateChangelistRequest {
    const message = createBaseCreateChangelistRequest();
    message.parent = object.parent ?? "";
    message.changelist = (object.changelist !== undefined && object.changelist !== null)
      ? Changelist.fromPartial(object.changelist)
      : undefined;
    message.changelistId = object.changelistId ?? "";
    return message;
  },
};

function createBaseGetChangelistRequest(): GetChangelistRequest {
  return { name: "" };
}

export const GetChangelistRequest = {
  encode(message: GetChangelistRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetChangelistRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetChangelistRequest();
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

  fromJSON(object: any): GetChangelistRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetChangelistRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<GetChangelistRequest>): GetChangelistRequest {
    return GetChangelistRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<GetChangelistRequest>): GetChangelistRequest {
    const message = createBaseGetChangelistRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseUpdateChangelistRequest(): UpdateChangelistRequest {
  return { changelist: undefined, updateMask: undefined };
}

export const UpdateChangelistRequest = {
  encode(message: UpdateChangelistRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.changelist !== undefined) {
      Changelist.encode(message.changelist, writer.uint32(10).fork()).ldelim();
    }
    if (message.updateMask !== undefined) {
      FieldMask.encode(FieldMask.wrap(message.updateMask), writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): UpdateChangelistRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpdateChangelistRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.changelist = Changelist.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.updateMask = FieldMask.unwrap(FieldMask.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): UpdateChangelistRequest {
    return {
      changelist: isSet(object.changelist) ? Changelist.fromJSON(object.changelist) : undefined,
      updateMask: isSet(object.updateMask) ? FieldMask.unwrap(FieldMask.fromJSON(object.updateMask)) : undefined,
    };
  },

  toJSON(message: UpdateChangelistRequest): unknown {
    const obj: any = {};
    message.changelist !== undefined &&
      (obj.changelist = message.changelist ? Changelist.toJSON(message.changelist) : undefined);
    message.updateMask !== undefined && (obj.updateMask = FieldMask.toJSON(FieldMask.wrap(message.updateMask)));
    return obj;
  },

  create(base?: DeepPartial<UpdateChangelistRequest>): UpdateChangelistRequest {
    return UpdateChangelistRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<UpdateChangelistRequest>): UpdateChangelistRequest {
    const message = createBaseUpdateChangelistRequest();
    message.changelist = (object.changelist !== undefined && object.changelist !== null)
      ? Changelist.fromPartial(object.changelist)
      : undefined;
    message.updateMask = object.updateMask ?? undefined;
    return message;
  },
};

function createBaseDeleteChangelistRequest(): DeleteChangelistRequest {
  return { name: "" };
}

export const DeleteChangelistRequest = {
  encode(message: DeleteChangelistRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DeleteChangelistRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeleteChangelistRequest();
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

  fromJSON(object: any): DeleteChangelistRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: DeleteChangelistRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<DeleteChangelistRequest>): DeleteChangelistRequest {
    return DeleteChangelistRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<DeleteChangelistRequest>): DeleteChangelistRequest {
    const message = createBaseDeleteChangelistRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseChangelist(): Changelist {
  return {
    name: "",
    description: "",
    creator: "",
    updater: "",
    createTime: undefined,
    updateTime: undefined,
    changes: [],
  };
}

export const Changelist = {
  encode(message: Changelist, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.description !== "") {
      writer.uint32(18).string(message.description);
    }
    if (message.creator !== "") {
      writer.uint32(26).string(message.creator);
    }
    if (message.updater !== "") {
      writer.uint32(34).string(message.updater);
    }
    if (message.createTime !== undefined) {
      Timestamp.encode(toTimestamp(message.createTime), writer.uint32(42).fork()).ldelim();
    }
    if (message.updateTime !== undefined) {
      Timestamp.encode(toTimestamp(message.updateTime), writer.uint32(50).fork()).ldelim();
    }
    for (const v of message.changes) {
      Changelist_Change.encode(v!, writer.uint32(58).fork()).ldelim();
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

          message.name = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.description = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.creator = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.updater = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.createTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.updateTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 7:
          if (tag !== 58) {
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
      name: isSet(object.name) ? String(object.name) : "",
      description: isSet(object.description) ? String(object.description) : "",
      creator: isSet(object.creator) ? String(object.creator) : "",
      updater: isSet(object.updater) ? String(object.updater) : "",
      createTime: isSet(object.createTime) ? fromJsonTimestamp(object.createTime) : undefined,
      updateTime: isSet(object.updateTime) ? fromJsonTimestamp(object.updateTime) : undefined,
      changes: Array.isArray(object?.changes) ? object.changes.map((e: any) => Changelist_Change.fromJSON(e)) : [],
    };
  },

  toJSON(message: Changelist): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.description !== undefined && (obj.description = message.description);
    message.creator !== undefined && (obj.creator = message.creator);
    message.updater !== undefined && (obj.updater = message.updater);
    message.createTime !== undefined && (obj.createTime = message.createTime.toISOString());
    message.updateTime !== undefined && (obj.updateTime = message.updateTime.toISOString());
    if (message.changes) {
      obj.changes = message.changes.map((e) => e ? Changelist_Change.toJSON(e) : undefined);
    } else {
      obj.changes = [];
    }
    return obj;
  },

  create(base?: DeepPartial<Changelist>): Changelist {
    return Changelist.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Changelist>): Changelist {
    const message = createBaseChangelist();
    message.name = object.name ?? "";
    message.description = object.description ?? "";
    message.creator = object.creator ?? "";
    message.updater = object.updater ?? "";
    message.createTime = object.createTime ?? undefined;
    message.updateTime = object.updateTime ?? undefined;
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
      sheet: isSet(object.sheet) ? String(object.sheet) : "",
      source: isSet(object.source) ? String(object.source) : "",
    };
  },

  toJSON(message: Changelist_Change): unknown {
    const obj: any = {};
    message.sheet !== undefined && (obj.sheet = message.sheet);
    message.source !== undefined && (obj.source = message.source);
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

export type ChangelistServiceDefinition = typeof ChangelistServiceDefinition;
export const ChangelistServiceDefinition = {
  name: "ChangelistService",
  fullName: "bytebase.v1.ChangelistService",
  methods: {
    createChangelist: {
      name: "CreateChangelist",
      requestType: CreateChangelistRequest,
      requestStream: false,
      responseType: Changelist,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([17, 112, 97, 114, 101, 110, 116, 44, 99, 104, 97, 110, 103, 101, 108, 105, 115, 116])],
          578365826: [
            new Uint8Array([
              49,
              58,
              10,
              99,
              104,
              97,
              110,
              103,
              101,
              108,
              105,
              115,
              116,
              34,
              35,
              47,
              118,
              49,
              47,
              123,
              112,
              97,
              114,
              101,
              110,
              116,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              125,
              47,
              99,
              104,
              97,
              110,
              103,
              101,
              108,
              105,
              115,
              116,
              115,
            ]),
          ],
        },
      },
    },
    getChangelist: {
      name: "GetChangelist",
      requestType: GetChangelistRequest,
      requestStream: false,
      responseType: Changelist,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              37,
              18,
              35,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              47,
              99,
              104,
              97,
              110,
              103,
              101,
              108,
              105,
              115,
              116,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    updateChangelist: {
      name: "UpdateChangelist",
      requestType: UpdateChangelistRequest,
      requestStream: false,
      responseType: Changelist,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [
            new Uint8Array([
              22,
              99,
              104,
              97,
              110,
              103,
              101,
              108,
              105,
              115,
              116,
              44,
              117,
              112,
              100,
              97,
              116,
              101,
              95,
              109,
              97,
              115,
              107,
            ]),
          ],
          578365826: [
            new Uint8Array([
              60,
              58,
              10,
              99,
              104,
              97,
              110,
              103,
              101,
              108,
              105,
              115,
              116,
              50,
              46,
              47,
              118,
              49,
              47,
              123,
              99,
              104,
              97,
              110,
              103,
              101,
              108,
              105,
              115,
              116,
              46,
              110,
              97,
              109,
              101,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              47,
              99,
              104,
              97,
              110,
              103,
              101,
              108,
              105,
              115,
              116,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    deleteChangelist: {
      name: "DeleteChangelist",
      requestType: DeleteChangelistRequest,
      requestStream: false,
      responseType: Empty,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              37,
              42,
              35,
              47,
              118,
              49,
              47,
              123,
              110,
              97,
              109,
              101,
              61,
              112,
              114,
              111,
              106,
              101,
              99,
              116,
              115,
              47,
              42,
              47,
              99,
              104,
              97,
              110,
              103,
              101,
              108,
              105,
              115,
              116,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
  },
} as const;

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

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
