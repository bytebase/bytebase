/* eslint-disable */
import * as _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export interface DataSourceOptions {
  /** srv is a boolean flag that indicates whether the host is a DNS SRV record. */
  srv: boolean;
  /** authentication_database is the database name to authenticate against, which stores the user credentials. */
  authenticationDatabase: string;
  /** sid and service_name are used for Oracle. */
  sid: string;
  serviceName: string;
}

export interface SSHConfig {
  /** The hostname of the SSH server agent. */
  host: string;
  /** The port of the SSH server agent. It's 22 typically. */
  port: string;
  /** The user to login the server. */
  user: string;
  /** The password to login the server. If it's empty string, no password is required. */
  password: string;
  /** The private key to login the server. If it's empty string, we will use the system default private key from os.Getenv("SSH_AUTH_SOCK"). */
  privateKey: string;
}

function createBaseDataSourceOptions(): DataSourceOptions {
  return { srv: false, authenticationDatabase: "", sid: "", serviceName: "" };
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
          if (tag != 8) {
            break;
          }

          message.srv = reader.bool();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.authenticationDatabase = reader.string();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.sid = reader.string();
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.serviceName = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): DataSourceOptions {
    return {
      srv: isSet(object.srv) ? Boolean(object.srv) : false,
      authenticationDatabase: isSet(object.authenticationDatabase) ? String(object.authenticationDatabase) : "",
      sid: isSet(object.sid) ? String(object.sid) : "",
      serviceName: isSet(object.serviceName) ? String(object.serviceName) : "",
    };
  },

  toJSON(message: DataSourceOptions): unknown {
    const obj: any = {};
    message.srv !== undefined && (obj.srv = message.srv);
    message.authenticationDatabase !== undefined && (obj.authenticationDatabase = message.authenticationDatabase);
    message.sid !== undefined && (obj.sid = message.sid);
    message.serviceName !== undefined && (obj.serviceName = message.serviceName);
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
    return message;
  },
};

function createBaseSSHConfig(): SSHConfig {
  return { host: "", port: "", user: "", password: "", privateKey: "" };
}

export const SSHConfig = {
  encode(message: SSHConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.host !== "") {
      writer.uint32(10).string(message.host);
    }
    if (message.port !== "") {
      writer.uint32(18).string(message.port);
    }
    if (message.user !== "") {
      writer.uint32(26).string(message.user);
    }
    if (message.password !== "") {
      writer.uint32(34).string(message.password);
    }
    if (message.privateKey !== "") {
      writer.uint32(42).string(message.privateKey);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SSHConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSSHConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag != 10) {
            break;
          }

          message.host = reader.string();
          continue;
        case 2:
          if (tag != 18) {
            break;
          }

          message.port = reader.string();
          continue;
        case 3:
          if (tag != 26) {
            break;
          }

          message.user = reader.string();
          continue;
        case 4:
          if (tag != 34) {
            break;
          }

          message.password = reader.string();
          continue;
        case 5:
          if (tag != 42) {
            break;
          }

          message.privateKey = reader.string();
          continue;
      }
      if ((tag & 7) == 4 || tag == 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): SSHConfig {
    return {
      host: isSet(object.host) ? String(object.host) : "",
      port: isSet(object.port) ? String(object.port) : "",
      user: isSet(object.user) ? String(object.user) : "",
      password: isSet(object.password) ? String(object.password) : "",
      privateKey: isSet(object.privateKey) ? String(object.privateKey) : "",
    };
  },

  toJSON(message: SSHConfig): unknown {
    const obj: any = {};
    message.host !== undefined && (obj.host = message.host);
    message.port !== undefined && (obj.port = message.port);
    message.user !== undefined && (obj.user = message.user);
    message.password !== undefined && (obj.password = message.password);
    message.privateKey !== undefined && (obj.privateKey = message.privateKey);
    return obj;
  },

  create(base?: DeepPartial<SSHConfig>): SSHConfig {
    return SSHConfig.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<SSHConfig>): SSHConfig {
    const message = createBaseSSHConfig();
    message.host = object.host ?? "";
    message.port = object.port ?? "";
    message.user = object.user ?? "";
    message.password = object.password ?? "";
    message.privateKey = object.privateKey ?? "";
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
