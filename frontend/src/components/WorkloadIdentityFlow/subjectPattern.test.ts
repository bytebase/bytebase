import { describe, expect, test } from "vitest";
import { WorkloadIdentityConfig_ProviderType } from "@/types/proto-es/v1/workload_identity_service_pb";
import {
  buildSubjectPattern,
  parseSubjectPattern,
  replaceSubjectFields,
  subjectEnvironment,
  validateSubjectFields,
} from "./subjectPattern";

const GITHUB = WorkloadIdentityConfig_ProviderType.GITHUB;
const GITLAB = WorkloadIdentityConfig_ProviderType.GITLAB;

const fields = (owner: string, repo = "", ownerId = "") => ({
  owner,
  ownerId,
  repo,
});

describe("parseSubjectPattern", () => {
  // The cases of TestParseGitHubSubjectPattern in
  // backend/plugin/idp/wif/github_test.go, so both sides split a pattern alike.
  test.each([
    ["repo:acme-corp/*", { fields: fields("acme-corp"), repoId: "", rest: "" }],
    [
      "repo:acme-corp@111/*",
      { fields: fields("acme-corp", "", "111"), repoId: "", rest: "" },
    ],
    [
      "repo:acme-corp/deploy:*",
      { fields: fields("acme-corp", "deploy"), repoId: "", rest: ":*" },
    ],
    [
      "repo:acme-corp@111/deploy@222:ref:refs/heads/main",
      {
        fields: fields("acme-corp", "deploy", "111"),
        repoId: "222",
        rest: ":ref:refs/heads/main",
      },
    ],
    ["repo:acme-corp/deploy*", undefined],
    ["repo:acme-corp/deploy", undefined],
    ["repo:acme-corp@abc/*", undefined],
    ["repo:acme*", undefined],
    ["repo:/deploy:*", undefined],
    ["project_path:acme/deploy:*", undefined],
  ])("GitHub %s", (pattern, expected) => {
    expect(parseSubjectPattern(GITHUB, pattern)).toEqual(expected);
  });

  test.each([
    ["project_path:acme/*", { fields: fields("acme"), repoId: "", rest: "" }],
    [
      "project_path:acme/deploy:ref_type:branch:ref:main",
      {
        fields: fields("acme", "deploy"),
        repoId: "",
        rest: ":ref_type:branch:ref:main",
      },
    ],
    [
      "project_path:acme/infra/deploy:*",
      { fields: fields("acme", "infra/deploy"), repoId: "", rest: ":*" },
    ],
    ["project_path:acme@1/deploy:*", undefined],
    ["repo:acme/deploy:*", undefined],
  ])("GitLab %s", (pattern, expected) => {
    expect(parseSubjectPattern(GITLAB, pattern)).toEqual(expected);
  });
});

describe("buildSubjectPattern", () => {
  test.each([
    [GITHUB, fields("acme-corp"), "repo:acme-corp/*"],
    [GITHUB, fields("acme-corp", "deploy"), "repo:acme-corp/deploy:*"],
    [
      GITHUB,
      fields("acme-corp", "deploy", "111"),
      "repo:acme-corp@111/deploy:*",
    ],
    [GITHUB, fields("acme-corp", "", "111"), "repo:acme-corp@111/*"],
    [GITLAB, fields("acme", "deploy"), "project_path:acme/deploy:*"],
    [GITLAB, fields("acme"), "project_path:acme/*"],
  ] as const)("%s %o", (provider, value, expected) => {
    expect(buildSubjectPattern(provider, value)).toBe(expected);
  });

  test("round-trips through parse", () => {
    for (const pattern of [
      "repo:acme-corp/*",
      "repo:acme-corp@111/deploy@222:environment:prod",
      "repo:acme-corp/deploy:ref:refs/heads/main",
    ]) {
      const parsed = parseSubjectPattern(GITHUB, pattern);
      expect(parsed).toBeDefined();
      if (!parsed) continue;
      expect(
        buildSubjectPattern(GITHUB, parsed.fields, parsed.repoId, parsed.rest)
      ).toBe(pattern);
    }
  });
});

describe("replaceSubjectFields", () => {
  test("keeps the remainder when the owner changes", () => {
    expect(
      replaceSubjectFields(
        GITHUB,
        "repo:acme-corp/deploy:environment:prod",
        fields("acme-labs", "deploy")
      )
    ).toEqual({ pattern: "repo:acme-labs/deploy:environment:prod" });
  });

  test("adds an owner ID and keeps the repository ID", () => {
    expect(
      replaceSubjectFields(
        GITHUB,
        "repo:acme-corp/deploy@222:ref:refs/heads/main",
        fields("acme-corp", "deploy", "111")
      )
    ).toEqual({ pattern: "repo:acme-corp@111/deploy@222:ref:refs/heads/main" });
  });

  test("drops the repository ID when the repository changes", () => {
    expect(
      replaceSubjectFields(
        GITHUB,
        "repo:acme-corp/deploy@222:*",
        fields("acme-corp", "infra")
      )
    ).toEqual({ pattern: "repo:acme-corp/infra:*" });
  });

  test("never widens by clearing the repository of a narrowed pattern", () => {
    expect(
      replaceSubjectFields(
        GITHUB,
        "repo:acme-corp/deploy:environment:prod",
        fields("acme-corp")
      )
    ).toEqual({ error: "repo-required" });
  });

  test("clears the repository of a pattern that admits every job", () => {
    expect(
      replaceSubjectFields(
        GITHUB,
        "repo:acme-corp/deploy:*",
        fields("acme-corp")
      )
    ).toEqual({ pattern: "repo:acme-corp/*" });
  });

  test("builds a fresh pattern when there is none", () => {
    expect(replaceSubjectFields(GITLAB, "", fields("acme", "deploy"))).toEqual({
      pattern: "project_path:acme/deploy:*",
    });
  });
});

describe("validateSubjectFields", () => {
  test.each([
    [GITHUB, fields(""), { owner: "required" }],
    [GITHUB, fields("acme*"), { owner: "invalid" }],
    [GITHUB, fields("acme", "", "12a"), { ownerId: "invalid" }],
    [GITHUB, fields("acme", "a/b"), { repo: "invalid" }],
    [GITLAB, fields("acme", "infra/deploy"), {}],
    [GITHUB, fields("acme-corp", "deploy", "111"), {}],
  ] as const)("%s %o", (provider, value, expected) => {
    expect(validateSubjectFields(provider, value)).toEqual(expected);
  });
});

describe("subjectEnvironment", () => {
  test.each([
    ["repo:acme-corp/deploy:environment:prod", "prod"],
    ["repo:acme-corp/deploy:environment:*", "*"],
    ["repo:acme-corp/deploy:*", undefined],
  ])("%s", (pattern, expected) => {
    expect(subjectEnvironment(pattern)).toBe(expected);
  });
});
