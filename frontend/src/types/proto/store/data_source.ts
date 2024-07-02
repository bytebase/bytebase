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
  SAECRET_TYPE_UNSPECIFIED = "SAECRET_TYPE_UNSPECIFIED",
  /** VAULT_KV_V2 - ref: https://developer.hashicorp.com/vault/api-docs/secret/kv/kv-v2 */
  VAULT_KV_V2 = "VAULT_KV_V2",
  /** AWS_SECRETS_MANAGER - ref: https://docs.aws.amazon.com/secretsmanager/latest/userguide/intro.html */
  AWS_SECRETS_MANAGER = "AWS_SECRETS_MANAGER",
  /** GCP_SECRET_MANAGER - ref: https://cloud.google.com/secret-manager/docs */
  GCP_SECRET_MANAGER = "GCP_SECRET_MANAGER",
  UNRECOGNIZED = "UNRECOGNIZED",
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

export function dataSourceExternalSecret_SecretTypeToNumber(object: DataSourceExternalSecret_SecretType): number {
  switch (object) {
    case DataSourceExternalSecret_SecretType.SAECRET_TYPE_UNSPECIFIED:
      return 0;
    case DataSourceExternalSecret_SecretType.VAULT_KV_V2:
      return 1;
    case DataSourceExternalSecret_SecretType.AWS_SECRETS_MANAGER:
      return 2;
    case DataSourceExternalSecret_SecretType.GCP_SECRET_MANAGER:
      return 3;
    case DataSourceExternalSecret_SecretType.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum DataSourceExternalSecret_AuthType {
  AUTH_TYPE_UNSPECIFIED = "AUTH_TYPE_UNSPECIFIED",
  /** TOKEN - ref: https://developer.hashicorp.com/vault/docs/auth/token */
  TOKEN = "TOKEN",
  /** VAULT_APP_ROLE - ref: https://developer.hashicorp.com/vault/docs/auth/approle */
  VAULT_APP_ROLE = "VAULT_APP_ROLE",
  UNRECOGNIZED = "UNRECOGNIZED",
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

export function dataSourceExternalSecret_AuthTypeToNumber(object: DataSourceExternalSecret_AuthType): number {
  switch (object) {
    case DataSourceExternalSecret_AuthType.AUTH_TYPE_UNSPECIFIED:
      return 0;
    case DataSourceExternalSecret_AuthType.TOKEN:
      return 1;
    case DataSourceExternalSecret_AuthType.VAULT_APP_ROLE:
      return 2;
    case DataSourceExternalSecret_AuthType.UNRECOGNIZED:
    default:
      return -1;
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
  SECRET_TYPE_UNSPECIFIED = "SECRET_TYPE_UNSPECIFIED",
  PLAIN = "PLAIN",
  ENVIRONMENT = "ENVIRONMENT",
  UNRECOGNIZED = "UNRECOGNIZED",
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

export function dataSourceExternalSecret_AppRoleAuthOption_SecretTypeToNumber(
  object: DataSourceExternalSecret_AppRoleAuthOption_SecretType,
): number {
  switch (object) {
    case DataSourceExternalSecret_AppRoleAuthOption_SecretType.SECRET_TYPE_UNSPECIFIED:
      return 0;
    case DataSourceExternalSecret_AppRoleAuthOption_SecretType.PLAIN:
      return 1;
    case DataSourceExternalSecret_AppRoleAuthOption_SecretType.ENVIRONMENT:
      return 2;
    case DataSourceExternalSecret_AppRoleAuthOption_SecretType.UNRECOGNIZED:
    default:
      return -1;
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
  saslConfig:
    | SASLConfig
    | undefined;
  /** additional_addresses is used for MongoDB replica set. */
  additionalAddresses: DataSourceOptions_Address[];
  /** replica_set is used for MongoDB replica set. */
  replicaSet: string;
  /** direct_connection is used for MongoDB to dispatch all the operations to the node specified in the connection string. */
  directConnection: boolean;
  /** region is the location of where the DB is, works for AWS RDS. For example, us-east-1. */
  region: string;
  /** account_id is used by Databricks. */
  accountId: string;
  /** warehouse_id is used by Databricks. */
  warehouseId: string;
  /** master_name is the master name used by connecting redis-master via redis sentinel. */
  masterName: string;
  /** master_username and master_obfuscated_password are master credentials used by redis sentinel mode. */
  masterUsername: string;
  masterObfuscatedPassword: string;
  redisType: DataSourceOptions_RedisType;
}

export enum DataSourceOptions_AuthenticationType {
  AUTHENTICATION_UNSPECIFIED = "AUTHENTICATION_UNSPECIFIED",
  PASSWORD = "PASSWORD",
  GOOGLE_CLOUD_SQL_IAM = "GOOGLE_CLOUD_SQL_IAM",
  AWS_RDS_IAM = "AWS_RDS_IAM",
  UNRECOGNIZED = "UNRECOGNIZED",
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
    case 3:
    case "AWS_RDS_IAM":
      return DataSourceOptions_AuthenticationType.AWS_RDS_IAM;
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
    case DataSourceOptions_AuthenticationType.AWS_RDS_IAM:
      return "AWS_RDS_IAM";
    case DataSourceOptions_AuthenticationType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function dataSourceOptions_AuthenticationTypeToNumber(object: DataSourceOptions_AuthenticationType): number {
  switch (object) {
    case DataSourceOptions_AuthenticationType.AUTHENTICATION_UNSPECIFIED:
      return 0;
    case DataSourceOptions_AuthenticationType.PASSWORD:
      return 1;
    case DataSourceOptions_AuthenticationType.GOOGLE_CLOUD_SQL_IAM:
      return 2;
    case DataSourceOptions_AuthenticationType.AWS_RDS_IAM:
      return 3;
    case DataSourceOptions_AuthenticationType.UNRECOGNIZED:
    default:
      return -1;
  }
}

export enum DataSourceOptions_RedisType {
  REDIS_TYPE_UNSPECIFIED = "REDIS_TYPE_UNSPECIFIED",
  STANDALONE = "STANDALONE",
  SENTINEL = "SENTINEL",
  CLUSTER = "CLUSTER",
  UNRECOGNIZED = "UNRECOGNIZED",
}

export function dataSourceOptions_RedisTypeFromJSON(object: any): DataSourceOptions_RedisType {
  switch (object) {
    case 0:
    case "REDIS_TYPE_UNSPECIFIED":
      return DataSourceOptions_RedisType.REDIS_TYPE_UNSPECIFIED;
    case 1:
    case "STANDALONE":
      return DataSourceOptions_RedisType.STANDALONE;
    case 2:
    case "SENTINEL":
      return DataSourceOptions_RedisType.SENTINEL;
    case 3:
    case "CLUSTER":
      return DataSourceOptions_RedisType.CLUSTER;
    case -1:
    case "UNRECOGNIZED":
    default:
      return DataSourceOptions_RedisType.UNRECOGNIZED;
  }
}

export function dataSourceOptions_RedisTypeToJSON(object: DataSourceOptions_RedisType): string {
  switch (object) {
    case DataSourceOptions_RedisType.REDIS_TYPE_UNSPECIFIED:
      return "REDIS_TYPE_UNSPECIFIED";
    case DataSourceOptions_RedisType.STANDALONE:
      return "STANDALONE";
    case DataSourceOptions_RedisType.SENTINEL:
      return "SENTINEL";
    case DataSourceOptions_RedisType.CLUSTER:
      return "CLUSTER";
    case DataSourceOptions_RedisType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export function dataSourceOptions_RedisTypeToNumber(object: DataSourceOptions_RedisType): number {
  switch (object) {
    case DataSourceOptions_RedisType.REDIS_TYPE_UNSPECIFIED:
      return 0;
    case DataSourceOptions_RedisType.STANDALONE:
      return 1;
    case DataSourceOptions_RedisType.SENTINEL:
      return 2;
    case DataSourceOptions_RedisType.CLUSTER:
      return 3;
    case DataSourceOptions_RedisType.UNRECOGNIZED:
    default:
      return -1;
  }
}

export interface DataSourceOptions_Address {
  host: string;
  port: string;
}

export interface SASLConfig {
  krbConfig?: KerberosConfig | undefined;
}

export interface KerberosConfig {
  primary: string;
  instance: string;
  realm: string;
  keytab: Uint8Array;
  kdcHost: string;
  kdcPort: string;
  kdcTransportProtocol: string;
}

function createBaseDataSourceExternalSecret(): DataSourceExternalSecret {
  return {
    secretType: DataSourceExternalSecret_SecretType.SAECRET_TYPE_UNSPECIFIED,
    url: "",
    authType: DataSourceExternalSecret_AuthType.AUTH_TYPE_UNSPECIFIED,
    appRole: undefined,
    token: undefined,
    engineName: "",
    secretName: "",
    passwordKeyName: "",
  };
}

export const DataSourceExternalSecret = {
  encode(message: DataSourceExternalSecret, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.secretType !== DataSourceExternalSecret_SecretType.SAECRET_TYPE_UNSPECIFIED) {
      writer.uint32(8).int32(dataSourceExternalSecret_SecretTypeToNumber(message.secretType));
    }
    if (message.url !== "") {
      writer.uint32(18).string(message.url);
    }
    if (message.authType !== DataSourceExternalSecret_AuthType.AUTH_TYPE_UNSPECIFIED) {
      writer.uint32(24).int32(dataSourceExternalSecret_AuthTypeToNumber(message.authType));
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

          message.secretType = dataSourceExternalSecret_SecretTypeFromJSON(reader.int32());
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

          message.authType = dataSourceExternalSecret_AuthTypeFromJSON(reader.int32());
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
      secretType: isSet(object.secretType)
        ? dataSourceExternalSecret_SecretTypeFromJSON(object.secretType)
        : DataSourceExternalSecret_SecretType.SAECRET_TYPE_UNSPECIFIED,
      url: isSet(object.url) ? globalThis.String(object.url) : "",
      authType: isSet(object.authType)
        ? dataSourceExternalSecret_AuthTypeFromJSON(object.authType)
        : DataSourceExternalSecret_AuthType.AUTH_TYPE_UNSPECIFIED,
      appRole: isSet(object.appRole) ? DataSourceExternalSecret_AppRoleAuthOption.fromJSON(object.appRole) : undefined,
      token: isSet(object.token) ? globalThis.String(object.token) : undefined,
      engineName: isSet(object.engineName) ? globalThis.String(object.engineName) : "",
      secretName: isSet(object.secretName) ? globalThis.String(object.secretName) : "",
      passwordKeyName: isSet(object.passwordKeyName) ? globalThis.String(object.passwordKeyName) : "",
    };
  },

  toJSON(message: DataSourceExternalSecret): unknown {
    const obj: any = {};
    if (message.secretType !== DataSourceExternalSecret_SecretType.SAECRET_TYPE_UNSPECIFIED) {
      obj.secretType = dataSourceExternalSecret_SecretTypeToJSON(message.secretType);
    }
    if (message.url !== "") {
      obj.url = message.url;
    }
    if (message.authType !== DataSourceExternalSecret_AuthType.AUTH_TYPE_UNSPECIFIED) {
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
    message.secretType = object.secretType ?? DataSourceExternalSecret_SecretType.SAECRET_TYPE_UNSPECIFIED;
    message.url = object.url ?? "";
    message.authType = object.authType ?? DataSourceExternalSecret_AuthType.AUTH_TYPE_UNSPECIFIED;
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
  return {
    roleId: "",
    secretId: "",
    type: DataSourceExternalSecret_AppRoleAuthOption_SecretType.SECRET_TYPE_UNSPECIFIED,
    mountPath: "",
  };
}

export const DataSourceExternalSecret_AppRoleAuthOption = {
  encode(message: DataSourceExternalSecret_AppRoleAuthOption, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.roleId !== "") {
      writer.uint32(10).string(message.roleId);
    }
    if (message.secretId !== "") {
      writer.uint32(18).string(message.secretId);
    }
    if (message.type !== DataSourceExternalSecret_AppRoleAuthOption_SecretType.SECRET_TYPE_UNSPECIFIED) {
      writer.uint32(24).int32(dataSourceExternalSecret_AppRoleAuthOption_SecretTypeToNumber(message.type));
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

          message.type = dataSourceExternalSecret_AppRoleAuthOption_SecretTypeFromJSON(reader.int32());
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
      type: isSet(object.type)
        ? dataSourceExternalSecret_AppRoleAuthOption_SecretTypeFromJSON(object.type)
        : DataSourceExternalSecret_AppRoleAuthOption_SecretType.SECRET_TYPE_UNSPECIFIED,
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
    if (message.type !== DataSourceExternalSecret_AppRoleAuthOption_SecretType.SECRET_TYPE_UNSPECIFIED) {
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
    message.type = object.type ?? DataSourceExternalSecret_AppRoleAuthOption_SecretType.SECRET_TYPE_UNSPECIFIED;
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
    authenticationType: DataSourceOptions_AuthenticationType.AUTHENTICATION_UNSPECIFIED,
    saslConfig: undefined,
    additionalAddresses: [],
    replicaSet: "",
    directConnection: false,
    region: "",
    accountId: "",
    warehouseId: "",
    masterName: "",
    masterUsername: "",
    masterObfuscatedPassword: "",
    redisType: DataSourceOptions_RedisType.REDIS_TYPE_UNSPECIFIED,
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
    if (message.authenticationType !== DataSourceOptions_AuthenticationType.AUTHENTICATION_UNSPECIFIED) {
      writer.uint32(96).int32(dataSourceOptions_AuthenticationTypeToNumber(message.authenticationType));
    }
    if (message.saslConfig !== undefined) {
      SASLConfig.encode(message.saslConfig, writer.uint32(106).fork()).ldelim();
    }
    for (const v of message.additionalAddresses) {
      DataSourceOptions_Address.encode(v!, writer.uint32(114).fork()).ldelim();
    }
    if (message.replicaSet !== "") {
      writer.uint32(122).string(message.replicaSet);
    }
    if (message.directConnection === true) {
      writer.uint32(128).bool(message.directConnection);
    }
    if (message.region !== "") {
      writer.uint32(138).string(message.region);
    }
    if (message.accountId !== "") {
      writer.uint32(146).string(message.accountId);
    }
    if (message.warehouseId !== "") {
      writer.uint32(154).string(message.warehouseId);
    }
    if (message.masterName !== "") {
      writer.uint32(162).string(message.masterName);
    }
    if (message.masterUsername !== "") {
      writer.uint32(170).string(message.masterUsername);
    }
    if (message.masterObfuscatedPassword !== "") {
      writer.uint32(178).string(message.masterObfuscatedPassword);
    }
    if (message.redisType !== DataSourceOptions_RedisType.REDIS_TYPE_UNSPECIFIED) {
      writer.uint32(184).int32(dataSourceOptions_RedisTypeToNumber(message.redisType));
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

          message.authenticationType = dataSourceOptions_AuthenticationTypeFromJSON(reader.int32());
          continue;
        case 13:
          if (tag !== 106) {
            break;
          }

          message.saslConfig = SASLConfig.decode(reader, reader.uint32());
          continue;
        case 14:
          if (tag !== 114) {
            break;
          }

          message.additionalAddresses.push(DataSourceOptions_Address.decode(reader, reader.uint32()));
          continue;
        case 15:
          if (tag !== 122) {
            break;
          }

          message.replicaSet = reader.string();
          continue;
        case 16:
          if (tag !== 128) {
            break;
          }

          message.directConnection = reader.bool();
          continue;
        case 17:
          if (tag !== 138) {
            break;
          }

          message.region = reader.string();
          continue;
        case 18:
          if (tag !== 146) {
            break;
          }

          message.accountId = reader.string();
          continue;
        case 19:
          if (tag !== 154) {
            break;
          }

          message.warehouseId = reader.string();
          continue;
        case 20:
          if (tag !== 162) {
            break;
          }

          message.masterName = reader.string();
          continue;
        case 21:
          if (tag !== 170) {
            break;
          }

          message.masterUsername = reader.string();
          continue;
        case 22:
          if (tag !== 178) {
            break;
          }

          message.masterObfuscatedPassword = reader.string();
          continue;
        case 23:
          if (tag !== 184) {
            break;
          }

          message.redisType = dataSourceOptions_RedisTypeFromJSON(reader.int32());
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
        : DataSourceOptions_AuthenticationType.AUTHENTICATION_UNSPECIFIED,
      saslConfig: isSet(object.saslConfig) ? SASLConfig.fromJSON(object.saslConfig) : undefined,
      additionalAddresses: globalThis.Array.isArray(object?.additionalAddresses)
        ? object.additionalAddresses.map((e: any) => DataSourceOptions_Address.fromJSON(e))
        : [],
      replicaSet: isSet(object.replicaSet) ? globalThis.String(object.replicaSet) : "",
      directConnection: isSet(object.directConnection) ? globalThis.Boolean(object.directConnection) : false,
      region: isSet(object.region) ? globalThis.String(object.region) : "",
      accountId: isSet(object.accountId) ? globalThis.String(object.accountId) : "",
      warehouseId: isSet(object.warehouseId) ? globalThis.String(object.warehouseId) : "",
      masterName: isSet(object.masterName) ? globalThis.String(object.masterName) : "",
      masterUsername: isSet(object.masterUsername) ? globalThis.String(object.masterUsername) : "",
      masterObfuscatedPassword: isSet(object.masterObfuscatedPassword)
        ? globalThis.String(object.masterObfuscatedPassword)
        : "",
      redisType: isSet(object.redisType)
        ? dataSourceOptions_RedisTypeFromJSON(object.redisType)
        : DataSourceOptions_RedisType.REDIS_TYPE_UNSPECIFIED,
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
    if (message.authenticationType !== DataSourceOptions_AuthenticationType.AUTHENTICATION_UNSPECIFIED) {
      obj.authenticationType = dataSourceOptions_AuthenticationTypeToJSON(message.authenticationType);
    }
    if (message.saslConfig !== undefined) {
      obj.saslConfig = SASLConfig.toJSON(message.saslConfig);
    }
    if (message.additionalAddresses?.length) {
      obj.additionalAddresses = message.additionalAddresses.map((e) => DataSourceOptions_Address.toJSON(e));
    }
    if (message.replicaSet !== "") {
      obj.replicaSet = message.replicaSet;
    }
    if (message.directConnection === true) {
      obj.directConnection = message.directConnection;
    }
    if (message.region !== "") {
      obj.region = message.region;
    }
    if (message.accountId !== "") {
      obj.accountId = message.accountId;
    }
    if (message.warehouseId !== "") {
      obj.warehouseId = message.warehouseId;
    }
    if (message.masterName !== "") {
      obj.masterName = message.masterName;
    }
    if (message.masterUsername !== "") {
      obj.masterUsername = message.masterUsername;
    }
    if (message.masterObfuscatedPassword !== "") {
      obj.masterObfuscatedPassword = message.masterObfuscatedPassword;
    }
    if (message.redisType !== DataSourceOptions_RedisType.REDIS_TYPE_UNSPECIFIED) {
      obj.redisType = dataSourceOptions_RedisTypeToJSON(message.redisType);
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
    message.authenticationType = object.authenticationType ??
      DataSourceOptions_AuthenticationType.AUTHENTICATION_UNSPECIFIED;
    message.saslConfig = (object.saslConfig !== undefined && object.saslConfig !== null)
      ? SASLConfig.fromPartial(object.saslConfig)
      : undefined;
    message.additionalAddresses = object.additionalAddresses?.map((e) => DataSourceOptions_Address.fromPartial(e)) ||
      [];
    message.replicaSet = object.replicaSet ?? "";
    message.directConnection = object.directConnection ?? false;
    message.region = object.region ?? "";
    message.accountId = object.accountId ?? "";
    message.warehouseId = object.warehouseId ?? "";
    message.masterName = object.masterName ?? "";
    message.masterUsername = object.masterUsername ?? "";
    message.masterObfuscatedPassword = object.masterObfuscatedPassword ?? "";
    message.redisType = object.redisType ?? DataSourceOptions_RedisType.REDIS_TYPE_UNSPECIFIED;
    return message;
  },
};

function createBaseDataSourceOptions_Address(): DataSourceOptions_Address {
  return { host: "", port: "" };
}

export const DataSourceOptions_Address = {
  encode(message: DataSourceOptions_Address, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.host !== "") {
      writer.uint32(10).string(message.host);
    }
    if (message.port !== "") {
      writer.uint32(18).string(message.port);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): DataSourceOptions_Address {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDataSourceOptions_Address();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.host = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.port = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DataSourceOptions_Address {
    return {
      host: isSet(object.host) ? globalThis.String(object.host) : "",
      port: isSet(object.port) ? globalThis.String(object.port) : "",
    };
  },

  toJSON(message: DataSourceOptions_Address): unknown {
    const obj: any = {};
    if (message.host !== "") {
      obj.host = message.host;
    }
    if (message.port !== "") {
      obj.port = message.port;
    }
    return obj;
  },

  create(base?: DeepPartial<DataSourceOptions_Address>): DataSourceOptions_Address {
    return DataSourceOptions_Address.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<DataSourceOptions_Address>): DataSourceOptions_Address {
    const message = createBaseDataSourceOptions_Address();
    message.host = object.host ?? "";
    message.port = object.port ?? "";
    return message;
  },
};

function createBaseSASLConfig(): SASLConfig {
  return { krbConfig: undefined };
}

export const SASLConfig = {
  encode(message: SASLConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.krbConfig !== undefined) {
      KerberosConfig.encode(message.krbConfig, writer.uint32(10).fork()).ldelim();
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
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SASLConfig {
    return { krbConfig: isSet(object.krbConfig) ? KerberosConfig.fromJSON(object.krbConfig) : undefined };
  },

  toJSON(message: SASLConfig): unknown {
    const obj: any = {};
    if (message.krbConfig !== undefined) {
      obj.krbConfig = KerberosConfig.toJSON(message.krbConfig);
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
    return message;
  },
};

function createBaseKerberosConfig(): KerberosConfig {
  return {
    primary: "",
    instance: "",
    realm: "",
    keytab: new Uint8Array(0),
    kdcHost: "",
    kdcPort: "",
    kdcTransportProtocol: "",
  };
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
    if (message.keytab.length !== 0) {
      writer.uint32(34).bytes(message.keytab);
    }
    if (message.kdcHost !== "") {
      writer.uint32(42).string(message.kdcHost);
    }
    if (message.kdcPort !== "") {
      writer.uint32(50).string(message.kdcPort);
    }
    if (message.kdcTransportProtocol !== "") {
      writer.uint32(58).string(message.kdcTransportProtocol);
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

          message.keytab = reader.bytes();
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

          message.kdcPort = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
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
      keytab: isSet(object.keytab) ? bytesFromBase64(object.keytab) : new Uint8Array(0),
      kdcHost: isSet(object.kdcHost) ? globalThis.String(object.kdcHost) : "",
      kdcPort: isSet(object.kdcPort) ? globalThis.String(object.kdcPort) : "",
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
    if (message.keytab.length !== 0) {
      obj.keytab = base64FromBytes(message.keytab);
    }
    if (message.kdcHost !== "") {
      obj.kdcHost = message.kdcHost;
    }
    if (message.kdcPort !== "") {
      obj.kdcPort = message.kdcPort;
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
    message.keytab = object.keytab ?? new Uint8Array(0);
    message.kdcHost = object.kdcHost ?? "";
    message.kdcPort = object.kdcPort ?? "";
    message.kdcTransportProtocol = object.kdcTransportProtocol ?? "";
    return message;
  },
};

function bytesFromBase64(b64: string): Uint8Array {
  if (globalThis.Buffer) {
    return Uint8Array.from(globalThis.Buffer.from(b64, "base64"));
  } else {
    const bin = globalThis.atob(b64);
    const arr = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; ++i) {
      arr[i] = bin.charCodeAt(i);
    }
    return arr;
  }
}

function base64FromBytes(arr: Uint8Array): string {
  if (globalThis.Buffer) {
    return globalThis.Buffer.from(arr).toString("base64");
  } else {
    const bin: string[] = [];
    arr.forEach((byte) => {
      bin.push(globalThis.String.fromCharCode(byte));
    });
    return globalThis.btoa(bin.join(""));
  }
}

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
