/* eslint-disable */
import type { CallContext, CallOptions } from "nice-grpc-common";
import * as _m0 from "protobufjs/minimal";
import { Duration } from "../google/protobuf/duration";
import { Timestamp } from "../google/protobuf/timestamp";
import { Expr } from "../google/type/expr";
import { Engine, engineFromJSON, engineToJSON } from "./common";
import { ColumnMetadata } from "./database_service";
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
  setting?: Setting | undefined;
}

/** The request message for updating a setting. */
export interface SetSettingRequest {
  /** The setting to update. */
  setting?:
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
  value?: Value | undefined;
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
  externalApproval?: AppIMSetting_ExternalApproval | undefined;
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
  /** The duration for refresh token. */
  refreshTokenDuration?: Duration | undefined;
}

export interface WorkspaceApprovalSetting {
  rules: WorkspaceApprovalSetting_Rule[];
}

export interface WorkspaceApprovalSetting_Rule {
  template?: ApprovalTemplate | undefined;
  condition?: Expr | undefined;
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
}

export interface SchemaTemplateSetting_FieldTemplate {
  id: string;
  engine: Engine;
  category: string;
  column?: ColumnMetadata | undefined;
}

export interface SchemaTemplateSetting_ColumnType {
  engine: Engine;
  enabled: boolean;
  types: string[];
}

export interface WorkspaceTrialSetting {
  instanceCount: number;
  expireTime?: Date | undefined;
  issuedTime?: Date | undefined;
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
  value?: DataClassificationSetting_DataClassificationConfig_DataClassification | undefined;
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
    refreshTokenDuration: undefined,
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
    if (message.refreshTokenDuration !== undefined) {
      Duration.encode(message.refreshTokenDuration, writer.uint32(50).fork()).ldelim();
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

          message.refreshTokenDuration = Duration.decode(reader, reader.uint32());
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
      refreshTokenDuration: isSet(object.refreshTokenDuration)
        ? Duration.fromJSON(object.refreshTokenDuration)
        : undefined,
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
    message.refreshTokenDuration !== undefined && (obj.refreshTokenDuration = message.refreshTokenDuration
      ? Duration.toJSON(message.refreshTokenDuration)
      : undefined);
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
    message.refreshTokenDuration = (object.refreshTokenDuration !== undefined && object.refreshTokenDuration !== null)
      ? Duration.fromPartial(object.refreshTokenDuration)
      : undefined;
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
  return { fieldTemplates: [], columnTypes: [] };
}

export const SchemaTemplateSetting = {
  encode(message: SchemaTemplateSetting, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.fieldTemplates) {
      SchemaTemplateSetting_FieldTemplate.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.columnTypes) {
      SchemaTemplateSetting_ColumnType.encode(v!, writer.uint32(18).fork()).ldelim();
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
    return message;
  },
};

function createBaseSchemaTemplateSetting_FieldTemplate(): SchemaTemplateSetting_FieldTemplate {
  return { id: "", engine: 0, category: "", column: undefined };
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
    };
  },

  toJSON(message: SchemaTemplateSetting_FieldTemplate): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.engine !== undefined && (obj.engine = engineToJSON(message.engine));
    message.category !== undefined && (obj.category = message.category);
    message.column !== undefined && (obj.column = message.column ? ColumnMetadata.toJSON(message.column) : undefined);
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

export interface SettingServiceImplementation<CallContextExt = {}> {
  listSettings(
    request: ListSettingsRequest,
    context: CallContext & CallContextExt,
  ): Promise<DeepPartial<ListSettingsResponse>>;
  getSetting(request: GetSettingRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Setting>>;
  setSetting(request: SetSettingRequest, context: CallContext & CallContextExt): Promise<DeepPartial<Setting>>;
}

export interface SettingServiceClient<CallOptionsExt = {}> {
  listSettings(
    request: DeepPartial<ListSettingsRequest>,
    options?: CallOptions & CallOptionsExt,
  ): Promise<ListSettingsResponse>;
  getSetting(request: DeepPartial<GetSettingRequest>, options?: CallOptions & CallOptionsExt): Promise<Setting>;
  setSetting(request: DeepPartial<SetSettingRequest>, options?: CallOptions & CallOptionsExt): Promise<Setting>;
}

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
