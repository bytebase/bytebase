/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";
import { Expr } from "../google/type/expr";
import {
  Engine,
  engineFromJSON,
  engineToJSON,
  engineToNumber,
  MaskingLevel,
  maskingLevelFromJSON,
  maskingLevelToJSON,
  maskingLevelToNumber,
} from "./common";

export const protobufPackage = "bytebase.store";

export enum SQLReviewRuleLevel {
  LEVEL_UNSPECIFIED = "LEVEL_UNSPECIFIED",
  ERROR = "ERROR",
  WARNING = "WARNING",
  DISABLED = "DISABLED",
  UNRECOGNIZED = "UNRECOGNIZED",
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

export function sQLReviewRuleLevelToNumber(object: SQLReviewRuleLevel): number {
  switch (object) {
    case SQLReviewRuleLevel.LEVEL_UNSPECIFIED:
      return 0;
    case SQLReviewRuleLevel.ERROR:
      return 1;
    case SQLReviewRuleLevel.WARNING:
      return 2;
    case SQLReviewRuleLevel.DISABLED:
      return 3;
    case SQLReviewRuleLevel.UNRECOGNIZED:
    default:
      return -1;
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
   * Format: users/{userUID} or groups/{group email}
   */
  member: string;
  /** The condition that is associated with this exception policy instance. */
  condition: Expr | undefined;
}

export enum MaskingExceptionPolicy_MaskingException_Action {
  ACTION_UNSPECIFIED = "ACTION_UNSPECIFIED",
  QUERY = "QUERY",
  EXPORT = "EXPORT",
  UNRECOGNIZED = "UNRECOGNIZED",
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

export function maskingExceptionPolicy_MaskingException_ActionToNumber(
  object: MaskingExceptionPolicy_MaskingException_Action,
): number {
  switch (object) {
    case MaskingExceptionPolicy_MaskingException_Action.ACTION_UNSPECIFIED:
      return 0;
    case MaskingExceptionPolicy_MaskingException_Action.QUERY:
      return 1;
    case MaskingExceptionPolicy_MaskingException_Action.EXPORT:
      return 2;
    case MaskingExceptionPolicy_MaskingException_Action.UNRECOGNIZED:
    default:
      return -1;
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

export interface SQLReviewRule {
  type: string;
  level: SQLReviewRuleLevel;
  payload: string;
  engine: Engine;
  comment: string;
}

export interface TagPolicy {
  /**
   * tags is the key - value map for resources.
   * for example, the environment resource can have the sql review config tag, like "bb.tag.review_config": "reviewConfigs/{review config resource id}"
   */
  tags: { [key: string]: string };
}

export interface TagPolicy_TagsEntry {
  key: string;
  value: string;
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

export interface IamPolicy {
  /**
   * Collection of binding.
   * A binding binds one or more members or groups to a single role.
   */
  bindings: Binding[];
}

/** EnvironmentTierPolicy is the tier of an environment. */
export interface EnvironmentTierPolicy {
  environmentTier: EnvironmentTierPolicy_EnvironmentTier;
}

export enum EnvironmentTierPolicy_EnvironmentTier {
  ENVIRONMENT_TIER_UNSPECIFIED = "ENVIRONMENT_TIER_UNSPECIFIED",
  PROTECTED = "PROTECTED",
  UNPROTECTED = "UNPROTECTED",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function environmentTierPolicy_EnvironmentTierFromJSON(object: any): EnvironmentTierPolicy_EnvironmentTier {
  switch (object) {
    case 0:
    case "ENVIRONMENT_TIER_UNSPECIFIED":
      return EnvironmentTierPolicy_EnvironmentTier.ENVIRONMENT_TIER_UNSPECIFIED;
    case 1:
    case "PROTECTED":
      return EnvironmentTierPolicy_EnvironmentTier.PROTECTED;
    case 2:
    case "UNPROTECTED":
      return EnvironmentTierPolicy_EnvironmentTier.UNPROTECTED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return EnvironmentTierPolicy_EnvironmentTier.UNRECOGNIZED;
  }
}

export function environmentTierPolicy_EnvironmentTierToJSON(object: EnvironmentTierPolicy_EnvironmentTier): string {
  switch (object) {
    case EnvironmentTierPolicy_EnvironmentTier.ENVIRONMENT_TIER_UNSPECIFIED:
      return "ENVIRONMENT_TIER_UNSPECIFIED";
    case EnvironmentTierPolicy_EnvironmentTier.PROTECTED:
      return "PROTECTED";
    case EnvironmentTierPolicy_EnvironmentTier.UNPROTECTED:
      return "UNPROTECTED";
    case EnvironmentTierPolicy_EnvironmentTier.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function environmentTierPolicy_EnvironmentTierToNumber(object: EnvironmentTierPolicy_EnvironmentTier): number {
  switch (object) {
    case EnvironmentTierPolicy_EnvironmentTier.ENVIRONMENT_TIER_UNSPECIFIED:
      return 0;
    case EnvironmentTierPolicy_EnvironmentTier.PROTECTED:
      return 1;
    case EnvironmentTierPolicy_EnvironmentTier.UNPROTECTED:
      return 2;
    case EnvironmentTierPolicy_EnvironmentTier.UNRECOGNIZED:
    default:
      return -1;
  }
}

/** SlowQueryPolicy is the policy configuration for slow query. */
export interface SlowQueryPolicy {
  active: boolean;
}

/** DisableCopyDataPolicy is the policy configuration for disabling copying data. */
export interface DisableCopyDataPolicy {
  active: boolean;
}

/** RestrictIssueCreationForSQLReviewPolicy is the policy configuration for restricting issue creation for SQL review. */
export interface RestrictIssueCreationForSQLReviewPolicy {
  disallow: boolean;
}

/** DataSourceQueryPolicy is the policy configuration for data source query. */
export interface DataSourceQueryPolicy {
  adminDataSourceRestriction: DataSourceQueryPolicy_Restricton;
}

export enum DataSourceQueryPolicy_Restricton {
  RESTRICTION_UNSPECIFIED = "RESTRICTION_UNSPECIFIED",
  /** FALLBACK - Allow to query admin data sources when there is no read-only data source. */
  FALLBACK = "FALLBACK",
  /** DISALLOW - Disallow to query admin data sources. */
  DISALLOW = "DISALLOW",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function dataSourceQueryPolicy_RestrictonFromJSON(object: any): DataSourceQueryPolicy_Restricton {
  switch (object) {
    case 0:
    case "RESTRICTION_UNSPECIFIED":
      return DataSourceQueryPolicy_Restricton.RESTRICTION_UNSPECIFIED;
    case 1:
    case "FALLBACK":
      return DataSourceQueryPolicy_Restricton.FALLBACK;
    case 2:
    case "DISALLOW":
      return DataSourceQueryPolicy_Restricton.DISALLOW;
    case -1:
    case "UNRECOGNIZED":
    default:
      return DataSourceQueryPolicy_Restricton.UNRECOGNIZED;
  }
}

export function dataSourceQueryPolicy_RestrictonToJSON(object: DataSourceQueryPolicy_Restricton): string {
  switch (object) {
    case DataSourceQueryPolicy_Restricton.RESTRICTION_UNSPECIFIED:
      return "RESTRICTION_UNSPECIFIED";
    case DataSourceQueryPolicy_Restricton.FALLBACK:
      return "FALLBACK";
    case DataSourceQueryPolicy_Restricton.DISALLOW:
      return "DISALLOW";
    case DataSourceQueryPolicy_Restricton.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function dataSourceQueryPolicy_RestrictonToNumber(object: DataSourceQueryPolicy_Restricton): number {
  switch (object) {
    case DataSourceQueryPolicy_Restricton.RESTRICTION_UNSPECIFIED:
      return 0;
    case DataSourceQueryPolicy_Restricton.FALLBACK:
      return 1;
    case DataSourceQueryPolicy_Restricton.DISALLOW:
      return 2;
    case DataSourceQueryPolicy_Restricton.UNRECOGNIZED:
    default:
      return -1;
  }
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
      automatic: isSet(object.automatic) ? globalThis.Boolean(object.automatic) : false,
      workspaceRoles: globalThis.Array.isArray(object?.workspaceRoles)
        ? object.workspaceRoles.map((e: any) => globalThis.String(e))
        : [],
      projectRoles: globalThis.Array.isArray(object?.projectRoles)
        ? object.projectRoles.map((e: any) => globalThis.String(e))
        : [],
      issueRoles: globalThis.Array.isArray(object?.issueRoles)
        ? object.issueRoles.map((e: any) => globalThis.String(e))
        : [],
    };
  },

  toJSON(message: RolloutPolicy): unknown {
    const obj: any = {};
    if (message.automatic === true) {
      obj.automatic = message.automatic;
    }
    if (message.workspaceRoles?.length) {
      obj.workspaceRoles = message.workspaceRoles;
    }
    if (message.projectRoles?.length) {
      obj.projectRoles = message.projectRoles;
    }
    if (message.issueRoles?.length) {
      obj.issueRoles = message.issueRoles;
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
    return {
      maskData: globalThis.Array.isArray(object?.maskData) ? object.maskData.map((e: any) => MaskData.fromJSON(e)) : [],
    };
  },

  toJSON(message: MaskingPolicy): unknown {
    const obj: any = {};
    if (message.maskData?.length) {
      obj.maskData = message.maskData.map((e) => MaskData.toJSON(e));
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
    maskingLevel: MaskingLevel.MASKING_LEVEL_UNSPECIFIED,
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
    if (message.maskingLevel !== MaskingLevel.MASKING_LEVEL_UNSPECIFIED) {
      writer.uint32(32).int32(maskingLevelToNumber(message.maskingLevel));
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

          message.maskingLevel = maskingLevelFromJSON(reader.int32());
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
      schema: isSet(object.schema) ? globalThis.String(object.schema) : "",
      table: isSet(object.table) ? globalThis.String(object.table) : "",
      column: isSet(object.column) ? globalThis.String(object.column) : "",
      maskingLevel: isSet(object.maskingLevel)
        ? maskingLevelFromJSON(object.maskingLevel)
        : MaskingLevel.MASKING_LEVEL_UNSPECIFIED,
      fullMaskingAlgorithmId: isSet(object.fullMaskingAlgorithmId)
        ? globalThis.String(object.fullMaskingAlgorithmId)
        : "",
      partialMaskingAlgorithmId: isSet(object.partialMaskingAlgorithmId)
        ? globalThis.String(object.partialMaskingAlgorithmId)
        : "",
    };
  },

  toJSON(message: MaskData): unknown {
    const obj: any = {};
    if (message.schema !== "") {
      obj.schema = message.schema;
    }
    if (message.table !== "") {
      obj.table = message.table;
    }
    if (message.column !== "") {
      obj.column = message.column;
    }
    if (message.maskingLevel !== MaskingLevel.MASKING_LEVEL_UNSPECIFIED) {
      obj.maskingLevel = maskingLevelToJSON(message.maskingLevel);
    }
    if (message.fullMaskingAlgorithmId !== "") {
      obj.fullMaskingAlgorithmId = message.fullMaskingAlgorithmId;
    }
    if (message.partialMaskingAlgorithmId !== "") {
      obj.partialMaskingAlgorithmId = message.partialMaskingAlgorithmId;
    }
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
    message.maskingLevel = object.maskingLevel ?? MaskingLevel.MASKING_LEVEL_UNSPECIFIED;
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
      maskingExceptions: globalThis.Array.isArray(object?.maskingExceptions)
        ? object.maskingExceptions.map((e: any) => MaskingExceptionPolicy_MaskingException.fromJSON(e))
        : [],
    };
  },

  toJSON(message: MaskingExceptionPolicy): unknown {
    const obj: any = {};
    if (message.maskingExceptions?.length) {
      obj.maskingExceptions = message.maskingExceptions.map((e) => MaskingExceptionPolicy_MaskingException.toJSON(e));
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
  return {
    action: MaskingExceptionPolicy_MaskingException_Action.ACTION_UNSPECIFIED,
    maskingLevel: MaskingLevel.MASKING_LEVEL_UNSPECIFIED,
    member: "",
    condition: undefined,
  };
}

export const MaskingExceptionPolicy_MaskingException = {
  encode(message: MaskingExceptionPolicy_MaskingException, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.action !== MaskingExceptionPolicy_MaskingException_Action.ACTION_UNSPECIFIED) {
      writer.uint32(8).int32(maskingExceptionPolicy_MaskingException_ActionToNumber(message.action));
    }
    if (message.maskingLevel !== MaskingLevel.MASKING_LEVEL_UNSPECIFIED) {
      writer.uint32(16).int32(maskingLevelToNumber(message.maskingLevel));
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

          message.action = maskingExceptionPolicy_MaskingException_ActionFromJSON(reader.int32());
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.maskingLevel = maskingLevelFromJSON(reader.int32());
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
      action: isSet(object.action)
        ? maskingExceptionPolicy_MaskingException_ActionFromJSON(object.action)
        : MaskingExceptionPolicy_MaskingException_Action.ACTION_UNSPECIFIED,
      maskingLevel: isSet(object.maskingLevel)
        ? maskingLevelFromJSON(object.maskingLevel)
        : MaskingLevel.MASKING_LEVEL_UNSPECIFIED,
      member: isSet(object.member) ? globalThis.String(object.member) : "",
      condition: isSet(object.condition) ? Expr.fromJSON(object.condition) : undefined,
    };
  },

  toJSON(message: MaskingExceptionPolicy_MaskingException): unknown {
    const obj: any = {};
    if (message.action !== MaskingExceptionPolicy_MaskingException_Action.ACTION_UNSPECIFIED) {
      obj.action = maskingExceptionPolicy_MaskingException_ActionToJSON(message.action);
    }
    if (message.maskingLevel !== MaskingLevel.MASKING_LEVEL_UNSPECIFIED) {
      obj.maskingLevel = maskingLevelToJSON(message.maskingLevel);
    }
    if (message.member !== "") {
      obj.member = message.member;
    }
    if (message.condition !== undefined) {
      obj.condition = Expr.toJSON(message.condition);
    }
    return obj;
  },

  create(base?: DeepPartial<MaskingExceptionPolicy_MaskingException>): MaskingExceptionPolicy_MaskingException {
    return MaskingExceptionPolicy_MaskingException.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<MaskingExceptionPolicy_MaskingException>): MaskingExceptionPolicy_MaskingException {
    const message = createBaseMaskingExceptionPolicy_MaskingException();
    message.action = object.action ?? MaskingExceptionPolicy_MaskingException_Action.ACTION_UNSPECIFIED;
    message.maskingLevel = object.maskingLevel ?? MaskingLevel.MASKING_LEVEL_UNSPECIFIED;
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
      rules: globalThis.Array.isArray(object?.rules)
        ? object.rules.map((e: any) => MaskingRulePolicy_MaskingRule.fromJSON(e))
        : [],
    };
  },

  toJSON(message: MaskingRulePolicy): unknown {
    const obj: any = {};
    if (message.rules?.length) {
      obj.rules = message.rules.map((e) => MaskingRulePolicy_MaskingRule.toJSON(e));
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
  return { id: "", condition: undefined, maskingLevel: MaskingLevel.MASKING_LEVEL_UNSPECIFIED };
}

export const MaskingRulePolicy_MaskingRule = {
  encode(message: MaskingRulePolicy_MaskingRule, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.condition !== undefined) {
      Expr.encode(message.condition, writer.uint32(18).fork()).ldelim();
    }
    if (message.maskingLevel !== MaskingLevel.MASKING_LEVEL_UNSPECIFIED) {
      writer.uint32(24).int32(maskingLevelToNumber(message.maskingLevel));
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

          message.maskingLevel = maskingLevelFromJSON(reader.int32());
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
      id: isSet(object.id) ? globalThis.String(object.id) : "",
      condition: isSet(object.condition) ? Expr.fromJSON(object.condition) : undefined,
      maskingLevel: isSet(object.maskingLevel)
        ? maskingLevelFromJSON(object.maskingLevel)
        : MaskingLevel.MASKING_LEVEL_UNSPECIFIED,
    };
  },

  toJSON(message: MaskingRulePolicy_MaskingRule): unknown {
    const obj: any = {};
    if (message.id !== "") {
      obj.id = message.id;
    }
    if (message.condition !== undefined) {
      obj.condition = Expr.toJSON(message.condition);
    }
    if (message.maskingLevel !== MaskingLevel.MASKING_LEVEL_UNSPECIFIED) {
      obj.maskingLevel = maskingLevelToJSON(message.maskingLevel);
    }
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
    message.maskingLevel = object.maskingLevel ?? MaskingLevel.MASKING_LEVEL_UNSPECIFIED;
    return message;
  },
};

function createBaseSQLReviewRule(): SQLReviewRule {
  return {
    type: "",
    level: SQLReviewRuleLevel.LEVEL_UNSPECIFIED,
    payload: "",
    engine: Engine.ENGINE_UNSPECIFIED,
    comment: "",
  };
}

export const SQLReviewRule = {
  encode(message: SQLReviewRule, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.type !== "") {
      writer.uint32(10).string(message.type);
    }
    if (message.level !== SQLReviewRuleLevel.LEVEL_UNSPECIFIED) {
      writer.uint32(16).int32(sQLReviewRuleLevelToNumber(message.level));
    }
    if (message.payload !== "") {
      writer.uint32(26).string(message.payload);
    }
    if (message.engine !== Engine.ENGINE_UNSPECIFIED) {
      writer.uint32(32).int32(engineToNumber(message.engine));
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

          message.level = sQLReviewRuleLevelFromJSON(reader.int32());
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

          message.engine = engineFromJSON(reader.int32());
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
      type: isSet(object.type) ? globalThis.String(object.type) : "",
      level: isSet(object.level) ? sQLReviewRuleLevelFromJSON(object.level) : SQLReviewRuleLevel.LEVEL_UNSPECIFIED,
      payload: isSet(object.payload) ? globalThis.String(object.payload) : "",
      engine: isSet(object.engine) ? engineFromJSON(object.engine) : Engine.ENGINE_UNSPECIFIED,
      comment: isSet(object.comment) ? globalThis.String(object.comment) : "",
    };
  },

  toJSON(message: SQLReviewRule): unknown {
    const obj: any = {};
    if (message.type !== "") {
      obj.type = message.type;
    }
    if (message.level !== SQLReviewRuleLevel.LEVEL_UNSPECIFIED) {
      obj.level = sQLReviewRuleLevelToJSON(message.level);
    }
    if (message.payload !== "") {
      obj.payload = message.payload;
    }
    if (message.engine !== Engine.ENGINE_UNSPECIFIED) {
      obj.engine = engineToJSON(message.engine);
    }
    if (message.comment !== "") {
      obj.comment = message.comment;
    }
    return obj;
  },

  create(base?: DeepPartial<SQLReviewRule>): SQLReviewRule {
    return SQLReviewRule.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SQLReviewRule>): SQLReviewRule {
    const message = createBaseSQLReviewRule();
    message.type = object.type ?? "";
    message.level = object.level ?? SQLReviewRuleLevel.LEVEL_UNSPECIFIED;
    message.payload = object.payload ?? "";
    message.engine = object.engine ?? Engine.ENGINE_UNSPECIFIED;
    message.comment = object.comment ?? "";
    return message;
  },
};

function createBaseTagPolicy(): TagPolicy {
  return { tags: {} };
}

export const TagPolicy = {
  encode(message: TagPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    Object.entries(message.tags).forEach(([key, value]) => {
      TagPolicy_TagsEntry.encode({ key: key as any, value }, writer.uint32(10).fork()).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TagPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTagPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          const entry1 = TagPolicy_TagsEntry.decode(reader, reader.uint32());
          if (entry1.value !== undefined) {
            message.tags[entry1.key] = entry1.value;
          }
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TagPolicy {
    return {
      tags: isObject(object.tags)
        ? Object.entries(object.tags).reduce<{ [key: string]: string }>((acc, [key, value]) => {
          acc[key] = String(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: TagPolicy): unknown {
    const obj: any = {};
    if (message.tags) {
      const entries = Object.entries(message.tags);
      if (entries.length > 0) {
        obj.tags = {};
        entries.forEach(([k, v]) => {
          obj.tags[k] = v;
        });
      }
    }
    return obj;
  },

  create(base?: DeepPartial<TagPolicy>): TagPolicy {
    return TagPolicy.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TagPolicy>): TagPolicy {
    const message = createBaseTagPolicy();
    message.tags = Object.entries(object.tags ?? {}).reduce<{ [key: string]: string }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = globalThis.String(value);
      }
      return acc;
    }, {});
    return message;
  },
};

function createBaseTagPolicy_TagsEntry(): TagPolicy_TagsEntry {
  return { key: "", value: "" };
}

export const TagPolicy_TagsEntry = {
  encode(message: TagPolicy_TagsEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== "") {
      writer.uint32(18).string(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): TagPolicy_TagsEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTagPolicy_TagsEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.key = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.value = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): TagPolicy_TagsEntry {
    return {
      key: isSet(object.key) ? globalThis.String(object.key) : "",
      value: isSet(object.value) ? globalThis.String(object.value) : "",
    };
  },

  toJSON(message: TagPolicy_TagsEntry): unknown {
    const obj: any = {};
    if (message.key !== "") {
      obj.key = message.key;
    }
    if (message.value !== "") {
      obj.value = message.value;
    }
    return obj;
  },

  create(base?: DeepPartial<TagPolicy_TagsEntry>): TagPolicy_TagsEntry {
    return TagPolicy_TagsEntry.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<TagPolicy_TagsEntry>): TagPolicy_TagsEntry {
    const message = createBaseTagPolicy_TagsEntry();
    message.key = object.key ?? "";
    message.value = object.value ?? "";
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
    return {
      bindings: globalThis.Array.isArray(object?.bindings) ? object.bindings.map((e: any) => Binding.fromJSON(e)) : [],
    };
  },

  toJSON(message: IamPolicy): unknown {
    const obj: any = {};
    if (message.bindings?.length) {
      obj.bindings = message.bindings.map((e) => Binding.toJSON(e));
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

function createBaseEnvironmentTierPolicy(): EnvironmentTierPolicy {
  return { environmentTier: EnvironmentTierPolicy_EnvironmentTier.ENVIRONMENT_TIER_UNSPECIFIED };
}

export const EnvironmentTierPolicy = {
  encode(message: EnvironmentTierPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.environmentTier !== EnvironmentTierPolicy_EnvironmentTier.ENVIRONMENT_TIER_UNSPECIFIED) {
      writer.uint32(8).int32(environmentTierPolicy_EnvironmentTierToNumber(message.environmentTier));
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): EnvironmentTierPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEnvironmentTierPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.environmentTier = environmentTierPolicy_EnvironmentTierFromJSON(reader.int32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): EnvironmentTierPolicy {
    return {
      environmentTier: isSet(object.environmentTier)
        ? environmentTierPolicy_EnvironmentTierFromJSON(object.environmentTier)
        : EnvironmentTierPolicy_EnvironmentTier.ENVIRONMENT_TIER_UNSPECIFIED,
    };
  },

  toJSON(message: EnvironmentTierPolicy): unknown {
    const obj: any = {};
    if (message.environmentTier !== EnvironmentTierPolicy_EnvironmentTier.ENVIRONMENT_TIER_UNSPECIFIED) {
      obj.environmentTier = environmentTierPolicy_EnvironmentTierToJSON(message.environmentTier);
    }
    return obj;
  },

  create(base?: DeepPartial<EnvironmentTierPolicy>): EnvironmentTierPolicy {
    return EnvironmentTierPolicy.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<EnvironmentTierPolicy>): EnvironmentTierPolicy {
    const message = createBaseEnvironmentTierPolicy();
    message.environmentTier = object.environmentTier ??
      EnvironmentTierPolicy_EnvironmentTier.ENVIRONMENT_TIER_UNSPECIFIED;
    return message;
  },
};

function createBaseSlowQueryPolicy(): SlowQueryPolicy {
  return { active: false };
}

export const SlowQueryPolicy = {
  encode(message: SlowQueryPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.active === true) {
      writer.uint32(8).bool(message.active);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SlowQueryPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSlowQueryPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.active = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SlowQueryPolicy {
    return { active: isSet(object.active) ? globalThis.Boolean(object.active) : false };
  },

  toJSON(message: SlowQueryPolicy): unknown {
    const obj: any = {};
    if (message.active === true) {
      obj.active = message.active;
    }
    return obj;
  },

  create(base?: DeepPartial<SlowQueryPolicy>): SlowQueryPolicy {
    return SlowQueryPolicy.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SlowQueryPolicy>): SlowQueryPolicy {
    const message = createBaseSlowQueryPolicy();
    message.active = object.active ?? false;
    return message;
  },
};

function createBaseDisableCopyDataPolicy(): DisableCopyDataPolicy {
  return { active: false };
}

export const DisableCopyDataPolicy = {
  encode(message: DisableCopyDataPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.active === true) {
      writer.uint32(8).bool(message.active);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DisableCopyDataPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDisableCopyDataPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.active = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DisableCopyDataPolicy {
    return { active: isSet(object.active) ? globalThis.Boolean(object.active) : false };
  },

  toJSON(message: DisableCopyDataPolicy): unknown {
    const obj: any = {};
    if (message.active === true) {
      obj.active = message.active;
    }
    return obj;
  },

  create(base?: DeepPartial<DisableCopyDataPolicy>): DisableCopyDataPolicy {
    return DisableCopyDataPolicy.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DisableCopyDataPolicy>): DisableCopyDataPolicy {
    const message = createBaseDisableCopyDataPolicy();
    message.active = object.active ?? false;
    return message;
  },
};

function createBaseRestrictIssueCreationForSQLReviewPolicy(): RestrictIssueCreationForSQLReviewPolicy {
  return { disallow: false };
}

export const RestrictIssueCreationForSQLReviewPolicy = {
  encode(message: RestrictIssueCreationForSQLReviewPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.disallow === true) {
      writer.uint32(8).bool(message.disallow);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): RestrictIssueCreationForSQLReviewPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRestrictIssueCreationForSQLReviewPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.disallow = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): RestrictIssueCreationForSQLReviewPolicy {
    return { disallow: isSet(object.disallow) ? globalThis.Boolean(object.disallow) : false };
  },

  toJSON(message: RestrictIssueCreationForSQLReviewPolicy): unknown {
    const obj: any = {};
    if (message.disallow === true) {
      obj.disallow = message.disallow;
    }
    return obj;
  },

  create(base?: DeepPartial<RestrictIssueCreationForSQLReviewPolicy>): RestrictIssueCreationForSQLReviewPolicy {
    return RestrictIssueCreationForSQLReviewPolicy.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<RestrictIssueCreationForSQLReviewPolicy>): RestrictIssueCreationForSQLReviewPolicy {
    const message = createBaseRestrictIssueCreationForSQLReviewPolicy();
    message.disallow = object.disallow ?? false;
    return message;
  },
};

function createBaseDataSourceQueryPolicy(): DataSourceQueryPolicy {
  return { adminDataSourceRestriction: DataSourceQueryPolicy_Restricton.RESTRICTION_UNSPECIFIED };
}

export const DataSourceQueryPolicy = {
  encode(message: DataSourceQueryPolicy, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.adminDataSourceRestriction !== DataSourceQueryPolicy_Restricton.RESTRICTION_UNSPECIFIED) {
      writer.uint32(8).int32(dataSourceQueryPolicy_RestrictonToNumber(message.adminDataSourceRestriction));
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DataSourceQueryPolicy {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDataSourceQueryPolicy();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.adminDataSourceRestriction = dataSourceQueryPolicy_RestrictonFromJSON(reader.int32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DataSourceQueryPolicy {
    return {
      adminDataSourceRestriction: isSet(object.adminDataSourceRestriction)
        ? dataSourceQueryPolicy_RestrictonFromJSON(object.adminDataSourceRestriction)
        : DataSourceQueryPolicy_Restricton.RESTRICTION_UNSPECIFIED,
    };
  },

  toJSON(message: DataSourceQueryPolicy): unknown {
    const obj: any = {};
    if (message.adminDataSourceRestriction !== DataSourceQueryPolicy_Restricton.RESTRICTION_UNSPECIFIED) {
      obj.adminDataSourceRestriction = dataSourceQueryPolicy_RestrictonToJSON(message.adminDataSourceRestriction);
    }
    return obj;
  },

  create(base?: DeepPartial<DataSourceQueryPolicy>): DataSourceQueryPolicy {
    return DataSourceQueryPolicy.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DataSourceQueryPolicy>): DataSourceQueryPolicy {
    const message = createBaseDataSourceQueryPolicy();
    message.adminDataSourceRestriction = object.adminDataSourceRestriction ??
      DataSourceQueryPolicy_Restricton.RESTRICTION_UNSPECIFIED;
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

function isObject(value: any): boolean {
  return typeof value === "object" && value !== null;
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
