/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Expr } from "../google/type/expr";

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

export interface Binding {
  /**
   * The role that is assigned to the members.
   * Format: roles/{role}
   */
  role: string;
  /**
   * Specifies the principals requesting access for a Bytebase resource.
   * For users, the member should be: users/{userUID}
   * For groups, the member should be: groups/{email}
   */
  members: string[];
  /**
   * The condition that is associated with this binding.
   * If the condition evaluates to true, then this binding applies to the current request.
   * If the condition evaluates to false, then this binding does not apply to the current request. However, a different role binding might grant the same role to one or more of the principals in this binding.
   */
  condition: Expr | undefined;
}

export interface ProjectIamPolicy {
  /**
   * Collection of binding.
   * A binding binds one or more members or groups to a single project role.
   */
  bindings: Binding[];
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

function createBaseBinding(): Binding {
  return { role: "", members: [], condition: undefined };
}

export const Binding = {
  encode(message: Binding, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.role !== "") {
      writer.uint32(10).string(message.role);
    }
    for (const v of message.members) {
      writer.uint32(18).string(v!);
    }
    if (message.condition !== undefined) {
      Expr.encode(message.condition, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Binding {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBinding();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.role = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.members.push(reader.string());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.condition = Expr.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Binding {
    return {
      role: isSet(object.role) ? globalThis.String(object.role) : "",
      members: globalThis.Array.isArray(object?.members) ? object.members.map((e: any) => globalThis.String(e)) : [],
      condition: isSet(object.condition) ? Expr.fromJSON(object.condition) : undefined,
    };
  },

  toJSON(message: Binding): unknown {
    const obj: any = {};
    if (message.role !== "") {
      obj.role = message.role;
    }
    if (message.members?.length) {
      obj.members = message.members;
    }
    if (message.condition !== undefined) {
      obj.condition = Expr.toJSON(message.condition);
    }
    return obj;
  },

  create(base?: DeepPartial<Binding>): Binding {
    return Binding.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Binding>): Binding {
    const message = createBaseBinding();
    message.role = object.role ?? "";
    message.members = object.members?.map((e) => e) || [];
    message.condition = (object.condition !== undefined && object.condition !== null)
      ? Expr.fromPartial(object.condition)
      : undefined;
    return message;
  },
};

function createBaseProjectIamPolicy(): ProjectIamPolicy {
  return { bindings: [] };
}

export const ProjectIamPolicy = {
  encode(message: ProjectIamPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.bindings) {
      Binding.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ProjectIamPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseProjectIamPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.bindings.push(Binding.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ProjectIamPolicy {
    return {
      bindings: globalThis.Array.isArray(object?.bindings) ? object.bindings.map((e: any) => Binding.fromJSON(e)) : [],
    };
  },

  toJSON(message: ProjectIamPolicy): unknown {
    const obj: any = {};
    if (message.bindings?.length) {
      obj.bindings = message.bindings.map((e) => Binding.toJSON(e));
    }
    return obj;
  },

  create(base?: DeepPartial<ProjectIamPolicy>): ProjectIamPolicy {
    return ProjectIamPolicy.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<ProjectIamPolicy>): ProjectIamPolicy {
    const message = createBaseProjectIamPolicy();
    message.bindings = object.bindings?.map((e) => Binding.fromPartial(e)) || [];
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
