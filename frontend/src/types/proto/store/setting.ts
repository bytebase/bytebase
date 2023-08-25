/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { ParsedExpr } from "../google/api/expr/v1alpha1/syntax";
import { Duration } from "../google/protobuf/duration";
import { Expr } from "../google/type/expr";
import { ApprovalTemplate } from "./approval";
import { Engine, engineFromJSON, engineToJSON } from "./common";
import { ColumnMetadata } from "./database";

export const protobufPackage = "bytebase.store";

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

export interface AgentPluginSetting {
  /** The URL for the agent API. */
  url: string;
  /** The token for the agent. */
  token: string;
}

export interface WorkspaceApprovalSetting {
  rules: WorkspaceApprovalSetting_Rule[];
}

export interface WorkspaceApprovalSetting_Rule {
  expression?: ParsedExpr | undefined;
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

export interface SMTPMailDeliverySetting {
  /** The SMTP server address. */
  server: string;
  /** The SMTP server port. */
  port: number;
  /** The SMTP server encryption. */
  encryption: SMTPMailDeliverySetting_Encryption;
  /** The CA, KEY, and CERT for the SMTP server. */
  ca: string;
  key: string;
  cert: string;
  authentication: SMTPMailDeliverySetting_Authentication;
  username: string;
  password: string;
  /** The sender email address. */
  from: string;
}

/** We support three types of SMTP encryption: NONE, STARTTLS, and SSL/TLS. */
export enum SMTPMailDeliverySetting_Encryption {
  ENCRYPTION_UNSPECIFIED = 0,
  ENCRYPTION_NONE = 1,
  ENCRYPTION_STARTTLS = 2,
  ENCRYPTION_SSL_TLS = 3,
  UNRECOGNIZED = -1,
}

export function sMTPMailDeliverySetting_EncryptionFromJSON(object: any): SMTPMailDeliverySetting_Encryption {
  switch (object) {
    case 0:
    case "ENCRYPTION_UNSPECIFIED":
      return SMTPMailDeliverySetting_Encryption.ENCRYPTION_UNSPECIFIED;
    case 1:
    case "ENCRYPTION_NONE":
      return SMTPMailDeliverySetting_Encryption.ENCRYPTION_NONE;
    case 2:
    case "ENCRYPTION_STARTTLS":
      return SMTPMailDeliverySetting_Encryption.ENCRYPTION_STARTTLS;
    case 3:
    case "ENCRYPTION_SSL_TLS":
      return SMTPMailDeliverySetting_Encryption.ENCRYPTION_SSL_TLS;
    case -1:
    case "UNRECOGNIZED":
    default:
      return SMTPMailDeliverySetting_Encryption.UNRECOGNIZED;
  }
}

export function sMTPMailDeliverySetting_EncryptionToJSON(object: SMTPMailDeliverySetting_Encryption): string {
  switch (object) {
    case SMTPMailDeliverySetting_Encryption.ENCRYPTION_UNSPECIFIED:
      return "ENCRYPTION_UNSPECIFIED";
    case SMTPMailDeliverySetting_Encryption.ENCRYPTION_NONE:
      return "ENCRYPTION_NONE";
    case SMTPMailDeliverySetting_Encryption.ENCRYPTION_STARTTLS:
      return "ENCRYPTION_STARTTLS";
    case SMTPMailDeliverySetting_Encryption.ENCRYPTION_SSL_TLS:
      return "ENCRYPTION_SSL_TLS";
    case SMTPMailDeliverySetting_Encryption.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

/** We support four types of SMTP authentication: NONE, PLAIN, LOGIN, and CRAM-MD5. */
export enum SMTPMailDeliverySetting_Authentication {
  AUTHENTICATION_UNSPECIFIED = 0,
  AUTHENTICATION_NONE = 1,
  AUTHENTICATION_PLAIN = 2,
  AUTHENTICATION_LOGIN = 3,
  AUTHENTICATION_CRAM_MD5 = 4,
  UNRECOGNIZED = -1,
}

export function sMTPMailDeliverySetting_AuthenticationFromJSON(object: any): SMTPMailDeliverySetting_Authentication {
  switch (object) {
    case 0:
    case "AUTHENTICATION_UNSPECIFIED":
      return SMTPMailDeliverySetting_Authentication.AUTHENTICATION_UNSPECIFIED;
    case 1:
    case "AUTHENTICATION_NONE":
      return SMTPMailDeliverySetting_Authentication.AUTHENTICATION_NONE;
    case 2:
    case "AUTHENTICATION_PLAIN":
      return SMTPMailDeliverySetting_Authentication.AUTHENTICATION_PLAIN;
    case 3:
    case "AUTHENTICATION_LOGIN":
      return SMTPMailDeliverySetting_Authentication.AUTHENTICATION_LOGIN;
    case 4:
    case "AUTHENTICATION_CRAM_MD5":
      return SMTPMailDeliverySetting_Authentication.AUTHENTICATION_CRAM_MD5;
    case -1:
    case "UNRECOGNIZED":
    default:
      return SMTPMailDeliverySetting_Authentication.UNRECOGNIZED;
  }
}

export function sMTPMailDeliverySetting_AuthenticationToJSON(object: SMTPMailDeliverySetting_Authentication): string {
  switch (object) {
    case SMTPMailDeliverySetting_Authentication.AUTHENTICATION_UNSPECIFIED:
      return "AUTHENTICATION_UNSPECIFIED";
    case SMTPMailDeliverySetting_Authentication.AUTHENTICATION_NONE:
      return "AUTHENTICATION_NONE";
    case SMTPMailDeliverySetting_Authentication.AUTHENTICATION_PLAIN:
      return "AUTHENTICATION_PLAIN";
    case SMTPMailDeliverySetting_Authentication.AUTHENTICATION_LOGIN:
      return "AUTHENTICATION_LOGIN";
    case SMTPMailDeliverySetting_Authentication.AUTHENTICATION_CRAM_MD5:
      return "AUTHENTICATION_CRAM_MD5";
    case SMTPMailDeliverySetting_Authentication.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
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
  sensitive: boolean;
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

export interface SemanticCategorySetting {
  categories: SemanticCategorySetting_SemanticCategory[];
}

export interface SemanticCategorySetting_SemanticCategory {
  /** id is the uuid for category item. */
  id: string;
  /** the title of the category item, it should not be empty. */
  title: string;
  /** the description of the category item, it can be empty. */
  description: string;
}

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
  return { expression: undefined, template: undefined, condition: undefined };
}

export const WorkspaceApprovalSetting_Rule = {
  encode(message: WorkspaceApprovalSetting_Rule, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.expression !== undefined) {
      ParsedExpr.encode(message.expression, writer.uint32(10).fork()).ldelim();
    }
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
        case 1:
          if (tag !== 10) {
            break;
          }

          message.expression = ParsedExpr.decode(reader, reader.uint32());
          continue;
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
      expression: isSet(object.expression) ? ParsedExpr.fromJSON(object.expression) : undefined,
      template: isSet(object.template) ? ApprovalTemplate.fromJSON(object.template) : undefined,
      condition: isSet(object.condition) ? Expr.fromJSON(object.condition) : undefined,
    };
  },

  toJSON(message: WorkspaceApprovalSetting_Rule): unknown {
    const obj: any = {};
    message.expression !== undefined &&
      (obj.expression = message.expression ? ParsedExpr.toJSON(message.expression) : undefined);
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
    message.expression = (object.expression !== undefined && object.expression !== null)
      ? ParsedExpr.fromPartial(object.expression)
      : undefined;
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

function createBaseSMTPMailDeliverySetting(): SMTPMailDeliverySetting {
  return {
    server: "",
    port: 0,
    encryption: 0,
    ca: "",
    key: "",
    cert: "",
    authentication: 0,
    username: "",
    password: "",
    from: "",
  };
}

export const SMTPMailDeliverySetting = {
  encode(message: SMTPMailDeliverySetting, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.server !== "") {
      writer.uint32(10).string(message.server);
    }
    if (message.port !== 0) {
      writer.uint32(16).int32(message.port);
    }
    if (message.encryption !== 0) {
      writer.uint32(24).int32(message.encryption);
    }
    if (message.ca !== "") {
      writer.uint32(34).string(message.ca);
    }
    if (message.key !== "") {
      writer.uint32(42).string(message.key);
    }
    if (message.cert !== "") {
      writer.uint32(50).string(message.cert);
    }
    if (message.authentication !== 0) {
      writer.uint32(56).int32(message.authentication);
    }
    if (message.username !== "") {
      writer.uint32(66).string(message.username);
    }
    if (message.password !== "") {
      writer.uint32(74).string(message.password);
    }
    if (message.from !== "") {
      writer.uint32(82).string(message.from);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SMTPMailDeliverySetting {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSMTPMailDeliverySetting();
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
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SMTPMailDeliverySetting {
    return {
      server: isSet(object.server) ? String(object.server) : "",
      port: isSet(object.port) ? Number(object.port) : 0,
      encryption: isSet(object.encryption) ? sMTPMailDeliverySetting_EncryptionFromJSON(object.encryption) : 0,
      ca: isSet(object.ca) ? String(object.ca) : "",
      key: isSet(object.key) ? String(object.key) : "",
      cert: isSet(object.cert) ? String(object.cert) : "",
      authentication: isSet(object.authentication)
        ? sMTPMailDeliverySetting_AuthenticationFromJSON(object.authentication)
        : 0,
      username: isSet(object.username) ? String(object.username) : "",
      password: isSet(object.password) ? String(object.password) : "",
      from: isSet(object.from) ? String(object.from) : "",
    };
  },

  toJSON(message: SMTPMailDeliverySetting): unknown {
    const obj: any = {};
    message.server !== undefined && (obj.server = message.server);
    message.port !== undefined && (obj.port = Math.round(message.port));
    message.encryption !== undefined && (obj.encryption = sMTPMailDeliverySetting_EncryptionToJSON(message.encryption));
    message.ca !== undefined && (obj.ca = message.ca);
    message.key !== undefined && (obj.key = message.key);
    message.cert !== undefined && (obj.cert = message.cert);
    message.authentication !== undefined &&
      (obj.authentication = sMTPMailDeliverySetting_AuthenticationToJSON(message.authentication));
    message.username !== undefined && (obj.username = message.username);
    message.password !== undefined && (obj.password = message.password);
    message.from !== undefined && (obj.from = message.from);
    return obj;
  },

  create(base?: DeepPartial<SMTPMailDeliverySetting>): SMTPMailDeliverySetting {
    return SMTPMailDeliverySetting.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SMTPMailDeliverySetting>): SMTPMailDeliverySetting {
    const message = createBaseSMTPMailDeliverySetting();
    message.server = object.server ?? "";
    message.port = object.port ?? 0;
    message.encryption = object.encryption ?? 0;
    message.ca = object.ca ?? "";
    message.key = object.key ?? "";
    message.cert = object.cert ?? "";
    message.authentication = object.authentication ?? 0;
    message.username = object.username ?? "";
    message.password = object.password ?? "";
    message.from = object.from ?? "";
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
  return { id: "", title: "", description: "", sensitive: false };
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
    if (message.sensitive === true) {
      writer.uint32(32).bool(message.sensitive);
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
        case 4:
          if (tag !== 32) {
            break;
          }

          message.sensitive = reader.bool();
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
      sensitive: isSet(object.sensitive) ? Boolean(object.sensitive) : false,
    };
  },

  toJSON(message: DataClassificationSetting_DataClassificationConfig_Level): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.title !== undefined && (obj.title = message.title);
    message.description !== undefined && (obj.description = message.description);
    message.sensitive !== undefined && (obj.sensitive = message.sensitive);
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
    message.sensitive = object.sensitive ?? false;
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

function createBaseSemanticCategorySetting(): SemanticCategorySetting {
  return { categories: [] };
}

export const SemanticCategorySetting = {
  encode(message: SemanticCategorySetting, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.categories) {
      SemanticCategorySetting_SemanticCategory.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SemanticCategorySetting {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSemanticCategorySetting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.categories.push(SemanticCategorySetting_SemanticCategory.decode(reader, reader.uint32()));
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SemanticCategorySetting {
    return {
      categories: Array.isArray(object?.categories)
        ? object.categories.map((e: any) => SemanticCategorySetting_SemanticCategory.fromJSON(e))
        : [],
    };
  },

  toJSON(message: SemanticCategorySetting): unknown {
    const obj: any = {};
    if (message.categories) {
      obj.categories = message.categories.map((e) =>
        e ? SemanticCategorySetting_SemanticCategory.toJSON(e) : undefined
      );
    } else {
      obj.categories = [];
    }
    return obj;
  },

  create(base?: DeepPartial<SemanticCategorySetting>): SemanticCategorySetting {
    return SemanticCategorySetting.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SemanticCategorySetting>): SemanticCategorySetting {
    const message = createBaseSemanticCategorySetting();
    message.categories = object.categories?.map((e) => SemanticCategorySetting_SemanticCategory.fromPartial(e)) || [];
    return message;
  },
};

function createBaseSemanticCategorySetting_SemanticCategory(): SemanticCategorySetting_SemanticCategory {
  return { id: "", title: "", description: "" };
}

export const SemanticCategorySetting_SemanticCategory = {
  encode(message: SemanticCategorySetting_SemanticCategory, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
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

  decode(input: _m0.Reader | Uint8Array, length?: number): SemanticCategorySetting_SemanticCategory {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSemanticCategorySetting_SemanticCategory();
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

  fromJSON(object: any): SemanticCategorySetting_SemanticCategory {
    return {
      id: isSet(object.id) ? String(object.id) : "",
      title: isSet(object.title) ? String(object.title) : "",
      description: isSet(object.description) ? String(object.description) : "",
    };
  },

  toJSON(message: SemanticCategorySetting_SemanticCategory): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.title !== undefined && (obj.title = message.title);
    message.description !== undefined && (obj.description = message.description);
    return obj;
  },

  create(base?: DeepPartial<SemanticCategorySetting_SemanticCategory>): SemanticCategorySetting_SemanticCategory {
    return SemanticCategorySetting_SemanticCategory.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SemanticCategorySetting_SemanticCategory>): SemanticCategorySetting_SemanticCategory {
    const message = createBaseSemanticCategorySetting_SemanticCategory();
    message.id = object.id ?? "";
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isObject(value: any): boolean {
  return typeof value === "object" && value !== null;
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
