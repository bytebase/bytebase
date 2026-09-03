import { describe, expect, test } from "vitest";
import { WorkloadIdentityConfig_ProviderType } from "@/types/proto-es/v1/workload_identity_service_pb";
import {
  parseWorkloadIdentitySubjectPattern,
  resolveWorkloadIdentityProviderType,
} from "./workloadIdentity";

const { GITHUB, GITLAB, PROVIDER_TYPE_UNSPECIFIED } =
  WorkloadIdentityConfig_ProviderType;

describe("resolveWorkloadIdentityProviderType", () => {
  test("returns a declared provider unchanged", () => {
    expect(
      resolveWorkloadIdentityProviderType({
        providerType: GITLAB,
        subjectPattern: "repo:acme-corp/*",
      })
    ).toBe(GITLAB);
  });

  // provider_type is optional and identities created through the API often
  // omit it. Leaving those unresolved seeds the edit form with an unspecified
  // provider, which recomputes an empty subject pattern on mount and locks the
  // operator out of the identity.
  test("infers the provider from the subject when none was declared", () => {
    expect(
      resolveWorkloadIdentityProviderType({
        providerType: PROVIDER_TYPE_UNSPECIFIED,
        subjectPattern: "repo:acme-corp/deploy:ref:refs/heads/main",
      })
    ).toBe(GITHUB);
    expect(
      resolveWorkloadIdentityProviderType({
        providerType: PROVIDER_TYPE_UNSPECIFIED,
        subjectPattern: "project_path:grp/proj:*",
      })
    ).toBe(GITLAB);
  });

  test("returns undefined when the subject names no provider", () => {
    expect(
      resolveWorkloadIdentityProviderType({
        providerType: PROVIDER_TYPE_UNSPECIFIED,
        subjectPattern: "custom:whatever",
      })
    ).toBeUndefined();
    expect(resolveWorkloadIdentityProviderType(undefined)).toBeUndefined();
  });
});

describe("parseWorkloadIdentitySubjectPattern", () => {
  // The parser and the form's initial provider must agree, so both read the
  // same resolution.
  test("parses an identity that never declared a provider", () => {
    expect(
      parseWorkloadIdentitySubjectPattern({
        workloadIdentityConfig: {
          providerType: PROVIDER_TYPE_UNSPECIFIED,
          subjectPattern: "repo:acme-corp/deploy:ref:refs/heads/main",
        },
      })
    ).toEqual({ owner: "acme-corp", repo: "deploy", branch: "main" });
  });
});
