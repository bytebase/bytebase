/* eslint-disable */
import * as _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

/** FieldMapping uses for mapping user info from identity provider to Bytebase. */
export interface FieldMapping {
  identifier: string;
  name: string;
  email: string;
}

/** OAuth2IdentityProviderConfig is the structure for OAuth2 identity provider config. */
export interface OAuth2IdentityProviderConfig {
  authURL: string;
  tokenURL: string;
  userInfoURL: string;
  clientID: string;
  clientSecret: string;
  scopes: string;
  fieldMapping?: FieldMapping;
}

/** OIDCIdentityProviderConfig is the structure for OIDC identity provider config. */
export interface OIDCIdentityProviderConfig {
  issuer: string;
  clientID: string;
  clientSecret: string;
  fieldMapping?: FieldMapping;
}

function createBaseFieldMapping(): FieldMapping {
  return { identifier: "", name: "", email: "" };
}

export const FieldMapping = {
  encode(message: FieldMapping, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.identifier !== "") {
      writer.uint32(10).string(message.identifier);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.email !== "") {
      writer.uint32(26).string(message.email);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): FieldMapping {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFieldMapping();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.identifier = reader.string();
          break;
        case 2:
          message.name = reader.string();
          break;
        case 3:
          message.email = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): FieldMapping {
    return {
      identifier: isSet(object.identifier) ? String(object.identifier) : "",
      name: isSet(object.name) ? String(object.name) : "",
      email: isSet(object.email) ? String(object.email) : "",
    };
  },

  toJSON(message: FieldMapping): unknown {
    const obj: any = {};
    message.identifier !== undefined && (obj.identifier = message.identifier);
    message.name !== undefined && (obj.name = message.name);
    message.email !== undefined && (obj.email = message.email);
    return obj;
  },

  fromPartial(object: DeepPartial<FieldMapping>): FieldMapping {
    const message = createBaseFieldMapping();
    message.identifier = object.identifier ?? "";
    message.name = object.name ?? "";
    message.email = object.email ?? "";
    return message;
  },
};

function createBaseOAuth2IdentityProviderConfig(): OAuth2IdentityProviderConfig {
  return {
    authURL: "",
    tokenURL: "",
    userInfoURL: "",
    clientID: "",
    clientSecret: "",
    scopes: "",
    fieldMapping: undefined,
  };
}

