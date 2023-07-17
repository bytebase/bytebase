/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { ParsedExpr } from "../google/api/expr/v1alpha1/syntax";
import { Expr } from "../google/type/expr";
import { ApprovalTemplate } from "./approval";
import { Engine, engineFromJSON, engineToJSON } from "./common";

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
  templates: SchemaTemplateSetting_SchemaTemplate[];
}

export interface SchemaTemplateSetting_SchemaTemplate {
  id: string;
  title: string;
  engine: Engine;
  type: SchemaTemplateSetting_SchemaTemplate_Type;
  fieldTemplatePayload?: SchemaTemplateSetting_SchemaTemplate_FieldTemplatePayload | undefined;
}

export enum SchemaTemplateSetting_SchemaTemplate_Type {
  TYPE_UNSPECIFIED = 0,
  TABLE = 1,
  FIELD = 2,
  UNRECOGNIZED = -1,
}

export function schemaTemplateSetting_SchemaTemplate_TypeFromJSON(
  object: any,
): SchemaTemplateSetting_SchemaTemplate_Type {
  switch (object) {
    case 0:
    case "TYPE_UNSPECIFIED":
      return SchemaTemplateSetting_SchemaTemplate_Type.TYPE_UNSPECIFIED;
    case 1:
    case "TABLE":
      return SchemaTemplateSetting_SchemaTemplate_Type.TABLE;
    case 2:
    case "FIELD":
      return SchemaTemplateSetting_SchemaTemplate_Type.FIELD;
    case -1:
    case "UNRECOGNIZED":
    default:
      return SchemaTemplateSetting_SchemaTemplate_Type.UNRECOGNIZED;
  }
}

