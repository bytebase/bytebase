/* eslint-disable */
import _m0 from "protobufjs/minimal";
import { Duration } from "../google/protobuf/duration";
import { Timestamp } from "../google/protobuf/timestamp";
import { Expr } from "../google/type/expr";
import { Engine, engineFromJSON, engineToJSON } from "./common";
import { ColumnConfig, ColumnMetadata, TableConfig, TableMetadata } from "./database_service";
import { ApprovalTemplate } from "./issue_service";
import { PlanType, planTypeFromJSON, planTypeToJSON } from "./subscription_service";

export const protobufPackage = "bytebase.v1";

export interface ListSettingsRequest {
  /**
   * The maximum number of settings to return. The service may return fewer than
   * this value.
   * If unspecified, at most 50 settings will be returned.
   * The maximum value is 1000; values above 1000 will be coerced to 1000.
   */
  pageSize: number;
  /**
   * A page token, received from a previous `ListSettings` call.
   * Provide this to retrieve the subsequent page.
   *
   * When paginating, all other parameters provided to `ListSettings` must match
   * the call that provided the page token.
   */
  pageToken: string;
}

export interface ListSettingsResponse {
  /** The settings from the specified request. */
  settings: Setting[];
  /**
   * A token, which can be sent as `page_token` to retrieve the next page.
   * If this field is omitted, there are no subsequent pages.
   */
  nextPageToken: string;
}

/** The request message for getting a setting. */
export interface GetSettingRequest {
  /** The resource name of the setting. */
  name: string;
}

/** The response message for getting a setting. */
export interface GetSettingResponse {
  setting: Setting | undefined;
}

/** The request message for updating a setting. */
export interface SetSettingRequest {
  /** The setting to update. */
  setting:
    | Setting
    | undefined;
  /**
   * validate_only is a flag to indicate whether to validate the setting value,
   * server would not persist the setting value if it is true.
   */
  validateOnly: boolean;
}

/** The schema of setting. */
export interface Setting {
  /**
   * The resource name of the setting. Must be one of the following forms:
   *
   * - `setting/{setting_name}`
   * For example, "settings/bb.branding.logo"
   */
  name: string;
  /** The value of the setting. */
  value: Value | undefined;
}

/** The data in setting value. */
export interface Value {
  /** Defines this value as being a string value. */
  stringValue?: string | undefined;
  smtpMailDeliverySettingValue?: SMTPMailDeliverySettingValue | undefined;
  appImSettingValue?: AppIMSetting | undefined;
  agentPluginSettingValue?: AgentPluginSetting | undefined;
  workspaceProfileSettingValue?: WorkspaceProfileSetting | undefined;
  workspaceApprovalSettingValue?: WorkspaceApprovalSetting | undefined;
  workspaceTrialSettingValue?: WorkspaceTrialSetting | undefined;
  externalApprovalSettingValue?: ExternalApprovalSetting | undefined;
  schemaTemplateSettingValue?: SchemaTemplateSetting | undefined;
  dataClassificationSettingValue?: DataClassificationSetting | undefined;
  semanticTypeSettingValue?: SemanticTypeSetting | undefined;
  maskingAlgorithmSettingValue?: MaskingAlgorithmSetting | undefined;
}

export interface SMTPMailDeliverySettingValue {
  /** The SMTP server address. */
  server: string;
  /** The SMTP server port. */
  port: number;
  /** The SMTP server encryption. */
  encryption: SMTPMailDeliverySettingValue_Encryption;
  /**
   * The CA, KEY, and CERT for the SMTP server.
   * Not used.
   */
  ca?: string | undefined;
  key?: string | undefined;
  cert?: string | undefined;
  authentication: SMTPMailDeliverySettingValue_Authentication;
  username: string;
  /** If not specified, server will use the existed password. */
  password?:
    | string
    | undefined;
  /** The sender email address. */
  from: string;
  /** The recipient email address, used with validate_only to send test email. */
  to: string;
}

/** We support three types of SMTP encryption: NONE, STARTTLS, and SSL/TLS. */
export enum SMTPMailDeliverySettingValue_Encryption {
  ENCRYPTION_UNSPECIFIED = 0,
  ENCRYPTION_NONE = 1,
  ENCRYPTION_STARTTLS = 2,
  ENCRYPTION_SSL_TLS = 3,
  UNRECOGNIZED = -1,
}

export function sMTPMailDeliverySettingValue_EncryptionFromJSON(object: any): SMTPMailDeliverySettingValue_Encryption {
  switch (object) {
    case 0:
    case "ENCRYPTION_UNSPECIFIED":
      return SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_UNSPECIFIED;
    case 1:
    case "ENCRYPTION_NONE":
      return SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_NONE;
    case 2:
    case "ENCRYPTION_STARTTLS":
      return SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_STARTTLS;
    case 3:
    case "ENCRYPTION_SSL_TLS":
      return SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_SSL_TLS;
    case -1:
    case "UNRECOGNIZED":
    default:
      return SMTPMailDeliverySettingValue_Encryption.UNRECOGNIZED;
  }
}

