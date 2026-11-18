import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  WorkloadIdentityConfig_ProviderType,
  WorkloadIdentityConfigSchema,
} from "@/types/proto-es/v1/workload_identity_service_pb";
import { workloadIdentityTrust } from "./workloadIdentityTrust";

const { GITHUB, GITLAB, OIDC, PROVIDER_TYPE_UNSPECIFIED } =
  WorkloadIdentityConfig_ProviderType;

const trust = (
  providerType: WorkloadIdentityConfig_ProviderType,
  subjectPattern: string,
  issuerUrl = ""
) =>
  workloadIdentityTrust(
    create(WorkloadIdentityConfigSchema, {
      providerType,
      subjectPattern,
      issuerUrl,
    })
  );

describe("workloadIdentityTrust", () => {
  test.each([
    [
      "owner and repository",
      GITHUB,
      "repo:example-org/billing-api:*",
      {
        provider: "github",
        primary: { kind: "repository", value: "example-org/billing-api" },
      },
    ],
    [
      "ids are dropped",
      GITHUB,
      "repo:example-org@4182733/docs@90211456:*",
      {
        provider: "github",
        primary: { kind: "repository", value: "example-org/docs" },
      },
    ],
    [
      "repository left blank",
      GITHUB,
      "repo:example-org/*",
      {
        provider: "github",
        primary: { kind: "any-repository", owner: "example-org" },
      },
    ],
    [
      "gitlab project left blank",
      GITLAB,
      "project_path:example-group/*",
      {
        provider: "gitlab",
        primary: { kind: "any-repository", owner: "example-group" },
      },
    ],
    [
      "one environment",
      GITHUB,
      "repo:example-org/billing-api:environment:prod",
      {
        provider: "github",
        primary: { kind: "repository", value: "example-org/billing-api" },
        secondary: { kind: "environment", value: "prod" },
      },
    ],
    [
      "any environment",
      GITHUB,
      "repo:example-org/billing-api:environment:*",
      {
        provider: "github",
        primary: { kind: "repository", value: "example-org/billing-api" },
        secondary: { kind: "any-environment" },
      },
    ],
    [
      "other remainder as written",
      GITLAB,
      "project_path:example-group/mobile/app:ref_type:branch:ref:main",
      {
        provider: "gitlab",
        primary: { kind: "repository", value: "example-group/mobile/app" },
        secondary: { kind: "pattern", value: "ref_type:branch:ref:main" },
      },
    ],
    [
      "pattern the fields cannot read",
      GITHUB,
      "repo:*",
      { provider: "github", primary: { kind: "pattern", value: "repo:*" } },
    ],
  ])("%s", (_name, providerType, pattern, expected) => {
    expect(trust(providerType, pattern)).toEqual(expected);
  });

  test("other OIDC shows the issuer host and the pattern", () => {
    expect(
      trust(OIDC, "pipeline:nightly:*", "https://oidc.ci.example.com/path")
    ).toEqual({
      provider: "oidc",
      primary: { kind: "issuer", value: "oidc.ci.example.com" },
      secondary: { kind: "pattern", value: "pipeline:nightly:*" },
    });
  });

  test("an unset provider reads as other OIDC", () => {
    expect(trust(PROVIDER_TYPE_UNSPECIFIED, "", "not a url").primary).toEqual({
      kind: "issuer",
      value: "not a url",
    });
  });
});
