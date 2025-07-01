import type { OAuth2IdentityProviderConfig } from "@/types/proto-es/v1/idp_service_pb";
import { IdentityProviderType } from "@/types/proto-es/v1/idp_service_pb";

export const identityProviderTypeToString = (
  type: IdentityProviderType
): string => {
  if (type === IdentityProviderType.OAUTH2) {
    return "OAuth 2.0";
  } else if (type === IdentityProviderType.OIDC) {
    return "OIDC";
  } else if (type == IdentityProviderType.LDAP) {
    return "LDAP";
  } else {
    throw new Error(`identity provider type ${type} not found`);
  }
};

export interface OAuth2IdentityProviderTemplate {
  title: string;
  name: string;
  domain: string;
  type: IdentityProviderType.OAUTH2;
  config: OAuth2IdentityProviderConfig;
}
