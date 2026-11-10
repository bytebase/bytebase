import { describe, expect, test } from "vitest";
import { WorkloadIdentityConfig_ProviderType } from "@/types/proto-es/v1/workload_identity_service_pb";
import { parseWorkloadIdentitySubjectPattern } from "./workloadIdentity";

const parse = (
  subjectPattern: string,
  providerType = WorkloadIdentityConfig_ProviderType.GITHUB
) =>
  parseWorkloadIdentitySubjectPattern({
    workloadIdentityConfig: { subjectPattern, providerType },
  });

describe("parseWorkloadIdentitySubjectPattern", () => {
  test.each([
    ["repo:acme-corp/*", { owner: "acme-corp", repo: "", branch: "" }],
    [
      "repo:acme-corp/deploy:*",
      { owner: "acme-corp", repo: "deploy", branch: "" },
    ],
    [
      "repo:acme-corp/deploy:ref:refs/heads/main",
      { owner: "acme-corp", repo: "deploy", branch: "main" },
    ],
    ["repo:acme-corp@111/*", { owner: "acme-corp", repo: "", branch: "" }],
    [
      "repo:acme-corp@111/deploy:*",
      { owner: "acme-corp", repo: "deploy", branch: "" },
    ],
    [
      "repo:acme-corp@111/deploy@222:ref:refs/heads/main",
      { owner: "acme-corp", repo: "deploy", branch: "main" },
    ],
  ])("reads GitHub %s without the IDs", (pattern, expected) => {
    expect(parse(pattern)).toEqual(expected);
  });

  test("reads a GitLab pattern", () => {
    expect(
      parse(
        "project_path:acme/deploy:ref_type:tag:ref:v1",
        WorkloadIdentityConfig_ProviderType.GITLAB
      )
    ).toEqual({ owner: "acme", repo: "deploy", branch: "v1", refType: "tag" });
  });

  test("returns nothing for a pattern of another provider", () => {
    expect(parse("project_path:acme/deploy:*")).toBeUndefined();
  });
});
