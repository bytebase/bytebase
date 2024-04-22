/* eslint-disable */
import Long from "long";
import _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export interface DataSourceExternalSecret {
  secretType: DataSourceExternalSecret_SecretType;
  url: string;
  authType: DataSourceExternalSecret_AuthType;
  appRole?: DataSourceExternalSecret_AppRoleAuthOption | undefined;
  token?:
    | string
    | undefined;
  /** engine name is the name for secret engine. */
  engineName: string;
  /** the secret name in the engine to store the password. */
  secretName: string;
  /** the key name for the password. */
  passwordKeyName: string;
}

export enum DataSourceExternalSecret_SecretType {
  SAECRET_TYPE_UNSPECIFIED = 0,
  /** VAULT_KV_V2 - ref: https://developer.hashicorp.com/vault/api-docs/secret/kv/kv-v2 */
  VAULT_KV_V2 = 1,
  /** AWS_SECRETS_MANAGER - ref: https://docs.aws.amazon.com/secretsmanager/latest/userguide/intro.html */
  AWS_SECRETS_MANAGER = 2,
  /** GCP_SECRET_MANAGER - ref: https://cloud.google.com/secret-manager/docs */
  GCP_SECRET_MANAGER = 3,
  UNRECOGNIZED = -1,
}

export function dataSourceExternalSecret_SecretTypeFromJSON(object: any): DataSourceExternalSecret_SecretType {
  switch (object) {
    case 0:
    case "SAECRET_TYPE_UNSPECIFIED":
      return DataSourceExternalSecret_SecretType.SAECRET_TYPE_UNSPECIFIED;
    case 1:
    case "VAULT_KV_V2":
      return DataSourceExternalSecret_SecretType.VAULT_KV_V2;
    case 2:
    case "AWS_SECRETS_MANAGER":
      return DataSourceExternalSecret_SecretType.AWS_SECRETS_MANAGER;
    case 3:
    case "GCP_SECRET_MANAGER":
      return DataSourceExternalSecret_SecretType.GCP_SECRET_MANAGER;
    case -1:
    case "UNRECOGNIZED":
    default:
      return DataSourceExternalSecret_SecretType.UNRECOGNIZED;
  }
}

