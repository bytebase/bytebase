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
  protectionRules: ProtectionRule[];
  issueLabels: Label[];
  /** Force issue labels to be used when creating an issue. */
  forceIssueLabels: boolean;
  /** Allow modifying statement after issue is created. */
  allowModifyStatement: boolean;
  /** Enable auto resolve issue. */
  autoResolveIssue: boolean;
}

export interface ProtectionRule {
  /** A unique identifier for a node in UUID format. */
  id: string;
  target: ProtectionRule_Target;
  /** The name of the branch/changelist or wildcard. */
  nameFilter: string;
  /**
   * The roles allowed to create branches or changelists, rebase branches, delete branches.
   * Format: roles/projectOwner.
   */
  allowedRoles: string[];
  branchSource: ProtectionRule_BranchSource;
}

/** The type of target. */
export enum ProtectionRule_Target {
  PROTECTION_TARGET_UNSPECIFIED = "PROTECTION_TARGET_UNSPECIFIED",
  BRANCH = "BRANCH",
  CHANGELIST = "CHANGELIST",
  UNRECOGNIZED = "UNRECOGNIZED",
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

export function protectionRule_TargetToNumber(object: ProtectionRule_Target): number {
  switch (object) {
    case ProtectionRule_Target.PROTECTION_TARGET_UNSPECIFIED:
      return 0;
    case ProtectionRule_Target.BRANCH:
      return 1;
    case ProtectionRule_Target.CHANGELIST:
      return 2;
    case ProtectionRule_Target.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum ProtectionRule_BranchSource {
  BRANCH_SOURCE_UNSPECIFIED = "BRANCH_SOURCE_UNSPECIFIED",
  DATABASE = "DATABASE",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function protectionRule_BranchSourceFromJSON(object: any): ProtectionRule_BranchSource {
  switch (object) {
    case 0:
    case "BRANCH_SOURCE_UNSPECIFIED":
      return ProtectionRule_BranchSource.BRANCH_SOURCE_UNSPECIFIED;
    case 1:
    case "DATABASE":
      return ProtectionRule_BranchSource.DATABASE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ProtectionRule_BranchSource.UNRECOGNIZED;
  }
}

export function protectionRule_BranchSourceToJSON(object: ProtectionRule_BranchSource): string {
  switch (object) {
    case ProtectionRule_BranchSource.BRANCH_SOURCE_UNSPECIFIED:
      return "BRANCH_SOURCE_UNSPECIFIED";
    case ProtectionRule_BranchSource.DATABASE:
      return "DATABASE";
    case ProtectionRule_BranchSource.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function protectionRule_BranchSourceToNumber(object: ProtectionRule_BranchSource): number {
  switch (object) {
    case ProtectionRule_BranchSource.BRANCH_SOURCE_UNSPECIFIED:
      return 0;
    case ProtectionRule_BranchSource.DATABASE:
      return 1;
    case ProtectionRule_BranchSource.UNRECOGNIZED:
    default:
      return -1;
  }
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
  return {
    protectionRules: [],
    issueLabels: [],
    forceIssueLabels: false,
    allowModifyStatement: false,
    autoResolveIssue: false,
  };
}

export const Project = {
  encode(message: Project, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.protectionRules) {
      ProtectionRule.encode(v!, writer.uint32(10).fork()).ldelim();
    }
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
        case 1:
          if (tag !== 10) {
            break;
          }

          message.protectionRules.push(ProtectionRule.decode(reader, reader.uint32()));
          continue;
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
      protectionRules: globalThis.Array.isArray(object?.protectionRules)
        ? object.protectionRules.map((e: any) => ProtectionRule.fromJSON(e))
        : [],
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
    if (message.protectionRules?.length) {
      obj.protectionRules = message.protectionRules.map((e) => ProtectionRule.toJSON(e));
    }
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
    message.protectionRules = object.protectionRules?.map((e) => ProtectionRule.fromPartial(e)) || [];
    message.issueLabels = object.issueLabels?.map((e) => Label.fromPartial(e)) || [];
    message.forceIssueLabels = object.forceIssueLabels ?? false;
    message.allowModifyStatement = object.allowModifyStatement ?? false;
    message.autoResolveIssue = object.autoResolveIssue ?? false;
    return message;
  },
};

function createBaseProtectionRule(): ProtectionRule {
  return {
    id: "",
    target: ProtectionRule_Target.PROTECTION_TARGET_UNSPECIFIED,
    nameFilter: "",
    allowedRoles: [],
    branchSource: ProtectionRule_BranchSource.BRANCH_SOURCE_UNSPECIFIED,
  };
}

export const ProtectionRule = {
  encode(message: ProtectionRule, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.target !== ProtectionRule_Target.PROTECTION_TARGET_UNSPECIFIED) {
      writer.uint32(16).int32(protectionRule_TargetToNumber(message.target));
    }
    if (message.nameFilter !== "") {
      writer.uint32(26).string(message.nameFilter);
    }
    for (const v of message.allowedRoles) {
      writer.uint32(34).string(v!);
    }
    if (message.branchSource !== ProtectionRule_BranchSource.BRANCH_SOURCE_UNSPECIFIED) {
      writer.uint32(40).int32(protectionRule_BranchSourceToNumber(message.branchSource));
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

          message.target = protectionRule_TargetFromJSON(reader.int32());
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

          message.allowedRoles.push(reader.string());
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.branchSource = protectionRule_BranchSourceFromJSON(reader.int32());
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
      id: isSet(object.id) ? globalThis.String(object.id) : "",
      target: isSet(object.target)
        ? protectionRule_TargetFromJSON(object.target)
        : ProtectionRule_Target.PROTECTION_TARGET_UNSPECIFIED,
      nameFilter: isSet(object.nameFilter) ? globalThis.String(object.nameFilter) : "",
      allowedRoles: globalThis.Array.isArray(object?.allowedRoles)
        ? object.allowedRoles.map((e: any) => globalThis.String(e))
        : [],
      branchSource: isSet(object.branchSource)
        ? protectionRule_BranchSourceFromJSON(object.branchSource)
        : ProtectionRule_BranchSource.BRANCH_SOURCE_UNSPECIFIED,
    };
  },

  toJSON(message: ProtectionRule): unknown {
    const obj: any = {};
    if (message.id !== "") {
      obj.id = message.id;
    }
    if (message.target !== ProtectionRule_Target.PROTECTION_TARGET_UNSPECIFIED) {
      obj.target = protectionRule_TargetToJSON(message.target);
    }
    if (message.nameFilter !== "") {
      obj.nameFilter = message.nameFilter;
    }
    if (message.allowedRoles?.length) {
      obj.allowedRoles = message.allowedRoles;
    }
    if (message.branchSource !== ProtectionRule_BranchSource.BRANCH_SOURCE_UNSPECIFIED) {
      obj.branchSource = protectionRule_BranchSourceToJSON(message.branchSource);
    }
    return obj;
  },

  create(base?: DeepPartial<ProtectionRule>): ProtectionRule {
    return ProtectionRule.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ProtectionRule>): ProtectionRule {
    const message = createBaseProtectionRule();
    message.id = object.id ?? "";
    message.target = object.target ?? ProtectionRule_Target.PROTECTION_TARGET_UNSPECIFIED;
    message.nameFilter = object.nameFilter ?? "";
    message.allowedRoles = object.allowedRoles?.map((e) => e) || [];
    message.branchSource = object.branchSource ?? ProtectionRule_BranchSource.BRANCH_SOURCE_UNSPECIFIED;
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