export function schemaTemplateSetting_SchemaTemplate_TypeToJSON(
  object: SchemaTemplateSetting_SchemaTemplate_Type,
): string {
  switch (object) {
    case SchemaTemplateSetting_SchemaTemplate_Type.TYPE_UNSPECIFIED:
      return "TYPE_UNSPECIFIED";
    case SchemaTemplateSetting_SchemaTemplate_Type.TABLE:
      return "TABLE";
    case SchemaTemplateSetting_SchemaTemplate_Type.FIELD:
      return "FIELD";
    case SchemaTemplateSetting_SchemaTemplate_Type.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface SchemaTemplateSetting_SchemaTemplate_FieldTemplatePayload {
  name: string;
  type: string;
  primary: boolean;
  nullable: boolean;
  default?: string | undefined;
  comment: string;
  category: string;
}

function createBaseWorkspaceProfileSetting(): WorkspaceProfileSetting {
  return { externalUrl: "", disallowSignup: false, require2fa: false, outboundIpList: [], gitopsWebhookUrl: "" };
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
  return { templates: [] };
}

export const SchemaTemplateSetting = {
  encode(message: SchemaTemplateSetting, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.templates) {
      SchemaTemplateSetting_SchemaTemplate.encode(v!, writer.uint32(10).fork()).ldelim();
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

          message.templates.push(SchemaTemplateSetting_SchemaTemplate.decode(reader, reader.uint32()));
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
      templates: Array.isArray(object?.templates)
        ? object.templates.map((e: any) => SchemaTemplateSetting_SchemaTemplate.fromJSON(e))
        : [],
    };
  },

  toJSON(message: SchemaTemplateSetting): unknown {
    const obj: any = {};
    if (message.templates) {
      obj.templates = message.templates.map((e) => e ? SchemaTemplateSetting_SchemaTemplate.toJSON(e) : undefined);
    } else {
      obj.templates = [];
    }
    return obj;
  },

  create(base?: DeepPartial<SchemaTemplateSetting>): SchemaTemplateSetting {
    return SchemaTemplateSetting.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SchemaTemplateSetting>): SchemaTemplateSetting {
    const message = createBaseSchemaTemplateSetting();
    message.templates = object.templates?.map((e) => SchemaTemplateSetting_SchemaTemplate.fromPartial(e)) || [];
    return message;
  },
};

function createBaseSchemaTemplateSetting_SchemaTemplate(): SchemaTemplateSetting_SchemaTemplate {
  return { id: "", title: "", engine: 0, type: 0, fieldTemplatePayload: undefined };
}

export const SchemaTemplateSetting_SchemaTemplate = {
  encode(message: SchemaTemplateSetting_SchemaTemplate, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.engine !== 0) {
      writer.uint32(24).int32(message.engine);
    }
    if (message.type !== 0) {
      writer.uint32(32).int32(message.type);
    }
    if (message.fieldTemplatePayload !== undefined) {
      SchemaTemplateSetting_SchemaTemplate_FieldTemplatePayload.encode(
        message.fieldTemplatePayload,
        writer.uint32(42).fork(),
      ).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SchemaTemplateSetting_SchemaTemplate {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSchemaTemplateSetting_SchemaTemplate();
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
          if (tag !== 24) {
            break;
          }

          message.engine = reader.int32() as any;
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.type = reader.int32() as any;
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.fieldTemplatePayload = SchemaTemplateSetting_SchemaTemplate_FieldTemplatePayload.decode(
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

  fromJSON(object: any): SchemaTemplateSetting_SchemaTemplate {
    return {
      id: isSet(object.id) ? String(object.id) : "",
      title: isSet(object.title) ? String(object.title) : "",
      engine: isSet(object.engine) ? engineFromJSON(object.engine) : 0,
      type: isSet(object.type) ? schemaTemplateSetting_SchemaTemplate_TypeFromJSON(object.type) : 0,
      fieldTemplatePayload: isSet(object.fieldTemplatePayload)
        ? SchemaTemplateSetting_SchemaTemplate_FieldTemplatePayload.fromJSON(object.fieldTemplatePayload)
        : undefined,
    };
  },

  toJSON(message: SchemaTemplateSetting_SchemaTemplate): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = message.id);
    message.title !== undefined && (obj.title = message.title);
    message.engine !== undefined && (obj.engine = engineToJSON(message.engine));
    message.type !== undefined && (obj.type = schemaTemplateSetting_SchemaTemplate_TypeToJSON(message.type));
    message.fieldTemplatePayload !== undefined && (obj.fieldTemplatePayload = message.fieldTemplatePayload
      ? SchemaTemplateSetting_SchemaTemplate_FieldTemplatePayload.toJSON(message.fieldTemplatePayload)
      : undefined);
    return obj;
  },

  create(base?: DeepPartial<SchemaTemplateSetting_SchemaTemplate>): SchemaTemplateSetting_SchemaTemplate {
    return SchemaTemplateSetting_SchemaTemplate.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SchemaTemplateSetting_SchemaTemplate>): SchemaTemplateSetting_SchemaTemplate {
    const message = createBaseSchemaTemplateSetting_SchemaTemplate();
    message.id = object.id ?? "";
    message.title = object.title ?? "";
    message.engine = object.engine ?? 0;
    message.type = object.type ?? 0;
    message.fieldTemplatePayload = (object.fieldTemplatePayload !== undefined && object.fieldTemplatePayload !== null)
      ? SchemaTemplateSetting_SchemaTemplate_FieldTemplatePayload.fromPartial(object.fieldTemplatePayload)
      : undefined;
    return message;
  },
};

function createBaseSchemaTemplateSetting_SchemaTemplate_FieldTemplatePayload(): SchemaTemplateSetting_SchemaTemplate_FieldTemplatePayload {
  return { name: "", type: "", primary: false, nullable: false, default: undefined, comment: "", category: "" };
}

export const SchemaTemplateSetting_SchemaTemplate_FieldTemplatePayload = {
  encode(
    message: SchemaTemplateSetting_SchemaTemplate_FieldTemplatePayload,
    writer: _m0.Writer = _m0.Writer.create(),
  ): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.type !== "") {
      writer.uint32(18).string(message.type);
    }
    if (message.primary === true) {
      writer.uint32(24).bool(message.primary);
    }
    if (message.nullable === true) {
      writer.uint32(32).bool(message.nullable);
    }
    if (message.default !== undefined) {
      writer.uint32(42).string(message.default);
    }
    if (message.comment !== "") {
      writer.uint32(50).string(message.comment);
    }
    if (message.category !== "") {
      writer.uint32(58).string(message.category);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SchemaTemplateSetting_SchemaTemplate_FieldTemplatePayload {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSchemaTemplateSetting_SchemaTemplate_FieldTemplatePayload();
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

          message.type = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.primary = reader.bool();
          continue;
        case 4:
          if (tag !== 32) {
            break;
          }

          message.nullable = reader.bool();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.default = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.comment = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.category = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SchemaTemplateSetting_SchemaTemplate_FieldTemplatePayload {
    return {
      name: isSet(object.name) ? String(object.name) : "",
      type: isSet(object.type) ? String(object.type) : "",
      primary: isSet(object.primary) ? Boolean(object.primary) : false,
      nullable: isSet(object.nullable) ? Boolean(object.nullable) : false,
      default: isSet(object.default) ? String(object.default) : undefined,
      comment: isSet(object.comment) ? String(object.comment) : "",
      category: isSet(object.category) ? String(object.category) : "",
    };
  },

  toJSON(message: SchemaTemplateSetting_SchemaTemplate_FieldTemplatePayload): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    message.type !== undefined && (obj.type = message.type);
    message.primary !== undefined && (obj.primary = message.primary);
    message.nullable !== undefined && (obj.nullable = message.nullable);
    message.default !== undefined && (obj.default = message.default);
    message.comment !== undefined && (obj.comment = message.comment);
    message.category !== undefined && (obj.category = message.category);
    return obj;
  },

  create(
    base?: DeepPartial<SchemaTemplateSetting_SchemaTemplate_FieldTemplatePayload>,
  ): SchemaTemplateSetting_SchemaTemplate_FieldTemplatePayload {
    return SchemaTemplateSetting_SchemaTemplate_FieldTemplatePayload.fromPartial(base ?? {});
  },

  fromPartial(
    object: DeepPartial<SchemaTemplateSetting_SchemaTemplate_FieldTemplatePayload>,
  ): SchemaTemplateSetting_SchemaTemplate_FieldTemplatePayload {
    const message = createBaseSchemaTemplateSetting_SchemaTemplate_FieldTemplatePayload();
    message.name = object.name ?? "";
    message.type = object.type ?? "";
    message.primary = object.primary ?? false;
    message.nullable = object.nullable ?? false;
    message.default = object.default ?? undefined;
    message.comment = object.comment ?? "";
    message.category = object.category ?? "";
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
