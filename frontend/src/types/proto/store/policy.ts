/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Expr } from "../google/type/expr";
import { Engine, engineFromJSON, engineToJSON, MaskingLevel, maskingLevelFromJSON, maskingLevelToJSON } from "./common";

export const protobufPackage = "bytebase.store";

export enum SQLReviewRuleLevel {
  LEVEL_UNSPECIFIED = 0,
  ERROR = 1,
  WARNING = 2,
  DISABLED = 3,
  UNRECOGNIZED = -1,
}

export function sQLReviewRuleLevelFromJSON(object: any): SQLReviewRuleLevel {
  switch (object) {
    case 0:
    case "LEVEL_UNSPECIFIED":
      return SQLReviewRuleLevel.LEVEL_UNSPECIFIED;
    case 1:
    case "ERROR":
      return SQLReviewRuleLevel.ERROR;
    case 2:
    case "WARNING":
      return SQLReviewRuleLevel.WARNING;
    case 3:
    case "DISABLED":
      return SQLReviewRuleLevel.DISABLED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return SQLReviewRuleLevel.UNRECOGNIZED;
  }
}

export function sQLReviewRuleLevelToJSON(object: SQLReviewRuleLevel): string {
  switch (object) {
    case SQLReviewRuleLevel.LEVEL_UNSPECIFIED:
      return "LEVEL_UNSPECIFIED";
    case SQLReviewRuleLevel.ERROR:
      return "ERROR";
    case SQLReviewRuleLevel.WARNING:
      return "WARNING";
    case SQLReviewRuleLevel.DISABLED:
      return "DISABLED";
    case SQLReviewRuleLevel.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface RolloutPolicy {
  automatic: boolean;
  workspaceRoles: string[];
  projectRoles: string[];
  /**
   * roles/LAST_APPROVER
   * roles/CREATOR
   */
  issueRoles: string[];
}

export interface IamPolicy {
  /** Collection of binding. */
  bindings: Binding[];
}

/** Reference: https://cloud.google.com/pubsub/docs/reference/rpc/google.iam.v1#binding */
export interface Binding {
  /**
   * Role that is assigned to the list of members.
   * Format: roles/{role}
   */
  role: string;
  /**
   * Specifies the principals requesting access for a Bytebase resource.
   * `members` can have the following values:
   *
   * * `allUsers`: A special identifier that represents anyone.
   * * `user:{emailid}`: An email address that represents a specific Bytebase account. For example, `alice@example.com`.
   */
  members: string[];
  /**
   * The condition that is associated with this binding.
   * If the condition evaluates to true, then this binding applies to the current request.
   * If the condition evaluates to false, then this binding does not apply to the current request. However, a different role binding might grant the same role to one or more of the principals in this binding.
   */
  condition: Expr | undefined;
}

export interface MaskingPolicy {
  maskData: MaskData[];
}

export interface MaskData {
  schema: string;
  table: string;
  column: string;
  maskingLevel: MaskingLevel;
  fullMaskingAlgorithmId: string;
  partialMaskingAlgorithmId: string;
}

/** MaskingExceptionPolicy is the allowlist of users who can access sensitive data. */
export interface MaskingExceptionPolicy {
  maskingExceptions: MaskingExceptionPolicy_MaskingException[];
}

export interface MaskingExceptionPolicy_MaskingException {
  /** action is the action that the user can access sensitive data. */
  action: MaskingExceptionPolicy_MaskingException_Action;
  /** Level is the masking level that the user can access sensitive data. */
  maskingLevel: MaskingLevel;
  /**
   * Member is the principal who bind to this exception policy instance.
   *
   * * `user:{emailid}`: An email address that represents a specific Bytebase account. For example, `alice@example.com`.
   */
  member: string;
  /** The condition that is associated with this exception policy instance. */
  condition: Expr | undefined;
}

export enum MaskingExceptionPolicy_MaskingException_Action {
  ACTION_UNSPECIFIED = 0,
  QUERY = 1,
  EXPORT = 2,
  UNRECOGNIZED = -1,
}

export function maskingExceptionPolicy_MaskingException_ActionFromJSON(
  object: any,
): MaskingExceptionPolicy_MaskingException_Action {
  switch (object) {
    case 0:
    case "ACTION_UNSPECIFIED":
      return MaskingExceptionPolicy_MaskingException_Action.ACTION_UNSPECIFIED;
    case 1:
    case "QUERY":
      return MaskingExceptionPolicy_MaskingException_Action.QUERY;
    case 2:
    case "EXPORT":
      return MaskingExceptionPolicy_MaskingException_Action.EXPORT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return MaskingExceptionPolicy_MaskingException_Action.UNRECOGNIZED;
  }
}

export function maskingExceptionPolicy_MaskingException_ActionToJSON(
  object: MaskingExceptionPolicy_MaskingException_Action,
): string {
  switch (object) {
    case MaskingExceptionPolicy_MaskingException_Action.ACTION_UNSPECIFIED:
      return "ACTION_UNSPECIFIED";
    case MaskingExceptionPolicy_MaskingException_Action.QUERY:
      return "QUERY";
    case MaskingExceptionPolicy_MaskingException_Action.EXPORT:
      return "EXPORT";
    case MaskingExceptionPolicy_MaskingException_Action.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface MaskingRulePolicy {
  rules: MaskingRulePolicy_MaskingRule[];
}

export interface MaskingRulePolicy_MaskingRule {
  /** A unique identifier for a node in UUID format. */
  id: string;
  condition: Expr | undefined;
  maskingLevel: MaskingLevel;
}

export interface SQLReviewPolicy {
  name: string;
  ruleList: SQLReviewRule[];
}

export interface SQLReviewRule {
  type: string;
  level: SQLReviewRuleLevel;
  payload: string;
  engine: Engine;
  comment: string;
}

function createBaseRolloutPolicy(): RolloutPolicy {
  return { automatic: false, workspaceRoles: [], projectRoles: [], issueRoles: [] };
}

export const RolloutPolicy = {
  encode(message: RolloutPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.automatic === true) {
      writer.uint32(8).bool(message.automatic);
    }
    for (const v of message.workspaceRoles) {
      writer.uint32(18).string(v!);
    }
    for (const v of message.projectRoles) {
      writer.uint32(26).string(v!);
    }
    for (const v of message.issueRoles) {
      writer.uint32(34).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RolloutPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRolloutPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.automatic = reader.bool();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.workspaceRoles.push(reader.string());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.projectRoles.push(reader.string());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.issueRoles.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): RolloutPolicy {
    return {
      automatic: isSet(object.automatic) ? Boolean(object.automatic) : false,
      workspaceRoles: Array.isArray(object?.workspaceRoles) ? object.workspaceRoles.map((e: any) => String(e)) : [],
      projectRoles: Array.isArray(object?.projectRoles) ? object.projectRoles.map((e: any) => String(e)) : [],
      issueRoles: Array.isArray(object?.issueRoles) ? object.issueRoles.map((e: any) => String(e)) : [],
    };
  },

  toJSON(message: RolloutPolicy): unknown {
    const obj: any = {};
    message.automatic !== undefined && (obj.automatic = message.automatic);
    if (message.workspaceRoles) {
      obj.workspaceRoles = message.workspaceRoles.map((e) => e);
    } else {
      obj.workspaceRoles = [];
    }
    if (message.projectRoles) {
      obj.projectRoles = message.projectRoles.map((e) => e);
    } else {
      obj.projectRoles = [];
    }
    if (message.issueRoles) {
      obj.issueRoles = message.issueRoles.map((e) => e);
    } else {
      obj.issueRoles = [];
    }
    return obj;
  },

  create(base?: DeepPartial<RolloutPolicy>): RolloutPolicy {
    return RolloutPolicy.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<RolloutPolicy>): RolloutPolicy {
    const message = createBaseRolloutPolicy();
    message.automatic = object.automatic ?? false;
    message.workspaceRoles = object.workspaceRoles?.map((e) => e) || [];
    message.projectRoles = object.projectRoles?.map((e) => e) || [];
    message.issueRoles = object.issueRoles?.map((e) => e) || [];
    return message;
  },
};

function createBaseIamPolicy(): IamPolicy {
  return { bindings: [] };
}

export const IamPolicy = {
  encode(message: IamPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.bindings) {
      Binding.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IamPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIamPolicy();
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

  fromJSON(object: any): IamPolicy {
    return { bindings: Array.isArray(object?.bindings) ? object.bindings.map((e: any) => Binding.fromJSON(e)) : [] };
  },

  toJSON(message: IamPolicy): unknown {
    const obj: any = {};
    if (message.bindings) {
      obj.bindings = message.bindings.map((e) => e ? Binding.toJSON(e) : undefined);
    } else {
      obj.bindings = [];
    }
    return obj;
  },

  create(base?: DeepPartial<IamPolicy>): IamPolicy {
    return IamPolicy.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<IamPolicy>): IamPolicy {
    const message = createBaseIamPolicy();
    message.bindings = object.bindings?.map((e) => Binding.fromPartial(e)) || [];
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
      role: isSet(object.role) ? String(object.role) : "",
      members: Array.isArray(object?.members) ? object.members.map((e: any) => String(e)) : [],
      condition: isSet(object.condition) ? Expr.fromJSON(object.condition) : undefined,
    };
  },

  toJSON(message: Binding): unknown {
    const obj: any = {};
    message.role !== undefined && (obj.role = message.role);
    if (message.members) {
      obj.members = message.members.map((e) => e);
    } else {
      obj.members = [];
    }
    message.condition !== undefined && (obj.condition = message.condition ? Expr.toJSON(message.condition) : undefined);
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

function createBaseMaskingPolicy(): MaskingPolicy {
  return { maskData: [] };
}

export const MaskingPolicy = {
  encode(message: MaskingPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.maskData) {
      MaskData.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MaskingPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMaskingPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.maskData.push(MaskData.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MaskingPolicy {
    return { maskData: Array.isArray(object?.maskData) ? object.maskData.map((e: any) => MaskData.fromJSON(e)) : [] };
  },

  toJSON(message: MaskingPolicy): unknown {
    const obj: any = {};
    if (message.maskData) {
      obj.maskData = message.maskData.map((e) => e ? MaskData.toJSON(e) : undefined);
    } else {
      obj.maskData = [];
    }
    return obj;
  },

  create(base?: DeepPartial<MaskingPolicy>): MaskingPolicy {
    return MaskingPolicy.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<MaskingPolicy>): MaskingPolicy {
    const message = createBaseMaskingPolicy();
    message.maskData = object.maskData?.map((e) => MaskData.fromPartial(e)) || [];
    return message;
  },
};

function createBaseMaskData(): MaskData {
  return {
    schema: "",
    table: "",
    column: "",
    maskingLevel: 0,
    fullMaskingAlgorithmId: "",
    partialMaskingAlgorithmId: "",
  };
}

export const MaskData = {
  encode(message: MaskData, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.schema !== "") {
      writer.uint32(10).string(message.schema);
    }
    if (message.table !== "") {
      writer.uint32(18).string(message.table);
    }
    if (message.column !== "") {
      writer.uint32(26).string(message.column);
    }
    if (message.maskingLevel !== 0) {
      writer.uint32(32).int32(message.maskingLevel);
    }
    if (message.fullMaskingAlgorithmId !== "") {
      writer.uint32(42).string(message.fullMaskingAlgorithmId);
    }
    if (message.partialMaskingAlgorithmId !== "") {
      writer.uint32(50).string(message.partialMaskingAlgorithmId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MaskData {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMaskData();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.schema = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.table = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.column = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.maskingLevel = reader.int32() as any;
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.fullMaskingAlgorithmId = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.partialMaskingAlgorithmId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MaskData {
    return {
      schema: isSet(object.schema) ? String(object.schema) : "",
      table: isSet(object.table) ? String(object.table) : "",
      column: isSet(object.column) ? String(object.column) : "",
      maskingLevel: isSet(object.maskingLevel) ? maskingLevelFromJSON(object.maskingLevel) : 0,
      fullMaskingAlgorithmId: isSet(object.fullMaskingAlgorithmId) ? String(object.fullMaskingAlgorithmId) : "",
      partialMaskingAlgorithmId: isSet(object.partialMaskingAlgorithmId)
        ? String(object.partialMaskingAlgorithmId)
        : "",
    };
  },

  toJSON(message: MaskData): unknown {
    const obj: any = {};
    message.schema !== undefined && (obj.schema = message.schema);
    message.table !== undefined && (obj.table = message.table);
    message.column !== undefined && (obj.column = message.column);
    message.maskingLevel !== undefined && (obj.maskingLevel = maskingLevelToJSON(message.maskingLevel));
    message.fullMaskingAlgorithmId !== undefined && (obj.fullMaskingAlgorithmId = message.fullMaskingAlgorithmId);
    message.partialMaskingAlgorithmId !== undefined &&
      (obj.partialMaskingAlgorithmId = message.partialMaskingAlgorithmId);
    return obj;
  },

  create(base?: DeepPartial<MaskData>): MaskData {
    return MaskData.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<MaskData>): MaskData {
    const message = createBaseMaskData();
    message.schema = object.schema ?? "";
    message.table = object.table ?? "";
    message.column = object.column ?? "";
    message.maskingLevel = object.maskingLevel ?? 0;
    message.fullMaskingAlgorithmId = object.fullMaskingAlgorithmId ?? "";
    message.partialMaskingAlgorithmId = object.partialMaskingAlgorithmId ?? "";
    return message;
  },
};

function createBaseMaskingExceptionPolicy(): MaskingExceptionPolicy {
  return { maskingExceptions: [] };
}

export const MaskingExceptionPolicy = {
  encode(message: MaskingExceptionPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.maskingExceptions) {
      MaskingExceptionPolicy_MaskingException.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MaskingExceptionPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMaskingExceptionPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.maskingExceptions.push(MaskingExceptionPolicy_MaskingException.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MaskingExceptionPolicy {
    return {
      maskingExceptions: Array.isArray(object?.maskingExceptions)
        ? object.maskingExceptions.map((e: any) => MaskingExceptionPolicy_MaskingException.fromJSON(e))
        : [],
    };
  },

  toJSON(message: MaskingExceptionPolicy): unknown {
    const obj: any = {};
    if (message.maskingExceptions) {
      obj.maskingExceptions = message.maskingExceptions.map((e) =>
        e ? MaskingExceptionPolicy_MaskingException.toJSON(e) : undefined
      );
    } else {
      obj.maskingExceptions = [];
    }
    return obj;
  },

  create(base?: DeepPartial<MaskingExceptionPolicy>): MaskingExceptionPolicy {
    return MaskingExceptionPolicy.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<MaskingExceptionPolicy>): MaskingExceptionPolicy {
    const message = createBaseMaskingExceptionPolicy();
    message.maskingExceptions =
      object.maskingExceptions?.map((e) => MaskingExceptionPolicy_MaskingException.fromPartial(e)) || [];
    return message;
  },
};

function createBaseMaskingExceptionPolicy_MaskingException(): MaskingExceptionPolicy_MaskingException {
  return { action: 0, maskingLevel: 0, member: "", condition: undefined };
}

export const MaskingExceptionPolicy_MaskingException = {
  encode(message: MaskingExceptionPolicy_MaskingException, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.action !== 0) {
      writer.uint32(8).int32(message.action);
    }
    if (message.maskingLevel !== 0) {
      writer.uint32(16).int32(message.maskingLevel);
    }
    if (message.member !== "") {
      writer.uint32(34).string(message.member);
    }
    if (message.condition !== undefined) {
      Expr.encode(message.condition, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MaskingExceptionPolicy_MaskingException {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMaskingExceptionPolicy_MaskingException();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.action = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.maskingLevel = reader.int32() as any;
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.member = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
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

  fromJSON(object: any): MaskingExceptionPolicy_MaskingException {
    return {
      action: isSet(object.action) ? maskingExceptionPolicy_MaskingException_ActionFromJSON(object.action) : 0,
      maskingLevel: isSet(object.maskingLevel) ? maskingLevelFromJSON(object.maskingLevel) : 0,
      member: isSet(object.member) ? String(object.member) : "",
      condition: isSet(object.condition) ? Expr.fromJSON(object.condition) : undefined,
    };
  },

  toJSON(message: MaskingExceptionPolicy_MaskingException): unknown {
    const obj: any = {};
    message.action !== undefined && (obj.action = maskingExceptionPolicy_MaskingException_ActionToJSON(message.action));
    message.maskingLevel !== undefined && (obj.maskingLevel = maskingLevelToJSON(message.maskingLevel));
    message.member !== undefined && (obj.member = message.member);
    message.condition !== undefined && (obj.condition = message.condition ? Expr.toJSON(message.condition) : undefined);
    return obj;
  },

  create(base?: DeepPartial<MaskingExceptionPolicy_MaskingException>): MaskingExceptionPolicy_MaskingException {
    return MaskingExceptionPolicy_MaskingException.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<MaskingExceptionPolicy_MaskingException>): MaskingExceptionPolicy_MaskingException {
    const message = createBaseMaskingExceptionPolicy_MaskingException();
    message.action = object.action ?? 0;
    message.maskingLevel = object.maskingLevel ?? 0;
    message.member = object.member ?? "";
    message.condition = (object.condition !== undefined && object.condition !== null)
      ? Expr.fromPartial(object.condition)
      : undefined;
    return message;
  },
};

function createBaseMaskingRulePolicy(): MaskingRulePolicy {
  return { rules: [] };
}

export const MaskingRulePolicy = {
  encode(message: MaskingRulePolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.rules) {
      MaskingRulePolicy_MaskingRule.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MaskingRulePolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMaskingRulePolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.rules.push(MaskingRulePolicy_MaskingRule.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MaskingRulePolicy {
    return {
      rules: Array.isArray(object?.rules)
        ? object.rules.map((e: any) => MaskingRulePolicy_MaskingRule.fromJSON(e))
        : [],
    };
  },

  toJSON(message: MaskingRulePolicy): unknown {
    const obj: any = {};
    if (message.rules) {
      obj.rules = message.rules.map((e) => e ? MaskingRulePolicy_MaskingRule.toJSON(e) : undefined);
    } else {
      obj.rules = [];
    }
    return obj;
  },

  create(base?: DeepPartial<MaskingRulePolicy>): MaskingRulePolicy {
    return MaskingRulePolicy.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<MaskingRulePolicy>): MaskingRulePolicy {
    const message = createBaseMaskingRulePolicy();
    message.rules = object.rules?.map((e) => MaskingRulePolicy_MaskingRule.fromPartial(e)) || [];
    return message;
  },
};

function createBaseMaskingRulePolicy_MaskingRule(): MaskingRulePolicy_MaskingRule {
  return { id: "", condition: undefined, maskingLevel: 0 };
}

export const MaskingRulePolicy_MaskingRule = {
  encode(message: MaskingRulePolicy_MaskingRule, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.condition !== undefined) {
      Expr.encode(message.condition, writer.uint32(18).fork()).ldelim();
    }
    if (message.maskingLevel !== 0) {
      writer.uint32(24).int32(message.maskingLevel);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MaskingRulePolicy_MaskingRule {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMaskingRulePolicy_MaskingRule();
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
          if (tag !== 18) {
            break;
          }

          message.condition = Expr.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.maskingLevel = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MaskingRulePolicy_MaskingRule {
    return {
      id: isSet(object.id) ? String(object.id) : "",
      condition: isSet(object.condition) ? Expr.fromJSON(object.condition) : undefined,
      maskingLevel: isSet(object.maskingLevel) ? maskingLevelFromJSON(object.maskingLevel) : 0,
    };
  },

  toJSON(message: MaskingRulePolicy_MaskingRule): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.condition !== undefined && (obj.condition = message.condition ? Expr.toJSON(message.condition) : undefined);
    message.maskingLevel !== undefined && (obj.maskingLevel = maskingLevelToJSON(message.maskingLevel));
    return obj;
  },

  create(base?: DeepPartial<MaskingRulePolicy_MaskingRule>): MaskingRulePolicy_MaskingRule {
    return MaskingRulePolicy_MaskingRule.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<MaskingRulePolicy_MaskingRule>): MaskingRulePolicy_MaskingRule {
    const message = createBaseMaskingRulePolicy_MaskingRule();
    message.id = object.id ?? "";
    message.condition = (object.condition !== undefined && object.condition !== null)
      ? Expr.fromPartial(object.condition)
      : undefined;
    message.maskingLevel = object.maskingLevel ?? 0;
    return message;
  },
};

function createBaseSQLReviewPolicy(): SQLReviewPolicy {
  return { name: "", ruleList: [] };
}

export const SQLReviewPolicy = {
  encode(message: SQLReviewPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    for (const v of message.ruleList) {
      SQLReviewRule.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SQLReviewPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSQLReviewPolicy();
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

          message.ruleList.push(SQLReviewRule.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SQLReviewPolicy {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      ruleList: Array.isArray(object?.ruleList) ? object.ruleList.map((e: any) => SQLReviewRule.fromJSON(e)) : [],
    };
  },

  toJSON(message: SQLReviewPolicy): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    if (message.ruleList) {
      obj.ruleList = message.ruleList.map((e) => e ? SQLReviewRule.toJSON(e) : undefined);
    } else {
      obj.ruleList = [];
    }
    return obj;
  },

  create(base?: DeepPartial<SQLReviewPolicy>): SQLReviewPolicy {
    return SQLReviewPolicy.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SQLReviewPolicy>): SQLReviewPolicy {
    const message = createBaseSQLReviewPolicy();
    message.name = object.name ?? "";
    message.ruleList = object.ruleList?.map((e) => SQLReviewRule.fromPartial(e)) || [];
    return message;
  },
};

function createBaseSQLReviewRule(): SQLReviewRule {
  return { type: "", level: 0, payload: "", engine: 0, comment: "" };
}

export const SQLReviewRule = {
  encode(message: SQLReviewRule, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.type !== "") {
      writer.uint32(10).string(message.type);
    }
    if (message.level !== 0) {
      writer.uint32(16).int32(message.level);
    }
    if (message.payload !== "") {
      writer.uint32(26).string(message.payload);
    }
    if (message.engine !== 0) {
      writer.uint32(32).int32(message.engine);
    }
    if (message.comment !== "") {
      writer.uint32(42).string(message.comment);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SQLReviewRule {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSQLReviewRule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.type = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.level = reader.int32() as any;
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.payload = reader.string();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.engine = reader.int32() as any;
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.comment = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SQLReviewRule {
    return {
      type: isSet(object.type) ? String(object.type) : "",
      level: isSet(object.level) ? sQLReviewRuleLevelFromJSON(object.level) : 0,
      payload: isSet(object.payload) ? String(object.payload) : "",
      engine: isSet(object.engine) ? engineFromJSON(object.engine) : 0,
      comment: isSet(object.comment) ? String(object.comment) : "",
    };
  },

  toJSON(message: SQLReviewRule): unknown {
    const obj: any = {};
    message.type !== undefined && (obj.type = message.type);
    message.level !== undefined && (obj.level = sQLReviewRuleLevelToJSON(message.level));
    message.payload !== undefined && (obj.payload = message.payload);
    message.engine !== undefined && (obj.engine = engineToJSON(message.engine));
    message.comment !== undefined && (obj.comment = message.comment);
    return obj;
  },

  create(base?: DeepPartial<SQLReviewRule>): SQLReviewRule {
    return SQLReviewRule.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SQLReviewRule>): SQLReviewRule {
    const message = createBaseSQLReviewRule();
    message.type = object.type ?? "";
    message.level = object.level ?? 0;
    message.payload = object.payload ?? "";
    message.engine = object.engine ?? 0;
    message.comment = object.comment ?? "";
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
