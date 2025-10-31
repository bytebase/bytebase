import type { IdentityProviderType } from "./proto-es/v1/idp_service_pb";

export type OAuthWindowEventPayload = {
  error: string;
  code: string;
};

export type OAuthState = {
  token: string;
  event: string;
  popup?: boolean;
  redirect?: string;
  timestamp: number;
  // IdP type to determine which context to use in callback (oauth2Context vs oidcContext)
  idpType: IdentityProviderType;
};