export const OAuth2IdentityProviderConfig = {
  encode(message: OAuth2IdentityProviderConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.authURL !== "") {
      writer.uint32(10).string(message.authURL);
    }
    if (message.tokenURL !== "") {
      writer.uint32(18).string(message.tokenURL);
    }
    if (message.userInfoURL !== "") {
      writer.uint32(26).string(message.userInfoURL);
    }
    if (message.clientID !== "") {
      writer.uint32(34).string(message.clientID);
    }
    if (message.clientSecret !== "") {
      writer.uint32(42).string(message.clientSecret);
    }
    if (message.scopes !== "") {
      writer.uint32(50).string(message.scopes);
    }
    if (message.fieldMapping !== undefined) {
      FieldMapping.encode(message.fieldMapping, writer.uint32(58).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): OAuth2IdentityProviderConfig {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseOAuth2IdentityProviderConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authURL = reader.string();
          break;
        case 2:
          message.tokenURL = reader.string();
          break;
        case 3:
          message.userInfoURL = reader.string();
          break;
        case 4:
          message.clientID = reader.string();
          break;
        case 5:
          message.clientSecret = reader.string();
          break;
        case 6:
          message.scopes = reader.string();
          break;
        case 7:
          message.fieldMapping = FieldMapping.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): OAuth2IdentityProviderConfig {
    return {
      authURL: isSet(object.authURL) ? String(object.authURL) : "",
      tokenURL: isSet(object.tokenURL) ? String(object.tokenURL) : "",
      userInfoURL: isSet(object.userInfoURL) ? String(object.userInfoURL) : "",
      clientID: isSet(object.clientID) ? String(object.clientID) : "",
      clientSecret: isSet(object.clientSecret) ? String(object.clientSecret) : "",
      scopes: isSet(object.scopes) ? String(object.scopes) : "",
      fieldMapping: isSet(object.fieldMapping) ? FieldMapping.fromJSON(object.fieldMapping) : undefined,
    };
  },

  toJSON(message: OAuth2IdentityProviderConfig): unknown {
    const obj: any = {};
    message.authURL !== undefined && (obj.authURL = message.authURL);
    message.tokenURL !== undefined && (obj.tokenURL = message.tokenURL);
    message.userInfoURL !== undefined && (obj.userInfoURL = message.userInfoURL);
    message.clientID !== undefined && (obj.clientID = message.clientID);
    message.clientSecret !== undefined && (obj.clientSecret = message.clientSecret);
    message.scopes !== undefined && (obj.scopes = message.scopes);
    message.fieldMapping !== undefined &&
      (obj.fieldMapping = message.fieldMapping ? FieldMapping.toJSON(message.fieldMapping) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<OAuth2IdentityProviderConfig>): OAuth2IdentityProviderConfig {
    const message = createBaseOAuth2IdentityProviderConfig();
    message.authURL = object.authURL ?? "";
    message.tokenURL = object.tokenURL ?? "";
    message.userInfoURL = object.userInfoURL ?? "";
    message.clientID = object.clientID ?? "";
    message.clientSecret = object.clientSecret ?? "";
    message.scopes = object.scopes ?? "";
    message.fieldMapping = (object.fieldMapping !== undefined && object.fieldMapping !== null)
      ? FieldMapping.fromPartial(object.fieldMapping)
      : undefined;
    return message;
  },
};

function createBaseOIDCIdentityProviderConfig(): OIDCIdentityProviderConfig {
  return { issuer: "", clientID: "", clientSecret: "", fieldMapping: undefined };
}

export const OIDCIdentityProviderConfig = {
  encode(message: OIDCIdentityProviderConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.issuer !== "") {
      writer.uint32(10).string(message.issuer);
    }
    if (message.clientID !== "") {
      writer.uint32(18).string(message.clientID);
    }
    if (message.clientSecret !== "") {
      writer.uint32(26).string(message.clientSecret);
    }
    if (message.fieldMapping !== undefined) {
      FieldMapping.encode(message.fieldMapping, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): OIDCIdentityProviderConfig {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseOIDCIdentityProviderConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.issuer = reader.string();
          break;
        case 2:
          message.clientID = reader.string();
          break;
        case 3:
          message.clientSecret = reader.string();
          break;
        case 4:
          message.fieldMapping = FieldMapping.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): OIDCIdentityProviderConfig {
    return {
      issuer: isSet(object.issuer) ? String(object.issuer) : "",
      clientID: isSet(object.clientID) ? String(object.clientID) : "",
      clientSecret: isSet(object.clientSecret) ? String(object.clientSecret) : "",
      fieldMapping: isSet(object.fieldMapping) ? FieldMapping.fromJSON(object.fieldMapping) : undefined,
    };
  },

  toJSON(message: OIDCIdentityProviderConfig): unknown {
    const obj: any = {};
    message.issuer !== undefined && (obj.issuer = message.issuer);
    message.clientID !== undefined && (obj.clientID = message.clientID);
    message.clientSecret !== undefined && (obj.clientSecret = message.clientSecret);
    message.fieldMapping !== undefined &&
      (obj.fieldMapping = message.fieldMapping ? FieldMapping.toJSON(message.fieldMapping) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<OIDCIdentityProviderConfig>): OIDCIdentityProviderConfig {
    const message = createBaseOIDCIdentityProviderConfig();
    message.issuer = object.issuer ?? "";
    message.clientID = object.clientID ?? "";
    message.clientSecret = object.clientSecret ?? "";
    message.fieldMapping = (object.fieldMapping !== undefined && object.fieldMapping !== null)
      ? FieldMapping.fromPartial(object.fieldMapping)
      : undefined;
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