export function dataSourceExternalSecret_SecretTypeToJSON(object: DataSourceExternalSecret_SecretType): string {
  switch (object) {
    case DataSourceExternalSecret_SecretType.SAECRET_TYPE_UNSPECIFIED:
      return "SAECRET_TYPE_UNSPECIFIED";
    case DataSourceExternalSecret_SecretType.VAULT_KV_V2:
      return "VAULT_KV_V2";
    case DataSourceExternalSecret_SecretType.AWS_SECRETS_MANAGER:
      return "AWS_SECRETS_MANAGER";
    case DataSourceExternalSecret_SecretType.GCP_SECRET_MANAGER:
      return "GCP_SECRET_MANAGER";
    case DataSourceExternalSecret_SecretType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum DataSourceExternalSecret_AuthType {
  AUTH_TYPE_UNSPECIFIED = 0,
  /** TOKEN - ref: https://developer.hashicorp.com/vault/docs/auth/token */
  TOKEN = 1,
  /** VAULT_APP_ROLE - ref: https://developer.hashicorp.com/vault/docs/auth/approle */
  VAULT_APP_ROLE = 2,
  UNRECOGNIZED = -1,
}

export function dataSourceExternalSecret_AuthTypeFromJSON(object: any): DataSourceExternalSecret_AuthType {
  switch (object) {
    case 0:
    case "AUTH_TYPE_UNSPECIFIED":
      return DataSourceExternalSecret_AuthType.AUTH_TYPE_UNSPECIFIED;
    case 1:
    case "TOKEN":
      return DataSourceExternalSecret_AuthType.TOKEN;
    case 2:
    case "VAULT_APP_ROLE":
      return DataSourceExternalSecret_AuthType.VAULT_APP_ROLE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return DataSourceExternalSecret_AuthType.UNRECOGNIZED;
  }
}

export function dataSourceExternalSecret_AuthTypeToJSON(object: DataSourceExternalSecret_AuthType): string {
  switch (object) {
    case DataSourceExternalSecret_AuthType.AUTH_TYPE_UNSPECIFIED:
      return "AUTH_TYPE_UNSPECIFIED";
    case DataSourceExternalSecret_AuthType.TOKEN:
      return "TOKEN";
    case DataSourceExternalSecret_AuthType.VAULT_APP_ROLE:
      return "VAULT_APP_ROLE";
    case DataSourceExternalSecret_AuthType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface DataSourceExternalSecret_AppRoleAuthOption {
  roleId: string;
  /** the secret id for the role without ttl. */
  secretId: string;
  type: DataSourceExternalSecret_AppRoleAuthOption_SecretType;
  /** The path where the approle auth method is mounted. */
  mountPath: string;
}

export enum DataSourceExternalSecret_AppRoleAuthOption_SecretType {
  SECRET_TYPE_UNSPECIFIED = 0,
  PLAIN = 1,
  ENVIRONMENT = 2,
  UNRECOGNIZED = -1,
}

export function dataSourceExternalSecret_AppRoleAuthOption_SecretTypeFromJSON(
  object: any,
): DataSourceExternalSecret_AppRoleAuthOption_SecretType {
  switch (object) {
    case 0:
    case "SECRET_TYPE_UNSPECIFIED":
      return DataSourceExternalSecret_AppRoleAuthOption_SecretType.SECRET_TYPE_UNSPECIFIED;
    case 1:
    case "PLAIN":
      return DataSourceExternalSecret_AppRoleAuthOption_SecretType.PLAIN;
    case 2:
    case "ENVIRONMENT":
      return DataSourceExternalSecret_AppRoleAuthOption_SecretType.ENVIRONMENT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return DataSourceExternalSecret_AppRoleAuthOption_SecretType.UNRECOGNIZED;
  }
}

export function dataSourceExternalSecret_AppRoleAuthOption_SecretTypeToJSON(
  object: DataSourceExternalSecret_AppRoleAuthOption_SecretType,
): string {
  switch (object) {
    case DataSourceExternalSecret_AppRoleAuthOption_SecretType.SECRET_TYPE_UNSPECIFIED:
      return "SECRET_TYPE_UNSPECIFIED";
    case DataSourceExternalSecret_AppRoleAuthOption_SecretType.PLAIN:
      return "PLAIN";
    case DataSourceExternalSecret_AppRoleAuthOption_SecretType.ENVIRONMENT:
      return "ENVIRONMENT";
    case DataSourceExternalSecret_AppRoleAuthOption_SecretType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface DataSourceOptions {
  /** srv is a boolean flag that indicates whether the host is a DNS SRV record. */
  srv: boolean;
  /** authentication_database is the database name to authenticate against, which stores the user credentials. */
  authenticationDatabase: string;
  /** sid and service_name are used for Oracle. */
  sid: string;
  serviceName: string;
  /**
   * SSH related
   * The hostname of the SSH server agent.
   */
  sshHost: string;
  /** The port of the SSH server agent. It's 22 typically. */
  sshPort: string;
  /** The user to login the server. */
  sshUser: string;
  /** The password to login the server. If it's empty string, no password is required. */
  sshObfuscatedPassword: string;
  /** The private key to login the server. If it's empty string, we will use the system default private key from os.Getenv("SSH_AUTH_SOCK"). */
  sshObfuscatedPrivateKey: string;
  /**
   * PKCS#8 private key in PEM format. If it's empty string, no private key is required.
   * Used for authentication when connecting to the data source.
   */
  authenticationPrivateKeyObfuscated: string;
  externalSecret: DataSourceExternalSecret | undefined;
  authenticationType: DataSourceOptions_AuthenticationType;
  saslConfig: SASLConfig | undefined;
}

export enum DataSourceOptions_AuthenticationType {
  AUTHENTICATION_UNSPECIFIED = 0,
  PASSWORD = 1,
  GOOGLE_CLOUD_SQL_IAM = 2,
  UNRECOGNIZED = -1,
}

export function dataSourceOptions_AuthenticationTypeFromJSON(object: any): DataSourceOptions_AuthenticationType {
  switch (object) {
    case 0:
    case "AUTHENTICATION_UNSPECIFIED":
      return DataSourceOptions_AuthenticationType.AUTHENTICATION_UNSPECIFIED;
    case 1:
    case "PASSWORD":
      return DataSourceOptions_AuthenticationType.PASSWORD;
    case 2:
    case "GOOGLE_CLOUD_SQL_IAM":
      return DataSourceOptions_AuthenticationType.GOOGLE_CLOUD_SQL_IAM;
    case -1:
    case "UNRECOGNIZED":
    default:
      return DataSourceOptions_AuthenticationType.UNRECOGNIZED;
  }
}

export function dataSourceOptions_AuthenticationTypeToJSON(object: DataSourceOptions_AuthenticationType): string {
  switch (object) {
    case DataSourceOptions_AuthenticationType.AUTHENTICATION_UNSPECIFIED:
      return "AUTHENTICATION_UNSPECIFIED";
    case DataSourceOptions_AuthenticationType.PASSWORD:
      return "PASSWORD";
    case DataSourceOptions_AuthenticationType.GOOGLE_CLOUD_SQL_IAM:
      return "GOOGLE_CLOUD_SQL_IAM";
    case DataSourceOptions_AuthenticationType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface SASLConfig {
  krbConfig?: KerberosConfig | undefined;
  plainConfig?: PlainSASLConfig | undefined;
}

export interface KerberosConfig {
  primary: string;
  instance: string;
  realm: string;
  keytab: string;
  kdcHost: string;
  kdcTransportProtocol: string;
}

export interface PlainSASLConfig {
  username: string;
  password: string;
}

function createBaseDataSourceExternalSecret(): DataSourceExternalSecret {
  return {
    secretType: 0,
    url: "",
    authType: 0,
    appRole: undefined,
    token: undefined,
    engineName: "",
    secretName: "",
    passwordKeyName: "",
  };
}

export const DataSourceExternalSecret = {
  encode(message: DataSourceExternalSecret, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.secretType !== 0) {
      writer.uint32(8).int32(message.secretType);
    }
    if (message.url !== "") {
      writer.uint32(18).string(message.url);
    }
    if (message.authType !== 0) {
      writer.uint32(24).int32(message.authType);
    }
    if (message.appRole !== undefined) {
      DataSourceExternalSecret_AppRoleAuthOption.encode(message.appRole, writer.uint32(34).fork()).ldelim();
    }
    if (message.token !== undefined) {
      writer.uint32(42).string(message.token);
    }
    if (message.engineName !== "") {
      writer.uint32(50).string(message.engineName);
    }
    if (message.secretName !== "") {
      writer.uint32(58).string(message.secretName);
    }
    if (message.passwordKeyName !== "") {
      writer.uint32(66).string(message.passwordKeyName);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DataSourceExternalSecret {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDataSourceExternalSecret();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.secretType = reader.int32() as any;
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.url = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.authType = reader.int32() as any;
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.appRole = DataSourceExternalSecret_AppRoleAuthOption.decode(reader, reader.uint32());
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.token = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.engineName = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.secretName = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.passwordKeyName = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DataSourceExternalSecret {
    return {
      secretType: isSet(object.secretType) ? dataSourceExternalSecret_SecretTypeFromJSON(object.secretType) : 0,
      url: isSet(object.url) ? globalThis.String(object.url) : "",
      authType: isSet(object.authType) ? dataSourceExternalSecret_AuthTypeFromJSON(object.authType) : 0,
      appRole: isSet(object.appRole) ? DataSourceExternalSecret_AppRoleAuthOption.fromJSON(object.appRole) : undefined,
      token: isSet(object.token) ? globalThis.String(object.token) : undefined,
      engineName: isSet(object.engineName) ? globalThis.String(object.engineName) : "",
      secretName: isSet(object.secretName) ? globalThis.String(object.secretName) : "",
      passwordKeyName: isSet(object.passwordKeyName) ? globalThis.String(object.passwordKeyName) : "",
    };
  },

  toJSON(message: DataSourceExternalSecret): unknown {
    const obj: any = {};
    if (message.secretType !== 0) {
      obj.secretType = dataSourceExternalSecret_SecretTypeToJSON(message.secretType);
    }
    if (message.url !== "") {
      obj.url = message.url;
    }
    if (message.authType !== 0) {
      obj.authType = dataSourceExternalSecret_AuthTypeToJSON(message.authType);
    }
    if (message.appRole !== undefined) {
      obj.appRole = DataSourceExternalSecret_AppRoleAuthOption.toJSON(message.appRole);
    }
    if (message.token !== undefined) {
      obj.token = message.token;
    }
    if (message.engineName !== "") {
      obj.engineName = message.engineName;
    }
    if (message.secretName !== "") {
      obj.secretName = message.secretName;
    }
    if (message.passwordKeyName !== "") {
      obj.passwordKeyName = message.passwordKeyName;
    }
    return obj;
  },

  create(base?: DeepPartial<DataSourceExternalSecret>): DataSourceExternalSecret {
    return DataSourceExternalSecret.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DataSourceExternalSecret>): DataSourceExternalSecret {
    const message = createBaseDataSourceExternalSecret();
    message.secretType = object.secretType ?? 0;
    message.url = object.url ?? "";
    message.authType = object.authType ?? 0;
    message.appRole = (object.appRole !== undefined && object.appRole !== null)
      ? DataSourceExternalSecret_AppRoleAuthOption.fromPartial(object.appRole)
      : undefined;
    message.token = object.token ?? undefined;
    message.engineName = object.engineName ?? "";
    message.secretName = object.secretName ?? "";
    message.passwordKeyName = object.passwordKeyName ?? "";
    return message;
  },
};

function createBaseDataSourceExternalSecret_AppRoleAuthOption(): DataSourceExternalSecret_AppRoleAuthOption {
  return { roleId: "", secretId: "", type: 0, mountPath: "" };
}

export const DataSourceExternalSecret_AppRoleAuthOption = {
  encode(message: DataSourceExternalSecret_AppRoleAuthOption, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.roleId !== "") {
      writer.uint32(10).string(message.roleId);
    }
    if (message.secretId !== "") {
      writer.uint32(18).string(message.secretId);
    }
    if (message.type !== 0) {
      writer.uint32(24).int32(message.type);
    }
    if (message.mountPath !== "") {
      writer.uint32(34).string(message.mountPath);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DataSourceExternalSecret_AppRoleAuthOption {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDataSourceExternalSecret_AppRoleAuthOption();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.roleId = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.secretId = reader.string();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.type = reader.int32() as any;
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.mountPath = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DataSourceExternalSecret_AppRoleAuthOption {
    return {
      roleId: isSet(object.roleId) ? globalThis.String(object.roleId) : "",
      secretId: isSet(object.secretId) ? globalThis.String(object.secretId) : "",
      type: isSet(object.type) ? dataSourceExternalSecret_AppRoleAuthOption_SecretTypeFromJSON(object.type) : 0,
      mountPath: isSet(object.mountPath) ? globalThis.String(object.mountPath) : "",
    };
  },

  toJSON(message: DataSourceExternalSecret_AppRoleAuthOption): unknown {
    const obj: any = {};
    if (message.roleId !== "") {
      obj.roleId = message.roleId;
    }
    if (message.secretId !== "") {
      obj.secretId = message.secretId;
    }
    if (message.type !== 0) {
      obj.type = dataSourceExternalSecret_AppRoleAuthOption_SecretTypeToJSON(message.type);
    }
    if (message.mountPath !== "") {
      obj.mountPath = message.mountPath;
    }
    return obj;
  },

  create(base?: DeepPartial<DataSourceExternalSecret_AppRoleAuthOption>): DataSourceExternalSecret_AppRoleAuthOption {
    return DataSourceExternalSecret_AppRoleAuthOption.fromPartial(base ?? {});
  },
  fromPartial(
    object: DeepPartial<DataSourceExternalSecret_AppRoleAuthOption>,
  ): DataSourceExternalSecret_AppRoleAuthOption {
    const message = createBaseDataSourceExternalSecret_AppRoleAuthOption();
    message.roleId = object.roleId ?? "";
    message.secretId = object.secretId ?? "";
    message.type = object.type ?? 0;
    message.mountPath = object.mountPath ?? "";
    return message;
  },
};

function createBaseDataSourceOptions(): DataSourceOptions {
  return {
    srv: false,
    authenticationDatabase: "",
    sid: "",
    serviceName: "",
    sshHost: "",
    sshPort: "",
    sshUser: "",
    sshObfuscatedPassword: "",
    sshObfuscatedPrivateKey: "",
    authenticationPrivateKeyObfuscated: "",
    externalSecret: undefined,
    authenticationType: 0,
    saslConfig: undefined,
  };
}

export const DataSourceOptions = {
  encode(message: DataSourceOptions, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.srv === true) {
      writer.uint32(8).bool(message.srv);
    }
    if (message.authenticationDatabase !== "") {
      writer.uint32(18).string(message.authenticationDatabase);
    }
    if (message.sid !== "") {
      writer.uint32(26).string(message.sid);
    }
    if (message.serviceName !== "") {
      writer.uint32(34).string(message.serviceName);
    }
    if (message.sshHost !== "") {
      writer.uint32(42).string(message.sshHost);
    }
    if (message.sshPort !== "") {
      writer.uint32(50).string(message.sshPort);
    }
    if (message.sshUser !== "") {
      writer.uint32(58).string(message.sshUser);
    }
    if (message.sshObfuscatedPassword !== "") {
      writer.uint32(66).string(message.sshObfuscatedPassword);
    }
    if (message.sshObfuscatedPrivateKey !== "") {
      writer.uint32(74).string(message.sshObfuscatedPrivateKey);
    }
    if (message.authenticationPrivateKeyObfuscated !== "") {
      writer.uint32(82).string(message.authenticationPrivateKeyObfuscated);
    }
    if (message.externalSecret !== undefined) {
      DataSourceExternalSecret.encode(message.externalSecret, writer.uint32(90).fork()).ldelim();
    }
    if (message.authenticationType !== 0) {
      writer.uint32(96).int32(message.authenticationType);
    }
    if (message.saslConfig !== undefined) {
      SASLConfig.encode(message.saslConfig, writer.uint32(106).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DataSourceOptions {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDataSourceOptions();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.srv = reader.bool();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.authenticationDatabase = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.sid = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.serviceName = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.sshHost = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.sshPort = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.sshUser = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.sshObfuscatedPassword = reader.string();
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.sshObfuscatedPrivateKey = reader.string();
          continue;
        case 10:
          if (tag !== 82) {
            break;
          }

          message.authenticationPrivateKeyObfuscated = reader.string();
          continue;
        case 11:
          if (tag !== 90) {
            break;
          }

          message.externalSecret = DataSourceExternalSecret.decode(reader, reader.uint32());
          continue;
        case 12:
          if (tag !== 96) {
            break;
          }

          message.authenticationType = reader.int32() as any;
          continue;
        case 13:
          if (tag !== 106) {
            break;
          }

          message.saslConfig = SASLConfig.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DataSourceOptions {
    return {
      srv: isSet(object.srv) ? globalThis.Boolean(object.srv) : false,
      authenticationDatabase: isSet(object.authenticationDatabase)
        ? globalThis.String(object.authenticationDatabase)
        : "",
      sid: isSet(object.sid) ? globalThis.String(object.sid) : "",
      serviceName: isSet(object.serviceName) ? globalThis.String(object.serviceName) : "",
      sshHost: isSet(object.sshHost) ? globalThis.String(object.sshHost) : "",
      sshPort: isSet(object.sshPort) ? globalThis.String(object.sshPort) : "",
      sshUser: isSet(object.sshUser) ? globalThis.String(object.sshUser) : "",
      sshObfuscatedPassword: isSet(object.sshObfuscatedPassword) ? globalThis.String(object.sshObfuscatedPassword) : "",
      sshObfuscatedPrivateKey: isSet(object.sshObfuscatedPrivateKey)
        ? globalThis.String(object.sshObfuscatedPrivateKey)
        : "",
      authenticationPrivateKeyObfuscated: isSet(object.authenticationPrivateKeyObfuscated)
        ? globalThis.String(object.authenticationPrivateKeyObfuscated)
        : "",
      externalSecret: isSet(object.externalSecret)
        ? DataSourceExternalSecret.fromJSON(object.externalSecret)
        : undefined,
      authenticationType: isSet(object.authenticationType)
        ? dataSourceOptions_AuthenticationTypeFromJSON(object.authenticationType)
        : 0,
      saslConfig: isSet(object.saslConfig) ? SASLConfig.fromJSON(object.saslConfig) : undefined,
    };
  },

  toJSON(message: DataSourceOptions): unknown {
    const obj: any = {};
    if (message.srv === true) {
      obj.srv = message.srv;
    }
    if (message.authenticationDatabase !== "") {
      obj.authenticationDatabase = message.authenticationDatabase;
    }
    if (message.sid !== "") {
      obj.sid = message.sid;
    }
    if (message.serviceName !== "") {
      obj.serviceName = message.serviceName;
    }
    if (message.sshHost !== "") {
      obj.sshHost = message.sshHost;
    }
    if (message.sshPort !== "") {
      obj.sshPort = message.sshPort;
    }
    if (message.sshUser !== "") {
      obj.sshUser = message.sshUser;
    }
    if (message.sshObfuscatedPassword !== "") {
      obj.sshObfuscatedPassword = message.sshObfuscatedPassword;
    }
    if (message.sshObfuscatedPrivateKey !== "") {
      obj.sshObfuscatedPrivateKey = message.sshObfuscatedPrivateKey;
    }
    if (message.authenticationPrivateKeyObfuscated !== "") {
      obj.authenticationPrivateKeyObfuscated = message.authenticationPrivateKeyObfuscated;
    }
    if (message.externalSecret !== undefined) {
      obj.externalSecret = DataSourceExternalSecret.toJSON(message.externalSecret);
    }
    if (message.authenticationType !== 0) {
      obj.authenticationType = dataSourceOptions_AuthenticationTypeToJSON(message.authenticationType);
    }
    if (message.saslConfig !== undefined) {
      obj.saslConfig = SASLConfig.toJSON(message.saslConfig);
    }
    return obj;
  },

  create(base?: DeepPartial<DataSourceOptions>): DataSourceOptions {
    return DataSourceOptions.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DataSourceOptions>): DataSourceOptions {
    const message = createBaseDataSourceOptions();
    message.srv = object.srv ?? false;
    message.authenticationDatabase = object.authenticationDatabase ?? "";
    message.sid = object.sid ?? "";
    message.serviceName = object.serviceName ?? "";
    message.sshHost = object.sshHost ?? "";
    message.sshPort = object.sshPort ?? "";
    message.sshUser = object.sshUser ?? "";
    message.sshObfuscatedPassword = object.sshObfuscatedPassword ?? "";
    message.sshObfuscatedPrivateKey = object.sshObfuscatedPrivateKey ?? "";
    message.authenticationPrivateKeyObfuscated = object.authenticationPrivateKeyObfuscated ?? "";
    message.externalSecret = (object.externalSecret !== undefined && object.externalSecret !== null)
      ? DataSourceExternalSecret.fromPartial(object.externalSecret)
      : undefined;
    message.authenticationType = object.authenticationType ?? 0;
    message.saslConfig = (object.saslConfig !== undefined && object.saslConfig !== null)
      ? SASLConfig.fromPartial(object.saslConfig)
      : undefined;
    return message;
  },
};

function createBaseSASLConfig(): SASLConfig {
  return { krbConfig: undefined, plainConfig: undefined };
}

export const SASLConfig = {
  encode(message: SASLConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.krbConfig !== undefined) {
      KerberosConfig.encode(message.krbConfig, writer.uint32(10).fork()).ldelim();
    }
    if (message.plainConfig !== undefined) {
      PlainSASLConfig.encode(message.plainConfig, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SASLConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSASLConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.krbConfig = KerberosConfig.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.plainConfig = PlainSASLConfig.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SASLConfig {
    return {
      krbConfig: isSet(object.krbConfig) ? KerberosConfig.fromJSON(object.krbConfig) : undefined,
      plainConfig: isSet(object.plainConfig) ? PlainSASLConfig.fromJSON(object.plainConfig) : undefined,
    };
  },

  toJSON(message: SASLConfig): unknown {
    const obj: any = {};
    if (message.krbConfig !== undefined) {
      obj.krbConfig = KerberosConfig.toJSON(message.krbConfig);
    }
    if (message.plainConfig !== undefined) {
      obj.plainConfig = PlainSASLConfig.toJSON(message.plainConfig);
    }
    return obj;
  },

  create(base?: DeepPartial<SASLConfig>): SASLConfig {
    return SASLConfig.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<SASLConfig>): SASLConfig {
    const message = createBaseSASLConfig();
    message.krbConfig = (object.krbConfig !== undefined && object.krbConfig !== null)
      ? KerberosConfig.fromPartial(object.krbConfig)
      : undefined;
    message.plainConfig = (object.plainConfig !== undefined && object.plainConfig !== null)
      ? PlainSASLConfig.fromPartial(object.plainConfig)
      : undefined;
    return message;
  },
};

function createBaseKerberosConfig(): KerberosConfig {
  return { primary: "", instance: "", realm: "", keytab: "", kdcHost: "", kdcTransportProtocol: "" };
}

export const KerberosConfig = {
  encode(message: KerberosConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.primary !== "") {
      writer.uint32(10).string(message.primary);
    }
    if (message.instance !== "") {
      writer.uint32(18).string(message.instance);
    }
    if (message.realm !== "") {
      writer.uint32(26).string(message.realm);
    }
    if (message.keytab !== "") {
      writer.uint32(34).string(message.keytab);
    }
    if (message.kdcHost !== "") {
      writer.uint32(42).string(message.kdcHost);
    }
    if (message.kdcTransportProtocol !== "") {
      writer.uint32(50).string(message.kdcTransportProtocol);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): KerberosConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseKerberosConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.primary = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.instance = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.realm = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.keytab = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.kdcHost = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.kdcTransportProtocol = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): KerberosConfig {
    return {
      primary: isSet(object.primary) ? globalThis.String(object.primary) : "",
      instance: isSet(object.instance) ? globalThis.String(object.instance) : "",
      realm: isSet(object.realm) ? globalThis.String(object.realm) : "",
      keytab: isSet(object.keytab) ? globalThis.String(object.keytab) : "",
      kdcHost: isSet(object.kdcHost) ? globalThis.String(object.kdcHost) : "",
      kdcTransportProtocol: isSet(object.kdcTransportProtocol) ? globalThis.String(object.kdcTransportProtocol) : "",
    };
  },

  toJSON(message: KerberosConfig): unknown {
    const obj: any = {};
    if (message.primary !== "") {
      obj.primary = message.primary;
    }
    if (message.instance !== "") {
      obj.instance = message.instance;
    }
    if (message.realm !== "") {
      obj.realm = message.realm;
    }
    if (message.keytab !== "") {
      obj.keytab = message.keytab;
    }
    if (message.kdcHost !== "") {
      obj.kdcHost = message.kdcHost;
    }
    if (message.kdcTransportProtocol !== "") {
      obj.kdcTransportProtocol = message.kdcTransportProtocol;
    }
    return obj;
  },

  create(base?: DeepPartial<KerberosConfig>): KerberosConfig {
    return KerberosConfig.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<KerberosConfig>): KerberosConfig {
    const message = createBaseKerberosConfig();
    message.primary = object.primary ?? "";
    message.instance = object.instance ?? "";
    message.realm = object.realm ?? "";
    message.keytab = object.keytab ?? "";
    message.kdcHost = object.kdcHost ?? "";
    message.kdcTransportProtocol = object.kdcTransportProtocol ?? "";
    return message;
  },
};

function createBasePlainSASLConfig(): PlainSASLConfig {
  return { username: "", password: "" };
}

export const PlainSASLConfig = {
  encode(message: PlainSASLConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.username !== "") {
      writer.uint32(10).string(message.username);
    }
    if (message.password !== "") {
      writer.uint32(18).string(message.password);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): PlainSASLConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePlainSASLConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.username = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.password = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): PlainSASLConfig {
    return {
      username: isSet(object.username) ? globalThis.String(object.username) : "",
      password: isSet(object.password) ? globalThis.String(object.password) : "",
    };
  },

  toJSON(message: PlainSASLConfig): unknown {
    const obj: any = {};
    if (message.username !== "") {
      obj.username = message.username;
    }
    if (message.password !== "") {
      obj.password = message.password;
    }
    return obj;
  },

  create(base?: DeepPartial<PlainSASLConfig>): PlainSASLConfig {
    return PlainSASLConfig.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<PlainSASLConfig>): PlainSASLConfig {
    const message = createBasePlainSASLConfig();
    message.username = object.username ?? "";
    message.password = object.password ?? "";
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
