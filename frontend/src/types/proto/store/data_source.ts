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
  keyName: string;
}

export enum DataSourceExternalSecret_SecretType {
  SAECRET_TYPE_UNSPECIFIED = 0,
  VAULT_KV_V2 = 1,
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
    case DataSourceExternalSecret_SecretType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum DataSourceExternalSecret_AuthType {
  AUTH_TYPE_UNSPECIFIED = 0,
  TOKEN = 1,
  APP_ROLE = 2,
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
    case "APP_ROLE":
      return DataSourceExternalSecret_AuthType.APP_ROLE;
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
    case DataSourceExternalSecret_AuthType.APP_ROLE:
      return "APP_ROLE";
    case DataSourceExternalSecret_AuthType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

/** app role auth method: https://developer.hashicorp.com/vault/docs/auth/approle */
export interface DataSourceExternalSecret_AppRoleAuthOption {
  roleId: string;
  secretId: string;
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
    keyName: "",
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
    if (message.keyName !== "") {
      writer.uint32(66).string(message.keyName);
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

          message.keyName = reader.string();
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
      keyName: isSet(object.keyName) ? globalThis.String(object.keyName) : "",
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
    if (message.keyName !== "") {
      obj.keyName = message.keyName;
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
    message.keyName = object.keyName ?? "";
    return message;
  },
};

function createBaseDataSourceExternalSecret_AppRoleAuthOption(): DataSourceExternalSecret_AppRoleAuthOption {
  return { roleId: "", secretId: "" };
}

export const DataSourceExternalSecret_AppRoleAuthOption = {
  encode(message: DataSourceExternalSecret_AppRoleAuthOption, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.roleId !== "") {
      writer.uint32(10).string(message.roleId);
    }
    if (message.secretId !== "") {
      writer.uint32(18).string(message.secretId);
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
