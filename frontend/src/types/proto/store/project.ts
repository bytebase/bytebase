/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export interface Project {
  protectionRules: ProtectionRule[];
}

export interface ProtectionRule {
  /** A unique identifier for a node in UUID format. */
  id: string;
  target: ProtectionRule_Target;
  /** The name of the branch/changelist or wildcard. */
  nameFilter: string;
  /**
   * The roles allowed to create branches or changelists.
   * Format: roles/OWNER.
   */
  createAllowedRoles: string[];
}

/** The type of target. */
export enum ProtectionRule_Target {
  PROTECTION_TARGET_UNSPECIFIED = 0,
  BRANCH = 1,
  CHANGELIST = 2,
  UNRECOGNIZED = -1,
}

export function protectionRule_TargetFromJSON(object: any): ProtectionRule_Target {
  switch (object) {
    case 0:
    case "PROTECTION_TARGET_UNSPECIFIED":
      return ProtectionRule_Target.PROTECTION_TARGET_UNSPECIFIED;
    case 1:
    case "BRANCH":
      return ProtectionRule_Target.BRANCH;
    case 2:
    case "CHANGELIST":
      return ProtectionRule_Target.CHANGELIST;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ProtectionRule_Target.UNRECOGNIZED;
  }
}

export function protectionRule_TargetToJSON(object: ProtectionRule_Target): string {
  switch (object) {
    case ProtectionRule_Target.PROTECTION_TARGET_UNSPECIFIED:
      return "PROTECTION_TARGET_UNSPECIFIED";
    case ProtectionRule_Target.BRANCH:
      return "BRANCH";
    case ProtectionRule_Target.CHANGELIST:
      return "CHANGELIST";
    case ProtectionRule_Target.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

function createBaseProject(): Project {
  return { protectionRules: [] };
}

export const Project = {
  encode(message: Project, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.protectionRules) {
      ProtectionRule.encode(v!, writer.uint32(10).fork()).ldelim();
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
        case 1:
          if (tag !== 10) {
            break;
          }

          message.protectionRules.push(ProtectionRule.decode(reader, reader.uint32()));
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
      protectionRules: Array.isArray(object?.protectionRules)
        ? object.protectionRules.map((e: any) => ProtectionRule.fromJSON(e))
        : [],
    };
  },

  toJSON(message: Project): unknown {
    const obj: any = {};
    if (message.protectionRules) {
      obj.protectionRules = message.protectionRules.map((e) => e ? ProtectionRule.toJSON(e) : undefined);
    } else {
      obj.protectionRules = [];
    }
    return obj;
  },

  create(base?: DeepPartial<Project>): Project {
    return Project.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Project>): Project {
    const message = createBaseProject();
    message.protectionRules = object.protectionRules?.map((e) => ProtectionRule.fromPartial(e)) || [];
    return message;
  },
};

function createBaseProtectionRule(): ProtectionRule {
  return { id: "", target: 0, nameFilter: "", createAllowedRoles: [] };
}

export const ProtectionRule = {
  encode(message: ProtectionRule, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.target !== 0) {
      writer.uint32(16).int32(message.target);
    }
    if (message.nameFilter !== "") {
      writer.uint32(26).string(message.nameFilter);
    }
    for (const v of message.createAllowedRoles) {
      writer.uint32(34).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ProtectionRule {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseProtectionRule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.id = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.target = reader.int32() as any;
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.nameFilter = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.createAllowedRoles.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ProtectionRule {
    return {
      id: isSet(object.id) ? String(object.id) : "",
      target: isSet(object.target) ? protectionRule_TargetFromJSON(object.target) : 0,
      nameFilter: isSet(object.nameFilter) ? String(object.nameFilter) : "",
      createAllowedRoles: Array.isArray(object?.createAllowedRoles)
        ? object.createAllowedRoles.map((e: any) => String(e))
        : [],
    };
  },

  toJSON(message: ProtectionRule): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.target !== undefined && (obj.target = protectionRule_TargetToJSON(message.target));
    message.nameFilter !== undefined && (obj.nameFilter = message.nameFilter);
    if (message.createAllowedRoles) {
      obj.createAllowedRoles = message.createAllowedRoles.map((e) => e);
    } else {
      obj.createAllowedRoles = [];
    }
    return obj;
  },

  create(base?: DeepPartial<ProtectionRule>): ProtectionRule {
    return ProtectionRule.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ProtectionRule>): ProtectionRule {
    const message = createBaseProtectionRule();
    message.id = object.id ?? "";
    message.target = object.target ?? 0;
    message.nameFilter = object.nameFilter ?? "";
    message.createAllowedRoles = object.createAllowedRoles?.map((e) => e) || [];
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Long ? string | number | Long : T extends Array<infer U> ? Array<DeepPartial<U>>
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