export function sMTPMailDeliverySettingValue_EncryptionToJSON(object: SMTPMailDeliverySettingValue_Encryption): string {
  switch (object) {
    case SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_UNSPECIFIED:
      return "ENCRYPTION_UNSPECIFIED";
    case SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_NONE:
      return "ENCRYPTION_NONE";
    case SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_STARTTLS:
      return "ENCRYPTION_STARTTLS";
    case SMTPMailDeliverySettingValue_Encryption.ENCRYPTION_SSL_TLS:
      return "ENCRYPTION_SSL_TLS";
    case SMTPMailDeliverySettingValue_Encryption.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

/** We support four types of SMTP authentication: NONE, PLAIN, LOGIN, and CRAM-MD5. */
export enum SMTPMailDeliverySettingValue_Authentication {
  AUTHENTICATION_UNSPECIFIED = 0,
  AUTHENTICATION_NONE = 1,
  AUTHENTICATION_PLAIN = 2,
  AUTHENTICATION_LOGIN = 3,
  AUTHENTICATION_CRAM_MD5 = 4,
  UNRECOGNIZED = -1,
}

export function sMTPMailDeliverySettingValue_AuthenticationFromJSON(
  object: any,
): SMTPMailDeliverySettingValue_Authentication {
  switch (object) {
    case 0:
    case "AUTHENTICATION_UNSPECIFIED":
      return SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_UNSPECIFIED;
    case 1:
    case "AUTHENTICATION_NONE":
      return SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_NONE;
    case 2:
    case "AUTHENTICATION_PLAIN":
      return SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_PLAIN;
    case 3:
    case "AUTHENTICATION_LOGIN":
      return SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_LOGIN;
    case 4:
    case "AUTHENTICATION_CRAM_MD5":
      return SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_CRAM_MD5;
    case -1:
    case "UNRECOGNIZED":
    default:
      return SMTPMailDeliverySettingValue_Authentication.UNRECOGNIZED;
  }
}

export function sMTPMailDeliverySettingValue_AuthenticationToJSON(
  object: SMTPMailDeliverySettingValue_Authentication,
): string {
  switch (object) {
    case SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_UNSPECIFIED:
      return "AUTHENTICATION_UNSPECIFIED";
    case SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_NONE:
      return "AUTHENTICATION_NONE";
    case SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_PLAIN:
      return "AUTHENTICATION_PLAIN";
    case SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_LOGIN:
      return "AUTHENTICATION_LOGIN";
    case SMTPMailDeliverySettingValue_Authentication.AUTHENTICATION_CRAM_MD5:
      return "AUTHENTICATION_CRAM_MD5";
    case SMTPMailDeliverySettingValue_Authentication.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface AppIMSetting {
  imType: AppIMSetting_IMType;
  appId: string;
  appSecret: string;
  externalApproval: AppIMSetting_ExternalApproval | undefined;
}

export enum AppIMSetting_IMType {
  IM_TYPE_UNSPECIFIED = 0,
  FEISHU = 1,
  UNRECOGNIZED = -1,
}

export function appIMSetting_IMTypeFromJSON(object: any): AppIMSetting_IMType {
  switch (object) {
    case 0:
    case "IM_TYPE_UNSPECIFIED":
      return AppIMSetting_IMType.IM_TYPE_UNSPECIFIED;
    case 1:
    case "FEISHU":
      return AppIMSetting_IMType.FEISHU;
    case -1:
    case "UNRECOGNIZED":
    default:
      return AppIMSetting_IMType.UNRECOGNIZED;
  }
}

export function appIMSetting_IMTypeToJSON(object: AppIMSetting_IMType): string {
  switch (object) {
    case AppIMSetting_IMType.IM_TYPE_UNSPECIFIED:
      return "IM_TYPE_UNSPECIFIED";
    case AppIMSetting_IMType.FEISHU:
      return "FEISHU";
    case AppIMSetting_IMType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface AppIMSetting_ExternalApproval {
  enabled: boolean;
  approvalDefinitionId: string;
}

export interface AgentPluginSetting {
  /** The URL for the agent API. */
  url: string;
  /** The token for the agent. */
  token: string;
}

export interface WorkspaceProfileSetting {
  /**
   * The URL user visits Bytebase.
   *
   * The external URL is used for:
   * 1. Constructing the correct callback URL when configuring the VCS provider. The callback URL points to the frontend.
   * 2. Creating the correct webhook endpoint when configuring the project GitOps workflow. The webhook endpoint points to the backend.
   */
  externalUrl: string;
  /** Disallow self-service signup, users can only be invited by the owner. */
  disallowSignup: boolean;
  /** Require 2FA for all users. */
  require2fa: boolean;
  /** outbound_ip_list is the outbound IP for Bytebase instance in SaaS mode. */
  outboundIpList: string[];
  /** The webhook URL for the GitOps workflow. */
  gitopsWebhookUrl: string;
  /** The duration for token. */
  tokenDuration:
    | Duration
    | undefined;
  /** The setting of custom announcement */
  announcement: Announcement | undefined;
}

export interface Announcement {
  /** The alert level of announcemnt */
  level: Announcement_AlertLevel;
  /** The text of announcemnt */
  text: string;
  /** The optional link, user can follow the link to check extra details */
  link: string;
}

/** We support three levels of AlertLevel: INFO, WARNING, and ERROR. */
export enum Announcement_AlertLevel {
  ALERT_LEVEL_UNSPECIFIED = 0,
  ALERT_LEVEL_INFO = 1,
  ALERT_LEVEL_WARNING = 2,
  ALERT_LEVEL_CRITICAL = 3,
  UNRECOGNIZED = -1,
}

export function announcement_AlertLevelFromJSON(object: any): Announcement_AlertLevel {
  switch (object) {
    case 0:
    case "ALERT_LEVEL_UNSPECIFIED":
      return Announcement_AlertLevel.ALERT_LEVEL_UNSPECIFIED;
    case 1:
    case "ALERT_LEVEL_INFO":
      return Announcement_AlertLevel.ALERT_LEVEL_INFO;
    case 2:
    case "ALERT_LEVEL_WARNING":
      return Announcement_AlertLevel.ALERT_LEVEL_WARNING;
    case 3:
    case "ALERT_LEVEL_CRITICAL":
      return Announcement_AlertLevel.ALERT_LEVEL_CRITICAL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Announcement_AlertLevel.UNRECOGNIZED;
  }
}

export function announcement_AlertLevelToJSON(object: Announcement_AlertLevel): string {
  switch (object) {
    case Announcement_AlertLevel.ALERT_LEVEL_UNSPECIFIED:
      return "ALERT_LEVEL_UNSPECIFIED";
    case Announcement_AlertLevel.ALERT_LEVEL_INFO:
      return "ALERT_LEVEL_INFO";
    case Announcement_AlertLevel.ALERT_LEVEL_WARNING:
      return "ALERT_LEVEL_WARNING";
    case Announcement_AlertLevel.ALERT_LEVEL_CRITICAL:
      return "ALERT_LEVEL_CRITICAL";
    case Announcement_AlertLevel.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface WorkspaceApprovalSetting {
  rules: WorkspaceApprovalSetting_Rule[];
}

export interface WorkspaceApprovalSetting_Rule {
  template: ApprovalTemplate | undefined;
  condition: Expr | undefined;
}

export interface ExternalApprovalSetting {
  nodes: ExternalApprovalSetting_Node[];
}

export interface ExternalApprovalSetting_Node {
  /**
   * A unique identifier for a node in UUID format.
   * We will also include the id in the message sending to the external relay service to identify the node.
   */
  id: string;
  /** The title of the node. */
  title: string;
  /** The external endpoint for the relay service, e.g. "http://hello:1234". */
  endpoint: string;
}

export interface SchemaTemplateSetting {
  fieldTemplates: SchemaTemplateSetting_FieldTemplate[];
  columnTypes: SchemaTemplateSetting_ColumnType[];
  tableTemplates: SchemaTemplateSetting_TableTemplate[];
}

export interface SchemaTemplateSetting_FieldTemplate {
  id: string;
  engine: Engine;
  category: string;
  column: ColumnMetadata | undefined;
  config: ColumnConfig | undefined;
}

export interface SchemaTemplateSetting_ColumnType {
  engine: Engine;
  enabled: boolean;
  types: string[];
}

export interface SchemaTemplateSetting_TableTemplate {
  id: string;
  engine: Engine;
  category: string;
  table: TableMetadata | undefined;
  config: TableConfig | undefined;
}

export interface WorkspaceTrialSetting {
  instanceCount: number;
  expireTime: Date | undefined;
  issuedTime: Date | undefined;
  subject: string;
  orgName: string;
  plan: PlanType;
}

export interface DataClassificationSetting {
  configs: DataClassificationSetting_DataClassificationConfig[];
}

export interface DataClassificationSetting_DataClassificationConfig {
  /** id is the uuid for classification. Each project can chose one classification config. */
  id: string;
  title: string;
  /**
   * levels is user defined level list for classification.
   * The order for the level decides its priority.
   */
  levels: DataClassificationSetting_DataClassificationConfig_Level[];
  /**
   * classification is the id - DataClassification map.
   * The id should in [0-9]+-[0-9]+-[0-9]+ format.
   */
  classification: { [key: string]: DataClassificationSetting_DataClassificationConfig_DataClassification };
}

export interface DataClassificationSetting_DataClassificationConfig_Level {
  id: string;
  title: string;
  description: string;
}

export interface DataClassificationSetting_DataClassificationConfig_DataClassification {
  /** id is the classification id in [0-9]+-[0-9]+-[0-9]+ format. */
  id: string;
  title: string;
  description: string;
  levelId?: string | undefined;
}

export interface DataClassificationSetting_DataClassificationConfig_ClassificationEntry {
  key: string;
  value: DataClassificationSetting_DataClassificationConfig_DataClassification | undefined;
}

export interface SemanticTypeSetting {
  types: SemanticTypeSetting_SemanticType[];
}

export interface SemanticTypeSetting_SemanticType {
  /** id is the uuid for semantic type. */
  id: string;
  /** the title of the semantic type, it should not be empty. */
  title: string;
  /** the description of the semantic type, it can be empty. */
  description: string;
  /** the partial mask algorithm id for the semantic type, if it is empty, should use the default partial mask algorithm. */
  partialMaskAlgorithmId: string;
  /** the full mask algorithm id for the semantic type, if it is empty, should use the default full mask algorithm. */
  fullMaskAlgorithmId: string;
}

export interface MaskingAlgorithmSetting {
  /** algorithms is the list of masking algorithms. */
  algorithms: MaskingAlgorithmSetting_Algorithm[];
}

export interface MaskingAlgorithmSetting_Algorithm {
  /** id is the uuid for masking algorithm. */
  id: string;
  /** title is the title for masking algorithm. */
  title: string;
  /** description is the description for masking algorithm. */
  description: string;
  /**
   * Category is the category for masking algorithm. Currently, it accepts 2 categories only: MASK and HASH.
   * The range of accepted Payload is decided by the category.
   * MASK: FullMask, RangeMask
   * HASH: MD5Mask
   */
  category: string;
  fullMask?: MaskingAlgorithmSetting_Algorithm_FullMask | undefined;
  rangeMask?: MaskingAlgorithmSetting_Algorithm_RangeMask | undefined;
  md5Mask?: MaskingAlgorithmSetting_Algorithm_MD5Mask | undefined;
}

export interface MaskingAlgorithmSetting_Algorithm_FullMask {
  /**
   * substitution is the string used to replace the original value, the
   * max length of the string is 16 bytes.
   */
  substitution: string;
}

export interface MaskingAlgorithmSetting_Algorithm_RangeMask {
  /**
   * We store it as a repeated field to face the fact that the original value may have multiple parts should be masked.
   * But frontend can be started with a single rule easily.
   */
  slices: MaskingAlgorithmSetting_Algorithm_RangeMask_Slice[];
}

export interface MaskingAlgorithmSetting_Algorithm_RangeMask_Slice {
  /** start is the start index of the original value, start from 0 and should be less than stop. */
  start: number;
  /** stop is the stop index of the original value, should be less than the length of the original value. */
  end: number;
  /** substitution is the string used to replace the OriginalValue[start:end). */
  substitution: string;
}

export interface MaskingAlgorithmSetting_Algorithm_MD5Mask {
  /** salt is the salt value to generate a different hash that with the word alone. */
  salt: string;
}

function createBaseListSettingsRequest(): ListSettingsRequest {
  return { pageSize: 0, pageToken: "" };
}

export const ListSettingsRequest = {
  encode(message: ListSettingsRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.pageSize !== 0) {
      writer.uint32(8).int32(message.pageSize);
    }
    if (message.pageToken !== "") {
      writer.uint32(18).string(message.pageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListSettingsRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListSettingsRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.pageSize = reader.int32();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.pageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListSettingsRequest {
    return {
      pageSize: isSet(object.pageSize) ? Number(object.pageSize) : 0,
      pageToken: isSet(object.pageToken) ? String(object.pageToken) : "",
    };
  },

  toJSON(message: ListSettingsRequest): unknown {
    const obj: any = {};
    message.pageSize !== undefined && (obj.pageSize = Math.round(message.pageSize));
    message.pageToken !== undefined && (obj.pageToken = message.pageToken);
    return obj;
  },

  create(base?: DeepPartial<ListSettingsRequest>): ListSettingsRequest {
    return ListSettingsRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListSettingsRequest>): ListSettingsRequest {
    const message = createBaseListSettingsRequest();
    message.pageSize = object.pageSize ?? 0;
    message.pageToken = object.pageToken ?? "";
    return message;
  },
};

function createBaseListSettingsResponse(): ListSettingsResponse {
  return { settings: [], nextPageToken: "" };
}

export const ListSettingsResponse = {
  encode(message: ListSettingsResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.settings) {
      Setting.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    if (message.nextPageToken !== "") {
      writer.uint32(18).string(message.nextPageToken);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ListSettingsResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseListSettingsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.settings.push(Setting.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.nextPageToken = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ListSettingsResponse {
    return {
      settings: Array.isArray(object?.settings) ? object.settings.map((e: any) => Setting.fromJSON(e)) : [],
      nextPageToken: isSet(object.nextPageToken) ? String(object.nextPageToken) : "",
    };
  },

  toJSON(message: ListSettingsResponse): unknown {
    const obj: any = {};
    if (message.settings) {
      obj.settings = message.settings.map((e) => e ? Setting.toJSON(e) : undefined);
    } else {
      obj.settings = [];
    }
    message.nextPageToken !== undefined && (obj.nextPageToken = message.nextPageToken);
    return obj;
  },

  create(base?: DeepPartial<ListSettingsResponse>): ListSettingsResponse {
    return ListSettingsResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ListSettingsResponse>): ListSettingsResponse {
    const message = createBaseListSettingsResponse();
    message.settings = object.settings?.map((e) => Setting.fromPartial(e)) || [];
    message.nextPageToken = object.nextPageToken ?? "";
    return message;
  },
};

function createBaseGetSettingRequest(): GetSettingRequest {
  return { name: "" };
}

export const GetSettingRequest = {
  encode(message: GetSettingRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetSettingRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetSettingRequest();
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

  fromJSON(object: any): GetSettingRequest {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: GetSettingRequest): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  create(base?: DeepPartial<GetSettingRequest>): GetSettingRequest {
    return GetSettingRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<GetSettingRequest>): GetSettingRequest {
    const message = createBaseGetSettingRequest();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseGetSettingResponse(): GetSettingResponse {
  return { setting: undefined };
}

export const GetSettingResponse = {
  encode(message: GetSettingResponse, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.setting !== undefined) {
      Setting.encode(message.setting, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): GetSettingResponse {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGetSettingResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.setting = Setting.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): GetSettingResponse {
    return { setting: isSet(object.setting) ? Setting.fromJSON(object.setting) : undefined };
  },

  toJSON(message: GetSettingResponse): unknown {
    const obj: any = {};
    message.setting !== undefined && (obj.setting = message.setting ? Setting.toJSON(message.setting) : undefined);
    return obj;
  },

  create(base?: DeepPartial<GetSettingResponse>): GetSettingResponse {
    return GetSettingResponse.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<GetSettingResponse>): GetSettingResponse {
    const message = createBaseGetSettingResponse();
    message.setting = (object.setting !== undefined && object.setting !== null)
      ? Setting.fromPartial(object.setting)
      : undefined;
    return message;
  },
};

function createBaseSetSettingRequest(): SetSettingRequest {
  return { setting: undefined, validateOnly: false };
}

export const SetSettingRequest = {
  encode(message: SetSettingRequest, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.setting !== undefined) {
      Setting.encode(message.setting, writer.uint32(10).fork()).ldelim();
    }
    if (message.validateOnly === true) {
      writer.uint32(16).bool(message.validateOnly);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SetSettingRequest {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSetSettingRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.setting = Setting.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.validateOnly = reader.bool();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SetSettingRequest {
    return {
      setting: isSet(object.setting) ? Setting.fromJSON(object.setting) : undefined,
      validateOnly: isSet(object.validateOnly) ? Boolean(object.validateOnly) : false,
    };
  },

  toJSON(message: SetSettingRequest): unknown {
    const obj: any = {};
    message.setting !== undefined && (obj.setting = message.setting ? Setting.toJSON(message.setting) : undefined);
    message.validateOnly !== undefined && (obj.validateOnly = message.validateOnly);
    return obj;
  },

  create(base?: DeepPartial<SetSettingRequest>): SetSettingRequest {
    return SetSettingRequest.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SetSettingRequest>): SetSettingRequest {
    const message = createBaseSetSettingRequest();
    message.setting = (object.setting !== undefined && object.setting !== null)
      ? Setting.fromPartial(object.setting)
      : undefined;
    message.validateOnly = object.validateOnly ?? false;
    return message;
  },
};

function createBaseSetting(): Setting {
  return { name: "", value: undefined };
}

export const Setting = {
  encode(message: Setting, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.value !== undefined) {
      Value.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Setting {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSetting();
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

          message.value = Value.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Setting {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      value: isSet(object.value) ? Value.fromJSON(object.value) : undefined,
    };
  },

  toJSON(message: Setting): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.value !== undefined && (obj.value = message.value ? Value.toJSON(message.value) : undefined);
    return obj;
  },

  create(base?: DeepPartial<Setting>): Setting {
    return Setting.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Setting>): Setting {
    const message = createBaseSetting();
    message.name = object.name ?? "";
    message.value = (object.value !== undefined && object.value !== null) ? Value.fromPartial(object.value) : undefined;
    return message;
  },
};

function createBaseValue(): Value {
  return {
    stringValue: undefined,
    smtpMailDeliverySettingValue: undefined,
    appImSettingValue: undefined,
    agentPluginSettingValue: undefined,
    workspaceProfileSettingValue: undefined,
    workspaceApprovalSettingValue: undefined,
    workspaceTrialSettingValue: undefined,
    externalApprovalSettingValue: undefined,
    schemaTemplateSettingValue: undefined,
    dataClassificationSettingValue: undefined,
    semanticTypeSettingValue: undefined,
    maskingAlgorithmSettingValue: undefined,
  };
}

export const Value = {
  encode(message: Value, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.stringValue !== undefined) {
      writer.uint32(10).string(message.stringValue);
    }
    if (message.smtpMailDeliverySettingValue !== undefined) {
      SMTPMailDeliverySettingValue.encode(message.smtpMailDeliverySettingValue, writer.uint32(18).fork()).ldelim();
    }
    if (message.appImSettingValue !== undefined) {
      AppIMSetting.encode(message.appImSettingValue, writer.uint32(26).fork()).ldelim();
    }
    if (message.agentPluginSettingValue !== undefined) {
      AgentPluginSetting.encode(message.agentPluginSettingValue, writer.uint32(34).fork()).ldelim();
    }
    if (message.workspaceProfileSettingValue !== undefined) {
      WorkspaceProfileSetting.encode(message.workspaceProfileSettingValue, writer.uint32(42).fork()).ldelim();
    }
    if (message.workspaceApprovalSettingValue !== undefined) {
      WorkspaceApprovalSetting.encode(message.workspaceApprovalSettingValue, writer.uint32(50).fork()).ldelim();
    }
    if (message.workspaceTrialSettingValue !== undefined) {
      WorkspaceTrialSetting.encode(message.workspaceTrialSettingValue, writer.uint32(58).fork()).ldelim();
    }
    if (message.externalApprovalSettingValue !== undefined) {
      ExternalApprovalSetting.encode(message.externalApprovalSettingValue, writer.uint32(66).fork()).ldelim();
    }
    if (message.schemaTemplateSettingValue !== undefined) {
      SchemaTemplateSetting.encode(message.schemaTemplateSettingValue, writer.uint32(74).fork()).ldelim();
    }
    if (message.dataClassificationSettingValue !== undefined) {
      DataClassificationSetting.encode(message.dataClassificationSettingValue, writer.uint32(82).fork()).ldelim();
    }
    if (message.semanticTypeSettingValue !== undefined) {
      SemanticTypeSetting.encode(message.semanticTypeSettingValue, writer.uint32(90).fork()).ldelim();
    }
    if (message.maskingAlgorithmSettingValue !== undefined) {
      MaskingAlgorithmSetting.encode(message.maskingAlgorithmSettingValue, writer.uint32(98).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Value {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseValue();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.stringValue = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.smtpMailDeliverySettingValue = SMTPMailDeliverySettingValue.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.appImSettingValue = AppIMSetting.decode(reader, reader.uint32());
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.agentPluginSettingValue = AgentPluginSetting.decode(reader, reader.uint32());
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.workspaceProfileSettingValue = WorkspaceProfileSetting.decode(reader, reader.uint32());
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.workspaceApprovalSettingValue = WorkspaceApprovalSetting.decode(reader, reader.uint32());
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.workspaceTrialSettingValue = WorkspaceTrialSetting.decode(reader, reader.uint32());
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.externalApprovalSettingValue = ExternalApprovalSetting.decode(reader, reader.uint32());
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.schemaTemplateSettingValue = SchemaTemplateSetting.decode(reader, reader.uint32());
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.dataClassificationSettingValue = DataClassificationSetting.decode(reader, reader.uint32());
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.semanticTypeSettingValue = SemanticTypeSetting.decode(reader, reader.uint32());
          continue;
        case 12:
          if (tag !== 98) {
            break;
          }

          message.maskingAlgorithmSettingValue = MaskingAlgorithmSetting.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Value {
    return {
      stringValue: isSet(object.stringValue) ? String(object.stringValue) : undefined,
      smtpMailDeliverySettingValue: isSet(object.smtpMailDeliverySettingValue)
        ? SMTPMailDeliverySettingValue.fromJSON(object.smtpMailDeliverySettingValue)
        : undefined,
      appImSettingValue: isSet(object.appImSettingValue) ? AppIMSetting.fromJSON(object.appImSettingValue) : undefined,
      agentPluginSettingValue: isSet(object.agentPluginSettingValue)
        ? AgentPluginSetting.fromJSON(object.agentPluginSettingValue)
        : undefined,
      workspaceProfileSettingValue: isSet(object.workspaceProfileSettingValue)
        ? WorkspaceProfileSetting.fromJSON(object.workspaceProfileSettingValue)
        : undefined,
      workspaceApprovalSettingValue: isSet(object.workspaceApprovalSettingValue)
        ? WorkspaceApprovalSetting.fromJSON(object.workspaceApprovalSettingValue)
        : undefined,
      workspaceTrialSettingValue: isSet(object.workspaceTrialSettingValue)
        ? WorkspaceTrialSetting.fromJSON(object.workspaceTrialSettingValue)
        : undefined,
      externalApprovalSettingValue: isSet(object.externalApprovalSettingValue)
        ? ExternalApprovalSetting.fromJSON(object.externalApprovalSettingValue)
        : undefined,
      schemaTemplateSettingValue: isSet(object.schemaTemplateSettingValue)
        ? SchemaTemplateSetting.fromJSON(object.schemaTemplateSettingValue)
        : undefined,
      dataClassificationSettingValue: isSet(object.dataClassificationSettingValue)
        ? DataClassificationSetting.fromJSON(object.dataClassificationSettingValue)
        : undefined,
      semanticTypeSettingValue: isSet(object.semanticTypeSettingValue)
        ? SemanticTypeSetting.fromJSON(object.semanticTypeSettingValue)
        : undefined,
      maskingAlgorithmSettingValue: isSet(object.maskingAlgorithmSettingValue)
        ? MaskingAlgorithmSetting.fromJSON(object.maskingAlgorithmSettingValue)
        : undefined,
    };
  },

  toJSON(message: Value): unknown {
    const obj: any = {};
    message.stringValue !== undefined && (obj.stringValue = message.stringValue);
    message.smtpMailDeliverySettingValue !== undefined &&
      (obj.smtpMailDeliverySettingValue = message.smtpMailDeliverySettingValue
        ? SMTPMailDeliverySettingValue.toJSON(message.smtpMailDeliverySettingValue)
        : undefined);
    message.appImSettingValue !== undefined &&
      (obj.appImSettingValue = message.appImSettingValue ? AppIMSetting.toJSON(message.appImSettingValue) : undefined);
    message.agentPluginSettingValue !== undefined && (obj.agentPluginSettingValue = message.agentPluginSettingValue
      ? AgentPluginSetting.toJSON(message.agentPluginSettingValue)
      : undefined);
    message.workspaceProfileSettingValue !== undefined &&
      (obj.workspaceProfileSettingValue = message.workspaceProfileSettingValue
        ? WorkspaceProfileSetting.toJSON(message.workspaceProfileSettingValue)
        : undefined);
    message.workspaceApprovalSettingValue !== undefined &&
      (obj.workspaceApprovalSettingValue = message.workspaceApprovalSettingValue
        ? WorkspaceApprovalSetting.toJSON(message.workspaceApprovalSettingValue)
        : undefined);
    message.workspaceTrialSettingValue !== undefined &&
      (obj.workspaceTrialSettingValue = message.workspaceTrialSettingValue
        ? WorkspaceTrialSetting.toJSON(message.workspaceTrialSettingValue)
        : undefined);
    message.externalApprovalSettingValue !== undefined &&
      (obj.externalApprovalSettingValue = message.externalApprovalSettingValue
        ? ExternalApprovalSetting.toJSON(message.externalApprovalSettingValue)
        : undefined);
    message.schemaTemplateSettingValue !== undefined &&
      (obj.schemaTemplateSettingValue = message.schemaTemplateSettingValue
        ? SchemaTemplateSetting.toJSON(message.schemaTemplateSettingValue)
        : undefined);
    message.dataClassificationSettingValue !== undefined &&
      (obj.dataClassificationSettingValue = message.dataClassificationSettingValue
        ? DataClassificationSetting.toJSON(message.dataClassificationSettingValue)
        : undefined);
    message.semanticTypeSettingValue !== undefined && (obj.semanticTypeSettingValue = message.semanticTypeSettingValue
      ? SemanticTypeSetting.toJSON(message.semanticTypeSettingValue)
      : undefined);
    message.maskingAlgorithmSettingValue !== undefined &&
      (obj.maskingAlgorithmSettingValue = message.maskingAlgorithmSettingValue
        ? MaskingAlgorithmSetting.toJSON(message.maskingAlgorithmSettingValue)
        : undefined);
    return obj;
  },

  create(base?: DeepPartial<Value>): Value {
    return Value.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Value>): Value {
    const message = createBaseValue();
    message.stringValue = object.stringValue ?? undefined;
    message.smtpMailDeliverySettingValue =
      (object.smtpMailDeliverySettingValue !== undefined && object.smtpMailDeliverySettingValue !== null)
        ? SMTPMailDeliverySettingValue.fromPartial(object.smtpMailDeliverySettingValue)
        : undefined;
    message.appImSettingValue = (object.appImSettingValue !== undefined && object.appImSettingValue !== null)
      ? AppIMSetting.fromPartial(object.appImSettingValue)
      : undefined;
    message.agentPluginSettingValue =
      (object.agentPluginSettingValue !== undefined && object.agentPluginSettingValue !== null)
        ? AgentPluginSetting.fromPartial(object.agentPluginSettingValue)
        : undefined;
    message.workspaceProfileSettingValue =
      (object.workspaceProfileSettingValue !== undefined && object.workspaceProfileSettingValue !== null)
        ? WorkspaceProfileSetting.fromPartial(object.workspaceProfileSettingValue)
        : undefined;
    message.workspaceApprovalSettingValue =
      (object.workspaceApprovalSettingValue !== undefined && object.workspaceApprovalSettingValue !== null)
        ? WorkspaceApprovalSetting.fromPartial(object.workspaceApprovalSettingValue)
        : undefined;
    message.workspaceTrialSettingValue =
      (object.workspaceTrialSettingValue !== undefined && object.workspaceTrialSettingValue !== null)
        ? WorkspaceTrialSetting.fromPartial(object.workspaceTrialSettingValue)
        : undefined;
    message.externalApprovalSettingValue =
      (object.externalApprovalSettingValue !== undefined && object.externalApprovalSettingValue !== null)
        ? ExternalApprovalSetting.fromPartial(object.externalApprovalSettingValue)
        : undefined;
    message.schemaTemplateSettingValue =
      (object.schemaTemplateSettingValue !== undefined && object.schemaTemplateSettingValue !== null)
        ? SchemaTemplateSetting.fromPartial(object.schemaTemplateSettingValue)
        : undefined;
    message.dataClassificationSettingValue =
      (object.dataClassificationSettingValue !== undefined && object.dataClassificationSettingValue !== null)
        ? DataClassificationSetting.fromPartial(object.dataClassificationSettingValue)
        : undefined;
    message.semanticTypeSettingValue =
      (object.semanticTypeSettingValue !== undefined && object.semanticTypeSettingValue !== null)
        ? SemanticTypeSetting.fromPartial(object.semanticTypeSettingValue)
        : undefined;
    message.maskingAlgorithmSettingValue =
      (object.maskingAlgorithmSettingValue !== undefined && object.maskingAlgorithmSettingValue !== null)
        ? MaskingAlgorithmSetting.fromPartial(object.maskingAlgorithmSettingValue)
        : undefined;
    return message;
  },
};

function createBaseSMTPMailDeliverySettingValue(): SMTPMailDeliverySettingValue {
  return {
    server: "",
    port: 0,
    encryption: 0,
    ca: undefined,
    key: undefined,
    cert: undefined,
    authentication: 0,
    username: "",
    password: undefined,
    from: "",
    to: "",
  };
}

export const SMTPMailDeliverySettingValue = {
  encode(message: SMTPMailDeliverySettingValue, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.server !== "") {
      writer.uint32(10).string(message.server);
    }
    if (message.port !== 0) {
      writer.uint32(16).int32(message.port);
    }
    if (message.encryption !== 0) {
      writer.uint32(24).int32(message.encryption);
    }
    if (message.ca !== undefined) {
      writer.uint32(34).string(message.ca);
    }
    if (message.key !== undefined) {
      writer.uint32(42).string(message.key);
    }
    if (message.cert !== undefined) {
      writer.uint32(50).string(message.cert);
    }
    if (message.authentication !== 0) {
      writer.uint32(56).int32(message.authentication);
    }
    if (message.username !== "") {
      writer.uint32(66).string(message.username);
    }
    if (message.password !== undefined) {
      writer.uint32(74).string(message.password);
    }
    if (message.from !== "") {
      writer.uint32(82).string(message.from);
    }
    if (message.to !== "") {
      writer.uint32(90).string(message.to);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SMTPMailDeliverySettingValue {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSMTPMailDeliverySettingValue();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.server = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.port = reader.int32();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.encryption = reader.int32() as any;
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.ca = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.key = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.cert = reader.string();
          continue;
        case 7:
          if (tag !== 56) {
            break;
          }

          message.authentication = reader.int32() as any;
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.username = reader.string();
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.password = reader.string();
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.from = reader.string();
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.to = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SMTPMailDeliverySettingValue {
    return {
      server: isSet(object.server) ? String(object.server) : "",
      port: isSet(object.port) ? Number(object.port) : 0,
      encryption: isSet(object.encryption) ? sMTPMailDeliverySettingValue_EncryptionFromJSON(object.encryption) : 0,
      ca: isSet(object.ca) ? String(object.ca) : undefined,
      key: isSet(object.key) ? String(object.key) : undefined,
      cert: isSet(object.cert) ? String(object.cert) : undefined,
      authentication: isSet(object.authentication)
        ? sMTPMailDeliverySettingValue_AuthenticationFromJSON(object.authentication)
        : 0,
      username: isSet(object.username) ? String(object.username) : "",
      password: isSet(object.password) ? String(object.password) : undefined,
      from: isSet(object.from) ? String(object.from) : "",
      to: isSet(object.to) ? String(object.to) : "",
    };
  },

  toJSON(message: SMTPMailDeliverySettingValue): unknown {
    const obj: any = {};
    message.server !== undefined && (obj.server = message.server);
    message.port !== undefined && (obj.port = Math.round(message.port));
    message.encryption !== undefined &&
      (obj.encryption = sMTPMailDeliverySettingValue_EncryptionToJSON(message.encryption));
    message.ca !== undefined && (obj.ca = message.ca);
    message.key !== undefined && (obj.key = message.key);
    message.cert !== undefined && (obj.cert = message.cert);
    message.authentication !== undefined &&
      (obj.authentication = sMTPMailDeliverySettingValue_AuthenticationToJSON(message.authentication));
    message.username !== undefined && (obj.username = message.username);
    message.password !== undefined && (obj.password = message.password);
    message.from !== undefined && (obj.from = message.from);
    message.to !== undefined && (obj.to = message.to);
    return obj;
  },

  create(base?: DeepPartial<SMTPMailDeliverySettingValue>): SMTPMailDeliverySettingValue {
    return SMTPMailDeliverySettingValue.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SMTPMailDeliverySettingValue>): SMTPMailDeliverySettingValue {
    const message = createBaseSMTPMailDeliverySettingValue();
    message.server = object.server ?? "";
    message.port = object.port ?? 0;
    message.encryption = object.encryption ?? 0;
    message.ca = object.ca ?? undefined;
    message.key = object.key ?? undefined;
    message.cert = object.cert ?? undefined;
    message.authentication = object.authentication ?? 0;
    message.username = object.username ?? "";
    message.password = object.password ?? undefined;
    message.from = object.from ?? "";
    message.to = object.to ?? "";
    return message;
  },
};

function createBaseAppIMSetting(): AppIMSetting {
  return { imType: 0, appId: "", appSecret: "", externalApproval: undefined };
}

export const AppIMSetting = {
  encode(message: AppIMSetting, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.imType !== 0) {
      writer.uint32(8).int32(message.imType);
    }
    if (message.appId !== "") {
      writer.uint32(18).string(message.appId);
    }
    if (message.appSecret !== "") {
      writer.uint32(26).string(message.appSecret);
    }
    if (message.externalApproval !== undefined) {
      AppIMSetting_ExternalApproval.encode(message.externalApproval, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): AppIMSetting {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAppIMSetting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.imType = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.appId = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.appSecret = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.externalApproval = AppIMSetting_ExternalApproval.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): AppIMSetting {
    return {
      imType: isSet(object.imType) ? appIMSetting_IMTypeFromJSON(object.imType) : 0,
      appId: isSet(object.appId) ? String(object.appId) : "",
      appSecret: isSet(object.appSecret) ? String(object.appSecret) : "",
      externalApproval: isSet(object.externalApproval)
        ? AppIMSetting_ExternalApproval.fromJSON(object.externalApproval)
        : undefined,
    };
  },

  toJSON(message: AppIMSetting): unknown {
    const obj: any = {};
    message.imType !== undefined && (obj.imType = appIMSetting_IMTypeToJSON(message.imType));
    message.appId !== undefined && (obj.appId = message.appId);
    message.appSecret !== undefined && (obj.appSecret = message.appSecret);
    message.externalApproval !== undefined && (obj.externalApproval = message.externalApproval
      ? AppIMSetting_ExternalApproval.toJSON(message.externalApproval)
      : undefined);
    return obj;
  },

  create(base?: DeepPartial<AppIMSetting>): AppIMSetting {
    return AppIMSetting.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<AppIMSetting>): AppIMSetting {
    const message = createBaseAppIMSetting();
    message.imType = object.imType ?? 0;
    message.appId = object.appId ?? "";
    message.appSecret = object.appSecret ?? "";
    message.externalApproval = (object.externalApproval !== undefined && object.externalApproval !== null)
      ? AppIMSetting_ExternalApproval.fromPartial(object.externalApproval)
      : undefined;
    return message;
  },
};

function createBaseAppIMSetting_ExternalApproval(): AppIMSetting_ExternalApproval {
  return { enabled: false, approvalDefinitionId: "" };
}

export const AppIMSetting_ExternalApproval = {
  encode(message: AppIMSetting_ExternalApproval, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.enabled === true) {
      writer.uint32(8).bool(message.enabled);
    }
    if (message.approvalDefinitionId !== "") {
      writer.uint32(18).string(message.approvalDefinitionId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): AppIMSetting_ExternalApproval {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAppIMSetting_ExternalApproval();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.enabled = reader.bool();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.approvalDefinitionId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): AppIMSetting_ExternalApproval {
    return {
      enabled: isSet(object.enabled) ? Boolean(object.enabled) : false,
      approvalDefinitionId: isSet(object.approvalDefinitionId) ? String(object.approvalDefinitionId) : "",
    };
  },

  toJSON(message: AppIMSetting_ExternalApproval): unknown {
    const obj: any = {};
    message.enabled !== undefined && (obj.enabled = message.enabled);
    message.approvalDefinitionId !== undefined && (obj.approvalDefinitionId = message.approvalDefinitionId);
    return obj;
  },

  create(base?: DeepPartial<AppIMSetting_ExternalApproval>): AppIMSetting_ExternalApproval {
    return AppIMSetting_ExternalApproval.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<AppIMSetting_ExternalApproval>): AppIMSetting_ExternalApproval {
    const message = createBaseAppIMSetting_ExternalApproval();
    message.enabled = object.enabled ?? false;
    message.approvalDefinitionId = object.approvalDefinitionId ?? "";
    return message;
  },
};

function createBaseAgentPluginSetting(): AgentPluginSetting {
  return { url: "", token: "" };
}

export const AgentPluginSetting = {
  encode(message: AgentPluginSetting, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.url !== "") {
      writer.uint32(10).string(message.url);
    }
    if (message.token !== "") {
      writer.uint32(18).string(message.token);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): AgentPluginSetting {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAgentPluginSetting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.url = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.token = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): AgentPluginSetting {
    return { url: isSet(object.url) ? String(object.url) : "", token: isSet(object.token) ? String(object.token) : "" };
  },

  toJSON(message: AgentPluginSetting): unknown {
    const obj: any = {};
    message.url !== undefined && (obj.url = message.url);
    message.token !== undefined && (obj.token = message.token);
    return obj;
  },

  create(base?: DeepPartial<AgentPluginSetting>): AgentPluginSetting {
    return AgentPluginSetting.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<AgentPluginSetting>): AgentPluginSetting {
    const message = createBaseAgentPluginSetting();
    message.url = object.url ?? "";
    message.token = object.token ?? "";
    return message;
  },
};

function createBaseWorkspaceProfileSetting(): WorkspaceProfileSetting {
  return {
    externalUrl: "",
    disallowSignup: false,
    require2fa: false,
    outboundIpList: [],
    gitopsWebhookUrl: "",
    tokenDuration: undefined,
    announcement: undefined,
  };
}

export const WorkspaceProfileSetting = {
  encode(message: WorkspaceProfileSetting, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.externalUrl !== "") {
      writer.uint32(10).string(message.externalUrl);
    }
    if (message.disallowSignup === true) {
      writer.uint32(16).bool(message.disallowSignup);
    }
    if (message.require2fa === true) {
      writer.uint32(24).bool(message.require2fa);
    }
    for (const v of message.outboundIpList) {
      writer.uint32(34).string(v!);
    }
    if (message.gitopsWebhookUrl !== "") {
      writer.uint32(42).string(message.gitopsWebhookUrl);
    }
    if (message.tokenDuration !== undefined) {
      Duration.encode(message.tokenDuration, writer.uint32(50).fork()).ldelim();
    }
    if (message.announcement !== undefined) {
      Announcement.encode(message.announcement, writer.uint32(58).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): WorkspaceProfileSetting {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseWorkspaceProfileSetting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.externalUrl = reader.string();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.disallowSignup = reader.bool();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.require2fa = reader.bool();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.outboundIpList.push(reader.string());
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.gitopsWebhookUrl = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.tokenDuration = Duration.decode(reader, reader.uint32());
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.announcement = Announcement.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): WorkspaceProfileSetting {
    return {
      externalUrl: isSet(object.externalUrl) ? String(object.externalUrl) : "",
      disallowSignup: isSet(object.disallowSignup) ? Boolean(object.disallowSignup) : false,
      require2fa: isSet(object.require2fa) ? Boolean(object.require2fa) : false,
      outboundIpList: Array.isArray(object?.outboundIpList) ? object.outboundIpList.map((e: any) => String(e)) : [],
      gitopsWebhookUrl: isSet(object.gitopsWebhookUrl) ? String(object.gitopsWebhookUrl) : "",
      tokenDuration: isSet(object.tokenDuration) ? Duration.fromJSON(object.tokenDuration) : undefined,
      announcement: isSet(object.announcement) ? Announcement.fromJSON(object.announcement) : undefined,
    };
  },

  toJSON(message: WorkspaceProfileSetting): unknown {
    const obj: any = {};
    message.externalUrl !== undefined && (obj.externalUrl = message.externalUrl);
    message.disallowSignup !== undefined && (obj.disallowSignup = message.disallowSignup);
    message.require2fa !== undefined && (obj.require2fa = message.require2fa);
    if (message.outboundIpList) {
      obj.outboundIpList = message.outboundIpList.map((e) => e);
    } else {
      obj.outboundIpList = [];
    }
    message.gitopsWebhookUrl !== undefined && (obj.gitopsWebhookUrl = message.gitopsWebhookUrl);
    message.tokenDuration !== undefined &&
      (obj.tokenDuration = message.tokenDuration ? Duration.toJSON(message.tokenDuration) : undefined);
    message.announcement !== undefined &&
      (obj.announcement = message.announcement ? Announcement.toJSON(message.announcement) : undefined);
    return obj;
  },

  create(base?: DeepPartial<WorkspaceProfileSetting>): WorkspaceProfileSetting {
    return WorkspaceProfileSetting.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<WorkspaceProfileSetting>): WorkspaceProfileSetting {
    const message = createBaseWorkspaceProfileSetting();
    message.externalUrl = object.externalUrl ?? "";
    message.disallowSignup = object.disallowSignup ?? false;
    message.require2fa = object.require2fa ?? false;
    message.outboundIpList = object.outboundIpList?.map((e) => e) || [];
    message.gitopsWebhookUrl = object.gitopsWebhookUrl ?? "";
    message.tokenDuration = (object.tokenDuration !== undefined && object.tokenDuration !== null)
      ? Duration.fromPartial(object.tokenDuration)
      : undefined;
    message.announcement = (object.announcement !== undefined && object.announcement !== null)
      ? Announcement.fromPartial(object.announcement)
      : undefined;
    return message;
  },
};

function createBaseAnnouncement(): Announcement {
  return { level: 0, text: "", link: "" };
}

export const Announcement = {
  encode(message: Announcement, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.level !== 0) {
      writer.uint32(8).int32(message.level);
    }
    if (message.text !== "") {
      writer.uint32(18).string(message.text);
    }
    if (message.link !== "") {
      writer.uint32(26).string(message.link);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Announcement {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAnnouncement();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.level = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.text = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.link = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Announcement {
    return {
      level: isSet(object.level) ? announcement_AlertLevelFromJSON(object.level) : 0,
      text: isSet(object.text) ? String(object.text) : "",
      link: isSet(object.link) ? String(object.link) : "",
    };
  },

  toJSON(message: Announcement): unknown {
    const obj: any = {};
    message.level !== undefined && (obj.level = announcement_AlertLevelToJSON(message.level));
    message.text !== undefined && (obj.text = message.text);
    message.link !== undefined && (obj.link = message.link);
    return obj;
  },

  create(base?: DeepPartial<Announcement>): Announcement {
    return Announcement.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<Announcement>): Announcement {
    const message = createBaseAnnouncement();
    message.level = object.level ?? 0;
    message.text = object.text ?? "";
    message.link = object.link ?? "";
    return message;
  },
};

function createBaseWorkspaceApprovalSetting(): WorkspaceApprovalSetting {
  return { rules: [] };
}

export const WorkspaceApprovalSetting = {
  encode(message: WorkspaceApprovalSetting, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.rules) {
      WorkspaceApprovalSetting_Rule.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): WorkspaceApprovalSetting {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseWorkspaceApprovalSetting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.rules.push(WorkspaceApprovalSetting_Rule.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): WorkspaceApprovalSetting {
    return {
      rules: Array.isArray(object?.rules)
        ? object.rules.map((e: any) => WorkspaceApprovalSetting_Rule.fromJSON(e))
        : [],
    };
  },

  toJSON(message: WorkspaceApprovalSetting): unknown {
    const obj: any = {};
    if (message.rules) {
      obj.rules = message.rules.map((e) => e ? WorkspaceApprovalSetting_Rule.toJSON(e) : undefined);
    } else {
      obj.rules = [];
    }
    return obj;
  },

  create(base?: DeepPartial<WorkspaceApprovalSetting>): WorkspaceApprovalSetting {
    return WorkspaceApprovalSetting.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<WorkspaceApprovalSetting>): WorkspaceApprovalSetting {
    const message = createBaseWorkspaceApprovalSetting();
    message.rules = object.rules?.map((e) => WorkspaceApprovalSetting_Rule.fromPartial(e)) || [];
    return message;
  },
};

function createBaseWorkspaceApprovalSetting_Rule(): WorkspaceApprovalSetting_Rule {
  return { template: undefined, condition: undefined };
}

export const WorkspaceApprovalSetting_Rule = {
  encode(message: WorkspaceApprovalSetting_Rule, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.template !== undefined) {
      ApprovalTemplate.encode(message.template, writer.uint32(18).fork()).ldelim();
    }
    if (message.condition !== undefined) {
      Expr.encode(message.condition, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): WorkspaceApprovalSetting_Rule {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseWorkspaceApprovalSetting_Rule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 2:
          if (tag !== 18) {
            break;
          }

          message.template = ApprovalTemplate.decode(reader, reader.uint32());
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

  fromJSON(object: any): WorkspaceApprovalSetting_Rule {
    return {
      template: isSet(object.template) ? ApprovalTemplate.fromJSON(object.template) : undefined,
      condition: isSet(object.condition) ? Expr.fromJSON(object.condition) : undefined,
    };
  },

  toJSON(message: WorkspaceApprovalSetting_Rule): unknown {
    const obj: any = {};
    message.template !== undefined &&
      (obj.template = message.template ? ApprovalTemplate.toJSON(message.template) : undefined);
    message.condition !== undefined && (obj.condition = message.condition ? Expr.toJSON(message.condition) : undefined);
    return obj;
  },

  create(base?: DeepPartial<WorkspaceApprovalSetting_Rule>): WorkspaceApprovalSetting_Rule {
    return WorkspaceApprovalSetting_Rule.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<WorkspaceApprovalSetting_Rule>): WorkspaceApprovalSetting_Rule {
    const message = createBaseWorkspaceApprovalSetting_Rule();
    message.template = (object.template !== undefined && object.template !== null)
      ? ApprovalTemplate.fromPartial(object.template)
      : undefined;
    message.condition = (object.condition !== undefined && object.condition !== null)
      ? Expr.fromPartial(object.condition)
      : undefined;
    return message;
  },
};

function createBaseExternalApprovalSetting(): ExternalApprovalSetting {
  return { nodes: [] };
}

export const ExternalApprovalSetting = {
  encode(message: ExternalApprovalSetting, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.nodes) {
      ExternalApprovalSetting_Node.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ExternalApprovalSetting {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExternalApprovalSetting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.nodes.push(ExternalApprovalSetting_Node.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ExternalApprovalSetting {
    return {
      nodes: Array.isArray(object?.nodes) ? object.nodes.map((e: any) => ExternalApprovalSetting_Node.fromJSON(e)) : [],
    };
  },

  toJSON(message: ExternalApprovalSetting): unknown {
    const obj: any = {};
    if (message.nodes) {
      obj.nodes = message.nodes.map((e) => e ? ExternalApprovalSetting_Node.toJSON(e) : undefined);
    } else {
      obj.nodes = [];
    }
    return obj;
  },

  create(base?: DeepPartial<ExternalApprovalSetting>): ExternalApprovalSetting {
    return ExternalApprovalSetting.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ExternalApprovalSetting>): ExternalApprovalSetting {
    const message = createBaseExternalApprovalSetting();
    message.nodes = object.nodes?.map((e) => ExternalApprovalSetting_Node.fromPartial(e)) || [];
    return message;
  },
};

function createBaseExternalApprovalSetting_Node(): ExternalApprovalSetting_Node {
  return { id: "", title: "", endpoint: "" };
}

export const ExternalApprovalSetting_Node = {
  encode(message: ExternalApprovalSetting_Node, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.endpoint !== "") {
      writer.uint32(26).string(message.endpoint);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ExternalApprovalSetting_Node {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExternalApprovalSetting_Node();
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

          message.title = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.endpoint = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): ExternalApprovalSetting_Node {
    return {
      id: isSet(object.id) ? String(object.id) : "",
      title: isSet(object.title) ? String(object.title) : "",
      endpoint: isSet(object.endpoint) ? String(object.endpoint) : "",
    };
  },

  toJSON(message: ExternalApprovalSetting_Node): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.title !== undefined && (obj.title = message.title);
    message.endpoint !== undefined && (obj.endpoint = message.endpoint);
    return obj;
  },

  create(base?: DeepPartial<ExternalApprovalSetting_Node>): ExternalApprovalSetting_Node {
    return ExternalApprovalSetting_Node.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<ExternalApprovalSetting_Node>): ExternalApprovalSetting_Node {
    const message = createBaseExternalApprovalSetting_Node();
    message.id = object.id ?? "";
    message.title = object.title ?? "";
    message.endpoint = object.endpoint ?? "";
    return message;
  },
};

function createBaseSchemaTemplateSetting(): SchemaTemplateSetting {
  return { fieldTemplates: [], columnTypes: [], tableTemplates: [] };
}

export const SchemaTemplateSetting = {
  encode(message: SchemaTemplateSetting, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.fieldTemplates) {
      SchemaTemplateSetting_FieldTemplate.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.columnTypes) {
      SchemaTemplateSetting_ColumnType.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.tableTemplates) {
      SchemaTemplateSetting_TableTemplate.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SchemaTemplateSetting {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSchemaTemplateSetting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.fieldTemplates.push(SchemaTemplateSetting_FieldTemplate.decode(reader, reader.uint32()));
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.columnTypes.push(SchemaTemplateSetting_ColumnType.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.tableTemplates.push(SchemaTemplateSetting_TableTemplate.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SchemaTemplateSetting {
    return {
      fieldTemplates: Array.isArray(object?.fieldTemplates)
        ? object.fieldTemplates.map((e: any) => SchemaTemplateSetting_FieldTemplate.fromJSON(e))
        : [],
      columnTypes: Array.isArray(object?.columnTypes)
        ? object.columnTypes.map((e: any) => SchemaTemplateSetting_ColumnType.fromJSON(e))
        : [],
      tableTemplates: Array.isArray(object?.tableTemplates)
        ? object.tableTemplates.map((e: any) => SchemaTemplateSetting_TableTemplate.fromJSON(e))
        : [],
    };
  },

  toJSON(message: SchemaTemplateSetting): unknown {
    const obj: any = {};
    if (message.fieldTemplates) {
      obj.fieldTemplates = message.fieldTemplates.map((e) =>
        e ? SchemaTemplateSetting_FieldTemplate.toJSON(e) : undefined
      );
    } else {
      obj.fieldTemplates = [];
    }
    if (message.columnTypes) {
      obj.columnTypes = message.columnTypes.map((e) => e ? SchemaTemplateSetting_ColumnType.toJSON(e) : undefined);
    } else {
      obj.columnTypes = [];
    }
    if (message.tableTemplates) {
      obj.tableTemplates = message.tableTemplates.map((e) =>
        e ? SchemaTemplateSetting_TableTemplate.toJSON(e) : undefined
      );
    } else {
      obj.tableTemplates = [];
    }
    return obj;
  },

  create(base?: DeepPartial<SchemaTemplateSetting>): SchemaTemplateSetting {
    return SchemaTemplateSetting.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SchemaTemplateSetting>): SchemaTemplateSetting {
    const message = createBaseSchemaTemplateSetting();
    message.fieldTemplates = object.fieldTemplates?.map((e) => SchemaTemplateSetting_FieldTemplate.fromPartial(e)) ||
      [];
    message.columnTypes = object.columnTypes?.map((e) => SchemaTemplateSetting_ColumnType.fromPartial(e)) || [];
    message.tableTemplates = object.tableTemplates?.map((e) => SchemaTemplateSetting_TableTemplate.fromPartial(e)) ||
      [];
    return message;
  },
};

function createBaseSchemaTemplateSetting_FieldTemplate(): SchemaTemplateSetting_FieldTemplate {
  return { id: "", engine: 0, category: "", column: undefined, config: undefined };
}

export const SchemaTemplateSetting_FieldTemplate = {
  encode(message: SchemaTemplateSetting_FieldTemplate, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.engine !== 0) {
      writer.uint32(16).int32(message.engine);
    }
    if (message.category !== "") {
      writer.uint32(26).string(message.category);
    }
    if (message.column !== undefined) {
      ColumnMetadata.encode(message.column, writer.uint32(34).fork()).ldelim();
    }
    if (message.config !== undefined) {
      ColumnConfig.encode(message.config, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SchemaTemplateSetting_FieldTemplate {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSchemaTemplateSetting_FieldTemplate();
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

          message.engine = reader.int32() as any;
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.category = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.column = ColumnMetadata.decode(reader, reader.uint32());
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.config = ColumnConfig.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SchemaTemplateSetting_FieldTemplate {
    return {
      id: isSet(object.id) ? String(object.id) : "",
      engine: isSet(object.engine) ? engineFromJSON(object.engine) : 0,
      category: isSet(object.category) ? String(object.category) : "",
      column: isSet(object.column) ? ColumnMetadata.fromJSON(object.column) : undefined,
      config: isSet(object.config) ? ColumnConfig.fromJSON(object.config) : undefined,
    };
  },

  toJSON(message: SchemaTemplateSetting_FieldTemplate): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.engine !== undefined && (obj.engine = engineToJSON(message.engine));
    message.category !== undefined && (obj.category = message.category);
    message.column !== undefined && (obj.column = message.column ? ColumnMetadata.toJSON(message.column) : undefined);
    message.config !== undefined && (obj.config = message.config ? ColumnConfig.toJSON(message.config) : undefined);
    return obj;
  },

  create(base?: DeepPartial<SchemaTemplateSetting_FieldTemplate>): SchemaTemplateSetting_FieldTemplate {
    return SchemaTemplateSetting_FieldTemplate.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SchemaTemplateSetting_FieldTemplate>): SchemaTemplateSetting_FieldTemplate {
    const message = createBaseSchemaTemplateSetting_FieldTemplate();
    message.id = object.id ?? "";
    message.engine = object.engine ?? 0;
    message.category = object.category ?? "";
    message.column = (object.column !== undefined && object.column !== null)
      ? ColumnMetadata.fromPartial(object.column)
      : undefined;
    message.config = (object.config !== undefined && object.config !== null)
      ? ColumnConfig.fromPartial(object.config)
      : undefined;
    return message;
  },
};

function createBaseSchemaTemplateSetting_ColumnType(): SchemaTemplateSetting_ColumnType {
  return { engine: 0, enabled: false, types: [] };
}

export const SchemaTemplateSetting_ColumnType = {
  encode(message: SchemaTemplateSetting_ColumnType, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.engine !== 0) {
      writer.uint32(8).int32(message.engine);
    }
    if (message.enabled === true) {
      writer.uint32(16).bool(message.enabled);
    }
    for (const v of message.types) {
      writer.uint32(26).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SchemaTemplateSetting_ColumnType {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSchemaTemplateSetting_ColumnType();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.engine = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.enabled = reader.bool();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.types.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SchemaTemplateSetting_ColumnType {
    return {
      engine: isSet(object.engine) ? engineFromJSON(object.engine) : 0,
      enabled: isSet(object.enabled) ? Boolean(object.enabled) : false,
      types: Array.isArray(object?.types) ? object.types.map((e: any) => String(e)) : [],
    };
  },

  toJSON(message: SchemaTemplateSetting_ColumnType): unknown {
    const obj: any = {};
    message.engine !== undefined && (obj.engine = engineToJSON(message.engine));
    message.enabled !== undefined && (obj.enabled = message.enabled);
    if (message.types) {
      obj.types = message.types.map((e) => e);
    } else {
      obj.types = [];
    }
    return obj;
  },

  create(base?: DeepPartial<SchemaTemplateSetting_ColumnType>): SchemaTemplateSetting_ColumnType {
    return SchemaTemplateSetting_ColumnType.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SchemaTemplateSetting_ColumnType>): SchemaTemplateSetting_ColumnType {
    const message = createBaseSchemaTemplateSetting_ColumnType();
    message.engine = object.engine ?? 0;
    message.enabled = object.enabled ?? false;
    message.types = object.types?.map((e) => e) || [];
    return message;
  },
};

function createBaseSchemaTemplateSetting_TableTemplate(): SchemaTemplateSetting_TableTemplate {
  return { id: "", engine: 0, category: "", table: undefined, config: undefined };
}

export const SchemaTemplateSetting_TableTemplate = {
  encode(message: SchemaTemplateSetting_TableTemplate, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.engine !== 0) {
      writer.uint32(16).int32(message.engine);
    }
    if (message.category !== "") {
      writer.uint32(26).string(message.category);
    }
    if (message.table !== undefined) {
      TableMetadata.encode(message.table, writer.uint32(34).fork()).ldelim();
    }
    if (message.config !== undefined) {
      TableConfig.encode(message.config, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SchemaTemplateSetting_TableTemplate {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSchemaTemplateSetting_TableTemplate();
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

          message.engine = reader.int32() as any;
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.category = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.table = TableMetadata.decode(reader, reader.uint32());
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.config = TableConfig.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SchemaTemplateSetting_TableTemplate {
    return {
      id: isSet(object.id) ? String(object.id) : "",
      engine: isSet(object.engine) ? engineFromJSON(object.engine) : 0,
      category: isSet(object.category) ? String(object.category) : "",
      table: isSet(object.table) ? TableMetadata.fromJSON(object.table) : undefined,
      config: isSet(object.config) ? TableConfig.fromJSON(object.config) : undefined,
    };
  },

  toJSON(message: SchemaTemplateSetting_TableTemplate): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.engine !== undefined && (obj.engine = engineToJSON(message.engine));
    message.category !== undefined && (obj.category = message.category);
    message.table !== undefined && (obj.table = message.table ? TableMetadata.toJSON(message.table) : undefined);
    message.config !== undefined && (obj.config = message.config ? TableConfig.toJSON(message.config) : undefined);
    return obj;
  },

  create(base?: DeepPartial<SchemaTemplateSetting_TableTemplate>): SchemaTemplateSetting_TableTemplate {
    return SchemaTemplateSetting_TableTemplate.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SchemaTemplateSetting_TableTemplate>): SchemaTemplateSetting_TableTemplate {
    const message = createBaseSchemaTemplateSetting_TableTemplate();
    message.id = object.id ?? "";
    message.engine = object.engine ?? 0;
    message.category = object.category ?? "";
    message.table = (object.table !== undefined && object.table !== null)
      ? TableMetadata.fromPartial(object.table)
      : undefined;
    message.config = (object.config !== undefined && object.config !== null)
      ? TableConfig.fromPartial(object.config)
      : undefined;
    return message;
  },
};

function createBaseWorkspaceTrialSetting(): WorkspaceTrialSetting {
  return { instanceCount: 0, expireTime: undefined, issuedTime: undefined, subject: "", orgName: "", plan: 0 };
}

export const WorkspaceTrialSetting = {
  encode(message: WorkspaceTrialSetting, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.instanceCount !== 0) {
      writer.uint32(8).int32(message.instanceCount);
    }
    if (message.expireTime !== undefined) {
      Timestamp.encode(toTimestamp(message.expireTime), writer.uint32(18).fork()).ldelim();
    }
    if (message.issuedTime !== undefined) {
      Timestamp.encode(toTimestamp(message.issuedTime), writer.uint32(26).fork()).ldelim();
    }
    if (message.subject !== "") {
      writer.uint32(34).string(message.subject);
    }
    if (message.orgName !== "") {
      writer.uint32(42).string(message.orgName);
    }
    if (message.plan !== 0) {
      writer.uint32(48).int32(message.plan);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): WorkspaceTrialSetting {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseWorkspaceTrialSetting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.instanceCount = reader.int32();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.expireTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.issuedTime = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.subject = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.orgName = reader.string();
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.plan = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): WorkspaceTrialSetting {
    return {
      instanceCount: isSet(object.instanceCount) ? Number(object.instanceCount) : 0,
      expireTime: isSet(object.expireTime) ? fromJsonTimestamp(object.expireTime) : undefined,
      issuedTime: isSet(object.issuedTime) ? fromJsonTimestamp(object.issuedTime) : undefined,
      subject: isSet(object.subject) ? String(object.subject) : "",
      orgName: isSet(object.orgName) ? String(object.orgName) : "",
      plan: isSet(object.plan) ? planTypeFromJSON(object.plan) : 0,
    };
  },

  toJSON(message: WorkspaceTrialSetting): unknown {
    const obj: any = {};
    message.instanceCount !== undefined && (obj.instanceCount = Math.round(message.instanceCount));
    message.expireTime !== undefined && (obj.expireTime = message.expireTime.toISOString());
    message.issuedTime !== undefined && (obj.issuedTime = message.issuedTime.toISOString());
    message.subject !== undefined && (obj.subject = message.subject);
    message.orgName !== undefined && (obj.orgName = message.orgName);
    message.plan !== undefined && (obj.plan = planTypeToJSON(message.plan));
    return obj;
  },

  create(base?: DeepPartial<WorkspaceTrialSetting>): WorkspaceTrialSetting {
    return WorkspaceTrialSetting.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<WorkspaceTrialSetting>): WorkspaceTrialSetting {
    const message = createBaseWorkspaceTrialSetting();
    message.instanceCount = object.instanceCount ?? 0;
    message.expireTime = object.expireTime ?? undefined;
    message.issuedTime = object.issuedTime ?? undefined;
    message.subject = object.subject ?? "";
    message.orgName = object.orgName ?? "";
    message.plan = object.plan ?? 0;
    return message;
  },
};

function createBaseDataClassificationSetting(): DataClassificationSetting {
  return { configs: [] };
}

export const DataClassificationSetting = {
  encode(message: DataClassificationSetting, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.configs) {
      DataClassificationSetting_DataClassificationConfig.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DataClassificationSetting {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDataClassificationSetting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.configs.push(DataClassificationSetting_DataClassificationConfig.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DataClassificationSetting {
    return {
      configs: Array.isArray(object?.configs)
        ? object.configs.map((e: any) => DataClassificationSetting_DataClassificationConfig.fromJSON(e))
        : [],
    };
  },

  toJSON(message: DataClassificationSetting): unknown {
    const obj: any = {};
    if (message.configs) {
      obj.configs = message.configs.map((e) =>
        e ? DataClassificationSetting_DataClassificationConfig.toJSON(e) : undefined
      );
    } else {
      obj.configs = [];
    }
    return obj;
  },

  create(base?: DeepPartial<DataClassificationSetting>): DataClassificationSetting {
    return DataClassificationSetting.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<DataClassificationSetting>): DataClassificationSetting {
    const message = createBaseDataClassificationSetting();
    message.configs = object.configs?.map((e) => DataClassificationSetting_DataClassificationConfig.fromPartial(e)) ||
      [];
    return message;
  },
};

function createBaseDataClassificationSetting_DataClassificationConfig(): DataClassificationSetting_DataClassificationConfig {
  return { id: "", title: "", levels: [], classification: {} };
}

export const DataClassificationSetting_DataClassificationConfig = {
  encode(
    message: DataClassificationSetting_DataClassificationConfig,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    for (const v of message.levels) {
      DataClassificationSetting_DataClassificationConfig_Level.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    Object.entries(message.classification).forEach(([key, value]) => {
      DataClassificationSetting_DataClassificationConfig_ClassificationEntry.encode(
        { key: key as any, value },
        writer.uint32(34).fork(),
      ).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DataClassificationSetting_DataClassificationConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDataClassificationSetting_DataClassificationConfig();
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

          message.title = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.levels.push(DataClassificationSetting_DataClassificationConfig_Level.decode(reader, reader.uint32()));
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          const entry4 = DataClassificationSetting_DataClassificationConfig_ClassificationEntry.decode(
            reader,
            reader.uint32(),
          );
          if (entry4.value !== undefined) {
            message.classification[entry4.key] = entry4.value;
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

  fromJSON(object: any): DataClassificationSetting_DataClassificationConfig {
    return {
      id: isSet(object.id) ? String(object.id) : "",
      title: isSet(object.title) ? String(object.title) : "",
      levels: Array.isArray(object?.levels)
        ? object.levels.map((e: any) => DataClassificationSetting_DataClassificationConfig_Level.fromJSON(e))
        : [],
      classification: isObject(object.classification)
        ? Object.entries(object.classification).reduce<
          { [key: string]: DataClassificationSetting_DataClassificationConfig_DataClassification }
        >((acc, [key, value]) => {
          acc[key] = DataClassificationSetting_DataClassificationConfig_DataClassification.fromJSON(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: DataClassificationSetting_DataClassificationConfig): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.title !== undefined && (obj.title = message.title);
    if (message.levels) {
      obj.levels = message.levels.map((e) =>
        e ? DataClassificationSetting_DataClassificationConfig_Level.toJSON(e) : undefined
      );
    } else {
      obj.levels = [];
    }
    obj.classification = {};
    if (message.classification) {
      Object.entries(message.classification).forEach(([k, v]) => {
        obj.classification[k] = DataClassificationSetting_DataClassificationConfig_DataClassification.toJSON(v);
      });
    }
    return obj;
  },

  create(
    base?: DeepPartial<DataClassificationSetting_DataClassificationConfig>,
  ): DataClassificationSetting_DataClassificationConfig {
    return DataClassificationSetting_DataClassificationConfig.fromPartial(base ?? {});
  },

  fromPartial(
    object: DeepPartial<DataClassificationSetting_DataClassificationConfig>,
  ): DataClassificationSetting_DataClassificationConfig {
    const message = createBaseDataClassificationSetting_DataClassificationConfig();
    message.id = object.id ?? "";
    message.title = object.title ?? "";
    message.levels =
      object.levels?.map((e) => DataClassificationSetting_DataClassificationConfig_Level.fromPartial(e)) || [];
    message.classification = Object.entries(object.classification ?? {}).reduce<
      { [key: string]: DataClassificationSetting_DataClassificationConfig_DataClassification }
    >((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = DataClassificationSetting_DataClassificationConfig_DataClassification.fromPartial(value);
      }
      return acc;
    }, {});
    return message;
  },
};

function createBaseDataClassificationSetting_DataClassificationConfig_Level(): DataClassificationSetting_DataClassificationConfig_Level {
  return { id: "", title: "", description: "" };
}

export const DataClassificationSetting_DataClassificationConfig_Level = {
  encode(
    message: DataClassificationSetting_DataClassificationConfig_Level,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DataClassificationSetting_DataClassificationConfig_Level {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDataClassificationSetting_DataClassificationConfig_Level();
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

          message.title = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.description = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DataClassificationSetting_DataClassificationConfig_Level {
    return {
      id: isSet(object.id) ? String(object.id) : "",
      title: isSet(object.title) ? String(object.title) : "",
      description: isSet(object.description) ? String(object.description) : "",
    };
  },

  toJSON(message: DataClassificationSetting_DataClassificationConfig_Level): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.title !== undefined && (obj.title = message.title);
    message.description !== undefined && (obj.description = message.description);
    return obj;
  },

  create(
    base?: DeepPartial<DataClassificationSetting_DataClassificationConfig_Level>,
  ): DataClassificationSetting_DataClassificationConfig_Level {
    return DataClassificationSetting_DataClassificationConfig_Level.fromPartial(base ?? {});
  },

  fromPartial(
    object: DeepPartial<DataClassificationSetting_DataClassificationConfig_Level>,
  ): DataClassificationSetting_DataClassificationConfig_Level {
    const message = createBaseDataClassificationSetting_DataClassificationConfig_Level();
    message.id = object.id ?? "";
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    return message;
  },
};

function createBaseDataClassificationSetting_DataClassificationConfig_DataClassification(): DataClassificationSetting_DataClassificationConfig_DataClassification {
  return { id: "", title: "", description: "", levelId: undefined };
}

export const DataClassificationSetting_DataClassificationConfig_DataClassification = {
  encode(
    message: DataClassificationSetting_DataClassificationConfig_DataClassification,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    if (message.levelId !== undefined) {
      writer.uint32(34).string(message.levelId);
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): DataClassificationSetting_DataClassificationConfig_DataClassification {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDataClassificationSetting_DataClassificationConfig_DataClassification();
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

          message.title = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.description = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.levelId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DataClassificationSetting_DataClassificationConfig_DataClassification {
    return {
      id: isSet(object.id) ? String(object.id) : "",
      title: isSet(object.title) ? String(object.title) : "",
      description: isSet(object.description) ? String(object.description) : "",
      levelId: isSet(object.levelId) ? String(object.levelId) : undefined,
    };
  },

  toJSON(message: DataClassificationSetting_DataClassificationConfig_DataClassification): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.title !== undefined && (obj.title = message.title);
    message.description !== undefined && (obj.description = message.description);
    message.levelId !== undefined && (obj.levelId = message.levelId);
    return obj;
  },

  create(
    base?: DeepPartial<DataClassificationSetting_DataClassificationConfig_DataClassification>,
  ): DataClassificationSetting_DataClassificationConfig_DataClassification {
    return DataClassificationSetting_DataClassificationConfig_DataClassification.fromPartial(base ?? {});
  },

  fromPartial(
    object: DeepPartial<DataClassificationSetting_DataClassificationConfig_DataClassification>,
  ): DataClassificationSetting_DataClassificationConfig_DataClassification {
    const message = createBaseDataClassificationSetting_DataClassificationConfig_DataClassification();
    message.id = object.id ?? "";
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.levelId = object.levelId ?? undefined;
    return message;
  },
};

function createBaseDataClassificationSetting_DataClassificationConfig_ClassificationEntry(): DataClassificationSetting_DataClassificationConfig_ClassificationEntry {
  return { key: "", value: undefined };
}

export const DataClassificationSetting_DataClassificationConfig_ClassificationEntry = {
  encode(
    message: DataClassificationSetting_DataClassificationConfig_ClassificationEntry,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      DataClassificationSetting_DataClassificationConfig_DataClassification.encode(
        message.value,
        writer.uint32(18).fork(),
      ).ldelim();
    }
    return writer;
  },

  decode(
    input: _m0.Reader | Uint8Array,
    length?: number,
  ): DataClassificationSetting_DataClassificationConfig_ClassificationEntry {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDataClassificationSetting_DataClassificationConfig_ClassificationEntry();
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

          message.value = DataClassificationSetting_DataClassificationConfig_DataClassification.decode(
            reader,
            reader.uint32(),
          );
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DataClassificationSetting_DataClassificationConfig_ClassificationEntry {
    return {
      key: isSet(object.key) ? String(object.key) : "",
      value: isSet(object.value)
        ? DataClassificationSetting_DataClassificationConfig_DataClassification.fromJSON(object.value)
        : undefined,
    };
  },

  toJSON(message: DataClassificationSetting_DataClassificationConfig_ClassificationEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = message.key);
    message.value !== undefined && (obj.value = message.value
      ? DataClassificationSetting_DataClassificationConfig_DataClassification.toJSON(message.value)
      : undefined);
    return obj;
  },

  create(
    base?: DeepPartial<DataClassificationSetting_DataClassificationConfig_ClassificationEntry>,
  ): DataClassificationSetting_DataClassificationConfig_ClassificationEntry {
    return DataClassificationSetting_DataClassificationConfig_ClassificationEntry.fromPartial(base ?? {});
  },

  fromPartial(
    object: DeepPartial<DataClassificationSetting_DataClassificationConfig_ClassificationEntry>,
  ): DataClassificationSetting_DataClassificationConfig_ClassificationEntry {
    const message = createBaseDataClassificationSetting_DataClassificationConfig_ClassificationEntry();
    message.key = object.key ?? "";
    message.value = (object.value !== undefined && object.value !== null)
      ? DataClassificationSetting_DataClassificationConfig_DataClassification.fromPartial(object.value)
      : undefined;
    return message;
  },
};

function createBaseSemanticTypeSetting(): SemanticTypeSetting {
  return { types: [] };
}

export const SemanticTypeSetting = {
  encode(message: SemanticTypeSetting, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.types) {
      SemanticTypeSetting_SemanticType.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SemanticTypeSetting {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSemanticTypeSetting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.types.push(SemanticTypeSetting_SemanticType.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SemanticTypeSetting {
    return {
      types: Array.isArray(object?.types)
        ? object.types.map((e: any) => SemanticTypeSetting_SemanticType.fromJSON(e))
        : [],
    };
  },

  toJSON(message: SemanticTypeSetting): unknown {
    const obj: any = {};
    if (message.types) {
      obj.types = message.types.map((e) => e ? SemanticTypeSetting_SemanticType.toJSON(e) : undefined);
    } else {
      obj.types = [];
    }
    return obj;
  },

  create(base?: DeepPartial<SemanticTypeSetting>): SemanticTypeSetting {
    return SemanticTypeSetting.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SemanticTypeSetting>): SemanticTypeSetting {
    const message = createBaseSemanticTypeSetting();
    message.types = object.types?.map((e) => SemanticTypeSetting_SemanticType.fromPartial(e)) || [];
    return message;
  },
};

function createBaseSemanticTypeSetting_SemanticType(): SemanticTypeSetting_SemanticType {
  return { id: "", title: "", description: "", partialMaskAlgorithmId: "", fullMaskAlgorithmId: "" };
}

export const SemanticTypeSetting_SemanticType = {
  encode(message: SemanticTypeSetting_SemanticType, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    if (message.partialMaskAlgorithmId !== "") {
      writer.uint32(34).string(message.partialMaskAlgorithmId);
    }
    if (message.fullMaskAlgorithmId !== "") {
      writer.uint32(42).string(message.fullMaskAlgorithmId);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SemanticTypeSetting_SemanticType {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSemanticTypeSetting_SemanticType();
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

          message.title = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.description = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.partialMaskAlgorithmId = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.fullMaskAlgorithmId = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SemanticTypeSetting_SemanticType {
    return {
      id: isSet(object.id) ? String(object.id) : "",
      title: isSet(object.title) ? String(object.title) : "",
      description: isSet(object.description) ? String(object.description) : "",
      partialMaskAlgorithmId: isSet(object.partialMaskAlgorithmId) ? String(object.partialMaskAlgorithmId) : "",
      fullMaskAlgorithmId: isSet(object.fullMaskAlgorithmId) ? String(object.fullMaskAlgorithmId) : "",
    };
  },

  toJSON(message: SemanticTypeSetting_SemanticType): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.title !== undefined && (obj.title = message.title);
    message.description !== undefined && (obj.description = message.description);
    message.partialMaskAlgorithmId !== undefined && (obj.partialMaskAlgorithmId = message.partialMaskAlgorithmId);
    message.fullMaskAlgorithmId !== undefined && (obj.fullMaskAlgorithmId = message.fullMaskAlgorithmId);
    return obj;
  },

  create(base?: DeepPartial<SemanticTypeSetting_SemanticType>): SemanticTypeSetting_SemanticType {
    return SemanticTypeSetting_SemanticType.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SemanticTypeSetting_SemanticType>): SemanticTypeSetting_SemanticType {
    const message = createBaseSemanticTypeSetting_SemanticType();
    message.id = object.id ?? "";
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.partialMaskAlgorithmId = object.partialMaskAlgorithmId ?? "";
    message.fullMaskAlgorithmId = object.fullMaskAlgorithmId ?? "";
    return message;
  },
};

function createBaseMaskingAlgorithmSetting(): MaskingAlgorithmSetting {
  return { algorithms: [] };
}

export const MaskingAlgorithmSetting = {
  encode(message: MaskingAlgorithmSetting, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.algorithms) {
      MaskingAlgorithmSetting_Algorithm.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MaskingAlgorithmSetting {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMaskingAlgorithmSetting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.algorithms.push(MaskingAlgorithmSetting_Algorithm.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MaskingAlgorithmSetting {
    return {
      algorithms: Array.isArray(object?.algorithms)
        ? object.algorithms.map((e: any) => MaskingAlgorithmSetting_Algorithm.fromJSON(e))
        : [],
    };
  },

  toJSON(message: MaskingAlgorithmSetting): unknown {
    const obj: any = {};
    if (message.algorithms) {
      obj.algorithms = message.algorithms.map((e) => e ? MaskingAlgorithmSetting_Algorithm.toJSON(e) : undefined);
    } else {
      obj.algorithms = [];
    }
    return obj;
  },

  create(base?: DeepPartial<MaskingAlgorithmSetting>): MaskingAlgorithmSetting {
    return MaskingAlgorithmSetting.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<MaskingAlgorithmSetting>): MaskingAlgorithmSetting {
    const message = createBaseMaskingAlgorithmSetting();
    message.algorithms = object.algorithms?.map((e) => MaskingAlgorithmSetting_Algorithm.fromPartial(e)) || [];
    return message;
  },
};

function createBaseMaskingAlgorithmSetting_Algorithm(): MaskingAlgorithmSetting_Algorithm {
  return {
    id: "",
    title: "",
    description: "",
    category: "",
    fullMask: undefined,
    rangeMask: undefined,
    md5Mask: undefined,
  };
}

export const MaskingAlgorithmSetting_Algorithm = {
  encode(message: MaskingAlgorithmSetting_Algorithm, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    if (message.category !== "") {
      writer.uint32(34).string(message.category);
    }
    if (message.fullMask !== undefined) {
      MaskingAlgorithmSetting_Algorithm_FullMask.encode(message.fullMask, writer.uint32(42).fork()).ldelim();
    }
    if (message.rangeMask !== undefined) {
      MaskingAlgorithmSetting_Algorithm_RangeMask.encode(message.rangeMask, writer.uint32(50).fork()).ldelim();
    }
    if (message.md5Mask !== undefined) {
      MaskingAlgorithmSetting_Algorithm_MD5Mask.encode(message.md5Mask, writer.uint32(58).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MaskingAlgorithmSetting_Algorithm {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMaskingAlgorithmSetting_Algorithm();
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

          message.title = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.description = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.category = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.fullMask = MaskingAlgorithmSetting_Algorithm_FullMask.decode(reader, reader.uint32());
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.rangeMask = MaskingAlgorithmSetting_Algorithm_RangeMask.decode(reader, reader.uint32());
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.md5Mask = MaskingAlgorithmSetting_Algorithm_MD5Mask.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MaskingAlgorithmSetting_Algorithm {
    return {
      id: isSet(object.id) ? String(object.id) : "",
      title: isSet(object.title) ? String(object.title) : "",
      description: isSet(object.description) ? String(object.description) : "",
      category: isSet(object.category) ? String(object.category) : "",
      fullMask: isSet(object.fullMask)
        ? MaskingAlgorithmSetting_Algorithm_FullMask.fromJSON(object.fullMask)
        : undefined,
      rangeMask: isSet(object.rangeMask)
        ? MaskingAlgorithmSetting_Algorithm_RangeMask.fromJSON(object.rangeMask)
        : undefined,
      md5Mask: isSet(object.md5Mask) ? MaskingAlgorithmSetting_Algorithm_MD5Mask.fromJSON(object.md5Mask) : undefined,
    };
  },

  toJSON(message: MaskingAlgorithmSetting_Algorithm): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.title !== undefined && (obj.title = message.title);
    message.description !== undefined && (obj.description = message.description);
    message.category !== undefined && (obj.category = message.category);
    message.fullMask !== undefined &&
      (obj.fullMask = message.fullMask
        ? MaskingAlgorithmSetting_Algorithm_FullMask.toJSON(message.fullMask)
        : undefined);
    message.rangeMask !== undefined && (obj.rangeMask = message.rangeMask
      ? MaskingAlgorithmSetting_Algorithm_RangeMask.toJSON(message.rangeMask)
      : undefined);
    message.md5Mask !== undefined &&
      (obj.md5Mask = message.md5Mask ? MaskingAlgorithmSetting_Algorithm_MD5Mask.toJSON(message.md5Mask) : undefined);
    return obj;
  },

  create(base?: DeepPartial<MaskingAlgorithmSetting_Algorithm>): MaskingAlgorithmSetting_Algorithm {
    return MaskingAlgorithmSetting_Algorithm.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<MaskingAlgorithmSetting_Algorithm>): MaskingAlgorithmSetting_Algorithm {
    const message = createBaseMaskingAlgorithmSetting_Algorithm();
    message.id = object.id ?? "";
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.category = object.category ?? "";
    message.fullMask = (object.fullMask !== undefined && object.fullMask !== null)
      ? MaskingAlgorithmSetting_Algorithm_FullMask.fromPartial(object.fullMask)
      : undefined;
    message.rangeMask = (object.rangeMask !== undefined && object.rangeMask !== null)
      ? MaskingAlgorithmSetting_Algorithm_RangeMask.fromPartial(object.rangeMask)
      : undefined;
    message.md5Mask = (object.md5Mask !== undefined && object.md5Mask !== null)
      ? MaskingAlgorithmSetting_Algorithm_MD5Mask.fromPartial(object.md5Mask)
      : undefined;
    return message;
  },
};

function createBaseMaskingAlgorithmSetting_Algorithm_FullMask(): MaskingAlgorithmSetting_Algorithm_FullMask {
  return { substitution: "" };
}

export const MaskingAlgorithmSetting_Algorithm_FullMask = {
  encode(message: MaskingAlgorithmSetting_Algorithm_FullMask, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.substitution !== "") {
      writer.uint32(10).string(message.substitution);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MaskingAlgorithmSetting_Algorithm_FullMask {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMaskingAlgorithmSetting_Algorithm_FullMask();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.substitution = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MaskingAlgorithmSetting_Algorithm_FullMask {
    return { substitution: isSet(object.substitution) ? String(object.substitution) : "" };
  },

  toJSON(message: MaskingAlgorithmSetting_Algorithm_FullMask): unknown {
    const obj: any = {};
    message.substitution !== undefined && (obj.substitution = message.substitution);
    return obj;
  },

  create(base?: DeepPartial<MaskingAlgorithmSetting_Algorithm_FullMask>): MaskingAlgorithmSetting_Algorithm_FullMask {
    return MaskingAlgorithmSetting_Algorithm_FullMask.fromPartial(base ?? {});
  },

  fromPartial(
    object: DeepPartial<MaskingAlgorithmSetting_Algorithm_FullMask>,
  ): MaskingAlgorithmSetting_Algorithm_FullMask {
    const message = createBaseMaskingAlgorithmSetting_Algorithm_FullMask();
    message.substitution = object.substitution ?? "";
    return message;
  },
};

function createBaseMaskingAlgorithmSetting_Algorithm_RangeMask(): MaskingAlgorithmSetting_Algorithm_RangeMask {
  return { slices: [] };
}

export const MaskingAlgorithmSetting_Algorithm_RangeMask = {
  encode(message: MaskingAlgorithmSetting_Algorithm_RangeMask, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.slices) {
      MaskingAlgorithmSetting_Algorithm_RangeMask_Slice.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MaskingAlgorithmSetting_Algorithm_RangeMask {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMaskingAlgorithmSetting_Algorithm_RangeMask();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.slices.push(MaskingAlgorithmSetting_Algorithm_RangeMask_Slice.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MaskingAlgorithmSetting_Algorithm_RangeMask {
    return {
      slices: Array.isArray(object?.slices)
        ? object.slices.map((e: any) => MaskingAlgorithmSetting_Algorithm_RangeMask_Slice.fromJSON(e))
        : [],
    };
  },

  toJSON(message: MaskingAlgorithmSetting_Algorithm_RangeMask): unknown {
    const obj: any = {};
    if (message.slices) {
      obj.slices = message.slices.map((e) =>
        e ? MaskingAlgorithmSetting_Algorithm_RangeMask_Slice.toJSON(e) : undefined
      );
    } else {
      obj.slices = [];
    }
    return obj;
  },

  create(base?: DeepPartial<MaskingAlgorithmSetting_Algorithm_RangeMask>): MaskingAlgorithmSetting_Algorithm_RangeMask {
    return MaskingAlgorithmSetting_Algorithm_RangeMask.fromPartial(base ?? {});
  },

  fromPartial(
    object: DeepPartial<MaskingAlgorithmSetting_Algorithm_RangeMask>,
  ): MaskingAlgorithmSetting_Algorithm_RangeMask {
    const message = createBaseMaskingAlgorithmSetting_Algorithm_RangeMask();
    message.slices = object.slices?.map((e) => MaskingAlgorithmSetting_Algorithm_RangeMask_Slice.fromPartial(e)) || [];
    return message;
  },
};

function createBaseMaskingAlgorithmSetting_Algorithm_RangeMask_Slice(): MaskingAlgorithmSetting_Algorithm_RangeMask_Slice {
  return { start: 0, end: 0, substitution: "" };
}

export const MaskingAlgorithmSetting_Algorithm_RangeMask_Slice = {
  encode(
    message: MaskingAlgorithmSetting_Algorithm_RangeMask_Slice,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.start !== 0) {
      writer.uint32(8).int32(message.start);
    }
    if (message.end !== 0) {
      writer.uint32(16).int32(message.end);
    }
    if (message.substitution !== "") {
      writer.uint32(26).string(message.substitution);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MaskingAlgorithmSetting_Algorithm_RangeMask_Slice {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMaskingAlgorithmSetting_Algorithm_RangeMask_Slice();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.start = reader.int32();
          continue;
        case 2:
          if (tag !== 16) {
            break;
          }

          message.end = reader.int32();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.substitution = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MaskingAlgorithmSetting_Algorithm_RangeMask_Slice {
    return {
      start: isSet(object.start) ? Number(object.start) : 0,
      end: isSet(object.end) ? Number(object.end) : 0,
      substitution: isSet(object.substitution) ? String(object.substitution) : "",
    };
  },

  toJSON(message: MaskingAlgorithmSetting_Algorithm_RangeMask_Slice): unknown {
    const obj: any = {};
    message.start !== undefined && (obj.start = Math.round(message.start));
    message.end !== undefined && (obj.end = Math.round(message.end));
    message.substitution !== undefined && (obj.substitution = message.substitution);
    return obj;
  },

  create(
    base?: DeepPartial<MaskingAlgorithmSetting_Algorithm_RangeMask_Slice>,
  ): MaskingAlgorithmSetting_Algorithm_RangeMask_Slice {
    return MaskingAlgorithmSetting_Algorithm_RangeMask_Slice.fromPartial(base ?? {});
  },

  fromPartial(
    object: DeepPartial<MaskingAlgorithmSetting_Algorithm_RangeMask_Slice>,
  ): MaskingAlgorithmSetting_Algorithm_RangeMask_Slice {
    const message = createBaseMaskingAlgorithmSetting_Algorithm_RangeMask_Slice();
    message.start = object.start ?? 0;
    message.end = object.end ?? 0;
    message.substitution = object.substitution ?? "";
    return message;
  },
};

function createBaseMaskingAlgorithmSetting_Algorithm_MD5Mask(): MaskingAlgorithmSetting_Algorithm_MD5Mask {
  return { salt: "" };
}

export const MaskingAlgorithmSetting_Algorithm_MD5Mask = {
  encode(message: MaskingAlgorithmSetting_Algorithm_MD5Mask, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.salt !== "") {
      writer.uint32(10).string(message.salt);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): MaskingAlgorithmSetting_Algorithm_MD5Mask {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMaskingAlgorithmSetting_Algorithm_MD5Mask();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.salt = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): MaskingAlgorithmSetting_Algorithm_MD5Mask {
    return { salt: isSet(object.salt) ? String(object.salt) : "" };
  },

  toJSON(message: MaskingAlgorithmSetting_Algorithm_MD5Mask): unknown {
    const obj: any = {};
    message.salt !== undefined && (obj.salt = message.salt);
    return obj;
  },

  create(base?: DeepPartial<MaskingAlgorithmSetting_Algorithm_MD5Mask>): MaskingAlgorithmSetting_Algorithm_MD5Mask {
    return MaskingAlgorithmSetting_Algorithm_MD5Mask.fromPartial(base ?? {});
  },

  fromPartial(
    object: DeepPartial<MaskingAlgorithmSetting_Algorithm_MD5Mask>,
  ): MaskingAlgorithmSetting_Algorithm_MD5Mask {
    const message = createBaseMaskingAlgorithmSetting_Algorithm_MD5Mask();
    message.salt = object.salt ?? "";
    return message;
  },
};

export type SettingServiceDefinition = typeof SettingServiceDefinition;
export const SettingServiceDefinition = {
  name: "SettingService",
  fullName: "bytebase.v1.SettingService",
  methods: {
    listSettings: {
      name: "ListSettings",
      requestType: ListSettingsRequest,
      requestStream: false,
      responseType: ListSettingsResponse,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([0])],
          578365826: [new Uint8Array([14, 18, 12, 47, 118, 49, 47, 115, 101, 116, 116, 105, 110, 103, 115])],
        },
      },
    },
    getSetting: {
      name: "GetSetting",
      requestType: GetSettingRequest,
      requestStream: false,
      responseType: Setting,
      responseStream: false,
      options: {
        _unknownFields: {
          8410: [new Uint8Array([4, 110, 97, 109, 101])],
          578365826: [
            new Uint8Array([
              23,
              18,
              21,
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
              115,
              101,
              116,
              116,
              105,
              110,
              103,
              115,
              47,
              42,
              125,
            ]),
          ],
        },
      },
    },
    setSetting: {
      name: "SetSetting",
      requestType: SetSettingRequest,
      requestStream: false,
      responseType: Setting,
      responseStream: false,
      options: {
        _unknownFields: {
          578365826: [
            new Uint8Array([
              40,
              58,
              7,
              115,
              101,
              116,
              116,
              105,
              110,
              103,
              50,
              29,
              47,
              118,
              49,
              47,
              123,
              115,
              101,
              116,
              116,
              105,
              110,
              103,
              46,
              110,
              97,
              109,
              101,
              61,
              115,
              101,
              116,
              116,
              105,
              110,
              103,
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

function isObject(value: any): boolean {
  return typeof value === "object" && value !== null;
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
