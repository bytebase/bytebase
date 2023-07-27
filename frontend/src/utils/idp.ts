import {
  IdentityProviderType,
  OAuth2AuthStyle,
  OAuth2IdentityProviderConfig,
} from "@/types/proto/v1/idp_service";

export const identityProviderTypeToString = (
  type: IdentityProviderType
): string => {
  if (type === IdentityProviderType.OAUTH2) {
    return "OAuth 2.0";
  } else if (type === IdentityProviderType.OIDC) {
    return "OIDC";
  } else {
    throw new Error(`identity provider type ${type} not found`);
  }
};

interface OAuth2IdentityProviderTemplate {
  title: string;
  name: string;
  domain: string;
  type: IdentityProviderType.OAUTH2;
  config: OAuth2IdentityProviderConfig;
}

export type IdentityProviderTemplate = OAuth2IdentityProviderTemplate;

export const identityProviderTemplateList: IdentityProviderTemplate[] = [
  {
    title: "GitHub",
    name: "",
    domain: "github.com",
    type: IdentityProviderType.OAUTH2,
    config: {
      clientId: "",
      clientSecret: "",
      authUrl: "https://github.com/login/oauth/authorize",
      tokenUrl: "https://github.com/login/oauth/access_token",
      userInfoUrl: "https://api.github.com/user",
      scopes: ["user"],
      skipTlsVerify: false,
      authStyle: OAuth2AuthStyle.IN_PARAMS,
      fieldMapping: {
        identifier: "email",
        displayName: "name",
        email: "email",
      },
    },
  },
  {
    title: "GitLab",
    name: "",
    domain: "gitlab.com",
    type: IdentityProviderType.OAUTH2,
    config: {
      clientId: "",
      clientSecret: "",
      authUrl: "https://gitlab.com/oauth/authorize",
      tokenUrl: "https://gitlab.com/oauth/token",
      userInfoUrl: "https://gitlab.com/api/v4/user",
      scopes: ["read_user"],
      skipTlsVerify: false,
      authStyle: OAuth2AuthStyle.IN_PARAMS,
      fieldMapping: {
        identifier: "email",
        displayName: "name",
        email: "email",
      },
    },
  },
  {
    title: "Google",
    name: "",
    domain: "google.com",
    type: IdentityProviderType.OAUTH2,
    config: {
      clientId: "",
      clientSecret: "",
      authUrl: "https://accounts.google.com/o/oauth2/v2/auth",
      tokenUrl: "https://oauth2.googleapis.com/token",
      userInfoUrl: "https://www.googleapis.com/oauth2/v2/userinfo",
      scopes: [
        "https://www.googleapis.com/auth/userinfo.email",
        "https://www.googleapis.com/auth/userinfo.profile",
      ],
      skipTlsVerify: false,
      authStyle: OAuth2AuthStyle.IN_PARAMS,
      fieldMapping: {
        identifier: "email",
        displayName: "name",
        email: "email",
      },
    },
  },
  {
    title: "Custom",
    name: "",
    domain: "",
    type: IdentityProviderType.OAUTH2,
    config: {
      clientId: "",
      clientSecret: "",
      authUrl: "",
      tokenUrl: "",
      userInfoUrl: "",
      scopes: [],
      skipTlsVerify: false,
      authStyle: OAuth2AuthStyle.IN_PARAMS,
      fieldMapping: {
        identifier: "",
        displayName: "",
        email: "",
      },
    },
  },
];
