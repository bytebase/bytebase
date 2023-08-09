/* eslint-disable */
import * as _m0 from "protobufjs/minimal";

export const protobufPackage = "bytebase.store";

export enum IdentityProviderType {
  IDENTITY_PROVIDER_TYPE_UNSPECIFIED = 0,
  OAUTH2 = 1,
  OIDC = 2,
  LDAP = 3,
  UNRECOGNIZED = -1,
}

export function identityProviderTypeFromJSON(object: any): IdentityProviderType {
  switch (object) {
    case 0:
    case "IDENTITY_PROVIDER_TYPE_UNSPECIFIED":
      return IdentityProviderType.IDENTITY_PROVIDER_TYPE_UNSPECIFIED;
    case 1:
    case "OAUTH2":
      return IdentityProviderType.OAUTH2;
    case 2:
    case "OIDC":
      return IdentityProviderType.OIDC;
    case 3:
    case "LDAP":
      return IdentityProviderType.LDAP;
    case -1:
    case "UNRECOGNIZED":
    default:
      return IdentityProviderType.UNRECOGNIZED;
  }
}

export function identityProviderTypeToJSON(object: IdentityProviderType): string {
  switch (object) {
    case IdentityProviderType.IDENTITY_PROVIDER_TYPE_UNSPECIFIED:
      return "IDENTITY_PROVIDER_TYPE_UNSPECIFIED";
    case IdentityProviderType.OAUTH2:
      return "OAUTH2";
    case IdentityProviderType.OIDC:
      return "OIDC";
    case IdentityProviderType.LDAP:
      return "LDAP";
    case IdentityProviderType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export enum OAuth2AuthStyle {
  OAUTH2_AUTH_STYLE_UNSPECIFIED = 0,
  /**
   * IN_PARAMS - IN_PARAMS sends the "client_id" and "client_secret" in the POST body
   * as application/x-www-form-urlencoded parameters.
   */
  IN_PARAMS = 1,
  /**
   * IN_HEADER - IN_HEADER sends the client_id and client_password using HTTP Basic Authorization.
   * This is an optional style described in the OAuth2 RFC 6749 section 2.3.1.
   */
  IN_HEADER = 2,
  UNRECOGNIZED = -1,
}

export function oAuth2AuthStyleFromJSON(object: any): OAuth2AuthStyle {
  switch (object) {
    case 0:
    case "OAUTH2_AUTH_STYLE_UNSPECIFIED":
      return OAuth2AuthStyle.OAUTH2_AUTH_STYLE_UNSPECIFIED;
    case 1:
    case "IN_PARAMS":
      return OAuth2AuthStyle.IN_PARAMS;
    case 2:
    case "IN_HEADER":
      return OAuth2AuthStyle.IN_HEADER;
    case -1:
    case "UNRECOGNIZED":
    default:
      return OAuth2AuthStyle.UNRECOGNIZED;
  }
}

export function oAuth2AuthStyleToJSON(object: OAuth2AuthStyle): string {
  switch (object) {
    case OAuth2AuthStyle.OAUTH2_AUTH_STYLE_UNSPECIFIED:
      return "OAUTH2_AUTH_STYLE_UNSPECIFIED";
    case OAuth2AuthStyle.IN_PARAMS:
      return "IN_PARAMS";
    case OAuth2AuthStyle.IN_HEADER:
      return "IN_HEADER";
    case OAuth2AuthStyle.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface IdentityProviderConfig {
  oauth2Config?: OAuth2IdentityProviderConfig | undefined;
  oidcConfig?: OIDCIdentityProviderConfig | undefined;
  ldapConfig?: LDAPIdentityProviderConfig | undefined;
}

/** OAuth2IdentityProviderConfig is the structure for OAuth2 identity provider config. */
export interface OAuth2IdentityProviderConfig {
  authUrl: string;
  tokenUrl: string;
  userInfoUrl: string;
  clientId: string;
  clientSecret: string;
  scopes: string[];
  fieldMapping?: FieldMapping | undefined;
  skipTlsVerify: boolean;
  authStyle: OAuth2AuthStyle;
}

/** OIDCIdentityProviderConfig is the structure for OIDC identity provider config. */
export interface OIDCIdentityProviderConfig {
  issuer: string;
  clientId: string;
  clientSecret: string;
  fieldMapping?: FieldMapping | undefined;
  skipTlsVerify: boolean;
  authStyle: OAuth2AuthStyle;
}

/** LDAPIdentityProviderConfig is the structure for LDAP identity provider config. */
export interface LDAPIdentityProviderConfig {
  /**
   * Host is the hostname or IP address of the LDAP server, e.g.
   * "ldap.example.com".
   */
  host: string;
  /**
   * Port is the port number of the LDAP server, e.g. 389. When not set, the
   * default port of the corresponding security protocol will be used, i.e. 389
   * for StartTLS and 636 for LDAPS.
   */
  port: number;
  /** SkipTLSVerify controls whether to skip TLS certificate verification. */
  skipTlsVerify: boolean;
  /**
   * BindDN is the DN of the user to bind as a service account to perform
   * search requests.
   */
  bindDn: string;
  /** BindPassword is the password of the user to bind as a service account. */
  bindPassword: string;
  /** BaseDN is the base DN to search for users, e.g. "ou=users,dc=example,dc=com". */
  baseDn: string;
  /** UserFilter is the filter to search for users, e.g. "(uid=%s)". */
  userFilter: string;
  /**
   * SecurityProtocol is the security protocol to be used for establishing
   * connections with the LDAP server. It should be either StartTLS or LDAPS, and
   * cannot be empty.
   */
  securityProtocol: string;
  /**
   * FieldMapping is the mapping of the user attributes returned by the LDAP
   * server.
   */
  fieldMapping?: FieldMapping | undefined;
}

/**
 * FieldMapping saves the field names from user info API of identity provider.
 * As we save all raw json string of user info response data into `principal.idp_user_info`,
 * we can extract the relevant data based with `FieldMapping`.
 *
 * e.g. For GitHub authenticated user API, it will return `login`, `name` and `email` in response.
 * Then the identifier of FieldMapping will be `login`, display_name will be `name`,
 * and email will be `email`.
 * reference: https://docs.github.com/en/rest/users/users?apiVersion=2022-11-28#get-the-authenticated-user
 */
export interface FieldMapping {
  /** Identifier is the field name of the unique identifier in 3rd-party idp user info. Required. */
  identifier: string;
  /** DisplayName is the field name of display name in 3rd-party idp user info. Optional. */
  displayName: string;
  /** Email is the field name of primary email in 3rd-party idp user info. Optional. */
  email: string;
  /** Phone is the field name of primary phone in 3rd-party idp user info. Optional. */
  phone: string;
}

export interface IdentityProviderUserInfo {
  /** Identifier is the value of the unique identifier in 3rd-party idp user info. */
  identifier: string;
  /** DisplayName is the value of display name in 3rd-party idp user info. */
  displayName: string;
  /** Email is the value of primary email in 3rd-party idp user info. */
  email: string;
  /** Phone is the value of primary phone in 3rd-party idp user info. */
  phone: string;
}

function createBaseIdentityProviderConfig(): IdentityProviderConfig {
  return { oauth2Config: undefined, oidcConfig: undefined, ldapConfig: undefined };
}

export const IdentityProviderConfig = {
  encode(message: IdentityProviderConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.oauth2Config !== undefined) {
      OAuth2IdentityProviderConfig.encode(message.oauth2Config, writer.uint32(10).fork()).ldelim();
    }
    if (message.oidcConfig !== undefined) {
      OIDCIdentityProviderConfig.encode(message.oidcConfig, writer.uint32(18).fork()).ldelim();
    }
    if (message.ldapConfig !== undefined) {
      LDAPIdentityProviderConfig.encode(message.ldapConfig, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IdentityProviderConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIdentityProviderConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.oauth2Config = OAuth2IdentityProviderConfig.decode(reader, reader.uint32());
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.oidcConfig = OIDCIdentityProviderConfig.decode(reader, reader.uint32());
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.ldapConfig = LDAPIdentityProviderConfig.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): IdentityProviderConfig {
    return {
      oauth2Config: isSet(object.oauth2Config) ? OAuth2IdentityProviderConfig.fromJSON(object.oauth2Config) : undefined,
      oidcConfig: isSet(object.oidcConfig) ? OIDCIdentityProviderConfig.fromJSON(object.oidcConfig) : undefined,
      ldapConfig: isSet(object.ldapConfig) ? LDAPIdentityProviderConfig.fromJSON(object.ldapConfig) : undefined,
    };
  },

  toJSON(message: IdentityProviderConfig): unknown {
    const obj: any = {};
    message.oauth2Config !== undefined &&
      (obj.oauth2Config = message.oauth2Config ? OAuth2IdentityProviderConfig.toJSON(message.oauth2Config) : undefined);
    message.oidcConfig !== undefined &&
      (obj.oidcConfig = message.oidcConfig ? OIDCIdentityProviderConfig.toJSON(message.oidcConfig) : undefined);
    message.ldapConfig !== undefined &&
      (obj.ldapConfig = message.ldapConfig ? LDAPIdentityProviderConfig.toJSON(message.ldapConfig) : undefined);
    return obj;
  },

  create(base?: DeepPartial<IdentityProviderConfig>): IdentityProviderConfig {
    return IdentityProviderConfig.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<IdentityProviderConfig>): IdentityProviderConfig {
    const message = createBaseIdentityProviderConfig();
    message.oauth2Config = (object.oauth2Config !== undefined && object.oauth2Config !== null)
      ? OAuth2IdentityProviderConfig.fromPartial(object.oauth2Config)
      : undefined;
    message.oidcConfig = (object.oidcConfig !== undefined && object.oidcConfig !== null)
      ? OIDCIdentityProviderConfig.fromPartial(object.oidcConfig)
      : undefined;
    message.ldapConfig = (object.ldapConfig !== undefined && object.ldapConfig !== null)
      ? LDAPIdentityProviderConfig.fromPartial(object.ldapConfig)
      : undefined;
    return message;
  },
};

function createBaseOAuth2IdentityProviderConfig(): OAuth2IdentityProviderConfig {
  return {
    authUrl: "",
    tokenUrl: "",
    userInfoUrl: "",
    clientId: "",
    clientSecret: "",
    scopes: [],
    fieldMapping: undefined,
    skipTlsVerify: false,
    authStyle: 0,
  };
}

export const OAuth2IdentityProviderConfig = {
  encode(message: OAuth2IdentityProviderConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.authUrl !== "") {
      writer.uint32(10).string(message.authUrl);
    }
    if (message.tokenUrl !== "") {
      writer.uint32(18).string(message.tokenUrl);
    }
    if (message.userInfoUrl !== "") {
      writer.uint32(26).string(message.userInfoUrl);
    }
    if (message.clientId !== "") {
      writer.uint32(34).string(message.clientId);
    }
    if (message.clientSecret !== "") {
      writer.uint32(42).string(message.clientSecret);
    }
    for (const v of message.scopes) {
      writer.uint32(50).string(v!);
    }
    if (message.fieldMapping !== undefined) {
      FieldMapping.encode(message.fieldMapping, writer.uint32(58).fork()).ldelim();
    }
    if (message.skipTlsVerify === true) {
      writer.uint32(64).bool(message.skipTlsVerify);
    }
    if (message.authStyle !== 0) {
      writer.uint32(72).int32(message.authStyle);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): OAuth2IdentityProviderConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseOAuth2IdentityProviderConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.authUrl = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.tokenUrl = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.userInfoUrl = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.clientId = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.clientSecret = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.scopes.push(reader.string());
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.fieldMapping = FieldMapping.decode(reader, reader.uint32());
          continue;
        case 8:
          if (tag !== 64) {
            break;
          }

          message.skipTlsVerify = reader.bool();
          continue;
        case 9:
          if (tag !== 72) {
            break;
          }

          message.authStyle = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): OAuth2IdentityProviderConfig {
    return {
      authUrl: isSet(object.authUrl) ? String(object.authUrl) : "",
      tokenUrl: isSet(object.tokenUrl) ? String(object.tokenUrl) : "",
      userInfoUrl: isSet(object.userInfoUrl) ? String(object.userInfoUrl) : "",
      clientId: isSet(object.clientId) ? String(object.clientId) : "",
      clientSecret: isSet(object.clientSecret) ? String(object.clientSecret) : "",
      scopes: Array.isArray(object?.scopes) ? object.scopes.map((e: any) => String(e)) : [],
      fieldMapping: isSet(object.fieldMapping) ? FieldMapping.fromJSON(object.fieldMapping) : undefined,
      skipTlsVerify: isSet(object.skipTlsVerify) ? Boolean(object.skipTlsVerify) : false,
      authStyle: isSet(object.authStyle) ? oAuth2AuthStyleFromJSON(object.authStyle) : 0,
    };
  },

  toJSON(message: OAuth2IdentityProviderConfig): unknown {
    const obj: any = {};
    message.authUrl !== undefined && (obj.authUrl = message.authUrl);
    message.tokenUrl !== undefined && (obj.tokenUrl = message.tokenUrl);
    message.userInfoUrl !== undefined && (obj.userInfoUrl = message.userInfoUrl);
    message.clientId !== undefined && (obj.clientId = message.clientId);
    message.clientSecret !== undefined && (obj.clientSecret = message.clientSecret);
    if (message.scopes) {
      obj.scopes = message.scopes.map((e) => e);
    } else {
      obj.scopes = [];
    }
    message.fieldMapping !== undefined &&
      (obj.fieldMapping = message.fieldMapping ? FieldMapping.toJSON(message.fieldMapping) : undefined);
    message.skipTlsVerify !== undefined && (obj.skipTlsVerify = message.skipTlsVerify);
    message.authStyle !== undefined && (obj.authStyle = oAuth2AuthStyleToJSON(message.authStyle));
    return obj;
  },

  create(base?: DeepPartial<OAuth2IdentityProviderConfig>): OAuth2IdentityProviderConfig {
    return OAuth2IdentityProviderConfig.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<OAuth2IdentityProviderConfig>): OAuth2IdentityProviderConfig {
    const message = createBaseOAuth2IdentityProviderConfig();
    message.authUrl = object.authUrl ?? "";
    message.tokenUrl = object.tokenUrl ?? "";
    message.userInfoUrl = object.userInfoUrl ?? "";
    message.clientId = object.clientId ?? "";
    message.clientSecret = object.clientSecret ?? "";
    message.scopes = object.scopes?.map((e) => e) || [];
    message.fieldMapping = (object.fieldMapping !== undefined && object.fieldMapping !== null)
      ? FieldMapping.fromPartial(object.fieldMapping)
      : undefined;
    message.skipTlsVerify = object.skipTlsVerify ?? false;
    message.authStyle = object.authStyle ?? 0;
    return message;
  },
};

function createBaseOIDCIdentityProviderConfig(): OIDCIdentityProviderConfig {
  return { issuer: "", clientId: "", clientSecret: "", fieldMapping: undefined, skipTlsVerify: false, authStyle: 0 };
}

export const OIDCIdentityProviderConfig = {
  encode(message: OIDCIdentityProviderConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.issuer !== "") {
      writer.uint32(10).string(message.issuer);
    }
    if (message.clientId !== "") {
      writer.uint32(18).string(message.clientId);
    }
    if (message.clientSecret !== "") {
      writer.uint32(26).string(message.clientSecret);
    }
    if (message.fieldMapping !== undefined) {
      FieldMapping.encode(message.fieldMapping, writer.uint32(34).fork()).ldelim();
    }
    if (message.skipTlsVerify === true) {
      writer.uint32(40).bool(message.skipTlsVerify);
    }
    if (message.authStyle !== 0) {
      writer.uint32(48).int32(message.authStyle);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): OIDCIdentityProviderConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseOIDCIdentityProviderConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.issuer = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.clientId = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.clientSecret = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.fieldMapping = FieldMapping.decode(reader, reader.uint32());
          continue;
        case 5:
          if (tag !== 40) {
            break;
          }

          message.skipTlsVerify = reader.bool();
          continue;
        case 6:
          if (tag !== 48) {
            break;
          }

          message.authStyle = reader.int32() as any;
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): OIDCIdentityProviderConfig {
    return {
      issuer: isSet(object.issuer) ? String(object.issuer) : "",
      clientId: isSet(object.clientId) ? String(object.clientId) : "",
      clientSecret: isSet(object.clientSecret) ? String(object.clientSecret) : "",
      fieldMapping: isSet(object.fieldMapping) ? FieldMapping.fromJSON(object.fieldMapping) : undefined,
      skipTlsVerify: isSet(object.skipTlsVerify) ? Boolean(object.skipTlsVerify) : false,
      authStyle: isSet(object.authStyle) ? oAuth2AuthStyleFromJSON(object.authStyle) : 0,
    };
  },

  toJSON(message: OIDCIdentityProviderConfig): unknown {
    const obj: any = {};
    message.issuer !== undefined && (obj.issuer = message.issuer);
    message.clientId !== undefined && (obj.clientId = message.clientId);
    message.clientSecret !== undefined && (obj.clientSecret = message.clientSecret);
    message.fieldMapping !== undefined &&
      (obj.fieldMapping = message.fieldMapping ? FieldMapping.toJSON(message.fieldMapping) : undefined);
    message.skipTlsVerify !== undefined && (obj.skipTlsVerify = message.skipTlsVerify);
    message.authStyle !== undefined && (obj.authStyle = oAuth2AuthStyleToJSON(message.authStyle));
    return obj;
  },

  create(base?: DeepPartial<OIDCIdentityProviderConfig>): OIDCIdentityProviderConfig {
    return OIDCIdentityProviderConfig.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<OIDCIdentityProviderConfig>): OIDCIdentityProviderConfig {
    const message = createBaseOIDCIdentityProviderConfig();
    message.issuer = object.issuer ?? "";
    message.clientId = object.clientId ?? "";
    message.clientSecret = object.clientSecret ?? "";
    message.fieldMapping = (object.fieldMapping !== undefined && object.fieldMapping !== null)
      ? FieldMapping.fromPartial(object.fieldMapping)
      : undefined;
    message.skipTlsVerify = object.skipTlsVerify ?? false;
    message.authStyle = object.authStyle ?? 0;
    return message;
  },
};

function createBaseLDAPIdentityProviderConfig(): LDAPIdentityProviderConfig {
  return {
    host: "",
    port: 0,
    skipTlsVerify: false,
    bindDn: "",
    bindPassword: "",
    baseDn: "",
    userFilter: "",
    securityProtocol: "",
    fieldMapping: undefined,
  };
}

export const LDAPIdentityProviderConfig = {
  encode(message: LDAPIdentityProviderConfig, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.host !== "") {
      writer.uint32(10).string(message.host);
    }
    if (message.port !== 0) {
      writer.uint32(16).int32(message.port);
    }
    if (message.skipTlsVerify === true) {
      writer.uint32(24).bool(message.skipTlsVerify);
    }
    if (message.bindDn !== "") {
      writer.uint32(34).string(message.bindDn);
    }
    if (message.bindPassword !== "") {
      writer.uint32(42).string(message.bindPassword);
    }
    if (message.baseDn !== "") {
      writer.uint32(50).string(message.baseDn);
    }
    if (message.userFilter !== "") {
      writer.uint32(58).string(message.userFilter);
    }
    if (message.securityProtocol !== "") {
      writer.uint32(66).string(message.securityProtocol);
    }
    if (message.fieldMapping !== undefined) {
      FieldMapping.encode(message.fieldMapping, writer.uint32(74).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): LDAPIdentityProviderConfig {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseLDAPIdentityProviderConfig();
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
          if (tag !== 16) {
            break;
          }

          message.port = reader.int32();
          continue;
        case 3:
          if (tag !== 24) {
            break;
          }

          message.skipTlsVerify = reader.bool();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.bindDn = reader.string();
          continue;
        case 5:
          if (tag !== 42) {
            break;
          }

          message.bindPassword = reader.string();
          continue;
        case 6:
          if (tag !== 50) {
            break;
          }

          message.baseDn = reader.string();
          continue;
        case 7:
          if (tag !== 58) {
            break;
          }

          message.userFilter = reader.string();
          continue;
        case 8:
          if (tag !== 66) {
            break;
          }

          message.securityProtocol = reader.string();
          continue;
        case 9:
          if (tag !== 74) {
            break;
          }

          message.fieldMapping = FieldMapping.decode(reader, reader.uint32());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): LDAPIdentityProviderConfig {
    return {
      host: isSet(object.host) ? String(object.host) : "",
      port: isSet(object.port) ? Number(object.port) : 0,
      skipTlsVerify: isSet(object.skipTlsVerify) ? Boolean(object.skipTlsVerify) : false,
      bindDn: isSet(object.bindDn) ? String(object.bindDn) : "",
      bindPassword: isSet(object.bindPassword) ? String(object.bindPassword) : "",
      baseDn: isSet(object.baseDn) ? String(object.baseDn) : "",
      userFilter: isSet(object.userFilter) ? String(object.userFilter) : "",
      securityProtocol: isSet(object.securityProtocol) ? String(object.securityProtocol) : "",
      fieldMapping: isSet(object.fieldMapping) ? FieldMapping.fromJSON(object.fieldMapping) : undefined,
    };
  },

  toJSON(message: LDAPIdentityProviderConfig): unknown {
    const obj: any = {};
    message.host !== undefined && (obj.host = message.host);
    message.port !== undefined && (obj.port = Math.round(message.port));
    message.skipTlsVerify !== undefined && (obj.skipTlsVerify = message.skipTlsVerify);
    message.bindDn !== undefined && (obj.bindDn = message.bindDn);
    message.bindPassword !== undefined && (obj.bindPassword = message.bindPassword);
    message.baseDn !== undefined && (obj.baseDn = message.baseDn);
    message.userFilter !== undefined && (obj.userFilter = message.userFilter);
    message.securityProtocol !== undefined && (obj.securityProtocol = message.securityProtocol);
    message.fieldMapping !== undefined &&
      (obj.fieldMapping = message.fieldMapping ? FieldMapping.toJSON(message.fieldMapping) : undefined);
    return obj;
  },

  create(base?: DeepPartial<LDAPIdentityProviderConfig>): LDAPIdentityProviderConfig {
    return LDAPIdentityProviderConfig.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<LDAPIdentityProviderConfig>): LDAPIdentityProviderConfig {
    const message = createBaseLDAPIdentityProviderConfig();
    message.host = object.host ?? "";
    message.port = object.port ?? 0;
    message.skipTlsVerify = object.skipTlsVerify ?? false;
    message.bindDn = object.bindDn ?? "";
    message.bindPassword = object.bindPassword ?? "";
    message.baseDn = object.baseDn ?? "";
    message.userFilter = object.userFilter ?? "";
    message.securityProtocol = object.securityProtocol ?? "";
    message.fieldMapping = (object.fieldMapping !== undefined && object.fieldMapping !== null)
      ? FieldMapping.fromPartial(object.fieldMapping)
      : undefined;
    return message;
  },
};

function createBaseFieldMapping(): FieldMapping {
  return { identifier: "", displayName: "", email: "", phone: "" };
}

export const FieldMapping = {
  encode(message: FieldMapping, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.identifier !== "") {
      writer.uint32(10).string(message.identifier);
    }
    if (message.displayName !== "") {
      writer.uint32(18).string(message.displayName);
    }
    if (message.email !== "") {
      writer.uint32(26).string(message.email);
    }
    if (message.phone !== "") {
      writer.uint32(34).string(message.phone);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): FieldMapping {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFieldMapping();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.identifier = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.displayName = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.email = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.phone = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): FieldMapping {
    return {
      identifier: isSet(object.identifier) ? String(object.identifier) : "",
      displayName: isSet(object.displayName) ? String(object.displayName) : "",
      email: isSet(object.email) ? String(object.email) : "",
      phone: isSet(object.phone) ? String(object.phone) : "",
    };
  },

  toJSON(message: FieldMapping): unknown {
    const obj: any = {};
    message.identifier !== undefined && (obj.identifier = message.identifier);
    message.displayName !== undefined && (obj.displayName = message.displayName);
    message.email !== undefined && (obj.email = message.email);
    message.phone !== undefined && (obj.phone = message.phone);
    return obj;
  },

  create(base?: DeepPartial<FieldMapping>): FieldMapping {
    return FieldMapping.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<FieldMapping>): FieldMapping {
    const message = createBaseFieldMapping();
    message.identifier = object.identifier ?? "";
    message.displayName = object.displayName ?? "";
    message.email = object.email ?? "";
    message.phone = object.phone ?? "";
    return message;
  },
};

function createBaseIdentityProviderUserInfo(): IdentityProviderUserInfo {
  return { identifier: "", displayName: "", email: "", phone: "" };
}

export const IdentityProviderUserInfo = {
  encode(message: IdentityProviderUserInfo, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.identifier !== "") {
      writer.uint32(10).string(message.identifier);
    }
    if (message.displayName !== "") {
      writer.uint32(18).string(message.displayName);
    }
    if (message.email !== "") {
      writer.uint32(26).string(message.email);
    }
    if (message.phone !== "") {
      writer.uint32(34).string(message.phone);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): IdentityProviderUserInfo {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIdentityProviderUserInfo();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.identifier = reader.string();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.displayName = reader.string();
          continue;
        case 3:
          if (tag !== 26) {
            break;
          }

          message.email = reader.string();
          continue;
        case 4:
          if (tag !== 34) {
            break;
          }

          message.phone = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): IdentityProviderUserInfo {
    return {
      identifier: isSet(object.identifier) ? String(object.identifier) : "",
      displayName: isSet(object.displayName) ? String(object.displayName) : "",
      email: isSet(object.email) ? String(object.email) : "",
      phone: isSet(object.phone) ? String(object.phone) : "",
    };
  },

  toJSON(message: IdentityProviderUserInfo): unknown {
    const obj: any = {};
    message.identifier !== undefined && (obj.identifier = message.identifier);
    message.displayName !== undefined && (obj.displayName = message.displayName);
    message.email !== undefined && (obj.email = message.email);
    message.phone !== undefined && (obj.phone = message.phone);
    return obj;
  },

  create(base?: DeepPartial<IdentityProviderUserInfo>): IdentityProviderUserInfo {
    return IdentityProviderUserInfo.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<IdentityProviderUserInfo>): IdentityProviderUserInfo {
    const message = createBaseIdentityProviderUserInfo();
    message.identifier = object.identifier ?? "";
    message.displayName = object.displayName ?? "";
    message.email = object.email ?? "";
    message.phone = object.phone ?? "";
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
