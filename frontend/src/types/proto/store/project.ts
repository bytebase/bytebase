/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export interface Label {
  value: string;
  color: string;
  group: string;
}

export interface Project {
  issueLabels: Label[];
  /** Force issue labels to be used when creating an issue. */
  forceIssueLabels: boolean;
  /** Allow modifying statement after issue is created. */
  allowModifyStatement: boolean;
  /** Enable auto resolve issue. */
  autoResolveIssue: boolean;
}

function createBaseLabel(): Label {
  return { value: "", color: "", group: "" };
}

export const Label = {
  encode(message: Label, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.value !== "") {
      writer.uint32(10).string(message.value);
    }
    if (message.color !== "") {
      writer.uint32(18).string(message.color);
    }
    if (message.group !== "") {
      writer.uint32(26).string(message.group);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Label {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseLabel();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.value = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.color = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.group = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Label {
    return {
      value: isSet(object.value) ? globalThis.String(object.value) : "",
      color: isSet(object.color) ? globalThis.String(object.color) : "",
      group: isSet(object.group) ? globalThis.String(object.group) : "",
    };
  },

  toJSON(message: Label): unknown {
    const obj: any = {};
    if (message.value !== "") {
      obj.value = message.value;
    }
    if (message.color !== "") {
      obj.color = message.color;
    }
    if (message.group !== "") {
      obj.group = message.group;
    }
    return obj;
  },

  create(base?: DeepPartial<Label>): Label {
    return Label.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Label>): Label {
    const message = createBaseLabel();
    message.value = object.value ?? "";
    message.color = object.color ?? "";
    message.group = object.group ?? "";
    return message;
  },
};

function createBaseProject(): Project {
  return { issueLabels: [], forceIssueLabels: false, allowModifyStatement: false, autoResolveIssue: false };
}

export const Project = {
  encode(message: Project, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.issueLabels) {
      Label.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.forceIssueLabels === true) {
      writer.uint32(24).bool(message.forceIssueLabels);
    }
    if (message.allowModifyStatement === true) {
      writer.uint32(32).bool(message.allowModifyStatement);
    }
    if (message.autoResolveIssue === true) {
      writer.uint32(40).bool(message.autoResolveIssue);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Project {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseProject();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 2:
          if (tag !== 18) {
            break;
          }

          message.issueLabels.push(Label.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.forceIssueLabels = reader.bool();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.allowModifyStatement = reader.bool();
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.autoResolveIssue = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Project {
    return {
      issueLabels: globalThis.Array.isArray(object?.issueLabels)
        ? object.issueLabels.map((e: any) => Label.fromJSON(e))
        : [],
      forceIssueLabels: isSet(object.forceIssueLabels) ? globalThis.Boolean(object.forceIssueLabels) : false,
      allowModifyStatement: isSet(object.allowModifyStatement)
        ? globalThis.Boolean(object.allowModifyStatement)
        : false,
      autoResolveIssue: isSet(object.autoResolveIssue) ? globalThis.Boolean(object.autoResolveIssue) : false,
    };
  },

  toJSON(message: Project): unknown {
    const obj: any = {};
    if (message.issueLabels?.length) {
      obj.issueLabels = message.issueLabels.map((e) => Label.toJSON(e));
    }
    if (message.forceIssueLabels === true) {
      obj.forceIssueLabels = message.forceIssueLabels;
    }
    if (message.allowModifyStatement === true) {
      obj.allowModifyStatement = message.allowModifyStatement;
    }
    if (message.autoResolveIssue === true) {
      obj.autoResolveIssue = message.autoResolveIssue;
    }
    return obj;
  },

  create(base?: DeepPartial<Project>): Project {
    return Project.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Project>): Project {
    const message = createBaseProject();
    message.issueLabels = object.issueLabels?.map((e) => Label.fromPartial(e)) || [];
    message.forceIssueLabels = object.forceIssueLabels ?? false;
    message.allowModifyStatement = object.allowModifyStatement ?? false;
    message.autoResolveIssue = object.autoResolveIssue ?? false;
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
