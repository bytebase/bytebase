// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  AuthorizationRequestSchema,
  LoginIdentityProviderSchema,
} from "@/types/proto-es/v1/auth_service_pb";
import { IdentityProviderType } from "@/types/proto-es/v1/idp_service_pb";
import { openWindowForSSO } from "./sso";

describe("openWindowForSSO", () => {
  test.each([
    {
      identityProvider: create(LoginIdentityProviderSchema, {
        type: IdentityProviderType.LDAP,
      }),
      message: "This identity provider type does not support SSO redirect.",
    },
    {
      identityProvider: create(LoginIdentityProviderSchema, {
        type: IdentityProviderType.OAUTH2,
        authorizationRequest: create(AuthorizationRequestSchema),
      }),
      message:
        "The identity provider published no authorization endpoint. Check its configuration.",
    },
    {
      identityProvider: create(LoginIdentityProviderSchema, {
        type: IdentityProviderType.OIDC,
        authorizationRequest: create(AuthorizationRequestSchema, {
          endpoint: "javascript:alert(1)",
        }),
      }),
      message:
        "Invalid authorization URL: it must be a valid HTTP or HTTPS URL.",
    },
  ])(
    "throws the localized SSO configuration error",
    async ({ identityProvider, message }) => {
      await expect(openWindowForSSO(identityProvider)).rejects.toMatchObject({
        message,
      });
    }
  );
});
