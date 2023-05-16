/* eslint-disable */
import * as _m0 from "protobufjs/minimal";
import { ParsedExpr } from "../google/api/expr/v1alpha1/syntax";
import { ApprovalTemplate } from "./approval";

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
  expression?: ParsedExpr;
  template?: ApprovalTemplate;
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

export interface AppIMSetting {
  imType: AppIMSetting_IMType;
  appId: string;
  appSecret: string;
  externalApproval?: AppIMSetting_ExternalApproval;
}

export enum AppIMSetting_IMType {
  IM_UNSPECIFIED = 0,
  IM_FEISHU = 1,
  UNRECOGNIZED = -1,
}

export function appIMSetting_IMTypeFromJSON(object: any): AppIMSetting_IMType {
  switch (object) {
    case 0:
    case "IM_UNSPECIFIED":
      return AppIMSetting_IMType.IM_UNSPECIFIED;
    case 1:
    case "IM_FEISHU":
      return AppIMSetting_IMType.IM_FEISHU;
    case -1:
    case "UNRECOGNIZED":
    default:
      return AppIMSetting_IMType.UNRECOGNIZED;
  }
}

export function appIMSetting_IMTypeToJSON(object: AppIMSetting_IMType): string {
  switch (object) {
    case AppIMSetting_IMType.IM_UNSPECIFIED:
      return "IM_UNSPECIFIED";
    case AppIMSetting_IMType.IM_FEISHU:
      return "IM_FEISHU";
    case AppIMSetting_IMType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface AppIMSetting_ExternalApproval {
  enabled: boolean;
  approvalDefinitionId: string;
}

function createBaseWorkspaceProfileSetting(): WorkspaceProfileSetting {
  return { externalUrl: "", disallowSignup: false, require2fa: false, outboundIpList: [] };
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
  return { expression: undefined, template: undefined };
}

export const WorkspaceApprovalSetting_Rule = {
  encode(message: WorkspaceApprovalSetting_Rule, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.expression !== undefined) {
      ParsedExpr.encode(message.expression, writer.uint32(10).fork()).ldelim();
    }
    if (message.template !== undefined) {
      ApprovalTemplate.encode(message.template, writer.uint32(18).fork()).ldelim();
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
    };
  },

  toJSON(message: WorkspaceApprovalSetting_Rule): unknown {
    const obj: any = {};
    message.expression !== undefined &&
      (obj.expression = message.expression ? ParsedExpr.toJSON(message.expression) : undefined);
    message.template !== undefined &&
      (obj.template = message.template ? ApprovalTemplate.toJSON(message.template) : undefined);
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

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
