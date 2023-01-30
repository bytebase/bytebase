import {
  IdentityProviderType,
  OAuth2IdentityProviderConfig,
} from "@/types/proto/v1/idp_service";

export const identityProviderTypeToString = (
  type: IdentityProviderType
): string => {
  if (type === IdentityProviderType.OAUTH2) {
    return "OAuth2";
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
    title: "Google",
    name: "idp-google",
    domain: "google.com",
    type: IdentityProviderType.OAUTH2,
    config: {
      clientId: "YOUR_CLIENT_ID",
      clientSecret: "YOUR_CLIENT_SECRET",
      authUrl: "https://accounts.google.com/o/oauth2/v2/auth",
      tokenUrl: "https://oauth2.googleapis.com/token",
      userInfoUrl: "https://www.googleapis.com/oauth2/v2/userinfo",
      scopes: [
        "https://www.googleapis.com/auth/userinfo.email",
        "https://www.googleapis.com/auth/userinfo.profile",
      ],
      fieldMapping: {
        identifier: "email",
        displayName: "name",
        email: "email",
      },
    },
  },
  {
    title: "GitHub",
    name: "idp-github",
    domain: "github.com",
    type: IdentityProviderType.OAUTH2,
    config: {
      clientId: "YOUR_CLIENT_ID",
      clientSecret: "YOUR_CLIENT_SECRET",
      authUrl: "https://github.com/login/oauth/authorize",
      tokenUrl: "https://github.com/login/oauth/access_token",
      userInfoUrl: "https://api.github.com/user",
      scopes: ["user"],
      fieldMapping: {
        identifier: "login",
        displayName: "name",
        email: "email",
      },
    },
  },
  {
    title: "GitLab",
    name: "idp-gitlab",
    domain: "gitlab.com",
    type: IdentityProviderType.OAUTH2,
    config: {
      clientId: "YOUR_CLIENT_ID",
      clientSecret: "YOUR_CLIENT_SECRET",
      authUrl: "https://gitlab.com/oauth/authorize",
      tokenUrl: "https://gitlab.com/oauth/token",
      userInfoUrl: "https://gitlab.com/api/v4/user",
      scopes: ["read_user"],
      fieldMapping: {
        identifier: "username",
        displayName: "name",
        email: "public_email",
      },
    },
  },
];
