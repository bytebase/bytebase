import { fromJson, toJson } from "@bufbuild/protobuf";
import type { IdentityProvider as OldIdentityProvider } from "@/types/proto/v1/idp_service";
import { IdentityProvider as OldIdentityProviderProto } from "@/types/proto/v1/idp_service";
import type { IdentityProvider as NewIdentityProvider } from "@/types/proto-es/v1/idp_service_pb";
import { IdentityProviderSchema } from "@/types/proto-es/v1/idp_service_pb";
import { IdentityProviderType as OldIdentityProviderType } from "@/types/proto/v1/idp_service";
import { IdentityProviderType as NewIdentityProviderType } from "@/types/proto-es/v1/idp_service_pb";

// Convert old proto to proto-es
export const convertOldIdentityProviderToNew = (oldProvider: OldIdentityProvider): NewIdentityProvider => {
  // Use toJSON to convert old proto to JSON, then fromJson to convert to proto-es
  const json = OldIdentityProviderProto.toJSON(oldProvider) as any; // Type assertion needed due to proto type incompatibility
  return fromJson(IdentityProviderSchema, json);
};

// Convert proto-es to old proto
export const convertNewIdentityProviderToOld = (newProvider: NewIdentityProvider): OldIdentityProvider => {
  // Use toJson to convert proto-es to JSON, then fromJSON to convert to old proto
  const json = toJson(IdentityProviderSchema, newProvider);
  return OldIdentityProviderProto.fromJSON(json);
};

// Convert old IdentityProviderType enum to new (string to numeric)
export const convertOldIdentityProviderTypeToNew = (oldType: OldIdentityProviderType): NewIdentityProviderType => {
  const mapping: Record<OldIdentityProviderType, NewIdentityProviderType> = {
    [OldIdentityProviderType.IDENTITY_PROVIDER_TYPE_UNSPECIFIED]: NewIdentityProviderType.IDENTITY_PROVIDER_TYPE_UNSPECIFIED,
    [OldIdentityProviderType.OAUTH2]: NewIdentityProviderType.OAUTH2,
    [OldIdentityProviderType.OIDC]: NewIdentityProviderType.OIDC,
    [OldIdentityProviderType.LDAP]: NewIdentityProviderType.LDAP,
    [OldIdentityProviderType.UNRECOGNIZED]: NewIdentityProviderType.IDENTITY_PROVIDER_TYPE_UNSPECIFIED,
  };
  return mapping[oldType] ?? NewIdentityProviderType.IDENTITY_PROVIDER_TYPE_UNSPECIFIED;
};

// Convert new IdentityProviderType enum to old (numeric to string)
export const convertNewIdentityProviderTypeToOld = (newType: NewIdentityProviderType): OldIdentityProviderType => {
  const mapping: Record<NewIdentityProviderType, OldIdentityProviderType> = {
    [NewIdentityProviderType.IDENTITY_PROVIDER_TYPE_UNSPECIFIED]: OldIdentityProviderType.IDENTITY_PROVIDER_TYPE_UNSPECIFIED,
    [NewIdentityProviderType.OAUTH2]: OldIdentityProviderType.OAUTH2,
    [NewIdentityProviderType.OIDC]: OldIdentityProviderType.OIDC,
    [NewIdentityProviderType.LDAP]: OldIdentityProviderType.LDAP,
  };
  return mapping[newType] ?? OldIdentityProviderType.UNRECOGNIZED;
};